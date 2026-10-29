// The command field of a runbook entry: a wrapping monospace textarea plus an
// "Insert placeholder" popover that writes {{name}} or {{name:default}} at the caret.
import { useEffect, useRef, useState } from "preact/hooks";
import { insertPlaceholder } from "./runbook";
import { Icon } from "./navigator";

const nameRe = /^[A-Za-z_][A-Za-z0-9_-]*$/;

export function CommandField(p: { value: string; names: string[]; invalid?: boolean; onChange(v: string): void }) {
  const area = useRef<HTMLTextAreaElement>(null);
  const caret = useRef<number | null>(null);
  const [pop, setPop] = useState(false);

  // grow with the text: a command is read whole, never scrolled inside a box
  useEffect(() => {
    const a = area.current;
    if (!a) return;
    a.style.height = "auto";
    a.style.height = a.scrollHeight + 2 + "px";
  }, [p.value]);

  const insert = (name: string, def: string) => {
    const a = area.current!;
    const at = caret.current ?? a.value.length;
    const r = insertPlaceholder(p.value, at, name, def);
    p.onChange(r.cmd);
    setPop(false);
    requestAnimationFrame(() => { a.focus(); a.setSelectionRange(r.cursor, r.cursor); });
  };

  return (
    <div class="cmd-field">
      <div class="cmd-head">
        <span class="rb-flabel">Command</span>
        <div class="ph-anchor">
          <button type="button" class="btn ph-btn" aria-haspopup="dialog" aria-expanded={pop}
            onMouseDown={() => { caret.current = area.current?.selectionStart ?? null; }}
            onClick={() => { if (caret.current === null) caret.current = area.current?.selectionStart ?? null; setPop(!pop); }}>
            <span class="ph-glyph" aria-hidden="true">{"{{ }}"}</span>Insert placeholder
          </button>
          {pop && <PlaceholderPop names={p.names} onInsert={insert} onClose={() => { setPop(false); area.current?.focus(); }} />}
        </div>
      </div>
      <textarea ref={area} class={"field cmd-area" + (p.invalid ? " invalid" : "")} aria-label="Command" rows={1} spellcheck={false}
        autocapitalize="off" autocomplete="off" value={p.value}
        onInput={(e) => p.onChange(e.currentTarget.value)}
        onBlur={(e) => { caret.current = e.currentTarget.selectionStart; }} />
    </div>
  );
}

function PlaceholderPop(p: { names: string[]; onInsert(name: string, def: string): void; onClose(): void }) {
  const [name, setName] = useState("");
  const [def, setDef] = useState("");
  const first = useRef<HTMLInputElement>(null);
  const box = useRef<HTMLDivElement>(null);
  useEffect(() => {
    first.current?.focus();
    box.current?.scrollIntoView({ block: "nearest" });
    const away = (e: MouseEvent) => { if (box.current && !box.current.contains(e.target as Node) && !(e.target as Element).closest?.(".ph-btn")) p.onClose(); };
    document.addEventListener("mousedown", away);
    return () => document.removeEventListener("mousedown", away);
  }, []);
  const err = !name ? "" : !nameRe.test(name) ? "Letters, digits, _ and -; not starting with a digit." : def.includes("}") ? "The default cannot contain }." : "";
  const token = name ? (def ? `{{${name}:${def}}}` : `{{${name}}}`) : "";
  const used = p.names.filter((n) => n !== name && n.toLowerCase().startsWith(name.toLowerCase())).slice(0, 6);
  return (
    <div ref={box} class="ph-pop" role="dialog" aria-label="Insert placeholder">
    <form onSubmit={(e) => { e.preventDefault(); if (name && !err) p.onInsert(name, def); }}
      onKeyDown={(e) => { if (e.key === "Escape") { e.stopPropagation(); p.onClose(); } }}>
      <label class="ph-row">
        <span class="rb-flabel">Name</span>
        <input ref={first} class="field" aria-label="Placeholder name" list="ph-names" value={name} spellcheck={false}
          autocomplete="off" placeholder="e.g. node" onInput={(e) => setName(e.currentTarget.value.trim())} />
        <datalist id="ph-names">{p.names.map((n) => <option value={n} key={n} />)}</datalist>
      </label>
      {used.length > 0 && (
        <div class="ph-used" aria-label="Names used in this runbook">
          {used.map((n) => <button type="button" class="ph-name" key={n} onClick={() => { setName(n); first.current?.focus(); }}>{n}</button>)}
        </div>
      )}
      <label class="ph-row">
        <span class="rb-flabel">Default <span class="mute">optional</span></span>
        <input class="field" aria-label="Default value" value={def} spellcheck={false} autocomplete="off"
          placeholder="asked with this value prefilled" onInput={(e) => setDef(e.currentTarget.value)} />
      </label>
      <div class="ph-out">
        {token ? <code class="ph-preview">{token}</code> : <span class="mute">The token appears here</span>}
      </div>
      {err && <p class="rb-issue" role="alert"><Icon name="warn" />{err}</p>}
      <div class="ph-actions">
        <button type="button" class="btn" onClick={p.onClose}>Cancel</button>
        <button type="submit" class="btn btn-pri" disabled={!name || !!err}>Insert</button>
      </div>
    </form>
    </div>
  );
}
