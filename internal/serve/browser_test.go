package serve

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestBrowserProxyAllowsFraming(t *testing.T) {
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/go" {
			http.Redirect(w, r, "http://"+r.Host+"/api/overview", http.StatusFound)
			return
		}
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; frame-ancestors 'none'")
		w.Header().Add("Set-Cookie", "m=1; Path=/; Domain=example.com; Secure; SameSite=Strict")
		io.WriteString(w, "console "+r.URL.Path)
	}))
	defer up.Close()
	var b browserProxies
	defer b.closeAll()
	u, _ := url.Parse(up.URL + "/x")
	origin, err := b.open(u)
	if err != nil {
		t.Fatal(err)
	}
	if again, _ := b.open(u); again != origin {
		t.Fatalf("one proxy per origin: %s then %s", origin, again)
	}
	res, err := http.Get(origin + "/x")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if string(body) != "console /x" {
		t.Fatalf("body %q", body)
	}
	if res.Header.Get("X-Frame-Options") != "" || strings.Contains(res.Header.Get("Content-Security-Policy"), "frame-ancestors") {
		t.Fatalf("framing headers kept: %v", res.Header)
	}
	if c := res.Header.Get("Set-Cookie"); c != "m=1; Path=/" {
		t.Fatalf("cookie %q", c)
	}
	noFollow := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	res, err = noFollow.Get(origin + "/go")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if loc := res.Header.Get("Location"); !strings.HasPrefix(loc, origin+"/api/overview") {
		t.Fatalf("redirect not kept on the proxy: %q", loc)
	}
}

// TestBrowserProxyInjectsNav: an HTML page gets the nav script (for the
// window's back / forward) right after <head>, its CSP goes, and the upstream
// is asked for a plain body; other types pass untouched.
func TestBrowserProxyInjectsNav(t *testing.T) {
	var enc []string
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		enc = append(enc, r.Header.Get("Accept-Encoding"))
		if r.URL.Path == "/app.js" {
			w.Header().Set("Content-Type", "text/javascript")
			io.WriteString(w, "var x = '<head>';")
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Security-Policy", "script-src 'self'")
		io.WriteString(w, `<!doctype html><html><HEAD lang="en"><title>A</title></head><body><a href="/b">b</a></body></html>`)
	}))
	defer up.Close()
	var b browserProxies
	defer b.closeAll()
	u, _ := url.Parse(up.URL + "/")
	origin, err := b.open(u)
	if err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequest("GET", origin+"/", nil)
	req.Header.Set("Accept-Encoding", "br") // what a browser sends
	res, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if !strings.HasPrefix(string(body), `<!doctype html><html><HEAD lang="en"><script>`) || !strings.Contains(string(body), "si:'nav'") {
		t.Fatalf("nav script not after <head>: %.120s", body)
	}
	if res.Header.Get("Content-Security-Policy") != "" {
		t.Errorf("CSP kept on HTML: %q", res.Header.Get("Content-Security-Policy"))
	}
	if strings.Contains(enc[0], "br") {
		t.Errorf("upstream asked for %q, want a plain (or gzip, decoded by the proxy) body", enc[0])
	}
	res, err = http.Get(origin + "/app.js")
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(res.Body)
	res.Body.Close()
	if string(body) != "var x = '<head>';" {
		t.Errorf("a script was changed: %q", body)
	}
}

