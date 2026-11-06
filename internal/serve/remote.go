package serve

// Remote: `si play --remote` puts a second listener on the network for a
// tablet or phone: the teleprompter (role tp), a draw pad (role draw) and a
// clicker (role clicker).
//
// Both listeners share one relay: pages POST small JSON messages to
// /_si/remote/send and read everyone else's on the /_si/remote/events SSE
// stream. The deck on the presenter machine posts its state and obeys `cmd`
// messages; remote pages post commands and ink. This replaces BroadcastChannel
// (one browser only) across devices; built decks never load any of it.
//
// The network listener is gated by a per-run key: the printed URL carries
// ?k=<key>, the first request sets it as a cookie, every request must match.
// It serves decks (never with the terminal), static files under the deck
// directory, the relay and live reload, nothing else. The loopback listener
// serves the relay without a key, like the rest of `si play` on loopback.

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

// RemoteRoles are the remote page roles, in the order they are listed.
var RemoteRoles = []string{"tp", "draw", "clicker"}

// RemoteRoleNames describes each role for the picker page and the CLI.
var RemoteRoleNames = map[string]string{"tp": "Teleprompter", "draw": "Draw pad", "clicker": "Clicker"}

const remoteCookie = "si_remote"

// relay fans messages out to every subscriber but the sender, and keeps each
// deck's last state so a page that connects late shows the current slide.
type relay struct {
	mu   sync.Mutex
	subs map[chan []byte]string // channel -> client id
	last map[string][]byte      // deck name -> last state message
}

func newRelay() *relay {
	return &relay{subs: map[chan []byte]string{}, last: map[string][]byte{}}
}

func (h *relay) events(w http.ResponseWriter, r *http.Request) {
	fl, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	ch := make(chan []byte, 256)
	h.mu.Lock()
	h.subs[ch] = r.URL.Query().Get("c")
	var replay [][]byte
	for _, m := range h.last {
		replay = append(replay, m)
	}
	h.mu.Unlock()
	defer func() {
		h.mu.Lock()
		delete(h.subs, ch)
		h.mu.Unlock()
	}()
	fmt.Fprint(w, "retry: 1000\n: connected\n\n")
	for _, m := range replay {
		fmt.Fprintf(w, "data: %s\n\n", m)
	}
	fl.Flush()
	for {
		select {
		case <-r.Context().Done():
			return
		case m := <-ch:
			fmt.Fprintf(w, "data: %s\n\n", m)
			fl.Flush()
		}
	}
}

// send accepts one JSON message and relays it. A slow subscriber drops
// messages rather than stalling the others (the next state catches it up).
func (h *relay) send(w http.ResponseWriter, r *http.Request) {
	raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var m struct{ Type, Name string }
	if json.Unmarshal(raw, &m) != nil || m.Type == "" {
		http.Error(w, "want a JSON message with a type", http.StatusBadRequest)
		return
	}
	// one line per SSE data field: JSON from JSON.stringify has no raw newlines,
	// but re-encode anyway so a hand-written body cannot break the stream
	var v any
	json.Unmarshal(raw, &v)
	raw, _ = json.Marshal(v)
	from := r.URL.Query().Get("c")
	h.mu.Lock()
	if m.Type == "state" && m.Name != "" {
		h.last[m.Name] = raw
	}
	for ch, id := range h.subs {
		if id != "" && id == from {
			continue
		}
		select {
		case ch <- raw:
		default:
		}
	}
	h.mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

// RemoteHandler is the network-facing handler of `si play --remote`: key gate,
// role pages, static files, relay and live reload. Roles are the enabled
// remote roles (a subset of RemoteRoles).
func (s *Server) RemoteHandler(key string, roles []string) http.Handler {
	allowed := map[string]bool{}
	for _, r := range roles {
		allowed[r] = true
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/_si/events", s.events)
	mux.HandleFunc("GET /_si/remote/events", s.relay.events)
	mux.HandleFunc("POST /_si/remote/send", s.relay.send)
	static := http.FileServer(http.Dir(s.dir))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		switch {
		case strings.Contains(p, "/."):
			http.NotFound(w, r) // no dotfiles (.git, .env) on the network
		case p == "/" || p == "/index.html":
			s.remotePicker(w, roles)
		case strings.HasSuffix(p, ".html"):
			md := filepath.Join(s.dir, filepath.FromSlash(strings.TrimSuffix(strings.TrimPrefix(p, "/"), ".html")+".md"))
			role := r.URL.Query().Get("role")
			if !allowed[role] {
				http.Error(w, fmt.Sprintf("si remote: role %q is not enabled (open a link from the remote page)", role), http.StatusForbidden)
				return
			}
			if !s.isDeck(md) {
				http.NotFound(w, r)
				return
			}
			s.deck(w, md, false)
		default:
			static.ServeHTTP(w, r)
		}
	})
	return remoteGate(key, mux)
}

