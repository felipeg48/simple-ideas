// The centre editor: one CodeMirror view, one EditorState per open file (undo
// history survives tab switches), autosave with etags, crash-safe buffers and
// conflict resolution.
import { useEffect, useRef, useState } from "preact/hooks";
import { Annotation, EditorState, type Extension } from "@codemirror/state";
import {
  EditorView, crosshairCursor, drawSelection, dropCursor, highlightActiveLine, highlightActiveLineGutter,
  highlightSpecialChars, keymap, lineNumbers, rectangularSelection,
} from "@codemirror/view";
import { defaultKeymap, history, historyKeymap, indentWithTab } from "@codemirror/commands";
import { HighlightStyle, bracketMatching, indentOnInput, syntaxHighlighting } from "@codemirror/language";
import { autocompletion, closeBrackets, closeBracketsKeymap, completionKeymap } from "@codemirror/autocomplete";
import { type Diagnostic, lintGutter, lintKeymap, setDiagnostics } from "@codemirror/lint";
import { highlightSelectionMatches, searchKeymap } from "@codemirror/search";
import { tags as t } from "@lezer/highlight";
import { ApiError, api, type Kind, type Problem, type Rules } from "./api";
import { createAutosave } from "./autosave";
import { clear, restore, stash } from "./buffers";
import { siMarkdown } from "./si-lang";
import { under } from "./tabs";

export interface DocStatus { dirty: boolean; saving: boolean; conflict: boolean; deleted: boolean; error?: string }

/** unsaved: the doc holds text that is not on disk (or may not be). */
export const unsaved = (s?: DocStatus) => !!s && (s.dirty || s.conflict || s.deleted || !!s.error);

/** EditorCtl lets the app save or forget documents (a dropped doc keeps any stashed buffer). */
export interface EditorCtl {
  flush(): Promise<void>;
  drop(path: string): void;
  /** unsaved lists open docs at or under path whose text is not safely on disk. */
  unsaved(path: string): string[];
}

interface Doc {
  path: string;
  state: EditorState;
  etag: string;
  saved: string; // text known to be on disk
  banner?: string;
  conflict?: { etag: string; content: string };
  error?: string;
  deleted?: boolean; // the file vanished from disk while it held unsaved text
  saving: boolean;
  chain: Promise<void>;
  auto: ReturnType<typeof createAutosave>;
}

const remote = Annotation.define<boolean>();

const highlight = HighlightStyle.define([
  { tag: t.heading, color: "var(--syn-heading)", fontWeight: "700" },
  { tag: [t.strong], color: "var(--syn-strong)", fontWeight: "700" },
  { tag: [t.emphasis], color: "var(--syn-strong)", fontStyle: "italic" },
  { tag: [t.link, t.url], color: "var(--syn-link)" },
  { tag: [t.monospace, t.string], color: "var(--syn-string)" },
  { tag: t.comment, color: "var(--syn-comment)" },
  { tag: [t.quote], color: "var(--syn-quote)", fontStyle: "italic" },
  { tag: [t.processingInstruction, t.contentSeparator, t.meta], color: "var(--syn-punct)" },
  { tag: [t.number, t.atom, t.bool], color: "var(--syn-number)" },
  { tag: [t.keyword, t.tagName], color: "var(--syn-keyword)" },
  { tag: [t.attributeName, t.propertyName], color: "var(--syn-attr)" },
]);

