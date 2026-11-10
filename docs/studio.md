# Studio — `si --web` browser editor (shipped, v0.9.0)

`si --web [dir]` opens a browser editor for a folder of decks: a file tree, a
CodeMirror markdown editor, a live slide preview (the same `si play` renderer,
in an iframe) and a `si check` problems panel. This release is **Write mode**
plus the runbook editor and generator (below); the design is in `docs/superpowers/specs/2026-10-28-si-studio-design.md`.

```bash
si --web examples                 # opens http://127.0.0.1:7070/?token=... in your browser
si --web . --addr 127.0.0.1:8080 --no-open
```

`--no-open` prints the URL instead of opening a browser. The URL carries a
per-launch token; keep it private.

## Why it is a documented exception to the project rules

- **Rule #7 (Go stdlib + goldmark only):** the editor front-end is built with
  Preact, TypeScript and CodeMirror 6, bundled by esbuild and **committed** under
  `internal/assets/web/studio/assets/` (`app.js`, `app.css`). The Go module adds
  no dependencies. Node is needed only for `make studio-ui` and
  `make studio-e2e`, never for `go build` or `go test`.
- **Rule #2 (self-contained output):** the Studio assets are served only by
  `si --web` under `/_si/studio/assets/`. A built deck never contains or loads
  any of it.

Both exceptions are contained because Studio is loopback only (see Security).

## Security (enforced)

- **Loopback only.** `si --web` refuses any non-loopback `--addr`
  ("si --web is loopback-only in this version"). The Studio exposes file writes
  and a shell terminal, so a reachable address would be remote code execution.
- **Token to cookie.** Each launch generates a random token. The opened URL
  carries it once; the gateway swaps it for an `HttpOnly`, `SameSite=Strict`
  cookie named `si_studio_PORT` and every other request needs that cookie. The
  port in the name keeps two Studios on different ports from overwriting each
  other's session. Cookies are scoped by host, not port, so the browser still
  sends the cookie value to every server on `127.0.0.1`; do not run untrusted
  local servers next to Studio.
- **Host allowlist.** Only `127.0.0.1:PORT`, `localhost:PORT` and the listen
  address itself (for example `[::1]:PORT`) are accepted, which blocks DNS
  rebinding.
- **Origin check.** A non-GET/HEAD request carrying a foreign `Origin` is
  rejected.
- **Terminal methods.** The shell endpoints accept only the drawer's methods
  (`POST` for new, in, resize and close; `GET` only for the output stream), so
  a link or image in a deck cannot start or drive a shell.
- **Public surface:** only `/_si/studio/assets/*` (clean paths) is reachable
  without the cookie.
- **File API confinement.** Paths that escape the workspace (symlinks included)
  and any path with a dot-segment (`.git`, `.env`, `..`) are refused.
- **Atomic writes.** Saves write a temp file and rename it into place.
- **Etag conflicts never overwrite.** `PUT` carries the etag the editor loaded;
  if the file changed on disk the server answers with a conflict and writes
  nothing.
- **Trusted folders only.** Studio and the preview run the folder's HTML and
  JavaScript with your shell's rights (si renders raw HTML in markdown), on the
  same origin as the editor API. Only open folders you trust. Moving the
  preview to a separate origin is planned.

## How it is wired

| Layer | File | What |
|---|---|---|
| CLI | `cmd/si/main.go` | `--web`, `--no-open`; refuses non-loopback; mints the token; opens the browser |
| Gateway | `internal/studio/gateway.go` | token to cookie, Host allowlist, Origin check, public asset path |
| Workspace | `internal/studio/workspace.go` | root confinement, symlink and dot-segment refusal, atomic write, etag |
| File API | `internal/studio/files.go` | `GET tree`, `GET/PUT/DELETE file`, `POST create`, `POST rename` under `/_si/studio/api/` |
| Deck API | `internal/studio/decks.go`, `check.go`, `rules.go` | `GET decks`, `GET check` (same lint as `si check`), `GET rules` |
| Assembly | `internal/studio/studio.go` | mux: API, UI, and the `si play` server (live reload, terminal) behind the gateway |
| Embed | `internal/assets/assets.go` | `StudioFS` over `web/studio/` |
| Bundle | `internal/assets/web/studio/` | `index.html` plus the committed `assets/app.js`, `assets/app.css` |
| Front-end | `web/studio/` | Preact + TypeScript + CodeMirror 6 source, esbuild (`build.mjs`), vitest, Playwright e2e |

