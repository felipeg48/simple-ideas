package main

// si build with a project structure map (si.conf structure:, RULES.md
// "Project structure"): the usual build writes its flat layout into a staging
// folder, then every file is routed into the folders of the map, links are
// rewritten from each page's new folder, and the bundles are zipped last.

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/config"
	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/felipeg48/simple-ideas/internal/render"
)

// collected is what the inner build copied beside its pages and inlined as
// canvases; nil collects nothing.
type collected struct {
	files    map[string]bool // absolute paths of copied media
	canvases []string        // absolute paths of canvas SVGs
}

func (c *collected) media(p string) {
	if c != nil {
		c.files[p] = true
	}
}

func (c *collected) canvas(ps []string) {
	if c == nil {
		return
	}
	for _, p := range ps {
		if !slices.Contains(c.canvases, p) {
			c.canvases = append(c.canvases, p)
		}
	}
}

// projectStructure is the structure map of the project folder src, or nil:
// src is not a folder, is a guide, has no structure: block, or this is the
// structured build's own inner pass.
func projectStructure(src string, o *options) *config.Structure {
	if o.staged {
		return nil
	}
	if st, err := os.Stat(src); err != nil || !st.IsDir() || deck.IsGuideDir(src) {
		return nil
	}
	return config.Load(o.conf, filepath.Join(src, "si.conf")).Structure
}

// projectName names the zips: the map's top folder, else the project folder.
func projectName(src string, st *config.Structure) string {
	if st.Project != "" {
		return st.Project
	}
	abs, _ := filepath.Abs(src)
	return filepath.Base(abs)
}

// sourceDirs are the folders of the map that exist under src (sources may
// live in any of them; markdown is where si new puts them).
func sourceDirs(src string, st *config.Structure) []string {
	var out []string
	for _, f := range st.Folders {
		d := filepath.Join(src, filepath.FromSlash(f.Path))
		if fi, err := os.Stat(d); err == nil && fi.IsDir() && !deck.IsGuideDir(d) {
			out = append(out, d)
		}
	}
	return out
}

// sourceKind is the kind of a source page in the map's words.
func sourceKind(d *deck.Deck) string {
	switch t := d.Meta.Type; {
	case t == "" || t == "deck" || t == "slides":
		return "deck"
	case slices.Contains(config.PageKinds, t):
		return t
	}
	return "article" // page, quiz: a document
}

// sources lists the decks and guides of the project: name -> kind.
func sources(src string, st *config.Structure) (decks, guides map[string]string, paths map[string]string) {
	decks, guides, paths = map[string]string{}, map[string]string{}, map[string]string{}
	for _, d := range append([]string{src}, sourceDirs(src, st)...) {
		files, _ := deck.List(d)
		for _, f := range files {
			if dk, err := deck.Parse(f); err == nil {
				decks[dk.Name], paths[dk.Name] = sourceKind(dk), f
			}
		}
		for _, g := range subGuides(d) {
			guides[filepath.Base(g.Dir)], paths[filepath.Base(g.Dir)] = guideKindOf(g), g.Dir
		}
	}
	return
}

var (
	mediaKind = map[string]string{".png": "image", ".jpg": "image", ".jpeg": "image", ".webp": "image", ".avif": "image", ".bmp": "image", ".ico": "image",
		".svg": "animation", ".gif": "animation",
		".mp4": "video", ".m4v": "video", ".webm": "video", ".mov": "video", ".ogv": "video", ".mp3": "video", ".m4a": "video", ".wav": "video", ".ogg": "video", ".vtt": "video"}
	linkRE = regexp.MustCompile(`"(?:[^"\\<>\n]|\\.)*"|'(?:[^'\\<>\n]|\\.)*'|url\([^)"'\s]*\)`) // quoted strings (JSON escapes kept in step) and CSS url()
)

