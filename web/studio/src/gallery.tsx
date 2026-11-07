// The start gallery: what si can make, each with a live thumbnail of its
// starter (rendered by the server, never written into the workspace). Shown
// on a folder with nothing to edit yet, and from ⌘K → "Start gallery".
import { useEffect, useRef, useState } from "preact/hooks";
import { Icon, type IconName } from "./navigator";
import { slug } from "./new-dialog";

export const galleryKinds: { kind: string; icon: IconName; label: string; what: string; folder?: boolean; name: string }[] = [
  { kind: "deck", icon: "deck", label: "A talk", what: "Slides with a teleprompter script", name: "My talk" },
  { kind: "lab", icon: "lab", label: "A course lab", what: "Hands-on steps with commands", name: "First lab" },
  { kind: "guide", icon: "guide", label: "A tutorial", what: "Pages in order, with a sidebar", folder: true, name: "Tutorial" },
  { kind: "article", icon: "article", label: "An article", what: "One long read with contents", name: "My article" },
  { kind: "site", icon: "site", label: "A site", what: "Pages with a menu and a home", folder: true, name: "Site" },
  { kind: "mindmap", icon: "mindmap", label: "A mind map", what: "Topics that branch, in 2D", name: "Ideas" },
  { kind: "mindmap3d", icon: "mindmap", label: "A 3D mind map", what: "The same map as blocks you fly over", name: "Ideas in 3D" },
  { kind: "codeshow", icon: "article", label: "A code show", what: "Code in a window, step by step", name: "Code walkthrough" },
];

export function Gallery(p: { dir: string; ok(kind: string, path: string, title: string): Promise<void>; onClose(): void }) {
  const [pick, setPick] = useState<string | null>(null);
  const [name, setName] = useState("");
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);
  const input = useRef<HTMLInputElement>(null);
  const first = useRef<HTMLButtonElement>(null);
  useEffect(() => { (pick ? input.current : first.current)?.focus(); }, [pick]);
  const info = galleryKinds.find((k) => k.kind === pick);
  const s = slug(name);
  const path = info && s ? (p.dir ? p.dir + "/" : "") + s + (info.folder ? "" : ".md") : "";
  const choose = (k: (typeof galleryKinds)[number]) => { setPick(k.kind); setName(k.name); setErr(""); };
  const submit = async (e: Event) => {
    e.preventDefault();
    if (!info || !s) { setErr("Give it a name."); return; }
    setBusy(true);
    try { await p.ok(info.kind, path, name.trim()); p.onClose(); } catch (x) { setErr((x as Error).message); setBusy(false); }
  };
  return (
    <div class="scrim scrim-top" onMouseDown={(e) => e.target === e.currentTarget && p.onClose()}>
      <div class="dialog gallery" role="dialog" aria-modal="true" aria-labelledby="gallery-title"
        onKeyDown={(e) => e.key === "Escape" && (pick ? setPick(null) : p.onClose())}>
        <h2 id="gallery-title">What do you want to make?</h2>
        <p class="dialog-body">Pick one to start from. Each is a small, working file you can change; everything is Markdown.</p>
        <div class="gallery-grid">
          {galleryKinds.map((k, i) => (
            <button type="button" ref={i === 0 ? first : undefined} class={"gal" + (pick === k.kind ? " on" : "")}
              aria-pressed={pick === k.kind} onClick={() => choose(k)}>
              <span class="gal-thumb" aria-hidden="true">
                <iframe src={`/_si/studio/api/gallery/${k.kind}?si-view=thumb`} loading="lazy" tabIndex={-1} title="" />
              </span>
              <span class="gal-text"><Icon name={k.icon} class={"k-" + k.icon} /><b>{k.label}</b><span>{k.what}</span></span>
            </button>
          ))}
        </div>
        {info && (
          <form class="gal-form" onSubmit={submit}>
            <label class="starter-name">
              <span>Name of the {info.label.replace(/^an? /, "")}</span>
              <input ref={input} class="field" value={name} spellcheck={false} onInput={(e) => { setName(e.currentTarget.value); setErr(""); }} />
            </label>
            <p class="dialog-body mute">{path ? <>Creates <code>{path}{info.folder ? "/" : ""}</code></> : " "}</p>
            {err && <p class="dialog-err" role="alert">{err}</p>}
            <div class="dialog-actions">
              <button type="button" class="btn" onClick={p.onClose}>Not now</button>
              <button type="submit" class="btn btn-pri" disabled={busy}>Create</button>
            </div>
          </form>
        )}
        {!info && <div class="dialog-actions"><button type="button" class="btn" onClick={p.onClose}>Not now</button></div>}
      </div>
    </div>
  );
}
