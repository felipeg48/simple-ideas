// Package serve is the `si play` server: decks are rendered from markdown on
// every request, and a polling watcher pushes a reload over server-sent
// events when a deck, an asset or a user stylesheet changes.
package serve

import (
	"fmt"
	"hash/fnv"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/felipeg48/simple-ideas/internal/assets"
	"github.com/felipeg48/simple-ideas/internal/config"
	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/felipeg48/simple-ideas/internal/render"
)

// Server renders the decks under Root (a directory or one .md file).
type Server struct {
	Root     string   // directory or deck file
	CSSPaths []string // user CSS files or directories (--css)
	Theme    string   // --theme override; empty keeps each deck's theme: front matter
	Conf     string   // --conf path; empty falls back to si.conf next to the deck
	Terminal bool     // enable the play terminal drawer; set only on a loopback address
	Dev      bool     // --dev: in-place hot reload instead of full-page reload
	Remote   bool     // --remote: decks join the relay (see remote.go)
	Title    string   // --title / --site: the index page title; empty uses the directory name
	Interval time.Duration

	dir     string
	file    string // set when Root is a single deck
	mu      sync.Mutex
	clients map[chan struct{}]bool
	term    *termHub
	relay   *relay
}

// New prepares a server for root.
func New(root string, css []string, theme, conf string) (*Server, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	st, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	s := &Server{Root: abs, CSSPaths: css, Theme: theme, Conf: conf, Interval: 300 * time.Millisecond, clients: map[chan struct{}]bool{}}
	if st.IsDir() {
		s.dir = abs
	} else {
		s.dir, s.file = filepath.Dir(abs), abs
	}
	s.term = newTermHub(s.dir)
	s.relay = newRelay()
	return s, nil
}

// Handler serves the index, the decks, the SSE endpoint and static files.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/_si/events", s.events)
	if s.Remote {
		mux.HandleFunc("GET /_si/remote/events", s.relay.events)
		mux.HandleFunc("POST /_si/remote/send", s.relay.send)
	}
	if s.Terminal {
		// Play terminal: loopback only (see Server.Terminal). Embedded xterm.js
		// assets are static; the shell endpoints run per tab.
		mux.Handle("/_si/term/assets/", http.StripPrefix("/_si/term/assets/", http.FileServer(http.FS(assets.TermFS))))
		// Method-restricted: a GET (link, <img>, navigation) never starts or
		// drives a shell; only the drawer's fetch POSTs do. out is the stream.
		mux.HandleFunc("POST /_si/term/api/new", s.term.newTab)
		mux.HandleFunc("GET /_si/term/api/out", s.term.out)
		mux.HandleFunc("POST /_si/term/api/in", s.term.in)
		mux.HandleFunc("POST /_si/term/api/resize", s.term.resize)
		mux.HandleFunc("POST /_si/term/api/close", s.term.close)
	}
	static := http.FileServer(http.Dir(s.dir))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		p := path.Clean("/" + r.URL.Path)
		switch {
		case p == "/" && s.file != "":
			http.Redirect(w, r, "/"+strings.TrimSuffix(filepath.Base(s.file), ".md")+".html", http.StatusFound)
		case p == "/" || p == "/index.html":
			if s.source("/index") == "" {
				if g, err := deck.LoadGuide(s.dir); err == nil && len(g.Pages) > 0 {
					http.Redirect(w, r, "/"+guideHome(g), http.StatusFound) // serving a guide folder: open its first page
					return
				}
			}
			s.index(w)
		case s.file == "" && fileExists(filepath.Join(s.dir, filepath.FromSlash(p), deck.NavFile)):
			if g, err := deck.LoadGuide(filepath.Join(s.dir, filepath.FromSlash(p))); err == nil && len(g.Pages) > 0 {
				http.Redirect(w, r, strings.TrimSuffix(p, "/")+"/"+guideHome(g), http.StatusFound) // a guide folder opens at its first page
				return
			}
			static.ServeHTTP(w, r)
		case strings.HasSuffix(p, ".html"):
			md := s.source(strings.TrimSuffix(p, ".html"))
			if md == "" {
				if !s.bookPage(w, p) { // a book's generated pages (cover, contents, index, book.html)
					static.ServeHTTP(w, r)
				}
				return
			}
			s.deck(w, md, s.Terminal)
		default:
			static.ServeHTTP(w, r)
		}
	})
	return mux
}

