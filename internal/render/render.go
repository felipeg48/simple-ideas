// Package render turns a parsed deck into one self-contained HTML page:
// CSS, JS, SVG canvases and images are inlined; the only optional network
// resource is the Mermaid script, loaded when a deck has a mermaid fence.
package render

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/config"
	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
)

// MermaidURL is the default Mermaid script (pinned); `mermaid:` in front
// matter overrides it, `mermaid: off` keeps mermaid fences as code.
const MermaidURL = "https://cdn.jsdelivr.net/npm/mermaid@11.4.1/dist/mermaid.min.js"

// Options control a render.
type Options struct {
	// LiveReload adds the si play reload client (EventSource on /_si/events).
	LiveReload bool
	// Dev upgrades the reload client to an in-place hot reload (si play --dev):
	// it morphs the changed slides without a full-page refresh, preserving the
	// current slide, step and overview, and shows a parse-error overlay.
	Dev bool
	// ShowProblems shows a banner listing the deck problems (si play).
	ShowProblems bool
	// UserCSS is extra CSS loaded in the si.user layer, after the theme.
	UserCSS []Stylesheet
	// Media says how images, videos and background files are referenced.
	Media MediaMode
	// SharedAssets (si build --share-assets, or --no-inline-media) links the
	// large libraries of a folder or site build (three.js for 3D mind maps)
	// from ShareDir, written once, instead of inlining them into every page;
	// a page then needs that folder beside it. AssetPrefix is set per page
	// ("../" for each folder level below the build's root).
	SharedAssets bool
	AssetPrefix  string
	// Theme overrides the deck's `theme:` front matter; empty keeps it.
	Theme string
	// Config is an optional si.conf: default theme, key remaps, feature toggles.
	Config *config.Config
	// Terminal enables the play terminal drawer in the page (SI_CONFIG.terminal).
	// Set only by si play on a loopback address; always false for build/export,
	// so a self-contained deck never loads the terminal runtime.
	Terminal bool
	// Remote joins the page to the `si play --remote` relay (SI_CONFIG.remote):
	// presenter state, commands and ink also travel over /_si/remote/*.
	// Play only; a built deck keeps BroadcastChannel alone.
	Remote bool
	// Pages are the other pages find (/) offers (see Pages); nil offers none.
	Pages []Page
	// Prev and Next link a lab to its neighbours built from the same
	// directory (si build); empty Href means no link.
	Prev, Next DocLink
	// Guide is the guide a page belongs to (its _nav.md); nil for a page
	// outside a guide. It gives the sidebar, breadcrumb, Prev / Next (which
	// then replace Prev and Next), the edit link and the shared partials.
	Guide *deck.Guide
	// Chapter is a book page's number ("3", "A"; "" for front matter) and
	// ChapterKind its kind ("Chapter", "Appendix"): sections, figures and
	// tables are numbered from it. Set by RenderGuide for a book.
	Chapter, ChapterKind string
	// BookPDF is the href of the book's PDF, shown on the cover and the bar.
	BookPDF  string
	bookPage string // a generated book page: "cover", "contents", "index", "book"
}

// DocLink is a link to another lab page.
type DocLink struct{ Href, Title string }

// Result is a rendered deck.
type Result struct {
	HTML     []byte
	Problems []deck.Problem // deck problems plus render problems (missing images, ...)
	Warnings []deck.Problem // non-fatal: large inlined files
	Media    []MediaFile    // files to copy next to the HTML (MediaCopy)
	Steps    int
	Canvases int

	// documents: what a whole-book pass needs after the page is rendered
	body   string            // the page's text (the main column's content)
	toc    []tocEntry        // its h2 / h3 headings
	labels map[string]string // id -> "Section 1.2", "Figure 1.1", "Table 1.1"
}

