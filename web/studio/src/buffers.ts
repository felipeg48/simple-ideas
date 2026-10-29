// Unsaved editor text survives a crash or a closed tab in localStorage.
const key = (path: string) => "si-studio-buffer:" + path;

export function stash(path: string, text: string, etag: string): void {
  try { localStorage.setItem(key(path), JSON.stringify({ text, etag })); } catch { /* quota or disabled storage: best effort */ }
}

export function restore(path: string): { text: string; etag: string } | null {
  try {
    const v = JSON.parse(localStorage.getItem(key(path)) || "null");
    return v && typeof v.text === "string" ? { text: v.text, etag: String(v.etag ?? "") } : null;
  } catch {
    return null;
  }
}

export function clear(path: string): void {
  try { localStorage.removeItem(key(path)); } catch { /* ignore */ }
}
