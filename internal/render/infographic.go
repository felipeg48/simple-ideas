package render

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"html"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/assets"
	"github.com/felipeg48/simple-ideas/internal/deck"
)

// Infographics: a `type: infographic` page or an ```infographic fence. The
// blocks (deck.ParseInfographic) become a bento grid of cards; charts are SVG
// drawn to scale from their tables; everything is sized from the frame
// (container units), so a landscape infographic is a 16:9 slide at any size
// and a portrait one a poster that scrolls. Styles: web/infographic.css.

// igOpts are the settings of one infographic: front matter of a page, or the
// options of a fence.
type igOpts struct {
	title, subtitle, layout, orient, aspect, sources, byline string
	steps                                                    bool
}

var igRatios = map[string][2]float64{"16:9": {16, 9}, "4:3": {4, 3}, "9:16": {9, 16}, "a4": {210, 297}}

// infographicFence renders an ```infographic fence; `page` on the fence line
// marks the body of a `type: infographic` file (deck.buildInfographic), which
// takes its settings from the front matter.
func (dr *deckRenderer) infographicFence(body string, attrs map[string]string, dir string) string {
	sc := dr.cur
	defer func() { dr.cur = sc }() // the blocks' own Markdown resets it
	m := dr.d.Meta
	o := igOpts{title: attrs["title"], subtitle: attrs["subtitle"], layout: attrs["layout"], orient: attrs["orientation"], aspect: attrs["aspect"], sources: attrs["sources"]}
	_, page := attrs["page"]
	if page {
		o = igOpts{title: m.Title, subtitle: m.Subtitle, layout: m.MapLayout, orient: m.Orientation, aspect: m.Aspect, sources: m.Sources,
			byline: strings.Trim(m.Author+" · "+m.Date, " ·")}
		if m.IGPortrait() {
			o.orient = "portrait"
		}
	} else {
		for k, v := range attrs {
			allowed := map[string][]string{"layout": deck.IGLayouts, "orientation": deck.IGOrientations, "aspect": deck.IGAspects, "steps": deck.OnOff}[k]
			switch {
			case k == "title" || k == "subtitle" || k == "sources":
			case allowed == nil:
				dr.problem(sc, "infographic: unknown fence option %q (use title=, subtitle=, layout=, orientation=, aspect=, steps=, sources=)", k)
			default:
				if msg := oneOfMsg("infographic "+k, v, allowed); msg != "" {
					dr.problem(sc, "%s", msg)
				}
			}
		}
		if o.orient == "" && (o.aspect == "9:16" || o.aspect == "a4" || dr.doc) {
			o.orient = "portrait" // a document is a column, not a slide
		}
	}
	if o.orient == "" {
		o.orient = "landscape"
	}
	if o.layout == "" {
		o.layout = "grid"
	}
	if o.aspect == "" && o.orient == "landscape" {
		o.aspect = "16:9"
	}
	// a fence on a slide reveals a block per click; a page shows them all
	// (it is read as often as presented) unless it says steps: on
	stepsOff := attrs["steps"] == "off" || (page && m.Steps != "on")
	o.steps = sc != nil && !dr.doc && o.orient == "landscape" && !stepsOff

	ig, probs := deck.ParseInfographic(body)
	for _, p := range probs {
		pr := dr.at(sc, "%s", p.Msg)
		if page && sc != nil {
			pr.Line = sc.s.Line + p.Line - 1
		}
		if p.Warn {
			dr.warnings = append(dr.warnings, pr)
		} else {
			dr.problems = append(dr.problems, pr)
		}
	}
	if ig.Title != "" {
		o.title = ig.Title
	}
	if ig.Subtitle != "" {
		o.subtitle = ig.Subtitle
	}
	dr.infographic = true
	h := dr.igHTML(ig, o, sc)
	dr.cur = sc
	return dr.igRemote(h, sc)
}

