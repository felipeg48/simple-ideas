// Package render turns a parsed deck into one self-contained HTML page:
// CSS, JS, SVG canvases and images are inlined; the only optional network
// resource is the Mermaid script, loaded when a deck has a mermaid fence.
package render

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/assets"
	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/yuin/goldmark"
)

// MermaidURL is the default Mermaid script (pinned); `mermaid:` in front
// matter overrides it, `mermaid: off` keeps mermaid fences as code.
const MermaidURL = "https://cdn.jsdelivr.net/npm/mermaid@11.4.1/dist/mermaid.min.js"

// Options control a render.
type Options struct {
	// LiveReload adds the si play reload client (EventSource on /_si/events).
	LiveReload bool
	// ShowProblems shows a banner listing the deck problems (si play).
	ShowProblems bool
	// UserCSS is extra CSS loaded in the si.user layer, after the theme.
	UserCSS []Stylesheet
	// Media says how images, videos and background files are referenced.
	Media MediaMode
}

// Result is a rendered deck.
type Result struct {
	HTML     []byte
	Problems []deck.Problem // deck problems plus render problems (missing images, ...)
	Warnings []deck.Problem // non-fatal: large inlined files
	Media    []MediaFile    // files to copy next to the HTML (MediaCopy)
	Steps    int
	Canvases int
}

type deckRenderer struct {
	d         *deck.Deck
	opt       Options
	md        goldmark.Markdown
	lastState map[string][]string
	words     map[string]bool
	files     map[string]string
	svgN      int
	mermaid   bool
	problems  []deck.Problem
	warnings  []deck.Problem
}

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
	dr := &deckRenderer{d: d, opt: opt, lastState: map[string][]string{}, words: map[string]bool{}, files: map[string]string{}}
	dr.md = dr.markdown()
	var slides strings.Builder
	res := Result{}
	for i, s := range d.Slides {
		sh, sc := dr.slide(s, i+1)
		slides.WriteString(sh)
		res.Steps += sc.steps
		res.Canvases += len(sc.canvases)
	}
	res.Problems = append(append([]deck.Problem{}, d.Problems...), dr.problems...)
	res.Warnings = dr.warnings
	for src, rel := range dr.files {
		res.Media = append(res.Media, MediaFile{Src: src, Rel: rel})
	}
	sort.Slice(res.Media, func(i, j int) bool { return res.Media[i].Rel < res.Media[j].Rel })
	res.HTML = dr.page(slides.String(), res.Problems, opt)
	return res
}

func (dr *deckRenderer) slide(s *deck.Slide, n int) (string, *slideCtx) {
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
	var b strings.Builder
	fmt.Fprintf(&b, `<section class="%s" id="s%d" data-title="%s" data-layout="%s"`,
		html.EscapeString(strings.Join(classes, " ")), n, html.EscapeString(plain(s.Title)), layout)
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
	b.WriteString(`<div class="si-inner">` + "\n")
	title := dr.inline(s.Title)
	bodyClass := "si-body"
	if sc.cols {
		bodyClass += " si-cols"
	}
	switch s.Kind {
	case deck.KindTitle, deck.KindClosing:
		fmt.Fprintf(&b, `<h1 class="si-heading">%s</h1>`+"\n", title)
		if body != "" {
			fmt.Fprintf(&b, `<div class="si-sub %s">%s</div>`+"\n", bodyClass, body)
		}
		if s.Kind == deck.KindTitle {
			if by := strings.Trim(dr.d.Meta.Author+" · "+dr.d.Meta.Date, " ·"); by != "" {
				fmt.Fprintf(&b, `<p class="si-byline">%s</p>`+"\n", html.EscapeString(by))
			}
		}
	case deck.KindSection:
		fmt.Fprintf(&b, `<p class="si-kicker">Part %d</p>`+"\n", s.Part)
		fmt.Fprintf(&b, `<h1 class="si-heading">%s</h1>`+"\n", title)
		if body != "" {
			fmt.Fprintf(&b, `<div class="si-sub %s">%s</div>`+"\n", bodyClass, body)
		}
	default:
		if s.Title != "" {
			fmt.Fprintf(&b, `<header class="si-head"><h2 class="si-heading">%s</h2></header>`+"\n", title)
		}
		fmt.Fprintf(&b, `<div class="%s">%s</div>`+"\n", bodyClass, body)
	}
	b.WriteString("</div>\n")
	if s.Notes != "" || s.HasMinutes {
		fmt.Fprintf(&b, `<aside class="si-notes" hidden>%s</aside>`+"\n", dr.postprocess(dr.toHTML(s.Notes, nil), s.Dir, sc))
	}
	b.WriteString("</section>\n")
	return b.String(), sc
}

