// Package deck parses a markdown slide deck into a model: front matter,
// slides, speaker notes, includes and per-slide directives. It does not
// render HTML; internal/render does that.
package deck

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Kind is the role of a slide in the deck.
type Kind string

// Slide kinds. The first `# H1` is the title slide; a later `# H1` (for
// example the first line of an included module) is a section slide; a deck
// that ends on a `# H1` closes with a closing slide.
const (
	KindTitle   Kind = "title"
	KindSection Kind = "section"
	KindClosing Kind = "closing"
	KindContent Kind = "content"
)

// Layouts and transitions accepted by the per-slide directives.
var (
	Layouts     = []string{"default", "center", "two-col", "split", "quote", "big-number", "image"}
	Transitions = []string{"fade", "slide", "zoom", "none"}
	Themes      = []string{"studio", "paper", "mono", "crdb", "cockroachdb", "cockroachlabs"}
	// FooterPositions are the accepted footer-position values (long form plus short aliases).
	FooterPositions = []string{"bottom-right", "bottom-left", "bottom-center", "top-right", "top-left", "top-center", "br", "bl", "bc", "tr", "tl", "tc"}
	Modes           = []string{"auto", "light", "dark"}
	Aspects         = []string{"16:9", "16:10", "4:3", "1:1", "9:16"}
)

// Directives lists every `<!-- name: value -->` directive. Slide-level ones
// are consumed by the parser; body-level ones stay in the markdown for the
// renderer.
var Directives = map[string]string{
	"include":    "file",  // splice another deck
	"layout":     "slide", // slide layout
	"transition": "slide", // slide transition
	"background": "slide", // slide background
	"footer":     "slide", // per-slide footer override (`off` hides it)
	"labs":       "slide", // turn the slide into a lab slide (comma-separated lab URLs)
	"class":      "slide", // extra classes on the slide
	"svg":        "body",  // inline an animated SVG canvas
	"step":       "body",  // reveal the next block as one step
	"steps":      "body",  // reveal the next list item by item
	"column":     "body",  // start the next column
	"image":      "body",  // size and place the next image
}

// Meta is the deck-level front matter.
type Meta struct {
	Title      string
	Author     string
	Date       string
	Theme      string // studio | paper | mono | crdb (aliases: cockroachdb, cockroachlabs)
	Mode       string // auto | light | dark
	Accent     string // CSS colour
	Transition string // default slide transition
	Aspect     string // aspect hint for overview cards and print pages
	Mermaid    string // "" (CDN default), "off", or a script URL
	Background string // default background of every slide
	Lang       string
	Status     string // optional course status badge on the index (e.g. beta, preview, new)
	Version    string // optional course version badge on the index (e.g. v1.2.0)
	Year       string // optional year shown in the title-slide copyright line
	Copyright  string // optional copyright holder shown on the title slide (© year holder)

	Footer           string // text shown in a corner of every slide; a slide's `footer:` overrides it, `footer: off` hides it
	FooterPos        string // footer corner: bottom-right (default), bottom-left, bottom-center, top-right, top-left, top-center
	Watermark        string // faint text, or an image path, shown across every slide
	WatermarkOpacity string // watermark opacity (a CSS number); default 0.06
	LabsLabel        string // heading on a lab slide (default "Now, let's do a Lab!")
	Runbook          string // path to a JSON runbook (command cheat-sheet); shown only in the `si play` terminal drawer
}

// MetaKeys are the accepted front matter keys.
var MetaKeys = []string{"title", "author", "date", "theme", "mode", "accent", "transition", "aspect", "mermaid", "lang", "background", "status", "version", "year", "copyright", "footer", "footer-position", "watermark", "watermark-opacity", "labs-label", "runbook"}

// Lab is one hands-on lab link on a lab slide: clickable Text opening URL.
type Lab struct {
	Text string
	URL  string
}

// Slide is one slide of a deck.
type Slide struct {
	Kind       Kind
	Title      string // heading text (inline markdown), may be empty on a content slide
	Body       string // markdown below the heading, directives for the renderer included
	Notes      string // markdown of the verbatim teleprompter script
	Points     string // markdown of the concise speaker-notes talking points
	Minutes    float64
	HasMinutes bool
	Layout     string
	Transition string
	Background string
	Footer     string // per-slide footer override: text, or "off" to hide; empty inherits the deck footer
	Labs       []Lab  // lab links; a non-empty list turns the slide into a lab slide
	Classes    []string
	Part       int    // section slides: 1-based number of the section in the deck
	Dir        string // directory of the source file: base for relative paths
	File       string // source file
	Line       int    // line of the slide's first line in File
}

