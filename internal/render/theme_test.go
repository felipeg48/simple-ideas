package render

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

func renderMD(t *testing.T, dir, md string) Result {
	t.Helper()
	p := filepath.Join(dir, "d.md")
	if err := os.WriteFile(p, []byte(md), 0o644); err != nil {
		t.Fatal(err)
	}
	d, err := deck.Parse(p)
	if err != nil {
		t.Fatal(err)
	}
	return Deck(d, Options{})
}

func TestCSSFileTheme(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "sunset.css"),
		[]byte(`:root[data-si-theme="custom"]{--si-accent:#ff7a59}`), 0o644); err != nil {
		t.Fatal(err)
	}
	res := renderMD(t, dir, "---\ntitle: t\ntheme: sunset.css\n---\n# Hi\n")
	if len(res.Problems) != 0 {
		t.Fatalf("problems: %v", res.Problems)
	}
	h := string(res.HTML)
	if !strings.Contains(h, `data-si-theme="custom"`) {
		t.Error(`expected data-si-theme="custom"`)
	}
	if !strings.Contains(h, "--si-accent:#ff7a59") {
		t.Error("theme file tokens not injected")
	}
}

func TestCSSFileThemeMissing(t *testing.T) {
	dir := t.TempDir()
	res := renderMD(t, dir, "---\ntitle: t\ntheme: nope.css\n---\n# Hi\n")
	if len(res.Problems) == 0 {
		t.Error("a missing theme file should be a problem")
	}
	if !strings.Contains(string(res.HTML), `data-si-theme="studio"`) {
		t.Error("should fall back to studio")
	}
}

func TestURLThemeFetched(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`:root[data-si-theme="custom"]{--si-accent:#0af}`))
	}))
	defer srv.Close()
	dir := t.TempDir()
	res := renderMD(t, dir, "---\ntitle: t\ntheme: "+srv.URL+"/theme.css\n---\n# Hi\n")
	if len(res.Problems) != 0 {
		t.Fatalf("problems: %v", res.Problems)
	}
	h := string(res.HTML)
	if !strings.Contains(h, `data-si-theme="custom"`) || !strings.Contains(h, "--si-accent:#0af") {
		t.Error("URL theme not fetched/injected")
	}
}

func TestURLThemeUnreachable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	url := srv.URL + "/missing.css"
	srv.Close() // now unreachable, and 404 would also fail
	dir := t.TempDir()
	res := renderMD(t, dir, "---\ntitle: t\ntheme: "+url+"\n---\n# Hi\n")
	if len(res.Problems) == 0 {
		t.Error("unreachable theme URL should be a problem")
	}
	if !strings.Contains(string(res.HTML), `data-si-theme="studio"`) {
		t.Error("should fall back to studio")
	}
}
