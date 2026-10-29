# si Studio · Plan 2a: Runbook generator, editor and placeholders Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Authors create runbooks with a guided generator, edit them visually (add, edit, duplicate, delete, reorder) or as raw JSON, and use `{{name}}` / `{{name:default}}` placeholders that the `si play` / `si --web` terminal drawer asks for before typing a command.

**Architecture:** si core gains placeholder parsing in `internal/render/runbook.go` (each injected entry carries its `vars`; placeholders are highlighted) and a value prompt in the drawer (`internal/assets/web/term/term.js`). Studio gains a pure TypeScript runbook model (parse, validate, serialize, operations, placeholders), a visual editor that edits the same CodeMirror document (so autosave, etags, conflicts and unsaved-text restore apply unchanged), and a generator wizard that creates the JSON through the existing file API.

**Tech Stack:** Go (stdlib), TypeScript, Preact, CodeMirror 6, vitest, Playwright (headless e2e).

**Spec:** `docs/superpowers/specs/2026-10-28-si-studio-design.md` — sections "Runbooks: generator and editor (decided)" and "Creating things".

## Global Constraints

- Runbook file format stays the v0.7 shape `{ "categories": [ { "name", "icon"?, "entries": [ { "label", "description"?, "command", "target"? } ] } ] }`, plus optional top-level `"name"` and `"description"`. si must keep accepting every existing runbook unchanged (`labs/runbook.json` of the CACI course is the real-world fixture: 14 categories, 249 entries).
- Placeholder syntax, exact: `{{name}}` or `{{name:default}}`, name matching `[A-Za-z_][A-Za-z0-9_-]*`, default any text without `}`. Regex (Go and TS identical): `\{\{([A-Za-z_][A-Za-z0-9_-]*)(?::([^}]*))?\}\}`. Must NOT match `{{.Server.Version}}`, `{{json .}}`, `${HOME}`, `{print $1}`, `{a: .b}`.
- The visual editor never drops data: unknown keys on the root, categories and entries are preserved on round trip; invalid JSON disables the visual view with the parse error (line, column) and never rewrites the text.
- Serialization: 2-space indent, trailing newline, key order `name, description, categories` / `name, icon, entries` / `label, description, command, target`, unknown keys after the known ones in their original order.
- Visual edits change the CodeMirror document, so they go through the existing autosave (700 ms), etag/409 conflict banner and localStorage stash. No new write path.
- Drag-and-drop reorder must have a keyboard equivalent (move up/down buttons with aria labels).
- Dark and light themes first-class; headless visual QA only (never visible windows, never Playwright MCP tools).
- `go.mod` unchanged; gofmt, `go vet ./...`, `go test ./...`, vitest, `make studio-ui`, `make studio-e2e` green. Plain commit messages, no attribution. Every new rule documented in `docs/RULES.md` with an example.

## Review Focus

- A command that is not a placeholder but contains braces (docker `{{.Names}}`, `${VAR}`, awk, jq): no prompt, typed verbatim. (Task 1 `TestPlaceholdersIgnoreTemplatesAndShell`; Task 3 `placeholders.test.ts`.)
- A runbook with extra keys or the 249-entry CACI runbook opened, edited visually and saved: the diff is only the edited entry. (Task 3 `roundTripPreservesUnknownKeys`; Task 6 e2e `visual edit is a minimal diff`.)
- Typing broken JSON in the raw view, then switching to Visual: the visual view refuses with the error and the text is untouched. (Task 4 e2e.)
- Cancelling the drawer's placeholder prompt (Esc): nothing is typed into the shell. (Task 2 e2e.)
- Generator: the target file already exists → the wizard says so and does not overwrite. (Task 5.)

---

### Task 1: Placeholders in the core runbook renderer

**Files:** Modify `internal/render/runbook.go`, `internal/render/render_test.go` (or a new `runbook_test.go`), `docs/RULES.md` (runbook section).

**Interfaces — produces:**
- `type Placeholder struct{ Name, Default string }` and `func Placeholders(cmd string) []Placeholder` (first occurrence wins, document order, duplicates removed; a later default does not override the first).
- Each injected `SI_RUNBOOK` entry gains `"vars": [{"name","default"}]` (omitted when empty).
- The highlighted `html` wraps every placeholder occurrence in `<span class="si-rb-var">…</span>` (after highlighting, on the escaped text).
- `runbookFile` accepts optional top-level `name`, `description` (ignored by the drawer except `name` as the drawer title when present).