// Problem is a deck error (or warning) at a source position.
type Problem struct {
	File string
	Line int
	Msg  string
}

func (p Problem) String() string {
	if p.Line > 0 {
		return fmt.Sprintf("%s:%d: %s", p.File, p.Line, p.Msg)
	}
	return fmt.Sprintf("%s: %s", p.File, p.Msg)
}

// Deck is a parsed deck.
type Deck struct {
	Path     string
	Name     string // file name without .md
	Meta     Meta
	Slides   []*Slide
	Problems []Problem
}

// Title is the deck title: front matter title, else the title slide.
func (d *Deck) Title() string {
	if d.Meta.Title != "" {
		return d.Meta.Title
	}
	if len(d.Slides) > 0 && d.Slides[0].Title != "" {
		return d.Slides[0].Title
	}
	return d.Name
}

// Minutes is the planned length: the sum of every slide's `(N min)`.
func (d *Deck) Minutes() float64 {
	var m float64
	for _, s := range d.Slides {
		m += s.Minutes
	}
	return m
}

func (d *Deck) problem(file string, line int, format string, a ...any) {
	d.Problems = append(d.Problems, Problem{File: rel(file), Line: line, Msg: fmt.Sprintf(format, a...)})
}

var (
	fenceRE     = regexp.MustCompile("^ {0,3}(`{3,}|~{3,})")
	sepRE       = regexp.MustCompile(`^ {0,3}---\s*$`)
	h1RE        = regexp.MustCompile(`^ {0,3}#\s+(.+?)\s*#*\s*$`)
	h2RE        = regexp.MustCompile(`^ {0,3}##\s+(.+?)\s*#*\s*$`)
	notesRE     = regexp.MustCompile(`^Notes:\s*(.*)$`)
	pointsRE    = regexp.MustCompile(`^Points:\s*(.*)$`)
	minutesRE   = regexp.MustCompile(`\((\d+(?:\.\d+)?)\s*min\)`)
	directiveRE = regexp.MustCompile(`^\s*<!--\s*([a-z][a-z-]*)\s*(?::\s*(.*?))?\s*-->\s*$`)
	metaRE      = regexp.MustCompile(`^([a-z][a-z-]*)\s*:\s*(.*?)\s*$`)
	colourRE    = regexp.MustCompile(`^(#[0-9a-fA-F]{3,8}|[a-zA-Z]+|(rgb|rgba|hsl|hsla|oklch|oklab|color-mix)\(.*\))$`)
	imageExtRE  = regexp.MustCompile(`(?i)\.(png|jpe?g|gif|webp|svg|avif)$`)
	labLinkRE   = regexp.MustCompile(`\[([^\]]+)\]\(\s*(https?://[^)\s]+)\s*\)`)
)

// ParseDirective reports whether line is a directive comment and returns its
// name and value. A comment whose first word is followed by anything but `:`
// or `-->` (`<!-- note to self -->`) is an ordinary comment.
func ParseDirective(line string) (name, value string, ok bool) {
	m := directiveRE.FindStringSubmatch(line)
	if m == nil {
		return "", "", false
	}
	return m[1], strings.TrimSpace(m[2]), true
}

type srcLine struct {
	file string
	no   int
	text string
}

// fenceTracker follows ``` and ~~~ fences so directives and separators inside
// code are left alone.
type fenceTracker struct{ open string }

// inFence updates the state with line and reports whether line belongs to a
// fence (opening and closing lines included).
func (f *fenceTracker) inFence(line string) bool {
	if f.open != "" {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, f.open) && strings.Trim(t, f.open[:1]) == "" {
			f.open = ""
		}
		return true
	}
	if m := fenceRE.FindStringSubmatch(line); m != nil {
		f.open = m[1]
		return true
	}
	return false
}

// Parse reads and parses the deck at path. Problems are collected in
// Deck.Problems; the error is only for an unreadable file.
func Parse(path string) (*Deck, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(abs)
	if err != nil {
		return nil, err
	}
	src := string(raw)
	if ext := strings.ToLower(filepath.Ext(abs)); ext == ".adoc" || ext == ".asciidoc" {
		src = adocToMarkdown(src) // AsciiDoc decks feed the same Markdown pipeline
	}
	d := &Deck{Path: abs, Name: strings.TrimSuffix(filepath.Base(abs), filepath.Ext(abs))}
	lines := splitLines(src)
	start := d.frontMatter(abs, lines)
	var out []srcLine
	for i, t := range lines[start:] {
		out = append(out, srcLine{abs, start + i + 1, t})
	}
	out = d.expandIncludes(abs, out, []string{abs})
	d.buildSlides(out)
	return d, nil
}

