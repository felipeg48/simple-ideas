package studio

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

// guides caches the guide, book or site that owns each folder during one
// request, so a tree walk loads every _nav.md / _site.md once.
type guides map[string]*deck.Guide

// of returns the guide whose page list holds file, or nil.
func (gs guides) of(file, root string) *deck.Guide {
	dir := filepath.Dir(file)
	g, ok := gs[dir]
	if !ok {
		g = deck.FindGuide(file, root)
		gs[dir] = g
	}
	if g != nil && g.Item(file) == nil {
		return nil // a file in the folder that the nav does not list
	}
	return g
}

// parsePage parses a file the way si check and si serve do: a page of a
// guide, book or site starts from its _nav.md / _site.md settings.
func (gs guides) parsePage(file, root string) (*deck.Deck, *deck.Guide, error) {
	if g := gs.of(file, root); g != nil {
		d, err := deck.ParseWith(file, g.Defaults())
		return d, g, err
	}
	d, err := deck.Parse(file)
	return d, nil, err
}

// guideKind is "site", "book" or "guide".
func guideKind(g *deck.Guide) string {
	switch {
	case g.IsSite():
		return "site"
	case g.IsBook():
		return "book"
	}
	return "guide"
}

// pageKind is what the Studio shows a file as: deck, lab, article, demo,
// page (a site page), or guide / book (a page of one).
func pageKind(d *deck.Deck, g *deck.Guide) string {
	if !d.IsDoc() {
		return "deck"
	}
	if d.Meta.Type == "guide" && g != nil {
		return guideKind(g)
	}
	return d.Meta.Type
}

// isNavFile reports whether name is a _nav.md or _site.md.
func isNavFile(name string) bool { return name == deck.NavFile || name == deck.SiteFile }

// pageURL is the live page of a workspace-relative file: "a/b.md" → "/a/b.html".
func pageURL(rel string) string { return "/" + strings.TrimSuffix(rel, filepath.Ext(rel)) + ".html" }

// navPath is the guide's _site.md or _nav.md.
func navPath(g *deck.Guide) string {
	if p := filepath.Join(g.Dir, deck.SiteFile); fileExists(p) {
		return p
	}
	return filepath.Join(g.Dir, deck.NavFile)
}

func fileExists(p string) bool { _, err := os.Stat(p); return err == nil }
