package serve

// Floating browser windows (`si play`, key w): the deck shows web pages (a
// broker's console, a dashboard) in movable windows over the slides. Many
// consoles forbid framing (X-Frame-Options, CSP frame-ancestors), so each
// target origin gets its own loopback reverse proxy that drops those headers;
// the window frames the proxy. Play-only and loopback-only, like the terminal.

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strings"
	"sync"
)

type browserProxies struct {
	mu      sync.Mutex
	byHost  map[string]string // target origin -> proxy origin
	servers []*http.Server
}

var cookieAttrRE = regexp.MustCompile(`(?i);\s*(domain=[^;]*|secure|samesite=[^;]*)`)

// open returns the proxy origin for the target's origin, starting it once.
func (b *browserProxies) open(target *url.URL) (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	key := target.Scheme + "://" + target.Host
	if o, ok := b.byHost[key]; ok {
		return o, nil
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	origin := "http://" + ln.Addr().String()
	up := &url.URL{Scheme: target.Scheme, Host: target.Host}
	rp := httputil.NewSingleHostReverseProxy(up)
	base := rp.Director
	rp.Director = func(r *http.Request) {
		base(r)
		r.Host = up.Host
		r.Header.Del("Origin") // the console sees itself, not the proxy
		r.Header.Del("Referer")
	}
	rp.ModifyResponse = func(res *http.Response) error {
		h := res.Header
		h.Del("X-Frame-Options")
		if csp := h.Get("Content-Security-Policy"); csp != "" {
			var keep []string
			for _, d := range strings.Split(csp, ";") {
				if !strings.HasPrefix(strings.TrimSpace(strings.ToLower(d)), "frame-ancestors") {
					keep = append(keep, d)
				}
			}
			h.Set("Content-Security-Policy", strings.Join(keep, ";"))
		}
		if loc := h.Get("Location"); strings.HasPrefix(loc, key) {
			h.Set("Location", origin+strings.TrimPrefix(loc, key))
		}
		if cs := h.Values("Set-Cookie"); len(cs) > 0 {
			h.Del("Set-Cookie")
			for _, c := range cs {
				h.Add("Set-Cookie", cookieAttrRE.ReplaceAllString(c, ""))
			}
		}
		return nil
	}
	srv := &http.Server{Handler: rp}
	go srv.Serve(ln)
	if b.byHost == nil {
		b.byHost = map[string]string{}
	}
	b.byHost[key] = origin
	b.servers = append(b.servers, srv)
	return origin, nil
}

func (b *browserProxies) closeAll() {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, s := range b.servers {
		s.Close()
	}
	b.servers, b.byHost = nil, nil
}

// browserOpen: POST {"url": "http://localhost:15672/"} -> {"url": "<proxied>"}.
func (s *Server) browserOpen(w http.ResponseWriter, r *http.Request) {
	var in struct{ URL string }
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<14)).Decode(&in); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	u, err := url.Parse(strings.TrimSpace(in.URL))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		http.Error(w, "an http(s) address, please", http.StatusBadRequest)
		return
	}
	origin, err := s.browser.open(u)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	out := origin + u.EscapedPath()
	if u.RawQuery != "" {
		out += "?" + u.RawQuery
	}
	if u.Fragment != "" {
		out += "#" + u.EscapedFragment()
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"url": out})
}