## Runbooks

A runbook is the JSON command cheat-sheet a deck shows in the `si play`
terminal drawer (`runbook: <file>.json` in front matter; see `docs/terminal.md`).
Studio edits and generates them. The file format is unchanged
(`{ "categories": [ { "name", "icon"?, "entries": [ { "label", "description"?,
"command", "target"? } ] } ] }`), plus optional top-level `name` and
`description`. Existing runbooks open unchanged.

### Placeholders

A command may contain `{{name}}` or `{{name:default}}` (name
`[A-Za-z_][A-Za-z0-9_-]*`, default any text without `}`). The drawer asks for
the values in an inline form before typing the command: each field is
prefilled with the value used earlier this browser session, else the default.
Enter types the command, Shift/Cmd/Ctrl+Enter types and runs it, Esc cancels and
types nothing. Values are kept in `sessionStorage` (`si-rb-vars`). Anything
else with braces (`{{.Names}}`, `${HOME}`, `{print $1}`) is typed verbatim.
The drawer title comes from the runbook `name`.

### Visual editor

Open a runbook `.json` file in Studio: each file has a **Visual / JSON**
toggle. Visual lists categories and entries; you can add, edit, duplicate and
delete them (delete asks to confirm), reorder by drag or with the move up/down
buttons (reorder is locked while the filter box is in use), and use **Insert
placeholder** while editing a command. Fields are validated inline.

- **Lossless.** Unknown keys at any level are kept; keys stay in their original
  order and absent keys stay absent. Objects created in Studio use the
  canonical key order. The file is written with a 2-space indent and a trailing
  newline, so editing one entry changes only that entry in the diff.
- **Same save path.** Visual edits change the editor document, so they use the
  normal autosave, etag conflict banner and local stash.
- **Broken input.** If the text is not valid JSON, Visual refuses with the
  parse error (line, column) and never rewrites the text. Parts of the file
  that do not have the runbook shape are shown read-only.

### New runbook wizard

**New runbook** walks through name and description, location and file name,
the commands (with placeholder help), optional deck links, a review and
create. Deck links add or replace `runbook:` in each chosen deck's front matter
using si's own front-matter rules. The wizard never overwrites: if the target
file exists it says so and stops.

## Writing help

- **Completion** (Ctrl+Space, or as you type after `<!-- ` or in front matter)
  knows every directive and front-matter key, each with a one-line help, and
  their values: layouts, transitions, `type:`, `theme:` (the built-in themes),
  `navigator:`, `mode:`, `aspect:`, `fonts:` examples, and the alert / badge
  kinds after a `·`. Directives and keys that take a path (`include`, `svg`,
  `embed`, `slides`, `code`, `runbook`, `demos`, `logo`, `watermark`,
  `theme`) offer the workspace's files of the right type, relative to the
  open file; `embed` and `demos` also offer folders that hold pages. Partials
  and nav files (`_*.md`) are left out. The help lines live in
  `internal/studio/rules.go`; a test fails when a directive has none.
- **New** (the **+** in the navigator, or right-click **New slides, article,
  guide…**) writes a small starter: slides, article, lab, demo script or site
  page as one file; a guide, book or site as a folder with its first pages
  (`internal/studio/starters/`). The name becomes the file or folder name and
  the title. It never overwrites. A page or folder created inside a guide,
  book or site is added to the end of its `_nav.md` / `_site.md`, so it is
  built.
- **Problems** come from one whole-workspace check (`GET checkall`, the same as
  `si check <dir>`: a guide, book or site is rendered once, not once per
  page) plus the open deck's own check for its slide map and minutes.
- **Build** (top bar) runs `si build` on the workspace into `html/`, with the
  Studio's `--css`, `--theme` and `--title`, optionally `--pdf`, and shows the
  command's output with a link to the built index. It calls the CLI's own entry
  point in-process.

