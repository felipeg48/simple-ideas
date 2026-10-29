import { describe, expect, it } from "vitest";
import { linkRunbook, metaValue, relPath, slugify } from "../src/runbook";

describe("generator helpers", () => {
  it("slugify", () => {
    expect(slugify("Lab ops")).toBe("lab-ops");
    expect(slugify("  RabbitMQ: Health & Alarms!  ")).toBe("rabbitmq-health-alarms");
    expect(slugify("Ünïcode café")).toBe("unicode-cafe");
    expect(slugify("***")).toBe("");
  });
  it("relPath is relative to the deck's folder", () => {
    expect(relPath("talk.md", "lab-ops.json")).toBe("lab-ops.json");
    expect(relPath("talk.md", "labs/x.json")).toBe("labs/x.json");
    expect(relPath("a/b/talk.md", "labs/x.json")).toBe("../../labs/x.json");
    expect(relPath("labs/talk.md", "labs/x.json")).toBe("x.json");
    expect(relPath("a/talk.md", "ab/x.json")).toBe("../ab/x.json");
  });
  it("linkRunbook adds, replaces or creates front matter", () => {
    expect(linkRunbook("---\ntitle: T\n---\n\n# T\n", "r.json")).toBe("---\ntitle: T\nrunbook: r.json\n---\n\n# T\n");
    expect(linkRunbook("---\ntitle: T\nrunbook: old.json\ntheme: x\n---\n# T\n", "r.json")).toBe("---\ntitle: T\nrunbook: r.json\ntheme: x\n---\n# T\n");
    expect(linkRunbook("# T\n", "r.json")).toBe("---\nrunbook: r.json\n---\n\n# T\n");
    expect(linkRunbook("---\r\ntitle: T\r\n---\r\n# T\r\n", "r.json")).toBe("---\r\ntitle: T\r\nrunbook: r.json\r\n---\r\n# T\r\n");
    // a --- slide separator later in the file is not front matter
    expect(linkRunbook("# T\n\n---\n\n# U\n", "r.json")).toBe("---\nrunbook: r.json\n---\n\n# T\n\n---\n\n# U\n");
  });

  it("linkRunbook refuses AsciiDoc decks", () => {
    expect(() => linkRunbook("= T\n", "r.json", "talks/t.adoc")).toThrow(/AsciiDoc/);
    expect(linkRunbook("# T\n", "r.json", "talks/t.md")).toBe("---\nrunbook: r.json\n---\n\n# T\n");
  });

  it("linkRunbook reads front matter exactly like the Go parser", () => {
    // a leading slide separator with slide content is not front matter: prepend, slide 1 untouched
    const sep = "---\n# Slide 1\ntext\n---\n# Slide 2\n";
    expect(linkRunbook(sep, "r.json")).toBe("---\nrunbook: r.json\n---\n\n" + sep);
    // trailing blanks on the fences still count
    expect(linkRunbook("--- \ntitle: T\n---\t\n# T\n", "r.json")).toBe("--- \ntitle: T\nrunbook: r.json\n---\t\n# T\n");
    // a BOM stays first and the existing block is used
    expect(linkRunbook("\uFEFF---\ntitle: T\n---\n# T\n", "r.json")).toBe("\uFEFF---\ntitle: T\nrunbook: r.json\n---\n# T\n");
    // an oddly spaced runbook line is replaced, not duplicated; comments and blanks are fine
    expect(linkRunbook("---\n# note\n\n  runbook :  old.json\n---\n# T\n", "r.json")).toBe("---\n# note\n\nrunbook: r.json\n---\n# T\n");
    // unclosed front matter is not front matter
    expect(linkRunbook("---\ntitle: T\n# T\n", "r.json")).toBe("---\nrunbook: r.json\n---\n\n---\ntitle: T\n# T\n");
  });
  it("linkRunbook keeps mixed line endings: only the runbook line changes", () => {
    const deck = "---\ntitle: T\n---\r\n# T\r\n\r\nbody\n";
    expect(linkRunbook(deck, "r.json")).toBe("---\ntitle: T\nrunbook: r.json\n---\r\n# T\r\n\r\nbody\n");
    const crlf = "---\r\ntitle: T\r\nrunbook: a.json\r\n---\n# T\n";
    expect(linkRunbook(crlf, "r.json")).toBe("---\r\ntitle: T\r\nrunbook: r.json\r\n---\n# T\n");
  });
  it("metaValue quotes only when Go would misread the bare value", () => {
    expect(metaValue("labs/x.json")).toBe("labs/x.json");
    expect(metaValue("../a b/x #1.json")).toBe('"../a b/x #1.json"');
  });
});
