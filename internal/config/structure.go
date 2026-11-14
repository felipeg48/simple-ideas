package config

import (
	"fmt"
	"path"
	"sort"
	"strings"
)

// Structure is the project structure map: the structure: block of si.conf,
// one indented folder per line (names end in /) with the kinds of output that
// go there. si build writes each output into its folder; without the block
// it writes today's flat layout.
//
//	structure:
//	  rabbitmq/
//	    md/           markdown
//	    slides/       deck
//	      pdf/        deck:pdf
//	    dist/         html/:zip, deck:zip
type Structure struct {
	Project string    // the top folder when the block has one (it names the project root), else ""
	Folders []*Folder // in file order; Path is relative to the project root

	stack []frame // parse state
	top   int     // indent of the top-level folders (-1: none yet)
}

// Folder is one line of the map.
type Folder struct {
	Path    string // slash path relative to the project root, no trailing slash
	Recipes []Recipe
	Line    int
}

// Recipe is one word after a folder: a kind (deck), a kind's format
// (deck:pdf) or a bundle (deck:zip, html/:zip).
type Recipe struct {
	Kind   string // deck, image, ... ("" for a folder bundle)
	Format string // pdf, zip, scorm, ... ("" for the kind's pages or files)
	Folder string // a folder bundle: the map folder zipped whole (html/:zip)
	Line   int
}

func (r Recipe) String() string {
	if r.Folder != "" {
		return r.Folder + "/:" + r.Format
	}
	if r.Format != "" {
		return r.Kind + ":" + r.Format
	}
	return r.Kind
}

type frame struct {
	indent, child int // the folder's indent and its children's (-1: none yet)
	path          string
}

// StructureKinds are the kinds a folder can hold, each with its formats.
var StructureKinds = map[string][]string{
	"markdown":    nil,
	"deck":        {"pdf", "mp4", "handout", "zip"},
	"lab":         {"zip"},
	"guide":       {"zip"},
	"course":      {"scorm", "pdf", "zip"},
	"site":        {"zip"},
	"article":     {"zip"},
	"timeline":    {"png", "pdf", "svg", "zip"},
	"infographic": {"png", "pdf", "svg", "zip"},
	"mindmap":     {"png", "pdf", "svg", "mp4", "zip"},
	"book":        {"pdf", "zip"},
	"demo":        {"zip"},
	"image":       nil,
	"animation":   nil,
	"video":       nil,
}

// PageKinds are the kinds of pages a source builds into.
var PageKinds = []string{"deck", "lab", "guide", "course", "site", "article", "timeline", "infographic", "mindmap", "book", "demo"}

// AssetKinds are the kinds of files a page uses.
var AssetKinds = []string{"image", "animation", "video"}

func kindNames() string {
	var ks []string
	for k := range StructureKinds {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return strings.Join(ks, ", ")
}

// line reads one indented line of the block.
func (s *Structure) line(raw string, ln int, warn func(string, ...any)) {
	indent := len(raw) - len(strings.TrimLeft(raw, " \t"))
	text := strings.TrimSpace(raw)
	for i := 1; i < len(text); i++ { // an inline comment
		if text[i] == '#' && (text[i-1] == ' ' || text[i-1] == '\t') {
			text = strings.TrimSpace(text[:i])
			break
		}
	}
	name := strings.Fields(text)[0]
	rest := text[len(name):]
	if !strings.HasSuffix(name, "/") || name == "/" {
		warn("line %d: structure: a folder name ends in / (got %q)", ln, name)
		return
	}
	name = strings.TrimSuffix(name, "/")
	if strings.HasPrefix(name, "/") || strings.Contains(name, "..") || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		warn("line %d: structure: %q: one folder per line, nested by indentation", ln, name+"/")
		return
	}
	for len(s.stack) > 0 && s.stack[len(s.stack)-1].indent >= indent {
		s.stack = s.stack[:len(s.stack)-1]
	}
	parent := ""
	if n := len(s.stack); n > 0 {
		f := &s.stack[n-1]
		if f.child == -1 {
			f.child = indent
		} else if f.child != indent {
			warn("line %d: structure: %s/ is indented %d, its sibling folders %d", ln, name, indent, f.child)
		}
		parent = f.path + "/"
	} else if s.top == -1 {
		s.top = indent
	} else if s.top != indent {
		warn("line %d: structure: %s/ is indented %d, the top folders %d", ln, name, indent, s.top)
	}
	f := &Folder{Path: parent + name, Line: ln}
	for _, w := range strings.Split(rest, ",") {
		if w = strings.TrimSpace(w); w == "" {
			continue
		}
		r, msg := parseRecipe(w)
		if msg != "" {
			warn("line %d: structure: %s", ln, msg)
			continue
		}
		r.Line = ln
		f.Recipes = append(f.Recipes, r)
	}
	s.Folders = append(s.Folders, f)
	s.stack = append(s.stack, frame{indent: indent, child: -1, path: f.Path})
}

