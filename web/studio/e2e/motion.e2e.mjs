// Headless end-to-end for the Studio's Motion view. Run: make build && node e2e/motion.e2e.mjs
// An empty anim.svg → Start a blank board → a box → frame 2 → drag the box
// (real pointer) → frame 3 → recolour and hide → the SVG on disk holds the
// frames; the snippet put on a slide plays: each build step moves the box.
// Screenshots light and dark in e2e/out/.
import { chromium } from "playwright";
import { spawn, execFileSync } from "node:child_process";
import { cpSync, mkdtempSync, readFileSync, mkdirSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, dirname } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const si = join(here, "../../../bin/si");
const out = join(here, "out");
mkdirSync(out, { recursive: true });
const root = mkdtempSync(join(tmpdir(), "si-e2e-motion-"));
cpSync(join(here, "fixture"), root, { recursive: true });
writeFileSync(join(root, "anim.svg"), "");

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
let failed = 0;
const check = (ok, what) => { console.log((ok ? "ok   " : "FAIL ") + what); if (!ok) failed++; };
const FILE = join(root, "anim.svg");
const disk = () => { try { return readFileSync(FILE, "utf8"); } catch { return ""; } };

try {
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  await ctx.addInitScript(() => { try { if (!localStorage.getItem("si-studio-theme")) localStorage.setItem("si-studio-theme", "light"); } catch {} });
  const page = await ctx.newPage();
  const errors = [];
  page.on("pageerror", (e) => errors.push(e.message));
  await page.goto(url);
  await page.waitForSelector(".app");
  await page.click('.row-main[title="anim.svg"]');
  await page.click('button:text-is("Start a blank board")');
  await until(() => disk().includes("/* si motion: frame1 */"), 6000, "blank board saved");
  await page.click('button:text-is("Add a box")');
  await until(() => disk().includes('data-part="rect1"'), 6000, "box saved");
  await page.click('button:text-is("Add a frame")');
  await until(() => disk().includes("/* si motion: frame1 frame2 */"), 6000, "frame 2 saved");
  check(await page.$eval('.mox-strip .btn[aria-pressed="true"]', (b) => b.textContent) === "2", "the new frame is the one being edited");

  // drag the box with a real pointer, in frame 2
  const box = { boundingBox: async () => (await page.evaluateHandle(() => document.querySelector(".mox-svg").shadowRoot.querySelector('[data-part="rect1"]'))).boundingBox() };
  const r = await box.boundingBox();
  await page.mouse.move(r.x + r.width / 2, r.y + r.height / 2);
  await page.mouse.down();
  for (let i = 1; i <= 10; i++) await page.mouse.move(r.x + r.width / 2 + i * 20, r.y + r.height / 2 + i * 5);
  await page.mouse.up();
  await until(() => /\[data-state~="frame2"\] \[data-part="rect1"\]\{transform:translate\(\d/.test(disk()), 6000, "the drag saved: " + disk());
  const moved = await until(async () => { const b = await box.boundingBox(); return b && b.x > r.x + 100 ? b : null; }, 4000, "the box moved on the board");
  check(moved.x > r.x + 100, `the box sits where it was dropped (${Math.round(r.x)} -> ${Math.round(moved.x)})`);
  await page.screenshot({ path: join(out, "motion-frame2.png") });

  // frame 3: recolour, size, timing
  await page.click('button:text-is("Add a frame")');
  await until(() => disk().includes("frame1 frame2 frame3"), 6000, "frame 3 saved");
  await page.selectOption("#mo-fill", "var(--si-ok)");
  await until(() => /\[data-state~="frame3"\] \[data-part="rect1"\]\{[^}]*fill:var\(--si-ok\)/.test(disk()), 6000, "colour saved");
  await page.fill("#mo-scale", "1.5");
  await page.press("#mo-scale", "Enter");
  await until(() => /frame3"\] \[data-part="rect1"\]\{transform:[^}]*scale\(1.5\)/.test(disk()), 6000, "size saved");
  await page.fill("#mo-dur", "1.2");
  await page.press("#mo-dur", "Enter");
  await page.selectOption("#mo-ease", "spring");
  await until(() => disk().includes('[data-state~="frame3"] *{transition-duration:1.2s;transition-timing-function:cubic-bezier(.3,1.5,.5,1)}'), 6000, "timing saved");
  // back to frame 1: the box goes home on the board
  await page.click('.mox-strip .btn:text-is("1")');
  await until(async () => { const b = await box.boundingBox(); return Math.abs(b.x - r.x) < 3; }, 4000, "frame 1 shows the box as drawn");
  check(true, "frame 1 shows the box where it was drawn");

  // Undo in the editor undoes the last op
  const before = disk();
  await page.keyboard.press("ControlOrMeta+Z");
  await until(() => disk() !== before, 6000, "undo saved");
  check(!disk().includes("1.2s;transition-timing-function:cubic-bezier(.3,1.5,.5,1)"), "undo takes the last change back");
  await page.keyboard.press("ControlOrMeta+Shift+Z");
  await until(() => disk() === before, 6000, "redo saved");

  const snippet = await page.$eval(".mox-snippet", (e) => e.textContent);
  check(snippet === "<!-- svg: anim.svg frame1 -->\n<!-- step: +frame2 -->\n<!-- step: -frame2 +frame3 -->\n", "the snippet: " + JSON.stringify(snippet));
  check(errors.length === 0, "no page errors " + errors.join("; "));

  // dark
  await page.evaluate(() => { localStorage.setItem("si-studio-theme", "dark"); });
  await page.reload();
  await page.waitForSelector(".mox-svg");
  await sleep(600);
  await page.screenshot({ path: join(out, "motion-dark.png") });
  await ctx.close();

  // the snippet on a slide: each click moves the box
  writeFileSync(join(root, "anim-talk.md"), "# Anim\n\n---\n\n## The box travels\n\n" + snippet + "\nNotes: (0.5 min)\nThe box starts here. [CLICK] It moves right. [CLICK] It grows and turns green.\n");
  execFileSync(si, ["check", join(root, "anim-talk.md")]);
  execFileSync(si, ["build", join(root, "anim-talk.md"), "-o", join(root, "built")], { stdio: "ignore" });
  const dp = await (await browser.newContext({ viewport: { width: 1440, height: 900 } })).newPage();
  await dp.goto(pathToFileURL(join(root, "built", "anim-talk.html")).href + "#2");
  await sleep(500);
  const t = () => dp.evaluate(() => { const e = document.querySelector('#s2 [data-part="rect1"]'); return { tf: getComputedStyle(e).transform, fill: getComputedStyle(e).fill }; });
  const t0 = await t();
  await dp.keyboard.press("ArrowRight");
  await sleep(1800);
  const t1 = await t();
  await dp.keyboard.press("ArrowRight");
  await sleep(2000);
  const t2 = await t();
  check(t0.tf === "none" && t1.tf !== "none" && t2.tf !== t1.tf, `steps move the box: ${t0.tf} -> ${t1.tf} -> ${t2.tf}`);
  check(t2.fill !== t1.fill, `frame 3 recolours it: ${t1.fill} -> ${t2.fill}`);
} catch (e) {
  console.log("FAIL " + e.message);
  failed++;
} finally {
  await browser.close();
}
console.log(failed ? `${failed} failed` : "motion e2e ok");
process.exit(failed ? 1 : 0);