// item is one file of the staged build and where it goes.
type item struct {
	rel          string   // slash path in the staging folder
	kind, format string   // what it is ("" kind: index, shared libraries, other)
	dests        []string // slash paths under the project root (none: dropped)
}

// stdoutRewriter shows the project root where the inner pass printed staging.
type stdoutRewriter struct {
	w          io.Writer
	from, into string
}

func (r stdoutRewriter) Write(p []byte) (int, error) {
	_, err := r.w.Write(bytes.ReplaceAll(p, []byte(r.from), []byte(r.into)))
	return len(p), err
}

// buildStructured is si build for a project with a structure map: the
// sequence of RULES.md "Project structure".
func buildStructured(src string, st *config.Structure, o *options, stdout, stderr io.Writer) int {
	root := o.output
	if root == "" {
		root = src
	}
	stage, err := os.MkdirTemp("", "si-structure-")
	if err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	defer os.RemoveAll(stage)
	inner := *o
	inner.output, inner.staged, inner.srcDirs = stage, true, sourceDirs(src, st)
	inner.collect = &collected{files: map[string]bool{}}
	// the formats the map asks for are built as if their flags were given
	for _, k := range config.PageKinds {
		inner.pdf = inner.pdf || st.Has(k, "pdf")
		inner.mp4 = inner.mp4 || st.Has(k, "mp4")
		inner.png = inner.png || st.Has(k, "png")
		inner.svg = inner.svg || st.Has(k, "svg")
	}
	if st.Has("deck", "handout") && inner.handout == "" {
		inner.handout = "deck"
	}
	if st.Has("course", "scorm") && inner.scorm == "" {
		inner.scorm = "lesson"
	}
	for _, k := range config.AssetKinds { // folders of pictures: pages link them, not inline them
		inner.noInline = inner.noInline || st.Has(k, "")
	}
	code := build(src, &inner, stdoutRewriter{stdout, stage, root}, stdoutRewriter{stderr, stage, root})
	items, err := route(stage, src, st, o, inner.collect)
	if err != nil {
		fmt.Fprintf(stderr, "si: structure: %v\n", err)
		return exitProblems
	}
	if err := place(stage, root, items); err != nil {
		fmt.Fprintf(stderr, "si: structure: %v\n", err)
		return exitProblems
	}
	for _, c := range inner.collect.canvases { // the canvases, inlined in the pages, also as files
		for _, f := range st.FoldersFor("animation", "") {
			if err := copyFile(c, filepath.Join(root, filepath.FromSlash(f), filepath.Base(c))); err != nil {
				fmt.Fprintf(stderr, "si: %v\n", err)
				code = exitProblems
			}
		}
	}
	if !bundle(root, projectName(src, st), st, items, stdout, stderr) {
		code = exitProblems
	}
	summary := map[string]int{}
	for _, it := range items {
		for _, d := range it.dests {
			top, _, _ := strings.Cut(d, "/")
			if !strings.Contains(d, "/") {
				top = "."
			}
			summary[top]++
		}
	}
	var parts []string
	for _, f := range st.Folders {
		if !strings.Contains(f.Path, "/") && summary[f.Path] > 0 {
			parts = append(parts, fmt.Sprintf("%s/ %d", f.Path, summary[f.Path]))
		}
	}
	fmt.Fprintf(stdout, "si: structure: %s -> %s\n", strings.Join(parts, ", "), root)
	return code
}

// ownerOf is the deck a derived file belongs to: its stem is the deck's name
// or starts with name- (a mind map's per-map pictures, a compact PDF).
func ownerOf(stem string, decks map[string]string) string {
	if decks[stem] != "" {
		return stem
	}
	best := ""
	for n := range decks {
		if strings.HasPrefix(stem, n+"-") && len(n) > len(best) {
			best = n
		}
	}
	return best
}

// wants reports whether the user asked for a format with a flag (it is kept,
// in today's place, when the map has no folder for it).
func wants(o *options, format string) bool {
	switch format {
	case "pdf":
		return o.pdf
	case "mp4":
		return o.mp4
	case "png":
		return o.png
	case "svg":
		return o.svg
	case "handout":
		return o.handout != ""
	case "scorm":
		return o.scorm != ""
	}
	return false
}

