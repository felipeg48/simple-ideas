// Pure runbook model: lossless parse/serialize, edit ops, placeholders.
export interface RbEntry { label: string; description?: string; command: string; target?: "bash" | "sql"; extra: Record<string, unknown> }
export interface RbCategory { name: string; icon?: string; entries: RbEntry[]; extra: Record<string, unknown> }
export interface Runbook { name?: string; description?: string; categories: RbCategory[]; extra: Record<string, unknown> }
export type ParseResult = { ok: true; rb: Runbook } | { ok: false; error: string; line: number; col: number };

const isObj = (v: unknown): v is Record<string, unknown> => typeof v === "object" && v !== null && !Array.isArray(v);

// split o into known keys (right type) and the rest, rest keeps original order
function split(o: Record<string, unknown>, known: Record<string, (v: unknown) => boolean>) {
  const got: Record<string, any> = {};
  const extra: Record<string, unknown> = {};
  for (const [k, v] of Object.entries(o)) {
    if (known[k]?.(v)) got[k] = v;
    else extra[k] = v;
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
  const categories = (got.categories as unknown[]).map((c): RbCategory => {
    const o = isObj(c) ? c : {};
    const g = split(o, { name: str, icon: str, entries: Array.isArray });
    const entries = ((g.got.entries as unknown[]) ?? []).map((e): RbEntry => {
      const x = split(isObj(e) ? e : {}, { label: str, description: str, command: str, target: tgt });
      return { label: x.got.label ?? "", command: x.got.command ?? "", ...(x.got.description !== undefined && { description: x.got.description }), ...(x.got.target && { target: x.got.target }), extra: x.extra };
    });
    return { name: g.got.name ?? "", ...(g.got.icon !== undefined && { icon: g.got.icon }), entries, extra: g.extra };
  });
  const rb: Runbook = { categories, extra };
  if (got.name !== undefined) rb.name = got.name;
  if (got.description !== undefined) rb.description = got.description;
  return { ok: true, rb };
}

export function serializeRunbook(rb: Runbook): string {
  const o: Record<string, unknown> = {};
  if (rb.name !== undefined) o.name = rb.name;
  if (rb.description !== undefined) o.description = rb.description;
  o.categories = rb.categories.map((c) => {
    const k: Record<string, unknown> = { name: c.name };
    if (c.icon !== undefined) k.icon = c.icon;
    k.entries = c.entries.map((e) => {
      const x: Record<string, unknown> = { label: e.label };
      if (e.description !== undefined) x.description = e.description;
      x.command = e.command;
      if (e.target !== undefined) x.target = e.target;
      return { ...x, ...e.extra };
    });
    return { ...k, ...c.extra };
  });
  return JSON.stringify({ ...o, ...rb.extra }, null, 2) + "\n";
}

const placeholderRe = /\{\{([A-Za-z_][A-Za-z0-9_-]*)(?::([^}]*))?\}\}/g;

// distinct names in document order; first default wins (same as Go)
export function placeholders(cmd: string): { name: string; def: string }[] {
  const out: { name: string; def: string }[] = [];
  for (const m of cmd.matchAll(placeholderRe)) {
    const i = out.findIndex((p) => p.name === m[1]);
    if (i < 0) out.push({ name: m[1], def: m[2] ?? "" });
    else if (!out[i].def && m[2]) out[i].def = m[2];
  }
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
  const seen = new Map<string, number>();
  rb.categories.forEach((c, ci) => {
    if (seen.has(c.name)) out.push({ path: `categories[${ci}].name`, msg: `duplicate category name "${c.name}"` });
    seen.set(c.name, ci);
    c.entries.forEach((e, ei) => {
      const p = `categories[${ci}].entries[${ei}]`;
      if (!e.label.trim()) out.push({ path: `${p}.label`, msg: "empty label" });
      if (!e.command.trim()) out.push({ path: `${p}.command`, msg: "empty command" });
      if (e.target !== undefined && e.target !== "bash" && e.target !== "sql") out.push({ path: `${p}.target`, msg: `unknown target "${e.target}"` });
    });
  });
  return out;
}
