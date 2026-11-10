// The Theme view: a .css theme file as token controls (colours with pickers,
// light beside dark, fonts, sizes, motion) beside a specimen deck drawn with
// it, light and dark, at four screen sizes, and the contrast checks. Every
// change is an op; Go rewrites the file (POST theme, internal/themefile) and
// the new text goes into the editor as one undoable change, so the Text view
// and the Theme view stay in step.
import { useEffect, useRef, useState } from "preact/hooks";
import { api, type Node } from "./api";
import { Field } from "./map-editor";
import { hex6, lengthParts, short, sliderRange, themeSlug } from "./theme-tokens";

interface Meta { name: string; desc: string; group: string; kind: string; units?: string[]; enum?: string[]; pair: boolean }
interface ThemeInfo { name: string; label: string; about: string; light: Record<string, string>; dark: Record<string, string> }
interface Spec { groups: string[]; tokens: Meta[]; themes: ThemeInfo[] }
interface Issue { line?: number; token?: string; mode?: string; msg: string; ratio?: number; want?: number; on?: string; fix?: string }
interface Resp {
  text: string; name: string; extends: string; light: Record<string, string>; dark: Record<string, string>;
  effLight: Record<string, string>; effDark: Record<string, string>; issues: Issue[]; extra: boolean; html: string;
}
interface Op { token?: string; mode?: string; value: string; field?: string }

let specCache: Promise<Spec> | null = null;
export const themeSpec = () => (specCache ??= api.themeSpec() as Promise<Spec>);

const SIZES: Record<string, [number, number, string]> = {
  desktop: [1440, 900, "Desktop"], phone: [400, 800, "Phone"], portrait: [820, 1180, "Tablet"], "4k": [3840, 2160, "4K"],
};
const OPEN = new Set(["Colours", "Type", "Headings"]);
const STACKS = [
  `ui-sans-serif, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif`,
  `"Iowan Old Style", "Palatino Linotype", Palatino, Georgia, serif`,
  `ui-monospace, "SF Mono", "JetBrains Mono", Menlo, Consolas, monospace`,
  `system-ui, sans-serif`, `Georgia, serif`, `"Avenir Next", Avenir, "Segoe UI", sans-serif`,
];

/** Frame draws the preview page at a real screen size, scaled to fit its box. */
function Frame(p: { html: string; mode: "light" | "dark"; size: string; slide: number }) {
  const box = useRef<HTMLDivElement>(null);
  const frame = useRef<HTMLIFrameElement>(null);
  const [fit, setFit] = useState({ scale: 0.5, dx: 0, dy: 0 });
  const [w, h] = SIZES[p.size];
  useEffect(() => {
    const el = box.current;
    if (!el) return;
    const place = () => {
      const scale = Math.min(el.clientWidth / w, el.clientHeight / h) || 0.5;
      setFit({ scale, dx: (el.clientWidth - w * scale) / 2, dy: (el.clientHeight - h * scale) / 2 });
    };
    place();
    const ro = new ResizeObserver(place);
    ro.observe(el);
    return () => ro.disconnect();
  }, [w, h]);
  const sync = () => {
    try {
      const win = frame.current?.contentWindow as (Window & { SI?: { go(i: number, s: number | string, instant?: boolean): void } }) | null;
      win?.document.documentElement.setAttribute("data-si-mode", p.mode);
      win?.SI?.go(p.slide, "end", true);
    } catch { /* not loaded yet */ }
  };
  useEffect(sync, [p.mode, p.slide]);
  return (
    <div class="thx-box" ref={box}>
      <iframe ref={frame} srcdoc={p.html} title={`Preview, ${p.mode}`} onLoad={sync}
        style={{ width: w + "px", height: h + "px", transform: `translate(${fit.dx}px, ${fit.dy}px) scale(${fit.scale})` }} />
      <span class="thx-tag">{p.mode}</span>
    </div>
  );
}

/** Colour is one light or dark value: a picker (for hex colours) and its text. */
function Colour(p: { id: string; label: string; value: string; placeholder: string; onSet(v: string): void; onPick(v: string): void }) {
  const shown = p.value || p.placeholder;
  return (
    <div class="thx-col">
      <input type="color" aria-label={p.label + " colour picker"} value={hex6(shown) || "#000000"} disabled={!hex6(shown)}
        onInput={(e) => p.onPick(e.currentTarget.value)} />
      <Field id={p.id} label={p.label} value={p.value} placeholder={p.placeholder} onSet={p.onSet} />
    </div>
  );
}