// igHTML draws the frame: header, grid of blocks, footer.
func (dr *deckRenderer) igHTML(ig *deck.Infographic, o igOpts, sc *slideCtx) string {
	var b strings.Builder
	style := ""
	if r, ok := igRatios[o.aspect]; ok {
		style = fmt.Sprintf(` style="--_ratio:%g/%g;--_ar:%.4f"`, r[0], r[1], r[0]/r[1])
	}
	fmt.Fprintf(&b, `<figure class="si-ig" data-orient="%s" data-layout="%s"%s>`+"\n"+`<div class="si-ig-frame">`+"\n", o.orient, html.EscapeString(o.layout), style)
	if o.title != "" || o.subtitle != "" {
		b.WriteString(`<header class="si-ig-head">`)
		if o.title != "" {
			b.WriteString(`<h2 class="si-ig-title">` + dr.inline(o.title) + `</h2>`)
		}
		if o.subtitle != "" {
			b.WriteString(`<p class="si-ig-sub">` + dr.inline(o.subtitle) + `</p>`)
		}
		b.WriteString("</header>\n")
	}
	cols := 2
	if o.orient == "landscape" {
		cols = igCols(ig.Blocks, o.layout)
	}
	pos, rows, _ := igPlace(ig.Blocks, cols, o.layout)
	fmt.Fprintf(&b, `<div class="si-ig-grid" style="--_cols:%d;--_rows:%d">`+"\n", cols, rows)
	for i, bl := range ig.Blocks {
		cls := "si-ig-block"
		if bl.Accent {
			cls += " si-ig-accent"
		}
		step := ""
		if o.steps {
			cls += " si-step"
			step = fmt.Sprintf(` data-step="%d"`, sc.steps+i+1)
		}
		chart := ""
		if bl.Chart != "" {
			chart = ` data-chart="` + bl.Chart + `"`
		}
		p := pos[i]
		fmt.Fprintf(&b, `<section class="%s" data-kind="%s"%s%s style="--r:%d;--c:%d;--h:%d;--w:%d">`,
			cls, bl.Kind, chart, step, p[0], p[1], p[2], p[3])
		b.WriteString(dr.igBlock(bl))
		b.WriteString("</section>\n")
	}
	if o.steps {
		sc.steps += len(ig.Blocks)
	}
	b.WriteString("</div>\n")
	var src []string
	if o.sources != "" {
		src = append(src, dr.inline(o.sources))
	}
	if s := ig.Sources; s != nil {
		for _, it := range igItems(s.Body) {
			src = append(src, dr.inline(it))
		}
		if len(src) == 0 && s.Body != "" {
			src = append(src, dr.inline(s.Body))
		}
	}
	if len(src) > 0 || o.byline != "" {
		b.WriteString(`<footer class="si-ig-foot">`)
		if len(src) > 0 {
			label := "Sources"
			if s := ig.Sources; s != nil && s.Title != "" {
				label = s.Title
			}
			b.WriteString(`<p class="si-ig-src"><b>` + dr.inline(label) + `</b> ` + strings.Join(src, ` <span aria-hidden="true">·</span> `) + `</p>`)
		}
		if o.byline != "" {
			b.WriteString(`<p class="si-ig-by">` + html.EscapeString(o.byline) + `</p>`)
		}
		b.WriteString("</footer>\n")
	}
	b.WriteString("</div>\n</figure>\n")
	return b.String()
}

// igCols is the number of columns of a landscape grid.
func igCols(blocks []*deck.IGBlock, layout string) int {
	n := len(blocks)
	switch layout {
	case "flow":
		return 1
	case "columns":
		return max(1, min(4, (n+1)/2))
	}
	cells := 0
	for _, b := range blocks {
		cells += min(b.Span, 4) * b.Rows
	}
	if cells <= 3 {
		return max(cells, 1)
	}
	// the column count whose packing leaves the fewest holes and the best
	// shaped cells: a 16:9 frame less its header is about 16:7, and a cell
	// reads best a little wider than tall
	best, score := 3, math.Inf(1)
	for cols := 2; cols <= 5; cols++ {
		_, rows, holes := igPlace(blocks, cols, "grid")
		cell := 16.0 / float64(cols) / (7.0 / float64(rows))
		if s := float64(holes) + 2*math.Abs(math.Log(cell/1.3)); s < score {
			best, score = cols, s
		}
	}
	return best
}

