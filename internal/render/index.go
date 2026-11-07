// The index page of a build or of si play: one entry per deck or document.

package render

import (
	"bytes"
	"fmt"
	"html"
	"strconv"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/assets"
	"github.com/felipeg48/simple-ideas/internal/deck"
)

// IndexEntry is one deck on the index page.
type IndexEntry struct {
	Href     string
	Title    string
	Subtitle string
	Slides   int
	Minutes  float64
	PDF      string // optional relative href to a slides PDF (pdfs/<name>.pdf); empty = no badge
	Status   string // optional status badge (e.g. beta, preview, new)
	Version  string // optional version badge (e.g. v1.2.0)
	Lab      bool   // a document (lab, article, guide): shown as "N sections", no slides/time
	Kind     string // "" or "deck", "lab", "article", "guide": the badge and data-kind of the tile
	Sections int    // lab: number of h2 sections
}

// Index renders the deck index page.
func Index(title string, entries []IndexEntry, opt Options) []byte {
	theme := assets.CanonicalTheme(opt.Theme)
	if _, ok := assets.Themes[theme]; !ok {
		theme = "studio" // the index page's own theme; follows --theme / the decks, else studio
	}
	var b bytes.Buffer
	fmt.Fprintf(&b, "<!doctype html>\n<html lang=\"en\" data-si-theme=\"%s\">\n<head>\n<meta charset=\"utf-8\">\n", theme)
	b.WriteString(`<meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover">
<meta name="color-scheme" content="light dark">
<link rel="icon" href="data:,">
`)
	fmt.Fprintf(&b, "<title>%s</title>\n<style>\n%s*,*::before,*::after{box-sizing:border-box}\nbody{margin:0;background:var(--si-bg);color:var(--si-fg)}\n%s%s</style>\n",
		html.EscapeString(title), assets.LayerOrder, assets.ThemeCSS(theme), assets.ThemeLooksCSS)
	if css := UserCSS(opt.UserCSS); css != "" {
		b.WriteString("<style>\n" + css + "</style>\n")
	}
	b.WriteString(`<script>try{var m=localStorage.getItem('si-mode');if(m)document.documentElement.setAttribute('data-si-mode',m)}catch(e){}</script>` + "\n")
	b.WriteString("</head>\n<body>\n<main class=\"si-index\" data-view=\"tiles\">\n")
	fmt.Fprintf(&b, "<header class=\"si-index-head\"><div><h1>%s</h1><p>%d deck(s)</p></div>"+
		"<div class=\"si-view\" role=\"group\" aria-label=\"View\"><button type=\"button\" data-view=\"tiles\">Tiles</button><button type=\"button\" data-view=\"list\">List</button></div></header>\n",
		html.EscapeString(title), len(entries))

	badges := func(e IndexEntry) string {
		var s string
		if e.Lab {
			k := e.Kind
			if k == "" || k == "deck" {
				k = "lab"
			}
			s += fmt.Sprintf("<span class=\"si-badge\" data-status=\"%s\">%s</span>", k, strings.ToUpper(k))
		}
		if e.Status != "" {
			s += fmt.Sprintf("<span class=\"si-badge\" data-status=\"%s\">%s</span>", html.EscapeString(strings.ToLower(e.Status)), html.EscapeString(strings.ToUpper(e.Status)))
		}
		if e.Version != "" {
			s += fmt.Sprintf("<span class=\"si-badge si-badge-ver\">%s</span>", html.EscapeString(e.Version))
		}
		return s
	}

	b.WriteString("<div class=\"grid\">\n")
	for _, e := range entries {
		mins := ""
		if e.Minutes > 0 {
			mins = fmt.Sprintf(" · %s min", strconv.FormatFloat(e.Minutes, 'f', -1, 64))
		}
		bg := badges(e)
		if bg != "" {
			bg = "<div class=\"si-badges\">" + bg + "</div>"
		}
		size, kind := fmt.Sprintf("%d slides%s", e.Slides, mins), "deck"
		if e.Lab {
			size, kind = e.SizeLabel(), "lab"
			if e.Kind == "guide" || e.Kind == "book" || e.Kind == "article" || e.Kind == "mindmap" {
				kind = e.Kind
			}
		}
		inner := fmt.Sprintf("<b>%s</b><span>%s</span>%s<small>%s</small>",
			html.EscapeString(e.Title), html.EscapeString(e.Subtitle), bg, size)
		if e.PDF == "" {
			fmt.Fprintf(&b, "<a class=\"si-card\" data-kind=\"%s\" href=\"%s\">%s</a>\n", kind, html.EscapeString(e.Href), inner)
		} else {
			// Two sibling links in a positioned wrapper: the card opens the deck, the
			// badge downloads the PDF. (Nesting <a> in <a> is invalid HTML.)
			fmt.Fprintf(&b, "<div class=\"si-card-wrap\"><a class=\"si-card\" data-kind=\"%s\" href=\"%s\">%s</a>"+
				"<a class=\"si-card-pdf\" href=\"%s\" download title=\"Download the slides as PDF\">PDF</a></div>\n",
				kind, html.EscapeString(e.Href), inner, html.EscapeString(e.PDF))
		}
	}
	b.WriteString("</div>\n")

	b.WriteString("<table class=\"si-list\"><thead><tr><th>Course</th><th>Status</th><th>Slides</th><th>Time</th><th>PDF</th></tr></thead><tbody>\n")
	for _, e := range entries {
		t := ""
		if e.Minutes > 0 {
			t = strconv.FormatFloat(e.Minutes, 'f', -1, 64) + " min"
		}
		pdf := "—"
		if e.PDF != "" {
			pdf = fmt.Sprintf("<a href=\"%s\" download>PDF</a>", html.EscapeString(e.PDF))
		}
		n := fmt.Sprint(e.Slides)
		if e.Lab {
			n, t = e.SizeLabel(), ""
		}
		fmt.Fprintf(&b, "<tr><td><a href=\"%s\">%s</a></td><td>%s</td><td>%s</td><td>%s</td><td>%s</td></tr>\n",
			html.EscapeString(e.Href), html.EscapeString(e.Title), badges(e), n, t, pdf)
	}
	b.WriteString("</tbody></table>\n</main>\n")
	b.WriteString(`<script>(function(){var m=document.querySelector('.si-index');if(!m)return;var btns=m.querySelectorAll('.si-view button');function set(v){m.setAttribute('data-view',v);btns.forEach(function(x){x.setAttribute('aria-pressed',x.getAttribute('data-view')===v?'true':'false')});try{localStorage.setItem('si-index-view',v)}catch(e){}}var saved;try{saved=localStorage.getItem('si-index-view')}catch(e){}set(saved==='list'?'list':'tiles');btns.forEach(function(b){b.addEventListener('click',function(){set(b.getAttribute('data-view'))})})})();</script>` + "\n")
	if opt.LiveReload {
		b.WriteString(LiveReloadJS)
	}
	b.WriteString("</body>\n</html>\n")
	return b.Bytes()
}

