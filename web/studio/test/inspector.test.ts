import { describe, expect, it } from "vitest";
import { setDirective } from "../src/inspector-edit";

const deck = "# Title\n\n---\n\n## Two\n\nBody.\n\n---\n\n## Three\n";

describe("setDirective", () => {
  it("adds a directive under the slide heading", () => {
    expect(setDirective(deck, 5, 8, "layout", "two-col")).toBe("# Title\n\n---\n\n## Two\n\n<!-- layout: two-col -->\n\nBody.\n\n---\n\n## Three\n");
  });
  it("replaces one that is there, only in that slide", () => {
    const t = "## A\n\n<!-- layout: quote -->\n\n---\n\n## B\n\n<!-- layout: quote -->\n";
    expect(setDirective(t, 1, 4, "layout", "big-number")).toBe("## A\n\n<!-- layout: big-number -->\n\n---\n\n## B\n\n<!-- layout: quote -->\n");
  });
  it("removes it when the value is empty, without leaving a double gap", () => {
    expect(setDirective("## A\n\n<!-- layout: quote -->\n\nBody\n", 1, 5, "layout", "")).toBe("## A\n\nBody\n");
  });
  it("puts it first in a slide without a heading", () => {
    expect(setDirective("**98%**\n\nkept\n", 1, 3, "layout", "big-number")).toBe("<!-- layout: big-number -->\n\n**98%**\n\nkept\n");
  });
  it("leaves the text alone when removing what is not there", () => {
    expect(setDirective(deck, 5, 8, "class", "")).toBe(deck);
  });
});
