package serve

// Floating browser windows (`si play`, key w): the deck shows web pages (a
// broker's console, a dashboard) in movable windows over the slides. Many
// consoles forbid framing (X-Frame-Options, CSP frame-ancestors), so each
// target origin gets its own loopback reverse proxy that drops those headers;
// the window frames the proxy. Play-only and loopback-only, like the terminal.

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strconv"
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
		r.Header.Del("Accept-Encoding") // a page as plain text, so navScript can go in
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
		if err := injectNav(res); err != nil {
			return err
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

// navScript goes into every proxied HTML page: it tells the window around it
// where the page is (on load, links, history.pushState/replaceState, back and
// forward) and runs the window's ← → ↻ (history.back/forward, reload), since
// the deck, on another origin, cannot reach the frame's history itself.
const navScript = `<script>(function(){var P=window.parent;if(P===window)return;` +
	`function send(k){try{P.postMessage({si:'nav',kind:k,url:location.href,title:document.title},'*')}catch(e){}}` +
	`['pushState','replaceState'].forEach(function(n){var f=history[n];history[n]=function(){var r=f.apply(this,arguments);send(n==='pushState'?'push':'replace');return r}});` +
	`addEventListener('popstate',function(){send('pop')});addEventListener('hashchange',function(){send('pop')});` +
	`addEventListener('load',function(){send('load')});` +
	`addEventListener('message',function(e){var d=e.data;if(e.source!==P||!d||d.si!=='nav-cmd')return;` +
	`if(d.cmd==='back')history.back();else if(d.cmd==='forward')history.forward();else if(d.cmd==='reload')location.reload()});` +
	`send('load')})();</script>`

var headRE, htmlRE = regexp.MustCompile(`(?i)<head(\s[^>]*)?>`), regexp.MustCompile(`(?i)<html(\s[^>]*)?>`)

// injectNav puts navScript into an HTML response (after <head>, or <html>, or
// first) and drops its CSP so the inline script runs. Other types, and a body
// still encoded (an upstream that ignored the plain-text request), pass as they are.
func injectNav(res *http.Response) error {
	ct := strings.ToLower(res.Header.Get("Content-Type"))
	if !strings.HasPrefix(ct, "text/html") || res.Header.Get("Content-Encoding") != "" || res.Body == nil {
		return nil
	}
	body, err := io.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return err
	}
	at, top := 0, body[:min(len(body), 64<<10)]
	if loc := headRE.FindIndex(top); loc != nil {
		at = loc[1]
	} else if loc := htmlRE.FindIndex(top); loc != nil {
		at = loc[1]
	}
	out := make([]byte, 0, len(body)+len(navScript))
	out = append(append(append(out, body[:at]...), navScript...), body[at:]...)
	res.Body = io.NopCloser(bytes.NewReader(out))
	res.ContentLength = int64(len(out))
	res.Header.Set("Content-Length", strconv.Itoa(len(out)))
	res.Header.Del("Content-Security-Policy")
	res.Header.Del("Content-Security-Policy-Report-Only")
	res.Header.Del("ETag")
	return nil
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
