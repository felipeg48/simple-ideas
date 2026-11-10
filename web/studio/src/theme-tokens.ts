// Small helpers of the Theme view, kept apart so vitest can reach them.

/** isThemeFile: a .css file the Theme view can edit: an si theme header, or --si-* tokens on :root. */
export const isThemeFile = (text: string) =>
  /^\s*\/\*\s*si theme:/.test(text) || /(^|\n)\s*:root\s*\{[^}]*--si-/.test(text);

/** hex6 turns a CSS colour into #rrggbb for <input type="color">, or "" when it is not a hex colour. */
export function hex6(v: string | undefined): string {
  const m = /^#([0-9a-f]{3,8})$/i.exec((v ?? "").trim());
  if (!m) return "";
  const h = m[1];
  if (h.length === 3 || h.length === 4) return "#" + h.slice(0, 3).split("").map((c) => c + c).join("").toLowerCase();
  if (h.length === 6 || h.length === 8) return "#" + h.slice(0, 6).toLowerCase();
  return "";
}

/** lengthParts splits a plain length ("3.4em") for a slider; null for anything else (max(), var()). */
export function lengthParts(v: string | undefined): { n: number; unit: string } | null {
  const m = /^(-?\d*\.?\d+)(em|rem|cqi|cqb|%|ch)$/.exec((v ?? "").trim());
  return m ? { n: parseFloat(m[1]), unit: m[2] } : null;
}

/** sliderRange is a slider's min, max and step for a length in its unit. */
export function sliderRange(unit: string, n: number): { min: number; max: number; step: number } {
  if (unit === "%" || unit === "ch") return { min: 0, max: Math.max(120, n * 2), step: 1 };
  if (unit === "cqi" || unit === "cqb") return { min: 0, max: Math.max(80, n * 2), step: 0.5 };
  return { min: unit === "em" && n < 0 ? -0.2 : 0, max: Math.max(6, n * 2), step: n !== 0 && Math.abs(n) < 0.2 ? 0.005 : 0.05 };
}

/** short is a token's label: --si-accent-2 → accent-2. */
export const short = (name: string) => name.replace(/^--si-/, "");

/** themeSlug: "Acme Brand" → acme-brand (the server makes the same file name). */
export const themeSlug = (s: string) => s.trim().toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-+|-+$/g, "");
