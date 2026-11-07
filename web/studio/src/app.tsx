// The Studio shell: top bar with the mode spine, navigator, tabbed editor,
// live preview, problems panel and status bar.
import type { JSX } from "preact";
import { useCallback, useEffect, useMemo, useRef, useState } from "preact/hooks";
import { setCompletionFiles } from "./si-lang";
import { BuildDialog } from "./build-dialog";
import { type Mode, ModeView } from "./modes";
import { type Command, Palette } from "./palette";
import { NewDialog } from "./new-dialog";
import { Gallery } from "./gallery";
import { Tour, tourDone } from "./tour";
import { Inspector } from "./inspector";
import { Storyboard } from "./storyboard";
import { setDirective } from "./inspector-edit";
import { ApiError, api, type Check, type DeckInfo, type Kind, type Node, type Problem, type Rules, isSlides, kindLabel, pageURL } from "./api";
import { clear, scoped } from "./buffers";
import { type DocStatus, Editor, type EditorCtl, setRbView, unsaved } from "./editor";
import { Dialog, Icon, type IconName, Navigator } from "./navigator";
import { Preview } from "./preview";
import { Problems } from "./problems";
import { slideForCursor } from "./slidemap";
import { StatusBar } from "./statusbar";
import { survivingTabs, under } from "./tabs";
import { type ThemeMode, applyTheme, storedTheme } from "./theme";

type CheckResult = Check | { error: string };
const isCheck = (c: CheckResult | undefined): c is Check => !!c && !("error" in c);
const WORKSPACE = "*workspace"; // the checks key of the whole-workspace pass

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
    if (e.status === 403) return e.message === "hidden path" ? "Names starting with a dot are not allowed." : e.message;
    if (e.status === 422) return "si cannot parse it: " + e.message;
    if (e.status === 404) return "It no longer exists.";
    return e.message;
  }
  return "Cannot reach si.";
};

const modes: { id: Mode; label: string; icon: IconName }[] = [
  { id: "write", label: "Write", icon: "pen" },
  { id: "rehearse", label: "Rehearse", icon: "mic" },
  { id: "present", label: "Present", icon: "play" },
  { id: "lab", label: "Lab", icon: "flask" },
];
const nextTheme: Record<ThemeMode, ThemeMode> = { system: "light", light: "dark", dark: "system" };
const themeLabel: Record<ThemeMode, string> = { system: "System", light: "Light", dark: "Dark" };

