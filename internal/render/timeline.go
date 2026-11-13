package render

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"html"
	"math"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/assets"
	"github.com/felipeg48/simple-ideas/internal/deck"
)

// Timelines: a `type: timeline` page (its body arrives as a ```timeline
// si-page fence, deck/timeline.go) and the ```timeline fence. The snake and
// line layouts place every milestone in % of the track (the line is an SVG
// that stretches with it, strokes in cqi); vertical is plain flow, and it is
// what a phone gets for every layout.

type tlOpts struct {
	layout, orient, scale, title, subtitle, closing, lang string
	alternate, steps                                      bool
	ratio                                                 float64 // width / height of the landscape frame
	page                                                  bool
}

// tlFenceKeys are the ```timeline fence options.
var tlFenceKeys = map[string][]string{"layout": deck.TimelineLayouts, "alternate": deck.OnOff, "scale": deck.TimelineScales,
	"aspect": deck.Aspects, "orientation": deck.TimelineOrientations, "steps": nil, "title": nil, "subtitle": nil, "closing": nil}

func (dr *deckRenderer) timelineFence(body string, attrs map[string]string, dir string) string {
	sc := dr.cur // dr.inline (titles, descriptions) clears dr.cur: keep the slide
	defer func() { dr.cur = sc }()
	_, page := attrs["si-page"]
	page = page && dr.d.IsTimeline()
	o := tlOpts{alternate: true, scale: "even", page: page, ratio: 16.0 / 9}
	aspect := ""
	if page {
		m := dr.d.Meta
		o.layout, o.orient, o.scale, o.title, o.subtitle, o.closing, o.lang = m.MapLayout, m.Orientation, m.Timeline.Scale, m.Title, m.Subtitle, m.Timeline.Closing, m.Lang
		o.alternate = m.Timeline.Alternate != "off"
		o.steps = m.Steps != "off"
		aspect = m.Aspect
	} else {
		for k, v := range attrs {
			allowed, known := tlFenceKeys[k]
			if !known {
				dr.problem(dr.cur, "timeline: unknown fence option %q (use layout=, alternate=, scale=, aspect=, orientation=, title=, subtitle=, closing=, steps)", k)
			} else if allowed != nil {
				if msg := oneOfMsg("timeline "+k, v, allowed); msg != "" {
					dr.problem(dr.cur, "%s", msg)
				}
			}
		}
		o.layout, o.orient, o.title, o.subtitle, o.closing = attrs["layout"], attrs["orientation"], attrs["title"], attrs["subtitle"], attrs["closing"]
		o.alternate = attrs["alternate"] != "off"
		_, o.steps = attrs["steps"]
		if attrs["scale"] != "" {
			o.scale = attrs["scale"]
		}
		aspect = attrs["aspect"]
	}
	if o.orient == "" {
		o.orient = "landscape"
	}
	if o.layout == "" {
		o.layout = "snake"
		if o.orient == "portrait" || (dr.doc && !page) {
			o.layout = "vertical"
		}
	}
	if a, b, ok := strings.Cut(aspect, ":"); ok {
		x, _ := strconv.ParseFloat(a, 64)
		y, _ := strconv.ParseFloat(b, 64)
		if x > 0 && y > 0 {
			o.ratio = x / y
		}
	}
	line := 0
	if dr.cur != nil {
		line = dr.cur.s.Line
	}
	t, probs, warns := deck.ParseTimeline(body, dr.d.Path, line)
	if !page { // a page's own problems come from the parser, with their lines
		dr.problems = append(dr.problems, probs...)
		dr.warnings = append(dr.warnings, warns...)
		cp, cw := t.Check(dr.d.Path, dir)
		dr.problems = append(dr.problems, cp...)
		dr.warnings = append(dr.warnings, cw...)
	}
	if o.title == "" {
		o.title = t.Title
	}
	if page && dr.doc {
		o.title = "" // a poster page has the document's own title above it
	}
	ms := t.Milestones()
	if len(ms) == 0 {
		dr.problem(dr.cur, "timeline: no milestones (write ### date · title lines under ## era headings)")
		return ""
	}
	if sc == nil || dr.doc {
		o.steps = false // build steps belong to slides
	}
	dr.timeline = true
	return dr.timelineHTML(t, o, dir, sc)
}

