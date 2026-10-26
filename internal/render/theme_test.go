package render

import (
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
