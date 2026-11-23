package deck

import (
	"fmt"
	"math"
	"regexp"
	"strings"
)

// Graphs (RULES §38): ```state (a state machine: `A → B on event`, then
// `event name` steps move the current state along), ```flowchart (`A → B:
// label`, a name ending in ? is a decision; `token A → B → C` steps move a
// token through it) and ```class (UML classes, `Name: field, method()`, and
// relations `A has many B`, `A is a B`, `A uses B`, `A → B`; one class per
// step). Nodes are laid out in layers left to right from the first one.

type gNode struct {
	name, shape string // shape: box, pill (start and end), dia (a decision)
	layer, idx  int
	x, y, w, h  float64 // centre and size
}

type gEdge struct {
	a, b  int
	label string
}

var (
	stEdgeRE  = regexp.MustCompile(`^(.+?)\s*(?:-->|->|→)\s*(.+?)(?:\s+on\s+(.+))?$`)
	fcEdgeRE  = regexp.MustCompile(`^(.+?)\s*(?:-->|->|→)\s*(.+?)(?:\s*:\s*(.+))?$`)
	tokenRE   = regexp.MustCompile(`(?i)^token\s+(.+)$`)
	eventRE   = regexp.MustCompile(`(?i)^event\s+(.+)$`)
	clsRelRE  = regexp.MustCompile(`(?i)^(.+?)\s+(has many|has one|has|is an?|uses|→|->)\s+(.+)$`)
	clsDeclRE = regexp.MustCompile(`^([A-Za-z_][\w<>]*)\s*:\s*(.*)$`)
)

// graph holds nodes by name, in the order they appear.
type graph struct {
	nodes []*gNode
	by    map[string]int
	edges []gEdge
}

func (g *graph) node(name string) int {
	name = strings.TrimSpace(name)
	k := strings.ToLower(name)
	if i, ok := g.by[k]; ok {
		return i
	}
	if g.by == nil {
		g.by = map[string]int{}
	}
	g.nodes = append(g.nodes, &gNode{name: name, shape: "box"})
	g.by[k] = len(g.nodes) - 1
	return len(g.nodes) - 1
}

// layout places the nodes in layers (the shortest path from the first
// node; unreachable ones after) left to right, nw × nh each.
func (g *graph) layout(nw, nh, gapX, gapY, top float64) (W, H float64) {
	for _, n := range g.nodes {
		n.layer = -1
	}
	queue := []int{0}
	if len(g.nodes) > 0 {
		g.nodes[0].layer = 0
	}
	for len(queue) > 0 {
		a := queue[0]
		queue = queue[1:]
		for _, e := range g.edges {
			if e.a == a && g.nodes[e.b].layer < 0 {
				g.nodes[e.b].layer = g.nodes[a].layer + 1
				queue = append(queue, e.b)
			}
		}
	}
	maxL := 0
	for _, n := range g.nodes {
		if n.layer < 0 {
			n.layer = maxL + 1
		}
		maxL = max(maxL, n.layer)
	}
	per := make([]int, maxL+1)
	for _, n := range g.nodes {
		n.idx = per[n.layer]
		per[n.layer]++
	}
	rows := 0
	for _, c := range per {
		rows = max(rows, c)
	}
	H = top + float64(rows)*(nh+gapY) - gapY + 70 // room for back edges under
	W = 24 + float64(maxL+1)*(nw+gapX) - gapX
	for _, n := range g.nodes {
		n.w, n.h = nw, nh
		if n.shape == "dia" {
			n.w, n.h = nw*0.9, nh*1.45
		}
		col := per[n.layer]
		n.x = 12 + nw/2 + float64(n.layer)*(nw+gapX)
		n.y = top + nh/2 + (float64(rows-col)*(nh+gapY))/2 + float64(n.idx)*(nh+gapY)
	}
	return W, H
}

