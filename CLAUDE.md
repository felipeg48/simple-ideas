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
   `docs/terminal.md`.
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
| `cmd/si/main.go` | CLI: GNU-style flag parsing (interspersed, long flags), commands play, build, check, new, rules, `--export`; `cmd/si/mindmap.go`: mind map exports (`--svg`, `--png`, `--pdf` through headless Chrome and the runtime's `#si-export`) and `si mindmap import/export`; `cmd/si/code.go`: `si code` (code window exports: PNG/SVG per step, an animated GIF of a show, frames from the runtime's `#si-code-export`) |
| `internal/codeshow/` | code windows in Markdown text, by lines: `Shows` / `ShowAt` (fences with `window=` or `id=`, same-id runs, their `<!-- callout: -->` lines), `Apply` (the Window view's ops), `Source` (an article of only the windows, for exports and the Studio preview) |
| `internal/mindmap/` | the mind map Markdown model for editors and converters: `Parse` / `String` (round trip, unknown text kept), `Apply` (the map editor's ops), `FenceAt` / `SetInfo` (fences), `Import` / `Export` (OPML, FreeMind, XMind, Mermaid). The renderer keeps its own reader for drawing |
| `internal/deck/` | parse a deck: front matter, includes (cycle check), `---` split, headings, `Notes: (N min)`, slide directives; `media.go` validates background and image options; `guide.go` parses a guide's `_nav.md` (`ParseWith` gives pages the guide's settings). No HTML. |
| `internal/render/` | deck → HTML. `markdown.go`: goldmark + AST transform for body directives (steps, columns, svg, image figures) and fenced code; `svg.go`: canvas inlining, id prefixing, state resolution, helper-attribute CSS; `media.go`: data URIs / link / copy modes, backgrounds, figures; `highlight.go`: tiny highlighter; `css.go`: user CSS layer; `render.go`: deck and slides; `page.go`: the page, head and the play scripts; `theme.go`: themes and timer animations; `index.go`: the index page; `chrome.go`: watermark, footer, lab slides; `document.go`: lab / article / guide page; `guide.go`: guide sidebar, breadcrumb, pager, partials, `RenderGuide` + link/anchor checks; `site.go`: site navigators and page frame; `plugins.go`: alert, badge, event, accordion, modal, carousel; `embed.go`: `slides:` and `embed:`; `mindmap.go`: mind maps (`type: mindmap` pages and the ```` ```mindmap ```` fence: list → JSON maps, linked map files inlined); `codewin.go`: code windows (window= / id= fences grouped into shows, callouts, step markers on slides) |
| `internal/serve/` | `si play` / `si serve`: on-the-fly render (decks, articles, guide pages with their `_nav.md`), static files, SSE `/_si/events`, mtime-polling watcher (decks, assets, `--css` paths) |
| `internal/studio/` | `si --web`: gateway (token, Host, Origin), confined workspace, file/deck/check/rules API, assembly with the `si play` server; `mindmap.go`: the map editor's `POST mindmap` (apply an op to the editor's text, return text + maps) and `GET mmcanvas` (the canvas page, `web/mindmap-edit.js`); `codewin.go`: the Window view's `POST codewin` (an op on the editor's text, back text + show + preview page) and `POST codewin/export` (`si code`) |
| `web/studio/` | Studio front-end source (Preact + TS + CodeMirror 6, esbuild, vitest, Playwright e2e); bundle committed to `internal/assets/web/studio/assets/` via `make studio-ui` |
| `internal/config/` | optional `si.conf` (flat `key: value`, stdlib): default theme, `keys.*` command remaps, `feature.*` on/off; injected into the deck as `window.SI_CONFIG`, read by `deck.js` |
| `internal/assets/` | embedded web files (`web/layout.css` unlayered mechanics, `web/theme.css` looks, `web/deck/*.js` runtime (one file per feature, joined in name order into one script), `web/doc.css` + `web/doc.js` lab documents (`type: lab`), `web/presenter.html` teleprompter, `web/mindmap.js` + `web/mindmap.css` mind maps: layout, lines, drill-down, search, Play; `web/codewin.js` + `web/codewin.css` code windows: chrome, backdrops, callouts, the morph between steps, `#si-code-export`; `web/mindmap3d.js` the 3D view on vendored `web/three.min.js` + `web/three-orbitcontrols.js`, inlined only with a 3D map) and `contract.go` (tokens, hooks, theme values, starter CSS) |
| `docs/docs.go` | embeds RULES.md + reference/user-manual.md + agent-rules.md for `si rules` / `si manual` / `si --export agent-rules` (in `docs/` so go:embed can reach them; no `.go` at the repo root) |
| `examples/` | showcase deck, included appendix, `assets/journey.svg` canvas, media, `css/harbor.css` theme, one example per document type (`article/`, `guide/`, `book/`, `demos/`, `site/`); embedded for `si new` and `--export example --type` |
| `docs/screenshots/` | headless screenshots referenced by the README |
| `internal/tui/` | si in the terminal: `slide.go` + `text.go` (the deck as text and colour: layouts, builds, code, tables, cards), `font.go` (block letters), `image.go` (half-block pixels), `palette.go` (theme tokens), `play.go` (`si play --terminal`, `--dump`), `notes.go` (the `--notes` teleprompter), `sync.go` (the local socket between them), `studio.go` (`si --terminal [dir]`: files, preview, problems, overlays, modes), `lab.go` (Lab mode), `pictures.go` (Kitty / iTerm2 pictures); see `docs/terminal-studio.md` |
| `desktop/` | si Studio.app for macOS: its own module; `main.go` (starts/stops `si --web` per folder), `app_darwin.m` (Cocoa + WebKit windows and menus), `build-mac.sh` (universal bundle + zip, `make desktop-mac`) |

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

- **`si play --remote`** (DONE, see `docs/remote.md`): a keyed second
  listener for a tablet: `tp` (teleprompter) and `draw` (pen pad) roles, synced
  through a server relay (SSE + POST); the transport is injected only in play.
  The phone `clicker` role is in (v1.11.0). Next: HTTPS or a pairing code for
  untrusted networks.
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
  Next: MP4 export.
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
  camera, a portrait 3D fit. Next: editing in the terminal Studio (read-only
  tree today), a 3D canvas in the map editor, multi-select and copy/paste of
  branches, XMind 8 export.
  **3D** (v1.13.0 DONE): `view: 3d` / `view=3d`, `camera: iso|perspective`,
  `floors: on|off`; pills on a shadow floor, cables with flow dots, linked maps
  as glass floors, fly-in drill-down, cards, `/` search, Play, 2D/3D and camera
  toggles; mockup `docs/superpowers/specs/2026-11-06-mind-maps-3d.html`.
  **Kinds** (v1.14.0 DONE): `kind: blocks|bricks|toy` / `kind=`, a Kind button
  in the 3D tools (never call bricks by a toy brand name). Next: a
  phone layout for 3D (portrait maps are small), the visual builder in 3D,
  exports of the 3D view.
- **Publishing (`si publish`, planned):** Medium is out (no new API tokens).
  LinkedIn via "Share on LinkedIn" (`w_member_social`, one-time loopback OAuth
  login, token ~60 days, stdlib only): a deck becomes a PDF document post
  (carousel) with text from `linkedin:` / `description:` / `Points:`; an article
  becomes a link post (`--url`); `--copy` puts rich text on the clipboard for
  LinkedIn's article editor (no articles API). Always a dry run unless `--yes`.
  Full articles: dev.to, Hashnode, Ghost or WordPress (real APIs). Check the
  current LinkedIn API docs before building.
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
