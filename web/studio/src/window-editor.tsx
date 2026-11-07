// The Window view: the code block under the cursor (or its run of same-id
// blocks, a show) as a code window, drawn by the real renderer in a preview,
// with panels for its look, its steps and its callouts. Every change is an op;
// Go rewrites the Markdown (POST codewin) and the new text goes into the
// editor as one undoable change. Dragging a callout in the preview writes its
// place back as at=.
import { useEffect, useRef, useState } from "preact/hooks";
import { api, type Problem } from "./api";
import { resolvedTheme } from "./theme";
import { Field } from "./map-editor";
import { EditorState } from "@codemirror/state";
import { EditorView, keymap, lineNumbers } from "@codemirror/view";
import { defaultKeymap, history, historyKeymap, indentWithTab } from "@codemirror/commands";

/** StepCode: a small code editor for one step; a pause in typing (or leaving it) writes the code back. */
function StepCode(p: { code: string; label: string; onCommit(code: string): void }) {
  const host = useRef<HTMLDivElement>(null);
  const view = useRef<EditorView | null>(null);
  const commit = useRef(p.onCommit);
  commit.current = p.onCommit;
  const sent = useRef(p.code);
  useEffect(() => {
    let timer = 0;
    const flush = () => { clearTimeout(timer); const v = view.current?.state.doc.toString(); if (v !== undefined && v !== sent.current) { sent.current = v; commit.current(v); } };
    view.current = new EditorView({
      parent: host.current!,
      state: EditorState.create({
        doc: p.code,
        extensions: [lineNumbers(), history(), keymap.of([...defaultKeymap, ...historyKeymap, indentWithTab]),
          EditorView.contentAttributes.of({ "aria-label": p.label }),
          EditorView.updateListener.of((u) => { if (u.docChanged) { clearTimeout(timer); timer = window.setTimeout(flush, 700); } }),
          EditorView.domEventHandlers({ blur: () => { flush(); } })],
      }),
    });
    return () => { flush(); view.current?.destroy(); view.current = null; };
  }, []);
  // the step changed outside (undo, the text view, another step): show it, unless it is what was just sent
  useEffect(() => {
    const v = view.current;
    if (!v || p.code === v.state.doc.toString()) return;
    sent.current = p.code;
    v.dispatch({ changes: { from: 0, to: v.state.doc.length, insert: p.code } });
  }, [p.code]);
  return <div class="cwx-code" ref={host} />;
}

interface Callout { line: number; text: string; at?: [number, number] }
interface Step { start: number; lang: string; info: string; attrs: Record<string, string>; code: string; callouts: Callout[] | null }
interface Resp { text: string; line: number; show: { id: string; steps: Step[] }; html: string; problems: Problem[] }
export interface CodeOp { kind: string; step?: number; index?: number; field?: string; value?: string; line?: number; text?: string; at?: number[]; to?: number }

