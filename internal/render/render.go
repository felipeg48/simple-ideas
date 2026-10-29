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
	// Theme overrides the deck's `theme:` front matter; empty keeps it.
	Theme string
	// Config is an optional si.conf: default theme, key remaps, feature toggles.
	Config *config.Config
	// Terminal enables the play terminal drawer in the page (SI_CONFIG.terminal).
	// Set only by si play on a loopback address; always false for build/export,
	// so a self-contained deck never loads the terminal runtime.
	Terminal bool
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
	d           *deck.Deck
	opt         Options
	md          goldmark.Markdown
	lastState   map[string][]string
	words       map[string]bool
	files       map[string]string
	svgN        int
	stageN      int
	cur         *slideCtx // the slide being rendered, for the stage fence's steps
	mermaid     bool
	themeName   string      // resolved theme name (or "custom" for a CSS-file theme)
	themeCSS    string      // the si.theme layer CSS for this deck
	anims       []animEntry // resolved break-timer animations (empty if none)
	problems    []deck.Problem
	warnings    []deck.Problem
	wmDone      bool   // watermark resolved once (deck-level)
	wmKind      string // "" | "text" | "img"
	wmText      string // watermark text
	wmImg       string // watermark image as a data URI (inlined once)
	runbookJSON string // window.SI_RUNBOOK payload (play terminal drawer); "" if no runbook
	doc         bool   // rendering a lab document (type: lab), not slides
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
	if d.IsLab() {
		return Document(d, opt)
	}
	dr := &deckRenderer{d: d, opt: opt, lastState: map[string][]string{}, words: map[string]bool{}, files: map[string]string{}}
	dr.md = dr.markdown()
	var slides strings.Builder
	res := Result{}
	dr.resolveTheme()
	dr.resolveAnim()
	dr.resolveRunbook()
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
	switch {
	case len(s.Labs) > 0:
		b.WriteString(dr.labSlide(s))
	case s.Kind == deck.KindTitle || s.Kind == deck.KindClosing:
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
	b.WriteString(dr.footer(s))
	if s.Notes != "" || s.HasMinutes {
		fmt.Fprintf(&b, `<aside class="si-notes" hidden>%s</aside>`+"\n", dr.postprocess(dr.toHTML(s.Notes, nil), s.Dir, sc))
	}
	if s.Points != "" {
		fmt.Fprintf(&b, `<aside class="si-points" hidden>%s</aside>`+"\n", dr.postprocess(dr.toHTML(s.Points, nil), s.Dir, sc))
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

// watermark returns the per-slide watermark element. The deck-level value is
// resolved once (text vs inlined image); every slide reuses it so the image is
// embedded a single time (as a --si-wm-img var set on .si-deck in page()).
func (dr *deckRenderer) watermark(sc *slideCtx) string {
	if !dr.wmDone {
		dr.wmDone = true
		if val := dr.d.Meta.Watermark; val != "" {
			if deck.IsImagePath(val) {
				if uri, ok := dr.media(filepath.Dir(dr.d.Path), val, sc); ok {
					dr.wmKind, dr.wmImg = "img", uri
				} else {
					dr.problem(sc, "missing file: %s", val)
				}
			} else {
				dr.wmKind, dr.wmText = "text", val
			}
		}
	}
	switch dr.wmKind {
	case "text":
		return `<div class="si-watermark si-wm-text" aria-hidden="true">` + html.EscapeString(dr.wmText) + "</div>\n"
	case "img":
		return `<div class="si-watermark si-wm-img" aria-hidden="true"></div>` + "\n"
	}
	return ""
}

// footer returns the per-slide footer element. A slide's own `footer:` wins over
// the deck default; `off` (or empty on both) hides it.
func (dr *deckRenderer) footer(s *deck.Slide) string {
	text := dr.d.Meta.Footer
	if s.Footer != "" {
		text = s.Footer
	}
	if text == "" || text == "off" {
		return ""
	}
	return fmt.Sprintf(`<div class="si-footer" data-pos="%s">%s</div>`+"\n", footerPos(dr.d.Meta.FooterPos), html.EscapeString(text))
}

// footerPos normalises a footer-position to its long form; empty is bottom-right.
func footerPos(p string) string {
	switch p {
	case "bl", "bottom-left":
		return "bottom-left"
	case "bc", "bottom-center":
		return "bottom-center"
	case "tr", "top-right":
		return "top-right"
	case "tl", "top-left":
		return "top-left"
	case "tc", "top-center":
		return "top-center"
	}
	return "bottom-right"
}

// labIconSVG is the beaker icon drawn on a lab slide.
const labIconSVG = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M9 3h6"/><path d="M10 3v6L4.5 18.2A2 2 0 0 0 6.2 21h11.6a2 2 0 0 0 1.7-2.8L14 9V3"/><path d="M7 14.5h10"/></svg>`

// labSlide renders a lab slide: a big centred beaker icon, the lab label as a
// heading (like the course title), and one button per lab URL.
func (dr *deckRenderer) labSlide(s *deck.Slide) string {
	label := dr.d.Meta.LabsLabel
	if label == "" {
		label = "Now, let's do a Lab!"
	}
	var b strings.Builder
	b.WriteString(`<div class="si-lab-icon" aria-hidden="true">` + labIconSVG + "</div>\n")
	fmt.Fprintf(&b, `<h1 class="si-heading si-lab-title">%s</h1>`+"\n", html.EscapeString(label))
	b.WriteString(`<div class="si-lab-links">` + "\n")
	for i, u := range s.Labs {
		name := u.Text
		if name == "" {
			if len(s.Labs) == 1 {
				name = "Open the Lab"
			} else {
				name = fmt.Sprintf("Lab %d", i+1)
			}
		}
		fmt.Fprintf(&b, `<a class="si-lab-link" href="%s" target="_blank" rel="noopener noreferrer">%s</a>`+"\n", html.EscapeString(u.URL), html.EscapeString(name))
	}
	b.WriteString("</div>\n")
	return b.String()
}

// copyrightLine builds the title-slide colophon from the year and copyright
// holder: "© 2026 CockroachLabs", "© 2026" or "© CockroachLabs". Empty when
// neither is set.
func copyrightLine(year, holder string) string {
	year, holder = strings.TrimSpace(year), strings.TrimSpace(holder)
	if year == "" && holder == "" {
		return ""
	}
	return strings.TrimSpace("© " + strings.TrimSpace(year+" "+holder))
}

var pageSizes = map[string]string{"16:9": "13.333in 7.5in", "16:10": "12in 7.5in", "4:3": "10in 7.5in", "1:1": "7.5in 7.5in", "9:16": "7.5in 13.333in"}

// resolveAnim resolves the configured break-timer animation(s) into dr.anims.
// timer.animation names one or more built-ins (waves/orbit/bokeh) or, with
// timer.animation.path set, <name>.js|.svg files in that directory (resolved
// relative to cwd, then the deck dir); a comma-separated list becomes a toggle
// in the HUD. A configured-but-missing animation is a deck problem; no
// configured animation leaves dr.anims empty (no layer).
func (dr *deckRenderer) resolveAnim() {
	if dr.opt.Config == nil {
		return
	}
	t := dr.opt.Config.Timer
	if t.Animation == "" {
		return
	}
	for _, raw := range strings.Split(t.Animation, ",") {
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}
		dr.resolveOneAnim(name, t.AnimationPath)
	}
}

func (dr *deckRenderer) resolveOneAnim(name, dir string) {
	if dir != "" {
		for _, k := range []string{"js", "svg"} {
			p := filepath.Join(dir, name+"."+k)
			if !filepath.IsAbs(p) {
				if _, err := os.Stat(p); err != nil {
					p = filepath.Join(filepath.Dir(dr.d.Path), dir, name+"."+k)
				}
			}
			if b, err := os.ReadFile(p); err == nil {
				dr.anims = append(dr.anims, animEntry{name, k, string(b)})
				return
			}
		}
	}
	if code, kind, ok := assets.Animation(name); ok {
		dr.anims = append(dr.anims, animEntry{name, kind, code})
		return
	}
	dr.problems = append(dr.problems, deck.Problem{File: dr.d.Path, Msg: fmt.Sprintf("timer.animation: no animation %q (built-ins: %s)", name, strings.Join(assets.AnimationNames(), ", "))})
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
	var b bytes.Buffer
	modeAttr := dr.head(&b)
	b.WriteString(`<style id="si-css">` + "\n" + assets.LayerOrder + assets.LayoutCSS + dr.themeCSS + assets.ThemeLooksCSS)
	b.WriteString(dr.helperCSS())
	fmt.Fprintf(&b, "@page{size:%s;margin:0}\n", pageSizes[aspect])
	b.WriteString(dr.accentCSS())
	b.WriteString("</style>\n")
	if dr.wmImg != "" || d.Meta.WatermarkOpacity != "" {
		var decls []string
		if dr.wmImg != "" {
			decls = append(decls, `--si-wm-img:url("`+dr.wmImg+`")`)
		}
		if d.Meta.WatermarkOpacity != "" {
			decls = append(decls, "--si-wm-opacity:"+d.Meta.WatermarkOpacity)
		}
		fmt.Fprintf(&b, "<style id=\"si-wm\">.si-deck{%s}</style>\n", strings.Join(decls, ";"))
	}
	dr.userAndMode(&b, modeAttr)
	b.WriteString("</head>\n<body>\n")
	ratio := strings.Replace(aspect, ":", "/", 1)
	fmt.Fprintf(&b, "<main class=\"si-deck\" aria-label=\"%s\" style=\"--_aspect:%s\" data-si-tx=\"none\">\n", html.EscapeString(plain(d.Title())), ratio)
	b.WriteString(slides)
	b.WriteString("</main>\n")
	b.WriteString(chromeHTML)
	dr.problemsBanner(&b, problems)
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
		if len(t.Times) > 0 || t.Unit != "" || t.Face != "" || len(dr.anims) > 0 {
			names := make([]string, 0, len(dr.anims))
			kinds := map[string]string{}
			for _, a := range dr.anims {
				names = append(names, a.name)
				kinds[a.name] = a.kind
			}
			timer = map[string]any{"time": t.Times, "unit": t.Unit, "face": t.Face, "animations": names, "animKinds": kinds}
		}
	}
	cfg, _ := json.Marshal(map[string]any{"keys": keys, "features": feats, "timer": timer, "terminal": opt.Terminal})
	fmt.Fprintf(&b, "<script>window.SI_CONFIG=%s;</script>\n", cfg)
	// Runbook cheat-sheet: play-only + loopback-only, like the terminal it rides
	// in. Never injected into a built deck (opt.Terminal is false for build).
	if opt.Terminal && dr.runbookJSON != "" {
		fmt.Fprintf(&b, "<script>window.SI_RUNBOOK=%s;</script>\n", strings.ReplaceAll(dr.runbookJSON, "</", `<\/`))
	}
	if len(dr.anims) > 0 {
		var js []string
		for _, a := range dr.anims {
			if a.kind == "js" {
				js = append(js, strconv.Quote(a.name)+":("+a.code+"\n)")
			} else {
				fmt.Fprintf(&b, "<template id=\"si-anim-svg-%s\">%s</template>\n", html.EscapeString(a.name), a.code)
			}
		}
		if len(js) > 0 {
			b.WriteString("<script>window.__siAnims={" + strings.Join(js, ",") + "};</script>\n")
		}
	}
	dr.mermaidScript(&b)
	presenter, _ := json.Marshal(assets.PresenterHTML)
	b.WriteString("<script>\nwindow.SI_PRESENTER = " + strings.ReplaceAll(string(presenter), "</", `<\/`) + ";\n")
	b.WriteString(assets.DeckJS)
	b.WriteString("</script>\n")
	dr.reload(&b)
	b.WriteString("</body>\n</html>\n")
	return b.Bytes()
}