- [ ] Write failing tests: `TestPlaceholdersParse` (`kubectl -n {{ns:messaging}} get pods {{pod}}` → [{ns messaging} {pod ""}]; repeat `{{ns}}` later → still one), `TestPlaceholdersIgnoreTemplatesAndShell` (the five non-matches of the Global Constraints → empty), `TestRunbookVarsAndHighlight` (injected JSON has `vars`, html contains `si-rb-var`), `TestRunbookAcceptsMetadata` (top-level name/description parse; CACI-shaped file without them still parses).
- [ ] Run `go test ./internal/render -run 'Placeholder|Runbook'` → FAIL.
- [ ] Implement.
- [ ] Run → PASS; `go test ./...` green.
- [ ] RULES.md: document placeholders (syntax, examples, what is not a placeholder, how the drawer asks) and optional metadata.
- [ ] Commit `feat: runbook placeholders {{name}} and {{name:default}}`.

---

### Task 2: Drawer asks for placeholder values

**Files:** Modify `internal/assets/web/term/term.js`; add an e2e scenario in `web/studio/e2e/studio.e2e.mjs` (a fixture deck with `runbook: rb.json` and an entry `echo hello-{{who:world}}`).

**Interfaces — consumes:** `entry.vars` from Task 1.

Behaviour: clicking an entry with `vars` opens an inline form inside the drawer (one field per var, prefilled with the last value used this session or the default; first field focused); Enter or "Type" types the substituted command, Shift/⌘+Enter or "Run" types and runs (`\r`), matching the existing click semantics; Esc or "Cancel" types nothing. Values persist in `sessionStorage` key `si-rb-vars` (name → value). Entries without vars behave exactly as today. Styles live in `term.js` (terminal contract exemption), dark.

- [ ] Write the e2e scenario `drawer asks for placeholders`: open the deck page through Studio's server, open the drawer, click the entry, see the form with `world` prefilled, change to `si`, Run → terminal output contains `hello-si`; click again → field prefilled `si`; Esc → nothing typed.
- [ ] Run `make build && make studio-e2e` → FAIL.
- [ ] Implement.
- [ ] Run → PASS; also verify `si play` (no `--web`) drawer still works by the existing serve terminal tests.
- [ ] Commit `feat: the runbook drawer asks for placeholder values`.

---

### Task 3: Runbook model (pure TypeScript)

**Files:** Create `web/studio/src/runbook.ts`, `web/studio/test/runbook.test.ts`, `web/studio/test/placeholders.test.ts`.