func splitLines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.Split(strings.TrimRight(s, "\n"), "\n")
}

// frontMatter parses `---` / `key: value` lines / `---` at the very top and
// returns the index of the first line after it (0 when there is none).
func (d *Deck) frontMatter(file string, lines []string) int {
	if len(lines) < 2 || strings.TrimSpace(lines[0]) != "---" {
		return 0
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		t := strings.TrimSpace(lines[i])
		if t == "---" {
			end = i
			break
		}
		if t != "" && !strings.HasPrefix(t, "#") && !metaRE.MatchString(t) {
			return 0 // not front matter: the deck starts with a separator
		}
	}
	if end < 0 {
		return 0
	}
	for i := 1; i < end; i++ {
		t := strings.TrimSpace(lines[i])
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		m := metaRE.FindStringSubmatch(t)
		key, val := m[1], strings.Trim(m[2], `"'`)
		if !d.setMeta(key, val) {
			d.problem(file, i+1, "unknown front matter key %q (known: %s)", key, strings.Join(MetaKeys, ", "))
			continue
		}
		if msg := validateMeta(key, val); msg != "" {
			d.problem(file, i+1, "%s", msg)
		}
		if key == "background" {
			d.checkBackground(file, i+1, filepath.Dir(file), val)
		}
	}
	return end + 1
}

func (d *Deck) setMeta(key, val string) bool {
	m := &d.Meta
	ptr := map[string]*string{
		"title": &m.Title, "author": &m.Author, "date": &m.Date, "theme": &m.Theme, "mode": &m.Mode,
		"accent": &m.Accent, "transition": &m.Transition, "aspect": &m.Aspect, "mermaid": &m.Mermaid, "lang": &m.Lang,
		"background": &m.Background, "status": &m.Status, "version": &m.Version, "year": &m.Year, "copyright": &m.Copyright,
		"footer": &m.Footer, "footer-position": &m.FooterPos, "watermark": &m.Watermark, "watermark-opacity": &m.WatermarkOpacity, "labs-label": &m.LabsLabel,
		"runbook": &m.Runbook,
	}[key]
	if ptr == nil {
		return false
	}
	*ptr = val
	return true
}

func validateMeta(key, val string) string {
	switch key {
	case "theme":
		lower := strings.ToLower(val)
		if strings.HasSuffix(lower, ".css") || strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
			return "" // a theme CSS file path or URL; resolved at render time
		}
		return oneOf("theme", val, Themes)
	case "mode":
		return oneOf("mode", val, Modes)
	case "transition":
		return oneOf("transition", val, Transitions)
	case "aspect":
		return oneOf("aspect", val, Aspects)
	case "footer-position":
		return oneOf("footer-position", val, FooterPositions)
	case "accent":
		if !colourRE.MatchString(val) {
			return fmt.Sprintf("accent %q is not a CSS colour", val)
		}
	}
	return ""
}

func oneOf(what, val string, allowed []string) string {
	for _, a := range allowed {
		if val == a {
			return ""
		}
	}
	return fmt.Sprintf("unknown %s %q (use %s)", what, val, strings.Join(allowed, ", "))
}

