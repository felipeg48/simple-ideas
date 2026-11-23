package deck

import (
	"strings"
	"testing"
)

func card(d *Diagram, k int, title string) DState {
	for _, s := range d.States[k] {
		if strings.HasPrefix(s.T, title) {
			return s
		}
	}
	return DState{}
}

func TestKanban(t *testing.T) {
	d, probs := ParseDiagram("kanban", map[string]string{"wip": "Doing:1"}, `columns: Backlog, Doing, Done
A-1 First · Backlog · @ada
A-2 Second · Doing

day 2
A-1 → Doing
A-2 blocked: waiting

day 3
A-2 unblocked
A-2 done
+ A-3 Third · Backlog
A-9 → Doing
A-1 → Nowhere`)
	if len(probs) != 2 || !strings.Contains(probs[0], `"A-9" is not a card`) || !strings.Contains(probs[1], `"Nowhere" is not a column`) {
		t.Errorf("problems %q", probs)
	}
	if len(d.States) != 3 || len(d.Steps) != 2 || d.Steps[0] != "day 2" {
		t.Fatalf("states %d steps %q", len(d.States), d.Steps)
	}
	a1, a1b := card(d, 0, "A-1 First"), card(d, 1, "A-1 First")
	if a1b.X <= a1.X || !strings.Contains(a1b.C, "is-on") {
		t.Errorf("A-1 should move right and be marked: %+v → %+v", a1, a1b)
	}
	if b := card(d, 1, "A-2"); !strings.Contains(b.C, "is-blocked") || b.U != "⚠ waiting" {
		t.Errorf("blocked card %+v", b)
	}
	if c := card(d, 2, "A-2"); !strings.Contains(c.C, "is-done") || strings.Contains(c.C, "is-blocked") {
		t.Errorf("done card %+v", c)
	}
	if card(d, 1, "A-3").O != 0 || card(d, 2, "A-3").O != 1 {
		t.Error("a card added in day 3 is hidden before it")
	}
	if len(d.Facts) != 1 || !strings.Contains(d.Facts[0], "Doing in day 2") {
		t.Errorf("facts %q", d.Facts)
	}
	if _, probs := ParseDiagram("kanban", nil, "A-1 · Doing"); len(probs) == 0 {
		t.Error("a board without columns")
	}
}

func TestGantt(t *testing.T) {
	d, probs := ParseDiagram("gantt", map[string]string{"start": "2026-12-01"}, `section Build
Design · 5d
API · 8d · after Design
Web · 1w · after Design
Tests · 4d · after API, Web
Launch · milestone · after Tests
today 2026-12-10 · Design done, API 50%, Nope 3%`)
	if len(probs) != 1 || !strings.Contains(probs[0], `"Nope" is not a task`) {
		t.Errorf("problems %q", probs)
	}
	if len(d.Facts) != 2 || d.Facts[0] != "Ends Dec 18, 2026, 17 days from the start" || d.Facts[1] != "Critical path: Design → API → Tests → Launch" {
		t.Errorf("facts %q", d.Facts)
	}
	var bars []int
	for i, e := range d.Els {
		if e.K == "bar" {
			bars = append(bars, i)
		}
	}
	if len(bars) != 4 || !strings.Contains(d.Els[bars[2]].C, "si-dg-bar") || strings.Contains(d.Els[bars[2]].C, "is-crit") || !strings.Contains(d.Els[bars[1]].C, "is-crit") {
		t.Fatalf("bars %v", bars)
	}
	s := d.States[1]
	if s[bars[0]].C != "is-done" || s[bars[1]].P != .5 || s[bars[1]].C != "" {
		t.Errorf("step 1: design %+v api %+v", s[bars[0]], s[bars[1]])
	}
	if d.In[bars[1]].W != 0 || d.States[0][bars[1]].W == 0 {
		t.Error("bars grow in from nothing")
	}
	if _, p := ParseDiagram("gantt", map[string]string{"start": "2026-12-01"}, "A · 2d · after B\nB · 1d · after A"); len(p) == 0 {
		t.Error("a loop of waits")
	}
}

func TestSequence(t *testing.T) {
	d, probs := ParseDiagram("sequence", nil, `Browser → Shop: POST /orders
Shop → Payments: authorise · 380 ms
Payments --> Shop: ok
note Shop: stock reserved
Shop -> Shop: log
nonsense`)
	if len(probs) != 1 {
		t.Errorf("problems %q", probs)
	}
	if len(d.Steps) != 5 || len(d.States) != 6 {
		t.Fatalf("steps %d states %d", len(d.Steps), len(d.States))
	}
	var msgs []int
	for i, e := range d.Els {
		if strings.Contains(e.C, "si-dg-msg") || strings.Contains(e.C, "si-dg-note") {
			msgs = append(msgs, i)
		}
	}
	if len(msgs) != 5 || !strings.Contains(d.Els[msgs[2]].C, "is-reply") || !strings.Contains(d.Els[msgs[4]].C, "is-self") {
		t.Fatalf("messages %v", msgs)
	}
	if d.States[0][msgs[0]].O != 0 || d.States[2][msgs[1]].O != 1 || d.States[2][msgs[1]].U != "380 ms" || d.States[2][msgs[2]].O != 0 {
		t.Error("messages appear one per step")
	}
}
