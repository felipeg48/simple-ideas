package studio

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const tok = "secret-token"

func gw() http.Handler {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	return NewGateway(tok, []string{"127.0.0.1:7070", "localhost:7070"}).Wrap(next)
}

func do(method, target, host string, mod func(*http.Request)) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, target, nil)
	r.Host = host
	if mod != nil {
		mod(r)
	}
	w := httptest.NewRecorder()
	gw().ServeHTTP(w, r)
	return w
}

func withCookie(v string) func(*http.Request) {
	return func(r *http.Request) { r.AddCookie(&http.Cookie{Name: CookieName, Value: v}) }
}

func TestGatewayRejectsForeignHost(t *testing.T) {
	if w := do("GET", "/", "evil.com:7070", withCookie(tok)); w.Code != 403 {
		t.Fatalf("got %d, want 403", w.Code)
	}
}

func TestGatewayLoginSetsCookieAndRedirects(t *testing.T) {
	w := do("GET", "/?token="+tok, "127.0.0.1:7070", nil)
	if w.Code != 303 || w.Header().Get("Location") != "/" {
		t.Fatalf("got %d to %q", w.Code, w.Header().Get("Location"))
	}
	c := w.Header().Get("Set-Cookie")
	for _, want := range []string{CookieName + "=" + tok, "HttpOnly", "SameSite=Strict", "Path=/"} {
		if !strings.Contains(c, want) {
			t.Errorf("cookie %q lacks %q", c, want)
		}
	}
	// other query params survive the redirect
	w = do("GET", "/x?a=1&token="+tok, "localhost:7070", nil)
	if loc := w.Header().Get("Location"); loc != "/x?a=1" {
		t.Errorf("location %q", loc)
	}
}

func TestGatewayRequiresCookie(t *testing.T) {
	const p = "/_si/studio/api/tree"
	if w := do("GET", p, "127.0.0.1:7070", nil); w.Code != 401 {
		t.Errorf("no cookie: %d", w.Code)
	}
	if w := do("GET", p, "127.0.0.1:7070", withCookie("nope")); w.Code != 401 {
		t.Errorf("wrong cookie: %d", w.Code)
	}
	if w := do("GET", p, "127.0.0.1:7070", withCookie(tok)); w.Code != 200 {
		t.Errorf("right cookie: %d", w.Code)
	}
	if w := do("GET", p+"?token=wrong", "127.0.0.1:7070", nil); w.Code != 401 {
		t.Errorf("wrong token: %d", w.Code)
	}
}

func TestGatewayRejectsForeignOrigin(t *testing.T) {
	origin := func(o string) func(*http.Request) {
		return func(r *http.Request) { withCookie(tok)(r); r.Header.Set("Origin", o) }
	}
	if w := do("PUT", "/_si/studio/api/file", "127.0.0.1:7070", origin("http://evil.com")); w.Code != 403 {
		t.Errorf("foreign origin: %d", w.Code)
	}
	if w := do("PUT", "/_si/studio/api/file", "127.0.0.1:7070", origin("http://127.0.0.1:7070")); w.Code != 200 {
		t.Errorf("same origin: %d", w.Code)
	}
}

func TestGatewayAssetsArePublic(t *testing.T) {
	if w := do("GET", "/_si/studio/assets/app.js", "127.0.0.1:7070", nil); w.Code != 200 {
		t.Fatalf("got %d", w.Code)
	}
}

func TestNewTokenIsRandom(t *testing.T) {
	a, b := NewToken(), NewToken()
	if a == b || len(a) != 64 {
		t.Fatalf("%q %q", a, b)
	}
}

func TestGatewayAssetsBypassRejectsTraversal(t *testing.T) {
	for _, p := range []string{
		"/_si/studio/assets/../api/tree",
		"/_si/studio/assets/%2e%2e/api/tree",
		"/_si/studio/assets/%2E%2E/api/tree",
		"/_si/studio/assets/./../api/tree",
	} {
		if w := do("GET", p, "127.0.0.1:7070", nil); w.Code != 401 {
			t.Errorf("%s: got %d, want 401", p, w.Code)
		}
	}
}
