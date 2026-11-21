// Package bookexport turns a book (or a guide, or one article) into files
// other tools read: Markdown, Word (.docx), RTF and EPUB. It walks the pages
// in _nav.md order, reads each page's Markdown (AsciiDoc already converted by
// the deck parser) with goldmark into one small model of blocks, numbers it
// the way the HTML book does (chapters, sections, figures, tables, filled
// cross-references, index terms shown as plain words), and writes every
// format from that model. Standard library and goldmark only: the .docx,
// .rtf and .epub files are written by hand.
package bookexport

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/felipeg48/simple-ideas/internal/codeshow"
	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

// Run is a piece of inline text with one look.
type Run struct {
	Text               string
	Bold, Italic, Code bool
	Link               string // an outside link (http, https, mailto); "" for none
	xref               string // a cross-reference waiting for its label: "file#id"
}

// Kind is the kind of a Block.
type Kind int

// The block kinds.
const (
	Heading     Kind = iota // Level 2-6 (a chapter's title is the Chapter, not a block)
	Para                    // Runs
	Code                    // Lang, Text, Callouts
	List                    // Ordered, Start, Items
	Table                   // Rows (the first is the header), Caption
	Image                   // Src, Alt, Caption
	Admonition              // Label, Blocks
	Quote                   // Blocks
	Rule                    // a thematic break
	Placeholder             // a figure si draws that has no picture here: Runs say what it is; Lang, Text keep the fence
)

// Block is one block of a chapter.
type Block struct {
	Kind     Kind
	Level    int
	Num      string // a numbered section: "1.2"
	ID       string
	Runs     []Run
	Lang     string
	Text     string
	Callouts []Callout
	Ordered  bool
	Start    int
	Items    [][]Block
	Rows     [][][]Run
	Caption  string // "Figure 1.1 · The five stations", "Table 2.1 · Cut-off times"
	Src, Alt string // an image: its file (absolute) or URL, and its alt text
	Label    string // an admonition: Note, Tip, Important, Warning, Caution
	Blocks   []Block
}

// Callout is one explanation under a code block, for its (N) marker.
type Callout struct {
	N    string
	Runs []Run
}

// Chapter is one page of the book: a chapter, an appendix or front matter.
type Chapter struct {
	Part   string // the part heading this page opens ("Part I · Foundations"), if any
	Kind   string // Chapter, Appendix or "" (front matter, a guide page)
	Num    string
	Title  string
	Name   string // file name without extension: per-chapter output files
	Blocks []Block
}

// Heading is the chapter's title as the exports print it: "Chapter 1 · The order journey".
func (c Chapter) Heading() string {
	if c.Num == "" {
		return c.Title
	}
	return c.Kind + " " + c.Num + " · " + c.Title
}

// Book is the whole export: its front matter and chapters in reading order.
type Book struct {
	Title, Subtitle, Author, Edition, Date, Copyright, Year, Lang string
	IsBook                                                        bool
	Colophon                                                      []Block // _cover.md
	Chapters                                                      []Chapter
}

// Byline is the author and date line.
func (b *Book) Byline() string {
	var parts []string
	for _, v := range []string{b.Author, b.Edition, b.Date} {
		if v = strings.TrimSpace(v); v != "" {
			parts = append(parts, v)
		}
	}
	return strings.Join(parts, " · ")
}

// Rights is the copyright line, or "".
func (b *Book) Rights() string {
	if b.Copyright == "" {
		return ""
	}
	if b.Year != "" {
		return "© " + b.Year + " " + b.Copyright
	}
	return "© " + b.Copyright
}

