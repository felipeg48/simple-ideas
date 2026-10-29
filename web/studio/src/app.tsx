// The Studio shell: top bar with the mode spine, navigator, tabbed editor,
// live preview, problems panel and status bar.
import type { JSX } from "preact";
import { useCallback, useEffect, useMemo, useRef, useState } from "preact/hooks";
import { ApiError, api, type Check, type DeckInfo, type Kind, type Node, type Problem, type Rules, deckURL } from "./api";
import { clear } from "./buffers";
import { type DocStatus, Editor, type EditorCtl } from "./editor";
import { Icon, type IconName, Navigator } from "./navigator";
import { Preview } from "./preview";
import { Problems } from "./problems";
import { slideForCursor } from "./slidemap";
import { StatusBar } from "./statusbar";
import { type ThemeMode, applyTheme, storedTheme } from "./theme";

type CheckResult = Check | { error: string };
const isCheck = (c: CheckResult | undefined): c is Check => !!c && !("error" in c);

const extKind = (path: string): Kind => {
  const e = path.split(".").pop()!.toLowerCase();
  if (e === "md" || e === "markdown") return "markdown";
  if (e === "svg") return "svg";
  if (e === "css") return "css";
  if (e === "json") return "json";
  if (["png", "jpg", "jpeg", "gif", "webp", "avif"].includes(e)) return "image";
  return "other";
};
const baseName = (p: string) => p.slice(p.lastIndexOf("/") + 1);
const fileURL = (p: string) => "/" + p.split("/").map(encodeURIComponent).join("/");

function flatten(n: Node | null, out = new Map<string, Node>()) {
  if (!n) return out;
  out.set(n.path, n);
  n.children?.forEach((c) => flatten(c, out));
  return out;
}

function load<T>(key: string, def: T): T {
  try { return { ...def, ...JSON.parse(localStorage.getItem(key) || "{}") }; } catch { return def; }
}
function persist(key: string, v: unknown) {
  try { localStorage.setItem(key, JSON.stringify(v)); } catch { /* ignore */ }
}

const why = (e: unknown, what: "create" | "rename" | "delete") => {
  if (e instanceof ApiError) {
    if (e.status === 409) return what === "delete" ? "The folder is not empty." : "Something with that name already exists.";
    if (e.status === 403) return "Names starting with a dot are not allowed.";
    if (e.status === 404) return "It no longer exists.";
    return e.message;
  }
  return "Cannot reach si.";
};

const modes: { id: string; label: string; icon: IconName }[] = [
  { id: "write", label: "Write", icon: "pen" },
  { id: "rehearse", label: "Rehearse", icon: "mic" },
  { id: "present", label: "Present", icon: "play" },
  { id: "lab", label: "Lab", icon: "flask" },
];
const nextTheme: Record<ThemeMode, ThemeMode> = { system: "light", light: "dark", dark: "system" };
const themeLabel: Record<ThemeMode, string> = { system: "System", light: "Light", dark: "Dark" };

/** Split is a draggable, keyboard-adjustable divider between two panes. */
function Split(p: { label: string; onMove(dx: number): void }) {
  const last = useRef(0);
  return (
    <div class="split" role="separator" aria-orientation="vertical" aria-label={p.label} tabIndex={0}
      onPointerDown={(e) => {
        last.current = e.clientX;
        e.currentTarget.setPointerCapture(e.pointerId);
        document.body.classList.add("dragging");
      }}
      onPointerMove={(e) => {
        if (!e.currentTarget.hasPointerCapture(e.pointerId)) return;
        p.onMove(e.clientX - last.current);
        last.current = e.clientX;
      }}
      onPointerUp={() => document.body.classList.remove("dragging")}
      onKeyDown={(e) => {
        if (e.key === "ArrowLeft" || e.key === "ArrowRight") {
          p.onMove(e.key === "ArrowLeft" ? -24 : 24);
          e.preventDefault();
        }
      }} />
  );
}

