// The play terminal's Viewer editor: CodeMirror 6 with syntax colours picked
// by file name. Bundled by build.mjs as an IIFE into
// internal/assets/web/term/editor.js; term.js loads it the first time Edit is
// pressed and calls window.SiEditor.create. Syntax colouring and JSON lint
// only, not a language server.
import { EditorState, Extension } from "@codemirror/state";
import {
  EditorView, keymap, lineNumbers, highlightActiveLine, highlightActiveLineGutter,
  drawSelection, highlightSpecialChars, rectangularSelection,
} from "@codemirror/view";
import { defaultKeymap, history, historyKeymap, indentWithTab } from "@codemirror/commands";
import {
  StreamLanguage, StreamParser, HighlightStyle, syntaxHighlighting, bracketMatching,
  indentOnInput, foldGutter, foldKeymap,
} from "@codemirror/language";
import { search, searchKeymap, highlightSelectionMatches } from "@codemirror/search";
import { closeBrackets, closeBracketsKeymap } from "@codemirror/autocomplete";
import { linter, lintGutter } from "@codemirror/lint";
import { tags as t } from "@lezer/highlight";
import { javascript } from "@codemirror/lang-javascript";
import { json, jsonParseLinter } from "@codemirror/lang-json";
import { markdown } from "@codemirror/lang-markdown";
import { css } from "@codemirror/lang-css";
import { html } from "@codemirror/lang-html";
import { yaml } from "@codemirror/legacy-modes/mode/yaml";
import { shell } from "@codemirror/legacy-modes/mode/shell";
import { python } from "@codemirror/legacy-modes/mode/python";
import { go } from "@codemirror/legacy-modes/mode/go";
import { c, cpp, java, kotlin, csharp, scala, dart, objectiveC } from "@codemirror/legacy-modes/mode/clike";
import { rust } from "@codemirror/legacy-modes/mode/rust";
import { ruby } from "@codemirror/legacy-modes/mode/ruby";
import { standardSQL } from "@codemirror/legacy-modes/mode/sql";
import { toml } from "@codemirror/legacy-modes/mode/toml";
import { properties } from "@codemirror/legacy-modes/mode/properties";
import { erlang } from "@codemirror/legacy-modes/mode/erlang";
import { dockerFile } from "@codemirror/legacy-modes/mode/dockerfile";
import { nginx } from "@codemirror/legacy-modes/mode/nginx";
import { lua } from "@codemirror/legacy-modes/mode/lua";
import { powerShell } from "@codemirror/legacy-modes/mode/powershell";
import { protobuf } from "@codemirror/legacy-modes/mode/protobuf";
import { diff } from "@codemirror/legacy-modes/mode/diff";
import { xml } from "@codemirror/legacy-modes/mode/xml";
import { swift } from "@codemirror/legacy-modes/mode/swift";
import { groovy } from "@codemirror/legacy-modes/mode/groovy";
import { perl } from "@codemirror/legacy-modes/mode/perl";
import { r } from "@codemirror/legacy-modes/mode/r";
import { cmake } from "@codemirror/legacy-modes/mode/cmake";

const legacy = (p: StreamParser<unknown>) => StreamLanguage.define(p);

