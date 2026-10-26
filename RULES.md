# simple-ideas markdown rules

Everything `si` understands, with an example for each rule. `si rules` prints
this file; `examples/showcase.md` uses every rule in a real talk.

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
| `theme` | `studio`, `paper`, `mono` | `studio` | built-in look (sans, serif, monospace) |
| `mode` | `auto`, `light`, `dark` | `auto` | `auto` follows `prefers-color-scheme`; `t` toggles while presenting |
| `accent` | CSS colour | theme accent | sets `--si-accent` |
| `transition` | `fade`, `slide`, `zoom`, `none` | `fade` | default slide transition |
| `aspect` | `16:9`, `16:10`, `4:3`, `1:1`, `9:16` | `16:9` | shape of overview tiles, teleprompter previews and printed pages; slides on screen always fill the window |
| `background` | as the [background directive](#7-backgrounds) | none | background of every slide without its own |
| `mermaid` | `off` or a script URL | pinned CDN URL | how Mermaid fences render |
| `lang` | language tag | `en` | `<html lang>` |

An included module's own front matter is ignored; the including deck decides.

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
| `class: dense dark-code` | slide | adds classes to the slide for your CSS |
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
The token reaches payments and turns red. Pause here.
- Mention the thirty-second retry
```

- A line starting with `Notes:` starts the slide's speaker notes; everything
  after it, to the end of the slide, is notes (markdown).
- `(N min)` on that first line is the slide's planned time; the deck total is
  the sum, shown in the index, the build summary and the teleprompter.
- Notes are hidden on the slides.

**Teleprompter (`p` or `Shift+S`).** A second window, synced with the
audience window over `BroadcastChannel`; it works in `si play` and in the
static HTML (`file://`), and either window drives both.

| In the teleprompter | |
|---|---|
| Notes | large type for reading at a distance; a guide line at reading height |
| Auto-scroll | `a` on/off, `+` / `-` speed 1 to 10 (lines per ten seconds, follows the text size) |
| Manual scroll | ↑ ↓, mouse wheel, trackpad; `r` back to the top |
| Text size | `[` / `]` |
| Mirror | `m` flips the notes for teleprompter glass |
| Notes only | `x` hides the previews, the notes fill the window |
| Previews | the current slide (with its step state) and the next step or slide |
| Clocks | time on this slide against its `(N min)` (amber when over, red at 1.5×), total against the deck plan, ahead/behind pace, wall clock |
| Timer | `t` pause/resume, `0` reset |
| Navigation | → ← Space PgDn PgUp Home End, buttons |

Settings (size, speed, mirror, notes only) are remembered per browser.

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
  route A→B. Add `step <node...>` lines to walk the flow on build steps — each
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

## 14. Presenting: keys, touch, overview, print

| Key | Action |
|---|---|
| → ↓ Space PgDn Enter | next step, then next slide |
| ← ↑ Shift+Space PgUp | previous step or slide |
| Home / End | first / last slide |
| `o` | overview grid of every slide (click one to open it, Esc closes) |
| `p` or `Shift+S` | teleprompter window |
| `t` | break timer HUD: pick minutes, countdown to project during a break |
| `a` | draw: a pen layer over the slide |
| `d` | light / dark (remembered) |
| `f` | full screen |
| `?` | key help |
| swipe left / right | next / previous on touch screens |

- **Break timer** (`t`): opens a full-screen HUD that floats over the deck.
  `1`–`7` pick 5 / 10 / 15 / 20 / 30 / 45 / 60 minutes and start the countdown;
  `Space` pauses and resumes; `t` or `Esc` closes it. At zero the digits turn
  critical and flash. While it is open the deck does not advance, so it is safe
  to leave projected during a break. It needs no network and works from
  `file://`.

- **Draw** (`a`): a transient pen over the current slide, for circling or
  underlining as you talk. Drag to draw (mouse, touch, or pen — pen pressure
  varies the stroke); `c` cycles colour, `e` erases strokes, `z` undoes the
  last, `x` clears all, `Esc` turns it off. Strokes use the theme colours and
  clear when you change slides; they are not saved and do not print.

- The URL hash is the position (`#7`, `#7.2`); reloads keep it, so live
  reload in `si play` stays on your slide.
- A progress bar and a slide counter (with the step) sit at the bottom; the
  controls appear when the mouse moves.
- When a slide is too tall for the window the fluid type shrinks a little,
  then the slide scrolls; nothing is cut off.
- **Print** (browser print, or print to PDF): one slide per page at the deck
  `aspect`, every step revealed, canvases in their final state. `si build --pdf`
  writes a `<name>.pdf` next to each deck using headless Chrome (needs Chrome or
  Chromium installed; without it, HTML is still written).
- **Open on a phone**: run `si play <dir> --addr 0.0.0.0:7070` to serve on your
  network; `si play` then prints a LAN URL and a QR code to scan. (The default
  address is loopback-only, so it prints the `--addr` hint instead.)

## 15. Styling: themes and your own CSS

- `theme:` picks a built-in theme (`studio`, `paper`, `mono`, `crdb`), each
  with a light and a dark palette. `crdb` is the CockroachDB University brand
  (cream / ink / purple; aliases `cockroachdb`, `cockroachlabs`).
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

## 16. Configuring keys and features (si.conf)

An optional `si.conf` next to the deck (or `--conf <path>`) sets project-wide
defaults without editing each deck. It uses the same `key: value` style as
front matter — no new syntax. Blank lines and `#` comments are ignored.

| Key | Effect |
|---|---|
| `theme: <name>` | default theme when a deck sets no `theme:` and there is no `--theme` |
| `keys.<action>: <key>` | remap a command key |
| `feature.<name>: off` | turn a feature off (its key stops working and its toolbar button is hidden) |

Remappable **actions**: `overview`, `presenter`, `draw`, `timer`, `mode`,
`fullscreen`, `help`. Toggleable **features**: `draw`, `timer`,
`presenter`, `overview`.

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
- a malformed `stage` block (unknown node, impossible `dot`/`flow` route, bad `dir`)
- an authored canvas SVG with a `<` inside its `<style>`, or with no `viewBox`
  and no `width`+`height` to derive one

Warnings (exit 0): inlined files above 2 MB.
