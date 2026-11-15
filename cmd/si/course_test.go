package main

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
	// the deck video lesson ships its video, still, captions and PDF, never
	// older than the deck: a build without --no-film films nothing
	for _, f := range []string{"recap.mp4", "recap.poster.jpg", "recap.vtt", "recap.pdf"} {
		st, err := os.Stat(filepath.Join(src, "03-css", f))
		if err != nil || st.ModTime().Before(deckChanged(filepath.Join(src, "03-css", "recap.md"))) {
			t.Errorf("%s: missing or older than its deck (%v)", f, err)
		}
	}
	if s, _ := os.ReadFile(filepath.Join(src, "_course.md")); !strings.Contains(string(s), "(03-css/recap.md) · kind=video · pdf · slides") {
		t.Error("the example course has a deck video lesson")
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

// A deck lesson's video is stale when an included section (or code file)
// changed after it, not only the deck file.
func TestDeckChangedCountsIncludes(t *testing.T) {
	dir := t.TempDir()
	deckF, sec, code := filepath.Join(dir, "talk.md"), filepath.Join(dir, "sections", "01.md"), filepath.Join(dir, "sections", "a.go")
	os.MkdirAll(filepath.Dir(sec), 0o755)
	os.WriteFile(deckF, []byte("# Talk\n\n<!-- include: sections/01.md -->\n"), 0o644)
	os.WriteFile(sec, []byte("# One\n\n<!-- include: a.go -->\n"), 0o644)
	os.WriteFile(code, []byte("package a\n"), 0o644)
	base := time.Now().Add(-3 * time.Hour)
	for _, f := range []string{deckF, sec, code} {
		os.Chtimes(f, base, base)
	}
	if !deckChanged(deckF).Equal(base) {
		t.Fatalf("all at base: %v", deckChanged(deckF))
	}
	later := base.Add(time.Hour)
	os.Chtimes(code, later, later) // a code file included by an included section
	if !deckChanged(deckF).Equal(later) {
		t.Errorf("a nested code include counts: %v", deckChanged(deckF))
	}
	latest := base.Add(2 * time.Hour)
	os.Chtimes(sec, latest, latest)
	if !deckChanged(deckF).Equal(latest) {
		t.Errorf("an included section counts: %v", deckChanged(deckF))
	}
}

// A deck lesson shown as its video: si build films nothing when the video,
// captions and PDFs beside the deck are newer than it, and copies them
// beside the lesson; --no-film builds the same.
func TestBuildCourseDeckVideoLesson(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "c")
	files := map[string]string{
		"_course.md": "---\ntitle: C\n---\n\n- Module 1: One\n  - [Exchanges](03/ex.md) · kind=video · pdf · slides\n",
		"03/ex.md":   "---\ntype: deck\n---\n# Exchanges\n\nNotes: A producer never writes to a queue.\n",
		"03/ex.mp4":  "x", "03/ex.vtt": "WEBVTT\n", "03/ex.pdf": "%PDF",
	}
	for name, body := range files {
		p := filepath.Join(src, name)
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	old := time.Now().Add(-time.Hour)
	os.Chtimes(filepath.Join(src, "03/ex.md"), old, old) // the deck is older than its video and PDF
	for _, extra := range [][]string{nil, {"--no-film"}} {
		out := filepath.Join(tmp, "html"+strings.Join(extra, ""))
		var stdout, stderr bytes.Buffer
		if code := run(append([]string{"build", src, "-o", out}, extra...), &stdout, &stderr); code != 0 {
			t.Fatalf("build %v: %d\n%s", extra, code, stderr.String())
		}
		if strings.Contains(stdout.String(), "making") {
			t.Errorf("nothing is stale, nothing is made:\n%s", stdout.String())
		}
		for _, f := range []string{"03/ex.html", "03/ex.mp4", "03/ex.vtt", "03/ex.pdf", "03/ex.slides.html"} {
			if _, err := os.Stat(filepath.Join(out, f)); err != nil {
				t.Errorf("%v: the build lacks %s", extra, f)
			}
		}
		if b, _ := os.ReadFile(filepath.Join(out, "03/ex.html")); !strings.Contains(string(b), "Download the slides (PDF)") || !strings.Contains(string(b), `<track kind="captions"`) {
			t.Error("the lesson shows the PDF button and the captions")
		}
	}
}
