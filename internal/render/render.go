// Package render turns a parsed deck into one self-contained HTML page:
// CSS, JS, SVG canvases and images are inlined; the only optional network
// resource is the Mermaid script, loaded when a deck has a mermaid fence.
package render

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/felipeg48/simple-ideas/internal/assets"
	"github.com/felipeg48/simple-ideas/internal/config"
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
	// Theme overrides the deck's `theme:` front matter; empty keeps it.
	Theme string
	// Config is an optional si.conf: default theme, key remaps, feature toggles.
	Config *config.Config
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
	stageN    int
	cur       *slideCtx // the slide being rendered, for the stage fence's steps
	mermaid   bool
	themeName string // resolved theme name (or "custom" for a CSS-file theme)
	themeCSS  string // the si.theme layer CSS for this deck
	animName  string // resolved timer-animation name (empty if none)
	animKind  string // "js" or "svg"
	animCode  string // the animation source
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
	dr.resolveTheme()
	dr.resolveAnim()
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

// resolveAnim resolves the configured break-timer animation into dr.anim*.
// timer.animation names a built-in (globe/ranges/roaches) or, with
// timer.animation.path set, a <name>.js|.svg file in that directory (resolved
// relative to cwd, then the deck dir). A configured-but-missing animation is a
// deck problem; no configured animation leaves the fields empty (no layer).
func (dr *deckRenderer) resolveAnim() {
	if dr.opt.Config == nil {
		return
	}
	t := dr.opt.Config.Timer
	if t.Animation == "" {
		return
	}
	if t.AnimationPath != "" {
		for _, k := range []string{"js", "svg"} {
			p := filepath.Join(t.AnimationPath, t.Animation+"."+k)
			if !filepath.IsAbs(p) {
				if _, err := os.Stat(p); err != nil {
					p = filepath.Join(filepath.Dir(dr.d.Path), t.AnimationPath, t.Animation+"."+k)
				}
			}
			if b, err := os.ReadFile(p); err == nil {
				dr.animName, dr.animKind, dr.animCode = t.Animation, k, string(b)
				return
			}
		}
	}
	if code, kind, ok := assets.Animation(t.Animation); ok {
		dr.animName, dr.animKind, dr.animCode = t.Animation, kind, code
		return
	}
	dr.problems = append(dr.problems, deck.Problem{File: dr.d.Path, Msg: fmt.Sprintf("timer.animation: no animation %q (built-ins: globe, ranges, roaches)", t.Animation)})
}

// resolveTheme sets dr.themeName and dr.themeCSS. The theme is the --theme flag,
// then the deck's theme: front matter, then si.conf, then studio. A value ending
// in .css is a user theme file (resolved relative to cwd, then the deck dir); an
// http(s):// value is fetched at build time and cached. Either way its contents
// become the si.theme layer (the built HTML stays self-contained — the fetch is
// build-time only). A read/fetch error is a deck problem and falls back to studio.
func (dr *deckRenderer) resolveTheme() {
	theme := dr.opt.Theme
	if theme == "" {
		theme = dr.d.Meta.Theme
	}
	if theme == "" && dr.opt.Config != nil {
		theme = dr.opt.Config.Theme
	}
	if theme == "" {
		theme = "studio"
	}
	if strings.HasPrefix(theme, "http://") || strings.HasPrefix(theme, "https://") {
		css, err := fetchThemeCSS(theme)
		if err != nil {
			dr.problems = append(dr.problems, deck.Problem{File: dr.d.Path, Msg: fmt.Sprintf("theme: cannot fetch %s: %v", theme, err)})
			dr.themeName, dr.themeCSS = "studio", assets.ThemeCSS("studio")
			return
		}
		dr.themeName, dr.themeCSS = "custom", "@layer si.theme{\n"+css+"\n}\n"
		return
	}
	if strings.HasSuffix(strings.ToLower(theme), ".css") {
		p := theme
		if !filepath.IsAbs(p) {
			if _, err := os.Stat(p); err != nil {
				p = filepath.Join(filepath.Dir(dr.d.Path), theme)
			}
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			dr.problems = append(dr.problems, deck.Problem{File: dr.d.Path, Msg: fmt.Sprintf("theme: cannot read %s: %v", theme, err)})
			dr.themeName, dr.themeCSS = "studio", assets.ThemeCSS("studio")
			return
		}
		dr.themeName = "custom"
		dr.themeCSS = "@layer si.theme{\n" + string(raw) + "\n}\n"
		return
	}
	dr.themeName = assets.CanonicalTheme(theme)
	dr.themeCSS = assets.ThemeCSS(dr.themeName)
}

