# 33. Animated charts

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
