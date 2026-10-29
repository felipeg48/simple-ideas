package studio

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStudioServesUIAndDecksBehindGateway(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "talk.md"), []byte("# Talk\n\n---\n\n# One\n\nbody\n"), 0o644)
	s, err := New(dir, Options{Token: "T", Addr: "127.0.0.1:7070"})
	if err != nil {
		t.Fatal(err)
	}
	get := func(path string, cookie bool) (int, string) {
		r := httptest.NewRequest("GET", path, nil)
		r.Host = "127.0.0.1:7070"
		if cookie {
			r.AddCookie(&http.Cookie{Name: CookieName, Value: "T"})
		}
		w := httptest.NewRecorder()
		s.Handler().ServeHTTP(w, r)
		return w.Code, w.Body.String()
	}
	if c, _ := get("/_si/studio/", false); c != 401 {
		t.Fatalf("no cookie: %d", c)
	}
	if c, _ := get("/talk.html", false); c != 401 {
		t.Fatalf("deck no cookie: %d", c)
	}
	if c, b := get("/_si/studio/", true); c != 200 || !strings.Contains(b, `<div id="app"`) {
		t.Fatalf("ui: %d %s", c, b)
	}
	if c, b := get("/talk.html", true); c != 200 || !strings.Contains(b, "si-slide") {
		t.Fatalf("deck: %d", c)
	}
	if c, b := get("/_si/studio/api/tree", true); c != 200 || !strings.Contains(b, "talk.md") {
		t.Fatalf("tree: %d %s", c, b)
	}
	if c, _ := get("/", true); c != 302 {
		t.Fatalf("root: %d", c)
	}
}
