import { describe, expect, it } from "vitest";
import { survivingTabs, under } from "../src/tabs";

describe("under", () => {
  it("matches the path itself and everything inside it, not name prefixes", () => {
    expect(under("a.md", "a.md")).toBe(true);
    expect(under("sec/s.md", "sec")).toBe(true);
    expect(under("section/s.md", "sec")).toBe(false);
    expect(under("b.md", "a.md")).toBe(false);
  });
});

describe("survivingTabs", () => {
  it("drops clean tabs whose file is gone, keeps unsaved ones and reports them missing", () => {
    const exists = new Set(["a.md", "b.md"]);
    const r = survivingTabs(["a.md", "gone.md", "dirty.md", "b.md"], (p) => exists.has(p), new Set(["dirty.md", "b.md"]));
    expect(r.tabs).toEqual(["a.md", "dirty.md", "b.md"]);
    expect(r.missing).toEqual(["dirty.md"]);
  });
});
