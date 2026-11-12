package webdemo

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
)

// The si 3 steps: transitions at scene cuts, camera moves (zoom to, pan to,
// zoom out), spotlight (alias focus), window and drag. The spec is
// docs/superpowers/specs/2026-11-12-si-3.md.

// Transition is how one scene gives way to the next in a video.
type Transition struct {
	Kind string  // cut, fade, slide-left, slide-right, slide-up, slide-down, wipe-…, pop, zoom, circle
	Dur  float64 // seconds (0 for cut)
}

func (t Transition) String() string {
	if t.Kind == "cut" {
		return "cut"
	}
	return fmt.Sprintf("%s %gs", t.Kind, t.Dur)
}

// TransitionNames are the transitions a script may name (slide and wipe
// are slide-left and wipe-left).
var TransitionNames = []string{"cut", "fade", "slide", "slide-left", "slide-right", "slide-up", "slide-down",
	"wipe", "wipe-left", "wipe-right", "wipe-up", "wipe-down", "pop", "zoom", "circle"}

// DefaultTransition is the cut every scene change gets unless the script,
// --transition or the step names another.
var DefaultTransition = Transition{"fade", 0.4}

// ParseTransition reads `fade`, `slide 0.6s`, `cut`.
func ParseTransition(s string) (Transition, error) {
	name, rest := cutWord(strings.ToLower(strings.TrimSpace(s)))
	if !slices.Contains(TransitionNames, name) {
		return Transition{}, fmt.Errorf("unknown transition %q (%s)", name, strings.Join(TransitionNames, ", "))
	}
	t := Transition{Kind: name, Dur: DefaultTransition.Dur}
	if name == "slide" || name == "wipe" {
		t.Kind = name + "-left"
	}
	if rest != "" {
		d, ok := duration(rest)
		if !ok || d <= 0 || d > 5*time.Second {
			return Transition{}, fmt.Errorf("transition %q: a length from 0.1s to 5s (fade 0.6s)", s)
		}
		t.Dur = d.Seconds()
	}
	if t.Kind == "cut" {
		t.Dur = 0
	}
	return t, nil
}

// transClauses reads the · clauses after a go, browser, terminal or window
// step: one transition at most.
func transClauses(st *Step, cs []string) error {
	for _, c := range cs {
		if st.Trans != nil {
			return fmt.Errorf("%q: one transition per step", st.Text)
		}
		t, err := ParseTransition(c)
		if err != nil {
			return fmt.Errorf("%q: %v", st.Text, err)
		}
		st.Trans = &t
	}
	return nil
}

// regionValue reads a region: value: a selector, optionally · zoom.
func regionValue(v string) (string, bool) {
	cs := strings.Split(v, "·") // a selector has no ·
	for i := range cs {
		cs[i] = strings.TrimSpace(cs[i])
	}
	zoom := len(cs) > 1 && cs[len(cs)-1] == "zoom"
	if zoom {
		cs = cs[:len(cs)-1]
	}
	return strings.Trim(strings.Join(cs, " · "), `"'`), zoom
}

// windowStep reads `window "name or title"`, `window 2`, `window main`.
func windowStep(st *Step, rest string) (*Step, error) {
	parts := clauses(rest)
	head := parts[0]
	st.Kind = "window"
	switch n, err := strconv.Atoi(head); {
	case head == "main":
		st.Name = "main"
	case err == nil && n >= 1:
		st.Code = n
	default:
		q, tail, err := quoted(head)
		if err != nil || q == "" || tail != "" {
			return nil, fmt.Errorf(`%q: write window "name or title", window 2 or window main`, st.Text)
		}
		st.Value.Text = q
	}
	return st, transClauses(st, parts[1:])
}

// point reads X,Y (CSS pixels).
func point(s string) (*[2]float64, bool) {
	x, y, ok := strings.Cut(s, ",")
	if !ok {
		return nil, false
	}
	fx, err1 := strconv.ParseFloat(strings.TrimSpace(x), 64)
	fy, err2 := strconv.ParseFloat(strings.TrimSpace(y), 64)
	return &[2]float64{fx, fy}, err1 == nil && err2 == nil
}