// head writes the page start up to <title> (shared by decks and labs) and
// returns the data-si-mode attribute pinned by front matter, if any.
func (dr *deckRenderer) head(b *bytes.Buffer) string {
	m := dr.d.Meta
	lang := m.Lang
	if lang == "" {
		lang = "en"
	}
	modeAttr := ""
	if m.Mode == "light" || m.Mode == "dark" {
		modeAttr = ` data-si-mode="` + m.Mode + `"`
	}
	fmt.Fprintf(b, "<!doctype html>\n<html lang=\"%s\" data-si-theme=\"%s\"%s>\n<head>\n", html.EscapeString(lang), dr.themeName, modeAttr)
	b.WriteString(`<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover">
<meta name="color-scheme" content="light dark">
<meta name="generator" content="simple-ideas">
<link rel="icon" href="data:,">
`)
	fmt.Fprintf(b, "<title>%s</title>\n", html.EscapeString(plain(dr.d.Title())))
	return modeAttr
}

// accentCSS is the front matter accent override, or "".
func (dr *deckRenderer) accentCSS() string {
	if a := dr.d.Meta.Accent; a != "" {
		return fmt.Sprintf("@layer si.theme{:root,:root:not([data-si-mode]),:root[data-si-mode=\"dark\"]{--si-accent:%s}}\n", a)
	}
	return ""
}

