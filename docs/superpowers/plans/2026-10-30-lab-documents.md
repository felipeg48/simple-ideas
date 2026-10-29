# si · Plan 2b: Lab documents (`type: lab`) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A Markdown file with front matter `type: lab` is rendered by si as a readable, self-contained document (a lab guide), not as slides — in `si build`, `si play`, `si check` and Studio's preview — so course labs need no pandoc.

**Architecture:** `internal/deck` learns the `type` key and, for labs, keeps the whole body as one document (no `---` slide split, includes still expand). `internal/render` gains `Document(d, opt) Result` that renders the body with the same goldmark pipeline (highlighter, media modes, relative `.md` → `.html` links) into a document page with its own layout CSS (`web/doc.css`, unlayered mechanics + tokens from the deck's theme). Build, play, check and the index route labs to it; Studio's tree marks them `lab`.

**Tech Stack:** Go (stdlib + goldmark), CSS, a little vanilla JS (TOC highlight, copy buttons, theme toggle) — no new dependencies.

**Spec:** `docs/superpowers/specs/2026-10-28-si-studio-design.md` — section "Lab documents: `type: lab` (decided)".

## Global Constraints

- Front matter `type`: `deck` (default) or `lab`; anything else is a `si check` problem. Files without `type` behave exactly as before (golden test unchanged).
- A lab is **self-contained** like a deck (rule #2): CSS, JS, images inlined (or copied with `--no-inline-media`); no network resources (Mermaid follows the existing rule).
- **Fluid** (rule #1): readable measure (max ~78ch for prose), full-width tables and code that scroll horizontally inside their own container (`overflow-x: auto`), never clipped; sidebar TOC collapses above the content under ~900 px; works at 400×800, 820×1180, 1440×900, 3840×2160.
- Uses the deck theme tokens (`--si-*`) so `theme:` and `--theme`/`--css` apply; light/dark follows `mode:` and the existing `d` key; print stylesheet prints the whole guide.
- Callouts: GitHub alert syntax `> [!NOTE]`, `> [!TIP]`, `> [!IMPORTANT]`, `> [!WARNING]`, `> [!CAUTION]` render as styled callouts (title + icon); other blockquotes stay quotes.
- Headings `##`/`###` get stable ids (GitHub-style slugs) and anchor links; the TOC lists them.
- Code blocks: existing highlighter, language label, a copy button, horizontal scroll.
- Every new rule (the `type` key, callouts) documented in `docs/RULES.md` with an example and used in `examples/` (a small lab in `examples/`), per CLAUDE.md rules #3/#4 (content, not instructions).
- New `--si-*` tokens / `.si-*` classes go into `internal/assets/contract.go` (`TestContractCoversBuiltinCSS`).
- `go.mod` unchanged; gofmt, vet, `go test ./...` green; headless visual QA only. Plain commits, no attribution. No version bump in this plan (the controller bumps after merging with plan 2a).

## Review Focus

- A lab whose body contains `---` (horizontal rule) or fenced `---` in code: it stays one document, rendered as an `<hr>` / code. (Task 1.)
- A wide table or a 200-character command in a lab at 400 px width: scrolls inside its container, page body never scrolls horizontally. (Task 3 visual QA.)
- A lab that includes another file or links to `other-lab.md`: include expands, link becomes `other-lab.html`. (Task 2.)
- A deck (no `type`) after this change: byte-identical golden output. (Task 2.)
- `type: lab` with slide-only directives (`<!-- layout: -->`, `<!-- steps -->`): a `si check` warning saying they have no effect in a lab, never a crash. (Task 1.)

---

### Task 1: `type` front matter and lab parsing

**Files:** `internal/deck/deck.go`, `internal/deck/deck_test.go`.

**Interfaces — produces:** `Meta.Type string` (`"deck"` default when absent, `"lab"`); `"type"` in `MetaKeys`; `func (d *Deck) IsLab() bool`; for labs, `d.Slides` holds exactly one slide of `Kind` `KindDocument` (new kind `"document"`) whose `Body` is the whole expanded body (front matter removed, includes expanded, `---` kept as text), `Title` = front-matter `title` or the first `# H1`; slide directives found in a lab produce a warning problem `"<name>" has no effect in a lab document`.

- [ ] Failing table tests: type validation (`type: lab` ok, `type: book` problem), lab keeps `---` and fenced `---`, include expands inside a lab, title from front matter vs first H1, slide-directive warning, a deck without `type` unchanged.
- [ ] Run `go test ./internal/deck` → FAIL; implement; → PASS; `go test ./...` green.
- [ ] Commit `feat(deck): type: lab front matter`.

### Task 2: `render.Document` — the lab page

**Files:** Create `internal/render/document.go`, `internal/assets/web/doc.css`, `internal/assets/web/doc.js`; modify `internal/assets/assets.go` (embed), `internal/assets/contract.go` (new tokens/hooks), `internal/render/render.go` (dispatch: `Deck` calls `Document` when `d.IsLab()`, so every caller works unchanged), golden test data `internal/render/testdata/lab.md` + `lab.golden.html`, `render_test.go`.

**Interfaces — produces:** `func Document(d *deck.Deck, opt Options) Result` (same `Result` as `Deck`: HTML, Problems, Steps 0, Canvases count); page structure `<body class="si-doc">` → header (title, author/date/version), `<nav class="si-doc-toc">` (h2/h3), `<main class="si-doc-body">`, footer (copyright/footer text); callouts `<div class="si-callout" data-kind="tip">`; heading ids; copy buttons; reuses the markdown pipeline, media modes, `.md`→`.html` link rewriting, highlighter, Mermaid rule, theme resolution (`resolveTheme`), `--css` user layer, `LiveReload`/`Dev` hooks for play.

- [ ] Failing tests: `TestLabGolden` (fixture with headings, a wide table, a long code line, each callout kind, an image, a link to `other.md`, an include) with `-update` support; `TestDeckGoldenUnchanged` is the existing golden test (must still pass untouched); `TestLabSelfContained` (no `http` resources except Mermaid when used); `TestLabHeadingIDs` (slugs, duplicates get `-2`); `TestContractCoversBuiltinCSS` covers doc.css.
- [ ] Implement; `go test ./...` green; read the golden diff.
- [ ] Commit `feat(render): lab documents`.

### Task 3: build, play, check, index, Studio tree; examples; docs; visual QA

**Files:** `cmd/si/main.go` (build summary line shows labs as `lab` with section count instead of slides/minutes), `internal/serve/serve.go` (nothing if Task 2's dispatch suffices — verify), `internal/render/render.go` `Entry` (index card: `lab` badge, "N sections" instead of slides/time), `internal/studio/workspace.go` (Tree kind `"lab"` for `type: lab` files that are decks by the current rule), `web/studio/src/navigator.tsx` (lab icon) + bundle rebuild, `examples/` (a short lab that belongs to the showcase talk, content not instructions), `docs/RULES.md` (new section "Lab documents" with example; callouts), `FEATURE.md`, `docs/studio.md` (preview of labs), `CLAUDE.md` layout table (doc.css/doc.js).

- [ ] Failing tests: `TestBuildWritesLab` (si build on a dir with a deck + a lab → both HTML files; index lists the lab with the `lab` badge), `TestTreeKindLab` (Studio tree), `TestCheckLab` (missing image in a lab → problem with file:line).
- [ ] Implement; `go test ./...`, `make studio-ui`, `make studio-e2e` green.
- [ ] Visual QA headless: the example lab and the CACI-shaped fixture at 1440×900, 820×1180, 400×800 (page body must not scroll horizontally), 3840×2160, light and dark, and print-to-PDF; read every screenshot and fix.
- [ ] Commit `feat: lab documents in build, check, index and Studio`.