// igPlace gives every block its row, column, height and width (1-based row
// and column) on a grid of cols columns: grid packs like CSS dense flow and
// then widens blocks into the holes left; columns fills column by column;
// flow is one block per row.
func igPlace(blocks []*deck.IGBlock, cols int, layout string) (pos [][4]int, rows, holes int) {
	pos = make([][4]int, len(blocks))
	if layout == "columns" || layout == "flow" {
		rows = max(1, (len(blocks)+cols-1)/cols)
		for i := range blocks {
			pos[i] = [4]int{i%rows + 1, i/rows + 1, 1, 1}
		}
		if n := len(blocks); n > 0 && n%rows != 0 { // the last column is short: its last block grows down
			pos[n-1][2] = rows - (n-1)%rows
		}
		return pos, rows, 0
	}
	var used [][]int // row -> col -> block index+1
	free := func(r, c, h, w int) bool {
		if c+w > cols {
			return false
		}
		for y := r; y < r+h; y++ {
			for y >= len(used) {
				used = append(used, make([]int, cols))
			}
			for x := c; x < c+w; x++ {
				if used[y][x] != 0 {
					return false
				}
			}
		}
		return true
	}
	for i, b := range blocks {
		w, h := min(b.Span, cols), b.Rows
		for r := 0; ; r++ {
			placed := false
			for c := 0; c < cols; c++ {
				if free(r, c, h, w) {
					for y := r; y < r+h; y++ {
						for x := c; x < c+w; x++ {
							used[y][x] = i + 1
						}
					}
					pos[i] = [4]int{r + 1, c + 1, h, w}
					placed = true
					break
				}
			}
			if placed {
				break
			}
		}
	}
	rows = len(used)
	for len(used) > 0 { // trailing empty rows (a probe looked ahead)
		empty := true
		for _, v := range used[len(used)-1] {
			empty = empty && v == 0
		}
		if !empty {
			break
		}
		used = used[:len(used)-1]
		rows--
	}
	for r := 0; r < rows; r++ { // holes: the block on the left widens, else the one above grows down
		for c := 0; c < cols; c++ {
			if used[r][c] != 0 {
				continue
			}
			holes++
			if c > 0 && used[r][c-1] != 0 && pos[used[r][c-1]-1][2] == 1 {
				k := used[r][c-1]
				pos[k-1][3]++
				used[r][c] = k
			} else if r > 0 && used[r-1][c] != 0 && pos[used[r-1][c]-1][3] == 1 {
				k := used[r-1][c]
				pos[k-1][2]++
				used[r][c] = k
			}
		}
	}
	return pos, max(rows, 1), holes
}

var (
	igImgRE  = regexp.MustCompile(`!\[([^\]]*)\]\(\s*<?([^)\s>]+)>?(?:\s+"([^"]*)")?\s*\)`)
	igItemRE = regexp.MustCompile(`^\s{0,3}(?:[-*+]|\d+[.)])\s+(.*)$`)
	igNumSRE = regexp.MustCompile(`^(.*?)([-+−]?(?:\d[\d,]*)?\.?\d+)(.*)$`)
)

// igItems are the items of the first list of a block (continuation lines
// joined to their item).
func igItems(body string) []string {
	var out []string
	inList := false
	for _, l := range strings.Split(body, "\n") {
		if m := igItemRE.FindStringSubmatch(l); m != nil {
			out = append(out, strings.TrimSpace(m[1]))
			inList = true
			continue
		}
		if t := strings.TrimSpace(l); t != "" && inList {
			out[len(out)-1] += " " + t // a continuation line
		}
	}
	return out
}

// igPicture pulls the first picture out of Markdown text: the <img> and the
// text without it.
func igPicture(s string) (img, rest string) {
	m := igImgRE.FindStringSubmatchIndex(s)
	if m == nil {
		return "", s
	}
	alt, src := s[m[2]:m[3]], s[m[4]:m[5]]
	img = fmt.Sprintf(`<img src="%s" alt="%s">`, html.EscapeString(src), html.EscapeString(alt))
	return img, strings.TrimSpace(s[:m[0]] + s[m[1]:])
}

