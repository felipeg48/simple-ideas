package studio

import (
	"net/http"

	"github.com/felipeg48/simple-ideas/internal/motion"
)

// The Motion view: an SVG canvas as frames. POST motion applies one op to the
// editor's text through internal/motion and answers with the new text, the
// frame model for the board and panels, and the lines that put the animation
// on a slide; the UI puts the text in the editor as one undoable change. The
// file itself is never read or written here: the editor saves it.
func (a *api) motion(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string     `json:"path"`
		Text string     `json:"text"`
		Op   *motion.Op `json:"op"`
	}
	if !decode(w, r, &req) {
		return
	}
	abs, ok := a.resolve(w, req.Path)
	if !ok {
		return
	}
	text := req.Text
	if req.Op != nil {
		next, err := motion.Apply(text, *req.Op)
		if err != nil {
			fail(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		text = next
	}
	m := motion.Read(text)
	writeJSON(w, http.StatusOK, map[string]any{"text": text, "model": m, "snippet": motion.Snippet(a.rel(abs), m)})
}
