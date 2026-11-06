package tui

import (
	"encoding/xml"
	"image/color"
	"io"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
)

// Animated SVG canvases (<!-- svg: file state -->) drawn with characters:
// boxes, wires, text and dots on a grid scaled from the viewBox, in the state
// the deck has reached. The same rules as the browser decide what shows: the
// helper attributes (data-show, data-hide, data-ok, data-alert, data-flow,
// data-packets, data-grow) and the canvas's own CSS state rules
// (.root[data-state~="w"] .part{transform | opacity | fill | display}). Time
// moves the rest: packets travel along their wires, flows march, alerts
// pulse, and a part that moves between two states slides there.

type svgNode struct {
	tag      string
	attr     map[string]string
	children []*svgNode
	text     string
	parent   *svgNode
}

func parseSVG(r io.Reader) (*svgNode, string, error) {
	dec := xml.NewDecoder(r)
	dec.Strict = false
	var root, cur *svgNode
	var style strings.Builder
	inStyle := false
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, "", err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			n := &svgNode{tag: t.Name.Local, attr: map[string]string{}, parent: cur}
			for _, a := range t.Attr {
				n.attr[a.Name.Local] = a.Value
			}
			if n.tag == "style" {
				inStyle = true
			}
			if cur != nil {
				cur.children = append(cur.children, n)
			} else {
				root = n
			}
			cur = n
		case xml.EndElement:
			if t.Name.Local == "style" {
				inStyle = false
			}
			if cur != nil {
				cur = cur.parent
			}
		case xml.CharData:
			if inStyle {
				style.Write(t)
			} else if cur != nil && (cur.tag == "text" || cur.tag == "tspan") {
				// a text run, kept in order among the tspans
				cur.children = append(cur.children, &svgNode{tag: "#text", text: string(t), parent: cur, attr: map[string]string{}})
			}
		}
	}
	return root, style.String(), nil
}

// ---------- the canvas's CSS ----------

type cssRule struct {
	state string     // the data-state word the rule needs ("" = always)
	chain []compound // descendant selectors after the root; the last matches the element
	decl  map[string]string
}

type compound struct {
	tag     string
	classes []string
}

var (
	cssCommentRE = regexp.MustCompile(`(?s)/\*.*?\*/`)
	cssRuleRE    = regexp.MustCompile(`([^{}]+)\{([^{}]*)\}`)
	stateCondRE  = regexp.MustCompile(`^\.[\w-]+\[data-state~="([\w-]+)"\]$`)
	compoundRE   = regexp.MustCompile(`^([a-z]+)?((?:\.[\w-]+)*)$`)
)

func parseCSS(src string) []cssRule {
	var out []cssRule
	for _, m := range cssRuleRE.FindAllStringSubmatch(cssCommentRE.ReplaceAllString(src, ""), -1) {
		decl := map[string]string{}
		for _, kv := range strings.Split(m[2], ";") {
			k, v, ok := strings.Cut(kv, ":")
			if ok {
				decl[strings.TrimSpace(k)] = strings.TrimSpace(v)
			}
		}
	sel:
		for _, sel := range strings.Split(m[1], ",") {
			parts := strings.Fields(strings.TrimSpace(sel))
			if len(parts) == 0 || strings.Contains(sel, ":") || strings.Contains(sel, ">") {
				continue
			}
			r := cssRule{decl: decl}
			if sm := stateCondRE.FindStringSubmatch(parts[0]); sm != nil {
				r.state = sm[1]
			} else if strings.Contains(parts[0], "[") || !strings.HasPrefix(parts[0], ".") {
				continue
			}
			for _, p := range parts[1:] {
				cm := compoundRE.FindStringSubmatch(p)
				if cm == nil {
					continue sel
				}
				c := compound{tag: cm[1]}
				if cm[2] != "" {
					c.classes = strings.Split(strings.TrimPrefix(cm[2], "."), ".")
				}
				r.chain = append(r.chain, c)
			}
			out = append(out, r)
		}
	}
	return out
}

