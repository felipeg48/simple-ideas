// The inspector: the slide under the cursor, its layout, transition,
// background, classes and eyebrow (edits written back as directives), and its
// notes, cues and build steps.
import { useEffect, useState } from "preact/hooks";
import type { Rules, Slide } from "./api";

const fmt = (m: number) => (Math.round(m * 10) / 10).toString();

export function Inspector(p: { slide: Slide | null; rules: Rules | null; onSet(name: string, value: string): void }) {
  const s = p.slide;
  if (!s) return <div class="insp insp-empty">Put the cursor in a slide to inspect it.</div>;
  return (
    <div class="insp" aria-label={`Slide ${s.index}`}>
      <div class="insp-head"><b>Slide {s.index}</b><span class="mute">{s.title || "no heading"}</span></div>
      <div class="insp-grid">
        <label for="insp-layout">Layout</label>
        <select id="insp-layout" value={s.layout || ""} onChange={(e) => p.onSet("layout", e.currentTarget.value)}>
          <option value="">default</option>
          {p.rules?.layouts.map((l) => <option value={l}>{l}</option>)}
        </select>
        <label for="insp-transition">Transition</label>
        <select id="insp-transition" value={s.transition || ""} onChange={(e) => p.onSet("transition", e.currentTarget.value)}>
          <option value="">deck default</option>
          {p.rules?.transitions.map((l) => <option value={l}>{l}</option>)}
        </select>
        <Field id="insp-background" label="Background" value={s.background || ""} placeholder="a colour, gradient or image" onSet={(v) => p.onSet("background", v)} />
        <Field id="insp-class" label="Classes" value={s.class || ""} placeholder="dark tight" onSet={(v) => p.onSet("class", v)} />
        <Field id="insp-eyebrow" label="Eyebrow" value={s.eyebrow || ""} placeholder="a small label over the heading" onSet={(v) => p.onSet("eyebrow", v)} />
      </div>
      <dl class="insp-facts">
        <dt>Time</dt><dd>{s.minutes ? fmt(s.minutes) + " min" : "—"}</dd>
        <dt>Script</dt><dd>{s.words ? `${s.words} words` : "no Notes:"}</dd>
        <dt>Builds</dt><dd>{s.steps ? `${s.steps} step directive${s.steps > 1 ? "s" : ""}` : "none"}</dd>
        <dt>Cues</dt><dd>{s.clicks ? `${s.clicks} [CLICK]` : "none"}</dd>
      </dl>
    </div>
  );
}

/** Field is a text input that writes its value back on Enter or when it loses focus. */
function Field(p: { id: string; label: string; value: string; placeholder: string; onSet(v: string): void }) {
  const [v, setV] = useState(p.value);
  useEffect(() => setV(p.value), [p.value]);
  const commit = () => { if (v.trim() !== p.value) p.onSet(v); };
  return (
    <>
      <label for={p.id}>{p.label}</label>
      <input id={p.id} class="field" value={v} placeholder={p.placeholder} spellcheck={false}
        onInput={(e) => setV(e.currentTarget.value)} onBlur={commit}
        onKeyDown={(e) => { if (e.key === "Enter") { e.preventDefault(); commit(); } if (e.key === "Escape") setV(p.value); }} />
    </>
  );
}
