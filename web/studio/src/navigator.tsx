// The left navigator: the workspace tree with si-aware icons, plus the
// context menu and dialogs for New file / New folder / Rename / Delete.
import type { ComponentChildren, JSX } from "preact";
import { useEffect, useRef, useState } from "preact/hooks";
import type { DeckInfo, Kind, Node } from "./api";

const paths: Record<string, JSX.Element> = {
  dir: <path d="M2.5 4.5a1 1 0 0 1 1-1h3l1.5 1.5h4.5a1 1 0 0 1 1 1v6a1 1 0 0 1-1 1h-9a1 1 0 0 1-1-1z" />,
  deck: <><rect x="2.5" y="3.5" width="11" height="7.5" rx="1.2" /><path d="M8 11v2.5M5.5 13.5h5" /></>,
  lab: <><path d="M4 2.5h5l3 3v8H4z" /><path d="M6 8h4M6 10h4M6 12h2" /></>,
  markdown: <><path d="M4 2.5h5l3 3v8H4z" /><path d="M6 8.5h4M6 10.5h3" /></>,
  svg: <><path d="M3 12.5 7 5l2.5 4.5L11 7l2 5.5z" /><circle cx="11" cy="4" r="1.2" /></>,
  css: <><path d="M5.5 3.5c-1.5 0-1.5 1-1.5 2s-1 2-1.5 2.5c.5.5 1.5 1 1.5 2.5s0 2 1.5 2M10.5 3.5c1.5 0 1.5 1 1.5 2s1 2 1.5 2.5c-.5.5-1.5 1-1.5 2.5s0 2-1.5 2" /></>,
  json: <><path d="M5.5 3c-1.5 0-1.5 1-1.5 2.5S3 7.5 2.5 8c.5.5 1.5 1 1.5 2.5S4 13 5.5 13M10.5 3c1.5 0 1.5 1 1.5 2.5s1 2 1.5 2.5c-.5.5-1.5 1-1.5 2.5s0 2.5-1.5 2.5" /><circle cx="8" cy="8" r=".6" /></>,
  image: <><rect x="2.5" y="3.5" width="11" height="9" rx="1.2" /><circle cx="6" cy="6.5" r="1" /><path d="m3 12 3.5-3.5 2.5 2.5 1.5-1.5L13 12" /></>,
  other: <path d="M4 2.5h5l3 3v8H4zM9 2.5v3h3" />,
  chevron: <path d="m6 4 4 4-4 4" />,
  plusFile: <><path d="M4 2.5h5l3 3v8H4z" /><path d="M8 7.5v4M6 9.5h4" /></>,
  plusDir: <><path d="M2.5 4.5a1 1 0 0 1 1-1h3l1.5 1.5h4.5a1 1 0 0 1 1 1v6a1 1 0 0 1-1 1h-9a1 1 0 0 1-1-1z" /><path d="M8 7v4M6 9h4" /></>,
  more: <><circle cx="4" cy="8" r=".9" /><circle cx="8" cy="8" r=".9" /><circle cx="12" cy="8" r=".9" /></>,
  sun: <><circle cx="8" cy="8" r="2.8" /><path d="M8 1.5v1.5M8 13v1.5M1.5 8H3M13 8h1.5M3.4 3.4l1 1M11.6 11.6l1 1M3.4 12.6l1-1M11.6 4.4l1-1" /></>,
  moon: <path d="M13 9.5A5.5 5.5 0 0 1 6.5 3a5.5 5.5 0 1 0 6.5 6.5z" />,
  system: <><rect x="2" y="3" width="12" height="8" rx="1.2" /><path d="M6 13.5h4M8 11v2.5" /></>,
  play: <path d="M5 3.5v9l7.5-4.5z" />,
  pen: <path d="m10.5 3 2.5 2.5-7 7H3.5V10zM9 4.5 11.5 7" />,
  mic: <><rect x="6" y="2" width="4" height="7.5" rx="2" /><path d="M3.5 8a4.5 4.5 0 0 0 9 0M8 12.5V14" /></>,
  flask: <path d="M6.5 2.5h3M7 2.5v4L3.5 12.5a1 1 0 0 0 .9 1.5h7.2a1 1 0 0 0 .9-1.5L9 6.5v-4M5 10h6" />,
  warn: <><path d="M8 2.5 14 13H2z" /><path d="M8 6.5v3M8 11.2v.3" /></>,
  ok: <><circle cx="8" cy="8" r="5.5" /><path d="m5.5 8 1.8 1.8L10.8 6" /></>,
  external: <path d="M9 2.5h4.5V7M13.5 2.5 8 8M11.5 9.5v3.5h-9v-9H6" />,
  close: <path d="m4.5 4.5 7 7M11.5 4.5l-7 7" />,
};

export type IconName = keyof typeof paths | Kind;

