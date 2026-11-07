// Accessibility check with axe-core (dev dependency, injected by this harness
// only, never into a built page): built slides, an article, a guide page, a 2D
// and a 3D mind map, a code window article, the teleprompter, and the Studio
// (Write, Map and Window views); the article and a slide again in every
// built-in theme, light and dark. Fails on any serious or critical violation.
// Run: make build && npm run e2e:a11y   (A11Y_REPORT=1 prints every violation)
import { chromium } from "playwright";
import { execFileSync, spawn } from "node:child_process";
import { cpSync, mkdtempSync, writeFileSync } from "node:fs";
import { createRequire } from "node:module";
import { tmpdir } from "node:os";
import { join } from "node:path";

const require = createRequire(import.meta.url);
const axePath = require.resolve("axe-core/axe.min.js");
const si = new URL("../../../bin/si", import.meta.url).pathname;
const ex = new URL("../../../examples/", import.meta.url).pathname;
const out = mkdtempSync(join(tmpdir(), "si-a11y-"));
const build = (src, dir) => execFileSync(si, ["build", src, "-o", join(out, dir)], { stdio: "ignore" });
build(join(ex, "showcase.md"), "deck");
build(join(ex, "article"), "article");
build(join(ex, "guide"), "guide");
build(join(ex, "mindmap"), "mindmap");
build(join(ex, "code"), "code");

// one small page per built-in theme: an article and a deck with code, a callout, a badge, a table
const themes = ["studio", "paper", "mono", "crdb", "acemq", "acemq-pro"];
const sample = (type, theme) => `---\ntype: ${type}\ntitle: Contrast in ${theme}\ntheme: ${theme}\n---\n# Contrast in ${theme}\n\n` +
  (type === "deck" ? "---\n\n## A slide\n\n" : "") +
  "Text with a [link](https://example.com), `inline code` and ==a highlight==.\n\n<!-- badge: Draft -->\n\n" +
  "```go title=\"retry.go\"\nfunc send() error { return nil } // done\n```\n\n| Step | Time |\n|---|---|\n| Retry | 0.2 s |\n\n> [!TIP]\n> A tip.\n";
const tdir = mkdtempSync(join(tmpdir(), "si-a11y-themes-"));
for (const t of themes) for (const type of ["article", "deck"]) writeFileSync(join(tdir, `${type}-${t}.md`), sample(type, t));
build(tdir, "themes");

