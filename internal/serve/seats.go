package serve

import (
	"context"
	"fmt"
	"html"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/felipeg48/simple-ideas/internal/assets"
	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/felipeg48/simple-ideas/internal/sandbox"
)

// A class on the presenter's machine (si class start, RULES §37): learners
// open a seat link (/go/CODE) through --tunnel or HTTPS and get the course
// with a terminal drawer whose shells run only in their own box. This is the
// one listener off the loopback that has a terminal: rule #8's exception for
// classes. It is its own mux, an allow-list, never Handler(): on a LAN a
// request can name any Host, so the loopback guards of Handler() do not hold
// here.

const (
	seatCookie   = "si_seat"
	seatWait     = 90 * time.Second // a new tab waits this long for the box
	joinFailsMax = 60               // wrong codes per minute, all learners together
)

// Classroom is a running class: its seats, their boxes and drawers.
type Classroom struct {
	Fleet *sandbox.Fleet
	// Joined is called on a seat's first /go (the class file is saved).
	Joined func(code string)

	mu    sync.Mutex
	class *sandbox.Class
	hubs  map[string]*termHub
	fails []time.Time
}

// NewClassroom runs class c on fleet f.
func NewClassroom(c *sandbox.Class, f *sandbox.Fleet) *Classroom {
	return &Classroom{class: c, Fleet: f, hubs: map[string]*termHub{}}
}

// Update changes the class (seats added or removed by the planner) under the
// classroom's lock.
func (c *Classroom) Update(fn func(*sandbox.Class)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	fn(c.class)
}

// View reads the class under the lock.
func (c *Classroom) View(fn func(*sandbox.Class)) { c.Update(fn) }

// Reset removes a seat's box and closes its shells; the next tab makes a
// fresh box from the image.
func (c *Classroom) Reset(code string) {
	c.mu.Lock()
	h := c.hubs[code]
	delete(c.hubs, code)
	c.mu.Unlock()
	if h != nil {
		h.closeAll()
	}
	c.Fleet.Reset(code)
}

// Close closes every shell and removes every box.
func (c *Classroom) Close() {
	c.mu.Lock()
	hubs := c.hubs
	c.hubs = map[string]*termHub{}
	c.mu.Unlock()
	for _, h := range hubs {
		h.closeAll()
	}
	c.Fleet.Close()
}

