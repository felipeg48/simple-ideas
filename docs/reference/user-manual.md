---
title: The si User Manual
author: simple-ideas
theme: crdb
aspect: 16:9
---

# The si User Manual

Markdown in, animated fluid HTML slide decks out.

Notes: (1 min)
This manual is itself an si deck — build it with `si manual --slides` or `si manual --pdf`. For the exhaustive directive list, run `si rules`.

---

## What si is

- Write a talk in **plain Markdown**; `si` renders animated, fluid HTML slides
- `si play` serves them with live reload; `si build` writes one self-contained HTML file per deck
- **Fluid, never fixed:** slides reflow from a phone to a 4K screen — no 1280×720 canvas
- Self-contained output: CSS, JS, SVG, images and video in a single file

---

## A deck is just Markdown

```markdown
---
title: My Talk
theme: crdb
---

# My Talk
A subtitle under the title.

---

## A content slide
- a point
- another point

Notes: (1 min)
Speaker notes for the teleprompter.
```

- Front matter sets `title`, `theme`, `aspect`, `mode`, and more
- `---` on its own line splits slides; `#` is the title slide, `##` a content slide

---

## The core syntax

| Write | You get |
|---|---|
| `# Title` | the title slide |
| `## Heading` | a content slide |
| `---` | a slide break |
| `- item` | a bullet list |
| ` ```lang ` | a highlighted code block |
| `Notes: (2 min)` | speaker notes + a per-slide time budget |
| `<!-- layout: two-col -->` | a layout directive (see next) |

Everything is in **`si rules`** (or RULES.md) with an example for each.

---

## Layouts

<!-- layout: two-col -->

Pick a layout per slide with `<!-- layout: name -->`:

- `two-col` — two columns (`<!-- column -->` splits them)
- `big-number` — one huge figure
- `quote` — a centred pull-quote
- `center` / `image` — centred / full-bleed image

<!-- column -->

```markdown
<!-- layout: big-number -->

**99.99%**

uptime across three regions.
```

---

## Build steps

Reveal content one keypress at a time:

<!-- steps -->
- First this appears
- then this
- then this — each on its own press

```markdown
<!-- steps -->
- First this appears
- then this
```

Notes: Steps also drive canvas and stage state (next slides).

---

## Stage diagrams

A ` ```stage ` fence generates a themed, animated flow — no external tool:

```stage
dir: LR
Draft --> Review --> Ship
dot Draft..Ship 2s
```

- Chains, fan-out and merges; node labels `A[Label]`, edge labels `-->|x|`
- `dot`/`flow A..B` animate a route; `step <node>` reveals on build steps
- Full guide: `docs/stage-diagrams.md`

---

## Animated canvases

One inline SVG whose parts change state slide by slide:

```markdown
<!-- svg: journey.svg paid -->
<!-- steps: +shipped -->
- the order ships
```

- Helper attributes: `data-show/hide/ok/alert/flow/packets/grow/move/count`
- State words in the markdown drive CSS transitions in the SVG
- Scaffold one with **`si new-canvas`**; recipe in `docs/authoring-canvases.md`

---

## Themes

- Four built-ins: `studio`, `paper`, `mono`, `crdb`
- `theme: my.css` loads a CSS token file; `theme: https://…/x.css` fetches one
- `si --export themes` writes a gallery; `si --export css-rules` a starter stylesheet
- Your own CSS layers in `si.user` after the theme — tokens are `--si-*`

---

## Presenting

| Key | Action |
|---|---|
| → ← Space | next / previous step or slide |
| `o` | overview grid |
| `p` | teleprompter (notes, timers, rehearsal pacing) |
| `t` | break timer (configurable presets + animation layer) |
| `a` | draw over the slide |
| `d` / `f` / `?` | light-dark / full screen / help |

Open on a phone: `si play --addr 0.0.0.0:7070` prints a LAN URL + QR.

---

## The CLI

```bash
si play <dir>              # serve with live reload
si build <dir> --pdf       # self-contained HTML (+ PDF)
si check <dir>             # validate; exit 1 on a problem
si new <name>              # scaffold a deck
si new-canvas <name>       # scaffold an animated canvas
si manual --slides         # this manual
si rules                   # the full format reference
si migrate <src>           # an AI migration prompt for an existing deck
```

---

## Where to go next

- **`si rules`** — the complete, authoritative directive reference
- `docs/stage-diagrams.md` — the stage flow DSL
- `docs/authoring-canvases.md` — hand-authored animated canvases
- `docs/timer-animations.md` — the break-timer animation layer
- `docs/reference/developer-reference.md` — architecture and how to extend si

Notes: (0.5 min)
Thanks — now go build a deck.
