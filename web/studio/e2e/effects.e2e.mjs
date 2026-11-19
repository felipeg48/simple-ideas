// The motion transitions (cube … stack) and build animations (animate:,
// steps: focus|deal|reorder): a built deck (file://) that uses every one.
// Steps through it forward at two sizes: each transition plays, cleans up
// its copies and lands on the next slide; going back plays the mirror; each
// animation runs and leaves the block as written; no page or console errors.
// Reduced motion shows the final state at once. SI_FX_SHOTS=dir saves
// mid-transition pictures. Headless; needs ../../bin/si (make build).
// Run: npm run e2e:effects
import { chromium } from "playwright";
import { execFileSync } from "node:child_process";
import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { pathToFileURL } from "node:url";

const si = new URL("../../../bin/si", import.meta.url).pathname;
const dir = mkdtempSync(join(tmpdir(), "si-effects-"));
const tx = ["cube", "flip", "iris", "wipe", "shatter", "depth", "blinds", "portal", "glitch", "stack"];
const titles = ["Order placed", "Payment authorised", "Picked", "Packed", "Label printed", "Handed to the courier", "Out for delivery", "The receipt", "Stock count drifted", "Delivered"];
let md = "# One order, ten moves\n\nTuesday 09:14\n";
tx.forEach((t, i) => {
  md += `\n---\n\n<!-- transition: ${t}${t === "cube" ? " 0.8s" : ""} -->\n## ${titles[i]}\n\nCart €84 · two pairs of shoes\n`;
  if (titles[i + 1] === "The receipt") md += "\n<!-- id: pay -->\n![receipt](receipt.svg)\n";
});
md += `
---

## The courier's log

<!-- animate: type -->
> Delivered. Two days, door to door.

<!-- step -->
<!-- animate: words -->
Nobody owns the promise from end to end.

---

## What we promised

<!-- animate: highlight -->
Shoes at the door in ==two days==.

<!-- step -->
<!-- animate: count -->
**12,480** orders on time

---

## The handoff

<!-- animate: draw -->
<!-- svg: handoff.svg -->

<!-- step -->
<!-- animate: decode -->
Stock count drifted at night.

---

## Where it broke

<!-- steps: focus -->
- Shop
- Payments
- Warehouse

---

## Five teams

<!-- steps: deal -->
- Shop
- Payments
- Courier

---

## Late orders by cause

<!-- steps: reorder -->
- Address wrong 12%
- Stock drift 61%
- Weather 4%
- Courier no-show 23%

---

## Thursday 18:02

<!-- step -->
<!-- animate: confetti -->
**Delivered**
`;
writeFileSync(join(dir, "talk.md"), md);
writeFileSync(join(dir, "receipt.svg"), `<svg xmlns="http://www.w3.org/2000/svg" width="320" height="180" viewBox="0 0 160 90"><rect x="4" y="4" width="152" height="82" rx="8" fill="#eef0ff" stroke="#4b5ff0" stroke-width="3"/><text x="80" y="54" text-anchor="middle" font-size="22" fill="#4b5ff0">PAY</text></svg>`);
writeFileSync(join(dir, "handoff.svg"), `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 300 90"><rect x="6" y="20" width="80" height="44" rx="8" fill="none" stroke="currentColor" stroke-width="2.2"/><text x="22" y="47" fill="currentColor">Shop</text><path d="M90 42 H150 M140 34 L150 42 L140 50" fill="none" stroke="currentColor" stroke-width="2.2"/><rect x="154" y="20" width="96" height="44" rx="8" fill="none" stroke="currentColor" stroke-width="2.2"/><text x="166" y="47" fill="currentColor">Payments</text></svg>`);
execFileSync(si, ["build", join(dir, "talk.md"), "-o", join(dir, "out")], { stdio: "ignore" });
const url = pathToFileURL(join(dir, "out", "talk.html")).href;
const shots = process.env.SI_FX_SHOTS || "";
if (shots) mkdirSync(shots, { recursive: true });

