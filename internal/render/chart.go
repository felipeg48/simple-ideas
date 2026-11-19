package render

import (
	"fmt"
	"html"
	"math"
	"sort"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/yuin/goldmark/ast"
)

// Charts: a ```chart fence (deck.ParseChart) is drawn with the infographic
// chart code (drawChart), one frame per step: a fence with steps has a frame
// per number column, and a run of chart fences with the same id= is one show
// whose frames follow each other (a step can turn bars into a donut). All
// frames sit in one grid cell, so the figure keeps the tallest one's height;
// the one on screen carries data-on. web/chart.js moves between frames: what
// both show (rows by data-key, arcs, lines, numbers) is tweened from the old
// values to the new ones, rows that change place glide (FLIP), and a frame
// of another kind cross-fades in. On a slide the frames are build steps.

type chartSrc struct {
	attrs map[string]string
	body  string
}

// chartShows groups chart fences: the first of a same-id run keeps the run,
// the others leave the tree (as code windows do).
func (dr *deckRenderer) chartShows(doc ast.Node, src []byte) {
	if dr.charts == nil {
		dr.charts = map[ast.Node][]chartSrc{}
	}
	var first ast.Node
	id := ""
	for c := doc.FirstChild(); c != nil; {
		next := c.NextSibling()
		n, ok := c.(*ast.FencedCodeBlock)
		if !ok || fenceLang(n, src) != "chart" {
			first, id = nil, ""
			c = next
			continue
		}
		attrs := map[string]string{}
		if n.Info != nil {
			attrs = fenceAttrs(string(n.Info.Segment.Value(src)))
		}
		cs := chartSrc{attrs: attrs, body: fenceBody(n, src)}
		if first != nil && attrs["id"] != "" && attrs["id"] == id {
			dr.charts[first] = append(dr.charts[first], cs)
			doc.RemoveChild(doc, c)
		} else {
			first, id = n, attrs["id"]
			dr.charts[n] = []chartSrc{cs}
		}
		c = next
	}
}

func fenceBody(n *ast.FencedCodeBlock, src []byte) string {
	var b strings.Builder
	for i := 0; i < n.Lines().Len(); i++ {
		seg := n.Lines().At(i)
		b.Write(seg.Value(src))
	}
	return strings.TrimRight(b.String(), "\n")
}

// chartFence writes a chart show: its frames, and on a slide one build step
// per frame after the first.
func (dr *deckRenderer) chartFence(group []chartSrc) string {
	sc := dr.cur
	dr.chart = true
	var frames []string
	title, id, race := "", "", false
	for _, g := range group {
		c, probs := deck.ParseChart(g.attrs, g.body)
		for _, p := range probs {
			dr.problem(sc, "%s", p)
		}
		if len(c.Head) == 0 {
			continue
		}
		title = firstNonEmpty(title, c.Title)
		id = firstNonEmpty(id, c.ID)
		race = race || c.Race
		sets := [][]int{}
		if c.Steps {
			for j := range c.Head {
				sets = append(sets, []int{j})
			}
		} else {
			all := []int{}
			for j := range c.Head {
				all = append(all, j)
			}
			sets = append(sets, all)
		}
		top, low := 0.0, 0.0 // steps share one scale (a race rescales each step)
		for _, r := range c.Vals {
			for _, v := range r {
				top, low = math.Max(top, v), math.Min(low, v)
			}
		}
		for si, cols := range sets {
			s := igSeries{}
			for _, j := range cols {
				s.names = append(s.names, c.Head[j])
			}
			order := make([]int, len(c.Labels))
			for i := range order {
				order[i] = i
			}
			sum := func(i int) float64 {
				t := 0.0
				for _, j := range cols {
					t += c.Vals[i][j]
				}
				return t
			}
			if c.Sort && c.Type != "line" && c.Type != "area" {
				sort.SliceStable(order, func(a, b int) bool { return sum(order[a]) > sum(order[b]) })
			}
			for _, i := range order {
				s.labels = append(s.labels, c.Labels[i])
				var v []float64
				var t []string
				for _, j := range cols {
					v = append(v, c.Vals[i][j])
					t = append(t, c.ChartValue(i, j))
					s.max, s.min = math.Max(s.max, c.Vals[i][j]), math.Min(s.min, c.Vals[i][j])
				}
				s.vals, s.text = append(s.vals, v), append(s.text, t)
			}
			if c.Steps && !c.Race {
				s.max, s.min = top, low
			}
			frame := func(lifted int) string {
				co := chartOpts{keys: order, ownHue: c.Sort || strings.HasSuffix(c.Type, "3d"), stacked: c.Stacked}
				if c.Explode != "" {
					co.explode = &lifted
				}
				var f strings.Builder
				fmt.Fprintf(&f, `<div class="si-chart-frame" data-kind="%s">`, c.Type)
				if c.Steps {
					f.WriteString(`<p class="si-chart-at" aria-live="polite">` + dr.inline(c.Head[cols[0]]) + `</p>`)
				}
				if strings.HasSuffix(c.Type, "3d") {
					f.WriteString(dr.draw3D(c, firstNonEmpty(c.Title, title), s, co))
				} else {
					f.WriteString(dr.drawChart(c.Type, firstNonEmpty(c.Title, title), s, co))
				}
				f.WriteString(`</div>`)
				return f.String()
			}
			frames = append(frames, frame(-1))
			if c.Explode != "" && si == len(sets)-1 { // the slice lifts out on a step of its own
				frames = append(frames, frame(c.ExplodeRow()))
			}
		}
	}
	if len(frames) == 0 {
		return "<figure class=\"si-figure si-chart\"></figure>\n"
	}
	on := 0
	if dr.doc || sc == nil {
		on = len(frames) - 1 // a page without the script, or printed, shows where the show ends
	}
	frames[on] = strings.Replace(frames[on], `<div class="si-chart-frame"`, `<div class="si-chart-frame" data-on`, 1)
	var b strings.Builder
	b.WriteString(`<figure class="si-figure si-chart"`)
	if id != "" {
		b.WriteString(` data-chart-id="` + html.EscapeString(id) + `"`)
	}
	if race {
		b.WriteString(` data-race`)
	}
	if (dr.doc || sc == nil) && len(frames) > 1 {
		b.WriteString(` data-chart-nav`)
	}
	b.WriteString(` aria-label="` + html.EscapeString(firstNonEmpty(plain(title), "Chart")) + `">`)
	if title != "" {
		b.WriteString(`<figcaption class="si-chart-title">` + dr.inline(title) + `</figcaption>`)
	}
	b.WriteString(`<div class="si-chart-stage">` + strings.Join(frames, "") + `</div>`)
	if sc != nil && !dr.doc {
		for i := 1; i < len(frames); i++ {
			fmt.Fprintf(&b, `<span class="si-step si-chart-mark" data-step="%d" hidden></span>`, sc.steps+i)
			sc.ops = append(sc.ops, "")
		}
		sc.steps += len(frames) - 1
	}
	b.WriteString("</figure>\n")
	return b.String()
}
