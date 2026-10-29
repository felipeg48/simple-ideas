// The live preview: the real deck page in an iframe (it hot-reloads itself),
// rendered at a real device size and scaled to fit; it follows the cursor's slide.
import { useEffect, useLayoutEffect, useMemo, useRef, useState } from "preact/hooks";
import { type DeckInfo, deckURL } from "./api";
import { Icon } from "./navigator";
import { resolvedTheme } from "./theme";

const presets = [
  { id: "fit", label: "Fit", title: "Fill the pane", w: 0, h: 0 },
  { id: "16:9", label: "16:9", title: "Laptop, 1440 × 810", w: 1440, h: 810 },
  { id: "phone", label: "Phone", title: "Phone, 400 × 800", w: 400, h: 800 },
  { id: "portrait", label: "Portrait", title: "Tablet portrait, 820 × 1180", w: 820, h: 1180 },
  { id: "4k", label: "4K", title: "4K screen, 3840 × 2160", w: 3840, h: 2160 },
];
const KEY = "si-studio-preview";

export function Preview(p: { deck: DeckInfo | null; slide: number | null; total: number }) {
  const frame = useRef<HTMLIFrameElement>(null);
  const stage = useRef<HTMLDivElement>(null);
  const [preset, setPreset] = useState(() => { try { return localStorage.getItem(KEY) || "16:9"; } catch { return "16:9"; } });
  const [box, setBox] = useState({ w: 0, h: 0 });
  const [loaded, setLoaded] = useState(false);
  const src = p.deck ? deckURL(p.deck.path) : "";
  // the hash is only the starting slide: later moves go through location.hash, never a reload
  const first = useMemo(() => src + (p.slide ? "#" + p.slide : ""), [src]);

  useLayoutEffect(() => {
    const el = stage.current!;
    const ro = new ResizeObserver(() => setBox({ w: el.clientWidth, h: el.clientHeight }));
    ro.observe(el);
    return () => ro.disconnect();
  }, []);

  const syncMode = () => {
    try { frame.current?.contentDocument?.documentElement.setAttribute("data-si-mode", resolvedTheme()); } catch { /* not loaded */ }
  };
  const syncSlide = () => {
    const w = frame.current?.contentWindow;
    try { if (w && p.slide && w.location.hash.split(".")[0] !== "#" + p.slide) w.location.hash = "#" + p.slide; } catch { /* not loaded */ }
  };
  useEffect(() => { addEventListener("si-theme", syncMode); return () => removeEventListener("si-theme", syncMode); }, []);
  useEffect(syncSlide, [p.slide, loaded]);
  useEffect(() => setLoaded(false), [src]);

  const pr = presets.find((x) => x.id === preset) || presets[1];
  const pad = 24, aw = Math.max(box.w - pad * 2, 1), ah = Math.max(box.h - pad * 2, 1);
  const scale = pr.w ? Math.min(aw / pr.w, ah / pr.h) : 1;
  const frameStyle = pr.w
    ? { width: pr.w + "px", height: pr.h + "px", transform: `scale(${scale})` }
    : { width: "100%", height: "100%" };
  const shell = pr.w ? { width: pr.w * scale + "px", height: pr.h * scale + "px" } : { inset: "0" };

  return (
    <section class="prev" aria-label="Live preview">
      <div class="ptool">
        <div class="seg" role="radiogroup" aria-label="Preview size">
          {presets.map((x) => (
            <button role="radio" aria-checked={x.id === preset} class={x.id === preset ? "on" : ""} title={x.title}
              onClick={() => { setPreset(x.id); try { localStorage.setItem(KEY, x.id); } catch { /* ignore */ } }}>{x.label}</button>
          ))}
        </div>
        <span class="ptool-info">
          {p.deck ? <>{p.slide ? <>Slide <b>{p.slide}</b> / {p.total}</> : p.deck.title}{pr.w ? <span class="zoom"> · {Math.round(scale * 100)}%</span> : null}</> : null}
        </span>
        {p.deck && (
          <a class="icon-btn" href={src + (p.slide ? "#" + p.slide : "")} target="_blank" rel="noopener" aria-label="Open deck in a new tab" title="Open deck in a new tab">
            <Icon name="external" />
          </a>
        )}
      </div>
      <div class={"stage" + (pr.w ? " framed" : "")} ref={stage}>
        {p.deck ? (
          // mount only once measured: a frame loaded at near-zero scale may never paint
          box.w > 0 && <div class="frame-shell" style={shell}>
            <iframe ref={frame} src={first} title={`Preview of ${p.deck.title}`} style={frameStyle}
              onLoad={() => { syncMode(); setLoaded(true); }} />
          </div>
        ) : (
          <div class="prev-empty">
            <Icon name="deck" />
            <p>Open a deck, or a file it includes, to see it rendered live here.</p>
          </div>
        )}
      </div>
    </section>
  );
}
