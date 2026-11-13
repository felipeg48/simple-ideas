package serve

// The play terminal's Files panel: a read-only tree of a folder on the
// presenter's machine and a viewer for one file (highlighted text, or the
// picture). Loopback-only and same-origin, like the shells it sits beside,
// which can already read every file the presenter can.

import (
	"bytes"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/felipeg48/simple-ideas/internal/render"
)

const (
	filesMaxEntries = 2000
	filesMaxBytes   = 1 << 20 // the viewer shows the first MB of a file
	filesMaxSave    = 5 << 20 // the editor saves files up to 5 MB
)

func mtimeOf(st os.FileInfo) string { return strconv.FormatInt(st.ModTime().UnixNano(), 10) }

type fileEntry struct {
	Name string `json:"name"`
	Dir  bool   `json:"dir"`
	Size int64  `json:"size"`
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(v)
}

// absDir is ?dir= (or ?path=) made absolute; empty means the deck's folder.
func (h *termHub) absPath(r *http.Request, key string) string {
	return h.resolve(r.URL.Query().Get(key))
}

func (h *termHub) resolve(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return h.dir
	}
	if strings.HasPrefix(p, "~") {
		if home, err := os.UserHomeDir(); err == nil {
			p = home + p[1:]
		}
	}
	if !filepath.IsAbs(p) {
		p = filepath.Join(h.dir, p)
	}
	return filepath.Clean(p)
}

