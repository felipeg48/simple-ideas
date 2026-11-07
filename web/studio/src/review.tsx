// Review: the comments reviewers left on the open file through a review link
// (si serve --review / si play --review), live. Reply, resolve or reopen; the
// comments live in .si/review/ next to the sources.
import { useEffect, useState } from "preact/hooks";

interface Reply { name: string; text: string; time: string }
interface Comment { id: string; anchor: string; quote?: string; name: string; text: string; time: string; resolved?: boolean; replies?: Reply[] }

// a comment sits on a slide, a section or the page, optionally pinned to one block in it (/b4-1a2b3c)
const where = (anchor: string) => {
  const a = anchor.split("/")[0];
  return a === "page" ? "The page" : a.startsWith("slide:") ? "Slide " + a.slice(6) : "Section " + a.slice(8).replace(/-/g, " ");
};

async function post(path: string, body: unknown) {
  const r = await fetch(path, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });
  if (!r.ok) throw new Error((await r.text()).trim() || r.statusText);
}

export function Review(p: { active: string | null; onClose(): void }) {
  const [cs, setCs] = useState<Comment[] | null>(null);
  const [err, setErr] = useState("");
  const [open, setOpen] = useState(false);
  const [draft, setDraft] = useState<Record<string, string>>({});
  const name = (() => { try { return localStorage.getItem("si-review-name") || "Author"; } catch { return "Author"; } })();
  const load = async () => {
    if (!p.active) return;
    try {
      const r = await fetch("/_si/review?page=" + encodeURIComponent(p.active));
      setCs(r.ok ? (await r.json()).comments : []);
    } catch (e) { setErr((e as Error).message); }
  };
  useEffect(() => {
    void load();
    const es = new EventSource("/_si/review/events");
    es.addEventListener("review", (e) => { if (JSON.parse((e as MessageEvent).data) === p.active) void load(); });
    return () => es.close();
  }, [p.active]);
  const act = async (path: string, body: object) => { setErr(""); try { await post(path, { page: p.active, ...body }); await load(); } catch (e) { setErr((e as Error).message); } };
  const shown = (cs || []).filter((c) => open || !c.resolved);
  return (
    <div class="scrim scrim-top" onMouseDown={(e) => e.target === e.currentTarget && p.onClose()}>
      <div class="dialog dialog-wide review" role="dialog" aria-modal="true" aria-labelledby="rev-title" onKeyDown={(e) => e.key === "Escape" && p.onClose()}>
        <h2 id="rev-title">Review{p.active ? <span class="mute"> · {p.active}</span> : null}</h2>
        <p class="dialog-body">Comments from a review link: <code>si serve . --review</code> prints a keyed link for people on your network; what they write appears here as they write it.</p>
        {err && <p class="dialog-err" role="alert">{err}</p>}
        <label class="rev-show"><input type="checkbox" checked={open} onChange={(e) => setOpen(e.currentTarget.checked)} /> Show resolved</label>
        <ul class="rev-list">
          {shown.map((c) => (
            <li class={c.resolved ? "done" : ""}>
              <div class="rev-head"><b>{where(c.anchor)}</b><span class="mute">{c.name} · {c.time.slice(0, 16).replace("T", " ")}</span>
                <button class="btn" type="button" onClick={() => act("/_si/review/resolve", { id: c.id, resolved: !c.resolved })}>{c.resolved ? "Reopen" : "Resolve"}</button></div>
              {c.quote && <q class="rev-quote">{c.quote}</q>}
              <p>{c.text}</p>
              {(c.replies || []).map((r) => <div class="rev-reply"><b>{r.name}</b> <span>{r.text}</span></div>)}
              <form class="rev-form" onSubmit={(e) => { e.preventDefault(); const t = (draft[c.id] || "").trim(); if (t) void act("/_si/review/reply", { id: c.id, name, text: t }).then(() => setDraft({ ...draft, [c.id]: "" })); }}>
                <input class="field" aria-label={"Reply to " + c.name} placeholder="Reply" value={draft[c.id] || ""} onInput={(e) => setDraft({ ...draft, [c.id]: e.currentTarget.value })} />
              </form>
            </li>
          ))}
          {cs && !shown.length && <li class="mute">{p.active ? "No open comments on this file." : "Open a file to see its comments."}</li>}
        </ul>
        <div class="dialog-actions"><button class="btn" type="button" onClick={p.onClose} autoFocus>Close</button></div>
      </div>
    </div>
  );
}