// SizeLabel is the size of a document entry: sections, pages, chapters or maps.
func (e IndexEntry) SizeLabel() string {
	switch e.Kind {
	case "guide":
		return fmt.Sprintf("%d pages", e.Sections)
	case "book":
		return fmt.Sprintf("%d chapters", e.Sections)
	case "mindmap":
		if e.Sections == 1 {
			return "1 map"
		}
		return fmt.Sprintf("%d maps", e.Sections)
	}
	return fmt.Sprintf("%d sections", e.Sections)
}

// Entry builds an index entry for a parsed deck.
func Entry(d *deck.Deck) IndexEntry {
	e := IndexEntry{Href: d.Name + ".html", Title: plain(d.Title()), Slides: len(d.Slides), Minutes: d.Minutes(), Status: d.Meta.Status, Version: d.Meta.Version}
	if d.IsDoc() {
		e.Lab, e.Kind, e.Sections, e.Slides, e.Minutes = true, d.Meta.Type, labSections(d), 0, 0
		if d.Meta.Description != "" {
			e.Subtitle = plain(d.Meta.Description)
			return e
		}
	}
	first := d.Slides
	if !e.Lab && len(first) > 1 {
		first = first[:1] // a deck's subtitle comes from its title slide; a lab's from its first text line
	}
outer:
	for _, s := range first {
		for _, line := range strings.Split(s.Body, "\n") {
			if t := strings.TrimSpace(line); t != "" && !strings.HasPrefix(t, "<!--") && !(e.Lab && strings.HasPrefix(t, "#")) {
				e.Subtitle = plain(t)
				break outer
			}
		}
	}
	return e
}

// labSections counts a lab's h2 headings, ignoring fenced code (a mind map
// page: its maps, one per h1).
func labSections(d *deck.Deck) int {
	head := "## "
	if d.IsMindmap() {
		head = "# "
	}
	n, fence := 0, false
	for _, s := range d.Slides {
		for _, line := range strings.Split(s.Body, "\n") {
			switch {
			case strings.HasPrefix(line, "```") || strings.HasPrefix(line, "~~~"):
				fence = !fence
			case !fence && strings.HasPrefix(line, head):
				n++
			}
		}
	}
	if n == 0 && d.IsMindmap() {
		n = 1 // a list with no # Title is one map
	}
	return n
}
