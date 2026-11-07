// Real pointer tests on built pages: mouse moves, clicks, double-clicks and
// drags as a person makes them (page.mouse), on a 3D mind map (WebGL through
// SwiftShader), a 2D map and a code window. Headless; builds the examples with
// ../../bin/si (make build) into a temp folder. Run: npm run e2e:pointer
import { chromium } from "playwright";
import { execFileSync } from "node:child_process";
import { mkdtempSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

const si = new URL("../../../bin/si", import.meta.url).pathname;
const ex = new URL("../../../examples/", import.meta.url).pathname;
const out = mkdtempSync(join(tmpdir(), "si-pointer-"));
execFileSync(si, ["build", join(ex, "mindmap"), "-o", join(out, "mindmap")], { stdio: "ignore" });
execFileSync(si, ["build", join(ex, "code"), "-o", join(out, "code")], { stdio: "ignore" });

const browser = await chromium.launch({ args: ["--use-gl=angle", "--use-angle=swiftshader", "--enable-unsafe-swiftshader"] });
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
async function page(file, size = { width: 1440, height: 900 }) {
  const p = await browser.newPage({ viewport: size });
  p.errors = [];
  p.on("pageerror", (e) => p.errors.push(e.message));
  await p.goto("file://" + join(out, file));
  return p;
}
// where a 3D topic's top face is on screen
const pillAt = (p, title) => p.evaluate((title) => {
  const d = document.querySelector(".si-mindmap").siMindmap.d3;
  const n = d.main.userData.all.find((x) => x.t === title);
  const v = new THREE.Vector3(); n.holder.getWorldPosition(v); v.y += n.holder.userData.size.h; // the top face
  v.project(d.camera);
  const r = d.renderer.domElement.getBoundingClientRect();
  return { x: r.left + (v.x + 1) / 2 * r.width, y: r.top + (1 - v.y) / 2 * r.height };
}, title);

const tests = {
  "3D: a click on a pill opens its card": async () => {
    const p = await page("mindmap/thursday.html");
    await sleep(3500);
    const at = await pillAt(p, "Checkout");
    await p.mouse.move(at.x, at.y, { steps: 5 });
    await p.mouse.click(at.x, at.y);
    await p.waitForSelector(".si-mm-card:not([hidden])", { timeout: 4000 });
    const t = await p.textContent(".si-mm-card strong");
    if (t !== "Checkout") throw new Error("card shows " + t);
    if (p.errors.length) throw new Error(p.errors[0]);
    await p.close();
  },
  "3D: a double-click on a map › pill flies into its map": async () => {
    const p = await page("mindmap/thursday.html");
    await sleep(3500);
    const at = await pillAt(p, "Payment");
    await p.mouse.move(at.x, at.y, { steps: 5 });
    await p.mouse.dblclick(at.x, at.y);
    await p.waitForFunction(() => document.querySelector(".si-mm-crumbs").textContent.includes("Payment"), null, { timeout: 6000 });
    if (p.errors.length) throw new Error(p.errors[0]);
    await p.close();
  },
  "2D: dragging a topic moves only that topic": async () => {
    const p = await page("mindmap/warehouse.html");
    await sleep(1500);
    const node = (t) => p.locator(".si-mm-node", { has: p.locator(`text="${t}"`) }).first();
    const before = await node("Zone").boundingBox(), other = await node("Wave").boundingBox();
    await p.mouse.move(before.x + before.width / 2, before.y + before.height / 2);
    await p.mouse.down();
    await p.mouse.move(before.x + before.width / 2 + 60, before.y + before.height / 2 + 40, { steps: 10 });
    await p.mouse.up();
    const after = await node("Zone").boundingBox(), still = await node("Wave").boundingBox();
    if (Math.abs(after.x - before.x - 60) > 6 || Math.abs(after.y - before.y - 40) > 6) throw new Error(`moved ${after.x - before.x},${after.y - before.y}`);
    if (Math.abs(still.x - other.x) > 1 || Math.abs(still.y - other.y) > 1) throw new Error("a sibling moved");
    await p.close();
  },
  "code window: dragging a callout moves its note": async () => {
    const p = await page("code/retry.html");
    await sleep(1500);
    const note = p.locator(".si-cw-bubble").first();
    const b = await note.boundingBox();
    await p.mouse.move(b.x + 20, b.y + 10);
    await p.mouse.down();
    await p.mouse.move(b.x - 30, b.y + 50, { steps: 10 });
    await p.mouse.up();
    const a = await note.boundingBox();
    if (Math.abs(a.x - b.x + 50) > 6 || Math.abs(a.y - b.y - 40) > 6) throw new Error(`moved ${a.x - b.x},${a.y - b.y}`);
    if (p.errors.length) throw new Error(p.errors[0]);
    await p.close();
  },
};

let failed = 0;
for (const [name, fn] of Object.entries(tests)) {
  try { await fn(); console.log("ok   " + name); }
  catch (e) { failed++; console.log("FAIL " + name + "\n     " + String(e.message).split("\n")[0]); }
}
await browser.close();
rmSync(out, { recursive: true, force: true });
console.log(failed ? `${failed} failed` : "all passed");
process.exit(failed ? 1 : 0);