// FromGuide reads a book or a guide: every page of _nav.md in order, the
// sections of _nav.md as parts.
func FromGuide(g *deck.Guide) (*Book, error) {
	b := &Book{Title: g.Meta.Title, Subtitle: g.Meta.Subtitle, Author: g.Meta.Author, Edition: g.Meta.Edition,
		Date: g.Meta.Date, Copyright: g.Meta.Copyright, Year: g.Meta.Year, Lang: g.Meta.Lang, IsBook: g.IsBook()}
	nums, kinds := g.BookNumbers()
	if !b.IsBook {
		nums, kinds = map[string]string{}, map[string]string{}
	}
	c := newConv()
	for _, name := range []string{"_cover.md", "_cover.adoc"} {
		if raw, err := os.ReadFile(filepath.Join(g.Dir, name)); err == nil {
			src := string(raw)
			if strings.HasSuffix(name, ".adoc") {
				src = deck.AdocToMarkdown(src)
			}
			b.Colophon = c.blocks(src, g.Dir, "", "", "")
			break
		}
	}
	part := ""
	var walk func([]*deck.NavItem) error
	walk = func(items []*deck.NavItem) error {
		for _, it := range items {
			if it.File == "" && it.Href == "" {
				part = it.Title
				if err := walk(it.Children); err != nil {
					return err
				}
				continue
			}
			if it.File != "" && it.Sub == "" {
				d, err := deck.ParseWith(it.File, g.Defaults())
				if err != nil {
					return err
				}
				ch := c.chapter(d, nums[it.File], kinds[it.File])
				if ch.Title == "" || ch.Title == d.Name {
					ch.Title = it.Title
				}
				ch.Part, part = part, ""
				b.Chapters = append(b.Chapters, ch)
			}
			if err := walk(it.Children); err != nil {
				return err
			}
		}
		return nil
	}
	if err := walk(g.Items); err != nil {
		return nil, err
	}
	c.resolve(b)
	return b, nil
}

// FromDeck reads one article (any document page) as a book of one chapter.
func FromDeck(d *deck.Deck) *Book {
	b := &Book{Title: d.Title(), Subtitle: d.Meta.Subtitle, Author: d.Meta.Author, Date: d.Meta.Date,
		Copyright: d.Meta.Copyright, Year: d.Meta.Year, Lang: d.Meta.Lang}
	c := newConv()
	ch := c.chapter(d, "", "")
	b.Chapters = []Chapter{ch}
	c.resolve(b)
	return b
}

// conv reads pages into blocks, keeping the labels cross-references point to.
type conv struct {
	md     goldmark.Markdown
	labels map[string]string // "file#id" (or "file") -> "Section 1.2"
	file   string            // the page being read (absolute)
	num    string            // its chapter number
	sec    int
	fig    int
	tbl    int
	slugs  map[string]bool
}

func newConv() *conv {
	return &conv{md: goldmark.New(goldmark.WithExtensions(extension.GFM)), labels: map[string]string{}}
}

func (c *conv) chapter(d *deck.Deck, num, kind string) Chapter {
	ch := Chapter{Kind: kind, Num: num, Title: plainTerms(d.Title()), Name: d.Name}
	c.file, c.num, c.sec, c.fig, c.tbl, c.slugs = d.Path, num, 0, 0, 0, map[string]bool{}
	if num != "" {
		c.labels[d.Path] = kind + " " + num
	}
	title := strings.TrimSpace(d.Title())
	for _, s := range d.Slides {
		body := s.Body
		if s.Title != "" && strings.TrimSpace(s.Title) != title { // a part whose heading the parser took off
			body = "## " + s.Title + "\n\n" + body
		}
		ch.Blocks = append(ch.Blocks, c.blocks(body, s.Dir, d.Path, title, num)...)
		title = "" // only the first H1 equal to the title is the title
	}
	return ch
}

var (
	customIDRE   = regexp.MustCompile(`\s*\{#([A-Za-z][\w-]*)\}\s*$`)
	alertRE      = regexp.MustCompile(`(?i)^\[!(note|tip|important|warning|caution)\]\s*`)
	calloutRE    = regexp.MustCompile(`^<(\d+|\.)>\s+(.+)$`)
	markRE       = regexp.MustCompile(`^(.*?)\s*(?:(?://|#|--|;|/\*)\s*)?((?:<(?:\d+|\.)>\s*)+)(?:\*/\s*)?$|^(.*?)\s*<!--\s*(\d+|\.)\s*-->\s*$`)
	markNumRE    = regexp.MustCompile(`<(\d+|\.)>|\d+|\.`)
	hiddenTermRE = regexp.MustCompile(`\(\(\((.+?)\)\)\)`)
	shownTermRE  = regexp.MustCompile(`\(\((.+?)\)\)`)
)

