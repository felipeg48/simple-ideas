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
