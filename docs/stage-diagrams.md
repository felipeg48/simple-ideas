# Stage diagrams: the `stage` fence

A ` ```stage ` fence turns a few lines of text into a **themed, animated flow
diagram**. It is si's native, offline alternative to Mermaid for *simple flows*
— chains, fan-out and merges. Everything is generated as one self-contained SVG
coloured with the deck tokens: no runtime script, scales by `viewBox`, works in
light and dark, and every motion pauses under `prefers-reduced-motion`.

Reach for `mermaid` instead when you need a dense graph (many crossing edges,
subgraphs, cycles) — `stage` lays out simple flows well and deliberately does
not do crossing-minimisation.

## The whole language on one screen

```
dir: LR                      # flow direction: LR (default), RL or TB
labels: flat                 # keep edge labels level (default: along the edge)
A --> B --> C                # a chain: three boxes, two arrows
A --> B                      # one edge per line also works
Cache -->|hit| Fast          # label an edge between pipes
Pay[Payment service]         # id[Label]: short id, friendly label
dot A..C 2s                  # a dot loops along the route A->C, every 2s
flow A..C                    # marching dashes along the route (default 1.6s)
# lines starting with # are comments
```

Rules of the grammar:

- **Nodes** are discovered from edges in first-seen order. Write a node alone on
  a line to place it without an edge yet.
- **`A[Label]`** gives node `A` a display label; later references can stay bare
  (`A`). Keep labels short — they do not wrap; a long one is set smaller so
  it stays inside its box whatever the theme font.
- **`A --> B --> C`** chains; **`A -->|text| B`** puts `text` on the arrow.
  A label wider than the gap between two columns (a routing key such as
  `-->|order.*.created|`) widens that gap so it clears the boxes and the
  arrowhead; past twice a box's width it is set smaller instead. Top to
  bottom (`dir: TB`), labels on sibling edges sit side by side between two
  rows: the gap between the siblings widens until no two labels overlap (up
  to twice a box's width; past that they are set smaller).
- **Labels follow their edge.** On a slanted edge (a fan-out, a merge, a
  diagonal) the label is turned to the edge's angle, centred on it and set
  just above the line, so it reads as part of its arrow. It is always
  upright: a label on an edge heading left is turned half round, never shown
  upside down. Labels on level and upright edges, and at the middle of a
  bowed skip edge, stay level. `labels: flat` keeps every label level.
- **`dir: RL`** flows right to left: the `LR` layout, mirrored.
- **`dot A..B [dur]`** and **`flow A..B [dur]`** animate the shortest route from
  `A` to `B`. A bare number means seconds (`2` → `2s`); default `1.6s`.
- **`step <node...>`** adds a build step that acts on those nodes (cumulatively)
  as you click through the slide. An optional op leads the line — `highlight`
  (default), `ok`, `alert`, `show`, `hide` — see example 7.
- A block with an unknown node, an impossible route, a bad `dir` or a
  `labels:` other than `along` or `flat` is reported by `si check`.

## Examples

### 1. A simple chain with a travelling dot

```stage
dir: LR
Cart --> Checkout --> Payment --> Shipped
dot Cart..Shipped 2.2s
```

The dot rides the whole pipeline on a loop. This is the bread-and-butter case:
a process with ordered steps.

### 2. Friendly labels, short ids

```stage
dir: LR
Req[Request] --> Val[Validate] --> Pay[Charge card] --> Done[Confirmation]
dot Req..Done 2s
```

Use `id[Label]` when the ids you reference in `dot`/`flow` want to stay short
but the boxes should read nicely.

### 3. Fan-out: one step branches

```stage
dir: LR
Request --> Auth
Auth -->|ok| Handler[Run handler] --> Response
Auth -->|denied| Reject[403 Reject]
dot Request..Response 2s
```

`Auth` fans out to two children; siblings stack into rows. The dot follows the
happy path `Request → Auth → Handler → Response`.

### 4. Top-to-bottom branch

```stage
dir: TB
Ingest --> Parse
Parse -->|valid| Store
Parse -->|invalid| Drop
flow Ingest..Store 1.4s
```

`dir: TB` flows downward — often the clearest shape for a decision that splits
two ways. Here dashes trace the valid route.

### 4b. Labels along slanted edges

```stage
dir: LR
CP[Control plane] -->|most free RAM| VM1[VM 1]
CP -->|place session| VM2[VM 2]
```

Both arrows leave `Control plane` at a slant, so each label is turned to its
arrow and sits just above it. Add `labels: flat` to keep them level, or write
`dir: RL` to run the same flow right to left (the labels stay upright).

### 5. Diamond: fork then re-merge

```stage
dir: LR
Read --> Cache
Cache -->|hit| Fast[Return fast]
Cache -->|miss| Disk[Read disk] --> Fill[Fill cache] --> Fast
dot Read..Fast 2.4s
flow Read..Disk 1s
```

A read either hits the cache (short path) or misses and rebuilds it (long
path), then both reach `Return fast`. The short "hit" edge **bows below** the
row to bypass the boxes between — that curve is the shortcut.

### 6. Dashes and a dot together

```stage
dir: LR
Client --> Edge --> Service --> DB[Database]
flow Client..DB 2s
dot Client..DB 2s
```

`flow` draws the path as marching dashes; `dot` sends a marker along it. Using
both reads as "traffic flowing, with a request in flight."

### 7. Step-driven: walk the flow click by click

```stage
dir: LR
Submit --> Check --> Deploy --> Done
step Submit
step ok Check
step alert Deploy
step show Done
```

Each `step` line is one build step. The first word can be an op that changes how
the node reacts:

| op | effect |
|---|---|
| `highlight` (default) | accent outline |
| `ok` | green outline + label |
| `alert` | red outline + label, pulsing |
| `show` | the node starts hidden and fades in on this step |
| `hide` | the node fades out on this step |

List more than one node after the op to act on them together (`step ok A B`).
Plain `step A` is `highlight A`. This uses the same step runtime as a
hand-authored canvas — the slide's step counter reflects the `step` lines
automatically, and `step` composes with `dot`/`flow` (the dot keeps looping
while you walk the steps).

## Sizing: make diagrams big

A stage diagram sizes like any figure: it fills the width and is capped in
height, so **wider diagrams render larger**. A tall fan-out on a slide with a
big two-line title gets squeezed. If a diagram looks small:

- Put it on a `##` content slide rather than a `#` title slide.
- Give a complex diagram its own slide with a short heading.
- Prefer `dir: LR` for long chains (wide), `dir: TB` for two-/three-way splits.

## What it does not do

- **No crossing minimisation.** Deep multi-row merges can overlap; that is the
  boundary where you switch to the `mermaid` fence.
- **No manual positioning, styling per node, or subgraphs.** The point is that
  a flow is described, not drawn — the theme and layout are automatic.
- **Step ops act on whole nodes.** `step` highlights/oks/alerts/shows/hides
  nodes; it does not recolour individual edges or animate along a route on a
  step. For finer step-driven state, hand-author a canvas — see
  `docs/authoring-canvases.md` and RULES.md §10.

## How it renders (for the curious)

The generator assigns each node a column by its longest path from a root, stacks
same-column nodes into centred rows, draws arrows clipped to box borders with a
rotated head, and emits the dot as a CSS `offset-path` animation and the flow as
an animated `stroke-dasharray`. One `@media (prefers-reduced-motion: reduce)`
block parks both. It is all in `internal/render/stage.go`.