**Interfaces — produces:**
```ts
export interface RbEntry { label: string; description?: string; command: string; target?: "bash" | "sql"; extra: Record<string, unknown> }
export interface RbCategory { name: string; icon?: string; entries: RbEntry[]; extra: Record<string, unknown> }
export interface Runbook { name?: string; description?: string; categories: RbCategory[]; extra: Record<string, unknown> }
export type ParseResult = { ok: true; rb: Runbook } | { ok: false; error: string; line: number; col: number }
export function parseRunbook(text: string): ParseResult          // also rejects non-runbook JSON (no categories array) with a clear error
export function serializeRunbook(rb: Runbook): string            // Global Constraints order + 2-space + trailing newline
export function isRunbookText(text: string): boolean             // JSON object with a "categories" array
export function placeholders(cmd: string): { name: string; def: string }[]   // same rules as Go
export function insertPlaceholder(cmd: string, at: number, name: string, def?: string): { cmd: string; cursor: number }
export const ops: {
  addCategory(rb, name, at?), renameCategory(rb, ci, name), deleteCategory(rb, ci), duplicateCategory(rb, ci), moveCategory(rb, from, to),
  addEntry(rb, ci, e, at?), updateEntry(rb, ci, ei, patch), deleteEntry(rb, ci, ei), duplicateEntry(rb, ci, ei), moveEntry(rb, from: [ci, ei], to: [ci, ei]),
}  // all pure: return a new Runbook
export function validate(rb: Runbook): { path: string; msg: string }[]   // empty label/command, unknown target, duplicate category names
```
- [ ] Write failing tests: `roundTripPreservesUnknownKeys` (parse → serialize keeps extra keys and order), `serializeIsCanonical`, `parseReportsLineAndColumn`, `rejectsNonRunbookJson`, `opsArePure` (each op; duplicate inserts after the source with label suffix " (copy)"; move across categories), `validateFlags`, placeholders shared vectors (the five non-matches + three matches, identical to Task 1's), `insertPlaceholder` (cursor after the inserted token).
- [ ] Run `cd web/studio && npx vitest run` → FAIL.
- [ ] Implement.
- [ ] Run → PASS.
- [ ] Commit `feat(studio): runbook model with lossless round trip`.

---

### Task 4: Visual runbook editor

**Files:** Create `web/studio/src/runbook-editor.tsx`, `web/studio/src/placeholder-input.tsx`; modify `web/studio/src/editor.tsx` (view toggle for runbook JSON), `web/studio/src/styles.css`; rebuild bundle.

**Interfaces — consumes:** Task 3 model; the editor's per-document `EditorState` (visual edits dispatch one CodeMirror transaction replacing the whole text with `serializeRunbook(next)`, so undo works and autosave runs).

UI (match the Studio look, dark and light): when the open file is JSON and `isRunbookText`, the editor header shows a segmented control **Visual · JSON** (remember the choice per file in workspace-scoped localStorage). Visual view: runbook name/description at the top; categories as collapsible sections (name editable inline, icon select from a short list of VS Code codicon names already used by the CACI runbook, entry count); entries as rows with label, description, target badge, the command in monospace with placeholders shown as chips; row actions: edit (inline form), duplicate, delete (confirm with the existing Dialog), drag handle + move up/down buttons; "+ Entry" per category, "+ Category" at the end; a filter box. The command field is a single-line-wrapping CodeMirror (bash highlighting not required) or textarea with an **Insert placeholder** button: popover with name (autocomplete from names already used in the runbook), optional default, preview of the token. Validation messages from `validate()` appear inline. Invalid JSON → the Visual tab shows "Fix the JSON to use the visual editor — <error> (line L, column C)" with a button that switches to JSON and places the cursor there.

- [ ] Write failing e2e scenarios: `runbook visual edit is a minimal diff` (open the CACI-shaped fixture runbook, edit one entry's label in Visual, wait for save → the file differs from the original only in that line), `duplicate delete and reorder` (duplicate an entry → "(copy)" appears below; move it up with the keyboard button; delete with confirm → file back to original order minus copy), `invalid json disables visual` (break the JSON in the JSON view, switch to Visual → error with line/column, text unchanged on disk after 2 s), `insert placeholder` (insert `{{ns:messaging}}` via the popover → chip shown, JSON has the token).
- [ ] Run `make build && make studio-e2e` → FAIL.
- [ ] Implement; rebuild bundle (`make studio-ui`).
- [ ] Run → PASS; screenshots dark/light 1440×900 of the visual editor read and fixed.
- [ ] Commit `feat(studio): visual runbook editor with placeholders`.

---

### Task 5: Runbook generator (wizard)

**Files:** Create `web/studio/src/runbook-wizard.tsx`; modify `web/studio/src/navigator.tsx` (toolbar button "New runbook" + folder context-menu item), `web/studio/src/app.tsx` (open the created file in Visual view); rebuild bundle.

**Interfaces — consumes:** Task 3 model and `placeholder-input.tsx`, file API `create` (409 when the file exists), `decks`, `file` GET/PUT for the optional deck link.

Wizard steps (Dialog, keyboard friendly, Back/Next): (1) **Runbook**: name, description, location (folder picker from the tree, default the folder of the selected item) and file name (slug of the name + `.json`); (2) **Commands**: first category name, then entries (label, description, command with Insert placeholder, target bash/sql), "+ Add another", "+ New category"; (3) **Link** (optional): checkboxes of decks in the workspace; checked decks get `runbook: <relative path>` added to (or replacing) their front matter via GET+PUT with etag; (4) **Review**: the JSON preview (read-only, highlighted) and Create. On 409 for the runbook file: inline error "A file with this name already exists" and stay on step 1. After create: open the file in Visual view.

- [ ] Write failing e2e scenario `generator creates and links a runbook`: New runbook → name "Lab ops", one category "Health", entry "Alarms" `rabbitmq-diagnostics check_alarms -n {{node:rabbit@rmq1}}` → link to `talk.md` → Create → `lab-ops.json` exists with that content (canonical serialization), `talk.md` front matter has `runbook: lab-ops.json`, the Visual view is open; running the wizard again with the same name shows the exists error and does not overwrite.
- [ ] Run → FAIL. Implement; rebuild bundle. Run → PASS; screenshots dark/light read and fixed.
- [ ] Commit `feat(studio): runbook generator`.

---

### Task 6: Docs and version

**Files:** `docs/studio.md` (runbook editor, generator, placeholders), `docs/RULES.md` (if anything from Tasks 2/5 changes user-facing rules), `FEATURE.md`, `HANDOFF.md`, `README.md` if commands change, `cmd/si/main.go` version `0.9.0`.

- [ ] Write the docs; `make test` exit 0 (no pipes); `make studio-e2e` green.
- [ ] Commit `feat: Studio runbook generator and editor (v0.9.0)`. No tag, no push.
