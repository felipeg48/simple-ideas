// The Motion view: an SVG file as frames. The board draws the SVG itself (in a
// shadow root, so its <style> stays inside) in the frame being edited; drag a
// part to move it in that frame, the panels set its size, turn, colour and
// whether it shows, and each frame's timing. Every change is an op; Go
// rewrites the SVG (POST motion, internal/motion) and the new text goes into
// the editor as one undoable change. Coordinates are viewBox units.
import { useEffect, useRef, useState } from "preact/hooks";
import { api } from "./api";
import { Field } from "./map-editor";

interface Look { x: number; y: number; scale: number; rotate: number; fill: string; hidden: boolean }
interface Frame { name: string; dur: number; ease: string; parts: Record<string, Look> }
interface Part { name: string; tag: string; index: number; label: string }
interface Model { viewBox: string; parts: Part[] | null; frames: Frame[] }
interface Resp { text: string; model: Model; snippet: string }
export interface MotionOp { kind: string; frame?: number; part?: string; index?: number; look?: Look; dur?: number; ease?: string; name?: string; shape?: string }

/** isMotionFile: an .svg the Motion view can open (any SVG, or an empty file to start a board in). */
export const isMotionFile = (path: string) => /\.svg$/i.test(path);

const ZERO: Look = { x: 0, y: 0, scale: 1, rotate: 0, fill: "", hidden: false };
const COLOURS: [string, string][] = [["", "as in frame 1"], ["var(--si-accent)", "accent"], ["var(--si-accent-2)", "accent 2"], ["var(--si-ok)", "ok"],
  ["var(--si-warn)", "warn"], ["var(--si-crit)", "crit"], ["var(--si-fg)", "text"], ["var(--si-muted)", "muted"], ["var(--si-surface)", "surface"]];
const EASES = ["", "ease", "linear", "ease-in-out", "spring", "snap"];
const SHAPES = "rect,circle,ellipse,line,polyline,polygon,path,text,g,image,use";
// inside the board only: the theme's tokens as the deck has them, and the same motion the deck gives a canvas
const BOARD_CSS = `:host{display:grid;place-items:center;width:100%;height:100%}
svg{width:100%;height:auto;max-height:100%;display:block;overflow:visible;background:#fff;box-shadow:0 0 0 1px rgba(0,0,0,.14),0 6px 24px rgba(0,0,0,.08);border-radius:4px;--si-accent:#3657d6;--si-accent-2:#0b8a7a;--si-ok:#178a4c;--si-warn:#c77c02;--si-crit:#d23c3c;--si-fg:#16171b;--si-muted:#5d626c;--si-surface:#fff}
svg *{transition:transform .7s cubic-bezier(.3,.1,.2,1),fill .7s,opacity .7s}
svg.dragging *{transition:none}
[data-part]{cursor:grab}
.mo-sel{outline:2px dashed #3657d6;outline-offset:3px}`;

/** boardSVG parses the file as SVG for the board, without anything that runs: no scripts,
 * no foreign HTML, no on… handlers, no javascript: links (the board lives in the Studio's page). */
export function boardSVG(text: string): SVGSVGElement | null {
  if (!/<svg\b/.test(text)) return null;
  const doc = new DOMParser().parseFromString(text, "image/svg+xml");
  const s = doc.documentElement;
  if (s.nodeName !== "svg" || doc.querySelector("parsererror")) return null;
  s.querySelectorAll("script,foreignObject,iframe,object,embed").forEach((e) => e.remove());
  for (const e of [s, ...Array.from(s.querySelectorAll("*"))]) {
    for (const a of Array.from(e.attributes)) {
      if (/^on/i.test(a.name) || (/href$/i.test(a.name) && /^\s*javascript:/i.test(a.value))) e.removeAttribute(a.name);
    }
  }
  return document.importNode(s, true) as unknown as SVGSVGElement;
}

