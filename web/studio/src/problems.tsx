import type { Problem } from "./api";
import { Icon } from "./navigator";

/** Problems is the bottom panel listing si check findings across the workspace's decks. */
export function Problems(p: { problems: Problem[]; errors: string[]; onGo(file: string, line: number): void; onClose(): void }) {
  const n = p.problems.length + p.errors.length;
  return (
    <section class="problems" aria-label="Problems">
      <header>
        <span class="grp">Problems</span>
        <span class={"badge" + (n ? " bad" : "")}>{n}</span>
        <button class="icon-btn" aria-label="Close problems panel" title="Close" onClick={p.onClose}><Icon name="close" /></button>
      </header>
      <ul>
        {p.errors.map((e) => <li class="prob"><Icon name="warn" class="bad" /><span class="msg">{e}</span></li>)}
        {p.problems.map((q) => (
          <li>
            <button class="prob" onClick={() => p.onGo(q.file, q.line)}>
              <Icon name="warn" class="bad" />
              <span class="msg">{q.msg}</span>
              <span class="loc">{q.file}{q.line ? ":" + q.line : ""}</span>
            </button>
          </li>
        ))}
        {!n && <li class="prob ok"><Icon name="ok" class="good" /><span class="msg">No problems. Every deck passes si check.</span></li>}
      </ul>
    </section>
  );
}
