// si-aware Markdown for CodeMirror: directive chips, front-matter keys and
// completion for directive names, front-matter keys, their values (layouts,
// transitions, types, themes, plugin kinds …) and workspace paths.
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

// ==highlight== phrases: tinted like the slide shows them
const hlMark = Decoration.mark({ class: "cm-si-hl" });
/** HL_RE matches a ==highlight== (exactly two = each side, like si's parser). */
export const HL_RE = /(?<![=])==(?=[^=\s])[^=\n]+?==(?!=)/g;
const highlights = (() => {
  const deco = new MatchDecorator({ regexp: HL_RE, decoration: () => hlMark });
  return ViewPlugin.fromClass(
    class {
      decorations: DecorationSet;
      constructor(view: EditorView) { this.decorations = deco.createDeco(view); }
      update(u: ViewUpdate) { this.decorations = deco.updateDeco(u, this.decorations); }
    },
    { decorations: (p) => p.decorations },
  );
})();

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

// The workspace files and the open file, for path completion (set by the app).
let ws = { files: [] as string[], current: "" };
export function setCompletionFiles(files: string[], current: string) { ws = { files, current }; }

/** relPath is the path from the folder of `from` to `to` (both workspace-relative). */
export function relPath(from: string, to: string): string {
  const a = from.split("/").slice(0, -1), b = to.split("/");
  let i = 0;
  while (i < a.length && i < b.length - 1 && a[i] === b[i]) i++;
  return [...Array(a.length - i).fill(".."), ...b.slice(i)].join("/");
}

// which files each path-valued directive or key offers
const pathExt: Record<string, RegExp> = {
  include: /\.(md|adoc|asciidoc|java|py|go|js|ts|tsx|rb|rs|c|cpp|cs|kt|sh|sql|ya?ml|json|toml|xml|html|css)$/i,
  svg: /\.svg$/i, embed: /\.(md|adoc)$|\/$/i, slides: /\.(md|adoc)$/i, code: /\.[\w]+$/i, runbook: /\.json$/i,
  demos: /\.md$|\/$/i, logo: /\.(svg|png|jpe?g|webp|gif)$/i, watermark: /\.(svg|png|jpe?g|webp|gif)$/i, theme: /\.css$/i,
};

/** pathOptions lists workspace files (and folders, for embed and demos) relative to the open file. */
function pathOptions(name: string): Completion[] {
  const ok = pathExt[name];
  if (!ok) return [];
  const dirs = new Set<string>();
  for (const f of ws.files) {
    if (!/\.(md|adoc)$/i.test(f)) continue; // a folder is offered only when it holds pages
    const parts = f.split("/");
    for (let i = 1; i < parts.length; i++) dirs.add(parts.slice(0, i).join("/") + "/");
  }
  return [...ws.files, ...dirs]
    .filter((f) => f !== ws.current && ok.test(f) && !/(^|\/)[._][^/]*$/.test(f) && !/(^|\/)\./.test(f)) // no hidden files, partials or nav files
    .map((f) => relPath(ws.current, f))
    .sort()
    .map((label) => ({ label, type: label.endsWith("/") ? "namespace" : "file" }));
}

/** valueOptions: what can follow `name:` (a directive or a front-matter key). */
function valueOptions(rules: Rules, name: string, typed: string): Completion[] {
  if (name === "layout") return opts(rules.layouts, "enum");
  if (name === "transition") return opts(rules.transitions, "enum");
  if ((name === "alert" || name === "badge") && /·\s*[\w]*$/.test(typed)) return opts(rules.values?.kinds ?? [], "enum");
  const vals = rules.values?.[name];
  if (vals) return opts(vals, "enum");
  if (rules.paths?.includes(name)) return pathOptions(name);
  return [];
}

// Fence lines: languages after ```, then options (code windows, or a mind map's).
const FENCE_LANGS = ["go", "csharp", "java", "javascript", "typescript", "python", "bash", "sql", "yaml", "json", "xml", "html", "css",
  "rust", "kotlin", "c", "cpp", "php", "ruby", "swift", "scala", "powershell", "dockerfile", "toml", "hcl", "protobuf", "graphql",
  "fsharp", "elixir", "mermaid", "stage", "mindmap"];
