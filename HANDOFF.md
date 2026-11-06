# HANDOFF — for the next agent

Read this, then `FEATURE.md`, then `CLAUDE.md`. This is a Go CLI (`si`) that turns
Markdown/AsciiDoc into animated, self-contained HTML slide decks.

## Current state (2026-11-05)

- **Latest release: v1.6.3.** `<!-- cards -->` (`render/cards.go`, a pass in
  `codeFeatures` so slides and documents share it; card class `.si-cell` because
  `.si-card` is the index tile), `si fix [--write]` (`deck.FixComponents`), `si new
  <name> --type …` (`studio.WriteStarter`, shared with the Studio), Studio titles drop
  `==` (`plainTitle`), editor colours `==highlight==` (`HL_RE`), `## Title {#id}` in
  documents, `fonts-subset:`, desktop File ▸ Open Recent (own list, NSMenuDelegate),
  CACI: cards converted, the two [CLICK] warnings fixed.
- **v1.6.2: components without raw HTML.** `==text==` highlight
  (`render/highlight_mark.go`, goldmark delimiter like ~~strike~~; `plain()` drops the
  marks); `badge` and new `logo` work on slides (CACI pill / lockup markup, with
  `si-*` hooks; generic looks under `:where()` so a theme's `.pill`/`.lockup`/`.hl` wins);
  a `· part` that is no kind stays text. AceMQ Pro sample and CACI converted (CACI
  backup `_backup-2026-11-06-pre-components/`).
- **v1.6.1** (fix: the deck teleprompter `p` opened window.open(''), which the shell sent to NSWorkspace → "The application can't be opened. -50"; only http(s) pages of other sites leave the app now)
- **v1.6.0 release:** (tagged; the release workflow publishes it, plus
  `si-Studio-<v>-macos.zip` from a macOS runner).
- **v1.6.0: si Studio for macOS + storyboard thumbnails.** `desktop/` (own module,
  cgo + Cocoa/WebKit, no Wails): one WKWebView window per folder, each running the
  bundled `si --web --addr 127.0.0.1:0`; window.open → app windows; external links →
  browser; ad-hoc signed universal bundle (`make desktop-mac`); `docs/desktop.md`.
  The window UI was not visually tested (no windows on the founder's screen);
  `desktop/main_test.go` tests start/stop headless. Storyboard thumbnails:
  deck.js `?si-view=thumb` (role `thumb`: no channel, all steps), live reload skips
  it. Next: Windows/Linux desktop, `si publish` (LinkedIn).
- **v1.5.0: Studio S4.** Inspector (`inspector.tsx`, `inspector-edit.ts`
  `setDirective`, `EditorCtl.edit` one undoable minimal change; check API slides
  carry layout/transition/background/class/eyebrow/words/clicks/steps), storyboard
  (`storyboard.tsx`), focus view, Write view switch, narrow layout (< 900 px: nav
  drawer, stacked panes), preview re-syncs the slide after every load. Bundle left as
  is (measured; see docs/studio.md). Next: the desktop shell (Wails), slide
  thumbnails in the storyboard, `si publish` (LinkedIn).
- **v1.4.0: Studio S3 (modes + ⌘K).** `modes.tsx` (Rehearse, Present, Lab on the
  previewed deck), `palette.tsx` (⌘K, `fuzzy`). deck.js: `?si-view=tp` makes the page
  the teleprompter of the same deck elsewhere in the browser (role `tp-local`, no
  BroadcastChannel of its own), `window.SI.run(cmd)` (Lab opens the terminal with
  it). Next (S4): inspector, storyboard, focus mode, a smaller bundle, narrow layouts;
  then the desktop shell.
- **v1.3.0: Studio S2 (writing).** Completion for every directive/key with help,
  values and workspace paths (`rules.go` help/values/paths; `si-lang.ts`
  `valueOptions`, `pathOptions`, `setCompletionFiles`). **New** dialog
  (`new-dialog.tsx`) + `POST starter` (`starters.go`, `starters/`): eight
  starters, never overwrite, auto-link into the enclosing `_nav.md`/`_site.md`.
  `GET checkall` (one pass like `si check <dir>`) replaces one check per deck on
  refresh; `POST build` runs the CLI in-process (`studio.Options.Run`) into
  `html/` (`build-dialog.tsx`). Next: S3 modes (Rehearse, Present, Lab) and ⌘K.
- **v1.2.0: Studio S1 (documents in `si --web`).** `internal/studio/kinds.go`:
  `guides` cache, `parsePage` (guide defaults like serve), `pageKind`, `guideKind`.
  Tree nodes carry kinds (article, guide, book, demo, site, page, nav) and `guide`
  on guide folders; `GET decks` adds `kind`, `url`, `guide` plus one entry per
  `_nav.md`/`_site.md`; `GET check` renders a guide/book/site whole
  (`RenderGuide`) and returns `kind`. Front-end: icons, badges, kind labels,
  Open vs Play, slide-follow only for decks. Fixed in `si serve`/`si play`: an
  `index.md` beats the generated folder index at `/` and `/index.html`.
  Next (S2): completion for every directive and key, New… starters per type,
  Build panel, workspace check; S3 modes (Rehearse, Present, Lab) and ⌘K.
- **v1.1.0: web fonts.** `fonts:` (front matter, `_nav.md`/`_site.md` via
  `Guide.Defaults` with file paths made absolute, `si.conf`): roles heading / body /
  mono, sources google (default), bunny, fontshare, fontsource, or a local font file
  (`render/fonts.go`). Stylesheets and files go through `deck.FetchAsset` (cached for
  good, `--offline`), latin + latin-ext subsets only, woff2 preferred, inlined as data
  URIs in `@layer si.theme` after the theme. The example site uses a local Fraunces
  (OFL, `examples/site/fonts/`). RULES §15 "Fonts".
- **v1.0.0: sites.** A folder with `_site.md` (`deck.SiteFile`, `Guide.IsSite()`):
  pages default to `type: page`, navigators `topbar` / `topbar-centred` / `mega` /
  `sidebar` with dropdowns and a phone menu (`render/site.go`), folder links to
  sub-guides/books/sites (`NavItem.Sub`, built and checked on their own). Plugins in
  every document (`render/plugins.go` + doc.js/doc.css): `alert` (dismiss remembered,
  `until=`), `badge`, `event` (HUD toast countdown, 48 h window), `accordion`,
  `modal` … `end`, `carousel`, `embed` (card / `#section` / `full` / deck slides /
  folder card, `render/embed.go`). `--site-title` (old `--site <text>` warns; `--site`
  is free for site features). `examples/site/` (Northwind Engineering),
  `--export example --type site`, RULES §23. Fixed: a folder link in a nav list
  stopped parsing the rest of the list. Next: v1.1.0 Google Fonts (inlined at build
  time), then publishing integrations (Medium).
- **v0.9.10: search (`/`) in every document**: `render/search.go` writes a `#si-search`
  JSON block (a guide/book page: every other page, its sections with a 220-char snippet,
  a book's index terms, filled by `fillGuideSearch` after RenderGuide; a lone page: the
  build's page list); doc.js searches the page DOM first; Search button in the header /
  guide bar; `si serve` renders guides whole (like books) so live pages get the data.
- **v0.9.9:** `watermark:` / `watermark-opacity:` in every document type (fixed
  `.si-doc-wm` layer, inherited from `_nav.md`, printed on every page); `[CLICK]`
  cues in `Notes:` (teleprompter ▶ CLICK badge `.si-cue`; `si check` warns when the
  count is < steps or > steps + 1) — CLAUDE.md rule 5, RULES §11, agent guide.
- **v0.9.8: demo scripts** (phase 4): `type: demo` (`session`, `slides`, `duration`
  facts, INSTRUCTOR ONLY badge, `> Say:` speech callouts, `text` blocks labelled
  expected output, `### … (N min)` steps checked against `duration`), `si build
  --no-instructor`, the ▶ Demos panel strips front matter, `examples/demos/order-demo.md`
  (the showcase lists it), `--export example --type demo`, RULES §22. Deferred from
  phase 4: search across a guide, side-by-side diffs, versioned guides.
- **v0.9.7: books** (`type: book` in `_nav.md`; `render/book.go`): `Guide.BookNumbers`
  (front matter / chapters / lettered appendices), `## ` sections numbered N.k,
  captioned figures (image title) and `Table:` captions numbered and given ids,
  empty-text links filled from a whole-book label map (`bookLinks`), `((term))` /
  `(((term)))` index (`indexTerms`), generated `index.html` (cover), `contents.html`,
  `book-index.html`, `book.html` (one file, ids prefixed `c-<page>-`), `book.pdf`
  with `--pdf`; `si serve` renders the whole book per request; RULES §21;
  `--export example --type book|reference`. Mockups: "si Documents", "si Books".
- **v0.9.6: documents phase 3**: `<!-- code: GitHub permalink | path= ref= repo= -->`
  and `<!-- diff: commit/compare URL | repo= ref=a..b path= -->` (`internal/deck/remote.go`:
  GitHub API with GITHUB_TOKEN / GH_TOKEN / `gh auth token`, else raw URLs; local clones via
  `git show` / `git diff`; disk cache in os.UserCacheDir()/si/fetch, commits forever, refs 1 h,
  stale copy on network errors; `--offline`; moving-ref warning), diff rendering with old/new
  line numbers (`diffHTML`), `repo:` front matter, `<!-- slides: deck.md 2-3 | #id -->` in
  documents (iframe srcdoc, `render/embed.go`; deck.js guards replaceState for about:srcdoc).
- **v0.9.5: documents phase 2** (code, slides and documents alike): code-file
  includes `<!-- include: f lines=5-10 | tag=x [hl= title= lang= linenos indent=keep] -->`
  (`internal/deck/codeinc.go`, become fenced blocks at parse time; same-line runs
  stay one document part), fence attributes `title= hl= linenos start=`,
  `<N>` callouts + `<N> text` list, `<!-- tabs -->` groups (synced by label and
  remembered in documents) — `internal/render/code.go`; AsciiDoc `include::f[lines=..|tag=..]`,
  `.Title`, `[source,lang,highlight=..]`, `%linenums`. Tokens `--si-code-hl`,
  `--si-tab-active`. RULES §13 "Code from files, highlighted lines, callouts and tabs".
- **v0.9.4: documents phase 1** (spec `docs/superpowers/specs/2026-11-05-documents-design.md`,
  design mockups approved by the founder): `type: article` (lede from `description:`,
  reading time, contents on the right), **guides** = a folder with `_nav.md`
  (`deck.LoadGuide`, pages via `deck.ParseWith` inherit the nav's front matter;
  sidebar with read marks, breadcrumb, Prev/Next in nav order, `edit-url`,
  `_header.md` / `_footer.md` partials, subfolders, `.adoc` pages;
  `render.RenderGuide` checks links + anchors between pages), `## Step N:` badges
  in labs and guides, document theme tokens (`--si-font-prose`, `--si-prose-size`,
  `--si-prose-leading`, `--si-doc-measure`, `--si-lede-size`, `--si-nav-bg`,
  `--si-nav-active`, `--si-step-badge`; unset by default, CSS falls back), RULES
  §15 "Themes for documents", §19 Articles, §20 Guides; **`si serve`** (play
  without the terminal, guide-aware); **`si --export example --type
  deck|slides|article|guide|tutorial|lab|all`** (examples in `examples/article/`,
  `examples/guide/`, embedded with `all:guide`). Next: phase 2 (code includes
  with lines/tag, tabs, callouts, hl), phase 3 (GitHub permalinks, diffs), phase 4
  (`type: demo`, CACI demos last).
- **v0.9.3:** find (`/`) also searches the other pages of the folder, fuzzily
  (`idx` → index.html): `render.Pages` (index, decks, labs, PDFs, other `.html`
  in the output dir, si.conf `links.<label>: <url>`) → `SI_CONFIG.pages`;
  `si build` bakes it in (works from `file://`), `si play` reads it per request.
  Terminal drawer: runbook entries get **Copy** (prompt + clipboard, not run) and
  **Run**; `sendToShell` waits for the prompt (output quiet 250 ms) and uses
  bracketed paste when the shell enabled it (fixes "clicked command not visible").
  New front matter **`demos: ../demos/`**: markdown demo scripts in a `▶ Demos`
  panel (drop-down + the whole script, Copy/Run on shell blocks),
  `internal/render/demos.go` → `window.SI_DEMOS`, play + loopback only.
- **v0.9.2:**
  - **`theme: acemq-pro`** ("AceMQ Pro"): the acemq tokens + `acemq.css` +
    `internal/assets/web/themes/acemq-pro.css`, the full AceMQ deck style-guide
    catalogue (`.heard`, `.inc`+`.verdict`, `.split2`/`.blist`/`.topo`, `.zones`,
    `.opts`, `table.tbl`, `.facts`, `.feat`, `.bench`/`.bks`, `.flowline`/`.mnotes`,
    `.poc`/`.term2`, `.floaters`, `.icon-chip`, `.btn`, motion `.r .f .pop .pulse
    .grow-x`). `Theme.Extras` chains extra stylesheets; `acemq-pro.sample.md` is
    its gallery deck (`si --export themes`). RULES.md §15.
  - **`si theme list`**; theme names match loosely (`CanonicalTheme`: case,
    spaces/underscores → `-`, aliases), so `--theme "AceMQ Pro"` works; an
    unknown `--theme` is a usage error.
  - **`si play --remote[=tp,draw]`** (`--remote-addr`, default 0.0.0.0:play+1):
    keyed LAN listener (`?k=` → HttpOnly cookie) for a tablet: `tp` teleprompter
    and `draw` pen pad, synced by a relay (`internal/serve/remote.go`, SSE + POST,
    last state replayed). Transport `render.RelayJS` is injected only under
    `--remote`; built decks contain none of it. `docs/remote.md`, RULES.md §11.
  - **`si --export agent-rules`**: writes `CLAUDE.md` + `AGENTS.md` from
    `docs/agent-rules.md` (never overwrites; `-o -` prints).
  - **Teleprompter rule** (CLAUDE.md rule #5): `Notes:` = what the speaker says
    to the audience, verbatim, no stage directions; `Points:` = main points +
    speaker reminders. Examples, scaffolds, samples and the migrate prompt follow it.
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
   the terminal — `creack/pty` + bundled xterm.js; see CLAUDE.md rule #8.)

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
