package deck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestLoadGuide(t *testing.T) {
	dir := writeTree(t, map[string]string{
		"_nav.md":        "---\ntitle: The order journey\ntheme: paper\nversion: v1\n---\n\n- [Introduction](index.md)\n- Getting started\n  - [Install](install.md)\n  - [Docs](https://example.com)\n- [Reset](ops/reset.adoc)\n",
		"index.md":       "# Introduction\n",
		"install.md":     "# Install\n",
		"ops/reset.adoc": "= Reset\n",
		"extra.md":       "# Not listed\n",
		"_header.md":     "header\n",
	})
	g, err := LoadGuide(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Problems) > 0 {
		t.Fatalf("problems: %v", g.Problems)
	}
	if g.Meta.Title != "The order journey" || g.Meta.Theme != "paper" || len(g.Pages) != 3 {
		t.Fatalf("guide: %+v pages %v", g.Meta, g.Pages)
	}
	if len(g.Items) != 3 || g.Items[1].Href != "" || len(g.Items[1].Children) != 2 || g.Items[1].Children[0].Parent != g.Items[1] {
		t.Errorf("tree: section with two children expected, got %+v", g.Items[1])
	}
	if it := g.Item(filepath.Join(dir, "install.md")); it == nil || it.Title != "Install" {
		t.Errorf("Item(install.md) = %+v", it)
	}
	if u := g.Unlisted(); len(u) != 1 || !strings.HasSuffix(u[0], "extra.md") {
		t.Errorf("unlisted: %v (partials and listed pages excluded)", u)
	}
	d, err := ParseWith(filepath.Join(dir, "install.md"), g.Defaults())
	if err != nil || d.Meta.Type != "guide" || d.Meta.Theme != "paper" || d.Meta.Title != "" || d.Title() != "Install" {
		t.Errorf("a page inherits type and theme, not the guide title: %+v (%v)", d.Meta, err)
	}
	if FindGuide(filepath.Join(dir, "ops", "reset.adoc"), dir) == nil {
		t.Error("FindGuide walks up to the _nav.md")
	}
}

func TestLoadGuideProblems(t *testing.T) {
	dir := writeTree(t, map[string]string{
		"_nav.md": "- [A](a.md)\n- [Gone](gone.md)\n- [A again](a.md)\n- [Out](../x.md)\nnot a list\n",
		"a.md":    "# A\n",
	})
	g, err := LoadGuide(dir)
	if err != nil {
		t.Fatal(err)
	}
	var msgs []string
	for _, p := range g.Problems {
		msgs = append(msgs, p.Msg)
	}
	all := strings.Join(msgs, "\n")
	for _, want := range []string{"missing page gone.md", "listed twice", "outside the guide folder", "want a list item"} {
		if !strings.Contains(all, want) {
			t.Errorf("problems lack %q:\n%s", want, all)
		}
	}
}
