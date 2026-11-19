// ```chart fences: a built deck (file://) with a bar race, a show that turns
// bars into a donut, and the same id on the next slide; then a page. Steps
// move the bars to their new order (they glide, then settle), values count,
// another kind cross-fades in, the same id carries across slides, reduced
// motion and a page that plays when it is scrolled into view end on the
// final values. Headless; needs ../../bin/si (make build). Run: npm run e2e:chart
import { chromium } from "playwright";
import { execFileSync } from "node:child_process";
import { mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { pathToFileURL } from "node:url";

const si = new URL("../../../bin/si", import.meta.url).pathname;
const dir = mkdtempSync(join(tmpdir(), "si-chart-"));
const race = `\`\`\`chart bar race unit=%
| Cause     | Q1 | Q2 | Q3 |
|-----------|----|----|----|
| Carrier   | 34 | 22 | 12 |
| Warehouse | 18 | 25 | 9  |
| Payment   | 9  | 14 | 21 |
| Address   | 25 | 11 | 6  |
| Weather   | 6  | 8  | 15 |
\`\`\``;
const mix = (kind, n) => `\`\`\`chart ${kind} id=mix
| Region | Orders |
|--------|--------|
| North  | ${n[0]} |
| South  | ${n[1]} |
| East   | ${n[2]} |
\`\`\``;
writeFileSync(join(dir, "talk.md"), `# Late orders by cause

${race}

---

# Orders by region

${mix("bar", ["1,200", "860", "1,540"])}

${mix("donut", ["1,200", "860", "1,540"])}

---

# The next quarter

${mix("column", ["1,400", "900", "1,600"])}

---

# Solid bars

${race.replace("bar race", "bar3d race")}

---

# Orders this quarter

\`\`\`chart donut3d explode=Late
| Status  | Share |
|---------|-------|
| On time | 78    |
| Late    | 15    |
| Lost    | 7     |
\`\`\`

---

# Checkout

\`\`\`chart funnel3d unit=%
| Step     | Share |
|----------|-------|
| Visits   | 100   |
| Cart     | 42    |
| Paid     | 15    |
\`\`\`
`);
writeFileSync(join(dir, "page.md"), `---
type: article
---

# Late orders

${"Words before the chart.\n\n".repeat(30)}

${race}
`);
execFileSync(si, ["build", join(dir, "talk.md"), "-o", join(dir, "out")], { stdio: "ignore" });
execFileSync(si, ["build", join(dir, "page.md"), "-o", join(dir, "out")], { stdio: "ignore" });
const url = pathToFileURL(join(dir, "out", "talk.html")).href;
const pageURL = pathToFileURL(join(dir, "out", "page.html")).href;

const browser = await chromium.launch();
let failed = 0;
const check = (ok, what) => { console.log((ok ? "ok   " : "FAIL ") + what); if (!ok) failed++; };
const eq = (a, b) => JSON.stringify(a) === JSON.stringify(b);
// the frame on screen: its index, its rows top to bottom as drawn, their values, any glide still running
const state = (page, slide) => page.evaluate((s) => {
  const fig = document.querySelector("#s" + s + " .si-chart");
  const frames = [...fig.querySelectorAll(".si-chart-frame")];
  const on = frames.findIndex((f) => f.hasAttribute("data-on"));
  const rows = [...frames[on].querySelectorAll(".si-ig-bar")].map((r) => ({
    label: r.querySelector(".si-ig-bl").textContent, top: r.querySelector(".si-ig-bl").getBoundingClientRect().top,
    value: r.querySelector(".si-ig-bv").textContent, width: +r.querySelector("rect").getAttribute("width"),
    moving: !!(r.style.translate || r.querySelector(".si-ig-bl").style.translate),
  }));
  const shown = [...rows].sort((a, b) => a.top - b.top);
  return { on, n: frames.length, order: shown.map((r) => r.label), values: shown.map((r) => r.value), widths: shown.map((r) => r.width),
    moving: rows.some((r) => r.moving), out: !!fig.querySelector(".is-out"), at: (frames[on].querySelector(".si-chart-at") || {}).textContent };
}, slide);

try {
  for (const [w, h] of [[1440, 900], [400, 800], [820, 1180], [3840, 2160]]) {
    const ctx = await browser.newContext({ viewport: { width: w, height: h } });
    const page = await ctx.newPage();
    const errors = [];
    page.on("pageerror", (e) => errors.push(e.message));
    await page.goto(url + "#1");
    await page.waitForTimeout(1800);
    let s = await state(page, 1);
    check(s.on === 0 && s.n === 3 && eq(s.order, ["Carrier", "Address", "Warehouse", "Payment", "Weather"]) && s.at === "Q1", `${w}x${h}: a race starts on its first column, sorted (${s.order})`);
    await page.keyboard.press("ArrowRight");
    await page.waitForTimeout(250);
    s = await state(page, 1);
    check(s.on === 1 && s.moving, `${w}x${h}: a step glides the bars to their new places`);
    await page.waitForTimeout(1500);
    s = await state(page, 1);
    check(eq(s.order, ["Warehouse", "Carrier", "Payment", "Address", "Weather"]) && !s.moving && s.values[0] === "25%" && s.at === "Q2", `${w}x${h}: and settles in the new order (${s.order}; ${s.values})`);
    await page.keyboard.press("ArrowRight");
    await page.waitForTimeout(1600);
    s = await state(page, 1);
    check(eq(s.order, ["Payment", "Weather", "Carrier", "Warehouse", "Address"]) && s.widths[0] === 100 && s.values[0] === "21%" && s.at === "Q3", `${w}x${h}: a race rescales each step to its top bar (${s.order}; ${s.widths})`);
    // the show: bars, then a donut of the same table
    await page.keyboard.press("ArrowRight");
    await page.waitForTimeout(1600);
    await page.keyboard.press("ArrowRight");
    await page.waitForTimeout(150);
    s = await state(page, 2);
    const kind = await page.evaluate(() => document.querySelector("#s2 .si-chart-frame[data-on]").getAttribute("data-kind"));
    check(s.on === 1 && kind === "donut" && s.out, `${w}x${h}: a step turns the bars into a donut, the bars fading out`);
    // the fold: each bar bends into its arc on a layer over the stage
    const fold = () => page.evaluate(() => [...document.querySelectorAll("#s2 .si-chart-fold path")].map((p) => p.getAttribute("d")));
    const f1 = await fold();
    await page.waitForTimeout(250);
    const f2 = await fold();
    check(f1.length === 3 && f2.length === 3 && !eq(f1, f2), `${w}x${h}: each of the 3 bars folds into its arc (${f1.length} paths, moving)`);
    await page.waitForTimeout(1600);
    check((await fold()).length === 0, `${w}x${h}: the fold layer is gone once the donut is drawn`);
    const arcs = await page.evaluate(() => [...document.querySelectorAll("#s2 .si-chart-frame[data-on] circle[data-key]")].map((c) => parseFloat(c.getAttribute("stroke-dasharray"))));
    s = await state(page, 2);
    check(!s.out && Math.abs(arcs.reduce((a, b) => a + b, 0) - 100) < 0.1, `${w}x${h}: the donut ends whole (${arcs})`);
    // the same id on the next slide carries on
    await page.keyboard.press("ArrowRight");
    await page.waitForTimeout(50);
    const tx = await page.evaluate(() => document.querySelector(".si-deck").getAttribute("data-si-tx"));
    check(tx === "canvas", `${w}x${h}: the same id on the next slide does not fade the slide (${tx})`);
    await page.waitForTimeout(1500);
    const cols = await page.evaluate(() => [...document.querySelectorAll("#s3 .si-chart-frame[data-on] .si-ig-bv")].map((b) => b.textContent));
    check(eq(cols, ["1,400", "900", "1,600"]), `${w}x${h}: and ends on its own values (${cols})`);
    // 3D: a race of solid bars, a donut whose slice lifts out, a funnel that drops in
    const cubs = () => page.evaluate(() => [...document.querySelectorAll("#s4 .si-chart-frame[data-on] .si-c3-cub")]
      .map((c) => ({ key: c.dataset.key, x: parseFloat(/translate3d\(([-\d.]+)/.exec(c.style.transform)[1]), h: c.style.getPropertyValue("--h"), v: c.querySelector("b").textContent }))
      .sort((a, b) => a.x - b.x));
    await page.keyboard.press("ArrowRight");
    await page.waitForTimeout(1800);
    await page.keyboard.press("ArrowRight");
    await page.waitForTimeout(1500);
    await page.keyboard.press("ArrowRight");
    await page.waitForTimeout(200);
    const mid = await cubs();
    await page.waitForTimeout(1400);
    const fin = await cubs();
    check(eq(fin.map((c) => c.key), ["2", "4", "0", "1", "3"]) && fin[0].h === "6em" && fin[0].v === "21%", `${w}x${h}: a 3D race ends in rank order, the top bar full height (${fin.map((c) => c.key)})`);
    check(!eq(mid.map((c) => c.x), fin.map((c) => c.x)), `${w}x${h}: its bars slide to their new rank`);
    await page.keyboard.press("ArrowRight");
    await page.waitForTimeout(1800);
    await page.keyboard.press("ArrowRight");
    await page.waitForTimeout(1300);
    const lift = await page.evaluate(() => [...document.querySelectorAll("#s5 .si-chart-frame[data-on] svg:last-child circle")].map((c) => c.getAttribute("transform")));
    check(lift[1] !== "translate(0 0)" && lift[0] === "translate(0 0)" && lift[2] === "translate(0 0)", `${w}x${h}: explode=Late lifts that slice out on its step (${lift})`);
    await page.keyboard.press("ArrowRight");
    await page.waitForTimeout(2600);
    const drop = await page.evaluate(() => [...document.querySelectorAll("#s6 .si-c3-cub")].map((c) => c.style.transform));
    check(eq(drop, ["translate3d(0px, 0em, 0px)", "translate3d(0px, -1.05em, 0px)", "translate3d(0px, -2.1em, 0px)"]) || eq(drop, ["translate3d(0,0em,0)", "translate3d(0,-1.05em,0)", "translate3d(0,-2.1em,0)"]), `${w}x${h}: a funnel's layers drop in and stack (${drop})`);
    check(errors.length === 0, `${w}x${h}: no page errors ${errors.join("; ")}`);
    await ctx.close();
  }

  // reduced motion: a step shows its final values at once
  const rm = await browser.newContext({ viewport: { width: 1440, height: 900 }, reducedMotion: "reduce" });
  const rp = await rm.newPage();
  await rp.goto(url + "#1");
  await rp.waitForTimeout(400);
  await rp.keyboard.press("ArrowRight");
  await rp.waitForTimeout(60);
  let s = await state(rp, 1);
  check(!s.moving && eq(s.order, ["Warehouse", "Carrier", "Payment", "Address", "Weather"]) && s.values[0] === "25%", "reduced motion: the final order and values at once");
  // a #N.k link opens on that step
  await rp.goto(url + "#1.2");
  await rp.waitForTimeout(300);
  s = await state(rp, 1);
  check(s.on === 2 && s.at === "Q3", "a #1.2 link shows the third column");
  await rm.close();

  // a page: the last column without the script; plays to it once in view
  const pc = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  const pp = await pc.newPage();
  await pp.goto(pageURL);
  await pp.waitForTimeout(300);
  const first = await pp.evaluate(() => [...document.querySelectorAll(".si-chart-frame")].findIndex((f) => f.hasAttribute("data-on")));
  await pp.evaluate(() => document.querySelector(".si-chart").scrollIntoView());
  await pp.waitForTimeout(7500);
  const end = await pp.evaluate(() => [...document.querySelectorAll(".si-chart-frame")].findIndex((f) => f.hasAttribute("data-on")));
  const nav = await pp.evaluate(() => !!document.querySelector(".si-chart-nav button[data-a=play]"));
  check(first === 0 && end === 2 && nav, `a page waits on the first column (${first}), plays to the last once in view (${end}), with ← → Play`);
  await pc.close();
} finally {
  await browser.close();
  rmSync(dir, { recursive: true, force: true });
}
console.log(failed ? `${failed} failed` : "chart e2e ok");
process.exit(failed ? 1 : 0);
