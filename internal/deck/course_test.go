package deck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadCourse(t *testing.T) {
	dir := writeTree(t, map[string]string{
		"_course.md": "---\ntitle: A first web page\nlevel: beginner\nnav: left · accordion · sticky\n---\n" +
			"- Module 1: How the web works · prerequisites=none\n" +
			"  - [Browsers](01/read.md)\n" +
			"  - [Slides](01/talk.md)\n" +
			"  - [A page loads](01/clip.mp4)\n" +
			"  - [Build it](01/lab.md)\n" +
			"  - [Live](01/demo.md)\n" +
			"  - [Filmed](01/film.md)\n" +
			"  - [Check](01/quiz.md)\n" +
			"  - [Forced](01/forced.md) · kind=lab · minutes=7\n",
		"01/read.md":   "---\nminutes: 12\nprerequisites: a browser\n---\n# Browsers\n\nWords.\n",
		"01/talk.md":   "---\ntype: deck\n---\n# Talk\n\nNotes: (3 min) Hello.\n\n---\n\n## Two\n\nNotes: (2 min) Bye.\n",
		"01/clip.mp4":  "not really a video",
		"01/lab.md":    "---\ntype: lab\n---\n# Lab\n\n## Step 1: One\n\n## Step 2: Two\n",
		"01/demo.md":   "---\ntype: demo\nduration: 8 min\n---\n# Live\n",
		"01/film.md":   "---\ntype: demo\napp: https://example.com\n---\n# Filmed\n",
		"01/quiz.md":   "---\ntype: quiz\npass: 70%\n---\n# Check\n\n## One?\n- [x] a\n- [ ] b\n",
		"01/forced.md": "# Forced\n",
	})
	g, err := LoadGuide(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !g.IsCourse() || g.Meta.Level != "beginner" || len(g.Problems) != 0 {
		t.Fatalf("course: %v %q %v", g.IsCourse(), g.Meta.Level, g.Problems)
	}
	if len(g.Items) != 1 || g.Items[0].Prerequisites != "none" || ModuleTitle(g.Items[0].Title) != "How the web works" {
		t.Fatalf("module: %+v", g.Items[0])
	}
	want := []struct {
		kind    string
		minutes int
	}{{"reading", 12}, {"slides", 5}, {"video", 0}, {"lab", 10}, {"demo", 8}, {"video", 0}, {"quiz", 1}, {"lab", 7}}
	ls := g.Lessons()
	if len(ls) != len(want) || len(g.Pages) != len(want) {
		t.Fatalf("lessons: %d, pages %d", len(ls), len(g.Pages))
	}
	for i, w := range want {
		if ls[i].Kind != w.kind || ls[i].Minutes != w.minutes {
			t.Errorf("lesson %d (%s): %s %d min, want %s %d", i, ls[i].Title, ls[i].Kind, ls[i].Minutes, w.kind, w.minutes)
		}
	}
	if ls[0].Prerequisites != "a browser" || ls[5].Script == "" || !strings.HasSuffix(ls[5].Video, "film.mp4") {
		t.Errorf("prerequisites / filmed demo: %+v %+v", ls[0], ls[5])
	}
	if len(g.Warnings) != 1 || !strings.Contains(g.Warnings[0].Msg, "not filmed yet") {
		t.Errorf("want the unfilmed demo warning, got %v", g.Warnings)
	}
	learner := g.ForAudience(false)
	if len(learner.Pages) != len(want)-1 || len(g.Pages) != len(want) {
		t.Errorf("learner build: %d pages (the demo left out), course keeps %d", len(learner.Pages), len(g.Pages))
	}
	g.Meta.InstructorDemos = "show-placeholder"
	if g.ForAudience(false) != g {
		t.Error("show-placeholder keeps the demos")
	}
	if d := g.Defaults(); d.Level != "" || d.Nav == "" {
		t.Errorf("lessons inherit the nav, not the level: %+v", d)
	}
}

func TestCourseProblems(t *testing.T) {
	for _, tc := range []struct{ nav, want string }{
		{"- [Loose](a.md)\n", "a top-level item is a module"},
		{"- Module 1: Empty\n", "has no lessons"},
		{"- Module 1: M\n  - [Gone](gone.mp4)\n", "missing video gone.mp4"},
		{"- Module 1: M\n  - [A](a.md) · kind=movie\n", "kind=movie is not one of"},
		{"- Module 1: M\n  - [A](a.md) · minutes=soon\n", "minutes=soon is not a number"},
		{"- Module 1: M\n  - [Home](index.md)\n", "index.html is the course home"},
	} {
		dir := writeTree(t, map[string]string{"_course.md": "---\ntitle: C\n---\n" + tc.nav, "a.md": "# A\n", "index.md": "# I\n"})
		g, err := LoadGuide(dir)
		if err != nil {
			t.Fatal(err)
		}
		var msgs []string
		for _, p := range g.Problems {
			msgs = append(msgs, p.Msg)
		}
		if !strings.Contains(strings.Join(msgs, "\n"), tc.want) {
			t.Errorf("%q: want %q, got %v", tc.nav, tc.want, msgs)
		}
	}
}

func TestParseNav(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want NavLayout
		err  bool
	}{
		{"left · accordion · sticky", NavLayout{Pos: "left", Accordion: true, AccordionSet: true, Sticky: true, StickySet: true}, false},
		{"top, tabs", NavLayout{Pos: "top", Tabs: true}, false},
		{"bottom", NavLayout{Pos: "bottom"}, false},
		{"right collapsed numbered not-sticky", NavLayout{Pos: "right", Collapsed: true, Numbered: true, StickySet: true}, false},
		{"left right", NavLayout{}, true},
		{"sideways", NavLayout{}, true},
	} {
		got, err := ParseNav(tc.in)
		if (err != nil) != tc.err || (!tc.err && got != tc.want) {
			t.Errorf("ParseNav(%q) = %+v, %v", tc.in, got, err)
		}
	}
}

