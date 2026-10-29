// The "New runbook" generator: a four-step dialog (Runbook, Commands, Link,
// Review) that writes a canonical runbook JSON file and can link decks to it.
// The file it writes is plain JSON: it opens in the Visual view afterwards and
// can be edited by hand in the JSON view like any other runbook.
import type { JSX } from "preact";
import { useEffect, useMemo, useRef, useState } from "preact/hooks";
import { ApiError, api, type DeckInfo, type Node } from "./api";
import { CommandField } from "./placeholder-input";
import { Icon } from "./navigator";
import { type Runbook, linkRunbook, placeholderMatches, placeholders, relPath, serializeRunbook, slugify, validate } from "./runbook";

type Target = "" | "bash" | "sql";
interface WEntry { label: string; description: string; command: string; target: Target }
interface WCat { name: string; entries: WEntry[] }

const blankEntry = (target: Target = ""): WEntry => ({ label: "", description: "", command: "", target });
const isBlank = (e: WEntry) => !e.label.trim() && !e.description.trim() && !e.command.trim();
const join = (dir: string, name: string) => (dir ? dir + "/" + name : name);
const EXISTS = "A file with this name already exists";
const steps = ["Runbook", "Commands", "Link", "Review"];

function dirs(n: Node | null, out: string[] = []): string[] {
  if (n?.kind === "dir") { out.push(n.path); n.children?.forEach((c) => dirs(c, out)); }
  return out;
}
function has(n: Node | null, path: string): boolean {
  return !!n && (n.path === path || !!n.children?.some((c) => has(c, path)));
}

/** build turns the wizard state into a runbook; blank entries are dropped, empty optional fields omitted. */
function build(name: string, desc: string, cats: WCat[]): Runbook {
  return {
    ...(name.trim() && { name: name.trim() }),
    ...(desc.trim() && { description: desc.trim() }),
    categories: cats.map((c) => ({
      name: c.name.trim(),
      entries: c.entries.filter((e) => !isBlank(e)).map((e) => ({
        label: e.label.trim(),
        ...(e.description.trim() && { description: e.description.trim() }),
        command: e.command.trim(),
        ...(e.target && { target: e.target }),
        extra: {},
      })),
      extra: {},
    })),
    extra: {},
  };
}

/** highlightJson colours keys, strings, numbers and the {{placeholders}} inside strings. */
function highlightJson(text: string): (string | JSX.Element)[] {
  const out: (string | JSX.Element)[] = [];
  const re = /("(?:[^"\\]|\\.)*")(\s*:)?|-?\d+(?:\.\d+)?(?:[eE][+-]?\d+)?|\btrue\b|\bfalse\b|\bnull\b/g;
  let i = 0, k = 0;
  for (const m of text.matchAll(re)) {
    if (m.index! > i) out.push(text.slice(i, m.index));
    if (m[2]) out.push(<span class="j-key" key={k++}>{m[1]}</span>, m[2]);
    else if (m[1]) {
      const parts: (string | JSX.Element)[] = [];
      let j = 0;
      for (const p of placeholderMatches(m[1])) {
        parts.push(m[1].slice(j, p.index), <span class="j-ph" key={p.index}>{p[0]}</span>);
        j = p.index! + p[0].length;
      }
      parts.push(m[1].slice(j));
      out.push(<span class="j-str" key={k++}>{parts}</span>);
    } else out.push(<span class="j-num" key={k++}>{m[0]}</span>);
    i = m.index! + m[0].length;
  }
  out.push(text.slice(i));
  return out;
}

