package render

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

// embedSlides renders `<!-- slides: ../decks/talk.md [23-24 | 23] -->` or
// `<!-- slides: ../decks/talk.md#live-a-first-broker -->` in a document: the
// deck (or those slides) rendered at build time, self-contained, in an iframe
// sized to the deck's aspect, with a caption naming the deck. Media are
// always inlined (the iframe document has no folder of its own).
func (dr *deckRenderer) embedSlides(val string, s *deck.Slide, sc *slideCtx) string {
	ref, rng, _ := strings.Cut(strings.TrimSpace(val), " ")
	ref, frag, _ := strings.Cut(ref, "#")
	rng = strings.TrimSpace(rng)
	if ref == "" {
		dr.problem(sc, "slides: needs a deck, e.g. <!-- slides: ../decks/talk.md 23-24 -->")
		return ""
	}
	target := filepath.Join(s.Dir, filepath.FromSlash(ref))
	sub, err := deck.Parse(target)
	if err != nil {
		dr.problem(sc, "slides: cannot read %s", ref)
		return ""
	}
	if sub.IsDoc() {
		dr.problem(sc, "slides: %s is a %s document, not a deck", ref, sub.Meta.Type)
		return ""
	}
	n := len(sub.Slides)
	from, to := 1, n
	switch {
	case frag != "":
		sl := newSlugger()
		from = 0
		for i, x := range sub.Slides {
			if sl.slug(html.UnescapeString(plain(dr.inline(x.Title)))) == frag {
				from = i + 1
				break
			}
		}
		if from == 0 {
			dr.problem(sc, "slides: %s has no slide with the heading id #%s", ref, frag)
			return ""
		}
		to = from
	case rng != "":
		a, b, ranged := strings.Cut(rng, "-")
		var e1, e2 error
		from, e1 = strconv.Atoi(strings.TrimSpace(a))
		to = from
		if ranged {
			to, e2 = strconv.Atoi(strings.TrimSpace(b))
		}
		if e1 != nil || e2 != nil || from < 1 || to > n || from > to {
			dr.problem(sc, "slides: %s has %d slides; want a range like 3 or 3-5, not %q", ref, n, rng)
			return ""
		}
	}
	part := *sub
	part.Slides = sub.Slides[from-1 : to]
	res := Deck(&part, Options{Media: MediaInline, Theme: dr.opt.Theme, Config: dr.opt.Config, UserCSS: dr.opt.UserCSS})
	for _, p := range res.Problems {
		dr.problem(sc, "slides: %s", p.Msg)
	}
	aspect := strings.ReplaceAll(sub.Meta.Aspect, ":", "/")
	if aspect == "" {
		aspect = "16/9"
	}
	which := fmt.Sprintf("slides %d–%d", from, to)
	if from == to {
		which = fmt.Sprintf("slide %d", from)
	}
	if from == 1 && to == n {
		which = fmt.Sprintf("%d slides", n)
	}
	title := plain(sub.Title())
	// the caption names the deck; it does not link to it, since the deck may not be part of the build
	return fmt.Sprintf(`<figure class="si-embed-slides" style="--_aspect:%s"><iframe srcdoc="%s" title="%s, %s" loading="lazy" allowfullscreen></iframe>`+
		`<figcaption>From <cite>%s</cite>, %s. Click the slides, then use the arrow keys.</figcaption></figure>`,
		html.EscapeString(aspect), html.EscapeString(string(res.HTML)), html.EscapeString(title), which,
		html.EscapeString(title), which)
}

