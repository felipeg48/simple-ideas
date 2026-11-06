package tui

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/felipeg48/simple-ideas/internal/render"
)

// SlideOpts are what a live presenter adds to a slide: real pictures, the
// clock for canvas animations, and the canvas states to tween from.
type SlideOpts struct {
	Pics   *Pictures
	T      float64             // seconds since the presenter started: packets, flows, pulses
	States map[string][]string // each canvas's state on this slide (canvasStates)
	Prev   map[string][]string // the state each canvas comes from
	P      float64             // how far the move from Prev is (0..1); 1 = arrived
}

type svgRef struct {
	path  string
	words []string
}

// canvasStates walks the deck up to slide i, build k, the way the browser
// does (svg: words relative to the state the last slide left the canvas in,
// then each build step's ops), and returns each canvas's state now and the
// state it comes from (the previous build, or the previous slide).
func canvasStates(d *deck.Deck, i, k int) (cur, prev map[string][]string) {
	cur, prev = map[string][]string{}, map[string][]string{}
	last := map[string][]string{}
	for j := 0; j <= i && j < len(d.Slides); j++ {
		r := &textRenderer{w: 80, k: 1 << 30, collect: true}
		r.body(d.Slides[j].Body)
		for _, s := range r.svgs {
			base := render.ResolveState(last[s.path], s.words)
			if j < i {
				st := base
				for _, ops := range r.stepOps {
					st = render.ApplyOps(st, ops)
				}
				last[s.path] = st
				continue
			}
			st := base
			for n := 0; n < k && n < len(r.stepOps); n++ {
				st = render.ApplyOps(st, r.stepOps[n])
			}
			cur[s.path] = st
			if k > 0 {
				p := base
				for n := 0; n < k-1 && n < len(r.stepOps); n++ {
					p = render.ApplyOps(p, r.stepOps[n])
				}
				prev[s.path] = p
			} else if l, ok := last[s.path]; ok {
				prev[s.path] = l
			}
		}
	}
	return cur, prev
}

// the parsed canvases, by file, reread when the file changes
var (
	canvasMu    sync.Mutex
	canvasCache = map[string]canvasEntry{}
)

type canvasEntry struct {
	c   *canvas
	mod time.Time
}

func cachedCanvas(path string) (*canvas, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	canvasMu.Lock()
	defer canvasMu.Unlock()
	if e, ok := canvasCache[path]; ok && e.mod.Equal(fi.ModTime()) {
		cp := *e.c
		return &cp, nil
	}
	c, err := loadCanvas(path)
	if err != nil || c == nil {
		return nil, err
	}
	canvasCache[path] = canvasEntry{c, fi.ModTime()}
	cp := *c
	return &cp, nil
}

// canvasLines draws the svg: directive f (path, words) in its state, or a
// placeholder when it cannot (a teleprompter preview, a missing file).
func (r *textRenderer) canvasLines(f []string) []string {
	if len(f) == 0 {
		return r.placeholder("◇ canvas", []string{"svg: needs a file"})
	}
	if r.collect {
		return nil
	}
	if r.w < 30 || r.opts == nil || r.opts.States == nil {
		return r.placeholder("◇ animated canvas", f)
	}
	c, err := cachedCanvas(filepath.Join(r.dir, f[0]))
	if err != nil || c == nil {
		return r.placeholder("◇ canvas", []string{"missing: " + f[0]})
	}
	c.pal = r.pal
	c.t = r.opts.T
	c.size(min(r.w, 120), max(6, r.maxH))
	state := r.opts.States[f[0]]
	var prev []string
	p := 1.0
	if pv, ok := r.opts.Prev[f[0]]; ok && r.opts.P < 1 && strings.Join(pv, " ") != strings.Join(state, " ") {
		prev, p = pv, r.opts.P
	}
	return c.render(state, prev, p)
}
