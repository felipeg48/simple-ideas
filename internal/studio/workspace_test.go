package studio

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, root, rel string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("# T\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestResolveRejectsEscapes(t *testing.T) {
	dir := t.TempDir()
	w, err := OpenWorkspace(dir)
	if err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(dir, "out")); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"../x", "/etc/passwd", "a/../../x", "out/f", "out"} {
		if _, err := w.Resolve(rel); !errors.Is(err, ErrOutside) {
			t.Errorf("Resolve(%q) err = %v, want ErrOutside", rel, err)
		}
	}
	got, err := w.Resolve("decks/a.md")
	if err != nil || got != filepath.Join(w.Root, "decks", "a.md") {
		t.Errorf("Resolve(decks/a.md) = %q, %v", got, err)
	}
	if r := w.Rel(got); r != "decks/a.md" {
		t.Errorf("Rel = %q", r)
	}
}

func TestTreeKinds(t *testing.T) {
	dir := t.TempDir()
	for _, f := range []string{"talk.md", "sections/one.md", "img/p.png", "theme.css", "rb.json", ".git/x", "node_modules/y"} {
		write(t, dir, f)
	}
	w, _ := OpenWorkspace(dir)
	root, err := w.Tree()
	if err != nil {
		t.Fatal(err)
	}
	kinds := map[string]string{}
	var order []string
	var walk func(n Node)
	walk = func(n Node) {
		kinds[n.Path] = n.Kind
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(root)
	for p, k := range map[string]string{"talk.md": "deck", "sections/one.md": "markdown", "img/p.png": "image",
		"theme.css": "css", "rb.json": "json", "sections": "dir"} {
		if kinds[p] != k {
			t.Errorf("%s kind = %q, want %q", p, kinds[p], k)
		}
	}
	for _, p := range []string{".git", ".git/x", "node_modules", "node_modules/y"} {
		if _, ok := kinds[p]; ok {
			t.Errorf("%s should be skipped", p)
		}
	}
	for _, c := range root.Children {
		order = append(order, c.Name)
	}
	if want := []string{"img", "sections", "rb.json", "talk.md", "theme.css"}; len(order) != len(want) ||
		order[0] != want[0] || order[1] != want[1] || order[2] != want[2] || order[3] != want[3] || order[4] != want[4] {
		t.Errorf("order = %v, want %v", order, want)
	}
}