const theme = EditorView.theme({
  "&": { height: "100%", color: "var(--ink)", backgroundColor: "var(--ed-bg)", fontSize: "13px" },
  ".cm-scroller": { fontFamily: "var(--mono)", lineHeight: "1.7" },
  ".cm-content": { padding: "14px 0 40vh", caretColor: "var(--accent)" },
  ".cm-line": { padding: "0 20px 0 6px" },
  "&.cm-focused": { outline: "none" },
  ".cm-cursor, .cm-dropCursor": { borderLeftColor: "var(--accent)", borderLeftWidth: "2px" },
  "&.cm-focused > .cm-scroller > .cm-selectionLayer .cm-selectionBackground, .cm-selectionBackground, ::selection":
    { backgroundColor: "var(--sel) !important" },
  ".cm-gutters": { backgroundColor: "var(--ed-bg)", color: "var(--gutter)", border: "none" },
  ".cm-lineNumbers .cm-gutterElement": { padding: "0 10px 0 14px", minWidth: "40px" },
  ".cm-activeLine": { backgroundColor: "var(--active-line)" },
  ".cm-activeLineGutter": { backgroundColor: "transparent", color: "var(--ink-2)" },
  ".cm-selectionMatch": { backgroundColor: "var(--match)" },
  ".cm-matchingBracket": { backgroundColor: "var(--match)", outline: "none" },
  ".cm-tooltip": { backgroundColor: "var(--pop)", border: "1px solid var(--line-2)", borderRadius: "10px",
    boxShadow: "var(--shadow-pop)", overflow: "hidden", color: "var(--ink)" },
  ".cm-tooltip-autocomplete > ul": { fontFamily: "var(--mono)", fontSize: "12px", padding: "4px" },
  ".cm-tooltip-autocomplete > ul > li": { padding: "3px 10px !important", borderRadius: "6px" },
  ".cm-tooltip-autocomplete > ul > li[aria-selected]": { backgroundColor: "var(--accent-soft)", color: "var(--ink)" },
  ".cm-completionDetail": { color: "var(--mute)", fontStyle: "normal", marginLeft: "1em" },
  ".cm-completionMatchedText": { textDecoration: "none", color: "var(--accent-ink)", fontWeight: "700" },
  ".cm-panels": { backgroundColor: "var(--panel)", color: "var(--ink)", borderColor: "var(--line)" },
  ".cm-panels.cm-panels-top": { borderBottom: "1px solid var(--line)" },
  ".cm-search input, .cm-search button": { fontFamily: "inherit" },
  ".cm-textfield": { backgroundColor: "var(--field)", border: "1px solid var(--line-2)", borderRadius: "6px", color: "var(--ink)" },
  ".cm-button": { backgroundImage: "none", backgroundColor: "var(--field)", border: "1px solid var(--line-2)", borderRadius: "6px", color: "var(--ink)" },
  ".cm-diagnostic": { padding: "6px 10px", borderLeftWidth: "3px" },
  ".cm-diagnostic-error": { borderLeftColor: "var(--red)" },
  ".cm-lintRange-error": { backgroundImage: "none", textDecoration: "underline wavy var(--red)", textUnderlineOffset: "3px" },
  ".cm-gutter-lint": { width: "14px" },
  ".cm-si-chip": { backgroundColor: "var(--chip-bg)", boxShadow: "inset 0 0 0 1px var(--chip-line)",
    borderRadius: "6px", padding: "1px 0", color: "var(--chip-punct)" },
  ".cm-si-chip-slide": { backgroundColor: "var(--chip2-bg)", boxShadow: "inset 0 0 0 1px var(--chip2-line)" },
  ".cm-si-chip-name": { color: "var(--chip-name)", fontWeight: "700" },
  ".cm-si-chip-value": { color: "var(--chip-value)" },
  // front matter parses as a setext heading in Markdown: undo the heading look
  ".cm-si-fm": { backgroundColor: "var(--fm-bg)" },
  ".cm-si-fm, .cm-si-fm *": { color: "var(--ink-2) !important", fontWeight: "400 !important" },
  ".cm-si-fm .cm-si-fm-key": { color: "var(--syn-attr) !important" },
});

function extensions(kind: Kind, rules: Rules | null, onUpdate: Extension): Extension {
  const md = kind === "deck" || kind === "markdown";
  return [
    lineNumbers(), highlightActiveLineGutter(), highlightSpecialChars(), history(), drawSelection(), dropCursor(),
    EditorState.allowMultipleSelections.of(true), indentOnInput(), bracketMatching(), closeBrackets(),
    rectangularSelection(), crosshairCursor(), highlightActiveLine(), highlightSelectionMatches(),
    autocompletion({ icons: false }), lintGutter(), syntaxHighlighting(highlight), theme,
    md ? EditorView.lineWrapping : [],
    md && rules ? siMarkdown(rules) : [],
    keymap.of([...closeBracketsKeymap, ...defaultKeymap, ...searchKeymap, ...historyKeymap,
      ...completionKeymap, ...lintKeymap, indentWithTab]),
    onUpdate,
  ];
}

const message = (e: unknown) =>
  e instanceof ApiError
    ? e.status === 415 ? "This is a binary file; Studio cannot edit it as text."
      : e.status === 413 ? "This file is too large to edit in Studio."
      : e.status === 404 ? "This file no longer exists."
      : e.message
    : "Cannot reach si: " + String((e as Error)?.message || e);

interface Props {
  path: string | null;
  kind: Kind;
  rules: Rules | null;
  problems: Problem[];
  reload: number;
  reveal: { path: string; line: number; n: number } | null;
  missing: Set<string>;
  onDiscard(path: string): void;
  ctl: { current: EditorCtl | null };
  onCursor(line: number, col: number): void;
  onStatus(path: string, s: DocStatus): void;
}

