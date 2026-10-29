# si Studio (v1.0.0): design

Status: approved direction (all four mockups combined), written 2026-10-28.
Mockups: `.superpowers/brainstorm/56617-1793228937/content/studio-directions.html`.

## Intent

`si` today is a CLI: authors edit Markdown or AsciiDoc in their own editor and
run `si play` / `si build`. si Studio is the authoring and presenting
environment for si itself: one app where a course author creates a project,
writes decks, sees them render live, rehearses with the teleprompter, presents
full screen, runs labs from the built-in terminal with runbooks, and exports
PDFs. It must feel like a professional tool with a "wow" factor (Xcode /
VS Code / Linear / Keynote quality), not a form over a text box.

Why now: the CACI course (5 decks, 2 reference decks, 9 labs, a 249-command
runbook, handouts, demo scripts) showed that the CLI + external editor loop is
the bottleneck; the presenting features (teleprompter, terminal, runbook
drawer) already exist and want one home.

Success: the founder authors and delivers a full course (like CACI) without
leaving Studio, on the desktop app and in a browser via `si --web`.

## One app, two shells

- **`si --web [dir]`**: the `si` binary serves Studio as a web app (like
  code-server) from its own HTTP server, opens the browser. Pure Go stdlib on
  the server side, consistent with the rest of `si`.
- **si Studio desktop**: a Wails app (separate Go module, separate binary,
  e.g. `cmd/si-studio/` with its own `go.mod`) that embeds the same web UI and
  the same Go handler in-process. No duplicate UI code: the desktop is a shell
  around the web app.
- Both open a **workspace**: a folder. No project file is required.

## The app: four modes on one spine

A mode switch at the top (Mission Control, mockup D) is the spine:
**Write · Rehearse · Present · Lab**. The ⌘K command palette (mockup C) works
in every mode. Dark and light app themes everywhere.

### Write (mockups A, B, C)

Default layout (Studio, A):
- **Navigator** (left): the workspace tree, grouped as si understands it:
  decks (main files, with their included sections nested and the section
  headings listed under each file), images/anim and diagrams, labs, runbooks,
  handouts, themes, `si.conf`. Counts and status badges (e.g. labs verified,
  runbook entry count). Create / rename / move / delete files and folders;
  drag images in.
- **Editor** (centre): tabs; Markdown and AsciiDoc with si-aware syntax
  highlighting; directives rendered as **chips** (e.g. an `svg:` directive
  shows the canvas file, its state words and a tiny thumbnail); completion for
  directive names, layouts, transitions, state words of the referenced canvas,
  include paths, image paths; inline `si check` diagnostics (gutter + a
  Problems panel). SVG canvases open as text with a live canvas preview;
  runbook JSON opens in a form editor (categories, entries, target) with a raw
  JSON toggle.
- **Live preview** (right): the real renderer (the same output as
  `si play --dev`), not an approximation. Follows the cursor: the slide that
  contains the cursor line is shown (slides already carry
  `data-src="file:line"`). Step controls, size presets (16:9, phone, portrait,
  4K) to honour the fluid-layout rule. Filmstrip under it.
- **Inspector** (far right): properties of the slide under the cursor (layout,
  transition, background, classes, steps, notes minutes, canvas states) with
  edits written back as directives; block time budget.
- **Status bar**: branch name (read-only), cursor position, file type, check
  summary (decks OK, total minutes, steps, canvases), render time.

Two alternative views inside Write:
- **Storyboard** (B): the whole deck as a canvas, one column per topic
  (the same topics as the `o` overview: sections and included modules). Click
  a slide: a floating editor with markdown beside the live render and a step
  scrubber. A **time ribbon** shows notes minutes per topic and the deck total.
- **Focus** (C): chrome fades; a calm typographic editor; the live slide
  floats picture-in-picture.

### Rehearse

The existing teleprompter (presenter view) inside the app: notes with
auto-scroll, timers, pace, previews, the pace ring from mockup D. Uses the
existing BroadcastChannel sync.

### Present

The deck full screen; on the desktop app, on a chosen display, with Rehearse
on the other. In the browser: the existing presenter popup flow.

### Lab (mockup D)

