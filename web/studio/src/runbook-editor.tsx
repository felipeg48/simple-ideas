// The Visual view of a runbook JSON file. It never writes anything itself: every
// edit is a pure function of the runbook, handed to the editor, which applies it
// as one CodeMirror transaction (so undo, autosave, conflicts and stash all apply).
import type { JSX } from "preact";
import { useEffect, useMemo, useRef, useState } from "preact/hooks";
import { type RbCategory, type RbEntry, type Runbook, ops, parseRunbook, placeholderRe, placeholders, serializeRunbook, validate } from "./runbook";
import { CommandField } from "./placeholder-input";
import { Dialog, Icon } from "./navigator";

// codicon names the CACI runbook uses (the drawer renders them)
const ICONS = ["checklist", "cloud", "dashboard", "database", "graph", "list-tree", "mail", "pulse", "server-environment", "shield", "sync", "tools", "trash", "warning"];
const FOLD_OVER = 40; // more entries than this: categories start collapsed

type Edit = (f: (rb: Runbook) => Runbook) => void;
interface Props { text: string; edit: Edit; toJson(line: number, col: number): void; undo(redo: boolean): void }

const readOnlyCat = (c: RbCategory) => c.raw !== undefined || "entries" in c.extra;
const under = (path: string, base: string) => path === base || path.startsWith(base + ".");

/** chips renders a command with its {{placeholders}} as chips. */
export function chips(cmd: string): (string | JSX.Element)[] {
  const out: (string | JSX.Element)[] = [];
  let i = 0;
  for (const m of cmd.matchAll(new RegExp(placeholderRe.source, "g"))) {
    if (m.index! > i) out.push(cmd.slice(i, m.index));
    out.push(
      <span class="rb-ph" key={m.index} title={m[0]}>
        <b>{m[1]}</b>{m[2] !== undefined && <i>{m[2] || " "}</i>}
      </span>,
    );
    i = m.index! + m[0].length;
  }
  if (i < cmd.length) out.push(cmd.slice(i));
  return out;
}

export function RunbookEditor(p: Props) {
  useUndoKeys(p.undo);
  const parsed = useMemo(() => parseRunbook(p.text), [p.text]);
  if (!parsed.ok) {
    const msg = parsed.error.replace(/ in JSON at position \d+.*$/s, "").replace(/^JSON\.parse: /, "");
    return (
      <div class="rb-view"><div class="rb-inner">
        <div class="rb-error" role="alert">
          <Icon name="warn" />
          <div>
            <b>Fix the JSON to use the visual editor</b> — {msg} (line {parsed.line}, column {parsed.col})
            <p class="mute">The visual editor stays off until the file parses, so it never rewrites text it cannot read.</p>
          </div>
          <button class="btn btn-pri" onClick={() => p.toJson(parsed.line, parsed.col)}>Show in JSON</button>
        </div>
      </div></div>
    );
  }
  return <Visual rb={parsed.rb} text={p.text} edit={p.edit} toJson={p.toJson} />;
}

/** useUndoKeys sends ⌘Z / ⇧⌘Z (Ctrl on other systems) outside text fields to the document history. */
function useUndoKeys(undo: (redo: boolean) => void) {
  const fn = useRef(undo);
  fn.current = undo;
  useEffect(() => {
    const on = (e: KeyboardEvent) => {
      if (!(e.metaKey || e.ctrlKey) || e.altKey || e.key.toLowerCase() !== "z") return;
      const t = e.target as HTMLElement;
      if (t.closest?.("input, textarea, select, .cm-editor, .dialog")) return;
      e.preventDefault();
      fn.current(e.shiftKey);
    };
    document.addEventListener("keydown", on);
    return () => document.removeEventListener("keydown", on);
  }, []);
}

