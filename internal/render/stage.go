package render

import (
	"fmt"
	"html"
	"math"
	"strings"
)

// A ```stage fence is a small Mermaid-like DSL that generates a themed, fluid,
// self-contained animated SVG. It handles simple flows — chains, fan-out and
// merges — laid out left-to-right or top-to-bottom with a longest-path layering
// (no crossing minimisation); dense graphs stay the job of the `mermaid` fence.
//
//	dir: LR                         # LR (default) or TB
//	Cart --> Pay[Payment] --> Done  # ids; A[Label] gives a different label
//	Cache -->|hit| Fast             # edge label between pipes
//	Cache -->|miss| Slow
//	dot Cart..Done 2s               # a dot loops along the route Cart->Done
//	flow Cache..Slow                # marching dashes along a route
//
// Nodes appear in first-seen order. `dot`/`flow A..B [dur]` find the shortest
// route A->B through the edges and animate it with CSS (offset-path / dashes),
// so there is no runtime JS and both pause under reduced motion.

type stageNode struct{ id, label string }
type stageEdge struct{ from, to, label string }
type stageAnim struct {
	kind     string // "dot" or "flow"
	from, to string
	dur      string
}

type stageGraph struct {
	dir   string
	nodes []stageNode
	idx   map[string]int
	edges []stageEdge
	anims []stageAnim

	// filled by layout()
	depth            []int
	cx, cy, tlx, tly []int
	vbW, vbH         int
}

// layout constants, in viewBox units (the SVG scales by viewBox, never px).
const (
	sBoxW   = 180
	sBoxH   = 76
	sColGap = 74 // gap between successive depths (along the flow)
	sRowGap = 30 // gap between siblings in the same depth
	sPad    = 28
	sDotR   = 8
)

func parseStage(body string) (*stageGraph, error) {
	g := &stageGraph{dir: "LR", idx: map[string]int{}}
	register := func(tok string) (string, error) {
		id, label := tok, tok
		if i := strings.IndexByte(tok, '['); i >= 0 {
			if !strings.HasSuffix(tok, "]") || i == 0 {
				return "", fmt.Errorf("bad node %q (use id[Label])", tok)
			}
			id, label = tok[:i], tok[i+1:len(tok)-1]
		}
		if j, ok := g.idx[id]; ok {
			if label != id && g.nodes[j].label == id {
				g.nodes[j].label = label // a later A[Label] names an earlier bare A
			}
			return id, nil
		}
		g.idx[id] = len(g.nodes)
		g.nodes = append(g.nodes, stageNode{id, label})
		return id, nil
	}
	anim := func(kind, v string) error {
		f := strings.Fields(v)
		if len(f) == 0 {
			return fmt.Errorf("%s needs A..B", kind)
		}
		ends := strings.SplitN(f[0], "..", 2)
		if len(ends) != 2 || ends[0] == "" || ends[1] == "" {
			return fmt.Errorf("%s path must be A..B, got %q", kind, f[0])
		}
		dur := "1.6s"
		if len(f) >= 2 {
			dur = normDur(f[1])
		}
		g.anims = append(g.anims, stageAnim{kind, ends[0], ends[1], dur})
		return nil
	}
	for _, raw := range strings.Split(body, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		switch {
		case strings.HasPrefix(line, "dir:"):
			d := strings.ToUpper(strings.TrimSpace(line[len("dir:"):]))
			if d != "LR" && d != "TB" {
				return nil, fmt.Errorf("dir must be LR or TB, got %q", d)
			}
			g.dir = d
		case strings.HasPrefix(line, "dot "):
			if err := anim("dot", line[len("dot "):]); err != nil {
				return nil, err
			}
		case strings.HasPrefix(line, "flow "):
			if err := anim("flow", line[len("flow "):]); err != nil {
				return nil, err
			}
		case strings.Contains(line, "-->"):
			if err := parseEdgeLine(g, line, register); err != nil {
				return nil, err
			}
		default:
			if _, err := register(line); err != nil {
				return nil, err
			}
		}
	}
	if len(g.nodes) == 0 {
		return nil, fmt.Errorf("no nodes")
	}
	for _, a := range g.anims {
		if _, ok := g.idx[a.from]; !ok {
			return nil, fmt.Errorf("%s references unknown node %q", a.kind, a.from)
		}
		if _, ok := g.idx[a.to]; !ok {
			return nil, fmt.Errorf("%s references unknown node %q", a.kind, a.to)
		}
	}
	return g, nil
}

