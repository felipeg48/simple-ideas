export type ThemeMode = "dark" | "light" | "system";
const KEY = "si-studio-theme";
const dark = () => matchMedia("(prefers-color-scheme: dark)");

export function storedTheme(): ThemeMode {
  try {
    const v = localStorage.getItem(KEY);
    if (v === "dark" || v === "light" || v === "system") return v;
  } catch { /* storage disabled */ }
  return "system";
}

/** resolvedTheme is the palette actually shown. */
export const resolvedTheme = (): "dark" | "light" =>
  document.documentElement.dataset.theme === "light" ? "light" : "dark";

let listening = false;

/** applyTheme sets <html data-theme> (system follows the OS live) and persists the choice. */
export function applyTheme(mode: ThemeMode): void {
  const root = document.documentElement;
  root.dataset.mode = mode;
  root.dataset.theme = mode === "system" ? (dark().matches ? "dark" : "light") : mode;
  try { localStorage.setItem(KEY, mode); } catch { /* ignore */ }
  if (!listening) {
    listening = true;
    dark().addEventListener("change", () => {
      if (root.dataset.mode === "system") applyTheme("system");
    });
  }
  window.dispatchEvent(new CustomEvent("si-theme"));
}