type deckRenderer struct {
	d             *deck.Deck
	opt           Options
	md            goldmark.Markdown
	lastState     map[string][]string
	words         map[string]bool
	files         map[string]string
	svgN          int
	stageN        int
	callouts      map[ast.Node]bool // fenced code blocks followed by a <N> callout list
	sec, fig, tbl int               // book: section, figure and table counters of the chapter
	labels        map[string]string // book: id -> label of numbered sections, figures, tables
	cur           *slideCtx         // the slide being rendered, for the stage fence's steps
	mermaid       bool
	mindmap       bool                 // a ```mindmap fence: the page loads mindmap.css and mindmap.js
	codewin       bool                 // a code window: the page loads codewin.css and codewin.js
	cw            map[ast.Node]*cwShow // the first fence of each code window and its steps
	mindmap3d     bool                 // a map starts in 3D: three.js and mindmap3d.js are inlined too
	mmN           int                  // ```mindmap fences so far (ids of their fallback lists, their build steps)
	curDir        string               // directory of the part being rendered (fences resolve paths against it)
	themeName     string               // resolved theme name (or "custom" for a CSS-file theme)
	themeCSS      string               // the si.theme layer CSS for this deck
	fontsCSS      string               // fonts: inlined @font-face rules and font tokens (fonts.go)
	fontSubsets   []string             // fonts-subset: the subsets kept from a split family
	anims         []animEntry          // resolved break-timer animations (empty if none)
	problems      []deck.Problem
	warnings      []deck.Problem
	wmDone        bool   // watermark resolved once (deck-level)
	wmKind        string // "" | "text" | "img"
	wmText        string // watermark text
	wmImg         string // watermark image as a data URI (inlined once)
	runbookJSON   string // window.SI_RUNBOOK payload (play terminal drawer); "" if no runbook
	demosJSON     string // window.SI_DEMOS payload (play terminal drawer); "" if no demos
	doc           bool   // rendering a lab document (type: lab), not slides
}

// animEntry is one resolved break-timer animation: its name, kind ("js"/"svg")
// and source. timer.animation may name several (comma-separated), which the HUD
// offers as a toggle.
type animEntry struct{ name, kind, code string }

func (dr *deckRenderer) at(sc *slideCtx, format string, a ...any) deck.Problem {
	p := deck.Problem{File: dr.d.Path, Msg: fmt.Sprintf(format, a...)}
	if sc != nil {
		p.File, p.Line = sc.s.File, sc.s.Line
	}
	if wd, err := os.Getwd(); err == nil {
		if r, err := filepath.Rel(wd, p.File); err == nil && !strings.HasPrefix(r, "..") {
			p.File = r
		}
	}
	return p
}

func (dr *deckRenderer) problem(sc *slideCtx, format string, a ...any) {
	dr.problems = append(dr.problems, dr.at(sc, format, a...))
}

func (dr *deckRenderer) warn(sc *slideCtx, format string, a ...any) {
	dr.warnings = append(dr.warnings, dr.at(sc, format, a...))
}

// Deck renders a parsed deck.
func Deck(d *deck.Deck, opt Options) Result {
	if d.IsDoc() {
		return Document(d, opt)
	}
	dr := &deckRenderer{d: d, opt: opt, lastState: map[string][]string{}, words: map[string]bool{}, files: map[string]string{}}
	dr.md = dr.markdown()
	var slides strings.Builder
	res := Result{}
	dr.resolveTheme()
	dr.resolveAnim()
	dr.resolveRunbook()
	dr.resolveDemos()
	topics := Topics(d)
	for i, s := range d.Slides {
		sh, sc := dr.slide(s, i+1, topics[i])
		slides.WriteString(sh)
		res.Steps += sc.steps
		res.Canvases += len(sc.canvases)
	}
	dr.finish(&res)
	res.HTML = dr.page(slides.String(), res.Problems, opt)
	return res
}

// finish collects the parser and render problems, warnings and media files.
func (dr *deckRenderer) finish(res *Result) {
	res.Problems = append(append([]deck.Problem{}, dr.d.Problems...), dr.problems...)
	res.Warnings = append(append([]deck.Problem{}, dr.d.Warnings...), dr.warnings...)
	for src, rel := range dr.files {
		res.Media = append(res.Media, MediaFile{Src: src, Rel: rel})
	}
	sort.Slice(res.Media, func(i, j int) bool { return res.Media[i].Rel < res.Media[j].Rel })
}

