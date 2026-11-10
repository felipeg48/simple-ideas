// "New…": pick what to write (slides, an article, a guide …) and a name; the
// server writes a small starter for that type and never overwrites.
import { useEffect, useRef, useState } from "preact/hooks";
import type { Kind } from "./api";
import { Icon } from "./navigator";

// kind is a starter kind (the server's StarterKinds); icon defaults to it
export const starterKinds: { kind: string; icon?: Kind; label: string; what: string; folder?: boolean }[] = [
  { kind: "deck", label: "Slides", what: "A talk: slides, notes for the teleprompter" },
  { kind: "article", label: "Article", what: "One long read with a lede and contents" },
  { kind: "guide", label: "Guide", what: "A folder of pages in order, with a sidebar", folder: true },
  { kind: "book", label: "Book", what: "Chapters, numbered sections, an index", folder: true },
  { kind: "site", label: "Site", what: "A folder of pages with a menu", folder: true },
  { kind: "resume", icon: "site", label: "Résumé", what: "A website: one page that prints, projects, about", folder: true },
  { kind: "portfolio", icon: "site", label: "Portfolio", what: "A website: project cards, a case study each", folder: true },
  { kind: "landing", icon: "site", label: "Landing page", what: "A website: a hero, features, questions", folder: true },
  { kind: "page", label: "Site page", what: "One plain page for a site" },
  { kind: "lab", label: "Lab", what: "Hands-on steps with commands and output" },
  { kind: "demo", label: "Demo script", what: "What the presenter says and runs, live" },
  { kind: "mindmap", label: "Mind map", what: "Topics that branch out, and maps that open into maps" },
];

/** slug turns a name into a file or folder name: "Our Q3 Plan" → "our-q3-plan". */
export const slug = (s: string) => s.trim().toLowerCase().replace(/\.md$/, "").replace(/[^a-z0-9]+/g, "-").replace(/^-+|-+$/g, "");

export function NewDialog(p: { dir: string; ok(kind: string, path: string, title: string): Promise<void>; onClose(): void }) {
  const [kind, setKind] = useState("deck");
  const [name, setName] = useState("");
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);
  const input = useRef<HTMLInputElement>(null);
  useEffect(() => input.current?.focus(), []);
  const info = starterKinds.find((k) => k.kind === kind)!;
  const s = slug(name);
  const path = s ? (p.dir ? p.dir + "/" : "") + s + (info.folder ? "/" : ".md") : "";
  const submit = async (e: Event) => {
    e.preventDefault();
    if (!s) { setErr("Give it a name."); return; }
    setBusy(true);
    try { await p.ok(kind, path.replace(/\/$/, ""), name.trim()); p.onClose(); } catch (x) { setErr((x as Error).message); setBusy(false); }
  };
  return (
    <div class="scrim" onMouseDown={(e) => e.target === e.currentTarget && p.onClose()}>
      <div class="dialog dialog-wide" role="dialog" aria-modal="true" aria-label="New">
        <form onSubmit={submit} onKeyDown={(e) => e.key === "Escape" && p.onClose()}>
          <h2>New</h2>
          <div class="starter-grid" role="radiogroup" aria-label="What to write">
            {starterKinds.map((k) => (
              <button type="button" role="radio" aria-checked={k.kind === kind} class={"starter" + (k.kind === kind ? " on" : "")}
                onClick={() => { setKind(k.kind); input.current?.focus(); }}>
                <Icon name={k.icon ?? (k.kind as Kind)} class={"k-" + (k.icon ?? k.kind)} />
                <b>{k.label}</b>
                <span>{k.what}</span>
              </button>
            ))}
          </div>
          <label class="starter-name">
            <span>Name</span>
            <input ref={input} class="field" value={name} spellcheck={false} placeholder={info.folder ? "Order lab" : "Retries without the stampede"}
              onInput={(e) => { setName(e.currentTarget.value); setErr(""); }} />
          </label>
          <p class="dialog-body mute">{path ? <>Creates <code>{path}</code>{info.folder ? " with its first pages" : ""}</> : " "}</p>
          {err && <p class="dialog-err" role="alert">{err}</p>}
          <div class="dialog-actions">
            <button type="button" class="btn" onClick={p.onClose}>Cancel</button>
            <button type="submit" class="btn btn-pri" disabled={busy}>Create</button>
          </div>
        </form>
      </div>
    </div>
  );
}
