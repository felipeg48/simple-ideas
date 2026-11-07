// A short guided tour of the Studio: six stops, each a ring around a part of
// the window and a note beside it. Skippable (Esc), remembered once done.
import { useEffect, useLayoutEffect, useState } from "preact/hooks";

export const tourStops: { sel: string; title: string; text: string }[] = [
  { sel: ".nav", title: "Your files", text: "Every deck, article, guide and mind map in this folder. Right-click for New, Rename and Delete." },
  { sel: ".ed", title: "Write", text: "Plain Markdown with completion. A mind map opens in Map view and a code block offers Window view: switch above the text." },
  { sel: ".prev-col", title: "The preview", text: "Updates as you type. Click a slide to jump to its line; the inspector edits a slide's layout and background." },
  { sel: ".modes", title: "Rehearse and Present", text: "Rehearse puts the deck beside its teleprompter; Present goes full screen with the teleprompter in another window." },
  { sel: ".build-btn", title: "Build", text: "Writes self-contained HTML (and PDFs) into a folder you can send or host anywhere." },
  { sel: ".brand", title: "⌘K finds anything", text: "Press ⌘K (Ctrl+K) to open any file or run any command, including this tour and the start gallery." },
];

const DONE = "si-tour-done";
export const tourDone = () => { try { return localStorage.getItem(DONE) === "1"; } catch { return true; } };
const markDone = () => { try { localStorage.setItem(DONE, "1"); } catch { /* private window */ } };

export function Tour(p: { onClose(): void }) {
  const [i, setI] = useState(0);
  const [box, setBox] = useState<DOMRect | null>(null);
  const stop = tourStops[i];
  const end = () => { markDone(); p.onClose(); };
  useLayoutEffect(() => {
    const el = document.querySelector(stop.sel) as HTMLElement | null;
    const place = () => setBox(el && el.offsetParent !== null ? el.getBoundingClientRect() : null);
    place();
    window.addEventListener("resize", place);
    return () => window.removeEventListener("resize", place);
  }, [i]);
  useEffect(() => {
    const key = (e: KeyboardEvent) => {
      if (e.key === "Escape") { e.preventDefault(); end(); }
      else if (e.key === "ArrowRight") setI((n) => Math.min(tourStops.length - 1, n + 1));
      else if (e.key === "ArrowLeft") setI((n) => Math.max(0, n - 1));
    };
    window.addEventListener("keydown", key);
    return () => window.removeEventListener("keydown", key);
  }, []);
  const pad = 6;
  const ring = box ? { left: box.left - pad, top: box.top - pad, width: box.width + pad * 2, height: box.height + pad * 2 } : null;
  // the note sits below the ring when there is room, else above, else in the middle
  const vw = window.innerWidth, vh = window.innerHeight, nw = Math.min(340, vw - 24);
  let note: { left: number; top: number } = { left: (vw - nw) / 2, top: vh / 2 - 80 };
  if (ring) {
    const left = Math.max(12, Math.min(vw - nw - 12, ring.left));
    if (ring.top + ring.height + 170 < vh) note = { left, top: ring.top + ring.height + 12 };
    else if (ring.top > 190) note = { left, top: ring.top - 178 };
  }
  return (
    <div class="tour" role="dialog" aria-modal="true" aria-labelledby="tour-title" aria-describedby="tour-text">
      {ring ? <div class="tour-ring" style={{ left: ring.left + "px", top: ring.top + "px", width: ring.width + "px", height: ring.height + "px" }} />
        : <div class="tour-dim" />}
      <div class="tour-note" style={{ left: note.left + "px", top: note.top + "px", width: nw + "px" }}>
        <span class="tour-k">{i + 1} of {tourStops.length}</span>
        <h2 id="tour-title">{stop.title}</h2>
        <p id="tour-text">{stop.text}</p>
        <div class="dialog-actions">
          <button class="btn" onClick={end}>Skip tour</button>
          {i > 0 && <button class="btn" onClick={() => setI(i - 1)}>Back</button>}
          {i < tourStops.length - 1
            ? <button class="btn btn-pri" autoFocus onClick={() => setI(i + 1)}>Next</button>
            : <button class="btn btn-pri" autoFocus onClick={end}>Done</button>}
        </div>
      </div>
    </div>
  );
}