Live slide + teleprompter notes + the real terminal (existing PTY terminal,
multi-tab) + the runbook drawer (existing v0.7 runbook format): filter, click
types, Shift/⌘-click runs.

### Labs, runbooks and handouts *(designer's call)*

No new si concepts. The navigator groups files by the folder conventions the
CACI course already uses: `labs/` (Markdown guides plus `files/`),
`handouts/`, and runbooks = any JSON a deck names in `runbook:` (plus `.json`
files with the runbook shape). They are edited as ordinary Markdown/JSON
files; HTML renders of guides stay the workspace's own scripts. The
"verified" badge in the mockups is decoration and is not in v1.

### Runbooks: generator and editor (decided)

- **Generator:** "New runbook" asks for the runbook's name and description,
  then guides the author through categories and entries (label, description,
  command, target `bash` or `sql`), producing a runbook JSON file in the
  v0.7 format.
- **Editor:** a visual editor for any runbook JSON: add, edit, duplicate,
  delete and reorder (drag and drop) categories and entries, with a
  **Raw JSON** toggle for manual editing. Both views stay in sync; invalid JSON
  or a malformed entry is flagged inline and never silently dropped.
- **Placeholders:** commands may contain placeholders `{name}` or
  `{name:default}` (the VS Code extension's `{node1}` style). The editor
  highlights them and offers insert helpers (pick a name, optional default,
  reuse names already used in the runbook). This needs a small `si` core change:
  when a drawer entry contains placeholders, the terminal drawer asks for the
  values (prefilled with defaults, remembered for the session) before typing
  or running the command. Documented in RULES.md like any runbook rule.

### Creating things

From the navigator and ⌘K: new workspace (from `si new` scaffolds), new deck,
new section module (wired into the main deck's includes), new canvas (from
`si new-canvas`), new lab guide, new runbook, new handout, new theme (from
`si --export css-rules`). Export: Build site (`si build`, with `--site`,
`--css`), PDF per deck (`--pdf`), with progress. Check workspace.

## Architecture

- **Go core stays the engine.** Studio is a new package (e.g.
  `internal/studio`) that exposes an HTTP JSON API over the existing packages:
  `deck` (parse, problems), `render` (pages, topics, runbooks), `serve`
  (render-on-request, SSE reload, terminal hub), build/PDF from `cmd/si`
  (refactored into a callable function). Studio reuses, it does not fork:
  the preview is served by the same code as `si play --dev`.
- **API surface (v1):** workspace tree; file read/write/create/rename/move/
  delete (with an mtime/etag check to avoid overwriting external edits);
  file-change events over SSE (the existing poll watcher); check/diagnostics
  per file; slide map for a file (slide index ↔ source lines, topics, minutes,
  steps, canvas states); render/preview routes (existing); build and PDF jobs
  with progress over SSE; rules metadata (directive list, layouts,
  transitions, front-matter keys from the existing contract/`Directives` so
  completion never drifts from RULES.md); terminal (existing routes); runbook
  read/write.
- **Saving:** autosave after a short idle (and on blur/mode switch); the
  preview reloads through the existing dev hot-reload path. Unsaved state is
  never lost on reload (buffer kept until written).
- **UI:** a single-page app embedded in the binary with `go:embed` (as the
  terminal assets are). The editor component is CodeMirror 6 (vendored,
  bundled once, committed), styled to the app theme. UI framework and build
  tooling are the builder's call with two constraints: the Go build must not
  need Node (commit the built bundle), and the bundle stays modest.
- **Desktop shell:** Wails, separate module, serving the same embedded assets.
  *(Designer's call.)* The desktop app also starts the Studio HTTP server on a
  loopback port behind the same auth gateway, and its windows load from it.
  This keeps one code path: SSE (reload, file events, job progress), the
  terminal's chunked streams and BroadcastChannel window sync all work
  unchanged, with no Wails-specific bridge. Present on a second display needs
  multi-window support; choose the Wails major version that gives it.

## Security (designed for remote later, local only in v1)

Studio can write files and run shells, so a local web server is a real attack
surface even on loopback (DNS rebinding, cross-site requests from any open
web page).

- v1: listen on loopback only, as the terminal does.
- Every API request goes through **one auth gateway** from day one:
  per-launch random token (passed in the opened URL, then a cookie,
  HttpOnly + SameSite=Strict), Host header must be the loopback host and port,
  Origin must match, no CORS. Static assets may be public; APIs and the
  terminal never are.
- Remote later (not v1): `--addr` beyond loopback only with explicit
  `--auth` (token or password) and TLS (`--tls-cert/--tls-key`); terminal off
  unless separately enabled. The gateway is the single place that changes.
- The desktop app uses the same loopback server and gateway as `si --web`.

## Rules and exceptions

The non-negotiable rules of CLAUDE.md hold for `si` and for every built deck:
fluid layout, self-contained output, every directive documented in RULES.md,
themes change tokens not mechanics, Go stdlib + goldmark in the core.
Documented exceptions, scoped exactly like the terminal:
- **Rule #7:** the desktop module (Wails and its dependencies, in its own
  `go.mod`) and the vendored front-end assets (CodeMirror, the UI bundle).
  The `si` core module adds no new Go dependencies for Studio.
- **Rule #2:** Studio's assets are served by Studio only; nothing from Studio
  ever ends up in a built deck.
CLAUDE.md, HANDOFF.md and a new `docs/studio.md` record these, as
`docs/terminal.md` does for the terminal.

## Theming

- App UI: dark and light themes, following the OS by default, with a toggle
  (persisted). Both palettes must be first-class (contrast-checked), with the
  mockups' look as the dark reference.
- The preview keeps the deck's own theme and its own light/dark mode (the
  existing `d` toggle); by default it follows the app theme.

