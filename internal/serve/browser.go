package serve

// Floating browser windows (`si play`, key w): the deck shows web pages (a
// broker's console, a dashboard) in movable windows over the slides. Many
// consoles forbid framing (X-Frame-Options, CSP frame-ancestors), so each
// target origin gets its own loopback reverse proxy that drops those headers;
// the window frames the proxy. Play-only and loopback-only, like the terminal.
// A page that sends the window to another site keeps it proxied: a redirect's
// Location to another http(s) origin goes through that origin's own proxy
// (started then), and the script put in each page hands cross-origin link
// clicks to the window, which opens them through /_si/browser/open.

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
		if err := injectNav(res, key); err != nil {
			return err
		}
		if loc := h.Get("Location"); loc != "" {
			h.Set("Location", b.through(loc, key, origin))
		}
		if cs := h.Values("Set-Cookie"); len(cs) > 0 {
			h.Del("Set-Cookie")
			for _, c := range cs {
				h.Add("Set-Cookie", cookieAttrRE.ReplaceAllString(c, ""))
			}
		}
		return nil
	}
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == fwdPath {
			b.forward(w, r, key)
			return
		}
		rp.ServeHTTP(w, r)
	})}
	go srv.Serve(ln)
	if b.byHost == nil {
		b.byHost = map[string]string{}
	}
	b.byHost[key] = origin
	b.servers = append(b.servers, srv)
	return origin, nil
}

// through maps an absolute redirect Location onto the proxy of its origin: the
// same proxy for this target (key -> origin), another proxy (started now) for
// another http(s) origin. A relative Location, or one that does not parse,
// stays as it is.
func (b *browserProxies) through(loc, key, origin string) string {
	u, err := url.Parse(loc)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return loc
	}
	o := origin
	if u.Scheme+"://"+u.Host != key {
		if o, err = b.open(u); err != nil {
			return loc
		}
	}
	u.Scheme, u.Host = "", ""
	return o + u.String()
}

// fwdPath, on every proxy, sends a request on to another origin's proxy:
// ?to=<real address> answers 307 to that address through its proxy (started
// then), so a POST form to another site arrives there with its fields (a 307
// keeps the method and the body). navScript points such forms here.
const fwdPath = "/__si_fwd"

func (b *browserProxies) forward(w http.ResponseWriter, r *http.Request, key string) {
	u, err := url.Parse(r.URL.Query().Get("to"))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		http.Error(w, "an http(s) address, please", http.StatusBadRequest)
		return
	}
	if r.URL.Query().Get("x") != "" {
		fetchThrough(w, r, u, key)
		return
	}
	o, err := b.open(u)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	u.Scheme, u.Host = "", ""
	http.Redirect(w, r, o+u.String(), http.StatusTemporaryRedirect)
}

// fetchThrough is fwdPath with ?x=1: a page's own fetch or XMLHttpRequest
// to another site (navScript points it here), sent on from this proxy, so
// the page sees a same-origin answer and no CORS rule of the proxy's origin
// gets in the way. The other site gets the request as the page's real origin
// (key) sent it, minus cookies: this proxy's cookies belong to the framed
// site, and the other site's own live on its proxy's origin, out of reach;
// its Set-Cookie is dropped for the same reason. A redirect it answers comes
// back through fwdPath, so fetch follows it here too.
func fetchThrough(w http.ResponseWriter, r *http.Request, to *url.URL, key string) {
	rp := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			u := *to
			pr.Out.URL, pr.Out.Host = &u, to.Host
			pr.Out.Header.Del("Cookie")
			pr.Out.Header.Del("Referer")
			if pr.In.Header.Get("Origin") != "" {
				pr.Out.Header.Set("Origin", key)
			}
		},
		ModifyResponse: func(res *http.Response) error {
			res.Header.Del("Set-Cookie")
			if loc := res.Header.Get("Location"); loc != "" {
				if l, err := to.Parse(loc); err == nil && (l.Scheme == "http" || l.Scheme == "https") {
					res.Header.Set("Location", fwdPath+"?x=1&to="+url.QueryEscape(l.String()))
				}
			}
			return nil
		},
	}
	rp.ServeHTTP(w, r)
}