export function App() {
  const [tree, setTree] = useState<Node | null>(null);
  const [decks, setDecks] = useState<DeckInfo[]>([]);
  const [rules, setRules] = useState<Rules | null>(null);
  const [ready, setReady] = useState(false);
  const [checks, setChecks] = useState<Record<string, CheckResult>>({});
  const saved = useMemo(() => load("si-studio-tabs", { tabs: [] as string[], active: null as string | null }), []);
  const [tabs, setTabs] = useState<string[]>(saved.tabs);
  const [active, setActive] = useState<string | null>(saved.active);
  const [status, setStatus] = useState<Record<string, DocStatus>>({});
  const [cursor, setCursor] = useState<{ line: number; col: number } | null>(null);
  const [reload, setReload] = useState(0);
  const [reveal, setReveal] = useState<{ path: string; line: number; n: number } | null>(null);
  const [problemsOpen, setProblemsOpen] = useState(false);
  const [deckPath, setDeckPath] = useState<string | null>(null);
  const [theme, setTheme] = useState<ThemeMode>(storedTheme);
  const [offline, setOffline] = useState(false);
  const [layout, setLayout] = useState(() => load("si-studio-layout", { nav: 248, prev: 0.5 }));
  const panes = useRef<HTMLDivElement>(null);
  const ctl = useRef<EditorCtl | null>(null);

  const nodes = useMemo(() => flatten(tree), [tree]);
  const kindOf = (p: string): Kind => nodes.get(p)?.kind ?? extKind(p);

  const refresh = useCallback(async () => {
    try {
      const [t, ds] = await Promise.all([api.tree(), api.decks()]);
      setTree(t);
      setDecks(ds);
      setOffline(false);
      const cs = await Promise.all(ds.map((d) =>
        api.check(d.path).then((c): [string, CheckResult] => [d.path, c], (e): [string, CheckResult] => [d.path, { error: (e as Error).message }])));
      setChecks(Object.fromEntries(cs));
    } catch {
      setOffline(true);
    }
  }, []);

  // first load: rules before the editor so chips and completion are there from the start
  useEffect(() => {
    api.rules().then(setRules, () => {}).finally(() => setReady(true));
    void refresh();
  }, []);

  // live: any workspace change refreshes tree, checks and clean buffers
  useEffect(() => {
    const es = new EventSource("/_si/events");
    let t: ReturnType<typeof setTimeout> | undefined;
    es.addEventListener("reload", () => {
      clearTimeout(t);
      t = setTimeout(() => { void refresh(); setReload((n) => n + 1); }, 120);
    });
    es.onerror = () => setOffline(true);
    es.onopen = () => setOffline((was) => { if (was) void refresh(); return false; });
    return () => { clearTimeout(t); es.close(); };
  }, []);

  useEffect(() => persist("si-studio-tabs", { tabs, active }), [tabs, active]);
  useEffect(() => persist("si-studio-layout", layout), [layout]);

  // drop restored tabs whose files are gone
  useEffect(() => {
    if (!tree) return;
    const gone = tabs.filter((t) => !nodes.has(t));
    if (gone.length) {
      setTabs((ts) => ts.filter((t) => nodes.has(t)));
      if (active && !nodes.has(active)) setActive(null);
    }
  }, [tree]);

  // the preview follows the deck that owns the open file
  useEffect(() => {
    const d = active && decks.find((d) => d.files.includes(active));
    if (d) setDeckPath(d.path);
    else if (!deckPath || !decks.some((d) => d.path === deckPath)) setDeckPath(decks[0]?.path ?? null);
  }, [active, decks]);

  useEffect(() => {
    const key = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "s") { e.preventDefault(); void ctl.current?.flush(); }
    };
    addEventListener("keydown", key);
    return () => removeEventListener("keydown", key);
  }, []);

  const deck = decks.find((d) => d.path === deckPath) || null;
  const check = deckPath && isCheck(checks[deckPath]) ? (checks[deckPath] as Check) : null;
  const slide = check && active && cursor ? slideForCursor(check.slides, active, cursor.line) : null;
  const { problems, errors } = useMemo(() => {
    const seen = new Set<string>(), problems: Problem[] = [], errors: string[] = [];
    for (const [d, c] of Object.entries(checks)) {
      if (!isCheck(c)) { errors.push(`${d}: ${c.error}`); continue; }
      for (const q of c.problems) {
        const k = `${q.file}:${q.line}:${q.msg}`;
        if (!seen.has(k)) { seen.add(k); problems.push(q); }
      }
    }
    return { problems, errors };
  }, [checks]);
  const dirty = useMemo(() => new Set(Object.entries(status).filter(([, s]) => s.dirty || s.conflict).map(([p]) => p)), [status]);

  const open = (path: string, line?: number) => {
    setTabs((ts) => (ts.includes(path) ? ts : [...ts, path]));
    setActive(path);
    if (line) setReveal({ path, line, n: Date.now() });
  };
  const closeTab = async (path: string) => {
    await ctl.current?.flush();
    ctl.current?.drop(path);
    setTabs((ts) => {
      const i = ts.indexOf(path), rest = ts.filter((t) => t !== path);
      if (active === path) setActive(rest[Math.min(i, rest.length - 1)] ?? null);
      return rest;
    });
    setStatus(({ [path]: _, ...s }) => s);
  };
  const onStatus = useCallback((path: string, s: DocStatus) => setStatus((m) => ({ ...m, [path]: s })), []);
  const onCursor = useCallback((line: number, col: number) => setCursor({ line, col }), []);

  const onCreate = async (path: string, dir: boolean) => {
    try { await api.create(path, dir); } catch (e) { throw new Error(why(e, "create")); }
    await refresh();
    if (!dir) open(path);
  };
  const onRename = async (from: string, to: string) => {
    await ctl.current?.flush();
    try { await api.rename(from, to); } catch (e) { throw new Error(why(e, "rename")); }
    ctl.current?.drop(from);
    const move = (t: string) => (t === from ? to : t.startsWith(from + "/") ? to + t.slice(from.length) : t);
    setTabs((ts) => ts.map(move));
    setActive((a) => (a ? move(a) : a));
    await refresh();
  };
  const onDelete = async (path: string, dir: boolean) => {
    try { await api.remove(path); } catch (e) { throw new Error(why(e, "delete")); }
    ctl.current?.drop(path);
    clear(path);
    const keep = (t: string) => t !== path && !t.startsWith(path + "/");
    setTabs((ts) => ts.filter(keep));
    setActive((a) => (a && keep(a) ? a : null));
    await refresh();
  };

  const activeKind = active ? kindOf(active) : null;
  const isImage = activeKind === "image";
  const textPath = active && !isImage ? active : null;
  const cols = `${layout.nav}px 1px minmax(0,1fr)`;
  const paneCols = `minmax(240px,${1 - layout.prev}fr) 1px minmax(240px,${layout.prev}fr)`;

  return (
    <div class="app">
      <header class="top">
        <div class="top-left">
        <div class="brand"><span class="logo" aria-hidden="true">si</span><span>Studio</span></div>
        <div class="crumbs" title={active || ""}>
          <span class="ws"><Icon name="dir" />{tree?.name ?? "…"}</span>
          {active && active.split("/").map((seg, i, a) => (
            <><span class="sep" aria-hidden="true">›</span><span class={i === a.length - 1 ? "leaf" : ""}>{seg}</span></>
          ))}
        </div>
        </div>
        <div class="modes" role="group" aria-label="Mode">
          {modes.map((m) => m.id === "write"
            ? <button class="on" aria-pressed="true"><Icon name={m.icon} />{m.label}</button>
            : <button aria-disabled="true" title="Coming soon" onClick={(e) => e.preventDefault()}><Icon name={m.icon} />{m.label}</button>)}
        </div>
        <div class="top-right">
          {offline && <span class="offline" role="status">Reconnecting…</span>}
          <button class="icon-btn" title={`Theme: ${themeLabel[theme]}`}
            aria-label={`Theme: ${themeLabel[theme]}. Switch to ${themeLabel[nextTheme[theme]]}`}
            onClick={() => { const t = nextTheme[theme]; setTheme(t); applyTheme(t); }}>
            <Icon name={theme === "system" ? "system" : theme === "light" ? "sun" : "moon"} />
          </button>
          <a class={"btn btn-pri" + (deck ? "" : " disabled")} aria-disabled={!deck}
            href={deck ? deckURL(deck.path) + (slide ? "#" + slide : "") : undefined} target="_blank" rel="noopener"
            title={deck ? `Play ${deck.title} in a new tab` : "No deck to play"}>
            <Icon name="play" />Play
          </a>
        </div>
      </header>

      <div class="body" style={{ gridTemplateColumns: cols }}>
        <Navigator tree={tree} decks={decks} active={active} dirty={dirty} onOpen={(n) => open(n.path)}
          onCreate={onCreate} onRename={onRename} onDelete={onDelete} />
        <Split label="Resize navigator" onMove={(dx) => setLayout((l) => ({ ...l, nav: Math.min(480, Math.max(180, l.nav + dx)) }))} />
        <div class={"work" + (problemsOpen ? " with-problems" : "")}>
          <div class="panes" ref={panes} style={{ gridTemplateColumns: paneCols }}>
            <section class="ed" aria-label="Editor">
              <div class="tabs" role="tablist" aria-label="Open files">
                {tabs.map((t) => {
                  const st = status[t];
                  return (
                    <div class={"tab" + (t === active ? " on" : "")} key={t} title={t}
                      onAuxClick={(e) => e.button === 1 && void closeTab(t)}>
                      <button role="tab" aria-selected={t === active} class="tab-main" onClick={() => setActive(t)}>
                        <Icon name={kindOf(t)} class={"k-" + kindOf(t)} />{baseName(t)}
                      </button>
                      <button class={"tab-x" + (st?.dirty || st?.conflict ? " dirty" : "")} aria-label={`Close ${baseName(t)}`}
                        onClick={() => void closeTab(t)}><Icon name="close" /></button>
                    </div>
                  );
                })}
              </div>
              <div class="ed-body">
                {ready && (
                  <Editor path={textPath} kind={textPath ? kindOf(textPath) : "other"} rules={rules} problems={problems}
                    reload={reload} reveal={reveal} ctl={ctl} onCursor={onCursor} onStatus={onStatus} />
                )}
                {isImage && active && (
                  <div class="imgview">
                    <img src={fileURL(active)} alt={baseName(active)} />
                    <span class="imgmeta">{active}</span>
                  </div>
                )}
                {!active && <Welcome decks={decks} name={tree?.name} onOpen={(p) => open(p)} />}
              </div>
            </section>
            <Split label="Resize preview" onMove={(dx) => {
              const w = panes.current?.clientWidth || 1;
              setLayout((l) => ({ ...l, prev: Math.min(0.75, Math.max(0.2, l.prev - dx / w)) }));
            }} />
            <Preview deck={deck} slide={slide} total={check?.slides.length ?? 0} />
          </div>
          {problemsOpen && (
            <Problems problems={problems} errors={errors} onClose={() => setProblemsOpen(false)}
              onGo={(f, l) => open(f, l)} />
          )}
        </div>
      </div>

      <StatusBar decks={decks.length} problems={problems.length + errors.length} check={check}
        cursor={textPath ? cursor : null} kind={activeKind} doc={active ? status[active] ?? null : null} slide={slide}
        onProblems={() => setProblemsOpen((o) => !o)} />
    </div>
  );
}

