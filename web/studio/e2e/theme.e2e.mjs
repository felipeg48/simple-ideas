// Headless end-to-end for the Studio's Theme view. Run: make build && node e2e/theme.e2e.mjs
// New theme from Paper → change the accent → the preview follows → a failing
// contrast pair → Fix → Use in this folder; screenshots light and dark at
// desktop and phone width in e2e/out/.
import { chromium } from "playwright";
import { spawn } from "node:child_process";
import { cpSync, mkdtempSync, readFileSync, mkdirSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const si = join(here, "../../../bin/si");
const out = join(here, "out");
mkdirSync(out, { recursive: true });
const root = mkdtempSync(join(tmpdir(), "si-e2e-theme-"));
cpSync(join(here, "fixture"), root, { recursive: true });

const srv = spawn(si, ["--web", root, "--addr", "127.0.0.1:0", "--no-open"], { stdio: ["ignore", "pipe", "pipe"] });
process.on("exit", () => { srv.kill(); rmSync(root, { recursive: true, force: true }); });
const url = await new Promise((res, rej) => {
  let buf = "";
  const t = setTimeout(() => rej(new Error("no URL from si --web: " + buf)), 10000);
  const on = (d) => { buf += d; const m = buf.match(/at (http:\/\/\S+)/); if (m) { clearTimeout(t); res(m[1]); } };
  srv.stdout.on("data", on); srv.stderr.on("data", on);
});
const browser = await chromium.launch({ headless: true });
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const until = async (fn, ms = 8000, what = "condition") => {
  const end = Date.now() + ms;
  for (;;) {
    try { const v = await fn(); if (v) return v; } catch { /* retry */ }
    if (Date.now() > end) throw new Error("timeout: " + what);
    await sleep(100);
  }
};
const FILE = join(root, "themes/acme.css");
const disk = () => { try { return readFileSync(FILE, "utf8"); } catch { return ""; } };

async function open(theme, size) {
  const ctx = await browser.newContext({ viewport: size });
  await ctx.addInitScript((t) => { try { localStorage.setItem("si-studio-theme", t); } catch {} }, theme);
  const page = await ctx.newPage();
  await page.goto(url);
  await page.waitForSelector(".app");
  return { ctx, page };
}

// make the theme
{
  const { ctx, page } = await open("dark", { width: 1440, height: 900 });
  await page.evaluate(() => window.siStudio.newTheme());
  await page.waitForSelector('[aria-label="New theme"] .thx-start');
  await page.click('.thx-start:has(b:text-is("Paper"))');
  await page.fill('[aria-label="New theme"] input.field', "Acme");
  await page.click('[aria-label="New theme"] button[type="submit"]');
  await page.waitForSelector(".thx-box iframe");
  await until(() => disk().startsWith("/* si theme: Acme · extends paper"), 5000, "theme file: " + JSON.stringify(disk().slice(0, 80)));
  await page.screenshot({ path: join(out, "theme-new.png") });

  // the accent, light: typed into its field
  await page.fill("#thx-accent", "#0b3d91");
  await page.press("#thx-accent", "Enter");
  await until(() => disk().includes("--si-accent: #0b3d91;"), 6000, "accent saved");
  const frame = await (await page.$(".thx-box iframe")).contentFrame();
  await until(() => frame.evaluate(() => getComputedStyle(document.documentElement).getPropertyValue("--si-accent").trim() === "#0b3d91"), 6000, "preview accent");

  // a muted text too light to read → a check → Fix
  await page.fill("#thx-muted", "#c9c9c9");
  await page.press("#thx-muted", "Enter");
  const fix = await until(() => page.$('.thx-issues li:has-text("muted") button'), 6000, "contrast issue");
  await page.screenshot({ path: join(out, "theme-check.png") });
  await fix.click();
  await until(() => !disk().includes("#c9c9c9"), 6000, "fixed");

  // the heading size slider and a font
  await page.fill("#thx-font-heading", '"Fraunces", Georgia, serif');
  await page.press("#thx-font-heading", "Enter");
  await until(() => disk().includes('--si-font-heading: "Fraunces", Georgia, serif;'), 6000, "font saved");

  // other slides and sizes
  for (let i = 0; i < 4; i++) await page.click('[aria-label="Next slide"]');
  await page.click('[aria-label="Screen size"] button:has-text("Tablet")');
  await until(async () => (await (await page.$(".thx-box iframe")).contentFrame()).evaluate(() => [...document.querySelectorAll(".si-slide")].findIndex((s) => s.classList.contains("si-current")) === 4), 5000, "slide 5");
  await sleep(800);
  await page.screenshot({ path: join(out, "theme-tablet.png") });
  await page.click('[aria-label="Screen size"] button:has-text("Desktop")');

  await page.click('button:has-text("Use in this folder")');
  await until(() => readFileSync(join(root, "si.conf"), "utf8").includes("theme: themes/acme.css"), 5000, "si.conf");
  await page.screenshot({ path: join(out, "theme-1440-dark.png") });
  await ctx.close();
}

// look at it: light and dark, desktop and phone
for (const [theme, size, name] of [["light", { width: 1440, height: 900 }, "theme-1440-light"], ["dark", { width: 400, height: 800 }, "theme-400-dark"], ["light", { width: 400, height: 800 }, "theme-400-light"]]) {
  const { ctx, page } = await open(theme, size);
  if (size.width < 600) await page.click(".nav-toggle");
  const row = '.row-main[title="themes"]';
  await page.waitForSelector(row);
  if (!(await page.$('.row-main[title="themes/acme.css"]'))) await page.click(row);
  await page.click('.row-main[title="themes/acme.css"]');
  await page.waitForSelector(".thx-box iframe");
  await sleep(1200);
  await page.screenshot({ path: join(out, name + ".png") });
  await ctx.close();
}
console.log("theme e2e: ok");
await browser.close();
process.exit(0);
