package ui

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

// session prints a play start, a build and a check through one printer.
func session(v View, color bool) string {
	var b bytes.Buffer
	p := New(&b, v, color)
	p.Header("🎬", "si play 3.32.0", "examples/ · theme harbor")
	p.Item(OK, "deck", "An order's journey", "24 slides")
	p.Item(OK, "live", "watching 41 files", "reload on save")
	p.Break()
	p.Item(Info, "open", "http://127.0.0.1:7070/showcase.html", "")
	p.Hint("m monitor · q quit")
	p.Header("📦", "si build", "examples/ → dist/")
	p.Output(OK, "deck", "showcase.html", 2<<20, 400*time.Millisecond)
	p.Output(OK, "lab", "order-lab.html", 212<<10, 100*time.Millisecond)
	p.Break()
	p.Summary(OK, "Done", "2 outputs · 2.2 MB · 0.5s", "open dist/index.html")
	p.Begin("si check", "examples/")
	p.Problem(Warn, "showcase.md", 212, "2 [CLICK] cues, 3 build steps")
	p.Problem(Fail, "appendix.md", 18, "missing image")
	p.Summary(Fail, "", "1 warning · 1 problem", "")
	return b.String()
}

func TestLooks(t *testing.T) {
	for _, tc := range []struct {
		v    View
		want []string
	}{
		{Ledger, []string{"si play 3.32.0", "✓", "deck", "showcase.html", "2.0 MB", "0.4s", "Done", "open dist/index.html", "showcase.md:212", "✗"}},
		{Badges, []string{" DECK ", " OPEN ", "  OK  ", " DONE ", " WARN ", "showcase.md:212\n          2 [CLICK]", " FAIL "}},
		{Checklist, []string{"🎬", "✅", "Deck", "🎉", "    open dist/index.html", "showcase.md line 212", "\n    2 [CLICK]", "❌"}},
		{Steps, []string{"◆", "├─", "│", "└─", "▇▇▇▇", "showcase.md", ":212", "2 [CLICK]"}},
	} {
		got := session(tc.v, true)
		for _, w := range tc.want {
			if !strings.Contains(got, w) {
				t.Errorf("%s: no %q in\n%s", tc.v, w, got)
			}
		}
		if !strings.Contains(got, "\x1b[") {
			t.Errorf("%s: no colour on a terminal", tc.v)
		}
		if plain := session(tc.v, false); strings.Contains(plain, "\x1b") {
			t.Errorf("%s: escape codes without colour:\n%q", tc.v, plain)
		}
		if strings.Contains(got, "\n\n\n") {
			t.Errorf("%s: two blank lines in a row:\n%s", tc.v, got)
		}
	}
}

func TestStepsTreeEnds(t *testing.T) {
	got := session(Steps, false)
	// the last item of each tree is └─, never ├─ followed by nothing
	for _, w := range []string{"└─ ▶ open http://127.0.0.1:7070/showcase.html", "└─ done 2 outputs", "└─ 1 warning · 1 problem"} {
		if !strings.Contains(got, w) {
			t.Errorf("no %q in\n%s", w, got)
		}
	}
}

func TestParseView(t *testing.T) {
	for _, v := range []string{"ledger", "Badges", " checklist ", "steps"} {
		if _, err := ParseView(v); err != nil {
			t.Errorf("%q: %v", v, err)
		}
	}
	_, err := ParseView("fancy")
	if err == nil || !strings.Contains(err.Error(), "ledger, badges, checklist or steps") {
		t.Errorf("a bad view: %v", err)
	}
}

func TestMonitor(t *testing.T) {
	at := time.Date(2026, 11, 17, 12, 40, 2, 0, time.UTC)
	s := Stats{Target: "examples/", Up: 42*time.Minute + 17*time.Second, CPU: 4.1, Mem: 38 << 20,
		CPUHist: []float64{1, 2, 4.1}, MemHist: []float64{30 << 20, 38 << 20}, Shells: 2, Pages: 3, Files: 41,
		Open: "http://127.0.0.1:7070/", Phone: "http://192.168.1.24:7070/", Slide: 7, Slides: 24, Step: 2, Steps: 3,
		Roles: map[string]int{"deck": 1, "audience": 2}, SlideFor: 72 * time.Second, TalkFor: 14*time.Minute + 3*time.Second,
		Web: "http://127.0.0.1:7070/_si/monitor?k=0123456789abcdef0123456789abcdef",
		Log: []LogLine{{at, OK, "reloaded"}, {at, Warn, "appendix.md:18: no alt text"}}, Keys: "q quit · m hide"}
	for _, tc := range []struct {
		v    View
		want []string
	}{
		{Ledger, []string{"up 00:42:17", "4.1%", "38 MB", "2 running", "41 watched", "192.168.1.24", "7 / 24", "step 2/3", "12:40:02", "q quit"}},
		{Badges, []string{"[CPU]", "[MEM]", "[SHELLS] 2", "[3 VIEWERS]", "[SLIDE 7/24]", "[RELOAD]", "[WARN]"}},
		{Checklist, []string{"running 42 min", "🔥  CPU", "🧠  Memory", "📱", "Slide 7 of 24", "🔄", "⚠️"}},
		{Steps, []string{"╭─ si play", "SERVER", "PRESENTING", "slide  7/24", "LINKS", "LOG", "╰─ q quit"}},
	} {
		got := Monitor(tc.v, s, false)
		for _, w := range append(tc.want, "deck 1 · audience 2", "on this slide 01:12 · talk 00:14:03", "_si/monitor") {
			if !strings.Contains(got, w) {
				t.Errorf("%s: no %q in\n%s", tc.v, w, got)
			}
		}
		if strings.Contains(got, "\x1b") {
			t.Errorf("%s: escape codes without colour", tc.v)
		}
		if tc.v == Steps { // the box is square
			for _, l := range strings.Split(strings.TrimSpace(got), "\n") {
				if Width(l) != 76 {
					t.Errorf("box line %d wide: %q", Width(l), l)
				}
			}
		}
	}
	// no deck has reported yet: no slide row
	s.Slides = 0
	if got := Monitor(Ledger, s, false); strings.Contains(got, "slide") {
		t.Errorf("a slide row with no position:\n%s", got)
	}
}
