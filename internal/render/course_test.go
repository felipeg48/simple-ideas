package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

func renderTestCourse(t *testing.T, instructor bool) map[string]string {
	t.Helper()
	g, err := deck.LoadGuide(filepath.Join("testdata", "course"))
	if err != nil {
		t.Fatal(err)
	}
	pages, probs, _ := RenderGuide(g, Options{Instructor: instructor})
	if len(probs) != 0 {
		t.Fatalf("problems: %v", probs)
	}
	out := map[string]string{}
	for _, p := range pages {
		out[p.Rel] = string(p.Result.HTML)
	}
	return out
}

// TestCourseGolden pins the course shell of one lesson (from the top bar to
// the bottom bar) in testdata/course.golden.html. Run `go test
// ./internal/render -update` after an intended change and read the diff.
func TestCourseGolden(t *testing.T) {
	page := renderTestCourse(t, false)["m1/read.html"]
	i, j := strings.Index(page, `<div class="si-course"`), strings.Index(page, `<footer class="si-doc-foot">`)
	if i < 0 || j < i {
		t.Fatal("no course shell")
	}
	got := page[i:j]
	golden := filepath.Join("testdata", "course.golden.html")
	if *update {
		if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("%v (run with -update to create it)", err)
	}
	if got != string(want) {
		t.Fatalf("course golden mismatch (run go test ./internal/render -update if intended)")
	}
}

func TestCourseBuild(t *testing.T) {
	learner := renderTestCourse(t, false)
	for _, rel := range []string{"index.html", "complete.html", "m1/read.html", "m1/talk.html", "m1/live.html", "m1/lab.html", "m1/quiz.html"} {
		if learner[rel] == "" {
			t.Fatalf("learner build lacks %s", rel)
		}
	}
	home := learner["index.html"]
	for _, want := range []string{`data-view="home"`, `class="si-course-continue" href="m1/read.html"`, `Beginner · 16 min · 1 module`, `<span>Read</span><span>Slides</span><span>Demo</span><span>Lab</span><span>Quiz</span>`, `data-si-course`} {
		if !strings.Contains(home, want) {
			t.Errorf("home lacks %q", want)
		}
	}
	read := learner["m1/read.html"]
	for _, want := range []string{`<span class="si-course-crumb-kind">Reading</span>`, `Lesson 1 of 5`, `Then: Slides, 2 min`, `data-pass="100"`, `aria-current="page"`} {
		if !strings.Contains(read, want) {
			t.Errorf("reading lacks %q", want)
		}
	}
	if strings.Contains(read, `<aside class="si-course-instr"`) {
		t.Error("a learner build shows no instructor box")
	}
	if live := learner["m1/live.html"]; !strings.Contains(live, "Your instructor shows this live") || strings.Contains(live, "Here it is") {
		t.Error("show-placeholder: the learner sees the placeholder, not the runbook")
	}
	if talk := learner["m1/talk.html"]; !strings.Contains(talk, `class="si-embed-slides si-course-slides"`) || strings.Contains(talk, "Hello. [CLICK]") {
		t.Error("slides: embedded, without the speaker's script")
	}
	if lab := learner["m1/lab.html"]; !strings.Contains(lab, `<details class="si-solution">`) {
		t.Error("lab: the solution is folded")
	}
	if !strings.Contains(learner["complete.html"], "si-course-cert") {
		t.Error("certificate: on")
	}
	inst := renderTestCourse(t, true)
	if !strings.Contains(inst["m1/read.html"], "Ask who has seen this before.") || !strings.Contains(inst["m1/live.html"], "Here it is") ||
		!strings.Contains(inst["m1/lab.html"], `<details class="si-solution" open>`) || !strings.Contains(inst["m1/talk.html"], "Hello.") {
		t.Error("instructor build: the note, the runbook, the open solution, the notes")
	}
	// without show-placeholder a learner build leaves the demo out entirely
	g, _ := deck.LoadGuide(filepath.Join("testdata", "course"))
	g.Meta.InstructorDemos = ""
	pages, _, _ := RenderGuide(g, Options{})
	for _, p := range pages {
		if p.Rel == "m1/live.html" || strings.Contains(string(p.Result.HTML), `href="live.html"`) {
			t.Errorf("%s: the demo should be gone", p.Rel)
		}
	}
}

func TestQuizRender(t *testing.T) {
	quiz := renderTestCourse(t, false)["m1/quiz.html"]
	for _, want := range []string{`<form class="si-quiz" data-pass="50"`, `<fieldset class="si-quiz-q" data-multi>`, `type="checkbox"`, `value="0" data-ok>`, `Choose every right answer.`} {
		if !strings.Contains(quiz, want) {
			t.Errorf("quiz lacks %q", want)
		}
	}
	d, err := deck.ParseSource(filepath.Join(t.TempDir(), "a.md"), "---\ntype: article\n---\n# A\n\n```quiz\n## Lonely\n- [x] only\n```\n", deck.Meta{})
	if err != nil {
		t.Fatal(err)
	}
	res := Deck(d, Options{})
	if len(res.Problems) != 1 || !strings.Contains(res.Problems[0].Msg, "at least two answers") {
		t.Errorf("a quiz problem: %v", res.Problems)
	}
}

func TestGuideNavLayout(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "_nav.md"), []byte("---\ntitle: G\nnav: right · accordion · numbered · not-sticky\n---\n- Part\n  - [A](a.md)\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "a.md"), []byte("# A\n"), 0o644)
	g, err := deck.LoadGuide(dir)
	if err != nil {
		t.Fatal(err)
	}
	pages, _, _ := RenderGuide(g, Options{})
	h := string(pages[0].Result.HTML)
	if !strings.Contains(h, `data-layout="guide" data-nav="right" data-nav-static`) || !strings.Contains(h, `class="si-doc-nav" aria-label="G" data-accordion data-numbered`) {
		t.Error("nav: on a guide sets the layout and the navigation attributes")
	}
}
