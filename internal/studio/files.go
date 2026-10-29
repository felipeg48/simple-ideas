package studio

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unicode/utf8"
)

const (
	maxFileSize = 5 << 20
	maxBody     = 12 << 20 // JSON escaping can inflate a 5 MiB file
	sniffLen    = 8 << 10
	apiPrefix   = "/_si/studio/api/"
)

// api holds the state shared by the studio HTTP handlers.
type api struct {
	ws *Workspace
}

func (a *api) routes(mux *http.ServeMux) {
	mux.HandleFunc("GET "+apiPrefix+"tree", a.tree)
	mux.HandleFunc("GET "+apiPrefix+"file", a.readFile)
	mux.HandleFunc("PUT "+apiPrefix+"file", a.writeFile)
	mux.HandleFunc("DELETE "+apiPrefix+"file", a.deleteFile)
	mux.HandleFunc("POST "+apiPrefix+"create", a.create)
	mux.HandleFunc("POST "+apiPrefix+"rename", a.rename)
	mux.HandleFunc("GET "+apiPrefix+"decks", a.decks)
	mux.HandleFunc("GET "+apiPrefix+"check", a.check)
	mux.HandleFunc("GET "+apiPrefix+"rules", a.rules)
}

func etagOf(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])[:16]
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func fail(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBody)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		fail(w, http.StatusBadRequest, "bad json")
		return false
	}
	return true
}

// resolve validates rel and maps it into the workspace. On failure it has
// already answered. Dot-segments (.git, .superpowers, ...) are off limits.
func (a *api) resolve(w http.ResponseWriter, rel string) (string, bool) {
	if rel == "" {
		fail(w, http.StatusBadRequest, "path required")
		return "", false
	}
	for _, seg := range strings.Split(rel, "/") {
		if strings.HasPrefix(seg, ".") {
			fail(w, http.StatusForbidden, "hidden path")
			return "", false
		}
	}
	abs, err := a.ws.Resolve(rel)
	if err != nil {
		fail(w, http.StatusForbidden, "outside workspace")
		return "", false
	}
	return abs, true
}

func (a *api) tree(w http.ResponseWriter, r *http.Request) {
	n, err := a.ws.Tree()
	if err != nil {
		fail(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, n)
}

func (a *api) readFile(w http.ResponseWriter, r *http.Request) {
	rel := r.URL.Query().Get("path")
	abs, ok := a.resolve(w, rel)
	if !ok {
		return
	}
	fi, err := os.Stat(abs)
	if err != nil {
		fail(w, http.StatusNotFound, "not found")
		return
	}
	if fi.IsDir() {
		fail(w, http.StatusBadRequest, "is a directory")
		return
	}
	if fi.Size() > maxFileSize {
		fail(w, http.StatusRequestEntityTooLarge, "too large")
		return
	}
	b, err := os.ReadFile(abs)
	if err != nil {
		fail(w, http.StatusInternalServerError, err.Error())
		return
	}
	head := b[:min(len(b), sniffLen)]
	// a multibyte rune cut at the sniff boundary is not invalid text
	for i := 0; i < 3 && len(head) < len(b) && !utf8.Valid(head); i++ {
		head = b[:len(head)+1]
	}
	if bytes.IndexByte(head, 0) >= 0 || !utf8.Valid(head) {
		fail(w, http.StatusUnsupportedMediaType, "binary")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"path": rel, "content": string(b), "etag": etagOf(b)})
}

func (a *api) writeFile(w http.ResponseWriter, r *http.Request) {
	abs, ok := a.resolve(w, r.URL.Query().Get("path"))
	if !ok {
		return
	}
	var req struct{ Content, Etag string }
	req.Etag = "\x00" // absent etag is not "create only"
	if !decode(w, r, &req) {
		return
	}
	if req.Etag == "\x00" {
		req.Etag = ""
	}
	// ponytail: check-then-rename is not atomic; an edit landing in between is lost. Needs file locks to close.
	mode := os.FileMode(0o644)
	cur, err := os.ReadFile(abs)
	switch {
	case err == nil:
		if etagOf(cur) != req.Etag {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "conflict", "etag": etagOf(cur), "content": string(cur)})
			return
		}
		if fi, err := os.Stat(abs); err == nil {
			mode = fi.Mode().Perm()
		}
	case errors.Is(err, os.ErrNotExist):
		if req.Etag != "" {
			fail(w, http.StatusNotFound, "not found")
			return
		}
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			fail(w, http.StatusInternalServerError, err.Error())
			return
		}
	default:
		fail(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := writeAtomic(abs, []byte(req.Content), mode); err != nil {
		fail(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"etag": etagOf([]byte(req.Content))})
}

// writeAtomic writes via a temp file in the same directory and a rename.
func writeAtomic(abs string, b []byte, mode os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(abs), ".si-studio-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // no-op after a successful rename
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), abs)
}

func (a *api) create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path, Content string
		Dir           bool
	}
	if !decode(w, r, &req) {
		return
	}
	abs, ok := a.resolve(w, req.Path)
	if !ok {
		return
	}
	if _, err := os.Lstat(abs); err == nil {
		fail(w, http.StatusConflict, "exists")
		return
	}
	if req.Dir {
		if err := os.MkdirAll(abs, 0o755); err != nil {
			fail(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, map[string]string{"path": req.Path})
		return
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		fail(w, http.StatusInternalServerError, err.Error())
		return
	}
	// O_EXCL: never clobber a file that appeared since the check
	f, err := os.OpenFile(abs, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if errors.Is(err, os.ErrExist) {
		fail(w, http.StatusConflict, "exists")
		return
	}
	if err == nil {
		_, err = f.WriteString(req.Content)
		if cerr := f.Close(); err == nil {
			err = cerr
		}
	}
	if err != nil {
		fail(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"path": req.Path, "etag": etagOf([]byte(req.Content))})
}

func (a *api) rename(w http.ResponseWriter, r *http.Request) {
	var req struct{ From, To string }
	if !decode(w, r, &req) {
		return
	}
	from, ok := a.resolve(w, req.From)
	if !ok {
		return
	}
	to, ok := a.resolve(w, req.To)
	if !ok {
		return
	}
	if _, err := os.Lstat(from); err != nil {
		fail(w, http.StatusNotFound, "not found")
		return
	}
	if _, err := os.Lstat(to); err == nil {
		fail(w, http.StatusConflict, "exists")
		return
	}
	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
		fail(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := os.Rename(from, to); err != nil {
		fail(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"path": req.To})
}

func (a *api) deleteFile(w http.ResponseWriter, r *http.Request) {
	abs, ok := a.resolve(w, r.URL.Query().Get("path"))
	if !ok {
		return
	}
	// os.Remove deletes files and empty dirs only, never a tree.
	err := os.Remove(abs)
	switch {
	case err == nil:
		w.WriteHeader(http.StatusNoContent)
	case errors.Is(err, os.ErrNotExist):
		fail(w, http.StatusNotFound, "not found")
	case errors.Is(err, syscall.ENOTEMPTY), errors.Is(err, syscall.EEXIST):
		fail(w, http.StatusConflict, "directory not empty")
	default:
		fail(w, http.StatusInternalServerError, err.Error())
	}
}
