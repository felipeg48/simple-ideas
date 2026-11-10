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
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

// RemoteRoles are the remote page roles, in the order they are listed.
var RemoteRoles = []string{"tp", "draw", "clicker"}

// RemoteRoleNames describes each role for the picker page and the CLI.
var RemoteRoleNames = map[string]string{"tp": "Teleprompter", "draw": "Draw pad", "clicker": "Clicker"}

const remoteCookie = "si_remote"

// audienceCookie carries the audience key: a different key and cookie, so an
// audience phone can never open the teleprompter, the clicker or the draw pad.
const audienceCookie = "si_audience"

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

func (h *relay) events(w http.ResponseWriter, r *http.Request) { h.stream(w, r, nil) }

// audienceEvents is the audience's stream: only where the talk is (slide,
// step, title), never the notes, points, previews, ink or commands the
// presenter's pages exchange.
func (h *relay) audienceEvents(w http.ResponseWriter, r *http.Request) {
	h.stream(w, r, audienceView)
}

// audienceView keeps what an audience page may see of a message, or nil.
func audienceView(raw []byte) []byte {
	var m map[string]any
	if json.Unmarshal(raw, &m) != nil || (m["type"] != "state" && m["type"] != "poll" && m["type"] != "cloud") {
		return nil
	}
	out := map[string]any{}
	for _, k := range []string{"type", "name", "mid", "i", "k", "kmax", "n", "title", "id", "counts", "total", "words"} {
		if v, ok := m[k]; ok {
			out[k] = v
		}
	}
	b, _ := json.Marshal(out)
	return b
}

func (h *relay) stream(w http.ResponseWriter, r *http.Request, filter func([]byte) []byte) {
	fl, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	sseHeaders(w)
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
	pass := func(m []byte) []byte {
		if filter == nil {
			return m
		}
		return filter(m)
	}
	fmt.Fprint(w, "retry: 1000\n: connected\n\n")
	for _, m := range replay {
		if m = pass(m); m != nil {
			fmt.Fprintf(w, "data: %s\n\n", m)
		}
	}
	fl.Flush()
	beat := time.NewTicker(20 * time.Second) // proxies (a --tunnel) close a silent stream
	defer beat.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-beat.C:
			fmt.Fprint(w, ": ping\n\n")
			fl.Flush()
		case m := <-ch:
			if m = pass(m); m != nil {
				fmt.Fprintf(w, "data: %s\n\n", m)
				fl.Flush()
			}
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
	key := ""
	if m.Type == "state" && m.Name != "" {
		key = m.Name
	}
	h.fanout(raw, key, from)
	w.WriteHeader(http.StatusNoContent)
}

// publish sends a message from the server itself (poll counts) to every page;
// keep names it as the message a late page is replayed ("" keeps none).
func (h *relay) publish(raw []byte, keep string) { h.fanout(raw, keep, "") }

func (h *relay) fanout(raw []byte, keep, from string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if keep != "" {
		h.last[keep] = raw
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
	mux.HandleFunc("POST /_si/qa", s.qaState) // a phone teleprompter's Q&A buttons (presenter key)
	mux.HandleFunc("POST /_si/cloud", s.cloudHide)
	static := http.FileServer(http.Dir(s.dir))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		switch {
		case strings.Contains(p, "/."):
			http.NotFound(w, r) // no dotfiles (.git, .env) on the network
		case p == "/" || p == "/index.html":
			s.remotePicker(w, roles)
		case allowed[strings.TrimPrefix(p, "/")]:
			// the short link a QR code carries (/clicker?k=…): the deck page in that
			// role, the key kept in the URL so no cookie is needed on the way
			decks := s.RemoteDecks()
			if len(decks) != 1 {
				http.NotFound(w, r)
				return
			}
			to := decks[0] + "?role=" + strings.TrimPrefix(p, "/") + "&k=" + url.QueryEscape(r.URL.Query().Get("k"))
			if sp := r.URL.Query().Get("speaker"); sp != "" {
				to += "&speaker=" + url.QueryEscape(sp) // a teleprompter for one speaker (speakers:)
			}
			http.Redirect(w, r, to, http.StatusFound)
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
	var aud http.Handler
	if s.AudienceKey != "" {
		aud = s.audienceHandler()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Referrer-Policy", "no-referrer")
		switch {
		case admit(w, r, key, remoteCookie):
			mux.ServeHTTP(w, r)
		case aud != nil && admit(w, r, s.AudienceKey, audienceCookie):
			aud.ServeHTTP(w, r)
		default:
			http.Error(w, "si remote: open the link printed by si play (it carries the key)", http.StatusForbidden)
		}
	})
}

// remoteGate admits a request whose ?k= or cookie matches key (the review
// listener's gate; the remote listener checks two keys itself).
func remoteGate(key string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !admit(w, r, key, remoteCookie) {
			http.Error(w, "si remote: open the link printed by si play (it carries the key)", http.StatusForbidden)
			return
		}
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

// admit reports whether the request carries key, as ?k= (which also sets the
// cookie, so later requests such as images and the relay pass) or as cookie.
func admit(w http.ResponseWriter, r *http.Request, key, cookie string) bool {
	ok := func(v string) bool { return v != "" && subtle.ConstantTimeCompare([]byte(v), []byte(key)) == 1 }
	if ok(r.URL.Query().Get("k")) {
		http.SetCookie(w, &http.Cookie{Name: cookie, Value: key, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode})
		return true
	}
	c, err := r.Cookie(cookie)
	return err == nil && ok(c.Value)
}

// AudienceOnlyHandler is the audience side alone, behind the audience key:
// what si play --tunnel puts on the internet, so nothing of the presenter's
// (teleprompter, clicker, draw pad, controls) is reachable from there.
func (s *Server) AudienceOnlyHandler() http.Handler {
	aud := s.audienceHandler()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Referrer-Policy", "no-referrer")
		if s.AudienceKey == "" || !admit(w, r, s.AudienceKey, audienceCookie) {
			http.Error(w, "si: open the audience link (it carries the key)", http.StatusForbidden)
			return
		}
		aud.ServeHTTP(w, r)
	})
}

