package render

import (
	"bytes"
	"fmt"
	"html"
	"strconv"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/renderer"
	ghtml "github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// divNode wraps blocks: a build step or a column.
type divNode struct {
	ast.BaseBlock
	class string
	attrs [][2]string
}

// rawNode is pre-rendered HTML: an inlined SVG canvas or a state-only step.
type rawNode struct {
	ast.BaseBlock
	html  string
	media bool
}

var (
	kindDiv = ast.NewNodeKind("SIDiv")
	kindRaw = ast.NewNodeKind("SIRaw")
)

func (n *divNode) Kind() ast.NodeKind         { return kindDiv }
func (n *divNode) Dump(src []byte, level int) { ast.DumpHelper(n, src, level, nil, nil) }
func (n *rawNode) Kind() ast.NodeKind         { return kindRaw }
func (n *rawNode) Dump(src []byte, level int) { ast.DumpHelper(n, src, level, nil, nil) }
func (n *rawNode) IsRaw() bool                { return true }
func (r *nodeRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(kindDiv, r.div)
	reg.Register(kindRaw, r.raw)
	reg.Register(ast.KindFencedCodeBlock, r.fence)
}

// nodeRenderer renders the custom nodes and fenced code (highlighting, mermaid).
type nodeRenderer struct{ dr *deckRenderer }

func (r *nodeRenderer) div(
	w util.BufWriter, _ []byte, node ast.Node, entering bool,
) (ast.WalkStatus, error) {
	n := node.(*divNode)
	if entering {
		fmt.Fprintf(w, `<div class="%s"`, n.class)
		for _, a := range n.attrs {
			fmt.Fprintf(w, ` %s="%s"`, a[0], html.EscapeString(a[1]))
		}
		w.WriteString(">\n")
	} else {
		w.WriteString("</div>\n")
	}
	return ast.WalkContinue, nil
}

func (r *nodeRenderer) raw(
	w util.BufWriter, _ []byte, node ast.Node, entering bool,
) (ast.WalkStatus, error) {
	if entering {
		w.WriteString(node.(*rawNode).html)
		w.WriteString("\n")
	}
	return ast.WalkSkipChildren, nil
}

func (r *nodeRenderer) fence(
	w util.BufWriter, src []byte, node ast.Node, entering bool,
) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n := node.(*ast.FencedCodeBlock)
	lang := strings.ToLower(string(n.Language(src)))
	var code bytes.Buffer
	for i := 0; i < n.Lines().Len(); i++ {
		seg := n.Lines().At(i)
		code.Write(seg.Value(src))
	}
	body := strings.TrimRight(code.String(), "\n")
	if lang == "mermaid" && r.dr.d.Meta.Mermaid != "off" {
		r.dr.mermaid = true
		fmt.Fprintf(w, "<pre class=\"si-mermaid\">%s</pre>\n", html.EscapeString(body))
		return ast.WalkSkipChildren, nil
	}
	if lang == "stage" {
		r.dr.stageN++
		svg, stepOps, err := genStage(body, r.dr.stageN)
		if err != nil {
			r.dr.problems = append(r.dr.problems, deck.Problem{File: r.dr.d.Path, Msg: "stage: " + err.Error()})
			fmt.Fprintf(w, "<figure class=\"si-figure\"><pre class=\"si-code\">stage: %s</pre></figure>\n", html.EscapeString(err.Error()))
			return ast.WalkSkipChildren, nil
		}
		fmt.Fprintf(w, "<figure class=\"si-figure\">%s</figure>\n", svg)
		// emit one state-step span per `step` line, numbered after the slide's
		// existing steps, so deck.js drives the generated canvas's data-state.
		if sc := r.dr.cur; sc != nil && len(stepOps) > 0 {
			for i, ops := range stepOps {
				fmt.Fprintf(w, `<span class="si-step si-state" data-step="%d" data-ops="%s" hidden></span>`+"\n",
					sc.steps+1+i, html.EscapeString(ops))
			}
			sc.steps += len(stepOps)
			sc.ops = append(sc.ops, stepOps...)
		}
		return ast.WalkSkipChildren, nil
	}
	if r.dr.doc { // a lab: a header bar with the language and a copy button
		fmt.Fprintf(w, `<div class="si-codeblock"><div class="si-code-head"><span class="si-code-lang">%s</span><button type="button" class="si-copy" aria-label="Copy code">Copy</button></div>`, html.EscapeString(lang))
		defer w.WriteString("</div>\n")
	}
	if lang != "" {
		fmt.Fprintf(w, `<pre class="si-code" data-lang="%s"><code class="language-%s">`, html.EscapeString(lang), html.EscapeString(lang))
	} else {
		w.WriteString(`<pre class="si-code"><code>`)
	}
	w.WriteString(highlight(body, lang))
	w.WriteString("</code></pre>\n")
	return ast.WalkSkipChildren, nil
}