const browser = await chromium.launch({ args: ["--use-gl=angle", "--use-angle=swiftshader", "--enable-unsafe-swiftshader"] });
const RULES = { runOnly: { type: "tag", values: ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"] } };
let failures = 0;
const seen = {};

async function audit(name, page, opts = {}) {
  await page.addScriptTag({ path: axePath });
  const res = await page.evaluate(async ([rules, exclude, process]) => {
    const ctx = exclude.length ? { exclude: exclude.map((s) => [s]) } : document;
    const r = await window.axe.run(ctx, rules);
    return r.violations.map((v) => ({ id: v.id, impact: v.impact, n: v.nodes.length, help: v.help,
      targets: v.nodes.slice(0, process.env.A11Y_REPORT ? 40 : 3).map((x) => x.target.join(" ") + (v.id === "color-contrast" && x.any[0] ? "  [" + [x.any[0].data.fgColor, x.any[0].data.bgColor, x.any[0].data.contrastRatio, x.any[0].data.expectedContrastRatio].join(" ") + "]" : "")) }));
  }, [RULES, opts.exclude || [], { env: { A11Y_REPORT: process.env.A11Y_REPORT || "" } }]);
  const bad = res.filter((v) => v.impact === "serious" || v.impact === "critical");
  for (const v of bad) seen[v.id] = (seen[v.id] || 0) + v.n;
  if (bad.length) {
    failures++;
    console.log(`FAIL ${name}`);
    for (const v of bad) console.log(`     ${v.impact} ${v.id} ×${v.n}: ${v.help}\n       ${v.targets.join("\n       ")}`);
  } else console.log(`ok   ${name}${process.env.A11Y_REPORT && res.length ? " (minor: " + res.map((v) => v.id).join(", ") + ")" : ""}`);
}

async function open(file, { dark = false, size = { width: 1440, height: 900 }, hash = "" } = {}) {
  const ctx = await browser.newContext({ viewport: size, colorScheme: dark ? "dark" : "light" });
  const p = await ctx.newPage();
  await p.goto("file://" + join(out, file) + hash);
  await p.waitForTimeout(700);
  return { ctx, p };
}

const pages = [
  ["deck: title slide", "deck/showcase.html", "#1"],
  ["deck: content slide", "deck/showcase.html", "#4"],
  ["article", "article/retry.html"],
  ["guide page", "guide/install.html"],
  ["mind map 2D", "mindmap/warehouse.html"],
  ["mind map 3D", "mindmap/thursday.html"],
  ["code window article", "code/retry.html"],
];
for (const dark of [false, true]) {
  for (const [name, file, hash] of pages) {
    const { ctx, p } = await open(file, { dark, hash });
    // the 3D canvas is a drawing: axe checks its controls, cards and fallback, not pixels
    await audit(`${name}${dark ? " (dark)" : ""}`, p);
    await ctx.close();
  }
}

// the teleprompter (the deck's p window)
{
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  const p = await ctx.newPage();
  await p.goto("file://" + join(out, "deck/showcase.html#4"));
  await p.waitForTimeout(500);
  const [tp] = await Promise.all([ctx.waitForEvent("page"), p.keyboard.press("p")]);
  await tp.waitForTimeout(900);
  await audit("teleprompter", tp);
  await ctx.close();
}

// every built-in theme, light and dark
for (const t of themes) for (const type of ["article", "deck"]) for (const dark of [false, true]) {
  const { ctx, p } = await open(`themes/${type}-${t}.html`, { dark, hash: type === "deck" ? "#2" : "" });
  await audit(`theme ${t} ${type}${dark ? " (dark)" : ""}`, p);
  await ctx.close();
}

// the Studio
{
  const root = mkdtempSync(join(tmpdir(), "si-a11y-studio-"));
  cpSync(new URL("./fixture", import.meta.url).pathname, root, { recursive: true });
  const srv = spawn(si, ["--web", root, "--addr", "127.0.0.1:0", "--no-open"], { stdio: ["ignore", "pipe", "pipe"] });
  const url = await new Promise((res, rej) => {
    let buf = "";
    const t = setTimeout(() => rej(new Error("no URL from si --web")), 10000);
    const on = (d) => { buf += d; const m = buf.match(/at (http:\/\/\S+)/); if (m) { clearTimeout(t); res(m[1]); } };
    srv.stdout.on("data", on); srv.stderr.on("data", on);
  });
  for (const dark of [false, true]) {
    const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } });
    await ctx.addInitScript((t) => { try { localStorage.setItem("si-studio-theme", t); } catch {} }, dark ? "dark" : "light");
    const p = await ctx.newPage();
    await p.goto(url);
    await p.waitForSelector(".app");
    const openFile = async (f) => { await p.waitForSelector(`.row-main[title="${f}"]`); await p.click(`.row-main[title="${f}"]`); await p.waitForTimeout(900); };
    const mode = dark ? " (dark)" : "";
    await openFile("talk.md");
    await audit("studio: write" + mode, p, { exclude: ["iframe"] });
    await openFile("thursday.md");
    await audit("studio: map" + mode, p, { exclude: ["iframe"] });
    await openFile("code.md");
    await p.click(".cm-line:has-text('func')").catch(() => {});
    await p.waitForTimeout(500);
    const win = await p.$("button:has-text('Window')");
    if (win) { await win.click(); await p.waitForTimeout(700); }
    await audit("studio: window" + mode, p, { exclude: ["iframe"] });
    await ctx.close();
  }
  srv.kill();
}

await browser.close();
console.log("\nserious/critical by rule:", JSON.stringify(seen));
if (failures) { console.log(`${failures} page(s) with serious or critical violations`); process.exit(1); }
console.log("all passed");
