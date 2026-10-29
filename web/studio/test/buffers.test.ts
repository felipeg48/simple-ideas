import { beforeEach, describe, expect, it, vi } from "vitest";
import { clear, restore, stash } from "../src/buffers";

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
});