function Visual(p: { rb: Runbook; text: string; edit: Edit; toJson(line: number, col: number): void }) {
  const { rb, edit } = p;
  const total = rb.categories.reduce((n, c) => n + c.entries.length, 0);
  const [open, setOpen] = useState<Set<number>>(() => new Set(total > FOLD_OVER ? [0] : rb.categories.map((_, i) => i)));
  const [editing, setEditing] = useState<string | null>(null); // "ci:ei"
  const [q, setQ] = useState("");
  const [dialog, setDialog] = useState<JSX.Element | null>(null);
  const [focusNew, setFocusNew] = useState<string | null>(null);
  const drag = useRef<[number, number] | null>(null);
  const [dropAt, setDropAt] = useState<string | null>(null);

  const issues = useMemo(() => validate(rb), [rb]);
  const names = useMemo(() => [...new Set(rb.categories.flatMap((c) => c.entries.flatMap((e) => placeholders(e.command).map((x) => x.name))))].sort(), [rb]);
  const reformats = useMemo(() => serializeRunbook(rb) !== p.text, [rb, p.text]);
  const rootIssues = issues.filter((i) => !i.path.startsWith("categories["));

  const needle = q.trim().toLowerCase();
  const hit = (e: RbEntry) => !needle || [e.label, e.description ?? "", e.command].some((s) => s.toLowerCase().includes(needle));
  const toggle = (ci: number) => setOpen((s) => { const n = new Set(s); n.has(ci) ? n.delete(ci) : n.add(ci); return n; });
  const swapOpen = (a: number, b: number) => setOpen((s) => { const n = new Set(s); const x = n.has(a), y = n.has(b); x ? n.add(b) : n.delete(b); y ? n.add(a) : n.delete(a); return n; });
  const structural = (f: (rb: Runbook) => Runbook) => { setEditing(null); edit(f); };

  const confirm = (title: string, body: string, run: () => void) =>
    setDialog(<Dialog title={title} body={body} okLabel="Delete" danger onClose={() => setDialog(null)} ok={async () => run()} />);

  const moveEntry = (ci: number, ei: number, d: -1 | 1, label: string) => {
    const to = ei + d;
    if (to < 0 || to >= rb.categories[ci].entries.length) return;
    structural((r) => ops.moveEntry(r, [ci, ei], [ci, to]));
    requestAnimationFrame(() => document.querySelector<HTMLElement>(`button[aria-label="${CSS.escape(`Move ${label} ${d < 0 ? "up" : "down"}`)}"]`)?.focus());
  };
  const moveCat = (ci: number, d: -1 | 1) => {
    const to = ci + d;
    if (to < 0 || to >= rb.categories.length) return;
    structural((r) => ops.moveCategory(r, ci, to));
    swapOpen(ci, to);
  };
  const drop = (ci: number, ei: number, after: boolean) => {
    const from = drag.current;
    drag.current = null;
    setDropAt(null);
    if (!from) return;
    let to = ei + (after ? 1 : 0);
    if (from[0] === ci && from[1] < to) to--;
    if (from[0] === ci && from[1] === to) return;
    structural((r) => ops.moveEntry(r, from, [ci, to]));
    setOpen((s) => new Set(s).add(ci));
  };
  const addEntry = (ci: number) => {
    const c = rb.categories[ci];
    const target = c.entries[c.entries.length - 1]?.target;
    edit((r) => ops.addEntry(r, ci, { label: "", command: "", ...(target && { target }) }));
    setOpen((s) => new Set(s).add(ci));
    const k = `${ci}:${c.entries.length}`;
    setEditing(k);
    setFocusNew(k);
  };
  const addCategory = () => {
    const ci = rb.categories.length;
    edit((r) => ops.addCategory(r, "New category"));
    setOpen((s) => new Set(s).add(ci));
    setFocusNew(`cat:${ci}`);
  };

  // focus the label of a just-added entry or the name of a just-added category
  useEffect(() => {
    if (!focusNew) return;
    const el = document.querySelector<HTMLInputElement>(`[data-focus="${focusNew}"]`);
    if (el) { el.focus(); el.select(); setFocusNew(null); }
  });

  const shown = rb.categories.map((c, ci) => ({ c, ci, entries: c.entries.map((e, ei) => ({ e, ei })).filter(({ e }) => !needle || c.name.toLowerCase().includes(needle) || hit(e)) }))
    .filter(({ c, entries }) => !needle || entries.length > 0 || c.name.toLowerCase().includes(needle));
  const matches = shown.reduce((n, s) => n + s.entries.length, 0);

  return (
    <div class="rb-view" onKeyDown={(e) => { if (e.key === "Escape" && editing && !(e.target as Element).closest(".ph-pop")) setEditing(null); }}>
      <div class="rb-inner">
        <header class="rb-head">
          <input class="rb-title" aria-label="Runbook name" placeholder="Untitled runbook" value={rb.name ?? ""} spellcheck={false}
            onInput={(e) => { const v = e.currentTarget.value; edit((r) => ({ ...r, name: v || undefined })); }} />
          <input class="rb-sub" aria-label="Runbook description" placeholder="Add a description" value={rb.description ?? ""}
            onInput={(e) => { const v = e.currentTarget.value; edit((r) => ({ ...r, description: v || undefined })); }} />
          <div class="rb-meta">
            <span>{rb.categories.length} {rb.categories.length === 1 ? "category" : "categories"}</span>
            <span>{total} {total === 1 ? "entry" : "entries"}</span>
            {names.length > 0 && <span>{names.length} {names.length === 1 ? "placeholder" : "placeholders"}</span>}
            {issues.length > 0 && <span class="rb-meta-bad"><Icon name="warn" />{issues.length} {issues.length === 1 ? "problem" : "problems"}</span>}
          </div>
          {rootIssues.map((i) => <p class="rb-issue" key={i.path}><Icon name="warn" />{i.msg}</p>)}
          {reformats && (
            <p class="rb-note"><Icon name="braces" />This file is not in the standard layout (2-space indent, one key per line). The first visual edit rewrites it in that layout; the data stays the same.</p>
          )}
        </header>

        <div class="rb-tools">
          <label class="rb-filter">
            <Icon name="search" />
            <input type="search" aria-label="Filter entries" placeholder="Filter by label, description or command" value={q}
              onInput={(e) => setQ(e.currentTarget.value)} onKeyDown={(e) => e.key === "Escape" && setQ("")} />
            {needle && <span class="rb-filter-n">{matches} of {total}</span>}
          </label>
          <button class="btn" onClick={() => setOpen(new Set(rb.categories.map((_, i) => i)))}>Expand all</button>
          <button class="btn" onClick={() => setOpen(new Set())}>Collapse all</button>
        </div>

        {shown.map(({ c, ci, entries }) => {
          const isOpen = !!needle || open.has(ci);
          const ro = readOnlyCat(c);
          const label = c.raw !== undefined ? `category ${ci + 1}` : c.name || "Untitled category";
          const catIssues = issues.filter((i) => under(i.path, `categories[${ci}]`) && !i.path.includes(".entries["));
          return (
            <section class={"rb-cat" + (ro ? " ro" : "")} key={ci} aria-label={label}>
              <div class={"rb-cat-head" + (dropAt === `${ci}:end` ? " drop-end" : "")}
                onDragOver={(e) => { if (drag.current && !ro) { e.preventDefault(); setDropAt(`${ci}:end`); } }}
                onDragLeave={() => setDropAt(null)}
                onDrop={(e) => { e.preventDefault(); drop(ci, c.entries.length - 1, true); }}>
                <button class="rb-chev" aria-expanded={isOpen} aria-label={(isOpen ? "Collapse " : "Expand ") + label} onClick={() => toggle(ci)} disabled={ro}>
                  <Icon name="chevron" class={"chev" + (isOpen && !ro ? " open" : "")} />
                </button>
                {ro ? (
                  <span class="rb-cat-name ro">{c.raw !== undefined ? JSON.stringify(c.raw) : c.name || "Untitled category"}</span>
                ) : (
                  <>
                    <select class="rb-icon" aria-label={`Icon for ${label}`} value={c.icon ?? ""}
                      onChange={(e) => { const v = e.currentTarget.value; edit((r) => ({ ...r, categories: r.categories.map((x, i) => (i === ci ? { ...x, icon: v || undefined } : x)) })); }}>
                      <option value="">no icon</option>
                      {(c.icon && !ICONS.includes(c.icon) ? [c.icon, ...ICONS] : ICONS).map((n) => <option value={n} key={n}>{n}</option>)}
                    </select>
                    <input class="rb-cat-name" aria-label="Category name" value={c.name} placeholder="Untitled category" spellcheck={false} data-focus={`cat:${ci}`}
                      onInput={(e) => { const v = e.currentTarget.value; edit((r) => ops.renameCategory(r, ci, v)); }} />
                  </>
                )}
                <span class="rb-count" title={`${c.entries.length} entries`}>{c.entries.length}</span>
                <span class="rb-acts">
                  <button class="icon-btn" aria-label={`Move ${label} up`} title="Move up" disabled={ci === 0} onClick={() => moveCat(ci, -1)}><Icon name="up" /></button>
                  <button class="icon-btn" aria-label={`Move ${label} down`} title="Move down" disabled={ci === rb.categories.length - 1} onClick={() => moveCat(ci, 1)}><Icon name="down" /></button>
                  {!ro && <button class="icon-btn" aria-label={`Duplicate ${label}`} title="Duplicate" onClick={() => { structural((r) => ops.duplicateCategory(r, ci)); }}><Icon name="copy" /></button>}
                  <button class="icon-btn danger" aria-label={`Delete ${label}`} title="Delete"
                    onClick={() => confirm(`Delete “${label}”?`, c.entries.length ? `Its ${c.entries.length} ${c.entries.length === 1 ? "entry goes" : "entries go"} too. ⌘Z brings it back.` : "⌘Z brings it back.",
                      () => { structural((r) => ops.deleteCategory(r, ci)); setOpen((s) => new Set([...s].filter((i) => i !== ci).map((i) => (i > ci ? i - 1 : i)))); })}>
                    <Icon name="trash" />
                  </button>
                </span>
              </div>
              {catIssues.map((i) => <p class="rb-issue rb-pad" key={i.path}><Icon name="warn" />{i.msg}</p>)}
              {ro && (
                <div class="rb-ro">
                  This category is not in the shape the visual editor understands, so it is shown read-only.
                  <button class="btn" onClick={() => p.toJson(1, 1)}>Fix this in the JSON view</button>
                </div>
              )}
              {isOpen && !ro && (
                <>
                  <ul class="rb-entries">
                    {entries.map(({ e, ei }) => (
                      <Row key={ei} e={e} ci={ci} ei={ei} last={ei === c.entries.length - 1} names={names}
                        issues={issues.filter((i) => under(i.path, `categories[${ci}].entries[${ei}]`))}
                        editing={editing === `${ci}:${ei}`} drop={dropAt === `${ci}:${ei}:a` ? "after" : dropAt === `${ci}:${ei}:b` ? "before" : null}
                        onEdit={() => setEditing(editing === `${ci}:${ei}` ? null : `${ci}:${ei}`)}
                        onPatch={(patch) => edit((r) => ops.updateEntry(r, ci, ei, patch))}
                        onDuplicate={() => structural((r) => ops.duplicateEntry(r, ci, ei))}
                        onMove={(d) => moveEntry(ci, ei, d, e.label)}
                        onDelete={() => confirm(`Delete “${e.label || "Untitled entry"}”?`, "⌘Z brings it back.", () => structural((r) => ops.deleteEntry(r, ci, ei)))}
                        onDragStart={() => { drag.current = [ci, ei]; }}
                        onDragEnd={() => { drag.current = null; setDropAt(null); }}
                        onDragOver={(after) => setDropAt(`${ci}:${ei}:${after ? "a" : "b"}`)}
                        onDrop={(after) => drop(ci, ei, after)}
                        toJson={() => p.toJson(1, 1)} />
                    ))}
                  </ul>
                  {!needle && (
                    <button class="rb-add" onClick={() => addEntry(ci)}><Icon name="plus" />Entry</button>
                  )}
                </>
              )}
            </section>
          );
        })}
        {needle && shown.length === 0 && <p class="rb-empty">No entry matches “{q.trim()}”.</p>}
        {!needle && <button class="rb-add rb-add-cat" onClick={addCategory}><Icon name="plus" />Category</button>}
      </div>
      {dialog}
    </div>
  );
}

