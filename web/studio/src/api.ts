// Typed client for the Studio backend (/_si/studio/api/), same origin, cookie auth.

export type Kind = "dir" | "deck" | "lab" | "article" | "guide" | "book" | "demo" | "site" | "page" | "mindmap" | "nav"
  | "markdown" | "svg" | "css" | "json" | "image" | "other";
/** guide marks a folder that holds a guide, book or site. */
export interface Node { name: string; path: string; kind: Kind; guide?: "guide" | "book" | "site"; children?: Node[] }
export interface FileData { path: string; content: string; etag: string }
/** A deck, document page, or a guide / book / site (its _nav.md or _site.md); url is its live page. */
export interface DeckInfo { path: string; title: string; files: string[]; runbook?: string; kind: Kind; url: string; guide?: string }
export interface Problem { file: string; line: number; msg: string }
export interface Slide {
  index: number; file: string; line: number; title: string; topic: number; minutes: number;
  layout?: string; transition?: string; background?: string; class?: string; eyebrow?: string;
  words?: number; clicks?: number; steps?: number;
}
export interface Check { problems: Problem[]; slides: Slide[]; minutes: number; steps: number; canvases: number; kind?: Kind; maps?: number; topics?: number }
export interface Rules {
  directives: Record<string, string>; layouts: string[]; transitions: string[]; metaKeys: string[];
  /** one-line help per directive and front-matter key */
  help?: Record<string, string>;
  /** accepted values per directive or key ("kinds": alert / badge kinds after a ·) */
  values?: Record<string, string[]>;
  /** directives and keys whose value is a workspace path */
  paths?: string[];
}

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
  checkAll: () => call<{ problems: Problem[]; checked: number }>("GET", "checkall"),
  build: (pdf: boolean) => call<{ code: number; output: string; url: string }>("POST", "build", { pdf }),
  /** mindmap redraws a mind map from text (a page, or the ```mindmap fence holding line), applying op first if given. */
  mindmap: (path: string, text: string, line: number, op: unknown) => call<any>("POST", "mindmap", { path, text, line, op }),
  /** codewin redraws the code window (or show) holding line, applying op first if given. */
  codewin: (path: string, text: string, line: number, op: unknown) => call<any>("POST", "codewin", { path, text, line, op }),
  /** codewinExport runs si code on the saved file: png, svg or gif next to it. */
  codewinExport: (path: string, format: string, id: string) => call<{ code: number; output: string }>("POST", "codewin/export", { path, format, id }),
  starter: (kind: Kind, path: string, title: string) =>
    call<{ created: string[]; open: string }>("POST", "starter", { kind, path, title }),
};

/** deckURL is the live-rendered page of a deck: "talks/a.md" → "/talks/a.html". */
export const deckURL = (deck: string) => "/" + deck.replace(/\.(md|adoc)$/i, "") + ".html";

/** pageURL is the live page of a deck or document (a guide's nav file opens its first page). */
export const pageURL = (d: DeckInfo) => d.url || deckURL(d.path);

/** isSlides: the kind is shown as slides (the preview follows the cursor's slide). */
export const isSlides = (k?: Kind | null) => k === "deck";

/** isMarkdownKind: an si source file (deck, any document, a nav file) or plain Markdown. */
export const isMarkdownKind = (k: Kind) =>
  ["deck", "lab", "article", "guide", "book", "demo", "site", "page", "mindmap", "nav", "markdown"].includes(k);

/** kindLabel names a kind for people. */
export const kindLabel: Record<string, string> = {
  deck: "Slides", lab: "Lab", article: "Article", guide: "Guide page", book: "Book page", demo: "Demo script",
  site: "Site", page: "Site page", mindmap: "Mind map", nav: "Navigation", markdown: "Markdown", svg: "SVG", css: "CSS", json: "JSON",
  image: "Image", other: "Text",
};
