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

	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/felipeg48/simple-ideas/internal/render"
)

// Server renders the decks under Root (a directory or one .md file).
type Server struct {
	Root     string   // directory or deck file
	CSSPaths []string // user CSS files or directories (--css)
	Interval time.Duration

	dir     string
	file    string // set when Root is a single deck
	mu      sync.Mutex
	clients map[chan struct{}]bool
}

// New prepares a server for root.
func New(root string, css []string) (*Server, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	st, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	s := &Server{Root: abs, CSSPaths: css, Interval: 300 * time.Millisecond, clients: map[chan struct{}]bool{}}
	if st.IsDir() {
		s.dir = abs
	} else {
		s.dir, s.file = filepath.Dir(abs), abs
	}
	return s, nil
}

// Handler serves the index, the decks, the SSE endpoint and static files.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/_si/events", s.events)
	static := http.FileServer(http.Dir(s.dir))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		p := path.Clean("/" + r.URL.Path)
		switch {
		case p == "/" && s.file != "":
			http.Redirect(w, r, "/"+strings.TrimSuffix(filepath.Base(s.file), ".md")+".html", http.StatusFound)
		case p == "/" || p == "/index.html":
			s.index(w)
		case strings.HasSuffix(p, ".html"):
			md := filepath.Join(s.dir, filepath.FromSlash(strings.TrimSuffix(p, ".html")+".md"))
			if _, err := os.Stat(md); err != nil {
				static.ServeHTTP(w, r)
				return
			}
			s.deck(w, md)
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

func (s *Server) deck(w http.ResponseWriter, md string) {
	d, err := deck.Parse(md)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	res := render.Deck(d, render.Options{LiveReload: true, ShowProblems: true, UserCSS: s.css(), Media: render.MediaLink})
	for _, p := range res.Problems {
		log.Printf("si play: %s", p)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Write(res.HTML)
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
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Write(render.Index(filepath.Base(s.dir), entries, render.Options{LiveReload: true, UserCSS: s.css()}))
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
