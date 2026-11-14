package render

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

var tlRE = regexp.MustCompile(`(?s)<div class="si-tl".*?</footer>\n</div>\n`)

// TestTimelineGolden renders testdata/timeline.md (a type: timeline page)
// and compares its timeline markup with testdata/timeline.golden.html.
func TestTimelineGolden(t *testing.T) {
	d, err := deck.Parse(filepath.Join("testdata", "timeline.md"))
	if err != nil {
		t.Fatal(err)
	}
	res := Deck(d, Options{})
	if len(res.Problems) > 0 {
		t.Fatalf("problems: %v", res.Problems)
	}
	if res.Steps != 2 {
		t.Errorf("steps = %d, want one per era after the first (2)", res.Steps)
	}
	page := string(res.HTML)
	for _, want := range []string{`<style id="si-tl-css">`, `window.siTimelineFit`, `class="si-slide si-content si-tl-slide"`} {
		if !strings.Contains(page, want) {
			t.Errorf("page lacks %s", want)
		}
	}
	got := tlRE.FindString(page)
	golden := filepath.Join("testdata", "timeline.golden.html")
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
		t.Fatalf("timeline golden mismatch (run go test ./internal/render -update if intended)\n%s", got)
	}
}

func TestTimelineFence(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "d.md")
	src := "# Deck\n\n---\n\n## History\n\n```timeline layout=line steps\n## One\n### 1900 · A\nUno.\n## Two\n### 1950 · B\nDos.\n![Pic](missing.png)\n```\n\n---\n\n## Bad\n\n```timeline wobble=1\n### 1900 · A\n```\n"
	if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	d, err := deck.Parse(p)
	if err != nil {
		t.Fatal(err)
	}
	res := Deck(d, Options{})
	all := ""
	for _, pr := range res.Problems {
		all += pr.Msg + "\n"
	}
	for _, want := range []string{"picture missing.png not found", `unknown fence option "wobble"`} {
		if !strings.Contains(all, want) {
			t.Errorf("problems lack %q:\n%s", want, all)
		}
	}
	if res.Steps != 1 || !strings.Contains(string(res.HTML), `data-layout="line"`) {
		t.Errorf("steps %d", res.Steps)
	}
}

// TestTimelineEraAndCardLayouts renders testdata/timeline.md as eras and as
// cards and compares each with its golden file.
func TestTimelineEraAndCardLayouts(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "timeline.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, layout := range []string{"eras", "cards"} {
		dir := t.TempDir()
		p := filepath.Join(dir, "timeline.md")
		src := strings.Replace(string(raw), "type: timeline\n", "type: timeline\nlayout: "+layout+"\n", 1)
		if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		d, err := deck.Parse(p)
		if err != nil {
			t.Fatal(err)
		}
		res := Deck(d, Options{})
		if len(res.Problems) > 0 {
			t.Fatalf("%s: problems: %v", layout, res.Problems)
		}
		if res.Steps != 2 {
			t.Errorf("%s: steps = %d, want one per era after the first (2)", layout, res.Steps)
		}
		got := tlRE.FindString(string(res.HTML))
		if !strings.Contains(got, `data-layout="`+layout+`" data-orient="landscape" style="--_tl-ar:1.78" data-frame`) {
			t.Errorf("%s: no landscape frame in\n%.300s", layout, got)
		}
		golden := filepath.Join("testdata", "timeline-"+layout+".golden.html")
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
			t.Errorf("%s golden mismatch (run go test ./internal/render -update if intended)\n%s", layout, got)
		}
	}
}

func TestTimelineCardGrid(t *testing.T) {
	for _, c := range []struct{ n, cols, rows int }{{30, 10, 3}, {12, 6, 2}, {4, 4, 1}, {1, 1, 1}} {
		if cols, rows := tlCardGrid(c.n, 16.0/9); cols != c.cols || rows != c.rows {
			t.Errorf("%d cards: %d×%d, want %d×%d", c.n, cols, rows, c.cols, c.rows)
		}
	}
}

// TestTimelinePortraitErasHaveNoFrame: a portrait eras timeline is a poster
// that flows (no 16:9 frame).
func TestTimelinePortraitErasHaveNoFrame(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "t.md")
	os.WriteFile(p, []byte("---\ntype: timeline\nlayout: eras\norientation: portrait\n---\n\n## One\n\n### 1900 · A\nUno.\n\n## Two\n\n### 1950 · B\nDos.\n"), 0o644)
	d, err := deck.Parse(p)
	if err != nil {
		t.Fatal(err)
	}
	page := string(Deck(d, Options{}).HTML)
	if !strings.Contains(page, `data-layout="eras" data-orient="portrait"`) || strings.Contains(page, `" data-frame`) {
		t.Errorf("portrait eras: want a flowing poster")
	}
	if !strings.Contains(page, `<header class="si-tl-era-head"><h2 class="si-tl-era-name">One</h2><p class="si-tl-era-span">1900</p></header>`) {
		t.Errorf("no era header with its dates")
	}
}
