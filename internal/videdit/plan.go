package videdit

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// Resolve turns every line's time into source seconds of a d-second
// source: end is d, a chapter its start, a said phrase when it starts in the
// cues; then the offset. Problems for anchors not found and times outside
// the source.
func (e *Edit) Resolve(d float64, cues []Cue, chapters []Chapter) {
	at := func(it *Item, w When) (float64, bool) {
		t := w.Sec
		switch {
		case w.End:
			t = d
		case w.Chapter != "":
			found := false
			for _, c := range chapters {
				if strings.EqualFold(c.Title, w.Chapter) {
					t, found = c.Start, true
					break
				}
			}
			if !found {
				var names []string
				for _, c := range chapters {
					names = append(names, fmt.Sprintf("%q", c.Title))
				}
				if len(names) == 0 {
					e.problem(it.Line, "chapter %q: the source has no chapters (embedded, or chapters.txt beside it)", w.Chapter)
				} else {
					e.problem(it.Line, "no chapter %q (%s)", w.Chapter, strings.Join(names, ", "))
				}
				return 0, false
			}
		case w.Said != "":
			if len(cues) == 0 {
				e.problem(it.Line, "said %q: no captions beside the source (<source>.vtt or .srt)", w.Said)
				return 0, false
			}
			v, ok := saidAt(cues, w.Said)
			if !ok {
				e.problem(it.Line, "%q is not said in the source's captions", w.Said)
				return 0, false
			}
			t = v
		}
		t += w.Off
		if t < 0 || (d > 0 && t > d+0.05) {
			e.problem(it.Line, "%s is outside the source (0:00–%s)", Stamp(t), Stamp(d))
			return 0, false
		}
		return min(t, d), true
	}
	for _, it := range e.Items {
		if it.NoTime {
			continue
		}
		var ok bool
		if it.Start, ok = at(it, it.From); !ok {
			continue
		}
		if it.To != nil {
			if it.End, ok = at(it, *it.To); ok && it.End <= it.Start {
				e.problem(it.Line, "the range ends (%s) before it starts (%s)", Stamp(it.End), Stamp(it.Start))
			}
		}
	}
}

// Seg is one piece of the output, in order.
type Seg struct {
	Kind  string  // src (a piece of the source at Speed), still (the frame at A), clip, image, card
	A, B  float64 // the source range it stands for (still: A = B; an insert: both its time)
	Speed float64
	Mute  bool   // src: silent (speed · mute)
	File  string // clip, image: the file (absolute)
	Text  string // card
	Keep  bool   // clip: the source's own sound under [A, B), the clip fitted to it
	Trans *Trans // the transition into this piece
	Len   float64
	Out   float64 // where it starts in the output
	Item  *Item   // the line that made it (nil: plain source)
}

// Plan is the output as pieces of the source and what was put in.
type Plan struct {
	Segs []*Seg
	D    float64 // the source's length
	Len  float64 // the output's
}

