import { describe, expect, it } from "vitest";
import { EditorState } from "@codemirror/state";
import { CompletionContext, type CompletionResult, type CompletionSource } from "@codemirror/autocomplete";
import { siMarkdown } from "../src/si-lang";
import type { Rules } from "../src/api";

const rules: Rules = {
  directives: { include: "file", layout: "slide", transition: "slide", svg: "body", steps: "body" },
  layouts: ["two-col", "big-number", "quote"],
  transitions: ["fade", "slide"],
  metaKeys: ["title", "author", "theme"],
};

async function complete(doc: string, pos = doc.length): Promise<string[]> {
  const state = EditorState.create({ doc, extensions: siMarkdown(rules) });
  const sources = state.languageDataAt<CompletionSource>("autocomplete", pos);
  const labels: string[] = [];
  for (const src of sources) {
    const r = (await src(new CompletionContext(state, pos, false))) as CompletionResult | null;
    if (r) labels.push(...r.options.map((o) => o.label));
  }
  return labels;
}

describe("siMarkdown", () => {
  it("completes directive names after <!-- ", async () => {
    const got = await complete("# T\n\n<!-- ");
    expect(got).toEqual(expect.arrayContaining(["layout", "steps", "svg"]));
  });
  it("completes layouts and transitions after their directive", async () => {
    expect(await complete("<!-- layout: ")).toEqual(expect.arrayContaining(["two-col", "quote"]));
    expect(await complete("<!-- transition: fa")).toContain("fade");
  });
  it("completes front-matter keys only inside the leading --- block", async () => {
    expect(await complete("---\nti", 6)).toContain("title");
    expect(await complete("---\ntitle: x\n---\n\nti")).not.toContain("title");
  });
});

describe("siMarkdown sources", () => {
  it("keeps one completion source across calls (the plugin matches sources by identity)", () => {
    const state = EditorState.create({ doc: "<!-- ", extensions: siMarkdown(rules) });
    const a = state.languageDataAt("autocomplete", 5), b = state.languageDataAt("autocomplete", 5);
    expect(a.length).toBeGreaterThan(0);
    expect(a).toEqual(b);
  });
});
