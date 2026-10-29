import type { Slide } from "./api";

/** slideForCursor returns the 1-based index of the last slide in file starting at or before line. */
export function slideForCursor(slides: Slide[], file: string, line: number): number | null {
  let hit: number | null = null;
  for (const s of slides) if (s.file === file && s.line <= line) hit = s.index;
  return hit;
}