// drawn are fence languages si draws as a figure: no picture of them here.
var drawn = map[string]string{"chart": "a chart", "scene": "a scene", "stage": "a stage diagram", "mindmap": "a mind map",
	"mermaid": "a Mermaid diagram", "quiz": "a quiz", "video": "a video", "poll": "a poll", "timeline": "a timeline", "flow": "a flow diagram"}

// blocks reads Markdown text into blocks. dir resolves relative image paths;
// title is a page title whose first H1 is dropped; num numbers the sections.
func (c *conv) blocks(src, dir, file, title, num string) []Block {
	s := []byte(src)
	doc := c.md.Parser().Parse(text.NewReader(s))
	var out []Block
	for n := doc.FirstChild(); n != nil; n = n.NextSibling() {
		// a "Table:" caption after a table, and a callout list after a code block
		if p, ok := n.(*ast.Paragraph); ok && len(out) > 0 {
			last := &out[len(out)-1]
			txt := strings.TrimSpace(linesOf(p, s))
			if cap, ok := strings.CutPrefix(txt, "Table:"); ok && last.Kind == Table {
				c.tableCaption(last, strings.TrimSpace(cap))
				continue
			}
			if last.Kind == Code {
				if co, ok := c.callouts(p, s); ok {
					last.Callouts = co
					continue
				}
			}
		}
		if h, ok := n.(*ast.Heading); ok && h.Level == 1 && title != "" && strings.TrimSpace(plainText(h, s)) == title {
			title = ""
			continue
		}
		out = append(out, c.block(n, s, dir, num)...)
	}
	return out
}

func (c *conv) block(n ast.Node, s []byte, dir, num string) []Block {
	switch n := n.(type) {
	case *ast.Heading:
		raw := plainText(n, s)
		id := ""
		if m := customIDRE.FindStringSubmatch(raw); m != nil {
			id = m[1]
		}
		runs := c.inlines(n, s)
		trimID(runs)
		if id == "" {
			id = c.slug(raw)
		} else {
			c.slugs[id] = true
		}
		b := Block{Kind: Heading, Level: max(n.Level, 2), ID: id, Runs: runs}
		if n.Level == 2 && num != "" {
			c.sec++
			b.Num = fmt.Sprintf("%s.%d", num, c.sec)
			c.labels[c.file+"#"+id] = "Section " + b.Num
		}
		return []Block{b}
	case *ast.Paragraph:
		if im, ok := soleImage(n, s); ok {
			return []Block{c.image(im, s, dir, num)}
		}
		if hb := c.inlines(n, s); len(hb) > 0 {
			return []Block{{Kind: Para, Runs: hb}}
		}
		return nil
	case *ast.TextBlock:
		return []Block{{Kind: Para, Runs: c.inlines(n, s)}}
	case *ast.FencedCodeBlock:
		info := ""
		if n.Info != nil {
			info = string(n.Info.Segment.Value(s))
		}
		lang, _ := codeshow.Info(info)
		lang = strings.ToLower(lang)
		body := strings.TrimRight(linesOf(n, s), "\n")
		if what, ok := drawn[lang]; ok {
			return []Block{{Kind: Placeholder, Lang: lang, Text: body, Runs: []Run{{Text: "Figure: " + what + " (in the HTML and PDF editions)", Italic: true}}}}
		}
		return []Block{{Kind: Code, Lang: lang, Text: body}}
	case *ast.CodeBlock:
		return []Block{{Kind: Code, Text: strings.TrimRight(linesOf(n, s), "\n")}}
	case *ast.List:
		b := Block{Kind: List, Ordered: n.IsOrdered(), Start: n.Start}
		for it := n.FirstChild(); it != nil; it = it.NextSibling() {
			var item []Block
			for k := it.FirstChild(); k != nil; k = k.NextSibling() {
				item = append(item, c.block(k, s, dir, num)...)
			}
			b.Items = append(b.Items, item)
		}
		return []Block{b}
	case *ast.Blockquote:
		var inner []Block
		for k := n.FirstChild(); k != nil; k = k.NextSibling() {
			inner = append(inner, c.block(k, s, dir, num)...)
		}
		if len(inner) > 0 && inner[0].Kind == Para && len(inner[0].Runs) > 0 {
			if m := alertRE.FindStringSubmatch(inner[0].Runs[0].Text); m != nil {
				inner[0].Runs[0].Text = strings.TrimLeft(inner[0].Runs[0].Text[len(m[0]):], " \n")
				if inner[0].Runs[0].Text == "" {
					inner[0].Runs = inner[0].Runs[1:]
				}
				if len(inner[0].Runs) == 0 {
					inner = inner[1:]
				}
				return []Block{{Kind: Admonition, Label: strings.ToUpper(m[1][:1]) + strings.ToLower(m[1][1:]), Blocks: inner}}
			}
		}
		return []Block{{Kind: Quote, Blocks: inner}}
	case *east.Table:
		b := Block{Kind: Table}
		for r := n.FirstChild(); r != nil; r = r.NextSibling() {
			var row [][]Run
			for cell := r.FirstChild(); cell != nil; cell = cell.NextSibling() {
				row = append(row, c.inlines(cell, s))
			}
			b.Rows = append(b.Rows, row)
		}
		return []Block{b}
	case *ast.ThematicBreak:
		return []Block{{Kind: Rule}}
	case *ast.HTMLBlock:
		name, val, ok := deck.ParseDirective(strings.TrimSpace(linesOf(n, s)))
		if ok && name == "svg" { // an animated canvas: its file, as a picture
			if f := strings.Fields(val); len(f) > 0 {
				return []Block{{Kind: Image, Src: filepath.Join(dir, filepath.FromSlash(f[0])), Alt: f[0]}}
			}
		}
		return nil // si directives and raw HTML have no place outside the web page
	}
	return nil
}

