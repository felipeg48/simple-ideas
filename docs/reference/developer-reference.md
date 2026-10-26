# si Developer Reference

Technical reference for working on `si` itself: the architecture, the render
pipeline, the package layout, and how to extend each part. For the *user-facing*
format, see `si rules` / `RULES.md` and `docs/reference/user-manual.md`.

## Principles (non-negotiable)

1. **Fluid, never fixed.** No fixed slide canvas scaled by `transform`. Type uses
   container units and `clamp()`; SVGs scale by `viewBox`; image sizes are `%`,
   `em` or `cqi`. `TestLayoutHasNoFixedCanvas` guards this.
2. **Self-contained output.** One deck = one HTML file (CSS, JS, SVG, images,
   video inlined). The only runtime network resource is the pinned Mermaid script,
   and only for decks with a Mermaid fence.
3. **Every rule is in RULES.md with an example and used in `examples/`.**
4. **Themes change tokens and looks, never layout mechanics.** Layout lives
   unlayered in `layout.css`; looks in `@layer si.theme`; user CSS in
   `@layer si.user`. Every `--si-*` token and `.si-*` hook is listed in
   `internal/assets/contract.go` (tests fail otherwise).
5. **Go stdlib + goldmark only.** No other dependencies. `gofmt`, `go vet ./...`
   and `go test ./...` green before every commit.

## Package layout

| Path | Responsibility |
|---|---|
| `cmd/si/` | CLI: flag parsing, the `play`/`build`/`check`/`new`/`new-canvas`/`manual`/`migrate`/`rules`/`--export` commands, PDF export (`deckToPDF`, `findChrome`), QR (`qr.go`), LAN URL |
| `internal/deck/` | Parse a deck: front matter, includes (cycle-checked), `---` split, headings, `Notes: (N min)`, slide directives, media option validation. **No HTML.** |
| `internal/render/` | Deck → HTML. `markdown.go` (goldmark + AST transform for body directives and fenced code: code highlight, `mermaid`, `stage`), `svg.go` (canvas inlining, id prefixing, state resolution, helper CSS, `lintSVG`), `stage.go` (the `stage` generator), `media.go` (data URIs / copy / backgrounds / figures), `highlight.go`, `css.go`, `render.go` (page assembly, theme/animation resolution, index) |
| `internal/serve/` | `si play`: on-the-fly render, static files, SSE live reload, mtime watcher |
| `internal/config/` | Optional `si.conf` (flat `key: value`, stdlib): default theme, `keys.*`, `feature.*`, `timer.*`. Injected as `window.SI_CONFIG`. |
| `internal/assets/` | Embedded web files (`web/layout.css`, `web/theme.css`, `web/deck.js`, `web/presenter.html`, `web/anim/*.js`) and `contract.go` (tokens, hooks, theme values, animation lookup) |
| `examples/` | The showcase deck + night-shift sample, embedded for `si new` / `--export example` |
| `docs/` | Authoring guides (stage, canvases, timer animations) and this reference |
| `rules.go` | Embeds RULES.md (`si rules`) and the user manual (`si manual`) |

## The render pipeline

1. `deck.List(src)` finds decks (non-recursive; skips subdirs, `_`-prefixed, README).
2. `deck.Parse(file)` produces a `*deck.Deck`: front matter `Meta`, `[]Slide`
   (each with `Body`, `Dir`, `Notes`, `Minutes`), resolving `<!-- include: -->`
   with cycle detection. No HTML is produced here.
3. `render.Deck(d, opt)` drives the conversion:
   - `resolveTheme()` → `dr.themeCSS` (built-in, CSS file, or fetched URL).
   - `resolveAnim()` → the timer animation code (built-in or path).
   - For each slide, `slide()` → `toHTML()` runs goldmark with a custom
     `nodeRenderer`. `transform()` walks top-level blocks and applies body
     directives (`svg`, `step`/`steps`, `column`, `image`); the `fence` renderer
     handles ` ```mermaid `, ` ```stage ` and highlighted code.
   - `page()` assembles the document: `<style>` (`LayerOrder` + `LayoutCSS` +
     theme CSS + `ThemeLooksCSS` + `helperCSS` + per-deck rules), the deck JSON,
     `window.SI_CONFIG`, the animation, `deck.js`, and the slides.
   - Returns `Result{HTML, Problems, Warnings, Media, Steps, Canvases}`.
4. `cmd/si/build` writes `<name>.html` per deck + `index.html`; `--pdf` runs
   `deckToPDF` (headless Chrome `--print-to-pdf`).

The browser runtime (`deck.js`) owns navigation, step/canvas state (`stateAt`,
`applyCanvas`, `applyCounts`), transitions, overview, the teleprompter sync
(BroadcastChannel), the break-timer HUD + animation layer, draw, print (`#print`),
background brightness overlay, and SVG packets.

## How to extend

### Add a directive
1. `internal/deck/deck.go`: add it to `Directives` (scope `slide` or `body`);
   validate its value in `slideDirective`.
2. Slide directive → use the field in `render.slide`. Body directive → handle it
   in `transform` in `internal/render/markdown.go` (applies to the next block).
3. CSS: mechanics in `web/layout.css`, looks in `web/theme.css`; any new `--si-*`
   token or `.si-*` class goes into `Tokens` / `Hooks` in `contract.go`.
4. Runtime: `web/deck.js` (and `web/presenter.html` if the teleprompter needs it).
5. Tests: a case in `deck_test.go`, a use in `internal/render/testdata/deck.md`
   (then `-update` the golden file and read the diff), an assertion in
   `render_test.go` if it emits CSS.
6. RULES.md: a section/row with an example; use it in `examples/showcase.md`;
   `si check examples`; screenshot it headless.

### Add a theme
Add an entry to `Themes` in `contract.go` (Base fonts + Light/Dark: the 12
required colours — bg, fg, muted, surface, surface-2, line, accent, accent-2,
on-accent, ok, warn, crit). `TestThemesDefineEveryColour` enforces the set.
Add any alias to `themeAliases` and to `deck.Themes`.

### Add a canvas helper attribute
1. `svg.go`: add it to `helperRE` and emit its state-keyed CSS in `helperCSS`
   (or handle it in `deck.js` for JS-driven helpers like `data-count`).
2. `web/layout.css`: a default rule under `.si-canvas [data-...]` if needed.
3. RULES.md §10 table + `docs/authoring-canvases.md`; use it in an example.

### Add a built-in timer animation
Drop a `web/anim/<name>.js` (a function expression `(canvas, api) → stop`; `api`
gives `tokens`, `fraction()`, `remaining()`, `reduced`) or `<name>.svg` into
`internal/assets/web/anim/`. `assets.Animation` / `AnimationNames` pick it up.
Document it in `docs/timer-animations.md`.

## Testing & QA

```bash
make test                          # go vet + go test ./...
go test ./internal/render -update  # rewrite the golden file after an intended markup change
make build && ./bin/si check examples
```

Visual checks use **headless Chrome** (never the Playwright MCP browser tools):

```bash
"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" \
  --headless=new --screenshot=out.png --window-size=1440,900 "file://.../deck.html#5.2"
```

Add `--virtual-time-budget=2000` so fades, counters and canvas transitions settle
before the frame is captured.

## The styling contract

`contract.go` is the single source of truth for the public CSS surface:
`Tokens` (every `--si-*`), `Hooks` (every `.si-*`), and `Themes`.
`TestContractCoversBuiltinCSS` fails if `layout.css`/`theme.css` use a token or
hook not listed, so `si --export css-rules` can never drift from reality.
