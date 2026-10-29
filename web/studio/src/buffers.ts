// Unsaved editor text survives a crash or a closed tab in localStorage.

/** workspaceId is set by the server on #app (data-ws): a hash of the folder Studio serves. */
const workspaceId = () => (typeof document !== "undefined" && document.getElementById("app")?.dataset.ws) || "";

/** scoped prefixes a localStorage key with the workspace, so two folders served
 * from the same origin (same port, one after the other) never mix tabs or buffers. */
export const scoped = (key: string, ws = workspaceId()) => (ws ? `si-studio@${ws}:` : "") + key;

const key = (path: string) => scoped("si-studio-buffer:" + path);

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