// route classifies every staged file and decides where it goes.
func route(stage, src string, st *config.Structure, o *options, col *collected) ([]*item, error) {
	decks, guides, _ := sources(src, st)
	media := map[string]bool{}
	for p := range col.files {
		if r, err := filepath.Rel(stage, p); err == nil {
			media[filepath.ToSlash(r)] = true
		}
	}
	units := map[string]int{} // guides per kind: one fills its folder, more get a folder each
	for _, k := range guides {
		units[k]++
	}
	var items []*item
	taken := map[string]bool{}
	err := filepath.WalkDir(stage, func(p string, e fs.DirEntry, err error) error {
		if err != nil || e.IsDir() {
			return err
		}
		r, _ := filepath.Rel(stage, p)
		rel := filepath.ToSlash(r)
		it := &item{rel: rel}
		items = append(items, it)
		seg0, rest, nested := strings.Cut(rel, "/")
		base := path.Base(rel)
		ext := strings.ToLower(path.Ext(rel))
		stem := strings.TrimSuffix(base, path.Ext(base))
		into := func(kind, format string, name func(folder string) string) bool {
			fs := st.FoldersFor(kind, format)
			for _, f := range fs {
				d := path.Join(f, name(f))
				if format == "" && mediaKind[ext] == kind && taken[d] { // two pictures of one name
					d = path.Join(f, strings.TrimPrefix(rel, seg0+"/"))
				}
				taken[d] = true
				it.dests = append(it.dests, d)
			}
			it.kind, it.format = kind, format
			return len(fs) > 0
		}
		flat := func(string) string { return base }
		guidePart := func() { // a file of a guide, course, site or book: with its folder
			gk := guides[seg0]
			if !into(gk, "", func(string) string {
				if units[gk] == 1 {
					return rest
				}
				return rel
			}) {
				it.dests = []string{rel}
			}
		}
		switch {
		case rel == "index.html":
			it.dests = []string{rel}
		case seg0 == render.ShareDir:
			it.kind = "shared" // placed beside the pages that use it, below
		case media[rel] && mediaKind[ext] != "":
			if !into(mediaKind[ext], "", flat) {
				if nested && guides[seg0] != "" {
					guidePart()
				} else {
					it.dests = []string{rel}
				}
			}
		case nested && guides[seg0] != "":
			gk := guides[seg0]
			switch {
			case gk == "course" && !strings.Contains(rest, "/") && strings.HasPrefix(stem, seg0+"-") && ext == ".pdf":
				if !into("course", "pdf", flat) {
					guidePart()
				}
			case gk == "book" && rest == "book.pdf":
				if !into("book", "pdf", flat) {
					guidePart()
				}
			default:
				guidePart()
			}
		case !nested && strings.HasSuffix(rel, "-scorm12.zip") && guides[strings.TrimSuffix(rel, "-scorm12.zip")] == "course":
			if !into("course", "scorm", flat) && wants(o, "scorm") {
				it.dests = []string{rel}
			}
		case (seg0 == "pdfs" || seg0 == "handouts") && !strings.Contains(rest, "/") && ownerOf(stem, decks) != "":
			format := map[string]string{"pdfs": "pdf", "handouts": "handout"}[seg0]
			if !into(decks[ownerOf(stem, decks)], format, flat) && wants(o, format) {
				it.dests = []string{rel}
			}
		case nested && strings.HasSuffix(seg0, ".voice") && decks[strings.TrimSuffix(seg0, ".voice")] != "":
			if !into(decks[strings.TrimSuffix(seg0, ".voice")], "", func(string) string { return rel }) {
				it.dests = []string{rel}
			}
		case !nested && ext == ".html" && decks[stem] != "":
			if !into(decks[stem], "", flat) {
				it.dests = []string{rel}
			}
		case !nested && ownerOf(stem, decks) != "" && (ext == ".mp4" || ext == ".vtt" || ext == ".png" || ext == ".svg" || ext == ".pdf"):
			format := map[string]string{".mp4": "mp4", ".vtt": "mp4", ".png": "png", ".svg": "svg", ".pdf": "pdf"}[ext]
			if !into(decks[ownerOf(stem, decks)], format, flat) && wants(o, format) {
				it.dests = []string{rel}
			}
		default:
			it.dests = []string{rel}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	// the shared libraries (three.js): one copy in each top folder whose pages use them
	tops := map[string]bool{}
	for _, it := range items {
		if !strings.HasSuffix(it.rel, ".html") || strings.Contains(it.rel, "/") {
			continue
		}
		if raw, err := os.ReadFile(filepath.Join(stage, it.rel)); err == nil && bytes.Contains(raw, []byte(render.ShareDir+"/")) {
			for _, d := range it.dests {
				top, _, ok := strings.Cut(d, "/")
				if !ok {
					top = ""
				}
				tops[top] = true
			}
		}
	}
	for _, it := range items {
		if it.kind == "shared" {
			for top := range tops {
				it.dests = append(it.dests, path.Join(top, it.rel))
			}
			sort.Strings(it.dests)
		}
	}
	return items, nil
}

// relTo is the link from a page in dir to target (both slash paths under the root).
func relTo(dir, target string) string {
	if dir == "." || dir == "" {
		return target
	}
	r, err := filepath.Rel(filepath.FromSlash(dir), filepath.FromSlash(target))
	if err != nil {
		return target
	}
	return filepath.ToSlash(r)
}

// nearest picks, of a file's copies, the one closest to the page at from.
func nearest(dests []string, from string) string {
	best, score := dests[0], -1
	fd := strings.Split(path.Dir(from), "/")
	for _, d := range dests {
		dd, n := strings.Split(path.Dir(d), "/"), 0
		for n < len(fd) && n < len(dd) && fd[n] == dd[n] {
			n++
		}
		if n > score {
			best, score = d, n
		}
	}
	return best
}

// rewrite points every local link of a page staged at from, written to to,
// at where its target went.
func rewrite(page []byte, from, to string, where map[string][]string) []byte {
	return linkRE.ReplaceAllFunc(page, func(m []byte) []byte {
		pre, body, post := string(m[:1]), string(m[1:len(m)-1]), string(m[len(m)-1:])
		if bytes.HasPrefix(m, []byte("url(")) {
			pre, body = "url(", string(m[4:len(m)-1])
		}
		p, frag := body, ""
		if i := strings.IndexAny(body, "#?"); i >= 0 {
			p, frag = body[:i], body[i:]
		}
		if p == "" || strings.Contains(p, ":") || strings.HasPrefix(p, "/") || !strings.ContainsAny(p, "./") {
			return m
		}
		t := path.Clean(path.Join(path.Dir(from), p))
		if t == ".." || strings.HasPrefix(t, "../") {
			return m
		}
		ds, dir := where[t], false
		if len(ds) == 0 {
			ds, dir = where[path.Join(t, "index.html")], true
		}
		if len(ds) == 0 {
			return m
		}
		target := nearest(ds, to)
		r := relTo(path.Dir(to), target)
		if dir {
			if r = path.Dir(r); strings.HasSuffix(p, "/") {
				r += "/"
			}
		}
		if r == p {
			return m
		}
		return []byte(pre + r + frag + post)
	})
}

// place writes every routed file under root, pages with their links rewritten.
func place(stage, root string, items []*item) error {
	where := map[string][]string{}
	for _, it := range items {
		if len(it.dests) > 0 {
			where[it.rel] = it.dests
		}
	}
	for _, it := range items {
		if len(it.dests) == 0 {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(stage, filepath.FromSlash(it.rel)))
		if err != nil {
			return err
		}
		for _, d := range it.dests {
			out := raw
			if strings.HasSuffix(it.rel, ".html") {
				out = rewrite(raw, it.rel, d, where)
			}
			dest := filepath.Join(root, filepath.FromSlash(d))
			if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(dest, out, 0o644); err != nil {
				return err
			}
		}
	}
	return nil
}

// localRefs are the files under root a written page links to.
func localRefs(root, page string) []string {
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(page)))
	if err != nil {
		return nil
	}
	var out []string
	for _, m := range linkRE.FindAll(raw, -1) {
		body := string(m[1 : len(m)-1])
		if bytes.HasPrefix(m, []byte("url(")) {
			body = string(m[4 : len(m)-1])
		}
		if i := strings.IndexAny(body, "#?"); i >= 0 {
			body = body[:i]
		}
		if body == "" || strings.Contains(body, ":") || strings.HasPrefix(body, "/") || !strings.ContainsAny(body, "./") {
			continue
		}
		t := path.Clean(path.Join(path.Dir(page), body))
		if fi, err := os.Stat(filepath.Join(root, filepath.FromSlash(t))); err == nil && !fi.IsDir() && !strings.HasPrefix(t, "../") {
			out = append(out, t)
		}
	}
	return out
}

// zipFiles writes the files (slash paths under root) into a zip at dest.
func zipFiles(root string, files []string, dest string) error {
	tmp, err := os.CreateTemp("", "si-zip-")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	zw := zip.NewWriter(tmp)
	for _, f := range files {
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(f)))
		if err != nil {
			return err
		}
		w, err := zw.Create(f)
		if err != nil {
			return err
		}
		if _, err := w.Write(raw); err != nil {
			return err
		}
	}
	if err := zw.Close(); err != nil {
		return err
	}
	tmp.Close()
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	return copyFile(tmp.Name(), dest)
}

