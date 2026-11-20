// ```scene fences: a built deck (file://) whose steps fly a parcel, fill a
// queue that a consumer takes from, and open a block; then a page. Each step
// plays and settles where the Go side says it ends (the figure's JSON), a
// #N.k link and reduced motion land on a step at once, and a page plays once
// it is scrolled into view, with ← → Play. No page errors. Headless; needs
// ../../bin/si (make build). Run: npm run e2e:scene
import { chromium } from "playwright";
import { execFileSync } from "node:child_process";
import { mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { pathToFileURL } from "node:url";

const si = new URL("../../../bin/si", import.meta.url).pathname;
const dir = mkdtempSync(join(tmpdir(), "si-scene-"));
const scene = `\`\`\`scene title="Tuesday"
box Shop
box Customer
parcel flies Shop -> Customer · arc
queue orders · Billing takes 2
Customer opens → Unpack, Keep
\`\`\``;
writeFileSync(join(dir, "talk.md"), `# An order's week\n\n${scene}\n`);
writeFileSync(join(dir, "page.md"), `---\ntype: article\n---\n\n# An order's week\n\n${"Words before the scene.\n\n".repeat(30)}\n${scene}\n`);
execFileSync(si, ["build", join(dir, "talk.md"), "-o", join(dir, "out")], { stdio: "ignore" });
execFileSync(si, ["build", join(dir, "page.md"), "-o", join(dir, "out")], { stdio: "ignore" });
const url = pathToFileURL(join(dir, "out", "talk.html")).href;
const pageURL = pathToFileURL(join(dir, "out", "page.html")).href;

const browser = await chromium.launch();
let failed = 0;
const check = (ok, what) => { console.log((ok ? "ok   " : "FAIL ") + what); if (!ok) failed++; };
// how far each object is from where step k says it is (0: all in place), and its step
const off = (page, k) => page.evaluate((k) => {
  const fig = document.querySelector(".si-scene");
  const S = JSON.parse(fig.getAttribute("data-scene")).s[k];
  let worst = 0;
  fig.querySelectorAll("[data-o]").forEach((g) => {
    const s = S[+g.dataset.o], op = +(g.getAttribute("opacity") || 1);
    if ((op > 0.5) !== !!s.v) worst = Math.max(worst, 999);
    const m = /translate\(([-\d.e]+) ([-\d.e]+)\)/.exec(g.getAttribute("transform") || "");
    if (m && s.v) worst = Math.max(worst, Math.abs(+m[1] - s.x), Math.abs(+m[2] - s.y));
  });
  return { worst, k: fig.siScene.k };
}, k);

try {
  for (const [w, h] of [[1440, 900], [400, 800]]) {
    const ctx = await browser.newContext({ viewport: { width: w, height: h } });
    const page = await ctx.newPage();
    const errors = [];
    page.on("pageerror", (e) => errors.push(e.message));
    page.on("console", (m) => { if (m.type() === "error") errors.push(m.text()); });
    await page.goto(url + "#1");
    await page.waitForTimeout(500);
    let o = await off(page, 0);
    check(o.k === 0 && o.worst < 0.01, `${w}x${h}: the slide opens on the scene before its first step`);
    await page.keyboard.press("ArrowRight");
    await page.waitForTimeout(500);
    o = await off(page, 1);
    check(o.k === 1 && o.worst > 1, `${w}x${h}: a step plays: the parcel is on its way (${o.worst.toFixed(1)} from the end)`);
    await page.waitForTimeout(2300);
    o = await off(page, 1);
    check(o.worst < 0.01, `${w}x${h}: and lands on Customer`);
    await page.keyboard.press("ArrowRight");
    await page.waitForTimeout(4500);
    o = await off(page, 2);
    const left = await page.evaluate(() => [...document.querySelectorAll(".si-scene-message")].filter((g) => +g.getAttribute("opacity") > 0.5).length);
    check(o.k === 2 && o.worst < 0.01 && left === 3, `${w}x${h}: five orders join the queue, Billing takes two (${left} left)`);
    await page.keyboard.press("ArrowRight");
    await page.waitForTimeout(2500);
    o = await off(page, 3);
    const kids = await page.evaluate(() => [...document.querySelectorAll(".si-scene-box.is-on text")].map((t) => t.textContent));
    check(o.worst < 0.01 && kids.join() === "Unpack,Keep", `${w}x${h}: Customer opens to show its parts (${kids})`);
    await page.keyboard.press("ArrowLeft");
    await page.waitForTimeout(200);
    o = await off(page, 2);
    check(o.k === 2 && o.worst < 0.01, `${w}x${h}: a step back sets the step at once`);
    check(errors.length === 0, `${w}x${h}: no page errors ${errors.join("; ")}`);
    await ctx.close();
  }

  const rm = await browser.newContext({ viewport: { width: 1440, height: 900 }, reducedMotion: "reduce" });
  const rp = await rm.newPage();
  await rp.goto(url + "#1");
  await rp.waitForTimeout(400);
  await rp.keyboard.press("ArrowRight");
  await rp.waitForTimeout(80);
  let o = await off(rp, 1);
  check(o.k === 1 && o.worst < 0.01, "reduced motion: the step's end at once");
  await rp.goto(url + "#1.3");
  await rp.waitForTimeout(400);
  o = await off(rp, 3);
  check(o.k === 3 && o.worst < 0.01, "a #1.3 link shows the third step's end");
  await rp.emulateMedia({ media: "print" });
  await rp.evaluate(() => window.dispatchEvent(new Event("beforeprint")));
  check((await off(rp, 3)).k === 3, "print shows the final step");
  await rm.close();

  const pc = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  const pp = await pc.newPage();
  const perr = [];
  pp.on("pageerror", (e) => perr.push(e.message));
  await pp.goto(pageURL);
  await pp.waitForTimeout(300);
  const first = (await off(pp, 0)).k;
  await pp.evaluate(() => document.querySelector(".si-scene").scrollIntoView());
  await pp.waitForTimeout(12000);
  o = await off(pp, 3);
  const nav = await pp.evaluate(() => !!document.querySelector(".si-scene-nav button[data-a=play]"));
  check(first === 0 && o.k === 3 && o.worst < 0.01 && nav, `a page waits on step 0 (${first}), plays to the end once in view (${o.k}), with ← → Play`);
  await pp.click(".si-scene-nav [data-a=prev]");
  check((await off(pp, 2)).k === 2, "← goes back a step");
  check(perr.length === 0, `a page: no errors ${perr.join("; ")}`);
  await pc.close();
} finally {
  await browser.close();
  rmSync(dir, { recursive: true, force: true });
}
if (failed) { console.log(`${failed} failed`); process.exit(1); }