// remoteGate admits a request whose ?k= or cookie matches key; a matching
// ?k= also sets the cookie, so later requests (images, the relay) pass.
func remoteGate(key string, next http.Handler) http.Handler {
	ok := func(v string) bool { return v != "" && subtle.ConstantTimeCompare([]byte(v), []byte(key)) == 1 }
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if k := r.URL.Query().Get("k"); ok(k) {
			http.SetCookie(w, &http.Cookie{Name: remoteCookie, Value: key, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode})
		} else if c, err := r.Cookie(remoteCookie); err != nil || !ok(c.Value) {
			http.Error(w, "si remote: open the link printed by si play (it carries the key)", http.StatusForbidden)
			return
		}
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

// isDeck reports whether md is a deck under the served root.
func (s *Server) isDeck(md string) bool {
	if s.file != "" {
		return md == s.file
	}
	files, err := deck.List(s.dir)
	if err != nil {
		return false
	}
	for _, f := range files {
		if f == md {
			return true
		}
	}
	return false
}

// RemoteDecks returns the URL paths (/name.html) of the served decks.
func (s *Server) RemoteDecks() []string {
	files := []string{s.file}
	if s.file == "" {
		files, _ = deck.List(s.dir)
	}
	var out []string
	for _, f := range files {
		if rel, err := filepath.Rel(s.dir, f); err == nil {
			out = append(out, "/"+filepath.ToSlash(strings.TrimSuffix(rel, ".md"))+".html")
		}
	}
	return out
}

// remotePicker lists every deck with one link per enabled role.
func (s *Server) remotePicker(w http.ResponseWriter, roles []string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	var b strings.Builder
	b.WriteString(`<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="color-scheme" content="light dark"><title>si remote</title><style>
body{margin:0;padding:max(1rem,4vw);font:1.1rem/1.4 ui-sans-serif,system-ui,sans-serif;background:Canvas;color:CanvasText}
h1{font-size:1.4rem;margin:0 0 1rem}ul{list-style:none;margin:0;padding:0;display:grid;gap:.8rem}
li{display:flex;flex-wrap:wrap;align-items:center;gap:.6rem;padding:.8rem 1rem;border:1px solid color-mix(in srgb,CanvasText 20%,transparent);border-radius:.7rem}
li b{flex:1 1 12rem}a{padding:.55rem 1rem;border-radius:.5rem;background:color-mix(in srgb,CanvasText 10%,transparent);color:inherit;text-decoration:none;font-weight:600}
</style></head><body><h1>si remote</h1><ul>`)
	for _, p := range s.RemoteDecks() {
		title := strings.TrimSuffix(strings.TrimPrefix(p, "/"), ".html")
		if d, err := deck.Parse(filepath.Join(s.dir, filepath.FromSlash(strings.TrimSuffix(strings.TrimPrefix(p, "/"), ".html")+".md"))); err == nil && d.Meta.Title != "" {
			title = d.Meta.Title
		}
		fmt.Fprintf(&b, "<li><b>%s</b>", html.EscapeString(title))
		for _, r := range roles {
			fmt.Fprintf(&b, `<a href="%s?role=%s">%s</a>`, html.EscapeString(p), r, RemoteRoleNames[r])
		}
		b.WriteString("</li>")
	}
	b.WriteString("</ul></body></html>")
	io.WriteString(w, b.String())
}
