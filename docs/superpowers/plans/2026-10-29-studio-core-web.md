# si Studio · Plan 1: Studio core and `si --web` (Write basics) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `si --web [dir]` opens a secure, local, browser-based Studio where an author browses the workspace, edits Markdown/AsciiDoc with si-aware highlighting, completion and diagnostics, autosaves safely, and sees the real renderer follow the cursor.

**Architecture:** A new stdlib-only Go package `internal/studio` composes the existing `serve.Server` (decks, SSE reload, terminal) with a JSON API (workspace tree, files, decks, check, rules) and an embedded single-page app, all behind one auth gateway. The SPA (Preact + TypeScript + CodeMirror 6) lives in `web/studio/`, is bundled by esbuild into `internal/assets/web/studio/` and committed, so `go build` never needs Node.

**Tech Stack:** Go (stdlib, existing goldmark + creack/pty only), Preact, TypeScript, CodeMirror 6, esbuild, vitest (unit), Playwright (headless e2e, dev only).

**Spec:** `docs/superpowers/specs/2026-10-28-si-studio-design.md`

This is plan 1 of 5. Later plans: (2) Write extras: inspector, Storyboard + time ribbon, Focus, ⌘K, creating things, Build/PDF jobs; (3) Rehearse, Present and Lab modes; (4) macOS desktop (Wails); (5) Windows and Linux desktop + ConPTY. v1.0.0 is tagged only after plan 4 (macOS + web); plan 1 ships as v0.8.0.

## Global Constraints

- The `si` core module adds **no new Go dependencies** (stdlib, goldmark, creack/pty only). `go.mod` must not change.
- Front-end dependencies are bundled and the bundle is **committed**; `go build ./...` and `go test ./...` work without Node. `make studio-ui` (needs Node 20+) rebuilds the bundle.
- `si --web` listens on **loopback only** in v1; a non-loopback `--addr` is refused with exit code 2 and the message `si --web is loopback-only in this version`.
- **Every** route served by `si --web` (API, deck pages, static workspace files, terminal) passes the auth gateway; only `/_si/studio/assets/*` and the login exchange are reachable without the session cookie.
- Nothing from Studio ever appears in `si build` output: `TestGolden` and `TestLayoutHasNoFixedCanvas` stay green, `internal/render` is unchanged by this plan except where a task says so.
- Studio UI has **first-class dark and light themes**, following `prefers-color-scheme` by default, with a toggle persisted in `localStorage` key `si-studio-theme` (`"dark" | "light" | "system"`).
- The preview is the real renderer (`serve.Server` with `Dev: true`), never an approximation.
- Never silently overwrite a file changed outside Studio; never lose unsaved text on reload.
- `gofmt`, `go vet ./...`, `go test ./...` green before every commit; `npm test` green in `web/studio` for front-end tasks. Plain commit messages, no attribution trailers.
- Visual QA is headless only (Chrome `--headless=new` or a headless Playwright script), at 1440×900 and 1280×800, dark and light. Never the Playwright MCP browser tools.
- Open source, Apache 2.0 (the repository licence); no signing.

## Review Focus

- A file edited in another editor while its Studio tab has unsaved changes: Studio must show a conflict and keep both versions, never overwrite silently. (Task 3 `TestWriteConflict`; Task 7 e2e `conflict banner`.)
- A path like `../../etc/passwd`, an absolute path, or a symlink pointing outside the workspace: every file API refuses it with 403. (Task 2 `TestResolveRejectsEscapes`.)
- Another website open in the browser, or a DNS-rebinding page, calling the API: rejected by Host/Origin checks even with no token mistake. (Task 1 `TestGatewayRejectsForeignHost`, `TestGatewayRejectsForeignOrigin`.)
- Opening an image, a PDF or a 50 MB file from the navigator: the editor must not load it; the API answers 415 for binary and 413 above 5 MiB; the UI shows a preview or a message. (Task 3 `TestReadRefusesBinaryAndHuge`; Task 6.)
- Reloading the page (or a crash) with unsaved edits: the text comes back from `localStorage` and is offered for saving. (Task 6 unit `restoresDirtyBuffer`; Task 7 e2e `reload keeps unsaved text`.)

---

## File structure

