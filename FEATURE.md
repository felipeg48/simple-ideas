# si — feature list

`si` turns Markdown or AsciiDoc into animated, fluid, self-contained HTML slide
decks. `si play` renders live with reload; `si build` writes one HTML file per
deck (plus an index). Go stdlib + goldmark only; every deck is self-contained.

## Authoring

- **Markdown or AsciiDoc** input (`.md` / `.adoc`); AsciiDoc is converted by a
  stdlib preprocessor (headers, sections, lists, tables, source/callouts, quote
  blocks, includes, admonitions). See `docs/asciidoc.md`.
- **Front matter:** `title`, `author`, `date`, `theme`, `mode`, `accent`,
  `transition`, `aspect`, `background`, `mermaid`, `lang`, `status`, `version`,
  `year`, `copyright`.
- **Slides & layouts:** `---` splits; layouts (`two-col`, `big-number`, `quote`,
  …); **build steps**; speaker `Notes: (N min)` (verbatim script) and `Points:`
  (talking points).
- **Includes:** `<!-- include: path -->` (split decks: thin main + `sections/`),
  cycle-checked.
- **Persistent SVG canvas** across slides: state words + per-step ops; helpers
  `data-show/hide/ok/alert/flow/packets/grow/move/count`. Scaffold with
  `si new-canvas`. See `docs/authoring-canvases.md`.
- **`stage` fence:** Mermaid-like animated flow diagrams (chains, fan-out,
  merges), step-drivable, native/offline. See `docs/stage-diagrams.md`.
- **Media:** colour/gradient/image/video backgrounds with overlays; sized,
  aligned, captioned, full-bleed images; inlined (or copied with
  `--no-inline-media`).
- **Mermaid** fences (pinned CDN, the only allowed network resource).

## Presenting (`si play`)

- **Live reload** over SSE (decks, assets, stylesheets); URL hash keeps the slide.
- **Teleprompter / presenter window:** two tabs — **Teleprompter** (verbatim
  `Notes:` script, large type, auto-scroll + speed, mirror) and **Speaker Notes**
  (`Points:` talking points); manual scroll, per-slide + total timers with pace,
  per-slide rehearsal timings, previews, synced over BroadcastChannel; works from
  `file://`.
- **Play terminal + runbook** (`si play`, loopback-only; never in a built deck):
  a bottom drawer of real PTY shells (xterm.js), plus an optional **runbook**
  right-drawer (`runbook: <file>.json` front matter) — a syntax-highlighted
  command cheat-sheet; filter, click to type into the shell, Shift-click to run.
  JSON shape matches the cockroach-engineering-tool VS Code extension. See
  `docs/terminal.md`.
- **Break timer HUD** (`t`): presets (`timer.time`/`timer.unit`), **digital or
  analog sport-clock face**, a background **animation layer** (built-ins
  `waves`/`orbit`/`bokeh`, custom via `timer.animation.path`, comma-list toggle,
  always-offered **None**). Space pauses + reveals the controls; resume hides
  them (focus mode). See `docs/timer-animations.md`.
- **Find palette** (`/`): search slide titles + body (ranked, highlighted), or
  type a number to jump to that slide.
- **Draw/annotate** (`a`): pen over the slide — four colours, **brush sizes**,
  erase/undo/clear.
- **Overview** grid (`o`); light/dark (`d`); full screen (`f`); key help (`?`).
- **Open on a phone:** `si play --addr 0.0.0.0` prints a LAN URL + a QR code.
- **si.conf:** default theme, key remaps (`keys.*`), feature toggles
  (`feature.*`), timer settings.

## Building & output

- **`si build`:** one self-contained HTML per deck + an `index.html` landing page
  with a **tile per deck**; **Tiles / List** view toggle (list is a table:
  Course · Status · Slides · Time · PDF).
- **`--site "Title"`** (alias `--title`): the index `<title>` + `<h1>`; the index
  follows the decks' theme.
- **`--pdf`:** renders `pdfs/<name>.pdf` per deck (headless Chrome) and adds a
  **PDF download badge** to each tile + a PDF column in list view.
- **Index badges:** `status:` (beta/preview/new/deprecated, colour-coded) and
  `version:` front matter render as badges; title slide shows a
  `© {year} {copyright}` **colophon**.
- **Themes:** `studio`, `paper`, `mono`, `crdb` (brand), a CSS theme file, or an
  `https://` URL; documented `--si-*` token + `.si-*` hook contract
  (`si --export css-rules`, enforced by `TestContractCoversBuiltinCSS`).

## Tooling