// TestBrowserProxyAcrossSites: a redirect to another origin goes through that
// origin's own proxy (started then), one to the same origin stays on its
// proxy, a relative one is left alone; each page's script knows its real
// origin and hands cross-origin link clicks to the window.
func TestBrowserProxyAcrossSites(t *testing.T) {
	two := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		io.WriteString(w, "<html><head></head><body>two "+r.URL.Path+"</body></html>")
	}))
	defer two.Close()
	one := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/away":
			http.Redirect(w, r, two.URL+"/landed?x=1", http.StatusFound)
		case "/rel":
			w.Header().Set("Location", "/here")
			w.WriteHeader(http.StatusFound)
		default:
			w.Header().Set("Content-Type", "text/html")
			io.WriteString(w, `<html><head></head><body><a href="`+two.URL+`/b">two</a></body></html>`)
		}
	}))
	defer one.Close()
	var b browserProxies
	defer b.closeAll()
	u1, _ := url.Parse(one.URL)
	o1, err := b.open(u1)
	if err != nil {
		t.Fatal(err)
	}
	noFollow := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	res, err := noFollow.Get(o1 + "/away")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	u2, _ := url.Parse(two.URL)
	o2, _ := b.open(u2)
	if o2 == o1 {
		t.Fatal("one proxy for two origins")
	}
	if loc := res.Header.Get("Location"); loc != o2+"/landed?x=1" {
		t.Fatalf("cross-site redirect %q, want %q", loc, o2+"/landed?x=1")
	}
	res, _ = noFollow.Get(o1 + "/rel")
	res.Body.Close()
	if loc := res.Header.Get("Location"); loc != "/here" {
		t.Errorf("relative redirect changed: %q", loc)
	}
	res, err = http.Get(o2 + "/landed")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if !strings.Contains(string(body), `var T="`+two.URL+`"`) || !strings.Contains(string(body), "real:real()") || !strings.Contains(string(body), "two /landed") {
		t.Fatalf("page on the second proxy lacks its real origin: %.300s", body)
	}
	for _, want := range []string{"window.open=", "'submit'", "navigation.addEventListener('navigate'"} {
		if !strings.Contains(navScript("x", "tok"), want) {
			t.Errorf("navScript lacks %s", want)
		}
	}
	if !strings.Contains(navScript("x", "tok"), "nav-open") {
		t.Error("the nav script does not hand cross-origin links to the window")
	}
	if strings.Contains(navScript("http://a/</script>", "tok"), "</script>\"") {
		t.Error("a target can end the script")
	}
}

// TestBrowserProxyForwardsPost: a POST form to another site, pointed by the
// nav script at its page's proxy's fwdPath, reaches the other site through
// that site's own proxy with its method and fields.
func TestBrowserProxyForwardsPost(t *testing.T) {
	two := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		io.WriteString(w, r.Method+" "+r.URL.Path+" user="+r.PostForm.Get("user")+" pass="+r.PostForm.Get("pass"))
	}))
	defer two.Close()
	one := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "one") }))
	defer one.Close()
	var b browserProxies
	defer b.closeAll()
	u, _ := url.Parse(one.URL + "/")
	oneProxy, err := b.open(u)
	if err != nil {
		t.Fatal(err)
	}
	res, err := http.PostForm(oneProxy+fwdPath+"?t="+b.fwdToken()+"&to="+url.QueryEscape(two.URL+"/login?next=/home"), url.Values{"user": {"ana"}, "pass": {"s3cret"}})
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if string(body) != "POST /login user=ana pass=s3cret" {
		t.Fatalf("forwarded: %q", body)
	}
	tu, _ := url.Parse(two.URL)
	twoProxy, _ := b.open(tu)
	if !strings.HasPrefix(res.Request.URL.String(), twoProxy+"/login?next=") {
		t.Errorf("not through the other site's proxy: %s (its proxy %s)", res.Request.URL, twoProxy)
	}
	for _, bad := range []string{"", "javascript:alert(1)", "/relative"} {
		r, err := http.Get(oneProxy + fwdPath + "?t=" + b.fwdToken() + "&to=" + url.QueryEscape(bad))
		if err != nil {
			t.Fatal(err)
		}
		r.Body.Close()
		if r.StatusCode != http.StatusBadRequest {
			t.Errorf("to=%q: status %d", bad, r.StatusCode)
		}
	}
	if s := navScript("http://x", "tok"); !strings.Contains(s, fwdPath+"?t=tok&to=") || !strings.Contains(s, "HTMLFormElement.prototype.submit") {
		t.Error("nav script does not point POST forms at the forward")
	}
	// without this run's key the forwarder serves nobody (another page in the
	// browser that guesses the port must not read other sites through it)
	for _, q := range []string{"?to=", "?t=wrong&to=", "?x=1&to=", "?t=wrong&x=1&to="} {
		r, err := http.Get(oneProxy + fwdPath + q + url.QueryEscape(two.URL+"/login"))
		if err != nil {
			t.Fatal(err)
		}
		r.Body.Close()
		if r.StatusCode != http.StatusForbidden {
			t.Errorf("%s without the key: status %d, want 403", q, r.StatusCode)
		}
	}
}
