# simple-ideas markdown rules

Everything `si` understands, with an example for each rule. `si rules` prints
this file; `examples/showcase.md` uses every rule in a real talk.

> **AsciiDoc too.** A deck can be `*.adoc` / `*.asciidoc` instead of `*.md`; si
> converts the AsciiDoc to this same format and everything below applies. See
> `docs/asciidoc.md` for the supported subset. On the docs site every example
> has an **AsciiDoc** tab beside the Markdown one.

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
23. [Sites](#23-sites)
24. [Mind maps](#24-mind-maps)
25. [Code windows and code shows](#25-code-windows-and-code-shows)
26. [Publishing](#26-publishing)
27. [Web demos](#27-web-demos)
28. [Courses](#28-courses)
29. [Timelines](#29-timelines)
30. [Infographics](#30-infographics)

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
| `theme` | `studio`, `paper`, `mono`, `slate`, `twilight`, `forest`, `sunrise`, `neon`, `newsprint`, … (`si theme list`) | `studio` | built-in look; pictures of each on [the Themes page](https://simple-ideas.org/themes.html) |
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
| `browser` | `Name=URL, …` | none | pages the presenter opens with **w** in floating browser windows over the slides (**`si play` only**); see §14 |
| `type` values | `deck`, `lab`, `article`, `guide`, `book`, `mindmap` | `deck` | `lab` §18, `article` §19, `guide` §20 (guide pages get it from `_nav.md`), `book` §21 (in `_nav.md` only), `mindmap` §24 (with `layout`, `lines`, `nodes`, `map-theme`, `watermark-position`) |
| `subtitle`, `edition` | text | none | books: the cover's subtitle and edition (§21) |
| `session`, `slides`, `duration` | text | none | demo scripts: the facts under the title; `duration: 14 min` is checked against the script's steps (§22) |
| `app`, `viewport`, `timeout`, `setup`, `voice` | URL or folder; WxH or phone, tablet, desktop; seconds; a shell command; engine:voice | none; desktop; 5s | web demos: `app:` turns a demo script into a script si runs against a web app (§27) |
| `setup`, `start`, `stop`, `cleanup` (each may repeat) | a command, or a source: `gist:ID@REV`, `github:owner/repo@REF/path`, `https://… · sha256 HEX` | none | a talk that sets up what it shows live: `si play` and `si serve` run `setup:` (must finish) and `start:` (keeps running, `· wait for "ready"`) in the order written before the first page, then `stop:`, the programs stopped, `cleanup:` when they quit; `si build` never runs them ([§14](#14-presenting-keys-touch-overview-print), A talk that sets itself up) |
| `prepare`, `step`, `cleanup` | web demos: a command run once before the first feature (`step:` is its alias); a command run after every program stopped | none | §27, Before and after |
| `description` | text | none | documents: the lede under the title, the index tile subtitle and `<meta name="description">` (§19, §20) |
| `edit-url` | URL | none | guides: base of the "Edit this page" link (§20) |
| `header`, `footer-page` | `off` | on | a guide page without the guide's `_header.md` / `_footer.md` (§20) |
| `repo` | `OWNER/REPO`, a github.com URL, or a local clone folder | none | the default repo of `code:` and `diff:` (§13) |
| `voices` | `Name=voice, Name=voice` (`Ada=af_heart, Ravi=kokoro:am_michael`) | every speaker in `voice:` | auto-play in a deck with `speakers:`: each speaker's part in their own voice; a bare name is a voice of the deck's engine (§11, Auto-play) |
| `linkedin`, `tags`, `canonical`, `cover` | text; comma-separated tags; a URL; a picture | none | `si publish` (§26): the LinkedIn post text, dev.to / Hashnode tags, the article's original URL, the picture of a LinkedIn link post |

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
| `transition: zoom`, `transition: morph 0.9s` | slide | [Transitions](#6-transitions) |
| `id: promise` | next block | names the block so `transition: morph` matches it on the next slide ([Transitions](#6-transitions)) |
| `background: assets/city.jpg opacity=0.4` | slide | [Backgrounds](#7-backgrounds) |
| `footer: off` | slide | overrides the deck `footer` on this slide (`off` hides it, e.g. the title) |
| `labs: [Create a Cluster](https://…), [Scale out](https://…)` | slide | turns the slide into a lab slide: a centred beaker icon, the `labs-label` heading, and a button per lab (opens in a new tab). A bare URL works too and shows a default label. |
| `class: dense dark-code` | slide | adds classes to the slide for your CSS |
| `eyebrow: Order #4711 · the promise` | slide | a small context label above the slide heading (`.si-kicker`); it is not part of the slide title, so find, the overview and the teleprompter show the heading alone. On a section slide it replaces the "Part N" kicker. |
| `image: width=40% align=right` | next image | [Images](#8-images) |
| `steps`, `step`, `steps: a b`, `step: a` | next block | [Build steps](#9-build-steps) |
| `svg: assets/canvas.svg state words` | body | [SVG canvases](#10-animated-svg-canvases) |
| `include: other.md` | deck | [Includes](#12-includes) |

Body directives (`column`, `id`, `image`, `step`, `steps`, `svg`) work on the
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

`<!-- transition: fade|slide|zoom|none|morph -->` sets how a slide arrives; the
front matter `transition:` sets the default (`fade`).

**Morph.** With `transition: morph`, what the slide shares with the slide
before it glides from its old place and size to its new one (colour and
opacity too); everything else fades out and in. Two things match when they
have the same name, the same heading text, the same picture, or the same
words, in that order. Name a block with `<!-- id: name -->` on the line
before it when its words change between the slides:

```markdown
<!-- transition: morph -->
## Five teams

- shop, payments, warehouse, courier
- Nobody owns the promise from end to end

<!-- id: promise -->
Two days, door to door.

---

<!-- transition: morph 0.9s -->
<!-- layout: center -->
## Five teams

**Nobody owns the promise from end to end**

<!-- id: promise -->
Two days, **door to door**.
```

The heading and the line in bold match by their words, the last paragraph by
its name. Words keep their shape as they move (they scale by font size, not
by box); pictures, code and tables scale to their new box. `morph 0.9s` sets
the length for that slide; `--si-morph-duration` (`.7s`) and
`--si-morph-ease` set it for the deck. The morph runs between neighbours
only, forward and back (going back from a morph slide morphs too); a jump,
the overview and print show the slide as it ends. Build steps reveal after
the morph, and only what is already shown takes part. Two slides that show
the same SVG keep the canvas transition below.

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
deck is one self-contained file. A file used on several slides (the same
photo as two backgrounds and a figure) is embedded once. Files above 2 MB draw
a warning; build with
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

**Frames, drawn in the Studio.** The Studio's Motion view (open an `.svg`,
pick Motion) builds a canvas from frames: draw frame 1, add a frame, drag,
resize, turn, recolour or hide parts, and set how long each move takes. It
writes a normal canvas you can still edit by hand: parts carry
`data-part="name"`, and one block in the `<style>` between
`/* si motion: frame1 frame2 frame3 */` and `/* si motion end */` holds a rule
per frame (coordinates in `viewBox` units, colours as tokens):

```css
/* si motion: frame1 frame2 frame3 */
[data-part]{transform-box:fill-box;transform-origin:center}
[data-state~="frame2"] [data-part="token"]{transform:translate(310px,0px)}
[data-state~="frame3"] *{transition-duration:1.2s;transition-timing-function:cubic-bezier(.3,1.5,.5,1)}
[data-state~="frame3"] [data-part="token"]{transform:translate(620px,40px) scale(1.2);fill:var(--si-ok)}
/* si motion end */
```

Frame 1 stays on; each click adds the next frame and takes the one before off:

```markdown
<!-- svg: anim/token.svg frame1 -->
<!-- step: +frame2 -->
<!-- step: -frame2 +frame3 -->
```

A part's CSS `transform` replaces its own `transform=` attribute, so draw a
moving part without one (or wrap it in a `<g>` and name the `<g>`).

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
| Rehearsal | the time you spent on each slide is recorded per run; next to the slide clock it shows `last M:SS` — your previous run's time for this slide, to pace against; and the whole talk against it: `vs last run 0:40 behind` (works without any `(N min)` plan) |
| Timer | `t` pause/resume, `0` reset (promotes this run to the `last` reference and starts fresh) |
| Record | **● Record** (Shift+R) under `si play`: records your voice from the microphone with every slide and step change; Stop saves `.si/recordings/<talk>-<date-time>/` beside the talk (`audio.webm`, `timeline.json`, `meta.json`). It feeds `--mp4 --narration` and `si rehearse` |
| Navigation | → ← Space PgDn PgUp Home End, buttons |

Settings (tab, size, speed, mirror, notes only) are remembered per browser.

**Two (or more) presenters.** A talk given by two people names them in the
front matter and gives each one a script: `Notes[Name]:` is that speaker's
verbatim script (spoken sentences, `[CLICK]` cues included), `Points[Name]:`
their own talking points and reminders.

```markdown
---
title: The Thursday Promise
speakers: Ana, Felipe
---

# The Thursday Promise

---

## Payment: declined, then paid

<!-- steps: on -->
- The card fails a fraud check
- A retry thirty seconds later goes through

Notes[Ana]: (1 min)
The token reaches payments and turns red. [CLICK] Felipe built what happens next. [CLICK]

Notes[Felipe]: (0.5 min)
Thank you, Ana. One retry, thirty seconds later, and the order is paid.

Points[Felipe]:
- If asked: a real fraud decline is never retried
```

- `speakers:` lists the names, comma-separated, in any number. Plain `Notes:`
  and `Points:` belong to the first speaker, so a deck that gains a second
  speaker keeps working; names match without regard to case.
- Order on a slide is speaking order, and a speaker may come back
  (Ana, Felipe, Ana). Each `(N min)` is that speaker's time; the slide's
  planned time is their sum.
- A part starts on the build step after the `[CLICK]` cues before it: above,
  Ana speaks on steps 0 and 1, Felipe from step 2. Put a `[CLICK]` where the
  turn passes so the teleprompters know who is speaking. `si check` counts the
  cues of every speaker on a slide together.
- `si check` fails on a name that is not in `speakers:` (or `Notes[Name]:`
  without `speakers:`) and warns about a speaker with no lines in the deck.
- **The teleprompter** has a speaker picker (`s` cycles it; `?speaker=Ana`
  on a `?si-view=tp` or remote teleprompter link picks it): your own script in
  full, every other part as one quiet line with who, how long and their first
  words, so you know your cue. *Everyone* shows every part under its name.
  When the turn passes, every teleprompter shows **→ Felipe** for a few
  seconds. A clock beside the slide clock counts your own speaking time
  against your planned minutes. The Speaker Notes tab shows only your points.
- `si play talk.md --remote` prints one teleprompter link and QR code per
  speaker (`/tp?k=…&speaker=Ana`); `q` shows them too.
- `speakers.control: turn` in `si.conf` lets only the speaker whose turn it is
  advance from a teleprompter (the other's Prev and Next are greyed); the
  default `any` lets every teleprompter drive, as with one speaker.
- In the terminal: `si play talk.md --terminal --notes --speaker Ana`.
  Handouts with `--handout=notes` or `points` put each part under its
  speaker's name; `si rehearse` adds a planned against spoken line per speaker.

**Auto-play: the deck presents itself.** Your `Notes:` are already the exact
words and `[CLICK]` already marks every advance, so a deck can give the talk
by itself: press **▶ Auto-play** in the deck's buttons, or `v`. It reads the
slide's notes aloud, advances at each `[CLICK]` (a build step, a canvas
change or the next slide, the same as →), and after the last part moves to
the next slide. A slide with no notes stays its `(N min)` share, or 3 seconds
(`autoplay.dwell` in si.conf). The spoken line shows as a caption (CC hides
it); the bar pauses, resumes and stops. Any key that moves the deck, a click
on the slide, the pen or a teleprompter's Next takes control back: auto-play
pauses where you are, and Resume carries on from there. The teleprompter
shows *auto-play* beside the slide number and follows along.

```markdown
---
title: The Thursday Promise
voice: kokoro:af_heart
speakers: Ada, Ravi
voices: Ada=af_heart, Ravi=am_michael
pronounce: SLA = service level agreement
---

## Payment: declined, then paid

<!-- steps -->
- Declined by the fraud check
- Paid on the retry

Notes[Ada]: (1 min) The token reaches payments and turns red. [CLICK]
Notes[Ravi]: Thirty seconds later the retry clears it. [CLICK]
```

Where the voice comes from:

- **`si play`** voices each part when the page asks for it
  (`GET /_si/voice/<slide>/<part>.wav`, slide from 1, part from 0), one part
  ahead, through the same cache as `si voice say` and `si demo run`: the
  second run is instant and only an edited part is voiced again. The voice is
  the front matter `voice:`, else si.conf `voice:`, else Kokoro `af_heart`
  when it is installed (`si voice install kokoro`), else the computer's own;
  `voices:` (or si.conf `voice.<Name>:`) gives each speaker theirs;
  `pronounce:` applies, the caption keeps the written words. Needs ffmpeg.
  `si play talk.md --autoplay` opens decks with the bar ready (`--open` starts
  at once where the browser lets a page speak).
- **`si build talk.md --autoplay`** voices every part at build time into
  `talk.voice/<slide>-<part>.m4a` beside the page, which plays them by
  relative URL (copy the folder with the page). `--autoplay=inline` puts
  them inside the page for a single file to share (a warning over 20 MB).
  `--voice kokoro:bf_emma` picks another voice for the build.
- **A deck with no audio** (a plain build, a part that failed) speaks with
  the browser's own voice (Web Speech), labelled *browser voice*, offline.

`#autoplay` at the end of the address (`talk.html#autoplay`) opens the deck
ready; browsers let a page speak only after a click or a key, so the first
one starts it. `si build talk.md --mp4 --voice kokoro` films the deck
narrated from its notes, each view held while its part is said (§14, Video).
Auto-play depends on the `[CLICK]` count: `si check` warns when it does not
fit the slide's steps.

**On another device (`si play --remote`).** Put the teleprompter on an iPad
(or any tablet or phone) and project from the laptop. `--remote` opens a
second listener on your network, next to the loopback one, and prints one
keyed URL and QR code per role:

```bash
si play talk.md --remote                    # every role
si play talk.md --remote=tp                 # teleprompter only
si play talk.md --remote --remote-addr 0.0.0.0:9000
```

Press `q` on the deck (the laptop's window, the one you project) to show the
same links as QR codes, big enough to scan from the room; `q` or Esc closes
them (`keys.remote` in si.conf remaps it). When the laptop is on more than
one network (Wi-Fi and Ethernet, or a VPN such as Tailscale), the other
addresses are listed under the codes.

**The room follows along (`--audience`).** `si play talk.md --audience`
gives the audience their own link: their phones show the current slide as
you present, and nothing else. The audience key is separate from yours, so a
phone in the room can never drive the deck or open the teleprompter, and
their pages never carry your `Notes:` or `Points:` (not even in the page
source). Put the code on a slide:

```markdown
# The Thursday Promise

<!-- qr: audience · caption=Follow along on your phone · size=small -->
```

**Live polls.** A slide asks, the room votes on their phones, and the bars
grow on the projected slide:

```markdown
## Which habit would your team start with?

<!-- poll: first-habit · Follow one order end to end · Retry once, then tell the customer · Lend people before adding systems -->
```

The first part is the poll's id (lowercase letters, digits, `-`), then 2 to
6 options; `· multiple` lets a phone pick several, and `· hidden` keeps the
results for the next click: until then the slide and the phones show only how
many have voted (no bars, no counts), so the first votes do not sway the rest.
That click is a build step of the slide, so give it a `[CLICK]` in `Notes:`:

```markdown
<!-- poll: retry · Once, after 30 s · Three times, backing off · Never · hidden -->

Notes:
Which one would you ship? Take a moment. [CLICK] Here is the room.
```

A phone votes once per
poll and can change its mind (unpicking everything takes the vote back); it
sees the results after voting. Late phones get the current counts. Without
`si play --audience` (a built deck, a document) the poll is the plain list of
its options. Votes are kept in memory for the run.

**A remote audience (`--tunnel`).** For a talk on a video call, `si play
talk.md --tunnel` puts the audience side on a temporary public HTTPS address
through Cloudflare's free quick tunnel (install `cloudflared` once:
`brew install cloudflared`). Paste the printed link in the call's chat, or
show `<!-- qr: audience -->`, which then carries that link. Only the audience
side is reachable there (never your teleprompter or controls); votes and
questions still live on your laptop, and the address is gone when `si play`
stops. A quick tunnel carries no event stream, so audience pages there ask
for the current slide about once a second.

**Word clouds.** A slide asks for a few words; each phone sends up to three
answers (30 characters each, a fourth replaces the first) and the slide shows
them as a cloud, each word sized by how many people gave it (the same word in
any case counts once):

```markdown
## In one word, what makes a delivery late?

<!-- cloud: late -->
```

`· hidden` keeps the words for the next click, as for polls. Free text from
the room reaches the screen, so the teleprompter's **Q&A** tab lists the
words with *hide* beside each: a hidden word leaves the slide and every phone
at once (*show* brings it back). Without `si play --audience` the slide says
where the words will appear.

**Questions from the room.** Under `--audience` every phone has an *Ask a
question* button (500 characters, an optional name, one question per phone
every 20 seconds). Questions go to you only, never to the other phones: a
**Q&A** tab appears in the teleprompter (key `3`), newest first, each with
*Show on screen* (a band at the foot of the projected slide; Esc on the deck
takes it off), *Answered* and *Hide*. When `si play` stops, the poll counts
and the questions are saved beside the talk in
`.si/audience/<talk>-<date-time>.json` (keep `.si/` out of git);
`audience.save: off` in `si.conf` keeps nothing. The file holds the cloud
words too.

The code appears only under `si play --audience` (in a built deck the figure
stays hidden). It works with `--remote` (your roles and theirs, two keys) and
`--https`, and `q` shows the audience code with the others.

`--https` serves the remote pages over HTTPS, so the phone keeps its screen
on and the notes are private on any network. si makes its own certificate
authority once (it can only vouch for local addresses) and each phone trusts
it once: scan the "first time on this phone" code, install the certificate,
and turn on full trust (iPhone: Settings › General › About › Certificate
Trust Settings; without that switch Safari still warns). `si ca trust` does
the same on the laptop. With Tailscale on the laptop and the phone,
`--https=tailscale` uses a real certificate for the machine's `ts.net` name
instead, with nothing to install. See `docs/remote.md`.

| Role | On the device | Drives the deck |
|---|---|---|
| `tp` | the teleprompter above: script, tabs, auto-scroll, clocks, previews, mirror; on a phone the script fills the screen (no previews, one clock line, big Prev / Next) | Prev / Next, keys |
| `draw` | the current slide with the pen always on (Apple Pencil pressure works); Prev / Next in the pen bar | the strokes appear on the projected slide live; Prev / Next |
| `clicker` | a phone remote: big Next and Prev, the slide and step (`2 / 22 · step 1/4`), its title, what comes next, and a stopwatch (tap it to reset) | Prev / Next |

- The deck on the laptop stays on `127.0.0.1`, so the terminal drawer keeps
  working. The remote listener (default: every interface, the play port + 1)
  serves the decks, their files and the sync, never the terminal.
- Pages sync through the server (server-sent events and small POSTs), the
  deck stays the source of truth, and a device that connects late jumps to
  the current slide. Ink is shared both ways in slide coordinates and clears
  when the slide changes, like the local pen (`a`). A phone that slept or
  left the browser reconnects when it is shown again and jumps to the
  current slide.
- A device page keeps its screen on (the browser's wake lock) where the
  browser allows it: on `localhost` or HTTPS. Over plain HTTP on the LAN
  (an iPhone or iPad at `http://192.168.…`) it cannot, so the page says so
  for a few seconds: set Auto-Lock to Never for the talk.
- The URL carries a random per-run key; the first visit stores it as a
  cookie, and anything without it gets 403. It is plain HTTP: use it on a
  network you trust.
- With a folder of decks the QR code opens a picker (deck × role).
- Play only: a built deck never contains any of it. See `docs/remote.md`.


**Teleprompter in a tab or frame.** Opening a deck page with `?si-view=tp`
(for example `talk.html?si-view=tp`) turns that page into the teleprompter of
the same deck open in another tab, window or frame of the same browser,
synced like the `p` window. `si --web` uses it for its Rehearse and Lab modes.
`?si-view=thumb` shows one slide as a still picture (every build step shown,
no sync, no live reload); the Studio storyboard uses it for its thumbnails.

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
  JavaScript/TypeScript, Java, Kotlin, C# (`csharp`, `cs`, `c#`), C, C++,
  PHP, Ruby, Swift, Scala, Rust, SQL, YAML, TOML, JSON, Bash, PowerShell,
  Dockerfile, Terraform/HCL (`hcl`, `tf`), Protobuf (`proto`), GraphQL (`gql`),
  F# (`fsharp`, `fs`), Elixir (`ex`, `exs`), CSS, HTML, XML (also `csproj`, `props`, `xaml`, `svg`, `pom`,
  `nuspec`: tags, attributes, comments)) and a language label written as a
  reader expects it (C#, JavaScript, Shell, XML…). Keywords, strings,
  comments and numbers are coloured, and so are calls (the name before a `(`)
  and, in C#, Java, Go, Kotlin, Swift, Scala, JavaScript, C++, Rust and PHP,
  types (PascalCase names); other languages get strings, calls and numbers.
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

  ````markdown
  ```stage
  dir: LR                              # LR (default) or TB
  Read --> Cache
  Cache -->|hit| Fast[Return fast]     # id[Label] renames; -->|x| labels an edge
  Cache -->|miss| Disk --> Fill --> Fast
  dot Read..Fast 2s                    # a dot loops along the route Read->Fast
  flow Read..Disk                      # marching dashes along a route
  ```
  ````

  (The `#` notes explain the lines; leave them out of a real fence.)

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


### Highlighted words: `==text==`

Two `=` on each side mark a phrase to stand out, in a heading or in body text:

```markdown
## Five things ==to keep==

Every team was green; ==the customer still waited==.
```

It renders as `<span class="hl">…</span>`, so a theme decides the look: the
built-in themes underline it with a band of the accent colour, and a theme can
sweep the band in as the slide arrives. It nests (`==**bold** words==`), stays
on one line, and leaves code, a single `=` and `===` alone. Plain-text copies
of a title (the tab, the teleprompter, search) drop the marks.

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
- The language comes from the file's name (`.cs` → C#, `.csproj`, `.config`,
  `.props`, `.xaml` → XML, `Dockerfile` → dockerfile, `.sh` → bash, `.yml` →
  YAML), then from its first characters (`<?xml` or a tag → XML, `{` or `[` →
  JSON, a `#!` line → bash or Python). `lang=` overrides both, here and on
  `<!-- code: -->`: `<!-- include: ../code/NuGet.config title=NuGet.config lang=xml -->`.
- `si check` reports a missing file, a range past the end, a tag that is not
  found or not closed, and an unknown option.

**Fence attributes.** After the language of a code fence:

````markdown
```java title="Retry.java" hl="3,5-6" linenos start=12
````

`title=` shows in the code block's header bar (documents), `hl=` highlights
lines (counted from the first line shown), `linenos` shows line numbers, from
`start=` (default 1). Copy never copies the numbers.

The language may be left out or written the pandoc way; all of these read
the same: ```` ```xml title="config.csproj" ````, ```` ``` title="config.csproj" ````
(no language: the title's extension names it, `.csproj` → `xml`, `.go` → `go`,
`Dockerfile` → `dockerfile`), ```` ```{.xml title="config.csproj"} ```` and
```` ```xml {title="config.csproj"} ````.

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
(the title), `[source,java,highlight=3;5..6]` and `%linenums`; si's own
fence options go in the same brackets (`[source,go,window=mac,numbers]`);
`<1>` callouts are written the same way.

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
| `a` | draw: a pen layer over the slide (drag, or hold ⌥ Option / Alt and move: the cursor draws, no click; si.conf `draw.hold` changes the key) |
| `d` | light / dark (remembered) |
| `f` | full screen |
| `v` | auto-play: the deck reads its `Notes:` aloud and advances at each `[CLICK]`; again pauses, Esc stops (§11) |
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
  varies the stroke), or hold ⌥ Option (Alt) and move: the cursor draws where
  it goes, no click, so a trackpad draws with one finger (`draw.hold` in
  si.conf picks another key: shift, ctrl, meta, or off); the toolbar picks the colour (accent, red, green, ink) and
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
- **Handouts**: `si build talk.md --handout` writes `handouts/<name>.pdf` for
  the audience: a header (title, author, date) and every slide in its final
  state, eight to a page. By default a handout carries **no speaker text**;
  ask for it by name:

  ```bash
  si build talk.md --handout              # slides only (also: --handout=slides)
  si build talk.md --handout=lines        # each slide with ruled lines for notes
  si build talk.md --handout=points       # each slide with its Points:
  si build talk.md --handout=notes        # each slide with its Notes: script
  ```

  Front matter: `handout:` sets a deck's layout for a plain `--handout`
  (`slides`, `lines`, `points` or `notes`); `handout-url: https://…` prints a
  QR code to that address in the header, so the reader can open the live
  deck. si.conf `handout.paper: letter` changes the paper (A4 by default).
  The index tile of a deck gets a HANDOUT download badge. Needs Chrome, like
  `--pdf`.
- **Video**: `si build talk.md --mp4` films the deck to `<name>.mp4` (1920×1080,
  H.264): every slide and every build step as the audience sees it, each held
  for its share of the slide's `(N min)` (else 3 s), then the slide transition,
  the build step and the canvas morph filmed as they move (30 fps). `--dwell 2`
  holds every view 2 s instead, for a short teaser:

  ```bash
  si build talk.md --mp4 --dwell 2   # talk.mp4: a 2 s look at every slide and step
  ```

  With a recording from the teleprompter's **● Record** button, `--narration`
  holds every view exactly as long as you spoke on it and puts your voice
  under the video (`latest`: the newest recording of that deck; or a
  recording's folder). A view you never showed gets the usual time, and si
  warns when the deck changed since the recording:

  ```bash
  si build talk.md --narration latest   # talk.mp4: the talk, paced and voiced as you gave it
  si rehearse talk.md                   # per slide: planned, spoken, and your words per minute
  ```

  `si rehearse` reads the same recording: each slide's `(N min)` against the
  time you spent on it (`← long` past 1.25×, `← fast` under 0.75×) and the
  words per minute its `Notes:` script implies at that pace. Recordings stay
  on your computer; keep `.si/` out of git.

  No recording? `--voice` narrates the video from the `Notes:` themselves,
  the way auto-play reads them (§11): each view stays while its parts are
  said, a view with nothing to say its `(N min)` share or `--dwell` (needs
  ffmpeg):

  ```bash
  si build talk.md --mp4 --voice kokoro:af_heart   # talk.mp4, narrated by the deck's own script
  ```

  It needs headless Chrome and ffmpeg (`brew install ffmpeg`). si runs the page
  slowed down while it films, so the motion is smooth whatever the machine;
  `--dissolve` films still views that dissolve into each other in 0.4 s
  instead (faster, and what si falls back to when filming the motion fails).
- **Open on a phone**: run `si play <dir> --addr 0.0.0.0:7070` to serve on your
  network; `si play` then prints a LAN URL and a QR code to scan. (The default
  address is loopback-only, so it prints the `--addr` hint instead.)

- **Play terminal** (`` ` ``, backtick): a drawer of real shells on your machine
  for live demos — play-only and loopback-only, never shipped in a built deck.
  Full reference: `docs/terminal.md`.

- **Runbook** (front matter `runbook: runbook.json`): a command cheat-sheet in a
  right drawer of the play terminal. The JSON is a list of categories, each with
  entries (`label`, `description`, `command`, `target: sql|bash`):

  ```json
  { "categories": [ { "name": "Database", "entries": [
      { "label": "Sessions", "description": "who is connected",
        "command": "SELECT pid, state FROM pg_stat_activity;", "target": "sql" } ] } ] }
  ```

  Each command is syntax-highlighted (by `target`) so you can read it at a glance;
  there's a filter box. Every entry has **Copy** (puts the command at the shell
  prompt without running it, and on the clipboard) and **Run** (the same, then
  Enter); clicking the entry itself is Copy, Shift-click Run. A command waits
  until the shell has drawn its prompt, and is sent as a paste where the shell
  supports it (zsh, bash 5.1+), so a multi-line command shows whole and runs only
  on Enter. Like the terminal, it
  appears only in `si play` on a loopback address — a built deck never loads it, so
  rules #2/#7 still hold for shipped HTML. `si check` validates the path and JSON.

  A command may hold **placeholders**: `{{name}}` or `{{name:default}}` (name
  `[A-Za-z_][A-Za-z0-9_-]*`, default any text without `}`). The drawer asks for a
  value for each distinct name (first occurrence and its default win), then Copy
  or Run the command with every occurrence replaced; Esc cancels and types nothing.
  Placeholders are highlighted in the command.

- **Browser windows** (key **w**, `si play` on a loopback address only, never
  in a built deck): web pages in floating windows over the slides, for a
  broker's console or a dashboard on a wide screen. **w** opens a picker with
  the deck's `browser:` links, the recent addresses and a box to type one.
  Drag a window by its title bar, resize it from the bottom-right corner,
  double-click the bar (or **□**) to maximise, **–** to minimise it to a pill in
  the bottom-left dock, **✕** to close; several windows at once, the one you
  touch comes to the front, and their places survive a live reload. Each page
  is shown through a loopback proxy si play starts for its origin, so consoles
  that refuse to be framed (X-Frame-Options, CSP `frame-ancestors`) still
  show. Keys typed in a window never drive the deck.

  ```markdown
  ---
  browser: RabbitMQ=http://localhost:15672, Grafana=http://localhost:3000
  ---
  ```

- **Demos** (front matter `demos: ../demos/`): demo scripts, plain markdown, in a
  second side panel of the play terminal (`▶ Demos`, one panel open at a time).
  A drop-down lists the demos by their `# ` title (else the file name). The panel
  opens on **Commands only**: the headings that hold commands and the runnable
  blocks, so `> Say:` lines and notes stay in the speaker's own window; **Full
  script** shows the whole script (headings, prose, tables, `>` quotes), and the
  choice is remembered. Every
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

### A talk that sets itself up, from anywhere

A talk that shows something live (a broker, a database, a cluster) can set it
up itself. Front matter `setup:`, `start:`, `stop:` and `cleanup:` may each
repeat; `setup:` and `start:` run **in the order written**:

```markdown
---
title: The orders broker, live
setup:   mkdir -p .live && echo 3 > .live/orders     # must finish, exit 0
start:   sh broker.sh as broker · wait for "ready"    # keeps running while you present
setup:   echo 4 > .live/orders                        # after the broker is up
stop:    echo "draining" > .live/state                # when si play quits, the programs still running
cleanup: rm -rf .live                                 # last, after the programs are stopped
---
```

| Key | When | Notes |
|---|---|---|
| `setup:` | before the first page, in order with `start:` | must exit 0 (within 10 minutes; `· within 30m` for longer); runs in the deck's folder (`· in <dir>` for another) |
| `start:` | before the first page, in order with `setup:` | a program kept running while si serves the talk: `as NAME`, `· wait for "text"`, `· wait for port 5672`, `· wait 3s`, `· within 60s`, as in a web demo (§27); its output goes to `<name>.log` in `$SI_RUN_LOGS` |
| `stop:` | when `si play` quits (Ctrl-C too), while the programs still run | |
| `cleanup:` | last, after si has stopped every program (SIGTERM, then SIGKILL after 5 s, the whole process group) | |

- **When.** `si play` and `si serve` run them before they serve the first
  page; meanwhile every page is a **getting ready** page that shows each step
  (waiting, running, ✓, ✗) and opens the talk once they are done. A failed step
  is shown there and in the terminal; si then runs `stop:` and `cleanup:` and
  exits 1. `si build` never runs them (a built deck is a file); `si check`
  checks each line and the pin of each source. A web demo script (§27) has its
  own meaning for these keys.
- **The programs.** The play terminal drawer's shells have `$SI_RUN_LOGS`, the
  folder of the programs' logs (`tail -f $SI_RUN_LOGS/broker.log`); the
  programs run on this machine, so the drawer, the runbook's commands and the
  talk's own `code:` reach them as they would any local program.
- **Sources.** A line may name a script from elsewhere instead of a command,
  with its arguments after it: `setup: github:acme/ops@v1.4/scripts/install-broker.sh --version 3`.
  The sources, their pins and the trust question are those of a web demo (§27,
  Scripts from anywhere): asked before running, once per pin, never silently.
- **A talk from a repo or a URL.** `si play`, `si serve` and `si build` take
  `github:owner/repo@REF/talks/talk.md` (or a folder, `…/talks/`): the repo is
  fetched once at that tag or commit into `~/.si/cache/repos`, its layout kept
  (a deck's `../images/…` still resolve), and served from there. A plain URL
  takes its fingerprint: `si play "https://example.com/talk.md · sha256 9f2a…"`
  (one file; pictures beside it are not fetched). The same trust rules hold:
  the first time, si shows where it comes from, its size and first lines, and
  asks; `--trust` says yes for one run (CI). A talk from a repo that sets
  nothing up just plays.

```bash
si play examples/live/talk.md                    # sets up, presents, cleans up on Ctrl-C
si play github:acme/talks@v2.1/orders/talk.md    # asked once for v2.1
si build github:acme/talks@v2.1/orders/ --trust  # CI: build a pinned talk
```

Example: `examples/live/talk.md` (a stand-in broker, `broker.sh`, started
before the title slide and stopped after the last).

### Studio (`si --web`)

`si --web [dir] [--addr 127.0.0.1:7070] [--no-open]` opens a browser editor for
the decks in `dir`: file tree, markdown editor, live preview and the `si check`
report. It listens on a loopback address only and is never part of a built deck.
See `docs/studio.md`.

```bash
si --web examples --no-open
```

### In the terminal

`si play talk.md --terminal` presents the deck in the terminal itself: no
browser, over SSH, in tmux. Every rule above draws as text and colour from the
same markdown:

- **The theme's colours** (its dark values, or light with `mode: light`): accent,
  muted text, the `==highlight==` band, card colours.
- **Titles of title, section and closing slides** in block letters made of
  half-block characters (falling back to bold text where a letter has no glyph
  or a word does not fit); the big number of a `big-number` slide too.
- **Builds** (`<!-- step -->`, `<!-- steps -->`) reveal one at a time; going back
  to a slide shows it whole.
- **Layouts:** columns (`<!-- column -->`, two-col and split with text left and
  media right) side by side when the terminal is 60 columns or wider, stacked
  below that; `center`, `quote` and `big-number` centred.
- **Code** highlighted with si's own highlighter, in a box with its title,
  `hl=` lines, `linenos` and callouts (❶); **tables** with borders and the
  column alignment; **cards**, **badges** and **logo lockups** as text.
- **Images** (PNG, JPEG, GIF) as real pictures in terminals that can show them
  (Kitty, Ghostty and WezTerm through the Kitty graphics protocol; iTerm2 through
  its inline images), and as half-block pixels everywhere else, inside tmux too
  (`SI_PICTURES=kitty|iterm|halfblock` overrides the guess); **stage diagrams**
  as boxes and arrows, the travelling dot moving along them.
- **Animated canvases** (`<!-- svg: -->`) drawn with characters: boxes, wires,
  text and dots from the SVG's shapes, in the state the deck has reached (the
  same words, `+w` / `-w` and build-step ops as in the browser). The helper
  attributes work (`data-show`, `data-hide`, `data-ok`, `data-alert`,
  `data-flow`, `data-packets`, `data-grow`) and so do the canvas's own CSS state
  rules for `opacity`, `display`, `fill` and `transform` (translate, scaleX).
  They move: packets travel along their wires, flows march, alerts pulse, and a
  part that moves between two states slides there when you press →. Other CSS
  (`offset-path`, counters, filters) shows the end state. SVG images, Mermaid
  diagrams and video show a labelled placeholder: the browser draws them.

Keys are the browser's: → Space `n` next, ← `p` back, `g` / `G` first and
last, `o` overview, `t` restart the timer, `?` keys, `q` quit. The deck reloads
when a file changes, and the slides reflow when the terminal is resized.

`si play talk.md --terminal --notes` in a second terminal (another tmux pane,
or the laptop screen while the projector shows the first) is the
teleprompter: it follows the presenter over a local socket and drives it back
(Space and ← move the slides from the notes). It shows the `Notes:` script with
the `[CLICK]` cues as badges (done ones ticked, the next one highlighted),
slide and total timers against the `(N min)` plan with the pace (on time,
ahead, behind), the clock, and the current and next slide. It scrolls by itself
(`a` on / off, `+` / `-` speed, ↑ ↓ by hand); Tab switches to the `Points:`.
It also follows a deck open in a browser: with `si play` (or `si serve`)
running, `--notes` listens to the server at `--addr` (default
`127.0.0.1:7070`) for the page of the same name, and Space / ← move the browser
deck. Without either it moves on its own.

`si play talk.md --terminal --dump 4 --size 100x30` prints slide 4 with every
build shown (ANSI colours) and exits: for scripts, READMEs and checks.

## 15. Styling: themes and your own CSS

- `theme:` picks a built-in theme (`studio`, `paper`, `mono`, `slate`,
  `twilight`, `forest`, `sunrise`, `neon`, `newsprint`), each with a light and a dark palette. `si theme list`
  prints them with their display names and aliases. Names are matched loosely:
  case is ignored and spaces or underscores count as `-`, so `theme: Twilight`
  and `--theme TWILIGHT` mean `twilight`. An unknown `--theme` is an error.
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

### Make a theme: `extends`, the Theme view, check, export, import

- **`extends`**: a theme file whose first comment starts with
  `si theme: <Name> · extends <built-in>` gets that built-in's tokens first and
  its own after (the file wins), so it only needs the tokens it changes:
  headings, padding, shadows and dark mode come from the built-in. Without the
  header a `.css` theme has nothing underneath, as before. An unknown built-in
  is a `si check` problem.
- **The Studio's Theme view** (`si --web`): ⌘K › *New theme* (or File › New
  Theme… in the Mac app) asks for a name and a built-in to start from and
  creates `themes/<name>.css`; opening any `.css` theme shows it as controls:
  colours light beside dark with a picker, fonts, sizes on sliders (never
  `px`), motion, beside a specimen deck drawn with it, light and dark, at four
  screen sizes, and the contrast checks with **Fix**. Every change is one
  undoable edit of the file, which stays plain CSS (the Text view edits it
  too). *Use in this folder* writes `theme:` into `si.conf`; a `.css` path in
  `si.conf` is relative to `si.conf`.
- **Checks**: `si theme check <file.css>` (or `si check <file.css>`) reports
  unknown tokens and values of the wrong kind (errors), and every
  text/background pair under its WCAG AA minimum in light or dark (warnings,
  each with a colour that passes): body and muted text 4.5:1 on the
  background and on surfaces, text on the accent 4.5:1, accents and status
  colours 3:1.
- **Share**: `si theme export <name|file.css> [-o file]` writes one
  self-contained file (a built-in in full; a file's `url()`s inlined);
  `--format dtcg` writes W3C Design Tokens JSON (`.tokens.json`: colours with
  their dark value in `$extensions`, font families, durations, numbers, `rem`
  and `em` sizes; other tokens are listed, never guessed).
  `si theme import <file.css|file.tokens.json|https://…> [-o dir] [--use]`
  checks a theme (a broken value refuses it), writes it to `themes/<name>.css`
  and, with `--use`, sets it in `si.conf`.
- **From a logo**: `si theme from <logo.png|jpg|gif|svg> [-o dir|file.css]
  [--name N] [--accent '#hex'] [--use]` reads the logo's colours (the pixels of
  a PNG, JPEG or GIF; the `fill`, `stroke` and `stop-color` values of an SVG,
  gradient stops included), leaves out white, black and transparency, and
  writes `themes/<name>.css` extending `studio`: the brand colour is the
  accent, the next one at least 40° away in hue is accent-2 (else a split
  complement), backgrounds, surfaces, lines and text are neutrals tinted with
  the accent's hue, and the status colours are set too, light and dark. Every
  pair the checks measure passes WCAG AA; a colour moved for that is noted in
  the file's header. `--accent` keeps a colour you know and takes the rest from
  the logo; a black-and-white logo needs it. The same logo always gives the
  same theme. In the Studio: *New theme* › **From a logo…**. Fonts stay yours
  (`fonts:` below).
- The index page of `si build` and `si play` follows the decks' theme, a
  `.css` theme included.

```css
/* si theme: Acme · extends paper
   tokens and looks only; edit here or in the Theme view of si Studio */
:root {
  --si-accent: #0b3d91;
  --si-font-heading: "Fraunces", Georgia, serif;
  --si-h1-size: 3.4em;
}
:root[data-si-mode="dark"] {
  --si-accent: #8fb3ff;
}
@media (prefers-color-scheme: dark) {
  :root:not([data-si-mode]) {
    --si-accent: #8fb3ff;
  }
}
```

Use it with `theme: themes/acme.css` in a deck's front matter (a path from
the deck) or in `si.conf`. Load a web font like Fraunces with `fonts:` (below).

A theme from a logo, set as the folder's theme:

```bash
si theme from acme-logo.svg --use        # themes/acme.css + theme: in si.conf
si theme from logo.png --name "Acme" --accent '#e4572e' -o brand/
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

A `.css` theme file without an `extends` header replaces the built-in theme,
so it must set the colour, font and layout tokens itself (`si --export
css-rules` writes a complete, commented starter with every token and its
studio default); with `si theme: Name · extends studio` it sets only what it
changes. The same file
styles decks, articles, guides and labs. To change only a few things on top of
a built-in theme, keep `theme: studio` (or any) and pass the overrides with
`--css`, which lands in the `si.user` layer after the theme.

### Fonts

`fonts:` gives the headings, the body text and the code free web fonts.
They are fetched when the deck or page is built and inlined as `@font-face`
data URIs, so the file still works offline and from `file://`. Put the key in
a deck's or page's front matter. A guide's `_nav.md` or a site's `_site.md`
sets it for every page, and `si.conf` sets it for every deck.

```yaml
fonts: heading=Fraunces:600, body=Inter:400;700, mono=JetBrains Mono
```

| Entry | Font |
|---|---|
| `heading=Fraunces:600` | Google Fonts (the default source), weight 600; one heading weight also sets `--si-heading-weight` |
| `body=bunny:Inter:400;700` | Bunny Fonts (the Google catalogue from a European, no-tracking CDN) |
| `heading=fontshare:Satoshi:700` | Fontshare (Indian Type Foundry's free faces: Satoshi, General Sans, Clash Display …) |
| `mono=fontsource:Fira Code` | Fontsource (Google's and other open-source fonts, via jsDelivr) |
| `heading=fonts/Brand.woff2:600` | a local `.woff2`, `.woff`, `.ttf` or `.otf` file you have the right to embed |

- Roles: `heading` (`--si-font-heading`), `body` (`--si-font-body` and
  `--si-font-prose`) and `mono` (`--si-font-mono`). They override the theme's
  fonts and nothing else; your `--css` still wins over them.
- Weights are 100 to 900 in steps of 100, separated by `;`. The defaults are
  700 for heading, 400;700 for body and 400 for mono. Every weight you list
  adds to the file, typically 20 to 60 KB per weight.
- A split family keeps only the latin and latin-ext character sets unless
  `fonts-subset:` names others: `fonts-subset: latin, greek` (or `cyrillic`,
  `vietnamese`, `cyrillic-ext` …; also in `si.conf`). Each set adds to the file.
- A stylesheet and its files are cached for good: after one online build,
  `--offline` builds the same output. `si check` reports a family or weight
  the source does not have.
- Google, Bunny and Fontsource fonts are under the SIL Open Font License or
  Apache 2.0, which allow embedding. Fontshare fonts are under the ITF Free
  Font License, free for personal and commercial use. A local file is your
  responsibility.
- `fonts: off` turns off fonts that `si.conf` or a `_nav.md` would set.

Example: the site example's `_site.md` sets
`fonts: heading=fonts/fraunces-600.woff2:600` (Fraunces, OFL, licence beside
the file).

## 16. Configuring keys and features (si.conf)

An optional `si.conf` next to the deck (or `--conf <path>`) sets project-wide
defaults without editing each deck. It uses the same `key: value` style as
front matter — no new syntax. Blank lines and `#` comments are ignored.

| Key | Effect |
|---|---|
| `theme: <name>` | default theme when a deck sets no `theme:` and there is no `--theme` |
| `keys.<action>: <key>` | remap a command key |
| `feature.<name>: off` | turn a feature off (its key stops working and its toolbar button is hidden) |
| `handout.paper: letter` | the paper of `si build --handout` PDFs: `a4` (default) or `letter` |
| `draw.hold: shift` | the key you hold in draw mode to draw with the cursor, no click: `alt` (⌥ Option, the default), `shift`, `ctrl`, `meta` (⌘ Command), or `off` |
| `links.<label>: <url>` | a page outside the build that find (`/`) offers, e.g. `links.Team wiki: https://wiki.example.com`; relative paths work too |
| `timer.time: 10,15,30` | break-timer presets (default `5,10,15,20,30,45,60`) |
| `timer.unit: seconds` | preset unit: `minutes` (default) or `seconds` |
| `timer.face: analog` | countdown face: `digital` (default) or `analog` (sport-clock dial) |
| `timer.animation: waves,orbit` | animation(s) behind the timer HUD: `waves`, `orbit`, `bokeh`, or custom names; a comma list gives a toggle |
| `timer.animation.path: ./anim` | directory of custom `<name>.js` / `<name>.svg` animations |
| `voice: kokoro:af_heart` | the voice of `si demo run` and `si voice say` when the script and `--voice` set none (§27) |
| `voice.style: calm, friendly` | the tone, for engines that take one (`openai`), when the script sets no `voice-style:` |
| `demo.with: subtitles, chapters` | what `si demo run` writes besides `<script>.mp4` when `--with` and the script's `with:` say nothing (§27) |
| `voice.Ada: kokoro:af_heart` | auto-play: one speaker's voice in a deck with `speakers:` (front matter `voices:` wins) |
| `autoplay.dwell: 5` | auto-play: seconds on a slide with no `Notes:` and no `(N min)` (default 3) |

Nesting is written as dotted keys (`timer.animation.path`), not YAML.

**Your own si.conf.** Besides the project's, si reads one si.conf of yours:
`~/.si/si.conf` (on every system; `$SI_HOME` moves the whole `~/.si`
folder). It is read first, so the project's file wins key by key; put there
what follows you from project to project (a theme you like, `voice:`).
`~/.si` is si's own folder: your si.conf, logins (`publish.json`), the
certificate authority (`ca/`), running sessions (`sessions/`), Kokoro
(`kokoro/`) and caches (`cache/`). si before v2.43.0 kept them in the system's
config and cache folders; they move into `~/.si` the first time they are
used.

It is also the only place for **API keys**: `voice.elevenlabs.key:` and
`voice.openai.key:`. A project si.conf goes to git with the decks, so a key in
it is ignored, and `si check` warns with the file and line. Do not write them
by hand: `si voice login openai` asks for the key without echo and writes it
there with the file readable by you only (si warns when others can read it);
`si voice logout openai` removes it. A key in the environment
(`OPENAI_API_KEY`, `ELEVENLABS_API_KEY`) wins over the file; si never prints
one.

```
# ~/.si/si.conf — yours, not the project's
theme: slate
voice: kokoro:bf_emma
voice.openai.key: sk-…        # written by si voice login openai
```

Remappable **actions**: `overview`, `presenter`, `draw`, `timer`, `find`,
`mode`, `fullscreen`, `help`, `terminal`, `remote`, `autoplay`. Toggleable **features**: `draw`, `timer`,
`presenter`, `overview`, `find`, `autoplay`.

```
# si.conf — flat key: value, same style as front matter (not YAML)
theme: slate
keys.draw: w          # draw with w instead of a
draw.hold: shift      # in draw mode, hold Shift (not Option) to draw with the cursor
feature.timer: off    # no break-timer HUD for these decks
```

Run `si --export conf` to print a starter `si.conf` with every key and feature
at its default. Theme precedence: `--theme` flag > a deck's `theme:` front
matter > `si.conf` > the built-in default (`studio`). `si play` re-reads
`si.conf` on every request, so edits show up on reload. An unknown key or a bad
value is a warning printed to the terminal (with the file and line), never a
failed build.

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

**Size budgets.** `si check --budget` also fails a page whose built HTML is
over its budget: 1 MB for a page, 1.5 MB for a page that draws a 3D mind map
(three.js is inlined). `budget.*` in `si.conf` sets the limits and turns the
check on without the flag:

```
budget.page: 900 KB     # any page
budget.3d: 1.5 MB       # a page with a 3D mind map
```

```bash
si check talk/ --budget   # talk.md: the built page is 1340 KB, over its budget of 1024 KB
```

Building with `--no-inline-media` (pictures and video beside the page) is the
usual fix.

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
- **`nav:`** in `_nav.md` (or `nav:` in si.conf for every guide) places the
  navigation: `left` (the default), `right`, `top` (a bar over the page whose
  sections open as lists) or `bottom` (a bar docked at the foot of the window,
  opening upwards); add `not-sticky` to let it scroll away, `accordion` (one
  section open at a time; the current page's stays open) or `collapsed`
  (every section closed but the current one), and `numbered`. A book takes
  the same values; a `_site.md` takes the position and `not-sticky` (`left`
  and `right` are a sidebar, `top` and `bottom` a top bar). Section 28 shows
  every value.

  ```markdown
  ---
  title: The order journey
  nav: right · accordion · numbered
  ---
  ```

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

With `app:` in its front matter, a demo script is a web demo that si runs against a web app (§27).


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
| `logo:` | text, or an image path (inlined like any image); an image followed by text (`logo: brand/mark.svg simple-ideas`) shows the picture with the name beside it |
| `favicon:` | the browser tab icon, an image path (inlined); without it pages carry no icon |
| `touch-icon:` | the home-screen icon (`apple-touch-icon`), an image path |
| `og-image:` | the picture link previews show (`og:image`, `twitter:card`), an absolute `https://` URL |
| `cta:` | `Label → page` (or `→ https://…`, `→ #modal`): the highlighted button |
| `version:` | a badge beside the logo, e.g. `v2.5.0 → https://…/releases/tag/v2.5.0` (the link is optional) |
| `title: off` | on a page: no `<h1>` (a home page that opens with its own content) |
| `cta:` on a page | a large button under that page's title and description, same `Label → page` form (a landing page's hero); hidden on paper |
| `unlisted: true` | on a page: build it as a full site page — the same navigator, `_header.md`, `_footer.md`, body shell and theme as every listed page — and let links to it resolve (from the navigator `cta:`, `_footer.md` or another page's body); the only difference is that it is kept out of the navigator menu, with no menu item of its own (a "Talk to us" page a button opens). A page that is in no menu list and is not `unlisted:` is still an orphan: a warning, not built. |

`favicon:`, `touch-icon:` and `og-image:` also work in any deck's or document's
front matter. si never adds an icon or branding of its own to pages you build.

```markdown
---
title: Northwind docs
logo: brand/mark.svg Northwind
favicon: brand/favicon.svg
touch-icon: brand/touch.png
og-image: https://docs.northwind.example/brand/card.png
---
```
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
runtime; nothing is fetched. `badge`, `logo`, `qr` and `cards` also work on slides;
the others, on a slide, are reported as a problem.
Options follow the text after a `·`: `key=value`, or a kind word (`info`,
`ok`, `warn`, `crit`). Any other `· part` stays in the text, so
`<!-- badge: Prepared for Northwind · Session 1 of 5 -->` keeps its dot.

```markdown
<!-- alert: The lab moves to the new cluster on 24 November. · until=2026-11-25 · warn -->
<!-- badge: Pipeline healthy, p95 checkout 410 ms · ok -->
<!-- event: 2026-11-20T17:00 Live: replaying Black Friday 2025 · Save a seat → #seat -->
```

| Plugin | What |
|---|---|
| `alert: text · kind · until=YYYY-MM-DD` | a dismissible band; the reader's dismissal is remembered in that browser; hidden after `until`. Kinds: `info` (default), `ok`, `warn`, `crit`. |
| `badge: text · kind` | a status pill, same kinds; on a slide, the pill paragraph a cover or closing slide carries |
| `cards: 2` | the `###` headings that follow become a grid of cards (2, 3 or 4 columns, or as many as fit): `### Title · tag · colour` (the title may be a link, `### [Dock board](work/dock.md) · Client`), colour one of `teal`, `orange`, `ok`, `crit`, `hi` (the highlighted card); the grid ends at the next `##`, `---` or directive; a `<!-- step -->` before it reveals the whole grid |
| `logo: [image] × Partner · alt=Brand` | a logo lockup: the image (or, with none, the theme's own logo: `--si-brand-logo`), a bar, and the partner's name; on slides and pages |
| `cloud: id · hidden` | a word cloud of the room's short answers under `si play --audience` (`hidden`: on the next click) |
| `poll: id · option · option … · multiple · hidden` | a live poll the room answers on their phones under `si play --audience` (`hidden`: results on the next click); elsewhere the list of options (above) |
| `qr: text or URL · caption=… · style=dots · logo=path · size=large · ec=H` | a QR code (below); `qr: audience` is the live follow-along code of `si play --audience` |
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
| `embed: blog/retries.md#what-we-changed` | that section's text, with a *From …* link; give a heading a fixed id with `## What we changed {#changes}` so the link survives a reworded title |
| `embed: blog/retries.md full` | the whole body |
| `embed: talks/journey.md 1-3` | a deck: its slides, live, in a frame (as `<!-- slides: -->`) |
| `embed: docs/` | a guide, book or site folder: a card with its first pages or chapters |
| `embed: talks/journey.md 1-3 bare` | the same without the caption (`full bare`: the body without its title and *From* line) |

Cards, on a slide or a page:

```markdown
<!-- cards: 2 -->

### High availability · In seconds · teal

One cluster, one site. **Recovers by itself.**

### Disaster recovery · By decision · orange

Two clusters, two sites. **Recovers by decision.**
```

On a cover slide:

```markdown
# Message flows, end to end

<!-- logo: × Northwind · alt=Harbor -->

<!-- badge: Prepared for Northwind · Session 1 of 5 -->
```

### QR codes

`<!-- qr: … -->` draws a QR code for a link or any text, on a slide or a page.
It is an SVG made at build time (nothing is fetched, no script), sized by its
container like any figure; on a slide it fits a square in either orientation.
A web address is also a link, so a reader on a laptop can click it.

```markdown
## Everything from today

<!-- qr: https://simple-ideas.org/rules/ · caption=simple-ideas.org/rules · size=large -->
```

| Option | What |
|---|---|
| `caption=text` | a line under the code |
| `size=small\|medium\|large` | the width (default medium); `--si-qr-size` overrides it |
| `style=dots` (or the word `dots`) | round modules and rounded finder squares; `square` is the default |
| `logo=path/mark.svg` | a picture in the middle; the code switches to error correction H, and a picture too big to read past is a problem (`logo-size=20` is its share of the width, in percent) |
| `ec=L\|M\|Q\|H` | error correction (default M, raised when it costs no size) |

The code is dark on light in every theme (`--si-qr-fg`, default `#111`, on
`--si-qr-bg`, default `#fff`): a light code on a dark ground is one many
phones do not read. To make one as a file, see `si qr` in the manual
(`si qr https://… --png --svg --pdf`).

Start one with `si new mysite --type site` (or `--type website`): `_site.md`,
a home page and an about page. Example: `si --export example --type site`
writes this site; `si serve si-example` serves it.

### Start from a kind

`--kind` (or the kind as the `--type`) starts a website made for one job,
with real content to replace:

```bash
si new marta --type website --kind resume     # or: si new marta --type resume
si new tomas --type portfolio
si new dockside --type landing
```

| Kind | What you get |
|---|---|
| `resume` | `index.md`: name, title, contact lines, experience, skills, education, set to print on A4 or Letter (the navigator and buttons stay off the paper; print the page or save it as PDF from the browser); `projects.md` (cards) and `about.md` |
| `portfolio` | a home with project cards, each with a picture (`images/*.svg`) and a linked title; one `type: article` case study per project under `work/`; `about.md` |
| `landing` | a product page: a hero with a page `cta:` button, features as cards, results as a table, questions as an accordion, a QR code, a `#demo` modal opened by the navigator's `cta:`; `pricing.md` and `_footer.md` |

A landing page's hero is its title, its `description:` and its own `cta:`:

```markdown
---
title: Every van, every door, one screen
description: Dockside plans your loading docks minute by minute.
cta: Start a free 30-day trial → pricing
---
```

## 24. Mind maps

A mind map is a Markdown list of topics under a `# Title`. It is a page of
its own (`type: mindmap`) or a ```` ```mindmap ```` fence inside a slide, an
article, a guide page or any other document. The browser lays it out, draws
the lines, and lets the reader fold branches, move topics, and open linked
maps with a zoom. Without JavaScript (a screen reader, a feed, a print of the
source) the map is the nested list it was written as.

### A mind map page

```markdown
---
type: mindmap
title: The Thursday Promise
description: Every hand-off between "Buy now" and the doorstep.
layout: map
lines: curved
nodes: box
map-theme: auto
watermark: Northwind Logistics
watermark-position: corner
---
# The Thursday Promise

- Checkout
  - Cart & address
  - Payment → payments.md
  - Promise date {badge=Thu}
- Warehouse → warehouse.md
  - Picking
  - Label printed → slides: talk.md#7
- Delivery
  - Dusk departure ![The first vans](img/dusk.jpg)
  - [!sticky] Drivers asked for a later cut-off on Thursdays.
```

`si build` writes one self-contained HTML file per map file (into `--output`
like every other page) and lists it on the index as *N maps*; `si play` and
`si serve` serve it live; `si check` reports its problems. The page fills the
window.

| Front matter | Values | Default |
|---|---|---|
| `type` | `mindmap` | |
| `layout` | `map` (branches on both sides), `right` (a tree to the right), `org` (top-down; leaves stack under their branch), `radial` (rings around the centre) | `map` |
| `lines` | `curved`, `straight`, `orthogonal` (elbows with round corners), `tapered` (thick at the parent, thin at the child) | `curved` |
| `nodes` | `box`, `pill`, `underline` (deep topics are text on a line in their branch colour) | `box` |
| `map-theme` | `auto` (the page theme: `theme:` and light/dark), `harbor` (navy and amber), `blueprint` (blue grid, white lines, monospace), `ink` (black on white), `garden` (greens) | `auto` |
| `watermark` | text drawn on the map (and its exports); a picture stays a page layer | none |
| `watermark-position` | `corner` (`© text` bottom right), `across` (faint diagonal text over the map) | `corner` |
| `view` | `2d`, `3d` (topics as pills on a floor; see *In 3D*) | `2d` |
| `camera` | `iso` (isometric), `perspective` (3D only) | `iso` |
| `floors` | `on` (linked maps hover above their topic), `off` (3D only) | `on` |
| `kind` | `blocks` (pills), `bricks` (studded bricks on a baseplate), `toy` (cartoon shapes on an island); 3D only, see *Kinds* | `blocks` |
| `flow` | `on` (dots run along the lines, from each topic to its children), `off` | 2D `off`, 3D `on` |

`title`, `description`, `theme`, `mode`, `accent`, `fonts`, `lang` work as on
any page. `layout`, `lines`, `nodes`, `map-theme`, `view`, `camera`, `kind`,
`floors` and `flow` in any other file draw a warning: a fence takes them as options
instead.

### Topics

One list item is one topic; a nested list holds its children; the order is
the order on the map. The central topic is the `# Title` (without one, the
page title).

| Write | You get |
|---|---|
| `- Picking` | a topic |
| `- Promise date {badge=Thu}` | a badge after the text (`_` is a space: `{badge=two_days}`) |
| `- Delivery {collapsed}` | the branch starts folded, with a `+3` chip |
| `- Route 7 {pos=40,-12}` | the topic moved by hand (x, y from its automatic place); only this topic moves |
| `- Warehouse {branch-pos=52,104}` | the topic and its whole branch moved by hand |
| `- Dusk departure ![alt](img/dusk.jpg)` | a picture in the topic (resolved and inlined like any image; the alt text names it when the topic has no text) |
| `- [!sticky] Check with ops.` | a sticky note on a dashed line, handwritten |
| a fenced code block inside an item | a code topic, highlighted like any code block |
| `- Payment → payments.md` | `map ›`: opens that mind map file with a zoom (drill-down) |
| `- Refunds → #refunds` | `map ›`: opens the map `# Refunds {#refunds}` of the same page |
| `- Label → slides: talk.md#7` | a `slide 7` chip: opens the deck at that slide |
| `- Docs → https://example.com` | a `link` chip: opens the page in a new tab |
| `- Retry → ../article/retry.md` | a `page` chip: opens that page (`.md` becomes `.html`) |

Options combine: `- Checkout {badge=Thu collapsed} → checkout.md`. `->`
works as well as `→`. A code topic:

````markdown
- Retry with backoff
  ```go
  for try := 1; try <= 5; try++ {
      time.Sleep(backoff(try))
  }
  ```
````

### Many maps

One file per map is the usual structure: link them with `→ other.md`. The
linked mind map files (and the ones they link, cycles included) are inlined
into the page, so drill-down works offline from one HTML file, and a
breadcrumb leads back. One big file works too: separate the maps with `---`
and give each a `# Title {#id}`:

```markdown
# Payment

- Refunds → #refunds

---

# Refunds {#refunds}

- Partial
- Full
```

A map's id is its `{#id}`, or its title as a slug. On a page, `page.html#refunds`
opens that map first.

### In a slide or a document

````markdown
## Where an order goes

```mindmap layout=right lines=tapered theme=harbor
# Order
- Checkout
  - Payment
- Delivery {badge=Thu}
```
````

The fence takes `layout=`, `lines=`, `nodes=`, `theme=`, `view=`, `camera=`,
`floors=`, `kind=` and `flow=` (the same values as the front matter; `theme=`
is the map theme): ```` ```mindmap view=3d camera=perspective ````. Topics, links and drill-down work
as on a page; `→ file.md` inlines that map file. In a slide the map is as tall
as `--si-figure-max`; in a document `--si-mm-height`. The map grows in each time
its slide comes on screen. Two fences with the same title on one page keep
their own ids.

**One branch per click.** On a slide, `steps` (or `reveal=branches`) makes each
first-level branch a build step: → shows them one by one, in 2D and in 3D, and
they count as the slide's steps, so put one `[CLICK]` per branch in `Notes:`:

````markdown
## Where a promise breaks

```mindmap steps flow=on layout=right
# A late parcel
- Payment
  - Bank timeout
- Warehouse
  - One short-staffed shift
- Road
  - A cut-off that leaves too early
```

Notes: (1 min)
A late parcel almost always has one of three causes. [CLICK] Payment… [CLICK] The warehouse… [CLICK] And the road…
````

In a document `steps` is reported (there are no clicks there); every branch shows.

### Reading a map

| Do | What happens |
|---|---|
| click a topic | folds or opens its branch; a topic that only links opens its link |
| click `map ›` | drills into that map with a zoom; the breadcrumb zooms back out |
| drag a topic | moves only that topic; its lines stretch to follow |
| Shift-drag a topic | moves the topic and its branch |
| arrow keys on a focused topic | nudge it (Shift: its branch); Enter or Space acts like a click |
| drag the canvas | pans |
| `+` `−`, the wheel (a page), Ctrl + wheel or a pinch (a map in a slide or a document) | zoom |
| Fit | the whole map |
| Tidy | puts every moved topic back in its automatic place |
| `/` (a page) or Find (any map) | searches every topic, sticky note and code topic of every map (linked and inlined maps too); ↑ ↓ choose, Enter unfolds the path, opens the right map with the breadcrumb, and zooms to the hit (in 3D the camera flies there and the pill lifts). In a slide `/` stays the deck's search: use the Find button |
| Play (a page) | the overview, then one branch per step with the others dimmed; ← → Space PgUp PgDn step, it advances by itself every 3 s, Esc stops |

Moves made in the browser last until the page reloads; write them down as
`{pos=…}` / `{branch-pos=…}` to keep them. On a narrow portrait screen (a
phone) any layout is drawn as `right`, which stays readable.

### In 3D

```markdown
---
type: mindmap
title: The Thursday Promise
view: 3d
camera: iso
floors: on
---
```

`view: 3d` (or `view=3d` on a fence) draws the same maps as rounded pills on a
floor: the central topic is the tallest, deeper topics lower, each a matte
shade of its branch colour with a soft shadow. Lines are cables on the floor
with dots flowing along them (`tapered` draws as `curved`). Pictures, badges,
link chips, sticky notes, code topics and the watermark are printed on top of
the pills; colours and fonts come from the same `--si-mm-*` tokens, so the page
theme, dark mode and the map themes all apply.

| Do (3D) | What happens |
|---|---|
| drag | orbits the camera; right-drag or two fingers pan |
| the wheel (a page), Ctrl + wheel or a pinch (a slide or a document), `+` `−` | zoom |
| hover a pill | lifts it |
| click a pill | a card: *Open map ›* (drill-down), the slide or page link, *Fold*/*Unfold*, *Focus* |
| double-click a `map ›` pill (or one on a floor) | the camera flies into it and the next map grows out of the floor; the breadcrumb flies back |
| the camera button | Isometric (no perspective, the default) or Perspective |
| Fit, Find, Play | as in 2D: Play flies branch by branch with the rest dimmed (← → Esc) |
| **2D** / **3D** | switches the view; the Markdown only sets where it starts |

With `floors: on`, each map a topic links to (`→ payments.md`, `→ #refunds`)
hovers as a glass floor above that topic, joined by a dashed beam, so a map and
the maps it opens read as one building.

#### Kinds

```markdown
---
type: mindmap
title: Payment
view: 3d
kind: toy
---
```

````markdown
```mindmap view=3d kind=bricks
# One safe retry
- Wait thirty seconds {badge=30_s}
- Same idempotency key
- One try only
```
````

| `kind` | Topics | Lines | Ground | Linked maps |
|---|---|---|---|---|
| `blocks` (default) | matte rounded pills, text on top | cables | a soft shadow | glass floors |
| `bricks` | studded plastic bricks snapped to a stud grid; the text is printed on the front face; sticky notes and code are smooth printed tiles | rows of round tiles | a studded baseplate | small studded plates |
| `toy` | cartoon shapes with ink outlines and three-tone shading: a round drum for the central topic, a hexagon for a topic that opens another map, pills for the rest, flat cards for notes and code | light roads | a floating grass island with trees on its rim (never on a topic) and clouds | small floating islands |

The **kind** button in the 3D tools cycles Blocks → Bricks → Toy; the Markdown
only sets the start. `bricks` and `blocks` take their colours from the
`--si-mm-*` tokens; `toy` keeps its own candy palette and turns to a night
scene on a dark page or a dark map theme. Search, Play (the dimmed branches
include studs and outlines), drill-down, the cards and the flow dots work the
same in every kind.

The 3D view uses three.js (r128, MIT), inlined into the page like everything
else (no network): about 620 KB, only on pages that have a map starting in 3D.
Pages with 2D maps only stay small and show no 3D button. A folder or site
with several 3D maps can carry three.js once: `si build maps/ --share-assets`
(and `--no-inline-media`, which already puts files beside the pages) writes it
to `_si/three.min.js` next to the pages and links it, so each 3D page is about
600 KB smaller (the example maps: 2.1 MB → 1.5 MB); those pages then need the
`_si/` folder beside them. Without the flag every page stays one
self-contained file. A browser without
WebGL shows the map in 2D with a short note. Reduced motion skips the grow-in,
the flow and the camera flights.

On a phone held upright, the 3D view starts closer, on the central topic and
its first ring of branches; pan, zoom or Fit for the rest. Its **List** tool
shows the map as the nested list it was written as, and **Map** brings the 3D
view back.

### Looks

Branch colours come from `--si-mm-1` … `--si-mm-5` (by default the theme's
accent, second accent, ok, warn and a mix of accent and crit), so every
built-in theme and dark mode work. A `map-theme` (or `theme=` on a fence) sets
every `--si-mm-*` token at once; your CSS can set any of them:

```css
.si-mindmap { --si-mm-1: #0f766e; --si-mm-grid: rgb(0 0 0 / .05); }
.si-mindmap { --si-mm-hand: "Caveat", cursive; }  /* load it with fonts: google:Caveat */
```

The tokens (`si --export css-rules` lists them all): `--si-mm-bg`,
`--si-mm-fg`, `--si-mm-muted`, `--si-mm-surface`, `--si-mm-line`,
`--si-mm-accent`, `--si-mm-on-accent`, `--si-mm-1` … `--si-mm-5`,
`--si-mm-font`, `--si-mm-mono`, `--si-mm-hand`, `--si-mm-grid`,
`--si-mm-sticky`, `--si-mm-sticky-fg`, `--si-mm-height`.

### Editing in the Studio

`si --web` (and the desktop app, which runs it) opens a mind map page in the
**map editor**: the map itself, drawn by the same runtime, with a panel for
the selected topic and one for the page. Text / Map in the bar above switches
between the Markdown and the map; Map is the default for a mind map page, and
a new map from New → Mind map opens there. In any other file, put the cursor
in a ```` ```mindmap ```` fence and press Map in the bar that appears.

The editor's 2D / 3D button edits in 3D too: click a pill to select it (it
stands lifted), and Tab, Enter, Delete, Space, F2 or typing work as in 2D
(renaming opens a box at the top of the canvas). Moving a topic to another
parent is done in 2D or through the topic panel's "what it opens" fields; in 3D
there is no drag to reparent.

| Do | What it writes |
|---|---|
| click a topic | selects it (↑ ↓ ← → move the selection; Esc clears it) |
| Tab / Enter | a child / a sibling, named *New topic*, selected |
| F2, a double-click, or start typing | renames the topic in place (Enter keeps, Esc cancels) |
| Delete or Backspace | removes the topic and its branch |
| drag a topic onto another | makes it that topic's last child; dropped on the top or bottom edge, it goes before or after it |
| Alt + ↑ / ↓ | moves the topic up or down among its siblings |
| Shift-drag / Alt-drag | places the topic (`{pos=x,y}`) / its branch (`{branch-pos=x,y}`) freely |
| Space | folds or opens the branch (`{collapsed}`) |
| Tidy | removes every `pos=` and `branch-pos=` of the map |
| ⌘-click / Ctrl-click / Shift-click | adds a topic to the selection, or takes it out |
| ⌘C / ⌘X | copies / cuts the selected branches as a Markdown list (`- Checkout` / `  - Cart & address` …) to the clipboard |
| ⌘V | pastes a Markdown list (a copy from any map or file, or a list typed anywhere) under the selected topic |
| ⌘Z / Ctrl+Z, ⇧⌘Z / Ctrl+Y | undo / redo (the editor's own history) |

The topic panel edits the text, the badge, what it opens (`#map`, a mind map
file, `slides: deck.md#N`, a URL; the list suggests them), a picture from the
workspace and its alt text, sticky note, folded, and a code topic's language
and code (Add code makes one). The page panel sets every front matter key of
the table above (on a fence: its options), switches between the maps of the
file, and adds a map (`---` and `# New map {#new-map}`). Every change is
written back into the Markdown by si itself, as one undoable change, and saved
like any edit; the rest of the file (front matter, comments, other text) is
kept as it was. Topics of a linked map that comes from another file show, but
are edited in their own file.

### Exports

```bash
si build map.md --svg              # every map: map-<id>.svg (standalone, fonts inside)
si build map.md --png --scale 3    # map-<id>.png at 3× (default 2×)
si build map.md --pdf              # pdfs/map.pdf: one page per map, sized to it
si build map.md --mp4              # map.mp4: the first map's Play, branch by branch (needs ffmpeg)
```

Exports draw the 2D map (a `view: 3d` map too), with its theme, lines, moved
topics and watermark; the PDF also has the linked maps, and a `map ›` topic
there jumps to its page. They need headless Chrome, like `--pdf` for decks.
`--mp4` films Play: the overview, then each first-level branch with the rest
dimmed (2.5 s each, a short zoom between), and the overview again; it also
needs ffmpeg (`brew install ffmpeg`), an optional tool like Chrome. A
`view: 3d` page is filmed in 3D: each Play view is rendered by Chrome's
software WebGL in the page's kind, camera and theme, held 2.5 s, with the
camera's flight to the next branch filmed as it moves (`--dissolve`: still
views dissolving into each other).

### Other mind map tools

```bash
si mindmap import plan.opml          # → plan.md (also .mm FreeMind, .xmind XMind, .mmd Mermaid)
si mindmap import plan.xmind -o maps/plan.md
si mindmap export map.md --to xmind  # → map.xmind (opml, freemind, xmind, mermaid)
si mindmap export map.md --to mermaid --map refunds -o -   # one map, to stdout
```

Import keeps the topics, their order and web links; every XMind sheet becomes
a map. Export writes one map (the first, or `--map id`); XMind takes every
map, one sheet each. Pictures, badges, sticky notes and code stay in si.
Import never overwrites a file.

### In the terminal

`si play talk.md --terminal` and `si --terminal` draw a mind map (a fence on a
slide, or a mind map page in the preview) as a tree: `✲` the central topic,
`├─ └─` its branches, `→ target` for links, `[badge]`, `▨` a picture, `✎` a
sticky note, `‹go›` a code topic. In the terminal Studio, focus the preview
(Tab) to edit the tree: ↑ ↓ pick a topic, Enter renames, `c` adds a child, `o`
a sibling, `x` deletes, `[` `]` move it; `e` still opens the file in `$EDITOR`.

### What si check reports

A missing file (`→ other.md`, a picture), a map id that does not exist
(`→ #ghost`), an unknown topic option (`{size=3}`), a `pos=` without two
numbers, an unknown fence option or value, `steps` outside a slide, a map with
no topics, and unknown front matter values. A slide with `steps` counts one
step per branch, so its `[CLICK]` cues are checked like any build.

Example: `examples/mindmap/` (three linked maps: `thursday.md` in 3D with
its linked maps as floors, `payments.md` with two maps in one file,
`warehouse.md` in 2D, `org` layout with `orthogonal` lines), and the article
`examples/article/retry.md` has a fence.
`si new ideas --type mindmap` writes a starter (`--type mindmap3d` one that starts in 3D).

## 25. Code windows and code shows

A fenced code block with `window=` is drawn as a window: macOS (traffic
lights and a centred title), Windows (title on the left, minimise, maximise,
close), a terminal (a tab; a file gets a `$ cat <title>` line above it, while a
shell block, `sh` `bash` `zsh` `console` `powershell`, shows only its own
lines, so write the prompt yourself: `$ go test ./...`) or `none` (a plain rounded
card), optionally on a backdrop, with numbered callouts you place by hand. A
run of code blocks with the same `id=` is a **show**: one window whose steps
morph into each other (the lines that stay glide to their new place, new lines
fade in, removed ones fade out). It works in every document type, on slides and
in mind maps.

````markdown
```go id=retry title="retry.go" window=mac theme=night backdrop=dusk numbers step="One attempt"
func send(order Order) error {
	resp, err := client.Post(url, order)
	if err != nil {
		return err
	}
	return check(resp)
}
```
<!-- callout: 2 One attempt: a dropped connection loses the order. at=60,-54 -->

```go id=retry title="retry.go" window=mac theme=night backdrop=dusk numbers step="Retry five times" hl=3
func send(order Order) error {
	var last error
	for try := 1; try <= 5; try++ {
		resp, err := client.Post(url, order)
		if err == nil {
			return check(resp)
		}
		last = err
	}
	return last
}
```
<!-- callout: 3 Five tries, back to back. -->
````

### Options (on the fence line)

| Option | Values | What it does |
|---|---|---|
| `window=` | `mac`, `windows`, `terminal`, `none` | turns the block into a window; without `window=` or `id=` nothing changes |
| `title=` | text (quote it when it has spaces) | the title bar: `title="PaymentClient.cs - Gateway"` |
| `theme=` | `auto` (default), `night`, `paper`, `dusk` | `auto` uses the deck or page colours (light and dark); the others are code-window looks |
| `backdrop=` | `none` (default), `dusk`, `ocean`, `mint`, `graphite` | a gradient behind the window, with room around it |
| `numbers` | (a flag; `linenos` works too) | line numbers in a gutter |
| `hl=` | `3`, `3,5-7` | highlighted lines of this step |
| `id=` | a name | blocks with the same id in a row are one show; on slides, the same id on consecutive slides morphs |
| `step=` | text | the step's name, shown under the window (`2 / 4 · Back off`) and in the Studio |
| `transition=` | `morph` (default), `fade`, `slide` | how a show goes from step to step |

```xml window=windows title="Gateway.csproj" theme=paper
<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
</Project>
```

```sh window=terminal title="checkout" backdrop=graphite
$ go test ./payments -count=100
ok  	payments	2.41s
```

An unknown value of `window=`, `theme=`, `backdrop=` or `transition=` is a
`si check` problem.

### Callouts

`<!-- callout: LINE text -->` right after the block (blank lines between are
fine, one per line) puts a numbered dot at the end of that code line and a
note beside the window, joined by a dashed wire. The text takes inline
Markdown. `at=dx,dy` is where the note sits, in pixels from the end of the
line; without it the note goes to the right of the window. Drag a note to move
it: in `si play` and in the Studio's Window view the new place is written back
into the Markdown as `at=` (si play finds the callout by its line and text, in
the page's file or a file it includes); on a built page the move is local.

```markdown
<!-- callout: 8 Declines are final: never retry a refused card. at=70,-64 -->
<!-- callout: 12 Jitter spreads the retries of a thousand orders. -->
```

On a narrow screen (a phone) the notes move under the window as a numbered
list; the dots stay on the lines. `si check` reports a callout on a line the
block does not have and a callout with no window above it.

### Shows in documents, on slides, in mind maps

- **Documents** (articles, guides, labs, sites): a show is one window with
  `← →` and Play under it (Play steps every 2.6 s).
- **Slides:** several same-id blocks on one slide are build steps, one click
  each (write one `[CLICK]` per extra step in `Notes:`); the same id on
  consecutive slides morphs as the slide changes, without the slide's fade, like
  an animated canvas.
- **Mind maps:** a code topic takes `window=` and `title=` and gets a title bar
  (macOS lights or the title).
- Without JavaScript every step is a plain code block with its callouts as a
  list.

### Exports

```bash
si code retry.md --png --scale 2   # retry-retry-1.png … one per step, backdrop and callouts included
si code retry.md --svg             # retry-retry-1.svg … (fonts embedded)
si code retry.md --gif             # retry-retry.gif, the whole show: 2.2 s per step, a short morph between
si code retry.md --mp4             # retry-retry.mp4, the same show as H.264 video (needs ffmpeg)
si code talk.md --id charge --gif -o out/   # one show of a deck (its id), into out/
```

Files are named `<file>-<id>-<step>`; a window without an id is
`window-<n>`, counting the file's windows from 1. A show split across
slides exports as one show. Exports need Chrome or Chromium (headless). The
GIF uses one shared 256-colour palette, and the morph is drawn as five frames
between steps. The MP4 (`--mp4`, at `--scale`, 30 fps, H.264, yuv420p,
faststart: it plays in browsers and posts to social sites) needs ffmpeg on the
PATH (`brew install ffmpeg`, `apt install ffmpeg`); si says so when it is
missing.

### In the Studio and the terminal

Put the cursor in a code block and choose **Window** in the bar above it: a
preview of the window and panels for its look (window, theme, backdrop,
transition, title, line numbers), its steps (name, order, add a step as a copy,
delete, and each step's code in a small editor) and its callouts (add one on a
line, edit, delete, drag in the preview to place it), plus PNG, SVG and GIF
exports next to the file. The editor completes code block options after
```` ```go ```` (`window=`, `title=`, `theme=`, `backdrop=`, `numbers`, `hl=`,
`id=`, `step=`, `transition=` and their values), mind map options after
```` ```mindmap ````, `title=` and `lang=` after an include's path, and the
block's line numbers after `<!-- callout: `. Every change is
written into the Markdown as one undoable edit. `si play --terminal` shows a
window's title (`● ● ●  retry.go` for `mac`) in its box and the callouts as
numbered notes; a show advances one step per build (→), with the step in the
box title (`retry.go · 2/4 Back off`) and that step's callouts, and the
`--notes` teleprompter counts them like any build. In the terminal Studio
(`si --terminal`), `:` offers every code block's window style, line numbers
and title, written into the fence line the same way.

Examples: `examples/code/retry.md` (a four-step show on a backdrop, a Windows
window with C# and a `.csproj`, a terminal), the showcase slide "The retry
rule" (two steps, one click), and a terminal window in
`examples/article/retry.md`.

## 26. Publishing

`si publish` posts a deck to LinkedIn as a document post (the deck as PDF,
which the feed shows as a swipeable carousel) and an article to LinkedIn as a
link post, or to dev.to, Hashnode, Ghost or WordPress. It is a **dry run** that prints the exact
requests unless you add `--yes`.

```markdown
---
title: The Thursday Promise
description: How one order crosses five hand-offs and still arrives on Thursday.
linkedin: Our checkout keeps a promise across five hand-offs. Here is how, in 22 slides. #logistics #retries
tags: go, reliability, payments
canonical: https://northwind.dev/talks/thursday
---
```

```bash
si publish talk.md --to linkedin                 # dry run
si publish talk.md --to linkedin --yes           # renders the PDF, uploads, posts
si publish article.md --to devto --base-url https://northwind.dev/retry --yes   # a draft; --public publishes
si publish article.md --to linkedin --url https://northwind.dev/retry --yes
si publish article.md --to ghost --yes           # a draft on your Ghost site; --public publishes
si publish article.md --to wordpress --public --yes
si publish article.md --to linkedin --copy       # rich text for LinkedIn's article editor
si publish login linkedin --client-id ID --client-secret SECRET   # or --token
si publish logout
```

- The post text is `linkedin:`, else `description:`, else the title.
- An article's body is its Markdown with includes expanded and the title
  heading dropped; relative links and images become absolute against
  `--base-url` (si warns about each relative image without it).
- Ghost and WordPress get the article as HTML (si renders the Markdown).
- `tags:` go to dev.to (the first four), Hashnode and Ghost (WordPress wants
  tag ids, so add them there); `canonical:` (or
  `--canonical`) becomes `canonical_url` / `originalArticleURL`, and is the
  link of a LinkedIn link post when `--url` is not given.
- A LinkedIn link post carries a picture: `cover:` (a PNG, JPEG or GIF next
  to the article, e.g. `cover: ../assets/dusk.jpg`), else the article's first
  local picture. si uploads it through LinkedIn's Images API first; the dry
  run shows that request and the `thumbnail` it attaches.
- Logins are stored in your config dir with mode 0600, never in the repo.
  LinkedIn logins last 60 days; the Hashnode API needs a Pro plan. Ghost:
  `si publish login ghost --url https://site --token <Admin API key>`;
  WordPress: `si publish login wordpress --url https://site --user you
  --token "<application password>"`.

Details, scopes and the exact requests: `docs/publish.md`.


## 27. Web demos

A demo script (§22) with an `app:` in its front matter is a **web demo**: si
opens the web app in headless Chrome and performs each feature the way a
person would, step by step, checking what the page shows. `si demo test`
runs it as a test, with a report and an exit code for CI; `si demo run`
performs the same steps on film and narrates them: one video per feature, one
with chapters, captions and the report.

```markdown
---
type: demo
title: Northwind Shop, from sign-in to receipt
app: app                # a URL (a ?token=… is kept), or the app's folder next to the script
viewport: desktop       # 1440x900; also phone (400x800), tablet (820x1180), WxH
timeout: 5s             # how long a step waits for its element
setup: ./seed.sh        # optional: run before each feature, in the script's folder
voice: kokoro:af_heart  # si demo run: engine:voice (see Voices below)
pronounce: si = ess eye # what the voice says for a word (captions keep the word)
---

## Pay for an order

What it does: a customer picks Express shipping and pays with a card.

Say: Sign in with the demo account.
- go /
- type "demo" into "Username"
- type $SHOP_PASSWORD into "Password"
- click "Sign in"
- expect text "Welcome back, demo"

Say: A test card goes through in a second.
- click "Order #4711"
- select "Express" in "Shipping"
- check "Gift wrap"
- click role=button "Pay"
- type "4242 4242 4242 4242" into "Card number"
- click "Confirm"
- wait for "Paid"
- expect url /receipt.html?order=4711
- expect no errors
```

- Each `##` heading is a **feature**: one test case, run in a fresh browser
  (no cookies or storage from the one before), after `setup:` and with the
  app's first page open; a feature whose first page step is a `go` opens on a
  plain page instead, so the video goes straight to that page. A folder `app:`
  never lists its files: a folder without `index.html` is a plain page (show
  files in a `terminal` with `ls` when they matter).
- `What it does:` describes the feature (the report's summary). Each `Say:`
  line is the narration of the steps under it, up to the next `Say:`.
  `Notes:` and `Points:` keep their usual meaning and are not steps.
- Steps are list items. A step waits up to `timeout:` for its element or
  condition, then fails with what it expected, what it found and the closest
  elements it saw.

| Step | What it does |
|---|---|
| `go /path` (or a full URL) | open a page of the app and wait for it to load |
| `click "Pay"` | a real mouse click at the element's centre |
| `type "text" into "Field"` · `type $VAR into "Field"` | replace the field's text; `$VAR` comes from the environment and is never printed (`••••` in the report) |
| `type "text"` | type where the focus is, replacing nothing: after a `click` in an editor, `press End`, then `type ", live"`. Inside quotes only `\"` and `\\` are escapes, so a shell command's `\n` stays as written |
| `press Enter` · `press cmd+k` | a key or a shortcut (alt, ctrl, cmd, shift) |
| `select "Express" in "Shipping"` | an option of a `<select>` (or a listbox) |
| `check "Gift wrap"` · `uncheck …` | a checkbox or switch |
| `hover "Help"` · `scroll to "Footer"` | move the mouse over it; scroll it into view |
| `wait 1s` · `wait for "Saved"` | a pause; or until something appears (`· within 40s` for longer than `timeout:`, also in a film, where steps otherwise give up after 3 s) |
| `note "Username: demo" at "Username"` | a callout in the video: a bubble pointing at the element, ringed, on screen to the end of its `Say:` block (`· for 3s` for less; `· position top\|right\|bottom\|left`, else where it fits; no `at` = a note at the top); a test checks its element is there |
| `expect text "Paid"` | the page shows these words |
| `expect url /orders/4711/receipt` | the address (a path, or a full URL; a `?…` or `#…` after it is fine) |
| `expect "Pay" disabled` | `visible`, `hidden`, `enabled`, `disabled`, `checked`, `unchecked` |
| `expect no errors` | no console errors and no failed requests (HTTP 400 and above) so far |
| `screenshot receipt` | a named picture in the report |
| `zoom to "Total"` · `pan to "…"` · `zoom out` | camera moves in the video (see Transitions, camera, spotlight, windows and drag below) |
| `spotlight "Pay"` (alias `focus`) | dims the rest of the page around the element; `· circle`, `· blur`, `· label "…"`, `· for 3s`, `spotlight off` |
| `window "si-presenter"` · `window 2` · `window main` | perform and film a popup or tab the app opened, then come back |
| `drag from "A" to "B"` · `drag by 300,80` | a real drag, the drawn cursor gliding with the button held |

**Elements** are found the way a person names them: `"Pay"` matches visible
text, a `<label>`, an `aria-label`, a placeholder, a title or an alt text
(exact first, then any case, then a part). `#id` and `[data-testid=pay]` are
CSS selectors; `role=button "Pay"` narrows by role. Visible, enabled
elements come first; text inside a button means the button.

```bash
SHOP_PASSWORD=shop-demo si demo test shop.md
si demo test shop.md --feature "Pay for an order" --junit junit.xml -o out/
```

```text
Pay for an order
  ✓ go /  0.06s
  ✓ type "demo" into "Username"  0.01s
  ✓ type $SHOP_PASSWORD into "Password"  0.02s
  ✗ expect text "10% off"  5.07s
      expected: the page shows "10% off"
      found:    the page shows: Northwind Shop ? Order #4711 … That coupon is not valid. …
si demo: 0 passed, 1 failed in 6.9s; report: demo-out/shop/report.html
```

- `si demo test` writes `report.html` (one page, screenshots inlined) and
  `report.json` into `demo-out/<script>/`, exits 1 when a feature failed,
  and with `--junit file.xml` writes JUnit XML (one test case per feature).
  `--viewport phone|tablet|WxH` overrides the script's size (both commands).
- A failed step stops its feature; the report shows expected and found, the
  closest elements, a screenshot, and the page's console errors and failed
  requests.
- `si check` reports a bad step with its line, and warns about a `$VAR` that
  is not set. The script can also be AsciiDoc, or JSON for tools:
  `{"type": "demo", "app": "…", "features": [{"name": "…", "what": "…",
  "blocks": [{"say": "…", "steps": ["go /", "click \"Pay\""]}]}]}`.
- Needs Google Chrome or Chromium (`si doctor`).

### The videos: `si demo run`

```bash
SHOP_PASSWORD=shop-demo si demo run shop.md
si demo run shop.md --feature "Pay for an order" --viewport phone
si demo run shop.md --voice system:Samantha -o videos/
si demo run broker.md --fast-forward 5s --with captions,subtitles
si demo run shop.md --with clips,report --with readme
```

The same steps, filmed: a drawn cursor glides to each element and clicks
ripple, `note` steps point at elements, text is typed at a person's pace, and a
field typed from a `$VAR` is blurred. **The narration sets the pace:** every
`Say:` line is voiced first and measured, the block's steps are spread over its
words, and a block whose steps need longer than its words waits for them. A
failing step ends that feature's clip there (in a run a step waits at most 3 s)
and goes into the report; the other features still run.

**One file by default:** `demo-out/<script>/<script>.mp4` (or `-o dir`), every
feature after a title card (the feature and its `What it does:`, as written;
a `` `code` `` span there shows in the accent colour, or `--si-code`, with no
backticks, and `` \` `` is a literal backtick), picture and
voice (H.264 + AAC, 30 fps), a chapter per feature embedded. The run still
checks every step, prints ✓ / ✗ and exits 1 when one failed. Anything else is
asked for with `--with`, a comma list that may repeat (`--with captions
--with subtitles` is `--with captions,subtitles`):

| `--with` | Writes |
|---|---|
| `captions` | `<script>.captions.mp4`: picture, voice and captions; the picture shrinks (aspect kept) into the top of the same frame, centered on a near-black backdrop, and the captions (white, two lines at most) sit in a band under it, so they never cover the picture |
| `silent` | `<script>.silent.mp4`: the picture alone, no sound (chapters kept) |
| `audio` | `<script>.m4a`: the narration alone, on the video's clock |
| `subtitles` | `<script>.srt` + `<script>.vtt`: the `Say:` lines timed, SubRip, and WebVTT for a web page's `<track>` |
| `transcript` | `<script>.txt`: each feature's name, then what its `Say:` lines say as one paragraph (the words as written, not as `pronounce:` makes the voice say them) |
| `chapters` | `chapters.txt`: YouTube chapter lines, `0:00 Sign in` |
| `clips` | the same per feature: `<feature>.mp4` always, plus `<feature>.captions.mp4`, `.silent.mp4`, `.m4a`, `.srt`, `.vtt` for the kinds asked for |
| `report` | `report.html` + `report.json` (and `shots/`): the run, linked to the videos it made |
| `readme` | `README.md` + `README.html`: the files made, the video in the page, the script, how it was made |
| `all` | all of the above |

```markdown
---
type: demo
app: app
with: captions, subtitles
---
```

`with:` in the script's front matter, or `demo.with:` in a si.conf (§16), sets
what a run makes when `--with` is not given: `--with` (or `--captions`, the
same as `--with captions`), else `with:`, else the si.conf beside the script,
else your own si.conf, else nothing. `si check` reports an unknown name with
its line. Only what is asked for is made: a run without captions renders no
captions, and the work in between stays in a temporary folder that goes away.
Captions, audio, subtitles and the transcript need `Say:` lines; a feature
without them has none. `<feature>` is the heading in lower case with dashes
(`pay-for-an-order`). `--no-captions`, from v2.38.0, is gone: captions are
made only when asked for.

**Long waits play faster.** A step that waits (`run … · wait`, `stop … ·
wait`, `wait for output`, `expect output … within`, `wait for "…"`, `start … ·
wait for …`) plays at speed for its first 3 s; the rest plays faster, so it
lasts 2 s at most, with a ⏩ 6× badge at the top right. The narration never
speeds up: a stretch under a `Say:` voice stays at speed, and the captions and
chapters are timed on the faster video. `--fast-forward 5s` changes the 3 s;
`--no-fast-forward` keeps every wait at speed.

**A region, not the whole page.** By default a clip is the whole viewport. To
film one feature of the page — a console's Agents panel, the topology lanes —
crop every clip to it:

```bash
si demo run console.md --region "#agents-panel"
si demo run console.md --crop 40,80,900,600
```

- `--region "<css selector>"` crops each feature's clip to that element's
  bounding box, measured on the page (if it moved, its box at the end of the
  take is used). The drawn cursor, the clicks and the `note` bubbles still play
  as before; only the output frame is cropped, so the clip is the element's
  size. `--crop x,y,w,h` crops to a fixed box in pixels instead (no element).
  Give one or the other, not both. A `--region` that matches no element fails
  the clip with a clear error.
- Different features can crop to different regions in one run: a `Region:` line
  under a `##` feature (beside its `What it does:`) crops that feature's clip,
  and `region:` in the front matter is the default for the features that have
  no `Region:` of their own. Quote a selector that has spaces (`region:
  ".lane .agents"`): in front matter, a space followed by `#` starts a comment,
  so an unquoted descendant selector would be cut short.
- Precedence, most specific first: a feature's `Region:`, then `--region` /
  `--crop`, then the front matter `region:`, then the whole page.
- The captions variant shrinks the cropped region (its aspect kept) into the
  top of the frame with the captions in the band below, as it does the whole
  page. In `<script>.mp4` the region clips are centered and letterboxed to the
  viewport so the features join. The `screenshot` step crops to the same region
  (it keeps the whole page only if the region cannot be measured). The
  fast-forward ⏩ badge sits at the frame's top right, so a region that does not
  include that corner does not show it.

```markdown
---
type: demo
app: app
region: "#console"          # the default region for every feature
---

## Agents fill the registry

What it does: the agents panel fills as workers register.
Region: ".agents-lane"      # this feature crops to the lane instead

Say: Three workers register and the panel lists them.
- go /
- expect text "3 agents"
```

**Voices.** `voice: engine:name` in front matter, or `--voice` (a Kokoro voice name alone, `--voice af_heart`, means `kokoro:af_heart`):

| Engine | What | Default voice |
|---|---|---|
| `kokoro` | open source (Apache 2.0), local, natural: si's own voice | `af_heart` |
| `system` | the computer's own: `say` on macOS, espeak-ng on Linux, Windows' voices | the system's (`system:Samantha`) |
| `elevenlabs` | cloud, an API key (`si voice login elevenlabs`) | a voice id (`elevenlabs:21m00Tcm4TlvDq8ikWAM`) |
| `openai` | cloud, an API key (`si voice login openai`); `gpt-4o-mini-tts`, a tone from `voice-style: calm, friendly` | `alloy` |

Which voice: `--voice`, else the script's `voice:`, else `voice:` in the
`si.conf` beside the script, else `voice:` in your own si.conf (§16), else
`kokoro:af_heart` when Kokoro is installed, else `system`. The tone likewise:
`voice-style:` in the script, else `voice.style:` in a si.conf.

Kokoro installs with one command:

```
si voice install kokoro
```

It makes a Python venv with `kokoro-onnx` in the folder `si doctor` names
(`~/.si/kokoro`, or `SI_KOKORO_DIR`) and downloads the model files
`kokoro-v1.0.onnx` and `voices-v1.0.bin` (about 350 MB, checked by SHA-256).
It needs Python 3.10 to 3.13 and changes nothing else on the computer; run it
again to repair, `si voice uninstall kokoro` to remove it. si uses that venv
on its own (`SI_KOKORO_PYTHON` picks another Python), and when your
`~/.si/si.conf` names no voice yet it writes `voice: kokoro:af_heart` there,
so every demo, auto-play and `si voice say` uses Kokoro by default.

The cloud engines need an API key. `si voice login openai` (or `elevenlabs`)
asks for it without echo and keeps it in your own si.conf, readable by you
only; `si voice logout openai` removes it. `OPENAI_API_KEY` and
`ELEVENLABS_API_KEY` in the environment still win. A key is never read from a
project's si.conf, which goes to git (§16).

Try a voice before a take: `si voice say "Every order, one journey" --voice
kokoro:bf_emma` says the line aloud (`-o line.wav` saves it), and `si voice list
kokoro` lists the voices. Each line is cached by its engine, voice, style and
words (in the user cache folder), so a re-run only voices the lines that
changed; a cloud call sends nothing but the `Say:` text.

`pronounce: word = how to say it` (several lines, or pairs split by `;`)
changes what the voice says, whole words in any case; the captions keep the
written word:

```markdown
pronounce: si = ess eye
pronounce: kubectl = cube control; YAML = yammel
```

`si demo run` needs Chrome and ffmpeg, and the voice's engine (`si doctor`).

### Terminals and programs

Many apps are a terminal and a web page together: start a broker, send it a
message, see it in the console. A web demo can start programs in the
background, type commands into a terminal the viewer sees, check what they
print, and switch between the terminal and the browser.

```markdown
---
type: demo
title: A message broker, from the terminal to its console
app: http://127.0.0.1:7310
shell: bash
start: go run ./broker serve --port 7310 as broker · wait for port 7310 · within 120s
setup: go run ./broker reset --port 7310
stop: go run ./broker reset --port 7310
---

## Send a message from the terminal

What it does: a client sends one message, and the console shows it waiting.

Say: From a terminal, we send the broker one message.
- terminal "Client" · backdrop=ocean
- run "go run ./broker send hello from the terminal" · wait
- expect output "sent 1 message"
- note "One message, sent" at line "sent 1 message"

Say: The broker's console shows it, waiting in the queue.
- browser
- expect text "1 message waiting"

## Watch a workload fill the queue

What it does: a load generator sends thirty messages while the console counts them.

Say: A load generator sends thirty messages.
- start "go run ./broker load --count 30" as load · wait for "sending"
- expect load running
- wait for "30 messages waiting"
- stop load · wait
- expect load exited 0
- expect output of load "sent 30 messages"
```

| Front matter | What it does |
|---|---|
| `start: <command> · wait for "text"` | runs the command in the background, in the script's folder, before the first feature, and waits until its output shows the text (or `· wait for port 5672`: a TCP port on 127.0.0.1 accepts; or `· wait 5s`); `· within 60s` gives it longer than 30 s. Several `start:` lines start in order, in order with `prepare:` and `step:` (see Before and after). `as NAME` (after the command) names it for the steps below; else it is `start-1`, `start-2` … Its output goes to `<name>.log` beside the report. When one does not come up, every feature fails with its last lines. Not filmed. |
| `prepare: <command>` (alias `step:`) | runs once before the first feature, in the order written among the `start:` lines, and must exit 0 within `timeout:` (`· within 5m` for longer); a failed one fails every feature with its output |
| `setup: <command>` | before each feature, in order when there are several; must exit 0 |
| `stop: <command>` | after the last feature (passed, failed or interrupted), in order, while the programs still run, e.g. `make demo-down` |
| `cleanup: <command>` | last, in order, after si has stopped every program, e.g. `rm -rf data/` |
| `shell: bash` | the shell of the terminals: `sh`, `bash` or `zsh` (default `$SHELL`, else `sh`; `cmd` on Windows) |
| `app: none` | a terminal-only demo: no browser; every feature begins with `terminal` |

| Step | What it does |
|---|---|
| `terminal "Name"` | opens a terminal named Name (or brings it back): a real shell in the script's folder, filmed as a code window (§25). Options after `·`: `window=terminal\|mac\|windows\|none` (default `terminal`), `title="…"` (default the name), `theme=night\|dusk\|paper\|auto` (default `night`), `backdrop=graphite\|ocean\|dusk\|mint\|none` (default `graphite`), `size=100x30` (columns x rows; default: as many as fit) |
| `run "command"` | types the command into the terminal in front (at a person's pace in `si demo run`, at once in `si demo test`) and presses Enter. It does not wait: a broker never ends; the next `expect output` is the wait |
| `run "command" · wait` | the same, and waits until it ends (up to a minute, `· within 5m` for longer); fails when it exits non-zero |
| `run "command" · exit 1` · `run "command" · fails` | a failure on purpose: waits until it ends, and passes only when it exits with that status (`· exit 2`) or with any non-zero one (`· fails`); exit 0 is then the failure. Show what the error says with `expect output "…"` after it |
| `press ctrl+c` | in a terminal, sends the key to the shell: `ctrl+<letter>`, Enter, Tab, Escape, Backspace, arrows, one character |
| `expect output "text"` | the terminal's output since its last `run` shows the text, within `timeout:` (`· within 30s` for longer); white space matches any white space, colours do not count |
| `expect no output "text"` | the text never appears from there to the end of the `Say:` block |
| `wait for output "text"` | like `expect output`, as a pause before what comes next |
| `browser` | brings the app back in front |
| `note "…" at line "text"` | in a terminal, a callout under the line that shows the text, ringed |
| `start "command" as NAME` | runs a program in the background for the rest of the feature (not filmed); `· wait for "text"`, `· wait for port N` or `· wait 3s` before going on |
| `stop NAME` | stops it now; `· wait` waits for it to end on its own instead (a workload of a fixed length) |
| `expect output of NAME "text"` | a background program's output shows the text |
| `expect NAME running` · `expect NAME exited 0` | its state |

**Three lifetimes.** A product demo needs traffic on screen: a load generator
running while the console's charts move. Programs live for the whole run, one
feature, or some blocks:

| Lifetime | How | Example |
|---|---|---|
| the whole run | front matter `start:` (and `stop:`, `cleanup:` to tear down), or `start … as NAME` in `## Before` | the broker, the cluster |
| one feature | `setup:` (must exit) for state; a `start` step at the top of the feature for a program, stopped when the feature ends | a seed, a steady workload |
| some blocks | `start "…" as NAME` … `stop NAME` | a burst, a second workload |

A program can also be shown: `terminal "Load"` then `run "…"` (no `wait`)
puts its scrolling output on camera, and `browser` goes back to the console
while it keeps running.

- **Stopped, always.** Each program runs in its own process group, so
  stopping it stops what it started too: SIGTERM, then SIGKILL after 5 s.
  That happens when its feature (a `start` step) or the run (a `start:`, or
  a `start` step in `## Before`) ends, passed or failed, and on Ctrl-C, after
  the `stop:` lines and before the `cleanup:` lines. A terminal hangs up its shell and
  everything running in it when its feature ends.
- **The terminal scene** is a page si serves on 127.0.0.1 with a random key:
  xterm.js inside the code window frame. It opens in its own tab, and the
  video follows the tab in front; the cursor, clicks, notes and captions of
  `si demo run` work on it as on the app. `si demo test` opens it too
  (nothing is filmed), so `expect output` is a real check either way.
- **si's prompt.** The shell starts without your own rc files: the prompt is
  the folder's name and `$`, history expansion (`!`) is off, and job control
  is off (a program run in a terminal stops with it). Commands are typed as
  written: a `$VAR` is expanded by the shell, so its value is never typed on
  screen, and the values of the `$VAR`s a command uses are masked (`••••`) in
  the logs and the report.
- On a failure the report and the JUnit case show the terminal's (or the
  program's) last 40 lines.
- `si check` follows the scenes: a browser step while a terminal is in front
  (write `- browser` first), a terminal step with no terminal, and a program
  name no `start` gave are problems, with their line.
- On Windows a terminal is `cmd` on pipes (no colours); `run … · wait` and the
  rest work the same.

Examples: `examples/webdemo/` (a small shop, `shop.md`, and `broken.md`,
whose coupon does not work; `broker.md`, a tiny broker in Go started with
`go run`, filmed in a terminal and in its console; `morning.md`, seeded and
cleaned up around its feature).

### Before and after

A real demo needs more than one program and one command: start a broker,
create its queues, load data, start a consumer, and clean all of it up
afterwards, every time, even when a step fails.

```markdown
---
type: demo
app: http://127.0.0.1:8080
start:   make demo-single-insecure · wait for port 22310   # keeps running
prepare: ./scripts/create-queues.sh                         # must finish, exit 0
step:    acemq-workloads run -f seed.yaml · within 5m       # step: is prepare:
start:   ./consumer · wait for "listening"                  # a second program, after the seed
setup:   acemq queue purge orders                           # before EACH feature
stop:    make demo-down                                     # while the programs still run
cleanup: rm -rf data/ tmp/                                  # always last
---

## Before
- terminal "Ops"
- run "acemq queue create audit" · wait
- expect output "created"

## Orders flow through
…

## After
- terminal "Ops"
- run "acemq queue delete audit" · wait
```

The order of a run:

1. `start:`, `prepare:` and `step:` lines, **interleaved in the order written**, once;
2. `## Before`;
3. for each feature: its `setup:` lines, then the feature;
4. `## After`;
5. `stop:` lines (the programs still run), then every program stopped (SIGTERM,
   SIGKILL after 5 s), then `cleanup:` lines.

- **Always.** `stop:` and `cleanup:` run after a pass, a failure, a failed
  `prepare:`, a failed `## Before`, and on Ctrl-C (a `prepare:` still running
  is stopped first). Their output goes to the report and to `lifecycle.log`
  beside it, never on film. One that fails is reported and makes the exit code
  1; the features keep their results.
- **`## Before` and `## After`** (any case) are not features: no clip, no
  chapter, no narration, no title card, never filmed, not counted and not in
  `--feature`. Any step works there (`terminal`, `run`, `start … as`, `expect
  output`, `go`, `click`, `expect text` …). A failed step in `## Before` fails
  the run: every feature is skipped and the report says why. `## After`
  always runs; its failures are reported but do not change the features'
  results (the exit code is 1). Programs started in `## Before` with
  `start … as NAME` live for the whole run.
- **The report** shows two extra parts, *Before the features* (the
  `start:`/`prepare:` lines and `## Before`) and *After the features*
  (`## After`, `stop:`, `cleanup:`); JUnit adds a test case for one that
  failed.
- **Many scripts at once.** `si demo test docs/demos/` and `si demo run
  docs/demos/` run every demo script in the folder (`type: demo` with `app:`,
  Markdown, AsciiDoc or JSON, in name order), one after another, each in its
  own folder (`demo-out/<script>/`), with one combined `report.html` and
  `report.json` in `demo-out/` and, with `--junit`, one JUnit file with a test
  suite per script.
- In JSON, `prepare`, `setup`, `stop` and `cleanup` take a string or a list;
  the `start` programs come before the `prepare` lines (JSON keeps no order
  between keys), and a feature named `Before` or `After` is that section.

Example: `examples/webdemo/morning.md` seeds the shop's orders with
`prepare:` and `step:`, checks them in `## Before`, copies them before each
feature with `setup:`, and removes everything with `cleanup:`.

### Scripts from anywhere

Each `start:`, `prepare:`, `step:`, `setup:`, `stop:` and `cleanup:` line takes
a command, or a script from elsewhere with its arguments after it:

| Write | Means |
|---|---|
| `./seed.sh` (any command) | run in the script's folder (`· in <dir>` for another, relative to it) |
| `gist:abc123@4f2e9c1` | a GitHub gist at that revision: its one file, or `gist:abc123@4f2e9c1/seed.sh` |
| `github:owner/repo@v1.2/scripts/seed.sh` | one file of a repo at a tag or commit |
| `github:owner/repo@v1.2/scripts/` | every script in that folder (`.sh`, `.py`, `.js` … or no extension), in name order |
| `https://…/seed.sh · sha256 9f2a…` | a plain URL, with the SHA-256 of the file (`shasum -a 256 seed.sh`) |

```markdown
prepare: github:acme/ops@v1.4/scripts/seed.sh --orders 50
setup:   gist:abc123@4f2e9c1/purge.sh orders
cleanup: https://example.com/ops/down.sh · sha256 9f2a…
```

- **Pinned only.** A `gist:` without `@revision`, a `github:` without
  `@tag` or `@commit`, or a URL without `· sha256` is a `si check` problem. A
  URL whose file no longer has that fingerprint, or a pin whose contents
  changed since you trusted it (a tag moved), stops the run.
- **Asked the first time.** Before running code from elsewhere si shows each
  source (where it comes from, its size, its first lines) and asks once; a yes
  is remembered for that pin in `~/.si/trusted`. `--trust` answers yes for
  one run, remembered nowhere (CI). Without a terminal to ask in and without
  `--trust`, si refuses and says why.
- **Cached by pin** in `~/.si/cache/scripts`: a pin is fetched once, then
  runs offline. `github:` uses GitHub's API (`$GITHUB_TOKEN` or `$GH_TOKEN`
  when set, sent only to the API and never printed), and `gh` when it is
  there for a private repo. A fetched script runs as a program (its `#!` line
  picks the interpreter; `sh` without one), in the script's folder.

**Demos from anywhere.** `si demo test` and `si demo run` take the same
places as a file or a folder:

| Write | Means |
|---|---|
| `si demo test github:owner/repo@v2/demos/` | the repo at that pin, fetched once into `~/.si/cache/repos` with its layout kept; every demo script in the folder |
| `si demo test github:owner/repo@v2/demos/shop.md` | one script of the repo, its folder and the rest of the repo alongside (`../../examples/…` resolves) |
| `si demo test "https://…/shop.md · sha256 …"` | one script from the web, its fingerprint checked (files beside it are not fetched: its `app:` is a URL or `none`) |

The same trust rules hold (the repo's pin is asked about once, with the
scripts it will run); the output folder is local, `demo-out/<repo>/<script>/`.

```bash
si demo test github:acme/shop@v2.0/demos/ --trust --junit junit.xml   # CI
```

### Transitions, camera, spotlight, windows and drag

A demo video cuts where the scene changes: a `go` that loads a page, a
`terminal` or `browser` switch, a `window` switch, and between features in
`<script>.mp4` (title card → clip → next card). Each of those cuts is a
**transition**, `fade 0.4s` unless the script says otherwise; while a page
loads, the old scene's last frame holds, so the video never shows a blank
page. The transitions are drawn when the video is encoded, from the take's own
frames: the length of the video never changes, so the voice is never cut or
sped up and the captions, badges and chapters keep their times.

```markdown
---
type: demo
app: app
transition: slide 0.6s      # every cut of this script (cut = the old hard cuts)
---

## Pay for an order

Say: The order page loads.
- go /order.html?id=4711 · fade
- terminal "Client" · pop
- run "curl -s localhost:8080/orders/4711"
- browser · wipe
```

| Transition | What it looks like |
|---|---|
| `cut` | no transition (before v3) |
| `fade` | the old scene dissolves into the new (the default, 0.4 s) |
| `slide` (`slide-left`), `slide-right`, `slide-up`, `slide-down` | the new scene pushes the old one out, moving that way |
| `wipe` (`wipe-left`), `wipe-right`, `wipe-up`, `wipe-down` | an edge sweeps across, showing the new scene behind it |
| `pop` | the new scene grows in from 92 % as it fades in |
| `zoom` | the old scene grows and fades into the new |
| `circle` | the new scene opens in a circle from the cursor |

- The length follows the name (`fade 0.8s`, `slide 500ms`; 0.1 s to 5 s).
- Which one: the step's own (`- go /orders · slide`, after `go`, `browser`,
  `terminal "…"` and `window …`), else `--transition`, else `transition:`
  in front matter, else `fade 0.4s`. Between features `<script>.mp4` uses the
  script's (or `--transition`'s): ffmpeg's `xfade`, `pop` there is a fade and
  `circle` opens from the centre. Each card and clip still starts where it
  would with cuts, so the chapters do not move.
- `si check` reports an unknown name with its line.

**Camera moves.** The picture can glide closer to what the narration is
about, and back:

```markdown
Say: The total updates as you choose.
- select "Express" in "Shipping"
- zoom to #total                # glide in to the element (with a margin), and stay
- pan to "Gift wrap" · 1.2s     # glide across at the same zoom
- zoom out                      # back to the whole page
```

| Step | What it does |
|---|---|
| `zoom to "Element"` | the picture glides from where it is to the element's box with a margin, grown to the video's shape (never closer than 3×), and stays there |
| `pan to "Element"` | glides to the element at the current zoom |
| `zoom out` | back to the whole picture |
| `… · 0.8s` | the move's length (default 0.8 s, eased at both ends); the next step waits for it |

- The box is measured on the page when the step runs (the element is
  scrolled into view first). The move is a crop of each frame, scaled back
  to the frame's size, so the video keeps its size; the drawn cursor, notes
  and spotlight are in the page and zoom with it.
- The take is supersampled: the page, its terminals and windows are filmed
  at 2 device pixels per CSS pixel (2880×1800 for a 1440×900 viewport), and
  every video is scaled back down to the viewport's size at the end
  (lanczos), so text stays crisp, even 3× zoomed. `--scale 1` (or
  `scale: 1` in front matter) films at the viewport's size: a little faster,
  less temporary disk, softer zooms.
- A scene change (`terminal`, `browser`, `window`) starts the new scene
  zoomed out.
- With a region (`--region`, `Region:`, `region:`) the camera works inside
  it: `zoom to` frames an element in the region at the region's shape, and
  `zoom out` comes back to the region. `Region: ".agents" · zoom` (and
  `region: … · zoom`, `--region ".agents · zoom"`) instead opens the clip on
  the whole page and, a second in, zooms into the region (measured as a
  crop is, at the end of the take) and stays there.
- The camera works with fast-forward (a move keeps its length wherever it
  falls), the transitions and the captions band.
- In `si demo test` a camera step checks its element is there; nothing moves.

**Spotlight** (alias `focus`) dims the rest of the page around one element:

```markdown
- spotlight "Pay"                                 # the rest dimmed to 60 %, a rounded box around Pay
- spotlight "Pay" · circle                        # a round cut-out
- spotlight "Agents panel" · label "Focus" · for 3s
- focus "Pay" · blur                              # blur the rest instead of darkening it
- spotlight off
```

- The dark layer fades in and the cut-out shrinks from the whole screen onto
  the element, then follows it (it scrolls, it grows). `· label "…"` puts a
  tag on the cut-out's edge, in the notes' style. The ring and the label take
  the accent colour (`--si-accent`) and the dark layer the text colour
  (`--si-fg`) of si's studio theme.
- It ends at `spotlight off`, after `· for 3s`, or with its `Say:` block;
  `note` bubbles and the cursor stay above it.
- In `si demo test` it checks the element is there and draws nothing.

**Windows.** An app may open a popup or a tab: si's own teleprompter (`p` in
a deck), a sign-in window, a print preview. `window` moves the performing and
the filming to it:

```markdown
- press p
- window "si-presenter" · pop     # by window name, title or address (a part of it, any case)
- expect text "Teleprompter"
- window main                     # back to the app's page (window 1)
- window 2                        # the first page the app opened, 3 the next …
```

The page is found among the pages the feature has open (up to `timeout:`),
given the video's viewport, and filmed from then on, with the cursor, notes
and spotlight. A window that never shows fails with the names, titles and
addresses that were open. Works in `si demo test` as well.

**Drag.** A real press, move and release, with the drawn cursor gliding and
the button held: drawing on a slide (`a` in a deck), moving map topics,
sliders, sortable lists:

```markdown
- drag from "Shop" to "Doorstep"      # element to element (their centres)
- drag from 400,300 to 900,420        # points, in CSS pixels of the viewport
- drag by 300,80                      # from where the cursor is
```

The pointer moves in 30 small steps (about a second on film), so a page that
follows `pointermove` sees a stroke, not a jump.

**Title cards** show `What it does:` as written: `si reads the notes…` stays
lower case.

## 28. Courses

A course is a **folder with a `_course.md`**: modules of lessons in order,
each lesson something si already makes. Readings are pages, slides are decks
that present themselves in the page, videos are `.mp4` files or filmed web
demos, labs are `type: lab`, instructor demos are `type: demo` runbooks, and
quizzes are `type: quiz`. It builds into a website bundle with a home page,
progress kept in the learner's browser and a completion page.

```text
web-course/
  _course.md          modules, lessons, course-wide settings
  01-web/browser.md   a reading
  01-web/journey.md   type: deck, a slides lesson
  01-web/page-load.mp4  a video lesson
  02-html/first-page.md  type: lab
  02-html/inspect.md  type: demo, an instructor demo
  02-html/quiz.md     type: quiz
```

`_course.md` is front matter plus a list: the top-level items are
**modules**, the links under them **lessons**.

```markdown
---
title: A first web page
description: How a browser turns a file into a page, then write one.
level: beginner
nav: left · accordion · sticky
theme: studio
instructor-demos: show-placeholder
certificate: on
prerequisites: a text editor and a web browser
---

- Module 1: How the web works
  - [What a browser does](01-web/browser.md)
  - [From address to pixels](01-web/journey.md)
  - [A page loads, step by step](01-web/page-load.mp4)
- Module 2: Writing HTML · prerequisites=Module 1
  - [Build your first page](02-html/first-page.md)
  - [Inspect a page live](02-html/inspect.md)
  - [Check yourself](02-html/quiz.md) · minutes=5
```

### Lesson kinds

A lesson's kind comes from what it links to; `· kind=reading` (or `slides`,
`video`, `lab`, `demo`, `quiz`) after the link sets it by hand.

| Links to | Kind | The page shows |
|---|---|---|
| a page, an article, a guide page | Reading | the page |
| `type: deck` | Slides | the deck in the page, with auto-play (`v`) |
| a `.mp4`, `.webm`, `.m4v` or `.mov` | Video | a player; the file is copied beside the page |
| `type: demo` with `app:` | Video | its `<script>.mp4`, filmed by `si build` when missing or older than the script |
| `type: lab` | Lab | the lab; its task-list boxes can be ticked |
| `type: demo` without `app:` | Demo | the runbook (instructor build only) |
| `type: quiz` | Quiz | the questions, scored |

**Minutes.** A lesson's `minutes:` front matter (or `· minutes=` in
`_course.md`) sets its length; otherwise si estimates it: reading time at 230
words a minute, a deck's `(N min)` notes, a video's length, five minutes a
lab step, a minute a quiz question, a demo's `duration:`. Modules and the
course add them up; `duration:` in `_course.md` overrides the course total.

**Prerequisites** on the course (`prerequisites:`), a module (`·
prerequisites=…`) or a lesson (front matter) are shown, not enforced.

### Two audiences

- **Learner** (`si build course/`, `si serve course/`): readings, slides,
  videos, labs, quizzes. Instructor demos are left out of the menu, or with
  `instructor-demos: show-placeholder` shown as "Your instructor shows this
  live". Decks carry no speaker notes; lab solutions are folded.
- **Instructor** (`si build course/ --instructor`, `si play course/`):
  everything, plus each demo's runbook, the decks' teleprompter (`p`), lab
  solutions open, and an **instructor box** above any lesson whose front
  matter says `instructor: …` (a note learners never see).

### The navigator: `nav:`

`nav:` in `_course.md` (or si.conf) takes a position and options, split by
`·`, commas or spaces:

| Value | What it does |
|---|---|
| `left` (default), `right` | the modules in a column beside the lesson |
| `top` | the modules as tabs, the open module's lessons in a row under them (`tabs`) |
| `bottom` | a bar docked at the foot of the window: where you are, Outline, Back, Next |
| `sticky` (default), `not-sticky` | stays while the page scrolls, or scrolls away |
| `accordion` (default) | one module open; opening another closes the rest; the current lesson's module stays open |
| `collapsed` | every module closed but the current one |
| `open` | every module open |
| `numbered` | numbers on guide sections (modules are always numbered) |

Each module shows its number, a progress ring and its minutes; each lesson
its kind (Read, Slides, Video, Lab, Demo, Quiz), its minutes and a done tick.
Menu text is at full strength; a hovered lesson gets a tinted row and an
accent bar, the current one a solid accent bar. On a phone every layout folds
into a **Menu** button and a bottom bar. Everything is drawn with the theme's
`--si-*` tokens, so every theme, light and dark, works.

### Quizzes

A `type: quiz` page (or a ```` ```quiz ```` fence in any lesson, article or
guide page): `## ` questions, the answers as a task list (`- [x]` marks a
right one; more than one makes it multiple choice), an optional `>
Explanation:` shown after answering, and a pass mark.

````markdown
---
type: quiz
title: Check yourself
pass: 70%
---

# Check yourself

## Which tag makes the largest heading?
- [x] `<h1>`
- [ ] `<h6>`
- [ ] `<head>`
> Explanation: `h1` to `h6` are headings; `head` holds the page's settings.

## Which of these are block elements?
- [x] `<p>`
- [ ] `<em>`
- [x] `<ul>`
````

A fence takes the mark as an option: ```` ```quiz pass=80% ````. A question
is right when exactly its right answers are picked; the score is the share
of right questions. Without a pass mark, answering every question completes
the quiz. `si check` reports a question with fewer than two answers or none
marked right. The answers are in the page, so a quiz checks understanding;
it is not an exam.

### Lab solutions

In a lab, `<!-- solution -->` up to `<!-- end -->` is the answer: folded behind
**Show the solution** for learners, open in the instructor build.

````markdown
## Step 3: Write the content

- [ ] The heading, the paragraphs, the list and the link all show

<!-- solution -->

```html
<h1>Sourdough at home</h1>
```

<!-- end -->
````

### Progress

Progress is kept in the learner's browser (localStorage, per course; nothing
is sent anywhere). A lesson is done when it is read to the end, its video
watched to 90 %, its lab's boxes all ticked, or its quiz passed. The menu
ticks done lessons and fills each module's ring; the top bar shows the
course's percentage; the home page's **Start** becomes **Continue: <the
next lesson>**. `complete.html` lists what is left and, when everything is
done, says so; with `certificate: on` the learner types a name and prints a
certificate.

### Build, serve, check

- **`si build course/ -o html/`** writes the website bundle: `index.html` (the
  course home: level, length, modules, Start / Continue, a card per module),
  one page per lesson mirroring the folders, `complete.html`, and the videos
  copied beside their pages (never inlined). Shared scripts are written once
  (`--share-assets` is on for courses). `--zip` also writes `<course>.zip`
  beside the output; `--instructor` builds the instructor's edition;
  `--no-film` skips filming web demo lessons.
- **`si serve course/`** serves the learner's course with live reload;
  **`si play course/`** the instructor's.
- **`si check course/`** reports a missing page or video, a lesson outside a
  module, an unknown `kind=`, bad `minutes=`, an unfilmed web demo (a
  warning), quiz problems and broken links between lessons.
- **`si new web-course --type course`** writes a small course with every kind
  of lesson; `si --export example --type course` the same example.

## 29. Timelines

A timeline is a one-page infographic: eras with their colours, dated
milestones with a short description and a picture, a closing line and the
sources. It is a page of its own (`type: timeline`) or a ```` ```timeline ````
fence inside a slide or a document.

### A timeline page

```markdown
---
type: timeline
title: Hitos en Algoritmos y Complejidad Computacional
subtitle: Del algoritmo de Euclides (300 a.C.) a la inteligencia artificial
lang: es-MX
layout: snake
closing: La pregunta P vs NP, planteada en 1971, sigue sin respuesta.
require: 20 dates, 15 images
---

## Antigüedad · color=#b3541e

### c. 300 a.C. · Algoritmo de Euclides
En "Los Elementos" (libro VII) Euclides describe el método para calcular el
máximo común divisor.
![Retrato de Euclides](img/euclides.jpg "Wellcome Collection, CC BY 4.0")

### c. 240 a.C. · Criba de Eratóstenes
Método sistemático para encontrar todos los números primos hasta un límite dado.

## Edad Media

### c. 825 · Al-Juarismi
Su nombre latinizado ("Algoritmi") da origen a la palabra "algoritmo".
![Al-Juarismi](https://upload.wikimedia.org/…/al-khwarizmi.jpg "Dominio público")

## Fuentes · sources
- Cormen, T. H., et al. (2022). Introduction to algorithms (4.ª ed.). MIT Press.

Notes: Empezamos en la Antigüedad. [CLICK] En la Edad Media…
```

- `## Era` is an era: a band of colour on the line and an entry in the
  legend. `· color=` takes any CSS colour; without it the era gets the next
  colour of a palette turned around the theme's accent (`--si-tl-base`),
  readable in light and dark.
- `### date · title` is a milestone. The lines under it are its description
  (one or two lines read best; `si check` warns above 220 characters).
  `![caption](src "credit")` is its picture, drawn as a medallion on the line,
  with the caption and the credit under the description. `src` is a file
  beside the page or an `https://` address, fetched once at build time
  (cached) and inlined, so the page stays one file.
- Dates are shown as written. si reads them only to check the order and for
  `scale: time`: `300 a.C.`, `300 BC`, `BCE`, `c.` / `ca.`, years, `1936-05`,
  ranges `2022–2024` (the start counts).
- `## Anything · sources` is the sources block, shown at the end.
- On a landscape page the timeline is one slide: `si play` presents it,
  `Notes:` and `Points:` at the end are its teleprompter script and speaker
  points, and each era after the first is a build step (`steps: off` shows
  them all at once). Text outside a milestone is not shown (`si check` says
  so).

| Front matter | Values | Default |
|---|---|---|
| `type` | `timeline` | |
| `title`, `subtitle` | the header | |
| `layout` | `snake` (the line wraps across rows: 30 milestones fit one 16:9 page), `line` (one horizontal line, two levels a side), `vertical` (one line down the page, cards left and right) | `snake`; `vertical` when portrait |
| `orientation` | `landscape` (a slide: a frame of `aspect:` that scales like a picture), `portrait` (a tall scrolling poster, a document page) | `landscape` |
| `aspect` | the landscape frame: `16:9`, `16:10`, `4:3`, `1:1` | `16:9` |
| `alternate` | `on` (milestones above and below the line, or left and right), `off` (one side) | `on` |
| `scale` | `even` (equal gaps), `time` (gaps follow the years, held between 0.6 and 1.8 of the usual gap; a shortened long gap gets a ≈ mark) | `even` |
| `closing` | a line at the end of the timeline | none |
| `steps` | `eras` (one build step per era on the slide), `off` | `eras` |
| `require` | a rubric for `si check`: `20 dates, 15 images, 7 eras, 4 sources` | none |
| `lang`, `theme` | as on any page; every built-in theme works, light and dark | |

On a phone (narrow or portrait screen) every layout becomes `vertical`, with
the line on the left.

### In a slide or a document

````markdown
## Sorting, briefly

```timeline layout=line steps
## Cards
### 1890 · Hollerith tabulator
Punched cards sorted by machine for the US census.
## Libraries
### 2002 · Timsort
Tim Peters writes the sort Python still uses.
```
````

The fence takes the same Markdown (a `# Title` line inside it is the title)
and the options `layout=`, `alternate=`, `scale=`, `aspect=`,
`orientation=`, `title=`, `subtitle=`, `closing=` and `steps` (one build
step per era after the first; slides only). On a slide it fills the slide; in
a document it is `vertical` unless `layout=` says otherwise.

### Check and export

- `si check` prints the counts (`timeline: 30 milestones, 30 pictures, 7
  eras, 5 sources`) and reports missing pictures, milestones out of
  chronological order, long or empty descriptions, and an unmet `require:`.
- `si build timeline.md --png` writes `timeline.png`: the landscape frame with
  every era shown, or the whole portrait poster; `--scale 3` for more pixels
  (default 2: a 16:9 timeline is 3200 × 1800). `--pdf` writes
  `pdfs/timeline.pdf` (one landscape page, or the printed poster). Both need
  Chrome.
- `si new history --type timeline` writes an example timeline with its
  pictures; `si --export example --type timeline` the same.

## 30. Infographics

`type: infographic` is a one-page visual: big numbers, comparisons, steps,
charts drawn from Markdown tables, quotes, icon rows and a sources footer, in
a bento grid with an editorial look. Every `##` heading is a block.

````markdown
---
type: infographic
title: The 600 ms checkout
subtitle: How one store cut its page load by three quarters
orientation: landscape   # landscape (a slide) | portrait (a tall poster)
aspect: 16:9             # 16:9 | 4:3 | 9:16 | a4
layout: grid             # grid (bento) | columns | flow (top to bottom)
theme: studio
sources: Real-user monitoring, weeks 1 to 6
---

## 2.4 s · stat
Median page load before the fix. ![](icons/clock.svg)

## Median page load · compare · wide
| Before | After |
|---|---|
| 2.4 s | 0.6 s |

## How the time came off · steps
1. **Measure** real visitors
2. **Shrink** the images
3. **Cache** at the edge

## Traffic by device · chart=donut
| Device | % |
|---|---|
| Phones | 64 |
| Desktops | 36 |

## What the team says · quote
> Every hundred milliseconds showed up in the sales graph.
> — The checkout team lead

## Sources · sources
- web.dev, Core Web Vitals (2024)

Notes: Our checkout took two point four seconds; six weeks later, zero point six.
````

### Blocks

`## Heading · kind · options`. The kind is one of:

| Kind | Write under the heading | Drawn as |
|---|---|---|
| `stat` | the number is the heading (`## 54% · stat`); a line of text, an optional `![](icon.svg)` | a big numeral, its unit smaller, the text under it |
| `compare` | a table, one column per side (`\| Before \| After \|`); 3+ columns: row labels first | the sides next to each other, the last one stressed; one row of two numbers adds the change (−75%) |
| `steps` | a numbered list | numbered circles; across when the block is wide |
| `chart=bar` | a table: labels, then one or more columns of numbers | horizontal bars to scale (the top one stressed; several columns: grouped, with a legend) |
| `chart=donut` | a table: labels, numbers | a donut of the shares, the first share in the middle, a legend |
| `chart=line` | a table: x labels, one or more columns of numbers | a line (area under the first) from zero, the last value at its end |
| `chart=progress` | a table: labels, a value (of 100, or of a third column: the target) | progress bars |
| `quote` | a `>` quote; a line starting `—` is the attribution | a large quote |
| `icons` | a list; each item may start with `![](icon.svg)` | an icon row |
| `list`, `text` | a list, any Markdown | as written |
| `image` | `![caption](picture.jpg)` | the picture filling the block |
| `sources` | a list | the footer, small (with `sources:` and `author · date`) |

Without a kind, si guesses: a heading that starts with a number is a `stat`,
a table of numbers `chart=bar` (any other table `compare`), a numbered list
`steps`, a list with pictures `icons`, a `>` quote `quote`, a lone picture
`image`, a heading named Sources (Fuentes, References) `sources`, anything
else `text`.

Options for the grid: `span=2` (or `wide`) covers two columns, `tall` two
rows, `accent` fills the block with the accent colour. A `·` part whose first
word is not an option stays in the heading (`## Paris · London`).

### Layout and orientation

- **`orientation: landscape`** (the default) is a slide: a 16:9 frame (or
  4:3) sized by container units, presented with `si play` (keys,
  teleprompter, `Notes:` and `Points:` after the last block). It shows every
  block at once; `steps: on` reveals one block per click (a `[CLICK]` per
  block in `Notes:`). On a phone or an
  upright tablet the blocks stack in one column and the slide scrolls.
- **`orientation: portrait`** (or `aspect: 9:16` / `a4`) is a tall poster
  page that scrolls: two columns, one on a phone; `aspect:` sets its least
  height.
- **`layout: grid`** packs the blocks into a bento grid, choosing the number
  of columns that leaves the fewest holes (a block next to a hole widens into
  it); `columns` fills columns top to bottom; `flow` is one block per row.
- Colours come from the theme (every built-in theme, light and dark): odd
  blocks take `--si-accent`, even ones `--si-accent-2`, chart series
  `--si-ig-1` to `--si-ig-4`; `--si-ig-bg` and `--si-ig-card` set the frame and
  the blocks.

### Pictures

Local files are inlined like any image. `https://` pictures are fetched at
build time and inlined (cached, so `--offline` works after one online build);
one that cannot be fetched stays a link, with a warning.

### In a deck or a document

An ```` ```infographic ```` fence holds the same blocks (a `# Title` line and
a paragraph under it are the header) and fills the slide, one build step per
block (`steps=off` shows them all). Options on the fence line: `title=`, `subtitle=`, `layout=`,
`orientation=`, `aspect=`, `steps=off`, `sources=`. In a document the fence
is a portrait poster.

````markdown
## Our quarter

```infographic layout=grid
## 18% · stat · accent
more checkouts

## Traffic · chart=donut
| Device | % |
|---|---|
| Phones | 64 |
| Desktops | 36 |
```
````

### Build, check, new

- **`si build talk.md --png --pdf`** writes `talk.png` (the slide at 1600 ×
  900, or the whole poster at 1080 wide; `--scale 2`, the default, doubles
  the pixels; `--scale 3` for print) and `pdfs/talk.pdf` (one page, vector
  text). `si play` presents a landscape one.
- **`si check`** reports an unknown kind or option, a chart cell that is not a
  number (or a negative one in a donut), a compare or chart without a table,
  an empty block, a stat without its number, and `[CLICK]` cues that do not
  match the blocks.
- **`si new poster --type infographic`** writes a starter; `si --export
  example --type infographic` the two examples (a landscape story and a
  portrait poster).
