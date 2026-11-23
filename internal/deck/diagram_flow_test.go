package deck

import (
	"strings"
	"testing"
)

func elsOf(d *Diagram, cls string) []int {
	var out []int
	for i, e := range d.Els {
		if strings.Contains(e.C, cls) {
			out = append(out, i)
		}
	}
	return out
}

func clipOf(d *Diagram) int {
	for i, e := range d.Els {
		if e.K == "clip" {
			return i
		}
	}
	return -1
}

func TestCFD(t *testing.T) {
	d, probs := ParseDiagram("cfd", nil, "| Day | Backlog | Doing | Done |\n|---|---|---|---|\n| 1 | 10 | 2 | 0 |\n| 2 | 9 | 2 | 1 |\n| 3 | 8 | x | 2 |")
	if len(probs) != 1 || !strings.Contains(probs[0], `"x" is not a number`) {
		t.Errorf("problems %q", probs)
	}
	if b := elsOf(d, "si-dg-band"); len(b) != 3 || !strings.Contains(d.Els[b[0]].C, "is-cb") || !strings.Contains(d.Els[b[2]].C, "is-c0") || !d.Els[b[0]].Clip {
		t.Errorf("bands %v", b)
	}
	c := clipOf(d)
	if c < 0 || len(d.States) != 2 || d.States[1][c].W <= d.States[0][c].W || d.In[c].W >= d.States[0][c].W {
		t.Fatalf("the clip widens step by step: clip %d states %d", c, len(d.States))
	}
	if len(d.Facts) != 1 || d.Facts[0] != "Throughput 1 item a day" {
		t.Errorf("facts %q", d.Facts)
	}
	if _, p := ParseDiagram("cfd", nil, "| Day | Done |\n| 1 | 1 |"); len(p) == 0 {
		t.Error("a cfd needs two states and two rows")
	}
}

func TestBurndown(t *testing.T) {
	d, probs := ParseDiagram("burndown", map[string]string{"days": "10"}, "| Day | Remaining |\n|--|--|\n| 1 | 40 |\n| 2 | 36 |\n| 3 | 32 |")
	if len(probs) != 0 || len(d.Steps) != 2 {
		t.Fatalf("problems %q steps %q", probs, d.Steps)
	}
	if len(d.Facts) != 1 || d.Facts[0] != "Burned 8 of 40 in 2 days; at this pace done by day 11 (plan: 10)" {
		t.Errorf("facts %q", d.Facts)
	}
	if len(elsOf(d, "si-dg-ideal")) != 1 || len(elsOf(d, "si-dg-actual")) != 1 {
		t.Error("an ideal and an actual line")
	}
}

func TestCycleTime(t *testing.T) {
	body := "A-1 · 2026-12-03 · 3d\nA-2 · 2026-12-01 · 20d\nA-3 · 2026-12-02 · 2\nA-4 · 2026-12-04 · 5d\nA-5 · 2026-12-05 · 4d\nA-6 · 2026-12-06 · 6d\nA-7 · 2026-12-07 · 3d\nA-8 · 2026-12-08 · 7d\nnope"
	d, probs := ParseDiagram("cycle-time", nil, body)
	if len(probs) != 1 {
		t.Errorf("problems %q", probs)
	}
	dots, pct := elsOf(d, "si-dg-dot"), elsOf(d, "si-dg-pct")
	if len(dots) != 8 || len(pct) != 3 {
		t.Fatalf("dots %v percentiles %v", dots, pct)
	}
	if !strings.Contains(d.Els[dots[0]].C, "is-late") { // sorted by date: A-2 (10 days) first, past the 85th
		t.Error("the slow item is marked")
	}
	last := d.States[len(d.States)-1]
	if d.States[0][pct[1]].O != 0 || last[pct[1]].O != 1 || last[pct[1]].T != "85% · 7 d" {
		t.Errorf("the percentile lines come last: %+v", last[pct[1]])
	}
	if d.Facts[0] != "85% of 8 items finished within 7 days; half within 4" {
		t.Errorf("facts %q", d.Facts)
	}
}

func TestThroughput(t *testing.T) {
	d, _ := ParseDiagram("throughput", nil, "| Week | Done |\n|--|--|\n| 45 | 6 |\n| 46 | 9 |\n| 47 | 9 |")
	cols := elsOf(d, "si-dg-col")
	if len(cols) != 3 || len(d.States) != 4 || d.States[0][cols[1]].H != 0 || d.States[2][cols[1]].H == 0 {
		t.Fatalf("columns grow one per step: %v %d", cols, len(d.States))
	}
	if d.Facts[0] != "24 finished in 3 weeks, 8 on average" {
		t.Errorf("facts %q", d.Facts)
	}
}
