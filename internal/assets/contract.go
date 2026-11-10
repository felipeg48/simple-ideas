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
	// documents (type: lab, article, guide); unset by default, the CSS falls back to the value shown
	{"--si-font-prose", "documents: font stack of the text of articles, guides and labs", "var(--si-font-body)"},
	{"--si-prose-size", "documents: size of the text", "1rem"},
	{"--si-prose-leading", "documents: line height of the text", "1.7"},
	{"--si-doc-measure", "documents: longest line of the text (articles use 88% of it); code and tables may use the whole column (use ch)", "78ch"},
	{"--si-lede-size", "documents: the lede (description:) under an article or guide page title", "1.25em"},
	{"--si-nav-bg", "guides: background of the guide navigation", "color-mix(in srgb, var(--si-surface-2) 55%, var(--si-bg))"},
	{"--si-nav-active", "guides: background of the current page in the navigation", "color-mix(in srgb, var(--si-accent) 12%, transparent)"},
	{"--si-step-badge", "labs and guides: background of the number badge of a `## Step N:` heading", "var(--si-accent)"},
	{"--si-code-hl", "code: background of a highlighted line (hl=)", "color-mix(in srgb, var(--si-accent) 12%, transparent)"},
	{"--si-tab-active", "code: underline of the selected code tab", "var(--si-accent)"},
	{"--si-diff-add", "diffs: background of an added line", "color-mix(in srgb, var(--si-ok) 14%, transparent)"},
	{"--si-diff-del", "diffs: background of a removed line", "color-mix(in srgb, var(--si-crit) 13%, transparent)"},
	// mind maps (type: mindmap, ```mindmap); data-mm-theme sets them all, auto reads the deck theme
	{"--si-mm-bg", "mind maps: background of the map", "var(--si-surface) (a page: var(--si-bg))"},
	{"--si-mm-fg", "mind maps: topic text", "var(--si-fg)"},
	{"--si-mm-muted", "mind maps: breadcrumbs, note lines, code comments", "var(--si-muted)"},
	{"--si-mm-surface", "mind maps: fill of a topic box (tinted with its branch colour)", "var(--si-bg)"},
	{"--si-mm-line", "mind maps: borders of buttons, code topics and deep topics", "var(--si-line)"},
	{"--si-mm-accent", "mind maps: the central topic, map › chips, focus, Play", "var(--si-accent)"},
	{"--si-mm-on-accent", "mind maps: text on --si-mm-accent", "var(--si-on-accent)"},
	{"--si-mm-1", "mind maps: colour of the 1st branch (and the 6th, 11th…)", "var(--si-accent)"},
	{"--si-mm-2", "mind maps: colour of the 2nd branch", "var(--si-accent-2)"},
	{"--si-mm-3", "mind maps: colour of the 3rd branch", "var(--si-ok)"},
	{"--si-mm-4", "mind maps: colour of the 4th branch", "var(--si-warn)"},
	{"--si-mm-5", "mind maps: colour of the 5th branch", "a mix of accent and crit"},
	{"--si-mm-c", "mind maps: the branch colour of one topic or line, set by the runtime from --si-mm-1…5", ""},
	{"--si-mm-font", "mind maps: font stack of topics", "var(--si-font-body)"},
	{"--si-mm-mono", "mind maps: font stack of code topics", "var(--si-font-mono)"},
	{"--si-mm-hand", "mind maps: font stack of sticky notes (load one with fonts:, e.g. google:Caveat)", "a handwriting stack"},
	{"--si-mm-grid", "mind maps: colour of the background grid (transparent: none)", "transparent"},
	{"--si-mm-sticky", "mind maps: sticky note paper", "#ffe27a"},
	{"--si-mm-sticky-fg", "mind maps: sticky note text", "#3d3000"},
	{"--si-mm-height", "mind maps: height of an embedded map (slides: --si-figure-max)", "clamp(24em, 62vh, 48em)"},
	// polls (<!-- poll: -->, si play --audience)
	{"--si-poll-bar", "polls: the bar of an option's share of the votes", "the accent at 30 %"},
	{"--si-poll-track", "polls: the row behind each option", "var(--si-surface-2)"},
	{"--si-qa-bg", "Q&A: the band of the question shown over the slide", "var(--si-surface)"},
	// QR codes (<!-- qr: -->): dark on light in every theme, since an inverted code reads badly
	{"--si-qr-fg", "QR codes: the modules", "#111"},
	{"--si-qr-bg", "QR codes: the background and quiet zone", "#fff"},
	{"--si-qr-size", "QR codes: the width (wins over size=small|medium|large)", "min(30cqi, 40cqb) on a slide, min(14rem, 60vw) in a document"},
	// code windows (a fence with window= or id=); theme=night|paper|dusk set them all, auto reads the deck theme
	{"--si-cw-bg", "code windows: the window and its code", "var(--si-surface)"},
	{"--si-cw-bar", "code windows: the title bar or terminal tabs", "var(--si-surface-2)"},
	{"--si-cw-rule", "code windows: the line under the title bar", "var(--si-line)"},
	{"--si-cw-fg", "code windows: code text", "var(--si-fg)"},
	{"--si-cw-muted", "code windows: title, line numbers, prompt", "var(--si-muted)"},
	{"--si-cw-k", "code windows: keywords, tags, the highlighted-line bar", "var(--si-accent)"},
	{"--si-cw-s", "code windows: strings", "var(--si-accent-2)"},
	{"--si-cw-c", "code windows: comments", "var(--si-muted)"},
	{"--si-cw-n", "code windows: numbers and variables", "var(--si-warn)"},
	{"--si-cw-f", "code windows: calls, flags, attribute names, keys", "the accent mixed with the text colour"},
	{"--si-cw-t", "code windows: types (PascalCase names)", "var(--si-ok)"},
	{"--si-cw-hl", "code windows: background of a highlighted line (hl=)", "accent at 12%"},
	{"--si-cw-backdrop", "code windows: the backdrop behind the window (backdrop=)", "transparent"},
	{"--si-cw-pad", "code windows: room between the backdrop's edge and the window", "0 (with a backdrop: clamp(1em, 6cqi, 3.5em))"},
	{"--si-cw-radius", "code windows: corner radius of the window and the backdrop", ".75em (windows: .5em)"},
	{"--si-cw-shadow", "code windows: the window's shadow", "a soft drop shadow"},
	{"--si-cw-size", "code windows: font size of the window, relative to the text around it", ".9em"},
	{"--si-cw-pin", "code windows: the numbered callout dots", "#ffd23f"},
	{"--si-cw-pin-fg", "code windows: the numbers on the dots", "#2a1f00"},
	{"--si-cw-note", "code windows: callout notes", "#fffdf5"},
	{"--si-cw-note-fg", "code windows: callout text", "#24221c"},
	{"--si-cw-wire", "code windows: the dashed wire from a dot to its note", "var(--si-fg) (on a backdrop: white)"},
}