function TokenRow(p: { m: Meta; r: Resp; op(o: Op): void; pick(o: Op): void }) {
  const { m, r } = p;
  const own = r.light[m.name], ownDark = r.dark[m.name];
  const eff = r.effLight[m.name] ?? "", effDark = r.effDark[m.name] ?? "";
  const changed = own !== undefined || ownDark !== undefined;
  const id = "thx-" + short(m.name);
  let control;
  if (m.kind === "color") {
    control = (
      <div class="thx-pair">
        <Colour id={id} label="light" value={own ?? ""} placeholder={eff} onSet={(v) => p.op({ token: m.name, mode: "light", value: v })}
          onPick={(v) => p.pick({ token: m.name, mode: "light", value: v })} />
        <Colour id={id + "-dark"} label="dark" value={ownDark ?? ""} placeholder={effDark} onSet={(v) => p.op({ token: m.name, mode: "dark", value: v })}
          onPick={(v) => p.pick({ token: m.name, mode: "dark", value: v })} />
      </div>
    );
  } else if (m.pair) {
    control = (
      <div class="thx-pair">
        <div class="thx-col"><Field id={id} label="light" value={own ?? ""} placeholder={eff} onSet={(v) => p.op({ token: m.name, mode: "light", value: v })} /></div>
        <div class="thx-col"><Field id={id + "-dark"} label="dark" value={ownDark ?? ""} placeholder={effDark} onSet={(v) => p.op({ token: m.name, mode: "dark", value: v })} /></div>
      </div>
    );
  } else if (m.kind === "enum") {
    control = (
      <select id={id} class="field" value={own ?? ""} onChange={(e) => p.op({ token: m.name, value: e.currentTarget.value })}>
        <option value="">{eff ? eff + " (base)" : "unset"}</option>
        {(m.enum ?? []).map((v) => <option value={v}>{v}</option>)}
      </select>
    );
  } else {
    const parts = lengthParts(own ?? eff);
    const range = parts ? sliderRange(parts.unit, parts.n) : null;
    control = (
      <div class="thx-one">
        {m.kind === "length" && parts && range && (
          <input type="range" aria-label={short(m.name) + " size"} min={range.min} max={range.max} step={range.step} value={parts.n}
            onInput={(e) => p.pick({ token: m.name, value: +(+e.currentTarget.value).toFixed(3) + parts.unit })} />
        )}
        <Field id={id} label="" value={own ?? ""} placeholder={eff} list={m.kind === "font" ? "thx-stacks" : undefined}
          onSet={(v) => p.op({ token: m.name, value: v })} />
      </div>
    );
  }
  return (
    <li class={"thx-tok" + (changed ? " changed" : "")}>
      <div class="thx-tok-head">
        <label for={id} title={m.desc}>{changed && <span class="thx-dot" aria-hidden="true" />}{short(m.name)}{changed && <span class="thx-sr"> (changed)</span>}</label>
        {changed && <button class="btn thx-reset" title="Back to the base theme's value"
          onClick={() => { if (own !== undefined) p.op({ token: m.name, mode: "light", value: "" }); if (ownDark !== undefined) p.op({ token: m.name, mode: "dark", value: "" }); }}>Reset</button>}
      </div>
      <span class="thx-desc">{m.desc}</span>
      {control}
    </li>
  );
}

