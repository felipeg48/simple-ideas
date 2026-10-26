# HANDOFF — for the next agent

Read this, then `FEATURE.md`, then `CLAUDE.md`. This is a Go CLI (`si`) that turns
Markdown/AsciiDoc into animated, self-contained HTML slide decks.

## Current state

- **Latest release: v0.5.4.** `git describe --tags` on `main`. CI + release
  workflows are green; every `v*` tag cross-compiles 6 binaries + `SHA256SUMS`.
- Working tree should be clean on `main`. Tests/vet/fmt all green.
- **Next up: v0.6.0 — the `si play` terminal drawer.** Decision is locked (full
  xterm.js + PTY, multi-tab, play+loopback only). Full brief in
  `docs/terminal.md`; summarised in `FEATURE.md` → "Planned — v0.6.0".

## Non-negotiable rules (from CLAUDE.md — do not break without the founder)

1. **Fluid & responsive, never fixed.** No fixed canvas; `cqi`/`clamp()`; SVGs by
   `viewBox`; columns stack on narrow/portrait. `TestLayoutHasNoFixedCanvas` guards it.
2. **Self-contained output.** One deck = one HTML file; only allowed network
   resource is pinned Mermaid. (v0.6.0 terminal is a documented play-only exception.)
3. **Every directive documented in RULES.md with an example, used in `examples/`.**
4. **Example decks carry content, not instructions.**
5. **Themes change tokens/looks, not layout.** Every `--si-*` token + `.si-*`
   class is listed in `internal/assets/contract.go` — `TestContractCoversBuiltinCSS`
   fails otherwise. Add new tokens/hooks there when you add CSS.
6. **Teleprompter stays a real teleprompter.**
7. **Go stdlib + goldmark only.** `gofmt`, `go vet ./...`, `go test ./...` green
   before every commit. (v0.6.0 will carve out a play-only exception for the
   terminal — xterm.js + a PTY dep.)

## How to work

- `make test` = gofmt check + vet + test. `make build` → `bin/si`.
  `make play-example`. **Run `go test ./...` and check its real exit code — never
  pipe it through grep (that masks the exit code and has shipped a red test before).**
- **Release:** bump `const version` in `cmd/si/main.go`, commit, `git tag -a vX.Y.Z`,
  push the tag — the workflow builds + publishes. README badges are dynamic.
- **Visual QA is mandatory for any visual change** and must be **headless only**
  (never open a window on the founder's screen): headless Chrome
  (`"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" --headless=new
  --screenshot=out.png --window-size=1440,900 --virtual-time-budget=2000 file://…`)
  or a headless Playwright script. **Never the Playwright MCP browser tools.**
  For deck JS, `node --check internal/assets/web/deck.js` catches syntax errors.
- **Commits:** plain messages, **no AI/Claude attribution** of any kind.
- **Autonomy:** the founder wants full autonomy — pick sensible defaults, say what
  you chose, don't ask for confirmation on routine work. DO surface decisions that
  break a non-negotiable rule (that's the founder's call), e.g. the terminal.
- Adding a directive end-to-end: see CLAUDE.md "Adding a directive".

## The courses (separate from this repo)

- Course **sources** live at `~/Desktop/simple-ideas-course/markdown/` (one thin
  `<course>.md` + `<course>/sections/*.md` + `<course>/images/{anim,diagrams}`),
  with a shared `si.conf` and `anim/` (globe/ranges/roaches — the CockroachDB-brand
  timer animations that do NOT live in this repo).
- **Built site** ("CockroachDB University") is written alongside, into
  `~/Desktop/simple-ideas-course/`. Central build:
  `si build ~/Desktop/simple-ideas-course/markdown/ --site "CockroachDB University" --pdf -o ~/Desktop/simple-ideas-course`
- ILT source material (for migrations) is at
  `~/Roach/Courses/cockroach-university-ilt-courses/courses/` — **only** look there.
- **Migration recipe** (proven): one general-purpose agent per course, authoring
  ONLY under its own `markdown/<course>` subtree; it reads `docs/RULES.md` +
  `docs/authoring-canvases.md` + `docs/stage-diagrams.md`, matches the house style
  of an existing migrated course, replaces every raster with native SVG/stage/code,
  adds MORE animations than the source, runs `si check` on its own deck, and does
  NOT touch `index.html`/other courses/the repo. The lead does the central build.

## Known minor issues / polish backlog

- Very wide full-bleed canvases scroll horizontally at the 400px phone floor
  (matches the existing house decks — not a regression; candidate for a
  canvas-responsiveness pass).
- `rules.go` → moved to `docs/docs.go` already (embedder lives with the docs).
- Roadmap ideas are in CLAUDE.md "Roadmap ideas".