/** codeFenceAt is the opening line (1-based) of the fenced code block holding line (not a mind map), or 0. */
export function codeFenceAt(text: string, line: number): number {
  const ls = text.split("\n");
  for (let i = 0; i < ls.length; i++) {
    const m = /^\s*(```+|~~~+)\s*([\w#+.-]*)/.exec(ls[i]);
    if (!m) continue;
    let end = ls.length;
    for (let j = i + 1; j < ls.length; j++) if (ls[j].trim() === m[1]) { end = j; break; }
    if (m[2] !== "mindmap" && m[2] !== "mermaid" && m[2] !== "stage" && line - 1 >= i && line - 1 <= end) return i + 1;
    i = end;
  }
  return 0;
}

const LOOKS: [string, string, string[]][] = [
  ["window", "Window", ["mac", "windows", "terminal", "none"]],
  ["theme", "Theme", ["auto", "night", "paper", "dusk"]],
  ["backdrop", "Backdrop", ["none", "dusk", "ocean", "mint", "graphite"]],
  ["transition", "Transition", ["morph", "fade", "slide"]],
];

export function WindowEditor(p: { path: string; text: string; line: number; setText(next: string): void; undo(redo: boolean): void }) {
  const frame = useRef<HTMLIFrameElement>(null);
  const [resp, setResp] = useState<Resp | null>(null);
  const [err, setErr] = useState("");
  const [sel, setSel] = useState(0);
  const [out, setOut] = useState("");
  const last = useRef("");
  const line = useRef(p.line);
  line.current = resp?.line ?? p.line;
  const props = useRef(p);
  props.current = p;
  const selRef = useRef(0);
  selRef.current = sel;

  const run = async (op: CodeOp | null) => {
    try {
      const r = await api.codewin(props.current.path, last.current || props.current.text, line.current, op) as Resp;
      if (op) props.current.setText(r.text);
      last.current = r.text;
      setResp(r); setErr("");
      if (sel >= r.show.steps.length) setSel(r.show.steps.length - 1);
    } catch (e) { setErr(e instanceof Error ? e.message : String(e)); }
  };
  const op = (o: CodeOp) => void run(o);

  // the editor's text changed (typing in the text view, Undo): redraw from it
  useEffect(() => {
    if (p.text === last.current) return;
    last.current = p.text;
    const t = setTimeout(() => void run(null), 150);
    return () => clearTimeout(t);
  }, [p.text]);
  useEffect(() => { last.current = p.text; void run(null); }, [p.path, p.line]);
  useEffect(() => {
    const onMsg = (e: MessageEvent) => {
      if (e.source !== frame.current?.contentWindow) return;
      const m = e.data || {};
      if (m.type === "si-cw-callout") void run({ kind: "move-callout", step: m.step, index: m.index, at: m.at });
    };
    addEventListener("message", onMsg);
    return () => removeEventListener("message", onMsg);
  }, []);
  // Undo and Redo while the text is hidden (not while typing in a field: the field has its own)
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (!(e.metaKey || e.ctrlKey) || e.key.toLowerCase() !== "z" || /input|textarea|select/i.test((e.target as HTMLElement).tagName)) return;
      e.preventDefault();
      props.current.undo(e.shiftKey);
    };
    addEventListener("keydown", onKey);
    return () => removeEventListener("keydown", onKey);
  }, []);
  // after each redraw: the Studio's light / dark choice, and the step being edited
  const onLoad = () => {
    try {
      const doc = frame.current?.contentDocument;
      doc?.documentElement.setAttribute("data-si-mode", resolvedTheme());
      doc?.querySelectorAll(".si-doc-head, .si-doc-foot").forEach((e) => e.remove()); // only the window
      const fig = doc?.querySelector(".si-cw") as (Element & { siCw?: { show(k: number, a: boolean): void } }) | null;
      fig?.siCw?.show(selRef.current, false);
    } catch { /* not loaded */ }
  };
  useEffect(() => { addEventListener("si-theme", onLoad); return () => removeEventListener("si-theme", onLoad); }, []);
  const go = (i: number) => { setSel(i); selRef.current = i; onLoad(); };

  const steps = resp?.show.steps ?? [];
  const first = steps[0];
  const isWindow = !!first && ("window" in first.attrs || !!resp?.show.id);
  const step = steps[Math.min(sel, steps.length - 1)];
  const lines = step ? step.code.split("\n").length : 0;
  const [newLine, setNewLine] = useState(1);
  const [newText, setNewText] = useState("");
  const exportAs = async (format: string) => {
    setOut("Exporting " + format.toUpperCase() + "…");
    await new Promise((r) => setTimeout(r, 900)); // autosave first: si code reads the saved file
    try {
      const r = await api.codewinExport(p.path, format, resp?.show.id ?? "");
      setOut(r.output.trim() || (r.code ? "Export failed" : "Done"));
    } catch (e) { setOut(e instanceof Error ? e.message : String(e)); }
  };

  return (
    <div class="mapx cwx">
      <div class="mapx-canvas">
        {resp?.html ? <iframe ref={frame} srcdoc={resp.html} title="Code window preview" onLoad={onLoad} />
          : <div class="ed-msg"><div class="ed-msg-card"><b>A plain code block</b><p>Make it a window to give it a title bar, a backdrop and callouts.</p>
            <button class="btn btn-pri" onClick={() => op({ kind: "set", field: "window", value: "mac" })}>Make it a window</button></div></div>}
        {err && <div class="banner banner-err mapx-err" role="alert"><span>{err}</span></div>}
      </div>
      <aside class="mapx-side" aria-label="Window panels">
        <section class="insp">
          <div class="insp-head"><b>Window</b><span class="mute">{resp?.show.id ? "show " + resp.show.id : first?.lang}</span></div>
          <div class="insp-grid">
            {LOOKS.map(([key, label, vals]) => (
              <>
                <label for={"cw-" + key}>{label}</label>
                <select id={"cw-" + key} value={first?.attrs[key] ?? ""} onChange={(e) => op({ kind: "set", field: key, value: e.currentTarget.value })}>
                  <option value="">{key === "window" ? (isWindow ? "none" : "a plain block") : "default"}</option>
                  {vals.map((v) => <option value={v}>{v}</option>)}
                </select>
              </>
            ))}
            <Field id="cw-title" label="Title" value={first?.attrs.title ?? ""} placeholder="retry.go" onSet={(v) => op({ kind: "set", field: "title", value: v })} />
            <span />
            <span class="mapx-checks">
              <label><input type="checkbox" checked={!!first && "numbers" in first.attrs} onChange={(e) => op({ kind: "set", field: "numbers", value: e.currentTarget.checked ? "true" : "" })} /> Line numbers</label>
            </span>
          </div>
        </section>
        <section class="insp">
          <div class="insp-head"><b>Steps</b><span class="mute">{steps.length} step{steps.length === 1 ? "" : "s"}</span></div>
          <ol class="cwx-steps">
            {steps.map((s, i) => (
              <li class={i === sel ? "on" : ""}>
                <button class="btn" aria-pressed={i === sel} onClick={() => go(i)}>{i + 1}</button>
                <Field id={"cw-step-" + i} label="" value={s.attrs.step ?? ""} placeholder={"Step " + (i + 1)} onSet={(v) => op({ kind: "set-step", step: i, field: "step", value: v })} />
                <button class="btn" aria-label={"Move step " + (i + 1) + " up"} disabled={i === 0} onClick={() => { op({ kind: "move-step", step: i, to: i - 1 }); go(i - 1); }}>↑</button>
                <button class="btn" aria-label={"Move step " + (i + 1) + " down"} disabled={i === steps.length - 1} onClick={() => { op({ kind: "move-step", step: i, to: i + 1 }); go(i + 1); }}>↓</button>
                <button class="btn btn-danger" aria-label={"Delete step " + (i + 1)} disabled={steps.length === 1} onClick={() => { op({ kind: "del-step", step: i }); go(Math.max(0, i - 1)); }}>×</button>
              </li>
            ))}
          </ol>
          <div class="mapx-actions">
            <button class="btn" onClick={() => { op({ kind: "add-step", step: sel }); go(sel + 1); }}>Add a step</button>
            <span class="mute">a copy of step {sel + 1}</span>
          </div>
          {step && <StepCode key={sel + ":" + (resp?.line ?? 0)} code={step.code} label={"Code of step " + (sel + 1)} onCommit={(code) => op({ kind: "set-code", step: sel, text: code })} />}
        </section>
        <section class="insp">
          <div class="insp-head"><b>Callouts</b><span class="mute">step {sel + 1}</span></div>
          <ol class="cwx-notes">
            {(step?.callouts ?? []).map((c, i) => (
              <li>
                <Field id={`cw-cl-${sel}-${i}`} label="" value={String(c.line)} onSet={(v) => op({ kind: "edit-callout", step: sel, index: i, line: +v || c.line })} />
                <Field id={`cw-ct-${sel}-${i}`} label="" value={c.text} onSet={(v) => op({ kind: "edit-callout", step: sel, index: i, text: v })} />
                <button class="btn btn-danger" aria-label={"Delete callout " + (i + 1)} onClick={() => op({ kind: "del-callout", step: sel, index: i })}>×</button>
              </li>
            ))}
          </ol>
          {isWindow && step && (
            <div class="cwx-add">
              <label for="cw-new-line">Line</label>
              <select id="cw-new-line" value={newLine} onChange={(e) => setNewLine(+e.currentTarget.value)}>
                {Array.from({ length: lines }, (_, i) => <option value={i + 1}>{i + 1}</option>)}
              </select>
              <input id="cw-new-text" class="field" placeholder="what this line does" value={newText} onInput={(e) => setNewText(e.currentTarget.value)}
                onKeyDown={(e) => { if (e.key === "Enter" && newText.trim()) { op({ kind: "add-callout", step: sel, line: newLine, text: newText }); setNewText(""); } }} />
              <button class="btn" disabled={!newText.trim()} onClick={() => { op({ kind: "add-callout", step: sel, line: newLine, text: newText }); setNewText(""); }}>Add callout</button>
            </div>
          )}
          <p class="mapx-hint">Drag a note in the preview to place it; its place is written as at=.</p>
        </section>
        {isWindow && (
          <section class="insp">
            <div class="insp-head"><b>Export</b><span class="mute">next to the file</span></div>
            <div class="mapx-actions">
              <button class="btn" onClick={() => void exportAs("png")}>PNG</button>
              <button class="btn" onClick={() => void exportAs("svg")}>SVG</button>
              <button class="btn" onClick={() => void exportAs("gif")}>GIF</button>
            </div>
            {out && <pre class="cwx-out">{out}</pre>}
          </section>
        )}
        {resp && resp.problems.length > 0 && <ul class="mapx-problems">{resp.problems.map((x) => <li>{x.msg}</li>)}</ul>}
      </aside>
    </div>
  );
}