// igBlock is the inside of one block.
func (dr *deckRenderer) igBlock(b *deck.IGBlock) string {
	label := ""
	if b.Title != "" {
		label = `<h3 class="si-ig-label">` + dr.inline(b.Title) + "</h3>"
	}
	switch b.Kind {
	case "stat":
		img, rest := igPicture(b.Body)
		out := ""
		if img != "" {
			out += `<span class="si-ig-icon">` + img + `</span>`
		}
		out += `<p class="si-ig-num">` + dr.igNumber(b.Title) + `</p>`
		if rest != "" {
			out += `<div class="si-ig-text">` + dr.toHTML(rest, nil) + `</div>`
		}
		return out
	case "compare":
		return label + dr.igCompare(b.Body)
	case "chart":
		return label + dr.igChart(b)
	case "steps":
		var s strings.Builder
		s.WriteString(label + `<ol class="si-ig-steps">`)
		for i, it := range igItems(b.Body) {
			fmt.Fprintf(&s, `<li><span class="si-ig-n">%d</span><span>%s</span></li>`, i+1, dr.inline(it))
		}
		return s.String() + "</ol>"
	case "icons", "list":
		var s strings.Builder
		s.WriteString(label + `<ul class="si-ig-` + b.Kind + `">`)
		for _, it := range igItems(b.Body) {
			img, rest := igPicture(it)
			if b.Kind == "icons" {
				if img == "" {
					img = `<span class="si-ig-dot"></span>`
				}
				s.WriteString(`<li><span class="si-ig-icon">` + img + `</span><span>` + dr.inline(rest) + `</span></li>`)
			} else {
				s.WriteString(`<li>` + dr.inline(it) + `</li>`)
			}
		}
		return s.String() + "</ul>"
	case "quote":
		var q, cite []string
		for _, l := range strings.Split(b.Body, "\n") {
			t := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(l), ">"))
			switch {
			case t == "":
			case strings.HasPrefix(t, "—") || strings.HasPrefix(t, "--") || strings.HasPrefix(t, "– "):
				cite = append(cite, strings.TrimSpace(strings.TrimLeft(t, "—–- ")))
			default:
				q = append(q, t)
			}
		}
		out := label + `<blockquote class="si-ig-quote"><p>` + dr.inline(strings.Join(q, " ")) + `</p>`
		if len(cite) > 0 {
			out += `<cite>` + dr.inline(strings.Join(cite, ", ")) + `</cite>`
		}
		return out + "</blockquote>"
	case "image":
		m := igImgRE.FindStringSubmatch(b.Body)
		if m == nil {
			return label + `<div class="si-ig-text">` + dr.toHTML(b.Body, nil) + `</div>`
		}
		cap := m[3]
		if cap == "" {
			cap = m[1]
		}
		out := `<figure class="si-ig-img"><img src="` + html.EscapeString(m[2]) + `" alt="` + html.EscapeString(m[1]) + `">`
		if cap != "" {
			out += `<figcaption>` + dr.inline(cap) + `</figcaption>`
		}
		return label + out + "</figure>"
	}
	return label + `<div class="si-ig-text">` + dr.toHTML(b.Body, nil) + `</div>`
}

// igNumber sets a stat's number big and its prefix and unit smaller:
// "$1.2B" -> $ 1.2 B, "2.4 s" -> 2.4 s.
func (dr *deckRenderer) igNumber(s string) string {
	m := igNumSRE.FindStringSubmatch(s)
	if m == nil || len([]rune(strings.TrimSpace(m[1]))) > 3 {
		return dr.inline(s)
	}
	out := ""
	if p := strings.TrimSpace(m[1]); p != "" {
		out += `<span class="si-ig-unit">` + html.EscapeString(p) + `</span>`
	}
	out += html.EscapeString(m[2])
	if u := strings.TrimSpace(m[3]); u != "" {
		out += `<span class="si-ig-unit">` + dr.inline(u) + `</span>`
	}
	return out
}

