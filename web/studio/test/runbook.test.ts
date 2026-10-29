import { describe, expect, it } from "vitest";
import caci from "./fixtures/caci-runbook.json?raw";
import { ops, parseRunbook, serializeRunbook, isRunbookText, validate, type Runbook } from "../src/runbook";


const ok = (t: string): Runbook => {
  const r = parseRunbook(t);
  if (!r.ok) throw new Error(r.error);
  return r.rb;
};
const sample = JSON.stringify({
  x: 1, categories: [{ name: "A", z: [1], entries: [{ command: "ls", label: "l", w: { a: 1 }, target: "sql" }, { label: "m", command: "pwd" }] }, { name: "B", entries: [] }], name: "R",
});

describe("runbook", () => {
  it("roundTripPreservesUnknownKeys", () => {
    const out = JSON.parse(serializeRunbook(ok(sample)));
    expect(Object.keys(out)).toEqual(["name", "categories", "x"]);
    expect(Object.keys(out.categories[0])).toEqual(["name", "entries", "z"]);
    expect(Object.keys(out.categories[0].entries[0])).toEqual(["label", "command", "target", "w"]);
    expect(out.categories[0].z).toEqual([1]);
  });
  it("serializeIsCanonical", () => {
    const s = serializeRunbook(ok(sample));
    expect(s.endsWith("}\n")).toBe(true);
    expect(s).toContain('\n  "name": "R"');
    expect(serializeRunbook(ok(s))).toBe(s);
  });
  it("caci fixture round trips", () => {
    const t = caci;
    const rb = ok(t);
    expect(rb.categories.length).toBe(14);
    expect(rb.categories.reduce((n, c) => n + c.entries.length, 0)).toBe(249);
    const s = serializeRunbook(rb);
    expect(JSON.parse(s)).toEqual(JSON.parse(t));
    expect(serializeRunbook(ok(s))).toBe(s);
    if (s !== t) console.log("caci fixture not byte-canonical");
  });
  it("parseReportsLineAndColumn", () => {
    const r = parseRunbook('{\n  "categories": [\n    ,\n  ]\n}');
    expect(r.ok).toBe(false);
    if (!r.ok) {
      expect(r.line).toBe(3);
      expect(r.col).toBeGreaterThan(0);
    }
  });
  it("rejectsNonRunbookJson", () => {
    for (const t of ["[]", "{}", '{"categories":1}', "3"]) {
      const r = parseRunbook(t);
      expect(r.ok).toBe(false);
      expect(isRunbookText(t)).toBe(false);
    }
    expect(isRunbookText('{"categories":[]}')).toBe(true);
  });
  it("opsArePure", () => {
    const rb = ok(sample);
    const snap = JSON.stringify(rb);
    const a = ops.addCategory(rb, "C", 0);
    expect(a.categories.map((c) => c.name)).toEqual(["C", "A", "B"]);
    expect(ops.renameCategory(rb, 1, "Q").categories[1].name).toBe("Q");
    expect(ops.deleteCategory(rb, 0).categories.length).toBe(1);
    expect(ops.duplicateCategory(rb, 0).categories.map((c) => c.name)).toEqual(["A", "A (copy)", "B"]);
    expect(ops.moveCategory(rb, 0, 1).categories.map((c) => c.name)).toEqual(["B", "A"]);
    expect(ops.addEntry(rb, 1, { label: "n", command: "c" }).categories[1].entries.length).toBe(1);
    expect(ops.updateEntry(rb, 0, 0, { label: "L" }).categories[0].entries[0].label).toBe("L");
    expect(ops.deleteEntry(rb, 0, 0).categories[0].entries.map((e) => e.label)).toEqual(["m"]);
    expect(ops.duplicateEntry(rb, 0, 0).categories[0].entries.map((e) => e.label)).toEqual(["l", "l (copy)", "m"]);
    expect(ops.moveEntry(rb, [0, 0], [0, 1]).categories[0].entries.map((e) => e.label)).toEqual(["m", "l"]);
    const x = ops.moveEntry(rb, [0, 0], [1, 0]);
    expect(x.categories[0].entries.map((e) => e.label)).toEqual(["m"]);
    expect(x.categories[1].entries.map((e) => e.label)).toEqual(["l"]);
    expect(JSON.stringify(rb)).toBe(snap);
  });
  it("validateFlags", () => {
    const rb = ok('{"categories":[{"name":"A","entries":[{"label":"","command":" ","target":"x"}]},{"name":"A","entries":[]}]}');
    const paths = validate(rb).map((v) => v.path);
    expect(paths).toEqual(["categories[0].entries[0].label", "categories[0].entries[0].command", "categories[1].name"]);
    rb.categories[0].entries[0].target = "x" as any;
    expect(validate(rb).map((v) => v.path)).toContain("categories[0].entries[0].target");
  });
});