// dragEnd reads one end of a drag: an element or X,Y; it returns the rest.
func dragEnd(s string) (Locator, *[2]float64, string, error) {
	s = strings.TrimSpace(s)
	if s != "" && (s[0] == '-' || (s[0] >= '0' && s[0] <= '9')) {
		w, r := cutWord(s)
		if p, ok := point(w); ok {
			return Locator{}, p, r, nil
		}
		return Locator{}, nil, s, fmt.Errorf("bad point")
	}
	l, r, err := locator(s)
	return l, nil, r, err
}

// dragStep reads `drag from "A" to "B"`, `drag from X,Y to X,Y`, `drag by DX,DY`.
func dragStep(st *Step, rest string) (*Step, error) {
	bad := fmt.Errorf(`%q: write drag from "Element" to "Element", drag from 400,300 to 900,420 or drag by 300,80`, st.Text)
	st.Kind, st.Drag = "drag", &Drag{}
	w, r := cutWord(rest)
	switch w {
	case "by":
		p, ok := point(r)
		if !ok {
			return nil, bad
		}
		st.Drag.By = p
		return st, nil
	case "from":
		var err error
		var tail string
		if st.Drag.From, st.Drag.FromXY, tail, err = dragEnd(r); err != nil {
			return nil, bad
		}
		to, r2 := cutWord(tail)
		if to != "to" {
			return nil, bad
		}
		if st.Drag.To, st.Drag.ToXY, tail, err = dragEnd(r2); err != nil || tail != "" {
			return nil, bad
		}
		return st, nil
	}
	return nil, bad
}

// cameraStep reads `zoom to "El"`, `pan to "El"`, `zoom out` (· 0.8s).
func cameraStep(st *Step, word, rest string) (*Step, error) {
	parts := clauses(rest)
	w, r := cutWord(parts[0])
	switch {
	case word == "zoom" && w == "out" && r == "":
		st.Kind = "zoom-out"
	case w == "to":
		st.Kind = word
		var err error
		var tail string
		if st.Target, tail, err = locator(r); err != nil || tail != "" {
			return nil, fmt.Errorf(`%q: write %s to "Element" (· 0.8s)`, st.Text, word)
		}
	default:
		return nil, fmt.Errorf(`%q: write zoom to "Element", pan to "Element" or zoom out (· 0.8s)`, st.Text)
	}
	for _, c := range parts[1:] {
		d, ok := duration(c)
		if !ok || d <= 0 || d > 10*time.Second {
			return nil, fmt.Errorf("%q: after ·, the move's length: 0.8s", st.Text)
		}
		st.Dur = d
	}
	return st, nil
}

// spotStep reads `spotlight "El" · circle · blur · label "Text" · for 3s`
// and `spotlight off`.
func spotStep(st *Step, rest string) (*Step, error) {
	parts := clauses(rest)
	st.Kind, st.Opts = "spotlight", map[string]string{}
	if parts[0] == "off" && len(parts) == 1 {
		st.Opts["off"] = "1"
		return st, nil
	}
	var err error
	var tail string
	if st.Target, tail, err = locator(parts[0]); err != nil || tail != "" {
		return nil, fmt.Errorf(`%q: write spotlight "Element" (· circle · blur · label "Text" · for 3s) or spotlight off`, st.Text)
	}
	for _, c := range parts[1:] {
		k, v := cutWord(c)
		switch {
		case c == "circle":
			st.Opts["shape"] = "circle"
		case c == "blur":
			st.Opts["look"] = "blur"
		case k == "label":
			q, t, err := quoted(v)
			if err != nil || q == "" || t != "" {
				return nil, fmt.Errorf(`%q: write · label "Text"`, st.Text)
			}
			st.Value.Text = q
		case k == "for":
			d, ok := duration(v)
			if !ok || d <= 0 {
				return nil, fmt.Errorf("%q: write · for 3s", st.Text)
			}
			st.For = d
		default:
			return nil, fmt.Errorf(`%q: a spotlight takes · circle, · blur, · label "Text" and · for 3s`, st.Text)
		}
	}
	return st, nil
}
