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
   video inlined (or copied with `--no-inline-media`). The only network
   resource allowed is the pinned Mermaid script, and only for decks with a
   Mermaid fence.
3. **Every rule is documented in RULES.md with an example, and used in
   `examples/`.** A directive that is not in RULES.md does not exist.
4. **Example decks carry content, never instructions.** No slide says "press
   the arrow" or "this slide shows the two-col layout"; the showcase is a real
   talk (an order's journey) that happens to use every rule.
5. **Themes change tokens and looks, never layout mechanics.** Layout rules
   live unlayered in `layout.css`; looks live in `@layer si.theme`; user CSS
   in `@layer si.user`. Every `--si-*` property and `.si-*` class used by the
   built-in CSS is listed in `internal/assets/contract.go`
   (`TestContractCoversBuiltinCSS` fails otherwise), so
   `si --export css-rules` cannot drift.
6. **The teleprompter stays a real teleprompter:** large notes, auto-scroll
   with adjustable speed, manual scroll, mirror, per-slide and total timers
   with pace, previews, synced over BroadcastChannel, works from `file://`.
7. Go stdlib plus goldmark only. `gofmt`, `go vet ./...` and `go test ./...`
   green before every commit. Plain commit messages, no attribution trailers.

## Layout of the code

| Path | What |
|---|---|
| `cmd/si/main.go` | CLI: GNU-style flag parsing (interspersed, long flags), commands play, build, check, new, rules, `--export` |
| `internal/deck/` | parse a deck: front matter, includes (cycle check), `---` split, headings, `Notes: (N min)`, slide directives; `media.go` validates background and image options. No HTML. |
| `internal/render/` | deck → HTML. `markdown.go`: goldmark + AST transform for body directives (steps, columns, svg, image figures) and fenced code; `svg.go`: canvas inlining, id prefixing, state resolution, helper-attribute CSS; `media.go`: data URIs / link / copy modes, backgrounds, figures; `highlight.go`: tiny highlighter; `css.go`: user CSS layer; `render.go`: page, index |
| `internal/serve/` | `si play`: on-the-fly render, static files, SSE `/_si/events`, mtime-polling watcher (decks, assets, `--css` paths) |
| `internal/config/` | optional `si.conf` (flat `key: value`, stdlib): default theme, `keys.*` command remaps, `feature.*` on/off; injected into the deck as `window.SI_CONFIG`, read by `deck.js` |
| `internal/assets/` | embedded web files (`web/layout.css` unlayered mechanics, `web/theme.css` looks, `web/deck.js` runtime, `web/presenter.html` teleprompter) and `contract.go` (tokens, hooks, theme values, starter CSS) |
| `rules.go` | embeds RULES.md for `si rules` |
| `examples/` | showcase deck, included appendix, `assets/journey.svg` canvas, media, `css/harbor.css` theme; embedded for `si new` and `--export example` |
| `docs/screenshots/` | headless screenshots referenced by the README |

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
4. Runtime behaviour: `web/deck.js` (and `web/presenter.html` if the
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

- More themes and a theme gallery page; `theme:` from a CSS file path.
- PDF export without the browser print dialog (headless Chrome `--print-to-pdf`).
- SVG helpers: `data-move` along a path, counters that tick between states,
  `data-grow` bars; an SVG lint in `si check` (less-than sign in `<style>`,
  missing viewBox).
- Shared media: a data URI used on many slides embedded once (the showcase
  repeats the dusk photo).
- Speaker view extras: notes from a separate file, rehearsal timings saved
  per slide.
- Replace the remaining Python deck builder workflows (fully Python-free).
- `si play` QR code for opening the deck on a phone on the same network.
