import { describe, expect, it } from "vitest";
import { EditorState } from "@codemirror/state";
import { CompletionContext, type CompletionResult, type CompletionSource } from "@codemirror/autocomplete";
import { HL_RE, relPath, setCompletionFiles, siMarkdown } from "../src/si-lang";
import type { Rules } from "../src/api";

const rules: Rules = {
  directives: { include: "file", layout: "slide", transition: "slide", svg: "body", steps: "body" },
  layouts: ["two-col", "big-number", "quote"],
  transitions: ["fade", "slide"],
  metaKeys: ["title", "author", "theme", "type", "fonts"],
  help: { alert: "a dismissible band", type: "deck, lab, article …" },
  values: { type: ["deck", "article", "site"], theme: ["studio", "paper"], kinds: ["info", "ok", "warn", "crit"] },
  paths: ["include", "svg", "embed", "theme"],
};
rules.directives.alert = "body";
rules.directives.embed = "body";

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

describe("values and paths", () => {
  it("completes front-matter values", async () => {
    expect(await complete("---\ntype: ar", 12)).toContain("article");
    expect(await complete("---\ntheme: ", 11)).toEqual(expect.arrayContaining(["studio", "paper"]));
  });
  it("completes alert kinds after a ·", async () => {
    expect(await complete("<!-- alert: Closed Monday · w")).toContain("warn");
    expect(await complete("<!-- alert: Clo")).toEqual([]);
  });
  it("completes workspace paths relative to the open file", async () => {
    setCompletionFiles(["blog/post.md", "talks/a.md", "images/aisle.svg", "index.md", "_site.md", "docs/_nav.md"], "blog/post.md");
    const got = await complete("<!-- embed: ");
    expect(got).toEqual(expect.arrayContaining(["../talks/a.md", "../index.md", "../talks/"]));
    expect(got).not.toContain("../images/aisle.svg");
    expect(got).not.toContain("../_site.md");
    expect(got).toContain("../docs/");
    expect(await complete("<!-- svg: ")).toEqual(["../images/aisle.svg"]);
  });
  it("stops after the directive closes", async () => {
    expect(await complete("<!-- layout: two-col -->")).toEqual([]);
  });
  it("relPath walks up and down", () => {
    expect(relPath("a/b/c.md", "a/d.md")).toBe("../d.md");
    expect(relPath("x.md", "a/b.md")).toBe("a/b.md");
    expect(relPath("a/x.md", "a/y.md")).toBe("y.md");
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

describe("highlight marks", () => {
  it("match ==phrases==, not a single = or ===", () => {
    const text = "## Five ==to keep== and a = b and ===no=== and ==two== more";
    expect([...text.matchAll(HL_RE)].map((m) => m[0])).toEqual(["==to keep==", "==two=="]);
  });
});

describe("fence, include and callout completion", () => {
  it("offers languages right after ``` on an opening fence, not a closing one", async () => {
    expect(await complete("```cs")).toContain("csharp");
    expect(await complete("```go\nx\n```")).toEqual([]);
  });
  it("offers code window options and their values", async () => {
    expect(await complete("```go ")).toEqual(expect.arrayContaining(["window", "title", "numbers", "id", "transition"]));
    expect(await complete("```go window=")).toEqual(["mac", "windows", "terminal", "none"]);
    expect(await complete("```go numbers backdrop=")).toContain("graphite");
  });
  it("offers mind map options after ```mindmap", async () => {
    expect(await complete("```mindmap ")).toEqual(expect.arrayContaining(["layout", "view", "kind", "steps", "flow"]));
    expect(await complete("```mindmap kind=")).toEqual(["blocks", "bricks", "toy"]);
  });
  it("offers title= and lang= after an include path, and languages for lang=", async () => {
    expect(await complete("<!-- include: ../code/NuGet.config ")).toEqual(["title", "lang"]);
    expect(await complete("<!-- include: ../code/NuGet.config lang=")).toContain("xml");
  });
  it("offers the block's line numbers after callout:, then at=", async () => {
    const doc = "```go window=mac\nfunc a() {\n\treturn\n}\n```\n<!-- callout: ";
    expect(await complete(doc)).toEqual(["1", "2", "3"]);
    expect(await complete("```go\nx\n```\n<!-- callout: 1 Note ")).toEqual(["at="]);
  });
});
