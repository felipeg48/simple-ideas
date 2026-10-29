# Studio — `si --web` browser editor (shipped, v0.8.0)

`si --web [dir]` opens a browser editor for a folder of decks: a file tree, a
CodeMirror markdown editor, a live slide preview (the same `si play` renderer,
in an iframe) and a `si check` problems panel. This release is **Write mode**
only; the design is in `docs/superpowers/specs/2026-10-28-si-studio-design.md`.

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
  cookie named `si_studio` and every other request needs that cookie.
- **Host allowlist.** Only `127.0.0.1:PORT` and `localhost:PORT` are accepted,
  which blocks DNS rebinding.
- **Origin check** on every non-GET request.
- **Public surface:** only `/_si/studio/assets/*` (clean paths) is reachable
  without the cookie.
- **File API confinement.** Paths that escape the workspace (symlinks included)
  and any path with a dot-segment (`.git`, `.env`, `..`) are refused.
- **Atomic writes.** Saves write a temp file and rename it into place.
- **Etag conflicts never overwrite.** `PUT` carries the etag the editor loaded;
  if the file changed on disk the server answers with a conflict and writes
  nothing.

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

## Rebuilding the UI

```bash
make studio-ui     # npm ci, vitest, esbuild into internal/assets/web/studio/assets/
make studio-e2e    # builds si, then runs the Playwright suite
npx playwright install chromium   # once, before the first studio-e2e
```

Commit the regenerated bundle with the source change.

## Known limits

- Desktop-only layout; it is not responsive to phone widths.
- The bundle is about 643 KB (`app.js`).
- Rehearse, Present, Lab, the inspector, the storyboard, the command palette
  (Cmd-K) and the runbook editor/generator arrive in later plans.
- `make studio-e2e` needs `npx playwright install chromium` once.
