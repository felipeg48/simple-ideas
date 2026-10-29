package studio

import (
	"net/http"
	"os"

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
	d, err := deck.Parse(abs)
	if err != nil {
		fail(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	// MediaLink: a check only needs problems and counts; inlining would
	// base64-encode every image of every deck on each save
	res := render.Deck(d, render.Options{ShowProblems: true, Media: render.MediaLink})
	problems := []problemJSON{}
	for _, p := range res.Problems {
		problems = append(problems, problemJSON{a.rel(p.File), p.Line, p.Msg})
	}
	topics := render.Topics(d)
	slides := []slideJSON{}
	for i, s := range d.Slides {
		slides = append(slides, slideJSON{i + 1, a.rel(s.File), s.Line, s.Title, topics[i], s.Minutes})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"problems": problems, "slides": slides,
		"minutes": d.Minutes(), "steps": res.Steps, "canvases": res.Canvases,
	})
}
