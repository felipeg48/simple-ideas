// The Video view (a `type: video` edit): the si play preview of the editor's
// text, with the timeline strip and a marker per line, above the text. A
// marker (or a callout) clicked selects its line in the editor; the editor's
// cursor line is marked in the preview. A callout dragged on the frame comes
// back from the page as a move; Go writes it into its line (POST video) and
// the new text goes into the editor as one undoable change. ⌘B builds the
// saved file (si build) and plays the result here.
import { useEffect, useRef, useState } from "preact/hooks";
import { api } from "./api";
import { resolvedTheme } from "./theme";

/** isVideoEdit: Markdown whose front matter says type: video. */
export function isVideoEdit(text: string): boolean {
  if (!text.startsWith("---")) return false;
  const end = text.indexOf("\n---", 3);
  const fm = end < 0 ? text.slice(0, 4000) : text.slice(0, end);
  return /^type:\s*video\s*(#.*)?$/m.test(fm);
}

interface Resp { text: string; html: string; out: string }
interface Build { running: boolean; code: number; output: string }

export function VideoEditor(p: { path: string; text: string; line: number; setText(next: string): void; undo(redo: boolean): void;
  select(line: number): void; flush(): Promise<void> }) {
  const frame = useRef<HTMLIFrameElement>(null);
  const [html, setHtml] = useState("");
  const [out, setOut] = useState("");
  const [err, setErr] = useState("");
  const [build, setBuild] = useState<Build | null>(null);
  const [built, setBuilt] = useState(""); // the built video's URL, once built
  const last = useRef("");
  const props = useRef(p);
  props.current = p;
  const keep = useRef<{ t: number } | null>(null);

  const video = () => { try { return frame.current?.contentDocument?.querySelector("video") ?? null; } catch { return null; } };
  const show = (r: Resp) => {
    const v = video();
    if (v) keep.current = { t: v.currentTime };
    last.current = r.text; setOut(r.out); setHtml(r.html); setErr("");
  };
  const draw = async (move: unknown) => {
    try {
      const r = await api.video(props.current.path, last.current || props.current.text, move) as Resp;
      if (move) props.current.setText(r.text);
      show(r);
    } catch (e) { setErr(e instanceof Error ? e.message : String(e)); }
  };

  // the editor's text changed (typing, Undo): draw it again, after a pause
  useEffect(() => {
    if (p.text === last.current) return;
    last.current = p.text;
    const t = setTimeout(() => void draw(null), 400);
    return () => clearTimeout(t);
  }, [p.text]);
  useEffect(() => { last.current = p.text; setBuilt(""); setBuild(null); void draw(null); }, [p.path]);
  const tell = (seek: boolean) => { try { frame.current?.contentWindow?.postMessage({ type: "si-ve-line", line: props.current.line, seek }, "*"); } catch { /* not loaded */ } };
  useEffect(() => tell(true), [p.line]);
  useEffect(() => {
    const onMsg = (e: MessageEvent) => {
      if (e.source !== frame.current?.contentWindow) return;
      const m = e.data || {};
      if (m.type === "si-ve-select" && m.line) props.current.select(m.line);
      if (m.type === "si-ve-move") void draw({ line: m.line, to: m.to, at: m.at, from: m.from, rect: m.rect });
    };
    addEventListener("message", onMsg);
    return () => removeEventListener("message", onMsg);
  }, []);

  const runBuild = async () => {
    setBuilt(""); setBuild({ running: true, code: 0, output: "Saving…\n" });
    await props.current.flush();
    try {
      let b = await api.videoBuild(props.current.path) as Build;
      setBuild(b);
      while (b.running) {
        await new Promise((r) => setTimeout(r, 600));
        b = await api.videoBuildState(props.current.path) as Build;
        setBuild(b);
      }
      if (b.code === 0 && out) setBuilt("/" + out.split("/").map(encodeURIComponent).join("/") + "?v=" + Date.now());
    } catch (e) { setBuild({ running: false, code: 1, output: e instanceof Error ? e.message : String(e) }); }
  };
  const buildRef = useRef(runBuild);
  buildRef.current = runBuild;
  // ⌘B builds; ⌘Z / ⇧⌘Z undo through the editor (not while typing in the text: it has its own)
  const onKey = (e: KeyboardEvent) => {
    if (!(e.metaKey || e.ctrlKey)) return;
    const k = e.key.toLowerCase(), t = e.target as HTMLElement | null;
    if (k === "b") { e.preventDefault(); e.stopPropagation(); if (!build?.running) void buildRef.current(); return; }
    if (k === "z" && !t?.closest?.(".cm-editor") && !/input|textarea|select/i.test(t?.tagName ?? "")) { e.preventDefault(); props.current.undo(e.shiftKey); }
  };
  const keyRef = useRef(onKey);
  keyRef.current = onKey;
  useEffect(() => {
    const h = (e: KeyboardEvent) => keyRef.current(e);
    addEventListener("keydown", h, true);
    return () => removeEventListener("keydown", h, true);
  }, []);
  const onLoad = () => {
    try {
      const w = frame.current?.contentWindow, doc = frame.current?.contentDocument;
      doc?.documentElement.setAttribute("data-si-mode", resolvedTheme());
      w?.addEventListener("keydown", (e) => keyRef.current(e), true);
      const v = video(), k = keep.current;
      if (v && k) { const go = () => { v.currentTime = k.t; }; if (v.readyState >= 1) go(); else v.addEventListener("loadedmetadata", go, { once: true }); }
      keep.current = null;
      tell(!k);
    } catch { /* not loaded */ }
  };

  const tail = (build?.output ?? "").trim().split("\n").slice(-6).join("\n");
  return (
    <div class="mapx vex">
      <div class="mapx-canvas">
        {html ? <iframe ref={frame} srcdoc={html} title="Video preview" onLoad={onLoad} /> : <div class="ed-msg"><div class="spinner" aria-label="Loading" /></div>}
        {err && <div class="banner banner-err mapx-err" role="alert"><span>{err}</span></div>}
      </div>
      <aside class="mapx-side" aria-label="Video panels">
        <section class="insp">
          <div class="insp-head"><b>Build</b><span class="mute">{out}</span></div>
          <div class="mapx-actions">
            <button class="btn btn-pri vex-build" disabled={!!build?.running} onClick={() => void runBuild()}>{build?.running ? "Building…" : "Build video"}</button>
            <span class="mute"><kbd>⌘</kbd><kbd>B</kbd></span>
          </div>
          {build && <pre class={"cwx-out vex-log" + (!build.running && build.code ? " vex-fail" : "")} role="status">{tail || (build.running ? "Building…" : "")}</pre>}
          {built && <video class="vex-result" src={built} controls preload="metadata" aria-label="The built video" />}
        </section>
        <section class="insp">
          <div class="insp-head"><b>Edit on the frame</b></div>
          <p class="mapx-hint">Pause, then drag a callout into place: a bubble's tail, a text, a rectangle (its corner resizes it), an arrow by either end. The place is written into its line in source pixels.</p>
          <p class="mapx-hint">Click a marker or a line under the player to select that line in the text; the line under the cursor is marked in the player.</p>
        </section>
      </aside>
    </div>
  );
}