// edgePath is the path of edge e in absolute coordinates, where its label
// goes and the arrowhead at its end.
func (g *graph) edgePath(e gEdge, below, above float64) (d string, lx, ly float64) {
	a, b := g.nodes[e.a], g.nodes[e.b]
	head := func(x, y, ang float64) string {
		s1, c1 := math.Sincos(ang + 2.6)
		s2, c2 := math.Sincos(ang - 2.6)
		return fmt.Sprintf(" M%.1f %.1f L%.1f %.1f L%.1f %.1f", x+9*c1, y+9*s1, x, y, x+9*c2, y+9*s2)
	}
	switch {
	case e.a == e.b: // a loop on itself, over the node
		x, y := a.x, a.y-a.h/2
		return fmt.Sprintf("M%.1f %.1f C%.1f %.1f %.1f %.1f %.1f %.1f", x-14, y, x-30, y-44, x+30, y-44, x+14, y) + head(x+14, y, math.Pi/2+0.5), x, y - 40
	case b.layer == a.layer: // the same layer: straight up or down between them
		y1, y2 := a.y+a.h/2, b.y-b.h/2
		ang := math.Pi / 2
		if b.y < a.y {
			y1, y2, ang = a.y-a.h/2, b.y+b.h/2, -math.Pi/2
		}
		return fmt.Sprintf("M%.1f %.1f L%.1f %.1f", a.x, y1, b.x, y2) + head(b.x, y2, ang), a.x + 30, (y1 + y2) / 2
	case b.layer > a.layer:
		x1, y1, x2, y2 := a.x+a.w/2, a.y, b.x-b.w/2, b.y
		mx := (x1 + x2) / 2
		return fmt.Sprintf("M%.1f %.1f C%.1f %.1f %.1f %.1f %.1f %.1f", x1, y1, mx, y1, mx, y2, x2, y2) + head(x2, y2, 0), mx, (y1+y2)/2 - 9
	case b.idx == 0 && above > 0: // back to the top box of its layer: over the top
		x1, y1, x2, y2 := a.x, a.y-a.h/2, b.x, b.y-b.h/2
		return fmt.Sprintf("M%.1f %.1f C%.1f %.1f %.1f %.1f %.1f %.1f", x1, y1, x1, above, x2, above, x2, y2) + head(x2, y2, math.Pi/2), (x1 + x2) / 2, above + 4
	default: // back (or across) a layer: a curve under both
		x1, y1, x2, y2 := a.x, a.y+a.h/2, b.x, b.y+b.h/2
		return fmt.Sprintf("M%.1f %.1f C%.1f %.1f %.1f %.1f %.1f %.1f", x1, y1, x1, below, x2, below, x2, y2) + head(x2, y2, -math.Pi/2), (x1 + x2) / 2, below - 22
	}
}

// drawGraph adds the edges (behind) and the nodes; it returns their
// element indexes.
func (m *metric) drawGraph(g *graph, below, above float64) (edgeAt, nodeAt []int) {
	for _, e := range g.edges {
		d, lx, ly := g.edgePath(e, below, above)
		edgeAt = append(edgeAt, m.add(DEl{K: "line", C: "si-dg-edge", D: d, TX: lx, TY: ly}, DState{O: 1, P: 1, T: e.label}))
	}
	for _, n := range g.nodes {
		k, c := "box", "si-dg-node si-dg-mid"
		switch n.shape {
		case "dia":
			k, c = "dia", "si-dg-node si-dg-dec si-dg-mid"
		case "pill":
			c += " is-end"
		}
		r := 8.0
		if n.shape == "pill" {
			r = n.h / 2
		}
		nodeAt = append(nodeAt, m.add(DEl{K: k, C: c, R: r}, DState{X: n.x - n.w/2, Y: n.y - n.h/2, W: n.w, H: n.h, O: 1, T: n.name}))
	}
	return edgeAt, nodeAt
}

func graphMetric(kind string, attrs map[string]string) *metric {
	m := &metric{d: &Diagram{Kind: kind, Title: attrs["title"]}}
	if m.d.Title != "" {
		m.add(DEl{K: "text", C: "si-dg-title"}, DState{X: 12, Y: 20, O: 1, T: strings.ToUpper(m.d.Title)})
	}
	return m
}

