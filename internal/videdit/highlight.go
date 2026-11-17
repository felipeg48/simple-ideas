package videdit

import (
	"fmt"
	"math"

	"github.com/felipeg48/simple-ideas/internal/webdemo"
)

// highlightCuts are the stretches of the output (out seconds) in the
// --with highlight reel: the lines marked · highlight, each where it shows;
// else the first sentence of each chapter (its first caption or said line,
// a breath around it; the chapter's first seconds when nothing is said
// there); else the first sentence of each said line.
func (e *Edit) highlightCuts(c *Ctx, pl *Plan, cues []webdemo.Cue) ([]webdemo.Cut, error) {
	var cs [][2]float64
	spans := map[*Item]bool{}
	for _, s := range pl.Spans(e) {
		spans[s.It] = true
		if s.It.HL {
			cs = append(cs, [2]float64{s.From, s.To})
		}
	}
	for _, it := range e.Items { // other edits marked: their output range (or its first seconds)
		if it.HL && !spans[it] && !it.NoTime {
			a := pl.Map(it.Start)
			b := a + math.Max(it.Dur, 3)
			if it.Range() {
				b = pl.Map(it.End)
			}
			cs = append(cs, [2]float64{a, b})
		}
	}
	first := func(from, to float64) ([2]float64, bool) { // the first said sentence in [from, to)
		for _, q := range cues {
			if q.At >= from-0.05 && q.At < to {
				end := q.At + q.Seconds*webdemo.FirstSentence(q.Text)
				return [2]float64{math.Max(from, q.At-0.4), math.Min(to, end+0.5)}, true
			}
		}
		return [2]float64{}, false
	}
	if len(cs) == 0 {
		var starts []float64
		for _, ch := range c.Chaps {
			if t := pl.Map(ch.Start); t < pl.Len && (len(starts) == 0 || t > starts[len(starts)-1]+0.5) {
				starts = append(starts, t)
			}
		}
		for i, s := range starts {
			end := pl.Len
			if i+1 < len(starts) {
				end = starts[i+1]
			}
			if r, ok := first(s, end); ok {
				cs = append(cs, r)
			} else {
				cs = append(cs, [2]float64{s, math.Min(end, s+5)})
			}
		}
	}
	if len(cs) == 0 {
		for _, q := range cues {
			if r, ok := first(q.At, q.At+q.Seconds+0.01); ok {
				cs = append(cs, r)
			}
		}
	}
	var out []webdemo.Cut
	for _, r := range cs {
		a, b := math.Max(0, r[0]), math.Min(pl.Len, r[1])
		if b-a >= 0.5 {
			out = append(out, webdemo.Cut{From: a, To: b})
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("--with highlight: mark lines with · highlight, or give the source chapters or captions")
	}
	return out, nil
}
