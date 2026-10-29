import { beforeEach, describe, expect, it, vi } from "vitest";
import { clear, restore, scoped, stash } from "../src/buffers";

describe("buffers", () => {
  beforeEach(() => {
    const m = new Map<string, string>();
    vi.stubGlobal("localStorage", {
      getItem: (k: string) => m.get(k) ?? null,
      setItem: (k: string, v: string) => void m.set(k, v),
      removeItem: (k: string) => void m.delete(k),
    });
  });
  it("restoresDirtyBuffer", () => {
    expect(restore("a.md")).toBeNull();
    stash("a.md", "# hi", "e1");
    expect(localStorage.getItem("si-studio-buffer:a.md")).not.toBeNull();
    expect(restore("a.md")).toEqual({ text: "# hi", etag: "e1" });
    clear("a.md");
    expect(restore("a.md")).toBeNull();
  });
  it("scopesKeysByWorkspace", () => {
    expect(scoped("si-studio-tabs", "abc")).toBe("si-studio@abc:si-studio-tabs");
    expect(scoped("si-studio-tabs", "def")).not.toBe(scoped("si-studio-tabs", "abc"));
    expect(scoped("si-studio-tabs", "")).toBe("si-studio-tabs");
  });
  it("stashIsScopedToTheServedWorkspace", () => {
    vi.stubGlobal("document", { getElementById: () => ({ dataset: { ws: "aaa" } }) });
    stash("a.md", "course A", "e1");
    vi.stubGlobal("document", { getElementById: () => ({ dataset: { ws: "bbb" } }) });
    expect(restore("a.md")).toBeNull();
    vi.stubGlobal("document", { getElementById: () => ({ dataset: { ws: "aaa" } }) });
    expect(restore("a.md")).toEqual({ text: "course A", etag: "e1" });
    vi.unstubAllGlobals();
  });
});