// by extension; names without one are in byName
const byExt: Record<string, () => Extension> = {
  js: () => javascript(), mjs: () => javascript(), cjs: () => javascript(),
  jsx: () => javascript({ jsx: true }),
  ts: () => javascript({ typescript: true }), mts: () => javascript({ typescript: true }), cts: () => javascript({ typescript: true }),
  tsx: () => javascript({ jsx: true, typescript: true }),
  json: () => [json(), linter(jsonParseLinter())], jsonc: () => json(), json5: () => json(),
  md: () => markdown(), markdown: () => markdown(), mdx: () => markdown(),
  css: () => css(), scss: () => css(), less: () => css(),
  html: () => html(), htm: () => html(), vue: () => html(), svelte: () => html(),
  xml: () => legacy(xml), svg: () => legacy(xml), xsd: () => legacy(xml), xsl: () => legacy(xml), plist: () => legacy(xml), pom: () => legacy(xml),
  yaml: () => legacy(yaml), yml: () => legacy(yaml),
  sh: () => legacy(shell), bash: () => legacy(shell), zsh: () => legacy(shell), ksh: () => legacy(shell), fish: () => legacy(shell),
  env: () => legacy(shell), mk: () => legacy(shell), mak: () => legacy(shell),
  py: () => legacy(python), pyi: () => legacy(python), pyw: () => legacy(python),
  go: () => legacy(go),
  java: () => legacy(java), kt: () => legacy(kotlin), kts: () => legacy(kotlin), scala: () => legacy(scala), sc: () => legacy(scala),
  groovy: () => legacy(groovy), gradle: () => legacy(groovy), dart: () => legacy(dart),
  c: () => legacy(c), h: () => legacy(c), cc: () => legacy(cpp), cpp: () => legacy(cpp), cxx: () => legacy(cpp),
  hpp: () => legacy(cpp), hh: () => legacy(cpp), hxx: () => legacy(cpp), m: () => legacy(objectiveC), mm: () => legacy(objectiveC),
  cs: () => legacy(csharp), rs: () => legacy(rust), swift: () => legacy(swift),
  rb: () => legacy(ruby), rake: () => legacy(ruby), gemspec: () => legacy(ruby),
  sql: () => legacy(standardSQL), toml: () => legacy(toml),
  ini: () => legacy(properties), cfg: () => legacy(properties), conf: () => legacy(properties), properties: () => legacy(properties),
  erl: () => legacy(erlang), hrl: () => legacy(erlang), config: () => legacy(erlang), "app.src": () => legacy(erlang),
  dockerfile: () => legacy(dockerFile), lua: () => legacy(lua),
  ps1: () => legacy(powerShell), psm1: () => legacy(powerShell), psd1: () => legacy(powerShell),
  proto: () => legacy(protobuf), diff: () => legacy(diff), patch: () => legacy(diff),
  pl: () => legacy(perl), pm: () => legacy(perl), r: () => legacy(r), cmake: () => legacy(cmake),
};
const byName: Record<string, () => Extension> = {
  dockerfile: () => legacy(dockerFile), containerfile: () => legacy(dockerFile),
  makefile: () => legacy(shell), gnumakefile: () => legacy(shell), ".env": () => legacy(shell),
  ".bashrc": () => legacy(shell), ".zshrc": () => legacy(shell), ".profile": () => legacy(shell), ".bash_profile": () => legacy(shell),
  gemfile: () => legacy(ruby), rakefile: () => legacy(ruby), vagrantfile: () => legacy(ruby),
  jenkinsfile: () => legacy(groovy), "cmakelists.txt": () => legacy(cmake),
  ".gitconfig": () => legacy(properties), ".editorconfig": () => legacy(properties), ".npmrc": () => legacy(properties),
  "nginx.conf": () => legacy(nginx), "go.mod": () => legacy(go), "go.sum": () => legacy(go),
};

