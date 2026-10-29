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
const RB = join(root, "labs/runbook.json");
const RB_ORIGINAL = readFileSync(RB, "utf8");
const TALK = join(root, "talk.md");
const TALK_ORIGINAL = readFileSync(TALK, "utf8");
const LABOPS = join(root, "lab-ops.json");
const reset = () => { writeFileSync(ONE, ORIGINAL); writeFileSync(RB, RB_ORIGINAL); writeFileSync(TALK, TALK_ORIGINAL); rmSync(LABOPS, { force: true }); };
async function openRunbook(page) {
  await openFile(page, "labs/runbook.json");
  await page.waitForSelector(".rb-view .rb-entry");
}
const labels = (page) => page.$$eval(".rb-cat:first-of-type .rb-entry .rb-label", (ns) => ns.map((n) => n.textContent));

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
    await page.click(".si-rb-item");
    await page.waitForSelector(".si-rb-form input");
    await page.fill(".si-rb-form input", "nada");
    await page.focus(".si-rb-form button:has-text('Cancel')");
    await page.keyboard.press("Enter");
    await sleep(600);
    if (await page.$(".si-rb-form")) throw new Error("form open after Enter on Cancel");
    if ((await term()).includes("nada")) throw new Error("Enter on Cancel typed something");
    await ctx.close();
  },
  async "runbook visual edit is a minimal diff"() {
    const { ctx, page } = await open("dark");
    await openRunbook(page);
    await page.click("button[aria-label='Edit Tool versions']");
    await page.fill(".rb-form input[aria-label='Label']", "Tool versions (all)");
    await until(() => readFileSync(RB, "utf8") !== RB_ORIGINAL, 5000, "edit saved");
    const a = RB_ORIGINAL.split("\n"), b = readFileSync(RB, "utf8").split("\n");
    const diff = a.map((l, i) => (l === b[i] ? null : [l, b[i]])).filter(Boolean);
    if (a.length !== b.length || diff.length !== 1 || !diff[0][1].includes('"label": "Tool versions (all)"'))
      throw new Error("not a one-line diff: " + JSON.stringify(diff.slice(0, 3)));
    await ctx.close();
  },
  async "runbook duplicate delete and reorder"() {
    const { ctx, page } = await open("dark");
    await openRunbook(page);
    await page.click("button[aria-label='Duplicate Docker resources']");
    await until(async () => { const l = await labels(page); return l[l.indexOf("Docker resources") + 1] === "Docker resources (copy)"; }, 3000, "copy below");
    await page.focus("button[aria-label='Move Docker resources (copy) up']");
    await page.keyboard.press("Enter");
    await until(async () => { const l = await labels(page); return l[l.indexOf("Docker resources (copy)") + 1] === "Docker resources"; }, 3000, "copy moved up");
    await page.click("button[aria-label='Delete Docker resources (copy)']");
    await page.click(".dialog button:has-text('Delete')");
    await until(() => readFileSync(RB, "utf8") === RB_ORIGINAL, 5000, "file back to the original");
    await ctx.close();
  },
  async "runbook filter locks reordering"() {
    const { ctx, page } = await open("dark");
    await openRunbook(page);
    await page.fill("input[aria-label='Filter entries']", "docker info");
    await page.waitForSelector(".rb-entry .rb-label:has-text('Docker resources')");
    for (const sel of ["Move Docker resources up", "Move Docker resources down", "Move Preflight and images down"])
      if (!(await page.isDisabled(`button[aria-label='${sel}']`))) throw new Error(sel + " enabled while filtering");
    if ((await page.getAttribute(".rb-entry .rb-grip", "draggable")) !== "false") throw new Error("drag enabled while filtering");
    await page.fill("input[aria-label='Filter entries']", "");
    if (await page.isDisabled("button[aria-label='Move Docker resources up']")) throw new Error("move still disabled after clearing the filter");
    if (readFileSync(RB, "utf8") !== RB_ORIGINAL) throw new Error("filtering changed the file");
    await ctx.close();
  },
  async "runbook keyboard category move and undo"() {
    const { ctx, page } = await open("dark");
    await openRunbook(page);
    const cats = () => JSON.parse(readFileSync(RB, "utf8")).categories.map((c) => c.name);
    const focused = () => page.evaluate(() => document.activeElement?.getAttribute("aria-label"));
    await page.focus("button[aria-label='Move Preflight and images down']");
    await page.keyboard.press("Enter");
    await until(async () => (await focused()) === "Move Preflight and images down", 3000, "focus follows the category");
    await page.keyboard.press("Enter");
    await until(() => cats()[2] === "Preflight and images", 5000, "moved down twice");
    await page.keyboard.press("ControlOrMeta+z");
    await until(() => cats()[1] === "Preflight and images", 5000, "undo in the visual view");
    await ctx.close();
  },
  async "runbook invalid json disables visual"() {
    const { ctx, page } = await open("dark");
    await openRunbook(page);
    await page.click(".rb-mode button:has-text('JSON')");
    await page.waitForSelector(".cm-content");
    const broken = RB_ORIGINAL.replace('"label": "Tool versions",', '"label": "Tool versions"');
    await setText(page, broken);
    await until(() => readFileSync(RB, "utf8") === broken, 5000, "broken text saved");
    await page.click(".rb-mode button:has-text('Visual')");
    const msg = await (await page.waitForSelector(".rb-error", { timeout: 3000 })).textContent();
    if (!/Fix the JSON to use the visual editor/.test(msg) || !/line 8, column \d+/.test(msg)) throw new Error("bad message: " + msg);
    await page.screenshot({ path: join(out, "runbook-invalid-dark.png") });
    await sleep(2000);
    if (readFileSync(RB, "utf8") !== broken) throw new Error("visual view rewrote broken JSON");
    await page.click(".rb-error button");
    await until(async () => (await page.textContent(".cm-activeLine")).includes('"description"'), 3000, "cursor at the error");
    await ctx.close();
  },
  async "runbook insert placeholder"() {
    const { ctx, page } = await open("dark");
    await openRunbook(page);
    await page.click("button[aria-label='Edit Run the preflight check']");
    await page.focus(".rb-form textarea[aria-label='Command']");
    await page.keyboard.press("ControlOrMeta+End");
    await page.keyboard.insertText(" --ns ");
    await page.click(".rb-form button:has-text('Insert placeholder')");
    await page.fill(".ph-pop input[aria-label='Placeholder name']", "ns");
    await page.fill(".ph-pop input[aria-label='Default value']", "messaging");
    if ((await page.textContent(".ph-pop .ph-preview")) !== "{{ns:messaging}}") throw new Error("bad preview");
    await page.screenshot({ path: join(out, "runbook-placeholder-dark.png") });
    await page.click(".ph-pop button:has-text('Insert')");
    await page.waitForSelector(".rb-entry .rb-ph:has-text('ns')", { timeout: 3000 });
    await until(() => readFileSync(RB, "utf8").includes('"command": "bash ../labs/files/lab00/preflight.sh --ns {{ns:messaging}}"'), 5000, "token saved");
    await ctx.close();
  },
  async "generator creates and links a runbook"() {
    const { ctx, page } = await open("dark");
    await page.click("button[aria-label='New runbook']");
    await page.fill(".wiz input[aria-label='Runbook name']", "Lab ops");
    if ((await page.inputValue(".wiz input[aria-label='File name']")) !== "lab-ops.json") throw new Error("file name not derived from the name");
    await page.keyboard.press("Enter");
    await page.fill(".wiz input[aria-label='Category name']", "Health");
    await page.fill(".wiz input[aria-label='Label']", "Alarms");
    await page.focus(".wiz textarea[aria-label='Command']");
    await page.keyboard.insertText("rabbitmq-diagnostics check_alarms -n ");
    await page.click(".wiz button:has-text('Insert placeholder')");
    await page.fill(".ph-pop input[aria-label='Placeholder name']", "node");
    await page.fill(".ph-pop input[aria-label='Default value']", "rabbit@rmq1");
    await page.keyboard.press("Enter");
    await until(async () => (await page.inputValue(".wiz textarea[aria-label='Command']")).endsWith("{{node:rabbit@rmq1}}"), 3000, "placeholder inserted");
    await page.click(".wiz button:has-text('Next')");
    await page.check(".wiz input[aria-label='Link talk.md']");
    await page.click(".wiz button:has-text('Next')");
    await page.waitForSelector(".wiz .wiz-json");
    await page.click(".wiz button:has-text('Create')");
    await page.waitForSelector(".rb-view", { timeout: 5000 });
    const want = JSON.stringify({ name: "Lab ops", categories: [{ name: "Health", entries: [{ label: "Alarms", command: "rabbitmq-diagnostics check_alarms -n {{node:rabbit@rmq1}}", target: "bash" }] }] }, null, 2) + "\n";
    if (readFileSync(LABOPS, "utf8") !== want) throw new Error("bad runbook: " + readFileSync(LABOPS, "utf8"));
    if (!readFileSync(TALK, "utf8").startsWith("---\ntitle: E2E Talk\nrunbook: lab-ops.json\n---\n")) throw new Error("deck not linked: " + readFileSync(TALK, "utf8"));
    if ((await page.textContent(".rb-mode button.on")) !== "Visual") throw new Error("not in the Visual view");
    if ((await page.textContent(".tab.on .tab-main")) !== "lab-ops.json") throw new Error("new file is not the active tab");
    await page.click("button[aria-label='New runbook']");
    await page.fill(".wiz input[aria-label='Runbook name']", "Lab ops");
    await page.keyboard.press("Enter");
    await page.waitForSelector(".wiz :text('A file with this name already exists')", { timeout: 3000 });
    if (!(await page.$(".wiz input[aria-label='Runbook name']"))) throw new Error("left step 1");
    if (readFileSync(LABOPS, "utf8") !== want) throw new Error("existing file overwritten");
    // the file appears on disk while the wizard is open: Create gets a 409 and goes back to step 1
    await page.fill(".wiz input[aria-label='File name']", "race.json");
    await page.keyboard.press("Enter");
    await page.fill(".wiz input[aria-label='Category name']", "C");
    await page.fill(".wiz input[aria-label='Label']", "L");
    await page.fill(".wiz textarea[aria-label='Command']", "true");
    await page.click(".wiz button:has-text('Next')");
    await page.click(".wiz button:has-text('Next')");
    writeFileSync(join(root, "race.json"), "mine\n");
    await page.click(".wiz button:has-text('Create')");
    await page.waitForSelector(".wiz :text('A file with this name already exists')", { timeout: 3000 });
    if (!(await page.$(".wiz input[aria-label='Runbook name']"))) throw new Error("409 did not return to step 1");
    if (readFileSync(join(root, "race.json"), "utf8") !== "mine\n") throw new Error("409 overwrote the file");
    rmSync(join(root, "race.json"));
    await ctx.close();
  },
  async "visual toggle follows the text"() {
    writeFileSync(join(root, "plain.json"), '{"a": 1}\n');
    const { ctx, page } = await open("dark");
    await page.waitForSelector('.row-main[title="plain.json"]');
    await openFile(page, "plain.json");
    await page.waitForSelector(".cm-content");
    if (await page.$(".rb-mode")) throw new Error("toggle on a plain JSON file");
    await setText(page, '{ "categories": [] }\n');
    await page.waitForSelector(".rb-mode", { timeout: 3000 });
    if (await page.$(".rb-view")) throw new Error("switched to Visual while typing");
    await page.click(".rb-mode button:has-text('Visual')");
    await page.waitForSelector(".rb-view");
    await ctx.close();
    rmSync(join(root, "plain.json"));
  },
  async "wizard Esc asks before discarding"() {
    const { ctx, page } = await open("dark");
    await openFile(page, "sections/one.md");
    await page.click("button[aria-label='New runbook']");
    await page.waitForSelector(".wiz input[aria-label='Runbook name']:focus");
    await page.keyboard.press("Escape"); // nothing typed: closes at once
    await sleep(300);
    if (await page.$(".wiz")) throw new Error("empty wizard did not close on Esc");
    await page.click("button[aria-label='New runbook']");
    await page.fill(".wiz input[aria-label='Runbook name']", "Keep me");
    await page.keyboard.press("Escape");
    await page.waitForSelector(".wiz-ask", { timeout: 2000 });
    await page.keyboard.press("Escape"); // second Esc keeps editing
    await sleep(300);
    if (await page.$(".wiz-ask") || (await page.inputValue(".wiz input[aria-label='Runbook name']")) !== "Keep me") throw new Error("Esc did not keep editing");
    await page.keyboard.press("Escape");
    await page.click(".wiz-ask button:has-text('Discard')");
    await sleep(300);
    if (await page.$(".wiz")) throw new Error("Discard did not close");
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
    for (const theme of ["dark", "light"]) {
      const { ctx, page } = await open(theme);
      await openRunbook(page);
      await page.click(".rb-tools button:has-text('Collapse all')");
      await page.click("button[aria-label='Expand Preflight and images']");
      await page.hover(".rb-entry:nth-child(3)");
      await sleep(300);
      await page.screenshot({ path: join(out, `runbook-overview-${theme}.png`) });
      await page.click("button[aria-label='Edit Docker resources']");
      await sleep(400);
      await page.screenshot({ path: join(out, `runbook-${theme}.png`) });
      await page.click(".rb-form button:has-text('Insert placeholder')");
      await sleep(300);
      await page.screenshot({ path: join(out, `runbook-pop-${theme}.png`) });
      await ctx.close();
    }
    writeFileSync(join(root, "notes.adoc"), "= Notes\n\n== One\n\nText.\n");
    for (const theme of ["dark", "light"]) {
      const { ctx, page } = await open(theme);
      await openFile(page, "sections/one.md");
      const shot = async (n) => { await sleep(250); await page.screenshot({ path: join(out, `wizard-${n}-${theme}.png`) }); };
      await page.click("button[aria-label='New runbook']");
      await page.click(".wiz button:has-text('Next')");
      await shot("1-errors");
      await page.fill(".wiz input[aria-label='Runbook name']", "Lab ops");
      await page.fill(".wiz input[aria-label='Description']", "Checks for the three-node RabbitMQ lab");
      await shot("1-runbook");
      await page.keyboard.press("Enter");
      await page.fill(".wiz input[aria-label='Category name']", "Health");
      await page.fill(".wiz input[aria-label='Label']", "Alarms");
      await page.fill(".wiz input[aria-label='Entry description']", "Memory and disk alarms on one node");
      await page.fill(".wiz textarea[aria-label='Command']", "rabbitmq-diagnostics check_alarms -n {{node:rabbit@rmq1}}");
      await page.click(".wiz button:has-text('Add another')");
      await page.fill(".wiz .wiz-entry >> nth=1 >> input[aria-label='Label']", "Queues");
      await page.click(".wiz .wiz-entry >> nth=1 >> button:has-text('Insert placeholder')");
      await shot("2-commands");
      await page.keyboard.press("Escape");
      await page.click(".wiz button:has-text('Next')");
      await shot("2-errors");
      await page.fill(".wiz .wiz-entry >> nth=1 >> textarea[aria-label='Command']", "rabbitmqctl -n {{node}} list_queues name messages");
      await page.click(".wiz button:has-text('Next')");
      await page.check(".wiz input[aria-label='Link talk.md']");
      await page.check(".wiz input[aria-label='Link rbdeck.md']");
      if (!(await page.isDisabled(".wiz input[aria-label='Link notes.adoc']"))) throw new Error("AsciiDoc deck can be linked");
      await shot("3-link");
      await page.click(".wiz button:has-text('Next')");
      await shot("4-review");
      await page.keyboard.press("Escape");
      await page.waitForSelector(".wiz-ask");
      await shot("5-discard");
      await ctx.close();
    }
    rmSync(join(root, "notes.adoc"));
  },
};

let failed = 0;
for (const [name, fn] of Object.entries(tests)) {
  if (process.env.ONLY && !name.includes(process.env.ONLY)) continue; // ONLY=generator npm run e2e
  reset();
  try { await fn(); console.log("ok   " + name); }
  catch (e) { failed++; console.log("FAIL " + name + "\n     " + String(e.message).split("\n")[0]); }
}
await browser.close();
srv.kill();
rmSync(root, { recursive: true, force: true });
console.log(failed ? `${failed} failed` : "all passed");
process.exit(failed ? 1 : 0);
