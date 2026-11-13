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
