package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

// A recording paces the video: each view's spoken time (a view visited twice
// adds up), less the transition into it after the first; latest finds the
// newest recording of that deck only.
func TestNarration(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "talk.md")
	os.WriteFile(src, []byte("# Talk\n"), 0o644)
	rec := func(name, timeline string) string {
		d := filepath.Join(dir, ".si", "recordings", name)
		os.MkdirAll(d, 0o755)
		deck := name[:len(name)-len("-2026-11-09-100000")] + ".md"
		os.WriteFile(filepath.Join(d, "meta.json"), []byte(`{"deck":"`+deck+`","audio":"audio.webm","seconds":10}`), 0o644)
		os.WriteFile(filepath.Join(d, "timeline.json"), []byte(timeline), 0o644)
		return d
	}
	rec("talk-2026-11-09-100000", `[{"t":0,"slide":0,"step":0}]`)
	newest := rec("talk-2026-11-09-120000", `[{"t":0,"slide":0,"step":0},{"t":3,"slide":1,"step":0},{"t":5,"slide":0,"step":0},{"t":6,"slide":1,"step":0}]`)
	rec("other-2026-11-10-120000", `[{"t":0,"slide":0,"step":0}]`)
	if got := findRecording("latest", src); got != newest {
		t.Fatalf("latest: %s", got)
	}
	if got := findRecording(filepath.Join(dir, ".si", "recordings", "other-2026-11-10-120000"), src); got != "" {
		t.Errorf("a recording of another deck: %s", got)
	}
	n, err := loadNarration(newest)
	if err != nil {
		t.Fatal(err)
	}
	// slide 0: 0-3 and 5-6 = 4 s; slide 1: 3-5 and 6-10 = 6 s
	if h, ok := n.hold(0, 0, 1.3, true); !ok || h != 4 {
		t.Errorf("first view: %v %v", h, ok)
	}
	if h, ok := n.hold(1, 0, 1.3, false); !ok || h < 4.69 || h > 4.71 {
		t.Errorf("a later view, less its transition: %v", h)
	}
	if _, ok := n.hold(2, 0, 1.3, false); ok {
		t.Error("a view the recording never showed")
	}
	if n.changed(src) {
		t.Error("no hash recorded: not a change")
	}
}

// The pace report: planned against spoken per slide, the script's wpm, long
// and fast marks.
func TestPaceReport(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "talk.md")
	os.WriteFile(src, []byte("# Talk\n\nNotes: (0.1 min)\none two three four five six seven eight nine ten [CLICK]\n\n---\n\n## Slow one\n\nNotes: (1 min)\nWords.\n"), 0o644)
	d, err := deck.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	n := &narration{spoken: map[[2]int]float64{{0, 0}: 12, {1, 0}: 30}}
	r := paceReport(d, n, "talk-x")
	for _, want := range []string{"0:06     0:12     50  ← long", "Slow one", "1:00     0:30      2  ← fast", "total"} {
		if !strings.Contains(r, want) {
			t.Errorf("missing %q in\n%s", want, r)
		}
	}
}
