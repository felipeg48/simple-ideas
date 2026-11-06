package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

func showcase(t *testing.T) *deck.Deck {
	t.Helper()
	d, err := deck.Parse("../../examples/showcase.md")
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// slideOf finds the slide whose title starts with prefix.
func slideOf(t *testing.T, d *deck.Deck, prefix string) int {
	for i, s := range d.Slides {
		if strings.HasPrefix(stripMarks(s.Title), prefix) {
			return i
		}
	}
	t.Fatalf("no slide %q", prefix)
	return -1
}

func TestCanvasStatesFollowTheBrowserRules(t *testing.T) {
	d := showcase(t)
	i := slideOf(t, d, "Payment: declined")
	cur, prev := canvasStates(d, i, 0)
	st := strings.Join(cur["assets/journey.svg"], " ")
	if !strings.Contains(st, "declined") || !strings.Contains(st, "placed") {
		t.Errorf("relative +declined on top of what the last slide left: %q", st)
	}
	if prev["assets/journey.svg"] == nil {
		t.Error("the state it comes from: the previous slide's")
	}
	_, steps := Slide(d, i, 0, 120, 34, PaletteFor("", ""))
	last, _ := canvasStates(d, i, steps)
	if strings.Join(last["assets/journey.svg"], " ") == st {
		t.Error("the builds change the canvas state")
	}
}

func TestCanvasAnimates(t *testing.T) {
	d := showcase(t)
	pal := PaletteFor("", "")
	i := slideOf(t, d, "The journey on one screen")
	cur, prev := canvasStates(d, i, 0)
	at := func(T, P float64) string {
		s, _ := Slide(d, i, 0, 120, 34, pal, &SlideOpts{T: T, States: cur, Prev: prev, P: P})
		return plain(s)
	}
	if at(0, 1) == at(0.5, 1) {
		t.Error("packets travel along the live wires")
	}
	if !strings.Contains(at(0, 1), "order #4711") || !strings.Contains(at(0, 1), "warehouse") {
		t.Errorf("the canvas draws its stations and token:\n%s", at(0, 1))
	}
	// the token slides from payments to the warehouse on the build that packs the order
	j := slideOf(t, d, "The warehouse falls behind")
	c1, p1 := canvasStates(d, j, 2)
	mid, _ := Slide(d, j, 2, 120, 34, pal, &SlideOpts{States: c1, Prev: p1, P: 0.3})
	end, _ := Slide(d, j, 2, 120, 34, pal, &SlideOpts{States: c1, Prev: p1, P: 1})
	col := func(s string) int {
		for _, l := range strings.Split(plain(s), "\n") {
			if k := strings.Index(l, "order #4711"); k >= 0 {
				return k
			}
		}
		return -1
	}
	if a, b := col(mid), col(end); a < 0 || b < 0 || a >= b {
		t.Errorf("mid-move the token is left of where it lands: %d → %d", a, b)
	}
}

func TestPresenterTicksOnlyForMovingSlides(t *testing.T) {
	d := showcase(t)
	m := &presenter{d: d, pal: PaletteFor("", ""), w: 120, h: 34, start: time.Now(), slideStart: time.Now()}
	m.Update(frameMsg{})
	if m.moving {
		t.Error("the title slide has nothing that moves")
	}
	m.goTo(slideOf(t, d, "The journey on one screen"))
	m.states = nil // as a key press does: the update after a move recomputes them
	m.Update(frameMsg{})
	if !m.moving || len(m.states) == 0 {
		t.Error("a canvas slide animates")
	}
}
