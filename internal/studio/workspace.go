package studio

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

// ErrOutside means a path escapes the workspace root.
var ErrOutside = errors.New("path outside workspace")

// Workspace is the directory tree the editor may read and write.
type Workspace struct {
	Root string // absolute, symlinks evaluated
}

// OpenWorkspace resolves dir to an absolute, symlink-free root.
func OpenWorkspace(dir string) (*Workspace, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	root, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return nil, err
	}
	return &Workspace{Root: root}, nil
}

// Resolve maps a slash-separated relative path to an absolute one inside
// Root. The path may not exist yet; its deepest existing ancestor is
// symlink-evaluated and must stay inside Root.
func (w *Workspace) Resolve(rel string) (string, error) {
	if filepath.IsAbs(rel) || strings.HasPrefix(rel, "/") || strings.ContainsRune(rel, 0) {
		return "", ErrOutside
	}
	p := filepath.Join(w.Root, filepath.FromSlash(rel)) // Join cleans ".."
	if !w.inside(p) {
		return "", ErrOutside
	}
	// Walk up to the deepest existing ancestor (Lstat so dangling links count).
	anc, tail := p, ""
	for {
		if _, err := os.Lstat(anc); err == nil {
			break
		}
		parent := filepath.Dir(anc)
		if parent == anc {
			break
		}
		tail = filepath.Join(filepath.Base(anc), tail)
		anc = parent
	}
	real, err := filepath.EvalSymlinks(anc)
	if err != nil {
		return "", ErrOutside // dangling or looping link: refuse
	}
	if !w.inside(real) {
		return "", ErrOutside
	}
	return filepath.Join(real, tail), nil
}

func (w *Workspace) inside(p string) bool {
	return p == w.Root || strings.HasPrefix(p, w.Root+string(filepath.Separator))
}

// Rel returns abs relative to Root, slash-separated ("." for the root).
func (w *Workspace) Rel(abs string) string {
	r, err := filepath.Rel(w.Root, abs)
	if err != nil {
		return filepath.ToSlash(abs)
	}
	return filepath.ToSlash(r)
}

// Node is one entry of the workspace tree.
type Node struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Kind     string `json:"kind"`
	Children []Node `json:"children,omitempty"`
}

var skipDirs = map[string]bool{"node_modules": true, "html": true, "slides": true}

// Tree lists the workspace: dirs first, then files, each alphabetical.
func (w *Workspace) Tree() (Node, error) {
	root := Node{Name: filepath.Base(w.Root), Path: "", Kind: "dir"}
	kids, err := w.walk(w.Root)
	root.Children = kids
	return root, err
}

func (w *Workspace) walk(dir string) ([]Node, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	decks := map[string]bool{}
	// Files under a "sections" dir are include modules, never decks.
	if filepath.Base(dir) != "sections" {
		list, _ := deck.List(dir)
		for _, d := range list {
			decks[filepath.Base(d)] = true
		}
	}
	var dirs, files []Node
	for _, e := range entries {
		n := e.Name()
		if strings.HasPrefix(n, ".") {
			continue
		}
		node := Node{Name: n, Path: w.Rel(filepath.Join(dir, n))}
		if e.IsDir() {
			if skipDirs[n] {
				continue
			}
			node.Kind = "dir"
			if node.Children, err = w.walk(filepath.Join(dir, n)); err != nil {
				return nil, err
			}
			dirs = append(dirs, node)
			continue
		}
		node.Kind = fileKind(n, decks[n])
		files = append(files, node)
	}
	byName := func(s []Node) { sort.Slice(s, func(i, j int) bool { return s[i].Name < s[j].Name }) }
	byName(dirs)
	byName(files)
	return append(dirs, files...), nil
}

func fileKind(name string, isDeck bool) string {
	if isDeck {
		return "deck"
	}
	switch strings.ToLower(filepath.Ext(name)) {
	case ".md", ".adoc":
		return "markdown"
	case ".svg":
		return "svg"
	case ".css":
		return "css"
	case ".json":
		return "json"
	case ".png", ".jpg", ".jpeg", ".gif", ".webp":
		return "image"
	}
	return "other"
}