func (s *Server) css() []render.Stylesheet {
	sheets, err := render.LoadCSS(s.CSSPaths)
	if err != nil {
		log.Printf("si play: %v", err)
	}
	return sheets
}

// deck renders one deck; term turns the terminal drawer on (loopback only).
func (s *Server) deck(w http.ResponseWriter, md string, term bool) {
	d, err := deck.Parse(md)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	cfg := config.Load(s.Conf, filepath.Join(s.dir, "si.conf"))
	for _, w := range cfg.Warnings {
		log.Printf("si.conf: %s", w)
	}
	// The terminal drawer shows only when the capability is on (loopback play)
	// and the feature is not explicitly disabled in si.conf.
	termOn := term
	if v, ok := cfg.Features["terminal"]; ok && !v {
		termOn = false
	}
	opt := render.Options{LiveReload: true, Dev: s.Dev, ShowProblems: true, UserCSS: s.css(), Media: render.MediaLink, Theme: s.Theme, Config: cfg, Terminal: termOn, Remote: s.Remote, Pages: s.pages(cfg)}
	if g := deck.FindGuide(md, s.dir); g != nil && g.Item(md) != nil { // a guide or book renders as a whole (numbering, search across pages)
		s.writeBookPage(w, g, opt, filepath.Join(filepath.Dir(md), strings.TrimSuffix(filepath.Base(md), filepath.Ext(md))+".html"))
		return
	}
	if g := deck.FindGuide(md, s.dir); g != nil && g.Item(md) != nil { // a guide page: the guide's nav and settings
		if gd, err := deck.ParseWith(md, g.Defaults()); err == nil {
			d, opt.Guide = gd, g
		}
	}
	res := render.Deck(d, opt)
	for _, p := range res.Problems {
		log.Printf("si play: %s", p)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Write(res.HTML)
}

func fileExists(p string) bool { _, err := os.Stat(p); return err == nil }

// bookPage serves a book's generated page at URL path p, if p is one.
func (s *Server) bookPage(w http.ResponseWriter, p string) bool {
	name := path.Base(p)
	if name != "index.html" && name != "contents.html" && name != "book-index.html" && name != "book.html" {
		return false
	}
	file := filepath.Join(s.dir, filepath.FromSlash(p))
	g := deck.FindGuide(file, s.dir)
	if g == nil || !g.IsBook() || filepath.Dir(file) != g.Dir {
		return false
	}
	cfg := config.Load(s.Conf, filepath.Join(s.dir, "si.conf"))
	opt := render.Options{LiveReload: true, ShowProblems: true, UserCSS: s.css(), Media: render.MediaLink, Theme: s.Theme, Config: cfg}
	s.writeBookPage(w, g, opt, file)
	return true
}

// writeBookPage renders the whole book and writes the page whose output file
// (inside the book folder) is file.
// ponytail: renders every chapter per request; fine for a reference-sized book.
func (s *Server) writeBookPage(w http.ResponseWriter, g *deck.Guide, opt render.Options, file string) {
	pages, probs, _ := render.RenderGuide(g, opt)
	for _, pr := range probs {
		log.Printf("si: %s", pr)
	}
	rel, _ := filepath.Rel(g.Dir, file)
	for _, pg := range pages {
		if pg.Rel == filepath.ToSlash(rel) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "no-store")
			w.Write(pg.Result.HTML)
			return
		}
	}
	http.NotFound(w, nil)
}

// source is the .md (or .adoc) file a URL path (without .html) names, or "".
func (s *Server) source(stem string) string {
	for _, ext := range []string{".md", ".adoc", ".asciidoc"} {
		f := filepath.Join(s.dir, filepath.FromSlash(stem)+ext)
		if _, err := os.Stat(f); err == nil {
			return f
		}
	}
	return ""
}

// guideHome is the first page of a guide, relative to its folder.
func guideHome(g *deck.Guide) string {
	if g.IsBook() {
		return "index.html" // the cover
	}
	r, _ := filepath.Rel(g.Dir, g.Pages[0])
	return filepath.ToSlash(strings.TrimSuffix(r, filepath.Ext(r)) + ".html")
}

