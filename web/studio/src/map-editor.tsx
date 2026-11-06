// The map editor: a mind map page (or one ```mindmap fence) drawn by the real
// runtime in an iframe (mindmap-edit.js), with panels for the selected topic
// and the map. The canvas and the panels send ops; Go rewrites the Markdown
// (POST mindmap) and the new text goes into the editor as one undoable change,
// so Undo, saving and the text view work as for any edit.
import { useEffect, useMemo, useRef, useState } from "preact/hooks";
import { api, type Problem } from "./api";
import { resolvedTheme } from "./theme";

export interface MapTopic {
  text: string; sticky?: boolean; img?: string; imgAlt?: string; badge?: string; collapsed?: boolean;
  target?: string; code?: { lang: string; text: string }; kids?: MapTopic[];
}
export interface MapModel { title: string; id?: string; heading: boolean; topics: MapTopic[] }
export interface MapOp { kind: string; map?: number; path?: number[]; text?: string; field?: string; value?: string; to?: number[]; index?: number; x?: number; y?: number; branch?: boolean }
interface Resp {
  text: string; data: unknown; opts: Record<string, string>; problems: Problem[];
  maps: MapModel[]; front: Record<string, string>; fence: boolean; fenceLine?: number;
  select: { map: number; path: number[] } | null;
}