// igCompare draws a compare table: two or more sides side by side (the last
// one stressed), with the change between two numbers; a table of three or
// more columns has its row labels in the first.
func (dr *deckRenderer) igCompare(body string) string {
	head, rows, ok := deck.IGTable(body)
	if !ok {
		return `<div class="si-ig-text">` + dr.toHTML(body, nil) + `</div>`
	}
	labelled := len(head) >= 3
	delta := ""
	if !labelled && len(head) == 2 && len(rows) == 1 {
		a, ok1 := deck.IGNumber(rows[0][0])
		z, ok2 := deck.IGNumber(rows[0][1])
		if ok1 && ok2 && a != 0 {
			delta = `<p class="si-ig-delta">` + igPercent((z-a)/math.Abs(a)*100) + `</p>`
		}
	}
	var b strings.Builder
	first := 0
	if labelled {
		first = 1
	}
	long := 3 // the longest value sets the size of them all
	for _, r := range rows {
		for _, c := range r[first:] {
			long = max(long, len([]rune(c)))
		}
	}
	fmt.Fprintf(&b, `<div class="si-ig-cmp" style="--_n:%d;--_len:%d">`, len(head)-first, long)
	for j := first; j < len(head); j++ {
		cls := "si-ig-side"
		if j == len(head)-1 {
			cls += " si-ig-win"
		}
		b.WriteString(`<div class="` + cls + `"><p class="si-ig-label">` + dr.inline(head[j]) + `</p>`)
		for _, r := range rows {
			if labelled {
				b.WriteString(`<p class="si-ig-cv"><span class="si-ig-rl">` + dr.inline(r[0]) + `</span>` + dr.igNumber(r[j]) + `</p>`)
			} else {
				b.WriteString(`<p class="si-ig-cv">` + dr.igNumber(r[j]) + `</p>`)
			}
		}
		if j == len(head)-1 {
			b.WriteString(delta) // the change, under the side it led to
		}
		b.WriteString(`</div>`)
	}
	b.WriteString(`</div>`)
	return b.String()
}

// igPercent writes a change as "−75%" or "+12.5%" (a true minus sign).
func igPercent(p float64) string {
	sign := "+"
	if p < 0 {
		sign, p = "−", -p
	}
	s := strconv.FormatFloat(p, 'f', 0, 64)
	if p < 10 {
		s = strconv.FormatFloat(math.Round(p*10)/10, 'f', -1, 64)
	}
	return sign + s + "%"
}

// igSeries is a chart table: row labels, series names and values.
type igSeries struct {
	labels, names []string
	vals          [][]float64 // [row][series]
	text          [][]string  // the cells as written
	max           float64
}

func igData(body string) (igSeries, bool) {
	head, rows, ok := deck.IGTable(body)
	if !ok || len(head) < 2 || len(rows) == 0 {
		return igSeries{}, false
	}
	s := igSeries{names: head[1:]}
	for _, r := range rows {
		s.labels = append(s.labels, r[0])
		var v []float64
		for _, c := range r[1:] {
			n, _ := deck.IGNumber(c)
			v = append(v, n)
			s.max = math.Max(s.max, n)
		}
		s.vals = append(s.vals, v)
		s.text = append(s.text, r[1:])
	}
	return s, true
}

// IGScale is a value's share of the axis, in percent: 0 at zero, 100 at top
// (a bar or progress drawn to scale).
func IGScale(v, top float64) float64 {
	if top <= 0 || v <= 0 {
		return 0
	}
	return math.Min(100, v/top*100)
}

func pct(f float64) string { return strconv.FormatFloat(math.Round(f*100)/100, 'f', -1, 64) }

