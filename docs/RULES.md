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
31. [Project structure](#31-project-structure)
32. [Video edits](#32-video-edits)
33. [Animated charts](#33-animated-charts)
34. [Scenes: objects that move](#34-scenes-objects-that-move)
35. [Architecture pages](#35-architecture-pages)
36. [Status boards](#36-status-boards)
37. [Sandbox labs](#37-sandbox-labs)

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
| `type` values | `deck`, `lab`, `article`, `guide`, `book`, `mindmap`, `architecture`, `status` | `deck` | `lab` §18, `article` §19, `guide` §20 (guide pages get it from `_nav.md`), `book` §21 (in `_nav.md` only), `mindmap` §24 (with `layout`, `lines`, `nodes`, `map-theme`, `watermark-position`), `architecture` §35, `status` (alias `status-board`) §36. A type si does not know is a `si check` problem and the page renders as an article |
| `number` | `on`, `off` | `on` | architecture pages: number the `##` sections (§35) |
| `updated`, `meta`, `tiles` | a date; `- label: value \| tone` items; `- num: …; lbl: …; sub: …` items | none | status boards: the first pill, the pill row and the KPI tiles (§36) |
| `subtitle`, `edition` | text | none | books: the cover's subtitle and edition (§21) |
| `session`, `slides`, `duration` | text | none | demo scripts: the facts under the title; `duration: 14 min` is checked against the script's steps (§22) |
| `app`, `viewport`, `timeout`, `setup`, `voice` | URL or folder; WxH or phone, tablet, desktop; seconds; a shell command; engine:voice | none; desktop; 5s | web demos: `app:` turns a demo script into a script si runs against a web app (§27) |
| `setup`, `start`, `stop`, `cleanup` (each may repeat) | a command, or a source: `gist:ID@REV`, `github:owner/repo@REF/path`, `https://… · sha256 HEX` | none | a talk that sets up what it shows live: `si play` and `si serve` run `setup:` (must finish) and `start:` (keeps running, `· wait for "ready"`) in the order written before the first page, then `stop:`, the programs stopped, `cleanup:` when they quit; `si build` never runs them ([§14](#14-presenting-keys-touch-overview-print), A talk that sets itself up) |
| `files` | `path · sha256 HEX, …` | none | for a deck or a web demo opened from a pinned URL: the files beside it to fetch too (pictures, scripts, a theme), each relative to the URL and checked against its own fingerprint; ignored for a local file (§27, Scripts from anywhere) |
| `prepare`, `step`, `cleanup` | web demos: a command run once before the first feature (`step:` is its alias); a command run after every program stopped | none | §27, Before and after |
| `description` | text | none | documents: the lede under the title, the index tile subtitle and `<meta name="description">` (§19, §20) |
| `edit-url` | URL | none | guides: base of the "Edit this page" link (§20) |
| `header`, `footer-page` | `off` | on | a guide page without the guide's `_header.md` / `_footer.md` (§20) |
| `repo` | `OWNER/REPO`, a github.com URL, or a local clone folder | none | the default repo of `code:` and `diff:` (§13) |
| `voices` | `Name=voice, Name=voice` (`Ada=af_heart, Ravi=kokoro:am_michael`) | every speaker in `voice:` | auto-play in a deck with `speakers:`: each speaker's part in their own voice; a bare name is a voice of the deck's engine (§11, Auto-play) |
| `linkedin`, `tags`, `canonical`, `cover` | text; comma-separated tags; a URL; a picture | none | `si publish` (§26): the LinkedIn post text, dev.to / Hashnode tags, the article's original URL, the picture of a LinkedIn link post |
| `poster` | a picture beside the page, or `off` | the first slide (`<deck>.poster.jpg`) | a course video lesson's still (§28, Video lessons from a deck) |

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
| `transition: zoom`, `transition: morph 0.9s`, `transition: cube 1.2s` | slide | [Transitions](#6-transitions) |
| `id: promise` | next block | names the block so `transition: morph` matches it on the next slide ([Transitions](#6-transitions)) |
| `background: assets/city.jpg opacity=0.4` | slide | [Backgrounds](#7-backgrounds) |
| `footer: off` | slide | overrides the deck `footer` on this slide (`off` hides it, e.g. the title) |
| `labs: [Create a Cluster](https://…), [Scale out](https://…)` | slide | turns the slide into a lab slide: a centred beaker icon, the `labs-label` heading, and a button per lab (opens in a new tab). A bare URL works too and shows a default label. |
| `class: dense dark-code` | slide | adds classes to the slide for your CSS |
| `eyebrow: Order #4711 · the promise` | slide | a small context label above the slide heading (`.si-kicker`); it is not part of the slide title, so find, the overview and the teleprompter show the heading alone. On a section slide it replaces the "Part N" kicker. |
| `image: width=40% align=right` | next image | [Images](#8-images) |
| `steps`, `step`, `steps: a b`, `step: a`, `steps: focus\|deal\|reorder\|carousel` | next block | [Build steps](#9-build-steps) |
| `animate: type\|words\|highlight\|count\|draw\|decode\|confetti\|tilt\|explode\|flap\|flyin\|orbit` | next block | a build animation ([Build steps](#9-build-steps)) |
| `svg: assets/canvas.svg state words` | body | [SVG canvases](#10-animated-svg-canvases) |
| `include: other.md` | deck | [Includes](#12-includes) |

Body directives (`animate`, `column`, `id`, `image`, `step`, `steps`, `svg`) work on the
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

`<!-- transition: NAME -->` sets how a slide arrives; the front matter
`transition:` sets the default (`fade`). `fade`, `slide`, `zoom` and `none`
are CSS; `morph` is below; the ten motion transitions are played by the
page itself:

| Name | What the room sees | Length |
|---|---|---|
| `cube` | the slide turns on the edge of a cube; the next face swings into view | 1s |
| `flip` | the slide flips over like a card; the next slide is on its back | 0.9s |
| `iris` | a circle opens from where you clicked (else the centre) and grows into the next slide | 0.9s |
| `wipe` | a soft band of light sweeps across and leaves the next slide behind it | 0.9s |
| `shatter` | the slide breaks into tiles that fall away, nearest the click first | 1.3s |
| `depth` | the slide sinks back and blurs; the next one arrives from in front of the camera | 1.1s |
| `blinds` | vertical strips turn one after another, each showing a strip of the next slide | 1.1s |
| `portal` | the camera flies into a picture on the slide, and that picture becomes the next slide | 1.1s |
| `glitch` | a short digital tear: colour channels split, slices jump, the next slide settles | 0.55s |
| `stack` | the next slide slides over this one like a card on a pile; this one sinks under it | 0.9s |

```markdown
<!-- transition: cube -->
## Payment authorised

---

<!-- transition: portal 1.4s -->
## The receipt
```

A length after the name (`cube 1.2s`, `iris 600ms`) sets it for that slide;
only `morph` and the motion transitions take one. A motion transition
belongs to the slide it arrives at: going back from that slide plays the
same motion in reverse (the mirror), going back to it fades. `portal` flies
into the block named with `<!-- id: name -->` on the slide before, else that
slide's first picture or diagram. A jump to a slide that is not the next
one (a `#7` link, picking a slide in the overview, find, Home and End)
plays the transition of the slide it lands on, as that slide arrives; going
back one slide plays the mirror. Reduced motion, print, PDF, the overview,
thumbnails and handouts show each slide as it ends. Two slides
that show the same SVG keep the canvas transition, and `morph` keeps its
own rules.

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
| `poster` | `assets/still.jpg` | a video's still before it plays |

**Videos.** An image whose file is a video (`.mp4`, `.webm`, `.mov`, `.m4v`,
`.ogv`) is a player, on a slide and in a document alike: the title (else the
alt text) is the caption, `poster=` in the `image` directive its still.

```markdown
<!-- image: poster=assets/cluster.jpg -->
![The cluster, narrated](assets/cluster.mp4)
```

It renders `<figure class="si-image" data-video><video controls
preload="metadata" playsinline src="assets/cluster.mp4" …></video><figcaption>`.
In a document (an article, guide, book, lab or course page) the video plays
in si's own player, with the captions (`cluster.vtt`) and chapters
(`cluster.chapters.txt`) beside it; on a slide it keeps the browser's
controls. The ```` ```video ```` fence (§25, Video player) has every option:
a dialog over the dimmed page, galleries, a start time.

A video without a `poster=` still opens on its first frame rather than on
black: `si build` uses `cluster.poster.jpg` beside the video, and when there is
none it writes one from the frame at 0.5 s with ffmpeg (skipped quietly
without ffmpeg). The still si writes carries its video's fingerprint (size
and SHA-256, in a JPEG comment), so it is made again when the video's
content changes, never because of file times (a git checkout scrambles
them); a still you put there yourself, without that comment, is kept as it
is. Give `poster=` to choose the picture. `si build talk.md --mp4` writes
`talk.poster.jpg` too: the deck's first slide at 1920×1080, at rest (taken
with reduced motion, so a theme whose slides animate in shows the slide, not
its empty start; a still that comes out blank gives way to the video's frame
at 0.5 s). PDFs, handouts and the dissolving stills are taken at rest too. A `poster=` file
that is missing is a `si check` problem.
A video is never inlined as a data URI: `si build` copies it beside the page
(in every media mode), and so does the `src` of a raw `<video>`, `<source>`
or `<audio>` you write yourself.

**Linked files.** A link to a local file si does not render (`[The slides
(PDF)](assets/talk.pdf)`, a `.zip`, a `.csv`, an `.mp4`) is copied beside the
built page the same way, keeping its path (a file outside the page's folder
goes to `media/`). A linked file that is missing fails `si check`. Links to
pages (`.md`, `.adoc`, `.html`), folders and outside URLs are left alone.

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
- `?play` before the hash plays the way there: `talk.html?play#5` opens slide 4 at its end and moves on to 5, so slide 5's transition and build animations run (`?play#5.2`: from step 1 to step 2). For a link that shows an effect.

**How a list arrives.** A first word after `steps:` changes how the items
come in; canvas tokens may follow it as above.

```markdown
<!-- steps: focus -->
- Shop
- Payments
- Warehouse
- Courier

<!-- steps: deal -->
- Shop
- Payments

<!-- steps: reorder -->
- Address wrong 12%
- Stock drift 61%
- Courier no-show 23%
```

- `steps: focus`: every item is shown; each step brings the next one into
  focus and blurs and dims the rest, like a camera changing focus.
- `steps: deal`: each item flips in from its top edge like a dealt card.
- `steps: reorder`: the list is shown as written; one build step re-sorts
  it and each item glides to its new place. The new order is by the last
  number in each item, highest first (`61%`, `23%`, `12%`), or the order you
  give as item numbers counted from 1: `<!-- steps: reorder 2 3 1 -->` puts
  item 2 first, then 3, then 1. Going back puts it as written again.

**Build animations.** `<!-- animate: NAME -->` animates the next block as it
arrives: with its build step when it has one (`<!-- step -->` before or
after it), else when the slide arrives.

```markdown
<!-- animate: type -->
> Delivered. Two days, door to door.

<!-- step -->
<!-- animate: count -->
**12,480** orders on time

<!-- animate: highlight -->
Shoes at the door in ==two days==.
```

| Name | What it does |
|---|---|
| `type` | the text types itself out with a blinking caret |
| `words` | words rise one by one out of a soft blur |
| `highlight` | the marker of each `==mark==` in the block paints in, left to right |
| `count` | the digits roll into place like an odometer, each column at its own speed |
| `draw` | an `svg:` canvas or a stage diagram draws its lines on like a pen, then fills |
| `decode` | the text arrives scrambled and decodes letter by letter |
| `confetti` | the block pops in and a burst of confetti falls away (then it is removed) |

**In 3D.** Six more give a slide depth, with CSS 3D only (no library), for
the moments that deserve it:

```markdown
<!-- steps: carousel -->
- Shop
- Payments
- Warehouse
- Courier
- Customer

<!-- animate: explode -->
- Database
- Queue
- API
- Browser

<!-- animate: orbit -->
- Order
- Shop
- Payments
- Courier

<!-- animate: flap -->
DELIVERED
```

| Name | What it does |
|---|---|
| `animate: tilt` | the slide's heading, the marked block and the rest become layers at different depths; the camera tilts once as the slide arrives, then follows the pointer while it is shown |
| `steps: carousel` | the list's items stand as cards in a ring; each step turns the next card to the front and highlights it |
| `animate: explode` | on a list: one plate per item, stacked and tilted, pulling apart so each name can be read (the first item is the bottom plate) |
| `animate: flap` | the text is shown as split-flap tiles; the letters flip through the alphabet until the word lands |
| `animate: flyin` | the words fly in from far behind the slide, out of a blur, one after another |
| `animate: orbit` | on a list: the first item in the middle, the rest on a tilted ring that turns, always facing you; it turns while the slide is shown and rests when you leave |

`carousel`, `explode`, `orbit` and `flap` keep their 3D look at rest, so
print, PDF, the overview and thumbnails show the ring, the plates or the
tiles as they end; `tilt` and the motion of the others never run there.

Every animation ends with the block exactly as written. Going back does
not replay it; reduced motion, print, PDF, the overview, thumbnails,
handouts, the teleprompter previews and a jump (`#5.2`) show the final
state.

**In documents.** In an article, guide or book each `animate:` block plays
once, when it is scrolled into view, and the lists `steps: focus`, `deal`,
`reorder` and `carousel` play through their items the same way (focus moves
down the list, deal turns each item over, reorder re-sorts after a moment,
carousel turns to each card). A `reorder` list is written in its new order,
so the page without its script ends there too. `tilt` belongs to a slide
and does nothing on a page; plain `<!-- steps -->` stays deck-only. Print,
reduced motion and a page without its script show every block as written.

```markdown
<!-- animate: count -->
**1,842** declines between 21:00 and 06:00

<!-- steps: deal -->
- Split every failure by its real cause
- Make the retry idempotent first
- Measure the window, then wait a little longer
```

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
- **Presenter-tool cues: `[TERM Name: command]` and `[BROWSER Name]`.** Where
  the script says them, `si play`'s own tools act (§14): `[TERM Lab 1:
  docker compose up -d]` opens the terminal drawer on the shell tab `Lab 1`
  (created when missing, raised when it is there) and runs the command;
  `[TERM Lab 1]` only opens the tab; `[TERM min]`, `[TERM max]` and
  `[TERM close]` set the drawer. Options after the name, each after a `·`:
  `· split right` (or `down`) opens a new tab beside (or under) the one in
  front, `· max` or `· min` sets the drawer once the tab is open,
  `· opacity N%` (40 % to 100 %) its opacity (the slide showing through),
  `· typing slow` or `· typing fast` how fast the command is typed (90 or
  20 ms a character; 45 by default): `[TERM Lab 2 · split right · typing
  slow: make up]`, `[TERM Lab 1 · max · opacity 80%: docker compose logs -f]`. `[BROWSER RabbitMQ]` opens the deck's
  `browser:` link named `RabbitMQ` (or a URL) in a window over the slides;
  after the name, `· at X,Y` (CSS pixels from the top left), `· size WxH`
  and `· zoom N%` (30 % to 300 %) place, size and zoom it, as a web demo's
  `embed-browser` does (§27): `[BROWSER RabbitMQ · at 560,40 · size
  1010x640 · zoom 125%]`; on a window already open they move it. `[BROWSER
  RabbitMQ: http://localhost:15672/#/queues]` takes the open window
  `RabbitMQ` to that page (an address relative to its page, `#/queues`,
  works too) instead of opening another; with no such window it opens the
  link, then goes there. `[BROWSER close]` closes every window. Closing the
  drawer or a window gives the keys back to the slides (→ moves on again).
  Like `[CLICK]` they are never read
  aloud: the teleprompter (and `si play --terminal --notes`) shows each as a
  badge with its text, auto-play in
  `si play` acts on them as it reaches them, and `si build talk.md --mp4
  --voice` films them at that point of the narration (without `--voice`,
  where the words before them would end read at auto-play's pace, 150 words
  a minute, fitted into the step's time; the deck then plays on
  `si play`'s page, the drawer and the windows in the video; the deck's
  `setup:` and `start:` run first, §14). They are no
  advance: they do not count as `[CLICK]` cues. `si check` reports a
  malformed cue (a window option it does not know too) and a `[BROWSER
  Name]` with no such `browser:` link (one
  with `/`, `.` or `:` is a URL). A cue ends at the first `]`.

  ```markdown
  Notes: (1 min)
  Let's see it in action. [TERM Lab 1: docker compose up -d --wait] The broker
  is up. [BROWSER RabbitMQ · at 560,40 · size 1010x640 · zoom 125%] Here is
  the queue, filling as the producer runs.
  [BROWSER RabbitMQ: http://localhost:15672/#/exchanges] And the exchange
  that feeds it. [TERM close] [BROWSER close] [CLICK]
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
pauses where you are, and Resume carries on from there: on the same slide
and build step it goes on mid-part, the voice file from the second it
stopped at and the browser's own voice from the sentence it was saying (it
speaks a part one sentence at a time), instead of starting the part over;
after a move to another slide or step, that part starts from its
beginning. The teleprompter shows *auto-play* beside the slide number and
follows along.

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
si play talk.md --remote                    # every role (tp, draw, clicker, monitor)
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

**The control room.** `si play` prints a keyed link to a web monitor
(`http://127.0.0.1:7070/_si/monitor?k=…`, this machine only; `--remote`
offers it to a tablet as the `monitor` role). It shows who is watching by
role, the slide and its timers, the links and their QR codes, and runs the
room: close, reopen or reset a poll and reveal a `· hidden` poll's results
without the click (the phones and the slide show them at once); approve a
question, put it on screen, mark it answered or hide it; hide a word from a
cloud or clear it. A closed poll takes no more votes; a reset starts a new
round and the phones' earlier picks are cleared. The states go into the
`.si/audience/` file.

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

A table of numbers can also be a chart that moves step by step: a
```` ```chart ```` fence (§33).

- Fenced code with a language gets a small built-in highlighter (Go, Python,
  JavaScript, TypeScript (`ts`, `tsx`: types, `interface`, `readonly`,
  `keyof`), Java, Kotlin (`kt`, `kts`: `data`, `suspend`, `companion`…), C# (`csharp`, `cs`, `c#`), C, C++,
  PHP, Ruby, Swift, Scala, Rust (`rs`: lifetimes `'a`, macros `println!`,
  the number types), SQL, YAML, TOML, JSON, Bash, PowerShell,
  Dockerfile, Terraform/HCL (`hcl`, `tf`), Protobuf (`proto`), GraphQL (`gql`),
  F# (`fsharp`, `fs`), Elixir (`ex`, `exs`), Groovy and Gradle (`groovy`,
  `gradle`, `jenkinsfile`: Gradle words like `implementation` and Jenkins
  words like `stage` coloured as calls), Kotlin scripts (`kts`), Erlang
  (`erlang`, `erl`: RabbitMQ's `advanced.config` and `enabled_plugins`),
  Makefile (`make`), nginx, Markdown (`md`), Windows batch (`bat`, `cmd`),
  INI and `.properties`, CSS, HTML, XML (also `csproj`, `props`, `xaml`, `svg`, `pom`,
  `nuspec`: tags, attributes, comments)) and a language label written as a
  reader expects it (C#, JavaScript, Shell, XML…). Keywords, strings,
  comments and numbers are coloured, and so are calls (the name before a `(`)
  and, in C#, Java, Go, Kotlin, Swift, Scala, JavaScript, TypeScript, Groovy,
  C++, Rust and PHP, types (PascalCase names); annotations and decorators
  (`@Override`, `@Bean`, `@Component`) in Java, Kotlin, TypeScript, Groovy and
  Scala; other languages get strings, calls and numbers.
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
  dir: LR                              # LR (default), RL or TB
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
  keeps the full size. A long edge label (`-->|order.*.created|`) widens the
  gap between its two columns so it clears the boxes and the arrowhead; one
  longer than twice a box is set smaller. With `dir: TB`, labels on sibling
  edges sit side by side, so the gap between the siblings widens until they
  clear each other (up to twice a box; past that they are set smaller). A
  label on a slanted edge (a fan-out, a merge) is turned to the edge's angle
  and set just above it, always upright; labels on level and upright edges
  stay level. `labels: flat` keeps every label level, and `dir: RL` flows
  right to left:

  ```stage
  dir: LR
  CP[Control plane] -->|most free RAM| VM1[VM 1]
  CP -->|place session| VM2[VM 2]
  ```

  Add `step <node...>` lines to walk the flow on build steps — each
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
- Every code window option works on an include too (`window=`, `theme=`,
  `backdrop=`, `numbers`, `id=`, `step=`, `transition=`, `as=cat`): see
  "Code from a file" in §25. `numbers` is `linenos`. An include of a `.md`
  file with `window=` or `id=` shows the Markdown as code.
- `si check` reports a missing file, a range past the end, a tag that is not
  found or not closed, and an unknown option (listing every option).
- A title with spaces is quoted: `title="Retry (part one)"`. Unquoted, its
  words read as options, and `si check` says so: `title with spaces must be
  quoted: title="Retry (part one)"`.

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
`# <1>`, `-- <1>`, `/* <1> */`, `<!-- 1 -->`), and write the explanations
right after the block, one `<N> text` line each. A line that does not start
with a marker carries on the explanation before it, and AsciiDoc's
self-numbering `<.>` counts up from 1, in the code and in the list:

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

The block and its list are one card: a `<figure class="si-codecallout">`
holds the code and the `<ol class="si-callouts">`, with one border and one
dashed line between them. The list takes the code's width (a long
explanation wraps, it never widens the card), a long code line scrolls
inside the card, and on paper the card is kept on one page. It is plain
markup, so a static viewer or a PDF shows the same card. Style it in your
CSS through `.si-codecallout` and `.si-callouts`.

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
  `title=`, `lang=`, `linenos`, and the code window options: `window=mac`,
  `id=`, `numbers`… of §25); a `#L5-L10` in the permalink is `lines=5-10`.
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
  Chromium installed; without it, HTML is still written). Every picture, PDF
  and page export takes one of a few Chrome slots shared by the whole
  machine (a quarter of the CPUs, at least two; lock files in the temp
  folder), so two builds, or a build beside `go test`, take turns instead of
  starving each other. A run that times out on a busy machine before its
  file is written gets one more try, and the error says how long it waited
  for a slot.
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
  A deck with `[TERM …]` / `[BROWSER …]` cues (§11) is filmed on `si play`'s
  page, with its `setup:` and `start:` run first and `stop:` / `cleanup:`
  after (§14; `--no-live` skips them). Two builds may film the same deck at
  once: each narrates into a file of its own.
- **Open on a phone**: run `si play <dir> --addr 0.0.0.0:7070` to serve on your
  network; `si play` then prints a LAN URL and a QR code to scan. (The default
  address is loopback-only, so it prints the `--addr` hint instead.)

- **Play terminal** (`` ` ``, backtick): a drawer of real shells on your machine
  for live demos — play-only and loopback-only, never shipped in a built deck.
  `shell:` in the front matter picks its shell: `shell: zsh` (a name on `PATH`
  or a path), `shell: clean` (si's neutral prompt, the folder's name and `$`,
  no rc files, as when filmed), or both, `shell: bash · clean`.
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

- **Browser windows** (key **w**, or **Window** in the bar; `si play` on a
  loopback address only, never in a built deck; the bar also gets
  **Terminal** for the drawer): web pages in floating windows over the slides, for a
  broker's console or a dashboard on a wide screen. **w** opens a picker with
  the deck's `browser:` links, the recent addresses and a box to type one.
  Drag a window by its title bar, resize it from any edge or corner, **←** /
  **→** go back and forward in that window's page (dimmed when there is no
  page that way; the address box follows the links you click inside), **−** /
  **+** zoom that window's page alone (click the percentage for 100%),
  double-click the bar (or **□**) to maximise, **–** to minimise it to a pill in
  the bottom-left dock, **✕** to close; several windows at once, the one you
  touch comes to the front, and their places survive a live reload. Each page
  is shown through a loopback proxy si play starts for its origin, so consoles
  that refuse to be framed (X-Frame-Options, CSP `frame-ancestors`) still
  show. A window keeps that across sites: a link to another origin (the
  console's "Docs", a login page) and a redirect to one open through that
  origin's own proxy, the address box always shows the real address, and
  **←** / **→** go back and forward across the sites. A page's own script
  that sends it to another site stays in the window too: `location.assign`,
  `location.replace`, `location.href = …`, `window.open` and forms (in
  Chrome, Edge and other browsers with the Navigation API; elsewhere only
  `window.open` and GET forms are caught, since a page cannot replace
  `location`'s methods). A form that POSTs to another site (a login page)
  arrives there with its fields, in every browser: si sends it on through
  that site's proxy. A page's own `fetch` and `XMLHttpRequest` calls to
  another site (an API on another port) go through the page's proxy too, so
  they get their answers whatever that site's CORS rules; they carry no
  cookies there and bring none back. Type the address in the box when a page
  still leaves.
  Keys typed in a window never drive the deck.

  ```markdown
  ---
  browser: RabbitMQ=http://localhost:15672, Grafana=http://localhost:3000
  ---
  ```

- **The presenter tools API, `window.siPlay`** (`si play` on a loopback
  address only, never in a built deck): what web demos (§27 `embed-terminal`,
  `embed-browser`, `slide`), a deck's `[TERM …]` / `[BROWSER …]` cues (§11)
  and auto-play use to drive the drawer, the browser windows and the deck by
  name. It is a contract (version 1; si's markup and CSS classes are not):
  a script of your own in the page, or a test, may call it too. Calls that
  open or type return a Promise; `output`, `frame`, `names` and `at` answer
  at once.

  | Call | What it does |
  |---|---|
  | `siPlay.terminal.open(name, {split})` | opens the drawer on the shell tab `name` (created when missing); `split: 'right'` or `'down'` puts it beside the tab in front; resolves when its shell has drawn its prompt |
  | `siPlay.terminal.run(name, command, {typing})` | runs a command in that tab (pasted; `typing: 55` types it, 55 ms a character) |
  | `siPlay.terminal.type(name, keys, {typing})` | sends keys as they are (`'\r'` Enter, `'\x03'` Ctrl+C) |
  | `siPlay.terminal.output(name)` | the tab's screen and scrollback as text (`''` for no such tab) |
  | `siPlay.terminal.state(s)` | the drawer: `'min'` (hidden, the shells run on), `'max'`, `'restore'`, `'close'` |
  | `siPlay.terminal.names()` | the open shell tabs |
  | `siPlay.terminal.opacity(n)` | the drawer's opacity in %, 40 to 100 (remembered; the slide shows through, the text stays opaque); no `n`: the current one. `open(name, {opacity})` sets it too |
  | `siPlay.browser.open(name, {x, y, w, h, zoom})` | a window with the `browser:` link `name` (or a URL), placed in CSS pixels, zoom `1.25` = 125 %; an open one comes to the front; resolves when its page has loaded |
  | `siPlay.browser.go(name, url)` | takes the open window `name` to `url` (relative to its page: `'#/queues'`); resolves when it has loaded (5 s at most) |
  | `siPlay.browser.close(name)` · `.min(name)` | closes (no name: every window) or minimises it; the keys go back to the deck |
  | `siPlay.browser.frame(name)` | the window's `<iframe>`, also `iframe[data-si-window="name"]` |
  | `siPlay.browser.names()` | the open windows |
  | `siPlay.slide(n, step)` · `.next()` · `.prev()` | moves the deck (slide from 1) like the keys; resolve with `siPlay.at()` |
  | `siPlay.at()` | `{slide, step, slides}` |
  | `siPlay.act(cues)` | runs cues as `[TERM …]` / `[BROWSER …]` do: `[{tool: 'term', name, cmd}, {tool: 'browser', name, o: {x, y, w, h, zoom}, url}, {tool: 'term', state: 'close'}]` |

- **The tools bar**: the play terminal's tools sit in a narrow bar of icons
  on its right edge, as in an editor: **Demos** (▷), **Runbook** (the book)
  and **Viewer** (the page). Demos and Runbook open as a side panel (one at a
  time; click the icon again to close it) that resizes by dragging its left
  edge; the width is remembered. **+** (new shell) and **✕** stay at the top.
  At the top left, **–** minimises the drawer (**`** brings it back, shells
  still running) and **□** maximises it to the whole window (or double-click
  the bar; **❐** restores; remembered). Drag the drawer's top edge to set its
  height (a fifth of the window up to all of it; remembered; maximise comes
  back to it). A shell always knows the size its tab shows: it starts at it,
  and every change (the window, the drawer's height, maximise, a side panel,
  a split, the text size) refits the tab and tells the shell before the next
  command goes in, so a long command line wraps where you see it wrap.

- **Drawer opacity**: the slider in the top bar (40 % to 100 %, remembered)
  lets the slide show through the drawer: the backgrounds go translucent and
  the slide behind is dimmed so the text, always fully opaque, keeps its
  contrast over a light slide. `terminal.opacity: 85%` in si.conf (§16) sets
  the starting value (`terminal.blur: 6px` blurs the slide behind too); a
  cue (`[TERM Lab 1 · opacity 80%]`, §11), a web demo step
  (`embed-terminal "Lab 1" · opacity 80%`, §27) and
  `siPlay.terminal.opacity(80)` set it while you present.

- **Text size per tab**: **−** **100%** **+** in the top bar size the active
  tab only, a shell's text or a Viewer's tree, file and editor, never the
  other tabs (click the percentage for 100%); **⌘=** / **⌘-** / **⌘0**
  (Ctrl on Windows and Linux) do the same while you are anywhere in the
  drawer, instead of zooming the page (Chrome, Edge and Firefox let the page
  keep these keys; a browser that zooms anyway still has **−** / **+**). The
  size you used last is where the next tab of that kind starts.

- **Shells and Viewers survive a reload.** Saving the deck reloads the page;
  the drawer comes back as it was: open or closed, the same tabs in the same
  order, the active one, each shell with its recent screen (the last 256 KB
  of its output) and still running (a `sleep 600` keeps sleeping), each Viewer
  on its folder and file, its tree with the same folders open and scrolled
  where it was, in the editor with unsaved changes kept, Follow on or off,
  each tab's text size. A shell whose program ended is dropped; a
  shell no page has shown for 10 minutes is closed; all close when si play
  stops.

- **Viewer tabs** (the Viewer icon in the tools bar): each click opens
  a new tab beside the shells, a tree of a folder and the file you pick
  (read-only until **Edit**), so several files can be open at once (the tab shows `Viewer · name`).
  A Viewer opens on the last folder used (first time: the deck's folder);
  type a path (`~` works) and Enter, **↑** for the parent, double-click a
  folder to make it the top, **⌂ Shell** for the folder the last shell you
  used is in now. Files show highlighted with line numbers (the first MB;
  pictures shown; binary files named, not shown); **Copy path** copies it.
  Runbook and Demos Run buttons still type into the last shell.
  **Edit** turns a text file (up to 1 MB) into an editor: syntax colours by
  file name, line numbers, bracket matching, ⌘F search, undo, Tab indents,
  **Save** or ⌘S (refused when the file changed on disk since you opened
  it: **Reload** or **Overwrite**), **Revert**, **Done** for the read-only
  view again. Unsaved changes put a dot in the tab (`Viewer · app.yaml ●`);
  closing the tab or opening another file asks **Save**, **Discard** or
  **Keep editing**. A save never reloads the deck page. Colours for JS, TS,
  JSX, TSX, JSON (with errors marked), Markdown, CSS, HTML, XML, YAML, shell
  (`.sh`, `.zsh`, `.env`, Makefile), Python, Go, Java, Kotlin, Scala,
  Groovy, Rust, C, C++, C#, Swift, Ruby, Perl, R, Lua, SQL, TOML,
  INI/properties/`.conf` (`rabbitmq.conf`), Erlang (`.erl`,
  `advanced.config`), Dockerfile, nginx, PowerShell, protobuf, CMake and
  diff. It colours and checks JSON; it is not a language server (no
  completion or go-to-definition). `.conf`, INI, `.properties` and
  `.cfg` files colour their values in both views: keys, `[sections]`,
  numbers, sizes and durations (`512MB`, `30s`), booleans
  (`true`/`off`/`yes`), quoted strings, paths and URLs, `#` and `;`
  comments.
  The tree has a toolbar (and a right-click menu on its rows): **New file**
  and **New folder** (in the folder you picked, or the top; a new file opens
  in the editor), **Rename** (F2, in place), **Delete** (Delete key; a bar in
  the pane names what goes, a folder with the number of items in it; a link
  is deleted, never what it points to), **Copy path** and **Reveal path**. A
  name already taken is refused. **Follow** (read-only view, off by default)
  shows the file as it changes on disk, once a second, scrolled to the end
  when you were at the end, like `tail -f`; a file over 1 MB shows its last
  MB, numbered from its real line (the header says `lines 48210–71034`). It
  pauses while the tab is hidden. While you edit, a change on disk
  shows a notice (**Reload** or **Keep editing**), never a refresh under
  you. `.log` files never live-reload the deck.
- **Split view, quick open, search, spotlight** (the workbench, v3.9.0).
  **◫** in the top bar (or right-click a tab: **Split right**, **Split
  down**) shows two tabs at once: the active one and the next (a new shell
  when there is only one). **Shift-click ◫** (or right-click a tab: **Show
  in a new pane**, **Add a pane**) adds a pane, up to four: three are one
  big pane and two beside it (or under it, **Stacked**), four a 2×2 grid.
  Each pane shows one tab; clicking a tab shows it in the focused pane (the
  one with the blue edge; click inside a pane to focus it), dragging a tab
  onto a pane puts it there, the right-click menu has **Show in the other
  pane** (with more panes, **Show in pane 3**), **Swap the panes** (**Rotate
  the panes**), **Stacked** / **Side by side**, **Close this pane** and
  **Close the split**; **◫** again closes it. Drag the dividers to size the
  panes (shells refit as you drag). A narrow drawer (a phone) stacks them.
  The split, its sizes and which tab is where survive a reload.
  **⌘P** (Ctrl+P; in a shell on Linux and Windows Ctrl+P stays the shell's)
  or the **Go to file** icon: type letters of a file's path (`chkgo` finds
  `src/orders/checkout.go`) in the active Viewer's folder (from a shell: the
  deck's folder); **Enter** opens it in the active Viewer (from a shell: a
  new one), **⌘Enter** in a new Viewer. **⌘⇧F** or the **Search** icon:
  search the text of the files in the same folder (**Aa** match case, **.\***
  a regular expression); results grouped by file with the matches marked;
  click one to open the file in a Viewer at that line, marked. Results come
  in as they are found; **Stop** (or Esc) ends a long search and keeps what
  came. Both skip
  `.git`, `node_modules`, `vendor`, `dist`, `build` and dot-folders (start
  the query with `.` to look in dot-folders); search reads text files up to 1 MB and
  shows the first 1000 matches. **Spotlight**: in the read-only view, click a
  line number to light that line for the audience (the rest dims),
  Shift-click another to light the lines between, the same number again or
  **Esc** to clear; in the editor, select lines and press **Spotlight**
  (**Esc** clears). On screen only: nothing is saved.

  ```text
  shell 1 │ Viewer · checkout.go                ◫ +
  $ go test ./orders │ 75  // the order is paid
  ok  orders 0.4s    │ 76  // so we publish it
                     │ 77  func Checkout(o Order) error {   ← lit
  ```

  ```text
  Viewer · rabbitmq.conf ●     Save  Revert  Done  Copy path
  1  # rabbit
  2  listeners.tcp.default = 5672
  3  vm_memory_high_watermark.absolute = 512MB

  Viewer · app.log      [Follow]  Edit  Copy path
  87  2026-11-13T10:01:00 WARN appended 87
  ```

- **Demos** (front matter `demos: ../demos/`): demo scripts, plain markdown, in a
  second side panel of the play terminal (the Demos icon, one panel open at a time).
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
  exits 1. `si build` does not run them (a built deck is a file), except
  to film the deck's `[TERM …]` / `[BROWSER …]` cues (§11): `si build
  talk.md --mp4` of a deck with cues runs `setup:` and `start:` (in order,
  with their waits) before it films, so the cues show the live system, then
  `stop:`, the programs' end and `cleanup:` (always, a failure too), as
  `si play` would; `--no-live` films without them. A course build filming a
  stale `kind=video` deck does the same (`--no-live` and `--trust` pass on).
  `si check` checks each line and the pin of each source. A web demo script (§27) has its
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
  (with the pictures its `files:` key lists, each with its own fingerprint;
  `gist:ID@REV/talk.md` brings the whole gist). The same trust rules hold:
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
| `terminal.opacity: 85%` | the opacity of `si play`'s terminal drawer, 40 % to 100 % (default 100 %): below 100 % the slide shows through, dimmed, and the text stays opaque; the drawer's slider changes it (remembered until this value changes). `terminal.blur: 6px` blurs the slide behind it too (§14) |
| `terminal.prompt: clean` | the shells of `si play`'s terminal drawer: `real` (default: your own `$SHELL` with its rc files and prompt) or `clean` (si's neutral prompt, the folder's name and `$`, no rc files; `PATH` and the environment kept). A deck asks for its own with `shell:` in its front matter: `shell: zsh` (a name on `PATH` or a path), `shell: clean`, `shell: bash · clean` (§14). Filming always uses `clean`: `si demo test`/`run` and `si build --mp4` with `[TERM …]` cues |
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
| `course.webhook: https://hooks.example.com/si` | where every course of the project sends its result events (a `_course.md` `webhook:` wins; §28) |
| `course.base: https://learn.example.com` | where the project's courses are published: their event and xAPI activity ids sit under it (a `_course.md` `webhook-base:` wins; §28) |
| `webhook-secret: …` | signs what `si serve` / `si play` forward through `/_si/relay` and what `si demo test --webhook` posts (`X-Si-Timestamp`, `X-Si-Signature` over `"<timestamp>.<body>"`); only in your own si.conf (a project's is ignored with a warning), or `SI_WEBHOOK_SECRET` |
| `voice.Ada: kokoro:af_heart` | auto-play: one speaker's voice in a deck with `speakers:` (front matter `voices:` wins) |
| `autoplay.dwell: 5` | auto-play: seconds on a slide with no `Notes:` and no `(N min)` (default 3) |
| `structure:` + an indented block | where `si build` writes each kind of output: one folder per line (§31) |

Nesting is written as dotted keys (`timer.animation.path`), not YAML. The one
exception is `structure:`, an indented block of folders (§31).

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
- `<!-- check: <command> · expect "text" -->` under a step shows how to
  check it; in `si play` a **✓ Check** button runs it (§28, Labs that check
  themselves). `<!-- solution -->` … `<!-- end -->` folds the step's answer.
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
  and links it from the cover and the bar: A4 pages, the cover and a title
  page (`_cover.md` and the `copyright:` line) unnumbered, the contents with
  the page of every chapter and section (si prints twice: once to learn the
  pages, once to write them in), a page of its own for each part, every
  chapter on a new page with its title at the top of each page and the page
  number at the bottom, figures, tables and code kept whole, a PDF outline.
  Code callouts, charts and scenes print in their final state. Needs Chrome.

  ```bash
  si build book/ -o html/ --pdf        # html/book.pdf
  ```
- **Export for a publisher.** The same book, read once, as Markdown, Word,
  RTF or EPUB (any of them, together; written beside the HTML). Chapters,
  appendices, sections, figures and tables keep their numbers, empty
  cross-references are filled ("Table 1.1"), `((index terms))` become the
  plain word, `<!-- … -->` directives and raw HTML are dropped, code callouts
  become `(1)` markers and a numbered list after the block, alerts (and
  AsciiDoc admonitions) a quote with a bold **Note:** label. Works the same
  from `.adoc` chapters.

  ```bash
  si build book/ -o out/ --md      # out/md/book.md (the whole book, a pandoc title block first),
                                   # out/md/<chapter>.md one per chapter, out/md/images/
  si build book/ -o out/ --docx    # out/book.docx: Word styles (Title, Heading 1-3, Code, Caption,
                                   # List Bullet/Number, Quote, Table Grid), a contents field Word
                                   # fills on open, each chapter on a new page, page numbers
  si build book/ -o out/ --rtf     # out/book.rtf: headings, lists, tables, code, pictures
  si build book/ -o out/ --epub    # out/book.epub: EPUB 3, one file per part and chapter
  ```

  Pictures travel as files (`image::` and `![]()` alike, and the SVG of a
  `<!-- svg: -->` canvas); Word and RTF get an SVG as a PNG Chrome draws
  (without Chrome, a placeholder line). Figures si draws itself (`chart`,
  `scene`, `stage`, `mindmap`, `quiz`, `video` fences) become a placeholder
  line naming the figure; a `mermaid` fence stays a fence in Markdown. A
  guide (`_nav.md` without `type: book`) exports the same way as `guide.*`,
  without numbers, and an article as `<name>.md`, `<name>.docx` and so on
  (`si build note.md --docx`). A deck, a site or a course says the flags are
  not for it.
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

On a phone (a view 480 px wide or less, or any upright one: a tablet held
upright too) the 3D view has its own layout: every branch on one side of the
central topic, the camera in front of it so the map runs down the screen and
the labels read level, a wider lens in perspective, and linked maps lined up
past the far end of the map, low, so they never cover it. Fit shows the whole
map while a first-level label stays at least 12 px tall; past that, the map
without its floors, then the central topic and its first ring (pan or pinch for
the rest). The tools turn into finger-sized icons that wrap (⌕ Find, ▶ Play,
◈ camera, ▦ kind, ☰ List, ⤢ Fit; 2D, + and − keep their text), and a topic's
card opens at the top, full width. Touch: one finger orbits, two fingers pinch
to zoom and pan, a tap on a topic opens its card (a pinch never does). The
**List** tool shows the map as the nested list it was written as, and **Map**
brings the 3D view back.

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

Exports draw the 2D map, with its theme, lines, moved topics and watermark;
the PDF also has the linked maps, and a `map ›` topic there jumps to its page.

A `view: 3d` page exports its 3D view instead: `--png` writes `map.png`, the
first map with its floors at the default camera, in the page's kind, camera
and theme (1600 × 1000 at `--scale`, up to 4×), rendered by Chrome's software
WebGL. A page of several maps also gets one picture of each map alone, no
floors: `map-<id>.png` (`map-launch.png`, `map-refunds.png`).
`--pdf` writes `pdfs/map.pdf`, one page of the first picture. The 3D view has
no SVG: it is raster only, so `--svg` on a 3D page writes the 2D maps
(`map-<id>.svg`) and says so.

```bash
si build payments.md --png --pdf --scale 2   # view: 3d → payments.png, pdfs/payments.pdf
```
 They need headless Chrome, like `--pdf` for decks.
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
si mindmap export map.md --to xmind8 -o map-x8.xmind       # the legacy XMind 8 file
```

Import keeps the topics, their order and web links; every XMind sheet becomes
a map (XMind and XMind 8 files alike). Export writes one map (the first, or
`--map id`); XMind takes every map, one sheet each. `xmind8` writes the XMind 8
file (a zip of `content.xml` in the xmind-2.0 namespace, `meta.xml` and
`META-INF/manifest.xml`) for XMind 8 and the tools that read only it: badges
become labels, web links and `→ #id` links to another map (the central topic
of that sheet) become topic links, and a code topic becomes a note on its
parent (plain text, and HTML paragraphs in monospace that XMind 8 shows
formatted); importing it back gives the badges and map links again. Pictures,
sticky-note looks and the other formats' missing pieces stay in si.
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
| `shadow=` | `lifted` (default), `soft`, `deep`, `none` | how far the window floats off the page: a drop shadow and a faint edge that show in light and dark (a backdrop gives a deeper one unless `shadow=` is set) |
| `numbers` | (a flag; `linenos` works too) | line numbers in a gutter |
| `start=` | a number (default 1) | the gutter's first number: `numbers start=120` for a block cut from line 120 of a file |
| `hl=` | `3`, `3,5-7` | highlighted lines of this step |
| `id=` | a name | blocks with the same id in a row are one show; on slides, the same id on consecutive slides morphs |
| `step=` | text | the step's name, shown under the window (`2 / 4 · Back off`) and in the Studio |
| `transition=` | `morph` (default), `fade`, `slide` | how a show goes from step to step; on a slide the window keeps the tallest step's height (and room for the longest list of notes on a phone), so a step moves only the code, never the title or text around it |
| `as=` | `cat` | a shell script in a `terminal` window shows as a file, under `$ cat <title>` |

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

An unknown value of `window=`, `theme=`, `backdrop=`, `transition=` or `as=`
is a `si check` problem.

### Code from a file

A window can show a real file: give the include (or a `<!-- code: -->`
permalink, §13) the same options as the fence. It draws exactly the window
the fence would, with the file's text, highlighted by its name (`.yaml`
`.yml`, `.sh` `.bash` `.zsh`, `.json`, `.toml`, `.ini` `.conf` `.properties`
`.env` (also `rabbitmq.conf`), `Dockerfile` `Containerfile`, `Makefile`,
`.sql`, `.xml`, `.md`, `nginx.conf`, `.tf`…; `lang=` overrides). The title is
the file name unless `title=` says otherwise, and `lines=` / `tag=` keep their
meaning: with `numbers`, the gutter counts from the first line shown
(`lines=40-52` numbers 40 to 52), as the file does.

```markdown
<!-- include: compose.yaml window=mac theme=night numbers hl=6 -->
<!-- callout: 6 The same five tries as production. -->

<!-- include: start.sh window=terminal as=cat numbers hl=4-5 -->
```

Includes with the same `id=` are a show like fences: two versions of a
file (`start-v1.sh`, `start-v2.sh`, both `title=start.sh`), or one file's
`tag=` regions, morph step to step, on slides (one click each) and in
documents (← → Play). Callouts go after the include line. In `si play` an
edit to the included file reloads the page, also for a file outside the
served folder (`../code/compose.yaml`). The Studio's Window view edits the
include's options and callouts (its code is the file: edit that), and
`si code` exports it.

A shell script in a `terminal` window shows only its lines, like a command
typed at the prompt; `as=cat` shows it as a file instead, under
`$ cat start.sh`. It works on a fence too.

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
window with C# and a `.csproj`, `compose.yaml` and `start.sh` shown from
their files, a terminal), the showcase slide "The retry
rule" (two steps, one click), and a terminal window in
`examples/article/retry.md`.

### Video player

A fenced code block whose language is `video` is a video player, authored
like a code window: the options go on the fence line, any body is the
caption. The player has its own controls (the HUD), themed with the page's
`--si-*` tokens: play and pause, a scrub bar with the buffered part, the time
under the pointer and chapter marks, the time, volume and mute, speed (1×,
1.25×, 1.5×, 2×), captions, a chapter list, picture in picture where the
browser has it, and fullscreen. Before the first play it shows the poster
with a big play button; while it plays the HUD hides itself until the
pointer moves. `si build` copies the video beside the page (as it does for
`![](clip.mp4)`, §8) and inlines the player's CSS and JS; nothing external
loads.

````markdown
```video src="assets/demo.mp4" poster="assets/demo.jpg" mode=modal title="The walkthrough"
Two minutes, end to end.
```
````

- **`mode=inline`** (default) — the player in place.
- **`mode=modal`** — the poster is a button; it opens the player large in a
  dialog over the page, which is dimmed and blurred behind it. Esc, the ✕
  button or a click on the dimmed page closes it (and pauses); focus stays in
  the dialog while it is open and goes back to the poster after. Every modal
  player on the page is one gallery: the dialog shows "2 / 9", the title and
  caption under the video, and ‹ › to step through them. `group=` makes
  separate galleries.
- **`mode=facade`** — like inline, but nothing loads until the first play;
  good for a page with many clips.

A block with only a `poster=` and no `src=` degrades: the poster shows with
the play control disabled, rather than erroring.

| Option | Values | What it does |
|---|---|---|
| `src=` | a video file or URL (mp4, webm, mov, m4v, ogv) | the video; omit it (with a `poster=`) for a disabled placeholder |
| `poster=` | an image file or URL | the still before play and the modal thumbnail; default `<video>.poster.jpg`, made from the first frame when ffmpeg is there |
| `title=` | text (quote it when it has spaces) | the accessible label, the poster's alt text, the title under the video in the dialog |
| `mode=` | `inline` (default), `modal`, `facade` | how it plays (above) |
| `captions=` | a `.vtt` or `.srt` file | the captions; default: `<video>.vtt` (or `.srt`) beside the video |
| `chapters=` | a `.txt` (`0:00 Title` lines) or `.vtt` file | marks on the scrub bar, the chapter list, the chapter's name in the HUD; default: `<video>.chapters.txt` (or `.chapters.vtt`) beside the video, or a `chapters.txt` beside a video alone in its folder (as `si demo run --with chapters` writes it) |
| `group=` | a name | `modal` only: players with the same group share the dialog's ‹ › (default: every modal player on the page) |
| `start=` | a time (`12`, `1:05`, `1:02:03`) | starts there |
| `hud=` | `on` (default), `native`, `off` | si's controls, the browser's own, or none (a click plays and pauses) |
| `autoplay` | a flag | `inline` only, always muted (browsers block sound) |
| `muted`, `loop` | flags | starts muted; loops |
| `lang=` | a language code | the captions' language (default: the page's `lang:`) |
| `width=` | a length (`40rem`, `70%`) | caps the player width |
| `aspect=` | a ratio (`16/9` default, `4/3`, `1/1`) | the player's aspect ratio (`ratio=` works too) |

Captions and chapters are read at build time and carried in the page (the
captions as a `data:` track), so a built page shows them from `file://` too.
`![caption](clip.mp4)` in an article, guide, book or lab (§8) and a course's
video lessons (§28) play in the same player and find the same files beside
the video, so the videos si films (`si demo run`, `si build --mp4 --voice`)
get their captions and chapters with nothing to write. On slides,
`![](clip.mp4)` keeps the browser's own player; use the fence for the HUD.

**Keys** while a player has focus (click it or Tab to it):

| Key | What |
|---|---|
| Space, K | play / pause |
| ← → | back / forward 5 s |
| J L | back / forward 10 s |
| Home End | start / end |
| M | mute |
| F | fullscreen |
| C | captions on / off |
| Esc | close the chapter list, leave fullscreen, close the dialog; on a slide, give the arrow keys back to the deck |
| ← → in the dialog | the previous / next video while paused; Shift+← → while playing |

On a touch screen a tap shows or hides the HUD and a double tap on the left
or right third seeks 10 s. Buttons carry labels for screen readers, focus is
always visible, and `prefers-reduced-motion` turns the HUD's animations off.

An unknown option, `mode=` or `hud=`, a `start=` that is not a time and a
missing captions or chapters file are `si check` problems. The player's look
is in `--si-vp-*` tokens (`si --export css-rules` lists them).

**On pages si did not build.** `si --export player [dir]` writes
`si-player.css` and `si-player.js` (the same two files si inlines), a README
and `demo.html`. Load the two files on any page and every element with
`data-video` becomes a player: a card with an `<img>` uses it as the poster,
its first heading as the title and its first paragraph as the caption, and a
click (or Enter) opens the dialog, with ‹ › through every card on the page.

```html
<link rel="stylesheet" href="si-player.css">
<script src="si-player.js" defer></script>

<figure class="video-card" data-video="clips/01-intro.mp4" data-captions="clips/01-intro.vtt"
        data-chapters="clips/01-intro.chapters.txt">
  <img src="shots/01-intro.png" alt="The complete film">
  <figcaption><h3>The complete film</h3><p>The whole story, end to end.</p></figcaption>
</figure>
```

| Attribute | What |
|---|---|
| `data-video` | the video file |
| `data-poster` | the still (default: the `<img>` inside) |
| `data-title`, `data-caption` | the title and text under the video (default: the heading and paragraph inside) |
| `data-captions` | a WebVTT file |
| `data-chapters` | `0:00 Title` lines or a WebVTT chapters file |
| `data-mode` | `modal` (default) or `inline` (plays in place of the picture) |
| `data-group` | separate galleries |
| `data-start` | a start time |

The drop-in fetches captions and chapters, so serve the page over http. Its
colours follow `--si-accent` when the page sets it; `window.siPlayer.init(el)`
wires players added after the page loaded.

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
  (no cookies or storage from the one before; `session: keep` keeps one
  profile for the whole run, see Promo kit), after `setup:` and with the
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
| `lower-third "Ana Ruiz" · "Head of Support"` | a name caption at the bottom left of the video from this step on, 4 s (`· for 6s`); nothing in a test (see Promo kit) |
| `click "Buy" in frame .preview` | any element step reaches into an iframe: `in frame` and the iframe's CSS selector, or its `"name"` (name, title or id); same-origin and cross-origin frames |

**Elements** are found the way a person names them: `"Pay"` matches visible
text, a `<label>`, an `aria-label`, a placeholder, a title or an alt text
(exact first, then any case, then a part). An element that starts with `#`,
`.` or `[`, or with a tag followed by CSS (`button.pay`, `li:first-child`,
`nav > a`, `form .name input`, `main #cart button`), or with two HTML tags
(`form input`, `nav a`), is a CSS selector, spaces included (plain words that
are not tags, `Pay now`, stay a name to quote): it runs to ` · `, to the end
of the step, to `in frame`, or to the word that follows an element (`to` in a
drag, the state in `expect`). `role=button "Pay"` narrows by role. Visible,
enabled elements come first; text inside a button means the button. An
element is scrolled into view only when it is not in view already, and then
below the site's fixed or sticky header (`header-offset:`, see Promo kit).

```markdown
- click .hm-eng.big .hm-open
- type "Ada" into form .name input
- expect #cart .total visible
- drag from .lane .card to #done .column
- wait for .toast .msg · within 10s
- type "3" into "Quantity" in frame .preview
- expect "Bought 3" visible in frame "Shop"
```

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
- `--webhook URL` posts one summary when the run ends (a script or a folder),
  so the team sees it in its channel; `--webhook-format json` (default),
  `slack`, `teams` or `discord`. json is `{"version": 1, "event":
  "demo.test", "title", "passed", "failed", "skipped", "seconds", "ok",
  "report", "failures": [{"script", "feature", "step"}], "si", "time"}`; the
  chat formats are one sentence ("si demo test docs/demos: 5 passed, 1 failed
  in 42.0 s. Failed: studio.md › Open a file › click "Open". Report: …").
  `report` is the report's path, or `$SI_REPORT_URL` when the CI job sets
  one. With `webhook-secret:` in your own si.conf (or `SI_WEBHOOK_SECRET`) the
  post carries `X-Si-Timestamp` and `X-Si-Signature: sha256=<HMAC-SHA256 of
  "<timestamp>.<body>">` (refuse a timestamp over 5 minutes off). A post
  that fails is a warning; it never changes the exit code.

```text
si demo test docs/demos/ --junit junit.xml --webhook "$SLACK_HOOK" --webhook-format slack
```
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
| `vertical` | `<script>.vertical.mp4`: 1080×1920 for phones and reels, the film across the middle over a blurred copy of itself (or a colour: `vertical-fill:`), the captions large under it (see Promo kit) |
| `highlight` | `<script>.highlight.mp4`: about 60 s, the first sentence of each feature with a breath around it, between the brand's cards (see Promo kit) |
| `words` | per-word captions: `captions` (turned on by it) shows the word being said marked in the accent colour, a picture per word, and the `.vtt` carries a time before each word (si's player marks the word too); word times share each caption's time by letters |
| `all` | all of the above but `words` (a look, not a file) |

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
| `shell: bash` | the shell of the terminals: `sh`, `bash` or `zsh` (default `$SHELL`, else `sh`); on Windows `pwsh`, `powershell` or `cmd` (default PowerShell, else `cmd`) |
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
| `terminal split right` · `terminal split down` | splits the terminal in front in two: a second shell beside it (right) or under it (down) in the same window, both filmed; the new pane takes the focus. Once per terminal (two panes) |
| `focus 1` · `focus 2` | the pane that `run`, `press` and `expect output` go to from then on (the other one dims a little on camera) |
| `run "command" in 2` · `expect output "text" in 2` | that pane for this step only, the focus stays (`wait for output "…" in 1` too); `in 2` before `·`: `run "make" in 1 · wait` |
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
- **On Windows** (10 1809 or later) a terminal is PowerShell (`pwsh`, else
  `powershell.exe`) under ConPTY, Windows' pseudo console: colours, full-screen
  programs and `run … · wait` as on a Mac. si's prompt is the folder's name
  and `$` (no profile, no PSReadLine), and each command's exit status reaches
  si through the window title, never shown. `shell: cmd` runs on pipes: no
  colours, the prompt is the starting folder's name and `$`, and si asks for
  the exit status after each command. `run … · wait`, `terminal split right`
  and `down`, `run … in 2`, `expect output … in 2` and the rest work the same
  (each pane its own shell; CI runs a split demo under ConPTY and cmd, and the
  `si play` drawer's split view, on Windows).

A server in one pane and its client in the other:

```markdown
- terminal "API" · window=mac
- run "./bin/api --port 8080"
- expect output "listening"
- terminal split right
- run "curl -s localhost:8080/orders/42" · wait
- expect output "paid"
- expect output "GET /orders/42" in 1
```

Examples: `examples/webdemo/` (a small shop, `shop.md`, and `broken.md`,
whose coupon does not work; `broker.md`, a tiny broker in Go started with
`go run`, filmed in a terminal and in its console; `morning.md`, seeded and
cleaned up around its feature).

### The deck on stage: embed-terminal and embed-browser

A course video often wants the slides on screen with `si play`'s own tools
over them: the terminal drawer running the lab's commands and a console in a
browser window over the slide, as the presenter would show them. With
`app:` a deck (a `.md` path next to the script), `si demo test` and `si demo
run` start `si play <deck> --addr 127.0.0.1:<free port>` themselves (with
the `start:` programs, first; `shell:` is the drawer's shell) and stop it at
the end; its drawer's shells start with a clean prompt (the folder's name and
`$`, none of your rc files), so no account, host or cloud name gets filmed; `app:` may also be the URL of a deck `si play` already serves.

When the deck has its own `setup:` and `start:` lines (§14), `si play` shows
its getting-ready page while they run; the demo waits until the talk itself
is served, then runs its own `start:`, `prepare:` and `step:` lines and the
features. So the deck sets up, the demo only adds to it: a demo's
`prepare:` never runs beside the deck's `setup:`, and needs no wait of its
own for what the deck starts. When a deck's `setup:` fails, the run fails
with its error (`getting ready failed: setup: …`). Give the stack to one of
them, not both: the deck's `setup:` when the talk needs it live, the demo's
`prepare:` when only the video does.

```markdown
---
title: The broker
setup: docker compose up -d --wait
---
```

```markdown
---
type: demo
app: talk.md
prepare: rabbitmqadmin declare queue name=orders   # after the deck's setup:
---
```

These steps drive the tools by name, through the page's public API
`window.siPlay` (§14), never through si's markup:

| Step | What it does |
|---|---|
| `embed-terminal "Lab 1"` | opens the drawer on the shell tab `Lab 1` (created when missing, raised when it is there); the `run`, `type` (with no field), `press`, `expect output` and `wait for output` steps after it go to that tab |
| `embed-terminal "Server" · split right` | the tab `Server` in a pane beside the tab in front (`split down`: below it); the steps after it go to `Server` |
| `embed-terminal "Lab 1" · opacity 80%` | the drawer at 80 % opacity (40 % to 100 %): the slide shows through, the text stays opaque; with `split` too |
| `embed-terminal min` · `max` · `restore` · `close` | the drawer: hidden (its shells run on), the whole window, back to its size, hidden and restored; after `min` or `close` the steps act on the page again |
| `embed-browser "RabbitMQ"` | opens the deck's `browser:` link `RabbitMQ` (or a URL, `embed-browser "http://localhost:15672/#/queues"`) in a window over the slides; an open one comes to the front |
| `embed-browser "RabbitMQ" · at 80,60 · size 1100x700 · zoom 125%` | places it (CSS pixels from the top left; zoom 30% to 300%), the same on every take |
| `embed-browser "RabbitMQ" close` · `min` | closes or minimises it |
| `click "Queues" in embed-browser "RabbitMQ"` | a step inside the window's page: any element step takes `in embed-browser "Name"` as it takes `in frame` |
| `slide 4` · `next` · `prev` | moves the deck (slide numbers from 1; `next` and `prev` step through the build steps, as → and ← do) |

```markdown
---
type: demo
title: The broker on stage
app: onstage/talk.md
shell: sh
start: si serve onstage/console --addr 127.0.0.1:7421 as console · wait for port 7421
---

## The queue fills

What it does: the drawer starts a producer under the slide, and the console shows the queue grow.

Say: Here is the slide about the queue. Let's see it in action.
- slide 2
- embed-terminal "Lab 1"
- run "echo sending 3 orders"
- expect output "sending 3 orders"

Say: And the console, right over the slide.
- embed-browser "Console" · at 700,60 · size 640x420 · zoom 110%
- click "Queues" in embed-browser "Console"
- expect "orders: 3 messages" visible in embed-browser "Console"
- embed-browser "Console" close
- embed-terminal close
- next
```

In a take, `run` and `type` type at a person's pace. A tab of the drawer is
not a `terminal` step's window: `run … · wait`, `· exit`, `· fails`, `expect
no output`, `in 2` and `note … at line` are for `terminal` steps, and `si
check` says so; follow a long command with `wait for output "…"`. The same
moments can live in a deck's own `Notes:` instead, as `[TERM Lab 1: make up]`
and `[BROWSER RabbitMQ]` cues (§11), filmed by `si build talk.md --mp4
--voice` with no demo script. Example: `examples/webdemo/onstage.md`.

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
step:    workloads run -f seed.yaml · within 5m             # step: is prepare:
start:   ./consumer · wait for "listening"                  # a second program, after the seed
setup:   mqctl queue purge orders                           # before EACH feature
stop:    make demo-down                                     # while the programs still run
cleanup: rm -rf data/ tmp/                                  # always last
---

## Before
- terminal "Ops"
- run "mqctl queue create audit" · wait
- expect output "created"

## Orders flow through
…

## After
- terminal "Ops"
- run "mqctl queue delete audit" · wait
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
| `si demo test "https://…/shop.md · sha256 …"` | one script from the web, its fingerprint checked, with the files its `files:` key lists beside it |
| `si demo test gist:abc123@4f2e9c1/shop.md` | one script of a gist at that revision, the gist's other files beside it (a gist is flat and pinned whole) |

The same trust rules hold (the repo's or the gist's pin, or the URL and the
files it lists, are asked about once, with the scripts they will run); the
output folder is local, `demo-out/<repo>/<script>/`.

A script from a URL names the files it needs beside it (`prepare:` scripts,
pictures, a theme) in `files:`, each with its own fingerprint; they are
fetched from the URL's folder, keep their folders, and a file whose
fingerprint no longer matches stops the run:

```markdown
---
type: demo
app: none
prepare: ./seed.sh
files: seed.sh · sha256 9f2a…, brand/logo.png · sha256 41c0…
---
```

A JSON script lists them as `"files"`, one string or a list of them:
`"files": ["seed.sh · sha256 9f2a…", "brand/logo.png · sha256 41c0…"]`.

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

### Promo kit

A promo of a public site is a web demo too, with a few things more: the
browser kept between features, sticky headers left clear, steps inside framed
pages, a first script read from the site, the brand around the film, and short
cuts made from the same take. Every one of them is off unless asked for.

```markdown
---
type: demo
title: Harbor
app: https://harbor.example
session: keep               # one browser profile for the whole run
header-offset: auto         # auto (the default), none, or the header's height: 72px
intro: Harbor · Every order out the door on time
outro: Start free · A hundred orders a month, on us
logo: brand/logo.svg        # on the intro and outro cards
url: https://harbor.example # the address the cards show
music: brand/bed.mp3 · 20%  # under the whole film, ducked while the voice speaks
theme: forest               # a built-in theme for every card (or brand/harbor.css)
vertical-fill: theme        # the vertical cut over the theme's background (blur, color #hex)
---

## Before

- go /
- click "Accept all"

## One queue for every shop

Lower-third: Ana Ruiz · Head of Operations, Harbor

Say: Harbor puts the orders from every shop into one queue.
- go /
- spotlight #queue-title · label "One queue"
- type "3" into "Quantity" in frame .preview
- click "Buy" in frame .preview
```

- **`session: keep`**: every feature (and `## Before`) runs in one browser
  profile, so a cookie banner accepted once, a sign-in or a language picked
  stays for the run: accept the banner in `## Before` (never filmed) and no
  chapter opens on it. The profile is si's own, never your Chrome's, and is
  removed when the run ends; each feature still opens in a new tab. `fresh`
  (the default) is a new profile per feature.
- **`header-offset:`** an element a step scrolls to (a click, `scroll to`,
  `spotlight`, `zoom to`, a note) lands below the site's fixed or sticky
  header, not under it: `auto` finds the bar fixed across the top of the page,
  `72px` gives its height, `none` turns it off. An element taller than the
  space below the header is put with its top just under it. An element in view
  already is not scrolled at all, so a spotlight on what shows never nudges the
  page.
- **`… in frame <css>`** (or `in frame "name"`) after any element: the step
  looks inside that iframe, same-origin or cross-origin; clicks, typing and
  the spotlight land where the element is in the page.
- **`intro:` / `outro:`** `Title · line under it` (or `on`: the script's
  title) are cards before the first chapter and after the last (3.5 s and
  4.5 s), with `logo:` above the title and `url:` under it, large on the
  outro. They belong to the first and last chapters, so chapters still start
  at 0:00. `theme:` (a built-in theme: `forest`, `slate`, …, or a theme
  `.css` file next to the script, `theme: brand/harbor.css`: the built-in it
  `extends`, under its own light tokens, §15) dresses them, the feature title
  cards and the lower thirds.
- **`music: file · 20%`** is looped under the whole film (and the vertical
  and highlight cuts) at that level (25% when not given), fading in and out,
  and pressed down while the voice speaks (a sidechain compressor keyed by the
  narration). `<script>.m4a` stays the narration alone.
- **`Lower-third: Name · Role`** under a feature heading puts a name caption
  at the bottom left for its first 4 s; the `lower-third` step puts one at that
  moment. With `--region` (or `region:`) a lower third is drawn at the
  region's size, after the crop, so it is never cut off.
- **`--with vertical`** writes `<script>.vertical.mp4` (1080×1920), **`--with
  highlight`** `<script>.highlight.mp4` (each feature's first spoken sentence,
  about 60 s in all, between the intro and outro cards).
- **`vertical-fill:`** what fills the vertical cut around the film: `blur` (a
  blurred, darkened copy of the film, the default), `theme` (the background
  of `theme:`), or `color #0b1020`.

**A first script from the site:** `si demo draft` reads a site in headless
Chrome and writes a promo script to start from.

```bash
si demo draft https://harbor.example --out harbor.md --pages 6
si demo test harbor.md
```

It stays on the site's own origin, obeys `robots.txt` (`Disallow`, `Allow`,
`Crawl-delay`, a group for `si` first), reads at most `--pages` pages (6), and
only loads them: nothing is clicked, typed or submitted. The script has
`session: keep`, a `## Before` that accepts a cookie banner when there is one,
a feature for the home page's hero and for each of its sections, one for each
other page, and a closing call to action (the hero's button, also the
`outro:`). Its `Say:` lines are the page's own words (a heading and its key
sentence), each under a `<!-- draft: polish this line -->` mark; its steps
use selectors that match one element each, so `si demo test` passes on the
site as it was read. Without `--out` it prints the script.

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
| `type: demo` with `app:` and `· kind=demo` | Demo | the same filmed video, for learners too, labelled Demo |
| `type: lab` | Lab | the lab; its task-list boxes can be ticked |
| `type: demo` without `app:` | Demo | the runbook (instructor build only) |
| `type: quiz` | Quiz | the questions, scored |
| `type: deck` with `· kind=video` | Video | the deck's narrated `<deck>.mp4`, filmed by `si build` when missing or older than the deck |

**Demo lessons for learners.** A filmed web demo (`type: demo` with `app:`,
such as one that shows the deck with `si play`'s drawer and a console over
it, §27) under `· kind=demo` is a learner lesson labelled **Demo** in the
menu and in the next-lesson line: the player with its captions, its done
tick following the video, as a Video lesson. So a course can say Video for
the narrated slides and Demo for "let's see it in action". A demo without
`app:` stays the instructor's live runbook.

```markdown
- Module 2: First messages
  - [Publish and consume](02/first-messages.md) · kind=video · pdf
  - [See it in action](02/in-action.md) · kind=demo
```

**Video lessons from a deck.** For self-paced learners a slides lesson can be
the deck's narrated video, with the slides to keep:

```markdown
- Module 3: Exchanges and routing
  - [Exchanges and routing](03/exchanges.md) · kind=video · pdf · slides
```

- `· kind=video` on a `type: deck` lesson shows `03/exchanges.mp4`. `si build`
  films it beside the deck (as `si build exchanges.md --mp4 --voice` would:
  narrated from its `Notes:` with `--voice`, else the deck's or si.conf's
  `voice:`) when it is missing or older than the deck; `--no-film` skips it.
  The deck's time is the newest of its file and the files its `<!-- include:
  -->` lines bring in (sections, code), so editing a section re-films it;
  pictures and canvases it uses do not count (touch the deck to re-film).
  The same build writes `exchanges.vtt` captions from the narration's words and
  timing, which the player shows (`<track kind="captions" default>`, the deck's
  `lang:`, else `en`).
- `· pdf` adds a **Download the slides (PDF)** button under the player
  (`exchanges.pdf` beside the deck, printed by the same build); `· handout`
  a **Download the handout (PDF)** one (`exchanges.handout.pdf`, the deck's
  `handout:` layout).
- `· slides` adds an **Open the interactive slides** link to
  `exchanges.slides.html`: the deck, built, in no menu.
- The done tick follows the video (watched to 90 %, or to the end).
- `· pdf`, `· handout` and `· slides` on any other lesson fail `si check`.
- **Poster.** A video lesson opens on a still, not on black: a deck's
  `exchanges.poster.jpg` is its first slide (1920×1080), written by the same
  build that films it; a `.mp4` lesson's `<video>.poster.jpg` is its frame at
  0.5 s (ffmpeg). `· poster=assets/cover.jpg` (beside `_course.md`) or
  `poster: cover.jpg` in the deck's front matter (beside the deck) picks
  another picture; `· poster=off` leaves the player without one. A missing
  poster file fails `si check`, and so does `· poster=` on a lesson that is
  not a video.

**Minutes.** A lesson's `minutes:` front matter (or `· minutes=` in
`_course.md`) sets its length; otherwise si estimates it: reading time at 230
words a minute, a deck's `(N min)` notes, a video's length, five minutes a
lab step, a minute a quiz question, a demo's `duration:`. Modules and the
course add them up; `duration:` in `_course.md` overrides the course total.

**Prerequisites** on the course (`prerequisites:`), a module (`·
prerequisites=…`) or a lesson (front matter) are shown, not enforced.

### Two audiences

- **Learner** (`si build course/`, `si serve course/`): readings, slides,
  videos, filmed demos, labs, quizzes. Instructor demos (no `app:`) are left out of the menu, or with
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
| `numbered` | lessons numbered within their module (`1.1`, `1.2`, `2.1`) in the menu and on the Previous / Next cards (modules are always numbered) |

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
marked right.

**A quiz is a self-check, not a secure exam.** A learner page keeps no right
answers in its source: each question carries only a SHA-256 hash of a
per-quiz salt and the ids of its right answers, and the page hashes the
learner's picks the same way to check them. Explanations are shown after
answering. A question has few answers, so anyone can try them all, or read
the explanations in the source; use an LMS's own exam for grades that matter.
The instructor build (`--instructor`, `si play`) keeps the plain answers.

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

A solution can hold anything a lab can, includes too. `si check` reports a
solution without its `<!-- end -->` at the `<!-- solution -->` line; an
include inside it that fails is reported on its own (the solution still
finds its end).

### Progress

Progress is kept in the learner's browser (localStorage, per course; nothing
is sent anywhere unless the course has a `webhook:`, below). A lesson is done when it is read to the end, its video
watched to 90 %, its lab's boxes all ticked, or its quiz passed. The menu
ticks done lessons and fills each module's ring; the top bar shows the
course's percentage; the home page's **Start** becomes **Continue: <the
next lesson>**. `complete.html` lists what is left and, when everything is
done, says so; with `certificate: on` the learner types a name and prints a
certificate.

### Results to your team: `webhook:`

A course with `webhook:` in `_course.md` sends small result events from the
learner's browser to an address you choose: a Slack, Teams or Discord
channel, a learning record store, or your own endpoint (a Zap, a sheet, a
small server). It works in the website bundle, from `file://` and inside a
SCORM package (as well as the LMS calls).

```markdown
---
title: A first web page
webhook: https://hooks.slack.com/services/T000/B000/XXXX
webhook-format: slack
webhook-events: quiz.submitted, course.completed
webhook-learner: ask
webhook-consent: required
webhook-notice: Your quiz scores go to your trainer, Ana.
webhook-base: https://learn.example.com/web101
---
```

| Key | What |
|---|---|
| `webhook:` | `https://…` (si check warns on anything else), or `relay` (below). si.conf `course.webhook:` is the default for every course |
| `webhook-format:` | `json` (default), `slack`, `teams`, `discord`, `xapi` |
| `webhook-events:` | which events (default all five) |
| `webhook-token:` | json: a `"token"` in every event, to tell classes apart. It is in the page: not a secret |
| `webhook-auth:` | xapi: the `Authorization` header (`Basic …`) of the learning record store. In the page too: use a write-only key, or the relay |
| `webhook-learner:` | `random` (default): a random id per browser; `ask`: the notice has a name field |
| `webhook-consent:` | `notice` (default): events are sent, the notice stays until OK; `required`: nothing is sent before OK (events wait), **No thanks** sends nothing, ever |
| `webhook-notice:` | the notice's text (default: "This course sends your progress (lessons done, quiz scores) to its organizers.") |
| `webhook-base:` | `https://…` where the course is published: `course.url` and `lesson.url` (the xAPI activity ids) are under it wherever the page is opened (`…/web101/` and `…/web101/02-html/quiz.html`). Without it: the page's own address when it is `http(s)`, else `urn:si:course:<id>` and `urn:si:course:<id>:<page>` (never a `file://` path). si.conf `course.base:` is the default for every course |

The events:

| Event | When | `data` |
|---|---|---|
| `lesson.completed` | a lesson is done (as for the ticks above) | `{}` |
| `quiz.submitted` | Check my answers | `score` (0-100), `pass`, `passMark`, `attempt`, `questions: [{"id": "q1", "correct": true}, …]` |
| `lab.step.checked` | a lab's **✓ Check** ran (below) | `step`, `title`, `pass`, `exit`, `ms` |
| `demo.watched` | a lesson's video ends, or the learner leaves after watching some | `percent`, `seconds` |
| `course.completed` | the last lesson that counts is done | `lessons` |

`json` events (`version: 1`; a field may be added, none changes without
version 2):

```json
{
  "version": 1, "event": "quiz.submitted", "id": "5f0c9a6b-2d41-4e8f-9a3c-0d1e2f3a4b5c",
  "time": "2026-11-14T10:02:11.418Z", "si": "3.18.0", "audience": "learner",
  "course": { "id": "web101", "title": "A first web page", "url": "https://learn.example.com/web101/index.html" },
  "lesson": { "id": "02-html/quiz.html", "title": "Check yourself", "kind": "quiz", "url": "https://learn.example.com/web101/02-html/quiz.html" },
  "learner": { "id": "l-3f9a0c71b2e4", "name": "Ana" },
  "data": { "score": 80, "pass": true, "passMark": 70, "attempt": 2,
            "questions": [ { "id": "q1", "correct": true }, { "id": "q2", "correct": false } ] },
  "token": "c42-7f3a"
}
```

`id` is a UUID per event (drop duplicates by it: an event queued offline may
arrive twice); `course.id` is the course folder's name, `lesson.id` the
page's path in the course (`course.completed` has no `lesson`);
`audience` is `instructor` in the instructor build, so you can ignore the
trainer's own clicks. `slack`, `teams` (an Adaptive Card) and `discord` get
one sentence: "Ana passed the quiz “Check yourself” in A first web page:
80 % (attempt 2)." `xapi` sends an xAPI 1.0.3 statement: the learner as an
account (`homePage` the course URL, `name` the learner id), ADL verbs
(`completed`, `passed` / `failed`, `answered` for a lab check,
`experienced` for a video), the lesson's URL as the activity, the course as
its parent, `result.score.scaled` and `success`; point `webhook:` at the
LRS's `…/statements` endpoint.

How it is sent: `json` and `slack` with `navigator.sendBeacon` (a
`text/plain` JSON body: no CORS preflight, and it leaves even as the page
closes); `teams`, `discord` and `xapi` with `fetch` (keepalive), which needs
the receiver to allow CORS. What cannot leave (offline, refused, a 5xx)
waits in localStorage (the newest 100) and the page tries again: after 2 s,
then 4 s, 8 s … up to a minute apart, and at once when the browser is back
online; what is still waiting when the learner leaves goes with the next page.

**Privacy.** Events never carry the answers a learner picked, a lab's output,
or a name the learner did not type; the learner id is random and kept in
that browser. A notice at the bottom of the course says results are sent
before anything else happens.

**The relay.** `webhook: relay` keeps the address (and `webhook-auth:`) off
the page: `si serve` (or `si play`) answers `POST /_si/relay`, forwards each
event to the course's address (`course.webhook:` in si.conf) and signs it:
`X-Si-Timestamp: <Unix seconds>` and `X-Si-Signature: sha256=<hex
HMAC-SHA256 of "<timestamp>.<body>">` with `webhook-secret:` from your own
si.conf or `SI_WEBHOOK_SECRET`. A receiver checks both: recompute the HMAC
over the timestamp header, a dot and the raw body, compare in constant time,
and refuse a timestamp more than **5 minutes** from its own clock, so a
captured post cannot be replayed later (drop a repeated event `id` inside the
window too). A page served by si sends through the relay whenever it is on:
always on a loopback address, and on any address with `--relay` (`si serve
course/ --addr 0.0.0.0:8080 --relay`). With `si play --remote` the network
listener relays too, behind the run's key like the other remote pages. The
relay only ever forwards to a course's own configured address. A static
build of a `webhook: relay` course sends nothing.

```text
X-Si-Timestamp: 1763114531
X-Si-Signature: sha256=hex(HMAC-SHA256(secret, "1763114531." + body))
```

### Labs that check themselves: `<!-- check: -->`

In a lab, `<!-- check: <command> -->` under a step says how to tell the step
works: `· expect "text"` (the output must contain it) and `· exit N` (the
exit code that passes, 0 by default).

```markdown
## Step 1: Create the file

Make a folder called `my-page` and, inside it, a file called `index.html`.

<!-- check: ls my-page/index.html · expect "index.html" -->
```

Every build shows it as **Run this to check:** with the command and what
passes. In `si play` on a loopback address (where the terminal drawer
works) it gets a **✓ Check** button: si runs the command in a process of its
own (`sh -c`, `cmd /C` on Windows; never the drawer's shell), with the
page's folder as the working folder and a 30 s limit, and shows Passed or Not
yet with the exit code and the output. si runs only commands written in a
`<!-- check: -->` of that page. In a course the result is a
`lab.step.checked` event (through the relay, signed). `si check` reports a
check with no command or an option it does not know.

### The live class view: `si serve --class`

For a class in the room (or on a call), serve the course with `--class` and
watch it from a laptop or a tablet:

```text
si serve web-course/ --class --addr 0.0.0.0:8000
  class view (this machine): http://127.0.0.1:8000/_si/class
  class view on another device (keyed link, keep it to yourself):
  http://192.168.1.20:8000/_si/class?k=…   (and its QR code)
```

The course needs no `webhook:` for it: a page served this way sends every
result event, as JSON, to the server itself (`POST /_si/class/event`), beside
whatever goes to the course's webhook (which keeps its own
`webhook-events:` and format). Learners are asked their name
(`webhook-learner: ask` is the default with `--class`; write
`webhook-learner: random` to keep them anonymous ids), and the notice and
`webhook-consent:` work as above: a learner who says No thanks never shows up.

```markdown
---
title: A first web page
webhook-learner: ask
webhook-notice: Your progress shows on the trainer's screen during the class.
---
```

The dashboard (`/_si/class`) opens on this machine's loopback, or anywhere
with the run's key (the printed link; a new key each run). It shows, live
(server-sent events, no reload):

- **Learners by lesson:** one row per learner (the typed name, else
  "Learner 3f9a"), one column per lesson under its module: ✓ done, the quiz
  score (green passed, red not yet; ×2 for attempts), a dot per lab check
  (red failing) with passed/total, ▶ the percent of a video watched.
- **Who's stuck:** a learner who failed the same check twice in a row, or
  made no progress (a lesson done, a quiz or check passed, a video watched)
  for N minutes on the lesson of their last event (N: 5, 10, 15 or 30, 10
  by default).
- **Lessons:** done, quiz average and passed, learners failing a check,
  average watched, per lesson. **Quiz questions:** for each quiz, how many
  got each question wrong (each learner's last attempt), each question
  named by its title from the quiz's source.
- **Module** filter, **Export CSV** (one row per learner and lesson), light
  and dark, the course's theme. More than one course in the folder: a Course
  picker.
- **Day:** today, live, or any earlier day with a saved
  `.si/class/<date>.jsonl`, read back as it stood at that day's last event
  (CSV too).

Every event is appended to `.si/class/<date>.jsonl` beside the served folder
as it arrives; a restarted server starts again from today's file. Kept: the
event, the lesson, the score, the step and the random learner id, and the
name the learner typed: no address, no browser, no answers, no lab output.
Delete the files when the class is over. `si play course/ --class` does the
same for the instructor's build (its events say `instructor`).

**In the Studio** (`si --web`), the **Class** mode does it from the editor:
pick the course, tick **Learners on my network** (or not: this machine
only), **Start the class**. The course is served on a port of its own with
the class on (as `si serve course/ --class`; only the course's folder,
read-only), the dashboard shows inside the Studio, and beside it the link
and QR code learners open and, on your network, the keyed link for a
tablet. Leaving the mode keeps the class running; **Stop the class** (or
quitting the Studio) ends it. The events go to the same
`.si/class/<date>.jsonl`.

**In the terminal**, `si play course/ --terminal --class` (or `si serve
course/ --terminal --class --addr 0.0.0.0:8000`) serves the course with the
class on and shows the class as text, refreshed every two seconds: the
links to share, the counts, learners by lessons (✓ done, a quiz's best
score in green or red, ✗ a failing lab check, · started), the lessons
numbered under the grid, and who is stuck. `r` refreshes, `q` quits and
stops the server.

```text
Class · A first web page
Learners: http://192.168.1.20:8000/

2 learners · 0 completed · 1 stuck · 9 events

Learner       1   2   3   4   5   6   7  done
Ana           ✓   ✓   ✓   ✓       ✗  85   5/11
Bruno         ✓   ·                       1/11
```

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
  of lesson; `si --export example --type course` the same example. The
  Studio's **New…** and start gallery offer it as **Course**.

### SCORM packages: `--scorm`

`si build course/ -o html/ --scorm` also writes **`<course>-scorm12.zip`**
beside the output: the learner build plus an `imsmanifest.xml` (SCORM 1.2,
which SkillJar, Moodle, Canvas, Docebo and Cornerstone all import).

| Flag | SCOs | What the LMS shows |
|---|---|---|
| `--scorm` | one per lesson (instructor demos left out) | its own menu of modules and lessons; the course menu in the pages is hidden |
| `--scorm=single` | one for the whole course (`scorm.html`) | the course as on the web, menu and all |

The pages find the LMS's SCORM API in a parent or opener window and report
to it: `LMSInitialize`, `cmi.core.lesson_status` (`incomplete`, then
`completed`; a quiz `passed` or `failed`), `cmi.core.score.raw` from the
quizzes (0 to 100; one SCO for the course reports their mean),
`cmi.suspend_data` with the progress (the done lessons, quiz scores, the last
lesson; so a learner continues on another device; SCORM 1.2 keeps 4096
characters of it, so a course whose progress is longer sends it packed, its
lessons by their place in the menu with a hash of the menu: a republished
course with other lessons starts that copy afresh, and progress that does
not fit even packed stays in the browser), `LMSCommit` and
`LMSFinish`. A quiz lesson with `pass:` gets that mark as its
`adlcp:masteryscore`. The package has no network dependencies and still
works with no LMS (progress in the browser, as on the web). `--scorm` packs
the learner build; it refuses `--instructor`.

```text
si build web-course -o html --scorm          # html/ and web-course-scorm12.zip
si build web-course -o html --scorm=single   # one SCO: scorm.html
```

### Printed handouts: `--pdf`

`si build course/ -o html/ --pdf` (needs Chrome) also writes the **learner
workbook**, `html/<course>-workbook.pdf` (and `workbook.html`): a cover with
the module plan and timings, then every lesson in order: the readings, the
lab steps without their solutions, the quiz questions without answers or
explanations, a deck's slide titles, and a lined **Notes** box after each
lesson. With `--instructor` it writes the **instructor guide**,
`<course>-instructor-guide.pdf`: the same order with each lesson's minutes
and instructor note, every demo's runbook (commands and expected output),
the decks' spoken script, the quiz answers ticked with their explanations,
and the lab solutions.

### Publish to SkillJar: `si publish --to skilljar`

```text
si publish login skilljar --token <API key>
si publish web-course --to skilljar --base-url https://files.example.com/web-course/
si publish web-course --to skilljar --base-url https://files.example.com/web-course/ --yes
```

The learner build becomes a Skilljar course: each module a section, each
reading an HTML lesson, each video an asset lesson, and each slides, lab or
quiz lesson (and a reading with a quiz in it) a SCORM 1.2 web-package lesson
of its own. Instructor demos are never published. Skilljar's API takes no
uploads: it fetches videos and packages from a URL, so si writes them into
`web-course/.si/publish/skilljar/` and you put that folder online at
`--base-url` before `--yes`. Without `--yes` it is a dry run that prints
every request (the key masked). The ids land in
`web-course/.si/publish/skilljar.json`, so the next publish updates the same
course in place: lessons patched and reordered, only changed files fetched
again, removed lessons deleted. Publish the course to your domain in the
Skilljar dashboard. See `docs/publish.md`.

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
| `layout` | `snake` (the line wraps across rows: 30 milestones fit one 16:9 page), `line` (one horizontal line, two levels a side), `vertical` (one line down the page, cards left and right), `eras` (one column per era: a coloured header with the era's name and dates, its milestones stacked under it, pictures as small medallions; 30 milestones fit one 16:9 page), `cards` (a grid of picture cards on a thin dated rail, each in its era's colour) | `snake`; `vertical` when portrait |
| `orientation` | `landscape` (a slide: a frame of `aspect:` that scales like a picture), `portrait` (a tall scrolling poster, a document page) | `landscape` |
| `aspect` | the landscape frame: `16:9`, `16:10`, `4:3`, `1:1` | `16:9` |
| `alternate` | `on` (milestones above and below the line, or left and right), `off` (one side) | `on` |
| `scale` | `even` (equal gaps), `time` (gaps follow the years, held between 0.6 and 1.8 of the usual gap; a shortened long gap gets a ≈ mark) | `even` |
| `closing` | a line at the end of the timeline | none |
| `steps` | `eras` (one build step per era on the slide), `off` | `eras` |
| `require` | a rubric for `si check`: `20 dates, 15 images, 7 eras, 4 sources` | none |
| `lang`, `theme` | as on any page; every built-in theme works, light and dark | |

On a phone (narrow or portrait screen) `snake` and `line` become `vertical`,
with the line on the left; `eras` stacks its era columns and `cards` shows one
or two cards a row. A portrait `eras` poster puts its eras side by side as far
as the width allows; a portrait `cards` poster is a grid of cards. On a wide
screen every layout but `vertical` is a frame of `aspect:` whose text shrinks
until the fullest card (or era column) fits.

```markdown
---
type: timeline
title: Hitos en Algoritmos y Complejidad Computacional
layout: eras
---
```

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
  (default 2: a 16:9 timeline is 3200 × 1800). For a landscape
  timeline `--pdf` writes two files: `pdfs/timeline.pdf`, several 16:9 pages
  in the same look with whole eras on each (about 12 milestones a page, so the
  text reads well; the title on every page, the closing line and sources on
  the last), and `pdfs/timeline-compact.pdf`, the whole timeline on one page
  like the PNG. Both are printed as vector PDF in the same look as the PNG:
  the text is selectable and searchable, and pictures keep their own
  resolution. A portrait poster's `--pdf` is the printed page. `--svg`
  writes `timeline.svg`: the same frame as a vector picture, its text as
  real text, its pictures inside it and the theme's colours resolved, its
  background glows as SVG gradients (a conic gradient as thin wedges of its
  colours), its shadows as blurred SVG filters (inset shadows and text
  shadows too), and the web fonts its text uses embedded, cut to the
  characters the picture shows (a font from Google Fonts in `fonts:` is
  fetched again as TrueType for that; a WOFF2 font from elsewhere, or one
  for text that needs shaping such as Arabic, goes in whole, and the build
  says so; an animated infographic is taken finished). All need Chrome.
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
aspect: 16:9             # 16:9 | 4:3 | 9:16 | a4; exact frames: phone | story | portrait | square | landscape | 1080x1350
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
| `chart=line` | a table: x labels, one or more columns of numbers | a line (area under the first) from zero, the last value at its end; when every x label is a number (a year) or a date (`2024-03-01`, `2024-03`, `Mar 2024`), points sit at their real value on a linear scale (numbers get ticks at round values, dates their own labels), else evenly spaced |
| `chart=progress` | a table: labels, a value (of 100, or of a third column: the target) | progress bars |
| `chart=column`, `chart=area`, `chart=pie` | a table, as above | vertical bars; a line with every series filled; a full pie (the charts of the ```` ```chart ```` fence, §33, which also animates them step by step) |
| `chart=bar3d`, `chart=donut3d`, `chart=funnel3d` | a table, as above | solid 3D bars on a floor, a donut tipped back, a funnel of stacked layers, in CSS 3D at rest (§33); `--png` and `--pdf` draw them, `--svg` leaves them out (CSS 3D is raster only) |
| `quote` | a `>` quote; a line starting `—` is the attribution | a large quote |
| `icons` | a list; each item may start with `![](icon.svg)` | an icon row |
| `list`, `text` | a list, any Markdown | as written |
| `image` | `![caption](picture.jpg)` | the picture filling the block |
| `flow` | paths: `A -> B -> C`, `A -[label]-> B` | boxes and numbered arrows (see Flow blocks) |
| `sources` | a list | the footer, small (with `sources:` and `author · date`) |

Without a kind, si guesses: a heading that starts with a number is a `stat`,
a table of numbers `chart=bar` (any other table `compare`), a numbered list
`steps`, a list with pictures `icons`, a `>` quote `quote`, a lone picture
`image`, a first line with `->` `flow`, a heading named Sources (Fuentes, References) `sources`, anything
else `text`.

Options for the grid: `span=2` (or `wide`) covers two columns, `tall` two
rows, `accent` fills the block with the accent colour; for motion,
`enter=` and `at=` (see Motion). A `·` part whose first
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
`orientation=`, `aspect=`, `steps=off`, `sources=`, `animate=build|loop`,
`duration=`. Animated on a slide, it plays when the slide becomes the current
one (each block as its click shows it, with steps) and `si build deck.md
--mp4` films it building. In a document the fence
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

### Motion

`animate: build` makes the infographic build itself once: the header fades
in, then each block comes in, one after another; `animate: loop` plays it
over and over (fading out at the end); `animate: off`, the default, shows it
still. `duration: 12s` sets the length (default: as long as the blocks need,
plus a moment to read). Per block, `· enter=fade|rise|grow|count` changes how
it comes in and `· at=3s` when.

````markdown
---
type: infographic
title: How a message travels through a broker
aspect: phone
animate: build
---

## The path · flow · dir=tb
[server] Producer -[publish]-> [api] Exchange -[route]-> [queue] Queue -[deliver]-> [function] Consumer
Consumer -[ack]-> Queue

## 50,000 · stat · accent
messages a second through one queue

## 2 ms · stat · enter=rise · at=4s
from publish to deliver
````

Without `enter=`, each kind has its own way in: numbers count up (`stat`,
`compare`), bars and progress grow, a donut sweeps, a line chart draws, steps
light one by one, a flow draws itself, a quote or a picture fades, the rest
rise. A page plays on load (the ▶ Play / ↺ Replay button in its corner, always
shown in the Studio preview, plays it again); a slide plays when it becomes
the current one, and again on every visit; with build steps, a block plays when
its click shows it. Under `prefers-reduced-motion`, in thumbnails and in print,
the infographic is shown finished. `si check` warns when the blocks need longer
than `duration:`.

### Flow blocks

`## Heading · flow` (or a block whose first line has an arrow) draws boxes and
arrows from paths, one per line:

| Write | Means |
|---|---|
| `Client -> Load balancer -> API` | a path: three boxes, two arrows |
| `API -[SQL]-> Database` | an arrow with a label |
| `Consumer -[ack]-> Queue` | an arrow back: the layout keeps the flow going one way, the two arrows side by side |
| `Cache` (alone on a line) | a box without arrows |
| `[db] Orders` | a box with an icon before its name (once is enough): `user`, `server`, `database` (`db`), `queue`, `cache`, `cloud`, `lock`, `globe` (`web`), `mobile` (`phone`), `function` (`fn`, `lambda`), `browser`, `mail`, `file`, `api` (`code`) |
| `## The path · flow · dir=tb` | the direction: `lr` left to right, `tb` top to bottom (without it, by the frame) |

Boxes meet by name across lines, so several lines make branches and merges.
si lays them out (columns by the longest path, siblings stacked): left to
right, or top to bottom in a tall frame (`phone`, `story`). The arrows are
numbered in the order written; animated, the boxes appear and the arrows draw
in that order, then a packet travels along each arrow in turn. The label of a
slanted arrow (a fan-out or a merge) is turned to run along it, upright,
beside its number; a level or upright arrow keeps a level label. A flow block is
`wide` unless it says `span=`. `si check` reports an arrow without a box on
one side, a label without its `]->` and an unknown icon, at its line, and a
`dir=` other than `lr` or `tb`.

### Sizes

`aspect:` takes the ratios above and, for an exact frame made to share or
export, a preset or a size:

| `aspect:` | Frame | Layout |
|---|---|---|
| `phone` | 9:19.5 (a phone's screen) | stats in pairs, every other block across; flows top to bottom |
| `story` | 9:16 | as `phone` |
| `portrait` | 4:5 | two columns |
| `square` | 1:1 | a slide's grid |
| `landscape` | 16:9 | a slide's grid |
| `1080x1350` | that exact ratio (and size for `--mp4`) | by its shape |

An exact frame keeps its shape at every size: its blocks share its height
(each row gets the room its blocks want) and its text is sized from its width,
so it fills a phone, scales up on a monitor (fit to the window's height) and
never scrolls. A tall one (`phone`, `story`, `portrait`, a tall size) is a
page; a wide or square one a slide.

### Build, check, new

- **`si build x.md --gif --mp4`** films an animated infographic: `x.gif`
  (720 px wide, 15 fps, one adaptive palette, only the changed pixels of each
  frame stored; loops forever for `animate: loop`, plays once and keeps its
  last frame for `build`) and `x.mp4` (full size: 1080 wide for a tall or
  square frame, 1920 for a landscape one, or the exact size; H.264, needs
  ffmpeg). Every frame is the infographic at that exact moment (the page's
  clock is set, not raced), so two builds give the same frames. `--fps 24`
  for the GIF, `--scale 1.5` for a width of its own (1.5 × the CSS width:
  540 tall, 900 square, 1280 landscape), `--size phone,square,landscape` for
  several sizes in one run: `x-phone.gif`, `x-square.mp4`, `x-square.png`, and
  so on. `si check` warns when the GIF would pass about 8 MB.
- **`si build talk.md --png --pdf`** writes `talk.png` (the slide at 1600 ×
  900, or the whole poster at 1080 wide; `--scale 2`, the default, doubles
  the pixels; `--scale 3` for print) and `pdfs/talk.pdf` (one page, vector
  text). **`--svg`** writes `talk.svg`, the frame as a vector picture (text
  as text, pictures inside, theme colours resolved; gradients (conic too)
  and shadows (inset and text shadows too) as SVG gradients and filters,
  used web fonts embedded and cut to the characters shown, the animation
  finished; see §29). `si play` presents a landscape one. Printed from the
  browser, a portrait poster that runs over several pages keeps its glows
  and padding on every page, and no block is cut between two pages.
- **Text fits its block.** A block whose text does not fit its cell gets a
  smaller text size, step by step, down to 60% of its own; `si check` warns
  when a block would need less than that (shorten it, make it `wide` or
  `tall`, or move a block out).
- **`si check`** reports an unknown kind or option (`enter=`, `at=`, `animate:`, `duration:`), a flow line it cannot read, a chart cell that is not a
  number (or a negative one in a donut), a compare or chart without a table,
  an empty block, a stat without its number, a block with more text than its
  cell holds at 60% text size, and `[CLICK]` cues that do not match the
  blocks.
- **`si new poster --type infographic`** writes a starter; `si --export
  example --type infographic` the two examples (a landscape story and a
  portrait poster).

## 31. Project structure

A course, a talk series, a training: the slides, the labs, the website, the
videos and the zips you hand over each want a folder of their own. A
`structure:` block in the project's `si.conf` is that map: one folder per
line, nested by indentation, its name ending in `/`; after it, what goes
there.

```text
structure:
  rabbitmq/
    md/           markdown
    slides/       deck
      pdf/        deck:pdf
    labs/         lab
    html/         site
      images/     image
      animations/ animation
      pdfs/       deck:pdf
    videos/       video
    dist/         html/:zip, deck:zip, course:scorm
```

`si build rabbitmq` then writes the decks into `slides/`, their PDFs into
`slides/pdf/` **and** `html/pdfs/`, the labs into `labs/`, the website into
`html/` with its pictures in `html/images/`, the videos into `videos/`, and
last the zips into `dist/`: `rabbitmq-html.zip` (the whole `html/` folder)
and `rabbitmq-deck.zip` (every deck with what it links to, the index and
labs included). `index.html`
stays at the root, its tiles pointing into the folders. The example project
is `examples/structure/`.

**No block, no change.** Without `structure:` a build writes what it always
did, every page side by side in the output folder.

**The top folder** names the project (the `<project>` of the zip names) and
is the folder that holds `si.conf`; a map without one top folder uses the
folder's own name. Paths of `html/:zip` are relative to it.

**The words after a folder** (a comma list):

| Kind | What goes there |
|---|---|
| `markdown` | sources: where `si new` puts them (sources may live in any folder of the map) |
| `deck`, `lab`, `guide`, `course`, `site`, `article`, `timeline`, `infographic`, `mindmap`, `book`, `demo` | the pages of that type (`type:` in front matter; a folder with `_nav.md`, `_course.md` or `_site.md` for guide, course, site, book). One guide, course, site or book fills its folder; several get a subfolder each |
| `image`, `animation`, `video` | the pictures (`.png`, `.jpg`, `.webp`), animations (`.svg`, `.gif`, and every SVG canvas also as a file) and videos and sound the pages use. With such a folder, pictures are linked from it, not inlined (as `--no-inline-media`); canvases stay inline in the slides, they must for their states |
| `kind:format` | a derived file: `deck:pdf`, `deck:mp4`, `deck:handout`, `course:scorm`, `course:pdf` (the workbook), `book:pdf`, `timeline:png` / `pdf` / `svg`, `infographic:png` / `pdf` / `svg`, `mindmap:png` / `pdf` / `svg` / `mp4`. Naming one is the same as its flag (`--pdf`, `--mp4`, `--handout`, `--scorm`, `--png`, `--svg`): the map builds it |
| `kind:zip` | a bundle: every output of that kind (pages and their derived files) with the pictures and videos they link to, and the built pages they link to (the index, a lab), then the pages and files those link to, and so on (each once), paths as in the project, so every local link inside the bundle works: `<project>-<kind>.zip` |
| `folder/:zip` | a bundle: a folder of the map, zipped whole after it is built: `<project>-<folder>.zip` |

A kind listed in two folders is written to both (copies). A derived file the
map does not place is not kept, unless you asked for it with its flag (then
it goes where it always did); a page of a kind with no folder goes to the
project root.

**The build, in order:**

1. read the sources, wherever they sit in the project (by `type:`);
2. render each page into its kind's folder;
3. copy the pictures, animations and videos into theirs;
4. make the derived files (headless Chrome: PDF, MP4, PNG, SVG);
5. copy what is listed twice;
6. zip the bundles, last, from the finished folders.

**Links follow the files.** Every page's links are rewritten from its new
folder: pictures, videos, PDFs, the other decks and labs, the index tiles
and their PDF badges, find (`/`), the navigation of guides, courses and
sites, and the shared `_si/` of `--share-assets` (a copy in each top folder
that uses it). A page links the nearest copy of a file listed twice, so
`html/` zipped alone still finds its PDFs in `html/pdfs/`.

**The output folder.** The map is laid out in the project folder itself;
`-o out/` lays it out under `out/` instead.

**From inside the project.** `si build md/intro.md` or `si build md/site`
(a page, or a guide, course or site folder, in one of the map's folders or
at the project root) finds the project: the nearest `si.conf` above it with
a `structure:` block, looking no higher than the top of the git repository
or your home folder. It then builds the whole project with its map, as `si
build rabbitmq` does, so the page lands in its kind's folder and its links,
the index and the bundles stay whole. A page outside the map's folders (a
`notes/` the map does not name) builds as it always did.

**`si new rabbitmq --structure`** writes `rabbitmq/si.conf` with the map
above, makes its folders and puts the starter deck in `md/`. Plain `si new
rabbitmq` does the same when `rabbitmq/si.conf` already has a
`structure:` block (its folders, its markdown folder).

**`si check`** checks every folder of sources and warns about the map: an
unknown kind or format, an indentation that does not line up, a folder name
without its `/` (each with its line in `si.conf`), a bundle naming a folder
the map does not define, a folder bundle that would be empty, a source of a
kind with no folder (it builds into the project root), and a source sitting
in a folder meant for another kind (a lab in `slides/`). A kind bundle for a
kind the project has no source of yet (`course:scorm` before the
`_course.md`) is a note, not a warning: `note: … nothing to bundle yet`, so
the map can be written ahead of the content and a new `--structure` project
checks clean.

**`si play` and `si serve`** serve the sources as always, live, from where
they sit: the map is about what `si build` writes.

## 32. Video edits

A recording almost always needs a few touches: a callout that says where to
look, a password blurred, a pause on the result, a zoom on a small field,
the first seconds cut, a broken stretch replaced or filmed again. A Markdown
file with `type: video` lists them beside the video; `si check` checks it,
`si play` previews it, `si build` writes the finished video (ffmpeg and
headless Chrome, as for `si demo run`).

```markdown
---
type: video
source: demo.mp4
out: demo.final.mp4
theme: studio
---
# Install a single node

## Timeline

- 0:00 · title "Install a single node" · 3s · fade
- 0:45–0:50 · blur 120,80,400x60
- 1:34 · bubble "Log in as the admin user" · to 820,410 · 4s · pop
- 1:52 · freeze 2s
- 2:10–2:30 · speed 2x · badge "⏩ 2×"
- 2:40 · zoom to 600,300 900x500 · 3s
- 4:00–4:04 · cut
- end · card "Next: clustering" · 3s
```

**Front matter.** `source:` the video (or a list, below); `out:` where the
result goes, beside the file (default `<source>.edited.mp4`; `-o dir` puts
it in `dir`); `theme:` and `fonts:` for the callouts and cards, as elsewhere;
`voice:` for `say`; `captions: on`; `script:` the demo script `refilm` runs;
`join:` the transition between sources.

**Lines.** Every line of the `## Timeline` list is `TIME · action · option ·
…`, text in double quotes; anything outside the list (prose, other lists
under other headings) is ignored, so the file can explain itself.

**Times are source times**: what any player shows for the source. `1:34`,
`1:34.5`, `1:02:03`, `94s`; ranges `1:36–1:46`, `1:36-1:46`, `1:36 to 1:46`;
`start`, `end`. An edit never moves another line's time: si works out where
each moment lands. The rule: a source time maps to the moment its frame
first shows in the output. After a cut everything is earlier; in a speed
range the time runs faster; a freeze at T shows frame T for its length (a
callout at T shows during it); an insert at T plays before frame T; a time
in a cut lands at the join (`si check` warns). Durations (`3s`, `freeze 2s`)
are seconds of the output.

**Anchors** instead of a clock: `at chapter "Install" + 5s` (the chapters in
the mp4, or a `chapters.txt` beside it: `0:42 Install`) and `at said
"management UI" - 1s` (when the words are said, from the `.vtt` or `.srt`
beside the source; `si demo run --with subtitles` writes one).

```markdown
- at chapter "Users" + 2s · keys "⌘ K"
- at said "one message ready" · bubble "Here it is" · to 1210,480
```

**Positions** are source pixels (`820,410`), percentages (`60%,40%`) or a
word: `top-left`, `top`, `top-right`, `left`, `center`, `right`,
`bottom-left`, `bottom`, `bottom-right`. A rectangle is `X,Y WxH` (or
`X,Y,WxH`; `W` and `H` take `%` too). `si check` reports any outside the
frame.

### Edits

| Line | What |
|---|---|
| `A–B · cut` | the range out |
| `T · trim start` · `T · trim end` | everything before / after T out |
| `T · freeze 2s` | hold frame T |
| `A–B · speed 2x` | faster (`0.5x` slower); the sound keeps its pitch; `· mute` silences it; `· badge "⏩ 2×"` shows a badge for the range |
| `A–B · replace with fix.mp4` | the clip, whole, with its own sound, instead of the range; `· keep audio` keeps the original sound and fits the clip to the range |
| `T · insert clip intro.mp4` | a clip before frame T |
| `T · insert image shot.png · 3s` | a picture |
| `T · insert card "Part two" · 3s` (or `card "…"`) | a title card in the theme |
| `A–B · mute` | the sound off |

A replace, insert or card takes a transition, `· fade [0.4s]`, `· dissolve`,
`· wipe`, `· slide`, at both of its joins; the outgoing picture is held while
the next fades in, so transitions never move times.

```markdown
- 0:03 · trim start
- 1:36–1:46 · replace with fixes/curl-ok.mp4 · fade 0.4s
- 2:00 · insert image diagrams/cluster.png · 4s · dissolve
- 2:30–2:34 · mute
- 5:58 · trim end
```

Several sources join in order, each trimmed with `· from` / `· to`:

```markdown
---
type: video
source:
  - intro.mp4
  - demo.mp4 · from 0:12 · to 5:40
  - outro.mp4
join: fade 0.5s
---
```

Times then refer to the joined video (kept in `.si/video/` beside the file
until a source changes).

### Callouts

| Line | What |
|---|---|
| `title "…"` | a large title over a darkened picture |
| `text "…" · at top-right` | text; `· font Inter`, `· size 48` (source pixels), `· color #fff` or `accent`, `fg`, `bg`, `muted`, `surface`; `· box` (on the theme's surface), `· shadow` |
| `bubble "…" · to 820,410` | a speech bubble whose tail points at the place; `· round`, `· think` |
| `arrow from 200,700 to 640,420 · "This queue"` | an arrow, its label at the tail |
| `box RECT` · `circle RECT` · `underline RECT` | outline a place |
| `spotlight RECT` | darken everything else |
| `blur RECT` · `pixelate RECT` | hide a place (done by ffmpeg: the pixels are gone, not covered) |
| `keys "⌘ K"` | key caps, bottom centre |
| `badge "Live"` | a pill, top right |
| `lower-third "Ada Lovelace" "Engineer"` | a name and a role, bottom left |

A callout shows 3 s (`· 4s`, or a range); it comes in with `· fade` (the
default), `· pop`, `· slide-in` or `· type` (the text typed; an arrow draws
itself), and leaves with a fade (`· out none`: at once). `· at POS` moves
`text`, `keys`, `badge` and `lower-third`. Callouts are drawn in the theme's
colours and fonts and scale with the frame, a phone-shaped recording too.

A callout can move: `· to A → B → C` (or `->`) moves a bubble's tail, an
arrow's head or (with `· at A → B`) a text, key caps, a badge or a lower
third through the places in turn; `· path A B C` says the same for any of
them and for a `box`, `circle`, `underline`, `spotlight`, `blur` or
`pixelate` (its top-left corner moves). A rectangle's place can carry a
size (`· to 100,100 300x80 → 700,400 500x200`): the size changes along
the way too; a place without one keeps the size before it. The move takes
the callout's whole length, or `· over 2s` (also written inside: `to A → B
over 2s`); it is eased (`· linear` for a steady speed), the same speed
along every stretch.

```markdown
- 0:12 · bubble "Follow the order" · to 200,640 → 900,420 → 1500,300 over 3s · 4s
- 0:20 · arrow from 1700,900 to 300,200 · path 300,200 820,240 · 2s
- 0:24 · box 100,100 300x80 · path 100,100 700,100 700,600 · linear · 3s
- 0:30–0:36 · blur 80,600 360x60 · to 80,600 → 900,600 → 900,200 480x90 over 5s
```

A moving `blur` or `pixelate` is done by ffmpeg like a still one: the
pixels under the place of the moment are hidden, whatever moves beneath, so
it can follow a token scrolling up a terminal or a window being dragged.

```markdown
- 0:02 · lower-third "Ada Lovelace" "Platform engineer" · 4s · slide-in
- 0:20 · text "Three nodes, one queue" · at top · box · type
- 0:31 · bubble "Leader moved" · to 62%,48% · think · 3s
- 0:40 · arrow from 200,700 to 640,420 · "This queue" · 3s
- 0:44 · box 120,80 400x60 · 2s · out none
- 0:44 · circle 900,500 160x160 · pop
- 0:47 · underline 120,140 400x8
- 0:50–0:56 · spotlight 600,300 500x200
- 1:02–1:10 · pixelate 40,20 300x40
- 1:20 · keys "⌘ ⇧ P" · 1.5s
- 1:25 · badge "Live" · at top-left · 5s
```

### The camera

`zoom to RECT` moves the camera onto the rectangle (its shape fitted to the
frame's) in 0.8 s (`· move 1.2s`), eased (`· linear` for a steady move);
with `· 3s` it holds that long and comes back, without a length it stays
until `pan to RECT` or `zoom out`. Callouts sit in source pixels, so they
zoom with the picture.

```markdown
- 2:40 · zoom to 600,300 900x500
- 2:46 · pan to 100,300 900x500 · move 1.2s
- 2:52 · zoom out
```

### Sound, captions and a new take

| Line | What |
|---|---|
| `T · say "…" · duck 30%` | new narration in the configured voice (`voice:`, `--voice`, si.conf `voice:`); the original sound drops to 30 % while it speaks |
| `0:00 · music bed.mp3 · 15%` | music under the whole video, ducked under the speech |
| `captions on` (or `captions: on`) | `<out>.vtt` and `<out>.srt`: the source's captions moved by the edits, plus the `say` lines |
| `captions words` (or `captions: words`) | the same, and the word being said is marked: the `.vtt` carries a time before each word (si's player and browsers that style `::cue(:past)` follow it) and `--with captions` draws the captions a word at a time (one ffmpeg input for all the pictures, so an hour-long video builds as easily as a minute); word times share each caption's time by letters |
| `A–B · refilm feature "Name"` | run that feature of the demo script again and splice the new take over the range |

```markdown
- 3:30 · say "Now the queue is replicated." · duck 20%
- 0:00 · music bed.mp3 · 12%
- 1:36–1:46 · refilm feature "Ops terminal" · fade
- captions on
```

`refilm` finds the script as `script:`, else the source's sibling
`<name>.md` with `type: demo`, else the one demo script beside it, and runs
only that feature, filmed as `si demo run` would. When the refilm is the
only line, the rest of the video is copied as it was (stream copy up to the
keyframe before the range and from the one after it); only the stretch
between is encoded again.

### Commands

- `si check edit.md`: unknown actions, bad times, anchors not found, times
  past the end, positions outside the frame, missing files, overlapping
  cuts, replaces, speeds and refilms, callouts inside a cut, `music` not at
  0:00, a refilm without its script or feature; every problem with its line.
- `si play edit.md`: the source playing with the callouts, blur and the
  camera drawn live over it, cuts skipped, speeds and freezes played, a
  strip with a marker per line (click to jump); it reloads when you save.
  Inserted and replaced clips, cards and sound are markers; the build is
  exact.
- `si build edit.md [--with captions,vertical,highlight] [--voice engine:name] [-o dir]`:
  the video, and with `--with` the captions variant (`.captions.mp4`, the
  picture over a band), the 9:16 cut (`.vertical.mp4`) and a highlight reel
  (`.highlight.mp4`): the lines marked `· highlight` (each where it shows in
  the edited video; an edit line its range), else the first sentence of
  each chapter (its first caption or `say` line, a breath around it), else
  the first sentence of each caption, joined with short fades.
- In the Studio (`si --web`) a video edit opens in the **Video view**: the
  preview above the text, the strip with a marker per line. Click a marker
  (or a row under the player) to select its line; the line under the cursor
  is marked in the player. Pause and drag a callout on the frame: a
  bubble's tail (`to X,Y`), a text, key caps, a badge or a lower third
  (`at X,Y`), an arrow by either end (`from … to …`), a rectangle (a box,
  circle, underline, spotlight, blur or pixelate) by its middle, resized by
  its corner (`X,Y WxH`); the place is written into that line in source
  pixels, one ⌘Z away. A moving callout (or a moving blur) shows its path
  while paused: a handle at each place (a dashed box at a rectangle's
  places); drag one and that place of `· to A → B` or `· path` is rewritten,
  its size and `over` kept. The line under the cursor that is a `zoom to` or
  `pan to` shows its rectangle on the frame, dragged and resized the same
  way. ⌘B builds the file and plays the result beside it.
- A folder build skips video edits (`si build edit.md` makes one); a folder
  check checks them.

## 33. Animated charts

A ```` ```chart ```` fence draws a chart from a Markdown table, in a slide or
a document page, and makes the numbers move: each number column can be a
build step, so the room sees the bars grow, shrink and change places. The
first column holds the labels; every column after it holds numbers.

````markdown
## Late orders by cause

```chart bar race unit=%
| Cause     | Q1 | Q2 | Q3 |
|-----------|----|----|----|
| Carrier   | 34 | 22 | 12 |
| Warehouse | 18 | 25 | 9  |
| Payment   | 9  | 14 | 21 |
| Address   | 25 | 11 | 6  |
```
````

The info string is `chart`, a type, then options:

| Type | Draws |
|---|---|
| `bar` (the default) | horizontal bars to scale; several number columns: grouped, with a legend |
| `column` | vertical bars, the value on top and the label under each |
| `line` | a line through the rows (an area under the first); x labels that are numbers or dates sit at their real value |
| `area` | a line with every series filled |
| `donut` | the shares round a ring, the first share in the middle |
| `pie` | the shares as a full pie |
| `progress` | bars of 100 (or of a third column, the target) |
| `bar3d` | solid bars in 3D on a floor; several number columns: one row of bars per column (a bar city) |
| `donut3d` | a donut tipped back to show its thickness |
| `funnel3d` | the rows as solid layers, each narrower by its value, stacked |

| Option | What it does |
|---|---|
| `steps` | every number column is one build step: the chart starts on the first and animates to the next at each click; the column's header (2024, Q2…) is shown big in the corner. All steps share one scale |
| `sort` | the rows in order of value, largest first; with `steps`, rows that change place glide to it, each keeping its colour |
| `race` | `sort` + `steps`, each step rescaled to its own top bar: a bar chart race |
| `stacked` | `bar` or `column`: the number columns stacked in one bar instead of side by side |
| `title="…"` | a caption over the chart (and its accessible name) |
| `unit=%` | written after every bare number (`12` → `12%`) |
| `explode="Label"` | `donut`, `pie`, `donut3d`: that slice lifts out, on a build step of its own |
| `id=name` | fences with the same `id=` in a row are one show (below) |

What moves between steps: a bar's length, a column's height, a donut's arcs,
a line's points and every number written (they count up or down); with
`sort` the bars glide to their new rank. The chart's first appearance grows:
bars from zero, a donut drawn round, a line drawn on, 3D bars rising while
the camera turns, funnel layers dropping in one by one. It happens when the
slide arrives or when the build step that holds the chart is shown; in a
document, when the chart is scrolled into view, then it plays through its
steps once, with ← → and Play under it.

**Switch type.** Charts with the same `id=` written one after another are one
show: each one after the first is a build step, and the chart turns from
one kind into the next. Bars and columns fold into a donut's or a pie's
arcs (each bar bends round the ring into its own slice, and unrolls back
the other way); a line becomes an area with its points in place; other
pairs fade the old one out while the new one grows. When the kind stays,
only the numbers move. The same `id=` on the next slide
carries on from what this slide showed, without the slide fade.

````markdown
## On time, by region

```chart bar id=regions
| Region | Orders |
|--------|--------|
| North  | 1,200  |
| South  | 860    |
| East   | 1,540  |
```

```chart donut id=regions
| Region | Orders |
|--------|--------|
| North  | 1,200  |
| South  | 860    |
| East   | 1,540  |
```
````

````markdown
## Orders this quarter

```chart donut3d explode="Late"
| Status  | Share |
|---------|-------|
| On time | 78    |
| Late    | 15    |
| Lost    | 7     |
```
````

````markdown
## Checkout, this week

```chart funnel3d unit=%
| Step     | Share |
|----------|-------|
| Visits   | 100   |
| Cart     | 42    |
| Checkout | 18    |
| Paid     | 15    |
```
````

Colours come from the theme (`--si-accent`, `--si-accent-2`, `--si-ok`,
`--si-warn`, or `--si-ig-1` to `--si-ig-4` when a theme sets them), in light
and dark. Reduced motion, print and PDF, the overview, thumbnails, handouts
and a `#5.2` link show the final values of the step at once. The same types,
the 3D ones too, are the `chart=` of an infographic block (§30).

**Pictures of a chart.** `si build talk.md --png` writes every chart of a
deck or document as a picture, numbered in page order: `talk-chart-1.png`
of its final step and, for a chart with more than one step (`steps`, a
show), one per step, `talk-chart-1-step-1.png`, `talk-chart-1-step-2.png`
and so on. `--svg` writes the same as SVG with real text (fonts embedded,
cut to the glyphs used). A 3D chart is PNG only: CSS 3D has no SVG, and
`--svg` says so and leaves its steps out. `--scale 3` sets the PNG's pixels
per CSS pixel (2 by default).

```sh
si build talk.md --png --svg
```
`si play --terminal` draws the last column as text bars.

`si check` reports a cell that is not a number, an unknown type or option, a
negative share in a donut, pie or funnel, `steps` or `race` with only one
number column, `stacked` on a type that cannot stack, and an `explode=` that
names no row.

## 34. Scenes: objects that move

A ```` ```scene ```` fence draws a few blocks and makes things move between
them, one click at a time: a parcel flies from the shop to the customer, a
token hops down a line of teams, an order splits into parcels, messages
queue up and a consumer takes them. It is written in plain words, in a slide
or a document page.

````markdown
## Tuesday to Thursday

```scene title="Tuesday → Thursday"
box Shop
box Customer
parcel flies Shop → Customer
```
````

**The first lines name the blocks.** `box Shop` is one block (its name is
its label; quote a name with a comma or an arrow in it: `box "A → B"`).
`row A, B, C` names several in a row; `column A, B, C` stacks them in one
column; `grid A, B, C, D` makes a small grid. `box Shop at 20,60` puts a
block at 20% across and 60% down the stage. With no places, si lays the
blocks out for you, left to right in the order they are named, and stacks
the blocks a split or a fan-out reaches in a column after it. Parcels,
tokens and messages need no line: write `parcel`, `token` or `message` in a
step and one appears; `parcel P2` names a second one.

**Every other line is one build step** (one click, one `[CLICK]` in
`Notes:`). `->` works as well as `→`, and `<->` as `↔`.

| Step | What moves |
|---|---|
| `parcel flies Shop → Customer` | a parcel flies along a curve from one block to the other, turning with the path and leaving a dashed trail; `parcel flies → Courier` flies on from where it is |
| `token hops Shop → Payments → Warehouse → Courier` | a token hops from block to block; each block lights up as it lands |
| `Order morphs → Event circle → Message diamond` | the block changes shape (`box`, `circle`, `diamond`, `pill`) and name, one after the other; the new name finds the block in later steps |
| `Order splits → Berlin, Lyon, Milan` | one block sends a parcel to each of the blocks named |
| `parcels merge → Delivery` | every parcel on the stage flies into the block and fuses with it, which lights up (with none on the stage, three wait at the left); or name them: `P1, P2 merge → Delivery` |
| `swap Courier ↔ Stock drift` | two blocks trade places, one over and one under |
| `tiles assemble 3×3` | tiles fly in from everywhere and snap into a 3 by 3 grid; `Plan assembles 3×3` builds the block `Plan` out of them |
| `tiles scatter` / `Plan scatters` | the tiles, or a block in pieces, burst outward and fade |
| `queue orders · Billing takes 3` | five messages line up in a lane called orders (`queue orders 8` for eight) and the consumer takes three from the front while the rest shuffle forward; a later `Billing takes 2` takes more. The consumer needs no box line |
| `Exchange fans out → Email, Stock, Analytics` | a copy of a message leaves the block for each block named; what flew into it goes in |
| `Warehouse opens → Pick, Pack, Ship` | the block grows to fill the stage and opens on its parts, which pop in; the parts need no box line |
| `route Depot → Stop 1 → Stop 2 → Stop 3` | a road draws itself under the blocks with a token riding its tip, and drops a pin at each stop |
| `parcel drops into Locker 12` | the parcel falls into the block, squashes as it lands and kicks up a little dust |

After a `·`, a step takes options: `arc` (the default for a flight) or
`straight`, `slow` or `fast`, or a time for the whole step: `· 2s`,
`· 800ms`. Lines starting with `//` are notes to yourself.

````markdown
## One order, three parcels

```scene
box Order €84
column Berlin, Lyon, Milan
Order €84 splits → Berlin, Lyon, Milan
parcels merge → Lyon · slow
```
````

````markdown
## orders.new

```scene title="orders.new"
queue orders · Billing takes 3
Billing takes 2
```
````

````markdown
## Who hears about an order

```scene title="order.placed"
box Shop
box Exchange
column Email, Stock, Analytics
message flies Shop → Exchange · straight
Exchange fans out → Email, Stock, Analytics
```
````

````markdown
## Inside the warehouse

```scene
row Shop, Warehouse
token hops Shop → Warehouse
Warehouse opens → Pick, Pack, Ship
```
````

````markdown
## Route 7, Thursday

```scene
row Depot, Stop 1, Stop 2, Stop 3
route Depot → Stop 1 → Stop 2 → Stop 3
parcel drops into Stop 3
```
````

The scene is one SVG that scales with the slide or the page (its stage is
16:9, wider when many blocks stand in a row). Blocks are the theme's
`--si-surface-2` with `--si-fg` text, lit blocks `--si-accent`; parcels,
tokens and pins are `--si-accent-2`, in light and dark. On a slide the
scene starts before its first step and each line is a click; in a document
it plays once when it is scrolled into view, with ← → and Play under it.
Reduced motion, print and PDF, the overview, thumbnails, handouts and a
`#5.2` link show the end of the step at once.

**Pictures of a scene.** `si build talk.md --png --svg` writes each scene
after each step, `talk-scene-1-step-1.png`, … and `talk-scene-1.png` of its
end, as charts do (§33). `si play --terminal` shows the blocks as text and
lists the steps.

`si check` reports, with the line in the fence, a line with no verb, a
block that was never named (with the names there are), an unknown option, a
block used where a parcel, token or message should be (`Shop flies …`), a
`takes` with no queue, and a grid or a route that cannot be drawn.

## 35. Architecture pages

`type: architecture` turns one Markdown file into a design document: the
`##` sections are numbered, the first paragraph after each is its lead, and
the blocks under it take a design-doc look. It is an article (§19) with a
section grammar; everything an article holds still works.

````markdown
---
type: architecture
title: Orders — Architecture & Design
subtitle: One writer, events for everyone else
---

## The orders service owns the data

Only one service writes an order; the others learn about it from the stream.

- `invariant` **One writer** — only the orders service writes the orders table.
- `invariant` **Events, not calls** — the warehouse hears about an order from the stream.
- `boundary` **Public edge** — the gateway is the only thing the internet reaches.

> Honest read: the stream is one broker today; it gets a replica before launch.

## How a checkout flows

From the browser to the warehouse, one hop at a time.

```flow
dir: LR
B[Browser] --> G[Gateway] -->|POST /orders| O[Orders] -->|order.created| W[Warehouse]
dot B..W 2s
```

## The data

**Order** — what was bought, by whom, and its state
**Parcel** — one box of an order, with its tracking number

| Region | Orders / day | Brokers |
|--------|--------------|---------|
| EU | 40 000 | 1 |
| US | 65 000 | 1 |

## Phases

1. One region, one broker. (now)
2. A broker replica and a second region.
3. Each region takes the writes of its own customers.
````

| You write | You get |
|---|---|
| `## Section` | a numbered section: a badge (1, 2, 3 …) and the title. `number: off` in the front matter drops the numbers. |
| the first paragraph after a `##` | the section's lead: a quieter introductory line (unless it is a definition list, below) |
| a list whose **every** item reads `` - `tag` **Title** — body `` | a grid of cards: the tag chip, the title, the body. The tag is optional (`- **Title** — body`); `—`, `–`, `--`, `-` or `:` separate the title from the body. A list where only some items read that way stays a list, and `si check` says so. |
| a table | a bordered table on a card |
| a quote (`> …`) | a note: a tinted box for caveats ("Honest read: …") |
| a ` ```flow ` fence | the stage fence (§13) by another name, on a card |
| a paragraph of `**Name** — what` lines | a definition list, two columns on a wide page |
| the ordered list of a section titled `Phases` | a row of phase cards (Phase 1, Phase 2 …); the item ending in `(now)` is highlighted and the mark itself is not shown |

The page has no contents column: the numbered sections are the outline.
`subtitle:` is the line under the title (`description:` wins when both are
set). Light and dark, the theme toggle, `--css`, `si build --pdf` (one
printed page per sheet, cards kept whole) and `--md` / `--docx` / `--rtf` /
`--epub` (§21) all work as for an article.

## 36. Status boards

`type: status` (or `status-board`) turns a checklist into a project status
board: a row of pills, KPI tiles, the overall progress, the shipped chips,
then one card per area with **Done**, **Missing** and **Improve** columns.
Every percentage, count and bar is computed from the lists, so none can
drift from them.

````markdown
---
type: status
title: Orders — Status Board
subtitle: Checkout and the warehouse
updated: 2026-11-22
meta:
  - build: green | ok
  - release: next week | warn
  - tests: 214 passing
tiles:
  - num: "{pct}"; lbl: Done; sub: done of done + missing
  - num: "{open}"; lbl: Still open; sub: {missing} missing · {improve} improve
  - gate: Load test; tone: warn; lbl: The one gate left; sub: 10k orders a minute
---

## Checkout

payments · receipts

### Done
- Cart and payment `v1.2 · 3f9c2a1`
- Receipts by email
- Faster address lookup `8d01b77` `improvement`

### Missing
- Saved cards `P1`
- Refunds `partial`

### Improve
- One-click reorder

## Warehouse

### Done
- Pick lists

### Missing
- Returns `P0`
- Stock alerts `gated`

### Improve
- Batch printing

## Shipped
- Receipts by email `3f9c2a1`
- Pick lists `8d01b77`

## What the words mean

Done is merged with tests green. A partial or gated item stays in Missing.
````

**Front matter.** `updated:` is the first pill. Each `meta:` item is a pill,
`label: value | tone`; the tone is `ok` (green), `warn` (amber) or `plain`
(the default). Each `tiles:` item is a tile of `key: value` pairs separated
by `;`: `num:` (the big value), `lbl:` (the label), `sub:` (the line under
it), `tone:` (`ok`, `warn`), and `gate:` in place of `num:` for the wide
highlight tile. A tile's text may use the board's own numbers: `{done}`,
`{missing}`, `{improve}`, `{open}` (missing + improve), `{pct}` (overall %
done), `{items}` and `{areas}`. `meta:` and `tiles:` take their items as
`- …` lines under them.

| You write | You get |
|---|---|
| `## Area` with `### Done`, `### Missing`, `### Improve` lists | an area card: its bar, its counts and % done, the three columns (green, amber, orange). A column may be left out or empty. |
| a line under `## Area`, before its first `###` | the area's scope, beside its title |
| a trailing `` `P0` `` … `` `P9` `` | a priority tag |
| a trailing `` `partial` ``, `` `in progress` `` or `` `wip` `` | an in-progress badge (orange) |
| a trailing `` `gated` ``, `` `gate` ``, `` `blocked` `` or `` `uncommitted` `` | a gated badge (amber) |
| a trailing `` `improvement` `` | the Improvement badge: an Improve item that shipped and moved to Done |
| any other trailing code span (`` `v1.2 · 3f9c2a1` ``) | a version · hash, in mono |
| `## Shipped` (any title starting with Shipped) and a list | the chips row; a trailing code span is the chip's hash |
| any other `##` section | a note, as Markdown; a list in it is not counted (a "Next ideas" list) |
| text before the first `##` | an introduction above the areas |

**The numbers.** Per area and overall: % done = done ÷ (done + missing),
rounded; Improve makes a thing better and does not count against it being
done (nothing missing is 100%). The bar's three segments are done, missing
and improve over every item. Each bar's label carries the numbers for
screen readers.

The colours are the theme's status colours: `--si-done` (default
`--si-ok`), `--si-missing` (`--si-warn`) and `--si-improve` (between
`--si-warn` and `--si-crit`); a theme or `--css` may set its own.
`si build --pdf` prints the board (cards kept whole). `--md` and the other
publisher exports are for articles and refuse a board with a message.

`si check` reports an unknown `### column` (with the three it knows), a
list item before the first column of an area, a column written twice, a
board without areas, a pill with an unknown tone and a tile with an unknown
key or no `num:`; it warns about an area without items and a line in a
column that is not a list item.

## 37. Sandbox labs

A lab can bring its own machine: `sandbox:` names a Docker image, and `si
play` makes a box from it before the page opens. The box has Docker inside
it (the learner runs `docker` and `docker compose` there), the lab's files
in `/lab`, and nothing of your own computer. The terminal drawer's tabs open
shells in the box, **✓ Check** runs the page's checks in the box, and links
written `sandbox:PORT` open the box's ports. When `si play` stops, the box
goes.

```markdown
---
type: lab
title: Put a Proxy in Front of the Shop
sandbox: docker:28-dind
sandbox-files: env
sandbox-setup: docker compose pull -q
sandbox-cleanup: docker compose down
sandbox-ports: 8080 Shop
sandbox-limits: memory=2g cpus=2
---

# Put a Proxy in Front of the Shop

## Start the shop

    docker compose up -d

<!-- check: docker compose ps --services --status running · expect "proxy" -->

## Look at it

Open [the shop front](sandbox:8080).
```

| Key | What it does |
|---|---|
| `sandbox:` | the image, pinned: `name:tag` or `name@sha256:…`. A Docker-in-Docker image (`docker:28-dind`, or your own built from it) gives the learner Docker; any other image gives a plain box. `si check` warns about an image with no tag or `:latest`. |
| `sandbox-files:` | a folder beside the page, copied into `/lab` (files and folders; links are skipped) |
| `sandbox-setup:` | a command run in `/lab` once Docker in the box answers (pull images, seed data); a failure stops `si play` with its output |
| `sandbox-cleanup:` | a command run in the box before it is removed (30 s at most) |
| `sandbox-ports:` | the box's ports the learner opens: `15672 Console, 8080 App` |
| `sandbox-limits:` | `memory=2g cpus=2 pids=2048` (these are the defaults) |

**In the page.** `[Console](sandbox:15672)` (and `sandbox:15672/path`)
becomes a link to `/_si/sandbox/port/15672/`, which `si play` sends on to
where the box publishes that port. `GET /_si/sandbox` says what runs:
the image, whether it runs unprivileged and its ports. A page's checks run
with `sh -c` in the box's `/lab`; a page without `sandbox:` keeps running its
checks on this computer.

**The sandbox host.** Boxes do not run on your own Docker. `si sandbox host
init` makes a Colima VM, `si-sandbox`, for them:

```text
si sandbox host init                 # once: Docker, sysbox, no host folder mounted
si sandbox host status               # VM, socket, mounts, sysbox, room for boxes
si sandbox host start | stop | rm
```

- The VM mounts one empty folder, `~/.si/sandbox/share`, never your home
  folder (Colima mounts the home folder by default); `init` and `status`
  check that the VM sees nothing else.
- sysbox (a pinned, checksummed release) runs every box unprivileged: root in
  the box is not root in the VM. A host without sysbox refuses boxes unless
  `si play --trusted-sandbox`, which runs them privileged: only for an image
  and people you trust.
- Your own `docker` context is left as it is.
- `init` sizes the VM to half the cores, three quarters of the memory and
  100 GiB of disk (`--cpu`, `--memory`, `--disk` to choose); a box takes 2 GiB
  unless `sandbox-limits:` says otherwise.

`sandbox.host:` in si.conf picks another host: `docker` (your own Docker:
`$DOCKER_HOST`, else `/var/run/docker.sock`) or a socket path;
`$SI_SANDBOX_HOST` wins over it.

**Your own image.** `si new web-lab --type sandbox` writes a lab with its
`env/` folder and a `Dockerfile` from `docker:28-dind`. Build it into the
sandbox host and name it in `sandbox:`:

```text
si sandbox build web-lab -t northwind/web-lab:1.0
si sandbox try web-lab/lab.md        # the box as si play makes it, a shell here
```

`si play` runs one box, for the file it serves or a folder's top-level
pages: pages that name the same image share it; two different images are a
problem. Pages in subfolders never start one (serving a folder of examples
starts no Docker). `sandbox:` in a course's `_course.md` gives the whole
course one box, and every lesson's checks run in it. `si serve`
and built pages have no box: the page shows the commands, and a
`sandbox:` link has nowhere to go. Boxes left by a `si` that was killed are
removed the next time one starts.

**The drawer on the lab page.** A lab page served by `si play` (or to a
class seat) has a **>_ Terminal** button (and the `` ` `` key) for the
drawer, and every ```` ```bash ```` (`sh`, `shell`, `zsh`, `console`) block a
**Run** button that types it into the drawer's shell, `$ ` prompts left out.
A built page has neither.

### Classes

`si class` serves a course with a `sandbox:` to a class from your computer:
each learner gets a seat code, and their own box from the course's image
when they first open the course. Learners reach only shells in their own
box: never a shell, a file or a page of si's on your computer.

```text
si class new docker-course --name "Docker basics" --roster people.csv --seats 5
si class start docker-basics --tunnel     # a link per seat; Ctrl+C removes every box
si class                                  # the planner: classes, seats, codes, live boxes
```

| Command | What it does |
|---|---|
| `si class` | the planner, a keyed page on this computer only: plan a class (course, image, memory and CPUs per box, how long a box lives, when it is removed idle, the last day, seats from a roster or a count), a printable sheet of codes with a QR code per seat, start and stop it, each seat's box live (starting, ready, its memory), **Reset** (a fresh box from the image), seats added or removed on the day, the images on the sandbox host and **Pull** |
| `si class new <course>` | plan a class of a course folder (its `_course.md` or a top-level lab has `sandbox:`) or of one lab page: `--name`, `--seats N`, `--roster file.csv` (rows `name,email[,code]`, a header row skipped), `--image` (instead of the course's), `--memory 2g`, `--cpus 2`, `--ttl 3h`, `--idle 30m`, `--expires YYYY-MM-DD` |
| `si class list` / `codes <id>` / `rm <id>` | the classes planned here, a class's seats and codes, forget a class (its codes stop working) |
| `si class start <id>` | serve the class and open the planner: `--tunnel` (a public HTTPS address, cloudflared; the default when it is installed), `--https` (this network, si's own certificate: a learner's device trusts it once at `http://<address>/ca`), `--local` (this computer, to try it); `--budget 48g`, `--addr`, `--trusted-sandbox` |

A class is a file, `.si/classes/<id>.json` (0600: it holds the codes). A
code is 8 letters and digits, printed `ABCD-EFGH`, typed in any case;
`<link>/go/ABCD-EFGH` (or `<link>/go` and typing it) gives the learner's
browser the seat (an HttpOnly cookie, Secure over HTTPS) and opens the
course. The same code comes back to the same box; a box removed for its
`--ttl` or `--idle` comes back fresh. Wrong codes are slowed down (60 a
minute for the whole class), and a class past its last day lets nobody in.

**What a learner reaches**: the course's pages, the drawer (each tab a
shell in their box; the Viewer and file search, which read your folder, are
not there), **✓ Check** run in their box, their box's `sandbox-ports:`
(proxied: the box publishes them on your computer only), and the course's
results for the class view (**Progress** in the planner: the live class
view of §28). The page sources, `si.conf`, `.si/` and si's own pages are not
served. Through a quick tunnel, which holds a stream back, the drawer reads
its shells by long poll: a key comes back in about 150 ms.

**The budget.** Every box takes its memory (`sandbox-limits:` or
`--memory`) from the budget: 80% of the sandbox host's memory unless
`--budget` says otherwise. A seat whose box would go past it waits until a
box goes (Reset, `--ttl`, `--idle`, Stop). `si sandbox host init` sizes the
VM to three quarters of your memory: on a 128 GB Mac a 96 GB VM, about 77
GB of budget, about 38 learners at 2 GB.
