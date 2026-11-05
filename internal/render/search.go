package render

import (
	"encoding/json"
	"html"
	"regexp"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

// Search in documents (`/`): doc.js searches the page itself (its headings
// and text) from the DOM, and the other pages from a JSON block written at
// build time: a guide or book page lists every other page with its sections
// (and a short snippet of their text) and, in a book, the index terms; a page
// on its own lists the other pages of the build (Options.Pages).

const searchSlot = "__SI_SEARCH__"

type searchItem struct {
	Href  string `json:"h"`
	Title string `json:"t"`
	Page  string `json:"p,omitempty"` // the page a section or term is on
	Kind  string `json:"k"`           // page, section, term
	Text  string `json:"x,omitempty"` // a snippet of the section's text
}

// searchScript is the data block of a page; slot leaves a placeholder that
// RenderGuide fills once every page of the guide is rendered.
func (dr *deckRenderer) searchScript() string {
	if dr.opt.Guide != nil {
		return `<script type="application/json" id="si-search">` + searchSlot + "</script>\n"
	}
	var items []searchItem
	self := dr.d.Name + ".html"
	for _, p := range dr.opt.Pages {
		if p.Href != self {
			items = append(items, searchItem{Href: p.Href, Title: p.Title, Kind: p.Kind})
		}
	}
	return searchJSON(items)
}

func searchJSON(items []searchItem) string {
	if items == nil {
		items = []searchItem{}
	}
	b, _ := json.Marshal(items)
	return `<script type="application/json" id="si-search">` + strings.ReplaceAll(string(b), "</", `<\/`) + "</script>\n"
}

var (
	secSplitRE = regexp.MustCompile(`<h([23]) id="([^"]+)"[^>]*>`)
	spaceRE    = regexp.MustCompile(`\s+`)
	stripRE    = regexp.MustCompile(`(?s)<pre.*?</pre>|<[^>]+>`)
)

// sectionSnippets maps each h2/h3 id of a page body to the first words of
// its text (code left out).
func sectionSnippets(body string) map[string]string {
	out := map[string]string{}
	idx := secSplitRE.FindAllStringSubmatchIndex(body, -1)
	for i, m := range idx {
		end := len(body)
		if i+1 < len(idx) {
			end = idx[i+1][0]
		}
		seg := body[m[1]:end]
		if j := strings.Index(seg, "</h"); j >= 0 {
			seg = seg[j:]
		}
		t := strings.TrimSpace(spaceRE.ReplaceAllString(html.UnescapeString(stripRE.ReplaceAllString(seg, " ")), " "))
		t = strings.TrimPrefix(t, "#")
		if r := []rune(strings.TrimSpace(t)); len(r) > 220 {
			t = string(r[:220]) + "…"
		}
		out[body[m[4]:m[5]]] = strings.TrimSpace(t)
	}
	return out
}

// fillGuideSearch writes each page's search data: the other pages, their
// sections and, for a book, its index terms; hrefs are relative to the page.
func fillGuideSearch(g *deck.Guide, pages []GuidePage, terms []ixEntry) {
	type pageData struct {
		file, rel, title string
		secs             []searchItem
	}
	var all []pageData
	for _, p := range pages {
		title := plain(p.Deck.Title())
		pd := pageData{file: p.File, rel: p.Rel, title: title}
		snips := sectionSnippets(p.Result.body)
		for _, e := range p.Result.toc {
			pd.secs = append(pd.secs, searchItem{Href: "#" + e.id, Title: html.UnescapeString(e.text), Page: title, Kind: "section", Text: snips[e.id]})
		}
		all = append(all, pd)
	}
	for i := range pages {
		p := &pages[i]
		if !strings.Contains(string(p.Result.HTML), searchSlot) {
			continue
		}
		var items []searchItem
		for _, o := range all {
			if o.rel == p.Rel || o.rel == "book.html" {
				continue
			}
			href := relHref(p.Rel, o.rel)
			items = append(items, searchItem{Href: href, Title: o.title, Kind: "page"})
			for _, s := range o.secs {
				s.Href = href + s.Href
				items = append(items, s)
			}
		}
		for _, t := range terms {
			rel, frag, _ := strings.Cut(t.href, "#")
			label := t.term
			if t.sub != "" {
				label += ", " + t.sub
			}
			items = append(items, searchItem{Href: relHref(p.Rel, rel) + "#" + frag, Title: label, Page: t.loc, Kind: "term"})
		}
		data := searchJSON(items)
		data = strings.TrimSuffix(strings.TrimPrefix(data, `<script type="application/json" id="si-search">`), "</script>\n")
		p.Result.HTML = []byte(strings.Replace(string(p.Result.HTML), searchSlot, data, 1))
	}
}

// relHref is the link from the built page `from` to the built page `to`,
// both paths inside the guide.
func relHref(from, to string) string {
	fromParts := strings.Split(from, "/")
	toParts := strings.Split(to, "/")
	i := 0
	for i < len(fromParts)-1 && i < len(toParts)-1 && fromParts[i] == toParts[i] {
		i++
	}
	return strings.Repeat("../", len(fromParts)-1-i) + strings.Join(toParts[i:], "/")
}