func (dr *deckRenderer) igChart(b *deck.IGBlock) string {
	s, ok := igData(b.Body)
	if !ok {
		return ""
	}
	series := len(s.names)
	var o strings.Builder
	legend := func(names []string) {
		o.WriteString(`<ul class="si-ig-legend">`)
		for i, n := range names {
			fmt.Fprintf(&o, `<li style="--_s:var(--_s%d)">%s</li>`, i%6+1, dr.inline(n))
		}
		o.WriteString(`</ul>`)
	}
	switch b.Chart {
	case "donut":
		total := 0.0
		for _, v := range s.vals {
			total += v[0]
		}
		o.WriteString(`<div class="si-ig-chart si-ig-donut">`)
		o.WriteString(`<svg viewBox="0 0 42 42" role="img" aria-label="` + html.EscapeString(plain(b.Title)) + `"><circle class="si-ig-track" cx="21" cy="21" r="15.9155" fill="none" stroke-width="6"/>`)
		at := 0.0
		for i, v := range s.vals {
			share := IGScale(v[0], total)
			fmt.Fprintf(&o, `<circle cx="21" cy="21" r="15.9155" fill="none" stroke-width="6" style="stroke:var(--_s%d)" stroke-dasharray="%s %s" stroke-dashoffset="%s"/>`,
				i%6+1, pct(share), pct(100-share), pct(25-at))
			at += share
		}
		if total > 0 {
			fmt.Fprintf(&o, `<text x="21" y="22.5" font-size="8.5" text-anchor="middle" class="si-ig-dv">%s</text><text x="21" y="28.5" font-size="3.4" text-anchor="middle" class="si-ig-dl">%s</text>`,
				igPercent(IGScale(s.vals[0][0], total))[1:], html.EscapeString(plain(dr.inline(s.labels[0]))))
		}
		o.WriteString(`</svg><ul class="si-ig-legend">`)
		for i, l := range s.labels {
			fmt.Fprintf(&o, `<li style="--_s:var(--_s%d)">%s <b>%s</b></li>`, i%6+1, dr.inline(l), html.EscapeString(s.text[i][0]))
		}
		o.WriteString(`</ul></div>`)
	case "line":
		top, lo := s.max, 0.0
		for _, v := range s.vals {
			for _, x := range v {
				lo = math.Min(lo, x)
			}
		}
		if top == lo {
			top = lo + 1
		}
		y := func(v float64) float64 { return 50 - (v-lo)/(top-lo)*50 }
		n := len(s.vals)
		x := func(i int) float64 {
			if n == 1 {
				return 50
			}
			return float64(i) / float64(n-1) * 100
		}
		num := func(f float64) string { return html.EscapeString(strconv.FormatFloat(f, 'f', -1, 64)) }
		o.WriteString(`<div class="si-ig-chart si-ig-line">`)
		fmt.Fprintf(&o, `<div class="si-ig-y" aria-hidden="true"><span>%s</span><span>%s</span></div><div class="si-ig-plot">`, num(top), num(lo))
		o.WriteString(`<svg viewBox="0 0 100 50" preserveAspectRatio="none" role="img" aria-label="` + html.EscapeString(plain(b.Title)) + `">`)
		o.WriteString(`<line class="si-ig-axis" vector-effect="non-scaling-stroke" x1="0" y1="50" x2="100" y2="50"/>`)
		colour := func(k int) string {
			if series == 1 {
				return "var(--_c)"
			}
			return "var(--_s" + strconv.Itoa(k%6+1) + ")"
		}
		for k := series - 1; k >= 0; k-- {
			var pts []string
			for i, v := range s.vals {
				pts = append(pts, pct(x(i))+","+pct(y(v[k])))
			}
			if k == 0 {
				fmt.Fprintf(&o, `<polygon class="si-ig-area" style="fill:color-mix(in srgb,%s 16%%,transparent)" points="0,50 %s 100,50"/>`, colour(0), strings.Join(pts, " "))
			}
			fmt.Fprintf(&o, `<polyline style="stroke:%s" vector-effect="non-scaling-stroke" points="%s"/>`, colour(k), strings.Join(pts, " "))
		}
		end := pct(y(s.vals[n-1][0]) * 2) // percent of the plot's height
		fmt.Fprintf(&o, `</svg><span class="si-ig-end" style="top:%s%%;background:%s"></span><b class="si-ig-last" style="top:%s%%">%s</b></div><div class="si-ig-x">`,
			end, colour(0), end, html.EscapeString(s.text[n-1][0]))
		for _, l := range s.labels {
			o.WriteString(`<span>` + dr.inline(l) + `</span>`)
		}
		o.WriteString(`</div></div>`)
		if series > 1 {
			legend(s.names)
		}
	case "progress":
		o.WriteString(`<div class="si-ig-chart si-ig-bars si-ig-progress">`)
		for i, v := range s.vals {
			top := 100.0
			if series > 1 {
				top = v[1] // | Goal | Done | Target |
			} else if s.max > 100 {
				top = s.max
			}
			fmt.Fprintf(&o, `<div class="si-ig-bar"><span class="si-ig-bl">%s</span><svg viewBox="0 0 100 4" preserveAspectRatio="none" aria-hidden="true"><rect class="si-ig-track" width="100" height="4" rx="2"/><rect width="%s" height="4" rx="2" style="fill:var(--_c)"/></svg><b class="si-ig-bv">%s</b></div>`,
				dr.inline(s.labels[i]), pct(IGScale(v[0], top)), html.EscapeString(s.text[i][0]))
		}
		o.WriteString(`</div>`)
	default: // bar
		o.WriteString(`<div class="si-ig-chart si-ig-bars">`)
		h := 4 * series
		for i, v := range s.vals {
			fmt.Fprintf(&o, `<div class="si-ig-bar" style="--_k:%d"><span class="si-ig-bl">%s</span><svg viewBox="0 0 100 %d" preserveAspectRatio="none" aria-hidden="true">`,
				series, dr.inline(s.labels[i]), h)
			for k, x := range v {
				fill := "var(--_s" + strconv.Itoa(k%6+1) + ")"
				if series == 1 {
					fill = "var(--_cm)" // one series: the top bar stands out
					if x == s.max {
						fill = "var(--_c)"
					}
				}
				fmt.Fprintf(&o, `<rect y="%d" width="%s" height="3.2" rx="1" style="fill:%s"/>`, k*4, pct(IGScale(x, s.max)), fill)
			}
			o.WriteString(`</svg><b class="si-ig-bv">` + html.EscapeString(strings.Join(s.text[i], " · ")) + `</b></div>`)
		}
		o.WriteString(`</div>`)
		if series > 1 {
			legend(s.names)
		}
	}
	return o.String()
}