## Modes and the command palette

The mode switch in the top bar works on the slide deck being previewed (any
other file offers a list of decks):

- **Write** — the editor and live preview.
- **Rehearse** — the live deck beside the real teleprompter (the deck page
  opened with `?si-view=tp`, synced over BroadcastChannel): notes, timers,
  pace and previews follow the deck, and the teleprompter's own controls drive
  it. **Teleprompter window** opens it in its own window instead.
- **Present** — the deck alone; **Full screen** puts it on the whole screen and
  **Open the teleprompter** opens the notes in another window.
- **Lab** — the deck with the `si play` terminal drawer opened (shells, the
  runbook and the ▶ Demos panel) beside the teleprompter.

Write stays mounted while another mode shows, so open files keep their state;
the Write preview is unmounted so only one copy of the deck answers the
teleprompter.

**⌘K / Ctrl+K** opens the command palette: every workspace file (fuzzy match on
the name or the path), the modes, New, Build, Show problems, light / dark, and
opening the previewed page in a new tab.

## Inspector, storyboard and focus

Write has three views (the switch beside Build, or ⌘K):

- **Editor** — the editor and the live preview. The **{ }** button in the
  preview toolbar opens the **inspector** for the slide under the cursor: its
  layout, transition, background, classes and eyebrow (a change is written into
  that slide as a directive — `<!-- layout: two-col -->` under its heading — as
  one undoable edit; an empty value removes the directive), and its time, the
  words of its `Notes:` script, its build steps and its `[CLICK]` cues.
