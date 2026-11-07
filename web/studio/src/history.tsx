// History: the workspace's git, in the Studio. Changed files with their diff,
// a commit of the ticked files, and the open file's history: any version can
// be viewed and restored (as one undoable edit in the editor). Every call goes
// through /api/git/* (confined paths, fixed git arguments).
import { useEffect, useRef, useState } from "preact/hooks";
import { api } from "./api";

interface GitFile { path: string; status: string; staged: boolean }
interface GitCommit { hash: string; short: string; author: string; date: string; subject: string }

export function History(p: { active: string | null; restore(path: string, text: string): boolean; onClose(): void }) {
  const [st, setSt] = useState<{ git: boolean; repo: boolean; branch?: string; files?: GitFile[] } | null>(null);
  const [tab, setTab] = useState<"changes" | "file">("changes");
  const [pick, setPick] = useState<string | null>(null);
  const [diff, setDiff] = useState("");
  const [ticked, setTicked] = useState<Set<string>>(new Set());
  const [msg, setMsg] = useState("");
  const [log, setLog] = useState<GitCommit[]>([]);
  const [rev, setRev] = useState<{ c: GitCommit; text: string } | null>(null);
  const [note, setNote] = useState("");
  const [err, setErr] = useState("");
  const box = useRef<HTMLDivElement>(null);

  const load = async () => {
    try {
      const s = await api.gitStatus();
      setSt(s);
      setTicked(new Set((s.files || []).map((f) => f.path)));
      const first = pick && s.files?.some((f) => f.path === pick) ? pick : s.files?.[0]?.path ?? null;
      setPick(first);
      if (p.active) setLog((await api.gitLog(p.active)).commits);
    } catch (e) { setErr((e as Error).message); }
  };
  useEffect(() => { void load(); box.current?.focus(); }, []);
  useEffect(() => {
    if (!pick) { setDiff(""); return; }
    api.gitDiff(pick).then((r) => setDiff(r.diff), (e) => { setDiff(""); setErr((e as Error).message); });
  }, [pick]);

  const commit = async (e: Event) => {
    e.preventDefault();
    setErr(""); setNote("");
    try {
      const r = await api.gitCommit(msg, [...ticked]);
      setMsg(""); setNote("Committed " + r.commit);
      await load();
    } catch (x) { setErr((x as Error).message); }
  };
  const view = async (c: GitCommit) => {
    try { setRev({ c, text: (await api.gitShow(p.active!, c.hash)).text }); } catch (x) { setErr((x as Error).message); }
  };
  const restore = () => {
    if (!rev || !p.active) return;
    if (p.restore(p.active, rev.text)) { setNote(`${p.active} is back to ${rev.c.short}: unsaved until autosave, ⌘Z undoes it`); setRev(null); }
    else setErr("Open the file in the editor to restore it.");
  };

  return (
    <div class="scrim scrim-top" onMouseDown={(e) => e.target === e.currentTarget && p.onClose()}>
      <div ref={box} tabIndex={-1} class="dialog dialog-wide history" role="dialog" aria-modal="true" aria-labelledby="hist-title"
        onKeyDown={(e) => e.key === "Escape" && (rev ? setRev(null) : p.onClose())}>
        <h2 id="hist-title">History{st?.branch ? <span class="mute"> · {st.branch}</span> : null}</h2>
        {err && <p class="dialog-err" role="alert">{err}</p>}
        {note && <p class="dialog-ok" role="status">{note}</p>}
        {st && !st.git && <p class="dialog-body">History needs git, which this computer does not have.</p>}
        {st && st.git && !st.repo && (
          <div class="dialog-body">
            <p>This folder is not a git repository yet.</p>
            <button class="btn" type="button" onClick={async () => { try { await api.gitInit(); await load(); } catch (x) { setErr((x as Error).message); } }}>Start a repository here</button>
          </div>
        )}
        {st?.repo && (
          <>
            <div class="hist-tabs" role="tablist">
              <button role="tab" aria-selected={tab === "changes"} class={tab === "changes" ? "on" : ""} onClick={() => setTab("changes")}>Changes ({st.files?.length ?? 0})</button>
              <button role="tab" aria-selected={tab === "file"} class={tab === "file" ? "on" : ""} disabled={!p.active} onClick={() => setTab("file")}>{p.active ? "History of " + p.active.split("/").pop() : "No open file"}</button>
            </div>
            {tab === "changes" && (
              <div class="hist-body">
                <ul class="hist-files" aria-label="Changed files">
                  {(st.files || []).map((f) => (
                    <li class={pick === f.path ? "on" : ""}>
                      <input type="checkbox" aria-label={"Commit " + f.path} checked={ticked.has(f.path)}
                        onChange={(e) => { const t = new Set(ticked); if (e.currentTarget.checked) t.add(f.path); else t.delete(f.path); setTicked(t); }} />
                      <button type="button" onClick={() => setPick(f.path)}><span class={"hist-st st-" + f.status}>{f.status[0].toUpperCase()}</span>{f.path}</button>
                    </li>
                  ))}
                  {!st.files?.length && <li class="mute">Nothing changed since the last commit.</li>}
                </ul>
                <pre class="hist-diff" tabIndex={0} aria-label="Changes of the file">{diff.split("\n").map((l) => (
                  <span class={l.startsWith("+") && !l.startsWith("+++") ? "add" : l.startsWith("-") && !l.startsWith("---") ? "del" : l.startsWith("@@") ? "hunk" : ""}>{l + "\n"}</span>
                ))}</pre>
                <form class="hist-commit" onSubmit={commit}>
                  <input class="field" aria-label="Commit message" placeholder="What changed" value={msg} onInput={(e) => setMsg(e.currentTarget.value)} />
                  <button class="btn btn-pri" type="submit" disabled={!msg.trim() || ticked.size === 0}>Commit {ticked.size} file{ticked.size === 1 ? "" : "s"}</button>
                </form>
              </div>
            )}
            {tab === "file" && (
              <div class="hist-body">
                <ul class="hist-log" aria-label="Versions">
                  {log.map((c) => (
                    <li class={rev?.c.hash === c.hash ? "on" : ""}><button type="button" onClick={() => view(c)}>
                      <b>{c.subject}</b><span class="mute">{c.short} · {c.author} · {c.date}</span></button></li>
                  ))}
                  {!log.length && <li class="mute">No commits of this file yet.</li>}
                </ul>
                {rev && (
                  <div class="hist-rev">
                    <pre class="hist-diff" tabIndex={0} aria-label={"The file at " + rev.c.short}>{rev.text}</pre>
                    <button class="btn btn-pri" type="button" onClick={restore}>Restore this version</button>
                  </div>
                )}
              </div>
            )}
          </>
        )}
        <div class="dialog-actions"><button class="btn" type="button" onClick={p.onClose}>Close</button></div>
      </div>
    </div>
  );
}