// userAndMode writes the --css user layer and the script that restores the
// viewer's light/dark choice before first paint (a mode pinned in front
// matter wins).
func (dr *deckRenderer) userAndMode(b *bytes.Buffer, modeAttr string) {
	if css := UserCSS(dr.opt.UserCSS); css != "" {
		b.WriteString(`<style id="si-user">` + "\n" + css + "</style>\n")
	}
	if modeAttr == "" {
		b.WriteString(`<script>try{var m=localStorage.getItem('si-mode');if(m)document.documentElement.setAttribute('data-si-mode',m)}catch(e){}</script>` + "\n")
	}
}

// problemsBanner lists the problems on the page (si play).
func (dr *deckRenderer) problemsBanner(b *bytes.Buffer, problems []deck.Problem) {
	if !dr.opt.ShowProblems || len(problems) == 0 {
		return
	}
	fmt.Fprintf(b, "<aside class=\"si-problems\"><b>%d problem(s) in this deck</b><ul>", len(problems))
	for _, p := range problems {
		fmt.Fprintf(b, "<li>%s</li>", html.EscapeString(p.String()))
	}
	b.WriteString("</ul></aside>\n")
}

// mermaidScript loads Mermaid when the page has a mermaid fence.
func (dr *deckRenderer) mermaidScript(b *bytes.Buffer) {
	if !dr.mermaid {
		return
	}
	src := MermaidURL
	if dr.d.Meta.Mermaid != "" {
		src = dr.d.Meta.Mermaid
	}
	fmt.Fprintf(b, "<script src=\"%s\"></script>\n", html.EscapeString(src))
}