- **Storyboard** — the whole deck at a glance: one column per topic (the topics
  of the deck's `o` overview), a time ribbon of the notes minutes per topic,
  and a card per slide: a live thumbnail (the deck page with `?si-view=thumb`,
  loaded lazily, reloaded after every save), layout, minutes, builds, and
  slides without a script. Click a card to go to its source.
- **Focus** — no navigator, the editor centred at a reading width, tabs faded,
  and the live slide floating in a corner.

On narrow screens (under 900 px: tablet portrait, phone) the navigator becomes
a drawer behind the **Files** button, the preview and the inspector stack under
the editor, the top bar keeps its icons, and the modes put the teleprompter
under the deck.

## Rebuilding the UI

```bash
make studio-ui     # npm ci, vitest, esbuild into internal/assets/web/studio/assets/
make studio-e2e    # builds si, then runs the Playwright suite
npx playwright install chromium   # once, before the first studio-e2e
```

Commit the regenerated bundle with the source change.

## Documents, guides, books and sites

Studio reads a workspace the way `si check` and `si serve` do:

- **The tree** gives every source its kind: slides, lab, article, demo script,
  site page, or a page of a guide or book. A folder that holds a guide, book or
  site has its own icon and a badge, and its `_nav.md` / `_site.md` shows as a
  navigation file with the guide's title.
- **The check** of a page in a guide, book or site renders the whole folder,
  with that folder's settings (theme, fonts, watermark), and reports what
  `si check` reports, links and anchors between pages included. A site page is
  never checked as slides, so plugins and `---` rules are not problems there.
  Opening the `_nav.md` or `_site.md` checks the whole guide, book or site.
- **The preview** shows the live page. Only slides follow the cursor slide by
  slide; a document shows its title and kind instead, and **Play** becomes
  **Open**. A `_nav.md` or `_site.md` previews the first page (a site's
  `index.md`).

## Mind maps: the map editor

A mind map page opens in **Map** view (the bar above the editor switches to
**Text**); in any other file, a ```` ```mindmap ```` fence under the cursor
offers **Map** too (2D or 3D: the canvas page carries three.js, so the 2D / 3D
button works on any map; in 3D a click selects a pill and the keys work as in
2D, without drag-to-reparent). The canvas is the real runtime (`internal/assets/web/mindmap.js`)
plus `mindmap-edit.js`, served by `GET /_si/studio/api/mmcanvas?path=&line=`
in an iframe. Clicks, keys and drags there become ops sent to the UI with
`postMessage` (`si-mm-op`); the UI posts them with the editor's current text to
`POST /_si/studio/api/mindmap`, where `internal/mindmap` parses the Markdown,
applies the op and writes it back (front matter, comments and other text kept;
the examples round-trip byte for byte), and `render.MindmapEdit` returns the
maps to redraw. The new text goes into CodeMirror as one change, so Undo, the
autosave, conflicts and the Text view behave as for typing; text typed in Text
view redraws the map the same way. Nothing in this path writes files.

Panels: the selected topic (text, badge, what it opens, picture and alt text,
sticky, folded, a code topic's language and code) and the page or fence (every
mind map key or option, the maps of the file, Add a map). Topics of a map
inlined from another file are shown read-only. The status bar counts maps and
topics. Several topics can be selected (⌘/Shift-click); ⌘C asks
`/api/mindmap` for op `copy` (`internal/mindmap` `Branches`: the branches as a
Markdown list, nothing written) and puts it on the clipboard, ⌘X then sends
`delete-many`, and a paste event in the canvas sends `paste` with the
clipboard's list. The editor draws the map in 2D, also for a `view: 3d` page
(its preview pane shows 3D). Keys and drags: RULES.md §24, *Editing in the
Studio*.

## Themes: the Theme view

⌘K › *New theme* (File › New Theme… in the Mac app, `window.siStudio.newTheme`)
asks for a name and a built-in to start from (the public ones, with their
colours) and `POST /_si/studio/api/theme/new` creates `themes/<name>.css`
with the header `si theme: Name · extends <built-in>` and no tokens yet; the
file opens in the **Theme** view. Any `.css` file with that header or with
`--si-*` tokens on `:root` gets the Theme | Text bar; the choice is
remembered per file. The Theme view takes the deck preview's room, since it
draws its own.

- **Controls** (right): the name, *Starts from* (the built-in it extends, or
  nothing), then the token groups from `GET /_si/studio/api/theme/spec`
  (`assets.AllMeta`: group, kind, units; the per-element tokens are left out).
  Colours are a light | dark pair, each a picker and a text field; shadows a
  pair of fields; fonts a field with system stacks offered; plain lengths a
  slider in their own unit (em, rem, cqi, cqb, %, ch); enumerations a select.
  The base theme's value shows as the placeholder; a changed token has a dot
  and **Reset**.
- **Preview** (left): the specimen deck (`internal/assets/web/themes/specimen.md`:
  title, list, two columns, table and callout, a code window, big number,
  quote, mind map, QR code) rendered with the theme as it is being edited,
  light, dark or both, at desktop, phone, tablet or 4K size (the page is drawn
  at that real size and scaled down, so container units behave as on the
  device); ← → move through the slides.
- **Checks**: `themefile.Check` on every change, light and dark; a failing
  pair has **Fix**, which sets the text colour to the nearest lightness (in
  OKLab, same hue) that passes.
- **Use in this folder** (`POST theme/use`) writes `theme: themes/<name>.css`
  into the folder's `si.conf`.

Every change is an op: `POST /_si/studio/api/theme {path, text, op}` parses
the editor's text with `internal/themefile`, sets one token (checked: known
token, value of its kind, no `px`, nothing that leaves the declaration),
writes the file back (only the token blocks change; other rules are kept as
written) and answers with the text, the file's own and the effective values
(the built-in underneath), the checks and the preview page. The text goes into
CodeMirror as one change, so Undo, autosave and the Text view behave as for
typing. A picker or slider sends its last value after a short pause.

## History (git)

**History** in the top bar (or ⌘K → History) shows the workspace's git: the
changed files with their diff (a new file is all additions), a commit of the
ticked files with a message, and the open file's history (`git log --follow`,
the last 50 commits). Click a version to read it; **Restore this version** puts
its text into the editor as one ordinary edit (⌘Z takes it back; autosave writes
it). A folder that is not a repository offers **Start a repository here**
(`git init`); without git on the computer the panel says so.

The branch bar at the top switches branches (uncommitted changes go along;
git refuses a switch that would lose them), **New branch** makes one from
here and switches to it, and with a remote **Pull** (↓ how many commits
behind) and **Push** (↑ how many ahead). Pull is fast-forward only: when your
branch and the remote both moved, it says so and leaves the merge to you.
Push sets the upstream the first time (to `origin`, or the only remote). They
use your own git credentials (SSH agent or credential helper); si never asks
for a password.

The calls are `GET /api/git/status`, `git/diff?path=`, `git/log?path=`,
`git/show?path=&rev=`, `git/branches` and `POST git/commit`, `git/restore`,
`git/init`, `git/branch {name, create}`, `git/pull`, `git/push`
(`internal/studio/git.go`): git runs with fixed arguments and no shell, in the
workspace root; paths go through the same confinement as every file call
(nothing outside the workspace, no dot paths) and after `--`; a revision must
be a hex hash; a branch name is checked by `git check-ref-format` and never
starts with `-`. Commits use your own git identity.

## Review (comments from a review link)

`si serve . --review` (or `si play talk.md --review`) prints a keyed link
(the port + 2, any interface; `--review-addr` picks another) for people on
your network. It serves the decks and documents read-only with a **Comments**
button: a reviewer writes a name and a note on the current slide (or the
section at the top of the page) and sees the others'. With the comment box
open, clicking a paragraph, a list item, code, a figure or a heading pins the
note to that block (an amber outline shows the pick; "Comment on the whole
slide instead" undoes it); pinned notes show as numbered amber pins beside
their block, and the Review panel quotes the block. A pin is stored as the
block's position and a hash of its text (`slide:3/b2-1a2b3c`), so it falls back
to its slide or section when that block is rewritten. Comments are stored in
`.si/review/<page>.json` next to the sources. **Review** in the Studio's top
bar (or ⌘K → Review) lists the open file's comments live (SSE, also when the
review link runs in another si), with reply, resolve and reopen.

Add `--https` for a network you do not control: the review link is then
HTTPS with si's own certificate (a reviewer's device trusts it once, from the
`/ca` page on the same port, as for `--remote --https`), or `--https=tailscale`
with a certificate every browser already trusts.

Security, like `--remote`: a 128-bit key per run (cookie, `HttpOnly`,
`SameSite=Strict`); reviewers' POSTs must carry this listener's Origin; names
up to 60 and texts up to 2000 characters, at most 500 comments a page; pages are
shown with `textContent`, never as HTML; the review listener serves no dot
paths and none of play's endpoints (position, terminal, callout), and only the
author (loopback) can resolve. Plain HTTP: use a network you trust.

## First run: the start gallery and the tour

A workspace with nothing si can open shows the **start gallery**
(`web/studio/src/gallery.tsx`): eight starters (a talk, a course lab, a
tutorial, an article, a site, a 2D and a 3D mind map, a code show), each with a
live thumbnail. `GET /_si/studio/api/gallery/{kind}` writes the starter into a
temporary folder, renders it like any file and caches the page; nothing lands
in the workspace until **Create**, which goes through the usual starter path
(`POST /_si/studio/api/starter`, never overwriting). ⌘K → *Start gallery* and
the welcome screen open it any time.

After the first file the **tour** (`web/studio/src/tour.tsx`) runs once: six
stops (files, Write with Map / Window views, preview, Rehearse and Present,
Build, ⌘K), a ring around each part and a note beside it; Next / Back / Skip,
← → and Esc. Done or skipped is remembered (`localStorage`, `si-tour-done`);
⌘K → *Take the tour* replays it. `e2e/start.e2e.mjs` covers the empty
folder → gallery → 3D mind map → Map view → tour → skip path; the a11y check
audits both.

## Known limits

- The bundle is about 700 KB (240 KB gzipped); most of it is CodeMirror. About 120 KB is the HTML / JavaScript / CSS grammar Markdown highlighting pulls in for raw HTML; dropping it would need a custom Markdown language. Not worth it on a loopback app today.
- Build runs in the request; a very large workspace (or `--pdf`) keeps the dialog busy until it finishes.
- `make studio-e2e` needs `npx playwright install chromium` once.