// ls: GET ?dir= -> {dir, parent, entries} (folders first, then files, by name).
func (h *termHub) ls(w http.ResponseWriter, r *http.Request) {
	dir := h.absPath(r, "dir")
	ents, err := os.ReadDir(dir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	out := make([]fileEntry, 0, len(ents))
	for _, e := range ents {
		fe := fileEntry{Name: e.Name(), Dir: e.IsDir()}
		if info, err := e.Info(); err == nil {
			if info.Mode()&os.ModeSymlink != 0 { // a link to a folder opens like one
				if st, err := os.Stat(filepath.Join(dir, e.Name())); err == nil {
					fe.Dir = st.IsDir()
				}
			}
			if !fe.Dir {
				fe.Size = info.Size()
			}
		}
		out = append(out, fe)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Dir != out[j].Dir {
			return out[i].Dir
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	more := len(out) > filesMaxEntries
	if more {
		out = out[:filesMaxEntries]
	}
	writeJSON(w, map[string]any{"dir": dir, "parent": filepath.Dir(dir), "entries": out, "more": more})
}

// fileLang picks the highlighter language from a file's name.
func fileLang(name string) string {
	base := strings.ToLower(filepath.Base(name))
	switch base {
	case "dockerfile", "containerfile":
		return "dockerfile"
	case "makefile":
		return "bash"
	}
	return strings.TrimPrefix(filepath.Ext(base), ".")
}

// cat: GET ?path= -> {path, size, mtime, html, lines, binary, truncated, image};
// with ?edit=1 also text, the whole file, when it fit and is text.
func (h *termHub) cat(w http.ResponseWriter, r *http.Request) {
	p := h.absPath(r, "path")
	st, err := os.Stat(p)
	if err != nil || st.IsDir() {
		http.Error(w, "not a file", http.StatusNotFound)
		return
	}
	res := map[string]any{"path": p, "size": st.Size(), "mtime": mtimeOf(st)}
	if ct := mime.TypeByExtension(strings.ToLower(filepath.Ext(p))); strings.HasPrefix(ct, "image/") {
		res["image"] = true
		writeJSON(w, res)
		return
	}
	f, err := os.Open(p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	defer f.Close()
	buf := make([]byte, filesMaxBytes)
	n, _ := f.Read(buf)
	for n < len(buf) {
		m, err := f.Read(buf[n:])
		n += m
		if err != nil || m == 0 {
			break
		}
	}
	buf = buf[:n]
	head := buf[:min(len(buf), 8192)]
	if bytes.IndexByte(head, 0) >= 0 || !utf8.Valid(bytes.ToValidUTF8(head, nil)) {
		res["binary"] = true
		writeJSON(w, res)
		return
	}
	text := strings.ToValidUTF8(string(buf), "�")
	res["truncated"] = st.Size() > int64(len(buf))
	res["lines"] = strings.Count(text, "\n") + 1
	res["html"] = render.Highlight(text, fileLang(p))
	if r.URL.Query().Get("edit") == "1" && !res["truncated"].(bool) && utf8.Valid(buf) {
		res["text"] = string(buf)
	}
	writeJSON(w, res)
}

// save: POST {path, text, mtime, force} writes a file from the Viewer's editor:
// refused (409) when the file changed since it was read, unless force; written
// to a temp file in the same folder and renamed over, keeping the mode.
func (h *termHub) save(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path  string `json:"path"`
		Text  string `json:"text"`
		Mtime string `json:"mtime"`
		Force bool   `json:"force"`
	}
	b, err := io.ReadAll(io.LimitReader(r.Body, filesMaxSave*2))
	if err != nil || json.Unmarshal(b, &req) != nil || strings.TrimSpace(req.Path) == "" {
		http.Error(w, "bad save request", http.StatusBadRequest)
		return
	}
	if len(req.Text) > filesMaxSave {
		http.Error(w, "too large: the editor saves files up to 5 MB", http.StatusRequestEntityTooLarge)
		return
	}
	p := h.resolve(req.Path)
	mode := os.FileMode(0o644)
	st, err := os.Stat(p)
	switch {
	case err == nil && !st.Mode().IsRegular():
		http.Error(w, "not a regular file: "+p, http.StatusBadRequest)
		return
	case err == nil:
		mode = st.Mode().Perm()
		if !req.Force && mtimeOf(st) != req.Mtime {
			http.Error(w, "the file changed on disk since it was opened", http.StatusConflict)
			return
		}
	case !os.IsNotExist(err):
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	// the real file behind a link is the one rewritten
	if real, err := filepath.EvalSymlinks(p); err == nil {
		p = real
	}
	tmp, err := os.CreateTemp(filepath.Dir(p), "."+filepath.Base(p)+".si-*")
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	_, err = tmp.WriteString(req.Text)
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Chmod(tmp.Name(), mode)
	}
	if err == nil {
		h.quiet(func() { err = os.Rename(tmp.Name(), p) })
	}
	if err != nil {
		os.Remove(tmp.Name())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	st, err = os.Stat(p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"path": p, "mtime": mtimeOf(st), "size": st.Size()})
}

// raw: GET ?path= serves a picture for the viewer (images only).
func (h *termHub) raw(w http.ResponseWriter, r *http.Request) {
	p := h.absPath(r, "path")
	ct := mime.TypeByExtension(strings.ToLower(filepath.Ext(p)))
	if !strings.HasPrefix(ct, "image/") {
		http.Error(w, "pictures only", http.StatusForbidden)
		return
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'") // an SVG shows, never runs
	http.ServeFile(w, r, p)
}

// cwd: GET ?id= -> {dir}: the folder a tab's shell is in now.
func (h *termHub) cwd(w http.ResponseWriter, r *http.Request) {
	s := h.get(r)
	if s == nil || s.cmd.Process == nil {
		http.Error(w, "no such tab", http.StatusNotFound)
		return
	}
	pid := s.cmd.Process.Pid
	dir := ""
	if runtime.GOOS == "linux" {
		dir, _ = os.Readlink("/proc/" + strconv.Itoa(pid) + "/cwd")
	} else if out, err := exec.Command("lsof", "-a", "-p", strconv.Itoa(pid), "-d", "cwd", "-Fn").Output(); err == nil {
		for _, l := range strings.Split(string(out), "\n") {
			if strings.HasPrefix(l, "n") {
				dir = l[1:]
			}
		}
	}
	if dir == "" {
		dir = h.dir
	}
	writeJSON(w, map[string]string{"dir": dir})
}
