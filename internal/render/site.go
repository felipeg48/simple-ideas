package render

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

// Sites: a folder with a _site.md (the guide format, `type: site`). Every page
// gets the site's navigator instead of the guide bar and sidebar:
//
//	topbar          logo left, menu, call to action right (the default)
//	topbar-centred  menu left, logo centred, call to action right
//	mega            a top bar whose sections open as wide panels
//	sidebar         a sticky column on the left
//
// A menu section (an item without a link) opens its children as a dropdown;
// a link to a folder that holds a guide, book or site opens it. On narrow
// screens the menu folds behind a Menu button.

// siteHome is the href from page to the site's (or a sub-guide's) first page.
func siteHome(g *deck.Guide, page string) string {
	if g.IsBook() || fileExists(filepath.Join(g.Dir, "index.md")) {
		return pageHref(page, filepath.Join(g.Dir, "index.md"))
	}
	return pageHref(page, g.Pages[0])
}

// subHome is the href from page to the first page of the sub-guide in dir.
func subHome(page, dir string) string {
	sub, err := deck.LoadGuide(dir)
	if err != nil || len(sub.Pages) == 0 {
		r, _ := filepath.Rel(filepath.Dir(page), dir)
		return filepath.ToSlash(r) + "/"
	}
	return siteHome(sub, page)
}

