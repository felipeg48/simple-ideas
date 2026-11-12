package webdemo

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A folder app never lists its files: a folder without index.html is blank.
func TestNoListing(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "talk.md"), []byte("# T"), 0o644)
	os.MkdirAll(filepath.Join(dir, "out"), 0o755)
	os.WriteFile(filepath.Join(dir, "out", "talk.html"), []byte("<h1>Talk</h1>"), 0o644)
	srv := httptest.NewServer(noListing(dir, http.FileServer(http.Dir(dir))))
	defer srv.Close()
	for path, want := range map[string]string{"/": "", "/out/": "", "/out/talk.html": "<h1>Talk</h1>"} {
		res, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(res.Body)
		res.Body.Close()
		if strings.Contains(string(b), "talk.md") || strings.Contains(string(b), "talk.html\"") {
			t.Errorf("%s lists files: %s", path, b)
		}
		if want != "" && string(b) != want {
			t.Errorf("%s = %q", path, b)
		}
	}
}

func TestFirstBrowserStep(t *testing.T) {
	f := &Feature{Blocks: []*Block{{Steps: []*Step{{Kind: "start"}, {Kind: "stop"}, {Kind: "go"}}}}}
	if s := firstBrowserStep(f); s == nil || s.Kind != "go" {
		t.Errorf("first browser step: %v", s)
	}
}
