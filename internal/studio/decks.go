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
	inc := map[string]bool{}
	a.ws.collectIncluded(a.ws.Root, inc)
	out := []deckInfo{}
	var walk func(dir string)
	walk = func(dir string) {
		list, _ := deck.List(dir)
		for _, f := range list {
			if inc[filepath.Clean(f)] {
				continue // included modules are not decks
			}
			if d, err := deck.Parse(f); err == nil {
				out = append(out, deckInfo{a.rel(d.Path), d.Title(), a.sourceFiles(d), d.Meta.Runbook})
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
	writeJSON(w, http.StatusOK, out)
}
