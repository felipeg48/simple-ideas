# 36. Status boards

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