export function Editor(p: Props) {
  const host = useRef<HTMLDivElement>(null);
  const view = useRef<EditorView | null>(null);
  const docs = useRef(new Map<string, Doc>());
  const failed = useRef(new Map<string, string>());
  const cur = useRef<string | null>(null);
  const props = useRef(p);
  props.current = p;
  const [, bump] = useState(0);
  const rerender = () => bump((n) => n + 1);

  const text = (d: Doc) => (cur.current === d.path && view.current ? view.current.state.doc.toString() : d.state.doc.toString());
  const status = (d: Doc) =>
    props.current.onStatus(d.path, { dirty: text(d) !== d.saved, saving: d.saving, conflict: !!d.conflict, deleted: !!d.deleted, error: d.error });

  async function save(d: Doc) {
    if (d.conflict || d.deleted || !docs.current.has(d.path)) return;
    const body = text(d);
    if (body === d.saved) { clear(d.path); return status(d); }
    d.saving = true;
    status(d);
    try {
      const r = await api.write(d.path, body, d.etag);
      d.etag = r.etag;
      d.saved = body;
      d.error = undefined;
      if (text(d) === body) clear(d.path);
    } catch (e) {
      if (e instanceof ApiError && e.status === 409 && e.body?.etag !== undefined)
        d.conflict = { etag: e.body.etag, content: e.body.content ?? "" };
      else if (e instanceof ApiError && e.status === 404) d.deleted = true; // never recreate silently
      else d.error = message(e);
    }
    d.saving = false;
    status(d);
    rerender();
  }

  /** replace puts disk content into a doc without marking it dirty. */
  function replace(d: Doc, content: string, etag: string) {
    d.etag = etag;
    d.saved = content;
    const st = cur.current === d.path && view.current ? view.current.state : d.state;
    if (st.doc.toString() !== content) {
      const head = Math.min(st.selection.main.head, content.length);
      const tr = st.update({ changes: { from: 0, to: st.doc.length, insert: content }, selection: { anchor: head },
        annotations: remote.of(true) });
      if (cur.current === d.path && view.current) view.current.dispatch(tr);
      else d.state = tr.state;
    }
    clear(d.path);
    status(d);
  }

  function makeDoc(path: string, kind: Kind, content: string, etag: string): Doc {
    const d = { path, etag, saved: content, saving: false, chain: Promise.resolve() } as Doc;
    d.auto = createAutosave(() => (d.chain = d.chain.then(() => save(d))), 700);
    const onUpdate = EditorView.updateListener.of((u) => {
      if (u.docChanged && !u.transactions.some((tr) => tr.annotation(remote))) {
        const body = u.state.doc.toString();
        if (body !== d.saved) stash(path, body, d.etag);
        else clear(path);
        d.auto.touch();
        status(d);
      }
      if (u.selectionSet || u.docChanged) {
        const h = u.state.selection.main.head, l = u.state.doc.lineAt(h);
        props.current.onCursor(l.number, h - l.from + 1);
      }
    });
    d.state = EditorState.create({ doc: content, extensions: extensions(kind, props.current.rules, onUpdate) });
    return d;
  }

  async function refresh(d: Doc) {
    if (d.conflict || text(d) !== d.saved) return;
    try {
      const f = await api.read(d.path);
      if (f.etag !== d.etag && text(d) === d.saved) replace(d, f.content, f.etag);
    } catch { /* a deleted file is reported by the tree; keep the buffer */ }
  }

  function show(d: Doc) {
    const v = view.current!;
    v.setState(d.state);
    applyDiagnostics();
    const h = d.state.selection.main.head, l = d.state.doc.lineAt(h);
    props.current.onCursor(l.number, h - l.from + 1);
  }

  async function open(path: string, kind: Kind) {
    let d = docs.current.get(path);
    if (d) { show(d); void refresh(d); return; }
    try {
      const f = await api.read(path);
      failed.current.delete(path);
      const st = restore(path);
      let banner: string | undefined;
      let content = f.content;
      if (st && st.text !== f.content) { content = st.text; banner = "Unsaved changes restored"; }
      else if (st) clear(path);
      d = makeDoc(path, kind, content, f.etag);
      d.saved = f.content;
      if (banner) { d.etag = st!.etag; d.banner = banner; d.auto.touch(); }
      docs.current.set(path, d);
    } catch (e) {
      failed.current.set(path, message(e));
    }
    if (cur.current === path) {
      if (d) show(d);
      if (d) status(d);
      rerender();
    }
  }

  function applyDiagnostics() {
    const v = view.current, path = cur.current;
    if (!v || !path || !docs.current.has(path)) return;
    const doc = v.state.doc;
    const diags: Diagnostic[] = props.current.problems.filter((q) => q.file === path).map((q) => {
      const l = doc.line(Math.min(Math.max(q.line, 1), doc.lines));
      return { from: l.from, to: l.to, severity: "error", message: q.msg, source: "si check" };
    });
    v.dispatch(setDiagnostics(v.state, diags));
  }

  // one view for the component's life
  useEffect(() => {
    const v = new EditorView({ parent: host.current!, state: EditorState.create({ extensions: theme }) });
    view.current = v;
    const flushCurrent = () => { const d = cur.current && docs.current.get(cur.current); if (d) void d.auto.flush(); };
    v.contentDOM.addEventListener("blur", flushCurrent);
    const onHide = () => document.visibilityState === "hidden" && flushCurrent();
    document.addEventListener("visibilitychange", onHide);
    p.ctl.current = {
      async flush() { await Promise.all([...docs.current.values()].map((d) => d.auto.flush().then(() => d.chain))); },
      drop(path) {
        for (const k of [...docs.current.keys()]) if (under(k, path)) docs.current.delete(k);
      },
      unsaved(path) {
        return [...docs.current.values()]
          .filter((d) => under(d.path, path) && (text(d) !== d.saved || d.conflict || d.deleted || d.error))
          .map((d) => d.path);
      },
    };
    return () => { document.removeEventListener("visibilitychange", onHide); v.destroy(); };
  }, []);

  // switch documents: save the one we leave, show (or load) the next
  useEffect(() => {
    const prev = cur.current && docs.current.get(cur.current);
    if (prev && view.current) { prev.state = view.current.state; void prev.auto.flush(); }
    cur.current = p.path;
    if (p.path) void open(p.path, p.kind);
    rerender();
  }, [p.path]);

  useEffect(applyDiagnostics, [p.problems, p.path]);

  // files gone from disk while holding unsaved text: say so instead of saving
  useEffect(() => {
    for (const d of docs.current.values()) {
      const gone = p.missing.has(d.path);
      if (gone !== !!d.deleted) { d.deleted = gone; status(d); }
    }
    rerender();
  }, [p.missing]);

  useEffect(() => {
    const d = p.path && docs.current.get(p.path);
    if (d && p.reload) void refresh(d);
  }, [p.reload]);

  useEffect(() => {
    const r = p.reveal, v = view.current;
    if (!r || !v || r.path !== cur.current || !docs.current.has(r.path)) return;
    const l = v.state.doc.line(Math.min(Math.max(r.line, 1), v.state.doc.lines));
    v.dispatch({ selection: { anchor: l.from }, effects: EditorView.scrollIntoView(l.from, { y: "center" }) });
    v.focus();
  }, [p.reveal, docs.current.has(p.reveal?.path ?? "")]);

  const d = p.path ? docs.current.get(p.path) : undefined;
  const err = p.path ? failed.current.get(p.path) : undefined;
  const loading = p.path && !d && !err;
  return (
    <div class="ed-host">
      {d?.conflict && (
        <div class="banner banner-warn" role="alert">
          <span><b>Changed on disk.</b> Someone else edited this file while you were typing.</span>
          <button class="btn btn-pri" onClick={() => { d.etag = d.conflict!.etag; d.conflict = undefined; void (d.chain = d.chain.then(() => save(d))); rerender(); }}>Keep mine</button>
          <button class="btn" onClick={() => { const c = d.conflict!; d.conflict = undefined; replace(d, c.content, c.etag); rerender(); }}>Use disk version</button>
        </div>
      )}
      {d?.deleted && (
        <div class="banner banner-err" role="alert">
          <span><b>Deleted on disk.</b> Your text is only in Studio now.</span>
          <button class="btn btn-pri" onClick={() => { d.deleted = false; d.etag = ""; void (d.chain = d.chain.then(() => save(d))); rerender(); }}>Save it again</button>
          <button class="btn" onClick={() => p.onDiscard(d.path)}>Discard</button>
        </div>
      )}
      {d?.banner && !d.conflict && (
        <div class="banner banner-info" role="status">
          <span>{d.banner}</span>
          <button class="btn" onClick={() => { d.banner = undefined; rerender(); }}>Dismiss</button>
        </div>
      )}
      {d?.error && !d.conflict && !d.deleted && (
        <div class="banner banner-err" role="alert">
          <span>Not saved: {d.error}</span>
          <button class="btn" onClick={() => { d.error = undefined; void (d.chain = d.chain.then(() => save(d))); }}>Retry</button>
        </div>
      )}
      {err && <div class="ed-msg"><div class="ed-msg-card"><b>Cannot open {p.path}</b><p>{err}</p></div></div>}
      {loading && <div class="ed-msg"><div class="spinner" aria-label="Loading" /></div>}
      <div class="ed-cm" ref={host} hidden={!d} />
    </div>
  );
}
