package studio

import (
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/felipeg48/simple-ideas/internal/mindmap"
	"github.com/felipeg48/simple-ideas/internal/render"
)

// The map editor: the canvas (a mind map page or fence, drawn by the real
// runtime with mindmap-edit.js) sends ops; POST mindmap applies one to the
// editor's text through internal/mindmap and answers with the new text and
// the maps to redraw. The UI puts the text in the editor as one undoable
// change and saves it the usual way; nothing here writes files.

type mindmapReq struct {
	Path string      `json:"path"`
	Text string      `json:"text"`
	Line int         `json:"line"` // > 0: the ```mindmap fence holding this line
	Op   *mindmap.Op `json:"op"`
}

func (a *api) mindmap(w http.ResponseWriter, r *http.Request) {
	var req mindmapReq
	if !decode(w, r, &req) {
		return
	}
	abs, ok := a.resolve(w, req.Path)
	if !ok {
		return
	}
	text := req.Text
	f, inFence := mindmap.Fence{}, false
	if req.Line > 0 {
		if f, inFence = mindmap.FenceAt(text, req.Line); !inFence {
			fail(w, http.StatusUnprocessableEntity, "no mind map fence at that line")
			return
		}
	}
	src := text
	if inFence {
		src = f.Body
	}
	doc := mindmap.Parse(src)
	if req.Op != nil && req.Op.Kind == "copy" { // the selected branches as a Markdown list; nothing changes
		text, err := doc.Branches(req.Op.Map, req.Op.Paths)
		if err != nil {
			fail(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"copy": text})
		return
	}
	var sel *mindmap.Sel
	if req.Op != nil {
		if inFence && req.Op.Kind == "meta" {
			text = f.Replace(text, mindmap.SetInfo(f.Info, req.Op.Field, req.Op.Value), f.Body)
		} else {
			s, err := doc.Apply(*req.Op)
			if err != nil {
				fail(w, http.StatusUnprocessableEntity, err.Error())
				return
			}
			sel = &s
			if inFence {
				text = f.Replace(text, f.Info, doc.String())
			} else {
				text = doc.String()
				if !strings.HasSuffix(req.Text, "\n") {
					text = strings.TrimSuffix(text, "\n")
				}
			}
		}
		if inFence {
			f, _ = mindmap.FenceAt(text, f.Start+1)
			doc = mindmap.Parse(f.Body)
		}
	}
	d, err := deck.ParseSource(abs, text, deck.Meta{Type: "deck"})
	if err != nil {
		fail(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	opt := render.Options{ShowProblems: true, Media: render.MediaInline}
	var view render.MindmapView
	front := doc.FrontMap()
	if inFence {
		view = render.MindmapEdit(d, f.Body, f.Info, opt)
		front = mindmap.InfoMap(f.Info)
	} else {
		view = render.MindmapEdit(d, "", "", opt)
	}
	problems := []problemJSON{}
	for _, p := range view.Problems {
		problems = append(problems, problemJSON{a.rel(p.File), p.Line, p.Msg})
	}
	maps := doc.Maps()
	if maps == nil {
		maps = []*mindmap.Map{}
	}
	out := map[string]any{"text": text, "data": view.Data, "opts": view.Opts, "problems": problems,
		"maps": maps, "front": front, "fence": inFence, "select": sel}
	if inFence {
		out["fenceLine"] = f.Start + 1
	}
	writeJSON(w, http.StatusOK, out)
}

// mmcanvas is the editor's canvas page for a saved file (the UI then sends
// the live text's maps with postMessage).
func (a *api) mmcanvas(w http.ResponseWriter, r *http.Request) {
	abs, ok := a.resolve(w, r.URL.Query().Get("path"))
	if !ok {
		return
	}
	raw, err := os.ReadFile(abs)
	if err != nil {
		fail(w, http.StatusNotFound, "not found")
		return
	}
	text := string(raw)
	body, info := "", ""
	if line, _ := strconv.Atoi(r.URL.Query().Get("line")); line > 0 {
		if f, ok := mindmap.FenceAt(text, line); ok {
			body, info = f.Body, f.Info
		}
	}
	d, err := deck.ParseSource(abs, text, deck.Meta{Type: "deck"})
	if err != nil {
		fail(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Write(render.MindmapCanvas(d, body, info, render.Options{Media: render.MediaInline}))
}
