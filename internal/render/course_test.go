package render

import (
	"bytes"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/felipeg48/simple-ideas/internal/config"
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
	if lab := learner["m1/lab.html"]; !strings.Contains(lab, `<div class="si-check" data-cmd="test -f page.html">`) || !strings.Contains(lab, "Run this to check:") ||
		!strings.Contains(lab, `<p class="si-check-bar" hidden>`) || regexp.MustCompile(`<body[^>]* data-si-checks`).MatchString(lab) || strings.Contains(lab, "window.SI_WEBHOOK =") {
		t.Error("lab: a built check is a command to run, no ✓ Check; no webhook, no events")
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

// TestCourseNumbered: nav: … numbered numbers lessons within their module
// (1.1, 1.2) in the menu and on the next-lesson card; without it, none.
func TestCourseNumbered(t *testing.T) {
	g, err := deck.LoadGuide(filepath.Join("testdata", "course"))
	if err != nil {
		t.Fatal(err)
	}
	read := func() string {
		pages, _, _ := RenderGuide(g, Options{})
		for _, p := range pages {
			if p.Rel == "m1/read.html" {
				return string(p.Result.HTML)
			}
		}
		t.Fatal("no m1/read.html")
		return ""
	}
	if strings.Contains(read(), `<span class="si-course-lt">1.1 `) {
		t.Error("no numbered: no lesson numbers")
	}
	g.Meta.Nav = "left numbered"
	page := read()
	for _, want := range []string{`<span class="si-course-lt">1.1 `, `<span class="si-course-lt">1.2 `} {
		if !strings.Contains(page, want) {
			t.Errorf("menu lacks %q", want)
		}
	}
	if !regexp.MustCompile(`<a rel="next" href="[^"]*"><small>[^<]*</small>1\.2 `).MatchString(page) {
		t.Error("the next-lesson card carries 1.2")
	}
}

// TestCourseWebhookConfig: a course with webhook: gets window.SI_WEBHOOK
// and webhook.js before course.js; the relay keeps the address and the auth
// off the page; webhook: relay in a static build sends nothing.
func TestCourseWebhookConfig(t *testing.T) {
	g, err := deck.LoadGuide(filepath.Join("testdata", "course"))
	if err != nil {
		t.Fatal(err)
	}
	g.Meta.Webhook, g.Meta.WebhookFormat, g.Meta.WebhookAuth, g.Meta.WebhookEvents = "https://lrs.example.com/statements", "xapi", "Basic dTpw", "quiz.submitted, course.completed"
	g.Meta.WebhookConsent, g.Meta.WebhookNotice = "required", "Scores go to </script> Ana."
	page := func(o Options) string {
		pages, _, _ := RenderGuide(g, o)
		for _, p := range pages {
			if p.Rel == "m1/quiz.html" {
				return string(p.Result.HTML)
			}
		}
		t.Fatal("no quiz page")
		return ""
	}
	SIVersion = "9.9.9"
	defer func() { SIVersion = "" }()
	built := page(Options{})
	for _, want := range []string{`window.SI_WEBHOOK = {`, `"url":"https://lrs.example.com/statements"`, `"format":"xapi"`, `"auth":"Basic dTpw"`,
		`"events":["quiz.submitted","course.completed"]`, `"consent":"required"`, `"si":"9.9.9"`, `"audience":"learner"`, `"course":{"id":"course","title":"Golden course"}`, "Scores go to \\u003c/script\\u003e Ana."} {
		if !strings.Contains(built, want) {
			t.Errorf("built page lacks %s", want)
		}
	}
	if strings.Index(built, "window.siWebhook = {") > strings.Index(built, "simple-ideas course runtime") {
		t.Error("webhook.js comes before course.js")
	}
	relayed := page(Options{Relay: true, Instructor: true})
	if !strings.Contains(relayed, `"relay":"/_si/relay"`) || strings.Contains(relayed, "lrs.example.com") || strings.Contains(relayed, "dTpw") || !strings.Contains(relayed, `"audience":"instructor"`) {
		t.Error("relayed: the page posts to /_si/relay and holds no address or auth; the instructor build says so")
	}
	g.Meta.Webhook = "relay"
	if strings.Contains(page(Options{}), "SI_WEBHOOK") {
		t.Error("webhook: relay in a static build: no events")
	}
	g.Meta.Webhook = ""
	if strings.Contains(page(Options{Config: &config.Config{CourseWebhook: "https://hooks.example.com/all"}}), `"url":"https://hooks.example.com/all"`) == false {
		t.Error("si.conf course.webhook is the default")
	}
}

// A video gets a still (issue #12): the lesson's own poster, none with off,
// else <video>.poster.jpg, made by MakePoster when it is missing; a video in
// a document gets the same.
func TestVideoPoster(t *testing.T) {
	dir := t.TempDir()
	clip, cover := filepath.Join(dir, "m1", "clip.mp4"), filepath.Join(dir, "cover.jpg")
	os.MkdirAll(filepath.Dir(clip), 0o755)
	os.WriteFile(clip, []byte("x"), 0o644)
	os.WriteFile(cover, []byte("x"), 0o644)
	page := filepath.Join(dir, "m1", "clip.md")
	body := func(poster string) (string, []MediaFile) {
		return videoBody(page, &deck.NavItem{Video: clip, Poster: poster})
	}
	if b, m := body(cover); !strings.Contains(b, ` poster="../cover.jpg"`) || len(m) != 2 {
		t.Errorf("· poster=: %s %v", b, m)
	}
	if b, _ := body("off"); strings.Contains(b, "poster=") {
		t.Errorf("poster=off: %s", b)
	}
	if b, _ := body(""); strings.Contains(b, "poster=") {
		t.Errorf("no still and no MakePoster: %s", b)
	}
	made := 0
	MakePoster = func(video, still string) bool { made++; return os.WriteFile(still, []byte("x"), 0o644) == nil }
	defer func() { MakePoster = nil }()
	if b, _ := body(""); !strings.Contains(b, ` poster="clip.poster.jpg"`) || made != 1 {
		t.Errorf("first frame: %s (made %d)", b, made)
	}
	if body(""); made != 1 {
		t.Error("a still made once is kept")
	}
	// a still si made carries its video's fingerprint: a new video remakes it,
	// the same one (whatever its mtime) does not; an unstamped still is kept
	still := PosterOf(clip)
	var jb bytes.Buffer
	jpeg.Encode(&jb, image.NewGray(image.Rect(0, 0, 2, 2)), nil)
	MakePoster = func(video, still string) bool {
		made++
		return os.WriteFile(still, jb.Bytes(), 0o644) == nil && StampPoster(still, video) == nil
	}
	os.WriteFile(still, jb.Bytes(), 0o644)
	if body(""); made != 1 {
		t.Error("a still without a fingerprint (made by hand) is kept")
	}
	StampPoster(still, clip)
	if posterStamp(still) != VideoStamp(clip) {
		t.Fatalf("stamp: %q", posterStamp(still))
	}
	sb, _ := os.ReadFile(still)
	if _, err := jpeg.Decode(bytes.NewReader(sb)); err != nil {
		t.Errorf("a stamped still is still a JPEG: %v", err)
	}
	os.Chtimes(clip, time.Now().Add(time.Hour), time.Now().Add(time.Hour))
	if body(""); made != 1 {
		t.Error("the same video (newer mtime) keeps its still")
	}
	os.WriteFile(clip, []byte("a new take"), 0o644)
	if body(""); made != 2 || posterStamp(still) != VideoStamp(clip) {
		t.Errorf("a changed video remakes its still (made %d)", made)
	}
	doc := filepath.Join(dir, "m1", "d.md")
	os.WriteFile(doc, []byte("---\ntype: lab\n---\n# D\n\n![A clip](clip.mp4)\n"), 0o644)
	d, err := deck.Parse(doc)
	if err != nil {
		t.Fatal(err)
	}
	if h := string(Deck(d, Options{Media: MediaLink}).HTML); !strings.Contains(h, `<video poster="clip.poster.jpg" controls`) {
		t.Errorf("a document video gets its still:\n%s", h[strings.Index(h, "<video"):][:200])
	}
}

func TestQuizRender(t *testing.T) {
	form := func(page string) string { // the quiz form only, not the page scripts
		i := strings.Index(page, `<form class="si-quiz"`)
		j := strings.Index(page[i:], `</form>`)
		return page[i : i+j]
	}
	quiz := form(renderTestCourse(t, false)["m1/quiz.html"])
	for _, want := range []string{`<form class="si-quiz" data-pass="50" data-salt="`, `<fieldset class="si-quiz-q" data-multi data-key="`, `type="checkbox"`, `Choose every right answer.`} {
		if !strings.Contains(quiz, want) {
			t.Errorf("quiz lacks %q", want)
		}
	}
	if strings.Contains(quiz, "data-ok") {
		t.Error("a learner page keeps the right answers only as a hash")
	}
	salt := regexp.MustCompile(`data-salt="([0-9a-f]{16})"`).FindStringSubmatch(quiz)
	if salt == nil || !strings.Contains(quiz, `data-key="`+QuizKey(salt[1], 1, []int{0, 1})+`"`) {
		t.Error("the key is QuizKey(salt, 1, the right answers)")
	}
	if inst := form(renderTestCourse(t, true)["m1/quiz.html"]); !strings.Contains(inst, `value="0" data-ok>`) || strings.Contains(inst, "data-key") {
		t.Error("an instructor page keeps the plain answers")
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

// TestCourseBook: the learner workbook keeps the answers, explanations and
// solutions out and leaves room for notes; the instructor guide has the
// runbook, the script, the answers and the solutions.
func TestCourseBook(t *testing.T) {
	g, err := deck.LoadGuide(filepath.Join("testdata", "course"))
	if err != nil {
		t.Fatal(err)
	}
	wb, probs := CourseBook(g, Options{})
	if len(probs) != 0 || wb.Rel != "workbook.html" {
		t.Fatalf("workbook: %s %v", wb.Rel, probs)
	}
	body := wb.Result.body
	for _, bad := range []string{"data-ok>", `class="si-quiz-why"`, `class="si-solution"`, "Hello.", "Here it is", "si-quiz-check"} {
		if strings.Contains(body, bad) {
			t.Errorf("the workbook carries %q", bad)
		}
	}
	for _, want := range []string{"si-cb-notes", "Module 1 · Everything", "Two?", "Your instructor shows this live"} {
		if !strings.Contains(body, want) {
			t.Errorf("the workbook lacks %q", want)
		}
	}
	ig, _ := CourseBook(g, Options{Instructor: true})
	body = ig.Result.body
	if ig.Rel != "instructor-guide.html" {
		t.Errorf("instructor guide: %s", ig.Rel)
	}
	for _, want := range []string{"data-ok checked", `<details class="si-solution" open>`, "Hello.", "Here it is", "Ask who has seen this before."} {
		if !strings.Contains(body, want) {
			t.Errorf("the instructor guide lacks %q", want)
		}
	}
	if strings.Contains(body, `class="si-cb-notes"`) || strings.Contains(body, "[CLICK]") {
		t.Error("the instructor guide has no notes boxes and no [CLICK] cues")
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

// TestDemoLessons (issue #13): a filmed web demo under kind=demo is a learner
// lesson labelled Demo with its video; a demo without app: stays the
// instructor's live runbook, out of the learner build.
func TestDemoLessons(t *testing.T) {
	dir := t.TempDir()
	for name, body := range map[string]string{
		"_course.md": "---\ntitle: T\n---\n- Module 1: M\n  - [Intro](m/intro.md)\n  - [D](m/d.md) · kind=demo\n  - [Live](m/live.md) · kind=demo\n",
		"m/intro.md": "# Intro\n\nWords.\n",
		"m/d.md":     "---\ntype: demo\ntitle: D\napp: http://localhost:1\n---\n\n## F\n\nSay: Look.\n- go /\n",
		"m/d.mp4":    "",
		"m/d.vtt":    "WEBVTT\n",
		"m/live.md":  "---\ntype: demo\ntitle: Live\n---\n\n## Here it is\n\nRun it.\n",
	} {
		p := filepath.Join(dir, filepath.FromSlash(name))
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte(body), 0o644)
	}
	g, err := deck.LoadGuide(dir)
	if err != nil {
		t.Fatal(err)
	}
	pages, probs, _ := RenderGuide(g.ForAudience(false), Options{})
	if len(probs) != 0 {
		t.Fatal(probs)
	}
	got := map[string]string{}
	for _, p := range pages {
		got[p.Rel] = string(p.Result.HTML)
	}
	d := got["m/d.html"]
	for _, want := range []string{`<span class="si-course-crumb-kind">Demo</span>`, `<video controls preload="metadata" playsinline src="d.mp4"><track kind="captions" src="d.vtt"`} {
		if !strings.Contains(d, want) {
			t.Errorf("the filmed demo lacks %s", want)
		}
	}
	if strings.Contains(d, "An instructor demo") {
		t.Error("a filmed demo is no instructor demo")
	}
	if !strings.Contains(got["m/intro.html"], "Then: Demo") {
		t.Error("the next-lesson line says Demo")
	}
	if _, ok := got["m/live.html"]; ok {
		t.Error("a demo without app: stays out of the learner build")
	}
}
