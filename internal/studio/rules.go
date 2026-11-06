package studio

import (
	"net/http"

	"github.com/felipeg48/simple-ideas/internal/assets"
	"github.com/felipeg48/simple-ideas/internal/deck"
)

// help is the one-line help the editor shows beside a completion: what the
// directive or front-matter key does, with its shape. RULES.md has the rest.
var help = map[string]string{
	// directives
	"include":    "splice another deck or section file, or a code file as a code block",
	"tabs":       "the code blocks that follow become tabs",
	"code":       "a file from git: a GitHub permalink, or repo= path= ref=",
	"diff":       "a commit or compare diff from git",
	"slides":     "documents: a deck's slides, live (path 2-4); a demo's front matter: its slides (6-9)",
	"alert":      "a dismissible band: text · info|ok|warn|crit · until=YYYY-MM-DD",
	"badge":      "a status pill: text · info|ok|warn|crit",
	"accordion":  "the ### questions that follow, one open at a time",
	"modal":      "a dialog (id) up to <!-- end -->, opened by a link to #id",
	"end":        "ends a modal",
	"carousel":   "the image paragraphs that follow slide one at a time (auto=6s)",
	"event":      "a countdown toast: YYYY-MM-DDTHH:MM text · Label → link",
	"embed":      "a document as a card, #section, full; a deck's slides; a guide folder",
	"layout":     "slide layout",
	"transition": "slide transition",
	"background": "slide background: a colour, gradient, image or video",
	"footer":     "this slide's footer (off hides it)",
	"labs":       "make this a lab slide (comma-separated lab links)",
	"class":      "extra classes on the slide",
	"eyebrow":    "a small label above the slide heading",
	"svg":        "an animated SVG canvas: path and state words",
	"step":       "reveal the next block as one build step",
	"steps":      "reveal the next list item by item",
	"column":     "start the next column",
	"image":      "size and place the next image (width=40% right …)",
	// front matter
	"title":             "the title (else the first # heading)",
	"type":              "deck, lab, article, guide, book, demo, site or page",
	"theme":             "a built-in theme, a .css file or an https:// URL",
	"fonts":             "web fonts: heading=Fraunces:600, body=Inter:400;700, mono=…",
	"description":       "the one-sentence lede and card text",
	"navigator":         "a site's menu: topbar, topbar-centred, mega or sidebar",
	"cta":               "a site's highlighted button: Label → page",
	"logo":              "a site's logo: text or an image path",
	"watermark":         "a faint mark on every slide or page: text or an image",
	"runbook":           "the JSON cheat-sheet in the si play terminal",
	"demos":             "demo scripts for the si play ▶ Demos panel (files or folders)",
	"mode":              "pin light or dark",
	"accent":            "the accent colour",
	"aspect":            "aspect for overview cards and print pages",
	"duration":          "a demo's planned length (minutes)",
	"session":           "a demo's session",
	"header":            "off leaves out _header.md",
	"footer-page":       "off leaves out _footer.md",
	"edit-url":          "base of the Edit this page link",
	"repo":              "the default repo for code: and diff:",
	"watermark-opacity": "watermark opacity (default 0.06)",
}

// values lists what a directive or key accepts, for completion after its colon.
var values = map[string][]string{
	"type":      deck.Types,
	"mode":      {"light", "dark"},
	"aspect":    deck.Aspects,
	"navigator": {"topbar", "topbar-centred", "mega", "sidebar"},
	"header":    {"off"},
	"footer":    {"off"},
	"fonts": {"heading=Fraunces:600", "body=Inter:400;700", "mono=JetBrains Mono", "body=bunny:Inter",
		"heading=fontshare:Satoshi:700", "mono=fontsource:Fira Code", "off"},
	"kinds": {"info", "ok", "warn", "crit"}, // alert and badge, after a ·
}

// paths are the directives and keys whose value is a workspace path.
var paths = []string{"include", "svg", "embed", "slides", "code", "runbook", "demos", "logo", "watermark", "theme"}

func (a *api) rules(w http.ResponseWriter, r *http.Request) {
	v := map[string][]string{"theme": assets.ThemeNames()}
	for k, l := range values {
		v[k] = l
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"directives":  deck.Directives,
		"layouts":     deck.Layouts,
		"transitions": deck.Transitions,
		"metaKeys":    deck.MetaKeys,
		"help":        help,
		"values":      v,
		"paths":       paths,
	})
}
