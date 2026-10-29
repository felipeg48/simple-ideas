// Typed client for the Studio backend (/_si/studio/api/), same origin, cookie auth.

export type Kind = "dir" | "deck" | "markdown" | "svg" | "css" | "json" | "image" | "other";
export interface Node { name: string; path: string; kind: Kind; children?: Node[] }
export interface FileData { path: string; content: string; etag: string }
export interface DeckInfo { path: string; title: string; files: string[] }
export interface Problem { file: string; line: number; msg: string }
export interface Slide { index: number; file: string; line: number; title: string; topic: number; minutes: number }
export interface Check { problems: Problem[]; slides: Slide[]; minutes: number; steps: number; canvases: number }
export interface Rules { directives: Record<string, string>; layouts: string[]; transitions: string[]; metaKeys: string[] }

/** ApiError carries the HTTP status and the decoded JSON body (if any). */
export class ApiError extends Error {
  constructor(public status: number, public body: any) {
    super((body && body.error) || `HTTP ${status}`);
  }
}

const base = "/_si/studio/api/";
const q = (p: string) => encodeURIComponent(p);

async function call<T>(method: string, path: string, body?: unknown): Promise<T> {
  const r = await fetch(base + path, {
    method,
    headers: body === undefined ? undefined : { "Content-Type": "application/json" },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  const text = await r.text();
  let data: any = null;
  try { data = text ? JSON.parse(text) : null; } catch { data = { error: text.trim() || r.statusText }; }
  if (!r.ok) throw new ApiError(r.status, data);
  return data as T;
}

export const api = {
  tree: () => call<Node>("GET", "tree"),
  read: (path: string) => call<FileData>("GET", `file?path=${q(path)}`),
  write: (path: string, content: string, etag: string) =>
    call<{ etag: string }>("PUT", `file?path=${q(path)}`, { content, etag }),
  create: (path: string, dir: boolean, content = "") =>
    call<{ path: string; etag?: string }>("POST", "create", { path, content, dir }),
  rename: (from: string, to: string) => call<unknown>("POST", "rename", { from, to }),
  remove: (path: string) => call<unknown>("DELETE", `file?path=${q(path)}`),
  decks: () => call<DeckInfo[]>("GET", "decks"),
  check: (deck: string) => call<Check>("GET", `check?deck=${q(deck)}`),
  rules: () => call<Rules>("GET", "rules"),
};

/** deckURL is the live-rendered page of a deck: "talks/a.md" → "/talks/a.html". */
export const deckURL = (deck: string) => "/" + deck.replace(/\.md$/i, "") + ".html";
