// The play terminal's Viewer: .conf colours (read-only and editor), New file,
// a taken name, Rename, Delete (file and folder), Follow (appended lines; a
// file over 1 MB numbered from its real first line), and the tree's open
// folders and scroll brought back by a reload. si play on a temp folder,
// headless. Needs ../../bin/si (make build). Run: npm run e2e:viewer
import { chromium } from "playwright";
import { spawn } from "node:child_process";
import { mkdtempSync, writeFileSync, mkdirSync, existsSync, appendFileSync, readFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

const si = new URL("../../../bin/si", import.meta.url).pathname;
const dir = mkdtempSync(join(tmpdir(), "si-viewer-"));
const mod = process.platform === "darwin" ? "Meta" : "Control";
writeFileSync(join(dir, "talk.md"), "# Talk\n\nHello.\n");
mkdirSync(join(dir, "conf"));
writeFileSync(join(dir, "conf", "rabbitmq.conf"), `# RabbitMQ node settings
listeners.tcp.default = 5672
management.tcp.port = 15672
heartbeat = 30s
vm_memory_high_watermark.absolute = 512MB
loopback_users.guest = false
log.console = true
ssl_options.cacertfile = /etc/rabbitmq/ca.pem
cluster_formation.peer_discovery_backend = classic_config
default_vhost = "/orders"
management.path_prefix = https://broker.example.com/admin

[legacy]
mode: on ; old style
`);
// a deep folder with enough files to scroll the tree
mkdirSync(join(dir, "src", "orders", "events"), { recursive: true });
for (let i = 0; i < 40; i++) writeFileSync(join(dir, "src", "orders", "events", `event${String(i).padStart(2, "0")}.go`), "package events\n");
let log = ""; for (let i = 1; i <= 80; i++) log += `2026-11-13T10:00:${String(i % 60).padStart(2, "0")} INFO line ${i}\n`;
writeFileSync(join(dir, "app.log"), log);
let big = ""; for (let i = 1; big.length < (1 << 20) + 20000; i++) big += `big line ${i} ${"x".repeat(30)}\n`;
writeFileSync(join(dir, "big.log"), big);

const port = 7391 + Math.floor(Math.random() * 400);
const srv = spawn(si, ["play", join(dir, "talk.md"), "--addr", "127.0.0.1:" + port], { stdio: "ignore" });
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const base = "http://127.0.0.1:" + port + "/talk.html";
for (let i = 0; i < 50; i++) { try { await fetch(base); break; } catch { await sleep(100); } }
const browser = await chromium.launch();
const page = await (await browser.newContext({ viewport: { width: 1440, height: 900 } })).newPage();
const errors = []; page.on("pageerror", (e) => errors.push(e.message));
let failed = 0; const check = (ok, what) => { console.log((ok ? "ok   " : "FAIL ") + what); if (!ok) failed++; };
let appender;
const P = ".si-term-pane.is-active";
const row = (name) => page.locator(P + " .si-fs-row .nm").getByText(name, { exact: true });
try {
  await page.goto(base);
  await sleep(500);
  await page.keyboard.press("`");
  await page.waitForFunction(() => window.siTerm);
  await page.evaluate(() => window.siTerm.viewer());

  // open folders and the tree's scroll come back after a reload
  await row("src/").waitFor();
  await row("src/").click();
  await row("orders/").click();
  await row("events/").click();
  await row("event39.go").waitFor();
  const top = await page.evaluate(() => { const t = document.querySelector(".si-term-pane.is-active .si-fs-tree"); t.scrollTop = t.scrollHeight; return t.scrollTop; });
  check(top > 100, "the tree scrolls (" + top + ")");
  await page.reload();
  await page.waitForFunction(() => document.querySelector(".si-term-pane.is-active .si-fs-tree"));
  await row("event39.go").waitFor({ timeout: 5000 }).catch(() => {});
  check(await row("event39.go").isVisible(), "a reload brings src/, orders/ and events/ back open");
  await page.waitForFunction((t) => Math.abs(document.querySelector(".si-term-pane.is-active .si-fs-tree").scrollTop - t) < 4, top, { timeout: 3000 }).catch(() => {});
  const top2 = await page.evaluate(() => document.querySelector(".si-term-pane.is-active .si-fs-tree").scrollTop);
  check(Math.abs(top2 - top) < 4, "and the tree where it was scrolled (" + top + " -> " + top2 + ")");
  await row("src/").click(); // fold it again for the rest

  await row("conf/").click();
  await row("rabbitmq.conf").click();
  await page.waitForSelector(P + " .si-fs-code");
  const kinds = await page.evaluate(() => [...document.querySelectorAll(".si-term-pane.is-active .si-fs-code span[class^=si-tk]")].map((s) => s.className + ":" + s.textContent));
  for (const k of ["si-tk-v:listeners.tcp.default", "si-tk-n:30s", "si-tk-n:512MB", "si-tk-k:false", "si-tk-y:/etc/rabbitmq/ca.pem", "si-tk-m:[legacy]", "si-tk-c:# RabbitMQ node settings", 'si-tk-s:"/orders"'])
    check(kinds.includes(k), "read-only colours " + k);
  await page.click(P + ' [data-f="edit"]');
  await page.waitForSelector(P + " .cm-content");
  await sleep(300);
  const cm = await page.evaluate(() => {
    const c = (txt) => { for (const s of document.querySelectorAll(".si-term-pane.is-active .cm-line span")) if (s.textContent === txt) return getComputedStyle(s).color; return ""; };
    return { key: c("listeners.tcp.default"), num: c("30s"), size: c("512MB"), bool: c("false"), path: c("/etc/rabbitmq/ca.pem"), sec: c("[legacy]"), str: c('"/orders"'), url: c("https://broker.example.com/admin") };
  });
  check(cm.key && cm.num && cm.bool && cm.path && cm.sec && cm.str && cm.url && new Set(Object.values(cm)).size >= 5, "editor colours conf kinds " + JSON.stringify(cm));
  await page.click(P + ' [data-f="done"]');

  // New file at the top: right-click empty tree space
  const tree = page.locator(P + " .si-fs-tree");
  const tb = await tree.boundingBox();
  await page.mouse.click(tb.x + 40, tb.y + tb.height - 20, { button: "right" });
  await page.click('.si-fs-menu button:has-text("New file")');
  await page.keyboard.type("notes.md"); await page.keyboard.press("Enter");
  await page.waitForSelector(P + " .cm-content");
  check(existsSync(join(dir, "notes.md")), "New file made notes.md and opened it in the editor");
  await page.keyboard.type("# Notes\n");
  await page.keyboard.press(mod + "+s"); await sleep(400);
  check(readFileSync(join(dir, "notes.md"), "utf8").startsWith("# Notes"), "typed and saved");
  await page.mouse.click(tb.x + 40, tb.y + tb.height - 20, { button: "right" });
  await page.click('.si-fs-menu button:has-text("New file")');
  await page.keyboard.type("talk.md"); await page.keyboard.press("Enter"); await sleep(300);
  check((await page.locator(P + " .si-fs-ask").innerText()).includes("taken"), "a taken name is refused in the panel");
  await page.click(P + ' .si-fs-ask button:has-text("OK")'); await page.keyboard.press("Escape");
  await row("notes.md").click({ button: "right" });
  await page.click('.si-fs-menu button:has-text("Rename")');
  await page.keyboard.press(mod + "+a"); await page.keyboard.type("plan.md"); await page.keyboard.press("Enter"); await sleep(400);
  check(existsSync(join(dir, "plan.md")) && !existsSync(join(dir, "notes.md")), "renamed notes.md to plan.md");
  check((await page.locator(".si-term-tab.is-active").innerText()).includes("plan.md"), "the open tab follows the rename");
  await row("plan.md").click();
  await page.click(P + ' [data-f="delete"]'); await sleep(300);
  check((await page.locator(P + " .si-fs-ask").innerText()).includes("plan.md"), "delete asks, naming the file");
  await page.click(P + ' .si-fs-ask button:has-text("Delete")'); await sleep(400);
  check(!existsSync(join(dir, "plan.md")), "deleted plan.md");
  await row("conf/").click({ button: "right" });
  await page.click('.si-fs-menu button:has-text("Delete")'); await sleep(300);
  check((await page.locator(P + " .si-fs-ask").innerText()).includes("1 item"), "folder delete counts what goes");
  await page.click(P + ' .si-fs-ask button:has-text("Delete")'); await sleep(400);
  check(!existsSync(join(dir, "conf")), "deleted conf/");

  // Follow
  await row("app.log").click();
  await page.waitForSelector(P + " .si-fs-code");
  await page.click(P + ' [data-f="follow"]');
  let n = 80;
  appender = setInterval(() => { n++; appendFileSync(join(dir, "app.log"), `2026-11-13T10:01:00 WARN appended ${n}\n`); }, 300);
  await sleep(2500);
  const f = await page.evaluate(() => { const b = document.querySelector(".si-term-pane.is-active .si-fs-body"); return { text: b.innerText, end: b.scrollHeight - b.scrollTop - b.clientHeight < 24 }; });
  check(f.text.includes("appended " + (n - 3)) && f.end, "Follow shows appended lines at the end (n=" + n + ")");
  clearInterval(appender);

  // Follow on a file over 1 MB: its last MB, numbered from the real line
  await row("big.log").click();
  await page.waitForFunction(() => /1 MB/.test(document.querySelector(".si-term-pane.is-active .si-fs-vh .in")?.textContent || ""));
  await page.waitForFunction(() => !document.querySelector('.si-term-pane.is-active [data-f="follow"]').hidden);
  await page.waitForFunction(() => document.querySelector('.si-term-pane.is-active [data-f="follow"]').classList.contains("is-on"));
  await page.waitForFunction(() => /last 1 MB/.test(document.querySelector(".si-term-pane.is-active .si-fs-vh .in").textContent), null, { timeout: 5000 }).catch(() => {});
  const num = await page.evaluate(() => {
    const pre = document.querySelector(".si-term-pane.is-active .si-fs-code");
    const first = pre.querySelector(".ln").textContent;
    const ln = getComputedStyle(pre).counterReset;
    return { first: +/big line (\d+)/.exec(first)?.[1], reset: +/ln (\d+)/.exec(ln)?.[1], info: document.querySelector(".si-term-pane.is-active .si-fs-vh .in").textContent };
  });
  check(num.first > 100 && num.reset === num.first - 1, "a big file's tail is numbered from its real line " + JSON.stringify(num));
  check(num.info.includes("lines " + num.first + "–"), "the header says which lines: " + num.info);
  check(errors.length === 0, "no page errors " + errors.join("; "));
} catch (e) { console.log("FAIL " + e.message); failed++; await page.screenshot({ path: join(dir, "fail.png") }); }
finally {
  clearInterval(appender); await browser.close(); srv.kill();
  if (!failed && !process.env.KEEP) rmSync(dir, { recursive: true, force: true });
  else console.log("files in " + dir);
}
process.exit(failed ? 1 : 0);