// tlPlace is where a milestone sits on the track, in % of its width and height.
type tlPlace struct {
	x, y, w float64
	row     int
	f       float64 // position along its row, 0..1
	above   bool
	level   int
}

// tlGeometry places the milestones (snake and line) and returns the track
// rows and each milestone's place.
func tlGeometry(t *deck.Timeline, o tlOpts) (rows int, counts []int, places []tlPlace) {
	ms := t.Milestones()
	n := len(ms)
	rows = 1
	if o.layout == "snake" {
		per := 10 * o.ratio / (16.0 / 9)
		rows = int(math.Ceil(float64(n) / per))
		if rows < 1 {
			rows = 1
		}
	}
	counts = make([]int, rows)
	for r := range counts {
		counts[r] = n / rows
		if r < n%rows {
			counts[r]++
		}
	}
	const m = 3.2 // side margin of the track (room for the snake's turns)
	levels := 1
	if o.layout == "line" && n > 12 {
		levels = 2
	}
	i := 0
	for r, c := range counts {
		fs := make([]float64, c)
		for k := range fs {
			fs[k] = (float64(k) + 0.5) / float64(c)
		}
		if o.scale == "time" && c > 1 {
			fs = tlTimeScale(ms[i:i+c], c)
		}
		for k := 0; k < c; k++ {
			p := tlPlace{row: r, f: fs[k], y: (float64(r) + 0.5) / float64(rows) * 100}
			p.x = m + p.f*(100-2*m)
			if r%2 == 1 {
				p.x = 100 - p.x
			}
			p.above = !o.alternate || i%2 == 0
			span := 1.0
			if o.alternate {
				span = 2
			}
			if levels == 2 {
				p.level = (i / 2) % 2
				if !o.alternate {
					p.level = i % 2
				}
				span *= 2
			}
			p.w = span * (100 - 2*m) / float64(c) * 0.94
			if o.scale == "time" {
				p.w *= 0.85
			}
			places = append(places, p)
			i++
		}
	}
	return rows, counts, places
}

// tlTimeScale spaces a row's milestones by real time: each gap follows its
// years, held between 0.6 and 1.8 of the mean so cards still fit; longer gaps
// are broken (the line shows a break mark).
func tlTimeScale(ms []*deck.Milestone, c int) []float64 {
	gaps := make([]float64, c-1)
	var dated []float64
	for k := 1; k < c; k++ {
		if ms[k].Dated && ms[k-1].Dated {
			gaps[k-1] = math.Max(ms[k].Year-ms[k-1].Year, 0)
			dated = append(dated, gaps[k-1])
		} else {
			gaps[k-1] = -1
		}
	}
	if len(dated) == 0 {
		out := make([]float64, c)
		for k := range out {
			out[k] = (float64(k) + 0.5) / float64(c)
		}
		return out
	}
	sort.Float64s(dated)
	med := dated[len(dated)/2]
	if med <= 0 {
		med = 1
	}
	w := make([]float64, c-1)
	total := 0.0
	for k, g := range gaps {
		if g < 0 {
			w[k] = 1
		} else {
			w[k] = math.Min(math.Max(g/med, 0.6), 1.8)
		}
		total += w[k]
	}
	// half a mean gap before the first and after the last
	mean := total / float64(len(w))
	span := total + mean
	out := make([]float64, c)
	pos := mean / 2
	for k := range out {
		out[k] = pos / span
		if k < len(w) {
			pos += w[k]
		}
	}
	return out
}