// parseEdgeLine handles "A --> B --> C" and edge labels "A -->|hit| B".
// A label written as |text| prefixes the node that follows the arrow.
func parseEdgeLine(g *stageGraph, line string, register func(string) (string, error)) error {
	parts := strings.Split(line, "-->")
	prev, err := register(strings.TrimSpace(parts[0]))
	if err != nil {
		return err
	}
	for _, seg := range parts[1:] {
		seg = strings.TrimSpace(seg)
		label := ""
		if strings.HasPrefix(seg, "|") {
			end := strings.IndexByte(seg[1:], '|')
			if end < 0 {
				return fmt.Errorf("edge label not closed in %q", line)
			}
			label = strings.TrimSpace(seg[1 : 1+end])
			seg = strings.TrimSpace(seg[end+2:])
		}
		if seg == "" {
			return fmt.Errorf("edge has no target node in %q", line)
		}
		id, err := register(seg)
		if err != nil {
			return err
		}
		g.edges = append(g.edges, stageEdge{prev, id, label})
		prev = id
	}
	return nil
}

// normDur appends the "s" unit when the caller wrote a bare number ("1" -> "1s").
func normDur(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "1.6s"
	}
	if c := s[len(s)-1]; c >= '0' && c <= '9' {
		return s + "s"
	}
	return s
}

// layout assigns each node a column by longest path from a root and stacks
// siblings into rows, centring shorter columns. ponytail: longest-path layering,
// no crossing minimisation — switch to the mermaid fence for dense graphs.
func (g *stageGraph) layout() {
	n := len(g.nodes)
	depth := make([]int, n)
	for it := 0; it < n; it++ { // relax; cap at n iterations bounds any cycle
		changed := false
		for _, e := range g.edges {
			if a, b := g.idx[e.from], g.idx[e.to]; depth[b] < depth[a]+1 {
				depth[b] = depth[a] + 1
				changed = true
			}
		}
		if !changed {
			break
		}
	}
	g.depth = depth
	cols := 1
	for _, d := range depth {
		if d+1 > cols {
			cols = d + 1
		}
	}
	rowOf := make([]int, n)
	count := make([]int, cols)
	for i := range g.nodes {
		rowOf[i] = count[depth[i]]
		count[depth[i]]++
	}
	maxRows := 1
	for _, c := range count {
		if c > maxRows {
			maxRows = c
		}
	}
	g.cx, g.cy = make([]int, n), make([]int, n)
	g.tlx, g.tly = make([]int, n), make([]int, n)
	totalRows := maxRows*rowExtent(g.dir) + (maxRows-1)*sRowGap
	for i := range g.nodes {
		d, r := depth[i], rowOf[i]
		colRows := count[d]*rowExtent(g.dir) + (count[d]-1)*sRowGap
		off := (totalRows - colRows) / 2
		if g.dir == "TB" {
			g.tlx[i] = sPad + off + r*(sBoxW+sRowGap)
			g.tly[i] = sPad + d*(sBoxH+sColGap)
		} else {
			g.tlx[i] = sPad + d*(sBoxW+sColGap)
			g.tly[i] = sPad + off + r*(sBoxH+sRowGap)
		}
		g.cx[i] = g.tlx[i] + sBoxW/2
		g.cy[i] = g.tly[i] + sBoxH/2
	}
	if g.dir == "TB" {
		g.vbW = sPad*2 + maxRows*sBoxW + (maxRows-1)*sRowGap
		g.vbH = sPad*2 + cols*sBoxH + (cols-1)*sColGap
	} else {
		g.vbW = sPad*2 + cols*sBoxW + (cols-1)*sColGap
		g.vbH = sPad*2 + maxRows*sBoxH + (maxRows-1)*sRowGap
	}
}

// rowExtent is the per-row size along the cross axis (box height for LR, width for TB).
func rowExtent(dir string) int {
	if dir == "TB" {
		return sBoxW
	}
	return sBoxH
}

// route is the shortest path of node indices from->to through the edges.
func (g *stageGraph) route(from, to int) ([]int, bool) {
	if from == to {
		return []int{from}, true
	}
	adj := map[int][]int{}
	for _, e := range g.edges {
		a := g.idx[e.from]
		adj[a] = append(adj[a], g.idx[e.to])
	}
	prev := map[int]int{from: -1}
	for q := []int{from}; len(q) > 0; {
		n := q[0]
		q = q[1:]
		if n == to {
			break
		}
		for _, m := range adj[n] {
			if _, seen := prev[m]; !seen {
				prev[m] = n
				q = append(q, m)
			}
		}
	}
	if _, ok := prev[to]; !ok {
		return nil, false
	}
	var path []int
	for n := to; n != -1; n = prev[n] {
		path = append([]int{n}, path...)
	}
	return path, true
}

