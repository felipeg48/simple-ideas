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
  …); **build steps**; speaker `Notes: (N min)`.
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
- **Teleprompter / presenter window:** large notes, auto-scroll + speed, manual
  scroll, mirror, per-slide + total timers with pace, per-slide rehearsal
  timings, previews, synced over BroadcastChannel; works from `file://`.
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

## Planned — v0.6.0

- **`si play` terminal drawer** (see `docs/terminal.md`): a bottom→middle HUD
  with **multiple tabs** to run **local** commands live during demos (`docker
  ps`, `cockroach start-single-node`, `cockroach sql`).
  - **Decided:** full interactive terminal — **xterm.js** front-end + a
    **PTY-backed** shell (e.g. `creack/pty`). Deliberately relaxes rules #2/#7
    for this subsystem (founder's call).
  - **`si play`-only + loopback-only** (disabled on `--addr 0.0.0.0`, never LAN
    RCE; built static decks have no server, so it's inert there).
  - Open: transport (stdlib has no WebSocket), per-tab process lifecycle,
    Ctrl-C/SIGINT, per-tab cwd/env. When built, carve the rule #7 exception into
    CLAUDE.md.
