package assets

import (
	"embed"
	"encoding/base64"
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"
)

// Token is a CSS custom property of the styling contract.
type Token struct {
	Name string
	Desc string
	// Unset is the effective value of a per-element token that is unset by
	// default (set by a directive on one slide or image); empty for theme tokens.
	Unset string
}

// Tokens is the complete list of `--si-*` custom properties a theme or a user
// stylesheet may set. It is the source of truth for the generated theme CSS
// and for `si --export css-rules`; a test fails if the built-in CSS uses a
// property that is not listed here.
var Tokens = []Token{
	{"--si-bg", "slide and page background", ""},
	{"--si-fg", "body text", ""},
	{"--si-muted", "secondary text: subtitles, captions, counters", ""},
	{"--si-surface", "cards, code blocks, quotes, overview tiles", ""},
	{"--si-surface-2", "table headers, inline code, columns of the split layout", ""},
	{"--si-line", "borders and rules", ""},
	{"--si-accent", "accent: heading rule, bullets, links, progress bar", ""},
	{"--si-accent-2", "second accent: bold text, h3, code strings", ""},
	{"--si-on-accent", "text drawn on an accent fill", ""},
	{"--si-badge-fg", "index status/version badge text colour (defaults to white)", ""},
	{"--si-badge-bg", "index status/version badge background (CockroachDB-brand defaults; override per data-status)", ""},
	{"--si-ok", "status colour for good states (SVG canvases: data-ok)", ""},
	{"--si-warn", "status colour for warnings", ""},
	{"--si-crit", "status colour for failures (SVG canvases: data-alert)", ""},
	{"--si-font-body", "font stack of body text", ""},
	{"--si-font-heading", "font stack of headings", ""},
	{"--si-font-mono", "font stack of code", ""},
	{"--si-heading-weight", "font weight of headings", ""},
	{"--si-heading-case", "text-transform of headings (none, uppercase)", ""},
	{"--si-heading-tracking", "letter-spacing of headings", ""},
	{"--si-radius", "corner radius of cards, code, images (use em)", ""},
	{"--si-shadow", "shadow of cards and code blocks", ""},
	{"--si-scale", "multiplier of all slide text; the base size stays fluid", ""},
	{"--si-h1-size", "title and section heading size, relative to the fluid base (use em)", ""},
	{"--si-h2-size", "slide heading size (use em)", ""},
	{"--si-gap", "gap between columns and blocks (use em)", ""},
	{"--si-pad-x", "horizontal slide padding (use cqi: percent of the slide width)", ""},
	{"--si-pad-y", "vertical slide padding (use cqb: percent of the slide height)", ""},
	{"--si-measure", "maximum line length of paragraphs (use ch)", ""},
	{"--si-figure-max", "tallest a canvas, diagram or image may be under a heading (use cqb; center layout +6cqb, columns +14cqb); a slide that still overflows shrinks its figures first", ""},
	{"--si-transition-duration", "slide transition length", ""},
	{"--si-step-duration", "build step reveal length", ""},
	{"--si-canvas-duration", "SVG canvas state change length", ""},
	{"--si-slide-bg", "per-slide background colour or gradient, set by the background directive", ""},
	{"--si-bg-fit", "background image or video fit (cover, contain); set by background fit=", "cover"},
	{"--si-bg-position", "background object-position; set by background position=", "center"},
	{"--si-bg-opacity", "background image or video opacity; set by background opacity=", "1"},
	{"--si-bg-blur", "background blur; set by background blur= (tenths of a percent of the slide width)", "0px"},
	{"--si-img-width", "image width; set by the image directive width=", "fit-content"},
	{"--si-img-ratio", "image aspect-ratio box; set by the image directive ratio=", "auto"},
	{"--si-img-fit", "image object-fit inside a ratio box; set by fit=", "cover"},
	{"--si-img-position", "image object-position inside a ratio box; set by position=", "center"},
	{"--si-ov-cols", "overview: number of topic columns, set by the runtime when the grid opens", ""},
	{"--si-wm-opacity", "watermark opacity (watermark-opacity front matter)", "0.06"},
	{"--si-wm-img", "watermark image, inlined as a data URI (set from the watermark front matter)", ""},
	{"--si-brand-logo", "brand logo drawn beside the footer and in .lockup .lk-logo (acemq theme; none hides it)", "none"},
}