export function RunbookWizard(p: { tree: Node | null; decks: DeckInfo[]; dir: string; onClose(): void; onCreated(path: string): Promise<void> }) {
  const [step, setStep] = useState(0);
  const [name, setName] = useState("");
  const [desc, setDesc] = useState("");
  const [dir, setDir] = useState(p.dir);
  const [file, setFile] = useState("");
  const [fileEdited, setFileEdited] = useState(false);
  const [cats, setCats] = useState<WCat[]>([{ name: "", entries: [blankEntry()] }]);
  const [links, setLinks] = useState<Set<string>>(() => new Set());
  const [tried, setTried] = useState(-1); // the step whose Next was pressed: show its problems
  const [fileErr, setFileErr] = useState("");
  const [err, setErr] = useState("");
  const [linkErrs, setLinkErrs] = useState<string[] | null>(null);
  const [busy, setBusy] = useState(false);
  const creating = useRef(false); // a fast double Enter must not fire two creates
  const created = useRef(false); // the file is on disk: from now on only "open it" is left
  const box = useRef<HTMLDivElement>(null);
  const focusNext = useRef<string | null>(null);

  const folders = useMemo(() => dirs(p.tree), [p.tree]);
  const fname = fileEdited ? file : (slugify(name) || "runbook") + ".json";
  const path = join(dir, fname.trim());
  const rb = useMemo(() => build(name, desc, cats), [name, desc, cats]);
  const json = useMemo(() => serializeRunbook(rb), [rb]);
  const names = useMemo(() => [...new Set(cats.flatMap((c) => c.entries.flatMap((e) => placeholders(e.command).map((x) => x.name))))], [cats]);
  const count = rb.categories.reduce((n, c) => n + c.entries.length, 0);

  // focus: a freshly added row, else the first field of the step
  useEffect(() => {
    const el = box.current?.querySelector<HTMLElement>(focusNext.current ?? ".wiz-body input, .wiz-body select, .wiz-body textarea, .wiz-create");
    focusNext.current = null;
    el?.focus();
  }, [step, cats.length, cats.map((c) => c.entries.length).join()]);

  // problems per step; field keys mark the inputs, messages are listed
  const problems = (s: number): { key: string; msg: string }[] => {
    const out: { key: string; msg: string }[] = [];
    if (s === 0) {
      if (!name.trim()) out.push({ key: "name", msg: "Give the runbook a name." });
      const f = fname.trim();
      if (!/^\w[\w. -]*\.json$/i.test(f)) out.push({ key: "file", msg: "Use letters, digits, spaces, . _ and - in the file name, ending in .json." });
      else if (has(p.tree, path)) out.push({ key: "file", msg: EXISTS });
    }
    if (s === 1) {
      cats.forEach((c, ci) => {
        if (!c.name.trim()) out.push({ key: `c${ci}`, msg: `Category ${ci + 1} needs a name.` });
        if (c.entries.every(isBlank)) out.push({ key: `c${ci}`, msg: `${c.name.trim() || `Category ${ci + 1}`} has no commands yet.` });
        c.entries.forEach((e, ei) => {
          if (isBlank(e)) return;
          if (!e.label.trim()) out.push({ key: `c${ci}e${ei}l`, msg: `A command in ${c.name.trim() || `category ${ci + 1}`} needs a label.` });
          if (!e.command.trim()) out.push({ key: `c${ci}e${ei}c`, msg: `${e.label.trim() || "An entry"} needs a command.` });
        });
      });
      for (const v of validate(rb)) if (v.msg.startsWith("duplicate")) out.push({ key: "", msg: v.msg[0].toUpperCase() + v.msg.slice(1) + "." });
    }
    return out;
  };
  const shown = tried === step ? problems(step) : [];
  const bad = (key: string) => shown.some((x) => x.key === key);
  if (step === 0 && fileErr && !shown.some((x) => x.msg === EXISTS)) shown.push({ key: "file", msg: fileErr });

  // a failed Next puts the caret in the first field that needs fixing
  useEffect(() => { if (tried >= 0) box.current?.querySelector<HTMLElement>(".wiz-body .invalid")?.focus(); }, [tried]);

  const go = (s: number) => { setTried(-1); setErr(""); setStep(s); };
  const next = () => {
    if (busy) return;
    if (step === 3) { void (created.current ? finish() : create()); return; }
    if (problems(step).length) { setTried(step); return; }
    go(step + 1);
  };

  // open the new file; a failure here leaves the dialog usable with a message
  const finish = async () => {
    setBusy(true);
    try { await p.onCreated(path); p.onClose(); } catch (e) { setErr(`It could not be opened: ${(e as Error).message}`); setLinkErrs((l) => l ?? []); setBusy(false); }
  };

  const create = async () => {
    if (creating.current) return;
    creating.current = true;
    try { await createOnce(); } finally { creating.current = false; }
  };
  const createOnce = async () => {
    setBusy(true);
    setErr("");
    try {
      await api.create(path, false, json);
      created.current = true;
    } catch (e) {
      setBusy(false);
      if (e instanceof ApiError && e.status === 409) { setFileErr(EXISTS); go(0); setTried(0); return; }
      setErr(e instanceof ApiError ? e.message : "Cannot reach si.");
      return;
    }
    // link the chosen decks: read, edit the front matter, write back only if nobody changed it meanwhile
    const fails: string[] = [];
    for (const d of links) {
      try {
        const f = await api.read(d);
        const text = linkRunbook(f.content, relPath(d, path));
        if (text !== f.content) await api.write(d, text, f.etag);
      } catch (e) {
        fails.push(e instanceof ApiError && e.status === 409 ? `${d} changed on disk while linking, so it was left alone.` : `${d}: ${e instanceof Error ? e.message : String(e)}`);
      }
    }
    if (fails.length) { setLinkErrs(fails); setBusy(false); return; }
    await finish();
  };

  const setCat = (ci: number, f: (c: WCat) => WCat) => setCats((cs) => cs.map((c, i) => (i === ci ? f(c) : c)));
  const setEntry = (ci: number, ei: number, patch: Partial<WEntry>) =>
    setCat(ci, (c) => ({ ...c, entries: c.entries.map((e, i) => (i === ei ? { ...e, ...patch } : e)) }));
  const addEntry = (ci: number) => {
    focusNext.current = `[data-k="c${ci}e${cats[ci].entries.length}l"]`;
    setCat(ci, (c) => ({ ...c, entries: [...c.entries, blankEntry(c.entries[c.entries.length - 1]?.target)] }));
  };
  const addCat = () => {
    focusNext.current = `[data-k="c${cats.length}"]`;
    setCats((cs) => [...cs, { name: "", entries: [blankEntry()] }]);
  };

  const onKey = (e: KeyboardEvent) => {
    const t = e.target as HTMLElement;
    if (t.closest(".ph-pop")) return; // the placeholder popover handles its own keys
    if (e.key === "Escape") { e.preventDefault(); p.onClose(); return; }
    if (e.key !== "Enter" || e.shiftKey || e.altKey) return;
    const mod = e.metaKey || e.ctrlKey;
    if (mod || (t.tagName === "INPUT" && !e.isComposing)) { e.preventDefault(); next(); }
  };

  const ph = <span class="wiz-tok">{"{{name}}"}</span>;
  const stepBody = [
    // 1. Runbook
    <div class="wiz-grid">
      <label class="wiz-f wide">
        <span class="rb-flabel">Name</span>
        <input class={"field plain" + (bad("name") ? " invalid" : "")} aria-label="Runbook name" value={name} placeholder="e.g. Lab ops"
          spellcheck={false} autocomplete="off" onInput={(e) => { setName(e.currentTarget.value); setFileErr(""); }} />
      </label>
      <label class="wiz-f wide">
        <span class="rb-flabel">Description <span class="mute">optional</span></span>
        <input class="field plain" aria-label="Description" value={desc} placeholder="What these commands are for"
          autocomplete="off" onInput={(e) => setDesc(e.currentTarget.value)} />
      </label>
      <label class="wiz-f">
        <span class="rb-flabel">Location</span>
        <select class="field plain" aria-label="Location" value={dir} onChange={(e) => { setDir(e.currentTarget.value); setFileErr(""); }}>
          {folders.map((d) => <option value={d} key={d}>{d ? d + "/" : `${p.tree?.name ?? "workspace"} (root)`}</option>)}
        </select>
      </label>
      <label class="wiz-f">
        <span class="rb-flabel">File name</span>
        <input class={"field" + (bad("file") || fileErr ? " invalid" : "")} aria-label="File name" value={fname} spellcheck={false} autocomplete="off"
          onInput={(e) => { setFile(e.currentTarget.value); setFileEdited(true); setFileErr(""); }} />
      </label>
      <p class="wiz-path wide"><Icon name="json" class="k-json" /><code>{path || "…"}</code></p>
    </div>,

    // 2. Commands
    <div class="wiz-cmds">
      <p class="wiz-tip">
        <span class="wiz-tip-ico" aria-hidden="true">{"{{ }}"}</span>
        <span>Write {ph} where a value is asked each time the command runs, or <span class="wiz-tok">{"{{name:default}}"}</span> to prefill it. <b>Insert placeholder</b> writes one at the cursor; names you reuse are offered again.</span>
      </p>
      {cats.map((c, ci) => (
        <section class="wiz-cat" key={ci} aria-label={c.name || `Category ${ci + 1}`}>
          <div class="wiz-cat-head">
            <label class="wiz-f grow">
              <span class="rb-flabel">{ci === 0 ? "First category" : "Category"}</span>
              <input data-k={`c${ci}`} class={"field plain" + (bad(`c${ci}`) ? " invalid" : "")} aria-label="Category name" value={c.name}
                placeholder={ci === 0 ? "e.g. Health" : "Category name"} autocomplete="off" onInput={(e) => { const v = e.currentTarget.value; setCat(ci, (x) => ({ ...x, name: v })); }} />
            </label>
            {cats.length > 1 && (
              <button type="button" class="icon-btn" aria-label={`Remove category ${c.name || ci + 1}`} title="Remove category"
                onClick={() => setCats((cs) => cs.filter((_, i) => i !== ci))}><Icon name="trash" /></button>
            )}
          </div>
          {c.entries.map((e, ei) => (
            <div class="wiz-entry" key={ei}>
              <div class="wiz-entry-grid">
                <label class="wiz-f">
                  <span class="rb-flabel">Label</span>
                  <input data-k={`c${ci}e${ei}l`} class={"field plain" + (bad(`c${ci}e${ei}l`) ? " invalid" : "")} aria-label="Label" value={e.label}
                    placeholder="e.g. Alarms" autocomplete="off" onInput={(ev) => setEntry(ci, ei, { label: ev.currentTarget.value })} />
                </label>
                <label class="wiz-f">
                  <span class="rb-flabel">Target</span>
                  <select class="field plain" aria-label="Target" value={e.target} onChange={(ev) => setEntry(ci, ei, { target: ev.currentTarget.value as Target })}>
                    <option value="">default (bash)</option>
                    <option value="bash">bash</option>
                    <option value="sql">sql</option>
                  </select>
                </label>
                {c.entries.length > 1 && (
                  <button type="button" class="icon-btn wiz-del" aria-label={`Remove ${e.label || `command ${ei + 1}`}`} title="Remove command"
                    onClick={() => setCat(ci, (x) => ({ ...x, entries: x.entries.filter((_, i) => i !== ei) }))}><Icon name="trash" /></button>
                )}
                <label class="wiz-f wide">
                  <span class="rb-flabel">Description <span class="mute">optional</span></span>
                  <input class="field plain" aria-label="Entry description" value={e.description} autocomplete="off"
                    placeholder="Shown under the label in the drawer" onInput={(ev) => setEntry(ci, ei, { description: ev.currentTarget.value })} />
                </label>
                <div class="wide">
                  <CommandField value={e.command} names={names} invalid={bad(`c${ci}e${ei}c`)} onChange={(v) => setEntry(ci, ei, { command: v })} />
                </div>
              </div>
            </div>
          ))}
          <button type="button" class="btn wiz-add" onClick={() => addEntry(ci)}><Icon name="plus" />Add another</button>
        </section>
      ))}
      <button type="button" class="btn wiz-add" onClick={addCat}><Icon name="plus" />New category</button>
    </div>,

    // 3. Link
    <div class="wiz-link">
      <p class="dialog-body">Decks that use this runbook open it in the presenter's command drawer. Checked decks get <code>runbook:</code> in their front matter (an existing <code>runbook:</code> line is replaced).</p>
      {p.decks.length === 0 ? <p class="wiz-empty mute">No decks in this workspace yet. You can link one later with a <code>runbook:</code> line.</p> : (
        <ul class="wiz-decks">
          {p.decks.map((d) => (
            <li key={d.path}>
              <label class={"wiz-deck" + (links.has(d.path) ? " on" : "")}>
                <input type="checkbox" aria-label={`Link ${d.path}`} checked={links.has(d.path)}
                  onChange={(e) => { const on = e.currentTarget.checked; setLinks((s) => { const n = new Set(s); on ? n.add(d.path) : n.delete(d.path); return n; }); }} />
                <Icon name="deck" class="k-deck" />
                <span class="t">{d.title || d.path}</span>
                <span class="p">{d.path}</span>
                {links.has(d.path) && <code class="wiz-fm">runbook: {relPath(d.path, path)}</code>}
              </label>
            </li>
          ))}
        </ul>
      )}
    </div>,

    // 4. Review
    <div class="wiz-review">
      <div class="wiz-sum">
        <span><Icon name="json" class="k-json" /><code>{path}</code></span>
        <span>{rb.categories.length} {rb.categories.length === 1 ? "category" : "categories"} · {count} {count === 1 ? "command" : "commands"}</span>
        {names.length > 0 && <span class="wiz-phs">{names.map((n) => <span class="ph-name" key={n}>{n}</span>)}</span>}
        {links.size > 0 && <span>Links {[...links].join(", ")}</span>}
      </div>
      <pre class="wiz-json" aria-label="Runbook JSON" tabIndex={0}>{highlightJson(json)}</pre>
      <p class="mute wiz-after">The file opens in the Visual view; switch to <b>JSON</b> there to edit it by hand.</p>
    </div>,
  ];

  return (
    <div class="scrim wiz-scrim">
      <div class="dialog wiz" role="dialog" aria-modal="true" aria-label="New runbook" ref={box} onKeyDown={onKey}>
        <header class="wiz-head">
          <h2><Icon name="terminal" />New runbook</h2>
          <ol class="wiz-steps">
            {steps.map((s, i) => (
              <li key={s} class={i === step ? "on" : i < step ? "done" : ""}>
                <button type="button" disabled={i >= step || busy || !!linkErrs} aria-current={i === step ? "step" : undefined} onClick={() => go(i)}>
                  <span class="n">{i < step ? <Icon name="ok" /> : i + 1}</span>{s}
                </button>
              </li>
            ))}
          </ol>
        </header>
        <div class="wiz-body">
          {linkErrs ? (
            <div class="wiz-done">
              <p><Icon name="ok" class="ok" /><b>Created <code>{path}</code>.</b></p>
              {linkErrs.length > 0 && <>
                <p class="dialog-err" role="alert">Some decks were not linked:</p>
                <ul class="wiz-errs">{linkErrs.map((m) => <li key={m}>{m}</li>)}</ul>
              </>}
            </div>
          ) : stepBody[step]}
        </div>
        {(shown.length > 0 || err) && (
          <ul class="wiz-errs" role="alert">
            {[...new Set(shown.map((x) => x.msg)), ...(err ? [err] : [])].map((m) => <li key={m}><Icon name="warn" />{m}</li>)}
          </ul>
        )}
        <footer class="wiz-foot">
          <span class="wiz-keys mute"><kbd>Enter</kbd> next · <kbd>Esc</kbd> cancel</span>
          {linkErrs ? (
            <button type="button" class="btn btn-pri" disabled={busy} onClick={() => void finish()}>Open runbook</button>
          ) : <>
            <button type="button" class="btn" onClick={p.onClose}>Cancel</button>
            {step > 0 && <button type="button" class="btn" disabled={busy} onClick={() => go(step - 1)}>Back</button>}
            <button type="button" class={"btn btn-pri" + (step === 3 ? " wiz-create" : "")} disabled={busy} onClick={next}>
              {step === 3 ? (busy ? "Creating…" : "Create") : "Next"}
            </button>
          </>}
        </footer>
      </div>
    </div>
  );
}
