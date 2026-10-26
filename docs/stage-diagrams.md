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
dir: LR                      # flow direction: LR (default) or TB
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
  (`A`). Keep labels short — they do not wrap.
- **`A --> B --> C`** chains; **`A -->|text| B`** puts `text` on the arrow.
- **`dot A..B [dur]`** and **`flow A..B [dur]`** animate the shortest route from
  `A` to `B`. A bare number means seconds (`2` → `2s`); default `1.6s`.
- **`step <node...>`** adds a build step that lights those nodes (cumulatively)
  as you click through the slide — see example 7.
- A block with an unknown node, an impossible route, or a bad `dir` is reported
  by `si check`.

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
Ingest --> Validate --> Transform --> Load
step Ingest
step Validate
step Transform
step Load
```

Each `step` line is one build step: clicking through the slide lights `Ingest`,
then `Validate`, and so on, each staying lit (the accent outline). This uses the
same step runtime as a hand-authored canvas — the slide's step counter reflects
the `step` lines automatically. List more than one node on a line to light them
together (`step Transform Load`). `step` composes with `dot`/`flow`: the dot
keeps looping while you walk the steps.

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
- **Step reveal is highlight-only.** `step` lights nodes with the accent outline;
  it does not hide/show nodes or re-colour edges. For richer step-driven state
  (show/hide, ok/alert per shape), hand-author a canvas — see
  `docs/authoring-canvases.md` and RULES.md §10.

## How it renders (for the curious)

The generator assigns each node a column by its longest path from a root, stacks
same-column nodes into centred rows, draws arrows clipped to box borders with a
rotated head, and emits the dot as a CSS `offset-path` animation and the flow as
an animated `stroke-dasharray`. One `@media (prefers-reduced-motion: reduce)`
block parks both. It is all in `internal/render/stage.go`.