func (dr *deckRenderer) slide(s *deck.Slide, n, topic int) (string, *slideCtx) {
	sc := &slideCtx{s: s}
	body := dr.postprocess(dr.toHTML(s.Body, sc), s.Dir, sc)
	// remember where this slide leaves each canvas, for relative states
	for _, c := range sc.canvases {
		state := c.base
		for _, ops := range sc.ops {
			state = applyOps(state, ops)
		}
		dr.lastState[c.path] = state
	}
	classes := []string{"si-slide", "si-" + string(s.Kind)}
	bgClasses, style, layer := dr.background(s, sc)
	classes = append(classes, bgClasses...)
	classes = append(classes, s.Classes...)
	layout := s.Layout
	if layout == "" {
		layout = "default"
	}
	if len(s.Labs) > 0 {
		classes = append(classes, "si-lab")
		layout = "center"
	}
	var b strings.Builder
	fmt.Fprintf(&b, `<section class="%s" id="s%d" data-title="%s" data-layout="%s"`,
		html.EscapeString(strings.Join(classes, " ")), n, html.EscapeString(plain(s.Title)), layout)
	fmt.Fprintf(&b, ` data-topic="%d"`, topic)
	if s.Transition != "" {
		fmt.Fprintf(&b, ` data-transition="%s"`, s.Transition)
	}
	if s.HasMinutes {
		fmt.Fprintf(&b, ` data-minutes="%s"`, strconv.FormatFloat(s.Minutes, 'f', -1, 64))
	}
	if style != "" {
		fmt.Fprintf(&b, ` style="%s"`, html.EscapeString(style))
	}
	fmt.Fprintf(&b, ` data-src="%s:%d">`+"\n", html.EscapeString(filepath.Base(s.File)), s.Line)
	b.WriteString(layer)
	b.WriteString(dr.watermark(sc))
	b.WriteString(`<div class="si-inner">` + "\n")
	title := dr.inline(s.Title)
	bodyClass := "si-body"
	if sc.cols {
		bodyClass += " si-cols"
	}
	eyebrow := ""
	if s.Eyebrow != "" {
		eyebrow = `<p class="si-kicker">` + html.EscapeString(s.Eyebrow) + "</p>\n"
	}
	switch {
	case len(s.Labs) > 0:
		b.WriteString(dr.labSlide(s))
	case s.Kind == deck.KindTitle || s.Kind == deck.KindClosing:
		b.WriteString(eyebrow)
		fmt.Fprintf(&b, `<h1 class="si-heading">%s</h1>`+"\n", title)
		if body != "" {
			fmt.Fprintf(&b, `<div class="si-sub %s">%s</div>`+"\n", bodyClass, body)
		}
		if s.Kind == deck.KindTitle {
			if by := strings.Trim(dr.d.Meta.Author+" · "+dr.d.Meta.Date, " ·"); by != "" {
				fmt.Fprintf(&b, `<p class="si-byline">%s</p>`+"\n", html.EscapeString(by))
			}
			if notice := copyrightLine(dr.d.Meta.Year, dr.d.Meta.Copyright); notice != "" {
				fmt.Fprintf(&b, `<p class="si-colophon">%s</p>`+"\n", html.EscapeString(notice))
			}
		}
	case s.Kind == deck.KindSection:
		if eyebrow == "" {
			eyebrow = fmt.Sprintf(`<p class="si-kicker">Part %d</p>`+"\n", s.Part)
		}
		b.WriteString(eyebrow)
		fmt.Fprintf(&b, `<h1 class="si-heading">%s</h1>`+"\n", title)
		if body != "" {
			fmt.Fprintf(&b, `<div class="si-sub %s">%s</div>`+"\n", bodyClass, body)
		}
	default:
		if s.Title != "" || eyebrow != "" {
			h := ""
			if s.Title != "" {
				h = `<h2 class="si-heading">` + title + `</h2>`
			}
			fmt.Fprintf(&b, `<header class="si-head">%s%s</header>`+"\n", strings.TrimSuffix(eyebrow, "\n"), h)
		}
		fmt.Fprintf(&b, `<div class="%s">%s</div>`+"\n", bodyClass, body)
	}
	b.WriteString("</div>\n")
	b.WriteString(dr.footer(s))
	if s.Notes != "" || s.HasMinutes {
		notes := dr.postprocess(dr.toHTML(s.Notes, nil), s.Dir, sc)
		// [CLICK] in the script marks where the speaker advances (a build step,
		// a canvas change, the next slide): a cue in the teleprompter, never read aloud
		clicks := len(clickCueRE.FindAllStringIndex(notes, -1))
		notes = clickCueRE.ReplaceAllString(notes, `<span class="si-cue">click</span>`)
		if clicks > 0 && (clicks < sc.steps || clicks > sc.steps+1) {
			dr.warnings = append(dr.warnings, dr.at(sc, "notes: %d [CLICK] cue(s) for %d build step(s): one per step, plus at most one for the next slide", clicks, sc.steps))
		}
		fmt.Fprintf(&b, `<aside class="si-notes" hidden>%s</aside>`+"\n", notes)
	}
	if s.Points != "" {
		fmt.Fprintf(&b, `<aside class="si-points" hidden>%s</aside>`+"\n", dr.postprocess(dr.toHTML(s.Points, nil), s.Dir, sc))
	}
	b.WriteString("</section>\n")
	return b.String(), sc
}

