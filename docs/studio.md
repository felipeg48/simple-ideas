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

## Known limits

- Desktop-only layout; it is not responsive to phone widths.
- Build runs in the request; a very large workspace (or `--pdf`) keeps the dialog busy until it finishes.
- The bundle is about 661 KB (`app.js`).
- The inspector, the storyboard and focus mode arrive in later plans.
- `make studio-e2e` needs `npx playwright install chromium` once.
