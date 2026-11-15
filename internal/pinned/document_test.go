package pinned

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDocument: a document from a pinned URL brings the files its files: key
// lists (each checked against its own pin, folders kept); a gist document
// brings the whole gist; both are cached by pin.
func TestDocument(t *testing.T) {
	prep, logo := []byte("echo prepared\n"), []byte("PNG")
	var doc []byte
	hits := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/demos/shop.md", func(w http.ResponseWriter, r *http.Request) { hits++; w.Write(doc) })
	mux.HandleFunc("/demos/prepare.sh", func(w http.ResponseWriter, r *http.Request) { w.Write(prep) })
	mux.HandleFunc("/demos/assets/logo.png", func(w http.ResponseWriter, r *http.Request) { w.Write(logo) })
	mux.HandleFunc("/gists/g2/r1", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"files": map[string]any{"demo.md": map[string]any{"content": "---\ntype: demo\n---\n"}, "prepare.sh": map[string]any{"content": "echo p"}}})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	doc = []byte("---\ntype: demo\napp: none\nfiles: prepare.sh · sha256 " + sum(prep) + ", assets/logo.png · sha256 " + sum(logo) + "\n---\n## F\n")
	c := testClient(t, srv.URL, nil)
	src, _ := Parse(srv.URL+"/demos/shop.md", sum(doc))
	p, err := c.Document(src)
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string][]byte{"shop.md": doc, "prepare.sh": prep, "assets/logo.png": logo} {
		if b, _ := os.ReadFile(filepath.Join(filepath.Dir(p), filepath.FromSlash(name))); string(b) != string(want) {
			t.Errorf("%s: %q", name, b)
		}
	}
	if _, err := c.Document(src); err != nil || hits != 1 {
		t.Errorf("cached by pin: %d hits, %v", hits, err)
	}
	// a file beside it that changed: nothing runs
	logo = []byte("GIF")
	if _, err := testClient(t, srv.URL, nil).Document(src); err == nil || !strings.Contains(err.Error(), "pin no longer matches") {
		t.Errorf("a changed file: %v", err)
	}
	for _, bad := range []string{"../x.sh · sha256 " + sum(prep), "/etc/x · sha256 " + sum(prep), "a/./x · sha256 " + sum(prep), "x.sh", "x.sh · sha256 12"} {
		if _, err := ParseFiles(bad); err == nil {
			t.Errorf("files: %q: no error", bad)
		}
	}
	if got := filesKey([]byte("= Shop\n:type: demo\n:files: a.sh · sha256 x\n\nfiles: no\n")); got != "a.sh · sha256 x" {
		t.Errorf("AsciiDoc header: %q", got)
	}
	if got := filesKey([]byte(`{"type":"demo","files":["a.sh · sha256 x","b/c.png · sha256 y"]}`)); got != "a.sh · sha256 x, b/c.png · sha256 y" {
		t.Errorf("JSON list: %q", got)
	}
	if got := filesKey([]byte(`{"type":"demo","files":"a.sh · sha256 x"}`)); got != "a.sh · sha256 x" {
		t.Errorf("JSON string: %q", got)
	}

	g, ok, err := ParseTarget("gist:g2@r1/demo.md")
	if !ok || err != nil {
		t.Fatal(ok, err)
	}
	gp, err := c.Document(g)
	if err != nil || filepath.Base(gp) != "demo.md" {
		t.Fatal(gp, err)
	}
	if b, _ := os.ReadFile(filepath.Join(filepath.Dir(gp), "prepare.sh")); string(b) != "echo p" {
		t.Errorf("the gist's other file: %q", b)
	}
	if _, _, err := ParseTarget("gist:g2@r1"); err == nil {
		t.Error("a gist target names its document")
	}
}