var (
	imgSrcRE   = regexp.MustCompile(`(<img\b[^>]*?\ssrc=")([^"]+)"`)
	clickCueRE = regexp.MustCompile(`\[(?i:click)\]`)
	mdLinkRE   = regexp.MustCompile(`(href=")([^"#:]+)\.(?:md|adoc|asciidoc)(#[^"]*)?"`)
	tagRE      = regexp.MustCompile(`<[^>]+>|[*_` + "`" + `]`)
)

// postprocess resolves relative images (data URI, link or copy) and points
// .md links at the generated .html pages.
func (dr *deckRenderer) postprocess(h, dir string, sc *slideCtx) string {
	h = imgSrcRE.ReplaceAllStringFunc(h, func(s string) string {
		m := imgSrcRE.FindStringSubmatch(s)
		src := html.UnescapeString(m[2])
		if isExternal(src) {
			return s
		}
		uri, ok := dr.media(dir, src, sc)
		if !ok {
			dr.problem(sc, "missing file: %s", src)
			return s
		}
		return m[1] + html.EscapeString(uri) + `"`
	})
	return mdLinkRE.ReplaceAllString(h, `${1}${2}.html${3}"`)
}

func plain(s string) string {
	return strings.TrimSpace(markRE.ReplaceAllString(tagRE.ReplaceAllString(s, ""), "$1"))
}

// markRE is a ==highlight== mark, dropped from plain text (titles, search).
var markRE = regexp.MustCompile(`==([^=\n]+?)==`)

// Topics numbers the overview column of every slide: 0 for the framing slides
// at the start of the deck (title, agenda), then a new topic at each section
// slide or at each included file that has no section slide of its own, so a
// deck split into `## …` modules still gets one column per module.
func Topics(d *deck.Deck) []int {
	out := make([]int, len(d.Slides))
	topic, n, prevFile, prevSection := 0, 0, "", false
	for i, s := range d.Slides {
		switch {
		case s.Kind == deck.KindSection:
			n++
			topic = n
		case s.File != prevFile && s.File != d.Path && !prevSection:
			n++ // an included module that starts without a section slide
			topic = n
		}
		out[i] = topic
		prevFile, prevSection = s.File, s.Kind == deck.KindSection
	}
	return out
}