export function ThemeEditor(p: { path: string; text: string; setText(next: string): void; undo(redo: boolean): void }) {
  const [spec, setSpec] = useState<Spec | null>(null);
  const [resp, setResp] = useState<Resp | null>(null);
  const [err, setErr] = useState("");
  const [note, setNote] = useState("");
  const [show, setShow] = useState<"light" | "dark" | "both">("both");
  const [size, setSize] = useState("desktop");
  const [slide, setSlide] = useState(0);
  const last = useRef("");
  const props = useRef(p);
  props.current = p;
  const chain = useRef<Promise<void>>(Promise.resolve());

  useEffect(() => { void themeSpec().then(setSpec).catch((e) => setErr(String(e))); }, []);
  const run = (op: Op | null) => {
    chain.current = chain.current.then(async () => {
      try {
        const r = await api.theme(props.current.path, last.current || props.current.text, op) as Resp;
        if (op) props.current.setText(r.text);
        last.current = r.text;
        setResp(r); setErr("");
      } catch (e) { setErr(e instanceof Error ? e.message : String(e)); }
    });
  };
  // a picker or a slider sends many values while it moves: send the last one
  const pickTimer = useRef(0);
  const pick = (o: Op) => { clearTimeout(pickTimer.current); pickTimer.current = window.setTimeout(() => run(o), 120); };

  // the editor's text changed (typing in the Text view, Undo): redraw from it
  useEffect(() => {
    if (p.text === last.current) return;
    const t = setTimeout(() => { last.current = p.text; run(null); }, 200);
    return () => clearTimeout(t);
  }, [p.text]);
  useEffect(() => { last.current = p.text; run(null); }, [p.path]);
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (!(e.metaKey || e.ctrlKey) || e.key.toLowerCase() !== "z" || /input|textarea|select/i.test((e.target as HTMLElement).tagName)) return;
      e.preventDefault();
      props.current.undo(e.shiftKey);
    };
    addEventListener("keydown", onKey);
    return () => removeEventListener("keydown", onKey);
  }, []);

  const use = async () => {
    try { const r = await api.themeUse(p.path); setNote(`${r.conf}: theme: ${r.theme}. Every deck in this folder without its own theme: uses it.`); }
    catch (e) { setNote(e instanceof Error ? e.message : String(e)); }
  };
  const slides = Math.max(1, (resp?.html.match(/<section[^>]*class="si-slide\b/g) ?? []).length);
  const modes = show === "both" ? (["light", "dark"] as const) : [show];
  const issues = resp?.issues ?? [];
  const changedCount = resp ? new Set([...Object.keys(resp.light), ...Object.keys(resp.dark)]).size : 0;
  return (
    <div class="mapx thx">
      <div class="mapx-canvas thx-canvas">
        <div class="thx-bar">
          <div class="seg" role="group" aria-label="Preview mode">
            {(["light", "dark", "both"] as const).map((m) => <button class={show === m ? "on" : ""} aria-pressed={show === m} onClick={() => setShow(m)}>{m[0].toUpperCase() + m.slice(1)}</button>)}
          </div>
          <div class="seg" role="group" aria-label="Screen size">
            {Object.entries(SIZES).map(([k, v]) => <button class={size === k ? "on" : ""} aria-pressed={size === k} onClick={() => setSize(k)}>{v[2]}</button>)}
          </div>
          <div class="seg" role="group" aria-label="Slide">
            <button aria-label="Previous slide" disabled={slide === 0} onClick={() => setSlide((s) => Math.max(0, s - 1))}>←</button>
            <span class="thx-n">{slide + 1} / {slides}</span>
            <button aria-label="Next slide" disabled={slide >= slides - 1} onClick={() => setSlide((s) => Math.min(slides - 1, s + 1))}>→</button>
          </div>
        </div>
        <div class={"thx-frames n" + modes.length + (SIZES[size][1] > SIZES[size][0] ? " tall" : "")}>
          {resp?.html ? modes.map((m) => <Frame key={m} html={resp.html} mode={m} size={size} slide={slide} />)
            : <div class="ed-msg"><div class="spinner" aria-label="Loading" /></div>}
        </div>
        {err && <div class="banner banner-err mapx-err" role="alert"><span>{err}</span></div>}
      </div>
      <aside class="mapx-side thx-side" aria-label="Theme panels">
        <datalist id="thx-stacks">{STACKS.map((s) => <option value={s} />)}</datalist>
        <section class="insp">
          <div class="insp-head"><b>Theme</b><span class="mute">{changedCount} token{changedCount === 1 ? "" : "s"} changed</span></div>
          <div class="insp-grid">
            <Field id="thx-name" label="Name" value={resp?.name ?? ""} placeholder="Acme" onSet={(v) => run({ field: "name", value: v })} />
            <label for="thx-extends">Starts from</label>
            <select id="thx-extends" value={resp?.extends ?? ""} onChange={(e) => run({ field: "extends", value: e.currentTarget.value })}>
              <option value="">nothing (every token here)</option>
              {(spec?.themes ?? []).map((t) => <option value={t.name}>{t.label}</option>)}
            </select>
          </div>
          <div class="mapx-actions">
            <button class="btn btn-pri" onClick={() => void use()}>Use in this folder</button>
          </div>
          <p class="mapx-hint">{note || <>Or one deck: <code>theme: {p.path}</code> in its front matter (a path from the deck).</>}</p>
          {resp?.extra && <p class="mapx-hint">This file has CSS rules of its own too; they are kept as they are (edit them in the Text view).</p>}
        </section>
        <section class="insp">
          <div class="insp-head"><b>Checks</b><span class="mute">{issues.length ? issues.length + " to look at" : "WCAG AA, light and dark: all pass"}</span></div>
          <ul class="thx-issues">
            {issues.map((is) => (
              <li class={is.mode ? "warn" : "err"}>
                <span>{is.mode ? <><b>{short(is.token ?? "")}</b> on {short(is.on ?? "")}, {is.mode}: {is.ratio}:1 (needs {is.want}:1)</> : is.msg}</span>
                {is.fix && <button class="btn" title={`Set ${is.token} to ${is.fix} in ${is.mode} mode`}
                  onClick={() => run({ token: is.token, mode: is.mode, value: is.fix! })}>Fix<i class="thx-sw" style={{ background: is.fix }} /></button>}
              </li>
            ))}
          </ul>
        </section>
        {spec && resp && spec.groups.map((g) => {
          const toks = spec.tokens.filter((t) => t.group === g);
          const n = toks.filter((t) => resp.light[t.name] !== undefined || resp.dark[t.name] !== undefined).length;
          return (
            <details class="insp thx-group" open={OPEN.has(g)}>
              <summary class="insp-head"><b>{g}</b><span class="mute">{n ? n + " changed" : toks.length + " tokens"}{g === "Colours" ? " · light | dark" : ""}</span></summary>
              <ul class="thx-toks">{toks.map((m) => <TokenRow m={m} r={resp} op={run} pick={pick} />)}</ul>
            </details>
          );
        })}
      </aside>
    </div>
  );
}

interface LogoTheme { path: string; name?: string; palette?: { hex: string; weight: number }[]; light?: Record<string, string>; dark?: Record<string, string> }
const LOGO_RE = /\.(png|jpe?g|gif|svg)$/i;
const images = (n: Node, out: string[] = []): string[] => {
  if (!n.children && LOGO_RE.test(n.path)) out.push(n.path);
  n.children?.forEach((c) => images(c, out));
  return out;
};

/** LogoPicker: a logo from the folder or from the computer, and the colours si found in it. */
function LogoPicker(p: { found: LogoTheme | null; busy: boolean; pick(logo: { path?: string; data?: string }): void; err(msg: string): void }) {
  const [files, setFiles] = useState<string[]>([]);
  const [path, setPath] = useState("");
  useEffect(() => { void api.tree().then((t) => setFiles(images(t))).catch(() => {}); }, []);
  const upload = (f: File | undefined) => {
    if (!f) return;
    if (f.size > 8 << 20) { p.err("A logo is at most 8 MiB."); return; }
    const rd = new FileReader();
    rd.onload = () => { setPath(""); p.pick({ data: String(rd.result).replace(/^data:[^,]*,/, "") }); };
    rd.readAsDataURL(f);
  };
  const f = p.found;
  const row = (m: Record<string, string> | undefined) => ["--si-bg", "--si-surface", "--si-fg", "--si-muted", "--si-accent", "--si-accent-2"].map((k) => <i title={`${k}: ${m?.[k]}`} style={{ background: m?.[k] }} />);
  return (
    <div class="thx-logo">
      <div class="thx-logo-pick">
        <label>
          <span>A logo in this folder</span>
          <select class="field" aria-label="Logo in this folder" value={path} onChange={(e) => { const v = e.currentTarget.value; setPath(v); if (v) p.pick({ path: v }); }}>
            <option value="">{files.length ? "Pick one…" : "no PNG, JPEG, GIF or SVG here"}</option>
            {files.map((x) => <option value={x}>{x}</option>)}
          </select>
        </label>
        <label>
          <span>Or one from your computer</span>
          <input type="file" class="field" aria-label="Logo file" accept=".png,.jpg,.jpeg,.gif,.svg,image/png,image/jpeg,image/gif,image/svg+xml" onChange={(e) => upload(e.currentTarget.files?.[0])} />
        </label>
      </div>
      {p.busy && <div class="spinner" aria-label="Reading the logo" />}
      {f?.palette && (
        <div class="thx-found" aria-live="polite">
          <div><span class="mute">In the logo</span><span class="thx-swatches thx-pal">{f.palette.map((c) => <i title={`${c.hex}, ${Math.round(c.weight * 100)}%`} style={{ background: c.hex }} />)}</span></div>
          <div><span class="mute">Light</span><span class="thx-swatches">{row(f.light)}</span></div>
          <div><span class="mute">Dark</span><span class="thx-swatches">{row(f.dark)}</span></div>
        </div>
      )}
    </div>
  );
}

/** NewThemeDialog: a name and a built-in (or a logo) to start from; the server writes themes/<name>.css. */
export function NewThemeDialog(p: { ok(path: string): void; onClose(): void }) {
  const [name, setName] = useState("");
  const [from, setFrom] = useState("studio");
  const [themes, setThemes] = useState<ThemeInfo[]>([]);
  const [err, setErr] = useState("");
  const [logo, setLogo] = useState<{ path?: string; data?: string } | null>(null);
  const [found, setFound] = useState<LogoTheme | null>(null);
  const [busy, setBusy] = useState(false);
  const input = useRef<HTMLInputElement>(null);
  useEffect(() => { input.current?.focus(); void themeSpec().then((s) => setThemes(s.themes)).catch(() => {}); }, []);
  const fromLogo = from === "logo";
  const s = themeSlug(name || (fromLogo ? found?.name ?? "" : ""));
  const pick = async (l: { path?: string; data?: string }) => {
    setLogo(l); setFound(null); setErr(""); setBusy(true);
    try { setFound(await api.themeLogo(l, name.trim(), false)); }
    catch (x) { setErr(x instanceof Error ? x.message : String(x)); }
    finally { setBusy(false); }
  };
  const submit = async (e: Event) => {
    e.preventDefault();
    if (fromLogo && !found) { setErr("Pick a logo."); return; }
    if (!s) { setErr("Give it a name."); return; }
    try {
      const r = fromLogo ? await api.themeLogo(logo!, name.trim() || found?.name || "", true) : await api.themeNew(name.trim(), from);
      p.ok(r.path); p.onClose();
    } catch (x) { setErr(x instanceof Error ? x.message : String(x)); }
  };
  return (
    <div class="scrim" onMouseDown={(e) => e.target === e.currentTarget && p.onClose()}>
      <div class="dialog dialog-wide" role="dialog" aria-modal="true" aria-label="New theme">
        <form onSubmit={submit} onKeyDown={(e) => e.key === "Escape" && p.onClose()}>
          <h2>New theme</h2>
          <div class="starter-grid" role="radiogroup" aria-label="Start from">
            {themes.map((t) => (
              <button type="button" role="radio" aria-checked={t.name === from} class={"starter thx-start" + (t.name === from ? " on" : "")} onClick={() => setFrom(t.name)}>
                <span class="thx-swatches" aria-hidden="true">
                  {["--si-bg", "--si-fg", "--si-accent", "--si-accent-2"].map((k) => <i style={{ background: t.light[k] }} />)}
                  {["--si-bg", "--si-fg", "--si-accent"].map((k) => <i style={{ background: t.dark[k] ?? t.light[k] }} />)}
                </span>
                <b>{t.label}</b>
                <span>{t.about}</span>
              </button>
            ))}
            <button type="button" role="radio" aria-checked={fromLogo} class={"starter thx-start" + (fromLogo ? " on" : "")} onClick={() => { setFrom("logo"); setErr(""); }}>
              <span class="thx-swatches" aria-hidden="true">{(found?.palette ?? []).slice(0, 4).map((c) => <i style={{ background: c.hex }} />)}{!found && <i class="thx-logo-ico" />}</span>
              <b>From a logo…</b>
              <span>Its colours, light and dark, every pair readable</span>
            </button>
          </div>
          {fromLogo && <LogoPicker found={found} busy={busy} pick={(l) => void pick(l)} err={setErr} />}
          <label class="starter-name">
            <span>Name</span>
            <input ref={input} class="field" value={name} spellcheck={false} placeholder={(fromLogo && found?.name) || "Acme"} onInput={(e) => { setName(e.currentTarget.value); setErr(""); }} />
          </label>
          <p class="dialog-body mute">{s ? <>Creates <code>themes/{s}.css</code>, {fromLogo ? "from the logo's colours, extending Studio" : <>starting from {themes.find((t) => t.name === from)?.label ?? from}</>}</> : " "}</p>
          {err && <p class="dialog-err" role="alert">{err}</p>}
          <div class="dialog-actions">
            <button type="button" class="btn" onClick={p.onClose}>Cancel</button>
            <button type="submit" class="btn btn-pri">Create</button>
          </div>
        </form>
      </div>
    </div>
  );
}
