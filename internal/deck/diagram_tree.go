package deck

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// ```wbs: an indented list (the first item the project) drawn as a work
// breakdown: the project on top, its parts in a row, their parts stacked
// under each, numbered 1, 1.1, 1.1.2, `· 3d` estimates added up the tree;
// one level per step. ```storymap: a backbone of activities (`## A | B |
// C`), the stories under each (a row of `|` cells, stories split by `·`),
// and `release Name: story, story` lines, one release slice per step.

var (
	wbsItemRE = regexp.MustCompile(`^(\s*)[-*+]\s+(.*)$`)
	wbsEstRE  = regexp.MustCompile(`^(\d+(?:\.\d+)?)\s*(d|w|h|pts?|points?)?$`)
)

type wbsNode struct {
	name, num string
	est       float64
	unit      string
	depth     int
	kids      []*wbsNode
	x, y      float64
}

func parseWBS(attrs map[string]string, body string) (*Diagram, []string) {
	var probs []string
	var root *wbsNode
	var stack []*wbsNode
	var indents []int
	for _, l := range strings.Split(body, "\n") {
		if strings.TrimSpace(l) == "" || strings.HasPrefix(strings.TrimSpace(l), "//") {
			continue
		}
		mm := wbsItemRE.FindStringSubmatch(strings.ReplaceAll(l, "\t", "    "))
		if mm == nil {
			probs = append(probs, fmt.Sprintf("wbs %q: write an indented list (- Project, then   - Part)", strings.TrimSpace(l)))
			continue
		}
		ind := len(mm[1])
		f := dotFields(mm[2])
		n := &wbsNode{name: f[0]}
		for _, o := range f[1:] {
			if e := wbsEstRE.FindStringSubmatch(o); e != nil {
				n.est, _ = strconv.ParseFloat(e[1], 64)
				n.unit = e[2]
			}
		}
		for len(indents) > 0 && indents[len(indents)-1] >= ind {
			indents, stack = indents[:len(indents)-1], stack[:len(stack)-1]
		}
		if len(stack) == 0 {
			if root != nil {
				probs = append(probs, fmt.Sprintf("wbs %q: one project at the top; indent the parts under it", n.name))
				continue
			}
			root = n
		} else {
			p := stack[len(stack)-1]
			n.depth = p.depth + 1
			p.kids = append(p.kids, n)
		}
		stack, indents = append(stack, n), append(indents, ind)
	}
	if root == nil {
		return emptyDiagram("wbs"), append(probs, "wbs: no items (- Parcel lockers, then   - Design)")
	}
	// numbers and rolled-up estimates
	var number func(n *wbsNode, num string) float64
	unit := ""
	number = func(n *wbsNode, num string) float64 {
		n.num = num
		if n.unit != "" {
			unit = n.unit
		}
		if len(n.kids) == 0 {
			return n.est
		}
		sum := 0.0
		for i, k := range n.kids {
			p := strconv.Itoa(i + 1)
			if num != "" {
				p = num + "." + p
			}
			sum += number(k, p)
		}
		if n.est == 0 {
			n.est = sum
		}
		return n.est
	}
	number(root, "")
	m := graphMetric("wbs", attrs)
	top := 16.0
	if m.d.Title != "" {
		top += 22
	}
	W := dgWidth(attrs, 1000)
	cols := max(1, len(root.kids))
	cw := math.Min(220, (W-24-float64(cols-1)*14)/float64(cols))
	rootW, bh := math.Min(300, W*.4), 46.0
	root.x, root.y = W/2, top+bh/2
	l1y := top + bh + 50
	maxY := l1y + bh
	var lines []int
	var boxes []int
	var depthOf []int
	est := func(n *wbsNode) string {
		if n.est == 0 {
			return n.num
		}
		e := trimNum(n.est) + " " + unitWord(orStr(n.unit, unit))
		if n.est == 1 {
			e = strings.TrimSuffix(e, "s")
		}
		if n.num == "" {
			return e
		}
		return n.num + " · " + e
	}
	rowW := float64(cols)*cw + float64(cols-1)*14
	x0 := (W - rowW) / 2
	var place func(n *wbsNode, x, y float64) float64
	place = func(n *wbsNode, x, y float64) float64 {
		for _, k := range n.kids {
			k.x, k.y = x+16*float64(k.depth-1), y+bh/2
			y = place(k, x, y+bh+10)
			maxY = math.Max(maxY, k.y+bh/2)
		}
		return y
	}
	for i, k := range root.kids {
		k.x, k.y = x0+cw/2+float64(i)*(cw+14), l1y+bh/2
		place(k, x0+float64(i)*(cw+14)+cw*0.12, l1y+bh+14)
	}
	// connectors: the project to each part (an elbow), a part down its column
	for _, k := range root.kids {
		d := fmt.Sprintf("M%.1f %.1f V%.1f H%.1f V%.1f", root.x, root.y+bh/2, root.y+bh/2+25, k.x, k.y-bh/2)
		lines = append(lines, m.add(DEl{K: "line", C: "si-dg-tree", D: d}, DState{O: 1, P: 1}))
		depthOf = append(depthOf, 1)
	}
	var connect func(n *wbsNode)
	connect = func(n *wbsNode) {
		for _, k := range n.kids {
			px := n.x - cw*0.5 + 10
			if n.depth >= 1 {
				px = n.x - cw*0.38 + 6
			}
			d := fmt.Sprintf("M%.1f %.1f V%.1f H%.1f", px, n.y+bh/2, k.y, k.x-cw*0.38)
			lines = append(lines, m.add(DEl{K: "line", C: "si-dg-tree", D: d}, DState{O: 1, P: 1}))
			depthOf = append(depthOf, k.depth)
			connect(k)
		}
	}
	for _, k := range root.kids {
		connect(k)
	}
	var all []*wbsNode
	var walk func(n *wbsNode)
	walk = func(n *wbsNode) {
		all = append(all, n)
		for _, k := range n.kids {
			walk(k)
		}
	}
	walk(root)
	depth := 0
	for _, n := range all {
		w := cw * 0.76
		switch n.depth {
		case 0:
			w = rootW
		case 1:
			w = cw
		}
		cls := fmt.Sprintf("si-dg-wb is-d%d", min(n.depth, 3))
		boxes = append(boxes, m.add(DEl{K: "box", C: cls, R: 6}, DState{X: n.x - w/2, Y: n.y - bh/2, W: w, H: bh, O: 1, T: n.name, U: est(n)}))
		depthOf = append(depthOf, n.depth)
		depth = max(depth, n.depth)
	}
	m.d.W, m.d.H = W, maxY+16
	// one level per step: lines and boxes of depth ≤ k
	all0 := len(m.d.Els) - len(boxes) - len(lines)
	levelState := func(k int) []DState {
		st := dsCopy(m.st0)
		for j := range depthOf {
			if depthOf[j] > k {
				st[all0+j].O, st[all0+j].P = 0, 0
			}
		}
		return st
	}
	m.d.States = append(m.d.States, levelState(0))
	for k := 1; k <= depth; k++ {
		m.d.States = append(m.d.States, levelState(k))
		m.d.Steps = append(m.d.Steps, fmt.Sprintf("level %d", k))
	}
	m.d.In = dsCopy(m.d.States[0])
	m.d.In[boxes[0]].O = 0
	if root.est > 0 {
		m.d.Facts = append(m.d.Facts, fmt.Sprintf("%d parts, %s %s in all", len(all)-1, trimNum(root.est), unitWord(unit)))
	}
	return m.d, probs
}