// expandIncludes splices `<!-- include: x.md -->` lines; stack holds the
// files being expanded, to refuse cycles.
func (d *Deck) expandIncludes(file string, lines []srcLine, stack []string) []srcLine {
	var out []srcLine
	var ft fenceTracker
	for _, l := range lines {
		if ft.inFence(l.text) {
			out = append(out, l)
			continue
		}
		name, val, ok := ParseDirective(l.text)
		if !ok || name != "include" {
			out = append(out, l)
			continue
		}
		if val == "" {
			d.problem(l.file, l.no, "include needs a file: <!-- include: other.md -->")
			continue
		}
		target := filepath.Join(filepath.Dir(l.file), val)
		if contains(stack, target) {
			chain := make([]string, 0, len(stack)+1)
			for _, s := range append(stack, target) {
				chain = append(chain, filepath.Base(s))
			}
			d.problem(l.file, l.no, "include cycle: %s", strings.Join(chain, " -> "))
			continue
		}
		raw, err := os.ReadFile(target)
		if err != nil {
			d.problem(l.file, l.no, "include not found: %s", val)
			continue
		}
		rawStr := string(raw)
		if ext := strings.ToLower(filepath.Ext(target)); ext == ".adoc" || ext == ".asciidoc" {
			rawStr = adocToMarkdown(rawStr) // included AsciiDoc is converted too
		}
		inc := splitLines(rawStr)
		// a module's own front matter is ignored: the including deck decides
		sub := &Deck{}
		start := sub.frontMatter(target, inc)
		var incLines []srcLine
		for i, t := range inc[start:] {
			incLines = append(incLines, srcLine{target, start + i + 1, t})
		}
		out = append(out, srcLine{l.file, l.no, "---"})
		out = append(out, d.expandIncludes(target, incLines, append(stack, target))...)
		out = append(out, srcLine{l.file, l.no, "---"})
	}
	return out
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// buildSlides splits on `---` outside fences and builds the slides.
func (d *Deck) buildSlides(lines []srcLine) {
	var chunks [][]srcLine
	cur := []srcLine{}
	var ft fenceTracker
	for _, l := range lines {
		if !ft.inFence(l.text) && sepRE.MatchString(l.text) {
			chunks = append(chunks, cur)
			cur = []srcLine{}
			continue
		}
		cur = append(cur, l)
	}
	chunks = append(chunks, cur)
	parts := 0
	for _, c := range chunks {
		c = trimBlank(c)
		if len(c) == 0 {
			continue
		}
		s := d.buildSlide(c, len(d.Slides) == 0)
		if s.Kind == KindSection {
			parts++
			s.Part = parts
		}
		d.Slides = append(d.Slides, s)
	}
	if len(d.Slides) == 0 {
		d.problem(d.Path, 0, "empty deck: start with a `# Title` line")
		return
	}
	if d.Slides[0].Kind != KindTitle {
		d.problem(d.Path, d.Slides[0].Line, "missing title: a deck starts with a `# H1` title slide")
	}
	if last := d.Slides[len(d.Slides)-1]; len(d.Slides) > 1 && last.Kind == KindSection {
		last.Kind = KindClosing
		last.Part = 0
	}
}

func trimBlank(c []srcLine) []srcLine {
	for len(c) > 0 && strings.TrimSpace(c[0].text) == "" {
		c = c[1:]
	}
	for len(c) > 0 && strings.TrimSpace(c[len(c)-1].text) == "" {
		c = c[:len(c)-1]
	}
	return c
}

func (d *Deck) buildSlide(c []srcLine, first bool) *Slide {
	s := &Slide{Kind: KindContent, File: c[0].file, Line: c[0].no, Dir: filepath.Dir(c[0].file)}
	var body, notes, points []string
	// directives may come before the heading
	for len(c) > 0 {
		t := c[0].text
		if strings.TrimSpace(t) == "" {
			c = c[1:]
			continue
		}
		name, val, ok := ParseDirective(t)
		if !ok {
			break
		}
		if !d.slideDirective(s, c[0], name, val) {
			body = append(body, t)
		}
		c = c[1:]
	}
	if len(c) > 0 {
		head := c[0].text
		switch {
		case h2RE.MatchString(head):
			s.Title = h2RE.FindStringSubmatch(head)[1]
			c = c[1:]
		case h1RE.MatchString(head):
			s.Title = h1RE.FindStringSubmatch(head)[1]
			s.Kind = KindSection
			if first {
				s.Kind = KindTitle
			}
			c = c[1:]
		}
	}
	// Notes: and Points: are trailing blocks in either order; once inside one,
	// the other's label switches capture so neither swallows the other.
	cur := "" // "", "notes", or "points"
	var ft fenceTracker
	for _, l := range c {
		if cur != "" {
			if m := notesRE.FindStringSubmatch(l.text); m != nil {
				cur, notes = "notes", append(notes, m[1])
				continue
			}
			if m := pointsRE.FindStringSubmatch(l.text); m != nil {
				cur, points = "points", append(points, m[1])
				continue
			}
			if cur == "notes" {
				notes = append(notes, l.text)
			} else {
				points = append(points, l.text)
			}
			continue
		}
		if ft.inFence(l.text) {
			body = append(body, l.text)
			continue
		}
		if m := notesRE.FindStringSubmatch(l.text); m != nil {
			cur = "notes"
			notes = append(notes, m[1])
			continue
		}
		if m := pointsRE.FindStringSubmatch(l.text); m != nil {
			cur = "points"
			points = append(points, m[1])
			continue
		}
		if name, val, ok := ParseDirective(l.text); ok {
			if d.slideDirective(s, l, name, val) {
				continue // consumed
			}
		} else if h1RE.MatchString(l.text) && !strings.HasPrefix(strings.TrimSpace(l.text), "##") {
			d.problem(l.file, l.no, "a second `# H1` inside a slide; separate slides with `---`")
		}
		body = append(body, l.text)
	}
	if len(notes) > 0 {
		if m := minutesRE.FindStringSubmatch(notes[0]); m != nil {
			s.Minutes, _ = strconv.ParseFloat(m[1], 64)
			s.HasMinutes = true
			notes[0] = strings.TrimSpace(minutesRE.ReplaceAllString(notes[0], ""))
		}
	}
	s.Body = strings.TrimSpace(strings.Join(body, "\n"))
	s.Notes = strings.TrimSpace(strings.Join(notes, "\n"))
	s.Points = strings.TrimSpace(strings.Join(points, "\n"))
	return s
}

// slideDirective applies a slide-level directive and validates body-level
// ones. It returns true when the line is consumed.
func (d *Deck) slideDirective(s *Slide, l srcLine, name, val string) bool {
	kind, known := Directives[name]
	if !known {
		d.problem(l.file, l.no, "unknown directive %q (known: %s)", name, strings.Join(directiveNames(), ", "))
		return true
	}
	switch name {
	case "layout":
		if msg := oneOf("layout", val, Layouts); msg != "" {
			d.problem(l.file, l.no, "%s", msg)
		}
		s.Layout = val
	case "transition":
		if msg := oneOf("transition", val, Transitions); msg != "" {
			d.problem(l.file, l.no, "%s", msg)
		}
		s.Transition = val
	case "background":
		d.checkBackground(l.file, l.no, s.Dir, val)
		s.Background = val
	case "footer":
		s.Footer = val
	case "labs":
		if m := labLinkRE.FindAllStringSubmatch(val, -1); len(m) > 0 {
			for _, g := range m {
				s.Labs = append(s.Labs, Lab{Text: strings.TrimSpace(g[1]), URL: g[2]})
			}
		} else {
			for _, u := range strings.Split(val, ",") {
				u = strings.TrimSpace(u)
				if u == "" {
					continue
				}
				if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
					d.problem(l.file, l.no, "labs: %q is not an http(s) URL or a [text](url) link", u)
					continue
				}
				s.Labs = append(s.Labs, Lab{URL: u})
			}
		}
		if len(s.Labs) == 0 {
			d.problem(l.file, l.no, "labs needs at least one lab: <!-- labs: [text](https://…) -->")
		}
	case "class":
		s.Classes = append(s.Classes, strings.Fields(val)...)
	case "svg":
		f := strings.Fields(val)
		if len(f) == 0 || !strings.HasSuffix(strings.ToLower(f[0]), ".svg") {
			d.problem(l.file, l.no, "svg needs a file: <!-- svg: assets/canvas.svg state words -->")
		} else if !exists(filepath.Join(s.Dir, f[0])) {
			d.problem(l.file, l.no, "missing file: %s", f[0])
		}
	case "image":
		if _, err := ParseImageAttrs(val); err != nil {
			d.problem(l.file, l.no, "%v", err)
		}
	case "column":
		if val != "" {
			d.problem(l.file, l.no, "column takes no value")
		}
	}
	return kind == "slide"
}

