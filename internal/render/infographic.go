package render

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"html"
	"math"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

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
	animate                                                  string  // build | loop | "" (off)
	duration                                                 float64 // duration: (0 = as long as the blocks need)
	page                                                     bool    // a type: infographic file, not a fence
}

var igRatios = map[string][2]float64{"16:9": {16, 9}, "4:3": {4, 3}, "9:16": {9, 16}, "a4": {210, 297}}

// igRatio is the width:height of an aspect ({0, 0} when it has none).
func igRatio(aspect string) [2]float64 {
	if w, h, ok := deck.IGSize(aspect); ok {
		return [2]float64{w, h}
	}
	return igRatios[aspect]
}

// igFixed reports whether an aspect is an exact frame (phone, square,
// 1080x1350): the frame keeps its shape and its blocks share it.
func igFixed(aspect string) bool { _, _, ok := deck.IGSize(aspect); return ok }

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
			byline: strings.Trim(m.Author+" · "+m.Date, " ·"), animate: m.Animate, page: true}
		o.duration, _ = deck.IGSeconds(m.Duration)
		if m.IGPortrait() {
			o.orient = "portrait"
		}
	} else {
		o.animate = attrs["animate"]
		for k, v := range attrs {
			allowed := map[string][]string{"layout": deck.IGLayouts, "orientation": deck.IGOrientations, "aspect": deck.IGAspects, "steps": deck.OnOff, "animate": deck.IGAnimates}[k]
			switch {
			case k == "title" || k == "subtitle" || k == "sources":
			case k == "duration":
				var ok bool
				if o.duration, ok = deck.IGSeconds(v); !ok || o.duration <= 0 {
					dr.problem(sc, "infographic: duration=%s: write seconds (duration=8s)", v)
				}
			case k == "aspect" && igFixed(v):
			case allowed == nil:
				dr.problem(sc, "infographic: unknown fence option %q (use title=, subtitle=, layout=, orientation=, aspect=, steps=, sources=, animate=, duration=)", k)
			default:
				if msg := oneOfMsg("infographic "+k, v, allowed); msg != "" {
					dr.problem(sc, "%s", msg)
				}
			}
		}
		if w, h, ok := deck.IGSize(o.aspect); o.orient == "" && ok {
			o.orient = map[bool]string{true: "portrait", false: "landscape"}[w < h]
		}
		if o.orient == "" && (o.aspect == "9:16" || o.aspect == "a4" || dr.doc) {
			o.orient = "portrait" // a document is a column, not a slide
		}
	}
	if o.animate == "off" {
		o.animate = ""
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
	style, r := "", igRatio(o.aspect)
	if r[0] > 0 {
		style = fmt.Sprintf(` style="--_ratio:%g/%g;--_ar:%.4f"`, r[0], r[1], r[0]/r[1])
	}
	fixed := igFixed(o.aspect)
	tall := fixed && r[0]/r[1] < 0.7 // a phone or a story: one column, flows top to bottom
	attrs := ""
	if fixed { // tall (a phone, a story: one column), portrait (two) or landscape (a slide)
		attrs += ` data-fixed="` + map[bool]string{true: "tall", false: o.orient}[tall] + `"`
	}
	var cues []igCue
	if o.animate != "" {
		var end, total float64
		cues, end, total = igSchedule(ig.Blocks, o.animate == "loop", o.duration)
		if o.duration > 0 && end > o.duration+0.05 {
			dr.warnings = append(dr.warnings, dr.at(sc, "infographic: the blocks need %.1fs to come in, longer than duration: %gs (they are cut short; raise duration or move at= earlier)", end, o.duration))
		}
		if o.page {
			px := igGIFSize(o.aspect)
			if est := IGGIFEstimate(px[0], px[1], total, 15); est > igGIFBudget {
				dr.warnings = append(dr.warnings, dr.at(sc, "infographic: an animated GIF of it would be about %.1f MB at %d px wide, 15 fps (over 8 MB: shorten duration:, or build it with a lower --fps or --scale)", float64(est)/1e6, px[0]))
			}
		}
		attrs += fmt.Sprintf(` data-animate="%s" data-duration="%s"`, o.animate, igSec(total))
	}
	fmt.Fprintf(&b, `<figure class="si-ig" data-orient="%s" data-layout="%s"%s%s>`+"\n"+`<div class="si-ig-frame">`+"\n", o.orient, html.EscapeString(o.layout), attrs, style)
	if o.title != "" || o.subtitle != "" {
		if o.animate != "" {
			b.WriteString(`<header class="si-ig-head" data-enter="fade" data-at="0" data-dur="0.6">`)
		} else {
			b.WriteString(`<header class="si-ig-head">`)
		}
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
	place := ig.Blocks
	if tall { // a phone or a story: two columns where stats pair up, every other block across
		place = make([]*deck.IGBlock, len(ig.Blocks))
		for i, bl := range ig.Blocks {
			c := *bl
			c.Span = 2
			if bl.Kind == "stat" && bl.Span == 1 {
				c.Span = 1
			}
			place[i] = &c
		}
	}
	pos, rows, _ := igPlace(place, cols, o.layout)
	if o.orient == "landscape" {
		for i, bl := range ig.Blocks {
			if f := igFit(bl, pos[i], cols, rows, r[0]/r[1], o.title != "", o.subtitle != ""); f < igFitFloor {
				dr.warnings = append(dr.warnings, dr.at(sc, "infographic: block %q holds more text than its cell, even at %d%% text size (it would need about %d%%): shorten it, make it wide or tall, or move a block out", bl.Title, int(igFitFloor*100), int(f*100)))
			}
		}
	}
	grid := ""
	if fixed && o.orient == "portrait" { // a fixed tall frame: rows shared by what they hold
		grid = ";grid-template-rows:" + igRowWeights(ig.Blocks, pos, rows, tall)
	}
	dir := "LR"
	if tall {
		dir = "TB"
	}
	fmt.Fprintf(&b, `<div class="si-ig-grid" style="--_cols:%d;--_rows:%d%s">`+"\n", cols, rows, grid)
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
		if cues != nil {
			step += fmt.Sprintf(` data-enter="%s" data-at="%s" data-dur="%s"`, cues[i].Enter, igSec(cues[i].At), igSec(cues[i].Dur))
		}
		p := pos[i]
		fmt.Fprintf(&b, `<section class="%s" data-kind="%s"%s%s style="--r:%d;--c:%d;--h:%d;--w:%d">`,
			cls, bl.Kind, chart, step, p[0], p[1], p[2], p[3])
		if bl.Kind == "flow" {
			b.WriteString(dr.igFlow(bl, map[string]string{"lr": "LR", "tb": "TB", "": dir}[bl.Dir]))
		} else {
			b.WriteString(dr.igBlock(bl))
		}
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
	out += `<span class="si-ig-cnt">` + html.EscapeString(m[2]) + `</span>`
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
	max, min      float64     // min: 0 or below
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
			s.min = math.Min(s.min, n)
		}
		s.vals = append(s.vals, v)
		s.text = append(s.text, r[1:])
	}
	return s, true
}

