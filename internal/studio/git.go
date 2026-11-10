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

	"github.com/felipeg48/simple-ideas/internal/tools"
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
	mux.HandleFunc("GET "+apiPrefix+"git/branches", a.gitBranches)
	mux.HandleFunc("POST "+apiPrefix+"git/branch", a.gitBranch)
	mux.HandleFunc("POST "+apiPrefix+"git/pull", a.gitPull)
	mux.HandleFunc("POST "+apiPrefix+"git/push", a.gitPush)
}

var revRE = regexp.MustCompile(`^[0-9a-f]{7,40}$`)

// git runs git in the workspace; out is stdout, err carries stderr.
func (a *api) git(args ...string) ([]byte, error) { return a.gitWait(20*time.Second, args...) }

// gitWait is git with a time limit (pull and push talk to the network).
func (a *api) gitWait(limit time.Duration, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), limit)
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
		g, _ := tools.Get("git")
		writeJSON(w, http.StatusOK, map[string]any{"git": false, "repo": false, "install": g.How(), "more": g.URL})
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

// branchRE is a branch name the Studio accepts (git check-ref-format has the
// last word); never one that reads as an option.
var branchRE = regexp.MustCompile(`^[A-Za-z0-9._][A-Za-z0-9._/-]{0,99}$`)

// gitBranches is the local branches, the current one, and how far it is from
// its upstream (ahead/behind; no upstream: "").
func (a *api) gitBranches(w http.ResponseWriter, r *http.Request) {
	if !a.inRepo() {
		fail(w, http.StatusConflict, "not a git repository")
		return
	}
	out, err := a.git("for-each-ref", "--format=%(refname:short)", "refs/heads/")
	if err != nil {
		fail(w, http.StatusInternalServerError, err.Error())
		return
	}
	branches := []string{}
	for _, b := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if b != "" {
			branches = append(branches, b)
		}
	}
	cur, _ := a.git("rev-parse", "--abbrev-ref", "HEAD")
	res := map[string]any{"current": strings.TrimSpace(string(cur)), "branches": branches}
	if up, err := a.git("rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}"); err == nil {
		res["upstream"] = strings.TrimSpace(string(up))
		if c, err := a.git("rev-list", "--left-right", "--count", "@{u}...HEAD"); err == nil {
			if f := strings.Fields(string(c)); len(f) == 2 {
				res["behind"], res["ahead"] = f[0], f[1]
			}
		}
	}
	remotes, _ := a.git("remote")
	res["remote"] = strings.TrimSpace(string(remotes)) != ""
	writeJSON(w, http.StatusOK, res)
}

// gitBranch switches to a branch, or makes one from here and switches to it
// (create). Uncommitted changes go along; git refuses when they would be lost.
func (a *api) gitBranch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name   string `json:"name"`
		Create bool   `json:"create"`
	}
	if !decode(w, r, &req) {
		return
	}
	name := strings.TrimSpace(req.Name)
	if !branchRE.MatchString(name) {
		fail(w, http.StatusBadRequest, "a branch name is letters, digits, . _ / -")
		return
	}
	if _, err := a.git("check-ref-format", "--branch", name); err != nil {
		fail(w, http.StatusBadRequest, "not a valid branch name: "+name)
		return
	}
	args := []string{"switch", "-q", name}
	if req.Create {
		args = []string{"switch", "-q", "-c", name}
	}
	if _, err := a.git(args...); err != nil {
		fail(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"current": name})
}

// gitPull brings the upstream's commits in when that needs no merge
// (--ff-only): the Studio never makes a merge behind the author's back.
func (a *api) gitPull(w http.ResponseWriter, r *http.Request) {
	if _, err := a.gitWait(90*time.Second, "pull", "--ff-only", "-q"); err != nil {
		fail(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// gitPush sends the branch to its upstream; a branch without one goes to
// origin (or the only remote) and remembers it.
func (a *api) gitPush(w http.ResponseWriter, r *http.Request) {
	args := []string{"push", "-q"}
	if _, err := a.git("rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}"); err != nil {
		out, _ := a.git("remote")
		rs := strings.Fields(string(out))
		remote := ""
		for _, x := range rs {
			if x == "origin" {
				remote = x
			}
		}
		if remote == "" && len(rs) == 1 {
			remote = rs[0]
		}
		if remote == "" {
			fail(w, http.StatusConflict, "this repository has no remote to push to (add one with git remote add origin <url>)")
			return
		}
		cur, _ := a.git("rev-parse", "--abbrev-ref", "HEAD")
		args = append(args, "-u", remote, strings.TrimSpace(string(cur)))
	}
	if _, err := a.gitWait(90*time.Second, args...); err != nil {
		fail(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
