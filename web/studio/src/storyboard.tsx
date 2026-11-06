// The storyboard: the whole deck at a glance, one column per topic (the same
// topics as the deck's `o` overview), a time ribbon of the notes minutes, and
// every slide as a card that jumps to its source.
import type { Check, Slide } from "./api";
import { Icon } from "./navigator";

const fmt = (m: number) => (Math.round(m * 10) / 10).toString();

export function Storyboard(p: { check: Check | null; current: number | null; title: string; onGo(file: string, line: number): void }) {
  const slides = p.check?.slides ?? [];
  if (!slides.length) return <div class="board board-empty">Open a slide deck to see its storyboard.</div>;
  const topics: Slide[][] = [];
  for (const s of slides) (topics[s.topic] ??= []).push(s);
  const groups = topics.filter(Boolean);
  const total = slides.reduce((a, s) => a + (s.minutes || 0), 0) || 1;
  const mins = (g: Slide[]) => g.reduce((a, s) => a + (s.minutes || 0), 0);
  return (
    <div class="board" aria-label={`Storyboard of ${p.title}`}>
      <div class="ribbon" role="img" aria-label={`${fmt(total)} minutes over ${groups.length} topics`}>
        {groups.map((g) => (
          <span style={{ flexGrow: Math.max(mins(g), 0.2) }} title={`${g[0].title || "Topic"}: ${fmt(mins(g))} min`}>
            <b>{g[0].title || "Untitled"}</b><i>{fmt(mins(g))} min</i>
          </span>
        ))}
      </div>
      <div class="board-cols">
        {groups.map((g) => (
          <section class="board-col">
            <h3>{g[0].title || "Untitled"}<span>{g.length} slide{g.length > 1 ? "s" : ""} · {fmt(mins(g))} min</span></h3>
            {g.map((s) => (
              <button class={"card" + (s.index === p.current ? " on" : "")} onClick={() => p.onGo(s.file, s.line)} title={`${s.file}:${s.line}`}>
                <span class="card-n">{s.index}</span>
                <span class="card-t">{s.title || <i class="mute">no heading</i>}</span>
                <span class="card-meta">
                  {s.layout && <em>{s.layout}</em>}
                  {s.minutes ? <span>{fmt(s.minutes)} min</span> : null}
                  {s.steps ? <span>{s.steps} build{s.steps > 1 ? "s" : ""}</span> : null}
                  {!s.words && <span class="warn"><Icon name="warn" />no script</span>}
                </span>
              </button>
            ))}
          </section>
        ))}
      </div>
    </div>
  );
}