var igRemoteRE = regexp.MustCompile(`(<img\b[^>]*?\ssrc=")(https://[^"]+)"`)

// igRemote fetches https pictures and inlines them, as fonts are (cached,
// --offline works after one online build); a picture that cannot be fetched
// stays a link, with a warning.
func (dr *deckRenderer) igRemote(h string, sc *slideCtx) string {
	return igRemoteRE.ReplaceAllStringFunc(h, func(s string) string {
		m := igRemoteRE.FindStringSubmatch(s)
		u := html.UnescapeString(m[2])
		raw, err := deck.FetchAsset(u, nil)
		if err != nil {
			dr.warn(sc, "infographic: picture %s: %v (left as a link)", u, err)
			return s
		}
		mime := http.DetectContentType(raw)
		if t := bytes.TrimSpace(raw); bytes.HasPrefix(t, []byte("<svg")) || (bytes.HasPrefix(t, []byte("<?xml")) && bytes.Contains(raw[:min(len(raw), 512)], []byte("<svg"))) {
			mime = "image/svg+xml"
		}
		if !strings.HasPrefix(mime, "image/") {
			dr.warn(sc, "infographic: %s is not a picture (%s)", u, mime)
			return s
		}
		return m[1] + "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(raw) + `"`
	})
}

// Infographic renders a portrait `type: infographic` file: a page that
// scrolls, the poster centred in it. (A landscape one is a deck slide.)
func Infographic(d *deck.Deck, opt Options) Result {
	dr := &deckRenderer{d: d, opt: opt, doc: true, lastState: map[string][]string{}, words: map[string]bool{}, files: map[string]string{}}
	dr.md = dr.markdown()
	dr.resolveTheme()
	var body strings.Builder
	for _, s := range d.Slides {
		sc := &slideCtx{s: s}
		body.WriteString(dr.postprocess(dr.toHTML(s.Body, sc), s.Dir, sc))
	}
	res := Result{}
	dr.finish(&res)
	var b bytes.Buffer
	modeAttr := dr.head(&b)
	b.WriteString(`<style id="si-css">` + "\n" + assets.LayerOrder + assets.DocCSS + dr.themeCSS + dr.accentCSS() + assets.InfographicCSS + "</style>\n")
	dr.userAndMode(&b, modeAttr)
	b.WriteString("</head>\n<body class=\"si-doc si-ig-page\" data-si-doc=\"infographic\">\n<main>\n")
	b.WriteString(body.String())
	b.WriteString("</main>\n")
	dr.problemsBanner(&b, res.Problems)
	dr.reload(&b)
	b.WriteString("</body>\n</html>\n")
	res.HTML, res.body = b.Bytes(), body.String()
	return res
}