func (dr *deckRenderer) markdown() goldmark.Markdown {
	return goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithRendererOptions(
			ghtml.WithUnsafe(),
			renderer.WithNodeRenderers(util.Prioritized(&nodeRenderer{dr: dr}, 100)),
		),
	)
}

// slideCtx is the per-slide state of the body transform.
type slideCtx struct {
	s        *deck.Slide
	steps    int
	ops      []string // every step's state ops, in order
	canvases []canvasRef
	cols     bool
}

type canvasRef struct {
	path string
	base []string
}

// toHTML converts a slide body: directives become steps, columns and canvases.
func (dr *deckRenderer) toHTML(md string, sc *slideCtx) string {
	src := []byte(md)
	doc := dr.md.Parser().Parse(text.NewReader(src))
	if sc != nil {
		dr.transform(doc, src, sc)
	}
	dr.cur = sc // the stage fence reads/updates this slide's step count
	defer func() { dr.cur = nil }()
	var buf bytes.Buffer
	if err := dr.md.Renderer().Render(&buf, src, doc); err != nil {
		dr.problem(sc, "render: %v", err)
	}
	return buf.String()
}

// inline renders a single line of inline markdown (a heading).
func (dr *deckRenderer) inline(s string) string {
	h := strings.TrimSpace(dr.toHTML(s, nil))
	h = strings.TrimPrefix(h, "<p>")
	return strings.TrimSuffix(h, "</p>")
}

func blockText(n ast.Node, src []byte) string {
	var b strings.Builder
	lines := n.Lines()
	for i := 0; i < lines.Len(); i++ {
		seg := lines.At(i)
		b.Write(seg.Value(src))
	}
	if hb, ok := n.(*ast.HTMLBlock); ok && hb.HasClosure() {
		b.Write(hb.ClosureLine.Value(src))
	}
	return strings.TrimSpace(b.String())
}

type placed struct {
	node ast.Node
	col  int
}