func (c *conv) image(im *ast.Image, s []byte, dir, num string) Block {
	dest := string(im.Destination)
	src := dest
	if !strings.Contains(dest, "://") && !strings.HasPrefix(dest, "data:") {
		src = filepath.Join(dir, filepath.FromSlash(dest))
	}
	b := Block{Kind: Image, Src: src, Alt: plainText(im, s)}
	if t := plainTerms(string(im.Title)); t != "" {
		id := "fig-" + c.slug(t)
		b.ID, b.Caption = id, t
		if num != "" {
			c.fig++
			l := fmt.Sprintf("Figure %s.%d", num, c.fig)
			b.Caption = l + " · " + t
			c.labels[c.file+"#"+id] = l
		}
	}
	return b
}

func (c *conv) tableCaption(b *Block, t string) {
	t = plainTerms(t)
	b.ID, b.Caption = "tbl-"+c.slug(t), t
	if c.num != "" {
		c.tbl++
		l := fmt.Sprintf("Table %s.%d", c.num, c.tbl)
		b.Caption = l + " · " + t
		c.labels[c.file+"#"+b.ID] = l
	}
}

// callouts reads a paragraph of `<N> text` lines after a code block.
func (c *conv) callouts(p *ast.Paragraph, s []byte) ([]Callout, bool) {
	var out []Callout
	last := 0
	for i := 0; i < p.Lines().Len(); i++ {
		seg := p.Lines().At(i)
		line := strings.TrimSpace(string(seg.Value(s)))
		m := calloutRE.FindStringSubmatch(line)
		switch {
		case m != nil:
			out = append(out, Callout{N: coNum(m[1], &last), Runs: []Run{{Text: m[2]}}})
		case len(out) > 0 && line != "":
			out[len(out)-1].Runs[0].Text += " " + line
		default:
			return nil, false
		}
	}
	for i := range out { // the explanation is Markdown too
		out[i].Runs = c.inlineText(out[i].Runs[0].Text)
	}
	return out, len(out) > 0
}

func (c *conv) inlineText(t string) []Run {
	s := []byte(t)
	doc := c.md.Parser().Parse(text.NewReader(s))
	if p := doc.FirstChild(); p != nil {
		return c.inlines(p, s)
	}
	return []Run{{Text: t}}
}

func coNum(n string, last *int) string {
	if n == "." {
		*last++
	} else {
		*last, _ = strconv.Atoi(n)
	}
	return strconv.Itoa(*last)
}

