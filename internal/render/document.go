package render

import (
	"bytes"
	"fmt"
	"html"
	"path/filepath"
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
	var body strings.Builder
	var toc []tocEntry
	sl, title := newSlugger(), html.UnescapeString(plain(dr.inline(d.Title())))
	for _, s := range d.Slides { // one part per run of lines of one file
		h, t := dr.docBody(s, sl, &title)
		body.WriteString(h)
		toc = append(toc, t...)
	}
	res := Result{}
	dr.finish(&res)
	res.HTML = dr.docPage(body.String(), toc, res.Problems)
	res.body, res.toc, res.labels = body.String(), toc, dr.labels
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

// docBody renders one part of the lab body (paths relative to the part's
// file): the first H1 equal to *title moves to the page header (title is then
// cleared), GitHub alerts become callouts, a paragraph holding one image a
// figure, h2/h3 get ids from sl (shared by all parts) and anchors, and tables
// scroll inside their own box. Problems point at the line of the failing
// reference.
func (dr *deckRenderer) docBody(s *deck.Slide, sl slugger, title *string) (string, []tocEntry) {
	sc := &slideCtx{s: s}
	np, nw := len(dr.problems), len(dr.warnings)
	defer func() {
		lines := strings.Split(s.Body, "\n")
		for _, ps := range [][]deck.Problem{dr.problems[np:], dr.warnings[nw:]} {
			for i := range ps {
				ps[i].Line = s.Line + refLine(lines, ps[i].Msg)
			}
		}
	}()
	src := []byte(s.Body)
	doc := dr.md.Parser().Parse(text.NewReader(src))
	dr.codeFeatures(doc, src, sc)
	dr.plugins(doc, src, s, sc)
	for c := doc.FirstChild(); c != nil; {
		next := c.NextSibling()
		switch n := c.(type) {
		case *ast.Heading:
			if *title != "" && n.Level == 1 && strings.TrimSpace(textOf(n, src)) == strings.TrimSpace(*title) {
				doc.RemoveChild(doc, n) // it is the title: the header shows it
				*title = ""
			}
		case *ast.Blockquote:
			if cl := callout(n, src); cl != nil {
				doc.ReplaceChild(doc, n, cl)
			} else if dr.d.IsDemo() {
				if say := sayCallout(n, src); say != nil {
					doc.ReplaceChild(doc, n, say)
				}
			}
		case *ast.Paragraph:
			if fig := imageFigure(n, src, nil); fig != nil {
				if im, ok := n.FirstChild().(*ast.Image); ok && string(im.Title) != "" {
					dr.numberFigure(fig.(*rawNode), string(im.Title), sl)
				}
				doc.ReplaceChild(doc, n, fig)
			}
		case *ast.HTMLBlock:
			if name, val, ok := deck.ParseDirective(blockText(n, src)); ok && name == "slides" {
				doc.ReplaceChild(doc, n, &rawNode{html: dr.embedSlides(val, s, sc)})
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
	for _, t := range tables { // a markdown table scrolls inside its own box; "Table: …" after it is its caption
		if p, ok := t.NextSibling().(*ast.Paragraph); ok && t.Parent() == doc {
			if cap, ok := strings.CutPrefix(strings.TrimSpace(textOf(p, src)), "Table:"); ok {
				doc.ReplaceChild(doc, p, &rawNode{html: dr.tableCaption(strings.TrimSpace(cap), sl)})
			}
		}
		if h, ok := t.FirstChild().(*east.TableHeader); ok && emptyRow(h) {
			h.SetAttributeString("hidden", "") // `| | |` header of a pandoc-style key/value table
		}
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
	h = docHeadRE.ReplaceAllStringFunc(h, func(m string) string {
		g := docHeadRE.FindStringSubmatch(m)
		txt := plain(g[2])
		id := sl.slug(html.UnescapeString(txt))
		toc = append(toc, tocEntry{int(g[1][0] - '0'), id, txt})
		inner := g[2]
		if g[1] == "2" && dr.opt.Chapter != "" { // a book chapter: 3.1, 3.2, …
			dr.sec++
			num := fmt.Sprintf("%s.%d", dr.opt.Chapter, dr.sec)
			inner = `<span class="si-sec-num">` + num + `</span>` + inner
			toc[len(toc)-1].text = num + " " + txt
			dr.label(id, "Section "+num)
		} else if g[1] == "2" && dr.d.Meta.Type != "article" && (dr.opt.Guide == nil || !dr.opt.Guide.IsBook()) {
			inner = stepHeading(inner) // "Step 3: Pay" gets a numbered badge
		}
		return fmt.Sprintf(`<h%s id="%s">%s<a class="si-anchor" href="#%s" aria-label="Link to this section">#</a></h%s>`, g[1], id, inner, id, g[1])
	})
	return h, toc
}

// emptyRow reports whether every cell of a table row is empty.
func emptyRow(row ast.Node) bool {
	for c := row.FirstChild(); c != nil; c = c.NextSibling() {
		if c.HasChildren() {
			return false
		}
	}
	return true
}

// refLine is the 0-based index of the first line of a part that holds the
// file a problem message names ("missing file: x", "x is 3.1 MB, ..."); 0
// when there is none.
func refLine(lines []string, msg string) int {
	var ref string
	if r, ok := strings.CutPrefix(msg, "missing file: "); ok {
		ref, _, _ = strings.Cut(r, " (")
	} else if r, rest, ok := strings.Cut(msg, " is "); ok && strings.Contains(rest, " MB, inlined") {
		ref = r
	}
	if ref == "" {
		return 0
	}
	for i, l := range lines {
		if strings.Contains(l, ref) {
			return i
		}
	}
	return 0
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

// pager links the previous and next lab of the same build (Options.Prev/Next).
func (dr *deckRenderer) pager() string {
	if dr.opt.Prev.Href == "" && dr.opt.Next.Href == "" {
		return ""
	}
	var b strings.Builder
	b.WriteString("<nav class=\"si-doc-pager\" aria-label=\"Other labs\">")
	for _, l := range []struct {
		DocLink
		rel, label string
	}{{dr.opt.Prev, "prev", "Previous"}, {dr.opt.Next, "next", "Next"}} {
		if l.Href != "" {
			fmt.Fprintf(&b, `<a rel="%s" href="%s"><small>%s</small><span>%s</span></a>`, l.rel, html.EscapeString(l.Href), l.label, html.EscapeString(l.Title))
		}
	}
	b.WriteString("</nav>\n")
	return b.String()
}

const searchIconSVG = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" aria-hidden="true"><circle cx="11" cy="11" r="6.5"/><path d="m16 16 4.5 4.5"/></svg>`

const modeIconSVG = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><circle cx="12" cy="12" r="8.5"/><path d="M12 3.5v17a8.5 8.5 0 0 0 0-17z" fill="currentColor"/></svg>`

func (dr *deckRenderer) docPage(body string, toc []tocEntry, problems []deck.Problem) []byte {
	d, m := dr.d, dr.d.Meta
	g := dr.opt.Guide
	kind := m.Type
	if g != nil {
		dr.guidePager()
	}
	var b bytes.Buffer
	modeAttr := dr.head(&b)
	b.WriteString(`<style id="si-css">` + "\n" + assets.LayerOrder + assets.DocCSS + dr.themeCSS + dr.accentCSS() + "</style>\n")
	dr.userAndMode(&b, modeAttr)
	bodyAttrs := fmt.Sprintf(` data-si-doc="%s"`, html.EscapeString(kind))
	if g != nil && g.IsBook() {
		bp := dr.opt.bookPage
		if bp == "" {
			bp = "chapter"
		}
		bodyAttrs += fmt.Sprintf(` data-si-book="%s"`, bp)
	}
	if g != nil {
		bodyAttrs += fmt.Sprintf(` data-si-guide="%s" data-si-page="%s"`, html.EscapeString(guideRel(g, g.Pages[0])+"|"+g.Meta.Title), html.EscapeString(guideRel(g, d.Path)))
	}
	if wm := dr.watermark(nil); wm != "" { // labs, articles, guides, books, demos: one fixed layer, repeated on every printed page
		var decls []string
		if dr.wmImg != "" {
			decls = append(decls, `--si-wm-img:url("`+dr.wmImg+`")`)
		}
		if m.WatermarkOpacity != "" {
			decls = append(decls, "--si-wm-opacity:"+m.WatermarkOpacity)
		}
		if len(decls) > 0 {
			fmt.Fprintf(&b, "<style id=\"si-wm\">.si-doc{%s}</style>\n", strings.Join(decls, ";"))
		}
		fmt.Fprintf(&b, "</head>\n<body class=\"si-doc\"%s>\n%s", bodyAttrs, strings.Replace(wm, `class="si-watermark `, `class="si-watermark si-doc-wm `, 1))
	} else {
		fmt.Fprintf(&b, "</head>\n<body class=\"si-doc\"%s>\n", bodyAttrs)
	}

	var meta []string
	byline := []string{m.Author, m.Date, m.Version}
	if g != nil { // a guide page's byline shows only what the page itself sets
		for i, inherited := range []string{g.Meta.Author, g.Meta.Date, g.Meta.Version} {
			if byline[i] == inherited {
				byline[i] = ""
			}
		}
	}
	for _, v := range byline {
		for _, part := range strings.Split(v, "·") { // "A · B" is two items: CSS draws the separators
			if part = strings.TrimSpace(part); part != "" {
				meta = append(meta, "<span>"+html.EscapeString(part)+"</span>")
			}
		}
	}
	if kind == "article" {
		meta = append(meta, fmt.Sprintf(`<span class="si-reading-time">%d min read</span>`, readingTime(body)))
	}
	metaHTML := ""
	if len(meta) > 0 {
		metaHTML = fmt.Sprintf("<p class=\"si-doc-meta\">%s</p>\n", strings.Join(meta, ""))
	}
	lede := ""
	if desc := strings.TrimSpace(m.Description); desc != "" && kind != "lab" {
		lede = "<p class=\"si-lede\">" + dr.inline(desc) + "</p>\n"
	}
	modeBtn := `<button type="button" class="si-doc-search" data-si-action="search" title="Search (/)" aria-label="Search">` + searchIconSVG + `<span>Search</span><kbd>/</kbd></button>` +
		`<button type="button" class="si-doc-mode" data-si-action="mode" title="Light / dark (d)" aria-label="Light / dark">` + modeIconSVG + "</button>\n"
	tocHTML := ""
	h2s := 0
	for _, e := range toc {
		if e.level == 2 {
			h2s++
		}
	}
	if len(toc) > 0 && (kind != "article" || h2s >= 3) {
		var t strings.Builder
		t.WriteString("<nav class=\"si-doc-toc\" aria-label=\"On this page\"><details open><summary>On this page</summary>\n<ol>\n")
		for _, e := range toc {
			fmt.Fprintf(&t, "<li class=\"si-toc-h%d\"><a href=\"#%s\">%s</a></li>\n", e.level, e.id, e.text)
		}
		t.WriteString("</ol></details></nav>\n")
		tocHTML = t.String()
	}

	if g != nil && g.IsSite() {
		var sb strings.Builder
		dr.sitePage(&sb, body, lede, metaHTML, tocHTML)
		b.WriteString(sb.String())
	} else if g != nil {
		if m.Header != "off" {
			if h := dr.partial("_header.md"); h != "" {
				b.WriteString("<div class=\"si-doc-site\"><div class=\"si-doc-site-in\">" + h + "</div></div>\n")
			}
		}
		home := pageHref(d.Path, g.Pages[0])
		if g.IsBook() {
			home = pageHref(d.Path, filepath.Join(g.Dir, "index.md"))
		}
		fmt.Fprintf(&b, "<div class=\"si-doc-guidebar\"><a class=\"si-doc-guide\" href=\"%s\">%s</a>", html.EscapeString(home), html.EscapeString(g.Meta.Title))
		if v := firstNonEmpty(g.Meta.Edition, g.Meta.Version); v != "" {
			fmt.Fprintf(&b, "<span class=\"si-doc-ver\">%s</span>", html.EscapeString(v))
		}
		if g.IsBook() {
			b.WriteString(`<nav class="si-book-bar" aria-label="Book">`)
			for _, l := range [][3]string{{"index.md", "Cover", "cover"}, {"contents.md", "Contents", "contents"}, {"book-index.md", "Index", "index"}} {
				cur := ""
				if dr.opt.bookPage == l[2] {
					cur = ` aria-current="page"`
				}
				fmt.Fprintf(&b, `<a href="%s"%s>%s</a>`, html.EscapeString(pageHref(d.Path, filepath.Join(g.Dir, l[0]))), cur, l[1])
			}
			if dr.opt.BookPDF != "" {
				r, _ := filepath.Rel(filepath.Dir(d.Path), filepath.Join(g.Dir, dr.opt.BookPDF))
				fmt.Fprintf(&b, `<a class="si-book-pdf" href="%s" download>PDF</a>`, html.EscapeString(filepath.ToSlash(r)))
			}
			b.WriteString(`</nav>`)
		}
		b.WriteString(modeBtn + "</div>\n")
		side := dr.opt.bookPage == ""
		layout := "guide"
		if !side {
			layout = "book-page"
		}
		fmt.Fprintf(&b, "<div class=\"si-doc-layout\" data-layout=\"%s\">\n", layout)
		if side {
			fmt.Fprintf(&b, "<nav class=\"si-doc-nav\" aria-label=\"%s\"><details open><summary>Contents <small class=\"si-doc-progress\"></small></summary>\n%s\n</details></nav>\n",
				html.EscapeString(g.Meta.Title), navTree(g, d.Path, toc))
		}
		b.WriteString("<main class=\"si-doc-body\">\n" + guideCrumb(g, d.Path))
		if dr.opt.Chapter != "" {
			fmt.Fprintf(&b, "<span class=\"si-book-chap\">%s %s</span>\n", html.EscapeString(dr.opt.ChapterKind), html.EscapeString(dr.opt.Chapter))
		}
		if dr.opt.bookPage != "cover" && dr.opt.bookPage != "book" {
			fmt.Fprintf(&b, "<h1 class=\"si-doc-title\">%s</h1>\n%s%s", dr.inline(d.Title()), lede, metaHTML)
		}
		b.WriteString(body + dr.pager())
		if m.EditURL != "" {
			r, _ := filepath.Rel(g.Dir, d.Path)
			fmt.Fprintf(&b, "<p class=\"si-doc-edit\"><a href=\"%s\">Edit this page</a></p>\n", html.EscapeString(strings.TrimRight(m.EditURL, "/")+"/"+filepath.ToSlash(r)))
		}
		if side {
			b.WriteString("</main>\n" + tocHTML + "</div>\n")
		} else {
			b.WriteString("</main>\n</div>\n")
		}
		if m.FooterPage != "off" {
			if f := dr.partial("_footer.md"); f != "" {
				b.WriteString("<div class=\"si-doc-sitefoot\"><div class=\"si-doc-site-in\">" + f + "</div></div>\n")
			}
		}
	} else {
		b.WriteString("<header class=\"si-doc-head\"><div class=\"si-doc-head-in\">\n")
		fmt.Fprintf(&b, "<h1 class=\"si-doc-title\">%s</h1>\n%s%s", dr.inline(d.Title()), lede, metaHTML)
		if d.IsDemo() {
			b.WriteString(demoFacts(m))
		}
		b.WriteString(modeBtn + "</div></header>\n<div class=\"si-doc-layout\">\n")
		if kind == "article" {
			b.WriteString("<main class=\"si-doc-body\">\n" + body + dr.pager() + "</main>\n" + tocHTML + "</div>\n")
		} else {
			b.WriteString(tocHTML + "<main class=\"si-doc-body\">\n" + body + dr.pager() + "</main>\n</div>\n")
		}
	}
	var foot []string
	if c := copyrightLine(m.Year, m.Copyright); c != "" {
		foot = append(foot, "<span>"+html.EscapeString(c)+"</span>")
	}
	if f := strings.TrimSpace(m.Footer); f != "" && f != "off" {
		foot = append(foot, "<span>"+html.EscapeString(f)+"</span>")
	}
	fmt.Fprintf(&b, "<footer class=\"si-doc-foot\">%s</footer>\n", strings.Join(foot, ""))
	b.WriteString(`<div class="si-doc-find" role="dialog" aria-label="Search" hidden><div class="si-doc-find-box"><input class="si-doc-find-input" type="search" placeholder="Search this page and the others…" aria-label="Search" autocomplete="off" spellcheck="false"><ol class="si-doc-find-list" role="listbox"></ol><p class="si-doc-find-hint">↑ ↓ to choose · Enter to open · Esc to close</p></div></div>` + "\n")
	b.WriteString(dr.searchScript())
	dr.problemsBanner(&b, problems)
	dr.mermaidScript(&b)
	b.WriteString("<script>\n" + assets.DocJS + "</script>\n")
	dr.reload(&b)
	b.WriteString("</body>\n</html>\n")
	return b.Bytes()
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if s != "" {
			return s
		}
	}
	return ""
}

// sayCallout turns a demo's `> Say: "…"` quote into a speech callout: what the
// presenter says to the room, verbatim. nil for any other quote.
func sayCallout(bq *ast.Blockquote, src []byte) ast.Node {
	p, ok := bq.FirstChild().(*ast.Paragraph)
	if !ok || !strings.HasPrefix(strings.TrimSpace(textOf(p, src)), "Say:") {
		return nil
	}
	for c := p.FirstChild(); c != nil; c = c.NextSibling() { // drop the "Say:" lead-in
		if t, ok := c.(*ast.Text); ok {
			v := string(t.Segment.Value(src))
			if i := strings.Index(v, "Say:"); i >= 0 {
				start := t.Segment.Start + i + len("Say:")
				for start < t.Segment.Stop && src[start] == ' ' {
					start++
				}
				t.Segment = t.Segment.WithStart(start)
				break
			}
		}
	}
	d := &divNode{class: "si-say"}
	d.AppendChild(d, &rawNode{html: `<p class="si-say-label">Say</p>`})
	for c := bq.FirstChild(); c != nil; {
		next := c.NextSibling()
		bq.RemoveChild(bq, c)
		d.AppendChild(d, c)
		c = next
	}
	return d
}

// demoFacts is the session / slides / duration block under a demo's title.
func demoFacts(m deck.Meta) string {
	var rows []string
	for _, kv := range [][2]string{{"Session", m.Session}, {"Slides", m.SlidesRef}, {"Duration", m.Duration}} {
		if strings.TrimSpace(kv[1]) != "" {
			rows = append(rows, fmt.Sprintf("<div><dt>%s</dt><dd>%s</dd></div>", kv[0], html.EscapeString(kv[1])))
		}
	}
	if len(rows) == 0 {
		return ""
	}
	return `<dl class="si-demo-facts">` + strings.Join(rows, "") + "</dl>\n"
}
