package studio

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

// Starters are the small files "New…" writes: one file for a deck, article,
// lab, demo, site page or mind map; a folder for a guide, book or site. {{title}},
// {{date}} and {{year}} are filled in.
//
//go:embed all:starters
var startersFS embed.FS

// starterFolder reports whether a kind is a folder of files.
func starterFolder(kind string) bool { return kind == "guide" || kind == "book" || kind == "site" }

// starterFiles lists a kind's files: starter path → path relative to the target.
func starterFiles(kind string) map[string]string {
	out := map[string]string{}
	if !starterFolder(kind) {
		if _, err := fs.Stat(startersFS, "starters/"+kind+".md"); err == nil {
			out["starters/"+kind+".md"] = ""
		}
		return out
	}
	fs.WalkDir(startersFS, "starters/"+kind, func(p string, e fs.DirEntry, err error) error {
		if err == nil && !e.IsDir() {
			out[p] = strings.TrimPrefix(p, "starters/"+kind+"/")
		}
		return nil
	})
	return out
}

// ErrStarterExists is returned when a starter would overwrite a file.
var ErrStarterExists = errors.New("exists")

// StarterKinds are the kinds WriteStarter knows.
var StarterKinds = []string{"deck", "article", "lab", "demo", "page", "guide", "book", "site", "mindmap"}

// WriteStarter writes the starter for kind at abs (a .md file, or a folder for
// guide, book and site), never overwriting. A page or folder made inside a
// guide, book or site (found from abs up to root) joins its menu. It returns
// the files written, the file to open, and the nav file it linked into ("").
func WriteStarter(kind, abs, title, root string) (created []string, open, linked string, err error) {
	files := starterFiles(kind)
	if len(files) == 0 {
		return nil, "", "", fmt.Errorf("unknown kind %q (%s)", kind, strings.Join(StarterKinds, ", "))
	}
	title = strings.TrimSpace(title)
	if title == "" {
		title = strings.ReplaceAll(strings.TrimSuffix(filepath.Base(abs), ".md"), "-", " ")
		title = strings.ToUpper(title[:1]) + title[1:]
	}
	now := time.Now()
	fill := strings.NewReplacer("{{title}}", title, "{{date}}", now.Format("2006-01-02"), "{{year}}", strconv.Itoa(now.Year()))
	// check every target first: a starter never overwrites and never half-writes
	targets := map[string]string{}
	for src, rel := range files {
		t := abs
		if rel != "" {
			t = filepath.Join(abs, filepath.FromSlash(rel))
		}
		if _, err := os.Lstat(t); err == nil {
			return nil, "", "", fmt.Errorf("%s %w", t, ErrStarterExists)
		}
		targets[src] = t
	}
	for src, t := range targets {
		raw, _ := startersFS.ReadFile(src)
		if err := os.MkdirAll(filepath.Dir(t), 0o755); err != nil {
			return created, "", "", err
		}
		f, err := os.OpenFile(t, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if errors.Is(err, os.ErrExist) {
			return created, "", "", fmt.Errorf("%s %w", t, ErrStarterExists)
		}
		if err == nil {
			_, err = f.WriteString(fill.Replace(string(raw)))
			if cerr := f.Close(); err == nil {
				err = cerr
			}
		}
		if err != nil {
			return created, "", "", err
		}
		created = append(created, t)
	}
	open = abs // the file to open: the file itself, or the folder's first page
	if starterFolder(kind) {
		open = filepath.Join(abs, "index.md")
		if kind == "book" {
			open = filepath.Join(abs, "preface.md")
		}
	}
	probe := abs // FindGuide looks from the folder of the file it is given
	if starterFolder(kind) {
		probe = filepath.Join(filepath.Dir(abs), "_")
	}
	if g := deck.FindGuide(probe, root); g != nil {
		rel, _ := filepath.Rel(g.Dir, abs)
		rel = filepath.ToSlash(rel)
		if starterFolder(kind) {
			rel += "/"
		}
		nav := navPath(g)
		if raw, err := os.ReadFile(nav); err == nil {
			s := strings.TrimRight(string(raw), "\n") + "\n- [" + title + "](" + rel + ")\n"
			if os.WriteFile(nav, []byte(s), 0o644) == nil {
				linked = nav
			}
		}
	}
	return created, open, linked, nil
}

func (a *api) starter(w http.ResponseWriter, r *http.Request) {
	var req struct{ Kind, Path, Title string }
	if !decode(w, r, &req) {
		return
	}
	if len(starterFiles(req.Kind)) == 0 {
		fail(w, http.StatusBadRequest, "unknown kind "+req.Kind)
		return
	}
	abs, ok := a.resolve(w, req.Path)
	if !ok {
		return
	}
	created, open, linked, err := WriteStarter(req.Kind, abs, req.Title, a.ws.Root)
	if errors.Is(err, ErrStarterExists) {
		fail(w, http.StatusConflict, a.rel(strings.TrimSuffix(err.Error(), " exists"))+" exists")
		return
	}
	if err != nil {
		fail(w, http.StatusInternalServerError, err.Error())
		return
	}
	for i := range created {
		created[i] = a.rel(created[i])
	}
	if linked != "" {
		linked = a.rel(linked)
	}
	writeJSON(w, http.StatusCreated, map[string]any{"created": created, "open": a.rel(open), "linked": linked})
}
