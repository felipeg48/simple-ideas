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

// A file theme's relative url()s are inlined against the theme's own directory,
// so fonts and logos next to it keep the deck self-contained.
func TestCSSFileThemeInlinesURLs(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "css", "fonts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "css", "fonts", "u.woff2"), []byte("wOF2"), 0o644); err != nil {
		t.Fatal(err)
	}
	css := `@font-face{font-family:U;src:url("fonts/u.woff2") format("woff2")} a{background:url(https://x.test/p.png)} b{background:url(missing.png)}`
	if err := os.WriteFile(filepath.Join(dir, "css", "brand.css"), []byte(css), 0o644); err != nil {
		t.Fatal(err)
	}
	h := string(renderMD(t, dir, "---\ntitle: t\ntheme: css/brand.css\n---\n# Hi\n").HTML)
	for _, want := range []string{`url("data:font/woff2;base64,d09GMg==")`, `url(https://x.test/p.png)`, `url(missing.png)`} {
		if !strings.Contains(h, want) {
			t.Errorf("missing %s", want)
		}
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

// A theme file with "extends <built-in>" in its header gets that theme's
// tokens first and its own after; without the header nothing is underneath.
func TestCSSFileThemeExtends(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "acme.css"), []byte("/* si theme: Acme · extends paper */\n:root { --si-accent: #e4572e; }\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "bare.css"), []byte(":root { --si-accent: #e4572e; }\n"), 0o644)
	h := string(renderMD(t, dir, "---\ntitle: t\ntheme: acme.css\n---\n# Hi\n").HTML)
	paper, mine := strings.Index(h, "--si-bg:#fbf7ee"), strings.Index(h, "--si-accent: #e4572e")
	if paper < 0 || mine < paper || !strings.Contains(h, "--si-h1-size:") {
		t.Errorf("extends: paper at %d, the file at %d", paper, mine)
	}
	if h := string(renderMD(t, dir, "---\ntitle: t\ntheme: bare.css\n---\n# Hi\n").HTML); strings.Contains(h, "--si-h1-size:") {
		t.Error("a theme file without the header got built-in tokens")
	}
	os.WriteFile(filepath.Join(dir, "bad.css"), []byte("/* si theme: X · extends nope */\n:root{}\n"), 0o644)
	if res := renderMD(t, dir, "---\ntitle: t\ntheme: bad.css\n---\n# Hi\n"); len(res.Problems) != 1 {
		t.Errorf("extends nope: %v", res.Problems)
	}
}

// The index page follows a .css theme too.
func TestIndexCSSTheme(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "acme.css")
	os.WriteFile(p, []byte("/* si theme: Acme · extends forest */\n:root { --si-accent: #e4572e; }\n"), 0o644)
	h := string(Index("x", nil, Options{Theme: p}))
	if !strings.Contains(h, `data-si-theme="custom"`) || !strings.Contains(h, "--si-accent: #e4572e") || !strings.Contains(h, "Georgia") {
		t.Error("index page does not follow the .css theme")
	}
	if h := string(Index("x", nil, Options{Theme: "nope"})); !strings.Contains(h, `data-si-theme="studio"`) {
		t.Error("unknown theme should fall back to studio")
	}
}
