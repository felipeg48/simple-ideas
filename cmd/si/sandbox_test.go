package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindLabBox(t *testing.T) {
	lab := func(img string) string { return "---\ntype: lab\ntitle: L\nsandbox: " + img + "\n---\n# L\n" }
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "deep"), 0o755)
	os.WriteFile(filepath.Join(dir, "deep", "lab.md"), []byte(lab("a:1")), 0o644)
	if b, err := findLabBox(dir); b != nil || err != nil {
		t.Errorf("a lab in a subfolder starts no box: %v %v", b, err)
	}
	os.WriteFile(filepath.Join(dir, "one.md"), []byte(lab("a:1")), 0o644)
	os.WriteFile(filepath.Join(dir, "two.md"), []byte(lab("a:1")), 0o644)
	b, err := findLabBox(dir)
	if err != nil || b == nil || b.spec.Image != "a:1" || b.course {
		t.Fatalf("shared box: %+v %v", b, err)
	}
	if b, _ := findLabBox(filepath.Join(dir, "deep", "lab.md")); b == nil {
		t.Error("the lab file itself")
	}
	os.WriteFile(filepath.Join(dir, "two.md"), []byte(lab("b:1")), 0o644)
	if _, err := findLabBox(dir); err == nil || !strings.Contains(err.Error(), "different sandboxes") {
		t.Errorf("two images: %v", err)
	}
	c := t.TempDir()
	os.WriteFile(filepath.Join(c, "_course.md"), []byte("---\ntitle: C\nsandbox: c:1\n---\n# C\n"), 0o644)
	if b, err := findLabBox(c); err != nil || b == nil || !b.course {
		t.Errorf("course box: %+v %v", b, err)
	}
}
