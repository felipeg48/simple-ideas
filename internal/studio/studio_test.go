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
			r.AddCookie(&http.Cookie{Name: "si_studio_7070", Value: "T"})
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
	// the UI is told its workspace id, so localStorage keys never mix folders
	_, b := get("/_si/studio/", true)
	other, err := New(t.TempDir(), Options{Token: "T", Addr: "127.0.0.1:7070"})
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("GET", "/_si/studio/", nil)
	r.Host = "127.0.0.1:7070"
	r.AddCookie(&http.Cookie{Name: "si_studio_7070", Value: "T"})
	w := httptest.NewRecorder()
	other.Handler().ServeHTTP(w, r)
	id := func(s string) string {
		i := strings.Index(s, `data-ws="`)
		if i < 0 {
			t.Fatalf("no data-ws in %s", s)
		}
		return s[i+9 : i+9+12]
	}
	if id(b) == id(w.Body.String()) {
		t.Errorf("two folders share workspace id %s", id(b))
	}
}

func TestStudioAcceptsIPv6LoopbackHost(t *testing.T) {
	s, err := New(t.TempDir(), Options{Token: "T", Addr: "[::1]:7070"})
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("GET", "/_si/studio/api/tree", nil)
	r.Host = "[::1]:7070"
	r.AddCookie(&http.Cookie{Name: "si_studio_7070", Value: "T"})
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("[::1] host: %d", w.Code)
	}
}

// Through --web the terminal endpoints accept only the drawer's methods: a
// GET (a link or <img> in a deck) never starts or drives a shell.
func TestStudioTerminalRoutesAreMethodRestricted(t *testing.T) {
	s, err := New(t.TempDir(), Options{Token: "T", Addr: "127.0.0.1:7070"})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	for _, p := range []string{"new", "in?id=1", "resize?id=1&cols=80&rows=24", "close?id=1"} {
		r := httptest.NewRequest("GET", "/_si/term/api/"+p, nil)
		r.Host = "127.0.0.1:7070"
		r.AddCookie(&http.Cookie{Name: "si_studio_7070", Value: "T"})
		w := httptest.NewRecorder()
		s.Handler().ServeHTTP(w, r)
		if w.Code == 200 {
			t.Errorf("GET %s: 200, want refused", p)
		}
	}
}
