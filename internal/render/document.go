package render

import (
	"bytes"
	"fmt"
	"html"
	"regexp"
	"strings"
	"unicode"

	"github.com/felipeg48/simple-ideas/internal/assets"
	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

// Document renders a lab (`type: lab`): one readable page with a header, a
// table of contents, the guide and a footer. It reuses the deck pipeline
// (markdown, media modes, .md links, highlighter, Mermaid, theme, user CSS,
// live reload); slide machinery (steps, canvases, layouts) does not apply.
func Document(d *deck.Deck, opt Options) Result {
	dr := &deckRenderer{d: d, opt: opt, doc: true, lastState: map[string][]string{}, words: map[string]bool{}, files: map[string]string{}}
	dr.md = dr.markdown()
	dr.resolveTheme()
	var body string
	var toc []tocEntry
	if len(d.Slides) > 0 {
		body, toc = dr.docBody(d.Slides[0])
	}
	res := Result{}
	dr.finish(&res)
	res.HTML = dr.docPage(body, toc, res.Problems)
	return res
}

type tocEntry struct {
	level int
	id    string
	text  string // escaped plain text
}

var (
	calloutRE = regexp.MustCompile(`(?i)^\[!(note|tip|important|warning|caution)\]$`)
	docHeadRE = regexp.MustCompile(`(?s)<h([23])>(.*?)</h[23]>`)
)

// docBody renders the lab body: the leading H1 moves to the page header,
// GitHub alerts become callouts, a paragraph holding one image a figure, h2/h3
// get ids and anchors, and tables scroll inside their own box.
func (dr *deckRenderer) docBody(s *deck.Slide) (string, []tocEntry) {
	sc := &slideCtx{s: s}
	src := []byte(s.Body)
	doc := dr.md.Parser().Parse(text.NewReader(src))
	for c, first := doc.FirstChild(), true; c != nil; first = false {
		next := c.NextSibling()
		switch n := c.(type) {
		case *ast.Heading:
			if first && n.Level == 1 && (dr.d.Meta.Title == "" || strings.TrimSpace(textOf(n, src)) == plain(dr.d.Meta.Title)) {
				doc.RemoveChild(doc, n) // it is the title: the header shows it
			}
		case *ast.Blockquote:
			if cl := callout(n, src); cl != nil {
				doc.ReplaceChild(doc, n, cl)
			}
		case *ast.Paragraph:
			if fig := imageFigure(n, src, nil); fig != nil {
				doc.ReplaceChild(doc, n, fig)
			}
		}
		c = next
	}
	var tables []ast.Node
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if _, ok := n.(*east.Table); ok && entering {
			tables = append(tables, n)
			return ast.WalkSkipChildren, nil
		}
		return ast.WalkContinue, nil
	})
	for _, t := range tables { // a markdown table scrolls inside its own box
		w := &divNode{class: "si-table-wrap"}
		t.Parent().ReplaceChild(t.Parent(), t, w)
		w.AppendChild(w, t)
	}
	var buf bytes.Buffer
	if err := dr.md.Renderer().Render(&buf, src, doc); err != nil {
		dr.problem(sc, "render: %v", err)
	}
	h := dr.postprocess(buf.String(), s.Dir, sc)
	var toc []tocEntry
	sl := newSlugger()
	h = docHeadRE.ReplaceAllStringFunc(h, func(m string) string {
		g := docHeadRE.FindStringSubmatch(m)
		txt := plain(g[2])
		id := sl.slug(html.UnescapeString(txt))
		toc = append(toc, tocEntry{int(g[1][0] - '0'), id, txt})
		return fmt.Sprintf(`<h%s id="%s">%s<a class="si-anchor" href="#%s" aria-label="Link to this section">#</a></h%s>`, g[1], id, g[2], id, g[1])
	})
	return h, toc
}

// calloutIcons are the 24×24 stroke icons of the five callout kinds.
var calloutIcons = map[string]string{
	"note":      `<circle cx="12" cy="12" r="9"/><path d="M12 11v5"/><path d="M12 7.5h.01"/>`,
	"tip":       `<path d="M9 18h6"/><path d="M10 21h4"/><path d="M12 3a6 6 0 0 0-3.6 10.8c.6.5 1 1.2 1 2V16h5.2v-.2c0-.8.4-1.5 1-2A6 6 0 0 0 12 3z"/>`,
	"important": `<path d="M4 5h16v11H9l-5 4z"/><path d="M12 8v3.5"/><path d="M12 14h.01"/>`,
	"warning":   `<path d="M12 3.5 2.5 20h19z"/><path d="M12 10v4"/><path d="M12 17h.01"/>`,
	"caution":   `<path d="M8.2 3h7.6L21 8.2v7.6L15.8 21H8.2L3 15.8V8.2z"/><path d="M12 8v4.5"/><path d="M12 16h.01"/>`,
}

