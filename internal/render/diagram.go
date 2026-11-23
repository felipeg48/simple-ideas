package render

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/assets"
	"github.com/felipeg48/simple-ideas/internal/deck"
)

// Delivery diagrams (RULES §38): a ```kanban, ```gantt or ```sequence fence
// (deck.ParseDiagram) is one inline SVG, every element drawn in its first
// state on a slide (its last in a document or printed), every state as JSON
// for web/diagram.js, which tweens one step to the next. On a slide each
// step is a build step (.si-dg-mark); in a page the diagram plays when it is
// scrolled into view, with ← → and Play under it. What si worked out (the
// critical path, a column over its WIP limit) is the caption.

func dgElSVG(b *strings.Builder, i int, e deck.DEl, s deck.DState) {
	cls := strings.TrimSpace("si-dg-e " + e.C + " " + s.C)
	fmt.Fprintf(b, `<g class="%s" data-e="%d" transform="translate(%g %g)"`, html.EscapeString(cls), i, s.X, s.Y)
	if s.O < 1 {
		fmt.Fprintf(b, ` opacity="%g"`, s.O)
	}
	b.WriteString(">")
	label := func(x, y float64) {
		if s.T == "" && s.U == "" {
			fmt.Fprintf(b, `<text class="t" x="%g" y="%g"></text>`, x, y)
			return
		}
		fmt.Fprintf(b, `<text class="t" x="%g" y="%g">%s`, x, y, html.EscapeString(s.T))
		if s.U != "" {
			fmt.Fprintf(b, `<tspan class="u" dx="6">%s</tspan>`, html.EscapeString(s.U))
		}
		b.WriteString(`</text>`)
	}
	switch e.K {
	case "box":
		fmt.Fprintf(b, `<rect width="%g" height="%g" rx="%g"/>`, s.W, s.H, e.R)
		ty := s.H / 2
		if s.U != "" {
			ty -= 8
		}
		tx := 10.0
		if strings.Contains(e.C, "si-dg-actor") { // a name in the middle of its box
			tx = s.W / 2
		}
		fmt.Fprintf(b, `<text class="t" x="%g" y="%g">%s</text>`, tx, ty, html.EscapeString(s.T))
		fmt.Fprintf(b, `<text class="u" x="10" y="%g">%s</text>`, s.H/2+10, html.EscapeString(s.U))
	case "bar":
		fmt.Fprintf(b, `<rect class="b" width="%g" height="%g" rx="%g"/><rect class="q" width="%g" height="%g" rx="%g"/>`, s.W, s.H, e.R, s.W*s.P, s.H, e.R)
	case "ms":
		b.WriteString(`<path d="M0 -9L9 0L0 9L-9 0Z"/>`)
	case "line":
		fmt.Fprintf(b, `<path d="%s" pathLength="1" stroke-dasharray="1 1" stroke-dashoffset="%g"/>`, e.D, 1-s.P)
		label(e.TX, e.TY)
	case "text":
		label(0, 0)
	}
	b.WriteString("</g>")
}