// transform walks the top-level blocks of a slide body and applies the body
// directives: svg (canvas), step / steps (build steps, optionally driving the
// canvas state) and column (column breaks).
func (dr *deckRenderer) transform(doc ast.Node, src []byte, sc *slideCtx) {
	var nodes []ast.Node
	for c := doc.FirstChild(); c != nil; c = c.NextSibling() {
		nodes = append(nodes, c)
	}
	type pend struct{ name, val string }
	var pending *pend
	var img *deck.ImageAttrs
	col, explicit := 0, false
	var out []placed
	stateOnly := func() {
		if pending == nil {
			return
		}
		sc.steps++
		ops := pending.val
		sc.ops = append(sc.ops, ops)
		out = append(out, placed{&rawNode{html: fmt.Sprintf(`<span class="si-step si-state" data-step="%d" data-ops="%s" hidden></span>`,
			sc.steps, html.EscapeString(ops))}, col})
		pending = nil
	}
	for _, c := range nodes {
		doc.RemoveChild(doc, c)
		if hb, ok := c.(*ast.HTMLBlock); ok {
			if name, val, ok := deck.ParseDirective(blockText(hb, src)); ok {
				switch name {
				case "column":
					stateOnly()
					col++
					explicit = true
					continue
				case "step", "steps":
					stateOnly()
					pending = &pend{name, val}
					continue
				case "image":
					a, _ := deck.ParseImageAttrs(val) // the parser reports bad options
					img = &a
					continue
				case "svg":
					c = dr.canvas(val, sc)
				default:
					continue // slide-level directives were consumed by the parser
				}
			}
		}
		if fig := imageFigure(c, src, img); fig != nil {
			c = fig
		} else if img != nil {
			dr.problem(sc, "an image directive must be followed by a paragraph holding one image")
		}
		img = nil
		if pending != nil {
			c = dr.applyStep(c, pending.name, pending.val, sc)
			pending = nil
		}
		out = append(out, placed{c, col})
	}
	stateOnly()

	layout := sc.s.Layout
	if explicit && (layout == "" || layout == "default") {
		sc.s.Layout = "two-col"
		layout = "two-col"
	}
	if img != nil {
		dr.problem(sc, "an image directive must be followed by a paragraph holding one image")
	}
	if layout != "two-col" && layout != "split" {
		for _, p := range out {
			doc.AppendChild(doc, p.node)
		}
		return
	}
	sc.cols = true
	groups := map[int][]ast.Node{}
	max := 0
	for _, p := range out {
		g := p.col
		if !explicit && isMedia(p.node) {
			g = 1
		}
		groups[g] = append(groups[g], p.node)
		if g > max {
			max = g
		}
	}
	for g := 0; g <= max; g++ {
		d := &divNode{class: "si-col"}
		if !explicit && max == 0 && layout == "two-col" {
			d.class = "si-col si-flow"
		}
		for _, n := range groups[g] {
			d.AppendChild(d, n)
		}
		doc.AppendChild(doc, d)
	}
}

func (dr *deckRenderer) applyStep(c ast.Node, name, val string, sc *slideCtx) ast.Node {
	if list, ok := c.(*ast.List); ok && name == "steps" {
		tokens := strings.Fields(val)
		items := 0
		for it := list.FirstChild(); it != nil; it = it.NextSibling() {
			sc.steps++
			it.SetAttributeString("class", "si-step")
			it.SetAttributeString("data-step", strconv.Itoa(sc.steps))
			if items < len(tokens) && tokens[items] != "." {
				it.SetAttributeString("data-ops", tokens[items])
				sc.ops = append(sc.ops, tokens[items])
			}
			items++
		}
		if len(tokens) > items {
			dr.problem(sc, "steps: %d state tokens for a list of %d items", len(tokens), items)
		}
		return list
	}
	sc.steps++
	d := &divNode{class: "si-step", attrs: [][2]string{{"data-step", strconv.Itoa(sc.steps)}}}
	if val != "" {
		d.attrs = append(d.attrs, [2]string{"data-ops", val})
		sc.ops = append(sc.ops, val)
	}
	d.AppendChild(d, c)
	return d
}

func isMedia(n ast.Node) bool {
	if d, ok := n.(*divNode); ok && d.ChildCount() == 1 {
		n = d.FirstChild()
	}
	switch v := n.(type) {
	case *rawNode:
		return v.media
	case *ast.FencedCodeBlock, *east.Table:
		return true
	case *ast.Paragraph:
		_, img := v.FirstChild().(*ast.Image)
		return v.ChildCount() == 1 && img
	}
	return false
}

// imageFigure turns a paragraph holding a single image into a figure (the
// image title becomes the caption); nil when c is something else.
func imageFigure(c ast.Node, src []byte, a *deck.ImageAttrs) ast.Node {
	p, ok := c.(*ast.Paragraph)
	if !ok || p.ChildCount() != 1 {
		return nil
	}
	im, ok := p.FirstChild().(*ast.Image)
	if !ok {
		return nil
	}
	return &rawNode{html: figure(textOf(im, src), string(im.Destination), string(im.Title), a), media: true}
}

func textOf(n ast.Node, src []byte) string {
	var b strings.Builder
	ast.Walk(n, func(c ast.Node, entering bool) (ast.WalkStatus, error) {
		if t, ok := c.(*ast.Text); ok && entering {
			b.Write(t.Segment.Value(src))
		}
		return ast.WalkContinue, nil
	})
	return b.String()
}