```
internal/studio/
  gateway.go        auth gateway: token, cookie, Host/Origin checks
  workspace.go      root, safe path resolution, tree with si-aware kinds
  files.go          file read/write/create/rename/delete handlers (etag, conflicts)
  decks.go          decks in the workspace and which files each deck includes
  check.go          diagnostics + slide map for a deck
  rules.go          directive names, layouts, transitions, front-matter keys
  studio.go         Studio: composes serve.Server + API + assets behind the gateway
  *_test.go
internal/assets/web/studio/   built SPA (index.html, app.js, app.css) — committed
internal/assets/assets.go     + StudioFS embed
cmd/si/main.go                + --web flag, studio() command
web/studio/                   SPA source (package.json, tsconfig, build.mjs, src/, test/, e2e/)
docs/studio.md                how Studio works + the rule exceptions
Makefile                      + studio-ui target
```

---

### Task 1: Auth gateway

**Files:**
- Create: `internal/studio/gateway.go`, `internal/studio/gateway_test.go`

**Interfaces:**
- Produces: `type Gateway struct{...}`; `func NewGateway(token string, hosts []string) *Gateway` (hosts like `127.0.0.1:7070`, `localhost:7070`); `func (g *Gateway) Wrap(next http.Handler) http.Handler`; `const CookieName = "si_studio"`; `func NewToken() string` (32 random bytes, hex).

Rules the gateway enforces, in this order:
1. `Host` header not in `hosts` → 403 (DNS rebinding).
2. Path under `/_si/studio/assets/` → pass through (static UI, no secrets).
3. Query `token=<t>` equal to the token (constant-time compare) → set cookie `si_studio=<t>; Path=/; HttpOnly; SameSite=Strict`, then 303 redirect to the same URL without `token`.
4. Missing or wrong cookie → 401.
5. Non-GET/HEAD request whose `Origin` header is present and not `http://<host>` → 403.
6. Otherwise pass to `next`.

- [ ] **Step 1: Write the failing tests**

```go
func TestGatewayRejectsForeignHost(t *testing.T)   // Host evil.com:7070 with valid cookie → 403
func TestGatewayLoginSetsCookieAndRedirects(t *testing.T) // GET /?token=T → 303 to "/", Set-Cookie si_studio=T, HttpOnly, SameSite=Strict
func TestGatewayRequiresCookie(t *testing.T)       // GET /_si/studio/api/tree no cookie → 401; wrong cookie → 401; right cookie → 200
func TestGatewayRejectsForeignOrigin(t *testing.T) // PUT with Origin http://evil.com → 403; Origin http://127.0.0.1:7070 → 200
func TestGatewayAssetsArePublic(t *testing.T)      // GET /_si/studio/assets/app.js no cookie → 200
func TestNewTokenIsRandom(t *testing.T)            // two tokens differ, len 64 hex
```
Use `httptest.NewRequest` with explicit `Host` and a `next` that writes 200.

- [ ] **Step 2: Run `go test ./internal/studio -run Gateway` — expect FAIL (undefined: NewGateway).**
- [ ] **Step 3: Implement `gateway.go`** (`crypto/subtle.ConstantTimeCompare`, `crypto/rand`).
- [ ] **Step 4: Run `go test ./internal/studio -run 'Gateway|Token'` — expect PASS.**
- [ ] **Step 5: Commit** `feat(studio): auth gateway for si --web`

---

### Task 2: Workspace and safe paths

**Files:**
- Create: `internal/studio/workspace.go`, `internal/studio/workspace_test.go`

**Interfaces:**
- Produces:
  - `type Workspace struct{ Root string }` (absolute, symlinks evaluated); `func OpenWorkspace(dir string) (*Workspace, error)`.
  - `func (w *Workspace) Resolve(rel string) (string, error)` → absolute path inside Root, or `ErrOutside` for `..` escapes, absolute input, or symlinks resolving outside Root (resolve with `filepath.EvalSymlinks` on the existing prefix).
  - `func (w *Workspace) Rel(abs string) string` (slash-separated).
  - `type Node struct{ Name, Path, Kind string; Children []Node }` with `json:"name"`, `json:"path"`, `json:"kind"`, `json:"children,omitempty"`.
  - `func (w *Workspace) Tree() (Node, error)`; Kind values: `dir`, `deck` (in `deck.List(dir)` for its directory), `markdown` (other `.md`/`.adoc`), `svg`, `css`, `json`, `image` (`.png .jpg .jpeg .gif .webp`), `other`. Skips names starting with `.` and directories `node_modules`, `html`, `slides`. Directories first, then files, each alphabetical.