// Hook is a stable selector user CSS may target.
type Hook struct {
	Selector string
	Desc     string
}

// Hooks lists every stable `.si-*` class and attribute hook. A test fails if
// the built-in CSS uses a `.si-*` class that is not listed here.
var Hooks = []Hook{
	{`:root[data-si-theme="studio"]`, "the deck theme from front matter (studio, paper, mono, crdb, acemq, acemq-pro)"},
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
	{`.si-tk-k, .si-tk-s, .si-tk-c, .si-tk-n, .si-tk-f, .si-tk-t, .si-tk-v, .si-tk-m, .si-tk-y`, "code tokens: keyword, string, comment, number, a call's name or a flag, a type (PascalCase), variable, command, key"},
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
	{`.si-doc-pager`, "lab: previous / next lab links at the end of the guide (si build of a folder with several labs)"},
	{`.si-doc[data-si-doc]`, "document: its type on the page body (lab, article, guide); a book page also has data-si-book (chapter, cover, contents, index, book)"},
	{`.si-lede`, "article, guide page: the description: line under the title"},
	{`.si-doc-search`, "documents: the Search button (/) in the header or guide bar"},
	{`.si-doc-find`, "documents: the search dialog (/): this page, then the other pages, their sections and a book's index terms"},
	{`.si-doc-find-box`, "documents: the search box"},
	{`.si-doc-find-input`, "documents: the search field"},
	{`.si-doc-find-list`, "documents: the search results"},
	{`.si-doc-find-t`, "documents: a result's title"},
	{`.si-doc-find-w`, "documents: where a result is (On this page, Page, the page title, Index)"},
	{`.si-doc-find-x`, "documents: a result's text snippet, the match in <mark>"},
	{`.si-doc-find-none`, "documents: the No matches line"},
	{`.si-doc-find-hint`, "documents: the key hint under the results"},
	{`.si-site-nav`, "site: the navigator (data-nav: topbar, topbar-centred, mega, sidebar)"},
	{`.si-site-in`, "site: the inner box of the navigator"},
	{`.si-site-logo`, "site: the logo or title link (logo: in _site.md)"},
	{`.si-site-menu`, "site: the menu (a details element: folded behind Menu on narrow screens)"},
	{`.si-site-links`, "site: the menu links"},
	{`.si-site-group`, "site: a menu section (an item without a link); data-current when it holds the page"},
	{`.si-site-group-t`, "site: the title of a menu section"},
	{`.si-site-drop`, "site: the dropdown (or mega panel) of a menu section"},
	{`.si-site-tools`, "site: search, light / dark and the call to action"},
	{`.si-site-cta`, "site: the call to action (cta: in _site.md)"},
	{`.si-site-version`, "site: the version badge beside the logo (version: in _site.md)"},
	{`.si-alert`, "plugin: an alert band (data-kind info, ok, warn, crit; data-until)"},
	{`.si-alert-x`, "plugin: the dismiss button of an alert"},
	{`.si-pill-row`, "plugin: the paragraph holding a badge"},
	{`.si-pill`, "plugin: a badge (data-kind)"},
	{`.si-accordion`, "plugin: an accordion"},
	{`.si-acc`, "plugin: one accordion item (a details element)"},
	{`.si-modal`, "plugin: a modal (a native dialog element)"},
	{`.si-modal-close`, "plugin: the close button of a modal"},
	{`.si-carousel`, "plugin: an image carousel"},
	{`.si-carousel-track`, "plugin: the sliding row of a carousel"},
	{`.si-carousel-ctl`, "plugin: the previous / next buttons of a carousel"},
	{`.si-carousel-dots`, "plugin: the position dots of a carousel"},
	{`.si-hud`, "plugin: the corner stack of event toasts"},
	{`.si-toast`, "plugin: an event toast"},
	{`.si-toast-when`, "plugin: the countdown of an event toast"},
	{`.si-toast-dot`, "plugin: the dot of an event toast"},
	{`.si-embed-card`, "embed: the card of an embedded document, guide, book or site (data-kind)"},
	{`.si-embed-kind`, "embed: the type label of a card"},
	{`.si-embed-desc`, "embed: the description on a card"},
	{`.si-embed-items`, "embed: the first pages or chapters on a guide or book card"},
	{`.si-embed-doc`, "embed: a document embedded in full, or one of its sections"},
	{`.si-embed-from`, "embed: the From … link under embedded text"},
	{`.si-pill-mark`, "badge on a slide: the coloured mark before the text"},
	{`.si-lockup`, "logo: the lockup (logo × partner name)"},
	{`.si-qr`, "qr: the figure (data-size: small, medium, large)"},
	{`.si-handout-head`, "handout (si build --handout): the printed header: title, author, date, the QR code"},
	{`.si-handout-qr`, "handout: the QR code to handout-url: and its address"},
	{`.si-qr-code`, "qr: the SVG code (modules in currentColor)"},
	{`.si-qr-cap`, "qr: the caption"},
	{`.si-poll`, "poll: the figure (data-poll: its id; data-multiple; data-hidden: results on the next click; data-voted on an audience phone)"},
	{`.si-cloud`, "word cloud: the figure (data-cloud: its id; data-hidden; data-live under si play --audience)"},
	{`.si-cloud-words`, "word cloud: the list of words"},
	{`.si-cloud-word`, "word cloud: one word (--_w 1-5: its weight; title: how many gave it)"},
	{`.si-cloud-live`, "word cloud: the count of answers, shown live"},
	{`.si-cloud-off`, "word cloud: the line shown without si play --audience"},
	{`.si-cloud-form`, "word cloud: an audience phone's answer box and Send button"},
	{`.si-poll-reveal`, "poll: the hidden build-step marker of a · hidden poll (is-visible once its results show)"},
	{`.si-poll-opts`, "poll: the list of options"},
	{`.si-poll-opt`, "poll: one option (data-i; --_share 0-100; aria-pressed when chosen)"},
	{`.si-poll-label`, "poll: the option's text"},
	{`.si-poll-n`, "poll: the option's vote count"},
	{`.si-poll-bar`, "poll: the bar behind the option"},
	{`.si-poll-total`, "poll: the total line"},
	{`.si-qa-show`, "Q&A: the question the presenter shows over the slide (si play --audience)"},
	{`.si-qa-text`, "Q&A: the question's text"},
	{`.si-qa-name`, "Q&A: who asked (when they gave a name)"},
	{`.si-qr-audience`, "qr: audience: the live audience code (si play --audience), hidden elsewhere"},
	{`.si-qr-slot`, "qr: audience: where si play puts the code"},
	{`.si-lk-img`, "logo: the logo image"},
	{`.si-lk-logo`, "logo: the theme's own logo (--si-brand-logo), when no image is given"},
	{`.si-lk-x`, "logo: the bar between logo and partner"},
	{`.si-lk-name`, "logo: the partner name"},
	{`.si-cards`, "cards: the grid (c2, c3, c4: columns)"},
	{`.si-cell`, "cards: one card (teal, orange, ok, crit, hi)"},
	{`.si-card-tag`, "cards: the tag above a card's title"},
	{`.si-cue`, "teleprompter: a [CLICK] cue in the Notes: script, where the speaker advances"},
	{`.si-voice`, "two presenters: one speaker's part of .si-notes / .si-points (data-speaker, data-minutes, data-step); labelled in handouts"},
	{`.si-doc-wm`, "documents: the watermark layer (watermark: front matter), fixed over the page and on every printed page"},
	{`.si-say`, "demo: a > Say: quote, what the presenter says to the room"},
	{`.si-say-label`, "demo: the Say label of a speech callout"},
	{`.si-demo-facts`, "demo: the session / slides / duration block under the title"},
	{`.si-reading-time`, "article: the reading time in the byline (230 words a minute)"},
	{`.si-step-num`, "lab, guide: the number badge of a `## Step N:` / `## N.` heading"},
	{`.si-doc-guidebar`, "guide: the top bar (guide title, version, light / dark)"},
	{`.si-doc-guide`, "guide: the guide title link in the top bar (to the first page)"},
	{`.si-doc-ver`, "guide: the version chip (version: in _nav.md)"},
	{`.si-doc-site`, "guide: the band holding _header.md, above the guide bar"},
	{`.si-doc-site-in`, "guide: the inner box of the _header.md / _footer.md bands"},
	{`.si-doc-sitefoot`, "guide: the band holding _footer.md, above the page footer"},
	{`.si-doc-nav`, "guide: the navigation from _nav.md (a sidebar; folds above the page on narrow screens)"},
	{`.si-nav-sec`, "guide: a section heading of the navigation (a _nav.md item without a link)"},
	{`.si-nav-mark`, "guide: the read / current marker before a page link; a read page's link has .is-read"},
	{`.si-nav-ext`, "guide: a navigation link to an outside URL"},
	{`.si-doc-progress`, "guide: the N of M read counter of the navigation"},
	{`.si-doc-crumb`, "guide: the breadcrumb (guide, section, page)"},
	{`.si-doc-edit`, "guide: the Edit this page link (edit-url: in _nav.md)"},
	{`.si-code-title`, "code: the title= of a code block (documents: in the header bar)"},
	{`.si-line`, "code: one line of a block with hl=, linenos or callouts; .si-hl when highlighted"},
	{`.si-hl`, "code: a highlighted line (hl=)"},
	{`.si-ln`, "code: a line number (linenos); drawn from data-n, never copied"},
	{`.si-co`, "code: a callout badge, in the code and in the list (data-co); .is-on while hovered"},
	{`.si-callouts`, "code: the <N> callout list under a code block"},
	{`.si-tabs`, "code: a <!-- tabs --> group"},
	{`.si-tabs-bar`, "code: the tab buttons of a group (role=tablist)"},
	{`.si-tab`, "code: one tab (data-label); hidden unless selected, all shown in print"},
	{`.si-diff-add`, "diffs: an added line"},
	{`.si-diff-del`, "diffs: a removed line"},
	{`.si-diff-ctx`, "diffs: an unchanged context line"},
	{`.si-diff-hunk`, "diffs: a hunk header (@@ … @@)"},
	{`.si-diff-meta`, "diffs: a file header line (diff --git, index, ---, +++)"},
	{`.si-src`, "code from git: the source link in the code block's header bar (documents)"},
	{`.si-embed-slides`, "documents: an embedded deck (<!-- slides: -->), an iframe and its caption"},
	{`.si-book-bar`, "book: Cover / Contents / Index / PDF links in the bar"},
	{`.si-book-pdf`, "book: the PDF link (si build --pdf)"},
	{`.si-book-chap`, "book: the Chapter N / Appendix A label above a chapter title"},
	{`.si-sec-num`, "book: a section number (3.2) in an h2"},
	{`.si-nav-num`, "book: a chapter or section number in the navigation"},
	{`.si-nav-secs`, "book: the sections of the chapter being read, in the navigation"},
	{`.si-xref`, "book: a cross-reference ([](#fig-x)) filled with its label"},
	{`.si-term`, "book: an index term shown in the text (( ... ))"},
	{`.si-ix`, "book: the anchor of an index term not shown in the text ((( ... )))"},
	{`.si-fig-num`, "book: Figure 3.1 / Table 3.1 in a caption"},
	{`.si-table-cap`, "documents: a table caption (a Table: paragraph after the table)"},
	{`.si-book-cover`, "book: the cover (index.html)"},
	{`.si-book-plate`, "book: the title plate of the cover"},
	{`.si-book-kind`, "book: Book · edition label on the cover"},
	{`.si-book-start`, "book: the links on the cover"},
	{`.si-book-pri`, "book: the main link on the cover (Contents)"},
	{`.si-book-colophon`, "book: the _cover.md text on the cover"},
	{`.si-book-contents`, "book: the contents list"},
	{`.si-book-part`, "book: a part (a _nav.md section) in the contents"},
	{`.si-book-ch`, "book: one chapter in the contents"},
	{`.si-book-num`, "book: the chapter number in the contents"},
	{`.si-book-ct`, "book: the chapter title link in the contents"},
	{`.si-book-time`, "book: the reading time in the contents"},
	{`.si-book-ixnote`, "book: the line above the index"},
	{`.si-book-az`, "book: the A to Z bar of the index"},
	{`.si-book-ix`, "book: the index columns"},
	{`.si-book-letter`, "book: one letter of the index"},
	{`.si-book-loc`, "book: the section links of an index term"},
	{`.si-book-sub`, "book: a sub-entry of an index term"},
	{`.si-book-chapter`, "book: one chapter in the one-file book (book.html); starts a printed page"},
	{`.si-book-part-page`, "book: the contents section of the one-file book"},
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
	{`.si-mindmap[data-layout][data-lines][data-nodes][data-mm-theme][data-full]`, "a mind map (page or ```mindmap fence) and its options; .si-mm-on once mindmap.js runs"},
	{`.si-mm-page, .si-mm-main, .si-mm-figure, .si-mm-list, .si-mm-on`, "a mind map page body and main, the figure of a fence, the nested-list fallback (hidden once the script runs)"},
	{`.si-mm-svg, .si-mm-edges, .si-mm-nodes, .si-mm-anim, .si-mm-dragging`, "the drawing, its groups of lines and topics, grow-in animation, panning or moving"},
	{`.si-mm-node, .si-mm-root, .si-mm-lv2, .si-mm-shape, .si-mm-hit, .si-mm-uline`, "a topic (the central one, deeper ones), its box, the hit area and line of an underline topic"},
	{`.si-mm-chip, .si-mm-chip-link, .si-mm-chip-href, .si-mm-chip-badge, .si-mm-chip-more`, "chips after a topic: map › (drill-down), a page or slide link, a badge, +N folded"},
	{`.si-mm-code, .si-mm-lang, .si-mm-pre, .si-mm-bar, .si-mm-wtitle, .si-mm-sticky`, "a code topic (its language label, highlighted text, and its title bar when it is a window=), a sticky note"},
	{`.si-mm-edge, .si-mm-tap, .si-mm-note, .si-mm-flow`, "a line (tapered, to a sticky note) and its flow overlay"},
	{`.si-mm-hide, .si-mm-step`, "a branch waiting for its build step (```mindmap steps), and the hidden step marks that drive it"},
	{`.si-mm-sel, .si-mm-drop-in, .si-mm-drop-before, .si-mm-drop-after, .si-mm-ghost, .si-mm-rename`, "the map editor (Studio): the selected topic, drop targets while dragging, the dragged copy, the rename box"},
	{`.si-mm-moving, .si-mm-pulse, .si-mm-dim, .si-mm-wm, .si-mm-wm-corner`, "a topic being moved, a search hit, a branch dimmed by Play, the watermark (across or in the corner)"},
	{`.si-mm-listview`, "a 3D map read as its list on a phone (the List tool)"},
	{`.si-mm-3d, .si-mm-is3d, .si-mm-card, .si-mm-x, .si-mm-msg`, "the 3D view (view: 3d; its canvas), a map shown in 3D, the card of a clicked 3D topic and its close button, a short message (no WebGL)"},
	{`.si-mm-crumbs, .si-mm-here, .si-mm-tools, .si-mm-find, .si-mm-tour`, "the breadcrumb of drilled maps, zoom/fit/tidy (+ Find, Play on a page), the / search box, the Play position"},
	{`.si-cw[data-window][data-cw-theme][data-backdrop][data-transition][data-cw-id][data-numbers][data-cw-nav]`, "a code window (a fence with window= or id=) and its options; .si-cw-on once codewin.js runs, .si-cw-listmode when its notes move under it"},
	{`.si-cw-stage, .si-cw-win, .si-cw-bar, .si-cw-title, .si-cw-lights, .si-cw-pad, .si-cw-ctl, .si-cw-tabs, .si-cw-tab, .si-cw-prompt`, "the backdrop, the window, its title bar (macOS lights, Windows controls) or terminal tabs and prompt"},
	{`.si-cw-body, .si-cw-code, .si-cw-gutter, .si-cw-l, .si-cw-hl`, "the code area, the lines (one .si-cw-l each, moved by the morph), line numbers, a highlighted line"},
	{`.si-cw-notes-layer, .si-cw-wires, .si-cw-dot, .si-cw-bubble, .si-cw-list`, "callouts: their layer, the dashed wires, the numbered dots, the notes, and the list used on narrow screens"},
	{`.si-cw-steps, .si-cw-name, .si-cw-nav, .si-cw-on, .si-cw-listmode`, "every step as a plain code block (without JavaScript), a step's name, ← → Play under a show in a document"},
}

