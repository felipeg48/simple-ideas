package deck

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// A guide is a folder with a _nav.md: a multi-page tutorial read in order.
// _nav.md holds front matter (the guide-wide settings every page inherits) and
// a nested markdown list: a `[Title](page.md)` item is a page, a plain item a
// section heading, indentation nests.
//
//	---
//	title: Order journey
//	theme: paper
//	---
//	- [Introduction](index.md)
//	- Getting started
//	  - [Install the lab](install.md)

// NavFile is the file that makes a folder a guide.
const NavFile = "_nav.md"

// Guide is a parsed _nav.md.
type Guide struct {
	Dir      string     // the guide folder
	Meta     Meta       // _nav.md front matter: the guide title and inherited settings
	Items    []*NavItem // the top level of the nav tree
	Pages    []string   // the page files (absolute), in nav order
	Problems []Problem
	Warnings []Problem
}

// NavItem is one entry of the nav tree.
type NavItem struct {
	Title    string // link text, or the section heading
	Href     string // as written; "" for a section heading
	File     string // the page file (absolute) for a local .md/.adoc link
	Line     int
	Children []*NavItem
	Parent   *NavItem
}

var navItemRE = regexp.MustCompile(`^(\s*)[-*+]\s+(.*?)\s*$`)
var navLinkRE = regexp.MustCompile(`^\[(.+?)\]\(([^)\s]+)\)$`)

// LoadGuide parses dir/_nav.md.
func LoadGuide(dir string) (*Guide, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	nav := filepath.Join(abs, NavFile)
	raw, err := os.ReadFile(nav)
	if err != nil {
		return nil, err
	}
	g := &Guide{Dir: abs}
	fm := &Deck{Path: nav}
	lines := splitLines(strings.TrimPrefix(string(raw), "\uFEFF"))
	start := fm.frontMatter(nav, lines)
	g.Meta, g.Problems, g.Warnings = fm.Meta, fm.Problems, fm.Warnings
	if g.Meta.Title == "" {
		g.Meta.Title = filepath.Base(abs)
	}
	type level struct {
		indent int
		item   *NavItem
	}
	var stack []level
	seen := map[string]int{}
	rel := rel(nav)
	for i, l := range lines[start:] {
		no := start + i + 1
		if strings.TrimSpace(l) == "" {
			continue
		}
		m := navItemRE.FindStringSubmatch(strings.ReplaceAll(l, "\t", "  "))
		if m == nil {
			g.Problems = append(g.Problems, Problem{File: rel, Line: no, Msg: "_nav.md: want a list item (`- [Title](page.md)` or `- Section`)"})
			continue
		}
		it := &NavItem{Title: m[2], Line: no}
		if lm := navLinkRE.FindStringSubmatch(m[2]); lm != nil {
			it.Title, it.Href = lm[1], lm[2]
			if local := !strings.Contains(it.Href, "://") && !strings.HasPrefix(it.Href, "mailto:"); local {
				path, _, _ := strings.Cut(it.Href, "#")
				f := filepath.Clean(filepath.Join(abs, filepath.FromSlash(path)))
				switch {
				case !isDeckFile(filepath.Base(f)):
					g.Problems = append(g.Problems, Problem{File: rel, Line: no, Msg: fmt.Sprintf("_nav.md: %s is not a .md or .adoc page", it.Href)})
				case !strings.HasPrefix(f, abs+string(filepath.Separator)):
					g.Problems = append(g.Problems, Problem{File: rel, Line: no, Msg: fmt.Sprintf("_nav.md: %s is outside the guide folder", it.Href)})
				default:
					if _, err := os.Stat(f); err != nil {
						g.Problems = append(g.Problems, Problem{File: rel, Line: no, Msg: fmt.Sprintf("_nav.md: missing page %s", it.Href)})
					} else if first, dup := seen[f]; dup {
						g.Problems = append(g.Problems, Problem{File: rel, Line: no, Msg: fmt.Sprintf("_nav.md: %s is listed twice (first on line %d)", it.Href, first)})
					} else {
						seen[f] = no
						it.File = f
						g.Pages = append(g.Pages, f)
					}
				}
			}
		}
		ind := len(m[1])
		for len(stack) > 0 && stack[len(stack)-1].indent >= ind {
			stack = stack[:len(stack)-1]
		}
		if len(stack) == 0 {
			g.Items = append(g.Items, it)
		} else {
			p := stack[len(stack)-1].item
			it.Parent = p
			p.Children = append(p.Children, it)
		}
		stack = append(stack, level{ind, it})
	}
	if len(g.Pages) == 0 {
		g.Problems = append(g.Problems, Problem{File: rel, Line: start + 1, Msg: "_nav.md lists no pages"})
	}
	return g, nil
}

// FindGuide returns the guide that holds file: the nearest folder at or above
// file's folder, up to root (inclusive), with a _nav.md; nil when there is none.
func FindGuide(file, root string) *Guide {
	dir, _ := filepath.Abs(filepath.Dir(file))
	root, _ = filepath.Abs(root)
	for {
		if _, err := os.Stat(filepath.Join(dir, NavFile)); err == nil {
			g, err := LoadGuide(dir)
			if err == nil {
				return g
			}
		}
		if dir == root || !strings.HasPrefix(dir, root) {
			return nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil
		}
		dir = parent
	}
}

// Defaults is the front matter a page starts from: the guide's settings, as
// type guide, without the guide's own title and description.
func (g *Guide) Defaults() Meta {
	m := g.Meta
	m.Type, m.Title, m.Description = "guide", "", ""
	return m
}

// Item returns the nav item of a page file, or nil.
func (g *Guide) Item(file string) *NavItem {
	var find func([]*NavItem) *NavItem
	find = func(items []*NavItem) *NavItem {
		for _, it := range items {
			if it.File == file {
				return it
			}
			if f := find(it.Children); f != nil {
				return f
			}
		}
		return nil
	}
	return find(g.Items)
}

// Unlisted are the .md/.adoc pages under the guide folder (and its
// subfolders) that _nav.md does not list; _-prefixed files and README.md are
// partials or notes, never pages.
func (g *Guide) Unlisted() []string {
	listed := map[string]bool{}
	for _, p := range g.Pages {
		listed[p] = true
	}
	var out []string
	filepath.WalkDir(g.Dir, func(p string, e os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		n := e.Name()
		if e.IsDir() {
			if p != g.Dir && (strings.HasPrefix(n, ".") || strings.HasPrefix(n, "_") || n == "node_modules") {
				return filepath.SkipDir
			}
			return nil
		}
		if isDeckFile(n) && !strings.HasPrefix(n, "_") && !strings.EqualFold(n, "README.md") && !listed[p] {
			out = append(out, p)
		}
		return nil
	})
	return out
}
