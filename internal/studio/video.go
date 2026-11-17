package studio

import (
	"bytes"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/felipeg48/simple-ideas/internal/config"
	"github.com/felipeg48/simple-ideas/internal/videdit"
)

// The Video view: a `type: video` file as its si play preview, drawn from
// the editor's text (POST video), with the Studio's hooks (videdit edit.js):
// a callout dragged on the frame comes back as a videdit.Move, applied here
// to the text, and the UI puts the new text in the editor as one undoable
// change. ⌘B builds the saved file with si build (POST video/build starts it,
// GET video/build reports its log and, once done, the video it wrote).

type videoReq struct {
	Path string        `json:"path"`
	Text string        `json:"text"`
	Move *videdit.Move `json:"move"`
}

func (a *api) video(w http.ResponseWriter, r *http.Request) {
	var req videoReq
	if !decode(w, r, &req) {
		return
	}
	abs, ok := a.resolve(w, req.Path)
	if !ok {
		return
	}
	text := req.Text
	if req.Move != nil {
		next, err := req.Move.Apply(text)
		if err != nil {
			fail(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		text = next
	}
	dir := path.Dir("/" + filepath.ToSlash(a.ws.Rel(abs)))
	base := (&url.URL{Path: strings.TrimSuffix(dir, "/") + "/"}).EscapedPath()
	cfg := config.Load(filepath.Join(a.ws.Root, "si.conf"))
	page := videdit.PreviewText(abs, text, cfg, videdit.PreviewOptions{Base: base, Edit: true})
	e := videdit.Parse(abs, text)
	writeJSON(w, http.StatusOK, map[string]any{"text": text, "html": string(page), "out": a.ws.Rel(e.OutPath(""))})
}

// videoProblems are si check's problems of a video edit (its lines, files,
// times and places), as the problems panel lists them.
func (a *api) videoProblems(abs string) []problemJSON {
	out := []problemJSON{}
	e, err := videdit.ParseFile(abs)
	if err != nil {
		return append(out, problemJSON{a.rel(abs), 0, err.Error()})
	}
	e.Check(videdit.FFprobe())
	for _, p := range e.Problems {
		msg := p.Msg
		if p.Warn {
			msg = "warning: " + msg
		}
		out = append(out, problemJSON{a.rel(abs), p.Line, msg})
	}
	return out
}

// builds are the video builds the Studio started, by file.
var builds = struct {
	sync.Mutex
	m map[string]*videoBuild
}{m: map[string]*videoBuild{}}

type videoBuild struct {
	mu      sync.Mutex
	out     bytes.Buffer
	running bool
	code    int
}

func (b *videoBuild) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.out.Write(p)
}

func (b *videoBuild) state() map[string]any {
	b.mu.Lock()
	defer b.mu.Unlock()
	return map[string]any{"running": b.running, "code": b.code, "output": b.out.String()}
}

func (a *api) videoBuild(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
	}
	if !decode(w, r, &req) {
		return
	}
	abs, ok := a.resolve(w, req.Path)
	if !ok {
		return
	}
	if !videdit.Is(abs) {
		fail(w, http.StatusBadRequest, "not a video edit (type: video)")
		return
	}
	if a.run == nil {
		fail(w, http.StatusNotImplemented, "builds are not available")
		return
	}
	builds.Lock()
	b := builds.m[abs]
	if b == nil || !b.state()["running"].(bool) {
		b = &videoBuild{running: true}
		builds.m[abs] = b
		go func() {
			code := a.run([]string{"build", abs}, b)
			b.mu.Lock()
			b.running, b.code = false, code
			b.mu.Unlock()
		}()
	}
	builds.Unlock()
	writeJSON(w, http.StatusOK, b.state())
}

func (a *api) videoBuildState(w http.ResponseWriter, r *http.Request) {
	abs, ok := a.resolve(w, r.URL.Query().Get("path"))
	if !ok {
		return
	}
	builds.Lock()
	b := builds.m[abs]
	builds.Unlock()
	if b == nil {
		writeJSON(w, http.StatusOK, map[string]any{"running": false, "code": -1, "output": ""})
		return
	}
	writeJSON(w, http.StatusOK, b.state())
}