- [ ] **Step 1: Write the failing tests**

```go
func TestResolveRejectsEscapes(t *testing.T)
// "../x", "/etc/passwd", "a/../../x" → ErrOutside; a symlink "out" → t.TempDir() elsewhere, Resolve("out/f") → ErrOutside; "decks/a.md" → Root/decks/a.md
func TestTreeKinds(t *testing.T)
// fixture: talk.md (# T), sections/one.md, img/p.png, theme.css, rb.json, .git/x, node_modules/y
// → talk.md kind "deck", sections/one.md "markdown", p.png "image", theme.css "css", rb.json "json"; .git and node_modules absent; dirs listed before files
```
- [ ] **Step 2: Run `go test ./internal/studio -run 'Resolve|Tree'` — expect FAIL.**
- [ ] **Step 3: Implement `workspace.go`.**
- [ ] **Step 4: Run the same — expect PASS.**
- [ ] **Step 5: Commit** `feat(studio): workspace tree and safe path resolution`

---

### Task 3: File API (read, write with etag, create, rename, delete)

**Files:**
- Create: `internal/studio/files.go`, `internal/studio/files_test.go`

**Interfaces:**
- Consumes: `Workspace.Resolve`, `Workspace.Rel` (Task 2).
- Produces handlers mounted by Task 5 under `/_si/studio/api/`:
  - `GET file?path=P` → 200 `{"path","content","etag"}`; 404 missing; 415 `{"error":"binary"}` when the first 8 KiB contain NUL or are not valid UTF-8; 413 above 5 MiB.
  - `PUT file?path=P` body `{"content","etag"}` → 200 `{"etag"}`; 409 `{"error":"conflict","etag","content"}` when the disk etag differs from `etag` (empty `etag` means "create only if missing", 409 if it exists). Writes atomically (temp file in the same dir + `os.Rename`), keeps file mode.
  - `POST create` `{"path","content","dir":bool}` → 201 `{"path","etag"}` / 409 exists; creates parent dirs.
  - `POST rename` `{"from","to"}` → 200 / 409 if `to` exists / 404.
  - `DELETE file?path=P` → 204; directories only if empty (409 otherwise).
  - `etag` = first 16 hex chars of SHA-256 of the file bytes.
  - All paths through `Resolve`; `ErrOutside` → 403.
- `func (a *api) routes(mux *http.ServeMux)` registers these (shared `api` struct holds the `*Workspace`).

- [ ] **Step 1: Write the failing tests** (table-driven, `httptest`):

```go
func TestReadWriteRoundTrip(t *testing.T)        // GET → PUT with etag → GET returns new content + new etag
func TestWriteConflict(t *testing.T)             // GET etag, change file on disk, PUT old etag → 409 with disk content; file unchanged
func TestReadRefusesBinaryAndHuge(t *testing.T)  // png bytes → 415; 6 MiB text → 413
func TestCreateRenameDelete(t *testing.T)        // create a/b.md (parents made) → 201; create again → 409; rename → ok; rename onto existing → 409; delete → 204; delete non-empty dir → 409
func TestFileAPIRejectsEscape(t *testing.T)      // GET file?path=../x → 403
```
- [ ] **Step 2: Run `go test ./internal/studio -run 'ReadWrite|Conflict|Binary|Create|Escape'` — expect FAIL.**
- [ ] **Step 3: Implement `files.go`.**
- [ ] **Step 4: Run the same — expect PASS.**
- [ ] **Step 5: Commit** `feat(studio): file API with etags and conflict detection`

---

### Task 4: Decks, check and rules API

**Files:**
- Create: `internal/studio/decks.go`, `internal/studio/check.go`, `internal/studio/rules.go`, `internal/studio/api_test.go`

