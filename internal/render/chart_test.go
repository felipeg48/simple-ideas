package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

func chartMD(t *testing.T, src string) Result {
	t.Helper()
	p := filepath.Join(t.TempDir(), "talk.md")
	os.WriteFile(p, []byte(src), 0o644)
	d, err := deck.Parse(p)
	if err != nil {
		t.Fatal(err)
	}
	return Deck(d, Options{})
}

func TestChartFence(t *testing.T) {
	res := chartMD(t, "# Late\n\n```chart bar race unit=%\n| Cause | Q1 | Q2 | Q3 |\n|---|---|---|---|\n| Carrier | 34 | 22 | 12 |\n| Payment | 9 | 14 | 21 |\n```\n\n---\n\n# Mix\n\n```chart bar id=m\n| R | N |\n|---|---|\n| a | 1 |\n```\n\n```chart donut id=m\n| R | N |\n|---|---|\n| a | 1 |\n```\n\n```chart funnel3d\n| S | N |\n|---|---|\n| Visits | 100 |\n| Paid | 15 |\n```\n")
	page := string(res.HTML)
	if len(res.Problems) != 0 {
		t.Fatalf("problems %v", res.Problems)
	}
	for _, want := range []string{
		`class="si-figure si-chart" data-race`,
		`<p class="si-chart-at" aria-live="polite">Q3</p>`,
		`class="si-step si-chart-mark" data-step="2"`,
		`data-chart-id="m"`, `data-kind="donut"`,
		`class="si-c3" data-c3="funnel3d"`, `data-from="translate3d(0,-9em,0)"`,
		`>34%<`, // unit= after a bare number
		`.si-chart-frame{`, "siChart",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("page lacks %s", want)
		}
	}
	// a race re-sorts each step: Q3 has Payment first
	q3 := page[strings.Index(page, ">Q3<"):]
	if strings.Index(q3, "Payment") > strings.Index(q3, "Carrier") {
		t.Error("the Q3 frame is not sorted by value")
	}
	// the same id is one show: one figure, two frames, one build step
	if n := strings.Count(page, `data-chart-id="m"`); n != 1 {
		t.Errorf("show figures %d, want 1", n)
	}
	if strings.Contains(page, `si-cw`) && strings.Contains(page, `data-cw-id="m"`) {
		t.Error("a chart id= made a code window")
	}
}

func TestChartProblems(t *testing.T) {
	for src, want := range map[string]string{
		"```chart scatter\n| a | n |\n|---|---|\n| x | 1 |\n```\n":            `unknown type "scatter"`,
		"```chart bar\n| a | n |\n|---|---|\n| x | lots |\n```\n":             `"lots" (row "x") is not a number`,
		"```chart bar steps\n| a | n |\n|---|---|\n| x | 1 |\n```\n":          "steps needs two or more number columns",
		"```chart donut explode=Nope\n| a | n |\n|---|---|\n| x | 1 |\n```\n": `explode="Nope" names no row`,
		"```chart line stacked\n| a | n |\n|---|---|\n| x | 1 |\n```\n":       "stacked is for bar and column",
		"```chart bar\nno table\n```\n":                                       "write a Markdown table",
	} {
		res := chartMD(t, "# T\n\n"+src)
		if len(res.Problems) == 0 || !strings.Contains(res.Problems[0].Msg, want) {
			t.Errorf("%q: problems %v, want %q", src, res.Problems, want)
		}
	}
}