// ---------- state ----------

func parseState(attrs map[string]string, body string) (*Diagram, []string) {
	var probs []string
	g := &graph{}
	var events []string
	for _, l := range diagramLines(body) {
		switch {
		case eventRE.MatchString(l):
			events = append(events, strings.TrimSpace(eventRE.FindStringSubmatch(l)[1]))
		case stEdgeRE.MatchString(l):
			mm := stEdgeRE.FindStringSubmatch(l)
			from := strings.TrimSpace(mm[1])
			if from == "[*]" || strings.EqualFold(from, "start") {
				g.node(mm[2])
				continue
			}
			g.edges = append(g.edges, gEdge{g.node(from), g.node(mm[2]), strings.TrimSpace(mm[3])})
		default:
			probs = append(probs, fmt.Sprintf("state %q: write A → B on event, or event name for a step", l))
		}
	}
	if len(g.nodes) == 0 {
		return emptyDiagram("state"), append(probs, "state: no transitions (Placed → Paid on authorised)")
	}
	g.nodes[0].shape = "pill"
	m := graphMetric("state", attrs)
	top := 20.0
	if m.d.Title != "" {
		top += 22
	}
	m.d.W, m.d.H = g.layout(150, 46, 110, 34, top+34)
	m.d.W = math.Max(m.d.W, dgWidth(attrs, 0))
	edgeAt, nodeAt := m.drawGraph(g, m.d.H-30, top+10)
	cur := 0
	m.st0[nodeAt[cur]].C = "is-on"
	m.d.States = append(m.d.States, m.st0)
	for _, ev := range events {
		st := dsCopy(m.st0)
		found := -1
		for i, e := range g.edges {
			if e.a == cur && strings.EqualFold(e.label, ev) {
				found = i
				break
			}
		}
		if found < 0 {
			probs = append(probs, fmt.Sprintf("state: no transition from %s on %q", g.nodes[cur].name, ev))
		} else {
			cur = g.edges[found].b
			st[edgeAt[found]].C = "is-on"
		}
		st[nodeAt[0]].C = ""
		st[nodeAt[cur]].C = "is-on"
		m.d.States = append(m.d.States, st)
		m.d.Steps = append(m.d.Steps, ev+" → "+g.nodes[cur].name)
	}
	m.d.In = dsCopy(m.st0)
	for _, i := range edgeAt {
		m.d.In[i].P = 0
	}
	return m.d, probs
}

// ---------- flowchart ----------

