package webdemo

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// The runner's side of the si 3 steps: the scene cuts a take records,
// camera moves, the spotlight, window and drag.

// transition is the cut a step makes: its own, else --transition, else the
// script's transition:, else fade 0.4s.
func (r *runner) transition(st *Step) Transition {
	if st != nil && st.Trans != nil {
		return *st.Trans
	}
	switch {
	case r.o.Transition != nil:
		return *r.o.Transition
	case r.s.Trans != nil:
		return *r.s.Trans
	}
	return DefaultTransition
}

// cutting marks a scene change starting now (with the transition of the
// step running); call what it returns when the new scene is in front.
func (r *runner) cutting() func() {
	tk := r.tk
	if tk == nil || !tk.on {
		return func() {}
	}
	c := cutAt{From: time.Since(tk.t0).Seconds(), Tr: r.tr, X: -1}
	if tk.cursor != nil {
		c.X, c.Y = tk.cursor[0], tk.cursor[1]
	}
	return func() {
		c.To = time.Since(tk.t0).Seconds()
		tk.cuts = append(tk.cuts, c)
		if n := len(tk.moves); n > 0 && tk.moves[n-1].Kind != "out" { // a new scene starts unzoomed
			tk.moves = append(tk.moves, move{At: c.To, Kind: "out"})
		}
	}
}

// elementBox is the box of the element the locator found last (CSS pixels).
func (r *runner) elementBox() (box, error) {
	v, err := r.c.Eval(`(function(e){var b=e.getBoundingClientRect();return [b.left,b.top,b.width,b.height]})(window.__siT)`)
	if err != nil {
		return box{}, err
	}
	var b [4]float64
	if err := json.Unmarshal(v, &b); err != nil {
		return box{}, err
	}
	return box{X: b[0], Y: b[1], W: b[2], H: b[3], DPR: 1}, nil
}

// cameraMove records zoom to, pan to or zoom out (the element found
// already) and lets it play.
func (r *runner) cameraMove(st *Step) error {
	tk := r.tk
	if tk == nil {
		return nil
	}
	m := move{Kind: map[string]string{"zoom": "zoom", "pan": "pan", "zoom-out": "out"}[st.Kind], Dur: 0.8}
	if st.Dur > 0 {
		m.Dur = st.Dur.Seconds()
	}
	if m.Kind != "out" {
		b, err := r.elementBox()
		if err != nil {
			return err
		}
		m.B = b
	}
	m.At = time.Since(tk.t0).Seconds()
	tk.moves = append(tk.moves, m)
	time.Sleep(time.Duration(m.Dur * float64(time.Second)))
	return nil
}

// spotlight puts the spotlight on the element found (in a take).
func (r *runner) spotlight(st *Step) {
	tk := r.tk
	if tk == nil {
		return
	}
	n := &liveNote{step: st}
	if st.For > 0 {
		n.until = time.Now().Add(st.For)
	}
	tk.spot = n
	r.showSpot(n, false)
}

// showSpot draws the spotlight; again: on a new document (no shrink, its
// element found anew).
func (r *runner) showSpot(n *liveNote, again bool) {
	ms := 0.0
	if !n.until.IsZero() {
		if ms = time.Until(n.until).Seconds() * 1000; ms <= 0 {
			r.tk.spot = nil
			return
		}
	}
	if again {
		if f, err := r.locate(n.step.Target, "any", false); err != nil || !f.Found {
			return
		}
	}
	label, _ := json.Marshal(n.step.Value.Text)
	r.js(`__siDemo.spot(%s,%q,%q,%g,%t)`, label, n.step.Opts["shape"], n.step.Opts["look"], ms, again)
}

// pointer is where the mouse is (the drawn cursor in a take; the last
// mouse event, else the middle of the page, in a test).
func (r *runner) pointer() (float64, float64) {
	if r.tk != nil && r.tk.cursor != nil {
		return r.tk.cursor[0], r.tk.cursor[1]
	}
	if r.moused {
		return r.mx, r.my
	}
	return float64(r.s.W) / 2, float64(r.s.H) / 2
}

// dragEndAt is where one end of a drag is.
func (r *runner) dragEndAt(l Locator, xy *[2]float64) (float64, float64, error) {
	if xy != nil {
		return xy[0], xy[1], nil
	}
	f, err := r.element(l, "click", false)
	return f.X, f.Y, err
}