// igXNumbers reads a line chart's x labels as numbers (years too) or, failing
// that, as dates; ok is false when any label is neither or all are equal.
func igXNumbers(labels []string) (v []float64, dates, ok bool) {
	num := func(l string) (float64, bool) {
		f, err := strconv.ParseFloat(strings.TrimSpace(l), 64)
		return f, err == nil
	}
	date := func(l string) (float64, bool) {
		for _, f := range []string{"2006-01-02", "2006-01", "Jan 2006", "January 2006", "2 Jan 2006", "Jan 2, 2006"} {
			if t, err := time.Parse(f, strings.TrimSpace(l)); err == nil {
				return float64(t.Unix()) / 86400, true
			}
		}
		return 0, false
	}
	for _, read := range []func(string) (float64, bool){num, date} {
		v = v[:0]
		for _, l := range labels {
			f, good := read(l)
			if !good {
				break
			}
			v = append(v, f)
		}
		if len(v) == len(labels) && len(v) > 1 && slices.Min(v) < slices.Max(v) {
			return v, dates, true
		}
		dates = true
	}
	return nil, false, false
}

// igXValues places each row of a line chart along x, in percent, by its
// value when every x label is a number or a date (a linear scale from the
// smallest to the largest); nil keeps the rows evenly spaced.
func igXValues(labels []string) (x []float64, dates bool) {
	v, dates, ok := igXNumbers(labels)
	if !ok {
		return nil, false
	}
	lo, hi := slices.Min(v), slices.Max(v)
	for _, f := range v {
		x = append(x, (f-lo)/(hi-lo)*100)
	}
	return x, dates
}

