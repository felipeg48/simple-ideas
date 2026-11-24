package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/felipeg48/simple-ideas/internal/render"
)

// TestMergeIndex: building one file after another into one folder keeps the
// first page in index.html; a page that is gone drops out.
func TestMergeIndex(t *testing.T) {
	out := t.TempDir()
	for _, f := range []string{"a.html", "b.html"} {
		os.WriteFile(filepath.Join(out, f), []byte("x"), 0o644)
	}
	old := []render.IndexEntry{{Href: "a.html", Title: "A"}, {Href: "gone.html", Title: "Gone"}, {Href: "b.html", Title: "B old"}}
	os.WriteFile(filepath.Join(out, "index.html"), render.Index("t", old, render.Options{}), 0o644)
	got := mergeIndex(out, []render.IndexEntry{{Href: "b.html", Title: "B"}})
	if len(got) != 2 || got[0].Title != "A" || got[1].Title != "B" {
		t.Fatalf("merged = %+v, want A then the new B", got)
	}
}
