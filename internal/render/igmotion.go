package render

import (
	"fmt"
	"html"
	"math"
	"strconv"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

// Animated infographics (`animate: build|loop`): si works out when each block
// comes in and how (igSchedule) and writes it on the block as data-enter,
// data-at and data-dur; web/infographic.js plays that timeline from one clock,
// which `si build --gif/--mp4` seeks frame by frame. Flow blocks (`A -> B`)
// are laid out by the stage fence's engine (stage.go) and drawn here with
// the parts the timeline animates: boxes, arrows, order badges and packets.

// IGRatio is the width and height of an infographic aspect (ok false when it
// has none: the frame takes its orientation's).
func IGRatio(aspect string) (w, h float64, ok bool) {
	r := igRatio(aspect)
	return r[0], r[1], r[0] > 0
}

// igCue is when and how one block comes in.
type igCue struct {
	At, Dur float64
	Enter   string // fade | rise | grow | count | light (steps) | draw (flow)
}

const (
	igStart  = 0.5 // the header fades in first
	igPacket = 0.7 // seconds a packet takes along one arrow
)

// igEnter is a block's way in: enter= as written, else by kind (numbers
// count up, charts grow, steps light one by one, a flow draws itself, a quote
// fades; the rest rise).
func igEnter(b *deck.IGBlock) string {
	if b.Enter != "" {
		return b.Enter
	}
	switch b.Kind {
	case "stat", "compare":
		return "count"
	case "chart":
		return "grow"
	case "steps":
		return "light"
	case "flow":
		return "draw"
	case "quote", "image":
		return "fade"
	}
	return "rise"
}

// igDur is how long a block takes to come in.
func igDur(b *deck.IGBlock, enter string) float64 {
	switch {
	case b.Kind == "flow" && b.Flow != nil && enter == "draw":
		return 0.3 + 0.45*float64(len(igFlowEvents(b.Flow)))
	case b.Kind == "steps" && enter == "light":
		return 0.35 + 0.4*float64(max(len(igItems(b.Body)), 1))
	case b.Kind == "chart" && enter == "grow":
		_, rows, _ := deck.IGTable(b.Body)
		return math.Min(2.2, 0.9+0.12*float64(len(rows)))
	case enter == "count":
		return 1.4
	}
	return 0.7
}

// igSchedule times the blocks: one after another in reading order, each
// starting when the one before is most of the way in, unless at= places it.
// end is when the last block is in (with one round of a flow's packets);
// total is the length of the whole animation: duration if given, else end
// and a moment to read it.
func igSchedule(blocks []*deck.IGBlock, loop bool, duration float64) (cues []igCue, end, total float64) {
	cursor := igStart
	for _, b := range blocks {
		e := igEnter(b)
		c := igCue{At: cursor, Dur: igDur(b, e), Enter: e}
		if b.At >= 0 {
			c.At = b.At
		}
		cues = append(cues, c)
		cursor = c.At + 0.7*c.Dur
		in := c.At + c.Dur
		if b.Kind == "flow" && b.Flow != nil {
			in += igPacket * float64(len(b.Flow.Edges)) // one trip of the packets
		}
		end = math.Max(end, in)
	}
	end = math.Round(end*100) / 100
	if duration > 0 {
		return cues, end, duration
	}
	hold := 1.5
	if loop {
		hold = 0.8
	}
	return cues, end, math.Round((end+hold)*10) / 10
}

func igSec(f float64) string { return strconv.FormatFloat(math.Round(f*100)/100, 'f', -1, 64) }

// igWeight is how much height a block wants in a fixed tall frame.
func igWeight(b *deck.IGBlock, tall bool) float64 {
	switch b.Kind {
	case "flow":
		if !tall {
			return 1.8 // across: as tall as one box and its labels
		}
		if b.Flow != nil {
			return 1 + 0.9*float64(len(b.Flow.Nodes))
		}
		return 3
	case "chart", "image":
		return 2
	case "steps", "list", "icons":
		return 0.9 + 0.32*float64(len(igItems(b.Body)))
	case "compare":
		return 1.4
	case "stat":
		return 1.1
	}
	return 1 + float64(len(b.Body))/220
}

// igRowWeights is a grid-template-rows that gives each row the share its
// tallest one-row block wants.
func igRowWeights(blocks []*deck.IGBlock, pos [][4]int, rows int, tall bool) string {
	w := make([]float64, rows)
	for i, b := range blocks {
		p := pos[i]
		per := igWeight(b, tall) / float64(p[2])
		for r := p[0] - 1; r < p[0]-1+p[2] && r < rows; r++ {
			w[r] = math.Max(w[r], per)
		}
	}
	parts := make([]string, rows)
	for i, v := range w {
		parts[i] = "minmax(0," + igSec(math.Max(v, 0.8)) + "fr)"
	}
	return strings.Join(parts, " ")
}

// igGIFBudget is the size above which si check warns about an animated GIF.
const igGIFBudget = 8 << 20

// igGIFSize is the default GIF size of an aspect: 720 px wide.
func igGIFSize(aspect string) [2]int {
	r := igRatio(aspect)
	if r[0] == 0 {
		r = [2]float64{16, 9}
	}
	return [2]int{720, int(math.Round(720 * r[1] / r[0]))}
}

// IGGIFEstimate guesses the bytes of an animated GIF of w×h pixels, seconds
// long at fps: a full first frame, then only the pixels that change.
// ponytail: about twice what the example GIFs measure (flat cards on a soft
// background: 466 KB for 8.7 s at 720 × 1560), not a model of LZW; si build
// prints the real size.
func IGGIFEstimate(w, h int, seconds, fps float64) int64 {
	px := float64(w * h)
	return int64(px*0.12 + seconds*fps*px*0.004)
}

// igFlowEvent is one beat of a flow drawing itself: a box appears or an
// arrow draws (the other is -1).
type igFlowEvent struct{ Node, Edge int }

// igFlowEvents orders a flow's drawing: arrows in the order written, each
// arrow's start box before it and its end box after; boxes without arrows
// last.
func igFlowEvents(f *deck.IGFlow) []igFlowEvent {
	var ev []igFlowEvent
	shown := make([]bool, len(f.Nodes))
	show := func(n int) {
		if !shown[n] {
			shown[n] = true
			ev = append(ev, igFlowEvent{n, -1})
		}
	}
	for k, e := range f.Edges {
		show(e.From)
		ev = append(ev, igFlowEvent{-1, k})
		show(e.To)
	}
	for n := range f.Nodes {
		show(n)
	}
	return ev
}

// igFlowGraph lays a flow out with the stage fence's engine (longest-path
// columns, siblings stacked) in direction dir (LR or TB).
func igFlowGraph(f *deck.IGFlow, dir string) *stageGraph {
	g := &stageGraph{dir: dir, idx: map[string]int{}}
	for i, n := range f.Nodes {
		g.idx[n] = i
		g.nodes = append(g.nodes, stageNode{n, n})
	}
	back := igBackEdges(f)
	for k, e := range f.Edges {
		if !back[k] { // a return arrow (an ack) does not push its target along
			g.edges = append(g.edges, stageEdge{f.Nodes[e.From], f.Nodes[e.To], e.Label})
		}
	}
	g.layout()
	return g
}

// igBackEdges marks the arrows that close a cycle (found depth first from the
// boxes in the order written), so the layout stays a flow.
func igBackEdges(f *deck.IGFlow) map[int]bool {
	out := map[int][]int{}
	for k, e := range f.Edges {
		out[e.From] = append(out[e.From], k)
	}
	back, state := map[int]bool{}, make([]int, len(f.Nodes)) // 0 new, 1 on the path, 2 done
	var visit func(n int)
	visit = func(n int) {
		state[n] = 1
		for _, k := range out[n] {
			switch to := f.Edges[k].To; state[to] {
			case 0:
				visit(to)
			case 1:
				back[k] = true
			}
		}
		state[n] = 2
	}
	for n := range f.Nodes {
		if state[n] == 0 {
			visit(n)
		}
	}
	return back
}

// igEdgeGeom is an arrow's geometry; an arrow with one going back between the
// same two boxes is moved off the centre line (off: +1 or -1) so both show.
func igEdgeGeom(g *stageGraph, a, b int, off float64) edgeSeg {
	if off == 0 {
		return g.edgeGeom(a, b)
	}
	ax, ay, bx, by := float64(g.cx[a]), float64(g.cy[a]), float64(g.cx[b]), float64(g.cy[b])
	dx, dy := bx-ax, by-ay
	l := math.Hypot(dx, dy)
	if l == 0 {
		return g.edgeGeom(a, b)
	}
	px, py := -dy/l*14*off, dx/l*14*off
	sx, sy := clip(ax+px, ay+py, bx+px, by+py)
	ex, ey := clip(bx+px, by+py, ax+px, ay+py)
	d := fmt.Sprintf("M%d %d L%d %d", rnd(sx), rnd(sy), rnd(ex), rnd(ey))
	return edgeSeg{d, ex, ey, math.Atan2(ey-sy, ex-sx) * 180 / math.Pi, (sx + ex) / 2, (sy + ey) / 2}
}

// igFlow draws a flow block: the label, then one SVG of boxes, arrows with
// their order number and label, and a packet per arrow (moved by
// infographic.js). data-ev numbers each part's beat in igFlowEvents.
func (dr *deckRenderer) igFlow(b *deck.IGBlock, dir string) string {
	label := ""
	if b.Title != "" {
		label = `<h3 class="si-ig-label">` + dr.inline(b.Title) + "</h3>"
	}
	f := b.Flow
	if f == nil || len(f.Nodes) == 0 {
		return label
	}
	g := igFlowGraph(f, dir)
	evOf := map[[2]int]int{}
	ev := igFlowEvents(f)
	for i, e := range ev {
		evOf[[2]int{e.Node, e.Edge}] = i
	}
	var s strings.Builder
	fmt.Fprintf(&s, `%s<div class="si-igf" data-events="%d"><svg viewBox="0 0 %d %d" role="img" aria-label="%s" preserveAspectRatio="xMidYMid meet">`,
		label, len(ev), g.vbW, g.vbH, html.EscapeString(plain(b.Title)+": "+igFlowText(f)))
	pair := map[[2]int]bool{}
	for _, e := range f.Edges {
		pair[[2]int{e.From, e.To}] = true
	}
	for k, e := range f.Edges {
		off := 0.0
		if pair[[2]int{e.To, e.From}] {
			off = 1 // both ways: each keeps to its right
		}
		seg := igEdgeGeom(g, e.From, e.To, off)
		fmt.Fprintf(&s, `<g class="si-igf-arrow" data-e="%d" data-ev="%d"><path class="si-igf-edge" pathLength="1" d="%s"/>`, k, evOf[[2]int{-1, k}], seg.d)
		fmt.Fprintf(&s, `<polygon class="si-igf-head" points="0,0 -13,-7 -13,7" transform="translate(%d %d) rotate(%.1f)"/>`, rnd(seg.hx), rnd(seg.hy), seg.ha)
		fmt.Fprintf(&s, `<g class="si-igf-n"><circle cx="%d" cy="%d" r="13"/><text x="%d" y="%d" font-size="15">%d</text></g>`, rnd(seg.lx), rnd(seg.ly), rnd(seg.lx), rnd(seg.ly), k+1)
		if e.Label != "" {
			x, y, anchor := rnd(seg.lx), rnd(seg.ly)-24, "middle"
			if dir == "TB" {
				x, y, anchor = rnd(seg.lx)+22, rnd(seg.ly), "start"
			}
			if off != 0 { // two ways between two boxes: each label on its own side
				dx, dy := float64(g.cx[e.To]-g.cx[e.From]), float64(g.cy[e.To]-g.cy[e.From])
				if dir == "TB" && dy > 0 { // the side away from the other arrow
					x, y, anchor = rnd(seg.lx)-22, rnd(seg.ly), "end"
				}
				if dir != "TB" {
					x, anchor = rnd(seg.lx), "middle"
					y = rnd(seg.ly) + map[bool]int{true: -24, false: 26}[dx < 0]
				}
			}
			fmt.Fprintf(&s, `<text class="si-igf-elbl" x="%d" y="%d" font-size="15" text-anchor="%s">%s</text>`, x, y, anchor, html.EscapeString(e.Label))
		}
		s.WriteString(`</g>`)
	}
	for i, n := range f.Nodes {
		icon := ""
		if i < len(f.Icons) {
			icon = igFlowIcons[f.Icons[i]]
		}
		size, tx, inner := 22.0, g.cx[i], float64(sBoxW-24)
		if icon != "" { // the icon at the left, the name centred in what is left
			const ico, gap = 32, 10
			inner -= ico + gap
			tx = g.tlx[i] + 12 + ico + gap + int(inner)/2
			fmt.Fprintf(&s, `<g class="si-igf-node" data-n="%d" data-ev="%d"><rect x="%d" y="%d" width="%d" height="%d" rx="14"/><svg class="si-igf-ico" x="%d" y="%d" width="%d" height="%d" viewBox="0 0 24 24">%s</svg>`,
				i, evOf[[2]int{i, -1}], g.tlx[i], g.tly[i], sBoxW, sBoxH, g.tlx[i]+14, g.cy[i]-ico/2, ico, ico, icon)
		} else {
			fmt.Fprintf(&s, `<g class="si-igf-node" data-n="%d" data-ev="%d"><rect x="%d" y="%d" width="%d" height="%d" rx="14"/>`,
				i, evOf[[2]int{i, -1}], g.tlx[i], g.tly[i], sBoxW, sBoxH)
		}
		if w := textW(n, size); w > inner {
			size *= math.Round(inner/w*100) / 100 // as labelFit rounds it
		}
		fmt.Fprintf(&s, `<text x="%d" y="%d" font-size="%s">%s</text></g>`, tx, g.cy[i], igSec(size), html.EscapeString(n))
	}
	for k := range f.Edges {
		fmt.Fprintf(&s, `<circle class="si-igf-pkt" data-e="%d" r="9" cx="0" cy="0"/>`, k)
	}
	s.WriteString(`</svg></div>`)
	return s.String()
}

// igFlowText is a flow in words, for screen readers: "A to B, B to C".
func igFlowText(f *deck.IGFlow) string {
	var parts []string
	for _, e := range f.Edges {
		t := f.Nodes[e.From] + " to " + f.Nodes[e.To]
		if e.Label != "" {
			t += " (" + e.Label + ")"
		}
		parts = append(parts, t)
	}
	if len(parts) == 0 {
		return strings.Join(f.Nodes, ", ")
	}
	return strings.Join(parts, ", ")
}
