package studio

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// History: git in the workspace, for the Studio's History panel. Every call
// runs git with fixed arguments (no shell), in the workspace root; paths come
// through resolve (confined, no dot segments) and go after `--`; revisions
// must be hex hashes.

func (a *api) gitRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET "+apiPrefix+"git/status", a.gitStatus)
	mux.HandleFunc("GET "+apiPrefix+"git/diff", a.gitDiff)
	mux.HandleFunc("GET "+apiPrefix+"git/log", a.gitLog)
	mux.HandleFunc("GET "+apiPrefix+"git/show", a.gitShow)
	mux.HandleFunc("POST "+apiPrefix+"git/restore", a.gitRestore)
	mux.HandleFunc("POST "+apiPrefix+"git/commit", a.gitCommit)
	mux.HandleFunc("POST "+apiPrefix+"git/init", a.gitInit)
}

var revRE = regexp.MustCompile(`^[0-9a-f]{7,40}$`)

// git runs git in the workspace; out is stdout, err carries stderr.
func (a *api) git(args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", append([]string{"-c", "core.quotepath=off", "-c", "color.ui=false"}, args...)...)
	cmd.Dir = a.ws.Root
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_OPTIONAL_LOCKS=0")
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) && ee.ExitCode() == 1 && len(args) > 0 && args[0] == "diff" {
			return out.Bytes(), nil // diff --no-index: 1 means "they differ"
		}
		msg := strings.TrimSpace(errb.String())
		if msg == "" {
			msg = err.Error()
		}
		return out.Bytes(), errors.New(msg)
	}
	return out.Bytes(), nil
}

func (a *api) inRepo() bool {
	out, err := a.git("rev-parse", "--is-inside-work-tree")
	return err == nil && strings.TrimSpace(string(out)) == "true"
}

// gitFile is one changed file, its path relative to the workspace.
type gitFile struct {
	Path   string `json:"path"`
	Status string `json:"status"` // modified, added, deleted, renamed, untracked
	Staged bool   `json:"staged"`
}

func (a *api) gitStatus(w http.ResponseWriter, r *http.Request) {
	if _, err := exec.LookPath("git"); err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"git": false, "repo": false})
		return
	}
	if !a.inRepo() {
		writeJSON(w, http.StatusOK, map[string]any{"git": true, "repo": false})
		return
	}
	prefix, _ := a.git("rev-parse", "--show-prefix")
	pre := strings.TrimSpace(string(prefix))
	out, err := a.git("status", "--porcelain=v1", "-z", "--untracked-files=all", "--", ".")
	if err != nil {
		fail(w, http.StatusInternalServerError, err.Error())
		return
	}
	files := []gitFile{}
	parts := strings.Split(string(out), "\x00")
	for i := 0; i < len(parts); i++ {
		e := parts[i]
		if len(e) < 4 {
			continue
		}
		x, y, p := e[0], e[1], e[3:]
		if x == 'R' || x == 'C' {
			i++ // the old name follows
		}
		p = strings.TrimPrefix(p, pre)
		if strings.HasPrefix(p, ".") || strings.Contains(p, "/.") {
			continue
		}
		st := "modified"
		switch {
		case x == '?':
			st = "untracked"
		case x == 'A' || y == 'A':
			st = "added"
		case x == 'D' || y == 'D':
			st = "deleted"
		case x == 'R':
			st = "renamed"
		}
		files = append(files, gitFile{Path: p, Status: st, Staged: x != ' ' && x != '?'})
	}
	branch, _ := a.git("rev-parse", "--abbrev-ref", "HEAD")
	writeJSON(w, http.StatusOK, map[string]any{"git": true, "repo": true, "branch": strings.TrimSpace(string(branch)), "files": files})
}

// gitPath confines a request's path and gives it to git relative to the root.
func (a *api) gitPath(w http.ResponseWriter, rel string) (string, bool) {
	abs, ok := a.resolve(w, rel)
	if !ok {
		return "", false
	}
	p, err := filepath.Rel(a.ws.Root, abs)
	if err != nil {
		fail(w, http.StatusForbidden, "outside workspace")
		return "", false
	}
	return filepath.ToSlash(p), true
}

