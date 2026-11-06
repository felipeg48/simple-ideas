// ⌘K / Ctrl+K: one box to open any file or run any Studio command.
import { useEffect, useMemo, useRef, useState } from "preact/hooks";
import { Icon, type IconName } from "./navigator";

export interface Command { id: string; label: string; hint?: string; icon: IconName; run(): void }

/** fuzzy scores label against q (letters in order); higher is better, -1 is no match. */
export function fuzzy(label: string, q: string): number {
  const l = label.toLowerCase(), s = q.toLowerCase().replace(/\s+/g, "");
  if (!s) return 0;
  const at = l.indexOf(s);
  if (at >= 0) return 1000 - at - l.length / 100; // a plain substring wins
  let i = 0, score = 0, last = -2;
  for (let j = 0; j < l.length && i < s.length; j++) {
    if (l[j] === s[i]) { score += last === j - 1 ? 3 : 1; last = j; i++; }
  }
  return i === s.length ? score - l.length / 100 : -1;
}

export function Palette(p: { commands: Command[]; onClose(): void }) {
  const [q, setQ] = useState("");
  const [sel, setSel] = useState(0);
  const input = useRef<HTMLInputElement>(null);
  const list = useRef<HTMLOListElement>(null);
  useEffect(() => input.current?.focus(), []);
  const hits = useMemo(() => p.commands
    .map((c) => ({ c, s: Math.max(fuzzy(c.label, q), c.hint ? fuzzy(c.hint, q) - 1 : -1) }))
    .filter((x) => x.s >= 0)
    .sort((a, b) => b.s - a.s)
    .slice(0, 50)
    .map((x) => x.c), [q, p.commands]);
  useEffect(() => setSel(0), [q]);
  useEffect(() => list.current?.children[sel]?.scrollIntoView({ block: "nearest" }), [sel]);
  const go = (c?: Command) => { if (!c) return; p.onClose(); c.run(); };
  return (
    <div class="scrim scrim-top" onMouseDown={(e) => e.target === e.currentTarget && p.onClose()}>
      <div class="palette" role="dialog" aria-modal="true" aria-label="Command palette">
        <input ref={input} class="palette-in" value={q} placeholder="Open a file or run a command…" spellcheck={false}
          role="combobox" aria-expanded="true" aria-controls="palette-list" aria-activedescendant={hits[sel] ? "pal-" + hits[sel].id : undefined}
          onInput={(e) => setQ(e.currentTarget.value)}
          onKeyDown={(e) => {
            if (e.key === "Escape") { e.preventDefault(); p.onClose(); }
            else if (e.key === "ArrowDown") { e.preventDefault(); setSel((s) => Math.min(s + 1, hits.length - 1)); }
            else if (e.key === "ArrowUp") { e.preventDefault(); setSel((s) => Math.max(s - 1, 0)); }
            else if (e.key === "Enter") { e.preventDefault(); go(hits[sel]); }
          }} />
        <ol id="palette-list" ref={list} class="palette-list" role="listbox">
          {hits.map((c, i) => (
            <li id={"pal-" + c.id} role="option" aria-selected={i === sel} class={i === sel ? "on" : ""}
              onMouseMove={() => setSel(i)} onClick={() => go(c)}>
              <Icon name={c.icon} class={"k-" + c.icon} /><span class="t">{c.label}</span>{c.hint && <span class="h">{c.hint}</span>}
            </li>
          ))}
          {!hits.length && <li class="none">Nothing matches.</li>}
        </ol>
      </div>
    </div>
  );
}
