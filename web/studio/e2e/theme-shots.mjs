// The pictures of the built-in themes (docs/site/themes/<name>.jpg) for the
// docs site's Themes page and the README: the showcase's "Payment: declined,
// then paid" slide in each theme, light beside dark, with the theme's name.
// Run with `make theme-shots` after adding or changing a theme.
// (The showcase's own accent: and watermark: are left out, so each theme shows
// its own colours.)
import { chromium } from "playwright";
import { execFileSync } from "node:child_process";
import { cpSync, mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

const repo = new URL("../../../", import.meta.url).pathname;
const si = join(repo, "bin/si");
const outDir = join(repo, "docs/site/themes");
const tmp = mkdtempSync(join(tmpdir(), "si-theme-shots-"));
const src = join(tmp, "src");
mkdirSync(src);
for (const f of ["assets", "appendix.md", "css", "runbook.json", "demos"]) cpSync(join(repo, "examples", f), join(src, f), { recursive: true });
writeFileSync(join(src, "showcase.md"), readFileSync(join(repo, "examples/showcase.md"), "utf8").split("\n").filter((l) => !/^(accent|watermark[\w-]*):/.test(l)).join("\n"));

const list = execFileSync(si, ["theme", "list"], { encoding: "utf8" });
const themes = [...list.matchAll(/^\s*([a-z0-9-]+)\s{2,}(.+?)\s{2,}/gm)].map((m) => ({ name: m[1], label: m[2].trim() }));
const unlisted = new Set(["crdb", "acemq", "acemq-pro"]); // as internal/docsite/themes.go
for (let i = themes.length - 1; i >= 0; i--) if (unlisted.has(themes[i].name)) themes.splice(i, 1);
if (!themes.length) throw new Error("si theme list gave no themes:\n" + list);

const browser = await chromium.launch();
mkdirSync(outDir, { recursive: true });
for (const t of themes) {
  const site = join(tmp, t.name);
  execFileSync(si, ["build", join(src, "showcase.md"), "-o", site, "--theme", t.name], { stdio: "ignore" });
  const shots = {};
  for (const cs of ["light", "dark"]) {
    const p = await browser.newPage({ viewport: { width: 1280, height: 720 }, colorScheme: cs });
    await p.goto("file://" + join(site, "showcase.html") + "#5.2");
    await p.waitForTimeout(1500);
    shots[cs] = (await p.screenshot()).toString("base64");
    await p.close();
  }
  const p = await browser.newPage({ viewport: { width: 1280, height: 400 } });
  await p.setContent(`<body style="margin:0;font:600 18px/1 ui-sans-serif,-apple-system,sans-serif;background:#fff;color:#16171b">
    <div style="display:flex;align-items:baseline;gap:12px;padding:14px 16px 10px"><span>${t.label}</span><code style="font:500 15px ui-monospace,monospace;color:#5d626c">theme: ${t.name}</code></div>
    <div style="display:flex;gap:8px;padding:0 8px 8px"><img src="data:image/png;base64,${shots.light}" style="width:50%;border-radius:6px;border:1px solid #ddd"><img src="data:image/png;base64,${shots.dark}" style="width:50%;border-radius:6px;border:1px solid #333"></div></body>`);
  await p.setViewportSize({ width: 1280, height: await p.evaluate(() => document.body.scrollHeight) });
  await p.screenshot({ path: join(outDir, t.name + ".jpg"), type: "jpeg", quality: 82 });
  await p.close();
  console.log("theme " + t.name);
}
await browser.close();
rmSync(tmp, { recursive: true, force: true });