interface RowProps {
  e: RbEntry; ci: number; ei: number; last: boolean; names: string[]; issues: { path: string; msg: string }[]; editing: boolean; drop: "before" | "after" | null;
  onEdit(): void; onPatch(p: Partial<RbEntry>): void; onDuplicate(): void; onMove(d: -1 | 1): void; onDelete(): void;
  onDragStart(): void; onDragEnd(): void; onDragOver(after: boolean): void; onDrop(after: boolean): void; toJson(): void;
}

function Row(p: RowProps) {
  const { e } = p;
  const label = e.label || "Untitled entry";
  const lower = (ev: DragEvent) => { const r = (ev.currentTarget as HTMLElement).getBoundingClientRect(); return ev.clientY > r.top + r.height / 2; };
  const dnd = {
    onDragOver: (ev: DragEvent) => { ev.preventDefault(); p.onDragOver(lower(ev)); },
    onDrop: (ev: DragEvent) => { ev.preventDefault(); p.onDrop(lower(ev)); },
  };
  if (e.raw !== undefined) {
    return (
      <li class="rb-entry ro" {...dnd}>
        <span class="rb-grip" />
        <div class="rb-main">
          <code class="rb-cmd">{JSON.stringify(e.raw)}</code>
          <p class="rb-issue"><Icon name="warn" />This entry is not an object. <button class="link" onClick={p.toJson}>Fix this in the JSON view</button></p>
        </div>
        <span class="rb-acts">
          <button class="icon-btn danger" aria-label={`Delete entry ${p.ei + 1}`} title="Delete" onClick={p.onDelete}><Icon name="trash" /></button>
        </span>
      </li>
    );
  }
  const target = e.target ?? ("target" in e.extra ? String(e.extra.target) : "");
  const field = (k: string) => p.issues.filter((i) => i.path.endsWith("." + k));
  return (
    <>
      <li class={"rb-entry" + (p.editing ? " editing" : "") + (p.drop ? " drop-" + p.drop : "")} {...dnd}>
        <span class="rb-grip" draggable title="Drag to reorder" aria-hidden="true"
          onDragStart={(ev) => { ev.dataTransfer?.setData("text/plain", label); if (ev.dataTransfer) ev.dataTransfer.effectAllowed = "move"; p.onDragStart(); }}
          onDragEnd={p.onDragEnd}>
          <Icon name="grip" />
        </span>
        <div class="rb-main" onDblClick={p.onEdit}>
          <div class="rb-line">
            <span class={"rb-label" + (e.label ? "" : " empty")}>{e.label}</span>
            {!e.label && <span class="rb-label-ph">Untitled entry</span>}
            <span class={"rb-target t-" + (target || "none")} title={target ? `target: ${target}` : "no target: highlighted as bash"}>{target || "bash"}</span>
          </div>
          {e.description && <p class="rb-desc">{e.description}</p>}
          {e.command ? <code class="rb-cmd">{chips(e.command)}</code> : <code class="rb-cmd empty">no command yet</code>}
          {!p.editing && p.issues.map((i) => <p class="rb-issue" key={i.path + i.msg}><Icon name="warn" />{i.msg}</p>)}
        </div>
        <span class="rb-acts">
          <button class={"icon-btn" + (p.editing ? " on" : "")} aria-label={`Edit ${label}`} aria-expanded={p.editing} title="Edit" onClick={p.onEdit}><Icon name="pen" /></button>
          <button class="icon-btn" aria-label={`Duplicate ${label}`} title="Duplicate" onClick={p.onDuplicate}><Icon name="copy" /></button>
          <button class="icon-btn" aria-label={`Move ${label} up`} title="Move up" disabled={p.ei === 0} onClick={() => p.onMove(-1)}><Icon name="up" /></button>
          <button class="icon-btn" aria-label={`Move ${label} down`} title="Move down" disabled={p.last} onClick={() => p.onMove(1)}><Icon name="down" /></button>
          <button class="icon-btn danger" aria-label={`Delete ${label}`} title="Delete" onClick={p.onDelete}><Icon name="trash" /></button>
        </span>
      </li>
      {p.editing && (
        <li class="rb-form" aria-label={`Edit ${label}`}>
          <div class="rb-form-grid">
            <label class="rb-f">
              <span class="rb-flabel">Label</span>
              <input class={"field plain" + (field("label").length ? " invalid" : "")} aria-label="Label" value={e.label} spellcheck={false} data-focus={`${p.ci}:${p.ei}`}
                onInput={(ev) => p.onPatch({ label: ev.currentTarget.value })} />
              {field("label").map((i) => <span class="rb-issue" key={i.msg}><Icon name="warn" />{i.msg}</span>)}
            </label>
            <label class="rb-f">
              <span class="rb-flabel">Target</span>
              <select class="field" aria-label="Target" value={target}
                onChange={(ev) => { const v = ev.currentTarget.value; if (v === "bash" || v === "sql" || v === "") p.onPatch({ target: (v || undefined) as RbEntry["target"] }); }}>
                <option value="">default (bash)</option>
                <option value="bash">bash</option>
                <option value="sql">sql</option>
                {target && target !== "bash" && target !== "sql" && <option value={target}>{target} (unknown)</option>}
              </select>
              {field("target").map((i) => <span class="rb-issue" key={i.msg}><Icon name="warn" />{i.msg}</span>)}
            </label>
            <label class="rb-f wide">
              <span class="rb-flabel">Description <span class="mute">optional</span></span>
              <input class="field plain" aria-label="Description" value={e.description ?? ""}
                onInput={(ev) => { const v = ev.currentTarget.value; p.onPatch({ description: v || undefined }); }} />
            </label>
            <div class="rb-f wide">
              <CommandField value={e.command} names={p.names} invalid={field("command").length > 0} onChange={(v) => p.onPatch({ command: v })} />
              {field("command").map((i) => <span class="rb-issue" key={i.msg}><Icon name="warn" />{i.msg}</span>)}
            </div>
          </div>
          <div class="rb-form-foot">
            <span class="mute">Changes save as you type · <kbd>Esc</kbd> closes</span>
            <button class="btn btn-pri" onClick={p.onEdit}>Done</button>
          </div>
        </li>
      )}
    </>
  );
}