// audienceHandler is what an audience phone may reach (si play --audience):
// the deck in the audience role, rendered without Notes: and Points:, its
// files, and the filtered stream of where the talk is. Nothing it sends is
// relayed (polls and questions come with their own endpoints).
func (s *Server) audienceHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/_si/events", s.events)
	mux.HandleFunc("GET /_si/remote/events", s.relay.audienceEvents)
	mux.HandleFunc("GET /_si/audience/now", s.audienceNow)
	mux.HandleFunc("POST /_si/audience/vote", s.vote)
	mux.HandleFunc("POST /_si/audience/ask", s.ask)
	mux.HandleFunc("POST /_si/audience/word", s.word)
	static := http.FileServer(http.Dir(s.dir))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		decks := s.RemoteDecks()
		switch {
		case strings.Contains(p, "/.") || strings.HasPrefix(p, "/_si/"):
			http.NotFound(w, r)
		case p == "/a" || p == "/" || p == "/index.html":
			if len(decks) == 1 { // the code on the slide: straight to the talk
				http.Redirect(w, r, decks[0]+"?role=audience&k="+url.QueryEscape(r.URL.Query().Get("k")), http.StatusFound)
				return
			}
			s.audiencePicker(w, decks)
		case strings.HasSuffix(p, ".html"):
			md := filepath.Join(s.dir, filepath.FromSlash(strings.TrimSuffix(strings.TrimPrefix(p, "/"), ".html")+".md"))
			if r.URL.Query().Get("role") != "audience" || !s.isDeck(md) {
				http.NotFound(w, r)
				return
			}
			s.deckPage(w, md, false, true)
		default:
			static.ServeHTTP(w, r)
		}
	})
	return mux
}

func (s *Server) audiencePicker(w http.ResponseWriter, decks []string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>Follow the talk</title>
<style>body{font:18px/1.5 system-ui,sans-serif;max-width:30rem;margin:0 auto;padding:1.2rem}a{display:block;padding:.8em;margin:.5em 0;border:1px solid #ccc;border-radius:.6em;text-decoration:none;color:inherit}@media (prefers-color-scheme:dark){body{background:#16171b;color:#eee}a{border-color:#444}}</style></head><body><h1>Follow the talk</h1>`)
	for _, d := range decks {
		fmt.Fprintf(w, `<a href="%s?role=audience">%s</a>`, html.EscapeString(d), html.EscapeString(strings.TrimSuffix(strings.TrimPrefix(d, "/"), ".html")))
	}
	fmt.Fprint(w, `</body></html>`)
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

// RemoteSpeakers is the `speakers:` of the one served deck (none for a folder):
// si play --remote prints a teleprompter link per speaker.
func (s *Server) RemoteSpeakers() []string {
	files := []string{s.file}
	if s.file == "" {
		files, _ = deck.List(s.dir)
	}
	if len(files) != 1 {
		return nil
	}
	d, err := deck.Parse(files[0])
	if err != nil {
		return nil
	}
	return d.Meta.SpeakerList()
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

// audienceNow is GET /_si/audience/now: where the talk is and the poll
// counts, as the audience may see them: what an audience page asks about once
// a second when no event stream reaches it (a Cloudflare quick tunnel does
// not carry server-sent events).
func (s *Server) audienceNow(w http.ResponseWriter, r *http.Request) {
	s.relay.mu.Lock()
	var out []json.RawMessage
	for _, m := range s.relay.last {
		if v := audienceView(m); v != nil {
			out = append(out, v)
		}
	}
	s.relay.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	if out == nil {
		out = []json.RawMessage{}
	}
	json.NewEncoder(w).Encode(out)
}

// sseHeaders marks an event stream so no proxy on the way (a Cloudflare
// tunnel, nginx) compresses or buffers it: without no-transform, a tunnel
// held every event back.
func sseHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store, no-transform")
	w.Header().Set("X-Accel-Buffering", "no")
}
