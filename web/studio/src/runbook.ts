// Pure runbook model: lossless parse/serialize, edit ops, placeholders.
// order: the object's keys as they were in the file; serialization keeps it so an edit
// only touches the edited value (objects created in Studio get the canonical order)
export interface RbEntry { label: string; description?: string; command: string; target?: "bash" | "sql"; extra: Record<string, unknown>; missing?: string[]; raw?: unknown; order?: string[] }
export interface RbCategory { name: string; icon?: string; entries: RbEntry[]; extra: Record<string, unknown>; missing?: string[]; raw?: unknown; order?: string[] }
export interface Runbook { name?: string; description?: string; categories: RbCategory[]; extra: Record<string, unknown>; order?: string[] }
export type ParseResult = { ok: true; rb: Runbook } | { ok: false; error: string; line: number; col: number };

const isObj = (v: unknown): v is Record<string, unknown> => typeof v === "object" && v !== null && !Array.isArray(v);

// split o into known keys (right type) and the rest, rest keeps original order
function split(o: Record<string, unknown>, known: Record<string, (v: unknown) => boolean>) {
  const got: Record<string, any> = {};
  const extra: Record<string, unknown> = {};
  const set = (t: object, k: string, v: unknown) => Object.defineProperty(t, k, { value: v, enumerable: true, writable: true, configurable: true });
  for (const [k, v] of Object.entries(o)) {
    if (Object.prototype.hasOwnProperty.call(known, k) && known[k](v)) set(got, k, v);
    else set(extra, k, v);
  }
  return { got, extra };
}
const str = (v: unknown) => typeof v === "string";
const tgt = (v: unknown) => v === "bash" || v === "sql";

export function isRunbookText(text: string): boolean {
  try {
    const v = JSON.parse(text);
    return isObj(v) && Array.isArray(v.categories);
  } catch {
    return false;
  }
}

// offset of the first JSON syntax error (minimal recursive-descent scan), or text.length at EOF
function scanErr(t: string): number {
  let i = 0;
  const ws = () => { while (/[ \t\r\n]/.test(t[i] ?? "")) i++; };
  const lit = (w: string) => { if (!t.startsWith(w, i)) throw i; i += w.length; };
  const val = (): void => {
    ws();
    const c = t[i];
    if (c === "{" || c === "[") {
      const close = c === "{" ? "}" : "]";
      i++; ws();
      if (t[i] === close) { i++; return; }
      for (;;) {
        ws();
        if (c === "{") { if (t[i] !== '"') throw i; str(); ws(); if (t[i] !== ":") throw i; i++; }
        val(); ws();
        if (t[i] === ",") { i++; continue; }
        if (t[i] === close) { i++; return; }
        throw i;
      }
    } else if (c === '"') str();
    else if (c === "t") lit("true");
    else if (c === "f") lit("false");
    else if (c === "n") lit("null");
    else {
      const m = /^-?(0|[1-9]\d*)(\.\d+)?([eE][+-]?\d+)?/.exec(t.slice(i));
      if (!m) throw i;
      i += m[0].length;
    }
  };
  const str = () => {
    i++;
    for (; i < t.length; i++) {
      if (t[i] === "\\") i++;
      else if (t[i] === '"') { i++; return; }
      else if (t[i] < " ") throw i;
    }
    throw i;
  };
  try { val(); ws(); return i < t.length ? i : t.length; } catch (e) { return typeof e === "number" ? e : 0; }
}

function errPos(text: string, msg: string): { line: number; col: number } {
  const lc = /line (\d+) column (\d+)/.exec(msg);
  if (lc) return { line: +lc[1], col: +lc[2] };
  const p = /position (\d+)/.exec(msg);
  const pos = p ? +p[1] : scanErr(text);
  const before = text.slice(0, pos);
  return { line: before.split("\n").length, col: pos - (before.lastIndexOf("\n") + 1) + 1 };
}