- **Commands:** `play`, `build`, `check`, `new`, `new-canvas`, `migrate`,
  `manual`, `rules`, `selfupdate`, `--export css-rules|rules|conf|example|themes`.
- **`si check`:** validates titles, directives, options, missing files, include
  cycles, malformed `stage` blocks, broken canvas SVGs.
- **`si migrate`:** prints an editorial-migration prompt (optionally piped to an
  agent with `--ai`).
- **`si selfupdate`:** fetches the latest release for the OS/arch, checksum-
  verified, replaces the running binary (never downgrades).
- **`make install`:** builds and installs to `~/.bin` (offers to add it to PATH).
- **Release:** GitHub Actions CI (gofmt/vet/test) + a release workflow that
  cross-compiles 6 targets (darwin/linux/windows × amd64/arm64) + `SHA256SUMS`
  on every `v*` tag. README badges.
- **Docs:** `si manual` renders the user manual (`--pdf` / `--slides`);
  `docs/reference/` holds the user manual + developer reference.

## Shipped — v0.6.3

- **`si play --dev` hot reload.** On save, instead of a full-page reload, the dev
  client fetches the re-rendered deck and morphs the changed slides in place —
  no white flash, and the current slide, step and overview are preserved. Falls
  back to a full reload when the slide count changes, and shows an overlay when a
  deck fails to render. (Plain `si play` still does the full-page live reload.)
  deck.js exposes `SI.refresh()`; the client is `DevReloadJS` in render.
- **Lab slides (`labs` directive).** `<!-- labs: [Create a Cluster](https://…),
  [Scale out](https://…) -->` on a slide renders a centred beaker icon, a big
  heading (`labs-label` front matter, default "Now, let's do a Lab!"), and one
  button per lab opening its URL in a new tab. Bare URLs also work (default
  label). New hooks `.si-lab`, `.si-lab-icon`, `.si-lab-title`, `.si-lab-links`,
  `.si-lab-link`.

## Shipped — v0.6.2

- **Footer** (front matter). `footer: <text>` puts small muted text in a corner of
  every slide; `footer-position:` picks the corner (`bottom-right` default, plus the
  other five, with short aliases `br`/`bl`/…). A slide's own `<!-- footer: … -->`
  overrides it and `<!-- footer: off -->` hides it (e.g. the title slide).
- **Watermark** (front matter). `watermark: Draft` draws a large faint angled word
  behind every slide; `watermark: ./img.png` inlines an image once (centred);
  `watermark-opacity:` tunes it (default 0.06). Shows in the overview and in print.
- Also fixed: front-matter keys may now contain hyphens (`metaRE`) — previously a
  hyphenated key silently rejected the whole front-matter block.
- New hooks `.si-footer`, `.si-watermark`, `.si-wm-text`, `.si-wm-img`; tokens
  `--si-wm-opacity`, `--si-wm-img`.

## Shipped — v0.6.1

- **Topic-column Overview.** Pressing `o` now groups the overview into one column
  per topic — the `<!-- include: -->` section each slide belongs to, emitted by
  the renderer as `data-topic` (0 for the title/agenda framing, 1..N per section).
  Each column is headed by its section title; framing slides form a leading
  "Intro" column. Keyboard: ← → move across topic columns, ↑ ↓ within a column,
  Enter opens the selected slide, Esc/`o` close; click-to-go still works. CSS
  drives off `--si-ov-cols`; new hooks `.si-ov-head`, `.si-slide.si-sel`.
  An included module without its own `#` section slide is a
  topic too (headed by its first slide's title), and a deck with no sections
  or modules falls back to a plain row-by-row grid instead of one tall column.

## Shipped — v0.6.0

- **`si play` terminal drawer** (see `docs/terminal.md`): a bottom→middle HUD
  with **multiple tabs** to run **local** commands live during demos (`docker
  ps`, `cockroach start-single-node`, `cockroach sql`). Toggle with `` ` ``.
  - Full interactive terminal — **xterm.js** front-end + a **PTY-backed** shell
    (`creack/pty`); one real `$SHELL` per tab, so Ctrl-C, REPLs and full-screen
    apps work. Deliberately relaxes rules #2/#7 for this subsystem (founder's
    call); carved into CLAUDE.md rule #7.
  - **`si play`-only + loopback-only** — `Server.Terminal` is set only on a
    loopback listen address, so the exec routes never exist on `--addr 0.0.0.0`;
    built static decks have no server, so it is inert there. `feature.terminal:
    off` opts out.
  - Transport: chunked-GET output + POST input (no WebSocket dependency added).
    Shells are reaped on tab close and on server shutdown.
