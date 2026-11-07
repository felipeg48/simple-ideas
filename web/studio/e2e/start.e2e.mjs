// First run of the Studio on an empty folder: the start gallery shows live
// thumbnails, creating a 3D mind map opens it in Map view, the tour starts
// and can be skipped, and it stays skipped. Run: make build && npm run e2e
import { chromium } from "playwright";
import { spawn } from "node:child_process";
import { existsSync, mkdtempSync, readFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

const si = new URL("../../../bin/si", import.meta.url).pathname;
const root = mkdtempSync(join(tmpdir(), "si-e2e-start-"));
const srv = spawn(si, ["--web", root, "--addr", "127.0.0.1:0", "--no-open"], { stdio: ["ignore", "pipe", "pipe"] });
const url = await new Promise((res, rej) => {
  let buf = "";
  const t = setTimeout(() => rej(new Error("no URL from si --web: " + buf)), 10000);
  const on = (d) => { buf += d; const m = buf.match(/at (http:\/\/\S+)/); if (m) { clearTimeout(t); res(m[1]); } };
  srv.stdout.on("data", on); srv.stderr.on("data", on);
});
const browser = await chromium.launch({ headless: true, args: ["--use-gl=angle", "--use-angle=swiftshader", "--enable-unsafe-swiftshader"] });
let failed = 0;
const check = (ok, what) => { console.log((ok ? "ok   " : "FAIL ") + what); if (!ok) failed++; };
try {
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  const page = await ctx.newPage();
  const errors = [];
  page.on("pageerror", (e) => errors.push(e.message));
  await page.goto(url);
  await page.waitForSelector(".dialog.gallery", { timeout: 8000 });
  const cards = await page.$$eval(".gal", (b) => b.map((x) => x.textContent));
  check(cards.length === 8 && cards.some((c) => c.includes("A 3D mind map")) && cards.some((c) => c.includes("A code show")), `empty folder → start gallery with 8 starters (${cards.length})`);
  // the thumbnails are rendered pages
  const thumbs = await page.$$eval(".gal-thumb iframe", (fs) => fs.map((f) => f.getAttribute("src")));
  const deckFrame = page.frames().find((f) => f.url().includes("/gallery/deck"));
  await page.waitForTimeout(1500);
  const deckText = deckFrame ? await deckFrame.evaluate(() => document.body.textContent) : "";
  check(thumbs.length === 8 && deckText.includes("Your title here"), "every card has a live thumbnail (the deck's shows its title)");
  await page.screenshot({ path: new URL("./out/start-gallery.png", import.meta.url).pathname });
  await page.click(".gal:has-text('A 3D mind map')");
  await page.fill(".gal-form input", "Launch plan");
  await page.click(".gal-form button[type=submit]");
  await page.waitForSelector(".tour-note", { timeout: 8000 });
  check(existsSync(join(root, "launch-plan.md")) && readFileSync(join(root, "launch-plan.md"), "utf8").includes("view: 3d"), "creates launch-plan.md from the 3D mind map starter");
  check(await page.$("[aria-label='Mind map view'] button[aria-pressed='true']:has-text('Map')") !== null, "it opens in Map view");
  check((await page.textContent(".tour-note")).includes("1 of 6"), "the tour starts after the first file");
  await page.click(".tour-note button:has-text('Next')");
  check((await page.textContent(".tour-note")).includes("2 of 6"), "Next moves the tour on");
  await page.screenshot({ path: new URL("./out/start-tour.png", import.meta.url).pathname });
  await page.click(".tour-note button:has-text('Skip tour')");
  check(await page.$(".tour") === null, "Skip tour closes it");
  await page.reload();
  await page.waitForSelector(".app");
  await page.waitForTimeout(1200);
  check(await page.$(".tour") === null && await page.$(".dialog.gallery") === null, "after a reload: no gallery (the folder has a file) and no tour (skipped is remembered)");
  // ⌘K → the tour again, and Esc skips it
  await page.click(".brand"); // focus the Studio, not the map's frame
  await page.keyboard.press("ControlOrMeta+K");
  await page.waitForSelector(".palette input, [role=dialog] input", { timeout: 4000 });
  await page.fill(".palette-in", "take the tour"); // the map's frame may hold the keyboard focus
  await page.press(".palette-in", "Enter");
  await page.waitForSelector(".tour-note");
  await page.keyboard.press("Escape");
  check(await page.$(".tour") === null, "⌘K → Take the tour, Esc skips it");
  check(errors.length === 0, "no page errors" + (errors.length ? ": " + errors.join(" | ") : ""));
  await ctx.close();
} catch (e) {
  console.log("FAIL", e.message);
  failed++;
} finally {
  await browser.close();
  srv.kill();
}
if (failed) process.exit(1);
console.log("all passed");