function Welcome(p: { decks: DeckInfo[]; name?: string; onOpen(path: string): void }): JSX.Element {
  return (
    <div class="welcome">
      <div class="welcome-card">
        <span class="logo big" aria-hidden="true">si</span>
        <h1>{p.name ?? "si Studio"}</h1>
        <p class="mute">{p.decks.length ? "Pick a deck to write." : "No decks yet. Create a Markdown file in the navigator."}</p>
        {p.decks.length > 0 && (
          <ul class="deck-list">
            {p.decks.map((d) => (
              <li><button onClick={() => p.onOpen(d.path)}>
                <Icon name="deck" class="k-deck" /><span class="t">{d.title || baseName(d.path)}</span><span class="p">{d.path}</span>
                {d.files.length > 1 && <span class="n">{d.files.length} files</span>}
              </button></li>
            ))}
          </ul>
        )}
        <dl class="keys">
          <dt><kbd>⌘</kbd><kbd>S</kbd></dt><dd>Save now (Studio also saves as you type)</dd>
          <dt><kbd>Ctrl</kbd><kbd>Space</kbd></dt><dd>Complete a directive, layout or transition</dd>
          <dt><kbd>⌘</kbd><kbd>F</kbd></dt><dd>Find in file</dd>
        </dl>
      </div>
    </div>
  );
}