**Interfaces:**
- Consumes: `deck.List`, `deck.Parse`, `render.Deck`, `render.Topics`, `deck.Directives`, `deck.Layouts`, `deck.Transitions`, `deck.MetaKeys`.
- Produces routes:
  - `GET decks` → `[{"path","title","files":[rel paths of every source file its slides come from, main first]}]` for every deck in the workspace (walk dirs; `deck.List` per dir).
  - `GET check?deck=P` → `{"problems":[{"file","line","msg"}],"slides":[{"index":1-based,"file","line","title","topic","minutes"}],"minutes","steps","canvases"}`; `file` fields are workspace-relative; problems come from `render.Deck(...).Problems` (render with `render.Options{ShowProblems: true}`), slides from `d.Slides` + `render.Topics(d)`.
  - `GET rules` → `{"directives":{name:scope},"layouts":[...],"transitions":[...],"metaKeys":[...]}`.

- [ ] **Step 1: Write the failing tests**

```go
func TestDecksListsIncludedFiles(t *testing.T)  // main.md includes sections/a.md → files ["main.md","sections/a.md"]
func TestCheckReportsProblemsAndSlides(t *testing.T) // deck with <!-- layout: nope --> → one problem with file+line; slides[1].file=="sections/a.md", topic 1
func TestRulesMatchDeckPackage(t *testing.T)    // rules.layouts == deck.Layouts; every deck.Directives key present
```
- [ ] **Step 2: Run `go test ./internal/studio -run 'Decks|Check|Rules'` — expect FAIL.**
- [ ] **Step 3: Implement the three files.**
- [ ] **Step 4: Run the same — expect PASS.**
- [ ] **Step 5: Commit** `feat(studio): decks, check and rules API`

---

### Task 5: Studio server and `si --web`

