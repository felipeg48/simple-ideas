package serve

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/felipeg48/simple-ideas/internal/sandbox"
)

func TestSandboxRoutes(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "lab.md"), []byte("---\ntype: lab\ntitle: L\nsandbox: docker:28-dind\nsandbox-ports: 8080 Shop\n---\n# L\n\nOpen [the shop](sandbox:8080) or [its API](sandbox:8080/api).\n"), 0o644)
	s, err := New(dir, nil, "", "")
	if err != nil {
		t.Fatal(err)
	}
	s.Terminal = true
	var box *sandbox.Box
	s.Sandbox = func() *sandbox.Box { return box }
	h := s.Handler()
	get := func(path string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", path, nil)
		r.Host, r.RemoteAddr = "127.0.0.1:7070", "127.0.0.1:50000"
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	if w := get("/_si/sandbox/port/8080/"); w.Code != http.StatusNotFound {
		t.Errorf("no box yet: %d", w.Code)
	}
	if w := get("/_si/sandbox"); !strings.Contains(w.Body.String(), `"running":false`) {
		t.Errorf("state %s", w.Body.String())
	}
	box = &sandbox.Box{Name: "si-box-x", Runtime: sandbox.SysboxRuntime,
		Spec:  deck.Sandbox{Image: "docker:28-dind", Ports: []deck.SandboxPort{{Port: 8080, Label: "Shop"}}},
		Ports: map[int]sandbox.HostPort{8080: {Addr: "127.0.0.1:55555", Label: "Shop"}}}
	w := get("/_si/sandbox/port/8080/api/v1?x=1")
	if w.Code != http.StatusFound || w.Header().Get("Location") != "http://127.0.0.1:55555/api/v1?x=1" {
		t.Errorf("redirect %d %q", w.Code, w.Header().Get("Location"))
	}
	if w := get("/_si/sandbox/port/9999/"); w.Code != http.StatusNotFound {
		t.Errorf("a port the lab does not open: %d", w.Code)
	}
	if w := get("/_si/sandbox"); !strings.Contains(w.Body.String(), `"addr":"127.0.0.1:55555"`) || !strings.Contains(w.Body.String(), `"unprivileged":true`) {
		t.Errorf("state %s", w.Body.String())
	}
	r := httptest.NewRequest("GET", "/_si/sandbox", nil)
	r.Host, r.RemoteAddr = "192.168.1.20:7070", "192.168.1.30:50000"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	if rec.Code != http.StatusForbidden {
		t.Errorf("off loopback: %d", rec.Code)
	}
	page := get("/lab.html").Body.String()
	if !strings.Contains(page, `href="/_si/sandbox/port/8080/"`) || !strings.Contains(page, `href="/_si/sandbox/port/8080/api"`) {
		t.Error("sandbox: links not pointed at si play")
	}
}