// Theme holds the token values of one theme. Base values apply in both modes.
// Label and About describe it for `si theme list`; Extras names the
// web/themes/<extra>.css stylesheets emitted after the tokens, in order (nil
// means the theme's own web/themes/<name>.css, if any).
type Theme struct {
	Label  string
	About  string
	Extras []string
	Base   map[string]string
	Light  map[string]string
	Dark   map[string]string
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
	// The six below (v2.15.0) are checked by TestNewThemesContrast: body, muted
	// and accent text 4.5:1 on the background, and text on an accent button.
	"twilight": {
		Label: "Twilight", About: "deep indigo and lavender, violet and coral accents",
		Base: map[string]string{"--si-font-body": sansStack, "--si-font-heading": sansStack, "--si-font-mono": monoStack, "--si-heading-weight": "780"},
		Light: map[string]string{
			"--si-bg": "#f5f3fa", "--si-fg": "#1b1830", "--si-muted": "#5a5672", "--si-surface": "#ffffff",
			"--si-surface-2": "#ebe7f4", "--si-line": "#d9d3e8", "--si-accent": "#6236c9", "--si-accent-2": "#b43a0c",
			"--si-on-accent": "#ffffff", "--si-ok": "#157a45", "--si-warn": "#9a5800", "--si-crit": "#c02a2a",
			"--si-shadow": "0 .1em .3em rgb(27 24 48 / .06), 0 .6em 1.6em rgb(98 54 201 / .09)",
		},
		Dark: map[string]string{
			"--si-bg": "#12101f", "--si-fg": "#ece9f7", "--si-muted": "#a19cbc", "--si-surface": "#1a1730",
			"--si-surface-2": "#221e3b", "--si-line": "#332d52", "--si-accent": "#a98bff", "--si-accent-2": "#ff8a65",
			"--si-on-accent": "#12101f", "--si-ok": "#4ade80", "--si-warn": "#fbbf24", "--si-crit": "#ff7a85",
			"--si-shadow": "0 .6em 1.8em rgb(0 0 0 / .45)",
		},
	},
	"forest": {
		Label: "Forest", About: "moss and pine greens, amber accent, serif headings",
		Base: map[string]string{"--si-font-body": sansStack, "--si-font-heading": serifStack, "--si-font-mono": monoStack,
			"--si-heading-weight": "700", "--si-heading-tracking": "-0.01em"},
		Light: map[string]string{
			"--si-bg": "#f3f5ef", "--si-fg": "#17211a", "--si-muted": "#536257", "--si-surface": "#ffffff",
			"--si-surface-2": "#e6ece2", "--si-line": "#d0dacb", "--si-accent": "#2b6a3d", "--si-accent-2": "#8f5410",
			"--si-on-accent": "#ffffff", "--si-ok": "#2b6a3d", "--si-warn": "#8f5410", "--si-crit": "#b3261e",
			"--si-shadow": "0 .3em 1.2em rgb(23 33 26 / .08)",
		},
		Dark: map[string]string{
			"--si-bg": "#0e1510", "--si-fg": "#e3eee4", "--si-muted": "#93a597", "--si-surface": "#152019",
			"--si-surface-2": "#1b2920", "--si-line": "#283a2d", "--si-accent": "#7fd39a", "--si-accent-2": "#e8b46a",
			"--si-on-accent": "#0e1510", "--si-ok": "#7fd39a", "--si-warn": "#e8b46a", "--si-crit": "#ff8a80",
			"--si-shadow": "0 .6em 1.8em rgb(0 0 0 / .45)",
		},
	},
	"sunrise": {
		Label: "Sunrise", About: "warm cream and peach, magenta and orange accents",
		Base: map[string]string{"--si-font-body": sansStack, "--si-font-heading": sansStack, "--si-font-mono": monoStack, "--si-radius": ".8em"},
		Light: map[string]string{
			"--si-bg": "#fff8f1", "--si-fg": "#2a1a14", "--si-muted": "#6e5447", "--si-surface": "#ffffff",
			"--si-surface-2": "#fbeadc", "--si-line": "#f0d8c6", "--si-accent": "#b8175a", "--si-accent-2": "#a34a05",
			"--si-on-accent": "#ffffff", "--si-ok": "#1d7a43", "--si-warn": "#a34a05", "--si-crit": "#b42318",
			"--si-shadow": "0 .3em 1.2em rgb(180 90 40 / .10)",
		},
		Dark: map[string]string{
			"--si-bg": "#1d1210", "--si-fg": "#fbe9df", "--si-muted": "#c0a092", "--si-surface": "#271916",
			"--si-surface-2": "#30201b", "--si-line": "#46302a", "--si-accent": "#ff6fa3", "--si-accent-2": "#ffb066",
			"--si-on-accent": "#1d1210", "--si-ok": "#6ee7a0", "--si-warn": "#ffb066", "--si-crit": "#ff8a80",
			"--si-shadow": "0 .6em 1.8em rgb(0 0 0 / .45)",
		},
	},
	"slate": {
		Label: "Slate", About: "cool blue-grey, corporate blue, crisp corners",
		Base: map[string]string{"--si-font-body": sansStack, "--si-font-heading": sansStack, "--si-font-mono": monoStack,
			"--si-heading-weight": "700", "--si-heading-tracking": "-0.015em", "--si-radius": ".3em"},
		Light: map[string]string{
			"--si-bg": "#f7f8fa", "--si-fg": "#0f172a", "--si-muted": "#4f5d77", "--si-surface": "#ffffff",
			"--si-surface-2": "#edf0f5", "--si-line": "#d8dee8", "--si-accent": "#1d5bb8", "--si-accent-2": "#0e6f67",
			"--si-on-accent": "#ffffff", "--si-ok": "#15803d", "--si-warn": "#9a5b00", "--si-crit": "#c0262d",
			"--si-shadow": "0 .1em .25em rgb(15 23 42 / .07), 0 .5em 1.4em rgb(15 23 42 / .07)",
		},
		Dark: map[string]string{
			"--si-bg": "#0b1220", "--si-fg": "#e6ebf3", "--si-muted": "#93a1b8", "--si-surface": "#111a2b",
			"--si-surface-2": "#172236", "--si-line": "#24324a", "--si-accent": "#6ea8ff", "--si-accent-2": "#2dd4bf",
			"--si-on-accent": "#0b1220", "--si-ok": "#4ade80", "--si-warn": "#fbbf24", "--si-crit": "#f87171",
			"--si-shadow": "0 .6em 1.8em rgb(0 0 0 / .45)",
		},
	},
	"neon": {
		Label: "Neon", About: "night-terminal dark, cyan and pink, monospace headings",
		Base: map[string]string{"--si-font-body": sansStack, "--si-font-heading": monoStack, "--si-font-mono": monoStack,
			"--si-heading-weight": "700", "--si-heading-tracking": "0", "--si-radius": ".35em"},
		Light: map[string]string{
			"--si-bg": "#f4f7fb", "--si-fg": "#0b1020", "--si-muted": "#4b5770", "--si-surface": "#ffffff",
			"--si-surface-2": "#e6ecf5", "--si-line": "#cdd7e6", "--si-accent": "#0b6f95", "--si-accent-2": "#a3107f",
			"--si-on-accent": "#ffffff", "--si-ok": "#127a46", "--si-warn": "#975a00", "--si-crit": "#bf1d3b",
			"--si-shadow": "0 .3em 1.2em rgb(11 16 32 / .08)",
		},
		Dark: map[string]string{
			"--si-bg": "#07090f", "--si-fg": "#e8f1ff", "--si-muted": "#8b97b3", "--si-surface": "#0d121d",
			"--si-surface-2": "#121a29", "--si-line": "#1f2a40", "--si-accent": "#22d3ee", "--si-accent-2": "#f472d0",
			"--si-on-accent": "#07090f", "--si-ok": "#4ade80", "--si-warn": "#facc15", "--si-crit": "#fb7185",
			"--si-shadow": "0 0 1.4em rgb(34 211 238 / .12)",
		},
	},
	"newsprint": {
		Label: "Newsprint", About: "editorial: serif type, black ink on newsprint, red accent",
		Base: map[string]string{"--si-font-body": serifStack, "--si-font-heading": serifStack, "--si-font-mono": monoStack,
			"--si-heading-weight": "800", "--si-heading-tracking": "-0.015em", "--si-radius": ".15em",
			"--si-prose-size": "1.06rem", "--si-prose-leading": "1.7"},
		Light: map[string]string{
			"--si-bg": "#f7f5f0", "--si-fg": "#121212", "--si-muted": "#545049", "--si-surface": "#fffefb",
			"--si-surface-2": "#ece8df", "--si-line": "#d6d0c3", "--si-accent": "#b3001b", "--si-accent-2": "#1f3a68",
			"--si-on-accent": "#ffffff", "--si-ok": "#2f6b2f", "--si-warn": "#8a5a00", "--si-crit": "#b3001b",
			"--si-shadow": "none",
		},
		Dark: map[string]string{
			"--si-bg": "#141312", "--si-fg": "#ecebe7", "--si-muted": "#a7a39b", "--si-surface": "#1c1b19",
			"--si-surface-2": "#24221f", "--si-line": "#36332e", "--si-accent": "#ff5c6c", "--si-accent-2": "#8fb3ff",
			"--si-on-accent": "#141312", "--si-ok": "#86c27f", "--si-warn": "#e6b450", "--si-crit": "#ff5c6c",
			"--si-shadow": "none",
		},
	},
	"studio": {
		Label: "Studio", About: "the default: neutral ink on warm paper, blue accent",
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
		Label: "Paper", About: "serif type on cream, rust accent",
		Base: map[string]string{"--si-font-body": serifStack, "--si-font-heading": serifStack, "--si-font-mono": monoStack,
			"--si-heading-weight": "700", "--si-heading-tracking": "-0.01em", "--si-prose-size": "1.06rem", "--si-prose-leading": "1.75"},
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
		Label: "Mono", About: "monospace everything, green accent, flat",
		Base: map[string]string{"--si-font-body": monoStack, "--si-font-heading": monoStack, "--si-font-mono": monoStack,
			"--si-heading-weight": "700", "--si-heading-tracking": "0", "--si-radius": ".2em", "--si-scale": ".94",
			"--si-prose-size": ".94rem", "--si-prose-leading": "1.65", "--si-doc-measure": "84ch"},
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
		Label: "CockroachDB University", About: "cream / ink / purple, Geist type",
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
		Label: "AceMQ", About: "orange / teal / ink, Urbanist + Mona Sans, cards, tags, steps, timeline, highlighter",
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

// AceMQ Pro: the acemq tokens and extras plus the full layout catalogue of
// the AceMQ deck style guide (web/themes/acemq-pro.css).
func init() {
	pro := Themes["acemq"]
	pro.Label = "AceMQ Pro"
	pro.About = "AceMQ plus the full deck-system catalogue: options, fact rows, benchmarks, flows, zones, verdicts, cover floaters"
	pro.Extras = []string{"acemq", "acemq-pro"}
	Themes["acemq-pro"] = pro
}

// themeAliases map alternate spellings to a built-in theme name.
var themeAliases = map[string]string{"cockroachdb": "crdb", "cockroachlabs": "crdb", "acemqpro": "acemq-pro"}

// themeSepRE matches the runs of spaces and underscores a theme name may be
// written with ("AceMQ Pro", "acemq_pro").
var themeSepRE = regexp.MustCompile(`[\s_]+`)

// CanonicalTheme resolves a theme name as a user may type it ("AceMQ Pro",
// "acemq_pro", "cockroachlabs") to its built-in name: lower case, spaces and
// underscores become "-", then aliases apply. Names that are no built-in theme
// pass through unchanged.
func CanonicalTheme(name string) string {
	n := themeSepRE.ReplaceAllString(strings.ToLower(strings.TrimSpace(name)), "-")
	if a, ok := themeAliases[n]; ok {
		return a
	}
	if _, ok := Themes[n]; ok {
		return n
	}
	return name
}

// ThemeAliases returns the aliases of a built-in theme, sorted.
func ThemeAliases(name string) []string {
	var out []string
	for a, n := range themeAliases {
		if n == name {
			out = append(out, a)
		}
	}
	sort.Strings(out)
	return out
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
	extras := th.Extras
	if extras == nil {
		extras = []string{name}
	}
	for _, x := range extras {
		b.WriteString(themeExtra(x))
	}
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

// ThemeSample returns a theme's own gallery deck (web/themes/<name>.sample.md),
// written by `si --export themes` in place of the generic sample; "" if none.
func ThemeSample(name string) string {
	md, err := themesFS.ReadFile("web/themes/" + name + ".sample.md")
	if err != nil {
		return ""
	}
	return string(md)
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