var (
	imgSrcRE = regexp.MustCompile(`(<img\b[^>]*?\ssrc=")([^"]+)"`)
	mdLinkRE = regexp.MustCompile(`(href=")([^"#:]+)\.md(#[^"]*)?"`)
	tagRE    = regexp.MustCompile(`<[^>]+>|[*_` + "`" + `]`)
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

func plain(s string) string { return strings.TrimSpace(tagRE.ReplaceAllString(s, "")) }

var pageSizes = map[string]string{"16:9": "13.333in 7.5in", "16:10": "12in 7.5in", "4:3": "10in 7.5in", "1:1": "7.5in 7.5in", "9:16": "7.5in 13.333in"}

func (dr *deckRenderer) page(slides string, problems []deck.Problem, opt Options) []byte {
	d := dr.d
	m := d.Meta
	theme := m.Theme
	if theme == "" {
		theme = "studio"
	}
	aspect := m.Aspect
	if aspect == "" {
		aspect = "16:9"
	}
	lang := m.Lang
	if lang == "" {
		lang = "en"
	}
	var b bytes.Buffer
	modeAttr := ""
	if m.Mode == "light" || m.Mode == "dark" {
		modeAttr = ` data-si-mode="` + m.Mode + `"`
	}
	fmt.Fprintf(&b, "<!doctype html>\n<html lang=\"%s\" data-si-theme=\"%s\"%s>\n<head>\n", html.EscapeString(lang), theme, modeAttr)
	b.WriteString(`<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover">
<meta name="color-scheme" content="light dark">
<meta name="generator" content="simple-ideas">
<link rel="icon" href="data:,">
`)
	fmt.Fprintf(&b, "<title>%s</title>\n", html.EscapeString(plain(d.Title())))
	b.WriteString(`<style id="si-css">` + "\n" + assets.LayerOrder + assets.LayoutCSS + assets.ThemeCSS(theme) + assets.ThemeLooksCSS)
	b.WriteString(dr.helperCSS())
	fmt.Fprintf(&b, "@page{size:%s;margin:0}\n", pageSizes[aspect])
	if m.Accent != "" {
		fmt.Fprintf(&b, "@layer si.theme{:root,:root:not([data-si-mode]),:root[data-si-mode=\"dark\"]{--si-accent:%s}}\n", m.Accent)
	}
	b.WriteString("</style>\n")
	if css := UserCSS(opt.UserCSS); css != "" {
		b.WriteString(`<style id="si-user">` + "\n" + css + "</style>\n")
	}
	// restore the viewer's light/dark choice before first paint (a mode pinned in front matter wins)
	if modeAttr == "" {
		b.WriteString(`<script>try{var m=localStorage.getItem('si-mode');if(m)document.documentElement.setAttribute('data-si-mode',m)}catch(e){}</script>` + "\n")
	}
	b.WriteString("</head>\n<body>\n")
	ratio := strings.Replace(aspect, ":", "/", 1)
	fmt.Fprintf(&b, "<main class=\"si-deck\" aria-label=\"%s\" style=\"--_aspect:%s\" data-si-tx=\"none\">\n", html.EscapeString(plain(d.Title())), ratio)
	b.WriteString(slides)
	b.WriteString("</main>\n")
	b.WriteString(chromeHTML)
	if opt.ShowProblems && len(problems) > 0 {
		fmt.Fprintf(&b, "<aside class=\"si-problems\"><b>%d problem(s) in this deck</b><ul>", len(problems))
		for _, p := range problems {
			fmt.Fprintf(&b, "<li>%s</li>", html.EscapeString(p.String()))
		}
		b.WriteString("</ul></aside>\n")
	}
	plan := make([]float64, len(d.Slides))
	for i, s := range d.Slides {
		plan[i] = s.Minutes
	}
	data, _ := json.Marshal(map[string]any{
		"name": d.Name, "title": plain(d.Title()), "minutes": d.Minutes(), "plan": plan, "transition": m.Transition, "aspect": ratio,
	})
	fmt.Fprintf(&b, "<script type=\"application/json\" id=\"si-data\">%s</script>\n", data)
	if dr.mermaid {
		src := MermaidURL
		if m.Mermaid != "" {
			src = m.Mermaid
		}
		fmt.Fprintf(&b, "<script src=\"%s\"></script>\n", html.EscapeString(src))
	}
	presenter, _ := json.Marshal(assets.PresenterHTML)
	b.WriteString("<script>\nwindow.SI_PRESENTER = " + strings.ReplaceAll(string(presenter), "</", `<\/`) + ";\n")
	b.WriteString(assets.DeckJS)
	b.WriteString("</script>\n")
	if opt.LiveReload {
		b.WriteString(LiveReloadJS)
	}
	b.WriteString("</body>\n</html>\n")
	return b.Bytes()
}

const chromeHTML = `<nav class="si-bar" aria-label="Deck controls">
<span class="si-controls">
<button type="button" data-si-action="prev" aria-label="Previous">&larr;</button><button type="button" data-si-action="next" aria-label="Next">&rarr;</button>
<button type="button" data-si-action="overview" title="Overview (o)">Overview</button><button type="button" data-si-action="presenter" title="Presenter teleprompter (p)">Presenter</button>
<button type="button" data-si-action="mode" title="Light / dark (t)">Theme</button><button type="button" data-si-action="fullscreen" title="Full screen (f)">Full screen</button><button type="button" data-si-action="help" title="Keys (?)">?</button>
</span>
<span class="si-counter" aria-live="polite"></span>
</nav>
<div class="si-progress"></div>
<div class="si-help" role="dialog" aria-label="Keyboard shortcuts"><div>
<table>
<tr><td>&rarr; &darr; Space PgDn Enter</td><td>next step, then next slide</td></tr>
<tr><td>&larr; &uarr; Shift+Space PgUp</td><td>previous step or slide</td></tr>
<tr><td>Home / End</td><td>first / last slide</td></tr>
<tr><td>o</td><td>overview grid; click a slide to open it</td></tr>
<tr><td>p</td><td>presenter teleprompter: notes, timer, next slide; drives this window</td></tr>
<tr><td>t</td><td>light / dark</td></tr>
<tr><td>f</td><td>full screen</td></tr>
<tr><td>swipe</td><td>next / previous on touch screens</td></tr>
<tr><td>#5 or #5.2</td><td>open slide 5 (at step 2)</td></tr>
<tr><td>print</td><td>one slide per page, every step shown</td></tr>
</table></div></div>
`

// LiveReloadJS reloads the page when si play reports a change; the URL hash
// keeps the slide and step.
const LiveReloadJS = `<script>(function(){if(!window.EventSource)return;var es=new EventSource('/_si/events');es.addEventListener('reload',function(){location.reload()});})();</script>` + "\n"

// IndexEntry is one deck on the index page.
type IndexEntry struct {
	Href     string
	Title    string
	Subtitle string
	Slides   int
	Minutes  float64
}

// Index renders the deck index page.
func Index(title string, entries []IndexEntry, opt Options) []byte {
	var b bytes.Buffer
	b.WriteString("<!doctype html>\n<html lang=\"en\" data-si-theme=\"studio\">\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString(`<meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover">
<meta name="color-scheme" content="light dark">
<link rel="icon" href="data:,">
`)
	fmt.Fprintf(&b, "<title>%s</title>\n<style>\n%s*,*::before,*::after{box-sizing:border-box}\nbody{margin:0;background:var(--si-bg);color:var(--si-fg)}\n%s%s</style>\n",
		html.EscapeString(title), assets.LayerOrder, assets.ThemeCSS("studio"), assets.ThemeLooksCSS)
	if css := UserCSS(opt.UserCSS); css != "" {
		b.WriteString("<style>\n" + css + "</style>\n")
	}
	b.WriteString(`<script>try{var m=localStorage.getItem('si-mode');if(m)document.documentElement.setAttribute('data-si-mode',m)}catch(e){}</script>` + "\n")
	fmt.Fprintf(&b, "</head>\n<body>\n<main class=\"si-index\">\n<h1>%s</h1>\n<p>%d deck(s)</p>\n<div class=\"grid\">\n", html.EscapeString(title), len(entries))
	for _, e := range entries {
		mins := ""
		if e.Minutes > 0 {
			mins = fmt.Sprintf(" · %s min", strconv.FormatFloat(e.Minutes, 'f', -1, 64))
		}
		fmt.Fprintf(&b, "<a class=\"si-card\" href=\"%s\"><b>%s</b><span>%s</span><small>%d slides%s</small></a>\n",
			html.EscapeString(e.Href), html.EscapeString(e.Title), html.EscapeString(e.Subtitle), e.Slides, mins)
	}
	b.WriteString("</div>\n</main>\n")
	if opt.LiveReload {
		b.WriteString(LiveReloadJS)
	}
	b.WriteString("</body>\n</html>\n")
	return b.Bytes()
}

// Entry builds an index entry for a parsed deck.
func Entry(d *deck.Deck) IndexEntry {
	e := IndexEntry{Href: d.Name + ".html", Title: plain(d.Title()), Slides: len(d.Slides), Minutes: d.Minutes()}
	if len(d.Slides) > 0 {
		for _, line := range strings.Split(d.Slides[0].Body, "\n") {
			if t := strings.TrimSpace(line); t != "" && !strings.HasPrefix(t, "<!--") {
				e.Subtitle = plain(t)
				break
			}
		}
	}
	return e
}