// tlBroken reports the gaps tlTimeScale had to shorten (a break mark).
func tlBroken(ms []*deck.Milestone) map[int]bool {
	out := map[int]bool{}
	var dated []float64
	for k := 1; k < len(ms); k++ {
		if ms[k].Dated && ms[k-1].Dated {
			dated = append(dated, ms[k].Year-ms[k-1].Year)
		}
	}
	if len(dated) == 0 {
		return out
	}
	sort.Float64s(dated)
	med := math.Max(dated[len(dated)/2], 1)
	for k := 1; k < len(ms); k++ {
		if ms[k].Dated && ms[k-1].Dated && (ms[k].Year-ms[k-1].Year)/med > 1.8 {
			out[k] = true
		}
	}
	return out
}

func (dr *deckRenderer) timelineHTML(t *deck.Timeline, o tlOpts, dir string, sc *slideCtx) string {
	var b strings.Builder
	ms := t.Milestones()
	snake := o.layout == "snake" || o.layout == "line"
	fmt.Fprintf(&b, `<div class="si-tl" data-layout="%s" data-orient="%s" style="--_tl-ar:%s"`, o.layout, o.orient, pct(o.ratio))
	if o.page {
		b.WriteString(` data-page`)
	}
	if o.lang != "" {
		fmt.Fprintf(&b, ` lang="%s"`, html.EscapeString(o.lang))
	}
	b.WriteString(">\n")
	// header: title, subtitle, the eras legend
	named := 0
	for _, e := range t.Eras {
		if e.Name != "" {
			named++
		}
	}
	if o.title != "" || o.subtitle != "" || named > 0 {
		b.WriteString(`<header class="si-tl-head"><div class="si-tl-titles">`)
		if o.title != "" {
			fmt.Fprintf(&b, `<h1 class="si-tl-title">%s</h1>`, dr.inline(o.title))
		}
		if o.subtitle != "" {
			fmt.Fprintf(&b, `<p class="si-tl-sub">%s</p>`, dr.inline(o.subtitle))
		}
		b.WriteString("</div>")
		if named > 0 {
			b.WriteString(`<ol class="si-tl-legend">`)
			for i, e := range t.Eras {
				if e.Name == "" {
					continue
				}
				span := ""
				if len(e.Items) > 0 {
					first, last := e.Items[0].Date, e.Items[len(e.Items)-1].Date
					span = html.EscapeString(first)
					if last != first {
						span += " – " + html.EscapeString(last)
					}
				}
				fmt.Fprintf(&b, `<li%s><b>%s</b><span>%s</span></li>`, tlEraStyle(e, i), dr.inline(e.Name), span)
			}
			b.WriteString("</ol>")
		}
		b.WriteString("</header>\n")
	}
	b.WriteString(`<div class="si-tl-track">` + "\n")
	var places []tlPlace
	var rows int
	var counts []int
	if snake {
		rows, counts, places = tlGeometry(t, o)
		fmt.Fprintf(&b, `<svg class="si-tl-svg" viewBox="0 0 100 100" preserveAspectRatio="none" aria-hidden="true">`)
		b.WriteString(tlPaths(t, o, rows, counts, places, -1))
		b.WriteString("</svg>\n")
	}
	broken := map[int]bool{}
	if o.scale == "time" && snake {
		broken = tlBroken(ms)
	}
	idx := 0
	step := 0
	if o.steps {
		step = sc.steps
	}
	for ei, e := range t.Eras {
		cls := "si-tl-era"
		attr := ""
		if o.steps && ei > 0 {
			cls += " si-step"
			attr = fmt.Sprintf(` data-step="%d"`, step+ei)
		}
		fmt.Fprintf(&b, `<section class="%s"%s%s>`+"\n", cls, attr, tlEraStyle(e, ei))
		if snake {
			fmt.Fprintf(&b, `<svg class="si-tl-svg" viewBox="0 0 100 100" preserveAspectRatio="none" aria-hidden="true">%s</svg>`+"\n", tlPaths(t, o, rows, counts, places, ei))
		}
		if e.Name != "" {
			fmt.Fprintf(&b, `<h2 class="si-tl-era-name">%s</h2>`+"\n", dr.inline(e.Name))
		}
		for _, m := range e.Items {
			side := "below"
			style := ""
			if snake {
				p := places[idx]
				if p.above {
					side = "above"
				}
				left, right := math.Max(p.x-p.w/2, 3.8), math.Min(p.x+p.w/2, 96.2) // an edge card is cut (clear of the turns), not moved
				edge := ""
				if left > p.x-p.w/2 {
					edge = ` data-edge="start"`
				} else if right < p.x+p.w/2 {
					edge = ` data-edge="end"`
				}
				h := 50 / float64(rows) // half a row: the room of a card on its side
				near, far := 0.0, h
				if p.level >= 0 && o.layout == "line" && len(ms) > 12 {
					near, far = float64(p.level)*h/2, float64(p.level+1)*h/2
				}
				style = fmt.Sprintf(` style="--_x:%s;--_y:%s;--_l:%s;--_w:%s;--_n:%s;--_f:%s"`, pct(p.x), pct(p.y), pct(left), pct(right-left), pct(near), pct(far)) + edge
			} else if idx%2 == 1 && o.alternate {
				side = "below"
			} else {
				side = "above"
			}
			brk := ""
			if broken[idx] {
				brk = ` data-break`
			}
			fmt.Fprintf(&b, `<article class="si-tl-ms" data-side="%s"%s%s>`, side, brk, style)
			pic := ""
			if m.Img != "" {
				if src, ok := dr.tlPicture(dir, m.Img, sc); ok {
					pic = src
				}
			}
			if pic != "" {
				fmt.Fprintf(&b, `<span class="si-tl-node si-tl-pic"><img src="%s" alt="%s" loading="lazy"></span>`, html.EscapeString(pic), html.EscapeString(m.Caption))
			} else {
				b.WriteString(`<span class="si-tl-node si-tl-dot"></span>`)
			}
			b.WriteString(`<div class="si-tl-card">`)
			fmt.Fprintf(&b, `<p class="si-tl-date"><time>%s</time></p>`, html.EscapeString(m.Date))
			fmt.Fprintf(&b, `<h3 class="si-tl-name">%s</h3>`, dr.inline(m.Title))
			if m.Text != "" {
				fmt.Fprintf(&b, `<p class="si-tl-desc">%s</p>`, dr.inline(m.Text))
			}
			if m.Img != "" && (m.Caption != "" || m.Credit != "") {
				b.WriteString(`<p class="si-tl-cap">`)
				if m.Caption != "" {
					fmt.Fprintf(&b, `<span>%s</span>`, html.EscapeString(m.Caption))
				}
				if m.Credit != "" {
					fmt.Fprintf(&b, ` <small>%s</small>`, html.EscapeString(m.Credit))
				}
				b.WriteString(`</p>`)
			}
			b.WriteString("</div></article>\n")
			idx++
		}
		b.WriteString("</section>\n")
	}
	b.WriteString("</div>\n")
	if o.closing != "" || len(t.Sources) > 0 {
		b.WriteString(`<footer class="si-tl-foot">`)
		if o.closing != "" {
			fmt.Fprintf(&b, `<p class="si-tl-closing">%s</p>`, dr.inline(o.closing))
		}
		if len(t.Sources) > 0 {
			title := t.SourcesTitle
			if title == "" {
				title = "Sources"
			}
			fmt.Fprintf(&b, `<section class="si-tl-sources"><h2>%s</h2><ul>`, dr.inline(title))
			for _, s := range t.Sources {
				fmt.Fprintf(&b, "<li>%s</li>", dr.inline(s))
			}
			b.WriteString("</ul></section>")
		}
		b.WriteString("</footer>\n")
	}
	b.WriteString("</div>\n")
	if o.steps && len(t.Eras) > 1 {
		sc.steps += len(t.Eras) - 1
	}
	return b.String()
}