// CodeLines is a code block's text with its callout markers written as (N):
// `// <1>` becomes `// (1)`, `<!-- 2 -->` becomes `<!-- (2) -->`.
func CodeLines(b Block) string {
	if len(b.Callouts) == 0 {
		return b.Text
	}
	lines := strings.Split(b.Text, "\n")
	last := 0
	for i, l := range lines {
		m := markRE.FindStringSubmatchIndex(l)
		switch {
		case m == nil:
		case m[4] >= 0:
			marks := markNumRE.FindAllStringSubmatch(l[m[4]:m[5]], -1)
			var ns []string
			for _, k := range marks {
				ns = append(ns, "("+coNum(k[1], &last)+")")
			}
			lines[i] = strings.TrimRight(l[:m[4]], " ") + " " + strings.Join(ns, " ") + strings.TrimSpace(l[m[5]:])
			if strings.HasSuffix(strings.TrimSpace(l), "*/") {
				lines[i] = strings.TrimSuffix(lines[i], "*/") + " */"
			}
		case m[8] >= 0:
			lines[i] = l[:m[8]] + "(" + coNum(l[m[8]:m[9]], &last) + ")" + l[m[9]:]
		}
	}
	return strings.Join(lines, "\n")
}

// inlines reads the inline children of n into runs, index terms resolved.
func (c *conv) inlines(n ast.Node, s []byte) []Run {
	var out []Run
	var walk func(n ast.Node, r Run)
	walk = func(n ast.Node, r Run) {
		for k := n.FirstChild(); k != nil; k = k.NextSibling() {
			switch k := k.(type) {
			case *ast.Text:
				t := string(k.Segment.Value(s))
				out = append(out, Run{Text: t, Bold: r.Bold, Italic: r.Italic, Link: r.Link, xref: r.xref})
				if k.HardLineBreak() {
					out = append(out, Run{Text: "\n"})
				} else if k.SoftLineBreak() {
					out = append(out, Run{Text: " ", Bold: r.Bold, Italic: r.Italic, Link: r.Link, xref: r.xref})
				}
			case *ast.String:
				out = append(out, Run{Text: string(k.Value), Bold: r.Bold, Italic: r.Italic, Link: r.Link})
			case *ast.CodeSpan:
				out = append(out, Run{Text: plainText(k, s), Code: true, Bold: r.Bold, Italic: r.Italic, Link: r.Link})
			case *ast.Emphasis:
				e := r
				if k.Level >= 2 {
					e.Bold = true
				} else {
					e.Italic = true
				}
				walk(k, e)
			case *ast.Link:
				e := r
				dest := string(k.Destination)
				if isOutside(dest) {
					e.Link = dest
				} else if k.FirstChild() == nil { // [](#tbl-x): the target's label
					e.xref = c.target(dest)
					out = append(out, Run{Bold: r.Bold, Italic: r.Italic, xref: e.xref})
					continue
				}
				walk(k, e)
			case *ast.AutoLink:
				u := string(k.URL(s))
				link := u
				if k.AutoLinkType == ast.AutoLinkEmail && !strings.HasPrefix(u, "mailto:") {
					link = "mailto:" + u
				}
				out = append(out, Run{Text: u, Link: link})
			case *ast.Image:
				out = append(out, Run{Text: plainText(k, s), Bold: r.Bold, Italic: r.Italic})
			case *ast.RawHTML:
				// inline HTML has no place outside the web page
			default:
				walk(k, r)
			}
		}
	}
	walk(n, Run{})
	return terms(merge(out))
}

// target is a link's "file#id" key in labels.
func (c *conv) target(dest string) string {
	path, frag, _ := strings.Cut(dest, "#")
	file := c.file
	if path != "" {
		file = filepath.Join(filepath.Dir(c.file), filepath.FromSlash(path))
		if ext := filepath.Ext(file); ext == ".html" {
			file = strings.TrimSuffix(file, ext) + ".md"
		}
	}
	if frag == "" {
		return file
	}
	return file + "#" + frag
}

// resolve fills every cross-reference with its label ("Table 1.1").
func (c *conv) resolve(b *Book) {
	fill := func(runs []Run) {
		for i := range runs {
			if k := runs[i].xref; k != "" {
				l := c.labels[k]
				if l == "" { // an .adoc page, or a page without a number
					l = c.labels[strings.TrimSuffix(k, filepath.Ext(k))+".adoc"]
				}
				if l == "" {
					l = "see the linked section"
				}
				runs[i].Text, runs[i].xref = l, ""
			}
		}
	}
	for i := range b.Chapters {
		Walk(b.Chapters[i].Blocks, fill)
	}
	Walk(b.Colophon, fill)
}