func TestQuizParseAndScore(t *testing.T) {
	intro, qs, probs := ParseQuiz("Ten questions.\n\n## Which tag makes a link?\n- [ ] `<p>`\n- [x] `<a>`\n> Explanation: `a` is for anchor.\n> It needs an href.\n\n## Pick the block tags\nMore than one.\n- [x] div\n- [ ] span\n- [x] p\n\n## Broken\n- [ ] only\n")
	if intro != "Ten questions." || len(qs) != 3 {
		t.Fatalf("intro %q, %d questions", intro, len(qs))
	}
	if qs[0].Explanation != "`a` is for anchor. It needs an href." || qs[0].Multi() || !qs[1].Multi() || len(qs[1].Text) != 1 {
		t.Errorf("questions: %+v", qs)
	}
	if len(probs) != 1 || !strings.Contains(probs[0].Msg, "at least two answers") || probs[0].Line != 15 {
		t.Errorf("problems: %v", probs)
	}
	qs = qs[:2]
	for _, tc := range []struct {
		picks [][]int
		want  int
	}{
		{[][]int{{1}, {0, 2}}, 100},
		{[][]int{{1}, {0}}, 50},
		{[][]int{{0}, {0, 1, 2}}, 0},
		{nil, 0},
	} {
		if got := Score(qs, tc.picks); got != tc.want {
			t.Errorf("Score(%v) = %d, want %d", tc.picks, got, tc.want)
		}
	}
	if _, one, _ := ParseQuiz("Is HTML a language?\n- [x] yes\n- [ ] no\n"); len(one) != 1 || one[0].Title != "Is HTML a language?" {
		t.Errorf("a quiz without headings is one question: %+v", one)
	}
	if n, ok := ParsePass("70%"); !ok || n != 70 {
		t.Error("pass 70%")
	}
	if _, ok := ParsePass("170%"); ok {
		t.Error("pass over 100")
	}
}

func TestVideoSeconds(t *testing.T) {
	// a minimal MP4: ftyp, then moov holding a version-0 mvhd of 90 s at 1000/s
	mvhd := make([]byte, 8+100)
	copy(mvhd, []byte{0, 0, 0, 108, 'm', 'v', 'h', 'd'})
	mvhd[8+12+2], mvhd[8+12+3] = 0x03, 0xe8 // timescale 1000
	mvhd[8+16+1], mvhd[8+16+2], mvhd[8+16+3] = 0x01, 0x5f, 0x90
	moov := append([]byte{0, 0, 0, byte(8 + len(mvhd)), 'm', 'o', 'o', 'v'}, mvhd...)
	file := append([]byte{0, 0, 0, 8, 'f', 't', 'y', 'p'}, moov...)
	p := filepath.Join(t.TempDir(), "v.mp4")
	os.WriteFile(p, file, 0o644)
	if s := VideoSeconds(p); s != 90 {
		t.Errorf("VideoSeconds = %v, want 90", s)
	}
	if m := videoMinutes(p); m != 2 {
		t.Errorf("videoMinutes = %d, want 2", m)
	}
}
