package studio

import (
	"bytes"
	"net/http"
	"path/filepath"

	"github.com/felipeg48/simple-ideas/internal/codeshow"
	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/felipeg48/simple-ideas/internal/render"
)

// The Window view: the code block (or the run of same-id blocks) under the
// cursor, as a code window. POST codewin applies one op to the editor's text
// through internal/codeshow and answers with the new text, the show's model
// for the panels and a preview page (the real renderer and codewin.js); the
// UI puts the text in the editor as one undoable change. POST codewin/export
// runs si code on the saved file.

type codewinReq struct {
	Path string       `json:"path"`
	Text string       `json:"text"`
	Line int          `json:"line"` // a 1-based line of the block
	Op   *codeshow.Op `json:"op"`
}

func (a *api) codewin(w http.ResponseWriter, r *http.Request) {
	var req codewinReq
	if !decode(w, r, &req) {
		return
	}
	abs, ok := a.resolve(w, req.Path)
	if !ok {
		return
	}
	text, line := req.Text, req.Line
	if req.Op != nil {
		next, first, err := codeshow.Apply(text, line, *req.Op)
		if err != nil {
			fail(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		text, line = next, first
	}
	sh, ok := codeshow.ShowAt(text, line)
	if !ok {
		fail(w, http.StatusUnprocessableEntity, "no code block at that line")
		return
	}
	look := map[string]string{}
	if d, err := deck.ParseSource(abs, text, deck.Meta{Type: "deck"}); err == nil {
		look = map[string]string{"theme": d.Meta.Theme, "fonts": d.Meta.Fonts, "accent": d.Meta.Accent, "mode": d.Meta.Mode}
	}
	page := []byte{}
	problems := []problemJSON{}
	if _, w := sh.Steps[0].Attrs["window"]; w || sh.ID != "" {
		if d, err := deck.ParseSource(abs, codeshow.ShowSource(text, sh, look), deck.Meta{Type: "article"}); err == nil {
			res := render.Document(d, render.Options{Media: render.MediaInline, ShowProblems: true})
			page = res.HTML
			for _, p := range res.Problems {
				problems = append(problems, problemJSON{a.rel(p.File), p.Line, p.Msg})
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"text": text, "line": sh.Steps[0].Start + 1, "show": sh, "html": string(page), "problems": problems})
}

// codewinExport runs `si code <file> --png|--svg|--gif` into the file's folder.
func (a *api) codewinExport(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path   string `json:"path"`
		Format string `json:"format"`
		ID     string `json:"id"`
	}
	if !decode(w, r, &req) {
		return
	}
	abs, ok := a.resolve(w, req.Path)
	if !ok {
		return
	}
	if req.Format != "png" && req.Format != "svg" && req.Format != "gif" {
		fail(w, http.StatusBadRequest, "format is png, svg or gif")
		return
	}
	if a.run == nil {
		fail(w, http.StatusNotImplemented, "exports are not available")
		return
	}
	args := []string{"code", abs, "--" + req.Format, "-o", filepath.Dir(abs)}
	if req.ID != "" {
		args = append(args, "--id", req.ID)
	}
	var out bytes.Buffer
	code := a.run(args, &out)
	writeJSON(w, http.StatusOK, map[string]any{"code": code, "output": out.String()})
}
