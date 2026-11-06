package studio

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

type deckInfo struct {
	Path  string   `json:"path"`
	Title string   `json:"title"`
	Files []string `json:"files"`
	// Runbook is the deck's `runbook:` front-matter value, as written.
	Runbook string `json:"runbook,omitempty"`
	// Kind is deck, lab, article, demo, page, or guide / book / site (a page of
	// one, or its _nav.md / _site.md); Guide is that folder's nav file.
	Kind  string `json:"kind"`
	URL   string `json:"url"`
	Guide string `json:"guide,omitempty"`
}

// sourceFiles lists the deck's own file first, then each included file in
// slide order, workspace-relative.
func (a *api) sourceFiles(d *deck.Deck) []string {
	main := a.rel(d.Path)
	out, seen := []string{main}, map[string]bool{main: true}
	for _, s := range d.Slides {
		if s.File == "" {
			continue
		}
		if f := a.rel(s.File); !seen[f] {
			seen[f] = true
			out = append(out, f)
		}
	}
	return out
}

// rel makes a parser-reported path workspace-relative.
func (a *api) rel(p string) string {
	// deck.rel yields cwd-relative paths; resolve against cwd, then trim the root.
	if abs, err := filepath.Abs(p); err == nil {
		return a.ws.Rel(abs)
	}
	return filepath.ToSlash(p)
}

func (a *api) decks(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, a.listDecks())
}

// listDecks lists every deck, document page and guide / book / site in the workspace.
func (a *api) listDecks() []deckInfo {
	inc := map[string]bool{}
	a.ws.collectIncluded(a.ws.Root, inc)
	out, gs := []deckInfo{}, guides{}
	var walk func(dir string)
	walk = func(dir string) {
		list, _ := deck.List(dir)
		for _, f := range list {
			if inc[filepath.Clean(f)] {
				continue // included modules are not decks
			}
			if d, g, err := gs.parsePage(f, a.ws.Root); err == nil {
				info := deckInfo{Path: a.rel(d.Path), Title: plainTitle(d.Title()), Files: a.sourceFiles(d), Runbook: d.Meta.Runbook, Kind: pageKind(d, g)}
				info.URL = pageURL(info.Path)
				if g != nil {
					info.Guide = a.rel(navPath(g))
				}
				out = append(out, info)
			}
		}
		if deck.IsGuideDir(dir) { // the guide, book or site itself: its nav file previews its first page
			if g, err := deck.LoadGuide(dir); err == nil && len(g.Pages) > 0 {
				nav := a.rel(navPath(g))
				home := g.Pages[0]
				if idx := filepath.Join(dir, "index.md"); g.Item(idx) != nil {
					home = idx
				}
				out = append(out, deckInfo{Path: nav, Title: g.Meta.Title, Files: []string{nav}, Kind: guideKind(g), URL: pageURL(a.rel(home)), Guide: nav})
			}
		}
		entries, _ := os.ReadDir(dir)
		for _, e := range entries {
			if e.IsDir() && !strings.HasPrefix(e.Name(), ".") && !skipDirs[e.Name()] {
				walk(filepath.Join(dir, e.Name()))
			}
		}
	}
	walk(a.ws.Root)
	return out
}
