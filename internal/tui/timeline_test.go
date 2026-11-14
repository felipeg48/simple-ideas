package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestTimelineAndInfographicAsText(t *testing.T) {
	r := &textRenderer{pal: PaletteFor("studio", "dark"), w: 80, k: 1 << 30}
	tl := "## Beginnings\n\n### 1990 · The first idea\nA sketch on a napkin.\n![A napkin](n.png)\n\n### 1994 · A prototype\nThree people.\n\n## Fuentes · sources\n- Notes\n"
	got := ansi.Strip(strings.Join(lines(r.body("`````timeline si-page\n"+tl+"`````\n")), "\n"))
	for _, want := range []string{"━━ BEGINNINGS  1990 – 1994", "1990  The first idea", "      A sketch on a napkin.", "▨ A napkin", "FUENTES", "· Notes"} {
		if !strings.Contains(got, want) {
			t.Errorf("timeline: missing %q in\n%s", want, got)
		}
	}
	ig := "## 3× · stat\nfaster\n\n## How · steps\n1. Measure\n2. Shrink\n\n## Sources · sources\n- Our data\n"
	got = ansi.Strip(strings.Join(lines(r.body("~~~~ infographic page\n"+ig+"~~~~\n")), "\n"))
	for _, want := range []string{"▄", "faster", "▍HOW · steps", "Measure", "SOURCES", "· Our data"} {
		if !strings.Contains(got, want) {
			t.Errorf("infographic: missing %q in\n%s", want, got)
		}
	}
}

func lines(bs []blk) []string {
	var out []string
	for _, b := range bs {
		out = append(out, b.lines...)
	}
	return out
}