// genStage turns a fence body into inline SVG. seq scopes the per-animation CSS
// so several stage diagrams on one slide do not collide.
func genStage(body string, seq int) (string, error) {
	g, err := parseStage(body)
	if err != nil {
		return "", err
	}
	g.layout()

	// resolve animation routes up front so a bad A..B is a clear error.
	routes := make([][]int, len(g.anims))
	for i, a := range g.anims {
		r, ok := g.route(g.idx[a.from], g.idx[a.to])
		if !ok {
			return "", fmt.Errorf("%s has no path %s..%s", a.kind, a.from, a.to)
		}
		routes[i] = r
	}

	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" class="si-stage s%d" role="img">`, g.vbW, g.vbH, seq)
	b.WriteString(stageStyle)
	for i, a := range g.anims {
		d := g.pathD(routes[i])
		if a.kind == "dot" {
			fmt.Fprintf(&b, `.s%d .a%d{offset-path:path('%s');animation:si-stage-move %s linear infinite}`, seq, i, d, a.dur)
		} else {
			fmt.Fprintf(&b, `.s%d .a%d{animation:si-stage-flow %s linear infinite}`, seq, i, a.dur)
		}
	}
	b.WriteString(stageKeyframes)
	b.WriteString(`</style>`)

	for _, e := range g.edges {
		b.WriteString(g.arrow(g.idx[e.from], g.idx[e.to]))
	}
	for i, a := range g.anims { // flow dashes ride under the boxes
		if a.kind == "flow" {
			fmt.Fprintf(&b, `<path class="flow a%d" d="%s"/>`, i, g.pathD(routes[i]))
		}
	}
	for i, nd := range g.nodes {
		fmt.Fprintf(&b, `<rect class="node" x="%d" y="%d" width="%d" height="%d" rx="14"/>`, g.tlx[i], g.tly[i], sBoxW, sBoxH)
		fmt.Fprintf(&b, `<text class="lbl" x="%d" y="%d">%s</text>`, g.cx[i], g.cy[i], html.EscapeString(nd.label))
	}
	for _, e := range g.edges { // edge labels on top, with a halo for legibility
		if e.label != "" {
			b.WriteString(g.edgeLabel(g.idx[e.from], g.idx[e.to], e.label))
		}
	}
	for i, a := range g.anims { // dots on top of everything
		if a.kind == "dot" {
			fmt.Fprintf(&b, `<circle class="dot a%d" r="%d"/>`, i, sDotR)
		}
	}
	b.WriteString(`</svg>`)
	return b.String(), nil
}

const stageStyle = `<style>` +
	`.si-stage{font-family:var(--si-font-body,system-ui,sans-serif)}` +
	`.si-stage .node{fill:var(--si-surface,#fff);stroke:var(--si-line,#d9d6ce);stroke-width:2.5}` +
	`.si-stage .lbl{fill:var(--si-fg,#16171b);font-size:22px;font-weight:700;text-anchor:middle;dominant-baseline:central}` +
	`.si-stage .edge{stroke:var(--si-muted,#6b7280);stroke-width:2.5;fill:none}` +
	`.si-stage .head{fill:var(--si-muted,#6b7280)}` +
	`.si-stage .elbl{fill:var(--si-fg,#16171b);font-size:15px;font-weight:600;text-anchor:middle;dominant-baseline:central;paint-order:stroke;stroke:var(--si-bg,#fff);stroke-width:4px}` +
	`.si-stage .flow{fill:none;stroke:var(--si-accent,#3657d6);stroke-width:3;stroke-dasharray:10 8}` +
	`.si-stage .dot{fill:var(--si-accent,#3657d6)}`

const stageKeyframes = `@keyframes si-stage-move{from{offset-distance:0%}to{offset-distance:100%}}` +
	`@keyframes si-stage-flow{to{stroke-dashoffset:-18}}` +
	`@media (prefers-reduced-motion:reduce){.si-stage .dot{animation:none;offset-distance:50%}.si-stage .flow{animation:none}}`

