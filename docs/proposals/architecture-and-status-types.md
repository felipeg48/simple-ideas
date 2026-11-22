---
type: article
title: "Proposal: `type: architecture` and `type: status` document types"
description: Two new si document types that turn one Markdown file into a sectioned architecture/design page or a project status board — the layouts prototyped in cockroach-education-platform/docs/{architecture,status-board}.html, made native to si.
---

# Proposal: `architecture` and `status` document types

Status: **proposed** · Author-facing feature · Implements two layouts already prototyped as
hand-written HTML (see **Reference artifacts** below).

## 1. Summary

Add two document types to si, selected by front matter like every other type
(`type: deck | lab | article | guide | book | demo | site | page`):

| `type:` | One file becomes… | Think of it as |
|---------|-------------------|----------------|
| `architecture` | a **sectioned design/architecture page** — numbered sections, each with a lead line and a panel of labelled cards, plus tables, flow diagrams, and notes | a long-form `article` with an opinionated "design doc" section grammar |
| `status` (alias `status-board`) | a **project status board** — a meta pill row, KPI tiles, an auto-computed progress bar, and per-area cards with Done / Missing / Improve columns | a dashboard rendered from a plain checklist |

Both build to **one self-contained HTML file** (same pipeline as `type: article`: `si build`,
`--css`, light/dark, theme toggle, offline), and both are **read-only** documents (no deck
runtime, no `setup:`/`start:`).

**Why native, not hand-HTML:** these two layouts were authored by hand for the Cockroach
Education Platform and are maintained by editing ~450 lines of HTML + CSS each — exactly the
toil si exists to remove. Authors should write Markdown and get the layout for free, the same
way `type: article` gives a byline + contents column.

## 2. Reference artifacts (visual source of truth)

The target rendering is prototyped in these files (committed in the CEP repo); the
implementation should reproduce their layout and tokens:

- `cockroach-education-platform/docs/architecture.html` → `type: architecture`
- `cockroach-education-platform/docs/status-board.html` → `type: status`

Both already share si-compatible chrome: a `.wrap` container, a `header.top`
(logo · title · `subtitle` · **theme toggle** cycling auto/light/dark), a self-contained
`<style>`, and a footer. That chrome is identical to what an `article` page needs — reuse it.

## 3. Shared foundations

Both types inherit the document scaffolding si already has for `article`:

- **Front matter:** `title` (required), `subtitle` (the grey line under the title),
  `theme:` / `--css`, standard build flags. Self-contained HTML, inlined CSS, no network.
- **Theme:** the existing auto/light/dark toggle and tokens. The reference palette:
  `--accent #5a3df0`, `--bg #f6f7f9`, `--surface #fff`, `--text #1b1f24`, `--muted #5b6673`,
  `--border #d8dee6`, `--shadow`, `--mono`, and the status triad **`--done #2e8540` ·
  `--missing #9a7400` · `--improve #e8590c`** (each with a `-bg` tint). A user `--css` overrides
  them, so the CU brand (Data Purple, Geist) drops in unchanged.
- **Numbering & IDs:** headings get stable slug ids and anchor links, as in `article`.

Everything below is the *type-specific section grammar* layered on top.

---

## 4. `type: architecture`

### 4.1 Front matter

```yaml
type: architecture
title: Cockroach Education Platform — Architecture & Design
subtitle: Self-hosted Docker lab platform · Go control plane, no k8s
number: true        # auto-number top-level sections (default: true)
```

### 4.2 Section grammar (Markdown → layout)

| You write | You get |
|-----------|---------|
| `## Section title` | a **numbered section** — a round `.n` badge (1, 2, 3 …) + the title. Numbering is automatic and in document order; `number: false` turns it off. |
| the first paragraph after an `## ` | the section's **`.lead`** (larger, muted intro line). |
| a **bulleted list** whose items read `` - `tag` **Title** — body `` | a **`.panel` of `.mini` cards**: `` `tag` `` → the `.tag` chip, `**Title**` → the card `<h3>`, the rest → the card body. The backtick tag is optional (`- **Title** — body`). 1-4 cards reflow as a responsive grid (`grid3`). |
| a **table** | a bordered, shadowed table inside a panel (used for capacity/cost matrices). |
| a **blockquote** (`> …`) | a **`.note`** callout (left-rule, tinted) — for "Honest read:" / caveats. |
| a fenced **` ```flow `** block (reuse si [stage diagrams](stage-diagrams.md)) | an **SVG flow diagram** with a `.legend` of swatches. Nodes/edges use the diagram hues (`--c-client`, `--c-edge`, `--c-ctrl`, `--c-data`, `--c-node`). |
| a section titled **Phases** with an ordered list | the **`.phases`** horizontal timeline; the item marked `(now)` gets the `.now` highlight. |
| `**Name** — description` lines under a `### ` | a **`.dlist`** (used for the data model: Cohort / Session / Environment / Check …). |

### 4.3 Example

```markdown
---
type: architecture
title: CEP — Architecture & Design
subtitle: Docker lab platform · Go control plane, no k8s
---

## The control plane is the scheduler

CEP runs no Kubernetes and no per-session VMs. A Go control plane drives the Docker
Engine API directly — it *is* the scheduler.

- `invariant` **One binary** — `cep ops` provisions, proxies, serves content, runs checks.
- `invariant` **Budget, not count** — admission is by a RAM budget, never a container cap.
- `safe nesting` **Nested-Docker box** — the student runs CRDB inside their own sandbox.

> Honest read: privileged DinD for the demo; `sysbox-runc` before untrusted users.

## Capacity, cost & the Phase-1 plan

Measured per session (one DinD box, 1–9 nested CRDB nodes):

| Nodes | RAM/box | Typical track |
|-------|---------|---------------|
| 1 | ~0.6 GB | simple SQL |
| 3 | ~1.5 GB | most labs |
| 9 | ~2.5 GB | multi-region |

## Phases

1. Single host, privileged DinD. (now)
2. Sysbox + bigger VM.
3. Multi-host placement; the control plane stays the scheduler.
```

