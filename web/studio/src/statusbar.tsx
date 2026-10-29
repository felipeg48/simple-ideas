import type { Check } from "./api";
import type { DocStatus } from "./editor";
import { Icon } from "./navigator";

const kindLabel: Record<string, string> = {
  deck: "si deck", markdown: "Markdown", svg: "SVG", css: "CSS", json: "JSON", image: "Image", other: "Text",
};
const fmt = (m: number) => (Math.round(m * 10) / 10).toString();

export function StatusBar(p: {
  decks: number; problems: number; check: Check | null; cursor: { line: number; col: number } | null;
  kind: string | null; doc: DocStatus | null; slide: number | null; onProblems(): void;
}) {
  const save = !p.doc ? "" : p.doc.conflict ? "Conflict" : p.doc.error ? "Not saved" : p.doc.saving ? "Saving…" : p.doc.dirty ? "Edited" : "Saved";
  return (
    <footer class="foot" aria-label="Status">
      <button class={"foot-check" + (p.problems ? " bad" : "")} onClick={p.onProblems} title="Show problems">
        <Icon name={p.problems ? "warn" : "ok"} />
        <span>si check</span>
        <b>{p.problems ? `${p.problems} problem${p.problems > 1 ? "s" : ""}` : `${p.decks} deck${p.decks === 1 ? "" : "s"} OK`}</b>
      </button>
      {p.check && (
        <span class="foot-stats">
          {fmt(p.check.minutes)} min · {p.check.steps} steps · {p.check.canvases} canvas{p.check.canvases === 1 ? "" : "es"}
        </span>
      )}
      <span class="spacer" />
      {p.slide && <span>Slide {p.slide}</span>}
      {p.cursor && <span>Ln {p.cursor.line}, Col {p.cursor.col}</span>}
      {p.kind && <span>{kindLabel[p.kind] || "Text"}</span>}
      {save && <span class={"foot-save s-" + save.replace(/\W/g, "").toLowerCase()}><i />{save}</span>}
    </footer>
  );
}