/** mindmapFenceAt finds the ```mindmap fence holding a 1-based line: its opening line, or 0. */
export function mindmapFenceAt(text: string, line: number): number {
  const ls = text.split("\n");
  for (let i = 0; i < ls.length; i++) {
    const m = /^\s*(```+|~~~+)\s*([\w+-]*)/.exec(ls[i]);
    if (!m) continue;
    let end = ls.length;
    for (let j = i + 1; j < ls.length; j++) if (ls[j].trim() === m[1]) { end = j; break; }
    if (m[2] === "mindmap" && line - 1 >= i && line - 1 <= end) return i + 1;
    i = end;
  }
  return 0;
}

/** isMindmapPage: the text is a mind map page (type: mindmap in its front matter). */
export const isMindmapPage = (text: string) => /^---\s*\n[\s\S]*?^type:\s*["']?mindmap\b/m.test(text.slice(0, 2048)) && text.startsWith("---");

const LOOKS: [string, string, string[]][] = [
  ["layout", "Layout", ["map", "right", "org", "radial"]],
  ["lines", "Lines", ["curved", "straight", "orthogonal", "tapered"]],
  ["nodes", "Topics", ["box", "pill", "underline"]],
  ["map-theme", "Theme", ["auto", "harbor", "blueprint", "ink", "garden"]],
  ["view", "View", ["2d", "3d"]],
  ["camera", "Camera", ["iso", "perspective"]],
  ["kind", "3D kind", ["blocks", "bricks", "toy"]],
  ["floors", "Floors", ["on", "off"]],
  ["flow", "Flow", ["off", "on"]],
];

const at = (maps: MapModel[], map: number, path: number[]): MapTopic | null => {
  let list: MapTopic[] | undefined = maps[map]?.topics, t: MapTopic | null = null;
  for (const i of path) { t = list?.[i] ?? null; if (!t) return null; list = t.kids; }
  return t;
};

/** rel is the path of to as written in from's file. */
export function rel(from: string, to: string): string {
  const a = from.split("/").slice(0, -1), b = to.split("/");
  let i = 0;
  while (i < a.length && i < b.length - 1 && a[i] === b[i]) i++;
  return "../".repeat(a.length - i) + b.slice(i).join("/");
}

export function MapEditor(p: {
  path: string; text: string; line: number; images: string[]; mapFiles: string[]; decks: string[];
  setText(next: string): void; undo(redo: boolean): void;
}) {
  const frame = useRef<HTMLIFrameElement>(null);
  const [resp, setResp] = useState<Resp | null>(null);
  const [err, setErr] = useState("");
  const [sel, setSel] = useState<{ map: number; path: number[] | null; readonly?: boolean }>({ map: 0, path: null });
  const ready = useRef(false);
  const last = useRef<{ text: string; resp: Resp | null }>({ text: "", resp: null });
  const pending = useRef<Resp["select"]>(null);
  const line = useRef(p.line);
  line.current = resp?.fenceLine ?? p.line;
  const props = useRef(p);
  props.current = p;
  const src = useMemo(() => `/_si/studio/api/mmcanvas?path=${encodeURIComponent(p.path)}${p.line ? "&line=" + p.line : ""}`, [p.path, p.line]);

  const send = (r: Resp, select: Resp["select"]) => {
    const w = frame.current?.contentWindow;
    if (w && ready.current) w.postMessage({ type: "si-mm-data", data: r.data, opts: r.opts, select }, "*");
  };
  const show = (r: Resp, select: Resp["select"] = null) => {
    last.current = { text: r.text, resp: r };
    setResp(r); setErr("");
    send(r, select);
  };
  const run = async (op: MapOp | null): Promise<Resp | null> => {
    const cur = props.current;
    try {
      const r = await api.mindmap(cur.path, last.current.text || cur.text, line.current, op) as Resp;
      if (op) { cur.setText(r.text); if (r.select) setSel({ map: r.select.map, path: r.select.path }); }
      show(r, op ? r.select : pending.current);
      pending.current = null;
      return r;
    } catch (e) { setErr(e instanceof Error ? e.message : String(e)); return null; }
  };
  // a code topic: a new child that becomes a fenced block
  const addCode = async () => {
    const r = await run({ kind: "add-child", map: sel.map, path: sel.path ?? [], text: "code" });
    if (r?.select) await run({ kind: "set", map: r.select.map, path: r.select.path, field: "lang", value: "text" });
  };

  // the editor's text changed (typing in the text view, Undo): redraw from it
  useEffect(() => {
    if (p.text === last.current.text) return;
    last.current.text = p.text;
    const t = setTimeout(() => void run(null), 120);
    return () => clearTimeout(t);
  }, [p.text]);

  useEffect(() => {
    ready.current = false;
    last.current = { text: p.text, resp: null };
    void run(null);
    const onMsg = (e: MessageEvent) => {
      if (e.source !== frame.current?.contentWindow) return;
      const m = e.data || {};
      if (m.type === "si-mm-ready") { ready.current = true; if (last.current.resp) send(last.current.resp, null); }
      else if (m.type === "si-mm-op") void run(m.op);
      else if (m.type === "si-mm-select") setSel({ map: m.map, path: m.path, readonly: m.readonly });
      else if (m.type === "si-mm-undo") props.current.undo(!!m.redo);
    };
    addEventListener("message", onMsg);
    return () => removeEventListener("message", onMsg);
  }, [src]);

  // the canvas follows the Studio's light / dark choice, like the preview
  const syncMode = () => { try { frame.current?.contentDocument?.documentElement.setAttribute("data-si-mode", resolvedTheme()); } catch { /* not loaded */ } };
  useEffect(() => { addEventListener("si-theme", syncMode); return () => removeEventListener("si-theme", syncMode); }, []);
  const opOn = (o: MapOp) => void run({ map: sel.map, path: sel.path ?? [], ...o });
  const topic = resp && sel.path ? at(resp.maps, sel.map, sel.path) : null;
  const map = resp?.maps[sel.map];
  const fence = !!resp?.fence;
  const here = p.path;
  const targets = useMemo(() => [
    ...(resp?.maps ?? []).filter((m) => m.id).map((m) => "#" + m.id),
    ...p.mapFiles.filter((f) => f !== here).map((f) => rel(here, f)),
    ...p.decks.map((f) => "slides: " + rel(here, f) + "#1"),
  ], [resp, p.mapFiles, p.decks, here]);

  return (
    <div class="mapx">
      <div class="mapx-canvas">
        <iframe ref={frame} src={src} title="Mind map editor" onLoad={syncMode} />
        {err && <div class="banner banner-err mapx-err" role="alert"><span>{err}</span></div>}
      </div>
      <aside class="mapx-side" aria-label="Map editor panels">
        <section class="insp">
          <div class="insp-head"><b>{topic ? (topic.code ? "Code" : topic.sticky ? "Sticky note" : "Topic") : sel.path ? "Central topic" : "Map"}</b>
            <span class="mute">{sel.readonly ? "from another file: open it to edit" : topic ? topic.text : map?.title}</span></div>
          {!sel.path && <p class="mapx-hint">Click a topic. <kbd>Tab</kbd> adds a child, <kbd>Enter</kbd> a sibling, <kbd>F2</kbd> or typing renames, <kbd>Delete</kbd> removes; drag onto a topic to move it there (its top or bottom edge: before or after it), Shift-drag to place it freely.</p>}
          {sel.path && !topic && !sel.readonly && map && (
            <div class="insp-grid">
              <Field id="mx-title" label="Title" value={map.heading || fence ? map.title : resp?.front.title ?? map.title} onSet={(v) => opOn({ kind: "rename", path: [], text: v })} />
            </div>
          )}
          {topic && !sel.readonly && (
            <div class="insp-grid">
              {topic.code ? (
                <>
                  <Field id="mx-lang" label="Language" value={topic.code.lang} onSet={(v) => opOn({ kind: "set", field: "lang", value: v })} />
                  <label for="mx-code">Code</label>
                  <textarea id="mx-code" class="field mapx-code" rows={6} value={topic.code.text} spellcheck={false}
                    onBlur={(e) => { if (e.currentTarget.value !== topic.code!.text) opOn({ kind: "set", field: "code", value: e.currentTarget.value }); }} />
                </>
              ) : (
                <>
                  <Field id="mx-text" label="Text" value={topic.text} onSet={(v) => opOn({ kind: "rename", text: v })} />
                  <Field id="mx-badge" label="Badge" value={topic.badge || ""} placeholder="Thu, 87%, new" onSet={(v) => opOn({ kind: "set", field: "badge", value: v })} />
                  <Field id="mx-target" label="Opens" value={topic.target || ""} list="mx-targets" placeholder="#map, other.md, slides: talk.md#5, https://…" onSet={(v) => opOn({ kind: "set", field: "target", value: v })} />
                  <Field id="mx-img" label="Picture" value={topic.img || ""} list="mx-images" placeholder="img/photo.jpg" onSet={(v) => opOn({ kind: "set", field: "img", value: v })} />
                  {topic.img && <Field id="mx-alt" label="Alt text" value={topic.imgAlt || ""} placeholder="what the picture shows" onSet={(v) => opOn({ kind: "set", field: "imgAlt", value: v })} />}
                  <span />
                  <span class="mapx-checks">
                    <label><input type="checkbox" checked={!!topic.sticky} onChange={(e) => opOn({ kind: "set", field: "sticky", value: String(e.currentTarget.checked) })} /> Sticky note</label>
                    <label><input type="checkbox" checked={!!topic.collapsed} disabled={!topic.kids?.length} onChange={(e) => opOn({ kind: "set", field: "collapsed", value: String(e.currentTarget.checked) })} /> Folded</label>
                  </span>
                </>
              )}
            </div>
          )}
          {sel.path && !sel.readonly && (
            <div class="mapx-actions">
              <button class="btn" onClick={() => opOn({ kind: "add-child", text: "New topic" })}>Add child</button>
              {sel.path.length > 0 && <button class="btn" onClick={() => opOn({ kind: "add-sibling", text: "New topic" })}>Add sibling</button>}
              <button class="btn" onClick={() => void addCode()}>Add code</button>
              {sel.path.length > 0 && <button class="btn btn-danger" onClick={() => opOn({ kind: "delete" })}>Delete</button>}
            </div>
          )}
        </section>
        <section class="insp">
          <div class="insp-head"><b>{fence ? "This fence" : "This page"}</b><span class="mute">{resp ? `${resp.maps.length} map${resp.maps.length === 1 ? "" : "s"}` : ""}</span></div>
          {resp && resp.maps.length > 1 && (
            <div class="mapx-maps" role="group" aria-label="Maps">
              {resp.maps.map((m, i) => (
                <button class={"btn" + (i === sel.map ? " on" : "")} aria-pressed={i === sel.map}
                  onClick={() => { setSel({ map: i, path: [] }); frame.current?.contentWindow?.postMessage({ type: "si-mm-show", map: i }, "*"); }}>
                  {i === 0 && !m.heading ? resp.front.title || "Map" : m.title}
                </button>
              ))}
            </div>
          )}
          <div class="insp-grid">
            {LOOKS.map(([key, label, vals]) => {
              const k = fence && key === "map-theme" ? "theme" : key;
              return (
                <>
                  <label for={"mx-" + key}>{label}</label>
                  <select id={"mx-" + key} value={resp?.front[k] ?? ""} onChange={(e) => void run({ kind: "meta", field: k, value: e.currentTarget.value })}>
                    <option value="">default</option>
                    {vals.map((v) => <option value={v}>{v}</option>)}
                  </select>
                </>
              );
            })}
            {!fence && <Field id="mx-wm" label="Watermark" value={resp?.front.watermark || ""} placeholder="text on the map" onSet={(v) => void run({ kind: "meta", field: "watermark", value: v })} />}
            {!fence && resp?.front.watermark && (
              <>
                <label for="mx-wmpos">Placement</label>
                <select id="mx-wmpos" value={resp.front["watermark-position"] || ""} onChange={(e) => void run({ kind: "meta", field: "watermark-position", value: e.currentTarget.value })}>
                  <option value="">corner</option><option value="across">across</option>
                </select>
              </>
            )}
          </div>
          <div class="mapx-actions">
            <button class="btn" onClick={() => void run({ kind: "add-map", text: "New map" })}>Add a map</button>
          </div>
          {resp && resp.problems.length > 0 && (
            <ul class="mapx-problems">{resp.problems.map((x) => <li>{x.msg}</li>)}</ul>
          )}
        </section>
        <datalist id="mx-targets">{targets.map((t) => <option value={t} />)}</datalist>
        <datalist id="mx-images">{p.images.map((f) => <option value={rel(here, f)} />)}</datalist>
      </aside>
    </div>
  );
}

/** Field is a text input that writes its value back on Enter or when it loses focus. */
export function Field(p: { id: string; label: string; value: string; placeholder?: string; list?: string; onSet(v: string): void }) {
  const [v, setV] = useState(p.value);
  useEffect(() => setV(p.value), [p.value]);
  const commit = () => { if (v.trim() !== p.value) p.onSet(v.trim()); };
  return (
    <>
      <label for={p.id}>{p.label}</label>
      <input id={p.id} class="field" value={v} placeholder={p.placeholder} list={p.list} spellcheck={false}
        onInput={(e) => setV(e.currentTarget.value)} onBlur={commit}
        onKeyDown={(e) => { if (e.key === "Enter") { e.preventDefault(); commit(); } if (e.key === "Escape") setV(p.value); }} />
    </>
  );
}
