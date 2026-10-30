# HANDOFF — for the next agent

Read this, then `FEATURE.md`, then `CLAUDE.md`. This is a Go CLI (`si`) that turns
Markdown/AsciiDoc into animated, self-contained HTML slide decks.

## Current state (2026-10-30)

- **Latest release: v0.9.1** (tagged, published by the release workflow; founder's
  `~/.bin/si` is 0.9.1 via `si selfupdate`). `main` == `origin/main`, tests green.
  CI + release workflows are green. v0.8.0 was never tagged (its Studio work ships in v0.9.x).
- **v0.9.1:** built-in **`theme: acemq`** (AceMQ brand: Urbanist/Mona Sans + logo
  embedded from `internal/assets/web/themes/acemq/`, extra CSS in
  `internal/assets/web/themes/acemq.css`, emitted only for that theme by `ThemeCSS`;
  component classes `.hl .cards .card .tag .tl .steps .callout on-ink us-N` documented
  in RULES.md §15 "acemq theme extras"; `--si-brand-logo` hides/swaps the logo);
  **`<!-- eyebrow: … -->`** slide directive (`.si-kicker` above the heading, not in
  the title); CSS-file themes inline relative `url()`s (`inlineURLs` in
  `internal/render/css.go`); stage labels shrink to fit (`labelFit`); `--si-figure-max`
  token and `fit()` shrinks figures before type; lab docs: key/value tables wrap,
  byline `·` split, wide tables/code use the column, Previous/Next pager between labs
  built from one folder (`.si-doc-pager`); `make install` does rm+cp (macOS kills
  in-place binary overwrites — that is what killed `si selfupdate` once).
- **v0.9.0:** Studio runbook generator + visual JSON editor + `{{name}}` /
  `{{name:default}}` placeholders (Go-template words `end else break continue nil
  true false` are excluded; `{{.X}}` never matches); **lab documents** (`type: lab`,
  `render.Document`, `doc.css`/`doc.js`, RULES.md §18; `si build` lists labs with a
  LAB badge); Studio `si --web` (v0.8.0 work: Preact + CodeMirror bundle committed in
  `internal/assets/web/studio/assets/`, rebuild with `make studio-ui`, e2e with
  `make studio-e2e`). Spec: `docs/superpowers/specs/2026-10-28-si-studio-design.md`;
  plans 1/2a/2b in `docs/superpowers/plans/` are done and merged.
- **Next up (Studio, toward v1.0.0):** Studio lab preview full width + cursor
  scroll to section (deferred from plan 2b); plan 2c (inspector, Storyboard, ⌘K,
  Build/PDF jobs); plan 3 (Rehearse/Present/Lab modes); plan 4 macOS Wails app;
  plan 5 Windows/Linux + ConPTY. Order agreed: web + macOS first. Open source
  (Apache 2.0), no code signing.
- **Open polish from v0.9.1:** acemq extras ≈148 KB per deck (an SVG logo would
  shrink it); generic class names (`.card`, `.tag`, `.y/.n`) may collide with user
  CSS; lab pager follows file names (`overview.md` sorts after `lab-08`); no way to
  exclude a top-level `.md` from `si build` except `_` prefix / front matter.
- Older: v0.7.x runbook drawer (`runbook:` JSON → `si play` terminal drawer,
  RULES.md §14, `docs/terminal.md`); v0.6.5 Points/teleprompter tabs; v0.6.3
  `--dev` hot reload + lab slides; v0.6.2 footer/watermark; v0.6.1 topic overview;
  v0.6.0 terminal drawer (play + loopback only).

## Non-negotiable rules (from CLAUDE.md — do not break without the founder)

1. **Fluid & responsive, never fixed.** No fixed canvas; `cqi`/`clamp()`; SVGs by
   `viewBox`; columns stack on narrow/portrait. `TestLayoutHasNoFixedCanvas` guards it.
2. **Self-contained output.** One deck = one HTML file; only allowed network
   resource is pinned Mermaid. (v0.6.0 terminal is a documented play-only
   exception — a built deck never loads it.)
3. **Every directive documented in RULES.md with an example, used in `examples/`.**
4. **Example decks carry content, not instructions.**
5. **Themes change tokens/looks, not layout.** Every `--si-*` token + `.si-*`
   class is listed in `internal/assets/contract.go` — `TestContractCoversBuiltinCSS`
   fails otherwise. Add new tokens/hooks there when you add CSS.
6. **Teleprompter stays a real teleprompter.**
7. **Go stdlib + goldmark only.** `gofmt`, `go vet ./...`, `go test ./...` green
   before every commit. (v0.6.0 carves a play-only, loopback-only exception for
   the terminal — `creack/pty` + bundled xterm.js; see CLAUDE.md rule #7.)

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

## Founder decks outside this repo (built with si 0.9.1)

- **CACI RabbitMQ course** `~/Downloads/CACI/` (5 sessions + ref decks A/B, labs
  0–8, handouts, instructor demos). `theme: acemq`, footer `CACI/AceMQ 2026`,
  eyebrows + highlighter titles; labs/handouts are `type: lab`. Builds:
  `si build decks --output slides --site "CACI Training" --pdf`,
  `si build labs --output labs/html --site "CACI Training · Labs"`,
  `si build handouts --output handouts --site "CACI Training · Session outputs"`.
  Runbook source of truth: `labs/runbook.md` → `python3 labs/runbook-to-json.py`
  → `labs/runbook.json` (founder does not use Studio for CACI). Backups:
  `_backup-2026-10-29-pre-v0.9/`, `_backup-2026-10-30-pre-acemq-theme/`.
- **PagoNxt pitch** `~/Downloads/PagoNxtAceMQBroker/` (original, untouched) and
  `~/Downloads/PagoNxtAceMQBrokerV2/` (approved restyle per `STYLE-GUIDE.md`,
  `theme: acemq`; "Trestle" never on slides → "AceMQ Enterprise Distribution for
  RabbitMQ" / "AceMQ Distribution"). Build:
  `si build md/ --output html/ --site "AceMQ for PagoNxt" --pdf && cp html/pdfs/pagonxt-acemq.pdf PagoNxt-AceMQ-V2.pdf`.
  Open question to the founder: the "Free trial today" slide vs the guide's
  no-free-offer rule.

## Known minor issues / polish backlog

- **`data-count` unbounded tween (found 2026-10-27, course migration).** The SVG
  counter helper tweens proportionally to the value delta, so a large range
  (e.g. `10,000,000 → 1`) never settles in a reasonable time — the number keeps
  animating. Workaround authors used: per-state `data-show` numbers instead of
  `data-count`. Fix candidate: cap the tween duration (ease by ratio/steps, not
  by absolute delta) so any range settles in a bounded time. Lives in the
  counter logic in `web/deck.js`. Low severity (workaroundable), real defect.
- Very wide full-bleed canvases scroll horizontally at the 400px phone floor
  (matches the existing house decks — not a regression; candidate for a
  canvas-responsiveness pass).
- `rules.go` → moved to `docs/docs.go` already (embedder lives with the docs).
- Roadmap ideas are in CLAUDE.md "Roadmap ideas".
