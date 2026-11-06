package studio

import (
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/felipeg48/simple-ideas/internal/render"
)

type problemJSON struct {
	File string `json:"file"`
	Line int    `json:"line"`
	Msg  string `json:"msg"`
}

type slideJSON struct {
	Index   int     `json:"index"`
	File    string  `json:"file"`
	Line    int     `json:"line"`
	Title   string  `json:"title"`
	Topic   int     `json:"topic"`
	Minutes float64 `json:"minutes"`
	// what the inspector shows and edits
	Layout     string `json:"layout,omitempty"`
	Transition string `json:"transition,omitempty"`
	Background string `json:"background,omitempty"`
	Class      string `json:"class,omitempty"`
	Eyebrow    string `json:"eyebrow,omitempty"`
	Words      int    `json:"words"`  // words in Notes: (the teleprompter script)
	Clicks     int    `json:"clicks"` // [CLICK] cues in Notes:
	Steps      int    `json:"steps"`  // step / steps directives in the body
}

func (a *api) check(w http.ResponseWriter, r *http.Request) {
	abs, ok := a.resolve(w, r.URL.Query().Get("deck"))
	if !ok {
		return
	}
	if fi, err := os.Stat(abs); err != nil || fi.IsDir() {
		fail(w, http.StatusNotFound, "not found")
		return
	}
	// MediaLink: a check only needs problems and counts; inlining would
	// base64-encode every image of every deck on each save
	opt := render.Options{ShowProblems: true, Media: render.MediaLink}
	problems, slides := []problemJSON{}, []slideJSON{}
	add := func(ps []deck.Problem) {
		for _, p := range ps {
			problems = append(problems, problemJSON{a.rel(p.File), p.Line, p.Msg})
		}
	}
	gs := guides{}
	var g *deck.Guide
	if isNavFile(filepath.Base(abs)) {
		g, _ = deck.LoadGuide(filepath.Dir(abs))
	} else {
		g = gs.of(abs, a.ws.Root)
	}
	if g != nil {
		// a guide, book or site is checked whole, like si check: numbering,
		// links and anchors between pages, settings from its nav file
		// ponytail: renders every page on each save; fine for guide-sized folders
		pages, probs, _ := render.RenderGuide(g, opt)
		add(probs)
		out := map[string]any{"problems": problems, "slides": slides, "minutes": 0.0, "steps": 0, "canvases": 0, "kind": guideKind(g)}
		for _, p := range pages {
			if filepath.Clean(p.File) == filepath.Clean(abs) {
				out["kind"] = pageKind(p.Deck, g)
				out["minutes"], out["steps"], out["canvases"] = p.Deck.Minutes(), p.Result.Steps, p.Result.Canvases
				out["slides"] = a.slides(p.Deck)
			}
		}
		writeJSON(w, http.StatusOK, out)
		return
	}
	d, err := deck.Parse(abs)
	if err != nil {
		fail(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	res := render.Deck(d, opt)
	add(res.Problems)
	writeJSON(w, http.StatusOK, map[string]any{
		"problems": problems, "slides": a.slides(d), "kind": pageKind(d, nil),
		"minutes": d.Minutes(), "steps": res.Steps, "canvases": res.Canvases,
	})
}

var stepRE = regexp.MustCompile(`<!--\s*steps?\s*(?::|-->)`)

// slides lists a deck's slides (a document's parts) for the editor's slide map.
func (a *api) slides(d *deck.Deck) []slideJSON {
	topics := render.Topics(d)
	out := []slideJSON{}
	for i, s := range d.Slides {
		out = append(out, slideJSON{Index: i + 1, File: a.rel(s.File), Line: s.Line, Title: s.Title, Topic: topics[i], Minutes: s.Minutes,
			Layout: s.Layout, Transition: s.Transition, Background: s.Background, Class: strings.Join(s.Classes, " "), Eyebrow: s.Eyebrow,
			Words: len(strings.Fields(s.Notes)), Clicks: strings.Count(s.Notes, "[CLICK]"), Steps: len(stepRE.FindAllString(s.Body, -1))})
	}
	return out
}