// IsImagePath reports whether a background value is a relative image path
// rather than a colour or gradient.
func IsImagePath(v string) bool {
	return imageExtRE.MatchString(v) && !strings.ContainsAny(v, "() ")
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func directiveNames() []string {
	names := make([]string, 0, len(Directives))
	for k := range Directives {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

// rel shortens an absolute path to one relative to the working directory.
func rel(p string) string {
	if wd, err := os.Getwd(); err == nil {
		if r, err := filepath.Rel(wd, p); err == nil && !strings.HasPrefix(r, "..") {
			return r
		}
	}
	return p
}

// isDeckFile reports whether a filename is a deck source (Markdown or AsciiDoc).
func isDeckFile(name string) bool {
	l := strings.ToLower(name)
	return strings.HasSuffix(l, ".md") || strings.HasSuffix(l, ".adoc") || strings.HasSuffix(l, ".asciidoc")
}

// List returns the decks in dir: every *.md / *.adoc / *.asciidoc except
// README.md and files whose name starts with `_` or `.` (modules meant only for
// inclusion). A file path is returned as is.
func List(path string) ([]string, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !st.IsDir() {
		return []string{path}, nil
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		n := e.Name()
		if e.IsDir() || !isDeckFile(n) || strings.HasPrefix(n, "_") ||
			strings.HasPrefix(n, ".") || strings.EqualFold(n, "README.md") {
			continue
		}
		out = append(out, filepath.Join(path, n))
	}
	sort.Strings(out)
	return out, nil
}