func unitWord(u string) string {
	switch u {
	case "d", "":
		return "days"
	case "w":
		return "weeks"
	case "h":
		return "hours"
	}
	return "points"
}

// ---------- storymap ----------

var releaseRE = regexp.MustCompile(`(?i)^release\s+(.+?)\s*:\s*(.*)$`)

func parseStorymap(attrs map[string]string, body string) (*Diagram, []string) {
	var probs []string
	var acts []string
	var stories [][]string
	type rel struct {
		name  string
		items map[string]bool
	}
	var rels []rel
	for _, l := range diagramLines(body) {
		switch {
		case strings.HasPrefix(l, "#"):
			for _, a := range strings.Split(strings.TrimLeft(l, "# "), "|") {
				if a = strings.TrimSpace(a); a != "" {
					acts = append(acts, a)
					stories = append(stories, nil)
				}
			}
		case releaseRE.MatchString(l):
			mm := releaseRE.FindStringSubmatch(l)
			r := rel{name: mm[1], items: map[string]bool{}}
			for _, s := range strings.Split(mm[2], ",") {
				if s = strings.TrimSpace(s); s != "" {
					r.items[strings.ToLower(s)] = true
				}
			}
			rels = append(rels, r)
		case strings.Contains(l, "|") || len(acts) > 0:
			cells := strings.Split(l, "|")
			if len(cells) > len(acts) {
				probs = append(probs, fmt.Sprintf("storymap %q: more cells than activities (%d)", l, len(acts)))
			}
			for i, c := range cells {
				if i < len(acts) {
					stories[i] = append(stories[i], dotFields(c)...)
				}
			}
		default:
			probs = append(probs, fmt.Sprintf("storymap %q: start with ## Activity | Activity", l))
		}
	}
	if len(acts) == 0 {
		return emptyDiagram("storymap"), append(probs, "storymap: no activities (## Find | Choose | Pay)")
	}
	// each story's release (its index; len(rels): not yet planned)
	relOf := func(s string) int {
		for i, r := range rels {
			if r.items[strings.ToLower(s)] {
				return i
			}
		}
		return len(rels)
	}
	known := map[string]bool{}
	for _, col := range stories {
		for _, s := range col {
			known[strings.ToLower(s)] = true
		}
	}
	for _, r := range rels {
		for s := range r.items {
			if !known[s] {
				probs = append(probs, fmt.Sprintf("storymap release %s: %q is not a story of the map", r.name, s))
			}
		}
	}
	m := graphMetric("storymap", attrs)
	top := 16.0
	if m.d.Title != "" {
		top += 22
	}
	W := dgWidth(attrs, 1000)
	n := len(acts)
	labW := 96.0
	cw := (W - 24 - labW - float64(n-1)*12) / float64(n)
	ah, sh := 44.0, 34.0
	colX := func(i int) float64 { return 12 + labW + float64(i)*(cw+12) }
	for i, a := range acts {
		m.add(DEl{K: "box", C: "si-dg-activity si-dg-mid", R: 6}, DState{X: colX(i), Y: top, W: cw, H: ah, O: 1, T: a})
	}
	// bands: one per release (and one for the rest), as tall as its tallest column
	bands := len(rels) + 1
	bandH := make([]float64, bands)
	count := make([][]int, bands)
	for b := range count {
		count[b] = make([]int, n)
	}
	type sPos struct{ band, col, slot int }
	var pos []sPos
	var names []string
	for c, col := range stories {
		for _, s := range col {
			b := relOf(s)
			pos = append(pos, sPos{b, c, count[b][c]})
			names = append(names, s)
			count[b][c]++
		}
	}
	for b := range bands {
		k := 0
		for _, v := range count[b] {
			k = max(k, v)
		}
		bandH[b] = math.Max(1, float64(k))*(sh+8) + 16
	}
	bandY := make([]float64, bands+1)
	bandY[0] = top + ah + 14
	for b := range bands {
		bandY[b+1] = bandY[b] + bandH[b]
	}
	var lineAt []int
	for b, r := range rels {
		y := bandY[b]
		lineAt = append(lineAt, m.add(DEl{K: "line", C: "si-dg-release", D: fmt.Sprintf("M0 0H%g", W-24), TX: 4, TY: 18}, DState{X: 12, Y: y - 6, O: 0, T: r.name}))
	}
	later := -1
	if count[len(rels)] != nil {
		total := 0
		for _, v := range count[len(rels)] {
			total += v
		}
		if total > 0 && len(rels) > 0 {
			later = m.add(DEl{K: "line", C: "si-dg-release is-later", D: fmt.Sprintf("M0 0H%g", W-24), TX: 4, TY: 18}, DState{X: 12, Y: bandY[len(rels)] - 6, O: 0, T: "later"})
		}
	}
	s0 := len(m.d.Els)
	for i, p := range pos {
		m.add(DEl{K: "box", C: "si-dg-story", R: 4}, DState{X: colX(p.col) + 6, Y: bandY[p.band] + 10 + float64(p.slot)*(sh+8), W: cw - 12, H: sh, O: 0, T: names[i]})
	}
	m.d.W, m.d.H = W, bandY[bands]+8
	m.d.States = append(m.d.States, m.st0)
	show := func(st []DState, upto int) { // releases 0..upto-1, and (upto > len) the rest
		for i, p := range pos {
			if p.band < upto {
				st[s0+i].O = 1
			}
		}
		for b := range rels {
			if b < upto {
				st[lineAt[b]].O, st[lineAt[b]].P = 1, 1
			}
		}
		if later >= 0 && upto > len(rels) {
			st[later].O, st[later].P = 1, 1
		}
	}
	steps := len(rels)
	if len(rels) == 0 || later >= 0 {
		steps++
	}
	for k := 1; k <= steps; k++ {
		st := dsCopy(m.st0)
		show(st, k)
		m.d.States = append(m.d.States, st)
		if k <= len(rels) {
			m.d.Steps = append(m.d.Steps, "release "+rels[k-1].name)
		} else {
			m.d.Steps = append(m.d.Steps, "the rest")
		}
	}
	m.d.In = dsCopy(m.st0)
	a0 := 0
	if m.d.Title != "" {
		a0 = 1 // the title comes first
	}
	for i := range acts {
		m.d.In[a0+i].O = 0
	}
	return m.d, probs
}
