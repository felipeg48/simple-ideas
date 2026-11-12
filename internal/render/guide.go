package render

import (
	"fmt"
	"html"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

// pageHref is the link from the page file `from` to the document file `to`,
// relative and with the .html extension the build gives it.
func pageHref(from, to string) string {
	r, err := filepath.Rel(filepath.Dir(from), to)
	if err != nil {
		r = filepath.Base(to)
	}
	return filepath.ToSlash(strings.TrimSuffix(r, filepath.Ext(r)) + ".html")
}

// guideRel is a file's path inside the guide, as built (.html, slashes).
func guideRel(g *deck.Guide, file string) string {
	r, err := filepath.Rel(g.Dir, file)
	if err != nil {
		r = filepath.Base(file)
	}
	return filepath.ToSlash(strings.TrimSuffix(r, filepath.Ext(r)) + ".html")
}

// navTree renders the guide sidebar for the page `page`: sections as headings,
// pages as links (the current one marked), each link carrying its path in the
// guide (data-page) so doc.js can mark the pages already read.
func navTree(g *deck.Guide, page string, toc []tocEntry) string {
	nums := map[string]string{}
	if g.IsBook() {
		nums, _ = g.BookNumbers()
	}
	var b strings.Builder
	var walk func(items []*deck.NavItem, depth int)
	walk = func(items []*deck.NavItem, depth int) {
		b.WriteString("<ul>")
		for _, it := range items {
			switch {
			case it.File != "":
				cur := ""
				if it.File == page {
					cur = ` aria-current="page"`
				}
				mark := `<span class="si-nav-mark" aria-hidden="true"></span>`
				if g.IsBook() {
					mark = `<span class="si-nav-num">` + html.EscapeString(nums[it.File]) + `</span>`
				}
				fmt.Fprintf(&b, `<li><a href="%s" data-page="%s"%s>%s%s</a>`,
					html.EscapeString(pageHref(page, it.File)), html.EscapeString(guideRel(g, it.File)), cur, mark, html.EscapeString(it.Title))
				if g.IsBook() && it.File == page { // the chapter being read lists its sections
					var secs []string
					for _, e := range toc {
						if e.level == 2 {
							num, rest, ok := strings.Cut(e.text, " ")
							if !ok || !strings.Contains(num, ".") {
								num, rest = "", e.text
							}
							secs = append(secs, fmt.Sprintf(`<li><a href="#%s"><span class="si-nav-num">%s</span>%s</a></li>`, e.id, num, rest))
						}
					}
					if len(secs) > 0 {
						b.WriteString(`<ul class="si-nav-secs">` + strings.Join(secs, "") + `</ul>`)
					}
				}
			case it.Href != "":
				fmt.Fprintf(&b, `<li><a href="%s" class="si-nav-ext">%s</a>`, html.EscapeString(it.Href), html.EscapeString(it.Title))
			default:
				fmt.Fprintf(&b, `<li class="si-nav-sec"><span>%s</span>`, html.EscapeString(it.Title))
			}
			if len(it.Children) > 0 {
				walk(it.Children, depth+1)
			}
			b.WriteString("</li>")
		}
		b.WriteString("</ul>")
	}
	walk(g.Items, 0)
	return b.String()
}

// guideCrumb is the breadcrumb of a page: guide › sections › page.
func guideCrumb(g *deck.Guide, page string) string {
	it := g.Item(page)
	if it == nil {
		return ""
	}
	var chain []string
	for p := it.Parent; p != nil; p = p.Parent {
		chain = append([]string{html.EscapeString(p.Title)}, chain...)
	}
	parts := []string{fmt.Sprintf(`<a href="%s">%s</a>`, html.EscapeString(pageHref(page, g.Pages[0])), html.EscapeString(g.Meta.Title))}
	for _, c := range chain {
		parts = append(parts, "<span>"+c+"</span>")
	}
	parts = append(parts, `<span aria-current="page">`+html.EscapeString(it.Title)+"</span>")
	return `<nav class="si-doc-crumb" aria-label="Breadcrumb">` + strings.Join(parts, `<i aria-hidden="true">›</i>`) + "</nav>\n"
}

// guidePager sets dr.opt.Prev / Next from the nav order.
func (dr *deckRenderer) guidePager() {
	g, page := dr.opt.Guide, dr.d.Path
	dr.opt.Prev, dr.opt.Next = DocLink{}, DocLink{}
	for i, p := range g.Pages {
		if p != page {
			continue
		}
		if i > 0 {
			dr.opt.Prev = DocLink{Href: pageHref(page, g.Pages[i-1]), Title: g.Item(g.Pages[i-1]).Title}
		}
		if i < len(g.Pages)-1 {
			dr.opt.Next = DocLink{Href: pageHref(page, g.Pages[i+1]), Title: g.Item(g.Pages[i+1]).Title}
		}
	}
}

var partialRefRE = regexp.MustCompile(`\b(href|src)="([^"]*)"`)

// partial renders the guide's _header.md / _footer.md for this page: markdown
// relative to the guide folder, its relative links and images rebased so they
// resolve from the page (which may sit in a subfolder). "" when absent.
func (dr *deckRenderer) partial(name string) string {
	g := dr.opt.Guide
	path := filepath.Join(g.Dir, name)
	src, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	empty := ""
	h, _ := dr.docBody(&deck.Slide{Body: string(src), Dir: g.Dir, File: path, Line: 1}, newSlugger(), &empty)
	pageDir := filepath.Dir(dr.d.Path)
	return partialRefRE.ReplaceAllStringFunc(h, func(m string) string {
		sm := partialRefRE.FindStringSubmatch(m)
		v := sm[2]
		if v == "" || strings.HasPrefix(v, "#") || strings.HasPrefix(v, "/") || strings.Contains(v, ":") {
			return m
		}
		target := filepath.Join(g.Dir, filepath.FromSlash(html.UnescapeString(v)))
		r, err := filepath.Rel(pageDir, target)
		if err != nil {
			return m
		}
		return fmt.Sprintf(`%s="%s"`, sm[1], html.EscapeString(filepath.ToSlash(r)))
	})
}

// readingTime is the article's reading time in minutes (230 words a minute,
// at least 1), counted on the rendered text outside code blocks.
func readingTime(body string) int {
	noCode := regexp.MustCompile(`(?s)<pre.*?</pre>`).ReplaceAllString(body, " ")
	words := len(strings.Fields(html.UnescapeString(tagRE.ReplaceAllString(noCode, " "))))
	return int(math.Max(1, math.Ceil(float64(words)/230)))
}

var stepRE = regexp.MustCompile(`^(?:Step\s+)?(\d{1,2})[:.]\s+(\S.*)$`)

// stepHeading turns "Step 3: Pay" or "3. Pay" (an h2 of a lab or guide) into a
// numbered step badge followed by the rest of the heading.
func stepHeading(inner string) string {
	m := stepRE.FindStringSubmatch(inner)
	if m == nil {
		return inner
	}
	return `<span class="si-step-num">` + m[1] + `</span>` + m[2]
}

// GuidePage is one rendered page of a guide.
type GuidePage struct {
	File   string // the source file
	Rel    string // its output path inside the guide (.html, slashes)
	Deck   *deck.Deck
	Result Result
}

// RenderGuide renders every page of g in nav order (each page starts from the
// guide's front matter) and checks the links between them: a link to a page
// that is not in the guide, or to an #anchor the target page does not have,
// is a problem. Pages under the folder that _nav.md does not list are
// warnings. opt.Guide is set per page.
func RenderGuide(g *deck.Guide, opt Options) (pages []GuidePage, problems, warnings []deck.Problem) {
	problems = append(problems, g.Problems...)
	warnings = append(warnings, g.Warnings...)
	opt.Guide = g
	nums, kinds := g.BookNumbers()
	// renderPage builds one page of the guide. Listed and unlisted pages both go
	// through it, so an unlisted page gets the identical render as a listed one
	// (for a site: the navigator, _header.md, _footer.md and the si-doc body
	// wrapper); it only stays out of the navigator menu.
	renderPage := func(f string, d *deck.Deck) {
		opt.AssetPrefix = strings.Repeat("../", strings.Count(guideRel(g, f), "/"))
		res := Deck(d, opt)
		problems = append(problems, res.Problems...)
		warnings = append(warnings, res.Warnings...)
		pages = append(pages, GuidePage{File: f, Rel: guideRel(g, f), Deck: d, Result: res})
	}
	for _, f := range g.Pages {
		d, err := deck.ParseWith(f, g.Defaults())
		if err != nil {
			problems = append(problems, deck.Problem{File: deck.Rel(f), Msg: err.Error()})
			continue
		}
		opt.Chapter, opt.ChapterKind = "", ""
		if g.IsBook() {
			opt.Chapter, opt.ChapterKind = nums[f], kinds[f]
			if guideRel(g, f) == "index.html" {
				problems = append(problems, deck.Problem{File: deck.Rel(f), Msg: "a book's index.html is its cover: name this page preface.md (or anything but index.md)"})
			}
		}
		renderPage(f, d)
	}
	var terms []ixEntry
	if g.IsBook() {
		opt.Chapter, opt.ChapterKind = "", ""
		pages, terms = finishBook(g, opt, pages, &problems)
	}
	// A page the nav list does not mention is normally an orphan: a warning,
	// not built. One that says `unlisted: true` is built through renderPage, the
	// same path as a listed page, so it carries the full site shell (navigator,
	// _header.md, _footer.md, the si-doc body wrapper and theme); it is only kept
	// out of the navigator menu (g.Items never holds it) and raises no orphan
	// warning. Its cta/footer/body links resolve in si check like any page's.
	opt.Chapter, opt.ChapterKind = "", ""
	for _, f := range g.Unlisted() {
		if d, err := deck.ParseWith(f, g.Defaults()); err == nil && d.Meta.Unlisted == "true" {
			renderPage(f, d)
			continue
		}
		warnings = append(warnings, deck.Problem{File: deck.Rel(f), Msg: "not listed in " + deck.NavFile + ": it is not built"})
	}
	fillGuideSearch(g, pages, terms)
	var subs []string // links into a sub-guide, book or site are checked when that one is built
	var walk func([]*deck.NavItem)
	walk = func(its []*deck.NavItem) {
		for _, it := range its {
			if it.Sub != "" {
				if r, err := filepath.Rel(g.Dir, it.Sub); err == nil {
					subs = append(subs, filepath.ToSlash(r)+"/")
				}
			}
			walk(it.Children)
		}
	}
	walk(g.Items)
	problems = append(problems, checkGuideLinks(pages, subs)...)
	return pages, problems, warnings
}

var (
	slideHashRE = regexp.MustCompile(`^\d+(?:\.\d+)?$`)
	idAttrRE    = regexp.MustCompile(`\sid="([^"]+)"`)
	hrefRE      = regexp.MustCompile(`<a\b[^>]*\shref="([^"]+)"`)
)

// checkGuideLinks finds links between the pages of a guide that point at a
// missing page or a missing #anchor. Links that leave the guide folder, and
// links to other files (images, PDFs, outside URLs), are not checked here.
func checkGuideLinks(pages []GuidePage, skip []string) []deck.Problem {
	ids := map[string]map[string]bool{}
	decks := map[string]bool{} // a deck's slides are #3 and #3.2: the runtime reads them from the hash
	for _, p := range pages {
		decks[p.Rel] = p.Deck != nil && !p.Deck.IsDoc()
		set := map[string]bool{}
		for _, m := range idAttrRE.FindAllStringSubmatch(string(p.Result.HTML), -1) {
			set[html.UnescapeString(m[1])] = true
		}
		ids[p.Rel] = set
	}
	var out []deck.Problem
	for _, p := range pages {
		seen := map[string]bool{}
		for _, m := range hrefRE.FindAllStringSubmatch(string(p.Result.HTML), -1) {
			href := html.UnescapeString(m[1])
			if seen[href] || strings.Contains(href, ":") || strings.HasPrefix(href, "/") {
				continue
			}
			seen[href] = true
			path, frag, _ := strings.Cut(href, "#")
			target := p.Rel
			if path != "" {
				if !strings.HasSuffix(path, ".html") {
					continue
				}
				target = filepath.ToSlash(filepath.Clean(filepath.Join(filepath.Dir(p.Rel), path)))
				if strings.HasPrefix(target, "../") {
					continue // outside the guide
				}
				inSub := false
				for _, sp := range skip {
					if strings.HasPrefix(target, sp) {
						inSub = true
					}
				}
				if inSub {
					continue
				}
			}
			set, ok := ids[target]
			switch {
			case !ok:
				out = append(out, deck.Problem{File: deck.Rel(p.File), Msg: fmt.Sprintf("link to %s: no such page in the guide", strings.TrimSuffix(path, ".html")+".md")})
			case frag != "" && !set[frag] && !(decks[target] && slideHashRE.MatchString(frag)):
				out = append(out, deck.Problem{File: deck.Rel(p.File), Msg: fmt.Sprintf("link to %s#%s: no such anchor on that page", strings.TrimSuffix(target, ".html")+".md", frag)})
			}
		}
	}
	return out
}