// igTicks are the nice x ticks of a numeric line chart (steps of 1, 2 or 5
// times a power of ten, whole numbers when the labels are), as
// {percent along x, value} pairs.
func igTicks(labels []string) [][2]float64 {
	v, _, _ := igXNumbers(labels)
	lo, hi := slices.Min(v), slices.Max(v)
	raw := (hi - lo) / 5
	mag := math.Pow(10, math.Floor(math.Log10(raw)))
	step := 10 * mag
	for _, m := range []float64{1, 2, 5} {
		if raw <= m*mag {
			step = m * mag
			break
		}
	}
	whole := true
	for _, f := range v {
		whole = whole && f == math.Trunc(f)
	}
	if whole {
		step = math.Max(1, math.Round(step))
	}
	var t [][2]float64
	for i := math.Ceil(lo / step); i*step <= hi+step*1e-9; i++ {
		f := i * step
		t = append(t, [2]float64{(f - lo) / (hi - lo) * 100, f})
	}
	return t
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
	if strings.HasSuffix(b.Chart, "3d") { // the 3D charts of a ```chart fence, at rest
		co := chartOpts{ownHue: true}
		for i := range s.labels {
			co.keys = append(co.keys, i)
		}
		return dr.draw3D(&deck.Chart{Type: b.Chart}, b.Title, s, co)
	}
	return dr.drawChart(b.Chart, b.Title, s, chartOpts{})
}

// chartOpts are what a ```chart fence adds to an infographic chart: rows
// named by data-key (the animation matches them across steps), a colour per
// row that stays with it when the rows re-sort, stacked series.
type chartOpts struct {
	keys    []int // the table row each drawn row is (nil: none written)
	ownHue  bool  // each row keeps its own colour (sorted rows)
	stacked bool
	explode *int // a donut's slice that can lift out: nil none, -1 not now, else the table row lifted
}

// lift is a donut slice's transform when explode= is set: moved off the
// centre along the slice's middle (by d, in the drawing's units) when it is
// the slice lifted out, else in place (written, so the step can tween it).
func (o chartOpts) lift(row int, at, share, d float64) string {
	if o.explode == nil {
		return ""
	}
	x, y := 0.0, 0.0
	if *o.explode == row {
		a := (at + share/2) / 100 * 2 * math.Pi // clockwise from the top
		x, y = math.Sin(a)*d, -math.Cos(a)*d
	}
	return ` transform="translate(` + pct(x) + ` ` + pct(y) + `)"`
}

func (o chartOpts) key(i int) string {
	if o.keys == nil {
		return ""
	}
	return ` data-key="` + strconv.Itoa(o.keys[i]) + `"`
}