func (c compound) matches(n *svgNode) bool {
	if c.tag != "" && c.tag != n.tag {
		return false
	}
	have := strings.Fields(n.attr["class"])
	for _, want := range c.classes {
		found := false
		for _, h := range have {
			if h == want {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func (r cssRule) matches(n *svgNode) bool {
	if len(r.chain) == 0 {
		return false // a rule on the root itself
	}
	if !r.chain[len(r.chain)-1].matches(n) {
		return false
	}
	i := len(r.chain) - 2
	for a := n.parent; a != nil && i >= 0; a = a.parent {
		if r.chain[i].matches(a) {
			i--
		}
	}
	return i < 0
}

// ---------- the state of each element ----------

type look struct {
	hidden      bool
	dx, dy      float64 // translate, in viewBox units
	sx          float64 // scaleX (bars), 1 by default
	fill        string  // the CSS value, for its colour role
	stroke      string
	ok, alert   bool
	flow        bool
	packets     bool
	bold        bool
	hasFill     bool // a filled shape (vs. fill:none)
	strokeShown bool
}

func hasWord(state []string, list string) bool {
	for _, w := range strings.Fields(list) {
		for _, s := range state {
			if s == w {
				return true
			}
		}
	}
	return false
}

func translateOf(v string) (dx, dy, sx float64, ok bool) {
	sx = 1
	for _, m := range transformRE.FindAllStringSubmatch(v, -1) {
		nums := numbers(m[2])
		ok = true
		switch m[1] {
		case "translate":
			if len(nums) > 0 {
				dx += nums[0]
			}
			if len(nums) > 1 {
				dy += nums[1]
			}
		case "translateX":
			if len(nums) > 0 {
				dx += nums[0]
			}
		case "translateY":
			if len(nums) > 0 {
				dy += nums[0]
			}
		case "scaleX", "scale":
			if len(nums) > 0 {
				sx = nums[0]
			}
		}
	}
	return
}

var (
	transformRE = regexp.MustCompile(`(translate|translateX|translateY|scaleX|scale)\(([^)]*)\)`)
	numRE       = regexp.MustCompile(`-?\d*\.?\d+(?:e-?\d+)?`)
)

func numbers(s string) []float64 {
	var out []float64
	for _, m := range numRE.FindAllString(s, -1) {
		f, _ := strconv.ParseFloat(m, 64)
		out = append(out, f)
	}
	return out
}

// lookOf computes an element's look in a state: inherited from its parents,
// then its CSS rules in source order, then its helper attributes.
func (c *canvas) lookOf(n *svgNode, state []string, parent look) look {
	l := parent
	l.dx, l.dy = parent.dx, parent.dy
	l.sx = 1
	if t := n.attr["transform"]; t != "" {
		dx, dy, _, _ := translateOf(t)
		l.dx, l.dy = l.dx+dx, l.dy+dy
	}
	if v, ok := n.attr["fill"]; ok {
		l.fill = v
	}
	if v, ok := n.attr["stroke"]; ok {
		l.stroke = v
	}
	if fw := n.attr["font-weight"]; fw == "bold" || fw >= "600" {
		l.bold = true
	}
	var tdx, tdy float64
	moved := false
	for _, r := range c.css {
		if r.state != "" && !hasWord(state, r.state) || !r.matches(n) {
			continue
		}
		for k, v := range r.decl {
			switch k {
			case "fill":
				l.fill = v
			case "stroke":
				l.stroke = v
			case "opacity":
				if f, err := strconv.ParseFloat(v, 64); err == nil {
					l.hidden = f < 0.2
				}
			case "display":
				l.hidden = v == "none"
			case "font-weight":
				l.bold = v == "bold" || v >= "600"
			case "transform":
				if dx, dy, sx, ok := translateOf(v); ok {
					tdx, tdy, moved = dx, dy, true
					l.sx = sx
				}
			}
		}
	}
	if moved {
		l.dx, l.dy = l.dx+tdx, l.dy+tdy
	}
	if v, ok := n.attr["opacity"]; ok {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f < 0.2 {
			l.hidden = true
		}
	}
	if v, ok := n.attr["data-show"]; ok && !hasWord(state, v) {
		l.hidden = true
	}
	if v, ok := n.attr["data-hide"]; ok && hasWord(state, v) {
		l.hidden = true
	}
	if v, ok := n.attr["data-grow"]; ok && !hasWord(state, v) {
		l.hidden = true
	}
	if v, ok := n.attr["data-ok"]; ok && hasWord(state, v) {
		l.ok = true
	}
	if v, ok := n.attr["data-alert"]; ok && hasWord(state, v) {
		l.alert = true
	}
	l.flow = hasWord(state, n.attr["data-flow"])
	l.packets = hasWord(state, n.attr["data-packets"])
	return l
}

// ---------- drawing ----------

type cell struct {
	ch     rune
	fg     color.Color
	bg     color.Color
	bold   bool
	border bool // a box's edge: wires and packets go under it
	text   bool // a letter: wires go under it too
}

type canvas struct {
	root       *svgNode
	css        []cssRule
	vx, vy     float64 // viewBox origin
	vw, vh     float64
	pal        Palette
	cols, rows int
	grid       [][]cell
	t          float64 // seconds, for packets, flows and pulses
}

func loadCanvas(path string) (*canvas, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	root, style, err := parseSVG(f)
	if err != nil || root == nil {
		return nil, err
	}
	c := &canvas{root: root, css: parseCSS(style)}
	if vb := numbers(root.attr["viewBox"]); len(vb) == 4 {
		c.vx, c.vy, c.vw, c.vh = vb[0], vb[1], vb[2], vb[3]
	} else {
		w, h := numbers(root.attr["width"]), numbers(root.attr["height"])
		if len(w) > 0 && len(h) > 0 {
			c.vw, c.vh = w[0], h[0]
		}
	}
	if c.vw <= 0 || c.vh <= 0 {
		c.vw, c.vh = 1200, 600
	}
	return c, nil
}

// size picks the grid for a width: cells are about twice as tall as wide.
func (c *canvas) size(maxCols, maxRows int) {
	c.cols = maxCols
	c.rows = int(math.Round(float64(c.cols) * c.vh / c.vw / 2))
	if c.rows > maxRows {
		c.rows = maxRows
		c.cols = int(math.Round(float64(c.rows) * 2 * c.vw / c.vh))
	}
	c.cols, c.rows = max(c.cols, 10), max(c.rows, 4)
}

func (c *canvas) X(x float64) int { return int(math.Round((x - c.vx) / c.vw * float64(c.cols-1))) }
func (c *canvas) Y(y float64) int { return int(math.Round((y - c.vy) / c.vh * float64(c.rows-1))) }

func (c *canvas) set(x, y int, ch rune, fg color.Color, bold bool) {
	if y < 0 || y >= c.rows || x < 0 || x >= c.cols {
		return
	}
	cl := &c.grid[y][x]
	cl.ch, cl.bold = ch, bold
	if fg != nil {
		cl.fg = fg
	}
}

func (c *canvas) paint(x, y int, bg color.Color) {
	if y < 0 || y >= c.rows || x < 0 || x >= c.cols {
		return
	}
	c.grid[y][x].bg = bg
	c.grid[y][x].ch = ' '
}

// role maps a CSS colour (usually var(--si-…, #fallback)) to the palette.
func (c *canvas) role(v string) color.Color {
	p := c.pal
	switch {
	case v == "" || v == "none" || v == "transparent":
		return nil
	case strings.Contains(v, "--si-crit"):
		return p.Crit
	case strings.Contains(v, "--si-warn"):
		return p.Warn
	case strings.Contains(v, "--si-ok"):
		return p.OK
	case strings.Contains(v, "--si-accent-2"):
		return p.Accent2
	case strings.Contains(v, "--si-on-accent"):
		return p.BG
	case strings.Contains(v, "--si-accent"):
		return p.Accent
	case strings.Contains(v, "--si-muted"):
		return p.Muted
	case strings.Contains(v, "--si-fg"):
		return p.FG
	case strings.Contains(v, "--si-line"), strings.Contains(v, "--si-surface"), strings.Contains(v, "--si-bg"):
		return p.Line
	}
	if h, ok := parseHex(strings.TrimSpace(v)); ok {
		r, g, b, _ := h.RGBA()
		if r>>8 > 0xe0 && g>>8 > 0xe0 && b>>8 > 0xe0 {
			return p.BG // white text sits on a coloured shape: it takes the slide's background
		}
		return h
	}
	if strings.HasPrefix(v, "var(") {
		if i := strings.LastIndex(v, ","); i > 0 {
			return c.role(strings.TrimSuffix(strings.TrimSpace(v[i+1:]), ")"))
		}
	}
	return nil
}

// render draws the canvas in a state. prev and p tween parts that move:
// at p < 1 a part between its old and new place.
func (c *canvas) render(state, prev []string, p float64) []string {
	c.grid = make([][]cell, c.rows)
	for y := range c.grid {
		c.grid[y] = make([]cell, c.cols)
		for x := range c.grid[y] {
			c.grid[y][x] = cell{ch: ' '}
		}
	}
	base := look{sx: 1, fill: "", stroke: ""}
	var walk func(n *svgNode, l, lp look)
	walk = func(n *svgNode, parent, parentPrev look) {
		if n.tag == "style" || n.tag == "defs" || n.tag == "title" || n.tag == "desc" {
			return
		}
		l := c.lookOf(n, state, parent)
		if prev != nil && p < 1 {
			lp := c.lookOf(n, prev, parentPrev)
			// slide from the old place; appear in the second half, vanish in the first
			l.dx = lp.dx + (l.dx-lp.dx)*ease(p)
			l.dy = lp.dy + (l.dy-lp.dy)*ease(p)
			if l.hidden != lp.hidden && p < 0.5 {
				l.hidden = lp.hidden
			}
			if l.sx != lp.sx {
				l.sx = lp.sx + (l.sx-lp.sx)*ease(p)
			}
			parentPrev = lp
		} else {
			parentPrev = l
		}
		if l.hidden {
			return
		}
		c.draw(n, l)
		for _, ch := range n.children {
			walk(ch, l, parentPrev)
		}
	}
	walk(c.root, base, base)
	out := make([]string, c.rows)
	for y, row := range c.grid {
		var b strings.Builder
		for _, cl := range row {
			st := lipgloss.NewStyle()
			if cl.fg != nil {
				st = st.Foreground(cl.fg)
			}
			if cl.bg != nil {
				st = st.Background(cl.bg)
			}
			if cl.bold {
				st = st.Bold(true)
			}
			b.WriteString(st.Render(string(cl.ch)))
		}
		out[y] = b.String()
	}
	return out
}

func ease(p float64) float64 { return 1 - math.Pow(1-p, 3) }

func attrF(n *svgNode, k string) float64 {
	v := numbers(n.attr[k])
	if len(v) == 0 {
		return 0
	}
	return v[0]
}

func (c *canvas) draw(n *svgNode, l look) {
	strokeCol := c.role(l.stroke)
	fillCol := c.role(l.fill)
	if l.ok {
		strokeCol = c.pal.OK
	}
	if l.alert {
		strokeCol = c.pal.Crit
		if int(c.t*1.6)%2 == 1 { // the pulse: a dimmer red every other beat
			strokeCol = mix(c.pal.Crit, c.pal.BG, 0.45)
		}
	}
	switch n.tag {
	case "rect":
		x, y := attrF(n, "x")+l.dx, attrF(n, "y")+l.dy
		w, h := attrF(n, "width")*l.sx, attrF(n, "height")
		x0, y0, x1, y1 := c.X(x), c.Y(y), c.X(x+w), c.Y(y+h)
		if x1 <= x0 {
			x1 = x0 + 1
		}
		filled := fillCol != nil && strokeCol == nil && l.fill != "" && l.fill != "none"
		if y1-y0 < 2 || filled && y1-y0 < 3 {
			// a pill, a tag, a bar: a band of colour (text drawn later sits on it)
			col := fillCol
			if col == nil {
				col = strokeCol
			}
			if col == nil {
				col = c.pal.Line
			}
			mid := (y0 + y1) / 2
			for xx := x0; xx <= x1; xx++ {
				c.paint(xx, mid, col)
			}
			return
		}
		col := strokeCol
		if col == nil {
			col = c.pal.Line
		}
		rounded := attrF(n, "rx") > 0
		tl, tr, bl, br := '┌', '┐', '└', '┘'
		if rounded {
			tl, tr, bl, br = '╭', '╮', '╰', '╯'
		}
		edge := func(x, y int, ch rune) {
			c.set(x, y, ch, col, false)
			if y >= 0 && y < c.rows && x >= 0 && x < c.cols {
				c.grid[y][x].border = true
			}
		}
		for xx := x0 + 1; xx < x1; xx++ {
			edge(xx, y0, '─')
			edge(xx, y1, '─')
		}
		for yy := y0 + 1; yy < y1; yy++ {
			edge(x0, yy, '│')
			edge(x1, yy, '│')
		}
		edge(x0, y0, tl)
		edge(x1, y0, tr)
		edge(x0, y1, bl)
		edge(x1, y1, br)
	case "circle", "ellipse":
		cx, cy := attrF(n, "cx")+l.dx, attrF(n, "cy")+l.dy
		col := fillCol
		if col == nil {
			col = strokeCol
		}
		if col == nil {
			col = c.pal.Accent
		}
		c.set(c.X(cx), c.Y(cy), '●', col, false)
	case "line":
		pts := [][2]float64{{attrF(n, "x1") + l.dx, attrF(n, "y1") + l.dy}, {attrF(n, "x2") + l.dx, attrF(n, "y2") + l.dy}}
		c.wire(pts, l, strokeCol)
	case "polyline", "polygon":
		v := numbers(n.attr["points"])
		var pts [][2]float64
		for i := 0; i+1 < len(v); i += 2 {
			pts = append(pts, [2]float64{v[i] + l.dx, v[i+1] + l.dy})
		}
		c.wire(pts, l, strokeCol)
	case "path":
		for _, pts := range pathPoints(n.attr["d"]) {
			for i := range pts {
				pts[i][0] += l.dx
				pts[i][1] += l.dy
			}
			c.wire(pts, l, strokeCol)
		}
	case "text":
		c.text(n, l, fillCol)
	}
}

// runs is a text's characters in document order; a tspan with its own
// position (x, y or dy) starts a new run placed on its own.
func runs(n *svgNode) (inline string, placed []*svgNode) {
	var b strings.Builder
	for _, ch := range n.children {
		switch ch.tag {
		case "#text":
			b.WriteString(ch.text)
		case "tspan":
			if ch.attr["x"] != "" || ch.attr["y"] != "" || ch.attr["dy"] != "" {
				placed = append(placed, ch)
				continue
			}
			in, more := runs(ch)
			b.WriteString(in)
			placed = append(placed, more...)
		}
	}
	return strings.Join(strings.Fields(b.String()), " "), placed
}

// text places a text by its anchor, and its positioned tspans below it.
func (c *canvas) text(n *svgNode, l look, col color.Color) {
	s, placed := runs(n)
	x, y := attrF(n, "x")+l.dx, attrF(n, "y")+l.dy
	for _, t := range placed {
		tx, ty := x, y
		if t.attr["x"] != "" {
			tx = attrF(t, "x") + l.dx
		}
		if t.attr["y"] != "" {
			ty = attrF(t, "y") + l.dy
		}
		if dy := t.attr["dy"]; dy != "" {
			d := attrF(t, "dy")
			if strings.HasSuffix(dy, "em") {
				d *= 16
			}
			ty += d
		}
		sub := &svgNode{tag: "text", attr: map[string]string{"x": strconv.FormatFloat(tx-l.dx, 'f', -1, 64), "y": strconv.FormatFloat(ty-l.dy, 'f', -1, 64),
			"text-anchor": n.attr["text-anchor"], "class": n.attr["class"]}, children: t.children}
		c.text(sub, l, col)
		y = ty
	}
	if s == "" {
		return
	}
	if col == nil {
		col = c.pal.FG
	}
	if l.ok {
		col = c.pal.OK
	}
	if l.alert {
		col = c.pal.Crit
	}
	x, y = attrF(n, "x")+l.dx, attrF(n, "y")+l.dy
	cx, cy := c.X(x), c.Y(y-4) // the baseline sits a little under the middle of the line
	rs := []rune(s)
	switch n.attr["text-anchor"] {
	case "middle":
		cx -= len(rs) / 2
	case "end":
		cx -= len(rs)
	}
	// text written on a pill or tag snaps onto its band when rounding put it a row off
	hasBg := func(y int) bool {
		x := cx + len(rs)/2
		return y >= 0 && y < c.rows && x >= 0 && x < c.cols && c.grid[y][x].bg != nil
	}
	if !hasBg(cy) {
		if hasBg(cy - 1) {
			cy--
		} else if hasBg(cy + 1) {
			cy++
		}
	}
	bold := l.bold || n.attr["class"] != "" && strings.Contains(n.attr["class"], "name")
	for i, r := range rs {
		c.set(cx+i, cy, r, col, bold)
		if x := cx + i; cy >= 0 && cy < c.rows && x >= 0 && x < c.cols {
			c.grid[cy][x].text = true
		}
	}
}

// wire draws a polyline; flows march and packets travel along it.
func (c *canvas) wire(pts [][2]float64, l look, col color.Color) {
	if len(pts) < 2 {
		return
	}
	if col == nil {
		col = c.pal.Line
	}
	if l.flow {
		col = c.pal.Accent
	}
	// the cells along the line, in order, leaving out the boxes' edges it meets
	var cells [][2]int
	for i := 0; i+1 < len(pts); i++ {
		for _, p := range line(c.X(pts[i][0]), c.Y(pts[i][1]), c.X(pts[i+1][0]), c.Y(pts[i+1][1])) {
			if p[1] >= 0 && p[1] < c.rows && p[0] >= 0 && p[0] < c.cols && (c.grid[p[1]][p[0]].border || c.grid[p[1]][p[0]].text) {
				continue
			}
			cells = append(cells, p)
		}
	}
	for i, p := range cells {
		ch := '─'
		var dx, dy int
		if i+1 < len(cells) {
			dx, dy = cells[i+1][0]-p[0], cells[i+1][1]-p[1]
		} else if i > 0 {
			dx, dy = p[0]-cells[i-1][0], p[1]-cells[i-1][1]
		}
		switch {
		case dx == 0 && dy != 0:
			ch = '│'
		case dy != 0 && (dx > 0) == (dy > 0):
			ch = '╲'
		case dy != 0:
			ch = '╱'
		}
		if l.flow && (i+int(c.t*8))%4 >= 2 { // marching dashes
			ch = ' '
		}
		c.set(p[0], p[1], ch, col, false)
	}
	if l.packets && len(cells) > 1 {
		n := 2
		for k := 0; k < n; k++ {
			f := math.Mod(c.t/1.6+float64(k)/float64(n), 1)
			p := cells[int(f*float64(len(cells)-1))]
			c.set(p[0], p[1], '●', c.pal.Accent, true)
		}
	}
}

// line is the cells from (x0,y0) to (x1,y1), Bresenham.
func line(x0, y0, x1, y1 int) [][2]int {
	var out [][2]int
	dx, dy := abs(x1-x0), -abs(y1-y0)
	sx, sy := 1, 1
	if x0 > x1 {
		sx = -1
	}
	if y0 > y1 {
		sy = -1
	}
	err := dx + dy
	for {
		out = append(out, [2]int{x0, y0})
		if x0 == x1 && y0 == y1 {
			return out
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

var pathTokRE = regexp.MustCompile(`[MmLlHhVvCcSsQqTtAaZz]|-?\d*\.?\d+(?:e-?\d+)?`)

// pathPoints turns path data into polylines (curves sampled, arcs straightened).
func pathPoints(d string) [][][2]float64 {
	toks := pathTokRE.FindAllString(d, -1)
	var out [][][2]float64
	var cur [][2]float64
	x, y, sx, sy := 0.0, 0.0, 0.0, 0.0
	cmd := ""
	i := 0
	num := func() float64 {
		if i >= len(toks) {
			return 0
		}
		f, _ := strconv.ParseFloat(toks[i], 64)
		i++
		return f
	}
	isNum := func() bool { return i < len(toks) && !strings.ContainsAny(toks[i][:1], "MmLlHhVvCcSsQqTtAaZz") }
	flush := func() {
		if len(cur) > 1 {
			out = append(out, cur)
		}
		cur = nil
	}
	for i < len(toks) {
		if !isNum() {
			cmd = toks[i]
			i++
		}
		rel := cmd == strings.ToLower(cmd)
		ox, oy := 0.0, 0.0
		if rel {
			ox, oy = x, y
		}
		switch strings.ToUpper(cmd) {
		case "M":
			flush()
			x, y = ox+num(), oy+num()
			sx, sy = x, y
			cur = append(cur, [2]float64{x, y})
			if rel {
				cmd = "l"
			} else {
				cmd = "L"
			}
		case "L", "T":
			x, y = ox+num(), oy+num()
			cur = append(cur, [2]float64{x, y})
		case "H":
			x = ox + num()
			cur = append(cur, [2]float64{x, y})
		case "V":
			y = oy + num()
			cur = append(cur, [2]float64{x, y})
		case "C":
			x1, y1, x2, y2 := ox+num(), oy+num(), ox+num(), oy+num()
			ex, ey := ox+num(), oy+num()
			for s := 1; s <= 8; s++ {
				t := float64(s) / 8
				u := 1 - t
				cur = append(cur, [2]float64{u*u*u*x + 3*u*u*t*x1 + 3*u*t*t*x2 + t*t*t*ex, u*u*u*y + 3*u*u*t*y1 + 3*u*t*t*y2 + t*t*t*ey})
			}
			x, y = ex, ey
		case "S", "Q":
			num()
			num()
			x, y = ox+num(), oy+num()
			cur = append(cur, [2]float64{x, y})
		case "A":
			for k := 0; k < 5; k++ {
				num()
			}
			x, y = ox+num(), oy+num()
			cur = append(cur, [2]float64{x, y})
		case "Z":
			x, y = sx, sy
			cur = append(cur, [2]float64{x, y})
			flush()
		default:
			i++ // an unknown token: skip it
		}
	}
	flush()
	return out
}