// Hook is a stable selector user CSS may target.
type Hook struct {
	Selector string
	Desc     string
}

// Hooks lists every stable `.si-*` class and attribute hook. A test fails if
// the built-in CSS uses a `.si-*` class that is not listed here.
var Hooks = []Hook{
	{`:root[data-si-theme="studio"]`, "the deck theme from front matter (studio, paper, mono, crdb, acemq)"},
	{`:root[data-si-mode="dark"]`, "explicit dark mode (d toggles); without the attribute prefers-color-scheme decides"},
	{`.si-deck`, "the full-viewport container of all slides"},
	{`.si-deck.si-overview`, "overview grid (o): slides laid out in one column per topic"},
	{`.si-ov-head`, "overview: a topic column header (the section title)"},
	{`.si-deck[data-si-tx="fade"]`, "transition running: fade, slide, zoom, none or canvas (same SVG on both slides)"},
	{`.si-slide`, "one slide; a size container, so cqi/cqb units refer to it"},
	{`.si-slide.si-current`, "the slide on screen"},
	{`.si-slide.si-sel`, "overview: the keyboard-selected tile (arrows move it, Enter opens it)"},
	{`.si-watermark`, "a faint watermark layer behind every slide's content (watermark front matter)"},
	{`.si-wm-text`, "a text watermark (watermark: some text)"},
	{`.si-wm-img`, "an image watermark (watermark: ./path.png), inlined once via --si-wm-img"},
	{`.si-footer`, "small footer text in a corner of every slide (footer / footer-position front matter)"},
	{`.si-lab`, "a lab slide (the labs directive): centred icon, heading, link buttons"},
	{`.si-lab-icon`, "the beaker icon on a lab slide"},
	{`.si-lab-title`, "the heading on a lab slide (labs-label)"},
	{`.si-lab-links`, "the row of lab buttons"},
	{`.si-lab-link`, "one lab button (links to a lab URL, opens in a new tab)"},
	{`.si-slide.si-past`, "slides before the current one"},
	{`.si-slide.si-future`, "slides after the current one"},
	{`.si-slide.si-title`, "the title slide (first # H1)"},
	{`.si-slide.si-section`, "a section slide (# H1 of an included module)"},
	{`.si-slide.si-closing`, "a closing slide (deck ends on a # H1)"},
	{`.si-slide.si-content`, "an ordinary ## H2 slide"},
	{`.si-slide[data-layout="center"]`, "layouts: default, center, two-col, split, quote, big-number"},
	{`.si-slide[data-transition="zoom"]`, "the slide's transition: fade, slide, zoom, none"},
	{`.si-slide.si-has-bg`, "a slide with a background (directive or front matter default)"},
	{`.si-slide.si-bg-dark`, "a slide on a dark background: tokens switch to a light-on-dark palette"},
	{`.si-slide.si-bg-light`, "a slide on a light background: tokens switch to a dark-on-light palette"},
	{`.si-bg`, "background media layer (image or video) under the slide content"},
	{`.si-bg-media`, "the background img or video (object-fit, opacity, blur)"},
	{`.si-bg-overlay[data-overlay="dark"]`, "overlay on a background: auto, dark, light"},
	{`.si-slide[data-layout="image"]`, "image layout: the first image fills the slide, words sit on top"},
	{`.si-image`, "a figure made from a paragraph holding one image"},
	{`.si-image[data-align="right"]`, "an image placed with the image directive: left, right, center"},
	{`.si-image[data-ratio]`, "an image in an aspect-ratio box (image directive ratio=)"},
	{`.si-caption`, "image caption, from the image title: ![alt](src \"caption\")"},
	{`.si-inner`, "padding box of a slide; holds the fluid font size"},
	{`.si-head`, "header of a content slide"},
	{`.si-heading`, "the slide heading (h1 or h2)"},
	{`.si-kicker`, "small label above a heading: the section \"Part 2\" kicker or a slide's `eyebrow:`"},
	{`.si-sub`, "subtitle block of title, section and closing slides"},
	{`.si-byline`, "author and date line of the title slide"},
	{`.si-colophon`, "copyright line on the title slide (© year copyright front matter)"},
	{`.si-body`, "content below the heading"},
	{`.si-body.si-cols`, "a body split into columns (two-col, split, or column directives)"},
	{`.si-col`, "one column of two-col and split layouts"},
	{`.si-col.si-flow`, "two-col without column breaks: one column flowing into two"},
	{`.si-step`, "a build step, hidden until reached"},
	{`.si-step.is-visible`, "a revealed build step"},
	{`.si-step.is-latest`, "the most recently revealed step"},
	{`.si-step.si-state`, "an invisible step that only changes canvas state"},
	{`.si-figure`, "the figure that holds an inlined SVG canvas"},
	{`.si-canvas`, "the inlined SVG root"},
	{`.si-canvas[data-state~="word"]`, "canvas in a state containing word"},
	{`.si-packets`, "the group of travelling dots added after a path with data-packets"},
	{`.si-packet`, "one travelling dot"},
	{`.si-code`, "a highlighted code block (pre)"},
	{`.si-tk-k, .si-tk-s, .si-tk-c, .si-tk-n, .si-tk-f, .si-tk-v, .si-tk-m, .si-tk-y`, "code tokens: keyword, string, comment, number, flag, variable, command, yaml key"},
	{`.si-mermaid`, "a Mermaid diagram"},
	{`.si-notes`, "verbatim teleprompter script (hidden in the deck, shown in presenter view)"},
	{`.si-points`, "speaker-notes talking points (hidden in the deck, shown in presenter view)"},
	{`.si-bar`, "bottom bar: counter and controls"},
	{`.si-counter`, "slide counter (3 / 12)"},
	{`.si-controls`, "buttons of the bottom bar"},
	{`body.si-ui`, "set while the mouse moves: controls visible"},
	{`.si-progress`, "progress bar"},
	{`.si-help`, "keyboard help overlay (?)"},
	{`.si-hud`, "break timer overlay (t): a full-screen countdown projected during a break"},
	{`.si-hud-time`, "the large mm:ss countdown digits; .is-over turns it critical at zero"},
	{`.si-hud-presets`, "the row of preset-minute buttons (1-7)"},
	{`.si-hud-dial`, "the analog sport-clock dial (timer.face: analog): depleting ring, sweeping hand, centre digits"},
	{`.si-hud-toggle`, "the HUD toggle row: pick the animation and the digital/analog face"},
	{`.si-hud-seg`, "a segmented toggle group inside .si-hud-toggle"},
	{`.si-hud-hint`, "the key hint under the countdown"},
	{`.si-anim`, "the break-timer animation layer behind the HUD (timer.animation)"},
	{`.si-doc`, "lab document (type: lab): the page body"},
	{`.si-doc-head`, "lab: the header band (title, author, date, version)"},
	{`.si-doc-head-in`, "lab: the centred inner box of the header"},
	{`.si-doc-title`, "lab: the title (h1)"},
	{`.si-doc-meta`, "lab: author, date and version line"},
	{`.si-doc-mode`, "lab: the light / dark button (d)"},
	{`.si-doc-layout`, "lab: the two-column grid of contents and guide; one column under ~900px"},
	{`.si-doc-toc`, "lab: the table of contents (h2 and h3), sticky beside the guide"},
	{`.si-toc-h2`, "lab: a table of contents entry for an h2; the current one has aria-current"},
	{`.si-toc-h3`, "lab: a table of contents entry for an h3 (indented)"},
	{`.si-doc-body`, "lab: the guide itself (readable measure)"},
	{`.si-doc-foot`, "lab: the footer (copyright and footer front matter)"},
	{`.si-anchor`, "lab: the # link after an h2 or h3"},
	{`.si-callout[data-kind]`, "lab: a callout from a GitHub alert: note, tip, important, warning, caution"},
	{`.si-callout-title`, "lab: the icon and title of a callout"},
	{`.si-codeblock`, "lab: a code block with its header bar"},
	{`.si-code-head`, "lab: the header bar of a code block"},
	{`.si-code-lang`, "lab: the language label of a code block"},
	{`.si-copy`, "lab: the copy button of a code block"},
	{`.si-table-wrap`, "lab: the box a table scrolls in"},
	{`.si-find`, "find-slide palette (/): jump by number or search titles + body text"},
	{`.si-find-input`, "the search box in the find palette"},
	{`.si-find-list`, "the results list in the find palette; the selected row is aria-selected"},
	{`.si-draw`, "annotation pen overlay (a): transient strokes drawn over the slide"},
	{`.si-drawing`, "on <body> while the pen is active: the overlay captures the pointer"},
	{`.si-draw-bar`, "the annotate toolbar: colour swatches, eraser, undo, clear, done"},
	{`.si-swatch`, "a colour swatch button in the annotate toolbar"},
	{`.si-brush`, "a brush-size button in the annotate toolbar (the dot shows the size)"},
	{`.si-draw-sep`, "a thin separator between groups in the annotate toolbar"},
	{`.si-problems`, "deck problems banner (si play only)"},
	{`.si-index`, "the deck index page"},
	{`.si-card`, "one deck card on the index page"},
	{`.si-card-wrap`, "wrapper around a deck card that also carries the PDF badge"},
	{`.si-card-pdf`, "the PDF download badge on a deck card (si build --pdf)"},
	{`.si-index-head`, "the index page header: title/count on the left, the view toggle on the right"},
	{`.si-view`, "the Tiles/List view toggle on the index page"},
	{`.si-badges`, "the row of status/version badges on a deck card"},
	{`.si-badge`, "a status or version badge (status: / version: front matter); data-status colours known values"},
	{`.si-badge-ver`, "the version-badge variant (outlined, not filled)"},
	{`.si-list`, "the index list/table view: Course, Status, Slides, Time, PDF columns"},
}