func parseFlowchart(attrs map[string]string, body string) (*Diagram, []string) {
	var probs []string
	g := &graph{}
	var tokens [][]int
	for _, l := range diagramLines(body) {
		switch {
		case tokenRE.MatchString(l):
			var path []int
			for _, n := range arrowRE.Split(tokenRE.FindStringSubmatch(l)[1], -1) {
				name := strings.TrimSpace(n)
				i, ok := g.by[strings.ToLower(name)]
				if !ok {
					probs = append(probs, fmt.Sprintf("flowchart token: %q is not a box of the chart", name))
					continue
				}
				path = append(path, i)
			}
			tokens = append(tokens, path)
		case fcEdgeRE.MatchString(l):
			mm := fcEdgeRE.FindStringSubmatch(l)
			g.edges = append(g.edges, gEdge{g.node(mm[1]), g.node(mm[2]), strings.TrimSpace(mm[3])})
		default:
			if strings.TrimSpace(l) != "" && !strings.ContainsAny(l, "→>") {
				g.node(l)
				continue
			}
			probs = append(probs, fmt.Sprintf("flowchart %q: write A → B (or A → B: label), or token A → B → C", l))
		}
	}
	if len(g.nodes) == 0 {
		return emptyDiagram("flowchart"), append(probs, "flowchart: no boxes (Start → Check stock)")
	}
	for _, n := range g.nodes {
		low := strings.ToLower(n.name)
		switch {
		case strings.HasSuffix(n.name, "?"):
			n.shape = "dia"
		case low == "start" || low == "end" || low == "done" || low == "stop":
			n.shape = "pill"
		}
	}
	m := graphMetric("flowchart", attrs)
	top := 20.0
	if m.d.Title != "" {
		top += 22
	}
	m.d.W, m.d.H = g.layout(140, 46, 72, 40, top+34)
	m.d.W = math.Max(m.d.W, dgWidth(attrs, 0))
	edgeAt, nodeAt := m.drawGraph(g, m.d.H-30, top+10)
	tok := m.add(DEl{K: "dot", C: "si-dg-token"}, DState{X: g.nodes[0].x, Y: g.nodes[0].y - g.nodes[0].h/2 - 10, W: 8})
	m.d.States = append(m.d.States, m.st0)
	for _, path := range tokens {
		for j, ni := range path {
			st := dsCopy(m.st0)
			n := g.nodes[ni]
			st[tok].O, st[tok].X, st[tok].Y = 1, n.x, n.y-n.h/2-10
			st[nodeAt[ni]].C = "is-on"
			if j > 0 {
				for ei, e := range g.edges {
					if e.a == path[j-1] && e.b == ni {
						st[edgeAt[ei]].C = "is-on"
					}
				}
			}
			m.d.States = append(m.d.States, st)
			m.d.Steps = append(m.d.Steps, n.name)
		}
	}
	m.d.In = dsCopy(m.st0)
	for _, i := range edgeAt {
		m.d.In[i].P = 0
	}
	for _, i := range nodeAt {
		m.d.In[i].O = 0
	}
	return m.d, probs
}

// ---------- class ----------