export function parseRunbook(text: string): ParseResult {
  let v: unknown;
  try {
    v = JSON.parse(text);
  } catch (e) {
    const msg = (e as Error).message;
    return { ok: false, error: msg, ...errPos(text, msg) };
  }
  if (!isObj(v) || !Array.isArray(v.categories))
    return { ok: false, error: 'not a runbook: expected a JSON object with a "categories" array', line: 1, col: 1 };
  const { got, extra } = split(v, { name: str, description: str, categories: Array.isArray });
  const miss = (o: Record<string, unknown>, ks: string[]) => ks.filter((k) => !(k in o));
  const categories = (got.categories as unknown[]).map((c): RbCategory => {
    if (!isObj(c)) return { name: "", entries: [], extra: {}, raw: c };
    const g = split(c, { name: str, icon: str, entries: Array.isArray });
    const entries = ((g.got.entries as unknown[]) ?? []).map((e): RbEntry => {
      if (!isObj(e)) return { label: "", command: "", extra: {}, raw: e };
      const x = split(e, { label: str, description: str, command: str, target: tgt });
      const missing = miss(e, ["label", "command"]);
      return { label: x.got.label ?? "", command: x.got.command ?? "", ...(x.got.description !== undefined && { description: x.got.description }), ...(x.got.target && { target: x.got.target }), extra: x.extra, ...(missing.length && { missing }), order: Object.keys(e) };
    });
    const missing = miss(c, ["name", "entries"]);
    return { name: g.got.name ?? "", ...(g.got.icon !== undefined && { icon: g.got.icon }), entries, extra: g.extra, ...(missing.length && { missing }), order: Object.keys(c) };
  });
  const rb: Runbook = { categories, extra, order: Object.keys(v) };
  if (got.name !== undefined) rb.name = got.name;
  if (got.description !== undefined) rb.description = got.description;
  return { ok: true, rb };
}

// canonical slot order; a typed value wins, else a raw wrong-typed one kept in extra, else nothing
function emit(keys: string[], typed: Record<string, unknown>, extra: Record<string, unknown>, missing: string[] = [], order?: string[]): Record<string, unknown> {
  const out: Record<string, unknown> = {};
  for (const k of keys) {
    const t = typed[k];
    const blank = t === "" || (Array.isArray(t) && t.length === 0);
    if (t !== undefined && !(blank && (missing.includes(k) || k in extra))) out[k] = t;
    else if (k in extra) out[k] = extra[k];
  }
  for (const k of Object.keys(extra)) if (!keys.includes(k)) def(out, k, extra[k]);
  if (!order) return out;
  // original key order; a key new to this object goes after its nearest canonical predecessor
  const list = order.filter((k) => Object.prototype.hasOwnProperty.call(out, k));
  for (const k of Object.keys(out)) {
    if (list.includes(k)) continue;
    let at = 0;
    list.forEach((x, j) => { const r = keys.indexOf(x); if (r >= 0 && r < keys.indexOf(k)) at = j + 1; });
    list.splice(at, 0, k);
  }
  const res: Record<string, unknown> = {};
  for (const k of list) def(res, k, out[k]);
  return res;
}
const def = (o: object, k: string, v: unknown) => Object.defineProperty(o, k, { value: v, enumerable: true, writable: true, configurable: true });

export function serializeRunbook(rb: Runbook): string {
  const cats = rb.categories.map((c) => {
    if (c.raw !== undefined) return c.raw;
    return emit(["name", "icon", "entries"], { name: c.name, icon: c.icon, entries: c.entries.map((e) => (e.raw !== undefined ? e.raw : emit(["label", "description", "command", "target"], e as unknown as Record<string, unknown>, e.extra, e.missing, e.order))) }, c.extra, c.missing, c.order);
  });
  return JSON.stringify(emit(["name", "description", "categories"], { name: rb.name, description: rb.description, categories: cats }, rb.extra, [], rb.order), null, 2) + "\n";
}

const placeholderRe = /\{\{([A-Za-z_][A-Za-z0-9_-]*)(?::([^}]*))?\}\}/g;
// Go-template keywords fit the syntax ({{end}}, {{else}}) but are not placeholders (same as Go)
const templateWords = new Set(["end", "else", "break", "continue", "nil", "true", "false"]);

/** placeholderMatches yields the placeholder matches of s, skipping Go-template keywords. */
export function placeholderMatches(s: string): RegExpExecArray[] {
  return [...s.matchAll(placeholderRe)].filter((m) => !templateWords.has(m[1])) as RegExpExecArray[];
}

// distinct names in document order; first occurrence and its default win (same as Go)
export function placeholders(cmd: string): { name: string; def: string }[] {
  const out: { name: string; def: string }[] = [];
  for (const m of placeholderMatches(cmd)) if (!out.some((p) => p.name === m[1])) out.push({ name: m[1], def: m[2] ?? "" });
  return out;
}