## Goals

- Author, preview, check, rehearse, present, run labs and export without
  leaving the app, on desktop and web.
- Live, real-renderer preview that follows the cursor and animates canvases.
- si-aware editing: chips, completion, diagnostics, inspector.
- The wow factor: fast, polished, consistent in dark and light.

## Non-goals (v1)

- Multi-user collaboration, accounts, cloud sync.
- Remote access (designed for, not shipped).
- A git client beyond showing the branch.
- WYSIWYG drag-and-drop slide design; Markdown/AsciiDoc stays the source of
  truth.
- New deck syntax. Studio edits what RULES.md defines; any new directive goes
  through the normal "add a directive end to end" process.

## Anti-goals (failure even if it works)

- Studio code or assets leaking into built decks, or the core gaining
  dependencies.
- Typing that lags, or a preview that is not the real renderer.
- Losing an author's unsaved text, or silently overwriting edits made in
  another editor.
- An app that only looks good in dark mode.

## Scope of v1.0.0 and platforms (decided)

- **Everything in this document ships in v1.0.0**: `si --web` and the desktop
  app, all four modes (Write with Storyboard, Focus and Inspector; Rehearse;
  Present; Lab), creating things, Build and PDF export, dark and light themes.
  Phasing inside the build is fine; the 1.0.0 tag waits for all of it.
- **Desktop platforms: macOS, Windows and Linux** are the target, released by
  the existing tag-driven release workflow next to the `si` binaries.
  **Order of work (decided): macOS desktop and `si --web` first**; Windows and
  Linux desktop builds follow once those are solid.
- **Licensing (decided):** everything is open source, Studio included, under
  the repository's Apache 2.0 licence. No private module.
- **Lab mode on every platform**, including Windows. The terminal needs a
  Windows pseudo-console (ConPTY) backend beside `creack/pty`; that is a
  further, documented rule #7 exception for the terminal subsystem, recorded in
  `docs/terminal.md` and `docs/studio.md`.
- **No code signing or notarisation on any platform** (decided). macOS builds
  ship unsigned; the release notes and README tell users how to open or sign
  the app themselves (Finder: right-click → Open, or
  `xattr -dr com.apple.quarantine "si Studio.app"`; or sign with their own
  identity). The same applies to Windows SmartScreen warnings.
- **Linux**: WebKitGTK is a documented runtime requirement of the desktop app;
  `si --web` has no such requirement.

## Left to the builder

- UI framework and front-end build tooling (within the constraints above).
- Exact API shapes, routes and package boundaries.
- Wails major version (given the multi-window requirement) and packaging.
- Command name for the desktop binary and how `si` launches it, if at all.
- Visual design details beyond the mockups and the theme rule.
- Phasing of the build, as long as each phase ships usable value and keeps
  the test suite and headless visual QA green.
