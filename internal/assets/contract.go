package assets

import (
	"fmt"
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
}

// Hook is a stable selector user CSS may target.
type Hook struct {
	Selector string
	Desc     string
}

// Hooks lists every stable `.si-*` class and attribute hook. A test fails if
// the built-in CSS uses a `.si-*` class that is not listed here.
var Hooks = []Hook{
	{`:root[data-si-theme="studio"]`, "the deck theme from front matter (studio, paper, mono)"},
	{`:root[data-si-mode="dark"]`, "explicit dark mode (t toggles); without the attribute prefers-color-scheme decides"},
	{`.si-deck`, "the full-viewport container of all slides"},
	{`.si-deck.si-overview`, "overview grid (o)"},
	{`.si-deck[data-si-tx="fade"]`, "transition running: fade, slide, zoom, none or canvas (same SVG on both slides)"},
	{`.si-slide`, "one slide; a size container, so cqi/cqb units refer to it"},
	{`.si-slide.si-current`, "the slide on screen"},
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
	{`.si-kicker`, "small label above a section heading (Part 2)"},
	{`.si-sub`, "subtitle block of title, section and closing slides"},
	{`.si-byline`, "author and date line of the title slide"},
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
	{`.si-notes`, "speaker notes (hidden in the deck, shown in presenter view)"},
	{`.si-bar`, "bottom bar: counter and controls"},
	{`.si-counter`, "slide counter (3 / 12)"},
	{`.si-controls`, "buttons of the bottom bar"},
	{`body.si-ui`, "set while the mouse moves: controls visible"},
	{`.si-progress`, "progress bar"},
	{`.si-help`, "keyboard help overlay (?)"},
	{`.si-problems`, "deck problems banner (si play only)"},
	{`.si-index`, "the deck index page"},
	{`.si-card`, "one deck card on the index page"},
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

// ThemeCSS returns the token declarations of a theme inside the si.theme
// layer: light values on :root, dark values under prefers-color-scheme (when
// no explicit mode is set) and under data-si-mode="dark".
func ThemeCSS(name string) string {
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
	b.WriteString("}\n")
	return b.String()
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
	b.WriteString("}\n\n/* dark mode: pressed t, or the system prefers dark and the deck has no mode pinned */\n")
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