export function insertPlaceholder(cmd: string, at: number, name: string, def = ""): { cmd: string; cursor: number } {
  const tok = def ? `{{${name}:${def}}}` : `{{${name}}}`;
  return { cmd: cmd.slice(0, at) + tok + cmd.slice(at), cursor: at + tok.length };
}

const clamp = (n: number, len: number) => Math.max(0, Math.min(len, n));
const ins = <T>(a: T[], i: number | undefined, v: T) => { const b = [...a]; b.splice(i === undefined ? b.length : clamp(i, b.length), 0, v); return b; };
const del = <T>(a: T[], i: number) => a.filter((_, j) => j !== i);
const mv = <T>(a: T[], from: number, to: number) => { const b = [...a]; const [x] = b.splice(from, 1); b.splice(clamp(to, b.length), 0, x); return b; };
const setCat = (rb: Runbook, ci: number, f: (c: RbCategory) => RbCategory): Runbook => ({ ...rb, categories: rb.categories.map((c, i) => (i === ci ? f(c) : c)) });
const omit = (o: Record<string, unknown>, k: string) => { const r: Record<string, unknown> = {}; for (const [x, v] of Object.entries(o)) if (x !== k) def(r, x, v); return r; };
const copyEntry = (e: RbEntry): RbEntry => ({ ...e, label: e.label + " (copy)", extra: structuredClone(e.extra) });

export const ops = {
  addCategory: (rb: Runbook, name: string, at?: number): Runbook => ({ ...rb, categories: ins(rb.categories, at, { name, entries: [], extra: {} }) }),
  renameCategory: (rb: Runbook, ci: number, name: string): Runbook => setCat(rb, ci, (c) => ({ ...c, name })),
  deleteCategory: (rb: Runbook, ci: number): Runbook => ({ ...rb, categories: del(rb.categories, ci) }),
  duplicateCategory: (rb: Runbook, ci: number): Runbook => {
    const c = rb.categories[ci];
    return { ...rb, categories: ins(rb.categories, ci + 1, { ...c, name: c.name + " (copy)", entries: c.entries.map((e) => ({ ...e, extra: structuredClone(e.extra) })), extra: structuredClone(c.extra) }) };
  },
  moveCategory: (rb: Runbook, from: number, to: number): Runbook => ({ ...rb, categories: mv(rb.categories, from, to) }),
  addEntry: (rb: Runbook, ci: number, e: Partial<RbEntry> & { label: string; command: string }, at?: number): Runbook =>
    setCat(rb, ci, (c) => ({ ...c, entries: ins(c.entries, at, { extra: {}, ...e }) })),
  updateEntry: (rb: Runbook, ci: number, ei: number, patch: Partial<RbEntry>): Runbook =>
    setCat(rb, ci, (c) => ({ ...c, entries: c.entries.map((e, i) => (i === ei ? { ...e, ...patch } : e)) })),
  // setTarget also drops a wrong-typed target kept in extra, so clearing really clears
  setTarget: (rb: Runbook, ci: number, ei: number, target?: "bash" | "sql"): Runbook =>
    setCat(rb, ci, (c) => ({ ...c, entries: c.entries.map((e, i) => (i === ei ? { ...e, target, extra: omit(e.extra, "target") } : e)) })),
  deleteEntry: (rb: Runbook, ci: number, ei: number): Runbook => setCat(rb, ci, (c) => ({ ...c, entries: del(c.entries, ei) })),
  duplicateEntry: (rb: Runbook, ci: number, ei: number): Runbook => setCat(rb, ci, (c) => ({ ...c, entries: ins(c.entries, ei + 1, copyEntry(c.entries[ei])) })),
  moveEntry: (rb: Runbook, from: [number, number], to: [number, number]): Runbook => {
    const e = rb.categories[from[0]].entries[from[1]];
    if (from[0] === to[0]) return setCat(rb, from[0], (c) => ({ ...c, entries: mv(c.entries, from[1], to[1]) }));
    return {
      ...rb,
      categories: rb.categories.map((c, i) => (i === from[0] ? { ...c, entries: del(c.entries, from[1]) } : i === to[0] ? { ...c, entries: ins(c.entries, to[1], e) } : c)),
    };
  },
};

