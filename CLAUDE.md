# simple-ideas: project instructions

`si` is a Go CLI that turns markdown into animated, fluid HTML slide decks:
`si play` renders on every request with live reload, `si build` writes
self-contained HTML. It grew out of a Python deck builder whose best idea was
one SVG canvas whose parts change state slide by slide; this project
generalises that idea and keeps the quality bar of those decks.

Full autonomy: do not ask for confirmation; pick sensible defaults and say
what you chose.

## Non-negotiable rules

1. **Fluid and responsive, never fixed.** No fixed slide canvas (no
   1280×720 scaled by `transform`). Slides fill the viewport and reflow; type
   uses container units (`cqi`, `cqb`) and `clamp()`; columns stack on narrow
   or portrait slides; SVGs scale by `viewBox`; image sizes are `%`, `em` or
   `cqi`, never `px`. Check every visual change at 1440×900, 400×800 (phone),
   820×1180 (portrait) and 3840×2160. `TestLayoutHasNoFixedCanvas` guards the
   obvious regressions.
2. **Self-contained output.** One deck = one HTML file: CSS, JS, SVGs, images,
   video inlined (or copied with `--no-inline-media`; `--share-assets` likewise
   writes three.js once into `_si/` beside the pages, opt-in only). The only network
   resource allowed is the pinned Mermaid script, and only for decks with a
   Mermaid fence. (three.js for 3D mind maps is inlined, not fetched: see the
   v1.13.0 exception under #8.)
3. **Every rule is documented in RULES.md with an example, and used in
   `examples/`.** A directive that is not in RULES.md does not exist.
4. **Example decks carry content, never instructions.** No slide says "press
   the arrow" or "this slide shows the two-col layout"; the showcase is a real
   talk (an order's journey) that happens to use every rule.
5. **`Notes:` is the teleprompter, `Points:` the speaker notes.** `Notes:` is
   what the speaker says to the public, verbatim, about what is on the slide:
   spoken sentences, never instructions to the speaker ("pause here", "mention
   X", "tell them", "press next", "this slide shows", "on a phone it stacks").
   `Points:` is the main points of the topic as short bullets, plus reminders
   for the speaker (pace, emphasis, what to skip, likely questions). The one
   allowed non-spoken mark in `Notes:` is the **`[CLICK]` cue** (recommended):
   placed where the speaker advances a build step, an animation or the slide,
   one per step (plus at most one for the next slide); the teleprompter shows
   it as a ▶ CLICK badge and `si check` warns when the count does not fit. Applies to
   every deck we write or ship (examples, scaffolds, samples, migrations) and is
   in the agent guide `docs/agent-rules.md`, exported by `si --export
   agent-rules` as CLAUDE.md + AGENTS.md; keep the two in step.
6. **Themes change tokens and looks, never layout mechanics.** Layout rules
   live unlayered in `layout.css`; looks live in `@layer si.theme`; user CSS
   in `@layer si.user`. Every `--si-*` property and `.si-*` class used by the
   built-in CSS is listed in `internal/assets/contract.go`
   (`TestContractCoversBuiltinCSS` fails otherwise), so
   `si --export css-rules` cannot drift.
7. **The teleprompter stays a real teleprompter:** large notes, auto-scroll
   with adjustable speed, manual scroll, mirror, per-slide and total timers
   with pace, previews, synced over BroadcastChannel, works from `file://`.
8. Go stdlib plus goldmark only. `gofmt`, `go vet ./...` and `go test ./...`
   green before every commit. Plain commit messages, no attribution trailers.
   **Exception (v0.6.0, `si play` terminal only):** the presenter terminal
   drawer adds `github.com/creack/pty` and bundles `xterm.js`. It is contained to
   `si play` on a loopback address — a built deck never loads it — so rules #2 and
   #8 hold everywhere except that one play-only, loopback-only subsystem. See
   `docs/terminal.md`. Web demos (`si demo test|run`, v2.36.0) reuse the same
   pty and xterm.js for their terminal scenes, on a keyed 127.0.0.1 page that
   only that run's headless Chrome opens; still never in a built deck.
   **Exception (v1.13.0, 3D mind maps only):** three.js r128 (MIT) and its
   OrbitControls are vendored in `internal/assets/web/` (`three.min.js`,
   `three-orbitcontrols.js`, `THREE-LICENSE`) and inlined, with
   `mindmap3d.js`, only into pages that have a map starting in 3D
   (`view: 3d`, `view=3d`): about 620 KB, no network, no Go dependency.
   2D-only pages never carry it.
   **Exception (v0.8.0, `si --web` Studio only):** the editor front-end vendors
   Preact and CodeMirror 6, bundled by esbuild and committed under
   `internal/assets/web/studio/assets/`; Node is needed only for `make studio-ui` /
   `make studio-e2e`, never for go build/test. The Studio assets are served only
   by `si --web` (loopback only), never in a built deck, so rule #2 holds too. The
   Go module adds no dependencies. See `docs/studio.md`.
   **Exception (v1.7.0, the terminal front end only):** `internal/tui` uses
   the Charm libraries (`charm.land/bubbletea/v2`, `charm.land/lipgloss/v2`, and
   their `x/ansi`) for `si play --terminal` and, later, `si --terminal`. A built
   deck never touches them; the HTML renderer stays goldmark-only.
   **Exception (v1.6.0, desktop shell only):** `desktop/` is its own Go module
   (`desktop/go.mod`) with cgo and Objective-C against Apple's Cocoa and WebKit
   frameworks (no third-party code). It bundles `si` and runs `si --web` per
   folder; the si module itself stays cgo-free. See `docs/desktop.md`.

## Layout of the code

| Path | What |
|---|---|
| `cmd/si/main.go` | CLI: GNU-style flag parsing (interspersed, long flags), commands play, build, check, new, rules, `--export`; `cmd/si/mindmap.go`: mind map exports (`--svg`, `--png`, `--pdf` through headless Chrome and the runtime's `#si-export`) and `si mindmap import/export`; `cmd/si/code.go`: `si code` (code window exports: PNG/SVG per step, an animated GIF of a show, frames from the runtime's `#si-code-export`); `cmd/si/video.go`: `--mp4` (frames from headless Chrome, a 30 fps image sequence encoded by ffmpeg if installed; mind map Play frames from `#si-export-play`); `cmd/si/publish.go`: `si publish` (dry run unless `--yes`), `login` / `logout`; `cmd/si/qrcmd.go`: `si qr` and `printQR` (terminal codes) |
| `internal/publish/` | `si publish` clients, stdlib only: LinkedIn (Documents + Posts API, 3-legged OAuth with a loopback redirect, `/v2/userinfo`), dev.to (Forem API), Hashnode (GraphQL); one request builder for dry run and send (secrets masked); logins in the config dir, 0600; tests against `httptest` fakes only; see `docs/publish.md` |
| `internal/pinned/` | code and documents from elsewhere, stdlib only: `gist:ID@REV`, `github:owner/repo@REF/path` (a file, a folder of scripts, or the whole tree via the tarball API), `https://… · sha256 HEX`; pinned only, cached by pin in `~/.si/cache/{scripts,repos}`, asked once per pin (`~/.si/trusted`, `--trust` for CI, refused without a terminal); used by `si demo` and `si play/serve/build` (`cmd/si/fetched.go`, `cmd/si/stage.go`) |
| `internal/docsite/` | `si --export docs-site`: the docs site source, one page per numbered RULES.md section (fence-aware split, `§N` and contents links rewritten; every ```markdown example gets an AsciiDoc tab, `adoc.go` + `deck.MarkdownToAdoc`), the manual as a deck, the examples si ships as live pages (`live/`), and `features.go`: one feature page per main kind of document (decks, articles, guides, mind maps, books, code) with its live example, the Markdown read from `examples/`, options, commands and rules links; `docs/site/` holds the hand-written pages (`_site.md` with the Features and Examples menus, home, examples, download); `make docs-site` builds it into `dist/docs-site` with `--share-assets` |
| `internal/skill/` | the si Agent Skill: `SKILL.md` (when to use si, the check-and-look loop, `docs/agent-rules.md`) and `references/` (one file per RULES.md section); `si --export skill`; the Claude Code plugin under `plugin/` is its output (`go test ./internal/skill -update`; `TestPluginInSync`), listed by `.claude-plugin/marketplace.json` (the `simple-ideas` marketplace) |
| `internal/mcp/` | a small MCP server (stdio, newline-delimited JSON-RPC, standard library): `server/discover` and `_meta` versions (2026-07-28), `initialize` for earlier clients, tools/list, tools/call; si's tools are in `cmd/si/mcp.go` (`si mcp [dir]`) |
| `internal/qr/` | the QR encoder (byte mode, versions 1-40, levels L/M/Q/H, `Cover`/`Hole` for a picture in the middle) and its renderers (ANSI, SVG, PNG/JPEG via `Image`, PDF); `testdata/golden.txt` pins 64 codes a real reader decoded (`QR_VISION=1 go test ./internal/qr` on macOS) |
| `internal/codeshow/` | code windows in Markdown text, by lines: `Shows` / `ShowAt` (fences with `window=` or `id=`, same-id runs, their `<!-- callout: -->` lines), `Apply` (the Window view's ops), `Source` (an article of only the windows, for exports and the Studio preview) |
| `internal/mindmap/` | the mind map Markdown model for editors and converters: `Parse` / `String` (round trip, unknown text kept), `Apply` (the map editor's ops), `FenceAt` / `SetInfo` (fences), `Import` / `Export` (OPML, FreeMind, XMind, Mermaid). The renderer keeps its own reader for drawing |
| `internal/deck/` | parse a deck: front matter, includes (cycle check), `---` split, headings, `Notes: (N min)`, slide directives; `media.go` validates background and image options; `guide.go` parses a guide's `_nav.md` (`ParseWith` gives pages the guide's settings). No HTML. |
| `internal/render/` | deck → HTML. `markdown.go`: goldmark + AST transform for body directives (steps, columns, svg, image figures) and fenced code; `svg.go`: canvas inlining, id prefixing, state resolution, helper-attribute CSS; `media.go`: data URIs / link / copy modes, backgrounds, figures; `highlight.go`: tiny highlighter; `css.go`: user CSS layer; `render.go`: deck and slides; `page.go`: the page, head and the play scripts; `theme.go`: themes and timer animations; `index.go`: the index page; `chrome.go`: watermark, footer, lab slides; `document.go`: lab / article / guide page; `guide.go`: guide sidebar, breadcrumb, pager, partials, `RenderGuide` + link/anchor checks; `site.go`: site navigators and page frame; `plugins.go`: alert, badge, event, accordion, modal, carousel; `embed.go`: `slides:` and `embed:`; `mindmap.go`: mind maps (`type: mindmap` pages and the ```` ```mindmap ```` fence: list → JSON maps, linked map files inlined); `codewin.go`: code windows (window= / id= fences grouped into shows, callouts, step markers on slides) |
| `internal/serve/` | `si play` / `si serve`: on-the-fly render (decks, articles, guide pages with their `_nav.md`), static files, SSE `/_si/events`, mtime-polling watcher (decks, assets, `--css` paths) |
| `internal/studio/` | `si --web`: gateway (token, Host, Origin), confined workspace, file/deck/check/rules API, assembly with the `si play` server; `mindmap.go`: the map editor's `POST mindmap` (apply an op to the editor's text, return text + maps) and `GET mmcanvas` (the canvas page, `web/mindmap-edit.js`); `codewin.go`: the Window view's `POST codewin` (an op on the editor's text, back text + show + preview page) and `POST codewin/export` (`si code`); `git.go`: the History panel's `git/*` calls |
| `web/studio/` | Studio front-end source (Preact + TS + CodeMirror 6, esbuild, vitest, Playwright e2e); bundle committed to `internal/assets/web/studio/assets/` via `make studio-ui` |
| `internal/config/` | optional `si.conf` (flat `key: value`, stdlib): default theme, `keys.*` command remaps, `feature.*` on/off; injected into the deck as `window.SI_CONFIG`, read by `deck.js` |
| `internal/assets/` | embedded web files (`web/layout.css` unlayered mechanics, `web/theme.css` looks, `web/deck/*.js` runtime (one file per feature, joined in name order into one script), `web/doc.css` + `web/doc.js` lab documents (`type: lab`), `web/presenter.html` teleprompter, `web/mindmap.js` + `web/mindmap.css` mind maps: layout, lines, drill-down, search, Play; `web/codewin.js` + `web/codewin.css` code windows: chrome, backdrops, callouts, the morph between steps, `#si-code-export`; `web/mindmap3d.js` the 3D view on vendored `web/three.min.js` + `web/three-orbitcontrols.js`, inlined only with a 3D map) and `contract.go` (tokens, hooks, theme values, starter CSS) |
| `docs/docs.go` | embeds RULES.md + reference/user-manual.md + agent-rules.md for `si rules` / `si manual` / `si --export agent-rules`, and `docs/site/` for the docs site (in `docs/` so go:embed can reach them; no `.go` at the repo root) |
| `examples/` | showcase deck, included appendix, `assets/journey.svg` canvas, media, `css/harbor.css` theme, one example per document type (`article/`, `guide/`, `book/`, `demos/`, `site/`); embedded for `si new` and `--export example --type` |
| `docs/screenshots/readme/` | headless screenshots referenced by the README and docs/terminal.md |
| `internal/tui/` | si in the terminal: `slide.go` + `text.go` (the deck as text and colour: layouts, builds, code, tables, cards), `font.go` (block letters), `image.go` (half-block pixels), `palette.go` (theme tokens), `play.go` (`si play --terminal`, `--dump`), `notes.go` (the `--notes` teleprompter), `sync.go` (the local socket between them), `studio.go` (`si --terminal [dir]`: files, preview, problems, overlays, modes), `lab.go` (Lab mode), `pictures.go` (Kitty / iTerm2 pictures); see `docs/terminal-studio.md` |
| `desktop/` | si Studio.app for macOS: its own module; `main.go` (starts/stops `si --web` per folder; `helpJS`: the Help menu and About run through `window.siStudio`), `app_darwin.m` (Cocoa + WebKit windows and menus), `build-mac.sh` (universal bundle + zip, `make desktop-mac`; signs with the self-signed "simple-ideas Self-Signed" identity when present, `make desktop-cert` creates it), `icon/gen.go` (the Stack app icon, `make desktop-icon`) |
| `internal/cdp/` | the Chrome DevTools Protocol client over `--remote-debugging-pipe` (stdlib, with events): used by `--mp4` motion filming and web demos |
| `internal/webdemo/` | web demos (RULES §27): the `type: demo` + `app:` script (Markdown, AsciiDoc, JSON), the runner over `internal/cdp` (locators, steps, expects), report.html/json and JUnit; `film.go` + `overlay.js` (the take: cursor, notes, blur, screencast), `encode.go` (ffmpeg, captions, chapters, SRT), `voice.go` (engines, cache, `pronounce:`), `readme.go`; `cmd/si/demo.go` is `si demo test|run`; spec `docs/superpowers/specs/2026-11-10-web-demos.md` |
| `internal/motion/` | animated SVG canvases as frames for the Studio's Motion view (`internal/studio/motion.go`, `web/studio/src/motion-editor.tsx`): reads/writes the `/* si motion: … */` block, keeps hand-written SVG and CSS; `transition: morph` itself lives in `deck/02-navigation.js` |
| `internal/themefile/` | theme CSS files: the `/* si theme: Name · extends <built-in> */` header, token read/write keeping other CSS, checks (values, px, contrast), W3C Design Tokens export/import; `cmd/si/themecmd.go` (`si theme check|export|import`), `cmd/si/mcptheme.go` (MCP theme tools), `internal/studio/theme.go` + `web/studio/src/theme-editor.tsx` (the Studio Theme view) |
| `internal/home/` | `~/.si` (`$SI_HOME`): si's own folder for the user si.conf, `publish.json`, `ca/`, `sessions/`, `kokoro/`, `cache/`; `Path` moves a pre-v2.43.0 copy from the system config/cache folders on first use |
| `internal/brand/` | what si says about itself, in one place: the docs site URL (`Site`), the repo, Help pages, `IssueURL` (Report a Problem / Request a Feature, issue forms in `.github/ISSUE_TEMPLATE/`), credits, build info; read by the CLI (`si --version --verbose`, `si report`), the Studio (`#si-about` JSON) and the docs site generator |
| `docs/brand/` | the Stack logo set (mark, mono, favicon, app tiles, wordmark, social card); embedded as `docs.Brand` for the docs site |
| `cmd/si/motion.go` | real-motion `--mp4`: headless Chrome over the DevTools pipe (stdlib), page slowed (Animation domain + a JS clock shim), screenshots stamped and resampled to 30 fps |

## Run and test

```bash
make test                         # go vet + go test ./...
make build && ./bin/si check examples
make play-example                 # live server on the examples
go test ./internal/render -update # rewrite the golden file after an intended markup change; review the diff
```

Visual checks must not open windows on the founder's screen: use headless
Chrome (`"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
--headless=new --screenshot=out.png --window-size=1440,900 file://...#5.2`)
or a headless Playwright script (dark mode via `colorScheme`, the
teleprompter via the popup page). Never the Playwright MCP browser tools.

## How the canvas animation works

1. `<!-- svg: path words -->` inlines the SVG with `class="si-canvas"`,
   `data-state` and `data-base`. `+w`/`-w` words resolve against the state the
   previous slide left that file in (`dr.lastState`, after its steps).
2. Steps carry `data-step` and `data-ops`; the runtime computes
   `base + ops(1..k)` and sets `data-state`.
3. Moving between consecutive slides with the same file: the runtime sets
   the incoming SVG to the outgoing state, flushes styles, then sets the new
   state, so every CSS transition in the SVG runs; the deck switches to
   `data-si-tx="canvas"` (no slide fade, only the words fade in).
4. Helper attributes (`data-show`, `data-hide`, `data-alert`, `data-ok`,
   `data-flow`, `data-packets`) become per-deck generated CSS rules
   (`helperCSS`), hide rules last so they win.

## Adding a directive end to end

1. `internal/deck/deck.go`: add it to `Directives` with scope `slide` or
   `body`; validate its value in `slideDirective` (slide directives also set
   a `Slide` field and are consumed).
2. Slide directive: use the field in `render.slide` (attribute, class or
   style). Body directive: handle it in `transform` in
   `internal/render/markdown.go` (it applies to the next top-level block).
3. CSS: mechanics in `web/layout.css`, looks in `web/theme.css`; any new
   `--si-*` token or `.si-*` class goes into `Tokens` / `Hooks` in
   `internal/assets/contract.go`.
4. Runtime behaviour: `web/deck/` (the part for that feature) (and `web/presenter.html` if the
   teleprompter must know).
5. Tests: a table case in `internal/deck/deck_test.go`, a use in
   `internal/render/testdata/deck.md` (then `-update` the golden file and read
   the diff), an assertion in `render_test.go` if it emits CSS.
6. RULES.md: a section or table row with an example; use it in
   `examples/showcase.md` (as content, not instructions); `si check
   examples`; screenshot it headless and look at it.

## Migrating an existing deck into si

Turning an existing deck (AsciiDoc/reveal.js, PowerPoint, a pile of slides)
into an si deck is not a mechanical format conversion — it is an editorial
pass. The steps that matter:

1. **Analyse the content first.** Read the whole source, understand the
   narrative, and decide what each slide is really for.
2. **Remove duplicate and unnecessary slides.** Source decks are full of
   near-duplicates ("... Pt. 1/Pt. 2"), restated headings, filler, and
   course cross-sell. Merge or cut them. Preserve every substantive teaching
   point, SQL snippet and setting name — prune redundancy, not material.
3. **Replace static images with native visuals.** Imported raster/SVG images
   do not follow the theme or dark mode. Prefer, in order: a stronger layout
   (`two-col`, `big-number`, `quote`), an inline `<!-- svg: -->` canvas or
   diagram coloured with `--si-*` tokens, or a highlighted code block. Reserve
   step-animated canvases (section 10 of RULES.md) for the 1–2 concepts per
   section that genuinely benefit; everything else can be a static themed SVG
   or a layout. No slide should be a wall of text.
4. **Verify visually.** `si check`, then screenshot headless and look — an
   agent-authored SVG that passes `check` can still be visually wrong.
5. **Ship it as a split deck, not one monolith.** The standard structure is a
   thin main deck (`<deck>.md`: front matter, title, agenda, one
   `<!-- include: sections/NN-*.md -->` per section, closing) plus one file per
   section under `sections/`, with `images/anim/` (animated canvases) and
   `images/diagrams/` (static diagrams). `si build <dir>` builds only the main
   deck — subdirectories and non-`.md` files are skipped. Section files
   reference shared assets as `../images/...` (a module's paths are relative to
   itself). See RULES.md §12.

## Roadmap ideas

- **si 3** (spec `docs/superpowers/specs/2026-11-12-si-3.md`): items 1–4
  DONE, unreleased (transitions, default fade 0.4s; camera `zoom to`/`pan
  to`/`zoom out`; `spotlight`/`focus`; `window`, `drag`; title-card casing;
  `internal/webdemo/effects.go`, `scriptfx.go`, `runfx.go`). Next: item 5
  (re-film `docs/demos/` with them, "What's new in si 3"), then v3.0.0.
- **Demo lifecycle, sources from anywhere** (spec
  `docs/superpowers/specs/2026-11-12-demo-lifecycle.md`, all three phases
  built, unreleased): `start:`/`prepare:`/`step:` in order, `setup:` per
  feature, `stop:`, `cleanup:` (always run), `## Before`/`## After`, folder
  runs with one report; `internal/pinned` (gist:, github:, pinned URLs, trust
  in `~/.si/trusted`, `--trust`, caches in `~/.si/cache`); `si
  play|serve|build github:…`; a deck's own `setup:`/`start:`/`stop:`/`cleanup:`
  run by `si play`/`si serve` (`cmd/si/stage.go`, getting-ready page).
- **v3.0.0: web demos** (spec `docs/superpowers/specs/2026-11-10-web-demos.md`,
  decided with the founder 2026-11-10; phase 1 test mode DONE v2.34.0, phases
  2 video + 3 voice DONE v2.35.0: `si demo run`, `internal/webdemo/film.go`,
  `encode.go`, `voice.go`, `overlay.js`): one `type: demo` script with
  `app: <url>` makes narrated videos of a web app (per feature and one with
  chapters, callouts, captions, voice engines local and cloud) and runs as a
  test (`si demo test`, report, JUnit). Phases: test mode, video, voice, the
  manual with videos, then Retake's CLAUDE.md gets the shared script format.
  **Phase 4, si's own videos** (DONE v2.39.0): six scripts in `docs/demos/`
  (decks, presenting, studio, themes, mindmaps, terminal); CI `demos` job runs
  `si demo test` on them every push; `.github/workflows/demos.yml` films them
  on macOS with Kokoro on minor tags (`v*.*.0`) and on demand, attaches
  `si-demo-videos.tar.gz` + `si-demo-<area>.mp4` to the release and starts
  docs-site; `pages.yml` unpacks the newest release's tarball into
  `$SI_DEMO_VIDEOS`; `internal/docsite/videos.go` turns `<!-- si-video: area -->`
  lines into the video + chapters (nothing without videos); `make docs-site`
  copies `videos/`. README "Watch it" posters in `docs/screenshots/readme/demo-*.jpg`.
  Next: a window step (the teleprompter is a popup) and a drag step (drawing).
  **Three videos per take** (DONE v2.38.0): `.video.mp4`, `.mp4`,
  `.captions.mp4` (picture shrunk above a captions band, never covered), `.vtt`
  beside `.srt`, long waits fast-forwarded with a badge (`fast-forward` in
  `encode.go`; the voice never sped up).
  **One file by default** (DONE v2.41.0, `internal/webdemo/outputs.go`):
  `si demo run` writes `<script>.mp4` (was `all.mp4`) alone; `--with
  captions,silent,audio,subtitles,transcript,chapters,clips,report,readme,all`
  (or `with:`, si.conf `demo.with:`) for the rest, made in a temp folder and
  moved out by `deliver`; `--no-captions` is gone.
  **`si voice`** (DONE v2.37.0, `cmd/si/voice.go`, `internal/tools/kokoro.go`):
  `install kokoro` (venv + kokoro-onnx 0.6.1 + SHA-256-pinned models in the
  user cache), `say` (plays like macOS say, falls back to system), `list`,
  `login|logout` (keys only in the user si.conf `<config dir>/si/si.conf`,
  0600; a project si.conf key is ignored with a warning; `config.APIKey`);
  `voice:` / `voice.style:` in si.conf. Next: a Windows run of install.
  **With a terminal** (spec `2026-11-10-web-demos-terminal.md`, DONE v2.36.0
  except `split`): `start:`/`stop:`/`shell:`, `terminal`/`run`/`expect
  output`/`browser`, `start … as NAME`/`stop NAME`; `internal/webdemo/
  scriptterm.go`, `procs.go`, `output.go`, `term.go` (PTY shell + keyed page,
  xterm.js from `assets.TermFS`; the creack/pty exception now also covers web
  demos: never in a built deck), `runterm.go`; `app: none` for a
  terminal-only demo. Next: `split`, ConPTY.
- **On hold until a user asks for it** (the founder, 2026-11-10; do not build
  before a real request):
  - **PowerPoint / Google Slides export.** Google Slides has no file format of
    its own; it imports `.pptx`, which PowerPoint and Keynote open too. Shape
    if asked: (1) `si build talk.md --pptx`: each slide a full-bleed picture of
    its last build step, `Notes:` as the speaker notes (stdlib `archive/zip` +
    OOXML; looks exact, text not editable); (2) editable text boxes for titles,
    bullets, tables and code, pictures for canvases (hard: fluid layouts vs
    fixed boxes); (3) `si publish --to google-slides` (Drive upload with
    conversion, a Google OAuth login like LinkedIn's).
- **Courses** (phases 1–2 DONE v3.2.0, spec
  `docs/superpowers/specs/2026-11-12-courses.md`, mockup
  `2026-11-12-courses-design.html`): `_course.md` (modules, lessons, kinds
  by link target, `· kind=`, minutes, prerequisites), learner / `--instructor`
  builds, `nav:` layouts (courses, guides, books, sites), course home,
  website bundle (`--zip`, videos copied, `--no-film`), `type: quiz` + the
  quiz fence, progress in localStorage, `complete.html` (`certificate: on`),
  `<!-- solution -->`; `internal/deck/course.go`, `quiz.go`,
  `internal/render/course.go`, `quiz.go`, `web/course.css`, `web/course.js`;
  RULES.md §28. Next: phase 3 (`--scorm` SCORM 1.2, `--pdf` workbook and
  instructor guide), phase 4 (`si publish --to skilljar`).
- **Auto-play** (DONE v2.40.0, spec `docs/superpowers/specs/2026-11-10-autoplay.md`):
  `v` / ▶ Auto-play reads `Notes:` aloud and advances at each `[CLICK]`
  (`deck.Parts`, `web/deck/05-autoplay.js`); `si play` voices parts at
  `/_si/voice/` (`internal/serve/voice.go`, `webdemo.ForDeck`); `si build
  --autoplay[=inline]` writes `<deck>.voice/`; `--mp4 --voice` narrates video;
  `voices:` per speaker. Next: resume mid-sentence.
- **Next five** (specs, 2026-11-09; the founder wants all, one at a time):
  `docs/superpowers/specs/2026-11-09-next-five.md` indexes audience mode
  (follow, polls, Q&A), handouts (`--handout`), record your talk
  (DONE v2.28.0: ● Record, `--narration`, `si rehearse`), theme from a logo (DONE v2.31.0, `internal/themefile/logo.go`), two presenters (DONE v2.29.0)
  (`Notes[Name]:`), and the Studio Theme editor (DONE v2.30.0; `2026-11-09-studio-theme-editor.md`;
  phase 1 fixes CSS-file themes: `extends <built-in>`, the index page's theme;
  theme import/export incl. W3C design tokens; MCP theme tools), and Motion
  (DONE v2.33.0; `2026-11-09-motion-editor.md`: `transition: morph`, the Studio Motion view). All five specs are built.
- **Audience mode** (phase 1 DONE, v2.16.0): `si play --audience`: a second
  key, `audienceHandler` (deck with `NoSpeakerText`, `audienceView`-filtered
  stream, no sends), runtime role `audience`, `<!-- qr: audience -->`.
  v2.17.0: polls (`render/poll.go`, `serve/poll.go`, `/_si/audience/vote`,
  `poll` relay messages). v2.18.0: Q&A (`serve/qa.go`, `/_si/audience/ask`,
  `/_si/qa`, teleprompter Q&A tab, `.si-qa-show`, results saved on exit).
  v2.24.0: word clouds (`render/poll.go` cloudHTML, `serve/cloud.go`,
  `/_si/audience/word`, presenter `/_si/cloud` hide, Q&A tab words).
  v2.20.0: poll `· hidden` (a `.si-poll-reveal` build step; CSS hides
  bars and counts until it is visible) and `audience.save: off`.
  v2.19.0: `--tunnel` (Cloudflare quick tunnel to `AudienceOnlyHandler`;
  polling fallback `/_si/audience/now` since quick tunnels carry no SSE). The
  founder ruled out a hosted relay on simple-ideas.org for now (cost).
- **Unlisted themes** (v2.19.1): `crdb`, `acemq`, `acemq-pro` (client work)
  stay in code but never in docs, site or help (`docsite.unlisted`, the
  theme-shots skip list; `TRADEMARKS.md`). Do not document them again.
- **Themes** (v2.15.0): six new built-ins (slate, twilight, forest, sunrise,
  neon, newsprint; `TestNewThemesContrast`: WCAG AA in light and dark); the
  docs site's Themes page is generated from `assets.Themes`
  (`internal/docsite/themes.go`) with a picture per theme in
  `docs/site/themes/<name>.jpg` (`make theme-shots`, `web/studio/e2e/theme-shots.mjs`;
  `TestThemesPageHasEveryTheme` fails when one is missing); the README shows
  the same pictures. Add a theme → `make theme-shots`.
- **Handouts** (DONE, v2.14.0): `si build --handout[=slides|lines|points|notes]`
  prints the deck page with `?si-handout=<mode>#print` (06-media.js sets
  `data-si-handout`, light mode and a portrait `@page` after the deck's own;
  print CSS in layout.css); `handout:` / `handout-url:` front matter,
  `handout.paper` in si.conf. No speaker text by default (the founder).
- **Menu-bar icon** (DONE, v2.12.0): si Studio.app's NSStatusItem lists
  running sessions from `<config dir>/si/sessions/<pid>.json`
  (`internal/serve/session.go`, written by si play / serve; `desktop/sessions.go`
  reads them, drops dead pids); `GET /_si/qr` (loopback) shows the phone codes.
  Only while the app is open (no login item), by the founder's choice.
- **Draw with the cursor** (DONE, v2.11.0): in draw mode, Option (Alt) held
  = the cursor draws (`deck/05-annotate-keys.js`; pointer e2e). An absolute
  trackpad mapping was tried and dropped (the pen jumped away from the
  cursor); `--trackpad` is gone entirely (v2.29.1). Do not bring it back.
- **AI agents** (DONE, v2.8.0): the `si` skill, the Claude Code plugin
  (`/plugin install si@simple-ideas`), `si --export skill`, docs page
  `agents.md`. v2.9.0: `si mcp` (`internal/mcp`: stdlib JSON-RPC over stdio,
  MCP 2026-07-28 stateless + `initialize` for older clients; tools in
  `cmd/si/mcp.go`, confined to the served folder; in the plugin's `.mcp.json`).
  Next: list the skill and the server in the public directories.
- **QR codes** (DONE, v2.6.0): `internal/qr`, `si qr` (PNG/JPG/SVG/PDF, logo,
  dots, Wi-Fi, vCard), `<!-- qr: -->`, the `q` overlay in `si play --remote`.
  v2.7.0: `si play --remote --https` (si's own name-constrained CA,
  `internal/serve/tls.go`, one port for HTTPS + the plain-HTTP setup page) and
  `--https=tailscale`. `--https` for `--review` is in (v2.23.0; `httpsFor`/`serveSplit` in main.go).
- **Brand, Help, About, Report a Problem** (DONE, v2.5.0): the Stack logo,
  `internal/brand` as the one place for links and credits, GitHub issue forms,
  `si report`; real-motion `--mp4`; review pins on blocks; 3D map editing.
  Next: Developer ID signing + notarisation (needs the founder's Apple account),
  the desktop app on Windows and Linux.
- **`si play --remote`** (DONE, see `docs/remote.md`): a keyed second
  listener for a tablet: `tp` (teleprompter) and `draw` (pen pad) roles, synced
  through a server relay (SSE + POST); the transport is injected only in play.
  The phone `clicker` role is in (v1.11.0); v2.5.1: a phone layout for `tp`,
  reconnect after sleep, a wake lock where the browser allows it. Next: HTTPS
  or a pairing code for untrusted networks (HTTPS would also give phones the
  wake lock); a native iPhone clicker.
- **`si play` terminal drawer** (DONE, v0.6.0, see `docs/terminal.md`): a
  bottom→middle HUD with multiple tabs to run local commands during demos
  (`docker ps`, `cockroach start-single-node`, `cockroach sql`). Shipped as full
  xterm.js + PTY (`creack/pty`), play-only + loopback-only. Transport: chunked-GET
  output + POST input (no WebSocket dep). Toggle key: `` ` ``. Next: persist tab
  layout across reloads; per-tab running indicator.
- A `stage` fence that generates animated flow SVGs (Mermaid-like) is in
  (`docs/stage-diagrams.md`); next: step-driven reveal beyond highlight
  (show/hide, ok/alert per node), single-level fan-out polish.
- **Web fonts** (DONE, v1.1.0, `render/fonts.go`): `fonts:` from Google, Bunny,
  Fontshare, Fontsource or a local file, inlined at build time. Next: a
  `fonts-subset:` key for non-latin scripts; fonts in a theme `.css` file.
- **Screen demo recorder: moved to its own project, Retake
  (`~/Progs/AI/retake-app`), not part of si:** scripted demos of native Mac apps
  recorded with narration into MP4s, and the same scripts as screen-level
  tests; a native Swift app. Plan, hard parts, phases and the founder's open
  questions: `docs/superpowers/specs/2026-11-06-screen-demo-recorder.md`.
- **Code windows** (v2.1.0 DONE, RULES.md §25, spec
  `docs/superpowers/specs/2026-11-06-code-windows.md`): `window=mac|windows|terminal|none`,
  `title=`, `theme=`, `backdrop=`, `numbers`, `hl=`, `id=` shows with `step=`
  names and `transition=morph|fade|slide`; `<!-- callout: LINE text at=dx,dy -->`;
  documents get ← → Play, slides get one build step per extra step and a
  slide-to-slide morph (deck.js canvasKey `cw:<id>`); mind map code topics take
  `window=`; `si code --png --svg --gif`; the Studio's Window view (look, steps,
  callouts, drag to place, exports). v2.1.1: fence-option completion, a step's
  code edited inside the Window view, callout drags saved by si play
  (POST /_si/callout), the desktop app's newer-si choice and release check, 3D
  maps on phones (closer Fit, List tool). v2.2.0: a show's steps are builds in
  the terminal; `--share-assets`; real-pointer tests (web/studio/e2e/pointer.e2e.mjs).
  MP4 export is in (v2.3.0).
- **Mind maps** (v1.12.0 DONE, RULES.md §24, spec
  `docs/superpowers/specs/2026-11-06-mind-maps.md`): `type: mindmap` pages and
  the ```` ```mindmap ```` fence; four layouts, four line styles, three node
  styles, five map themes, drill-down between inlined map files, folding,
  moving topics, `/` search, Play, terminal trees, Studio "New mind map".
  **v2.0.0 DONE:** the Studio map editor (Map view for pages and fences: select,
  Tab/Enter, rename in place, drag to reparent or reorder, Shift/Alt-drag writes
  `pos=`/`branch-pos=`, Tidy, topic and page panels, undo through the editor;
  ops applied by `internal/mindmap`), `si build --svg/--png/--pdf` for mind
  maps, `si mindmap import/export` (OPML, FreeMind, XMind, Mermaid), fence
  `steps` (one branch per click, 2D and 3D), `flow: on`, toy labels facing the
  camera, a portrait 3D fit. Terminal Studio map editing has indent/outdent/undo (v2.27.0). Next: multi-select and copy/paste of
  branches, XMind 8 export.
  **3D** (v1.13.0 DONE): `view: 3d` / `view=3d`, `camera: iso|perspective`,
  `floors: on|off`; pills on a shadow floor, cables with flow dots, linked maps
  as glass floors, fly-in drill-down, cards, `/` search, Play, 2D/3D and camera
  toggles; mockup `docs/superpowers/specs/2026-11-06-mind-maps-3d.html`.
  **Kinds** (v1.14.0 DONE): `kind: blocks|bricks|toy` / `kind=`, a Kind button
  in the 3D tools (never call bricks by a toy brand name). Next: a
  phone layout for 3D (portrait maps are small), the visual builder in 3D,
  exports of the 3D view.
- **Publishing (`si publish`, DONE v2.3.0, see `docs/publish.md`):** LinkedIn
  document posts (a deck's PDF) and link posts, `--copy` rich text for the
  article editor, dev.to and Hashnode articles; always a dry run unless `--yes`.
  A link post's picture (`cover:`, Images API) is in (v2.3.1).
  Ghost and WordPress are in (v2.25.0, `internal/publish/blogs.go`). Next: re-check the `Linkedin-Version` header each year (LinkedIn sunsets versions).
- **Video (DONE v2.3.0/v2.3.1):** `--mp4` for code shows, a mind map's Play
  (3D too) and decks (every slide and step, dissolves, `--dwell`). Real motion
  (`cmd/si/motion.go`) is in (v2.5.0).
- **Docs site (DONE v2.3.0):** `make docs-site`; `.github/workflows/pages.yml`
  deploys it once the owner turns Pages on and sets `SI_PAGES=on`.
- **Studio first run (DONE v2.3.0):** start gallery + tour; the desktop Help
  menu opens them (v2.3.1).
- **v2.4.0 (DONE):** Studio History (git, `internal/studio/git.go`), review
  links (`--review`, `internal/serve/review.go`, comments in `.si/review/`),
  `si check --budget`, map editor multi-select + copy/cut/paste, terminal
  Studio editing (`internal/tui/edit.go`). Paragraph pins and 3D map editing
  are in (v2.5.0). History branches and pull/push are in (v2.26.0).
- More themes; `theme:` from a remote URL (the CSS-file path and the
  `si --export themes` gallery already exist).
- SVG helpers: `data-move` along a path, counters that tick between states,
  `data-grow` bars. (The `si check` SVG lint — `<` in `<style>`, missing
  viewBox — is done.)
- Speaker view extras: notes from a separate file. (Rehearsal timings per
  slide and pace against the last run are in.)
- Replace the remaining Python deck builder workflows (fully Python-free).
- `si play` QR code for opening the deck on a phone on the same network. (Done:
  `si play --addr 0.0.0.0` prints a LAN URL + QR.)
- AI-assisted migration: `si migrate <src>` prints an editorial-migration prompt
  (optionally piped to an agent with `--ai`). Next: a tighter agent loop.