func parseClass(attrs map[string]string, body string) (*Diagram, []string) {
	var probs []string
	g := &graph{}
	members := map[int][]string{}
	type rel struct {
		a, b int
		kind string
	}
	var rels []rel
	for _, l := range diagramLines(body) {
		switch {
		case clsRelRE.MatchString(l) && !strings.Contains(l, ":"):
			mm := clsRelRE.FindStringSubmatch(l)
			rels = append(rels, rel{g.node(mm[1]), g.node(mm[3]), strings.ToLower(mm[2])})
		case clsDeclRE.MatchString(l):
			mm := clsDeclRE.FindStringSubmatch(l)
			i := g.node(mm[1])
			for _, f := range strings.Split(mm[2], ",") {
				if f = strings.TrimSpace(f); f != "" {
					members[i] = append(members[i], f)
				}
			}
		default:
			probs = append(probs, fmt.Sprintf("class %q: write Name: field, method(), or A has many B, A is a B, A uses B", l))
		}
	}
	if len(g.nodes) == 0 {
		return emptyDiagram("class"), append(probs, "class: no classes (Order: id, total())")
	}
	m := graphMetric("class", attrs)
	top := 20.0
	if m.d.Title != "" {
		top += 22
	}
	// layers by the relations, left to right (a base class before the ones
	// that extend it), every card as tall as the tallest
	rowH := 0.0
	for i := range g.nodes {
		rowH = math.Max(rowH, 34+18*float64(max(1, len(members[i])))+10)
	}
	for _, r := range rels {
		if strings.HasPrefix(r.kind, "is") {
			g.edges = append(g.edges, gEdge{a: r.b, b: r.a})
		} else {
			g.edges = append(g.edges, gEdge{a: r.a, b: r.b})
		}
	}
	m.d.W, m.d.H = g.layout(190, rowH, 110, 36, top)
	m.d.H -= 50
	m.d.W = math.Max(m.d.W, dgWidth(attrs, 0))
	for i, n := range g.nodes {
		h := 34 + 18*float64(max(1, len(members[i]))) + 10
		n.y += (h - n.h) / 2
		n.h = h
	}
	var relAt []int
	seen := map[[2]int]int{}
	for _, r := range rels {
		a, b := *g.nodes[r.a], *g.nodes[r.b]
		pair := [2]int{min(r.a, r.b), max(r.a, r.b)}
		if k := seen[pair]; k > 0 { // the same two classes again: beside the first
			off := 14 * float64(k)
			ang := math.Atan2(b.y-a.y, b.x-a.x) + math.Pi/2
			dx, dy := off*math.Cos(ang), off*math.Sin(ang)
			a.x, a.y, b.x, b.y = a.x+dx, a.y+dy, b.x+dx, b.y+dy
		}
		seen[pair]++
		x1, y1, x2, y2 := clipToBox(&a, b.x, b.y), 0.0, 0.0, 0.0
		y1 = x1[1]
		end := clipToBox(&b, a.x, a.y)
		x2, y2 = end[0], end[1]
		ang := math.Atan2(y2-y1, x2-x1[0])
		d := fmt.Sprintf("M%.1f %.1f L%.1f %.1f", x1[0], y1, x2, y2)
		label, cls := "", "si-dg-rel"
		switch {
		case strings.HasPrefix(r.kind, "is"): // inheritance: a hollow triangle
			s1, c1 := math.Sincos(ang + 2.7)
			s2, c2 := math.Sincos(ang - 2.7)
			d += fmt.Sprintf(" M%.1f %.1f L%.1f %.1f L%.1f %.1f Z", x2+13*c1, y2+13*s1, x2, y2, x2+13*c2, y2+13*s2)
			cls += " is-inherit"
		case r.kind == "uses":
			d += arrowAt(x2, y2, ang)
			cls += " is-uses"
			label = "uses"
		default:
			d += arrowAt(x2, y2, ang)
			switch r.kind {
			case "has many":
				label = "1 → *"
			case "has one", "has":
				label = "1 → 1"
			}
		}
		relAt = append(relAt, m.add(DEl{K: "line", C: cls, D: d, TX: (x1[0] + x2) / 2, TY: (y1+y2)/2 - 8}, DState{O: 1, P: 1, T: label}))
	}
	var nodeAt []int
	for i, n := range g.nodes {
		nodeAt = append(nodeAt, m.add(DEl{K: "card", C: "si-dg-cls", R: 6, Lines: members[i]}, DState{X: n.x - n.w/2, Y: n.y - n.h/2, W: n.w, H: n.h, O: 1, T: n.name}))
	}
	// one class per step, a relation once both of its ends are shown
	show := func(st []DState, upto int) {
		for i := range g.nodes {
			if i >= upto {
				st[nodeAt[i]].O = 0
			}
		}
		for j, r := range rels {
			if r.a >= upto || r.b >= upto {
				st[relAt[j]].O, st[relAt[j]].P = 0, 0
			}
		}
	}
	base := dsCopy(m.st0)
	show(m.st0, 1)
	m.d.States = append(m.d.States, m.st0)
	for k := 2; k <= len(g.nodes); k++ {
		st := dsCopy(base)
		show(st, k)
		m.d.States = append(m.d.States, st)
		m.d.Steps = append(m.d.Steps, g.nodes[k-1].name)
	}
	m.d.In = dsCopy(m.st0)
	m.d.In[nodeAt[0]].O = 0
	return m.d, probs
}

func arrowAt(x, y, ang float64) string {
	s1, c1 := math.Sincos(ang + 2.6)
	s2, c2 := math.Sincos(ang - 2.6)
	return fmt.Sprintf(" M%.1f %.1f L%.1f %.1f L%.1f %.1f", x+10*c1, y+10*s1, x, y, x+10*c2, y+10*s2)
}

// clipToBox is where the line from n's centre towards (tx, ty) leaves n.
func clipToBox(n *gNode, tx, ty float64) [2]float64 {
	dx, dy := tx-n.x, ty-n.y
	if dx == 0 && dy == 0 {
		return [2]float64{n.x, n.y}
	}
	sx, sy := math.Inf(1), math.Inf(1)
	if dx != 0 {
		sx = (n.w / 2) / math.Abs(dx)
	}
	if dy != 0 {
		sy = (n.h / 2) / math.Abs(dy)
	}
	s := math.Min(sx, sy)
	return [2]float64{n.x + dx*s, n.y + dy*s}
}