### 4.4 Component / CSS inventory (from `architecture.html`)

`wrap top brand logo subtitle themebtn meta pill · section > h2 > .n + .lead · panel · mini
(+ .tag + h3 + p) · grid3 · table · flow + flow-lbl + legend + sw · note · dlist · phases +
phase + now`. Diagram hue tokens: `--c-client/-edge/-ctrl/-data/-node` (+ `-bg`) and `--line`.

---

## 5. `type: status` (alias `status-board`)

### 5.1 Front matter

```yaml
type: status            # or: status-board
title: CEP — Status Board
subtitle: Self-hosted Docker lab platform · Go control plane, no k8s
updated: 2026-11-22
meta:                   # the pill row (value · tone: ok | warn | plain)
  - git: on main · committed        | ok
  - build: go build ✓ · vet ✓       | ok
  - e2e: verified on Colima         | ok
  - tests: unit 11/11               | plain
tiles:                  # KPI tiles across the top
  - num: "25";  lbl: Sessions / VM;      sub: Phase-1 budget
  - num: "✓";   lbl: Shipped & committed; sub: lifecycle · Run · colors
  - num: 1/1;   lbl: Repos green;        sub: cep-ops builds + vets
  - gate: "Passed ✓ on Colima"; lbl: End-to-end gate; sub: create → CRDB → teardown
```

`meta` → the `.meta` pill row (tone picks `.ok` green / `.warn` amber / plain). `tiles` → the
`.tiles` KPI row; a tile with a `gate:` key renders as the wide `.tile.gate` highlight.

### 5.2 Areas (Markdown → layout)

Each `## Area name` becomes a **`.card`**. Inside it, three fixed sub-lists:

```markdown
## Control Plane

### Done
- Session API + TTL reaper
- Track registry + cohort binding
- Token auth (per-session cookie)

### Missing
- xterm.js terminal `P2`

### Improve
- Push live readiness to the page `partial`
```

- `### Done` / `### Missing` / `### Improve` → the three `.col`s (green / amber / orange dots).
- A trailing `` `P0` ``/`` `P2` `` in an item → a `.tag`; `` `partial` ``/`` `uncommitted` `` →
  a `.badge`.
- **The per-area bar and the overall progress bar are auto-computed** from the item counts
  (`done / (done+missing+improve)`), so the author never hand-maintains a percentage — the bug
  that made the prototype drift. `%` and segment widths are derived at build time.

A `## Shipped` section with a bulleted list renders as the **`.chips`** row; a `` `hash` `` in
an item becomes the `.hash` commit pill.

### 5.3 Component / CSS inventory (from `status-board.html`)

`wrap top brand logo subtitle themebtn · meta + pill (+ ok/warn) · tiles + tile (num/lbl/sub) +
gate · overall + head + pct · bar + seg (done/missing/improve) + barlabel · section-h · areas +
card + cols + col + dot + tag + badge (part) · chips + chip + hash`. Palette: the shared triad
with `-bg` tints.

---

## 6. Integration into si

1. **`internal/studio/rules.go`** — extend the `type` rule value to
   `"deck, lab, article, guide, book, demo, site, page, architecture or status"`.
2. **`internal/studio/starters.go`** — add `architecture` and `status` to `StarterKinds` and a
   one-file starter each ("New… → Architecture / Status board").
3. **`docs/RULES.md`** — add **§21 Architecture documents** and **§22 Status boards**, in the
   same shape as §18 Lab / §19 Articles / §20 Guides (front matter, the section grammar table,
   one worked example).
4. **`examples/`** — add `examples/architecture/<name>.md` and `examples/status/<name>.md`
   (shipped with this proposal).
5. **Build path** — both route through the existing document (not deck) renderer, like
   `article`; they add only a type-specific section transform + the two CSS blocks.
6. **Studio** — list both under Documents; live preview as for articles.

## 7. Rendering notes & decisions

- **Reuse, don't reinvent:** sections, leads, tables, blockquotes, anchors, theme toggle, and
  self-contained inlining already exist for `article`. The new code is the *section grammar*
  (cards, phases, areas, tiles) + two scoped CSS blocks.
- **Auto-computed status bars** are the key correctness win over the hand-HTML prototype — the
  percentages can never drift from the lists.
- **Diagrams** should reuse si stage diagrams rather than raw inline SVG, so authors describe a
  flow in text and si draws it.
- **Graceful fallback:** an `architecture`/`status` file opened by an older si (unknown type)
  should degrade to `article` rendering, not error.

## 8. Open questions

1. `status` card columns — fixed **Done / Missing / Improve**, or author-defined column labels?
   (Proposal: fixed triad now; configurable later.)
2. Card tag syntax — the `` `tag` `` backtick convention vs a dedicated `::: cards` fence.
   (Proposal: backtick-list now; it needs no new parser.)
3. Should `tiles`/`meta` also be expressible in the body (a `## Overview` block) for authors who
   dislike long front matter?

## 9. Acceptance criteria

- `si build file.md` with `type: architecture` reproduces `architecture.html`'s layout from
  Markdown; with `type: status`, reproduces `status-board.html`, **with bars computed from the
  lists**.
- Both honour `--css` (CU brand drops in), light/dark, and build fully offline.
- `docs/RULES.md` §21/§22 and `examples/architecture`, `examples/status` ship with it.
- An unknown-type fallback renders as `article`.
