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