// drag presses the button at one end, moves in small steps (the drawn cursor
// with it) and lets go at the other.
func (r *runner) drag(st *Step) error {
	d := st.Drag
	var x0, y0, x1, y1 float64
	var err error
	if d.By != nil {
		x0, y0 = r.pointer()
		x1, y1 = x0+d.By[0], y0+d.By[1]
	} else {
		if x0, y0, err = r.dragEndAt(d.From, d.FromXY); err != nil {
			return err
		}
		if x1, y1, err = r.dragEndAt(d.To, d.ToXY); err != nil {
			return err
		}
	}
	r.glide(x0, y0)
	if err := r.mouse("mouseMoved", x0, y0, nil); err != nil {
		return err
	}
	if err := r.mouse("mousePressed", x0, y0, map[string]any{"button": "left", "buttons": 1, "clickCount": 1}); err != nil {
		return err
	}
	const n = 30
	pause := 5 * time.Millisecond
	if r.tk != nil {
		pause = 30 * time.Millisecond // about a second for the stroke
	}
	for i := 1; i <= n; i++ {
		e := ease(float64(i) / n)
		x, y := x0+(x1-x0)*e, y0+(y1-y0)*e
		if err := r.mouse("mouseMoved", x, y, map[string]any{"button": "left", "buttons": 1}); err != nil {
			return err
		}
		if r.tk != nil {
			r.tk.cursor = &[2]float64{x, y}
			r.js(`__siDemo.move(%g,%g,%d)`, x, y, pause.Milliseconds())
		}
		time.Sleep(pause)
	}
	if err := r.mouse("mouseReleased", x1, y1, map[string]any{"button": "left", "buttons": 0, "clickCount": 1}); err != nil {
		return err
	}
	return r.loaded()
}

// window brings a page the app opened (a popup, a tab) in front: by its
// name, title or address (a part of it), by number (1 is the app's page,
// then the others in the order they opened), or main (the app's page).
func (r *runner) window(st *Step) error {
	if st.Name == "main" || st.Code == 1 {
		r.cur = nil
		return r.show(r.app)
	}
	var session string
	err := r.until(func() (bool, error) {
		s, seen, err := r.findWindow(st)
		if err != nil {
			return true, err
		}
		if s == "" {
			return false, &failure{expected: "a window " + map[bool]string{true: fmt.Sprint(st.Code), false: st.Value.String()}[st.Code > 0], found: "no such window", candidates: seen}
		}
		session = s
		return true, nil
	})
	if err != nil {
		return err
	}
	r.cur = nil
	return r.show(session)
}

// findWindow looks through the feature's pages for st's window; it
// attaches to the one it picks and returns its session, and what it saw.
func (r *runner) findWindow(st *Step) (string, []string, error) {
	var ts struct {
		TargetInfos []struct {
			TargetID, Type, Title, URL, BrowserContextID string
		}
	}
	if err := r.c.CallIn("", "Target.getTargets", nil, &ts); err != nil {
		return "", nil, err
	}
	for _, t := range ts.TargetInfos {
		if t.Type == "page" && t.BrowserContextID == r.ctx && t.TargetID != r.appTarget && !r.termTargets[t.TargetID] && !contains(r.wins, t.TargetID) {
			r.wins = append(r.wins, t.TargetID)
		}
	}
	var seen []string
	for i, id := range r.wins {
		s, err := r.attachWindow(id)
		if err != nil {
			continue // closed since
		}
		if st.Code > 0 {
			if i+2 == st.Code {
				return s, nil, nil
			}
			continue
		}
		var v struct {
			Result struct{ Value string }
		}
		r.c.CallIn(s, "Runtime.evaluate", map[string]any{"expression": `[window.name,document.title,location.href].join('\n')`, "returnByValue": true}, &v)
		names := strings.Split(v.Result.Value, "\n")
		if strings.Contains(strings.ToLower(v.Result.Value), strings.ToLower(st.Value.Text)) {
			return s, nil, nil
		}
		seen = append(seen, strings.Join(names, " · "))
	}
	return "", seen, nil
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// attachWindow attaches to a page the app opened (once): its events, the
// viewport of the take, the overlay.
func (r *runner) attachWindow(id string) (string, error) {
	if s := r.winSess[id]; s != "" {
		return s, nil
	}
	var att struct{ SessionID string }
	if err := r.c.CallIn("", "Target.attachToTarget", map[string]any{"targetId": id, "flatten": true}, &att); err != nil {
		return "", err
	}
	s := att.SessionID
	for _, m := range []string{"Page.enable", "Runtime.enable"} {
		if err := r.c.CallIn(s, m, nil, nil); err != nil {
			return "", err
		}
	}
	r.c.CallIn(s, "Emulation.setDeviceMetricsOverride", map[string]any{"width": r.s.W, "height": r.s.H, "deviceScaleFactor": r.k, "mobile": false}, nil)
	if r.tk != nil {
		r.c.CallIn(s, "Page.addScriptToEvaluateOnNewDocument", map[string]any{"source": overlaySource()}, nil)
		r.c.CallIn(s, "Runtime.evaluate", map[string]any{"expression": overlaySource()}, nil)
	}
	r.winSess[id] = s
	return s, nil
}