func parseRecipe(w string) (Recipe, string) {
	kind, format, _ := strings.Cut(strings.ToLower(w), ":")
	if strings.HasSuffix(kind, "/") {
		if format != "zip" {
			return Recipe{}, fmt.Sprintf("%q: a folder is bundled as %s:zip", w, kind)
		}
		return Recipe{Folder: path.Clean(strings.TrimSuffix(kind, "/")), Format: format}, ""
	}
	formats, ok := StructureKinds[kind]
	if !ok {
		return Recipe{}, fmt.Sprintf("unknown kind %q (kinds: %s)", kind, kindNames())
	}
	if format != "" {
		found := false
		for _, f := range formats {
			found = found || f == format
		}
		if !found {
			if len(formats) == 0 {
				return Recipe{}, fmt.Sprintf("%q: %s has no formats", w, kind)
			}
			return Recipe{}, fmt.Sprintf("unknown format %q for %s (%s)", format, kind, strings.Join(formats, ", "))
		}
	}
	return Recipe{Kind: kind, Format: format}, ""
}

// finish names the project root and checks the folder bundles.
func (s *Structure) finish(warn func(string, ...any)) {
	s.stack = nil
	var tops []*Folder
	for _, f := range s.Folders {
		if !strings.Contains(f.Path, "/") {
			tops = append(tops, f)
		}
	}
	if len(tops) == 1 && len(tops[0].Recipes) == 0 && len(s.Folders) > 1 {
		s.Project = tops[0].Path
		var rest []*Folder
		for _, f := range s.Folders[1:] {
			f.Path = strings.TrimPrefix(f.Path, s.Project+"/")
			rest = append(rest, f)
		}
		s.Folders = rest
	}
	for _, f := range s.Folders {
		for i, r := range f.Recipes {
			if r.Folder == "" {
				continue
			}
			if s.Project != "" {
				f.Recipes[i].Folder = strings.TrimPrefix(r.Folder, s.Project+"/")
			}
			if s.Folder(f.Recipes[i].Folder) == nil {
				warn("line %d: structure: %s names a folder the map does not define", r.Line, r)
			}
		}
	}
}

// Folder is the map folder at p, or nil.
func (s *Structure) Folder(p string) *Folder {
	for _, f := range s.Folders {
		if f.Path == p {
			return f
		}
	}
	return nil
}

// FoldersFor are the folders that hold kind (format "") or kind:format, in
// map order.
func (s *Structure) FoldersFor(kind, format string) []string {
	var out []string
	if s == nil {
		return nil
	}
	for _, f := range s.Folders {
		for _, r := range f.Recipes {
			if r.Folder == "" && r.Kind == kind && r.Format == format {
				out = append(out, f.Path)
				break
			}
		}
	}
	return out
}

// Has reports whether any folder holds kind:format.
func (s *Structure) Has(kind, format string) bool { return len(s.FoldersFor(kind, format)) > 0 }
