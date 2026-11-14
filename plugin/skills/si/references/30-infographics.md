# 30. Infographics

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

## The path · flow
Producer -[publish]-> Exchange -[route]-> Queue -[deliver]-> Consumer
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

Boxes meet by name across lines, so several lines make branches and merges.
si lays them out (columns by the longest path, siblings stacked): left to
right, or top to bottom in a tall frame (`phone`, `story`). The arrows are
numbered in the order written; animated, the boxes appear and the arrows draw
in that order, then a packet travels along each arrow in turn. A flow block is
`wide` unless it says `span=`. `si check` reports an arrow without a box on
one side and a label without its `]->`, at its line.

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
  as text, pictures inside, theme colours resolved; background glows and
  shadows left out). `si play` presents a landscape one.
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
