---
type: article
title: "Proposal: `type: architecture` and `type: status` document types"
description: Two new si document types that turn one Markdown file into a sectioned architecture/design page or a project status board — the layouts prototyped in cockroach-education-platform/docs/{architecture,status-board}.html, made native to si.
---

# Proposal: `architecture` and `status` document types

Status: **implemented in si v3.43.0** (RULES.md §35 Architecture pages, §36 Status boards) ·
Author-facing feature · Implements two layouts already prototyped as hand-written HTML (see
**Reference artifacts** below). Where the shipped syntax differs from this proposal, §10
**Decisions** says how and why; RULES.md is the reference.

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

Self-contained reference layouts ship **with this proposal** under `docs/proposals/reference/`
— open them to see the exact target rendering, tokens and CSS:

- `reference/architecture.html` → `type: architecture`, on the **CockroachDB / CU theme**.
- `reference/status-board.html` → `type: status`.
- `reference/cu-theme.css` → the **CockroachDB / Cockroach University brand** as an si `--css`
  theme (Data Purple `#9059FF` + taupe + Geist). The architecture reference already uses it;
  it re-skins either type by token override alone.

Both already share si-compatible chrome: a `.wrap` container, a `header.top`
(logo · title · `subtitle` · **theme toggle** cycling auto/light/dark), a self-contained
`<style>`, and a footer. That chrome is identical to what an `article` page needs — reuse it.

## 3. Shared foundations

Both types inherit the document scaffolding si already has for `article`:

- **Front matter:** `title` (required), `subtitle` (the grey line under the title),
  `theme:` / `--css`, standard build flags. Self-contained HTML, inlined CSS, no network.
- **Theme:** the existing auto/light/dark toggle and token set. Neutral defaults ship in the
  renderer; the **intended brand skin is the CockroachDB / Cockroach University theme**
  (Data Purple `--accent #9059FF`, taupe surfaces `#F9F8F3` / `#F2F1EE`, `#101913` ink, Geist),
  provided as `reference/cu-theme.css` and applied with `--css` (or `theme:`) — exactly how si
  themes any document. The status triad is **`--done` · `--missing` · `--improve`** (each with a
  `-bg` tint). Because every component reads tokens, re-skinning is a token override, no
  structural CSS.
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
| a fenced **` ```flow `** block (reuse si [stage diagrams](../stage-diagrams.md)) | an **SVG flow diagram** with a `.legend` of swatches. Nodes/edges use the diagram hues (`--c-client`, `--c-edge`, `--c-ctrl`, `--c-data`, `--c-node`). |
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
  (% done = `done / (done+missing)`, as the reference board computes it; the bar's segments are
  each column over every item), so the author never hand-maintains a percentage — the bug
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

## 10. Decisions (implementation, v3.43.0)

The open questions and the gaps found while building it, decided:

1. **Columns:** fixed **Done / Missing / Improve** (`### Done` …, any case). An unknown `###`
   under an area is a `si check` problem naming the three. Configurable labels can come later.
2. **Cards:** the backtick-list convention, no new fence. A list becomes cards only when
   **every** item reads `` `tag` **Title** — body `` (tag optional; `—`, `–`, `--`, `-` or `:`
   after the title); a list where only some items do stays a list and `si check` warns.
3. **`tiles` / `meta` in the body:** no. They stay in the front matter, which now accepts `- …`
   items under `meta:` and `tiles:` (only there, so a deck that starts with `---` and a list is
   unchanged). Text before the first `##` is an introduction instead.
4. **% done** = done ÷ (done + missing), rounded, as `status-board.html` computes it (15 done,
   17 missing → 47%); Improve does not count against done; nothing missing is 100%. The bar's
   three segments are done, missing and improve over every item, rounded to two decimals.
5. **Tile numbers from the board:** a tile's `num`, `lbl` and `sub` may say `{done}`,
   `{missing}`, `{improve}`, `{open}` (missing + improve), `{pct}`, `{items}`, `{areas}`, so the
   "Still open" tile cannot drift either. `tone: ok|warn` colours a tile's number (a gate tile is
   amber unless `tone: ok`).
6. **Item marks:** trailing code spans only. `P0`–`P9` a priority tag; `partial`,
   `in progress`, `wip` an in-progress badge; `gate`, `gated`, `blocked`, `uncommitted` a gated
   badge; `improvement` the Improvement badge (a shipped Improve item, as the founder's board
   marks it); anything else a mono version · hash. A code span inside the text stays code.
7. **`## Shipped`** (any title starting with Shipped) is the chips row; every other `##` without
   columns is a note rendered as Markdown and not counted (a "Next ideas" list).
8. **Area layout:** one card per row with a header strip, its bar, then the three columns (the
   founder's own board); the columns stack when the card is narrower than 38rem (a container
   query), so phones, portrait and 4K all work.
9. **`flow`:** an architecture page's ```` ```flow ```` is the stage fence under another name,
   with the stage syntax (`A[Label] -->|label| B`, `dot A..B`). The proposal's
   `client "Browser" -> edge …` syntax and the per-role hues (`--c-client` …) are not built:
   stage boxes take the theme tokens. The example uses the stage syntax.
10. **Phases:** a `##` whose title starts with "Phase" and has an ordered list; items are
    labelled Phase 1, 2 … by CSS; `(now)` is removed from the text and highlights the card.
11. **Tokens:** `--si-done`, `--si-missing`, `--si-improve` (defaults: the theme's `--si-ok`,
    `--si-warn` and a mix of `--si-warn` and `--si-crit`), so every built-in theme and a `--css`
    file (the CU brand) re-skin both types; looks in `@layer si.theme`, grids unlayered
    (`web/doc.css`). The `.logo` square of the reference is not built: the page uses si's
    document header (title, subtitle, search, light/dark).
12. **Contents column:** neither type has one; numbered sections and area cards are the outline.
13. **Unknown types:** an unknown `type:` stays a `si check` problem (a typo must be caught) and
    now renders as an article instead of a deck. Older si versions cannot be changed; they report
    `type: architecture` as unknown.
14. **Exports:** `si build --pdf` prints both (cards kept whole, colours printed). `--md`,
    `--docx`, `--rtf`, `--epub` accept an architecture page (as an article); a status board is
    refused with a message, since its numbers are computed, not written.
