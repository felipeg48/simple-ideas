// Build: runs `si build` on the workspace into html/ and shows what it printed.
import { useState } from "preact/hooks";
import { api } from "./api";
import { Icon } from "./navigator";

export function BuildDialog(p: { onClose(): void }) {
  const [pdf, setPdf] = useState(false);
  const [busy, setBusy] = useState(false);
  const [res, setRes] = useState<{ code: number; output: string; url: string } | null>(null);
  const [err, setErr] = useState("");
  const run = async (e: Event) => {
    e.preventDefault();
    setBusy(true); setErr(""); setRes(null);
    try { setRes(await api.build(pdf)); } catch (x) { setErr((x as Error).message); }
    setBusy(false);
  };
  return (
    <div class="scrim" onMouseDown={(e) => e.target === e.currentTarget && !busy && p.onClose()}>
      <div class="dialog dialog-wide" role="dialog" aria-modal="true" aria-label="Build">
        <form onSubmit={run} onKeyDown={(e) => e.key === "Escape" && !busy && p.onClose()}>
          <h2>Build</h2>
          <p class="dialog-body">Writes every deck, document, guide, book and site in this folder as self-contained HTML into <code>html/</code>.</p>
          <label class="check-row"><input id="build-pdf" type="checkbox" checked={pdf} onChange={(e) => setPdf(e.currentTarget.checked)} /> Also write a PDF per deck and book (needs Chrome)</label>
          {busy && <p class="dialog-body" role="status">Building…</p>}
          {res && (
            <>
              <p class={res.code ? "dialog-err" : "dialog-ok"} role="status">
                <Icon name={res.code ? "warn" : "ok"} />{res.code ? "Built with problems (see the log)." : "Built."}
                {!res.code && <> <a href={res.url} target="_blank" rel="noopener">Open the built index</a></>}
              </p>
              <pre class="build-log">{res.output.trim()}</pre>
            </>
          )}
          {err && <p class="dialog-err" role="alert">{err}</p>}
          <div class="dialog-actions">
            <button type="button" class="btn" disabled={busy} onClick={p.onClose}>Close</button>
            <button type="submit" class="btn btn-pri" disabled={busy}>{res ? "Build again" : "Build"}</button>
          </div>
        </form>
      </div>
    </div>
  );
}