export function MotionEditor(p: { path: string; text: string; setText(next: string): void; undo(redo: boolean): void }) {
  const host = useRef<HTMLDivElement>(null);
  const [resp, setResp] = useState<Resp | null>(null);
  const [err, setErr] = useState("");
  const [frame, setFrame] = useState(0);
  const [sel, setSel] = useState("");
  const [copied, setCopied] = useState(false);
  const last = useRef("");
  const props = useRef(p);
  props.current = p;
  const state = useRef({ frame: 0, sel: "", model: null as Model | null });
  state.current = { frame, sel, model: resp?.model ?? null };

  const run = async (op: MotionOp | null) => {
    try {
      const r = await api.motion(props.current.path, last.current, op) as Resp;
      if (op) props.current.setText(r.text);
      last.current = r.text;
      setResp(r); setErr("");
      if (state.current.frame >= r.model.frames.length) setFrame(r.model.frames.length - 1);
      return r;
    } catch (e) { setErr(e instanceof Error ? e.message : String(e)); return null; }
  };
  const op = (o: MotionOp) => void run(o);

  // the editor's text changed (typing in the text view, Undo): redraw from it
  useEffect(() => {
    if (p.text === last.current) return;
    last.current = p.text;
    const t = setTimeout(() => void run(null), 150);
    return () => clearTimeout(t);
  }, [p.text]);
  useEffect(() => { last.current = p.text; setFrame(0); setSel(""); void run(null); }, [p.path]);
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (!(e.metaKey || e.ctrlKey) || e.key.toLowerCase() !== "z" || /input|textarea|select/i.test((e.target as HTMLElement).tagName)) return;
      e.preventDefault();
      props.current.undo(e.shiftKey);
    };
    addEventListener("keydown", onKey);
    return () => removeEventListener("keydown", onKey);
  }, []);

  const svgEl = () => host.current?.shadowRoot?.querySelector("svg") as SVGSVGElement | null;
  const stateOf = (i: number, m: Model | null) => !m || !m.frames.length ? "" : i === 0 ? m.frames[0].name : m.frames[0].name + " " + m.frames[i].name;
  const show = (i: number) => { svgEl()?.setAttribute("data-state", stateOf(i, state.current.model)); };
  const mark = () => {
    const s = svgEl();
    s?.querySelectorAll(".mo-sel").forEach((e) => e.classList.remove("mo-sel"));
    if (state.current.sel) s?.querySelector(`[data-part="${CSS.escape(state.current.sel)}"]`)?.classList.add("mo-sel");
  };

  // draw the board when the SVG text changes; switching frames only changes the state, so the parts glide
  useEffect(() => {
    const h = host.current;
    if (!h) return;
    const root = h.shadowRoot ?? h.attachShadow({ mode: "open" });
    root.replaceChildren();
    const s = boardSVG(resp?.text ?? "");
    if (s) { const st = document.createElement("style"); st.textContent = BOARD_CSS; root.append(st, s); }
    if (s) { s.removeAttribute("width"); s.removeAttribute("height"); s.setAttribute("data-state", stateOf(frame, resp?.model ?? null)); }
    mark();
  }, [resp?.text]);
  useEffect(() => { show(frame); }, [frame]);
  useEffect(() => { mark(); }, [sel]);

  // pick and drag: a part moves in the frame being edited (viewBox units)
  useEffect(() => {
    const h = host.current;
    if (!h) return;
    let drag: { el: SVGGraphicsElement; name: string; x0: number; y0: number; look: Look; moved: boolean } | null = null;
    const pt = (e: PointerEvent) => {
      const s = svgEl(), m = s?.getScreenCTM();
      if (!s || !m) return { x: 0, y: 0 };
      const q = new DOMPoint(e.clientX, e.clientY).matrixTransform(m.inverse());
      return { x: q.x, y: q.y };
    };
    const down = async (e: PointerEvent) => {
      const t = (e.composedPath()[0] as Element | undefined)?.closest?.(SHAPES) as SVGGraphicsElement | null;
      const s = svgEl();
      if (!t || !s || !s.contains(t)) { setSel(""); return; }
      const part = t.closest("[data-part]") as SVGGraphicsElement | null;
      let name = part?.getAttribute("data-part") ?? "";
      if (!name) { // name it first: a part needs a name to move
        const shapes = Array.from(s.querySelectorAll(SHAPES));
        const index = shapes.indexOf(t);
        const taken = new Set((state.current.model?.parts ?? []).map((x) => x.name));
        let n = 1;
        while (taken.has(t.tagName + n)) n++;
        name = t.tagName + n;
        await run({ kind: "name", index, name });
        setSel(name);
        return;
      }
      setSel(name);
      const m = state.current.model!;
      const look = { ...ZERO, ...(m.frames[state.current.frame]?.parts[name] ?? {}) };
      const p0 = pt(e);
      drag = { el: part!, name, x0: p0.x, y0: p0.y, look, moved: false };
      h.setPointerCapture(e.pointerId);
      s.classList.add("dragging");
    };
    const move = (e: PointerEvent) => {
      if (!drag) return;
      const q = pt(e), dx = q.x - drag.x0, dy = q.y - drag.y0;
      drag.moved = drag.moved || Math.abs(dx) + Math.abs(dy) > 0.5;
      const l = drag.look;
      drag.el.style.transform = `translate(${l.x + dx}px,${l.y + dy}px) rotate(${l.rotate}deg) scale(${l.scale})`;
    };
    const up = (e: PointerEvent) => {
      if (!drag) return;
      const d = drag;
      drag = null;
      svgEl()?.classList.remove("dragging");
      if (!d.moved) { d.el.style.transform = ""; return; }
      const q = pt(e), r = (v: number) => Math.round(v * 10) / 10;
      void run({ kind: "look", frame: state.current.frame, part: d.name, look: { ...d.look, x: r(d.look.x + q.x - d.x0), y: r(d.look.y + q.y - d.y0) } });
    };
    h.addEventListener("pointerdown", down);
    h.addEventListener("pointermove", move);
    h.addEventListener("pointerup", up);
    return () => { h.removeEventListener("pointerdown", down); h.removeEventListener("pointermove", move); h.removeEventListener("pointerup", up); };
  }, []);

  // Play: frame 1 to the last, each held for its move and a breath
  const [playing, setPlaying] = useState(false);
  const play = async () => {
    const m = resp?.model;
    if (!m || playing) return;
    setPlaying(true);
    for (let i = 0; i < m.frames.length; i++) {
      setFrame(i);
      await new Promise((r) => setTimeout(r, ((m.frames[i].dur || 0.7) + 0.5) * 1000));
    }
    setPlaying(false);
  };

  const m = resp?.model;
  const frames = m?.frames ?? [];
  const f = frames[Math.min(frame, frames.length - 1)];
  const parts = m?.parts ?? [];
  const look: Look = { ...ZERO, ...(f && sel ? f.parts[sel] ?? {} : {}) };
  const setLook = (k: keyof Look, v: number | string | boolean) => op({ kind: "look", frame, part: sel, look: { ...look, [k]: v } });
  const blank = !!resp && !/<svg\b/.test(resp.text);

  return (
    <div class="mapx mox">
      <div class="mapx-canvas mox-board">
        {blank ? <div class="ed-msg"><div class="ed-msg-card"><b>An empty board</b><p>Start an animation here: draw frame 1, add a frame, move things, and si plays the motion between them.</p>
          <button class="btn btn-pri" onClick={() => op({ kind: "blank" })} disabled={!!resp.text.trim()}>Start a blank board</button>
          {resp.text.trim() && <p class="mute">This file is not an SVG; empty it, or open an .svg file.</p>}</div></div> : null}
        <div ref={host} class="mox-svg" aria-label="Motion board: drag a part to move it in this frame" hidden={blank} />
        {err && <div class="banner banner-err mapx-err" role="alert"><span>{err}</span></div>}
      </div>
      <aside class="mapx-side" aria-label="Motion panels">
        <section class="insp">
          <div class="insp-head"><b>Frames</b><span class="mute">{frames.length} frame{frames.length === 1 ? "" : "s"}</span></div>
          <ol class="mox-strip" aria-label="Frames">
            {frames.map((x, i) => (
              <li><button class="btn" aria-pressed={i === frame} onClick={() => setFrame(i)} title={x.name}>{i + 1}</button></li>
            ))}
          </ol>
          <div class="mapx-actions">
            <button class="btn" disabled={!m} onClick={async () => { const r = await run({ kind: "add-frame", frame }); if (r) setFrame(frame + 1); }}>Add a frame</button>
            <button class="btn btn-danger" disabled={frames.length < 2} onClick={() => { op({ kind: "del-frame", frame }); setFrame(Math.max(0, frame - 1)); }}>Delete frame {frame + 1}</button>
            <button class="btn" disabled={frames.length < 2 || playing} onClick={() => void play()}>▶ Play</button>
          </div>
          <p class="mapx-hint">{frame === 0 ? "Frame 1 is the picture as it starts." : "A new frame starts as a copy; drag parts to where they go."}</p>
        </section>
        {f && frame > 0 && (
          <section class="insp">
            <div class="insp-head"><b>Into frame {frame + 1}</b><span class="mute">{f.name}</span></div>
            <div class="insp-grid">
              <Field id="mo-dur" label="Seconds" value={f.dur ? String(f.dur) : ""} placeholder="0.7" onSet={(v) => op({ kind: "timing", frame, dur: +v || 0, ease: f.ease })} />
              <label for="mo-ease">Easing</label>
              <select id="mo-ease" value={f.ease} onChange={(e) => op({ kind: "timing", frame, dur: f.dur, ease: e.currentTarget.value })}>
                {EASES.map((x) => <option value={x}>{x || "default"}</option>)}
              </select>
            </div>
          </section>
        )}
        <section class="insp">
          <div class="insp-head"><b>Part</b><span class="mute">{sel || "click one on the board"}</span></div>
          {sel && f && (
            <div class="insp-grid">
              <Field id="mo-x" label="Right" value={String(look.x)} onSet={(v) => setLook("x", +v || 0)} />
              <Field id="mo-y" label="Down" value={String(look.y)} onSet={(v) => setLook("y", +v || 0)} />
              <Field id="mo-scale" label="Size" value={String(look.scale)} onSet={(v) => setLook("scale", +v || 1)} />
              <Field id="mo-rotate" label="Turn (°)" value={String(look.rotate)} onSet={(v) => setLook("rotate", +v || 0)} />
              <label for="mo-fill">Colour</label>
              <select id="mo-fill" value={look.fill} onChange={(e) => setLook("fill", e.currentTarget.value)}>
                {COLOURS.map(([v, l]) => <option value={v}>{l}</option>)}
                {look.fill && !COLOURS.some(([v]) => v === look.fill) && <option value={look.fill}>{look.fill}</option>}
              </select>
              <span />
              <span class="mapx-checks"><label><input type="checkbox" checked={look.hidden} onChange={(e) => setLook("hidden", e.currentTarget.checked)} /> Hidden in this frame</label></span>
            </div>
          )}
          <div class="mapx-actions">
            {["rect", "circle", "text", "line"].map((s) => (
              <button class="btn" disabled={!m || blank} onClick={async () => { const r = await run({ kind: "add-shape", shape: s }); const n = r?.model.parts?.at(-1)?.name; if (n) setSel(n); }}>Add {s === "rect" ? "a box" : s === "circle" ? "a circle" : s === "text" ? "text" : "a line"}</button>
            ))}
            {sel && <button class="btn btn-danger" onClick={() => { const x = parts.find((q) => q.name === sel); if (x) { op({ kind: "del-shape", index: x.index }); setSel(""); } }}>Delete {sel}</button>}
          </div>
          <p class="mapx-hint">Positions are in the picture's own units, so it scales with the slide. Words and colours stay in the text view too.</p>
        </section>
        {m && (
          <section class="insp">
            <div class="insp-head"><b>Use it on a slide</b><span class="mute">one click per frame</span></div>
            <pre class="cwx-out mox-snippet">{resp!.snippet}</pre>
            <div class="mapx-actions">
              <button class="btn" onClick={() => { void navigator.clipboard?.writeText(resp!.snippet).then(() => { setCopied(true); setTimeout(() => setCopied(false), 1500); }, () => {}); }}>{copied ? "Copied" : "Copy"}</button>
            </div>
            <p class="mapx-hint">Paste it into a slide; the path is from the workspace, so make it relative to the deck if the deck lives in a folder.</p>
          </section>
        )}
      </aside>
    </div>
  );
}