// Theme holds the token values of one theme. Base values apply in both modes.
type Theme struct {
	Base  map[string]string
	Light map[string]string
	Dark  map[string]string
}

const (
	sansStack  = `ui-sans-serif, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif`
	serifStack = `"Iowan Old Style", "Palatino Linotype", Palatino, "Book Antiqua", Georgia, serif`
	monoStack  = `ui-monospace, "SF Mono", "JetBrains Mono", Menlo, Consolas, "Liberation Mono", monospace`
	// CockroachDB University brand fonts: Geist / Geist Mono named first, then a
	// system fallback so decks stay self-contained (no bundled font file).
	crdbSans = `"Geist", ui-sans-serif, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif`
	crdbMono = `"Geist Mono", ui-monospace, "SF Mono", Menlo, Consolas, "Liberation Mono", monospace`
	// AceMQ brand fonts: embedded by web/themes/acemq.css (latin subsets).
	aceBody    = `"Mona Sans", "Helvetica Neue", "Segoe UI", system-ui, sans-serif`
	aceHeading = `"Urbanist", "Mona Sans", "Helvetica Neue", system-ui, sans-serif`
)

// layoutDefaults are mode-independent defaults shared by every theme.
var layoutDefaults = map[string]string{
	"--si-scale":               "1",
	"--si-h1-size":             "3.1em",
	"--si-h2-size":             "1.8em",
	"--si-gap":                 "1.4em",
	"--si-pad-x":               "max(1.1rem, 7cqi)",
	"--si-pad-y":               "max(1rem, 6cqb)",
	"--si-measure":             "72ch",
	"--si-figure-max":          "60cqb",
	"--si-transition-duration": ".55s",
	"--si-step-duration":       ".45s",
	"--si-canvas-duration":     ".7s",
	"--si-slide-bg":            "transparent",
	"--si-heading-weight":      "760",
	"--si-heading-case":        "none",
	"--si-heading-tracking":    "-0.02em",
	"--si-radius":              ".55em",
}