// drawChart draws a chart (bar, column, line, area, donut, pie, progress)
// of a table: infographic blocks and ```chart fences.
func (dr *deckRenderer) drawChart(kind, title string, s igSeries, co chartOpts) string {
	series := len(s.names)
	var o strings.Builder
	legend := func(names []string) {
		o.WriteString(`<ul class="si-ig-legend">`)
		for i, n := range names {
			fmt.Fprintf(&o, `<li style="--_s:var(--_s%d)">%s</li>`, i%6+1, dr.inline(n))
		}
		o.WriteString(`</ul>`)
	}
	hue := func(i int) string { // a row's own colour
		if co.keys != nil {
			i = co.keys[i]
		}
		return "var(--_s" + strconv.Itoa(i%6+1) + ")"
	}
	switch kind {
	case "donut", "pie":
		total := 0.0
		for _, v := range s.vals {
			total += v[0]
		}
		r, w, extra := "15.9155", "6", ""
		if kind == "pie" { // a circle its stroke fills: pathLength keeps the dashes in percent
			r, w, extra = "10.5", "21", ` pathLength="100"`
		}
		o.WriteString(`<div class="si-ig-chart si-ig-donut">`)
		o.WriteString(`<svg viewBox="0 0 42 42" role="img" aria-label="` + html.EscapeString(plain(title)) + `"><circle class="si-ig-track" cx="21" cy="21" r="` + r + `" fill="none" stroke-width="` + w + `"` + extra + `/>`)
		at := 0.0
		for i, v := range s.vals {
			share := IGScale(v[0], total)
			row := i
			if co.keys != nil {
				row = co.keys[i]
			}
			fmt.Fprintf(&o, `<circle%s cx="21" cy="21" r="%s" fill="none" stroke-width="%s"%s style="stroke:%s" stroke-dasharray="%s %s" stroke-dashoffset="%s"%s/>`,
				co.key(i), r, w, extra, hue(i), pct(share), pct(100-share), pct(25-at), co.lift(row, at, share, 2.5))
			at += share
		}
		if total > 0 && kind == "donut" {
			fmt.Fprintf(&o, `<text x="21" y="22.5" font-size="8.5" text-anchor="middle" class="si-ig-dv">%s</text><text x="21" y="28.5" font-size="3.4" text-anchor="middle" class="si-ig-dl">%s</text>`,
				igPercent(IGScale(s.vals[0][0], total))[1:], html.EscapeString(plain(dr.inline(s.labels[0]))))
		}
		o.WriteString(`</svg><ul class="si-ig-legend">`)
		for i, l := range s.labels {
			fmt.Fprintf(&o, `<li%s style="--_s:%s">%s <b>%s</b></li>`, co.key(i), hue(i), dr.inline(l), html.EscapeString(s.text[i][0]))
		}
		o.WriteString(`</ul></div>`)
	case "line", "area":
		top, lo := s.max, s.min
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
		xv, dates := igXValues(s.labels) // numbers or dates: placed by value
		if xv != nil {
			x = func(i int) float64 { return xv[i] }
		}
		num := func(f float64) string { return html.EscapeString(strconv.FormatFloat(f, 'f', -1, 64)) }
		o.WriteString(`<div class="si-ig-chart si-ig-line">`)
		fmt.Fprintf(&o, `<div class="si-ig-y" aria-hidden="true"><span>%s</span><span>%s</span></div><div class="si-ig-plot">`, num(top), num(lo))
		o.WriteString(`<svg viewBox="0 0 100 50" preserveAspectRatio="none" role="img" aria-label="` + html.EscapeString(plain(title)) + `">`)
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
			if k == 0 || kind == "area" {
				share := 16
				if kind == "area" {
					share = 30
				}
				fmt.Fprintf(&o, `<polygon class="si-ig-area" style="fill:color-mix(in srgb,%s %d%%,transparent)" points="0,50 %s 100,50"/>`, colour(k), share, strings.Join(pts, " "))
			}
			fmt.Fprintf(&o, `<polyline style="stroke:%s" vector-effect="non-scaling-stroke" points="%s"/>`, colour(k), strings.Join(pts, " "))
		}
		end := pct(y(s.vals[n-1][0]) * 2) // percent of the plot's height
		xcls := "si-ig-x"
		if xv != nil {
			xcls += " si-ig-xv"
		}
		fmt.Fprintf(&o, `</svg><span class="si-ig-end" style="top:%s%%;background:%s"></span><b class="si-ig-last" style="top:%s%%">%s</b></div><div class="%s">`,
			end, colour(0), end, html.EscapeString(s.text[n-1][0]), xcls)
		switch {
		case xv == nil:
			for _, l := range s.labels {
				o.WriteString(`<span>` + dr.inline(l) + `</span>`)
			}
		case dates: // a date tick is the row's own label, at its date
			// ponytail: a date label is wide, so one closer than a fifth of the
			// axis to the last one kept (or to the end) is left out; measure if
			// that ever leaves too few.
			kept := -100.0
			for i, l := range s.labels {
				if i > 0 && i < n-1 && (xv[i]-kept < 22 || 100-xv[i] < 22) {
					continue
				}
				kept = xv[i]
				fmt.Fprintf(&o, `<span style="left:%s%%;translate:-%s%% 0">%s</span>`, pct(xv[i]), pct(xv[i]), dr.inline(l))
			}
		default: // numbers: ticks at nice values
			for _, t := range igTicks(s.labels) {
				fmt.Fprintf(&o, `<span style="left:%s%%;translate:-%s%% 0">%s</span>`, pct(t[0]), pct(t[0]), html.EscapeString(strconv.FormatFloat(t[1], 'f', -1, 64)))
			}
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
			fmt.Fprintf(&o, `<div class="si-ig-bar"%s><span class="si-ig-bl">%s</span><svg viewBox="0 0 100 4" preserveAspectRatio="none" aria-hidden="true"><rect class="si-ig-track" width="100" height="4" rx="2"/><rect width="%s" height="4" rx="2" style="fill:var(--_c)"/></svg><b class="si-ig-bv">%s</b></div>`,
				co.key(i), dr.inline(s.labels[i]), pct(IGScale(v[0], top)), html.EscapeString(s.text[i][0]))
		}
		o.WriteString(`</div>`)
	default: // bar, column
		top := s.max
		if co.stacked {
			top = 0
			for _, v := range s.vals {
				sum := 0.0
				for _, x := range v {
					sum += math.Max(0, x)
				}
				top = math.Max(top, sum)
			}
		}
		fill := func(i, k int, x float64) string {
			switch {
			case series > 1:
				return "var(--_s" + strconv.Itoa(k%6+1) + ")"
			case co.ownHue:
				return hue(i)
			case x == s.max:
				return "var(--_c)" // one series: the top bar stands out
			}
			return "var(--_cm)"
		}
		lanes := series // grouped: a lane per series; stacked: one
		if co.stacked {
			lanes = 1
		}
		col := kind == "column"
		if col {
			fmt.Fprintf(&o, `<div class="si-ig-chart si-ig-cols" style="--_n:%d">`, len(s.vals))
		} else {
			o.WriteString(`<div class="si-ig-chart si-ig-bars">`)
		}
		for i, v := range s.vals {
			value := `<b class="si-ig-bv">` + html.EscapeString(strings.Join(s.text[i], " · ")) + `</b>`
			label := `<span class="si-ig-bl">` + dr.inline(s.labels[i]) + `</span>`
			if col {
				fmt.Fprintf(&o, `<div class="si-ig-col"%s>%s<svg viewBox="0 0 %d 100" preserveAspectRatio="none" aria-hidden="true">`, co.key(i), value, 4*lanes)
			} else {
				fmt.Fprintf(&o, `<div class="si-ig-bar" style="--_k:%d"%s>%s<svg viewBox="0 0 100 %d" preserveAspectRatio="none" aria-hidden="true">`,
					lanes, co.key(i), label, 4*lanes)
			}
			at := 0.0
			for k, x := range v {
				size := IGScale(x, top)
				lane := k
				if co.stacked {
					lane = 0
				}
				if col {
					fmt.Fprintf(&o, `<rect x="%d" y="%s" width="3.2" height="%s" rx="1" style="fill:%s"/>`, lane*4, pct(100-at-size), pct(size), fill(i, k, x))
				} else if co.stacked {
					fmt.Fprintf(&o, `<rect x="%s" y="0" width="%s" height="3.2" rx="1" style="fill:%s"/>`, pct(at), pct(size), fill(i, k, x))
				} else {
					fmt.Fprintf(&o, `<rect y="%d" width="%s" height="3.2" rx="1" style="fill:%s"/>`, lane*4, pct(size), fill(i, k, x))
				}
				if co.stacked {
					at += size
				}
			}
			o.WriteString(`</svg>`)
			if col {
				o.WriteString(label + `</div>`)
			} else {
				o.WriteString(value + `</div>`)
			}
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
	dr.timelineJS(&b) // the fit pass and the motion
	dr.reload(&b)
	b.WriteString("</body>\n</html>\n")
	res.HTML, res.body = b.Bytes(), body.String()
	return res
}

// igFitFloor is the smallest text size (of the block's own) infographic.js
// lowers a block to.
const igFitFloor = 0.6

// igFit estimates the text size (1 = as designed) a landscape block needs so
// its text fits its cell: the frame is 100/1.12 em wide (font-size 1.12cqi),
// characters about 0.46 em wide, lines 1.35 em. Only text kinds are estimated
// (charts, pictures and comparisons scale with their cell): 1 for the rest.
// ponytail: a character-count model, not a layout; the browser's fit pass
// (infographic.js) is the truth, this only warns in si check before it.
func igFit(b *deck.IGBlock, p [4]int, cols, rows int, ratio float64, title, sub bool) float64 {
	switch b.Kind {
	case "text", "list", "steps", "quote", "stat", "icons":
	default:
		return 1
	}
	const gap = 0.85
	w := 100/1.12 - 6
	h := (100/1.12)/ratio - 4 - 2*1.2 - 2 // padding, gaps, footer
	if title {
		h -= 2.7
	}
	if sub {
		h -= 1.6
	}
	cw, ch := (w-gap*float64(cols-1))/float64(cols), (h-gap*float64(rows-1))/float64(rows)
	bw := cw*float64(p[3]) + gap*float64(p[3]-1) - 2.4
	bh := ch*float64(p[2]) + gap*float64(p[2]-1) - 2.1
	var paras []string
	for _, l := range strings.Split(b.Body, "\n") {
		l = strings.TrimSpace(l)
		if l == "" || strings.HasPrefix(l, "|") || strings.HasPrefix(l, "![") {
			continue
		}
		l = strings.TrimLeft(l, "-*+>0123456789.) ")
		paras = append(paras, strings.NewReplacer("**", "", "*", "", "`", "").Replace(l))
	}
	need := func(f float64) float64 {
		height := 0.0
		if b.Kind == "stat" {
			height += 3.2 * f // the number
		} else if b.Title != "" {
			height += 1.8 * f // the label and its rule
		}
		perLine := math.Max(1, bw/(0.46*f))
		for _, t := range paras {
			lines := math.Ceil(float64(len([]rune(t))) / perLine)
			height += lines*1.35*f + 0.5*f
		}
		return height
	}
	for f := 1.0; f >= 0.2; f -= 0.05 {
		if need(f) <= bh {
			return f
		}
	}
	return 0.2
}
