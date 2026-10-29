import { afterEach, describe, expect, it, vi } from "vitest";
import { createAutosave } from "../src/autosave";

describe("createAutosave", () => {
  afterEach(() => vi.useRealTimers());
  it("saves once after the delay, flush saves immediately", async () => {
    vi.useFakeTimers();
    const save = vi.fn(() => Promise.resolve());
    const a = createAutosave(save, 700);
    a.touch();
    vi.advanceTimersByTime(400);
    a.touch();
    vi.advanceTimersByTime(600);
    expect(save).not.toHaveBeenCalled();
    vi.advanceTimersByTime(100);
    expect(save).toHaveBeenCalledTimes(1);

    a.touch();
    await a.flush();
    expect(save).toHaveBeenCalledTimes(2);
    vi.advanceTimersByTime(2000);
    expect(save).toHaveBeenCalledTimes(2);

    await a.flush(); // nothing pending: no save
    expect(save).toHaveBeenCalledTimes(2);
  });
});