**Files:**
- Create: `internal/studio/studio.go`, `internal/studio/studio_test.go`
- Modify: `internal/assets/assets.go` (add `//go:embed web/studio` → `var StudioFS embed.FS` subtree `web/studio`), `cmd/si/main.go` (flag + command + usage text), `cmd/si/main_test.go`
- Create placeholder: `internal/assets/web/studio/index.html` (minimal page, replaced by Task 6's build)

**Interfaces:**
- Consumes: `NewGateway`, `NewToken` (Task 1), `OpenWorkspace` (Task 2), `api.routes` (Tasks 3–4), `serve.New`, `serve.Server.{Handler,Watch,CloseTerminals,Dev,Terminal,Title}`.
- Produces:
  - `type Options struct{ CSS []string; Theme, Conf, Title, Token, Addr string }`.
  - `func New(dir string, opt Options) (*Studio, error)`; `func (s *Studio) Handler() http.Handler`; `func (s *Studio) Watch(stop <-chan struct{})`; `func (s *Studio) Close()`.
  - Routes (all wrapped by the gateway built from `opt.Token` and hosts `127.0.0.1:<port>`, `localhost:<port>` derived from `opt.Addr`): `/_si/studio/` → SPA `index.html`; `/_si/studio/assets/*` → `StudioFS`; `/_si/studio/api/*` → Tasks 3–4; everything else → the embedded `serve.Server` (Dev: true, Terminal: true, its `/_si/events`). `GET /` redirects to `/_si/studio/`.
  - `web()` calls `net.Listen` first and passes `ln.Addr().String()` as `Options.Addr`, so `--addr 127.0.0.1:0` (used by the e2e test) gets the real port in the gateway's allowed hosts.
  - CLI: `si --web [dir]` (dir defaults to `.`), honours `--addr` (default `127.0.0.1:7070`), `--css`, `--theme`, `--conf`, `--site`; opens the browser at `http://<addr>/?token=<t>` unless `--no-open`; prints that URL. Non-loopback `--addr` → exit 2, `si --web is loopback-only in this version`.

- [ ] **Step 1: Write the failing tests**

```go
// studio_test.go
func TestStudioServesUIAndDecksBehindGateway(t *testing.T)
// New(fixture, Options{Token:"T", Addr:"127.0.0.1:7070"}); requests with Host 127.0.0.1:7070:
// GET /_si/studio/ no cookie → 401; with cookie → 200 contains "<div id=\"app\""
// GET /talk.html with cookie → 200 contains "si-slide"; GET /_si/studio/api/tree with cookie → 200 JSON root
// main_test.go
func TestWebRefusesNonLoopback(t *testing.T) // run([]string{"--web","--addr","0.0.0.0:0","--no-open","."}) → 2, stderr contains "loopback-only"
func TestWebFlagParses(t *testing.T)         // parseArgs("--web","decks") → o.web true, args ["decks"]
```
- [ ] **Step 2: Run `go test ./internal/studio ./cmd/si -run 'Studio|Web'` — expect FAIL.**
- [ ] **Step 3: Implement** `studio.go`, the embed, `--web`/`--no-open` flags in `parseArgs`, a `web(src string, o *options, stdout, stderr io.Writer) int` beside `play` (same listen/shutdown pattern, Ctrl+C closes terminals), and the usage line `si --web [dir] [--addr 127.0.0.1:7070] [--no-open]   the Studio: edit, preview and check decks in the browser (loopback only)`.
- [ ] **Step 4: Run `go test ./...` — expect PASS; `go vet ./...` clean.**
- [ ] **Step 5: Commit** `feat: si --web serves Studio behind the auth gateway`

---

### Task 6: SPA shell, navigator, editor, preview, autosave (front-end)

**Files:**
- Create: `web/studio/package.json`, `tsconfig.json`, `build.mjs`, `vitest.config.ts`, `src/main.tsx`, `src/api.ts`, `src/theme.ts`, `src/app.tsx`, `src/navigator.tsx`, `src/editor.tsx`, `src/si-lang.ts`, `src/preview.tsx`, `src/problems.tsx`, `src/statusbar.tsx`, `src/autosave.ts`, `src/buffers.ts`, `src/slidemap.ts`, `src/styles.css`, `test/*.test.ts`
- Output (committed): `internal/assets/web/studio/{index.html,app.js,app.css}`
- Modify: `Makefile` (target `studio-ui: cd web/studio && npm ci && npm test && npm run build`)

**Interfaces:**
- Consumes: the API of Tasks 3–5 (paths under `/_si/studio/api/`), `/_si/events` SSE (`reload` events) and deck pages `/<deck>.html#N`.
- Produces (pure, unit-tested modules):
  - `slidemap.ts`: `slideForCursor(slides: Slide[], file: string, line: number): number | null` — the last slide with `slide.file === file && slide.line <= line`.
  - `autosave.ts`: `createAutosave(save: () => Promise<void>, delayMs = 700): { touch(): void; flush(): Promise<void> }`.
  - `buffers.ts`: `stash(path: string, text: string, etag: string)`, `restore(path: string): {text, etag} | null`, `clear(path: string)` over `localStorage` key `si-studio-buffer:<path>`.
  - `si-lang.ts`: CodeMirror extension `siMarkdown(rules: Rules)` = markdown language + a decoration that styles every `<!-- name: value -->` line whose `name` is in `rules.directives` as a chip (class `cm-si-chip`), and completion after `<!-- ` for directive names, after `layout: ` for layouts, after `transition: ` for transitions, and front-matter keys inside the leading `---` block.
- UI behaviour:
  - Layout (spec "Write", mockup A): top bar (app name, workspace name, mode switch showing **Write**, theme toggle), left navigator (Task 2 tree, si-aware icons by `kind`, context menu New file / New folder / Rename / Delete via Task 3), centre editor with tabs, right preview iframe, bottom problems panel, status bar (`si check` summary: decks OK, minutes, steps, canvases; cursor Ln/Col).
  - Opening a `kind` `image` shows the image (`/<path>` from the workspace server); 413/415 shows a message, never loads the editor.
  - Editor: `siMarkdown` + lint diagnostics from `check` problems for the open file; dirty buffers stashed on every change, cleared after a successful save; on load, a stashed buffer newer than disk is restored with a banner "Unsaved changes restored".
  - Autosave 700 ms after the last keystroke and on tab switch/blur; PUT with the last etag; on 409 show a conflict banner with "Keep mine" (re-PUT with the returned etag) and "Use disk version".
  - Preview: for the open file, pick the first deck from `decks` whose `files` contains it; iframe `src=/<deck>.html`; on cursor move set the iframe hash to `#<slideForCursor(...)>`; a `reload` SSE event refreshes tree, check and (if not dirty) the open file.
  - Themes: CSS custom properties for dark (mockup palette: `#0b1019` background, `#ff7b33` accent, `#6aa8f0` second accent) and a light palette of equal care; `theme.ts` exports `applyTheme(mode: "dark"|"light"|"system")` and persists `si-studio-theme`.

- [ ] **Step 1: Write failing unit tests** in `web/studio/test/`:

```ts
// slidemap.test.ts
it("maps a cursor line to the slide that starts at or before it", ...) // slides at a.md:1, sections/s.md:1, sections/s.md:9 ; (sections/s.md, 12) → 3 ; (a.md, 0) → null
// autosave.test.ts
it("saves once after the delay, flush saves immediately", ...) // fake timers
// buffers.test.ts
it("restoresDirtyBuffer", ...) // stash then restore returns text+etag; clear removes it
// si-lang.test.ts
it("completes directive names after <!-- ", ...) // completion list contains "layout", "steps", "svg"
```
- [ ] **Step 2: Run `cd web/studio && npm test` — expect FAIL.**
- [ ] **Step 3: Implement the modules and components; `build.mjs` bundles `src/main.tsx` with esbuild (minified, ES2020) to `internal/assets/web/studio/app.js` + `app.css` and writes `index.html` with `<div id="app">`.**
- [ ] **Step 4: Run `npm test` — expect PASS; run `make studio-ui`; run `go test ./...` — expect PASS.**
- [ ] **Step 5: Commit** `feat(studio): Write mode UI: navigator, editor, live preview, autosave` (source + built bundle).

---

### Task 7: Headless end-to-end and visual QA

**Files:**
- Create: `web/studio/e2e/studio.e2e.mjs` (Playwright, headless Chromium), `web/studio/e2e/fixture/` (a small split deck: `talk.md` with front matter + `<!-- include: sections/one.md -->`, `sections/one.md`, an `img/p.png`)
- Modify: `web/studio/package.json` (script `e2e`), `Makefile` (target `studio-e2e`)

**Interfaces:**
- Consumes: the `si` binary built by `make build`, run as `bin/si --web e2e/fixture --addr 127.0.0.1:0 --no-open` (the test reads the printed URL).

- [ ] **Step 1: Write the e2e scenarios** (each a named test):
  - `edit updates preview`: open `sections/one.md`, change its `##` heading text, wait ≤ 3 s, the preview iframe contains the new heading.
  - `cursor follows slide`: put the cursor on the second slide's heading → iframe URL hash `#3`.
  - `conflict banner`: type in the editor, write the same file on disk from the test, wait for save → conflict banner visible; "Keep mine" → disk has the editor text.
  - `reload keeps unsaved text`: type, reload before 700 ms → "Unsaved changes restored" banner and the text present.
  - `diagnostic shows`: add `<!-- layout: nope -->` → a lint marker on that line and an entry in the problems panel.
  - `image opens as image`: click `img/p.png` → an `<img>` is shown, no editor.
  - `screenshots`: dark and light at 1440×900 and 1280×800 written to `web/studio/e2e/out/` (git-ignored).
- [ ] **Step 2: Run `make build && cd web/studio && npm run e2e` — fix until all pass.**
- [ ] **Step 3: Open the screenshots (Read tool) and fix any overlap, clipped text or unreadable contrast in either theme; re-run.**
- [ ] **Step 4: Commit** `test(studio): headless end-to-end and visual QA` (add `web/studio/e2e/out/` and `web/studio/node_modules/` to `.gitignore`).

---

### Task 8: Docs, rule exceptions, release v0.8.0

**Files:**
- Create: `docs/studio.md` (what `si --web` is, how it is wired — table like `docs/terminal.md`, security model, the rule #7/#2 exceptions for vendored front-end assets and Studio-only serving, how to rebuild the UI)
- Modify: `docs/RULES.md` (§14: `si --web`), `README.md` (commands table row), `FEATURE.md` (Shipped — v0.8.0), `CLAUDE.md` (layout table rows for `internal/studio/`, `web/studio/`, and the Studio exception next to the terminal one), `HANDOFF.md` (current state, next up: plan 2), `cmd/si/main.go` (`const version = "0.8.0"`)

- [ ] **Step 1: Write the docs.**
- [ ] **Step 2: Run `make test`; check its real exit code (no pipes) — expect 0.**
- [ ] **Step 3: Commit** `feat: si --web Studio, Write mode (v0.8.0)`; tag `v0.8.0` only when the founder asks to release.