// embed renders `<!-- embed: … -->`: a deck (or some of its slides) in a
// playable frame, a document as a card (default), in full (`full`) or one
// section (`#id`), and a guide, book or site folder as a card listing its
// pages. Everything is rendered when the page is built.
func (dr *deckRenderer) embed(val string, s *deck.Slide, sc *slideCtx) string {
	ref, rest, _ := strings.Cut(strings.TrimSpace(val), " ")
	rest = strings.TrimSpace(rest)
	path, frag, _ := strings.Cut(ref, "#")
	if path == "" {
		dr.problem(sc, "embed: needs a file or folder, e.g. <!-- embed: articles/retry.md -->")
		return ""
	}
	target := filepath.Join(s.Dir, filepath.FromSlash(path))
	if st, err := os.Stat(target); err == nil && st.IsDir() {
		return dr.embedFolder(target, path, sc)
	}
	d, err := deck.Parse(target)
	if err != nil {
		dr.problem(sc, "embed: cannot read %s", path)
		return ""
	}
	href := strings.TrimSuffix(path, filepath.Ext(path)) + ".html"
	if !d.IsDoc() {
		return dr.embedSlides(val, s, sc)
	}
	res := Deck(d, Options{Media: MediaInline, Theme: dr.opt.Theme, Config: dr.opt.Config})
	switch {
	case frag != "":
		sec := sectionHTML(res.body, frag)
		if sec == "" {
			dr.problem(sc, "embed: %s has no section #%s", path, frag)
			return ""
		}
		return fmt.Sprintf(`<section class="si-embed-doc">%s<p class="si-embed-from">From <a href="%s#%s">%s</a></p></section>`,
			sec, html.EscapeString(href), html.EscapeString(frag), html.EscapeString(plain(d.Title())))
	case rest == "full":
		return fmt.Sprintf(`<section class="si-embed-doc"><h2>%s</h2>%s<p class="si-embed-from">From <a href="%s">%s</a></p></section>`,
			dr.inline(d.Title()), res.body, html.EscapeString(href), html.EscapeString(plain(d.Title())))
	}
	var facts []string
	switch d.Meta.Type {
	case "article":
		facts = append(facts, fmt.Sprintf("%d min read", readingTime(res.body)))
	case "lab", "demo":
		if d.Meta.Duration != "" {
			facts = append(facts, d.Meta.Duration)
		}
		facts = append(facts, fmt.Sprintf("%d sections", labSections(d)))
	}
	return embedCard(d.Meta.Type, plain(d.Title()), d.Meta.Description, href, facts, nil)
}

// embedCard is the card of an embedded document or folder.
func embedCard(kind, title, desc, href string, facts []string, items [][2]string) string {
	var b strings.Builder
	fmt.Fprintf(&b, `<a class="si-embed-card" data-kind="%s" href="%s"><span class="si-embed-kind">%s</span><b>%s</b>`,
		html.EscapeString(kind), html.EscapeString(href), html.EscapeString(kind), html.EscapeString(title))
	if desc != "" {
		fmt.Fprintf(&b, `<span class="si-embed-desc">%s</span>`, html.EscapeString(desc))
	}
	if len(items) > 0 {
		b.WriteString(`<ol class="si-embed-items">`)
		for _, it := range items {
			fmt.Fprintf(&b, `<li><span>%s</span>%s</li>`, html.EscapeString(it[0]), html.EscapeString(it[1]))
		}
		b.WriteString(`</ol>`)
	}
	if len(facts) > 0 {
		fmt.Fprintf(&b, `<small>%s</small>`, html.EscapeString(strings.Join(facts, " · ")))
	}
	b.WriteString(`</a>`)
	return b.String()
}

// embedFolder is the card of a guide, book or site folder.
func (dr *deckRenderer) embedFolder(dir, path string, sc *slideCtx) string {
	g, err := deck.LoadGuide(dir)
	if err != nil || len(g.Pages) == 0 {
		dr.problem(sc, "embed: %s is not a guide, book or site (no _nav.md / _site.md with pages)", path)
		return ""
	}
	kind := "guide"
	if g.IsBook() {
		kind = "book"
	} else if g.IsSite() {
		kind = "site"
	}
	nums, _ := g.BookNumbers()
	var items [][2]string
	for _, p := range g.Pages {
		if it := g.Item(p); it != nil && len(items) < 6 {
			items = append(items, [2]string{nums[p], it.Title})
		}
	}
	home := "index.html"
	if !g.IsBook() && !fileExists(filepath.Join(g.Dir, "index.md")) {
		r, _ := filepath.Rel(g.Dir, g.Pages[0])
		home = filepath.ToSlash(strings.TrimSuffix(r, filepath.Ext(r)) + ".html")
	}
	facts := []string{fmt.Sprintf("%d pages", len(g.Pages))}
	if g.IsBook() {
		facts = []string{fmt.Sprintf("%d chapters", len(nums))}
	}
	return embedCard(kind, g.Meta.Title, firstNonEmpty(g.Meta.Description, g.Meta.Subtitle), strings.TrimSuffix(path, "/")+"/"+home, facts, items)
}

// sectionHTML is the h2 or h3 section with that id in a page body, up to the
// next heading of the same or a higher level.
func sectionHTML(body, id string) string {
	re := regexp.MustCompile(`<h([23]) id="` + regexp.QuoteMeta(id) + `"`)
	m := re.FindStringSubmatchIndex(body)
	if m == nil {
		return ""
	}
	level := body[m[2]:m[3]]
	rest := body[m[0]:]
	stop := regexp.MustCompile(`<h[2` + map[string]string{"2": "", "3": "3"}[level] + `] id=`)
	if loc := stop.FindStringIndex(rest[3:]); loc != nil {
		rest = rest[:loc[0]+3]
	}
	return rest
}
