// The play terminal drawer as a filmed demo drives it: long command lines
// stay whole after the drawer is minimised and brought back (the shell's
// idea of the width is xterm's: stty size = the tab's cols, every wrapped
// row read back in order), and the drawer's opacity (siPlay.terminal.opacity,
// the bar's control, si.conf terminal.opacity) shows the slide through the
// background while the text stays opaque. si play on a temp folder with /bin/sh
// and si's clean prompt (as si demo films it), headless. Needs ../../bin/si
// (make build); SI_BIN overrides it. Run: npm run e2e:drawer
import { chromium } from "playwright";
import { spawn } from "node:child_process";
import { mkdtempSync, writeFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

const si = process.env.SI_BIN || new URL("../../../bin/si", import.meta.url).pathname;
const dir = mkdtempSync(join(tmpdir(), "si-drawer-"));
writeFileSync(join(dir, "talk.md"), "# Queues\n\nA queue holds messages until a consumer takes them.\n\n---\n\n## Two\n\nMore.\n");
writeFileSync(join(dir, "si.conf"), "terminal.prompt: clean\nterminal.opacity: 90%\n");

const port = 7391 + Math.floor(Math.random() * 400);
const srv = spawn(si, ["play", join(dir, "talk.md"), "--addr", "127.0.0.1:" + port], { stdio: "ignore", env: { ...process.env, SHELL: "/bin/sh" } });
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const base = "http://127.0.0.1:" + port + "/talk.html";
for (let i = 0; i < 50; i++) { try { await fetch(base); break; } catch { await sleep(100); } }
const browser = await chromium.launch();
const page = await (await browser.newContext({ viewport: { width: 1600, height: 900 } })).newPage();
const errors = []; page.on("pageerror", (e) => errors.push(e.message));
let failed = 0; const check = (ok, what) => { console.log((ok ? "ok   " : "FAIL ") + what); if (!ok) failed++; };
const out = (n) => ev((n) => window.siPlay.terminal.output(n), n);
// si play reloads the page when it sees a file change in the deck folder (a
// shell's own files can do that); a call cut off by such a reload waits for
// the reloaded drawer and runs again, so the checks test the drawer, not timing
const ev = async (fn, arg) => {
  for (let i = 0; ; i++) {
    try { return await page.evaluate(fn, arg); }
    catch (e) {
      if (i >= 3 || !/context was destroyed|navigat/i.test(String(e))) throw e;
      await page.waitForFunction(() => window.siTerm && window.siPlay, null, { timeout: 20000 }).catch(() => {});
    }
  }
};
const until = async (fn, ms = 15000) => { for (let t = Date.now(); Date.now() - t < ms;) { if (await fn()) return true; await sleep(100); } return false; };

const put = `curl -s -u acemq:acemq-lab -X PUT -H 'content-type: application/json' -d '{"durable":true,"arguments":{"x-queue-type":"quorum"}}' -w 'created: %{http_code}\\n' http://localhost:15801/api/queues/%2F/ops.smoke`;
const pub = `curl -s -u acemq:acemq-lab -X POST -H 'content-type: application/json' -d '{"properties":{},"routing_key":"ops.smoke","payload":"hello from the terminal","payload_encoding":"string"}' http://localhost:15801/api/exchanges/%2F/amq.default/publish`;
// the commands echo themselves (no broker needed): the line typed is the line checked
const line = (c, tag) => `echo ${tag} ${c.replace(/'/g, "")}`;
try {
  await page.goto(base);
  await page.waitForFunction(() => window.siPlay);
  // the founder's sequence: a short command, the drawer minimised, a browser
  // window over the slides, the window minimised, back to the same tab, long lines
  await ev(() => window.siPlay.terminal.open("Ops"));
  await ev(() => window.siPlay.terminal.run("Ops", "echo first", { typing: 20 }));
  check(await until(async () => /\nfirst/.test(await out("Ops"))), "a short command runs");
  await ev(() => window.siPlay.terminal.state("min"));
  await ev((u) => window.siPlay.browser.open(u, { x: 560, y: 40, w: 1010, h: 640 }), base + "#2");
  await sleep(400);
  await ev((u) => window.siPlay.browser.min(u), base + "#2");
  await ev(() => window.siPlay.terminal.open("Ops"));

  const size = await ev(() => window.siTerm.api.size ? window.siTerm.api.size("Ops") : null);
  await ev(() => window.siPlay.terminal.run("Ops", "stty size", { typing: 10 }));
  await until(async () => /stty size\n\d+ \d+/.test(await out("Ops")));
  const m = /stty size\n(\d+) (\d+)/.exec(await out("Ops"));
  check(size && size.cols > 150 && m && +m[2] === size.cols && +m[1] === size.rows, `the shell's size is the tab's (stty ${m && m[1] + "x" + m[2]}, xterm ${size && size.rows + "x" + size.cols})`);

  await ev((c) => window.siPlay.terminal.run("Ops", c, { typing: 4 }), line(put, "PUT"));
  await ev((c) => window.siPlay.terminal.run("Ops", c), line(pub, "POST"));
  check(await until(async () => (await out("Ops")).split("\nPOST ").length > 1), "both long commands ran");
  const text = await out("Ops");
  for (const [tag, c] of [["PUT", put], ["POST", pub]]) {
    const l = line(c, tag);
    check(text.includes("$ " + l + "\n"), `${tag}: the long line reads back whole after the prompt (${l.length} chars)`);
  }
  // no row of the screen starts in the middle of a word of a command (the overwrite left "ttp://…")
  const rows = await ev(() => { const b = document.querySelector(".si-term-pane.is-active .xterm-rows"); return b ? Array.from(b.children, (r) => r.textContent) : []; });
  check(!rows.some((r) => /^(ttp:|ad_encoding)/.test(r)), "no row starts with the tail of an overwritten line");
  await page.screenshot({ path: join(process.env.SI_SHOTS || dir, "drawer-long.png") });

  // opacity: si.conf's 90 %, then the API; the text stays opaque
  const look = () => ev(() => { const r = document.querySelector(".si-term"), x = document.querySelector(".si-term-pane.is-active .xterm-rows");
    return { bg: getComputedStyle(r).backgroundColor, op: getComputedStyle(r).opacity, text: x && getComputedStyle(x).color, pct: r.dataset.opacity }; });
  let l = await look();
  check(/rgba\(11, 14, 20, 0\.9\)/.test(l.bg) && l.op === "1", `si.conf terminal.opacity: 90% (${l.bg})`);
  await ev(() => window.siPlay.terminal.opacity(60));
  l = await look();
  check(/rgba\(11, 14, 20, 0\.6\)/.test(l.bg) && l.op === "1" && l.text === "rgb(230, 230, 230)", `siPlay.terminal.opacity(60): background ${l.bg}, text ${l.text}`);
  await ev(() => window.siPlay.terminal.opacity(10));
  check((await look()).pct === "40", "opacity is kept at 40 % or more");
  await ev(() => window.siPlay.terminal.opacity(70));
  await page.screenshot({ path: join(process.env.SI_SHOTS || dir, "drawer-opacity.png") });
  // a live reload (si.conf changed above) may land at the same moment and abort this one: then reload again
  await page.reload().catch(() => page.reload()); await page.waitForFunction(() => window.siTerm);
  check(await until(async () => (await look()).pct === "70"), "the opacity is remembered");
  check(errors.length === 0, "no page errors" + (errors.length ? ": " + errors.join("; ") : ""));
} catch (e) {
  console.log("FAIL " + e.message); failed++;
} finally {
  await browser.close();
  srv.kill();
  rmSync(dir, { recursive: true, force: true });
}
if (failed) { console.log(failed + " failed"); process.exit(1); }
console.log("drawer e2e: all passed");
