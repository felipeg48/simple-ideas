// si-aware Markdown for CodeMirror: directive chips, front-matter keys and
// completion for directive names, layouts, transitions and front-matter keys.
import { EditorState, type Extension, type Text } from "@codemirror/state";
import { Decoration, type DecorationSet, EditorView, MatchDecorator, ViewPlugin, type ViewUpdate } from "@codemirror/view";
import { markdown } from "@codemirror/lang-markdown";
import type { Completion, CompletionContext, CompletionResult } from "@codemirror/autocomplete";
import type { Rules } from "./api";

const chip = (scope: string) => Decoration.mark({ class: `cm-si-chip cm-si-chip-${scope}` });
const chipName = Decoration.mark({ class: "cm-si-chip-name" });
const chipValue = Decoration.mark({ class: "cm-si-chip-value" });
const fmKey = Decoration.mark({ class: "cm-si-fm-key" });
const fmLine = Decoration.line({ class: "cm-si-fm" });

/** frontMatterEnd is the line number of the closing --- of a leading front-matter block, 0 if none. */
function frontMatterEnd(doc: Text): number {
  if (doc.lines < 2 || doc.line(1).text.trim() !== "---") return 0;
  const max = Math.min(doc.lines, 200);
  for (let n = 2; n <= max; n++) if (doc.line(n).text.trim() === "---") return n;
  return 0;
}

/** inFrontMatter: line n lies strictly inside the leading --- block (open blocks count while typing). */
function inFrontMatter(doc: Text, n: number): boolean {
  if (n < 2 || doc.line(1).text.trim() !== "---") return false;
  for (let i = 2; i < n; i++) if (doc.line(i).text.trim() === "---") return false;
  return doc.line(n).text.trim() !== "---";
}

function chips(rules: Rules) {
  const deco = new MatchDecorator({
    regexp: /<!--\s*([a-z][\w-]*)\s*:([^\n]*?)-->/g,
    decorate(add, from, to, m) {
      const scope = rules.directives[m[1]];
      if (!scope) return;
      add(from, to, chip(scope));
      const n = from + m[0].indexOf(m[1]);
      add(n, n + m[1].length, chipName);
      const v = m[2].trim();
      if (v) {
        const vs = from + m[0].indexOf(":") + 1 + m[2].indexOf(v);
        add(vs, vs + v.length, chipValue);
      }
    },
  });
  return ViewPlugin.fromClass(
    class {
      decorations: DecorationSet;
      constructor(view: EditorView) { this.decorations = deco.createDeco(view); }
      update(u: ViewUpdate) { this.decorations = deco.updateDeco(u, this.decorations); }
    },
    { decorations: (p) => p.decorations },
  );
}

// ponytail: recomputes front-matter marks on every doc change; it scans at most 200 lines.
const frontMatter = ViewPlugin.fromClass(
  class {
    decorations: DecorationSet;
    constructor(view: EditorView) { this.decorations = this.build(view.state.doc); }
    update(u: ViewUpdate) { if (u.docChanged) this.decorations = this.build(u.state.doc); }
    build(doc: Text): DecorationSet {
      const end = frontMatterEnd(doc);
      if (!end) return Decoration.none;
      const out = [];
      for (let n = 1; n <= end; n++) {
        const l = doc.line(n);
        out.push(fmLine.range(l.from));
        const m = /^([\w-]+)\s*:/.exec(l.text);
        if (m) out.push(fmKey.range(l.from, l.from + m[1].length));
      }
      return Decoration.set(out, true);
    }
  },
  { decorations: (p) => p.decorations },
);

const opts = (labels: string[], type: string, detail?: (l: string) => string, apply?: (l: string) => string): Completion[] =>
  labels.map((label) => ({ label, type, detail: detail?.(label), apply: apply?.(label) }));

export function siCompletions(rules: Rules) {
  const names = Object.keys(rules.directives).sort();
  return (ctx: CompletionContext): CompletionResult | null => {
    const line = ctx.state.doc.lineAt(ctx.pos);
    const before = line.text.slice(0, ctx.pos - line.from);
    let m = /<!--\s*(layout|transition)\s*:\s*([\w-]*)$/.exec(before);
    if (m) {
      const list = m[1] === "layout" ? rules.layouts : rules.transitions;
      return { from: ctx.pos - m[2].length, options: opts(list, "enum"), validFor: /^[\w-]*$/ };
    }
    m = /<!--\s*([\w-]*)$/.exec(before);
    if (m) {
      const after = line.text.slice(ctx.pos - line.from);
      const closes = /-->/.test(after);
      return {
        from: ctx.pos - m[1].length,
        options: names.map((n) => ({
          label: n, type: "keyword", detail: rules.directives[n],
          apply: closes ? `${n}: ` : (view: EditorView, _c: Completion, from: number, to: number) => {
            view.dispatch({ changes: { from, to, insert: `${n}:  -->` }, selection: { anchor: from + n.length + 2 } });
          },
        })),
        validFor: /^[\w-]*$/,
      };
    }
    if (inFrontMatter(ctx.state.doc, line.number)) {
      m = /^transition\s*:\s*([\w-]*)$/.exec(before);
      if (m) return { from: ctx.pos - m[1].length, options: opts(rules.transitions, "enum"), validFor: /^[\w-]*$/ };
      m = /^([\w-]*)$/.exec(before);
      if (m && (m[1] || ctx.explicit))
        return { from: line.from, options: opts(rules.metaKeys, "property", () => "front matter", (k) => `${k}: `), validFor: /^[\w-]*$/ };
    }
    return null;
  };
}

/** siMarkdown is Markdown plus si directive chips, front-matter styling and si completion. */
export function siMarkdown(rules: Rules): Extension {
  // one source object: the completion plugin tracks sources by identity
  const data = [{ autocomplete: siCompletions(rules) }];
  return [markdown(), chips(rules), frontMatter, EditorState.languageData.of(() => data)];
}
