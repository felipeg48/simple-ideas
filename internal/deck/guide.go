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

// NavFile is the file that makes a folder a guide (or a book); SiteFile makes
// it a site: the same format, with `type: site` implied.
const (
	NavFile  = "_nav.md"
	SiteFile = "_site.md"
)

// navFile is the guide or site file of dir, "" if it has none.
func navFile(dir string) string {
	for _, n := range []string{SiteFile, NavFile} {
		if st, err := os.Stat(filepath.Join(dir, n)); err == nil && !st.IsDir() {
			return filepath.Join(dir, n)
		}
	}
	return ""
}

// IsGuideDir reports whether dir holds a _nav.md or a _site.md.
func IsGuideDir(dir string) bool { return navFile(dir) != "" }

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
	Sub      string // a link to a folder that is a guide, book or site of its own: that folder
}

var navItemRE = regexp.MustCompile(`^(\s*)[-*+]\s+(.*?)\s*$`)
var navLinkRE = regexp.MustCompile(`^\[(.+?)\]\(([^)\s]+)\)$`)

// LoadGuide parses dir/_nav.md.
func LoadGuide(dir string) (*Guide, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	nav := navFile(abs)
	if nav == "" {
		return nil, fmt.Errorf("%s has no %s or %s", abs, NavFile, SiteFile)
	}
	raw, err := os.ReadFile(nav)
	if err != nil {
		return nil, err
	}
	g := &Guide{Dir: abs}
	fm := &Deck{Path: nav}
	lines := splitLines(strings.TrimPrefix(string(raw), "\uFEFF"))
	start := fm.frontMatter(nav, lines)
	g.Meta, g.Problems, g.Warnings = fm.Meta, fm.Problems, fm.Warnings
	if filepath.Base(nav) == SiteFile && g.Meta.Type == "" {
		g.Meta.Type = "site"
	}
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
				st, err := os.Stat(f)
				switch {
				case err == nil && st.IsDir() && IsGuideDir(f):
					it.Sub = f // a guide, book or site of its own, linked from this menu
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
		if IsGuideDir(dir) {
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
	m.Type, m.Title, m.Description, m.Subtitle = "guide", "", "", ""
	m.Fonts = absFontFiles(m.Fonts, g.Dir)
	if g.IsSite() {
		m.Type = "page" // a site's pages are plain pages unless they say otherwise (type: deck, article, lab …)
		m.Navigator, m.Logo, m.CTA = "", "", ""
	}
	return m
}

// absFontFiles makes the font file paths in a fonts: value absolute against
// dir, so pages in subfolders find the files a _nav.md or _site.md names.
func absFontFiles(val, dir string) string {
	parts := strings.Split(val, ",")
	for i, p := range parts {
		role, rest, ok := strings.Cut(p, "=")
		file, w, _ := strings.Cut(strings.TrimSpace(rest), ":")
		switch strings.ToLower(filepath.Ext(file)) {
		case ".woff2", ".woff", ".ttf", ".otf":
			if ok && !filepath.IsAbs(file) {
				if w != "" {
					w = ":" + w
				}
				parts[i] = role + "=" + filepath.Join(dir, file) + w
			}
		}
	}
	return strings.Join(parts, ",")
}

// IsSite reports whether the guide is a site (`_site.md`, or `type: site`).
func (g *Guide) IsSite() bool { return g.Meta.Type == "site" }

// IsBook reports whether the guide is a book (`type: book` in _nav.md).
func (g *Guide) IsBook() bool { return g.Meta.Type == "book" }

// BookNumbers numbers a book's pages in nav order: pages before the first
// section are front matter (no number), pages under a section called
// "Appendices" (or "Appendix …") are lettered A, B, …, every other page is a
// chapter 1, 2, …. kinds holds "Chapter" or "Appendix" per numbered page.
func (g *Guide) BookNumbers() (nums, kinds map[string]string) {
	nums, kinds = map[string]string{}, map[string]string{}
	ch, ap, seenSection := 0, 0, false
	var walk func(items []*NavItem, appendix bool)
	walk = func(items []*NavItem, appendix bool) {
		for _, it := range items {
			if it.File == "" && it.Href == "" { // a part or section heading
				seenSection = true
				walk(it.Children, appendix || strings.HasPrefix(strings.ToLower(strings.TrimSpace(it.Title)), "appendi"))
				continue
			}
			if it.File != "" && seenSection {
				if appendix {
					nums[it.File], kinds[it.File] = string(rune('A'+ap)), "Appendix"
					ap++
				} else {
					ch++
					nums[it.File], kinds[it.File] = fmt.Sprint(ch), "Chapter"
				}
			}
			walk(it.Children, appendix)
		}
	}
	walk(g.Items, false)
	return nums, kinds
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
			if p != g.Dir && (strings.HasPrefix(n, ".") || strings.HasPrefix(n, "_") || n == "node_modules" || IsGuideDir(p)) {
				return filepath.SkipDir // a sub-guide, book or site lists its own pages
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