export function Icon({ name, class: cls }: { name: IconName; class?: string }) {
  return (
    <svg class={"ico " + (cls || "")} viewBox="0 0 16 16" aria-hidden="true" fill="none" stroke="currentColor"
      stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round">
      {paths[name] || paths.other}
    </svg>
  );
}

/** Menu is a small popup menu at viewport coordinates; Escape or an outside click closes it. */
export function Menu(p: { x: number; y: number; items: { label: string; danger?: boolean; run(): void }[]; onClose(): void }) {
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    ref.current?.querySelector("button")?.focus();
    const down = (e: MouseEvent) => !ref.current?.contains(e.target as globalThis.Node) && p.onClose();
    const key = (e: KeyboardEvent) => {
      if (e.key === "Escape") p.onClose();
      if (e.key === "ArrowDown" || e.key === "ArrowUp") {
        const bs = [...ref.current!.querySelectorAll("button")];
        const i = bs.indexOf(document.activeElement as HTMLButtonElement);
        bs[(i + (e.key === "ArrowDown" ? 1 : bs.length - 1)) % bs.length]?.focus();
        e.preventDefault();
      }
    };
    addEventListener("mousedown", down, true);
    addEventListener("keydown", key, true);
    return () => { removeEventListener("mousedown", down, true); removeEventListener("keydown", key, true); };
  }, []);
  const x = Math.min(p.x, innerWidth - 200), y = Math.min(p.y, innerHeight - 40 * p.items.length);
  return (
    <div class="menu" role="menu" ref={ref} style={{ left: x + "px", top: y + "px" }}>
      {p.items.map((it) => (
        <button role="menuitem" class={it.danger ? "danger" : ""} onClick={() => { p.onClose(); it.run(); }}>{it.label}</button>
      ))}
    </div>
  );
}

/** Dialog is a modal with an optional text field; ok() may throw to show an error and stay open. */
export function Dialog(p: {
  title: string; body?: ComponentChildren; value?: string; select?: [number, number]; okLabel: string; danger?: boolean;
  ok(value: string): Promise<void>; onClose(): void;
}) {
  const [value, setValue] = useState(p.value ?? "");
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);
  const input = useRef<HTMLInputElement>(null);
  const box = useRef<HTMLFormElement>(null);
  useEffect(() => {
    const i = input.current;
    if (i) { i.focus(); i.setSelectionRange(...(p.select || [0, i.value.length])); }
    else box.current?.querySelector<HTMLButtonElement>("button[type=submit]")?.focus();
  }, []);
  const submit = async (e: Event) => {
    e.preventDefault();
    setBusy(true);
    try { await p.ok(value.trim()); p.onClose(); } catch (x) { setErr((x as Error).message); setBusy(false); }
  };
  return (
    <div class="scrim" onMouseDown={(e) => e.target === e.currentTarget && p.onClose()}>
      <div class="dialog" role="dialog" aria-modal="true" aria-label={p.title}>
      <form ref={box} onSubmit={submit} onKeyDown={(e) => e.key === "Escape" && p.onClose()}>
        <h2>{p.title}</h2>
        {p.body && <p class="dialog-body">{p.body}</p>}
        {p.value !== undefined && (
          <input ref={input} class="field" value={value} spellcheck={false} aria-label={p.title}
            onInput={(e) => { setValue(e.currentTarget.value); setErr(""); }} />
        )}
        {err && <p class="dialog-err" role="alert">{err}</p>}
        <div class="dialog-actions">
          <button type="button" class="btn" onClick={p.onClose}>Cancel</button>
          <button type="submit" class={"btn " + (p.danger ? "btn-danger" : "btn-pri")} disabled={busy}>{p.okLabel}</button>
        </div>
      </form>
      </div>
    </div>
  );
}

const parentOf = (path: string) => path.includes("/") ? path.slice(0, path.lastIndexOf("/")) : "";
const join = (dir: string, name: string) => (dir ? dir + "/" + name : name);

interface Props {
  tree: Node | null;
  decks: DeckInfo[];
  active: string | null;
  dirty: Set<string>;
  onOpen(n: Node): void;
  onCreate(path: string, dir: boolean): Promise<void>;
  onRename(from: string, to: string): Promise<void>;
  onDelete(path: string, dir: boolean): Promise<void>;
}