func rnd(f float64) int { return int(math.Round(f)) }

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// clip returns where the segment from a box centre toward (tx,ty) leaves the box.
func clip(cx, cy, tx, ty float64) (float64, float64) {
	dx, dy := tx-cx, ty-cy
	if dx == 0 && dy == 0 {
		return cx, cy
	}
	t := math.Inf(1)
	if dx != 0 {
		t = math.Min(t, (sBoxW/2)/math.Abs(dx))
	}
	if dy != 0 {
		t = math.Min(t, (sBoxH/2)/math.Abs(dy))
	}
	return cx + dx*t, cy + dy*t
}

// edgeSeg is one edge's drawn geometry: the path, the arrow-head position and
// angle, and where to anchor an edge label.
type edgeSeg struct {
	d          string  // SVG path data
	hx, hy, ha float64 // head tip x,y and angle (degrees)
	lx, ly     float64 // label anchor
}

// edgeGeom routes an edge: a straight clipped line between adjacent boxes, or,
// for a "skip" edge that spans more than one column on the same row, a Bézier
// that bows off-axis so it bypasses the boxes in between instead of crossing them.
func (g *stageGraph) edgeGeom(a, b int) edgeSeg {
	ax, ay := float64(g.cx[a]), float64(g.cy[a])
	bx, by := float64(g.cx[b]), float64(g.cy[b])
	skipLR := g.dir == "LR" && g.cy[a] == g.cy[b] && absInt(g.depth[b]-g.depth[a]) > 1
	skipTB := g.dir == "TB" && g.cx[a] == g.cx[b] && absInt(g.depth[b]-g.depth[a]) > 1
	switch {
	case skipLR:
		sx, sy := ax, float64(g.tly[a]+sBoxH) // bottom centre of a
		ex, ey := bx, float64(g.tly[b]+sBoxH)
		yb := math.Max(sy, ey) + float64(sRowGap+20)
		d := fmt.Sprintf("M%d %d C%d %d %d %d %d %d", rnd(sx), rnd(sy), rnd(sx), rnd(yb), rnd(ex), rnd(yb), rnd(ex), rnd(ey))
		return edgeSeg{d, ex, ey, math.Atan2(ey-yb, 0) * 180 / math.Pi, (sx + ex) / 2, yb}
	case skipTB:
		sx, sy := float64(g.tlx[a]+sBoxW), ay // right centre of a
		ex, ey := float64(g.tlx[b]+sBoxW), by
		xb := math.Max(sx, ex) + float64(sRowGap+20)
		d := fmt.Sprintf("M%d %d C%d %d %d %d %d %d", rnd(sx), rnd(sy), rnd(xb), rnd(sy), rnd(xb), rnd(ey), rnd(ex), rnd(ey))
		return edgeSeg{d, ex, ey, math.Atan2(0, ex-xb) * 180 / math.Pi, xb, (sy + ey) / 2}
	default:
		sx, sy := clip(ax, ay, bx, by)
		ex, ey := clip(bx, by, ax, ay)
		d := fmt.Sprintf("M%d %d L%d %d", rnd(sx), rnd(sy), rnd(ex), rnd(ey))
		return edgeSeg{d, ex, ey, math.Atan2(ey-sy, ex-sx) * 180 / math.Pi, (sx + ex) / 2, (sy + ey) / 2}
	}
}

// arrow draws an edge path plus a triangular head oriented along it.
func (g *stageGraph) arrow(a, b int) string {
	s := g.edgeGeom(a, b)
	return fmt.Sprintf(
		`<path class="edge" d="%s"/>`+
			`<polygon class="head" points="0,0 -13,-7 -13,7" transform="translate(%d %d) rotate(%.1f)"/>`,
		s.d, rnd(s.hx), rnd(s.hy), s.ha)
}

func (g *stageGraph) edgeLabel(a, b int, label string) string {
	s := g.edgeGeom(a, b)
	return fmt.Sprintf(`<text class="elbl" x="%d" y="%d">%s</text>`, rnd(s.lx), rnd(s.ly), html.EscapeString(label))
}

// pathD is the polyline through node centres along a route.
func (g *stageGraph) pathD(route []int) string {
	var b strings.Builder
	for i, n := range route {
		if i == 0 {
			fmt.Fprintf(&b, "M%d %d", g.cx[n], g.cy[n])
		} else {
			fmt.Fprintf(&b, " L%d %d", g.cx[n], g.cy[n])
		}
	}
	return b.String()
}