// seat is the request's seat code from its cookie, "" when it has none that
// the class still has.
func (c *Classroom) seat(r *http.Request) string {
	k, err := r.Cookie(seatCookie)
	if err != nil {
		return ""
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.class.Seat(k.Value); !ok || c.class.Expired(time.Now()) {
		return ""
	}
	return sandbox.NormalCode(k.Value)
}

// hub is the seat's drawer: every tab is a shell in the seat's box.
func (c *Classroom) hub(code string) *termHub {
	c.mu.Lock()
	defer c.mu.Unlock()
	h := c.hubs[code]
	if h == nil {
		h = newTermHub("/")
		h.need = func() (*sandbox.Box, error) {
			ctx, cancel := context.WithTimeout(context.Background(), seatWait)
			defer cancel()
			return c.Fleet.Get(ctx, code)
		}
		c.hubs[code] = h
	}
	return h
}

// box is the seat's box when it is ready (nil while it starts).
func (c *Classroom) box(code string) *sandbox.Box {
	ctx, cancel := context.WithTimeout(context.Background(), seatWait)
	defer cancel()
	b, _ := c.Fleet.Get(ctx, code)
	return b
}

// ClassroomHandler is what learners reach: the seat links, the course pages
// with a drawer in their own box, the page checks in that box, its ports,
// and the class events. Nothing else of the server.
func (s *Server) ClassroomHandler(c *Classroom) http.Handler {
	mux := http.NewServeMux()
	pages := s.pageRoot(true)
	mux.HandleFunc("GET /go/{code}", c.join)
	mux.HandleFunc("GET /go", c.join)
	mux.Handle("GET /_si/term/assets/", http.StripPrefix("/_si/term/assets/", http.FileServer(http.FS(assets.TermFS))))
	type seatFunc func(w http.ResponseWriter, r *http.Request, code string)
	seat := func(post bool, h seatFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			code := c.seat(r)
			if code == "" {
				if strings.HasPrefix(r.URL.Path, "/_si/") {
					http.Error(w, "open your seat link first", http.StatusForbidden)
				} else {
					joinPage(w, http.StatusUnauthorized, "")
				}
				return
			}
			if post && !sameOrigin(r) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			w.Header().Set("Referrer-Policy", "no-referrer")
			c.Fleet.Begin(code)
			defer c.Fleet.End(code)
			h(w, r, code)
		}
	}
	term := func(f func(*termHub) http.HandlerFunc) seatFunc {
		return func(w http.ResponseWriter, r *http.Request, code string) { f(c.hub(code))(w, r) }
	}
	mux.HandleFunc("POST /_si/term/api/new", seat(true, term(func(h *termHub) http.HandlerFunc { return h.newTab })))
	mux.HandleFunc("GET /_si/term/api/out", seat(false, term(func(h *termHub) http.HandlerFunc { return h.out })))
	mux.HandleFunc("GET /_si/term/api/read", seat(false, term(func(h *termHub) http.HandlerFunc { return h.read })))
	mux.HandleFunc("GET /_si/term/api/list", seat(false, term(func(h *termHub) http.HandlerFunc { return h.list })))
	mux.HandleFunc("POST /_si/term/api/in", seat(true, term(func(h *termHub) http.HandlerFunc { return h.in })))
	mux.HandleFunc("POST /_si/term/api/resize", seat(true, term(func(h *termHub) http.HandlerFunc { return h.resize })))
	mux.HandleFunc("POST /_si/term/api/close", seat(true, term(func(h *termHub) http.HandlerFunc { return h.close })))
	mux.HandleFunc("POST /_si/check", seat(true, func(w http.ResponseWriter, r *http.Request, code string) {
		b := c.box(code)
		if b == nil {
			http.Error(w, "your sandbox is not running yet", http.StatusServiceUnavailable)
			return
		}
		s.checkIn(w, r, func(*deck.Deck) *sandbox.Box { return b })
	}))
	mux.HandleFunc("GET /_si/sandbox", seat(false, c.state))
	mux.HandleFunc("/_si/sandbox/port/{port}/", seat(false, c.port))
	mux.HandleFunc("POST /_si/class/event", seat(true, func(w http.ResponseWriter, r *http.Request, _ string) { s.classEvent(w, r) }))
	mux.HandleFunc("GET /_si/events", seat(false, func(w http.ResponseWriter, r *http.Request, _ string) { s.eventsAs(w, r, "learner") }))
	mux.HandleFunc("/", seat(false, func(w http.ResponseWriter, r *http.Request, _ string) {
		p := path.Clean("/" + r.URL.Path)
		switch ext := path.Ext(p); {
		case strings.HasPrefix(p, "/_si/") || strings.Contains(p, "/.") || strings.Contains(p, "/_"),
			ext == ".md" || ext == ".adoc" || ext == ".asciidoc" || ext == ".conf":
			http.NotFound(w, r) // the sources (solutions, the instructor's notes), si.conf, .si/ (the seat codes)
		default:
			pages(w, r)
		}
	}))
	return mux
}

