// transition: morph: a built deck (file://), two morph slides in a row. The
// named block and the shared heading start the animation in their old box and
// end in the new one, at four sizes; unmatched words fade; reduced motion and
// a jump (not a neighbour) give no glide. Headless; needs ../../bin/si
// (make build). Run: npm run e2e:morph
import { chromium } from "playwright";
import { execFileSync } from "node:child_process";
import { mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { pathToFileURL } from "node:url";

const si = new URL("../../../bin/si", import.meta.url).pathname;
const dir = mkdtempSync(join(tmpdir(), "si-morph-"));
writeFileSync(join(dir, "talk.md"), `# Five teams, one promise

---

<!-- transition: morph -->
## Five teams

- shop
- payments
- warehouse
- courier

<!-- id: promise -->
Two days, door to door.

---

<!-- transition: morph 0.9s -->
<!-- layout: center -->
## Five teams

<!-- id: promise -->
**Two days, door to door.**

---

## Elsewhere
`);
execFileSync(si, ["build", join(dir, "talk.md"), "-o", join(dir, "out")], { stdio: "ignore" });
const url = pathToFileURL(join(dir, "out", "talk.html")).href;

const browser = await chromium.launch();
let failed = 0;
const check = (ok, what) => { console.log((ok ? "ok   " : "FAIL ") + what); if (!ok) failed++; };
const near = (a, b) => Math.abs(a.x - b.x) < 2 && Math.abs(a.y - b.y) < 2 && Math.abs(a.w - b.w) < 2 && Math.abs(a.h - b.h) < 2;
const box = (page, slide, sel) => page.evaluate(([s, q]) => {
  const g = document.createRange(); g.selectNodeContents(document.querySelector("#s" + s + " " + q)); const r = g.getBoundingClientRect();
  return { x: r.left, y: r.top, w: r.width, h: r.height };
}, [slide, sel]);
// pause every animation at time t (ms; -1 = its end) and measure
const at = (page, sel, t) => page.evaluate(([q, t]) => {
  const anims = document.getAnimations();
  anims.forEach((a) => { a.pause(); a.currentTime = t < 0 ? a.effect.getComputedTiming().endTime - 1 : t; });
  const g = document.createRange(); g.selectNodeContents(document.querySelector("#s3 " + q)); const r = g.getBoundingClientRect();
  return { n: anims.length, x: r.left, y: r.top, w: r.width, h: r.height };
}, [sel, t]);
try {
  for (const [w, h] of [[1440, 900], [400, 800], [820, 1180], [3840, 2160]]) {
    const ctx = await browser.newContext({ viewport: { width: w, height: h } });
    const page = await ctx.newPage();
    const errors = [];
    page.on("pageerror", (e) => errors.push(e.message));
    await page.goto(url + "#2");
    await page.waitForTimeout(800);
    const from = await box(page, 2, "[data-si-morph=promise]");
    const fromH = await box(page, 2, ".si-heading");
    await page.keyboard.press("ArrowRight");
    const start = await at(page, "[data-si-morph=promise]", 0);
    const startH = await at(page, ".si-heading", 0);
    const mid = await at(page, "[data-si-morph=promise]", 450);
    const end = await at(page, "[data-si-morph=promise]", -1);
    const tx = await page.evaluate(() => document.querySelector(".si-deck").getAttribute("data-si-tx"));
    await page.evaluate(() => document.getAnimations().forEach((a) => a.finish()));
    const to = await box(page, 3, "[data-si-morph=promise]");
    check(tx === "morph" && start.n > 0, `${w}x${h}: morph runs (${start.n} animations)`);
    check(!near(from, to), `${w}x${h}: the named block moves (${Math.round(from.x)},${Math.round(from.y)} -> ${Math.round(to.x)},${Math.round(to.y)})`);
    check(Math.abs(start.x - from.x) < 2 && Math.abs(start.y - from.y) < 2 && Math.abs(start.h - from.h) < 2, `${w}x${h}: it starts where it was, at its old size (bold words may be wider)`);
    check(!near(mid, from) && !near(mid, to), `${w}x${h}: mid-way it is between`);
    check(near(end, to), `${w}x${h}: it ends in its new box`);
    check(near(startH, fromH), `${w}x${h}: the shared heading starts where it was`);
    check(errors.length === 0, `${w}x${h}: no page errors ${errors.join("; ")}`);
    // a jump is not a morph
    await page.goto(url + "#1");
    await page.waitForTimeout(300);
    await page.evaluate(() => { location.hash = "#3"; });
    await page.waitForTimeout(100);
    check(await page.evaluate(() => document.querySelector(".si-deck").getAttribute("data-si-tx")) !== "morph", `${w}x${h}: a jump fades`);
    await ctx.close();
  }
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 }, reducedMotion: "reduce" });
  const page = await ctx.newPage();
  await page.goto(url + "#2");
  await page.waitForTimeout(500);
  await page.keyboard.press("ArrowRight");
  const n = await page.evaluate(() => document.querySelector("#s3 [data-si-morph=promise]").getAnimations().length);
  check(n === 0, "reduced motion: no glide");
  // back from a morph slide morphs too
  const back = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  const bp = await back.newPage();
  await bp.goto(url + "#3");
  await bp.waitForTimeout(500);
  await bp.keyboard.press("ArrowLeft");
  check(await bp.evaluate(() => document.querySelector("#s2 [data-si-morph=promise]").getAnimations().length) > 0, "going back from a morph slide glides too");
} finally {
  await browser.close();
  rmSync(dir, { recursive: true, force: true });
}
console.log(failed ? `${failed} failed` : "morph e2e ok");
process.exit(failed ? 1 : 0);