// pages is the find (/) page list for the served folder, read per request so
// a new deck or .html file shows up after a reload.
// ponytail: parses every deck per request; fine for a folder of talks.
func (s *Server) pages(cfg *config.Config) []render.Page {
	files := []string{s.file}
	if s.file == "" {
		files, _ = deck.List(s.dir)
	}
	var decks []*deck.Deck
	for _, f := range files {
		if d, err := deck.Parse(f); err == nil {
			decks = append(decks, d)
		}
	}
	return render.Pages(decks, s.file == "", false, s.dir, cfg)
}

func (s *Server) index(w http.ResponseWriter) {
	files, err := deck.List(s.dir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var entries []render.IndexEntry
	for _, f := range files {
		if d, err := deck.Parse(f); err == nil {
			entries = append(entries, render.Entry(d))
		}
	}
	if dirs, err := os.ReadDir(s.dir); err == nil { // a guide in a subfolder is one tile
		for _, e := range dirs {
			if !e.IsDir() || strings.HasPrefix(e.Name(), ".") || strings.HasPrefix(e.Name(), "_") {
				continue
			}
			if g, err := deck.LoadGuide(filepath.Join(s.dir, e.Name())); err == nil && len(g.Pages) > 0 {
				entries = append(entries, render.IndexEntry{Href: e.Name() + "/" + guideHome(g), Title: g.Meta.Title, Subtitle: g.Meta.Description,
					Lab: true, Kind: guideKind(g), Sections: len(g.Pages), Version: g.Meta.Version, Status: g.Meta.Status})
			}
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	title := s.Title
	if title == "" {
		title = filepath.Base(s.dir)
	}
	w.Write(render.Index(title, entries, render.Options{LiveReload: true, UserCSS: s.css()}))
}

// events is the server-sent events endpoint: one `reload` event per change.
func (s *Server) events(w http.ResponseWriter, r *http.Request) {
	fl, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	ch := make(chan struct{}, 1)
	s.mu.Lock()
	s.clients[ch] = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.clients, ch)
		s.mu.Unlock()
	}()
	fmt.Fprint(w, "retry: 1000\n: connected\n\n")
	fl.Flush()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ch:
			fmt.Fprint(w, "event: reload\ndata: changed\n\n")
			fl.Flush()
		}
	}
}

// Broadcast tells every connected page to reload.
func (s *Server) Broadcast() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for ch := range s.clients {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

// CloseTerminals tears down any open play-terminal shells. No-op unless the
// terminal was enabled. Call on server shutdown so demo shells don't orphan.
func (s *Server) CloseTerminals() {
	if s.term != nil {
		s.term.closeAll()
	}
}

// Watch polls the deck directory and the CSS paths and broadcasts a reload
// when anything changes. It returns when stop is closed.
// ponytail: mtime polling, no fsnotify dependency; fine for deck-sized trees.
func (s *Server) Watch(stop <-chan struct{}) {
	last := s.signature()
	t := time.NewTicker(s.Interval)
	defer t.Stop()
	for {
		select {
		case <-stop:
			return
		case <-t.C:
			if sig := s.signature(); sig != last {
				last = sig
				log.Printf("si play: change detected, reloading")
				s.Broadcast()
			}
		}
	}
}

func (s *Server) signature() uint64 {
	h := fnv.New64a()
	add := func(p string, info fs.FileInfo) {
		fmt.Fprintf(h, "%s|%d|%d\n", p, info.ModTime().UnixNano(), info.Size())
	}
	walk := func(root string) {
		filepath.WalkDir(root, func(p string, e fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			name := e.Name()
			if p != root && (strings.HasPrefix(name, ".") || name == "node_modules") {
				if e.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if info, err := e.Info(); err == nil && !e.IsDir() {
				add(p, info)
			}
			return nil
		})
	}
	walk(s.dir)
	for _, c := range s.CSSPaths {
		walk(c)
	}
	return h.Sum64()
}

func guideKind(g *deck.Guide) string {
	if g.IsBook() {
		return "book"
	}
	return "guide"
}