type OptTable = Record<string, { values?: string[]; flag?: boolean; detail: string }>;
const CODE_OPTS: OptTable = {
  window: { values: ["mac", "windows", "terminal", "none"], detail: "draw the block as a window" },
  title: { detail: "the window or card title: title=\"retry.go\"" },
  theme: { values: ["auto", "night", "paper", "dusk"], detail: "the window's code colours" },
  backdrop: { values: ["dusk", "ocean", "mint", "graphite", "none"], detail: "the background around the window" },
  numbers: { flag: true, detail: "line numbers" },
  hl: { detail: "highlighted lines: hl=2,4-6" },
  id: { detail: "blocks with the same id= are one show" },
  step: { detail: "this step's name in a show" },
  transition: { values: ["morph", "fade", "slide"], detail: "how a show moves between steps" },
};
const MAP_OPTS: OptTable = {
  layout: { values: ["map", "right", "org", "radial"], detail: "how branches are laid out" },
  lines: { values: ["curved", "straight", "orthogonal", "tapered"], detail: "the lines between topics" },
  nodes: { values: ["box", "pill", "underline"], detail: "the shape of topics" },
  theme: { values: ["auto", "harbor", "blueprint", "ink", "garden"], detail: "the map's look" },
  view: { values: ["2d", "3d"], detail: "start in 2D or 3D" },
  camera: { values: ["iso", "perspective"], detail: "the 3D camera" },
  floors: { values: ["on", "off"], detail: "linked maps hover above in 3D" },
  kind: { values: ["blocks", "bricks", "toy"], detail: "the 3D look" },
  steps: { flag: true, detail: "on a slide, one branch per click" },
  flow: { values: ["on", "off"], detail: "dots flowing along the lines" },
};
const INCLUDE_OPTS: OptTable = {
  title: { detail: "the code card's title: title=NuGet.config" },
  lang: { values: FENCE_LANGS.filter((l) => !["mermaid", "stage", "mindmap"].includes(l)), detail: "the language, when the file name does not tell" },
};