func (dr *deckRenderer) siteNav(g *deck.Guide, page string) string {
	kind := g.Meta.Navigator
	switch kind {
	case "", "topbar":
		kind = "topbar"
	case "topbar-centred", "topbar-centered", "centred", "centered":
		kind = "topbar-centred"
	case "mega", "sidebar":
	default:
		dr.problem(nil, "navigator: %q is not one of topbar, topbar-centred, mega, sidebar", g.Meta.Navigator)
		kind = "topbar"
	}
	link := func(it *deck.NavItem) string {
		switch {
		case it.File != "":
			cur := ""
			if it.File == page {
				cur = ` aria-current="page"`
			}
			return fmt.Sprintf(`<a href="%s"%s>%s</a>`, html.EscapeString(pageHref(page, it.File)), cur, html.EscapeString(it.Title))
		case it.Sub != "":
			cur := ""
			if strings.HasPrefix(page, it.Sub+string(filepath.Separator)) {
				cur = ` aria-current="page"`
			}
			return fmt.Sprintf(`<a href="%s"%s>%s</a>`, html.EscapeString(subHome(page, it.Sub)), cur, html.EscapeString(it.Title))
		case it.Href != "":
			return fmt.Sprintf(`<a href="%s">%s</a>`, html.EscapeString(it.Href), html.EscapeString(it.Title))
		}
		return ""
	}
	var menu strings.Builder
	for _, it := range g.Items {
		if it.File == "" && it.Sub == "" && it.Href == "" { // a section: a dropdown of its children
			var kids []string
			open := false
			for _, c := range it.Children {
				if c.File == page {
					open = true
				}
				kids = append(kids, link(c))
			}
			cur := ""
			if open {
				cur = ` data-current`
			}
			fmt.Fprintf(&menu, `<div class="si-site-group"%s><span class="si-site-group-t" tabindex="0">%s</span><div class="si-site-drop">%s</div></div>`,
				cur, html.EscapeString(it.Title), strings.Join(kids, ""))
			continue
		}
		menu.WriteString(link(it))
	}
	logo := html.EscapeString(g.Meta.Title)
	l := strings.TrimSpace(g.Meta.Logo)
	img, text, _ := strings.Cut(l, " ") // "mark.svg simple-ideas": the picture, then the name beside it
	if l != "" && deck.IsImagePath(img) {
		if uri, ok := dr.media(g.Dir, img, nil); ok {
			alt := g.Meta.Title
			if text = strings.TrimSpace(text); text != "" {
				alt = ""
			}
			logo = fmt.Sprintf(`<img src="%s" alt="%s">`, uri, html.EscapeString(alt))
			if text != "" {
				logo += "<span>" + html.EscapeString(text) + "</span>"
			}
		}
	} else if l != "" {
		logo = html.EscapeString(l)
	}
	if v := strings.TrimSpace(g.Meta.Version); v != "" { // version: v2.5.0 → release notes
		label, href, ok := strings.Cut(v, "→")
		if !ok {
			label, href, _ = strings.Cut(v, "->")
		}
		label, href = strings.TrimSpace(label), strings.TrimSpace(href)
		if href != "" {
			logo += fmt.Sprintf(`</a><a class="si-site-version" href="%s" title="Release notes">%s`, html.EscapeString(href), html.EscapeString(label))
		} else {
			logo += fmt.Sprintf(`<span class="si-site-version">%s</span>`, html.EscapeString(label))
		}
	}
	cta := ctaLink(g.Meta.CTA, g.Dir, page)
	tools := `<button type="button" class="si-doc-search" data-si-action="search" title="Search (/)" aria-label="Search">` + searchIconSVG + `<span>Search</span><kbd>/</kbd></button>` +
		`<button type="button" class="si-doc-mode" data-si-action="mode" title="Light / dark (d)" aria-label="Light / dark">` + modeIconSVG + `</button>`
	return fmt.Sprintf(`<header class="si-site-nav" data-nav="%s"><div class="si-site-in"><a class="si-site-logo" href="%s">%s</a>`+
		`<details class="si-site-menu"><summary>Menu</summary><nav class="si-site-links" aria-label="%s">%s</nav></details>`+
		`<div class="si-site-tools">%s%s</div></div></header>`+"\n",
		kind, html.EscapeString(siteHome(g, page)), logo, html.EscapeString(g.Meta.Title), menu.String(), cta, tools)
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// sitePage writes the body of a site page: _header.md, the navigator, the
// page (title, lede, byline, body), the contents for an article, _footer.md.
// ctaLink renders a cta: value, "Label → page" (or → https://…, → #modal),
// as the highlighted button; a page path is relative to dir.
func ctaLink(c, dir, page string) string {
	c = strings.TrimSpace(c)
	if c == "" {
		return ""
	}
	label, href, ok := strings.Cut(c, "→")
	if !ok {
		label, href, _ = strings.Cut(c, "->")
	}
	label, href = strings.TrimSpace(label), strings.TrimSpace(href)
	if href != "" && !strings.Contains(href, ":") && !strings.HasPrefix(href, "#") {
		href = pageHref(page, filepath.Join(dir, filepath.FromSlash(strings.TrimSuffix(href, ".md")+".md")))
	}
	return fmt.Sprintf(`<a class="si-site-cta" href="%s">%s</a>`, html.EscapeString(href), html.EscapeString(label))
}

func (dr *deckRenderer) sitePage(b *strings.Builder, body, lede, metaHTML, tocHTML string) {
	g, d, m := dr.opt.Guide, dr.d, dr.d.Meta
	if m.Header != "off" {
		if h := dr.partial("_header.md"); h != "" {
			b.WriteString("<div class=\"si-doc-site\"><div class=\"si-doc-site-in\">" + h + "</div></div>\n")
		}
	}
	nav := dr.siteNav(g, d.Path)
	side := strings.Contains(nav, `data-nav="sidebar"`)
	b.WriteString(nav)
	layout := "site"
	if side {
		layout = "site-sidebar"
	}
	fmt.Fprintf(b, "<div class=\"si-doc-layout\" data-layout=\"%s\">\n<main class=\"si-doc-body\">\n", layout)
	if m.Title != "off" {
		fmt.Fprintf(b, "<h1 class=\"si-doc-title\">%s</h1>\n%s%s", dr.inline(d.Title()), lede, metaHTML)
	}
	if c := ctaLink(m.CTA, filepath.Dir(d.Path), d.Path); c != "" { // a page's own cta: a button under its title
		b.WriteString("<p class=\"si-page-cta\">" + c + "</p>\n")
	}
	b.WriteString(body)
	b.WriteString("</main>\n")
	if m.Type == "article" {
		b.WriteString(tocHTML)
	}
	b.WriteString("</div>\n")
	if m.FooterPage != "off" {
		if f := dr.partial("_footer.md"); f != "" {
			b.WriteString("<div class=\"si-doc-sitefoot\"><div class=\"si-doc-site-in\">" + f + "</div></div>\n")
		}
	}
}