// bundle zips last: kind:zip (every output of a kind, with the files its pages
// link to, paths as in the project), then folder/:zip (a folder of the map, whole).
func bundle(root, project string, st *config.Structure, items []*item, stdout, stderr io.Writer) bool {
	ok := true
	write := func(files []string, folder, what string) {
		name := project + "-" + what + ".zip"
		if len(files) == 0 {
			fmt.Fprintf(stderr, "  note: structure: %s/%s: nothing to bundle yet, nothing of that kind was built\n", folder, name)
			return
		}
		dest := filepath.Join(root, filepath.FromSlash(folder), name)
		if err := zipFiles(root, files, dest); err != nil {
			fmt.Fprintf(stderr, "si: %v\n", err)
			ok = false
			return
		}
		fmt.Fprintf(stdout, "  %-40s (zip, %d files, %d KB)\n", path.Join(folder, name), len(files), fileKB(dest))
	}
	for _, f := range st.Folders {
		for _, r := range f.Recipes {
			if r.Folder != "" || r.Format != "zip" {
				continue
			}
			seen := map[string]bool{}
			var files []string
			add := func(p string) {
				if !seen[p] {
					seen[p] = true
					files = append(files, p)
				}
			}
			for _, it := range items {
				if it.kind != r.Kind || it.format == "zip" || len(it.dests) == 0 {
					continue
				}
				add(it.dests[0])
				if strings.HasSuffix(it.dests[0], ".html") {
					// its pictures and videos, and the built pages it links to
					// (the index, a lab) with theirs: one level, so the bundle's
					// links work without pulling in the whole project
					for _, ref := range localRefs(root, it.dests[0]) {
						add(ref)
						if strings.HasSuffix(ref, ".html") {
							for _, r2 := range localRefs(root, ref) {
								if !strings.HasSuffix(r2, ".html") {
									add(r2)
								}
							}
						}
					}
				}
			}
			sort.Strings(files)
			write(files, f.Path, r.Kind)
		}
	}
	for _, f := range st.Folders {
		for _, r := range f.Recipes {
			if r.Folder == "" {
				continue
			}
			var files []string
			base := filepath.Join(root, filepath.FromSlash(r.Folder))
			filepath.WalkDir(base, func(p string, e fs.DirEntry, err error) error {
				if err == nil && !e.IsDir() {
					rel, _ := filepath.Rel(base, p)
					files = append(files, filepath.ToSlash(rel))
				}
				return nil
			})
			name := project + "-" + strings.ReplaceAll(r.Folder, "/", "-") + ".zip"
			if len(files) == 0 {
				fmt.Fprintf(stderr, "  warning: structure: %s/%s would be empty: nothing was built into %s/\n", f.Path, name, r.Folder)
				continue
			}
			dest := filepath.Join(root, filepath.FromSlash(f.Path), name)
			if err := zipFiles(base, files, dest); err != nil {
				fmt.Fprintf(stderr, "si: %v\n", err)
				ok = false
				continue
			}
			fmt.Fprintf(stdout, "  %-40s (zip of %s/, %d files, %d KB)\n", path.Join(f.Path, name), r.Folder, len(files), fileKB(dest))
		}
	}
	return ok
}

