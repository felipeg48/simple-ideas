// Headless end-to-end for the Studio's Class mode. Run: make build && node e2e/class.e2e.mjs
// A course folder → Class → Start the class (this machine only) → the
// dashboard shows in the Studio with the learners' link and QR → a learner's
// event arrives live → Stop. Screenshots at desktop and phone width in e2e/out/.
import { chromium } from "playwright";
import { spawn, spawnSync } from "node:child_process";
import { mkdtempSync, mkdirSync, rmSync, readdirSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const si = join(here, "../../../bin/si");
const out = join(here, "out");
mkdirSync(out, { recursive: true });
const root = mkdtempSync(join(tmpdir(), "si-e2e-class-"));
const made = spawnSync(si, ["new", join(root, "web101"), "--type", "course"], { encoding: "utf8" });
if (made.status !== 0) throw new Error("si new: " + made.stderr);

const srv = spawn(si, ["--web", root, "--addr", "127.0.0.1:0", "--no-open"], { stdio: ["ignore", "pipe", "pipe"] });
process.on("exit", () => { srv.kill(); rmSync(root, { recursive: true, force: true }); });
const url = await new Promise((res, rej) => {
  let buf = "";
  const t = setTimeout(() => rej(new Error("no URL from si --web: " + buf)), 10000);
  const on = (d) => { buf += d; const m = buf.match(/at (http:\/\/\S+)/); if (m) { clearTimeout(t); res(m[1]); } };
  srv.stdout.on("data", on); srv.stderr.on("data", on);
});
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
let failed = 0; const check = (ok, what) => { console.log((ok ? "ok   " : "FAIL ") + what); if (!ok) failed++; };
const browser = await chromium.launch({ headless: true });
const page = await (await browser.newContext({ viewport: { width: 1440, height: 900 } })).newPage();
const errors = []; page.on("pageerror", (e) => errors.push(e.message));
try {
  await page.goto(url);
  await page.waitForSelector(".app");
  await page.locator(".modes button", { hasText: "Class" }).click();
  await page.getByRole("button", { name: "Start the class" }).waitFor();
  check(await page.getByLabel("Learners on my network").isChecked(), "the network choice is offered, on by default");
  await page.getByLabel("Learners on my network").uncheck(); // this machine only: nothing reachable from outside in a test
  await page.getByRole("button", { name: "Start the class" }).click();
  const frame = page.locator("iframe.class-dash");
  await frame.waitFor();
  const dash = await frame.getAttribute("src");
  check(/^http:\/\/127\.0\.0\.1:\d+\/_si\/class$/.test(dash), "the dashboard of the course's class: " + dash);
  check(await page.locator(".class-qr svg").count() === 1, "a QR code of the learners' link");
  const learners = await page.locator(".class-link a").getAttribute("href");
  const inner = page.frameLocator("iframe.class-dash");
  await inner.getByText("Waiting for the first learner", { exact: false }).waitFor();

  // a learner finishes the first lesson: the dashboard shows them, live
  const lessonDir = readdirSync(join(root, "web101")).filter((f) => /^\d/.test(f)).sort()[0];
  const lesson = readdirSync(join(root, "web101", lessonDir)).filter((f) => f.endsWith(".md")).sort()[0];
  const pageID = lessonDir + "/" + lesson.replace(/\.md$/, ".html");
  const ev = { version: 1, event: "lesson.completed", id: "e-1", audience: "learner", lesson: { id: pageID }, learner: { id: "l-0123456789ab", name: "Ana" }, data: {} };
  const r = await fetch(learners + "_si/class/event?page=/" + pageID, { method: "POST", headers: { Origin: learners.replace(/\/$/, ""), "Content-Type": "text/plain" }, body: JSON.stringify(ev) });
  check(r.status === 202, "the course takes a learner's event: " + r.status);
  await inner.getByText("Ana").first().waitFor({ timeout: 8000 });
  check(true, "the learner shows in the Studio's class view, live");
  await page.screenshot({ path: join(out, "class-desktop.png") });
  await page.setViewportSize({ width: 400, height: 800 });
  await sleep(300);
  await page.screenshot({ path: join(out, "class-phone.png") });
  await page.setViewportSize({ width: 1440, height: 900 });

  // leaving and coming back attaches to the running class
  await page.locator(".modes button", { hasText: "Write" }).click();
  await page.locator(".modes button", { hasText: "Class" }).click();
  await frame.waitFor();
  check(await frame.getAttribute("src") === dash, "back in Class: the same class");

  await page.getByRole("button", { name: "Stop the class" }).click();
  await page.getByRole("button", { name: "Start the class" }).waitFor();
  let gone = false; try { await fetch(dash); } catch { gone = true; }
  check(gone, "Stop ends the class");
  check(errors.length === 0, "no page errors " + errors.join("; "));
} catch (e) {
  console.log("FAIL " + e.message);
  failed++;
  await page.screenshot({ path: join(out, "class-fail.png") }).catch(() => {});
}
await browser.close();
process.exit(failed ? 1 : 0);
