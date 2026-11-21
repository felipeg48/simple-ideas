package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// si build <book> --md --docx --rtf --epub writes every format beside the HTML.
func TestBuildBookExports(t *testing.T) {
	out := t.TempDir()
	var stdout, stderr bytes.Buffer
	if code := run([]string{"build", "-o", out, "--md", "--docx", "--rtf", "--epub", filepath.Join("..", "..", "examples", "book")}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr.String())
	}
	for _, f := range []string{"book.html", "md/book.md", "md/01-journey.md", "md/images/stations.svg", "book.docx", "book.rtf", "book.epub"} {
		if fi, err := os.Stat(filepath.Join(out, filepath.FromSlash(f))); err != nil || fi.Size() == 0 {
			t.Errorf("no %s", f)
		}
	}
}

// An article exports on its own; a deck says the flags are not for it.
func TestBuildArticleExportAndDeckRefusal(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	os.WriteFile(filepath.Join(dir, "note.md"), []byte("---\ntype: article\n---\n# Note\n\nSome *words*.\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "talk.md"), []byte("# Talk\n\nhello\n"), 0o644)
	var stdout, stderr bytes.Buffer
	if code := run([]string{"build", "-o", out, "--md", filepath.Join(dir, "note.md")}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr.String())
	}
	md, _ := os.ReadFile(filepath.Join(out, "md", "note.md"))
	if !strings.Contains(string(md), "Some *words*.") {
		t.Errorf("note.md:\n%s", md)
	}
	stderr.Reset()
	if code := run([]string{"build", "-o", t.TempDir(), "--docx", filepath.Join(dir, "talk.md")}, &stdout, &stderr); code == 0 || !strings.Contains(stderr.String(), "export books, guides and articles") {
		t.Errorf("a deck with --docx: exit %d, %s", code, stderr.String())
	}
}