// navScript goes into every proxied HTML page: it tells the window around it
// where the page is (on load, links, history.pushState/replaceState, back and
// forward), with its real address (the target origin T in place of the
// proxy's), runs the window's ← → ↻ (history.back/forward, reload), since
// the deck, on another origin, cannot reach the frame's history itself, and
// hands a navigation to another origin to the window ({si:'nav-open'}), which
// opens it through that origin's proxy: a link click; window.open; a GET form;
// and, where the browser has the Navigation API (its navigate event: Chrome,
// Edge, recent Safari and Firefox), location.assign/replace and location.href
// =, each cancelled and reopened. location's own methods cannot be replaced
// (the browser forbids it), so without that API a script that sets location
// still leaves the proxy. A POST form to another site (submitted, or sent by
// form.submit()) keeps its fields: its action (or its button's formaction) is
// pointed at this proxy's fwdPath, which hands it, body and all, to the other
// site's proxy. Its own fetch and XMLHttpRequest calls to an absolute
// http(s) address on another origin go to fwdPath?x=1 (fetchThrough), and
// to its own site's address to this proxy.
func navScript(target string) string {
	t, _ := json.Marshal(target) // escapes <, so it cannot end the script
	return `<script>(function(){var P=window.parent;if(P===window)return;var T=` + string(t) + `;` +
		`function real(){var o=location.origin;return location.href.indexOf(o)===0?T+location.href.slice(o.length):location.href}` +
		`function send(k){try{P.postMessage({si:'nav',kind:k,url:location.href,real:real(),title:document.title},'*')}catch(e){}}` +
		`function away(h){var u;try{u=new URL(String(h),location.href)}catch(x){return false}` +
		`if(!/^https?:$/.test(u.protocol)||u.origin===location.origin)return false;try{P.postMessage({si:'nav-open',url:u.href},'*')}catch(x){}return true}` +
		`addEventListener('click',function(e){if(e.defaultPrevented||e.button!==0||e.metaKey||e.ctrlKey||e.shiftKey||e.altKey)return;` +
		`var a=e.target.closest&&e.target.closest('a[href]');if(a&&away(a.href))e.preventDefault()});` +
		`var wo=window.open;window.open=function(u){if(u!=null&&u!==''&&away(u))return null;return wo.apply(window,arguments)};` +
		// fetch and XMLHttpRequest to an absolute http(s) address elsewhere:
		// the framed site's own origin to this proxy, another through fwdPath
		`function via(h){var u;try{u=new URL(String(h),location.href)}catch(x){return null}if(!/^https?:$/.test(u.protocol)||u.origin===location.origin)return null;` +
		`if(u.origin===T)return location.origin+u.pathname+u.search+u.hash;return location.origin+'` + fwdPath + `?x=1&to='+encodeURIComponent(u.href)}` +
		`var F=window.fetch;if(F)window.fetch=function(i,o){var R=window.Request&&i instanceof Request,v=via(R?i.url:i);if(!v)return F.apply(window,arguments);if(!R)return F.call(window,v,o);` +
		`return(/^(GET|HEAD)$/.test(i.method)?Promise.resolve(void 0):i.blob()).then(function(b){return F.call(window,new Request(v,{method:i.method,headers:i.headers,body:b,credentials:i.credentials,cache:i.cache,redirect:i.redirect,integrity:i.integrity,keepalive:i.keepalive,signal:i.signal}),o)})};` +
		`var XO=XMLHttpRequest.prototype.open;XMLHttpRequest.prototype.open=function(m,h){var v=via(h);if(!v)return XO.apply(this,arguments);var a=[].slice.call(arguments);a[1]=v;return XO.apply(this,a)};` +
		`function meth(f,s){return((s&&s.getAttribute('formmethod'))||f.getAttribute('method')||'get').toLowerCase()}` +
		`function post(f,s){var fa=s&&s.hasAttribute('formaction'),u;try{u=new URL(fa?s.formAction:f.action,location.href)}catch(x){return}` +
		`if(!/^https?:$/.test(u.protocol)||u.origin===location.origin)return;var to=location.origin+'` + fwdPath + `?to='+encodeURIComponent(u.href);if(fa)s.formAction=to;else f.action=to}` +
		`var fs=HTMLFormElement.prototype.submit;HTMLFormElement.prototype.submit=function(){if(meth(this)==='post')post(this);return fs.apply(this,arguments)};` +
		`addEventListener('submit',function(e){var f=e.target,s=e.submitter,m=meth(f,s);if(e.defaultPrevented)return;if(m==='post'){post(f,s);return}if(m!=='get')return;` +
		`var u;try{u=new URL(f.action,location.href)}catch(x){return}u.search=new URLSearchParams(new FormData(f)).toString();if(away(u.href))e.preventDefault()});` +
		`if(window.navigation)navigation.addEventListener('navigate',function(e){if(!e.cancelable||e.navigationType==='traverse'||e.navigationType==='reload')return;` +
		`var u;try{u=new URL(e.destination.url)}catch(x){return}if(u.origin!==location.origin&&away(u.href))e.preventDefault()});` +
		`['pushState','replaceState'].forEach(function(n){var f=history[n];history[n]=function(){var r=f.apply(this,arguments);send(n==='pushState'?'push':'replace');return r}});` +
		`addEventListener('popstate',function(){send('pop')});addEventListener('hashchange',function(){send('pop')});` +
		`addEventListener('load',function(){send('load')});` +
		`addEventListener('message',function(e){var d=e.data;if(e.source!==P||!d||d.si!=='nav-cmd')return;` +
		`if(d.cmd==='back')history.back();else if(d.cmd==='forward')history.forward();else if(d.cmd==='reload')location.reload()});` +
		`send('load')})();</script>`
}

var headRE, htmlRE = regexp.MustCompile(`(?i)<head(\s[^>]*)?>`), regexp.MustCompile(`(?i)<html(\s[^>]*)?>`)

// injectNav puts navScript into an HTML response (after <head>, or <html>, or
// first) and drops its CSP so the inline script runs. Other types, and a body
// still encoded (an upstream that ignored the plain-text request), pass as they are.
// target is the real origin the page comes from.
func injectNav(res *http.Response, target string) error {
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
	nav := navScript(target)
	out := make([]byte, 0, len(body)+len(nav))
	out = append(append(append(out, body[:at]...), nav...), body[at:]...)
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