// Themes are the built-in themes. studio is the default.
var Themes = map[string]Theme{
	"studio": {
		Base: map[string]string{"--si-font-body": sansStack, "--si-font-heading": sansStack, "--si-font-mono": monoStack},
		Light: map[string]string{
			"--si-bg": "#f6f5f1", "--si-fg": "#16171b", "--si-muted": "#5d626c", "--si-surface": "#ffffff",
			"--si-surface-2": "#ecebe6", "--si-line": "#d9d6ce", "--si-accent": "#3657d6", "--si-accent-2": "#0b8a7a",
			"--si-on-accent": "#ffffff", "--si-ok": "#178a4c", "--si-warn": "#c77c02", "--si-crit": "#d23c3c",
			"--si-shadow": "0 .1em .3em rgb(20 20 30 / .06), 0 .6em 1.6em rgb(20 20 30 / .07)",
		},
		Dark: map[string]string{
			"--si-bg": "#0f1116", "--si-fg": "#eceef3", "--si-muted": "#99a0ad", "--si-surface": "#171a22",
			"--si-surface-2": "#1e222c", "--si-line": "#2c313c", "--si-accent": "#7d9bff", "--si-accent-2": "#45d0bf",
			"--si-on-accent": "#0f1116", "--si-ok": "#4ade80", "--si-warn": "#fbbf24", "--si-crit": "#f87171",
			"--si-shadow": "0 .6em 1.8em rgb(0 0 0 / .35)",
		},
	},
	"paper": {
		Base: map[string]string{"--si-font-body": serifStack, "--si-font-heading": serifStack, "--si-font-mono": monoStack,
			"--si-heading-weight": "700", "--si-heading-tracking": "-0.01em"},
		Light: map[string]string{
			"--si-bg": "#fbf7ee", "--si-fg": "#231f1a", "--si-muted": "#6f665a", "--si-surface": "#fffdf7",
			"--si-surface-2": "#f1eadb", "--si-line": "#e0d6c2", "--si-accent": "#b2431c", "--si-accent-2": "#2f6d62",
			"--si-on-accent": "#fffdf7", "--si-ok": "#3d7d3a", "--si-warn": "#b7791f", "--si-crit": "#b42318",
			"--si-shadow": "0 .3em 1.2em rgb(80 60 30 / .08)",
		},
		Dark: map[string]string{
			"--si-bg": "#1b1813", "--si-fg": "#efe8da", "--si-muted": "#a89d8b", "--si-surface": "#24201a",
			"--si-surface-2": "#2c271f", "--si-line": "#3b342a", "--si-accent": "#f08a5d", "--si-accent-2": "#7cc4b4",
			"--si-on-accent": "#1b1813", "--si-ok": "#86c27f", "--si-warn": "#e6b450", "--si-crit": "#f07167",
			"--si-shadow": "0 .6em 1.8em rgb(0 0 0 / .4)",
		},
	},
	"mono": {
		Base: map[string]string{"--si-font-body": monoStack, "--si-font-heading": monoStack, "--si-font-mono": monoStack,
			"--si-heading-weight": "700", "--si-heading-tracking": "0", "--si-radius": ".2em", "--si-scale": ".94"},
		Light: map[string]string{
			"--si-bg": "#f4f6f3", "--si-fg": "#111611", "--si-muted": "#5a6659", "--si-surface": "#ffffff",
			"--si-surface-2": "#e7ece6", "--si-line": "#cfd8cd", "--si-accent": "#18794e", "--si-accent-2": "#6a4cc2",
			"--si-on-accent": "#ffffff", "--si-ok": "#18794e", "--si-warn": "#a86b00", "--si-crit": "#c4302b",
			"--si-shadow": "none",
		},
		Dark: map[string]string{
			"--si-bg": "#0a0d0b", "--si-fg": "#d7f5dc", "--si-muted": "#7f9a84", "--si-surface": "#111612",
			"--si-surface-2": "#161d18", "--si-line": "#243027", "--si-accent": "#3ee07f", "--si-accent-2": "#b39dff",
			"--si-on-accent": "#0a0d0b", "--si-ok": "#3ee07f", "--si-warn": "#f5c451", "--si-crit": "#ff6b6b",
			"--si-shadow": "none",
		},
	},
	// CockroachDB University brand (aliases cockroachdb, cockroachlabs).
	// Light values are the brand's own cream/ink/purple; dark is derived.
	"crdb": {
		Base: map[string]string{"--si-font-body": crdbSans, "--si-font-heading": crdbSans, "--si-font-mono": crdbMono},
		Light: map[string]string{
			"--si-bg": "#f2f1ee", "--si-fg": "#101913", "--si-muted": "#6d6d6d", "--si-surface": "#ffffff",
			"--si-surface-2": "#ece9e2", "--si-line": "#e5e4dc", "--si-accent": "#9059ff", "--si-accent-2": "#502482",
			"--si-on-accent": "#ffffff", "--si-ok": "#43541f", "--si-warn": "#9a5b00", "--si-crit": "#b4232f",
			"--si-shadow": "0 .1em .3em rgb(16 25 19 / .06), 0 .6em 1.6em rgb(80 36 130 / .1)",
		},
		Dark: map[string]string{
			"--si-bg": "#101913", "--si-fg": "#f2f1ee", "--si-muted": "#9a9d94", "--si-surface": "#18231b",
			"--si-surface-2": "#1f2b22", "--si-line": "#2b3a2f", "--si-accent": "#a67cff", "--si-accent-2": "#c8acff",
			"--si-on-accent": "#101913", "--si-ok": "#8dbb3a", "--si-warn": "#e0a53a", "--si-crit": "#ef6b74",
			"--si-shadow": "0 .6em 1.8em rgb(0 0 0 / .45)",
		},
	},
	// AceMQ brand (orange / teal / ink). Its components, background rhythm,
	// footer logo, motion and fonts live in web/themes/acemq.css.
	"acemq": {
		Base: map[string]string{"--si-font-body": aceBody, "--si-font-heading": aceHeading,
			"--si-font-mono":      `ui-monospace, "SF Mono", Menlo, Consolas, monospace`,
			"--si-heading-weight": "800", "--si-heading-tracking": "-.032em", "--si-h2-size": "2.2em", "--si-gap": "1.1em",
			"--si-pad-x": "max(1.1rem, 5.7cqi)", "--si-pad-y": "max(1rem, 7.5cqb)", "--si-measure": "68ch",
			"--si-transition-duration": ".5s", "--si-step-duration": ".5s"},
		Light: map[string]string{
			"--si-bg": "#ffffff", "--si-fg": "#121316", "--si-muted": "#5f6670", "--si-surface": "#ffffff",
			"--si-surface-2": "#f6f6f7", "--si-line": "#e9ebee", "--si-accent": "#ff6600", "--si-accent-2": "#236b61",
			"--si-on-accent": "#ffffff", "--si-ok": "#236b61", "--si-warn": "#8a5a00", "--si-crit": "#c2363b",
			"--si-shadow": "0 .05em .1em rgb(18 19 22 / .04), 0 .4em 1.2em rgb(18 19 22 / .06), inset 0 1px 0 rgb(255 255 255 / .9)",
		},
		Dark: map[string]string{
			"--si-bg": "#121316", "--si-fg": "#f4f5f6", "--si-muted": "#b3b8c0", "--si-surface": "#1c1e23",
			"--si-surface-2": "#17191d", "--si-line": "rgb(255 255 255 / .12)", "--si-accent": "#ff6600", "--si-accent-2": "#8fd5cc",
			"--si-on-accent": "#ffffff", "--si-ok": "#8fd5cc", "--si-warn": "#f2b33d", "--si-crit": "#ff7b7f",
			"--si-shadow": "0 .4em 1.4em rgb(0 0 0 / .35), inset 0 1px 0 rgb(255 255 255 / .05)",
		},
	},
}