// Walk calls f on every run list in blocks, nested ones included.
func Walk(blocks []Block, f func([]Run)) {
	for i := range blocks {
		bl := &blocks[i]
		f(bl.Runs)
		for _, co := range bl.Callouts {
			f(co.Runs)
		}
		for _, it := range bl.Items {
			Walk(it, f)
		}
		for _, row := range bl.Rows {
			for _, cell := range row {
				f(cell)
			}
		}
		Walk(bl.Blocks, f)
	}
}

func isOutside(dest string) bool {
	return strings.Contains(dest, "://") || strings.HasPrefix(dest, "mailto:")
}

// merge joins neighbouring runs of the same look.
func merge(runs []Run) []Run {
	var out []Run
	for _, r := range runs {
		if n := len(out); n > 0 && r.xref == "" && out[n-1].xref == "" && out[n-1].Bold == r.Bold && out[n-1].Italic == r.Italic &&
			out[n-1].Code == r.Code && out[n-1].Link == r.Link && r.Text != "\n" && out[n-1].Text != "\n" {
			out[n-1].Text += r.Text
			continue
		}
		out = append(out, r)
	}
	return out
}

// terms writes index terms as the book shows them: ((term)) is the word,
// ((term, sub)) the sub-entry, (((hidden))) nothing.
func terms(runs []Run) []Run {
	out := runs[:0]
	for _, r := range runs {
		if !r.Code {
			r.Text = plainTerms(r.Text)
		}
		if r.Text != "" || r.xref != "" {
			out = append(out, r)
		}
	}
	return out
}

func plainTerms(t string) string {
	if !strings.Contains(t, "((") {
		return t
	}
	t = hiddenTermRE.ReplaceAllString(t, "")
	t = shownTermRE.ReplaceAllStringFunc(t, func(m string) string {
		v := shownTermRE.FindStringSubmatch(m)[1]
		if i := strings.LastIndex(v, ","); i >= 0 {
			v = v[i+1:]
		}
		return strings.TrimSpace(v)
	})
	for strings.Contains(t, "  ") {
		t = strings.ReplaceAll(t, "  ", " ")
	}
	return t
}

func trimID(runs []Run) {
	if n := len(runs); n > 0 {
		runs[n-1].Text = customIDRE.ReplaceAllString(runs[n-1].Text, "")
	}
}

// slug is a heading's id, GitHub style, as the HTML book makes it.
func (c *conv) slug(t string) string {
	t = plainTerms(customIDRE.ReplaceAllString(t, ""))
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
	for n := 2; c.slugs[id]; n++ {
		id = fmt.Sprintf("%s-%d", base, n)
	}
	c.slugs[id] = true
	return id
}

func soleImage(p ast.Node, s []byte) (*ast.Image, bool) {
	var im *ast.Image
	for k := p.FirstChild(); k != nil; k = k.NextSibling() {
		switch k := k.(type) {
		case *ast.Image:
			if im != nil {
				return nil, false
			}
			im = k
		case *ast.Text:
			if strings.TrimSpace(string(k.Segment.Value(s))) != "" {
				return nil, false
			}
		default:
			return nil, false
		}
	}
	return im, im != nil
}

func linesOf(n ast.Node, s []byte) string {
	var b strings.Builder
	for i := 0; i < n.Lines().Len(); i++ {
		seg := n.Lines().At(i)
		b.Write(seg.Value(s))
	}
	return b.String()
}

// plainText is the text of n's inline children.
func plainText(n ast.Node, s []byte) string {
	var b strings.Builder
	_ = ast.Walk(n, func(k ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch k := k.(type) {
		case *ast.Text:
			b.Write(k.Segment.Value(s))
			if k.SoftLineBreak() {
				b.WriteByte(' ')
			}
		case *ast.String:
			b.Write(k.Value)
		}
		return ast.WalkContinue, nil
	})
	return b.String()
}