// tlEraStyle is an era's colour: its own color=, else the theme palette's
// entry i (timeline.css derives it from --si-tl-base).
func tlEraStyle(e *deck.TimelineEra, i int) string {
	if e.Color != "" {
		return fmt.Sprintf(` style="--si-tl-c:%s;--_tl-i:%d"`, html.EscapeString(e.Color), i)
	}
	return fmt.Sprintf(` style="--_tl-i:%d"`, i)
}

// tlPaths draws the line of era ei (-1: the whole line, the faint track
// under the eras): from half-way to the previous milestone to half-way to
// the next, a row's end turning down into the next row.
func tlPaths(t *deck.Timeline, o tlOpts, rows int, counts []int, places []tlPlace, ei int) string {
	const m = 3.2
	pt := func(r int, f float64) (float64, float64) {
		x := m + f*(100-2*m)
		if r%2 == 1 {
			x = 100 - x
		}
		return x, (float64(r) + 0.5) / float64(rows) * 100
	}
	seg := func(ra int, fa float64, rb int, fb float64) string {
		var p strings.Builder
		x, y := pt(ra, fa)
		fmt.Fprintf(&p, "M%s %s", pct(x), pct(y))
		for r := ra; r <= rb; r++ {
			end := 1.0
			if r == rb {
				end = fb
			}
			x, y = pt(r, end)
			fmt.Fprintf(&p, "L%s %s", pct(x), pct(y))
			if r < rb {
				_, y2 := pt(r+1, 0)
				sweep := 1
				if r%2 == 1 {
					sweep = 0
				}
				fmt.Fprintf(&p, "A%s %s 0 0 %d %s %s", pct(m*0.85), pct((y2-y)/2), sweep, pct(x), pct(y2))
			}
		}
		return p.String()
	}
	last := len(places) - 1
	if ei < 0 {
		d := seg(0, 0, places[last].row, 1)
		return `<path class="si-tl-rail" d="` + d + `"/>`
	}
	// the era's first and last milestone indexes
	start := 0
	for k := 0; k < ei; k++ {
		start += len(t.Eras[k].Items)
	}
	if len(t.Eras[ei].Items) == 0 {
		return ""
	}
	end := start + len(t.Eras[ei].Items) - 1
	ra, fa := places[start].row, 0.0
	if start > 0 && places[start-1].row == ra {
		fa = (places[start-1].f + places[start].f) / 2
	}
	rb, fb := places[end].row, 1.0
	if end < last && places[end+1].row == rb {
		fb = (places[end].f + places[end+1].f) / 2
	} else if end < last { // the turn into the next row is this era's
		rb, fb = rb+1, 0
	}
	d := seg(ra, fa, rb, fb)
	return `<path class="si-tl-band" d="` + d + `"/><path class="si-tl-stroke" d="` + d + `"/>`
}