/** Split is a draggable, keyboard-adjustable divider between two panes. */
function Split(p: { label: string; now: number; min: number; max: number; onMove(dx: number): void }) {
  const last = useRef(0);
  return (
    <div class="split" role="separator" aria-orientation="vertical" aria-label={p.label} tabIndex={0}
      aria-valuenow={p.now} aria-valuemin={p.min} aria-valuemax={p.max}
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
  const [mode, setMode] = useState<Mode>("write");
  const [paletteOpen, setPaletteOpen] = useState(false);
  // Write's three views: the editor with its preview, the storyboard, and focus (calm editor, floating preview)
  const [view, setView] = useState<"edit" | "board" | "focus">("edit");
  const [navOpen, setNavOpen] = useState(false); // narrow screens: the navigator slides over the editor
  const [inspOn, setInspOn] = useState(() => load("si-studio-insp", { on: false }).on);
  useEffect(() => persist("si-studio-insp", { on: inspOn }), [inspOn]);
  const deckRef = useRef<string | null>(null);
  const saved = useMemo(() => load(scoped("si-studio-tabs"), { tabs: [] as string[], active: null as string | null }), []);
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
  const [missing, setMissing] = useState<Set<string>>(() => new Set());
  const [layout, setLayout] = useState(() => load("si-studio-layout", { nav: 248, prev: 0.5 }));
  const [dialog, setDialog] = useState<JSX.Element | null>(null);
  const panes = useRef<HTMLDivElement>(null);
  const ctl = useRef<EditorCtl | null>(null);

  const nodes = useMemo(() => flatten(tree), [tree]);
  const kindOf = (p: string): Kind => nodes.get(p)?.kind ?? extKind(p);
  const files = useMemo(() => {
    const all = [...nodes.values()], of = (k: Kind) => all.filter((n) => n.kind === k).map((n) => n.path);
    return { images: of("image"), maps: of("mindmap"), decks: of("deck") };
  }, [nodes]);

  const refresh = useCallback(async () => {
    try {
      const [t, ds] = await Promise.all([api.tree(), api.decks()]);
      setTree(t);
      setDecks(ds);
      setOffline(false);
      // one pass over the workspace for problems (a guide, book or site is rendered
      // once, not once per page), plus the open deck's own check for its slide map
      const one = (p: string) => api.check(p).then((c): CheckResult => c, (e): CheckResult => ({ error: (e as Error).message }));
      const cur = deckRef.current && ds.some((d) => d.path === deckRef.current) ? deckRef.current : ds[0]?.path;
      const [all, c] = await Promise.all([
        api.checkAll().then((a): CheckResult => ({ problems: a.problems, slides: [], minutes: 0, steps: 0, canvases: 0 }),
          (e): CheckResult => ({ error: (e as Error).message })),
        cur ? one(cur) : Promise.resolve(null),
      ]);
      setChecks(cur && c ? { [WORKSPACE]: all, [cur]: c } : { [WORKSPACE]: all });
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

  useEffect(() => persist(scoped("si-studio-tabs"), { tabs, active }), [tabs, active]);
  useEffect(() => persist("si-studio-layout", layout), [layout]);

  // tabs whose files are gone close, unless they hold unsaved text: those stay, marked deleted
  useEffect(() => {
    if (!tree) return;
    const keep = new Set(Object.entries(status).filter(([, s]) => unsaved(s)).map(([p]) => p));
    const r = survivingTabs(tabs, (p) => nodes.has(p), keep);
    if (r.tabs.length !== tabs.length) {
      setTabs(r.tabs);
      if (active && !r.tabs.includes(active)) setActive(r.tabs[0] ?? null);
    }
    setMissing((m) => (m.size === r.missing.length && r.missing.every((p) => m.has(p)) ? m : new Set(r.missing)));
  }, [tree]);

  // path completion offers the workspace files, relative to the open file
  useEffect(() => setCompletionFiles([...nodes.values()].filter((n) => n.kind !== "dir").map((n) => n.path), active ?? ""), [nodes, active]);

  // the preview follows the deck that owns the open file
  useEffect(() => {
    const d = active && decks.find((d) => d.files.includes(active));
    if (d) setDeckPath(d.path);
    else if (!deckPath || !decks.some((d) => d.path === deckPath)) setDeckPath(decks[0]?.path ?? null);
  }, [active, decks]);

  useEffect(() => {
    const key = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "s") { e.preventDefault(); void ctl.current?.flush(); }
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "k") { e.preventDefault(); e.stopPropagation(); setPaletteOpen((o) => !o); }
    };
    addEventListener("keydown", key, true); // capture: before the editor's own Mod-k
    return () => removeEventListener("keydown", key, true);
  }, []);

  // the open deck's own check (slides, minutes) when the preview moves to another deck
  useEffect(() => {
    deckRef.current = deckPath;
    if (!deckPath || checks[deckPath]) return;
    api.check(deckPath).then((c) => setChecks((cs) => ({ ...cs, [deckPath]: c })),
      (e) => setChecks((cs) => ({ ...cs, [deckPath]: { error: (e as Error).message } })));
  }, [deckPath]);

  const deck = decks.find((d) => d.path === deckPath) || null;
  const check = deckPath && isCheck(checks[deckPath]) ? (checks[deckPath] as Check) : null;
  // documents have parts, not slides: only a deck follows the cursor slide by slide
  const slide = check && active && cursor && isSlides(deck?.kind) ? slideForCursor(check.slides, active, cursor.line) : null;
  const { problems, errors } = useMemo(() => {
    const seen = new Set<string>(), problems: Problem[] = [], errors: string[] = [];
    for (const [d, c] of Object.entries(checks)) {
      if (!isCheck(c)) { errors.push(`${d === WORKSPACE ? "Workspace check" : d}: ${c.error}`); continue; }
      for (const q of c.problems) {
        const k = `${q.file}:${q.line}:${q.msg}`;
        if (!seen.has(k)) { seen.add(k); problems.push(q); }
      }
    }
    return { problems, errors };
  }, [checks]);
  const dirty = useMemo(() => new Set(Object.entries(status).filter(([, s]) => unsaved(s)).map(([p]) => p)), [status]);

  const open = (path: string, line?: number) => {
    setNavOpen(false);
    setTabs((ts) => (ts.includes(path) ? ts : [...ts, path]));
    setActive(path);
    if (line) setReveal({ path, line, n: Date.now() });
  };
  const closeTab = async (path: string) => {
    await ctl.current?.flush();
    // text that did not reach disk (conflict, deleted, failed save) would be lost: ask first
    if (ctl.current?.unsaved(path).includes(path)) {
      setDialog(
        <Dialog title={`Close ${baseName(path)}?`} okLabel="Close without saving" danger onClose={() => setDialog(null)}
          body="Its text is not on disk (a conflict, a deleted file or a failed save). Closing the tab throws it away."
          ok={async () => discard(path)} />,
      );
      return;
    }
    ctl.current?.drop(path);
    setTabs((ts) => {
      const i = ts.indexOf(path), rest = ts.filter((t) => t !== path);
      if (active === path) setActive(rest[Math.min(i, rest.length - 1)] ?? null);
      return rest;
    });
    setStatus(({ [path]: _, ...s }) => s);
  };
  const discard = (path: string) => {
    ctl.current?.drop(path);
    clear(path);
    setTabs((ts) => {
      const rest = ts.filter((t) => t !== path);
      if (active === path) setActive(rest[0] ?? null);
      return rest;
    });
    setStatus(({ [path]: _, ...s }) => s);
    setMissing((m) => { const n = new Set(m); n.delete(path); return n; });
  };
  const onStatus = useCallback((path: string, s: DocStatus) => setStatus((m) => ({ ...m, [path]: s })), []);
  const onCursor = useCallback((line: number, col: number) => setCursor({ line, col }), []);

  const onCreate = async (path: string, dir: boolean) => {
    try { await api.create(path, dir); } catch (e) { throw new Error(why(e, "create")); }
    await refresh();
    if (!dir) open(path);
  };
  const onStarter = async (kind: string, path: string, title: string) => {
    let r;
    try { r = await api.starter(kind, path, title); } catch (e) { throw new Error(why(e, "create")); }
    await refresh();
    open(r.open);
  };
  const [tourOn, setTourOn] = useState(false);
  const showGallery = () => setDialog(
    <Gallery dir={active && active.includes("/") ? active.slice(0, active.lastIndexOf("/")) : ""}
      ok={async (k, path, title) => { await onStarter(k, path, title); if (!tourDone()) setTimeout(() => setTourOn(true), 400); }}
      onClose={() => setDialog(null)} />);
  // first run: a folder with nothing si can open gets the start gallery
  const greeted = useRef(false);
  useEffect(() => {
    if (!tree || greeted.current) return;
    greeted.current = true;
    const docs = [...nodes.values()].some((n) => n.kind !== "dir" && n.kind !== "image" && n.kind !== "other" && n.kind !== "css" && n.kind !== "json" && n.kind !== "svg");
    if (!docs) showGallery();
  }, [tree]);
  // the slide under the cursor, and the inspector's edits written into its lines
  const curSlide = check && slide ? check.slides.find((s) => s.index === slide) ?? null : null;
  const setSlideDirective = (name: string, value: string) => {
    if (!curSlide || !active || curSlide.file !== active) return;
    const next = check!.slides.find((s) => s.file === curSlide.file && s.line > curSlide.line);
    ctl.current?.edit(active, (t) => setDirective(t, curSlide.line, next ? next.line - 1 : t.split("\n").length, name, value));
  };
  const commands: Command[] = [
    { id: "view-board", label: "Show the storyboard", icon: "deck", run: () => { setMode("write"); setView("board"); } },
    { id: "view-focus", label: view === "focus" ? "Leave focus mode" : "Focus mode", icon: "system", run: () => { setMode("write"); setView((v) => (v === "focus" ? "edit" : "focus")); } },
    { id: "inspector", label: inspOn ? "Hide the inspector" : "Show the inspector", icon: "braces", run: () => setInspOn((o) => !o) },
    ...modes.map((m): Command => ({ id: "mode-" + m.id, label: `Mode: ${m.label}`, icon: m.icon, run: () => setMode(m.id) })),
    { id: "new", label: "New slides, article, guide, site…", icon: "plus", run: () => setDialog(
      <NewDialog dir={active && active.includes("/") ? active.slice(0, active.lastIndexOf("/")) : ""} ok={onStarter} onClose={() => setDialog(null)} />) },
    { id: "build", label: "Build the folder to HTML…", icon: "braces", run: () => setDialog(<BuildDialog onClose={() => setDialog(null)} />) },
    { id: "gallery", label: "Start gallery: pick something to make", icon: "plus", run: showGallery },
    { id: "tour", label: "Take the tour of the Studio", icon: "play", run: () => { setMode("write"); setView("edit"); setTourOn(true); } },
    { id: "problems", label: "Show problems", icon: "warn", run: () => setProblemsOpen(true) },
    { id: "theme", label: "Switch light / dark", icon: "moon", run: () => { const t = nextTheme[theme]; setTheme(t); applyTheme(t); } },
    ...(deck ? [{ id: "open-live", label: `Open ${deck.title} in a new tab`, icon: "external" as IconName, run: () => window.open(pageURL(deck), "_blank", "noopener") }] : []),
    ...[...nodes.values()].filter((n) => n.kind !== "dir").map((n): Command => ({
      id: "f-" + n.path, label: baseName(n.path), hint: n.path, icon: n.kind, run: () => { setMode("write"); open(n.path); } })),
  ];
  // a runbook made by the generator opens in the Visual view
  const onRunbook = async (path: string) => {
    setRbView(path, "visual");
    await refresh();
    open(path);
  };
  const onRename = async (from: string, to: string) => {
    await ctl.current?.flush();
    // a doc that still differs from disk would lose its text in the move: make the user settle it first
    const blocked = ctl.current?.unsaved(from) ?? [];
    if (blocked.length)
      throw new Error(`${blocked.map(baseName).join(", ")} has changes that are not on disk yet (conflict, deleted or failed save). Resolve them in the editor first.`);
    try { await api.rename(from, to); } catch (e) { throw new Error(why(e, "rename")); }
    ctl.current?.drop(from);
    const move = (t: string) => (under(t, from) ? to + t.slice(from.length) : t);
    setTabs((ts) => ts.map(move));
    setActive((a) => (a ? move(a) : a));
    await refresh();
  };
  const onDelete = async (path: string, dir: boolean) => {
    try { await api.remove(path); } catch (e) { throw new Error(why(e, "delete")); }
    ctl.current?.drop(path);
    clear(path);
    const keep = (t: string) => !under(t, path);
    setTabs((ts) => ts.filter(keep));
    setActive((a) => (a && keep(a) ? a : null));
    await refresh();
  };

  const activeKind = active ? kindOf(active) : null;
  const isImage = activeKind === "image";
  const textPath = active && !isImage ? active : null;
  const cols = view === "focus" ? "minmax(0,1fr)" : `${layout.nav}px 1px minmax(0,1fr)`;
  const paneCols = view === "focus" ? "minmax(0,1fr) 0 0" : `minmax(240px,${1 - layout.prev}fr) 1px minmax(240px,${layout.prev}fr)`;

  return (
    <div class="app">
      <header class="top">
        <div class="top-left">
        <button class="icon-btn nav-toggle" aria-expanded={navOpen} aria-label="Files" title="Files" onClick={() => setNavOpen((o) => !o)}><Icon name="dir" /></button>
        <div class="brand"><span class="logo" aria-hidden="true">si</span><span>Studio</span></div>
        <div class="crumbs" title={active || ""}>
          <span class="ws"><Icon name="dir" />{tree?.name ?? "…"}</span>
          {active && active.split("/").map((seg, i, a) => (
            <><span class="sep" aria-hidden="true">›</span><span class={i === a.length - 1 ? "leaf" : ""}>{seg}</span></>
          ))}
        </div>
        </div>
        <div class="modes" role="group" aria-label="Mode">
          {modes.map((m) => (
            <button class={m.id === mode ? "on" : ""} aria-pressed={m.id === mode} onClick={() => setMode(m.id)}
              title={m.id === "write" ? "Write" : m.id === "rehearse" ? "The deck beside its teleprompter" : m.id === "present" ? "Full screen, teleprompter in another window" : "The deck with its terminal and runbook"}>
              <Icon name={m.icon} />{m.label}
            </button>
          ))}
        </div>
        <div class="top-right">
          {mode === "write" && (
            <div class="views" role="radiogroup" aria-label="Write view">
              {([["edit", "Editor", "pen"], ["board", "Storyboard", "deck"], ["focus", "Focus", "system"]] as const).map(([id, label, icon]) => (
                <button role="radio" aria-checked={view === id} class={view === id ? "on" : ""} title={label} aria-label={label} onClick={() => setView(id)}>
                  <Icon name={icon} />
                </button>
              ))}
            </div>
          )}
          {offline && <span class="offline" role="status">Reconnecting…</span>}
          <button class="icon-btn" title={`Theme: ${themeLabel[theme]}`}
            aria-label={`Theme: ${themeLabel[theme]}. Switch to ${themeLabel[nextTheme[theme]]}`}
            onClick={() => { const t = nextTheme[theme]; setTheme(t); applyTheme(t); }}>
            <Icon name={theme === "system" ? "system" : theme === "light" ? "sun" : "moon"} />
          </button>
          <button class="btn build-btn" title="Build the folder to self-contained HTML" onClick={() => setDialog(<BuildDialog onClose={() => setDialog(null)} />)}>
            <Icon name="braces" />Build
          </button>
          <a class={"btn btn-pri" + (deck ? "" : " disabled")} aria-disabled={!deck}
            href={deck ? pageURL(deck) + (slide ? "#" + slide : "") : undefined} target="_blank" rel="noopener"
            title={deck ? (isSlides(deck.kind) ? `Play ${deck.title} in a new tab` : `Open ${deck.title} in a new tab`) : "Nothing to play"}>
            <Icon name="play" />{deck && !isSlides(deck.kind) ? "Open" : "Play"}
          </a>
        </div>
      </header>

      {mode !== "write" && <ModeView mode={mode} deck={deck} decks={decks} slide={slide} onPick={(d) => setDeckPath(d)} />}
      {/* Write stays mounted while another mode shows, so open files keep their state */}
      <div class={"body" + (view === "focus" ? " focus" : "") + (navOpen ? " nav-open" : "")} style={{ "--cols": cols }} hidden={mode !== "write"}>
        <Navigator tree={tree} decks={decks} active={active} dirty={dirty} onOpen={(n) => open(n.path)}
          onCreate={onCreate} onRename={onRename} onDelete={onDelete} onRunbook={onRunbook} onStarter={onStarter} />
        <Split label="Resize navigator" now={layout.nav} min={180} max={480} onMove={(dx) => setLayout((l) => ({ ...l, nav: Math.min(480, Math.max(180, l.nav + dx)) }))} />
        <div class={"work" + (problemsOpen ? " with-problems" : "")}>
          {view === "board" && <Storyboard check={check} current={slide} title={deck?.title ?? ""} url={deck && isSlides(deck.kind) ? pageURL(deck) : null} onGo={(f, l) => { setView("edit"); open(f, l); }} />}
          {/* the editor stays mounted under the storyboard, so open files keep their state */}
          <div class={"panes" + (view === "focus" ? " focus" : "")} ref={panes} style={{ "--pcols": paneCols }} hidden={view === "board"}>
            <section class="ed" aria-label="Editor">
              <div class="tabs" role="group" aria-label="Open files">
                {tabs.map((t) => {
                  const st = status[t];
                  return (
                    <div class={"tab" + (t === active ? " on" : "") + (missing.has(t) ? " gone" : "")} key={t}
                      title={missing.has(t) ? t + " (deleted on disk)" : t}
                      onAuxClick={(e) => e.button === 1 && void closeTab(t)}>
                      <button aria-current={t === active ? "true" : undefined} class="tab-main" onClick={() => setActive(t)}>
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
                    reload={reload} reveal={reveal} missing={missing} onDiscard={discard} ctl={ctl} onCursor={onCursor} onStatus={onStatus} files={files} />
                )}
                {isImage && active && (
                  <div class="imgview">
                    <img src={fileURL(active)} alt={baseName(active)} />
                    <span class="imgmeta">{active}</span>
                  </div>
                )}
                {!active && <Welcome decks={decks} name={tree?.name} onOpen={(p) => open(p)} onGallery={showGallery} onTour={() => setTourOn(true)} />}
              </div>
            </section>
            <Split label="Resize preview" now={Math.round(layout.prev * 100)} min={20} max={75} onMove={(dx) => {
              const w = panes.current?.clientWidth || 1;
              setLayout((l) => ({ ...l, prev: Math.min(0.75, Math.max(0.2, l.prev - dx / w)) }));
            }} />
            {/* only one live deck at a time: in the other modes the mode view owns the deck, and a
                second copy here would also answer the teleprompter */}
            {mode === "write" ? (
              <div class="prev-col">
                <Preview deck={deck} slide={slide} total={check?.slides.length ?? 0} inspector={{ on: inspOn, toggle: () => setInspOn((o) => !o) }} />
                {inspOn && isSlides(deck?.kind) && <Inspector slide={curSlide} rules={rules} onSet={setSlideDirective} />}
              </div>
            ) : <section class="prev" />}
          </div>
          {problemsOpen && (
            <Problems problems={problems} errors={errors} onClose={() => setProblemsOpen(false)}
              onGo={(f, l) => open(f, l)} />
          )}
        </div>
      </div>

      {paletteOpen && <Palette commands={commands} onClose={() => setPaletteOpen(false)} />}
      <StatusBar decks={decks.length} problems={problems.length + errors.length} check={check}
        cursor={textPath ? cursor : null} kind={activeKind} doc={active ? status[active] ?? null : null} slide={slide}
        onProblems={() => setProblemsOpen((o) => !o)} />
      {dialog}
      {tourOn && <Tour onClose={() => setTourOn(false)} />}
    </div>
  );
}

function Welcome(p: { decks: DeckInfo[]; name?: string; onOpen(path: string): void; onGallery(): void; onTour(): void }): JSX.Element {
  return (
    <div class="welcome">
      <div class="welcome-card">
        <span class="logo big" aria-hidden="true">si</span>
        <h1>{p.name ?? "si Studio"}</h1>
        <p class="mute">{p.decks.length ? "Pick something to write." : "Nothing here yet: start from the gallery, or create a Markdown file in the navigator."}</p>
        <div class="welcome-actions">
          <button class="btn btn-pri" onClick={p.onGallery}><Icon name="plus" />Start gallery</button>
          <button class="btn" onClick={p.onTour}><Icon name="play" />Take the tour</button>
        </div>
        {p.decks.length > 0 && (
          <ul class="deck-list">
            {p.decks.map((d) => (
              <li><button onClick={() => p.onOpen(d.path)}>
                <Icon name={d.kind || "deck"} class={"k-" + (d.kind || "deck")} /><span class="t">{d.title || baseName(d.path)}</span>
                <span class="p">{kindLabel[d.kind] || "Slides"} · {d.path}</span>
                {d.files.length > 1 && <span class="n">{d.files.length} files</span>}
              </button></li>
            ))}
          </ul>
        )}
        <dl class="keys">
          <dt><kbd>⌘</kbd><kbd>K</kbd></dt><dd>Open any file or run a command</dd>
          <dt><kbd>⌘</kbd><kbd>S</kbd></dt><dd>Save now (Studio also saves as you type)</dd>
          <dt><kbd>Ctrl</kbd><kbd>Space</kbd></dt><dd>Complete a directive, layout or transition</dd>
          <dt><kbd>⌘</kbd><kbd>F</kbd></dt><dd>Find in file</dd>
        </dl>
      </div>
    </div>
  );
}