// structureWarnings are si check's warnings on a project's map and its
// sources, and its notes: a kind bundle (deck:zip, course:scorm) for a kind
// the project has no source of yet is a note, nothing to bundle yet, since a
// map is often written before the content.
func structureWarnings(src string, st *config.Structure) (out, notes []string) {
	decks, guides, paths := sources(src, st)
	kinds := map[string]bool{}
	for _, k := range decks {
		kinds[k] = true
	}
	for _, k := range guides {
		kinds[k] = true
	}
	md := st.FoldersFor("markdown", "")
	home := "the project root"
	if len(md) > 0 {
		home = md[0] + "/"
	}
	names := make([]string, 0, len(paths))
	for n := range paths {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		k := decks[n]
		if k == "" {
			k = guides[n]
		}
		p := paths[n]
		rel, _ := filepath.Rel(src, p)
		rel = filepath.ToSlash(rel)
		if !st.Has(k, "") {
			out = append(out, fmt.Sprintf("%s: a %s, and the structure has no %s folder: it builds into the project root", rel, k, k))
		}
		dir := path.Dir(rel)
		if guides[n] != "" {
			dir = path.Dir(dir)
		}
		if f := st.Folder(dir); f != nil {
			holds := false
			for _, r := range f.Recipes {
				holds = holds || r.Kind == "markdown" || (r.Kind == k && r.Format == "")
			}
			if !holds && len(f.Recipes) > 0 {
				var rs []string
				for _, r := range f.Recipes {
					rs = append(rs, r.String())
				}
				out = append(out, fmt.Sprintf("%s: a %s source in %s/, the folder for %s; sources go in %s", rel, k, dir, strings.Join(rs, ", "), home))
			}
		}
	}
	// what each folder (with its subfolders) will receive
	gets := func(folder string) bool {
		for _, f := range st.Folders {
			if f.Path != folder && !strings.HasPrefix(f.Path, folder+"/") {
				continue
			}
			for _, r := range f.Recipes {
				if r.Folder == "" && (kinds[r.Kind] || (slices.Contains(config.AssetKinds, r.Kind) && len(kinds) > 0)) {
					return true
				}
			}
		}
		return false
	}
	for _, f := range st.Folders {
		for _, r := range f.Recipes {
			switch {
			case r.Folder != "" && st.Folder(r.Folder) != nil && !gets(r.Folder):
				out = append(out, fmt.Sprintf("si.conf line %d: structure: %s would be empty: no source builds into %s/", r.Line, r, r.Folder))
			case r.Folder == "" && (r.Format == "zip" || r.Format == "scorm") && !kinds[r.Kind]:
				notes = append(notes, fmt.Sprintf("si.conf line %d: structure: %s: nothing to bundle yet, the project has no %s", r.Line, r, r.Kind))
			}
		}
	}
	return out, notes
}

