import { describe, expect, it } from "vitest";
import { eolOf, toLF, withEol } from "../src/eol";

describe("eol", () => {
  it("roundTripsCRLF", () => {
    const disk = "# A\r\n\r\nbody\r\n";
    expect(eolOf(disk)).toBe("\r\n");
    expect(toLF(disk)).toBe("# A\n\nbody\n");
    expect(withEol(toLF(disk), eolOf(disk))).toBe(disk);
  });
  it("leavesLFAlone", () => {
    expect(eolOf("a\nb\n")).toBe("\n");
    expect(withEol("a\nb\n", "\n")).toBe("a\nb\n");
  });
});
