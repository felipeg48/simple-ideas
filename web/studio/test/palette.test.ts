import { describe, expect, it } from "vitest";
import { fuzzy } from "../src/palette";

describe("fuzzy", () => {
  it("prefers a substring, then letters in order", () => {
    expect(fuzzy("retries.md", "retr")).toBeGreaterThan(fuzzy("rehearse-tracker.md", "retr"));
    expect(fuzzy("Mode: Rehearse", "reh")).toBeGreaterThan(0);
    expect(fuzzy("showcase.md", "shcs")).toBeGreaterThan(0);
    expect(fuzzy("index.md", "xyz")).toBe(-1);
    expect(fuzzy("anything", "")).toBe(0);
  });
});