func (a *api) gitDiff(w http.ResponseWriter, r *http.Request) {
	p, ok := a.gitPath(w, r.URL.Query().Get("path"))
	if !ok {
		return
	}
	out, err := a.git("diff", "--no-color", "HEAD", "--", p)
	if err != nil || len(out) == 0 { // a new file, or a repo without commits: all of it is new
		if tracked, _ := a.git("ls-files", "--", p); len(bytes.TrimSpace(tracked)) == 0 {
			out, err = a.git("diff", "--no-color", "--no-index", "--", os.DevNull, p)
		}
	}
	if err != nil {
		fail(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"path": r.URL.Query().Get("path"), "diff": string(out)})
}

type gitCommit struct {
	Hash    string `json:"hash"`
	Short   string `json:"short"`
	Author  string `json:"author"`
	Date    string `json:"date"`
	Subject string `json:"subject"`
}

func (a *api) gitLog(w http.ResponseWriter, r *http.Request) {
	args := []string{"log", "-n", "50", "--date=short", "--format=%H%x1f%h%x1f%an%x1f%ad%x1f%s"}
	if rel := r.URL.Query().Get("path"); rel != "" {
		p, ok := a.gitPath(w, rel)
		if !ok {
			return
		}
		args = append(args, "--follow", "--", p)
	}
	out, err := a.git(args...)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"commits": []gitCommit{}, "note": err.Error()})
		return
	}
	commits := []gitCommit{}
	for _, l := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		f := strings.Split(l, "\x1f")
		if len(f) == 5 {
			commits = append(commits, gitCommit{f[0], f[1], f[2], f[3], f[4]})
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"commits": commits})
}

// atRev is the file's text at a commit.
func (a *api) atRev(w http.ResponseWriter, rel, rev string) (string, string, bool) {
	if !revRE.MatchString(rev) {
		fail(w, http.StatusBadRequest, "a commit is a hex hash")
		return "", "", false
	}
	p, ok := a.gitPath(w, rel)
	if !ok {
		return "", "", false
	}
	out, err := a.git("show", rev+":./"+p)
	if err != nil {
		fail(w, http.StatusNotFound, err.Error())
		return "", "", false
	}
	return p, string(out), true
}

func (a *api) gitShow(w http.ResponseWriter, r *http.Request) {
	if _, text, ok := a.atRev(w, r.URL.Query().Get("path"), r.URL.Query().Get("rev")); ok {
		writeJSON(w, http.StatusOK, map[string]any{"text": text})
	}
}

// gitRestore writes the file as it was at a commit (the editor then shows it
// as an ordinary change: undoable, unsaved until autosave).
func (a *api) gitRestore(w http.ResponseWriter, r *http.Request) {
	var req struct{ Path, Rev string }
	if !decode(w, r, &req) {
		return
	}
	_, text, ok := a.atRev(w, req.Path, req.Rev)
	if !ok {
		return
	}
	abs, _ := a.ws.Resolve(req.Path)
	if err := os.WriteFile(abs, []byte(text), 0o644); err != nil {
		fail(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"text": text})
}

func (a *api) gitCommit(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Message string   `json:"message"`
		Paths   []string `json:"paths"`
	}
	if !decode(w, r, &req) {
		return
	}
	msg := strings.TrimSpace(req.Message)
	if msg == "" || len(req.Paths) == 0 {
		fail(w, http.StatusBadRequest, "a commit needs a message and at least one file")
		return
	}
	var ps []string
	for _, rel := range req.Paths {
		p, ok := a.gitPath(w, rel)
		if !ok {
			return
		}
		ps = append(ps, p)
	}
	if _, err := a.git(append([]string{"add", "-A", "--"}, ps...)...); err != nil {
		fail(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if _, err := a.git(append([]string{"commit", "-q", "-m", msg, "--"}, ps...)...); err != nil {
		fail(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	head, _ := a.git("rev-parse", "--short", "HEAD")
	writeJSON(w, http.StatusOK, map[string]any{"commit": strings.TrimSpace(string(head))})
}

func (a *api) gitInit(w http.ResponseWriter, r *http.Request) {
	if a.inRepo() {
		fail(w, http.StatusConflict, "already a git repository")
		return
	}
	if _, err := a.git("init", "-q"); err != nil {
		fail(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"repo": true})
}
