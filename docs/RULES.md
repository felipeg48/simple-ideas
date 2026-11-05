# simple-ideas markdown rules

Everything `si` understands, with an example for each rule. `si rules` prints
this file; `examples/showcase.md` uses every rule in a real talk.

> **AsciiDoc too.** A deck can be `*.adoc` / `*.asciidoc` instead of `*.md`; si
> converts the AsciiDoc to this same format and everything below applies. See
> `docs/asciidoc.md` for the supported subset.

> **The layout rule.** Slides are fluid and responsive, never fixed. There is
> no 1280×720 canvas scaled by a transform: every slide fills the viewport and
> reflows, type follows the slide size (container units and `clamp()`), two
> columns stack on narrow or portrait screens, SVGs scale by their `viewBox`.
> A deck must read well from a ~400 px phone to a 4K display, in portrait and
> landscape. Image widths take `%`, `em` or `cqi`; `px` sizes are refused.

## Contents

1. [Decks and files](#1-decks-and-files)
2. [Front matter](#2-front-matter)
3. [Slides and headings](#3-slides-and-headings)
4. [Directives](#4-directives)
5. [Layouts and columns](#5-layouts-and-columns)
6. [Transitions](#6-transitions)
7. [Backgrounds](#7-backgrounds)
8. [Images](#8-images)
9. [Build steps](#9-build-steps)
10. [Animated SVG canvases](#10-animated-svg-canvases)
11. [Speaker notes and the teleprompter](#11-speaker-notes-and-the-teleprompter)
12. [Includes](#12-includes)
13. [Code, Mermaid, tables, links](#13-code-mermaid-tables-links)
14. [Presenting: keys, touch, overview, print](#14-presenting-keys-touch-overview-print)
15. [Styling: themes and your own CSS](#15-styling-themes-and-your-own-css)
16. [Configuring keys and features (si.conf)](#16-configuring-keys-and-features-siconf)
17. [What si check reports](#17-what-si-check-reports)
18. [Lab documents](#18-lab-documents)
19. [Articles](#19-articles)
20. [Guides (tutorials)](#20-guides-tutorials)
21. [Books (references)](#21-books-references)
22. [Demo scripts](#22-demo-scripts)

## 1. Decks and files

- Every `*.md` file in the directory you give `si` is a deck, except
  `README.md` and files whose name starts with `_` or `.` (use `_module.md`
  for a part that only exists to be included).
- One deck becomes one HTML file with the same name: `talk.md` → `talk.html`.
- Paths (images, SVGs, backgrounds, includes) are relative to the markdown
  file that mentions them.

## 2. Front matter

Optional deck settings: `key: value` lines between two `---` lines at the very
top of the file. Flat keys only; `#` starts a comment line.

```markdown
---
title: The Thursday Promise
author: Ada Lindqvist
date: Spring 2026
theme: studio
mode: auto
accent: "#3657d6"
transition: fade
aspect: 16:9
background: linear-gradient(160deg, #f6f5f1, #e9eefb)
mermaid: off
lang: en
---
```

| Key | Values | Default | Effect |
|---|---|---|---|
| `title` | text | the `# H1` | page title and index card |
| `author`, `date` | text | none | byline on the title slide |
| `year`, `copyright` | text | none | a copyright line on the title slide: `© {year} {copyright}` (either alone works) |
| `theme` | `studio`, `paper`, `mono` | `studio` | built-in look (sans, serif, monospace) |
| `mode` | `auto`, `light`, `dark` | `auto` | `auto` follows `prefers-color-scheme`; `t` toggles while presenting |
| `accent` | CSS colour | theme accent | sets `--si-accent` |
| `transition` | `fade`, `slide`, `zoom`, `none` | `fade` | default slide transition |
| `aspect` | `16:9`, `16:10`, `4:3`, `1:1`, `9:16` | `16:9` | shape of overview tiles, teleprompter previews and printed pages; slides on screen always fill the window |
| `background` | as the [background directive](#7-backgrounds) | none | background of every slide without its own |
| `mermaid` | `off` or a script URL | pinned CDN URL | how Mermaid fences render |
| `lang` | language tag | `en` | `<html lang>` |
| `status` | short text (`beta`, `preview`, `new`, `deprecated`, …) | none | a badge on the deck's index tile / list row; `beta`/`new`/`deprecated` are colour-coded |
| `version` | text (e.g. `v1.2.0`) | none | an outlined version badge on the index tile / list row |
| `footer` | text | none | small text in a corner of every slide; a slide's own `<!-- footer: … -->` overrides it, `<!-- footer: off -->` hides it (e.g. the title slide) |
| `footer-position` | `bottom-right` (default), `bottom-left`, `bottom-center`, `top-right`, `top-left`, `top-center` (short: `br`, `bl`, `bc`, `tr`, `tl`, `tc`) | `bottom-right` | which corner the footer sits in |
| `watermark` | text, or an image path | none | a faint layer behind every slide — literal text (e.g. `Draft`) is drawn large and angled; an image path is inlined once and centred. In every document type too (labs, articles, guides, books, demos): a fixed layer over the page, on every printed page; a guide or book sets it once in `_nav.md`. |
| `watermark-opacity` | number `0`–`1` | `0.06` | how visible the watermark is |
| `labs-label` | text | `Now, let's do a Lab!` | the heading on every lab slide (see the `labs` directive) |
| `type` | `deck`, `lab` | `deck` | `lab` renders the file as a readable guide page instead of slides; see [Lab documents](#18-lab-documents). Anything else is a `si check` problem |
| `runbook` | path to a `.json` file | none | a command cheat-sheet shown in the **`si play` terminal drawer only** (never in a built deck); see §14 |
| `demos` | markdown files or folders, comma-separated | none | demo scripts shown in the **`si play` terminal drawer only**, with Copy / Run on every shell block; see §14 |
| `type` values | `deck`, `lab`, `article`, `guide`, `book` | `deck` | `lab` §18, `article` §19, `guide` §20 (guide pages get it from `_nav.md`), `book` §21 (in `_nav.md` only) |
| `subtitle`, `edition` | text | none | books: the cover's subtitle and edition (§21) |
| `session`, `slides`, `duration` | text | none | demo scripts: the facts under the title; `duration: 14 min` is checked against the script's steps (§22) |
| `description` | text | none | documents: the lede under the title, the index tile subtitle and `<meta name="description">` (§19, §20) |
| `edit-url` | URL | none | guides: base of the "Edit this page" link (§20) |
| `header`, `footer-page` | `off` | on | a guide page without the guide's `_header.md` / `_footer.md` (§20) |
| `repo` | `OWNER/REPO`, a github.com URL, or a local clone folder | none | the default repo of `code:` and `diff:` (§13) |

An included module's own front matter is ignored; the including deck decides.

The generated `index.html` offers a **Tiles / List** toggle (top-right, remembered
per browser); list view is a table with Course, Status, Slides, Time and a PDF
column. `status`/`version` badges and the PDF column come from each deck's front
matter and `si build --pdf`.

## 3. Slides and headings

```markdown
# The Thursday Promise

How one order crosses five teams, fails twice, and still arrives on time.

---

## One promise, five hand-offs

- The shop promises **Thursday** at checkout
- Nobody owns the promise end to end

---

# Thank you

Questions are welcome.
```

- The first `# H1` is the **title slide**; the paragraphs under it are the
  subtitle.
- `---` on its own line separates slides (inside a code fence it stays code).
- `## H2` starts a **content slide** and is its heading. A slide without a
  heading is fine (a big number, a quote, a full-bleed image).
- A later `# H1` (usually the first line of an [included](#12-includes)
  module) is a **section slide** with a "Part N" kicker.
- A deck that ends on a `# H1` slide ends on a **closing slide**, centred like
  the title.
- Slide-level directives may come before or after the heading.

## 4. Directives

Directives are HTML comments on a line of their own:

```markdown
<!-- name: value -->
<!-- name -->
```

They stay invisible on GitHub and in other markdown viewers. A comment whose
first word is followed by anything other than `:` or `-->` is an ordinary
comment (`<!-- note to self -->`, `<!-- TODO -->`); a lowercase single word
that is not a directive (`<!-- wobble -->`) is an error.

| Directive | Scope | Section |
|---|---|---|
| `layout: center` | slide | [Layouts](#5-layouts-and-columns) |
| `column` | body | [Layouts](#5-layouts-and-columns) |
| `transition: zoom` | slide | [Transitions](#6-transitions) |
| `background: assets/city.jpg opacity=0.4` | slide | [Backgrounds](#7-backgrounds) |
| `footer: off` | slide | overrides the deck `footer` on this slide (`off` hides it, e.g. the title) |
| `labs: [Create a Cluster](https://…), [Scale out](https://…)` | slide | turns the slide into a lab slide: a centred beaker icon, the `labs-label` heading, and a button per lab (opens in a new tab). A bare URL works too and shows a default label. |
| `class: dense dark-code` | slide | adds classes to the slide for your CSS |
| `eyebrow: Order #4711 · the promise` | slide | a small context label above the slide heading (`.si-kicker`); it is not part of the slide title, so find, the overview and the teleprompter show the heading alone. On a section slide it replaces the "Part N" kicker. |
| `image: width=40% align=right` | next image | [Images](#8-images) |
| `steps`, `step`, `steps: a b`, `step: a` | next block | [Build steps](#9-build-steps) |
| `svg: assets/canvas.svg state words` | body | [SVG canvases](#10-animated-svg-canvases) |
| `include: other.md` | deck | [Includes](#12-includes) |

Body directives (`column`, `image`, `step`, `steps`, `svg`) work on the
top-level blocks of a slide, not inside lists or quotes.

## 5. Layouts and columns

`<!-- layout: NAME -->` picks a slide layout. Every layout is fluid.

| Layout | What it does |
|---|---|
| `default` | heading at the top, content below, vertically centred |
| `center` | everything centred; lists stay left-aligned inside a centred box |
| `two-col` | equal columns; stack on narrow or portrait screens |
| `split` | 2 : 3 columns, the second on a panel (text + code, text + picture) |
| `quote` | the blockquote large, a paragraph after it as the attribution |
| `big-number` | the first paragraph huge in the accent colour, the rest as caption |
| `image` | the first image fills the slide; heading and text sit on it |

Columns: `<!-- column -->` starts the next column (any number). Using it
without a layout implies `two-col`.

```markdown
<!-- layout: two-col -->
## What changed

### Before
- Each team watched its own queue

<!-- column -->

### After
- One tracking screen
```

Without `<!-- column -->`, `two-col` and `split` put media blocks (an SVG
canvas, an image on its own line, a code block, a Mermaid diagram, a table)
in the second column and everything else in the first; a `two-col` slide with
no media flows its text into two columns.

```markdown
<!-- layout: big-number -->

**98.7%**

of Thursday promises kept in the first month.
```

```markdown
<!-- layout: quote -->

> The customer never saw the declined card. She saw Thursday.

Route 7 courier log
```

## 6. Transitions

`<!-- transition: fade|slide|zoom|none -->` sets how a slide arrives; the
front matter `transition:` sets the default (`fade`).

When two consecutive slides show **the same SVG file** with `svg:`, the slide
transition is replaced by the canvas transition: the picture stays put, its
state animates, and the words around it fade in. See
[SVG canvases](#10-animated-svg-canvases).

With `prefers-reduced-motion: reduce` every transition and animation is off.

## 7. Backgrounds

```markdown
<!-- background: VALUE [fit=cover|contain] [position=center|left-top|...]
                        [opacity=0..1] [blur=N] [overlay=auto|dark|light|none]
                        [poster=image] -->
```

(Write it on one line; it is wrapped here for reading.) `VALUE` is one of:

| Value | Example |
|---|---|
| a colour | `<!-- background: #101820 -->` |
| a CSS gradient | `<!-- background: linear-gradient(135deg, #1d2b64, #0b8a7a) -->` |
| an image | `<!-- background: assets/dusk.jpg opacity=0.35 blur=4 -->` |
| a video (mp4, webm, mov, m4v, ogv) | `<!-- background: assets/loop.mp4 poster=assets/still.jpg -->` |
| `none` | turns off the deck default on one slide |

Options (images and videos; `overlay` also for colours):

| Option | Meaning |
|---|---|
| `fit=cover` (default) / `contain` | `object-fit` of the media |
| `position=center` | `object-position`; use `-` for spaces: `left-top`, `center-bottom` |
| `opacity=0.35` | fade the media into the page background |
| `blur=4` | blur radius in tenths of a percent of the slide width (scales with the slide) |
| `overlay=auto` (default for media) | `auto` measures the image and picks `dark` or `light`; `dark` darkens and switches text to light; `light` brightens and switches text to dark; `none` leaves it alone |
| `poster=assets/still.jpg` | video poster; shown with reduced motion |

Readable text is automatic: dark backgrounds (colours, gradients by their
first hex colour, media by measurement) get the `si-bg-dark` class, which
switches the slide tokens to a light-on-dark palette; light ones get
`si-bg-light`. Videos are muted, loop, play only on the current slide, and
stay on their poster or first frame with reduced motion.

A deck-wide default goes in front matter: `background: linear-gradient(...)`.

## 8. Images

A paragraph that holds only one image becomes a figure; the image **title**
becomes the caption:

```markdown
![Warehouse floor plan](assets/floor.svg "Six aisles, one packing table")
```

Images inside a sentence stay inline. Size and place an image with the
`image` directive on the line before it (the same "applies to the next block"
rule as steps):

```markdown
<!-- image: width=46% align=right -->
![Floor plan](assets/floor.svg "Two pickers meet in the middle")

- Text flows beside the image and moves below it on a phone
```

| Option | Values | Effect |
|---|---|---|
| `width` | `40%`, `24em`, `30cqi` (never `px`) | figure width |
| `align` | `left`, `right`, `center` | `left`/`right` float beside the text; stacks on narrow screens |
| `ratio` | `16:9`, `4:3`, `1:1` | crops the image into an aspect-ratio box |
| `fit` | `cover`, `contain` | how the image fills the ratio box |
| `position` | `center`, `left-bottom` | which part stays visible when cropped |

Side by side: put images in columns (`two-col` + `<!-- column -->`) or use
`split` (text left, image right). Full bleed: `<!-- layout: image -->`.

Every image, background, poster and SVG is embedded as a data URI, so one
deck is one self-contained file. Files above 2 MB draw a warning; build with
`--no-inline-media` to copy media next to the HTML instead. `si play` links
files from the deck directory directly. A missing file fails `si check`.

## 9. Build steps

Steps reveal content one press at a time. → ↓ Space PgDn go through the
steps of a slide before moving to the next slide; ← ↑ PgUp go back.

```markdown
<!-- steps -->
- revealed first
- then this
- then this

<!-- step -->
A paragraph (or any block) revealed as one step.
```

- `<!-- steps -->` before a list: each item is one step.
- `<!-- step -->` before any block: that block is one step.
- Steps can **drive an SVG canvas**: words after the colon change the state of
  the slide's canvas when the step is reached.
  - `<!-- step: paid -->` reveals the next block and adds `paid`.
  - `<!-- steps: placed paid -declined,retry . -->` gives each list item one
    token: item 1 adds `placed`, item 2 adds `paid`, item 3 removes `declined`
    and adds `retry`, item 4 changes nothing (`.`). Commas join several words
    in one token.
  - A `step:` with no block after it (the last line of the slide, or followed
    by another directive) is an invisible step that only changes the canvas.
- Going back undoes the state change. Overview and print show every step.
- The URL hash keeps the step: `#5.2` is slide 5 at step 2.

## 10. Animated SVG canvases

One SVG file, many slides: the picture stays, parts appear, fail and recover.

```markdown
## The journey on one screen

<!-- svg: assets/journey.svg placed -->

---

## Payment: declined, then paid

<!-- svg: assets/journey.svg +declined -->
<!-- steps: . -declined,retry,paid -->
- 21:04: the card fails a fraud check
- 21:05: an automatic retry goes through
```

**The directive.** `<!-- svg: PATH words... -->` inlines the SVG (so it can
use the deck's CSS tokens) and sets its state: the root `<svg>` gets
`class="si-canvas"` and `data-state="words"`.

**State words carry across slides.** Plain words replace the state. When all
words start with `+` or `-`, they change the state the previous slide with
the same file ended in (after its steps): `+declined` above means "placed,
plus declined". Steps change it further (see [Build steps](#9-build-steps)).

**The animation.** On two consecutive slides with the same file, the incoming
copy starts in the outgoing state and switches to its own, so every CSS
transition in the SVG runs: parts fade in, move, change colour. The slide
itself does not fade; only the words around the canvas do.

### Authoring contract

1. Give the root `<svg>` a `viewBox` (or `width` and `height`, from which `si`
   derives one) and a class of your own, for example `class="journey"`.
2. Colour with the deck tokens and a fallback, so the file also looks right on
   its own: `fill: var(--si-surface, #fff)`.
3. Show and hide parts with **helper attributes** (no CSS needed):

   | Attribute | When the state contains the word |
   |---|---|
   | `data-show="paid retry"` | visible (hidden otherwise); fades |
   | `data-hide="delivered"` | hidden; fades. `data-hide` wins over `data-show` |
   | `data-alert="declined"` | shapes inside get the `--si-crit` stroke and pulse; a `text` with it turns `--si-crit` |
   | `data-ok="paid"` | shapes inside get the `--si-ok` stroke; a `text` with it turns `--si-ok` |
   | `data-flow="shipped"` | the path gets marching dashes |
   | `data-packets="paid"` | dots travel along the path (`data-packets-count`, `data-packets-dur` seconds, `data-packets-r`) |
   | `data-grow="loaded"` | the element scales up from its base (0 → full height, grown from the bottom) — for bars |
   | `data-move="sent"` | the element travels to the end of its own `offset-path` (set the path in your SVG CSS) |
   | `data-count="s1:0 s2:4200"` | the element's text ticks to the number of the active state word (last match wins) |

   Each attribute takes several words (any of them matches).
4. For anything else write CSS **inside the SVG** with the state selector and
   a transition, and it animates between slides and steps:

   ```css
   .journey .token { transition: transform 1s ease; }
   .journey[data-state~="paid"] .token { transform: translateX(235px); }
   .journey[data-state~="packed"] .token { transform: translateX(470px); }
   ```

   Same specificity, later rule wins: list stages in order so the furthest
   one applies when several words are present.
5. No need to scope animations to the current slide; animations pause on
   hidden slides.
6. Never write a less-than sign inside the SVG `<style>` (not even in a
   comment): inline SVG styles are parsed as markup in HTML.
7. Ids are prefixed per copy (`c3-box`), and `url(#id)` / `href="#id"`
   references with them. `<image href>` files are embedded.
8. With reduced motion, transitions are instant and packets are hidden.

`si new` and `examples/assets/journey.svg` are a complete worked example. To
build one from scratch, run `si new-canvas <name>` for a working, commented
starter canvas plus a deck that drives it, and follow the step-by-step recipe in
`docs/authoring-canvases.md`.

## 11. Speaker notes and the teleprompter

```markdown
Notes: (1.5 min)
The token reaches payments and turns red: the bank declined the card.
Thirty seconds later one retry goes through, and the order is paid.

Points: What to cover, not a script
- Payments goes red: the retry kicks in after 30s
- Pause after the red tile; this is the moment to remember
```

- A line starting with `Notes:` starts the slide's **verbatim teleprompter
  script**: the words the speaker says to the audience, word for word, about
  what is on the slide. Write it as speech, never as instructions to the
  speaker: no "pause here", "mention X", "tell them", "press next", "this slide
  shows". Everything after it, to the end of the slide (or a `Points:` line),
  is the script (markdown).
- **`[CLICK]` cues (recommended on slides with build steps or animations).**
  Write `[CLICK]` in the script where the speaker advances: the next build
  step, a canvas or stage change, the next slide. It is the one mark in
  `Notes:` that is not read aloud: the teleprompter shows it as a ▶ CLICK
  badge, so the speaker knows when to press. Put one per build step, and at
  most one more for moving on to the next slide; `si check` warns when a slide
  has fewer cues than steps, or more than steps + 1.

  ```markdown
  Notes: (1 min)
  The token reaches payments and turns red. [CLICK] Thirty seconds later one
  retry goes through. [CLICK] And the payment tile turns green.
  ```
- A line starting with `Points:` starts the slide's **speaker-notes talking
  points**: the main points of the topic as short bullets, to glance at if
  you'd rather talk than read. Reminders for the speaker (pace, emphasis,
  likely questions, what to skip) belong here, not in `Notes:`. Both
  blocks are optional and may appear in either order; neither swallows the other.
- `(N min)` on the `Notes:` line is the slide's planned time; the deck total is
  the sum, shown in the index, the build summary and the teleprompter.
- Both are hidden on the slides.

**Teleprompter (`p` or `Shift+S`).** A second window, synced with the
audience window over `BroadcastChannel`; it works in `si play` and in the
static HTML (`file://`), and either window drives both. It has two tabs:
**Teleprompter** (the verbatim `Notes:` script, with auto-scroll) and
**Speaker Notes** (the `Points:` talking points).

| In the teleprompter | |
|---|---|
| Tabs | `1` Teleprompter (script), `2` Speaker Notes (points); choice remembered per browser |
| Notes | large type for reading at a distance; a guide line at reading height (teleprompter tab only) |
| Auto-scroll | `a` on/off, `+` / `-` speed 1 to 10 (lines per ten seconds, follows the text size) |
| Manual scroll | ↑ ↓, mouse wheel, trackpad; `r` back to the top (scrolls whichever tab is showing) |
| Text size | `[` / `]` (both tabs) |
| Mirror | `m` flips the script for teleprompter glass |
| Notes only | `x` hides the previews, the prompt fills the window |
| Previews | the current slide (with its step state) and the next step or slide |
| Clocks | time on this slide against its `(N min)` (amber when over, red at 1.5×), total against the deck plan, ahead/behind pace, wall clock |
| Rehearsal | the time you spent on each slide is recorded per run; next to the slide clock it shows `last M:SS` — your previous run's time for this slide, to pace against |
| Timer | `t` pause/resume, `0` reset (promotes this run to the `last` reference and starts fresh) |
| Navigation | → ← Space PgDn PgUp Home End, buttons |

Settings (tab, size, speed, mirror, notes only) are remembered per browser.

**On another device (`si play --remote`).** Put the teleprompter on an iPad
(or any tablet or phone) and project from the laptop. `--remote` opens a
second listener on your network, next to the loopback one, and prints one
keyed URL and QR code per role:

```bash
si play talk.md --remote                    # both roles
si play talk.md --remote=tp                 # teleprompter only
si play talk.md --remote --remote-addr 0.0.0.0:9000
```

| Role | On the device | Drives the deck |
|---|---|---|
| `tp` | the teleprompter above: script, tabs, auto-scroll, clocks, previews, mirror | Prev / Next, keys |
| `draw` | the current slide with the pen always on (Apple Pencil pressure works); Prev / Next in the pen bar | the strokes appear on the projected slide live; Prev / Next |

- The deck on the laptop stays on `127.0.0.1`, so the terminal drawer keeps
  working. The remote listener (default: every interface, the play port + 1)
  serves the decks, their files and the sync, never the terminal.
- Pages sync through the server (server-sent events and small POSTs), the
  deck stays the source of truth, and a device that connects late jumps to
  the current slide. Ink is shared both ways in slide coordinates and clears
  when the slide changes, like the local pen (`a`).
- The URL carries a random per-run key; the first visit stores it as a
  cookie, and anything without it gets 403. It is plain HTTP: use it on a
  network you trust.
- With a folder of decks the QR code opens a picker (deck × role).
- Play only: a built deck never contains any of it. See `docs/remote.md`.

## 12. Includes

```markdown
<!-- include: appendix.md -->
```

The file's slides are spliced in place; its `# H1` becomes a section slide
("Part N"). Its own front matter is ignored. Includes nest; a cycle
(`a.md` → `b.md` → `a.md`) is an error. Paths inside the module stay relative
to the module.

**Recommended layout for a multi-section deck** (and the standard output of a
migration): a thin main deck plus one file per section.

```
deck/
  my-deck.md             front matter, title, agenda, one include per section, closing
  sections/01-intro.md   one file per section — edit these
  sections/02-....md
  images/anim/*.svg       step-animated canvases
  images/diagrams/*.svg   static themed diagrams
```

`si build deck/` builds only `my-deck.md`: `si` skips subdirectories and
non-`.md` files, so the `sections/` fragments never render as separate decks.
A section lives in `sections/`, so its own paths are relative to itself — it
references shared assets as `../images/...`.

## 13. Code, Mermaid, stage diagrams, tables, links

- Fenced code with a language gets a small built-in highlighter (Go, Python,
  JavaScript/TypeScript, Java, Kotlin, Rust, SQL, YAML, TOML, JSON, Bash,
  CSS, HTML) and a language label; other languages are shown plain.
- ```` ```mermaid ```` fences render with Mermaid, themed from the deck tokens
  and re-rendered on light/dark switch. **This is the only network
  resource:** the pinned Mermaid script from jsDelivr is added only to decks
  that have a Mermaid fence. Offline, the diagram source shows as text. Set
  `mermaid: off` to always keep the source, or `mermaid: <url>` to load the
  script from elsewhere.
- ```` ```stage ```` fences generate a themed, animated flow diagram — a
  native, offline, themeable alternative to Mermaid for **simple flows**
  (chains, fan-out, merges). Boxes, arrows, labels and a travelling dot or
  marching dashes are emitted as self-contained SVG coloured with the deck
  tokens; no runtime script, and motion pauses under reduced motion. Use
  `mermaid` for dense graphs instead.

  ```stage
  dir: LR                              # LR (default) or TB
  Read --> Cache
  Cache -->|hit| Fast[Return fast]     # id[Label] renames; -->|x| labels an edge
  Cache -->|miss| Disk --> Fill --> Fast
  dot Read..Fast 2s                    # a dot loops along the route Read->Fast
  flow Read..Disk                      # marching dashes along a route
  ```

  Nodes appear in first-seen order; `dot`/`flow A..B [dur]` take the shortest
  route A→B. Labels do not wrap: a long one (`Q[Matching queue]`) is set
  smaller so it stays inside its box in every built-in theme, a short one
  keeps the full size. Add `step <node...>` lines to walk the flow on build steps — each
  `step` line is one click. An optional op leads the line:
  `highlight` (default, accent outline), `ok` (green), `alert` (red, pulsing),
  `show` (the node starts hidden and fades in), `hide` (fades out):

  ```stage
  dir: LR
  Submit --> Check --> Deploy --> Done
  step Submit
  step ok Check
  step alert Deploy
  step show Done
  ```

  A bad block (unknown node, no route, malformed `dir`, unknown `step` node) is
  a `si check` problem. The generator lays out chains, fan-out and merges with
  longest-path layering (no crossing minimisation); diagrams size like any
  figure, so give a wide one its own `##` slide. Full reference with many
  examples: `docs/stage-diagrams.md`.
- GitHub tables, strikethrough (`~~x~~`), autolinks and task lists work.
- Relative links to `.md` files point at the generated `.html`:
  `[appendix](appendix.md)` → `appendix.html`.
- Raw HTML is allowed.


### Code from files, highlighted lines, callouts and tabs

These work on slides and in documents (labs, articles, guides).

**Code from a file.** An include of anything that is not `.md` / `.adoc`
becomes a code block, highlighted by the file's extension:

```markdown
<!-- include: src/App.java -->                     the whole file
<!-- include: src/App.java lines=5-10 -->          lines 5 to 10 (also 5-, -10, 5-10,20-24)
<!-- include: src/App.java tag=retry -->           between tag::retry[] and end::retry[]
<!-- include: src/App.java tag=retry hl=3-6 title=Java lang=java -->
```

- **Prefer `tag=` to `lines=`**: a tag moves with the code, line numbers do
  not. Mark the region in a comment of the file's language
  (`// tag::retry[]` … `// end::retry[]`, `# tag::retry[]`); marker lines are
  never shown; `tag=a,b` joins several in file order.
- A `lines=` or `tag=` include shows the file's own line numbers. The
  indentation every shown line shares is removed (`indent=keep` keeps it). The
  title is the file name unless `title=` says otherwise.
- `si check` reports a missing file, a range past the end, a tag that is not
  found or not closed, and an unknown option.

**Fence attributes.** After the language of a code fence:

````markdown
```java title="Retry.java" hl="3,5-6" linenos start=12
````

`title=` shows in the code block's header bar (documents), `hl=` highlights
lines (counted from the first line shown), `linenos` shows line numbers, from
`start=` (default 1). Copy never copies the numbers.

**Callouts.** End a code line with `<N>`, bare or as its comment (`// <1>`,
`# <1>`, `-- <1>`, `<!-- 1 -->`), and write the explanations right after the
block, one `<N> text` line each:

````markdown
```java
var key = order.id();                     // <1>
sleep(Duration.ofSeconds(30));            // <2>
```
<1> The order id is the idempotency key.
<2> One retry only, thirty seconds later.
````

The markers become numbered badges (and leave the code, so Copy copies clean
code); hovering a badge or its line lights up both. `si check` reports a
marker without a line and a line without a marker.

**Tabs.** `<!-- tabs -->` turns the code blocks that follow it (with their
callouts, up to the first other block) into one group of tabs, labelled by
`title=` or the language:

````markdown
<!-- tabs -->
<!-- include: code/Retry.java tag=retry title=Java -->
<!-- include: code/retry.py tag=retry title=Python -->
```bash title="curl"
curl -X POST localhost:8082/orders/4711/pay
```
````

In a document, choosing a tab switches every group on the page that has a tab
with the same label, and the choice is remembered for the next page. On a
slide the group switches on its own. Print shows every tab, labelled. ← / →
move between tabs.

**Code and diffs from git.** Paste a GitHub permalink (on GitHub: the
file, then **Copy permalink**), or name the repo, path and commit:

```markdown
<!-- code: https://github.com/acme/app/blob/3f2a9c1/src/App.java#L5-L10 -->
<!-- code: https://github.com/acme/app/blob/3f2a9c1/src/App.java tag=retry hl=2 -->
<!-- code: path=src/App.java ref=3f2a9c1 tag=retry -->        repo: from the front matter
<!-- code: repo=../app path=src/App.java ref=HEAD -->         a local clone (git show)
<!-- diff: https://github.com/acme/app/commit/3f2a9c1 path=src/App.java -->
<!-- diff: https://github.com/acme/app/compare/v1.1...v1.2 path=src/ -->
<!-- diff: repo=../app ref=v1.1..v1.2 path=src/App.java -->
```

- `code:` takes every option of a code include (`lines=`, `tag=`, `hl=`,
  `title=`, `lang=`, `linenos`); a `#L5-L10` in the permalink is `lines=5-10`.
  The code block's header links to the source.
- `diff:` shows a commit's changes, or the changes from A to B, filtered to
  `path=` (a file, or a folder: everything under it): file headers, hunk
  headers, removed and added lines with their old and new line numbers. Copy
  copies the patch. A ```` ```diff ```` fence renders the same way.
- `repo:` in the front matter (or `_nav.md` for a whole guide) is the default
  repo: `OWNER/REPO`, a github.com URL, or a local folder holding a clone.
- Everything is **fetched when you build** (and on every `si play` / `si
  serve` request), cached on disk, and inlined: a reader never fetches it. A
  commit is cached for good; a branch or tag (`main`, `v1.2`) for an hour, and
  `si check` warns that it can move: with `lines=` the shown code would drift,
  so pin a commit.
- **Private repos**: set `GITHUB_TOKEN` (or `GH_TOKEN`), or log in with the
  GitHub CLI (`gh auth login`); si then uses the GitHub API. A local clone
  needs only `git`.
- `--offline` (with `si build`, `si check`, `si serve`) uses the cache only and
  reports anything that was never fetched; without it, a network failure falls
  back to a cached copy when there is one.

**Slides in a document.** A document (lab, article, guide page) can show a
deck, or some of its slides, playable in place:

```markdown
<!-- slides: ../decks/talk.md -->                     the whole deck
<!-- slides: ../decks/talk.md 23-24 -->               slides 23 to 24 (or one: 23)
<!-- slides: ../decks/talk.md#live-a-first-broker --> the slide with that heading id
```

The deck is rendered when the page is built, self-contained in a frame of the
deck's `aspect`: click it, then the arrow keys step through slides and build
steps; canvases animate. The caption names the deck. `si check` reports a
missing deck, a range past its end and an unknown heading id. On a slide this
directive is a problem (decks do not nest).

**AsciiDoc** maps onto the same features: `include::App.java[lines=5..10]`,
`include::App.java[tag=retry]` (`tags=a;b`), `.Retry.java` before a listing
(the title), `[source,java,highlight=3;5..6]` and `%linenums`; `<1>`
callouts are written the same way.

## 14. Presenting: keys, touch, overview, print

| Key | Action |
|---|---|
| → ↓ Space PgDn Enter | next step, then next slide |
| ← ↑ Shift+Space PgUp | previous step or slide |
| Home / End | first / last slide |
| `o` | overview of every slide: one column per section (`#` slide) or included module, headed by its title; a deck with neither gets a plain grid. ← → ↑ ↓ move, Enter or click opens, Esc closes |
| `p` or `Shift+S` | teleprompter window |
| `t` | break timer HUD: pick minutes, countdown to project during a break |
| `/` | find a slide: search titles and text, or type a number to jump |
| `a` | draw: a pen layer over the slide |
| `d` | light / dark (remembered) |
| `f` | full screen |
| `?` | key help |
| swipe left / right | next / previous on touch screens |

- **Break timer** (`t`): opens a full-screen HUD that floats over the deck.
  Number keys pick a preset and start the countdown; `Space` pauses and resumes;
  `t` or `Esc` closes it. At zero the digits turn critical and flash. While it is
  open the deck does not advance, so it is safe to leave projected during a
  break. It needs no network and works from `file://`. The presets default to
  `5,10,15,20,30,45,60` minutes; override them in `si.conf` with `timer.time` and
  `timer.unit` (minutes or seconds). The countdown shows as digits or, with
  `timer.face: analog`, an analog sport-clock dial — toggle it live in the HUD.
  A `timer.animation` runs a themed animation layer behind the translucent HUD
  (SLIDE │ ANIMATION │ TIMER) — built-ins `waves`, `orbit`, `bokeh`, or your own;
  list several (comma-separated) for a toggle (a **None** option is always
  offered). Once the countdown starts the HUD hides its presets and toggles so
  only the clock and the animation show — `Space` still pauses; any other key or
  a click brings the controls back. See §16 and `docs/timer-animations.md`.

- **Find a slide** (`/`): opens a search palette over the deck. Type to search
  slide titles and body text — results list the slide number, title and the
  matched snippet, ranked title-matches first; `↑`/`↓` move the selection and
  `Enter` (or a click) jumps. Type a **number** to go straight to that slide
  (e.g. `/12` → slide 12). `Esc` closes. Disable with `feature.find: off`.
  Below the slides it also lists the **other pages** of the folder, matched
  loosely (the letters in order, not necessarily together: `idx` or `inde`
  finds `index.html`, `lab3` finds `lab-03-quorum.html`): the index, the other
  decks and labs, their PDFs (`--pdf`), any other `.html` already in the
  output folder, and the `links.*` of `si.conf`. `Enter` opens the page.
  `si build` writes the list into each deck (names and titles only, so it works
  from `file://`); `si play` reads the folder on every request.

- **Draw** (`a`): a transient pen over the current slide, for circling or
  underlining as you talk. Drag to draw (mouse, touch, or pen — pen pressure
  varies the stroke); the toolbar picks the colour (accent, red, green, ink) and
  the brush size (thin / medium / thick); `c` cycles colour, `e` erases strokes,
  `z` undoes the last, `x` clears all, `Esc` turns it off. Strokes use the theme
  colours and clear when you change slides; they are not saved and do not print.

- The URL hash is the position (`#7`, `#7.2`); reloads keep it, so live
  reload in `si play` stays on your slide.
- A progress bar and a slide counter (with the step) sit at the bottom; the
  controls appear when the mouse moves.
- When a slide is too tall for the window its canvases, diagrams and images
  shrink first (down to about 60 %) so a figure plus bullets clears the
  footer, then the fluid type shrinks a little, then the slide scrolls;
  nothing is cut off. The figure cap itself is the `--si-figure-max` token
  (default `60cqb`; the `center` layout adds `6cqb`, columns `14cqb`), so a
  theme with a taller footer band can lower it for every slide:

  ```css
  :root { --si-figure-max: 52cqb; }
  ```
- **Print** (browser print, or print to PDF): one slide per page at the deck
  `aspect`, every step revealed, canvases in their final state. `si build --pdf`
  writes a `<name>.pdf` next to each deck using headless Chrome (needs Chrome or
  Chromium installed; without it, HTML is still written).
- **Open on a phone**: run `si play <dir> --addr 0.0.0.0:7070` to serve on your
  network; `si play` then prints a LAN URL and a QR code to scan. (The default
  address is loopback-only, so it prints the `--addr` hint instead.)

- **Play terminal** (`` ` ``, backtick): a drawer of real shells on your machine
  for live demos — play-only and loopback-only, never shipped in a built deck.
  Full reference: `docs/terminal.md`.

- **Runbook** (front matter `runbook: cockroach.json`): a command cheat-sheet in a
  right drawer of the play terminal. The JSON is a list of categories, each with
  entries (`label`, `description`, `command`, `target: sql|bash`):

  ```json
  { "categories": [ { "name": "Cluster Info", "entries": [
      { "label": "Show Nodes", "description": "live nodes",
        "command": "SELECT node_id, is_live FROM crdb_internal.gossip_nodes;", "target": "sql" } ] } ] }
  ```

  Each command is syntax-highlighted (by `target`) so you can read it at a glance;
  there's a filter box. Every entry has **Copy** (puts the command at the shell
  prompt without running it, and on the clipboard) and **Run** (the same, then
  Enter); clicking the entry itself is Copy, Shift-click Run. A command waits
  until the shell has drawn its prompt, and is sent as a paste where the shell
  supports it (zsh, bash 5.1+), so a multi-line command shows whole and runs only
  on Enter. The shape matches the cockroach-engineering-tool VS Code extension's
  `runbook.json`, so an exported runbook drops in unchanged. Like the terminal, it
  appears only in `si play` on a loopback address — a built deck never loads it, so
  rules #2/#7 still hold for shipped HTML. `si check` validates the path and JSON.

  A command may hold **placeholders**: `{{name}}` or `{{name:default}}` (name
  `[A-Za-z_][A-Za-z0-9_-]*`, default any text without `}`). The drawer asks for a
  value for each distinct name (first occurrence and its default win), then Copy
  or Run the command with every occurrence replaced; Esc cancels and types nothing.
  Placeholders are highlighted in the command.

- **Demos** (front matter `demos: ../demos/`): demo scripts, plain markdown, in a
  second side panel of the play terminal (`▶ Demos`, one panel open at a time).
  A drop-down lists the demos by their `# ` title (else the file name); the panel
  shows the whole script (headings, prose, tables, `>` quotes) and every
  ```` ```bash ```` / `sh` / `shell` / `zsh` / `console` / `sql` / `powershell`
  block gets **Copy** and **Run**, as in the runbook. Blocks in any other
  language (```` ```text ```` for expected output) stay plain. The value is a
  comma-separated list of `.md` files or folders, relative to the deck; a folder
  gives its `*.md` except `README.md`, `index.md` and `_`-prefixed files. Like
  the runbook: play and loopback only, never in a built deck; `si check`
  reports a missing path. The last demo picked is remembered.

  ```markdown
  ---
  title: Session 1
  demos: ../demos/
  ---
  ```

  ```json
  { "label": "Pods", "command": "kubectl -n {{ns:messaging}} get pods {{pod}}", "target": "bash" }
  ```

  Not placeholders, typed verbatim: `{{.Server.Version}}`, `{{json .}}`,
  `${HOME}`, `{print $1}`, `{a: .b}`, and the Go-template keywords `{{end}}`,
  `{{else}}`, `{{break}}`, `{{continue}}`, `{{nil}}`, `{{true}}`, `{{false}}`. The file may also carry a top-level `name`
  (the drawer title) and `description`.

### Studio (`si --web`)

`si --web [dir] [--addr 127.0.0.1:7070] [--no-open]` opens a browser editor for
the decks in `dir`: file tree, markdown editor, live preview and the `si check`
report. It listens on a loopback address only and is never part of a built deck.
See `docs/studio.md`.

```bash
si --web examples --no-open
```

## 15. Styling: themes and your own CSS

- `theme:` picks a built-in theme (`studio`, `paper`, `mono`, `crdb`,
  `acemq`, `acemq-pro`), each with a light and a dark palette. `si theme list`
  prints them with their display names and aliases. Names are matched loosely:
  case is ignored and spaces or underscores count as `-`, so `theme: AceMQ Pro`
  and `--theme "AceMQ Pro"` mean `acemq-pro`. An unknown `--theme` is an error. `crdb` is the CockroachDB
  University brand (cream / ink / purple; aliases `cockroachdb`,
  `cockroachlabs`). `acemq` is the AceMQ brand (orange / teal / ink, Urbanist
  and Mona Sans embedded) with extra components, below. `acemq-pro` (AceMQ
  Pro) is `acemq` plus the rest of the AceMQ deck style guide's layouts.
- `theme:` (or `--theme`) can instead be a path ending in `.css`: the file's
  contents become the `si.theme` layer, so you ship a whole palette from one
  file. Define the `--si-*` tokens (see `si --export css-rules`); resolved
  relative to the deck, then the current directory. A missing file is a `si
  check` problem. `examples/css/harbor.css` is a complete example theme.
- `theme:` can also be an `https://…` (or `http://…`) URL — for example a raw
  file from a GitHub repo. It is fetched **at build time** and cached for an
  hour under the temp dir, and its contents are inlined into the deck, so the
  built HTML stays self-contained (no runtime network dependency). An
  unreachable URL is a `si check` problem (a fresh-enough cache is reused if the
  network is down); builds that use a URL theme need network access the first
  time.
- `--theme <name|file.css>` on `si play` / `si build` overrides the deck's
  `theme:` front matter, so one set of decks can be built in several themes.
- `si --export themes [--output dir]` writes one sample deck per built-in theme;
  `si play dir` (or `si build dir`) then shows them side by side as a gallery.
- `--css <file.css|dir>` (repeatable) loads your stylesheets after the theme;
  `si play` watches them and reloads, static output inlines them (relative
  `url()` files become data URIs).
- Cascade layers: `@layer si.theme, si.user`. Your CSS lands in `si.user`, so
  it overrides the theme without specificity games. The layout mechanics
  (slide sizing, fluid type, container queries, transitions, steps) are
  unlayered and stay in charge: **themes change tokens and looks, never the
  fluid layout.** Use relative units.
- The contract is the `--si-*` custom properties and the `.si-*` hooks.
  `si --export css-rules --output my-theme.css` writes a commented starter
  file listing every token with its default and every hook with a one-line
  description. `examples/css/harbor.css` is a complete example theme.

```css
:root { --si-accent: #e4572e; --si-font-heading: Georgia, serif; }
:root[data-si-mode="dark"] { --si-bg: #0b1a28; --si-fg: #f1ece2; }
.si-step.is-latest { color: var(--si-accent-2); }
```

### acemq theme extras

`theme: acemq` adds more than tokens. Only decks using it carry the extras
(fonts and logo inlined, about 145 KB):

- **Background rhythm**, automatic: odd slides are plain with a faint grid and
  an orange glow at the top, even slides are flat `--si-surface-2`.
- **Eyebrow**: `<!-- eyebrow: Context -->` renders as an orange uppercase label
  with a dash. **Highlighter**: `<span class="hl">` in a heading draws an
  orange band that sweeps in. Use one per slide.
- **Footer**: the `footer:` text gets an orange dot, and the AceMQ logo is
  drawn at the bottom left of every slide that has a footer. Set
  `:root { --si-brand-logo: none; }` (in `--css`) to hide the logo, or
  `url(...)` to swap it.
- **Cover and closing**: the title slide is left-aligned. Optional blocks:
  `.lockup` (`<span class="lk-logo">` draws the brand logo, `<img>`,
  `.lk-x` divider, `.lk-name` wordmark; a slide with a `.lockup` drops the
  footer logo so the logo shows once, and keeps the footer text), `.pill`
  with `.pill-mark`,
  `.cover-sub`, `.lead`, `.contact` (a row of `.tag`s).
- **Slide classes** (`<!-- class: ... -->`): `on-ink` (a dark slide in light
  mode; use once or twice), `dense` (smaller table), `us-2` … `us-6` (tint
  table column N orange: "our" column), `us-row` (tint the first body row),
  `vs` (red marker on the second column's `###`).
- **Components** (raw HTML): `.cards` (`c2`, `c4`) of `.card` (`hi` the one
  recommended card, `orange`, `teal`/`ok`, `crit`) with `h4`, `p`, `.big`
  (`<em>` unit in orange) and `.tag` (`orange`, `teal`, `crit`); `.tl` of
  `.ev` incidents (`.when`, `.what`, `.impact`); `.steps` of `.st` (`.num`,
  `h4`, `p`); `.callout` (`warn`) under a table or grid; verdict marks
  `.y`, `.n`, `.p`, `.u` in table cells.
- Code blocks are a dark terminal in both modes; tables are cards; motion
  (rise, pop, highlighter) is off under reduced motion and in print.

```markdown
<!-- eyebrow: Two paths -->
<!-- class: us-3 -->
<!-- footer: Source: AceMQ docs -->
## Same workload, <span class="hl">half the nodes.</span>

| | Today | AceMQ |
|---|---|---|
| Nodes | 6 | <span class="y">3</span> |

<div class="cards c2">
<div class="card hi"><span class="tag">Recommended</span><h4>AceMQ</h4><p>Quorum by default.</p></div>
<div class="card crit"><span class="tag">Today</span><h4>Mirrored</h4><p>Split brain on a lost AZ.</p></div>
</div>
```

Lab documents (`type: lab`) under `acemq` get the fonts and palette.

### acemq-pro theme extras (AceMQ Pro)

`theme: acemq-pro` is everything in `acemq` (above) plus the layout catalogue
of the AceMQ deck style guide, as raw-HTML components. All of them reflow:
grids use auto-fit and stack on narrow or portrait slides, and each pops in
with a stagger on the active slide. `si --export themes` writes a full
AceMQ Pro sample deck that uses every one.

| Component | Markup | Use for |
|---|---|---|
| Fact grid | `.heard` of `.hc` (`.k` label, `.big` with `<em>` unit, `p`; `hi`) | "What we heard" |
| Incidents + verdict | `.inc` of `.ic` (`.when`, `h4`, `p`), then `<p class="verdict">` (`<b>` in orange) | outages with one root cause |
| Argument + visual | `.split2` with `.blist` (`li.ok`, `li.crit`) and `.topo` (a card for an SVG, `.cap`) or `.zones` | a claim next to a diagram |
| Zones | `.zones` of `.zg` (`.zl` label, `.zc` cells: `ok`, `us`, `hot`, `down`) | before / after topology |
| Options | `.opts` of `.opt` (`.ol` letter, `.tag`, `h4`, `p`, `dl` of `<div><dt><dd>`; `hi`) | options A–D, one recommended |
| HTML table | `table.tbl` with `.us` cells | a matrix with "our" column |
| Fact rows | `.facts` of `.fr` (`.fn` number with `<em>` unit, then a `div` with `h4`, `p`) | big number, then why |
| Feature grid | `.feat` of `.fc` (`.icon-chip`, `h4`, `p`) | product capabilities |
| Benchmark | `.bench` of `.brow` (`style="--w:84%"`; `us` row orange; `.bl`, `.bar`, `.bv`), then `.bks` of `.bk` (`.big`, `p`; `hi`) | measured results |
| Flow | `.flowline` of `.fs` (`h4`, `p`; `hot` pulses), then `.mnotes` (`p`, `p.warn`) | migration, process |
| Plans | `.poc` of `.pc2` (`.tag`, `h4`, `p`, `dl`; `hi`) with a `.term2` terminal (`.c` comment, `.k` prompt, `.ok` output) | trial vs PoC |
| Cover floaters | `.floaters` of three `.fl` (text, `<img>` or `<svg>`) on the title slide | tilted tiles that drift; hidden on narrow slides |
| Parts | `.icon-chip` (`--sm`, `--teal`, `--ink`; holds an inline `<svg>` or a glyph), `.btn`, `.btn--ghost` | icons, a call to action |
| Motion | `.r` rise, `.f` fade, `.pop` pop, `.pulse` ring ×3, `.grow-x` draw in; `style="--i:N"` staggers | hand-placed raw HTML |

`.pop` is the guide's `.p`: in si `.p` stays the "partial" verdict mark.
There is no icon font (a deck stays self-contained), so icon chips take an
inline SVG.

```markdown
<!-- eyebrow: Options -->
## Four ways forward, <span class="hl">one we recommend.</span>

<div class="opts">
<div class="opt"><span class="ol">A</span><span class="tag">Today</span><h4>Keep as is</h4><p>Hand-run clusters.</p></div>
<div class="opt hi"><span class="ol">B</span><span class="tag">Recommended</span><h4>AceMQ Broker</h4><p>Quorum by default.</p><dl><div><dt>Lost zone</dt><dd>Alert</dd></div></dl></div>
</div>
<p class="verdict">Both keep your clients. <b>Only B removes the outage.</b></p>
```

### Themes for documents (labs, articles, guides)

Every theme styles documents too, from the same tokens as slides. Documents
read for minutes, so they add a few tokens of their own. Each is **unset by
default** and the built-in CSS falls back to the value below, so a theme (or a
`.css` theme file) that never sets them still works; set only what you want
to change.

| Token | Fallback | What it sets |
|---|---|---|
| `--si-font-prose` | `var(--si-font-body)` | font of the text (articles, guides, labs) |
| `--si-prose-size` | `1rem` | size of the text (paper `1.06rem`, mono `.94rem`) |
| `--si-prose-leading` | `1.7` | line height of the text |
| `--si-doc-measure` | `78ch` | longest line; articles use 88% of it; code and tables may use the whole column |
| `--si-lede-size` | `1.25em` | the `description:` lede under an article or guide page title |
| `--si-nav-bg` | `surface-2` mixed into `bg` | background of a guide's navigation |
| `--si-nav-active` | `accent` at 12% | the current page in the navigation |
| `--si-step-badge` | `var(--si-accent)` | the number badge of a `## Step N:` heading (labs, guides) |

A custom theme for documents is an ordinary theme file: set the colour tokens
(light and dark) and, if you like, the document tokens. Hook classes for
finer control are listed by `si --export css-rules` (`.si-doc[data-si-doc]`,
`.si-lede`, `.si-doc-nav`, `.si-doc-crumb`, `.si-step-num`, `.si-doc-site`,
...).

```css
/* harbor-docs.css: use with  theme: harbor-docs.css  in _nav.md or a page */
:root {
  --si-bg: #f7f4ee; --si-fg: #1d2430; --si-muted: #5b6574; --si-surface: #ffffff;
  --si-surface-2: #ece7dd; --si-line: #dcd4c6; --si-accent: #0b6e8a; --si-accent-2: #a2462b;
  --si-on-accent: #ffffff; --si-ok: #2f7d4a; --si-warn: #b7791f; --si-crit: #b42318;
  --si-font-body: "Iowan Old Style", Georgia, serif;
  --si-font-heading: ui-sans-serif, system-ui, sans-serif;
  --si-font-mono: ui-monospace, Menlo, monospace;
  --si-heading-weight: 700; --si-heading-tracking: -0.01em; --si-heading-case: none;
  --si-radius: .4em; --si-shadow: none;
  /* documents */
  --si-prose-size: 1.08rem; --si-doc-measure: 72ch; --si-step-badge: var(--si-accent-2);
}
:root[data-si-mode="dark"] {
  --si-bg: #11161d; --si-fg: #e8e4dc; --si-muted: #9aa3b0; --si-surface: #19202a;
  --si-surface-2: #202834; --si-line: #2c3644; --si-accent: #5cc3e0; --si-accent-2: #f0a27c;
  --si-on-accent: #11161d;
}
/* repeat the dark block inside @media (prefers-color-scheme: dark) { :root:not([data-si-mode]) { … } } */
```

A `.css` theme file replaces the built-in theme, so it must set the colour,
font and layout tokens itself (`si --export css-rules` writes a complete,
commented starter with every token and its studio default). The same file
styles decks, articles, guides and labs. To change only a few things on top of
a built-in theme, keep `theme: studio` (or any) and pass the overrides with
`--css`, which lands in the `si.user` layer after the theme.

## 16. Configuring keys and features (si.conf)

An optional `si.conf` next to the deck (or `--conf <path>`) sets project-wide
defaults without editing each deck. It uses the same `key: value` style as
front matter — no new syntax. Blank lines and `#` comments are ignored.

| Key | Effect |
|---|---|
| `theme: <name>` | default theme when a deck sets no `theme:` and there is no `--theme` |
| `keys.<action>: <key>` | remap a command key |
| `feature.<name>: off` | turn a feature off (its key stops working and its toolbar button is hidden) |
| `links.<label>: <url>` | a page outside the build that find (`/`) offers, e.g. `links.Team wiki: https://wiki.example.com`; relative paths work too |
| `timer.time: 10,15,30` | break-timer presets (default `5,10,15,20,30,45,60`) |
| `timer.unit: seconds` | preset unit: `minutes` (default) or `seconds` |
| `timer.face: analog` | countdown face: `digital` (default) or `analog` (sport-clock dial) |
| `timer.animation: waves,orbit` | animation(s) behind the timer HUD: `waves`, `orbit`, `bokeh`, or custom names; a comma list gives a toggle |
| `timer.animation.path: ./anim` | directory of custom `<name>.js` / `<name>.svg` animations |

Nesting is written as dotted keys (`timer.animation.path`), not YAML.

Remappable **actions**: `overview`, `presenter`, `draw`, `timer`, `find`,
`mode`, `fullscreen`, `help`. Toggleable **features**: `draw`, `timer`,
`presenter`, `overview`, `find`.

```
# si.conf — flat key: value, same style as front matter (not YAML)
theme: crdb
keys.draw: w          # draw with w instead of a
feature.timer: off    # no break-timer HUD for these decks
```

Run `si --export conf` to print a starter `si.conf` with every key and feature
at its default. Theme precedence: `--theme` flag > a deck's `theme:` front
matter > `si.conf` > the built-in default (`studio`). `si play` re-reads
`si.conf` on every request, so edits show up on reload. An unknown key or a bad
value is a warning printed to the terminal, never a failed build.

## 17. What si check reports

`si check <dir|file.md>` exits 1 on any of:

- a deck that does not start with a `# H1` title slide, or is empty
- an unknown directive, layout, transition, theme, mode, aspect or front
  matter key; an accent that is not a colour
- a bad background or image option (`opacity=2`, `width=300px`)
- a missing file: image, SVG, background, poster, include, image inside an SVG
- an include cycle
- a `steps:` list with more state tokens than items
- an `image` directive not followed by a paragraph holding one image
- a second `# H1` inside a slide
- a `type` that is not `deck` or `lab`
- a malformed `stage` block (unknown node, impossible `dot`/`flow` route, bad `dir`)
- an authored canvas SVG with a `<` inside its `<style>`, or with no `viewBox`
  and no `width`+`height` to derive one

Warnings (exit 0): inlined files above 2 MB; slide or body directives in a lab.

## 18. Lab documents

`type: lab` in the front matter turns the file into one readable guide page,
for hands-on material that people follow at their own pace: a header with the
title and byline, a table of contents beside the text, the guide, and a footer.
There are no slides, steps, canvases or teleprompter.

```markdown
---
type: lab
title: Trace an Order Through the Hand-offs
---

# Trace an Order Through the Hand-offs

> [!NOTE]
> Everything here is read-only until the last step.

## Step 1: read the timeline
```

- The `# H1` is the page title: the first `# H1` whose text equals the title
  (compared as plain text, so `**bold**` or `` `code` `` in either still
  matches) is dropped from the body; any other H1 stays. `##` and `###`
  headings get stable GitHub-style ids and anchor links, and fill the table of
  contents. The contents sit beside the text on wide screens and collapse
  above it under about 900 px.
- A `---` line inside a lab is a horizontal rule, never a slide break.
- **Callouts** use GitHub alert syntax: `> [!NOTE]`, `> [!TIP]`,
  `> [!IMPORTANT]`, `> [!WARNING]`, `> [!CAUTION]`. These five are the only
  kinds (there is no separate checkpoint kind; use `[!IMPORTANT]` or
  `[!TIP]`). Each becomes a callout with an icon and a title. Callouts are
  recognised at the top level only (not inside lists or other quotes); any
  other blockquote stays a quote.
- Code blocks keep the highlighter, a language label and a copy button. The
  prose keeps a readable measure (78ch); on a wide screen a code block or a
  markdown table that needs more room grows to the whole text column, and
  only beyond that scrolls sideways inside its own box; the page never does.
  Raw HTML tables are not wrapped. A table whose header row is all empty
  (`| | |`, as pandoc metadata tables have) shows no header and is a
  key/value table: it stays at the prose measure and its value cells wrap
  (long words and `code` break) so it fits a phone without scrolling.

  ```markdown
  | | |
  |---|---|
  | **Time** | 30 minutes |
  | **Folder** | `labs/files/lab01` |
  ```
- Images become figures, links to other `.md` files become `.html`, and the
  theme, `mode:`, `accent:`, `--theme` and `--css` apply as they do for a deck
  (the `d` key toggles dark mode). The print stylesheet prints the whole guide.
- An include (`<!-- include: parts/setup.md -->`) pastes the file in place
  without a slide break. Keep included parts in a subfolder: `si build` builds
  every top-level `.md` of a folder (except `README.md`, which is never built)
  as its own deck or lab. A split lab keeps paths relative to each module, as
  a split deck does: `sections/01-setup.md` writes `![x](../images/x.svg)`,
  and problems point at the module's own file and line.
- Slide and body directives (`layout`, `steps`, `svg`, and so on) have no
  effect in a lab. Each one is reported by `si check` as a warning and
  otherwise ignored; an unknown directive (`<!-- columns -->`) is also only a
  warning in a lab (in a deck it is an error). A `Notes:` line is not special
  in a lab: it renders as ordinary text.
- Deck-only front matter keys (`aspect`, `transition`, `background`,
  `labs-label`, `footer-position`, `runbook`, `demos`) have no effect in a
  lab; `si check` warns about each. `watermark` and `watermark-opacity` work
  in every document type (labs, articles, guides, books, demos): a faint
  fixed layer over the page, on every printed page; guide and book pages
  inherit it from `_nav.md`. `footer`,
  `year` and `copyright` fill the page footer; `author`, `date` and `version`
  the byline. A ` · ` inside one of them splits it into separate byline items
  (`date: Harbor Labs · day 1` is two), and the dots between items are drawn
  so that a wrapped line never starts or ends with a lone separator.
- When `si build` builds a folder with more than one lab, each lab page ends
  with **Previous / Next** links to its neighbours in file-name order
  (`lab-01.md`, `lab-02.md`, … — number the files to set the order); decks in
  the same folder are skipped. A single lab, `si build file.md` and `si play`
  show no such links. Other links are ordinary links (`[Lab 2](lab-02.md)`
  becomes `lab-02.html`).
- `si build` writes one HTML file per lab and lists it in `index.html` with a
  `LAB` badge and its section count instead of slides and time. A slides PDF
  (`--pdf`) is not made for labs; print the page instead.
- In Studio, labs show a lab icon in the tree and open in the preview like
  decks.

The showcase has one: `examples/order-lab.md`.

**Search (`/`).** Every document (lab, article, guide page, book page, demo)
has a Search button and opens a search box on `/`, in `si serve` and in the
built files alike (it works from `file://`). It finds the words on this page
(a heading first, then the text, with the match marked), then the other
pages: in a guide or book every page, every section with a snippet of its
text, and a book's index terms; for a page on its own, the other pages of the
build. ↑ ↓ choose, Enter opens, Esc closes. Nothing is fetched: the other
pages' data is written into each page when it is built.

Labs share the document engine with articles (§19) and guides (§20). A
`## Step N:` (or `## N.`) heading in a lab or a guide page shows N in a
numbered badge; the contents list keeps the full text.

## 19. Articles

`type: article` is one long read: a blog post, a write-up, a case study.

```markdown
---
type: article
title: One retry fixed nine in ten night-time declines
description: Order #4711 failed at 21:03 and was paid at 21:05.
author: Jonas Berg · Payments team
date: 5 November 2026
---

# One retry fixed nine in ten night-time declines

For a year we treated every night-time card decline as a fraud signal …

## The declines were timeouts
```

- The header holds the title, the **lede** (`description:`, one or two
  sentences; it is also the page's `<meta name="description">` and the
  subtitle of its index tile), the byline (`author`, `date`, `version`) and
  the **reading time**, counted at 230 words a minute on the text outside code
  blocks.
- "On this page" (the `##` / `###` headings) appears on the right when the
  article has three or more `##` sections, and above the text on narrow
  screens; with fewer sections there is none.
- The text column is narrower than a lab's (88% of `--si-doc-measure`) and
  slightly larger. Everything else is as in a lab (§18): callouts, code with
  copy, key/value tables, figures, includes, print, theme and dark mode.
- `## Step N:` headings stay plain in an article.
- `si build` writes `<name>.html` with an ARTICLE badge on the index tile.

Example: `si --export example --type article`.

## 20. Guides (tutorials)

A guide is a **folder with a `_nav.md`**: a tutorial of several pages read in
order, with a navigation sidebar, a breadcrumb, Previous / Next links and
shared header and footer.

```text
order-journey/
  _nav.md          the order of the pages, and the guide-wide settings
  _header.md       optional: a band at the top of every page
  _footer.md       optional: a band above the footer of every page
  index.md
  install.md
  happy/place.md   pages may sit in subfolders
  reset.adoc       and may be AsciiDoc
```

`_nav.md` is front matter plus a markdown list. An item with a link is a
page; an item without one is a section heading; indentation nests.

```markdown
---
title: The order journey
description: Follow order #4711 from checkout to the doorstep.
version: v1.0
theme: paper
footer: Northwind Logistics engineering
edit-url: https://github.com/acme/order-journey/edit/main/docs
---

- [Introduction](index.md)
- Getting started
  - [Install the lab](install.md)
- The happy path
  - [Place an order](happy/place.md)
  - [Pay for the order](happy/pay.md)
- [Reset the lab](reset.adoc)
- [Status page](https://status.example.com)
```

- **Inherited settings.** Every page starts from the front matter of
  `_nav.md` (theme, mode, accent, footer, year, copyright, edit-url, author,
  version, …) and its own front matter overrides it key by key. Pages are
  `type: guide` unless they say otherwise (a `type: lab` page keeps its LAB
  badge). The guide's `title` and `description` stay the guide's: each page has
  its own `# Title` and `description:`.
- **Navigation.** The sidebar follows `_nav.md`: sections as small headings,
  the current page marked, pages the reader has opened marked as read (kept in
  that browser), and an "N of M read" count. On narrow screens it folds above
  the page. A link to an outside URL opens it.
- **Page.** A breadcrumb (guide › section › page), the title, the
  `description:` as a lede, the page, then **Previous / Next** in nav order
  and, with `edit-url:`, an "Edit this page" link (`edit-url` + the page's path
  in the guide). "On this page" sits on the right on wide screens.
- **Bar and partials.** A top bar shows the guide title (a link to the first
  page), the `version` and the light / dark button. `_header.md` and
  `_footer.md` are markdown, rendered on every page with their links resolved
  from the guide folder; `header: off` or `footer-page: off` in a page's front
  matter drops one on that page.
- **Links.** Link to the `.md` (or `.adoc`) file, with an anchor if you like
  (`[ports](install.md#step-2-start-the-stack)`); it becomes `.html`. Heading
  anchors are GitHub-style (lower case, spaces to hyphens, punctuation dropped).
- **`si check <guide>`** reports a page that `_nav.md` lists but does not
  exist (or is listed twice, or is outside the folder), a link between pages
  to a missing page or to an anchor the target does not have, and, as a
  warning, a page in the folder that `_nav.md` does not list (it is not built).
  `_`-prefixed files and `README.md` are never pages.
- **`si build <guide> -o out/`** writes one self-contained HTML file per page,
  mirroring the folders (`happy/place.md` → `out/happy/place.html`), and an
  `index.html` (the `index.md` page, or a redirect to the first page). In
  `si build <folder>`, each subfolder that holds a `_nav.md` is built into
  `out/<subfolder>/` and becomes one GUIDE tile on the index.
- **`si serve <folder>`** serves decks, articles, guides and labs with live
  reload; a guide folder URL opens its first page.

Example: `si --export example --type guide` (or `--type all` for every kind
side by side, then `si serve si-example`).

## 21. Books (references)

A book is a guide (§20) whose `_nav.md` says **`type: book`**: a whole
reference with a cover, a numbered contents page, parts, chapters,
appendices, numbered sections, figures and tables, cross-references and an
index, as one page per chapter **and** as one file (and PDF).

```markdown
---
type: book
title: The Northwind Operations Reference
subtitle: How an order moves through the stations, and how each break is repaired.
author: Ada Lindqvist · Northwind Logistics operations
edition: 2nd edition
date: November 2026
theme: paper
---

- [Preface](preface.md)
- Part I · Foundations
  - [The order journey](01-journey.md)
  - [Promises and clocks](02-promises.md)
- Part II · When it breaks
  - [Payments that time out](03-payments.md)
- Appendices
  - [Glossary](a-glossary.md)
```

- **Numbering.** Pages listed before the first section are front matter
  (preface, foreword): not numbered. A section of `_nav.md` is a **part**.
  Every page after the first section is a chapter, numbered 1, 2, 3 in nav
  order; pages under a section called "Appendices" (or "Appendix …") are
  lettered A, B. A chapter's `##` headings are numbered 1.1, 1.2 (in the text,
  the sidebar and "On this page"); `## Step N:` headings get no badge in a book.
- **Figures and tables.** An image with a title is a captioned figure:
  `![Stations](images/stations.svg "The five stations")` becomes "Figure 1.1 ·
  The five stations". A paragraph `Table: Who owns each station` right after a
  table is its caption: "Table 1.1 · Who owns each station". (Outside a book
  both captions work too, without numbers.) Their ids are `fig-` / `tbl-` plus
  the caption in GitHub style: `#fig-the-five-stations`, `#tbl-who-owns-each-station`.
- **Cross-references.** A link with empty text is filled with the target's
  label: `[](#tbl-who-owns-each-station)` → "Table 1.1",
  `[](02-promises.md#the-promise-tile)` → "Section 2.1", `[](01-journey.md)` →
  "Chapter 1". `si check` reports a cross-reference to something that has no
  number.
- **Index.** `((idempotency key))` shows the term (lightly marked) and indexes
  it; `(((payment, retry)))` indexes without showing anything; a comma makes a
  sub-entry (`((promise, computed at checkout))` shows "computed at
  checkout", filed under "promise"). This is AsciiDoc's own syntax. The index
  page lists every term A–Z with the section numbers where it appears (or the
  chapter, or the page title for front matter), each a link to the spot.
- **Generated pages.** `index.html` is the **cover** (title, subtitle,
  edition, author, date, links to the contents, the first chapter, the PDF and
  the one-file book, and `_cover.md` as the colophon), `contents.html` lists
  front matter, parts, numbered chapters with their sections and reading
  times, appendices and the index, `book-index.html` is the index, and
  `book.html` is the **whole book in one file** (cover, contents, every
  chapter, index; links between chapters become in-page links). Every page has
  a bar with Cover · Contents · Index (· PDF). A book has no `index.md`: name the
  first page `preface.md` or similar.
- **PDF.** `si build <book> --pdf` also prints `book.html` to `book.pdf`
  (each chapter starts a new page) and links it from the cover and the bar.
- `si serve` serves a book live (the whole book is numbered on every
  request); in a folder, a book is one BOOK tile on the index.

Example: `si --export example --type book` (also `--type reference`).

## 22. Demo scripts

`type: demo` is the presenter's run sheet for a live part of a session: a
document (built like a lab) **and** the content of the ▶ Demos panel of the
`si play` terminal (a deck lists it with `demos:`, §14).

````markdown
---
type: demo
title: "Live: order #4711 meets a slow bank"
session: The Thursday Promise
slides: 6 (Payment: declined, then paid)
duration: 6 min
---

# Live: order #4711 meets a slow bank

## What this demo must accomplish
1. **A timeout is not a decline.** The room sees the bank answer late.

## Before the session (T-30 min)
```bash
docker compose up -d --wait
```

## The script
### Slide 6, "Payment: declined, then paid" (4 min)
> Say: "I am going to make the bank slow, the way it is every night at nine."
```bash
curl -X POST localhost:8081/admin/latency -d '{"ms": 25000}'
```
```text
{"ms": 25000}
```
Point at the latency while the room waits.

## If it goes wrong
## Reset between rehearsals
````

- The header carries an **INSTRUCTOR ONLY** badge, and `session`, `slides`
  and `duration` fill a facts block under the title.
- **`> Say:` is what the presenter says to the room, word for word** (the
  teleprompter rule): spoken sentences, never directions. It renders as a
  speech callout. Directions to the presenter ("point at the latency") are
  plain text. Any other quote stays a quote.
- Shell blocks (`bash`, `sh`, `zsh`, `console`, `sql`, `powershell`) are
  commands: Copy / Run in the ▶ Demos panel. `text` (or `output`) blocks are
  what the room should see: labelled "expected output". Follow every command
  block with its expected output.
- Script steps are `### Slide N, "title" (M min)` (or any `###` with
  `(M min)`); `si check` warns when they do not add up to `duration:`.
- The usual sections: *What this demo must accomplish*, *Before the session
  (T-30 min)*, *The script*, *If it goes wrong*, *Reset between rehearsals*.
- `si build` builds a demo like a lab (DEMO badge on its index tile);
  `si build --no-instructor` leaves every demo out, for the copy you hand to
  participants. The ▶ Demos panel shows the script without its front matter.

Example: `si --export example --type demo`; the showcase deck lists it, so
`si --export example && si play si-example` shows it in the ▶ Demos panel.


## 23. Sites

A **site** is a folder with a `_site.md`: the guide format (front matter plus
a nested markdown list), but every page gets the site's **navigator** instead
of the guide bar and sidebar. Pages are plain pages by default (`type: page`:
no `---` slide breaks, no slide directives), and any page may say
`type: article` (byline, contents) or `type: deck` (a talk, built as slides).
A link in the list to a folder that holds its own `_nav.md` or `_site.md`
(a guide, a book, another site) opens that one; it is built and checked on its
own, and links into it are not checked by the parent.

````markdown
---
title: Northwind Engineering
description: How Northwind Logistics moves forty thousand orders a day.
navigator: topbar
logo: Northwind Engineering
cta: Run the lab → docs/index
theme: studio
---

- [Home](index.md)
- Writing
  - [Retries without the stampede](blog/retries.md)
  - [The order journey, the talk](talks/journey.md)
- [Docs](docs/)
- [About](about.md)
````

| Key | What |
|---|---|
| `navigator:` | `topbar` (default: logo left, menu, call to action right), `topbar-centred` (menu left, logo centred), `mega` (sections open as wide panels), `sidebar` (a sticky column on the left). On narrow screens every navigator folds its menu behind a **Menu** button. |
| `logo:` | text, or an image path (inlined like any image) |
| `cta:` | `Label → page` (or `→ https://…`, `→ #modal`): the highlighted button |
| `title: off` | on a page: no `<h1>` (a home page that opens with its own content) |
| `header: off` / `footer-page: off` | on a page: leave out `_header.md` / `_footer.md` |

An item without a link (`- Writing` above) is a menu section: its children
open as a dropdown. `_header.md` and `_footer.md` in the site folder are
rendered above the navigator and below every page, with their links rewritten
for each page. `/` searches every page of the site; `watermark:` works as in
every document. `si build site/`, `si check site/` and `si serve site/` handle
the whole folder. The index page title is `--site-title <text>` (the old
`--site <text>` still works, with a warning).

### Plugins

Plugins are directives that work in every document (pages, articles, guides,
books, labs, demos), so in every site. They are plain HTML plus a few lines of
runtime; nothing is fetched. On a slide they are reported as a problem.
Options follow the text after a `·`; a `·` inside the text starts an option,
so `si check` reports a kind it does not know.

```markdown
<!-- alert: The lab moves to the new cluster on 24 November. · until=2026-11-25 · warn -->
<!-- badge: Pipeline healthy, p95 checkout 410 ms · ok -->
<!-- event: 2026-11-20T17:00 Live: replaying Black Friday 2025 · Save a seat → #seat -->
```

| Plugin | What |
|---|---|
| `alert: text · kind · until=YYYY-MM-DD` | a dismissible band; the reader's dismissal is remembered in that browser; hidden after `until`. Kinds: `info` (default), `ok`, `warn`, `crit`. |
| `badge: text · kind` | a status pill, same kinds |
| `event: YYYY-MM-DDTHH:MM text · Label → link` | a corner toast with a live countdown, shown from two days before the time until it starts |
| `accordion` | the `###` questions that follow become a one-open-at-a-time list; it ends at the next `##`, `---` or directive |
| `modal: id` … `end` | the blocks up to `<!-- end -->` become a dialog; any `[link](#id)` (or the `cta:`) opens it, and so does a URL ending in `#id`. A document part ends at `##`, so use `###` headings inside a modal. |
| `carousel: auto=7s` | the image paragraphs that follow slide one at a time: arrows, dots, swipe and ←/→; `auto=` advances it (never with reduced motion); printing shows every image |
| `embed: …` | another document, inside this page (below) |

```markdown
<!-- carousel: auto=7s -->

![Pickers at aisle 14](images/aisle.svg "Evening wave, aisle 14")

![The packing line](images/packing.svg "Packing, stations 3 to 5")

<!-- accordion -->

### Can I run the lab on a laptop?

Yes. It needs Docker and about 4 GB of memory.

### Is the lab the real code?

It is a trimmed copy with stubbed providers.
```

### Embed

`<!-- embed: … -->` puts another si document inside the page:

| Form | Result |
|---|---|
| `embed: blog/retries.md` | a card: type, title, description, reading time, linking to the page |
| `embed: blog/retries.md#what-we-changed` | that section's text, with a *From …* link |
| `embed: blog/retries.md full` | the whole body |
| `embed: talks/journey.md 1-3` | a deck: its slides, live, in a frame (as `<!-- slides: -->`) |
| `embed: docs/` | a guide, book or site folder: a card with its first pages or chapters |

Example: `si --export example --type site` writes this site; `si serve
si-example` serves it.