/** fenceOpens reports whether line n starts a fenced block (not closes one). */
function fenceOpens(doc: Text, n: number): boolean {
  let open = false;
  for (let i = 1; i < n; i++) if (/^\s*(`{3,}|~{3,})/.test(doc.line(i).text)) open = !open;
  return !open;
}

/** optionsAt completes `name` or `name=value` for the last word typed after the fixed part of a line. */
function optionsAt(table: OptTable, token: string, pos: number): CompletionResult | null {
  const eq = token.indexOf("=");
  if (eq >= 0) {
    const vals = table[token.slice(0, eq)]?.values;
    return vals ? { from: pos - (token.length - eq - 1), options: opts(vals, "enum"), validFor: /^[\w-]*$/ } : null;
  }
  return {
    from: pos - token.length,
    options: Object.entries(table).map(([label, o]) => ({ label, type: "property", detail: o.detail, apply: o.flag ? label : `${label}=` })),
    validFor: /^[\w-]*$/,
  };
}

/** blockLines: the lines of the fenced block that ends just above line n (for callout line numbers). */
function blockLines(doc: Text, n: number): string[] {
  let end = 0;
  for (let i = n - 1; i >= 1; i--) {
    const t = doc.line(i).text;
    if (/^\s*(`{3,}|~{3,})\s*$/.test(t)) { end = i; break; }
    if (!/^\s*<!--\s*callout:/.test(t) && t.trim() !== "") return [];
  }
  if (!end) return [];
  for (let i = end - 1; i >= 1; i--) if (/^\s*(`{3,}|~{3,})/.test(doc.line(i).text)) {
    const out: string[] = [];
    for (let k = i + 1; k < end; k++) out.push(doc.line(k).text);
    return out;
  }
  return [];
}

export function siCompletions(rules: Rules) {
  const names = Object.keys(rules.directives).sort();
  return (ctx: CompletionContext): CompletionResult | null => {
    const line = ctx.state.doc.lineAt(ctx.pos);
    const before = line.text.slice(0, ctx.pos - line.from);
    const fence = /^\s*(?:`{3,}|~{3,})([\w#+.-]*)(.*)$/.exec(before);
    if (fence && fenceOpens(ctx.state.doc, line.number)) {
      if (fence[2] === "") {
        if (!fence[1] && !ctx.explicit) return null;
        return { from: ctx.pos - fence[1].length, options: opts(FENCE_LANGS, "type"), validFor: /^[\w#+.-]*$/ };
      }
      if (!/^\s/.test(fence[2])) return null;
      const token = /(\S*)$/.exec(fence[2])![1];
      return optionsAt(fence[1].toLowerCase() === "mindmap" ? MAP_OPTS : CODE_OPTS, token, ctx.pos);
    }
    let m = /<!--\s*([\w-]+)\s*:(.*?)([^\s·,]*)$/.exec(before);
    if (m && (m[1] === "include" || m[1] === "code") && /\S\s/.test(m[2]) && !(m[2] + m[3]).includes("-->"))
      return optionsAt(INCLUDE_OPTS, m[3], ctx.pos);
    if (m && m[1] === "callout" && !(m[2] + m[3]).includes("-->")) {
      if (/^\s*$/.test(m[2])) {
        const lines = blockLines(ctx.state.doc, line.number);
        return lines.length ? { from: ctx.pos - m[3].length, options: lines.map((t, i) => ({ label: String(i + 1), type: "constant", detail: t.trim().slice(0, 60) })), validFor: /^\d*$/ } : null;
      }
      return /^\s*\d+\s+\S/.test(m[2]) ? { from: ctx.pos - m[3].length, options: [{ label: "at=", type: "property", detail: "where the note sits: at=dx,dy from the line end" }], validFor: /^[\w=]*$/ } : null;
    }
    if (m && rules.directives[m[1]] && !(m[2] + m[3]).includes("-->")) {
      const options = valueOptions(rules, m[1], m[2] + m[3]);
      if (options.length) return { from: ctx.pos - m[3].length, options, validFor: /^[^\s·,]*$/ };
      return null;
    }
    m = /<!--\s*([\w-]*)$/.exec(before);
    if (m) {
      const after = line.text.slice(ctx.pos - line.from);
      const closes = /-->/.test(after);
      return {
        from: ctx.pos - m[1].length,
        options: names.map((n) => ({
          label: n, type: "keyword", detail: rules.help?.[n] ?? rules.directives[n],
          apply: closes ? `${n}: ` : (view: EditorView, _c: Completion, from: number, to: number) => {
            view.dispatch({ changes: { from, to, insert: `${n}:  -->` }, selection: { anchor: from + n.length + 2 } });
          },
        })),
        validFor: /^[\w-]*$/,
      };
    }
    if (inFrontMatter(ctx.state.doc, line.number)) {
      m = /^([\w-]+)\s*:(.*?)([^\s,]*)$/.exec(before);
      if (m) {
        const options = valueOptions(rules, m[1], m[2] + m[3]);
        return options.length ? { from: ctx.pos - m[3].length, options, validFor: /^[^\s,]*$/ } : null;
      }
      m = /^([\w-]*)$/.exec(before);
      if (m && (m[1] || ctx.explicit))
        return { from: line.from, options: opts(rules.metaKeys, "property", (k) => rules.help?.[k] ?? "front matter", (k) => `${k}: `), validFor: /^[\w-]*$/ };
    }
    return null;
  };
}

/** siMarkdown is Markdown plus si directive chips, front-matter styling and si completion. */
export function siMarkdown(rules: Rules): Extension {
  // one source object: the completion plugin tracks sources by identity
  const data = [{ autocomplete: siCompletions(rules) }];
  return [markdown(), chips(rules), highlights, frontMatter, EditorState.languageData.of(() => data)];
}