// fetchThemeCSS downloads a theme stylesheet at build time and caches it for an
// hour under the temp dir (so `si play` does not refetch on every request). On a
// fetch error a fresh-enough cache is reused if present, otherwise the error is
// returned. The response is capped so a huge/garbage URL cannot blow up memory.
func fetchThemeCSS(url string) (string, error) {
	sum := sha256.Sum256([]byte(url))
	cache := filepath.Join(os.TempDir(), "si-theme-cache", fmt.Sprintf("%x.css", sum[:8]))
	if fi, err := os.Stat(cache); err == nil && time.Since(fi.ModTime()) < time.Hour {
		if b, err := os.ReadFile(cache); err == nil {
			return string(b), nil
		}
	}
	readCache := func() (string, bool) {
		if b, err := os.ReadFile(cache); err == nil {
			return string(b), true
		}
		return "", false
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		if b, ok := readCache(); ok {
			return b, nil // stale cache beats a hard failure
		}
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		if b, ok := readCache(); ok {
			return b, nil
		}
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20)) // 2 MiB cap
	if err != nil {
		return "", err
	}
	_ = os.MkdirAll(filepath.Dir(cache), 0o755)
	_ = os.WriteFile(cache, body, 0o644)
	return string(body), nil
}

func (dr *deckRenderer) page(slides string, problems []deck.Problem, opt Options) []byte {
	d := dr.d
	m := d.Meta
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
	fmt.Fprintf(&b, "<!doctype html>\n<html lang=\"%s\" data-si-theme=\"%s\"%s>\n<head>\n", html.EscapeString(lang), dr.themeName, modeAttr)
	b.WriteString(`<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover">
<meta name="color-scheme" content="light dark">
<meta name="generator" content="simple-ideas">
<link rel="icon" href="data:,">
`)
	fmt.Fprintf(&b, "<title>%s</title>\n", html.EscapeString(plain(d.Title())))
	b.WriteString(`<style id="si-css">` + "\n" + assets.LayerOrder + assets.LayoutCSS + dr.themeCSS + assets.ThemeLooksCSS)
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
	keys := map[string]string{}
	feats := map[string]bool{}
	var timer any
	if opt.Config != nil {
		if opt.Config.Keys != nil {
			keys = opt.Config.Keys
		}
		if opt.Config.Features != nil {
			feats = opt.Config.Features
		}
		t := opt.Config.Timer
		if len(t.Times) > 0 || t.Unit != "" || dr.animName != "" {
			timer = map[string]any{"time": t.Times, "unit": t.Unit, "animation": dr.animName, "animKind": dr.animKind}
		}
	}
	cfg, _ := json.Marshal(map[string]any{"keys": keys, "features": feats, "timer": timer})
	fmt.Fprintf(&b, "<script>window.SI_CONFIG=%s;</script>\n", cfg)
	if dr.animName != "" {
		if dr.animKind == "js" {
			b.WriteString("<script>window.__siAnim=(" + dr.animCode + "\n);</script>\n")
		} else {
			fmt.Fprintf(&b, "<template id=\"si-anim-svg\">%s</template>\n", dr.animCode)
		}
	}
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
<button type="button" data-si-action="overview" title="Overview (o)">Overview</button><button type="button" data-si-action="presenter" title="Presenter teleprompter (p)">Presenter</button><button type="button" data-si-action="draw" title="Annotate (a)">Draw</button>
<button type="button" data-si-action="mode" title="Light / dark (d)">Theme</button><button type="button" data-si-action="fullscreen" title="Full screen (f)">Full screen</button><button type="button" data-si-action="help" title="Keys (?)">?</button>
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
<tr><td>t</td><td>break timer: pick minutes, countdown to project during a break</td></tr>
<tr><td>d</td><td>light / dark</td></tr>
<tr><td>a</td><td>draw: pen over the slide (c colour, e erase, z undo, x clear, Esc off)</td></tr>
<tr><td>f</td><td>full screen</td></tr>
<tr><td>swipe</td><td>next / previous on touch screens</td></tr>
<tr><td>#5 or #5.2</td><td>open slide 5 (at step 2)</td></tr>
<tr><td>print</td><td>one slide per page, every step shown</td></tr>
</table></div></div>
<div class="si-hud" role="dialog" aria-label="Break timer" aria-hidden="true"><div>
<div class="si-hud-time">0:00</div>
<div class="si-hud-presets"><button type="button" data-si-min="5">5</button><button type="button" data-si-min="10">10</button><button type="button" data-si-min="15">15</button><button type="button" data-si-min="20">20</button><button type="button" data-si-min="30">30</button><button type="button" data-si-min="45">45</button><button type="button" data-si-min="60">60</button></div>
<p class="si-hud-hint">1-7 pick minutes · space pause · t / Esc close</p>
</div></div>
<svg class="si-draw" aria-hidden="true"></svg>
<div class="si-draw-bar" aria-label="Annotate">
<button type="button" class="si-swatch" data-si-pen="var(--si-accent)" title="Accent" style="--_c:var(--si-accent)"></button><button type="button" class="si-swatch" data-si-pen="var(--si-crit)" title="Red" style="--_c:var(--si-crit)"></button><button type="button" class="si-swatch" data-si-pen="var(--si-ok)" title="Green" style="--_c:var(--si-ok)"></button><button type="button" class="si-swatch" data-si-pen="var(--si-fg)" title="Ink" style="--_c:var(--si-fg)"></button>
<button type="button" data-si-draw="erase" title="Eraser (e)">Erase</button><button type="button" data-si-draw="undo" title="Undo (z)">Undo</button><button type="button" data-si-draw="clear" title="Clear all (x)">Clear</button><button type="button" data-si-draw="done" title="Done (Esc)">Done</button>
</div>
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
