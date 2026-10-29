// Headless end-to-end and visual QA for `si --web`. Run: make build && npm run e2e
import { chromium } from "playwright";
import { spawn } from "node:child_process";
import { cpSync, mkdtempSync, readFileSync, writeFileSync, rmSync, existsSync, mkdirSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const si = join(here, "../../../bin/si");
const out = join(here, "out");
mkdirSync(out, { recursive: true });

// the tests mutate files, so run against a throwaway copy of the fixture
const root = mkdtempSync(join(tmpdir(), "si-e2e-"));
cpSync(join(here, "fixture"), root, { recursive: true });
const ONE = join(root, "sections/one.md");
const ORIGINAL = readFileSync(ONE, "utf8");

const srv = spawn(si, ["--web", root, "--addr", "127.0.0.1:0", "--no-open"], { stdio: ["ignore", "pipe", "pipe"] });
const url = await new Promise((res, rej) => {
  let buf = "";
  const t = setTimeout(() => rej(new Error("no URL from si --web: " + buf)), 10000);
  const on = (d) => { buf += d; const m = buf.match(/at (http:\/\/\S+)/); if (m) { clearTimeout(t); res(m[1]); } };
  srv.stdout.on("data", on); srv.stderr.on("data", on);
});

const browser = await chromium.launch({ headless: true });
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const until = async (fn, ms = 5000, what = "condition") => {
  const end = Date.now() + ms;
  for (;;) {
    try { const v = await fn(); if (v) return v; } catch { /* retry */ }
    if (Date.now() > end) throw new Error("timeout: " + what);
    await sleep(100);
  }
};

async function open(theme, size = { width: 1440, height: 900 }) {
  const ctx = await browser.newContext({ viewport: size });
  await ctx.addInitScript((t) => { try { if (!localStorage.getItem("si-studio-theme")) localStorage.setItem("si-studio-theme", t); } catch {} }, theme);
  const page = await ctx.newPage();
  page.on("dialog", (d) => d.accept());
  await page.goto(url);
  await page.waitForSelector(".app");
  return { ctx, page };
}
async function openFile(page, path) {
  const sel = `.row-main[title="${path}"]`;
  if (!(await page.$(sel))) await page.click(`.row-main[title="${path.split("/")[0]}"]`);
  await page.click(sel);
}
async function setText(page, text) {
  await page.click(".cm-content");
  await page.keyboard.press("ControlOrMeta+A");
  await page.keyboard.insertText(text);
}
const frameOf = async (page) => (await page.$("iframe")).contentFrame();
const frameUrl = async (page) => (await frameOf(page)).url();
const frameText = async (page) => (await frameOf(page)).evaluate(() => document.body.textContent);
const reset = () => writeFileSync(ONE, ORIGINAL);

const tests = {
  async "edit updates preview"() {
    const { ctx, page } = await open("dark");
    await openFile(page, "sections/one.md");
    await page.waitForSelector(".cm-content");
    await setText(page, ORIGINAL.replace("First point", "Renamed heading"));
    await until(async () => (await frameText(page)).includes("Renamed heading"), 3000, "preview shows new heading");
    await ctx.close();
  },
  async "cursor follows slide"() {
    const { ctx, page } = await open("dark");
    await openFile(page, "sections/one.md");
    await page.click(".cm-line:has-text('Second point')");
    await until(async () => (await frameUrl(page)).endsWith("#3"), 3000, "iframe hash #3");
    await ctx.close();
  },
  async "conflict banner"() {
    const { ctx, page } = await open("dark");
    await openFile(page, "sections/one.md");
    await page.waitForSelector(".cm-content");
    await setText(page, "## Mine\n");
    writeFileSync(ONE, "## Theirs\n");
    await page.waitForSelector(".banner-warn", { timeout: 6000 });
    await page.screenshot({ path: join(out, "conflict.png") });
    await page.click("text=Keep mine");
    await until(() => readFileSync(ONE, "utf8") === "## Mine\n", 4000, "disk has editor text");
    await ctx.close();
  },
  async "reload keeps unsaved text"() {
    const { ctx, page } = await open("dark");
    await openFile(page, "sections/one.md");
    await page.waitForSelector(".cm-content");
    // block saves so the text can only survive through the localStorage stash (a crash before the save lands)
    const block = (r) => (r.request().method() === "PUT" ? r.abort() : r.continue());
    await page.route("**/*", block);
    await setText(page, "## Unsaved survivor\n");
    await sleep(1000);
    await page.unroute("**/*", block);
    await page.reload();
    await page.waitForSelector(".app");
    if (!(await page.$(".banner-info"))) await openFile(page, "sections/one.md");
    await page.waitForSelector(".banner-info:has-text('Unsaved changes restored')", { timeout: 6000 });
    await until(async () => (await page.textContent(".cm-content")).includes("Unsaved survivor"), 3000, "text restored");
    await ctx.close();
  },
  async "deleted on disk keeps unsaved text"() {
    const { ctx, page } = await open("dark");
    await openFile(page, "sections/one.md");
    await page.waitForSelector(".cm-content");
    await setText(page, "## Typed then deleted\n");
    rmSync(ONE);
    await page.waitForSelector(".banner-err:has-text('Deleted on disk')", { timeout: 8000 });
    await page.click("text=Save it again");
    await until(() => existsSync(ONE) && readFileSync(ONE, "utf8").includes("Typed then deleted"), 5000, "file recreated with typed text");
    await ctx.close();
  },
  async "diagnostic shows"() {
    const { ctx, page } = await open("dark");
    await openFile(page, "sections/one.md");
    await page.waitForSelector(".cm-content");
    await setText(page, ORIGINAL + "\n<!-- layout: nope -->\n");
    await page.waitForSelector(".cm-lintRange, .cm-lint-marker", { timeout: 8000 });
    await page.click(".foot-check");
    await page.waitForSelector(".problems button.prob", { timeout: 4000 });
    await page.screenshot({ path: join(out, "problems.png") });
    await ctx.close();
  },
  async "crlf file opens clean and saves crlf"() {
    const crlf = ORIGINAL.replace(/\r?\n/g, "\r\n");
    writeFileSync(ONE, crlf);
    const { ctx, page } = await open("dark");
    await openFile(page, "sections/one.md");
    await page.waitForSelector(".cm-content");
    await sleep(1500); // past the autosave delay: a dirty doc would have been written by now
    if (await page.$(".tab-x.dirty")) throw new Error("CRLF file opened dirty");
    if (readFileSync(ONE, "utf8") !== crlf) throw new Error("opening rewrote the file");
    await page.click(".cm-line:has-text('Second point')");
    await page.keyboard.press("End");
    await page.keyboard.insertText(" edited");
    await until(() => readFileSync(ONE, "utf8").includes("Second point edited"), 5000, "edit saved");
    const disk = readFileSync(ONE, "utf8");
    if (/[^\r]\n/.test(disk) || disk !== crlf.replace("Second point", "Second point edited"))
      throw new Error("save changed line endings: " + JSON.stringify(disk.slice(0, 80)));
    await ctx.close();
  },
  async "closing a tab with unsaved text asks first"() {
    const { ctx, page } = await open("dark");
    await openFile(page, "sections/one.md");
    await page.waitForSelector(".cm-content");
    await page.route("**/_si/studio/api/file*", (r) => (r.request().method() === "PUT" ? r.abort() : r.continue()));
    await setText(page, "## Never saved\n");
    await page.waitForSelector(".banner-err:has-text('Not saved')", { timeout: 6000 });
    await page.click(".tab-x");
    await page.waitForSelector(".dialog:has-text('Close one.md?')", { timeout: 3000 });
    await page.click(".dialog button:has-text('Cancel')");
    if (!(await page.$(".tab"))) throw new Error("tab closed despite Cancel");
    await page.click(".tab-x");
    await page.click(".dialog button:has-text('Close without saving')");
    await until(async () => !(await page.$(".tab")), 3000, "tab closed");
    await ctx.close();
  },
  async "image opens as image"() {
    const { ctx, page } = await open("dark");
    await openFile(page, "img/p.png");
    await page.waitForSelector(".imgview img");
    if (await page.locator(".cm-content").isVisible()) throw new Error("editor visible for image");
    await ctx.close();
  },
  async "drawer asks for placeholders"() {
    const { ctx, page } = await open("dark");
    await page.goto(new URL("/rbdeck.html", url).href);
    await page.waitForSelector(".si-deck, .si-slide, body");
    await page.keyboard.press("`");
    await page.waitForSelector(".si-rb.is-open .si-rb-item", { timeout: 8000 });
    const term = () => page.evaluate(() => document.querySelector(".si-term-main").innerText);
    await sleep(1500); // shell start
    await page.click(".si-rb-item");
    await page.waitForSelector(".si-rb-form input");
    if ((await page.inputValue(".si-rb-form input")) !== "world") throw new Error("default not prefilled");
    await page.screenshot({ path: join(out, "drawer-form-dark.png") });
    await page.fill(".si-rb-form input", "si");
    await page.click(".si-rb-form button:has-text('Run')");
    await until(async () => (await term()).includes("hello-si"), 6000, "output hello-si");
    if (await page.$(".si-rb-form")) throw new Error("form still open");
    await page.click(".si-rb-item");
    await page.waitForSelector(".si-rb-form input");
    if ((await page.inputValue(".si-rb-form input")) !== "si") throw new Error("last value not remembered");
    await page.fill(".si-rb-form input", "nope");
    await page.keyboard.press("Escape");
    await sleep(600);
    if (await page.$(".si-rb-form")) throw new Error("form open after Esc");
    if ((await term()).includes("nope")) throw new Error("Esc typed something");
    await ctx.close();
  },
  async screenshots() {
    for (const theme of ["dark", "light"]) for (const [w, h] of [[1440, 900], [1280, 800]]) {
      const { ctx, page } = await open(theme, { width: w, height: h });
      await openFile(page, "sections/one.md");
      await page.waitForSelector(".cm-content");
      await sleep(1500);
      await page.screenshot({ path: join(out, `${theme}-${w}x${h}.png`) });
      await ctx.close();
    }
  },
};

let failed = 0;
for (const [name, fn] of Object.entries(tests)) {
  reset();
  try { await fn(); console.log("ok   " + name); }
  catch (e) { failed++; console.log("FAIL " + name + "\n     " + String(e.message).split("\n")[0]); }
}
await browser.close();
srv.kill();
rmSync(root, { recursive: true, force: true });
console.log(failed ? `${failed} failed` : "all passed");
process.exit(failed ? 1 : 0);
