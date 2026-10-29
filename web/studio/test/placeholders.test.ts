import { describe, expect, it } from "vitest";
import { insertPlaceholder, placeholders } from "../src/runbook";

describe("placeholders (vectors shared with Go)", () => {
  it("parses distinct in order", () => {
    expect(placeholders("kubectl -n {{ns:messaging}} get pods {{pod}} -n {{ns:other}} {{ns}}")).toEqual([
      { name: "ns", def: "messaging" },
      { name: "pod", def: "" },
    ]);
  });
  it("ignores templates and shell", () => {
    for (const c of ["docker inspect -f '{{.Server.Version}}' x", "docker inspect {{json .}}", "echo ${HOME}", "awk '{print $1}'", "jq '{a: .b}'"])
      expect(placeholders(c)).toEqual([]);
  });
  it("insertPlaceholder puts cursor after token", () => {
    expect(insertPlaceholder("ls  -l", 3, "dir")).toEqual({ cmd: "ls {{dir}} -l", cursor: 10 });
    expect(insertPlaceholder("ls", 2, "d", "/tmp")).toEqual({ cmd: "ls{{d:/tmp}}", cursor: 12 });
  });
});