// diagramFence writes a diagram of kind: the SVG, its states and, on a
// slide, a build step per step.
func (dr *deckRenderer) diagramFence(kind string, attrs map[string]string, body string, problem func(string)) string {
	sc := dr.cur
	dr.diagram = true
	if attrs["width"] == "" && dr.dgWidth != "" {
		attrs["width"] = dr.dgWidth
	}
	d, probs := deck.ParseDiagram(kind, attrs, body)
	for _, p := range probs {
		problem(p)
	}
	if attrs["steps"] == "off" && len(d.States) > 1 { // the last state only
		d.States, d.Steps = d.States[len(d.States)-1:], nil
	}
	if len(d.In) != len(d.Els) {
		d.In = nil
	}
	on := 0
	if dr.doc || sc == nil {
		on = len(d.States) - 1
	}
	data, _ := json.Marshal(map[string]any{"e": d.Els, "s": d.States, "in": d.In, "l": d.Steps})
	label := firstNonEmpty(plain(d.Title), strings.ToUpper(kind[:1])+kind[1:])
	var b strings.Builder
	fmt.Fprintf(&b, `<figure class="si-figure si-dg si-dg-%s" aria-label="%s" data-on="%d" data-dg="%s"`, kind, html.EscapeString(label), on, html.EscapeString(string(data)))
	if (dr.doc || sc == nil) && len(d.Steps) > 0 {
		b.WriteString(` data-dg-nav`)
	}
	fmt.Fprintf(&b, `><svg class="si-dg-svg" viewBox="0 0 %g %g" role="img" aria-label="%s">`, d.W, d.H, html.EscapeString(label))
	if len(d.States) > 0 {
		for i, e := range d.Els {
			if i < len(d.States[on]) {
				dgElSVG(&b, i, e, d.States[on][i])
			}
		}
	}
	b.WriteString(`</svg>`)
	if len(d.Facts) > 0 && attrs["facts"] != "off" {
		b.WriteString(`<figcaption class="si-dg-facts">`)
		for _, f := range d.Facts {
			fmt.Fprintf(&b, `<span>%s</span>`, html.EscapeString(f))
		}
		b.WriteString(`</figcaption>`)
	}
	if sc != nil && !dr.doc {
		for i := range d.Steps {
			fmt.Fprintf(&b, `<span class="si-step si-dg-mark" data-step="%d" hidden></span>`, sc.steps+i+1)
			sc.ops = append(sc.ops, "")
		}
		sc.steps += len(d.Steps)
	}
	b.WriteString("</figure>\n")
	return b.String()
}

// DiagramPage renders a `type: diagram` file: one frame (orientation:
// landscape, 16:9, or portrait, 3:4) with the title and description over
// the diagrams of its body. The frame carries .si-ig-frame, so si build's
// pictures, PDF, GIF and MP4 cut and film it as they do an infographic.
func DiagramPage(d *deck.Deck, opt Options) Result {
	dr := &deckRenderer{d: d, opt: opt, doc: true, lastState: map[string][]string{}, words: map[string]bool{}, files: map[string]string{}}
	dr.md = dr.markdown()
	dr.resolveTheme()
	if d.Meta.Orientation == "portrait" {
		dr.dgWidth = "600"
	}
	var body strings.Builder
	for _, s := range d.Slides {
		sc := &slideCtx{s: s}
		body.WriteString(dr.postprocess(dr.toHTML(s.Body, sc), s.Dir, sc))
	}
	dr.diagram = true
	res := Result{}
	dr.finish(&res)
	orient := "landscape"
	if d.Meta.Orientation == "portrait" {
		orient = "portrait"
	}
	var b bytes.Buffer
	modeAttr := dr.head(&b)
	b.WriteString(`<style id="si-css">` + "\n" + assets.LayerOrder + assets.DocCSS + dr.themeCSS + dr.accentCSS() + assets.DiagramCSS)
	if dr.chart || dr.infographic {
		b.WriteString(assets.InfographicCSS)
	}
	if dr.scene {
		b.WriteString(assets.SceneCSS)
	}
	b.WriteString("</style>\n")
	dr.userAndMode(&b, modeAttr)
	fmt.Fprintf(&b, "</head>\n<body class=\"si-doc si-dg-page\" data-si-doc=\"diagram\">\n<main>\n<div class=\"si-ig-frame si-dg-frame\" data-orient=\"%s\">\n", orient)
	if t := d.Title(); t != "" {
		fmt.Fprintf(&b, "<header class=\"si-dg-head\"><h1>%s</h1>", dr.inline(t))
		if desc := strings.TrimSpace(firstNonEmpty(d.Meta.Subtitle, d.Meta.Description)); desc != "" {
			fmt.Fprintf(&b, "<p>%s</p>", dr.inline(desc))
		}
		b.WriteString("</header>\n")
	}
	b.WriteString(`<div class="si-dg-body">` + body.String() + "</div>\n</div>\n</main>\n")
	dr.problemsBanner(&b, res.Problems)
	dr.timelineJS(&b)
	dr.reload(&b)
	b.WriteString("</body>\n</html>\n")
	res.HTML, res.body = b.Bytes(), body.String()
	return res
}