func decls(maps ...map[string]string) string {
	merged := map[string]string{}
	for _, m := range maps {
		for k, v := range m {
			merged[k] = v
		}
	}
	keys := make([]string, 0, len(merged))
	for k := range merged {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&b, "%s:%s;", k, merged[k])
	}
	return b.String()
}

// themeAliases map alternate spellings to a built-in theme name.
var themeAliases = map[string]string{"cockroachdb": "crdb", "cockroachlabs": "crdb"}

// CanonicalTheme resolves a theme alias (e.g. "cockroachlabs") to its built-in
// name; unknown names pass through unchanged.
func CanonicalTheme(name string) string {
	if a, ok := themeAliases[name]; ok {
		return a
	}
	return name
}

// ThemeNames returns the built-in theme names in a stable gallery order
// (studio first as the default, then the rest alphabetically).
func ThemeNames() []string {
	rest := make([]string, 0, len(Themes))
	for n := range Themes {
		if n != "studio" {
			rest = append(rest, n)
		}
	}
	sort.Strings(rest)
	return append([]string{"studio"}, rest...)
}

// ThemeCSS returns the token declarations of a theme inside the si.theme
// layer: light values on :root, dark values under prefers-color-scheme (when
// no explicit mode is set) and under data-si-mode="dark".
func ThemeCSS(name string) string {
	name = CanonicalTheme(name)
	th, ok := Themes[name]
	if !ok {
		th = Themes["studio"]
	}
	light := decls(layoutDefaults, th.Base, th.Light)
	dark := decls(th.Dark)
	var b strings.Builder
	b.WriteString("@layer si.theme{\n")
	fmt.Fprintf(&b, ":root{%scolor-scheme:light}\n", light)
	fmt.Fprintf(&b, "@media (prefers-color-scheme: dark){:root:not([data-si-mode]){%scolor-scheme:dark}}\n", dark)
	fmt.Fprintf(&b, ":root[data-si-mode=\"dark\"]{%scolor-scheme:dark}\n", dark)
	b.WriteString(themeExtra(name))
	b.WriteString("}\n")
	return b.String()
}