// tlPicture is a milestone picture for the page: a local file by the media
// mode, an https:// picture fetched once (cached) and inlined.
func (dr *deckRenderer) tlPicture(dir, src string, sc *slideCtx) (string, bool) {
	if strings.HasPrefix(src, "data:") {
		return src, true
	}
	if strings.HasPrefix(src, "https://") || strings.HasPrefix(src, "http://") {
		raw, err := deck.FetchAsset(src, nil)
		if err != nil {
			dr.problem(sc, "timeline: cannot fetch %s: %v", src, err)
			return "", false
		}
		typ := extraTypes[strings.ToLower(filepath.Ext(cleanSrc(src)))]
		if typ == "" {
			typ = http.DetectContentType(raw)
		}
		return "data:" + typ + ";base64," + base64.StdEncoding.EncodeToString(raw), true
	}
	return dr.media(dir, src, sc) // a missing file: Timeline.Check reports it
}

func (dr *deckRenderer) timelineCSS(b *bytes.Buffer) {
	if dr.timeline {
		b.WriteString(`<style id="si-tl-css">` + "\n" + assets.TimelineCSS + "</style>\n")
	}
}

func (dr *deckRenderer) timelineJS(b *bytes.Buffer) {
	if dr.timeline {
		b.WriteString("<script>\n" + assets.TimelineJS + "</script>\n")
	}
}
