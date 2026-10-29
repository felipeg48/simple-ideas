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

## Rebuilding the UI

```bash
make studio-ui     # npm ci, vitest, esbuild into internal/assets/web/studio/assets/
make studio-e2e    # builds si, then runs the Playwright suite
npx playwright install chromium   # once, before the first studio-e2e
```

Commit the regenerated bundle with the source change.

## Known limits

- Desktop-only layout; it is not responsive to phone widths.
- The bundle is about 661 KB (`app.js`).
- Rehearse, Present, Lab, the inspector, the storyboard and the command palette
  (Cmd-K) arrive in later plans.
- `make studio-e2e` needs `npx playwright install chromium` once.