// checkStructured is si check of a project with a map: its notes, then every
// folder that holds sources.
func checkStructured(src string, st *config.Structure, budget config.Budget, stdout, stderr io.Writer) int {
	warns, notes := structureWarnings(src, st)
	for _, w := range warns {
		fmt.Fprintf(stderr, "warning: %s\n", w)
	}
	for _, n := range notes {
		fmt.Fprintf(stderr, "note: %s\n", n)
	}
	code := exitOK
	for _, d := range append([]string{src}, sourceDirs(src, st)...) {
		files, _ := deck.List(d)
		if len(files) == 0 && len(subGuides(d)) == 0 {
			continue
		}
		if c := check(d, budget, stdout, stderr); c != exitOK {
			code = c
		}
	}
	return code
}

// structureTemplate is the map si new --structure writes (the founder's).
const structureTemplate = `structure:
  %s/
    md/           markdown
    slides/       deck
      pdf/        deck:pdf
    labs/         lab
    html/         site
      images/     image
      animations/ animation
      pdfs/       deck:pdf
    videos/       video
    dist/         html/:zip, deck:zip, course:scorm
`

// newStructured is si new with a project structure: the map in si.conf (an
// existing one is kept), its folders, and the starter decks in the markdown
// folder.
func newStructured(name string, stdout, stderr io.Writer) int {
	base := filepath.Base(name)
	conf := filepath.Join(name, "si.conf")
	c := config.Load("", conf)
	wrote := false
	if c.Structure == nil {
		if fi, err := os.Stat(conf); err == nil && !fi.IsDir() {
			fmt.Fprintf(stderr, "si: %s has no structure: block; add one (si rules, Project structure) or drop --structure\n", conf)
			return exitProblems
		}
		if err := os.MkdirAll(name, 0o755); err != nil {
			fmt.Fprintf(stderr, "si: %v\n", err)
			return exitProblems
		}
		if err := os.WriteFile(conf, []byte(fmt.Sprintf(structureTemplate, base)), 0o644); err != nil {
			fmt.Fprintf(stderr, "si: %v\n", err)
			return exitProblems
		}
		c, wrote = config.Load("", conf), true
	}
	for _, w := range c.Warnings {
		fmt.Fprintf(stderr, "warning: %s\n", w)
	}
	st := c.Structure
	for _, f := range st.Folders {
		if err := os.MkdirAll(filepath.Join(name, filepath.FromSlash(f.Path)), 0o755); err != nil {
			fmt.Fprintf(stderr, "si: %v\n", err)
			return exitProblems
		}
	}
	md := name
	if fs := st.FoldersFor("markdown", ""); len(fs) > 0 {
		md = filepath.Join(name, filepath.FromSlash(fs[0]))
	}
	if err := writeExamples(md, base+".md"); err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	if inner := filepath.Join(md, "si.conf"); md != name { // one si.conf, at the project root
		if raw, err := os.ReadFile(inner); err == nil {
			if wrote {
				old, _ := os.ReadFile(conf)
				os.WriteFile(conf, append(append(raw, '\n'), old...), 0o644)
			}
			os.Remove(inner)
		}
	}
	var folders []string
	for _, f := range st.Folders {
		folders = append(folders, f.Path+"/")
	}
	rel, _ := filepath.Rel(name, filepath.Join(md, base+".md"))
	fmt.Fprintf(stdout, "si: created %s with the folders %s\n  sources in %s/%s\n  si build %s   (writes each kind into its folder)\n",
		name, strings.Join(folders, " "), name, filepath.ToSlash(rel), name)
	return exitOK
}

// hasStructure reports whether dir's own si.conf has a structure: block.
func hasStructure(dir string) bool {
	if _, err := os.Stat(filepath.Join(dir, "si.conf")); err != nil {
		return false
	}
	return config.Load("", filepath.Join(dir, "si.conf")).Structure != nil
}