export function Navigator(p: Props) {
  const [open, setOpen] = useState<Set<string>>(() => new Set([""]));
  const [menu, setMenu] = useState<{ x: number; y: number; node: Node } | null>(null);
  const [dialog, setDialog] = useState<JSX.Element | null>(null);
  const titles = new Map(p.decks.map((d) => [d.path, d.title]));

  // reveal the active file: open every folder above it
  useEffect(() => {
    if (!p.active) return;
    const parts = p.active.split("/");
    setOpen((s) => {
      const n = new Set(s);
      for (let i = 1; i < parts.length; i++) n.add(parts.slice(0, i).join("/"));
      return n;
    });
  }, [p.active]);

  const toggle = (path: string) => setOpen((s) => { const n = new Set(s); n.has(path) ? n.delete(path) : n.add(path); return n; });
  const close = () => setDialog(null);

  const create = (dir: string, isDir: boolean) => setDialog(
    <Dialog title={isDir ? "New folder" : "New file"} value={isDir ? "" : ".md"} select={[0, 0]}
      okLabel="Create" onClose={close}
      body={<>In <code>{dir || "workspace root"}</code></>}
      ok={async (name) => { if (!name) throw new Error("Enter a name."); await p.onCreate(join(dir, name), isDir); }} />,
  );
  const rename = (n: Node) => {
    const dot = n.kind === "dir" ? -1 : n.name.lastIndexOf(".");
    setDialog(
      <Dialog title={`Rename ${n.name}`} value={n.path} okLabel="Rename" onClose={close}
        select={[n.path.length - n.name.length, dot > 0 ? n.path.length - n.name.length + dot : n.path.length]}
        ok={async (to) => { if (to && to !== n.path) await p.onRename(n.path, to); }} />,
    );
  };
  const remove = (n: Node) => setDialog(
    <Dialog title={`Delete ${n.name}?`} okLabel="Delete" danger onClose={close}
      body={n.kind === "dir" ? "Only empty folders can be deleted." : "The file is removed from disk. This cannot be undone."}
      ok={() => p.onDelete(n.path, n.kind === "dir")} />,
  );

  const openMenu = (e: MouseEvent, node: Node) => {
    e.preventDefault();
    e.stopPropagation();
    const r = (e.currentTarget as HTMLElement).getBoundingClientRect();
    setMenu({ x: e.type === "contextmenu" && e.clientX ? e.clientX : r.right - 8, y: e.type === "contextmenu" && e.clientY ? e.clientY : r.bottom, node });
  };

  const row = (n: Node, depth: number): JSX.Element => {
    const isDir = n.kind === "dir", on = n.path === p.active, expanded = open.has(n.path);
    const title = n.kind === "deck" || n.kind === "lab" ? titles.get(n.path) : undefined;
    return (
      <li key={n.path} role="treeitem" aria-expanded={isDir ? expanded : undefined} aria-selected={on}>
        <div class={"row" + (on ? " on" : "")} style={{ "--depth": depth }} onContextMenu={(e) => openMenu(e, n)}>
          <button class="row-main" title={n.path} onClick={() => (isDir ? toggle(n.path) : p.onOpen(n))}
            onKeyDown={(e) => {
              if (e.key === "ContextMenu" || (e.shiftKey && e.key === "F10")) openMenu(e as unknown as MouseEvent, n);
              if (isDir && e.key === "ArrowRight" && !expanded) toggle(n.path);
              if (isDir && e.key === "ArrowLeft" && expanded) toggle(n.path);
            }}>
            {isDir ? <Icon name="chevron" class={"chev" + (expanded ? " open" : "")} /> : <span class="chev" />}
            <Icon name={n.kind} class={"k-" + n.kind} />
            <span class="name">{n.name}</span>
            {title && <span class="sub">{title}</span>}
            {p.dirty.has(n.path) && <span class="dot" aria-label="unsaved" />}
          </button>
          <button class="row-more" aria-label={`Actions for ${n.name}`} onClick={(e) => openMenu(e, n)}>
            <Icon name="more" />
          </button>
        </div>
        {isDir && expanded && n.children && n.children.length > 0 && (
          <ul role="group">{n.children.map((c) => row(c, depth + 1))}</ul>
        )}
      </li>
    );
  };

  const target = (n: Node) => (n.kind === "dir" ? n.path : parentOf(n.path));
  return (
    <nav class="nav" aria-label="Workspace files">
      <div class="nav-head">
        <span class="grp">Workspace</span>
        <button class="icon-btn" aria-label="New file" title="New file" onClick={() => create("", false)}><Icon name="plusFile" /></button>
        <button class="icon-btn" aria-label="New folder" title="New folder" onClick={() => create("", true)}><Icon name="plusDir" /></button>
      </div>
      <div class="nav-scroll" onContextMenu={(e) => p.tree && openMenu(e, p.tree)}>
        {!p.tree ? <div class="nav-empty">Loading…</div>
          : !p.tree.children?.length ? <div class="nav-empty">This folder is empty. Create a deck with <b>New file</b>.</div>
          : <ul role="tree" aria-label={p.tree.name}>{p.tree.children.map((c) => row(c, 0))}</ul>}
      </div>
      {menu && (
        <Menu x={menu.x} y={menu.y} onClose={() => setMenu(null)} items={[
          { label: "New file…", run: () => create(target(menu.node), false) },
          { label: "New folder…", run: () => create(target(menu.node), true) },
          ...(menu.node.path ? [
            { label: "Rename…", run: () => rename(menu.node) },
            { label: "Delete…", danger: true, run: () => remove(menu.node) },
          ] : []),
        ]} />
      )}
      {dialog}
    </nav>
  );
}