// callout turns a GitHub alert (a blockquote whose first line is `[!NOTE]`,
// `[!TIP]`, `[!IMPORTANT]`, `[!WARNING]` or `[!CAUTION]`) into a callout;
// nil for an ordinary quote.
func callout(bq *ast.Blockquote, src []byte) ast.Node {
	p, ok := bq.FirstChild().(*ast.Paragraph)
	if !ok || p.Lines().Len() == 0 {
		return nil
	}
	line := p.Lines().At(0)
	m := calloutRE.FindStringSubmatch(strings.TrimSpace(string(line.Value(src))))
	if m == nil {
		return nil
	}
	for c := p.FirstChild(); c != nil; {
		next := c.NextSibling()
		t, ok := c.(*ast.Text)
		if !ok || t.Segment.Start >= line.Stop {
			break
		}
		p.RemoveChild(p, c)
		c = next
	}
	if p.ChildCount() == 0 {
		bq.RemoveChild(bq, p)
	}
	kind := strings.ToLower(m[1])
	d := &divNode{class: "si-callout", attrs: [][2]string{{"data-kind", kind}}}
	d.AppendChild(d, &rawNode{html: fmt.Sprintf(`<p class="si-callout-title"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">%s</svg>%s</p>`,
		calloutIcons[kind], strings.ToUpper(kind[:1])+kind[1:])})
	for c := bq.FirstChild(); c != nil; {
		next := c.NextSibling()
		bq.RemoveChild(bq, c)
		d.AppendChild(d, c)
		c = next
	}
	return d
}

// slugger makes GitHub-style heading ids: lower case, spaces to hyphens,
// punctuation dropped; a repeated slug gets -2, -3, ...
type slugger map[string]bool

func newSlugger() slugger { return slugger{} }

func (s slugger) slug(t string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(t)) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_':
			b.WriteRune(r)
		case r == ' ':
			b.WriteByte('-')
		}
	}
	id := b.String()
	if id == "" {
		id = "section"
	}
	base := id
	for n := 2; s[id]; n++ { // skip ids taken by an earlier heading, even a literal "clone-2"
		id = fmt.Sprintf("%s-%d", base, n)
	}
	s[id] = true
	return id
}

const modeIconSVG = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><circle cx="12" cy="12" r="8.5"/><path d="M12 3.5v17a8.5 8.5 0 0 0 0-17z" fill="currentColor"/></svg>`

func (dr *deckRenderer) docPage(body string, toc []tocEntry, problems []deck.Problem) []byte {
	d, m := dr.d, dr.d.Meta
	var b bytes.Buffer
	modeAttr := dr.head(&b)
	b.WriteString(`<style id="si-css">` + "\n" + assets.LayerOrder + assets.DocCSS + dr.themeCSS + dr.accentCSS() + "</style>\n")
	dr.userAndMode(&b, modeAttr)
	b.WriteString("</head>\n<body class=\"si-doc\">\n<header class=\"si-doc-head\"><div class=\"si-doc-head-in\">\n")
	fmt.Fprintf(&b, "<h1 class=\"si-doc-title\">%s</h1>\n", dr.inline(d.Title()))
	var meta []string
	for _, v := range []string{m.Author, m.Date, m.Version} {
		if v = strings.TrimSpace(v); v != "" {
			meta = append(meta, "<span>"+html.EscapeString(v)+"</span>")
		}
	}
	if len(meta) > 0 {
		fmt.Fprintf(&b, "<p class=\"si-doc-meta\">%s</p>\n", strings.Join(meta, ""))
	}
	b.WriteString(`<button type="button" class="si-doc-mode" data-si-action="mode" title="Light / dark (d)" aria-label="Light / dark">` + modeIconSVG + "</button>\n")
	b.WriteString("</div></header>\n<div class=\"si-doc-layout\">\n")
	if len(toc) > 0 {
		b.WriteString("<nav class=\"si-doc-toc\" aria-label=\"Contents\"><details open><summary>On this page</summary>\n<ol>\n")
		for _, e := range toc {
			fmt.Fprintf(&b, "<li class=\"si-toc-h%d\"><a href=\"#%s\">%s</a></li>\n", e.level, e.id, e.text)
		}
		b.WriteString("</ol></details></nav>\n")
	}
	b.WriteString("<main class=\"si-doc-body\">\n" + body + "</main>\n</div>\n")
	var foot []string
	if c := copyrightLine(m.Year, m.Copyright); c != "" {
		foot = append(foot, "<span>"+html.EscapeString(c)+"</span>")
	}
	if f := strings.TrimSpace(m.Footer); f != "" && f != "off" {
		foot = append(foot, "<span>"+html.EscapeString(f)+"</span>")
	}
	fmt.Fprintf(&b, "<footer class=\"si-doc-foot\">%s</footer>\n", strings.Join(foot, ""))
	dr.problemsBanner(&b, problems)
	dr.mermaidScript(&b)
	b.WriteString("<script>\n" + assets.DocJS + "</script>\n")
	dr.reload(&b)
	b.WriteString("</body>\n</html>\n")
	return b.Bytes()
}