export function validate(rb: Runbook): { path: string; msg: string }[] {
  const out: { path: string; msg: string }[] = [];
  for (const k of ["name", "description"]) if (k in rb.extra) out.push({ path: k, msg: `wrong type for ${k}` });
  const seen = new Map<string, number>();
  rb.categories.forEach((c, ci) => {
    if (c.raw !== undefined) { out.push({ path: `categories[${ci}]`, msg: "category is not an object" }); return; }
    for (const k of ["name", "icon", "entries"]) if (k in c.extra) out.push({ path: `categories[${ci}].${k}`, msg: `wrong type for ${k}` });
    if (seen.has(c.name)) out.push({ path: `categories[${ci}].name`, msg: `duplicate category name "${c.name}"` });
    seen.set(c.name, ci);
    c.entries.forEach((e, ei) => {
      const p = `categories[${ci}].entries[${ei}]`;
      if (e.raw !== undefined) { out.push({ path: p, msg: "entry is not an object" }); return; }
      for (const k of ["description", "target"]) if (k in e.extra) out.push({ path: `${p}.${k}`, msg: k === "target" ? `unknown target ${JSON.stringify(e.extra[k])}` : `wrong type for ${k}` });
      for (const k of ["label", "command"]) if (k in e.extra) out.push({ path: `${p}.${k}`, msg: `wrong type for ${k}` });
      if (!e.label.trim()) out.push({ path: `${p}.label`, msg: "empty label" });
      if (!e.command.trim()) out.push({ path: `${p}.command`, msg: "empty command" });
      if (e.target !== undefined && e.target !== "bash" && e.target !== "sql") out.push({ path: `${p}.target`, msg: `unknown target "${e.target}"` });
    });
  });
  return out;
}

// --- generator helpers ---

/** slugify turns a runbook name into a file-name stem: "Lab ops" → "lab-ops". */
export const slugify = (s: string) =>
  s.normalize("NFKD").replace(/[\u0300-\u036f]/g, "").toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-+|-+$/g, "");

/** relPath is the path of `file` as seen from the folder of `from` (both workspace-relative). */
export function relPath(from: string, file: string): string {
  const a = from.split("/").slice(0, -1), b = file.split("/");
  let i = 0;
  while (i < a.length && i < b.length - 1 && a[i] === b[i]) i++;
  return [...a.slice(i).map(() => ".."), ...b.slice(i)].join("/");
}

const metaRE = /^([a-z][a-z-]*)\s*:\s*(.*?)\s*$/; // same as internal/deck metaRE

/** metaValue is a front-matter value; anything but a plain path is quoted (si strips the quotes). */
export const metaValue = (v: string) => (/^[\w./-]+$/.test(v) ? v : `"${v}"`);

/**
 * linkRunbook sets `runbook: <rel>` in a deck's front matter. Front matter is
 * recognised exactly as internal/deck does: a `---` first line (after a BOM),
 * then only blank, `#` or `key: value` lines up to a closing `---`. An existing
 * runbook line is replaced, else one is added before the closing fence; with no
 * front matter a new block is prepended. Every other byte stays as it was.
 */
export function linkRunbook(deck: string, rel: string): string {
  const bom = deck.startsWith("\uFEFF") ? "\uFEFF" : "";
  const lines = deck.slice(bom.length).split(/(?<=\n)/); // each line keeps its own ending
  const body = (l: string) => l.replace(/\r?\n$/, "");
  const eol = (l: string) => l.slice(body(l).length);
  const line = `runbook: ${metaValue(rel)}`;
  let end = -1;
  if (lines.length >= 2 && body(lines[0]).trim() === "---") {
    for (let i = 1; i < lines.length; i++) {
      const t = body(lines[i]).trim();
      if (t === "---") { end = i; break; }
      if (t && !t.startsWith("#") && !metaRE.test(t)) break;
    }
  }
  if (end < 0) {
    const nl = eol(lines[0]) || "\n";
    return bom + `---${nl}${line}${nl}---${nl}${nl}` + lines.join("");
  }
  let found = false;
  for (let i = 1; i < end; i++) {
    const t = body(lines[i]).trim();
    if (!t.startsWith("#") && metaRE.exec(t)?.[1] === "runbook") { lines[i] = line + eol(lines[i]); found = true; }
  }
  if (!found) lines.splice(end, 0, line + eol(lines[0]));
  return bom + lines.join("");
}