// reload adds the si play reload client (hot reload with --dev).
func (dr *deckRenderer) reload(b *bytes.Buffer) {
	if dr.opt.Dev {
		b.WriteString(DevReloadJS)
	} else if dr.opt.LiveReload {
		b.WriteString(LiveReloadJS)
	}
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
<tr><td>/</td><td>find a slide: search titles and text, or type a number to jump</td></tr>
<tr><td>d</td><td>light / dark</td></tr>
<tr><td>a</td><td>draw: pen over the slide (c colour, e erase, z undo, x clear, Esc off)</td></tr>
<tr><td>f</td><td>full screen</td></tr>
<tr><td>swipe</td><td>next / previous on touch screens</td></tr>
<tr><td>#5 or #5.2</td><td>open slide 5 (at step 2)</td></tr>
<tr><td>print</td><td>one slide per page, every step shown</td></tr>
</table></div></div>
<div class="si-hud" role="dialog" aria-label="Break timer" aria-hidden="true"><div>
<div class="si-hud-time">0:00</div>
<svg class="si-hud-dial" viewBox="0 0 100 100" aria-hidden="true" fill="none" stroke="currentColor" hidden>
<circle cx="50" cy="50" r="44" stroke-width="3" opacity="0.18"></circle>
<circle cx="50" cy="50" r="44" stroke-width="3" stroke-linecap="round" transform="rotate(-90 50 50)"></circle>
<line x1="50" y1="50" x2="50" y2="9" stroke-width="2" stroke-linecap="round"></line>
<circle cx="50" cy="50" r="2.6" fill="currentColor" stroke="none"></circle>
<text x="50" y="50" fill="currentColor" stroke="none" text-anchor="middle" dominant-baseline="central">0:00</text>
</svg>
<div class="si-hud-presets"><button type="button" data-si-min="5">5</button><button type="button" data-si-min="10">10</button><button type="button" data-si-min="15">15</button><button type="button" data-si-min="20">20</button><button type="button" data-si-min="30">30</button><button type="button" data-si-min="45">45</button><button type="button" data-si-min="60">60</button></div>
<div class="si-hud-toggle"></div>
<p class="si-hud-hint">1-7 pick minutes · space pause · t / Esc close</p>
</div></div>
<div class="si-find" role="dialog" aria-label="Find slide" aria-hidden="true"><div>
<input class="si-find-input" type="text" autocomplete="off" spellcheck="false" placeholder="Search slides, or type a number to jump…" aria-label="Search slides or go to a number">
<ul class="si-find-list" role="listbox"></ul>
</div></div>
<svg class="si-draw" aria-hidden="true"></svg>
<div class="si-draw-bar" aria-label="Annotate">
<button type="button" class="si-swatch" data-si-pen="var(--si-accent)" title="Accent" style="--_c:var(--si-accent)"></button><button type="button" class="si-swatch" data-si-pen="var(--si-crit)" title="Red" style="--_c:var(--si-crit)"></button><button type="button" class="si-swatch" data-si-pen="var(--si-ok)" title="Green" style="--_c:var(--si-ok)"></button><button type="button" class="si-swatch" data-si-pen="var(--si-fg)" title="Ink" style="--_c:var(--si-fg)"></button>
<span class="si-draw-sep" aria-hidden="true"></span><button type="button" class="si-brush" data-si-size="2" title="Thin" style="--_s:2px"></button><button type="button" class="si-brush" data-si-size="5" title="Medium" style="--_s:5px"></button><button type="button" class="si-brush" data-si-size="9" title="Thick" style="--_s:9px"></button>
<span class="si-draw-sep" aria-hidden="true"></span><button type="button" data-si-draw="erase" title="Eraser (e)">Erase</button><button type="button" data-si-draw="undo" title="Undo (z)">Undo</button><button type="button" data-si-draw="clear" title="Clear all (x)">Clear</button><button type="button" data-si-draw="done" title="Done (Esc)">Done</button>
</div>
`

// LiveReloadJS reloads the page when si play reports a change; the URL hash
// keeps the slide and step.
const LiveReloadJS = `<script>(function(){if(!window.EventSource)return;var es=new EventSource('/_si/events');es.addEventListener('reload',function(){location.reload()});})();</script>` + "\n"

// DevReloadJS is the si play --dev hot-reload client: on a change it fetches the
// re-rendered deck and morphs the slides in place (when the slide count is
// unchanged), so the current slide, step and overview are preserved with no
// full-page flash. It falls back to a full reload when the structure changes,
// and shows an overlay when the deck fails to render.
const DevReloadJS = `<script>(function(){
  if(!window.EventSource)return;
  var es=new EventSource('/_si/events');
  es.addEventListener('reload',hot);
  function overlay(msg){
    var o=document.getElementById('si-dev-error');
    if(!msg){if(o)o.remove();return;}
    if(!o){o=document.createElement('div');o.id='si-dev-error';document.body.appendChild(o);}
    o.textContent=msg;
  }
  var NAV=/\bsi-(current|past|future|sel)\b/g;
  function hot(){
    fetch(location.href,{cache:'no-store'}).then(function(r){
      return r.text().then(function(t){return {ok:r.ok,status:r.status,t:t};});
    }).then(function(res){
      if(!res.ok){overlay('si play '+res.status+'\n'+res.t.replace(/<[^>]*>/g,' ').slice(0,1500).trim());return;}
      overlay(null);
      var doc=new DOMParser().parseFromString(res.t,'text/html');
      var nd=doc.querySelector('.si-deck'),od=document.querySelector('.si-deck');
      if(!nd||!od){location.reload();return;}
      var ns=nd.querySelectorAll('.si-slide'),os=od.querySelectorAll('.si-slide');
      if(ns.length!==os.length){location.reload();return;}
      ['si-css','si-user','si-wm','si-data'].forEach(function(id){
        var a=document.getElementById(id),b=doc.getElementById(id);
        if(b&&a)a.textContent=b.textContent;else if(b&&!a)document.head.appendChild(b.cloneNode(true));else if(a&&!b)a.remove();
      });
      for(var i=0;i<os.length;i++){
        var a=os[i],b=ns[i];
        if(a.innerHTML!==b.innerHTML)a.innerHTML=b.innerHTML;
        var nav=(a.className.match(NAV)||[]).join(' ');
        a.className=b.className+(nav?' '+nav:'');
        ['data-title','data-layout','data-topic','data-transition','data-minutes','style'].forEach(function(at){
          var v=b.getAttribute(at);if(v==null)a.removeAttribute(at);else a.setAttribute(at,v);
        });
      }
      if(window.SI&&window.SI.refresh)window.SI.refresh();else location.reload();
    }).catch(function(e){overlay('si play: '+((e&&e.message)||'reload failed'));});
  }
})();</script>
<style>#si-dev-error{position:fixed;left:1rem;right:1rem;bottom:1rem;z-index:9999;max-height:60vh;overflow:auto;white-space:pre-wrap;background:#7f1d1d;color:#fff;padding:1rem 1.2rem;border-radius:.5rem;font:13px/1.45 ui-monospace,SFMono-Regular,Menlo,monospace;box-shadow:0 10px 40px rgba(0,0,0,.45)}</style>
` + "\n"

// IndexEntry is one deck on the index page.
type IndexEntry struct {
	Href     string
	Title    string
	Subtitle string
	Slides   int
	Minutes  float64
	PDF      string // optional relative href to a slides PDF (pdfs/<name>.pdf); empty = no badge
	Status   string // optional status badge (e.g. beta, preview, new)
	Version  string // optional version badge (e.g. v1.2.0)
	Lab      bool   // a `type: lab` document: shown as "N sections", no slides/time
	Sections int    // lab: number of h2 sections
}

// Index renders the deck index page.
func Index(title string, entries []IndexEntry, opt Options) []byte {
	theme := assets.CanonicalTheme(opt.Theme)
	if _, ok := assets.Themes[theme]; !ok {
		theme = "studio" // the index page's own theme; follows --theme / the decks, else studio
	}
	var b bytes.Buffer
	fmt.Fprintf(&b, "<!doctype html>\n<html lang=\"en\" data-si-theme=\"%s\">\n<head>\n<meta charset=\"utf-8\">\n", theme)
	b.WriteString(`<meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover">
<meta name="color-scheme" content="light dark">
<link rel="icon" href="data:,">
`)
	fmt.Fprintf(&b, "<title>%s</title>\n<style>\n%s*,*::before,*::after{box-sizing:border-box}\nbody{margin:0;background:var(--si-bg);color:var(--si-fg)}\n%s%s</style>\n",
		html.EscapeString(title), assets.LayerOrder, assets.ThemeCSS(theme), assets.ThemeLooksCSS)
	if css := UserCSS(opt.UserCSS); css != "" {
		b.WriteString("<style>\n" + css + "</style>\n")
	}
	b.WriteString(`<script>try{var m=localStorage.getItem('si-mode');if(m)document.documentElement.setAttribute('data-si-mode',m)}catch(e){}</script>` + "\n")
	b.WriteString("</head>\n<body>\n<main class=\"si-index\" data-view=\"tiles\">\n")
	fmt.Fprintf(&b, "<header class=\"si-index-head\"><div><h1>%s</h1><p>%d deck(s)</p></div>"+
		"<div class=\"si-view\" role=\"group\" aria-label=\"View\"><button type=\"button\" data-view=\"tiles\">Tiles</button><button type=\"button\" data-view=\"list\">List</button></div></header>\n",
		html.EscapeString(title), len(entries))

	badges := func(e IndexEntry) string {
		var s string
		if e.Lab {
			s += "<span class=\"si-badge\" data-status=\"lab\">LAB</span>"
		}
		if e.Status != "" {
			s += fmt.Sprintf("<span class=\"si-badge\" data-status=\"%s\">%s</span>", html.EscapeString(strings.ToLower(e.Status)), html.EscapeString(strings.ToUpper(e.Status)))
		}
		if e.Version != "" {
			s += fmt.Sprintf("<span class=\"si-badge si-badge-ver\">%s</span>", html.EscapeString(e.Version))
		}
		return s
	}

	b.WriteString("<div class=\"grid\">\n")
	for _, e := range entries {
		mins := ""
		if e.Minutes > 0 {
			mins = fmt.Sprintf(" · %s min", strconv.FormatFloat(e.Minutes, 'f', -1, 64))
		}
		bg := badges(e)
		if bg != "" {
			bg = "<div class=\"si-badges\">" + bg + "</div>"
		}
		size, kind := fmt.Sprintf("%d slides%s", e.Slides, mins), "deck"
		if e.Lab {
			size, kind = fmt.Sprintf("%d sections", e.Sections), "lab"
		}
		inner := fmt.Sprintf("<b>%s</b><span>%s</span>%s<small>%s</small>",
			html.EscapeString(e.Title), html.EscapeString(e.Subtitle), bg, size)
		if e.PDF == "" {
			fmt.Fprintf(&b, "<a class=\"si-card\" data-kind=\"%s\" href=\"%s\">%s</a>\n", kind, html.EscapeString(e.Href), inner)
		} else {
			// Two sibling links in a positioned wrapper: the card opens the deck, the
			// badge downloads the PDF. (Nesting <a> in <a> is invalid HTML.)
			fmt.Fprintf(&b, "<div class=\"si-card-wrap\"><a class=\"si-card\" data-kind=\"%s\" href=\"%s\">%s</a>"+
				"<a class=\"si-card-pdf\" href=\"%s\" download title=\"Download the slides as PDF\">PDF</a></div>\n",
				kind, html.EscapeString(e.Href), inner, html.EscapeString(e.PDF))
		}
	}
	b.WriteString("</div>\n")

	b.WriteString("<table class=\"si-list\"><thead><tr><th>Course</th><th>Status</th><th>Slides</th><th>Time</th><th>PDF</th></tr></thead><tbody>\n")
	for _, e := range entries {
		t := ""
		if e.Minutes > 0 {
			t = strconv.FormatFloat(e.Minutes, 'f', -1, 64) + " min"
		}
		pdf := "—"
		if e.PDF != "" {
			pdf = fmt.Sprintf("<a href=\"%s\" download>PDF</a>", html.EscapeString(e.PDF))
		}
		n := fmt.Sprint(e.Slides)
		if e.Lab {
			n, t = fmt.Sprintf("%d sections", e.Sections), ""
		}
		fmt.Fprintf(&b, "<tr><td><a href=\"%s\">%s</a></td><td>%s</td><td>%s</td><td>%s</td><td>%s</td></tr>\n",
			html.EscapeString(e.Href), html.EscapeString(e.Title), badges(e), n, t, pdf)
	}
	b.WriteString("</tbody></table>\n</main>\n")
	b.WriteString(`<script>(function(){var m=document.querySelector('.si-index');if(!m)return;var btns=m.querySelectorAll('.si-view button');function set(v){m.setAttribute('data-view',v);btns.forEach(function(x){x.setAttribute('aria-pressed',x.getAttribute('data-view')===v?'true':'false')});try{localStorage.setItem('si-index-view',v)}catch(e){}}var saved;try{saved=localStorage.getItem('si-index-view')}catch(e){}set(saved==='list'?'list':'tiles');btns.forEach(function(b){b.addEventListener('click',function(){set(b.getAttribute('data-view'))})})})();</script>` + "\n")
	if opt.LiveReload {
		b.WriteString(LiveReloadJS)
	}
	b.WriteString("</body>\n</html>\n")
	return b.Bytes()
}

// Entry builds an index entry for a parsed deck.
func Entry(d *deck.Deck) IndexEntry {
	e := IndexEntry{Href: d.Name + ".html", Title: plain(d.Title()), Slides: len(d.Slides), Minutes: d.Minutes(), Status: d.Meta.Status, Version: d.Meta.Version}
	if d.IsLab() {
		e.Lab, e.Sections, e.Slides, e.Minutes = true, labSections(d), 0, 0
	}
	first := d.Slides
	if !e.Lab && len(first) > 1 {
		first = first[:1] // a deck's subtitle comes from its title slide; a lab's from its first text line
	}
outer:
	for _, s := range first {
		for _, line := range strings.Split(s.Body, "\n") {
			if t := strings.TrimSpace(line); t != "" && !strings.HasPrefix(t, "<!--") && !(e.Lab && strings.HasPrefix(t, "#")) {
				e.Subtitle = plain(t)
				break outer
			}
		}
	}
	return e
}

// labSections counts a lab's h2 headings, ignoring fenced code.
func labSections(d *deck.Deck) int {
	n, fence := 0, false
	for _, s := range d.Slides {
		for _, line := range strings.Split(s.Body, "\n") {
			switch {
			case strings.HasPrefix(line, "```") || strings.HasPrefix(line, "~~~"):
				fence = !fence
			case !fence && strings.HasPrefix(line, "## "):
				n++
			}
		}
	}
	return n
}
