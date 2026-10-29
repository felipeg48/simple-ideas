import { describe, expect, it } from "vitest";
import { slideForCursor } from "../src/slidemap";
import type { Slide } from "../src/api";

const s = (index: number, file: string, line: number): Slide => ({ index, file, line, title: "", topic: 0, minutes: 0 });

describe("slideForCursor", () => {
  const slides = [s(1, "a.md", 1), s(2, "sections/s.md", 1), s(3, "sections/s.md", 9)];
  it("maps a cursor line to the slide that starts at or before it", () => {
    expect(slideForCursor(slides, "sections/s.md", 12)).toBe(3);
    expect(slideForCursor(slides, "sections/s.md", 9)).toBe(3);
    expect(slideForCursor(slides, "sections/s.md", 8)).toBe(2);
    expect(slideForCursor(slides, "a.md", 5)).toBe(1);
    expect(slideForCursor(slides, "a.md", 0)).toBeNull();
    expect(slideForCursor(slides, "other.md", 3)).toBeNull();
  });
});
