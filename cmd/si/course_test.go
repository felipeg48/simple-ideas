package main

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewAndBuildCourse(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "web101")
	var stdout, stderr bytes.Buffer
	if code := run([]string{"new", src, "--type", "course"}, &stdout, &stderr); code != 0 {
		t.Fatalf("new --type course: %d\n%s", code, stderr.String())
	}
	if code := run([]string{"check", src}, &stdout, &stderr); code != 0 {
		t.Fatalf("check: %d\n%s", code, stderr.String())
	}
	out := filepath.Join(tmp, "html")
	if code := run([]string{"build", src, "-o", out, "--zip", "--no-film"}, &stdout, &stderr); code != 0 {
		t.Fatalf("build: %d\n%s", code, stderr.String())
	}
	for _, f := range []string{"index.html", "complete.html", "01-web/page-load.html", "01-web/page-load.mp4", "02-html/quiz.html"} {
		if _, err := os.Stat(filepath.Join(out, f)); err != nil {
			t.Errorf("the bundle lacks %s", f)
		}
	}
	if _, err := os.Stat(filepath.Join(out, "02-html/inspect.html")); err != nil {
		t.Error("instructor-demos: show-placeholder keeps the demo page")
	}
	if b, _ := os.ReadFile(filepath.Join(out, "02-html/inspect.html")); strings.Contains(string(b), "I just changed the news") {
		t.Error("a learner build carries no demo runbook")
	}
	zr, err := zip.OpenReader(filepath.Join(tmp, "web101.zip"))
	if err != nil {
		t.Fatalf("--zip: %v", err)
	}
	defer zr.Close()
	found := false
	for _, f := range zr.File {
		found = found || f.Name == "index.html"
	}
	if !found {
		t.Error("the zip lacks index.html")
	}
	inst := filepath.Join(tmp, "inst")
	if code := run([]string{"build", src, "-o", inst, "--instructor", "--no-film"}, &stdout, &stderr); code != 0 {
		t.Fatalf("build --instructor: %d\n%s", code, stderr.String())
	}
	if b, _ := os.ReadFile(filepath.Join(inst, "02-html/inspect.html")); !strings.Contains(string(b), "I just changed the news") {
		t.Error("the instructor build carries the runbook")
	}
}
