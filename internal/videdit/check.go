package videdit

import (
	"fmt"
	"math"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/webdemo"
)

// Check adds to e.Problems what si check reports beyond the file itself:
// missing files, music not at the start, a refilm without its script; then,
// with ffprobe, times past the source's end, anchors not found, positions
// outside the frame, overlapping ranges, callouts inside cuts. The note
// says what could not be checked ("" when everything was).
func (e *Edit) Check(ffprobe string) (note string) {
	missing := false
	for _, s := range e.Sources {
		if !fileExists(e.Abs(s.File)) {
			e.problem(s.Line, "missing source %s", s.File)
			missing = true
		}
	}
	for _, it := range e.Items {
		switch it.Action {
		case "replace", "insert-clip", "insert-image", "music":
			if !fileExists(e.Abs(it.File)) {
				e.problem(it.Line, "missing file %s", it.File)
			}
		}
		if it.Action == "music" && (it.From.Sec != 0 || it.From.End || it.From.Chapter != "" || it.From.Said != "" || it.From.Off != 0) {
			e.problem(it.Line, "music plays under the whole video: write it at 0:00")
		}
		if it.Action == "refilm" {
			sp, err := e.ScriptPath()
			if err != nil {
				e.problem(it.Line, "refilm: %v", err)
				continue
			}
			if s, err := webdemo.ParseFile(sp); err == nil {
				found := false
				var names []string
				for _, f := range s.Features {
					found = found || strings.EqualFold(f.Name, it.Feature)
					names = append(names, fmt.Sprintf("%q", f.Name))
				}
				if !found {
					e.problem(it.Line, "refilm: no feature %q in %s (%s)", it.Feature, rel(e.Dir, sp), strings.Join(names, ", "))
				}
			}
		}
	}
	if missing {
		return ""
	}
	if ffprobe == "" {
		return "ffprobe not found: times, positions and overlaps were not checked against the source"
	}
	d, w, h, cues, chaps, err := e.probeSources(ffprobe)
	if err != nil {
		e.problem(1, "%v", err)
		return ""
	}
	e.Resolve(d, cues, chaps)
	for _, it := range e.Items {
		e.checkPlace(it, w, h)
	}
	lens := map[string]float64{}
	pl := e.Plan(d, func(f string) float64 {
		if v, ok := lens[f]; ok {
			return v
		}
		in, err := Probe(ffprobe, f)
		lens[f] = in.D
		if err != nil {
			return 0
		}
		return in.D
	})
	for _, it := range e.Items {
		if it.IsCallout() || it.Action == "say" {
			if c := pl.Cut(it.Start, e); c != nil {
				e.warn(it.Line, "%s at %s is inside the %s on line %d: it shows at the join", word(it.Action), Stamp(it.Start), word(c.Action), c.Line)
			}
		}
	}
	return ""
}

// probeSources is the length of the sources joined, the first one's size,
// and (one source) the captions and chapters beside it, without joining.
func (e *Edit) probeSources(ffprobe string) (d float64, w, h int, cues []Cue, chaps []Chapter, err error) {
	for i, s := range e.Sources {
		f := e.Abs(s.File)
		in, err := Probe(ffprobe, f)
		if err != nil {
			return 0, 0, 0, nil, nil, err
		}
		if i == 0 {
			w, h = in.W, in.H
		}
		to := s.To
		if to == 0 || to > in.D {
			if to > in.D {
				e.problem(s.Line, "source %s: to %s is past its end (%s)", s.File, Stamp(to), Stamp(in.D))
			}
			to = in.D
		}
		d += math.Max(0, to-s.From)
		if len(e.Sources) == 1 {
			for _, q := range cuesBeside(f) {
				if q.To > s.From {
					cues = append(cues, Cue{q.Text, math.Max(0, q.From-s.From), q.To - s.From})
				}
			}
			for _, c := range chaptersOf(f, in) {
				if c.Start >= s.From {
					chaps = append(chaps, Chapter{c.Title, c.Start - s.From})
				}
			}
		}
	}
	return d, w, h, cues, chaps, nil
}

// checkPlace reports points and rectangles outside a w×h frame.
func (e *Edit) checkPlace(it *Item, w, h int) {
	if w == 0 || h == 0 {
		return
	}
	pt := func(name string, p *Pos) {
		if p == nil || p.Word != "" {
			return
		}
		x, y := p.Px(w, h)
		if x < 0 || y < 0 || x > float64(w) || y > float64(h) {
			e.problem(it.Line, "%s %g,%g is outside the %d×%d frame", name, x, y, w, h)
		}
	}
	pt("at", it.At)
	pt("to", it.Point)
	pt("from", it.Tail)
	if r := it.Rect; r != nil {
		x, y, rw, rh := r.Px(w, h)
		if x < 0 || y < 0 || x+rw > float64(w)+1 || y+rh > float64(h)+1 {
			e.problem(it.Line, "%g,%g %gx%g is outside the %d×%d frame", x, y, rw, rh, w, h)
		}
	}
}
