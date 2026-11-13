package deck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseYear(t *testing.T) {
	for _, c := range []struct {
		in   string
		want float64
		ok   bool
	}{
		{"c. 300 a.C.", -300, true},
		{"240 BC", -240, true},
		{"ca. 100 BCE", -100, true},
		{"c. 825", 825, true},
		{"1202", 1202, true},
		{"2022–2024", 2022, true},
		{"2022-2024", 2022, true},
		{"1936-05", 1936 + 4.0/12, true},
		{"siglo IX", 0, false},
	} {
		got, ok := ParseYear(c.in)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("ParseYear(%q) = %v, %v; want %v, %v", c.in, got, ok, c.want, c.ok)
		}
	}
}

const tlSample = `## Antigüedad · color=#b3541e

### c. 300 a.C. · Algoritmo de Euclides
El máximo común divisor.
![Euclides](img/e.jpg "Dominio público")

### 1202 · Fibonacci

## Edad Media

### c. 825 · Al-Juarismi
Da nombre al algoritmo.

## Fuentes · sources
- Cormen (2022).
- Knuth (1968).
`

func TestParseTimeline(t *testing.T) {
	tl, probs, _ := ParseTimeline(tlSample, "t.md", 1)
	if len(probs) > 0 {
		t.Fatalf("problems: %v", probs)
	}
	if len(tl.Eras) != 2 || tl.Eras[0].Color != "#b3541e" || tl.Eras[1].Name != "Edad Media" {
		t.Fatalf("eras: %+v", tl.Eras)
	}
	ms := tl.Milestones()
	if len(ms) != 3 || ms[0].Date != "c. 300 a.C." || ms[0].Title != "Algoritmo de Euclides" || ms[0].Text != "El máximo común divisor." {
		t.Fatalf("milestones: %+v", ms[0])
	}
	if ms[0].Img != "img/e.jpg" || ms[0].Caption != "Euclides" || ms[0].Credit != "Dominio público" {
		t.Errorf("picture: %+v", ms[0])
	}
	if tl.SourcesTitle != "Fuentes" || len(tl.Sources) != 2 {
		t.Errorf("sources: %q %v", tl.SourcesTitle, tl.Sources)
	}
	if got := tl.Summary(); got != "3 milestones, 1 pictures, 2 eras, 2 sources" {
		t.Errorf("summary %q", got)
	}
	probs, warns := tl.Check("t.md", t.TempDir())
	all := ""
	for _, p := range append(probs, warns...) {
		all += p.String() + "\n"
	}
	for _, want := range []string{"out of chronological order", "picture img/e.jpg not found", `"Fibonacci" has no description`} {
		if !strings.Contains(all, want) {
			t.Errorf("check lacks %q:\n%s", want, all)
		}
	}
	r, err := ParseRequire("20 dates, 1 images, 2 eras")
	if err != nil {
		t.Fatal(err)
	}
	if un := tl.Unmet(r); len(un) != 1 || !strings.Contains(un[0], "20 dates") {
		t.Errorf("unmet %v", un)
	}
	if _, err := ParseRequire("lots of pictures"); err == nil {
		t.Error("ParseRequire accepted nonsense")
	}
}

func TestTimelinePage(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "t.md")
	src := "---\ntype: timeline\ntitle: T\nlayout: snake\nclosing: Fin.\nrequire: 2 dates\n---\n\n## Era\n\n### 1900 · A\nUno.\n\n### 1950 · B\nDos.\n\nNotes: Hola. [CLICK]\n"
	if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	d, err := Parse(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Problems) > 0 || d.IsDoc() || len(d.Slides) != 1 || d.Timeline == nil {
		t.Fatalf("problems %v, doc %v, slides %d", d.Problems, d.IsDoc(), len(d.Slides))
	}
	s := d.Slides[0]
	if s.Notes != "Hola. [CLICK]" || !strings.Contains(s.Body, "timeline si-page") || d.Meta.Timeline.Closing != "Fin." {
		t.Errorf("slide %+v", s)
	}
	// portrait is a document; a bad layout is a problem; a timeline key elsewhere a warning
	os.WriteFile(p, []byte(strings.Replace(src, "layout: snake", "orientation: portrait\nlayout: radial", 1)), 0o644)
	d, _ = Parse(p)
	if !d.IsDoc() || len(d.Problems) != 1 || !strings.Contains(d.Problems[0].Msg, "timeline layout") {
		t.Errorf("portrait: doc %v, problems %v", d.IsDoc(), d.Problems)
	}
	os.WriteFile(p, []byte("---\nclosing: x\n---\n# Deck\n"), 0o644)
	d, _ = Parse(p)
	if len(d.Warnings) != 1 || !strings.Contains(d.Warnings[0].Msg, "for timelines") {
		t.Errorf("deck warnings %v", d.Warnings)
	}
}