const browser = await chromium.launch();
let failed = 0;
const check = (ok, what) => { console.log((ok ? "ok   " : "FAIL ") + what); if (!ok) failed++; };
const state = (page) => page.evaluate(() => {
  const d = document.querySelector(".si-deck"), cur = document.querySelector(".si-slide.si-current");
  return {
    tx: d.getAttribute("data-si-tx"), hash: location.hash, anims: document.getAnimations().length,
    tiles: document.querySelectorAll(".si-tx-tile,.si-tx-bar,.si-fx").length,
    on: document.querySelectorAll(".si-tx-on,.si-tx-run").length,
    live: document.querySelectorAll("[data-si-live]").length,
    visible: cur && getComputedStyle(cur).visibility === "visible" && getComputedStyle(cur).opacity === "1",
  };
});
const settle = async (page) => { await page.evaluate(() => document.getAnimations().forEach((a) => { try { a.finish(); } catch (e) {} })); await page.waitForTimeout(80); };
const text = (page, q) => page.evaluate((q) => { const e = document.querySelector(".si-slide.si-current " + q); return e ? e.textContent.replace(/\s+/g, " ").trim() : null; }, q);
try {
  for (const [w, h] of [[1440, 900], [400, 800]]) {
    const ctx = await browser.newContext({ viewport: { width: w, height: h } });
    const page = await ctx.newPage();
    const errors = [];
    page.on("pageerror", (e) => errors.push(e.message));
    page.on("console", (m) => { if (m.type() === "error") errors.push(m.text()); });
    await page.goto(url + "#1");
    await page.waitForTimeout(400);
    for (let i = 0; i < tx.length; i++) {
      await page.keyboard.press("ArrowRight");
      await page.waitForTimeout(20);
      const s = await state(page);
      check(s.tx === tx[i] && s.anims > 0, `${w}x${h}: ${tx[i]} plays (${s.anims} animations)`);
      if (shots) {
        await page.evaluate(() => document.getAnimations().forEach((a) => { a.pause(); a.currentTime = a.effect.getComputedTiming().endTime * 0.45; }));
        await page.screenshot({ path: join(shots, `${tx[i]}-${w}x${h}.png`) });
      }
      await settle(page);
      await page.waitForTimeout(150);
      const e = await state(page);
      check(e.tiles === 0 && e.on === 0 && e.visible && e.hash === `#${i + 2}`, `${w}x${h}: ${tx[i]} lands on slide ${i + 2} and cleans up`);
    }
    // back: the mirror of the slide being left
    await page.keyboard.press("ArrowLeft");
    await page.waitForTimeout(20);
    let s = await state(page);
    check(s.tx === "stack" && s.anims > 0, `${w}x${h}: going back from stack plays it mirrored`);
    await settle(page);
    await page.keyboard.press("ArrowRight");
    await settle(page);
    // the animations
    await page.keyboard.press("ArrowRight"); // the courier's log: type runs with the slide
    await page.waitForTimeout(500);
    const typing = await text(page, "blockquote");
    check(typing !== null && typing.length < "Delivered. Two days, door to door.".length, `${w}x${h}: type is typing ("${typing}")`);
    await page.waitForTimeout(2600);
    check(await text(page, "blockquote") === "Delivered. Two days, door to door.", `${w}x${h}: type ends with the words as written`);
    await page.keyboard.press("ArrowRight");
    await page.waitForTimeout(30);
    check(await page.evaluate(() => document.querySelectorAll(".si-current .si-word").length) === 8, `${w}x${h}: words rise one by one`);
    await settle(page);
    check(await text(page, "[data-si-anim=words]") === "Nobody owns the promise from end to end." && (await state(page)).live === 0, `${w}x${h}: words end as written`);
    await page.keyboard.press("ArrowRight"); // highlight with the slide
    await page.waitForTimeout(30);
    check(await page.evaluate(() => document.querySelector(".si-current .hl").getAnimations().length) === 1, `${w}x${h}: the marker sweeps`);
    await settle(page);
    await page.keyboard.press("ArrowRight"); // count
    await page.waitForTimeout(30);
    check(await page.evaluate(() => document.querySelectorAll(".si-current .si-odo").length) === 5, `${w}x${h}: count rolls five digits`);
    if (shots) { await page.waitForTimeout(500); await page.screenshot({ path: join(shots, `count-${w}x${h}.png`) }); }
    await settle(page);
    check(await text(page, "[data-si-anim=count]") === "12,480 orders on time", `${w}x${h}: count ends on the number`);
    await page.keyboard.press("ArrowRight"); // draw with the slide
    await page.waitForTimeout(400);
    if (shots) await page.screenshot({ path: join(shots, `draw-${w}x${h}.png`) });
    check(await page.evaluate(() => document.querySelectorAll(".si-current [data-si-anim=draw] rect, .si-current [data-si-anim=draw] path").length > 0 && document.querySelector(".si-current [data-si-anim=draw] rect").getAnimations().length === 1), `${w}x${h}: draw strokes the diagram on`);
    await settle(page);
    check(await page.evaluate(() => document.querySelector(".si-current [data-si-anim=draw] rect").style.strokeDasharray === ""), `${w}x${h}: draw leaves the strokes as drawn`);
    await page.keyboard.press("ArrowRight"); // decode
    await page.waitForTimeout(300);
    check(await text(page, "[data-si-anim=decode]") !== "Stock count drifted at night.", `${w}x${h}: decode starts scrambled`);
    await page.waitForTimeout(1500);
    check(await text(page, "[data-si-anim=decode]") === "Stock count drifted at night.", `${w}x${h}: decode ends as written`);
    await page.keyboard.press("ArrowRight"); // focus
    await page.keyboard.press("ArrowRight");
    await page.keyboard.press("ArrowRight");
    await page.waitForTimeout(800);
    const f = await page.evaluate(() => [...document.querySelectorAll(".si-current .si-steps-focus li")].map((e) => getComputedStyle(e).opacity));
    check(f[0] !== "1" && f[1] === "1" && f[2] !== "1", `${w}x${h}: focus dims all but the latest (${f.join(" ")})`);
    if (shots) await page.screenshot({ path: join(shots, `focus-${w}x${h}.png`) });
    await page.keyboard.press("ArrowRight");
    await page.keyboard.press("ArrowRight"); // deal
    await page.keyboard.press("ArrowRight");
    await page.keyboard.press("ArrowRight");
    await page.keyboard.press("ArrowRight");
    await page.keyboard.press("ArrowRight"); // reorder
    const before = await page.evaluate(() => [...document.querySelectorAll(".si-current .si-steps-reorder li")].map((e) => e.textContent.split(" ")[0]).join(","));
    check(before === "Address,Stock,Weather,Courier", `${w}x${h}: reorder starts in source order (${before})`);
    await page.keyboard.press("ArrowRight");
    await page.waitForTimeout(30);
    const moving = await page.evaluate(() => [...document.querySelectorAll(".si-current .si-steps-reorder li")].filter((e) => e.getAnimations().length).length);
    const after = await page.evaluate(() => [...document.querySelectorAll(".si-current .si-steps-reorder li")].map((e) => e.textContent.split(" ")[0]).join(","));
    check(after === "Stock,Courier,Address,Weather" && moving > 0, `${w}x${h}: reorder sorts by the numbers, highest first, gliding (${after}, ${moving} moving)`);
    await page.keyboard.press("ArrowLeft");
    const back = await page.evaluate(() => [...document.querySelectorAll(".si-current .si-steps-reorder li")].map((e) => e.textContent.split(" ")[0]).join(","));
    check(back === before, `${w}x${h}: back puts it in source order again`);
    await page.keyboard.press("ArrowRight");
    await settle(page);
    await page.keyboard.press("ArrowRight"); // confetti slide
    await page.keyboard.press("ArrowRight");
    await page.waitForTimeout(400);
    check(await page.evaluate(() => document.querySelectorAll(".si-current canvas.si-fx").length) === 1, `${w}x${h}: confetti bursts`);
    if (shots) await page.screenshot({ path: join(shots, `confetti-${w}x${h}.png`) });
    await page.waitForTimeout(2300);
    check((await state(page)).tiles === 0, `${w}x${h}: the confetti canvas is gone after`);
    check(errors.length === 0, `${w}x${h}: no page or console errors ${errors.join("; ")}`);
    await ctx.close();
  }
  // reduced motion: everything arrives in its final state
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 }, reducedMotion: "reduce" });
  const page = await ctx.newPage();
  const errors = [];
  page.on("pageerror", (e) => errors.push(e.message));
  await page.goto(url + "#1");
  await page.waitForTimeout(300);
  await page.keyboard.press("ArrowRight");
  let s = await state(page);
  check(s.anims === 0 && s.tiles === 0 && s.on === 0, `reduced motion: cube does not play (${s.anims})`);
  await page.goto(url + "#11");
  await page.waitForTimeout(300);
  await page.keyboard.press("ArrowRight");
  await page.waitForTimeout(30);
  check(await text(page, "blockquote") === "Delivered. Two days, door to door." && (await state(page)).live === 0, "reduced motion: type shows the words at once");
  await page.keyboard.press("ArrowRight");
  check(await page.evaluate(() => document.querySelectorAll(".si-current .si-word").length) === 0, "reduced motion: words do not split");
  // a jump shows the final state
  const jp = await (await browser.newContext({ viewport: { width: 1440, height: 900 } })).newPage();
  await jp.goto(url + "#13.1");
  await jp.waitForTimeout(100);
  check(await jp.evaluate(() => document.querySelectorAll(".si-current .si-odo,[data-si-live]").length) === 0, "a #N.k jump shows the final state");
  check(errors.length === 0, `reduced motion: no page errors ${errors.join("; ")}`);
} finally {
  await browser.close();
  if (process.env.SI_FX_KEEP) console.log("kept " + dir); else rmSync(dir, { recursive: true, force: true });
}
console.log(failed ? `${failed} failed` : "effects e2e ok");
process.exit(failed ? 1 : 0);
