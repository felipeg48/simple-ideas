package studio

import (
	"bytes"
	"net/http"
	"path/filepath"

	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/felipeg48/simple-ideas/internal/render"
)

// checkAll checks the whole workspace like `si check <dir>`: every deck and
// document on its own, every guide, book and site as a whole.
func (a *api) checkAll(w http.ResponseWriter, r *http.Request) {
	problems, checked := a.problemsAll()
	writeJSON(w, http.StatusOK, map[string]any{"problems": problems, "checked": checked})
}

// Problem is one si check finding, with a workspace-relative file.
type Problem = problemJSON

// DeckInfo is a deck, document page, or guide / book / site of a workspace.
type DeckInfo = deckInfo

// Check checks the whole workspace like si check (the terminal Studio's problems).
func (w *Workspace) Check() ([]Problem, int) { return (&api{ws: w}).problemsAll() }

// Decks lists the workspace's decks, document pages and guides, with kinds and titles.
func (w *Workspace) Decks() []DeckInfo { return (&api{ws: w}).listDecks() }

func (a *api) problemsAll() ([]problemJSON, int) {
	problems, checked := []problemJSON{}, 0
	opt := render.Options{ShowProblems: true, Media: render.MediaLink}
	add := func(ps []deck.Problem) {
		for _, p := range ps {
			problems = append(problems, problemJSON{a.rel(p.File), p.Line, p.Msg})
		}
	}
	for _, info := range a.listDecks() {
		abs := filepath.Join(a.ws.Root, filepath.FromSlash(info.Path))
		switch {
		case isNavFile(filepath.Base(abs)):
			if g, err := deck.LoadGuide(filepath.Dir(abs)); err == nil {
				_, probs, _ := render.RenderGuide(g, opt)
				add(probs)
				checked += len(g.Pages)
			}
		case info.Guide != "": // checked with its guide
		default:
			d, err := deck.Parse(abs)
			if err != nil {
				problems = append(problems, problemJSON{info.Path, 0, err.Error()})
				continue
			}
			add(render.Deck(d, opt).Problems)
			checked++
		}
	}
	return problems, checked
}

// build runs `si build` on the workspace into its html/ folder (which the
// Studio tree and the decks list skip) and returns the CLI's own output.
func (a *api) build(w http.ResponseWriter, r *http.Request) {
	var req struct{ PDF bool }
	if !decode(w, r, &req) {
		return
	}
	if a.run == nil {
		fail(w, http.StatusNotImplemented, "build is not available")
		return
	}
	args := []string{"build", a.ws.Root, "-o", filepath.Join(a.ws.Root, "html")}
	for _, c := range a.css {
		args = append(args, "--css", c)
	}
	if a.theme != "" {
		args = append(args, "--theme", a.theme)
	}
	if a.title != "" {
		args = append(args, "--title", a.title)
	}
	if req.PDF {
		args = append(args, "--pdf")
	}
	var out bytes.Buffer
	code := a.run(args, &out)
	writeJSON(w, http.StatusOK, map[string]any{"code": code, "output": out.String(), "url": "/html/index.html"})
}