// themesFS holds per-theme extra stylesheets (web/themes/<name>.css) and their
// assets (web/themes/<name>/...).
//
//go:embed web/themes
var themesFS embed.FS

var themeURLRE = regexp.MustCompile(`url\("([a-z0-9-]+/[a-z0-9.-]+)"\)`)

var themeMIME = map[string]string{".woff2": "font/woff2", ".png": "image/png", ".svg": "image/svg+xml"}

// themeExtra returns a theme's extra stylesheet (empty for token-only themes)
// with its url("<name>/<file>") references inlined as data URIs, so only decks
// using that theme carry its fonts and images.
func themeExtra(name string) string {
	css, err := themesFS.ReadFile("web/themes/" + name + ".css")
	if err != nil {
		return ""
	}
	return themeURLRE.ReplaceAllStringFunc(string(css), func(u string) string {
		p := themeURLRE.FindStringSubmatch(u)[1]
		raw, err := themesFS.ReadFile("web/themes/" + p)
		if err != nil {
			return u
		}
		return `url("data:` + themeMIME[path.Ext(p)] + ";base64," + base64.StdEncoding.EncodeToString(raw) + `")`
	})
}

// StarterCSS is the commented stylesheet written by `si --export css-rules`.
func StarterCSS() string {
	th := Themes["studio"]
	all := map[string]string{}
	for _, m := range []map[string]string{layoutDefaults, th.Base, th.Light} {
		for k, v := range m {
			all[k] = v
		}
	}
	var b strings.Builder
	b.WriteString(`/* simple-ideas starter stylesheet
 *
 * Generated by: si --export css-rules
 * Use with:     si play --css my-theme.css decks/
 *               si --css my-theme.css --output html/ decks/
 *
 * Your CSS loads in the "si.user" cascade layer, after the built-in theme
 * ("si.theme"), so every rule here wins over the theme. Layout mechanics
 * (slide sizing, fluid type, container queries, transitions) are unlayered
 * and stay in charge: themes change tokens and looks, never the fluid layout.
 * Use relative units (em, %, cqi, cqb, ch); never fixed pixel sizes.
 * Relative url(...) paths are inlined as data URIs in static output.
 * Defaults below are the studio theme, light mode.
 */

/* ---------------------------------------------------------------- tokens */
:root {
`)
	for _, t := range Tokens {
		switch {
		case t.Unset != "":
			fmt.Fprintf(&b, "  /* %s; unset by default (acts as %s) */\n  /* %s: %s; */\n", t.Desc, t.Unset, t.Name, t.Unset)
		case all[t.Name] == "":
			fmt.Fprintf(&b, "  /* %s; unset by default */\n  /* %s: transparent; */\n", t.Desc, t.Name)
		default:
			fmt.Fprintf(&b, "  /* %s */\n  %s: %s;\n", t.Desc, t.Name, all[t.Name])
		}
	}
	b.WriteString("}\n\n/* dark mode: pressed d, or the system prefers dark and the deck has no mode pinned */\n")
	b.WriteString(":root[data-si-mode=\"dark\"] {\n")
	keys := make([]string, 0, len(th.Dark))
	for k := range th.Dark {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(&b, "  %s: %s;\n", k, th.Dark[k])
	}
	b.WriteString("}\n@media (prefers-color-scheme: dark) {\n  :root:not([data-si-mode]) {\n    /* repeat the dark values here */\n  }\n}\n")
	b.WriteString("\n/* ----------------------------------------------------------------- hooks */\n")
	for _, h := range Hooks {
		fmt.Fprintf(&b, "\n/* %s */\n/* %s { } */\n", h.Desc, h.Selector)
	}
	return b.String()
}