// Plan lays the edits of a d-second source out as output pieces. clipLen
// gives a clip's length (replace with, insert clip, refilm: Item.File,
// absolute). Problems: overlapping ranges, points inside ranges.
func (e *Edit) Plan(d float64, clipLen func(string) float64) *Plan {
	type rng struct {
		a, b float64
		it   *Item
	}
	var rs []rng
	type point struct {
		t  float64
		it *Item
	}
	var ps []point
	for _, it := range e.Items {
		switch it.Action {
		case "cut", "speed", "replace", "refilm":
			if it.End > it.Start {
				rs = append(rs, rng{it.Start, it.End, it})
			}
		case "trim-start":
			if it.Start > 0 {
				rs = append(rs, rng{0, it.Start, it})
			}
		case "trim-end":
			if it.Start < d {
				rs = append(rs, rng{it.Start, d, it})
			}
		case "freeze", "insert-clip", "insert-image", "card":
			ps = append(ps, point{it.Start, it})
		}
	}
	sort.SliceStable(rs, func(i, j int) bool { return rs[i].a < rs[j].a })
	keep := rs[:0]
	for _, r := range rs {
		if n := len(keep); n > 0 && r.a < keep[n-1].b-1e-9 {
			e.problem(r.it.Line, "%s overlaps the %s on line %d", word(r.it.Action), word(keep[n-1].it.Action), keep[n-1].it.Line)
			continue
		}
		keep = append(keep, r)
	}
	rs = keep
	for i, p := range ps { // a point inside a cut or a replaced range goes after it
		for _, r := range rs {
			if p.t > r.a+1e-9 && p.t < r.b-1e-9 {
				e.warn(p.it.Line, "%s at %s is inside the %s on line %d: it plays after it", word(p.it.Action), Stamp(p.t), word(r.it.Action), r.it.Line)
				ps[i].t = r.b
			}
		}
	}
	sort.SliceStable(ps, func(i, j int) bool { return ps[i].t < ps[j].t })
	pl := &Plan{D: d}
	add := func(s *Seg) {
		if s.Kind == "src" && s.B-s.A < 1e-6 {
			return
		}
		if s.Speed == 0 {
			s.Speed = 1
		}
		pl.Segs = append(pl.Segs, s)
	}
	used := make([]bool, len(ps))
	pointSegs := func(lo, hi float64, incHi bool) {
		for k, p := range ps {
			if used[k] || p.t < lo-1e-9 || p.t > hi+1e-9 || (!incHi && p.t > hi-1e-9) {
				continue
			}
			used[k] = true
			it := p.it
			s := &Seg{A: p.t, B: p.t, Item: it, Trans: it.Trans}
			switch it.Action {
			case "freeze":
				s.Kind, s.Len, s.Trans = "still", it.Dur, nil
			case "insert-clip":
				s.Kind, s.File, s.Len = "clip", e.Abs(it.File), clipLen(e.Abs(it.File))
			case "insert-image":
				s.Kind, s.File, s.Len = "image", e.Abs(it.File), it.Dur
			case "card":
				s.Kind, s.Text, s.Len = "card", it.Str, it.Dur
			}
			add(s)
		}
	}
	src := func(a, b float64, incHi bool) {
		for k, p := range ps {
			if used[k] || p.t < a-1e-9 || p.t > b+1e-9 || (!incHi && p.t > b-1e-9) {
				continue
			}
			add(&Seg{Kind: "src", A: a, B: p.t})
			a = p.t
			pointSegs(p.t, p.t, true)
		}
		add(&Seg{Kind: "src", A: a, B: b})
		pointSegs(b, b, incHi)
	}
	cur := 0.0
	for _, r := range rs {
		src(cur, r.a, true)
		it := r.it
		switch it.Action {
		case "speed":
			add(&Seg{Kind: "src", A: r.a, B: r.b, Speed: it.Speed, Mute: it.Flags["mute"], Item: it})
		case "replace", "refilm":
			s := &Seg{Kind: "clip", A: r.a, B: r.b, File: e.Abs(it.File), Item: it, Trans: it.Trans, Keep: it.Flags["keep-audio"]}
			s.Len = clipLen(s.File)
			if s.Keep {
				s.Len = r.b - r.a
			}
			if s.Len <= 0 {
				s.Len = r.b - r.a
			}
			add(s)
		}
		cur = r.b
	}
	src(cur, d, true)
	pointSegs(-1, math.Inf(1), true) // anything left (past the end)
	for i, s := range pl.Segs {
		if s.Len <= 0 && s.Kind == "src" {
			s.Len = (s.B - s.A) / s.Speed
		}
		if i == 0 {
			s.Trans = nil
		} else if p := pl.Segs[i-1]; p.Item != nil && p.Item.Trans != nil && s.Trans == nil && p.Kind != "still" {
			s.Trans = p.Item.Trans // the way out of an inserted or replaced piece
		}
		s.Out = pl.Len
		pl.Len += s.Len
	}
	return pl
}

func word(action string) string { return strings.ReplaceAll(action, "-", " ") }

// Map is where source time t first shows in the output (see the spec's
// mapping rule): in a piece of source, its place at the piece's speed; at a
// freeze, the freeze's start; in a replaced range, the same distance into
// the replacement; in a cut, the join.
func (p *Plan) Map(t float64) float64 {
	for _, s := range p.Segs {
		switch {
		case s.Kind == "src" && t >= s.A-1e-9 && t < s.B-1e-9:
			return s.Out + (t-s.A)/s.Speed
		case s.Kind == "still" && math.Abs(t-s.A) < 1e-9:
			return s.Out
		case s.Kind == "clip" && s.B > s.A && t >= s.A-1e-9 && t < s.B-1e-9:
			return s.Out + min(t-s.A, s.Len)
		}
	}
	for _, s := range p.Segs {
		if s.A >= t-1e-9 {
			return s.Out
		}
	}
	return p.Len
}

// Cut reports the cut or trim line whose range holds t (nil: none).
func (p *Plan) Cut(t float64, e *Edit) *Item {
	for _, it := range e.Items {
		a, b := it.Start, it.End
		switch it.Action {
		case "cut":
		case "trim-start":
			a, b = 0, it.Start
		case "trim-end":
			a, b = it.Start, p.D+1
		default:
			continue
		}
		if t >= a && t < b {
			return it
		}
	}
	return nil
}

// Span is when a line shows in the output.
type Span struct {
	It       *Item
	From, To float64
}

// Spans are the output times of the lines drawn over the picture (callouts,
// blur, pixelate, speed badges), the camera, the sound lines and mutes.
func (p *Plan) Spans(e *Edit) []Span {
	var out []Span
	for _, it := range e.Items {
		if it.NoTime {
			continue
		}
		from := p.Map(it.Start)
		to := from + it.Dur
		if it.Range() {
			to = p.Map(it.End)
		}
		switch it.Action {
		case "speed":
			if it.Badge == "" {
				continue
			}
		case "say", "music", "zoom", "pan", "zoom-out":
			to = from
		case "mute":
		default:
			if !it.IsCallout() && it.Action != "blur" && it.Action != "pixelate" {
				continue
			}
		}
		out = append(out, Span{it, from, min(to, p.Len)})
	}
	return out
}
