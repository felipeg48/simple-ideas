package render

import (
	"html"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/config"
	"github.com/felipeg48/simple-ideas/internal/deck"
)

// Page is one entry of the page list that find (/) searches next to the
// slides: Href is relative to the deck (or a URL), Kind is index, deck, lab,
// pdf, page (another .html beside the decks) or link (si.conf links.*).
type Page struct {
	Href  string `json:"href"`
	Title string `json:"title"`
	Kind  string `json:"kind"`
}

var titleRE = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)

// Pages lists what a deck in a built or served folder can link to: the index
// (when there is one), every deck and lab, their slide PDFs (pdf), other .html
// files already in dir, and the si.conf links. Only names and titles: a deck
// stays self-contained.
func Pages(decks []*deck.Deck, index, pdf bool, dir string, cfg *config.Config) []Page {
	var out []Page
	seen := map[string]bool{}
	add := func(p Page) {
		if !seen[p.Href] {
			seen[p.Href] = true
			out = append(out, p)
		}
	}
	if index {
		add(Page{Href: "index.html", Title: "Index", Kind: "index"})
	}
	for _, d := range decks {
		if d == nil {
			continue
		}
		kind := "deck"
		if d.IsLab() {
			kind = "lab"
		}
		add(Page{Href: d.Name + ".html", Title: d.Title(), Kind: kind})
	}
	if pdf {
		for _, d := range decks {
			if d != nil && !d.IsLab() {
				add(Page{Href: "pdfs/" + d.Name + ".pdf", Title: d.Title() + " (PDF)", Kind: "pdf"})
			}
		}
	}
	if dir != "" {
		files, _ := filepath.Glob(filepath.Join(dir, "*.html"))
		for _, f := range files {
			name := filepath.Base(f)
			if seen[name] {
				continue
			}
			add(Page{Href: name, Title: htmlTitle(f, strings.TrimSuffix(name, ".html")), Kind: "page"})
		}
	}
	if cfg != nil {
		for _, l := range cfg.Links {
			add(Page{Href: l.URL, Title: l.Label, Kind: "link"})
		}
	}
	return out
}

// htmlTitle reads the <title> of an HTML file (from its first 8 KB), or def.
func htmlTitle(path, def string) string {
	f, err := os.Open(path)
	if err != nil {
		return def
	}
	defer f.Close()
	buf := make([]byte, 8192)
	n, _ := f.Read(buf)
	if m := titleRE.FindSubmatch(buf[:n]); m != nil {
		if t := strings.TrimSpace(html.UnescapeString(string(m[1]))); t != "" {
			return t
		}
	}
	return def
}
