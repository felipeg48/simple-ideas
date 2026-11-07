// About and Help: si's mark, its version and build, the docs site, Report a
// Problem / Request a Feature, and the open-source parts it is built with. The
// facts come from internal/brand, written into the page as #si-about.
import type { JSX } from "preact";

export interface AboutInfo {
  name: string; tagline: string; about: string; version: string; commit: string; date: string;
  site: string; repo: string; pages: Record<string, string>; os: string; copyright: string; license: string;
  credits: { name: string; license: string; url: string; note?: string }[];
  bug: string; bugDesktop: string; feature: string; featureDesktop: string;
}

export const about: AboutInfo = (() => {
  try { return JSON.parse(document.getElementById("si-about")?.textContent || "{}"); } catch { return {} as AboutInfo; }
})();

// the Stack: three isometric slabs, the deck, the article and the book from one file
function slab(y: number, color: string, o: number): JSX.Element {
  return (
    <g fill={color}>
      <path d={`M32 ${y}l22 11-22 11-22-11z`} opacity={o} />
      <path d={`M10 ${y + 11}v5l22 11v-5z`} opacity={o * 0.75} />
      <path d={`M54 ${y + 11}v5L32 ${y + 27}v-5z`} opacity={o * 0.55} />
    </g>
  );
}
export function Mark(p: { size?: number; class?: string }): JSX.Element {
  return (
    <svg class={p.class} width={p.size ?? 24} height={p.size ?? 24} viewBox="0 0 64 64" aria-hidden="true">
      {slab(30, "#1fc8a0", 0.55)}{slab(19, "#1fc8a0", 0.85)}{slab(8, "#5b4bdb", 1)}
    </svg>
  );
}

// help opens a page of the docs site, or a new issue; desktop says whether
// the desktop app asked (the issue form's "where" field)
export function help(what: string, desktop = false) {
  const url = what === "report" ? (desktop ? about.bugDesktop : about.bug)
    : what === "feature" ? (desktop ? about.featureDesktop : about.feature)
      : about.pages?.[what] || about.site;
  if (url) window.open(url, "_blank", "noopener");
}

export function About(p: { onClose(): void; binary?: { path?: string; note?: string } }) {
  const a = about;
  const link = (href: string, label: string) => <a href={href} target="_blank" rel="noopener">{label}</a>;
  const desktop = !!p.binary;
  return (
    <div class="scrim scrim-top" onMouseDown={(e) => e.target === e.currentTarget && p.onClose()}>
      <div class="dialog about" role="dialog" aria-modal="true" aria-labelledby="about-title" onKeyDown={(e) => e.key === "Escape" && p.onClose()}>
        <div class="about-head">
          <Mark size={56} />
          <div>
            <h2 id="about-title">{a.name}</h2>
            <p class="mute">{a.tagline}</p>
          </div>
        </div>
        <p class="about-ver">Version {a.version}{a.commit ? <span class="mute"> · {a.commit}{a.date ? " · " + a.date.slice(0, 10) : ""}</span> : null}</p>
        {p.binary?.path && <p class="mute about-bin">Runs {p.binary.path}{p.binary.note ? " — " + p.binary.note : ""}</p>}
        <p>{a.about}</p>
        <p class="about-links">
          {link(a.site, "Website")}{link(a.repo, "GitHub")}
          {link(desktop ? a.bugDesktop : a.bug, "Report a Problem")}{link(desktop ? a.featureDesktop : a.feature, "Request a Feature")}
          {link(a.pages?.releases, "Release Notes")}
        </p>
        <h3>Built with</h3>
        <ul class="about-credits">
          {(a.credits || []).map((c) => <li>{link(c.url, c.name)} <span class="mute">{c.license}{c.note ? " · " + c.note : ""}</span></li>)}
        </ul>
        <p class="mute">{a.copyright} · {a.license}</p>
        <div class="dialog-actions"><button class="btn btn-pri" onClick={p.onClose} autoFocus>Close</button></div>
      </div>
    </div>
  );
}
