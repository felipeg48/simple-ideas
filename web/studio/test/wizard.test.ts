import { describe, expect, it } from "vitest";
import { linkRunbook, relPath, slugify } from "../src/runbook";

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
});