// join is GET /go/{code} (or /go?code=): the seat's cookie, then the course.
func (c *Classroom) join(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	if code == "" {
		code = r.URL.Query().Get("code")
	}
	now := time.Now()
	c.mu.Lock()
	keep := c.fails[:0]
	for _, t := range c.fails {
		if now.Sub(t) < time.Minute {
			keep = append(keep, t)
		}
	}
	c.fails = keep
	if len(c.fails) >= joinFailsMax {
		c.mu.Unlock()
		http.Error(w, "too many wrong codes: wait a minute", http.StatusTooManyRequests)
		return
	}
	st, ok := c.class.Seat(code)
	expired := c.class.Expired(now)
	first := ok && st.Joined.IsZero()
	if ok && first {
		st.Joined = now
	}
	if !ok {
		c.fails = append(c.fails, now)
	}
	c.mu.Unlock()
	switch {
	case code == "":
		joinPage(w, http.StatusOK, "")
		return
	case !ok:
		joinPage(w, http.StatusNotFound, "That code is not a seat of this class.")
		return
	case expired:
		joinPage(w, http.StatusGone, "This class is over.")
		return
	}
	code = st.Code
	if first && c.Joined != nil {
		c.Joined(code)
	}
	go c.Fleet.Get(context.Background(), code) // start the box now: it is ready by the first tab
	http.SetCookie(w, &http.Cookie{Name: seatCookie, Value: code, Path: "/", MaxAge: 7 * 24 * 3600, HttpOnly: true,
		SameSite: http.SameSiteLaxMode, Secure: r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"})
	w.Header().Set("Referrer-Policy", "no-referrer")
	http.Redirect(w, r, "/", http.StatusFound)
}

// state is GET /_si/sandbox for a seat: its box as /_si/sandbox says it in
// si play, and how far its start is.
func (c *Classroom) state(w http.ResponseWriter, r *http.Request, code string) {
	type port struct {
		Port  int    `json:"port"`
		Label string `json:"label"`
	}
	st := c.Fleet.States()[code]
	out := struct {
		Running bool   `json:"running"`
		State   string `json:"state"`
		Step    string `json:"step,omitempty"`
		Image   string `json:"image"`
		Ports   []port `json:"ports"`
	}{State: st.State, Step: st.Step, Image: c.Fleet.Spec.Image, Ports: []port{}}
	out.Running = st.State == "ready"
	for _, p := range c.Fleet.Spec.Ports {
		out.Ports = append(out.Ports, port{p.Port, p.Label})
	}
	writeJSON(w, out)
}

// port is /_si/sandbox/port/{port}/…: the seat's box port, proxied (the box
// publishes it on the sandbox host's loopback only, which learners cannot
// reach).
func (c *Classroom) port(w http.ResponseWriter, r *http.Request, code string) {
	n, _ := strconv.Atoi(r.PathValue("port"))
	b := c.box(code)
	if b == nil {
		http.Error(w, "your sandbox is not running yet", http.StatusServiceUnavailable)
		return
	}
	hp, ok := b.Ports[n]
	if !ok {
		http.Error(w, "the lab's sandbox-ports: has no port "+strconv.Itoa(n), http.StatusNotFound)
		return
	}
	prefix := "/_si/sandbox/port/" + strconv.Itoa(n)
	rp := &httputil.ReverseProxy{Rewrite: func(pr *httputil.ProxyRequest) {
		pr.SetURL(&url.URL{Scheme: "http", Host: hp.Addr})
		pr.Out.URL.Path = strings.TrimPrefix(r.URL.Path, prefix)
		pr.Out.URL.RawPath = ""
		pr.Out.Host = hp.Addr
		var keep []string // the app's own cookies, never the seat's
		for _, k := range pr.In.Cookies() {
			if k.Name != seatCookie {
				keep = append(keep, k.Name+"="+k.Value)
			}
		}
		pr.Out.Header.Del("Cookie")
		if len(keep) > 0 {
			pr.Out.Header.Set("Cookie", strings.Join(keep, "; "))
		}
	}}
	rp.ServeHTTP(w, r)
}

// joinPage asks for a seat code.
func joinPage(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if msg != "" {
		msg = `<p class="m">` + html.EscapeString(msg) + `</p>`
	}
	fmt.Fprintf(w, `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>Join the class</title>
<style>body{font:18px/1.5 system-ui,sans-serif;max-width:26rem;margin:0 auto;padding:2rem 1.2rem;background:#fafaf9;color:#1c1c1e}h1{font-size:1.5rem}input,button{font:inherit;padding:.6em .8em;border-radius:.5em;border:1px solid #bbb}input{width:100%%;box-sizing:border-box;letter-spacing:.15em;text-transform:uppercase}button{margin-top:.8em;background:#2457d6;color:#fff;border:0;width:100%%}.m{color:#b42318}@media (prefers-color-scheme:dark){body{background:#16171b;color:#eee}input{background:#222;color:#eee;border-color:#444}.m{color:#ff8a80}}</style></head>
<body><h1>Join the class</h1>%s<p>Type the seat code your instructor gave you.</p><form action="/go" method="get"><input name="code" autocomplete="off" autofocus placeholder="ABCD-EFGH" aria-label="Seat code"><button>Join</button></form></body></html>`, msg)
}
