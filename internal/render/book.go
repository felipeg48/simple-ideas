package render

import (
	"fmt"
	"html"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

// Books: a guide whose _nav.md says `type: book`. RenderGuide numbers its
// chapters (Options.Chapter), the page renderer numbers sections, figures and
// tables inside each chapter, and a whole-book pass then fills cross-references,
// builds the index from ((terms)) and writes the generated pages: the cover
// (index.html), the contents (contents.html), the index (book-index.html) and
// the whole book in one file (book.html).

func (dr *deckRenderer) label(id, l string) {
	if dr.labels == nil {
		dr.labels = map[string]string{}
	}
	dr.labels[id] = l
}

var figTitleRE = regexp.MustCompile(`<figcaption class="si-caption">([^<]*)</figcaption>`)

// numberFigure gives a captioned figure an id (fig-<caption slug>) and, in a
// book chapter, its number: "Figure 3.2 · caption".
func (dr *deckRenderer) numberFigure(fig *rawNode, title string, sl slugger) {
	id := sl.slug("fig " + title)
	num := ""
	if dr.opt.Chapter != "" {
		dr.fig++
		num = fmt.Sprintf("%s.%d", dr.opt.Chapter, dr.fig)
		dr.label(id, "Figure "+num)
	}
	fig.html = figTitleRE.ReplaceAllStringFunc(fig.html, func(m string) string {
		t := figTitleRE.FindStringSubmatch(m)[1]
		if num != "" {
			t = `<b class="si-fig-num">Figure ` + num + `</b> · ` + t
		}
		return `<figcaption class="si-caption">` + t + `</figcaption>`
	})
	if i := strings.Index(fig.html, "<figure"); i >= 0 {
		fig.html = fig.html[:i] + `<figure id="` + id + `"` + fig.html[i+len("<figure"):]
	}
}

// tableCaption is the caption of a markdown table (a "Table: …" paragraph
// right after it): an id (tbl-<slug>) and, in a book chapter, "Table 3.1".
func (dr *deckRenderer) tableCaption(text string, sl slugger) string {
	id := sl.slug("tbl " + text)
	num := ""
	if dr.opt.Chapter != "" {
		dr.tbl++
		num = fmt.Sprintf("%s.%d", dr.opt.Chapter, dr.tbl)
		dr.label(id, "Table "+num)
		num = `<b class="si-fig-num">Table ` + num + `</b> · `
	}
	return fmt.Sprintf(`<p class="si-table-cap" id="%s">%s%s</p>`, id, num, dr.inline(text))
}

// ---------- the whole-book pass ----------

type ixEntry struct {
	term, sub string
	href      string // page rel + #anchor
	loc       string // "3.2", "3", or the page title
}

var (
	ixHiddenRE = regexp.MustCompile(`\(\(\((.+?)\)\)\)`)
	ixShownRE  = regexp.MustCompile(`\(\((.+?)\)\)`)
	htmlPartRE = regexp.MustCompile(`<[^>]+>|[^<]+`)
	h2OpenRE   = regexp.MustCompile(`^<h2 id="([^"]+)"`)
	emptyARE   = regexp.MustCompile(`<a href="([^"]+)"></a>`)
)

// indexTerms replaces ((term)) (shown and indexed) and (((term))) (indexed
// only) in the text of a page's HTML, outside code, and returns the entries.
// "((a, b))" is the sub-entry b of a; the text shows b.
func indexTerms(h, rel, chapterLoc string, labels map[string]string, n *int) (string, []ixEntry) {
	var out strings.Builder
	var entries []ixEntry
	skip := 0 // depth inside pre / code / script / style / title / textarea
	loc := chapterLoc
	add := func(raw string, shown bool) string {
		*n++
		id := fmt.Sprintf("ix-%d", *n)
		term, sub, _ := strings.Cut(html.UnescapeString(raw), ",")
		term, sub = strings.TrimSpace(term), strings.TrimSpace(sub)
		entries = append(entries, ixEntry{term: term, sub: sub, href: rel + "#" + id, loc: loc})
		if !shown {
			return `<span class="si-ix" id="` + id + `"></span>`
		}
		vis := term
		if sub != "" {
			vis = sub
		}
		return `<span class="si-term" id="` + id + `">` + html.EscapeString(vis) + `</span>`
	}
	for _, t := range htmlPartRE.FindAllString(h, -1) {
		if strings.HasPrefix(t, "<") {
			low := strings.ToLower(t)
			for _, tag := range []string{"pre", "code", "script", "style", "title", "textarea"} {
				if strings.HasPrefix(low, "<"+tag) && (len(low) == len(tag)+2 || low[len(tag)+1] == ' ' || low[len(tag)+1] == '>') {
					skip++
				} else if strings.HasPrefix(low, "</"+tag+">") {
					skip--
				}
			}
			if m := h2OpenRE.FindStringSubmatch(t); m != nil {
				if l, ok := labels[m[1]]; ok {
					loc = strings.TrimPrefix(l, "Section ")
				}
			}
			out.WriteString(t)
			continue
		}
		if skip > 0 || !strings.Contains(t, "((") {
			out.WriteString(t)
			continue
		}
		t = ixHiddenRE.ReplaceAllStringFunc(t, func(m string) string { return add(ixHiddenRE.FindStringSubmatch(m)[1], false) })
		t = ixShownRE.ReplaceAllStringFunc(t, func(m string) string { return add(ixShownRE.FindStringSubmatch(m)[1], true) })
		out.WriteString(t)
	}
	return out.String(), entries
}

// bookLinks fills empty links ([](#fig-flow), [](03-queues.md#quorum-queues),
// [](03-queues.md)) with the target's label; a target without one is a problem.
func bookLinks(h, rel string, labels map[string]string, chapters map[string]string, file string, probs *[]deck.Problem) string {
	return emptyARE.ReplaceAllStringFunc(h, func(m string) string {
		href := html.UnescapeString(emptyARE.FindStringSubmatch(m)[1])
		if strings.Contains(href, ":") {
			return m
		}
		path, frag, _ := strings.Cut(href, "#")
		target := rel
		if path != "" {
			target = filepath.ToSlash(filepath.Clean(filepath.Join(filepath.Dir(rel), path)))
		}
		l := ""
		if frag != "" {
			l = labels[target+"#"+frag]
		} else {
			l = chapters[target]
		}
		if l == "" { // reported once, and shown as plain text so the link checks do not report it again
			*probs = append(*probs, deck.Problem{File: deck.Rel(file), Msg: fmt.Sprintf("cross-reference %s: no numbered section, figure, table or chapter there", href)})
			return `<span class="si-xref">` + html.EscapeString(href) + `</span>`
		}
		return fmt.Sprintf(`<a href="%s" class="si-xref">%s</a>`, html.EscapeString(href), html.EscapeString(l))
	})
}

// finishBook runs the whole-book pass over rendered pages and appends the
// generated ones.
func finishBook(g *deck.Guide, opt Options, pages []GuidePage, probs *[]deck.Problem) ([]GuidePage, []ixEntry) {
	nums, kinds := g.BookNumbers()
	labels, chapters := map[string]string{}, map[string]string{}
	for _, p := range pages {
		for id, l := range p.Result.labels {
			labels[p.Rel+"#"+id] = l
		}
		if n := nums[p.File]; n != "" {
			chapters[p.Rel] = kinds[p.File] + " " + n
		}
	}
	var entries []ixEntry
	ixN := 0
	for i := range pages {
		p := &pages[i]
		own := map[string]string{}
		for id, l := range p.Result.labels {
			own[id] = l
		}
		loc := nums[p.File]
		if loc == "" {
			loc = plain(p.Deck.Title())
		}
		page, es := indexTerms(string(p.Result.HTML), p.Rel, loc, own, &ixN)
		ixN -= len(es) // the body pass below numbers the same occurrences the same way
		body, _ := indexTerms(p.Result.body, p.Rel, loc, own, &ixN)
		entries = append(entries, es...)
		page = bookLinks(page, p.Rel, labels, chapters, p.File, probs)
		body = bookLinks(body, p.Rel, labels, chapters, p.File, new([]deck.Problem))
		p.Result.HTML, p.Result.body = []byte(page), body
	}
	gen := func(rel, title, kind, body string) GuidePage {
		o := opt
		o.Guide, o.bookPage, o.Chapter = g, kind, ""
		d := &deck.Deck{Path: filepath.Join(g.Dir, strings.TrimSuffix(rel, ".html")+".md"), Name: strings.TrimSuffix(rel, ".html"), Meta: g.Defaults()}
		d.Meta.Title = title
		dr := &deckRenderer{d: d, opt: o, doc: true, lastState: map[string][]string{}, words: map[string]bool{}, files: map[string]string{}}
		dr.md = dr.markdown()
		dr.resolveTheme()
		if kind == "cover" || kind == "book" {
			body = dr.coverHTML(pages, nums, kind == "book") + body
		}
		return GuidePage{File: d.Path, Rel: rel, Deck: d, Result: Result{HTML: dr.docPage(body, nil, nil), body: body}}
	}
	contents := contentsHTML(g, pages, nums, kinds, "")
	index := indexHTML(entries, "")
	out := append(pages,
		gen("index.html", g.Meta.Title, "cover", ""),
		gen("contents.html", "Contents", "contents", contents),
		gen("book-index.html", "Index", "index", index))
	out = append(out, gen("book.html", g.Meta.Title, "book", wholeBook(g, pages, nums, kinds, entries)))
	return out, entries
}

// coverHTML is the cover: kind and edition, title, subtitle, byline, the way in.
func (dr *deckRenderer) coverHTML(pages []GuidePage, nums map[string]string, oneFile bool) string {
	g, m := dr.opt.Guide, dr.opt.Guide.Meta
	var b strings.Builder
	b.WriteString(`<section class="si-book-cover"><div class="si-book-plate">`)
	kind := "Book"
	if m.Edition != "" {
		kind += " · " + m.Edition
	}
	fmt.Fprintf(&b, `<span class="si-book-kind">%s</span><h1 class="si-doc-title">%s</h1>`, html.EscapeString(kind), dr.inline(m.Title))
	if m.Subtitle != "" {
		fmt.Fprintf(&b, `<p class="si-lede">%s</p>`, dr.inline(m.Subtitle))
	}
	var by []string
	for _, v := range []string{m.Author, m.Date} {
		for _, part := range strings.Split(v, "·") {
			if part = strings.TrimSpace(part); part != "" {
				by = append(by, "<span>"+html.EscapeString(part)+"</span>")
			}
		}
	}
	if len(by) > 0 {
		b.WriteString(`<p class="si-doc-meta">` + strings.Join(by, "") + `</p>`)
	}
	if oneFile { // the one-file book (and its PDF): no links out of the page
		b.WriteString(`<p class="si-book-start"><a class="si-book-pri" href="#si-book-contents">Contents</a></p></div>`)
	} else {
		b.WriteString(`<p class="si-book-start"><a class="si-book-pri" href="contents.html">Contents</a>`)
		for _, p := range pages {
			if nums[p.File] != "" {
				fmt.Fprintf(&b, `<a href="%s">Start reading: %s</a>`, html.EscapeString(p.Rel), html.EscapeString(plain(p.Deck.Title())))
				break
			}
		}
		if dr.opt.BookPDF != "" {
			fmt.Fprintf(&b, `<a href="%s" download>Download the PDF</a>`, html.EscapeString(dr.opt.BookPDF))
		}
		b.WriteString(`<a href="book.html">The whole book, one page</a></p></div>`)
	}
	if c := dr.partial("_cover.md"); c != "" {
		b.WriteString(`<div class="si-book-colophon">` + c + `</div>`)
	}
	b.WriteString(`</section>`)
	_ = g
	return b.String()
}

// contentsHTML lists the book: front matter, parts with numbered chapters and
// their numbered sections, appendices, the index. prefix turns page hrefs into
// in-page anchors for the one-file book ("" keeps page links).
func contentsHTML(g *deck.Guide, pages []GuidePage, nums, kinds map[string]string, prefix string) string {
	byFile := map[string]*GuidePage{}
	for i := range pages {
		byFile[pages[i].File] = &pages[i]
	}
	href := func(p *GuidePage, frag string) string {
		if prefix != "" {
			if frag == "" {
				return "#" + chapterAnchor(p.Rel)
			}
			return "#" + chapterAnchor(p.Rel) + "-" + frag
		}
		if frag != "" {
			return p.Rel + "#" + frag
		}
		return p.Rel
	}
	var b strings.Builder
	chapter := func(it *deck.NavItem) {
		p := byFile[it.File]
		if p == nil {
			return
		}
		n := nums[it.File]
		fmt.Fprintf(&b, `<div class="si-book-ch"><span class="si-book-num">%s</span><a class="si-book-ct" href="%s">%s</a><span class="si-book-time">%d min</span>`,
			html.EscapeString(n), html.EscapeString(href(p, "")), html.EscapeString(it.Title), readingTime(p.Result.body))
		var secs []string
		for _, e := range p.Result.toc {
			if e.level == 2 {
				secs = append(secs, fmt.Sprintf(`<li><a href="%s">%s</a></li>`, html.EscapeString(href(p, e.id)), e.text))
			}
		}
		if len(secs) > 0 {
			b.WriteString(`<ol>` + strings.Join(secs, "") + `</ol>`)
		}
		b.WriteString(`</div>`)
	}
	var walk func(items []*deck.NavItem)
	walk = func(items []*deck.NavItem) {
		for _, it := range items {
			switch {
			case it.File == "" && it.Href == "":
				fmt.Fprintf(&b, `<div class="si-book-part"><h2>%s</h2>`, html.EscapeString(it.Title))
				walk(it.Children)
				b.WriteString(`</div>`)
			case it.File != "":
				chapter(it)
				walk(it.Children)
			}
		}
	}
	b.WriteString(`<div class="si-book-contents">`)
	walk(g.Items)
	ix := "book-index.html"
	if prefix != "" {
		ix = "#si-book-index"
	}
	fmt.Fprintf(&b, `<div class="si-book-ch"><span class="si-book-num">⁂</span><a class="si-book-ct" href="%s">Index</a></div></div>`, ix)
	return b.String()
}

// indexHTML is the index: an A–Z bar, then terms (with sub-entries) and their
// locators, linked to each occurrence.
func indexHTML(entries []ixEntry, prefix string) string {
	type locs struct {
		subs  map[string][]ixEntry
		top   []ixEntry
		label string
	}
	terms := map[string]*locs{}
	for _, e := range entries {
		k := strings.ToLower(e.term)
		t := terms[k]
		if t == nil {
			t = &locs{subs: map[string][]ixEntry{}, label: e.term}
			terms[k] = t
		}
		if e.sub != "" {
			t.subs[e.sub] = append(t.subs[e.sub], e)
		} else {
			t.top = append(t.top, e)
		}
	}
	keys := make([]string, 0, len(terms))
	for k := range terms {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	link := func(es []ixEntry) string {
		var parts []string
		seen := map[string]bool{}
		for _, e := range es {
			if seen[e.loc] {
				continue
			}
			seen[e.loc] = true
			h := e.href
			if prefix != "" {
				rel, frag, _ := strings.Cut(e.href, "#")
				h = "#" + chapterAnchor(rel) + "-" + frag
			}
			parts = append(parts, fmt.Sprintf(`<a href="%s">%s</a>`, html.EscapeString(h), html.EscapeString(e.loc)))
		}
		return strings.Join(parts, "")
	}
	letterOf := func(k string) string {
		for _, r := range k {
			if unicode.IsLetter(r) {
				return strings.ToUpper(string(r))
			}
			return "#"
		}
		return "#"
	}
	var b strings.Builder
	fmt.Fprintf(&b, `<p class="si-book-ixnote">%d terms. Each number is a section: 2.3 is chapter 2, section 3.</p>`, len(keys))
	have := map[string]bool{}
	for _, k := range keys {
		have[letterOf(k)] = true
	}
	b.WriteString(`<nav class="si-book-az" aria-label="Letters">`)
	for _, l := range strings.Split("#ABCDEFGHIJKLMNOPQRSTUVWXYZ", "") {
		if have[l] {
			fmt.Fprintf(&b, `<a href="#ix-%s">%s</a>`, strings.ToLower(l), l)
		} else if l != "#" {
			fmt.Fprintf(&b, `<span>%s</span>`, l)
		}
	}
	b.WriteString(`</nav><div class="si-book-ix">`)
	cur := ""
	for _, k := range keys {
		if l := letterOf(k); l != cur {
			if cur != "" {
				b.WriteString(`</ul></div>`)
			}
			cur = l
			fmt.Fprintf(&b, `<div class="si-book-letter" id="ix-%s"><h2>%s</h2><ul>`, strings.ToLower(l), l)
		}
		t := terms[k]
		fmt.Fprintf(&b, `<li><span>%s</span><span class="si-book-loc">%s</span></li>`, html.EscapeString(t.label), link(t.top))
		subs := make([]string, 0, len(t.subs))
		for s := range t.subs {
			subs = append(subs, s)
		}
		sort.Strings(subs)
		for _, s := range subs {
			fmt.Fprintf(&b, `<li class="si-book-sub"><span>%s</span><span class="si-book-loc">%s</span></li>`, html.EscapeString(s), link(t.subs[s]))
		}
	}
	if cur != "" {
		b.WriteString(`</ul></div>`)
	}
	b.WriteString(`</div>`)
	return b.String()
}

// chapterAnchor is a page's anchor in the one-file book.
func chapterAnchor(rel string) string {
	return "c-" + strings.NewReplacer("/", "-", ".html", "").Replace(rel)
}

var (
	bookIDRE   = regexp.MustCompile(`\sid="([^"]+)"`)
	bookHrefRE = regexp.MustCompile(`\shref="([^"]+)"`)
)

// wholeBook is the one-file edition: cover, contents, every chapter (ids made
// unique by chapter, links between pages turned into in-page anchors) and the
// index. It prints as the book's PDF.
func wholeBook(g *deck.Guide, pages []GuidePage, nums, kinds map[string]string, entries []ixEntry) string {
	rels := map[string]bool{}
	for _, p := range pages {
		rels[p.Rel] = true
	}
	var b strings.Builder
	b.WriteString(`<section class="si-book-part-page" id="si-book-contents"><h1>Contents</h1>` + contentsHTML(g, pages, nums, kinds, "#") + `</section>`)
	for _, p := range pages {
		a := chapterAnchor(p.Rel)
		body := bookIDRE.ReplaceAllStringFunc(p.Result.body, func(m string) string {
			return ` id="` + a + "-" + bookIDRE.FindStringSubmatch(m)[1] + `"`
		})
		body = bookHrefRE.ReplaceAllStringFunc(body, func(m string) string {
			h := html.UnescapeString(bookHrefRE.FindStringSubmatch(m)[1])
			if strings.Contains(h, ":") || strings.HasPrefix(h, "/") {
				return m
			}
			path, frag, _ := strings.Cut(h, "#")
			target := p.Rel
			if path != "" {
				target = filepath.ToSlash(filepath.Clean(filepath.Join(filepath.Dir(p.Rel), path)))
			}
			switch {
			case target == "contents.html":
				return ` href="#si-book-contents"`
			case target == "book-index.html":
				return ` href="#si-book-index"`
			case !rels[target]:
				return m
			case frag == "":
				return ` href="#` + chapterAnchor(target) + `"`
			default:
				return ` href="#` + chapterAnchor(target) + "-" + html.EscapeString(frag) + `"`
			}
		})
		head := ""
		if n := nums[p.File]; n != "" {
			head = `<span class="si-book-chap">` + html.EscapeString(kinds[p.File]+" "+n) + `</span>`
		}
		fmt.Fprintf(&b, `<section class="si-book-chapter" id="%s">%s<h1 class="si-doc-title">%s</h1>%s</section>`, a, head, html.EscapeString(plain(p.Deck.Title())), body)
	}
	b.WriteString(`<section class="si-book-chapter" id="si-book-index"><h1 class="si-doc-title">Index</h1>` + indexHTML(entries, "#") + `</section>`)
	return b.String()
}