export function languageFor(file: string): Extension {
  const base = (file.split(/[\\/]/).pop() || "").toLowerCase();
  if (byName[base]) return byName[base]();
  if (base.startsWith(".env.") || base.endsWith(".env")) return legacy(shell);
  if (base.startsWith("dockerfile.") || base.endsWith(".dockerfile")) return legacy(dockerFile);
  if (/(^|\/)nginx\//.test(file) && base.endsWith(".conf")) return legacy(nginx);
  if (base.endsWith(".app.src")) return legacy(erlang);
  const ext = base.includes(".") ? base.slice(base.lastIndexOf(".") + 1) : "";
  return byExt[ext] ? byExt[ext]() : [];
}

// the drawer's colours (term.js: #0b0e14 ground, the .si-tk-* tokens)
const theme = EditorView.theme({
  "&": { height: "100%", backgroundColor: "#0b0e14", color: "#c7d0e0", fontSize: "12px" },
  ".cm-scroller": { fontFamily: "ui-monospace,SFMono-Regular,Menlo,Consolas,monospace", lineHeight: "1.5" },
  ".cm-content": { caretColor: "#7dd3fc", padding: "6px 0" },
  ".cm-cursor, .cm-dropCursor": { borderLeftColor: "#7dd3fc" },
  "&.cm-focused > .cm-scroller > .cm-selectionLayer .cm-selectionBackground, .cm-selectionBackground, .cm-content ::selection": { backgroundColor: "#264f78" },
  ".cm-gutters": { backgroundColor: "#0b0e14", color: "#3d4760", border: "none" },
  ".cm-activeLineGutter": { backgroundColor: "#11151f", color: "#9aa6bf" },
  ".cm-activeLine": { backgroundColor: "#11151f" },
  ".cm-matchingBracket, &.cm-focused .cm-matchingBracket": { backgroundColor: "#1d2f45", outline: "1px solid #7dd3fc55" },
  ".cm-nonmatchingBracket": { color: "#ff8a8a" },
  ".cm-selectionMatch": { backgroundColor: "#1d2536" },
  ".cm-searchMatch": { backgroundColor: "#5a4a1088", outline: "1px solid #ffd08066" },
  ".cm-searchMatch.cm-searchMatch-selected": { backgroundColor: "#7a5f10" },
  ".cm-panels": { backgroundColor: "#11151f", color: "#c8d0e0", borderTop: "1px solid #1d2536" },
  ".cm-panels.cm-panels-top": { borderBottom: "1px solid #1d2536" },
  ".cm-panel input, .cm-textfield": { background: "#0b0e14", color: "#e6e6e6", border: "1px solid #2a3550", borderRadius: "4px" },
  ".cm-panel button, .cm-button": { backgroundImage: "none", background: "#1b2436", color: "#e6e6e6", border: "1px solid #2a3550", borderRadius: "4px" },
  ".cm-panel label": { color: "#9aa7c0" },
  ".cm-foldPlaceholder": { background: "#1d2536", border: "none", color: "#9aa6bf" },
  ".cm-tooltip": { background: "#11151f", color: "#e6e6e6", border: "1px solid #2a3550" },
  ".cm-diagnostic-error": { borderLeftColor: "#ff8a8a" },
}, { dark: true });

const colours = HighlightStyle.define([
  { tag: [t.keyword, t.controlKeyword, t.operatorKeyword, t.moduleKeyword, t.definitionKeyword, t.modifier], color: "#7dd3fc" },
  { tag: [t.typeName, t.className, t.namespace, t.tagName, t.heading], color: "#7dd3fc" },
  { tag: [t.string, t.special(t.string), t.regexp, t.inserted, t.character], color: "#a5e075" },
  { tag: [t.number, t.bool, t.null, t.atom, t.unit], color: "#f5a05a" },
  { tag: [t.function(t.variableName), t.function(t.propertyName), t.labelName], color: "#f5a05a" },
  { tag: [t.comment, t.lineComment, t.blockComment, t.docComment], color: "#5b6680", fontStyle: "italic" },
  { tag: [t.meta, t.processingInstruction, t.annotation, t.self, t.macroName], color: "#c792ea" },
  { tag: [t.propertyName, t.attributeName, t.definition(t.variableName), t.special(t.variableName)], color: "#ffd080" },
  { tag: [t.deleted, t.invalid], color: "#ff8a8a" },
  { tag: [t.link, t.url], color: "#7dd3fc", textDecoration: "underline" },
  { tag: t.strong, fontWeight: "bold", color: "#e6e6e6" },
  { tag: t.emphasis, fontStyle: "italic" },
  { tag: [t.operator, t.punctuation, t.bracket, t.separator], color: "#9aa6bf" },
]);

type Options = { doc: string; file: string; onChange?: () => void; onSave?: () => void };

function create(parent: HTMLElement, o: Options) {
  const state = () => EditorState.create({
    doc: o.doc,
    extensions: [
      lineNumbers(), highlightActiveLineGutter(), foldGutter(), lintGutter(), highlightSpecialChars(), history(),
      drawSelection(), rectangularSelection(), indentOnInput(), bracketMatching(), closeBrackets(),
      highlightActiveLine(), highlightSelectionMatches(), search({ top: true }), EditorState.tabSize.of(4),
      keymap.of([
        { key: "Mod-s", preventDefault: true, run: () => { o.onSave?.(); return true; } },
        ...closeBracketsKeymap, ...defaultKeymap, ...searchKeymap, ...historyKeymap, ...foldKeymap, indentWithTab,
      ]),
      theme, syntaxHighlighting(colours), languageFor(o.file),
      EditorView.updateListener.of((u) => { if (u.docChanged) o.onChange?.(); }),
    ],
  });
  const view = new EditorView({ state: state(), parent });
  return {
    getText: () => view.state.doc.toString(),
    setText: (text: string) => { o.doc = text; view.setState(state()); }, // a fresh history: Revert and Reload are not undone
    focus: () => view.focus(),
    destroy: () => view.destroy(),
  };
}

(window as unknown as { SiEditor: unknown }).SiEditor = { create, languageFor };
