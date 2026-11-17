// The play workbench: quick open (⌘P), search in files (⌘⇧F) streamed with
// Stop, a result opened at its line, spotlight (view and editor), ⌘= ⌘- ⌘0
// sizing the tab (the key cancelled, the page not zoomed), split view up to
// four panes (dividers, a dragged tab, reload). si play on a temp folder,
// headless. Needs ../../bin/si (make build). Run: npm run e2e:workbench
import { chromium } from "playwright";
import { spawn } from "node:child_process";
import { mkdtempSync, writeFileSync, mkdirSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { fileURLToPath } from "node:url";
const si = process.env.SI_BIN || fileURLToPath(new URL("../../../bin/si", import.meta.url)); // SI_BIN: bin\\si.exe on Windows
const dir = mkdtempSync(join(tmpdir(), "si-wb-"));
const out = mkdtempSync(join(tmpdir(), "si-wb-shots-")) + "/"; // not in the deck's folder: a file there live-reloads the page
writeFileSync(join(dir, "talk.md"), "# Talk\n\nHello.\n");
mkdirSync(join(dir, "src", "orders"), { recursive: true });
mkdirSync(join(dir, "node_modules", "x"), { recursive: true });
let code = ""; for (let i = 1; i <= 120; i++) code += i === 77 ? "func Checkout(o Order) error { return publish(o) }\n" : `// line ${i} of the order service\n`;
writeFileSync(join(dir, "src", "orders", "checkout.go"), code);
writeFileSync(join(dir, "src", "orders", "refund.go"), "package orders\n\nfunc Refund() {}\n");
writeFileSync(join(dir, "node_modules", "x", "checkout.js"), "Checkout\n");
const port = 7391 + Math.floor(Math.random() * 400);
const srv = spawn(si, ["play", join(dir, "talk.md"), "--addr", "127.0.0.1:" + port], { stdio: "ignore" });
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const base = "http://127.0.0.1:" + port + "/talk.html";
for (let i = 0; i < 50; i++) { try { await fetch(base); break; } catch { await sleep(100); } }
const browser = await chromium.launch();
const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } });
const page = await ctx.newPage();
const errors = []; page.on("pageerror", (e) => errors.push(e.message));
let failed = 0; const check = (ok, what) => { console.log((ok ? "ok   " : "FAIL ") + what); if (!ok) failed++; };
const mod = process.platform === "darwin" ? "Meta" : "Control";
try {
  await page.goto(base);
  await sleep(500);
  await page.keyboard.press("`");
  await page.waitForFunction(() => window.siTerm);
  await page.waitForSelector(".si-term-pane.is-active .xterm");
  await page.evaluate(() => window.siTerm.viewer());
  await page.waitForSelector(".si-term-pane.is-active .si-fs-row");

  // quick open from the Viewer
  await page.keyboard.press(mod + "+p");
  await page.waitForSelector(".si-qo input");
  await page.keyboard.type("chkgo");
  await page.waitForFunction(() => document.querySelectorAll(".si-qo-row").length === 1);
  const rows = await page.locator(".si-qo-row").allTextContents();
  check(rows.length === 1, "quick open: one hit for chkgo, node_modules skipped (" + rows.join(" | ") + ")");
  await page.screenshot({ path: out + "wb-quickopen.png" });
  await page.keyboard.press("Enter");
  await page.waitForFunction(() => document.querySelector(".si-term-pane.is-active .si-fs-vh .nm")?.textContent === "checkout.go");
  check((await page.locator(".si-term-tab").count()) === 2, "Enter opened in the active Viewer (no new tab)");

  // search in files -> open at line
  await page.keyboard.press(mod + "+Shift+f");
  await page.waitForSelector(".si-sr.is-open input:focus");
  await page.keyboard.type("checkout(");
  await page.waitForSelector(".si-sr-line");
  const sum = await page.textContent(".si-sr-sum");
  check(/^1 match in 1 file/.test(sum), "search: " + sum);
  await page.click(".si-sr-line");
  await page.waitForSelector(".si-term-pane.is-active .ln.hit");
  const hit = await page.evaluate(() => { const p = document.querySelector(".si-term-pane.is-active .si-fs-code"); const h = p.querySelector(".ln.hit"); const b = p.parentElement; const r = h.getBoundingClientRect(), br = b.getBoundingClientRect(); return { n: [...p.children].indexOf(h) + 1, vis: r.top >= br.top && r.bottom <= br.bottom }; });
  check(hit.n === 77 && hit.vis, "result opened at line 77, marked and in view " + JSON.stringify(hit));
  await page.click('.si-sr-t[data-o="case"]'); await sleep(400);
  check((await page.textContent(".si-sr-sum")).startsWith("0 matches"), "match case: checkout( has no hit");
  await page.click('.si-sr-t[data-o="case"]');
  await page.click('.si-sr-t[data-o="re"]');
  await page.fill(".si-sr input", "func \\w+\\("); await page.keyboard.press("Enter"); await sleep(400);
  check((await page.textContent(".si-sr-sum")).startsWith("2 matches in 2 files"), "regex: " + (await page.textContent(".si-sr-sum")));
  await page.screenshot({ path: out + "wb-search.png" });
  // the results stream: a server that has sent one match so far shows it at
  // once, Stop cancels the request (a fetch that answers slowly, in the page)
  await page.click('.si-sr-t[data-o="re"]');
  await page.evaluate(() => {
    const real = window.fetch; window.__si_real = real;
    window.fetch = function (u, o) {
      if (!/grep\?stream=1.*q=slowpoke/.test(String(u))) return real.apply(this, arguments);
      let ctl; const enc = new TextEncoder();
      const rs = new ReadableStream({ start(c) { ctl = c; c.enqueue(enc.encode(JSON.stringify({ dir: "/x" }) + "\n" + JSON.stringify({ path: "/x/a.go", rel: "a.go", line: 3, text: "a slowpoke", hits: [[2, 10]] }) + "\n")); } });
      o.signal.addEventListener("abort", () => { window.__si_aborted = true; try { ctl.error(new DOMException("stopped", "AbortError")); } catch (e) {} });
      return Promise.resolve(new Response(rs));
    };
  });
  await page.fill(".si-sr input", "slowpoke"); await page.keyboard.press("Enter");
  await page.waitForSelector(".si-sr-line");
  const live = await page.evaluate(() => ({ sum: document.querySelector(".si-sr-sum").textContent, rows: document.querySelectorAll(".si-sr-line").length, stop: !document.querySelector(".si-sr-stop").hidden }));
  check(live.rows === 1 && /^Searching… 1 match in 1 file/.test(live.sum) && live.stop, "a match shows while the search still runs, with Stop " + JSON.stringify(live));
  await page.click(".si-sr-stop");
  const stopped = await page.evaluate(() => ({ aborted: !!window.__si_aborted, stop: !document.querySelector(".si-sr-stop").hidden, sum: document.querySelector(".si-sr-sum").textContent, rows: document.querySelectorAll(".si-sr-line").length }));
  check(stopped.aborted && !stopped.stop && /stopped/.test(stopped.sum) && stopped.rows === 1, "Stop cancels the request and keeps what came " + JSON.stringify(stopped));
  await page.evaluate(() => { window.fetch = window.__si_real; });
  await page.fill(".si-sr input", "checkout("); await page.keyboard.press("Enter");
  await page.waitForFunction(() => /^1 match in 1 file/.test(document.querySelector(".si-sr-sum").textContent));
  check(await page.locator(".si-sr-stop").isHidden(), "Stop hides when the search ends");

  // spotlight: line numbers 75, shift 78
  const gut = async (n) => { const b = await page.locator(".si-term-pane.is-active .si-fs-code .ln").nth(n - 1).boundingBox(); return { x: b.x + 12, y: b.y + b.height / 2 }; };
  let g = await gut(75); await page.mouse.click(g.x, g.y);
  g = await gut(78); await page.keyboard.down("Shift"); await page.mouse.click(g.x, g.y); await page.keyboard.up("Shift");
  const on = await page.evaluate(() => [...document.querySelectorAll(".si-term-pane.is-active .ln.on")].map((x) => [...x.parentElement.children].indexOf(x) + 1));
  check(JSON.stringify(on) === "[75,76,77,78]", "spotlight 75-78: " + JSON.stringify(on));
  await sleep(400);
  const dim = await page.evaluate(() => getComputedStyle(document.querySelectorAll(".si-term-pane.is-active .ln")[10]).opacity);
  check(+dim < 0.5, "the rest is dimmed (" + dim + ")");
  await page.screenshot({ path: out + "wb-spotlight.png" });
  await page.keyboard.press("Escape");
  check((await page.locator(".si-term-pane.is-active .si-fs-code.is-spot").count()) === 0, "Esc clears the spotlight");

  // editor spotlight
  await page.click('.si-term-pane.is-active [data-f="edit"]');
  await page.waitForSelector(".si-term-pane.is-active .cm-content");
  await page.locator(".si-term-pane.is-active .cm-line").nth(2).click();
  await page.keyboard.down("Shift"); await page.keyboard.press("ArrowDown"); await page.keyboard.up("Shift");
  await page.click('.si-term-pane.is-active [data-f="spot"]');
  check((await page.locator(".si-term-pane.is-active .cm-si-spotting .cm-si-spot").count()) === 2, "editor: Spotlight selection lights 2 lines");
  await page.screenshot({ path: out + "wb-editor-spot.png" });
  await page.keyboard.press("Escape");
  check((await page.locator(".si-term-pane.is-active .cm-si-spotting").count()) === 0, "editor: Esc clears");
  await page.click('.si-term-pane.is-active [data-f="done"]');

  // ⌘= ⌘- ⌘0 (Ctrl+ elsewhere) size the tab, cancelled before the browser
  // zooms the page: the page's zoom (devicePixelRatio, visualViewport) stays
  await page.evaluate(() => { window.__si_keys = []; addEventListener("keydown", (e) => window.__si_keys.push(e), true); });
  const zoomNow = () => page.evaluate(() => ({ pct: document.querySelector(".si-term-zoom .pct").textContent, dpr: devicePixelRatio, scale: visualViewport.scale, cancelled: window.__si_keys.map((e) => e.defaultPrevented) }));
  await page.locator(".si-term-pane.is-active .si-fs-body").click();
  const z0 = await zoomNow();
  await page.keyboard.press(mod + "+Equal");
  const z1 = await zoomNow();
  check(z1.pct === "110%" && z1.cancelled.at(-1) === true, mod + "+= sizes the tab and is cancelled " + JSON.stringify(z1));
  check(z1.dpr === z0.dpr && z1.scale === z0.scale, "the page is not zoomed (devicePixelRatio " + z0.dpr + " -> " + z1.dpr + ", scale " + z0.scale + " -> " + z1.scale + ")");
  await page.keyboard.press(mod + "+Minus");
  check((await zoomNow()).pct === "100%", mod + "+- back to 100%");
  await page.keyboard.press(mod + "+Equal"); await page.keyboard.press(mod + "+Equal");
  await page.locator('[data-act="zout"]').focus(); // the drawer's own bar too
  await page.keyboard.press(mod + "+Digit0");
  const z2 = await zoomNow();
  check(z2.pct === "100%" && z2.cancelled.at(-1) === true && z2.dpr === z0.dpr, mod + "+0 from the drawer's bar resets the tab, not the page " + JSON.stringify(z2));
  await page.click('[data-act="zin"]');
  check((await zoomNow()).pct === "110%", "the + button sizes the tab");
  await page.click(".si-term-zoom .pct");

  // split right: the Viewer beside the shell
  await page.click('[data-act="split"]');
  await sleep(300);
  const panes = await page.evaluate(() => [...document.querySelectorAll(".si-term-pane.is-active")].map((p) => { const r = p.getBoundingClientRect(); return [Math.round(r.left), Math.round(r.width)]; }));
  check(panes.length === 2 && Math.abs(panes[0][1] - panes[1][1]) < 4, "split: two panes side by side " + JSON.stringify(panes));
  const cols0 = await page.evaluate(() => document.querySelector(".si-term-pane.is-active .xterm-screen").getBoundingClientRect().width);
  const d = await page.locator(".si-term-div.v").boundingBox();
  await page.mouse.move(d.x + d.width / 2, d.y + 50); await page.mouse.down();
  await page.mouse.move(d.x - 300, d.y + 50, { steps: 6 }); await page.mouse.up(); await sleep(300);
  const W = () => page.evaluate(() => [...document.querySelectorAll(".si-term-pane.is-active")].map((p) => p.getBoundingClientRect()).sort((a, b) => a.left - b.left).map((r) => Math.round(r.width)));
  const ratio = await W();
  check(ratio[0] < ratio[1] - 400, "divider drag resized the panes " + JSON.stringify(ratio));
  const xs = await page.evaluate(() => { const s = [...document.querySelectorAll(".si-term-pane.is-active")].find((p) => p.querySelector(".xterm")); return s.querySelector(".xterm-screen").getBoundingClientRect().width; });
  check(Math.abs(xs - cols0) > 100, "the shell refit to its pane (" + Math.round(cols0) + " -> " + Math.round(xs) + ")");
  const shellPane = page.locator(".si-term-pane.is-active:has(.xterm)");
  await shellPane.click(); await page.keyboard.type("echo split-ok\n");
  await page.waitForFunction(() => /split-ok/.test([...document.querySelectorAll(".si-term-pane.is-active .xterm-rows")].map((r) => r.textContent).join(" ")), null, { timeout: 8000 }).then(() => check(true, "typed into the split shell"), () => check(false, "typed into the split shell"));
  await page.screenshot({ path: out + "wb-split.png" });
  // a third tab dragged onto the left pane
  await page.click('[data-act="new"]'); await sleep(500);
  const third = page.locator(".si-term-tab").nth(2);
  const left = await page.locator(".si-term-pane.is-active").first().boundingBox();
  await third.dragTo(page.locator(".si-term-main"), { targetPosition: { x: 40, y: 120 } });
  await sleep(300);
  const leftTab = await page.evaluate(() => document.querySelectorAll(".si-term-tab")[2].className);
  check(/is-active|is-shown/.test(leftTab), "dragged tab shown in a pane (" + leftTab + ")");
  void left;
  // reload keeps the split
  const before = await W();
  await page.reload(); await sleep(1500);
  await page.waitForFunction(() => document.querySelectorAll(".si-term-pane.is-active").length === 2, null, { timeout: 5000 }).catch(() => {});
  const after = await W();
  check(after.length === 2 && Math.abs(after[0] / (after[0] + after[1]) - before[0] / (before[0] + before[1])) < 0.01, "reload keeps split and ratio " + JSON.stringify(before) + " " + JSON.stringify(after));
  await sleep(1500);
  const txt = await page.evaluate(() => [...document.querySelectorAll(".si-term-pane.is-active .xterm-rows")].map((r) => r.textContent.trim().length));
  await page.locator(".si-term-tab", { hasText: "shell 1" }).click(); await sleep(800);
  check((await page.locator(".si-term-pane.is-active").count()) === 2, "clicking a tab keeps the split");
  const all = await page.evaluate(() => [...document.querySelectorAll(".si-term-pane .xterm-rows")].map((r) => r.textContent).join(" "));
  check(/split-ok/.test(all), "shell output replayed after reload (" + JSON.stringify(txt) + ")");
  await page.screenshot({ path: out + "wb-split-reload.png" });
  // close the split
  await page.click('[data-act="split"]'); await sleep(200);
  check((await page.locator(".si-term-pane.is-active").count()) === 1, "split button closes the split");
  // up to four panes: Shift-click ◫ adds one; a 2×2 grid, both dividers, reload
  while ((await page.locator(".si-term-tab").count()) < 4) { await page.click('[data-act="new"]'); await sleep(300); }
  await page.click('[data-act="split"]'); await sleep(200);
  await page.click('[data-act="split"]', { modifiers: ["Shift"] }); await sleep(200);
  const three = await page.locator(".si-term-pane.is-active").count();
  await page.click('[data-act="split"]', { modifiers: ["Shift"] }); await sleep(300);
  await page.click('[data-act="split"]', { modifiers: ["Shift"] }); await sleep(200); // a fifth is refused
  const boxes = () => page.evaluate(() => [...document.querySelectorAll(".si-term-pane.is-active")].map((p) => { const r = p.getBoundingClientRect(); return [Math.round(r.left), Math.round(r.top), Math.round(r.width), Math.round(r.height)]; }));
  const g4 = await boxes();
  check(three === 3 && g4.length === 4 && new Set(g4.map((b) => b[0])).size === 2 && new Set(g4.map((b) => b[1])).size === 2, "three panes, then four in a 2×2 grid, no fifth " + JSON.stringify(g4));
  const dh = await page.locator(".si-term-div.h").boundingBox();
  await page.mouse.move(dh.x + 60, dh.y + dh.height / 2); await page.mouse.down();
  await page.mouse.move(dh.x + 60, dh.y - 80, { steps: 5 }); await page.mouse.up(); await sleep(300);
  const hs = (bs) => { const tops = [...new Set(bs.map((b) => b[1]))].sort((a, b) => a - b); const h = (t) => bs.find((b) => b[1] === t)[3]; return [h(tops[0]), h(tops[1])]; };
  const rowsH = hs(await boxes());
  check(rowsH[0] < rowsH[1] - 100, "the horizontal divider resizes the rows " + JSON.stringify(rowsH));
  await page.reload(); await sleep(1200);
  await page.waitForFunction(() => document.querySelectorAll(".si-term-pane.is-active").length === 4, null, { timeout: 5000 }).catch(() => {});
  const g4b = await boxes();
  check(g4b.length === 4 && Math.abs(hs(g4b)[0] - rowsH[0]) < 4, "reload keeps four panes and their sizes " + JSON.stringify(g4b));
  await page.screenshot({ path: out + "wb-four.png" });
  const inPane = page.locator(".si-term-tab.is-shown").first();
  await inPane.click({ button: "right" });
  await page.click('.si-fs-menu button:has-text("Close this pane")'); await sleep(300);
  check((await page.locator(".si-term-pane.is-active").count()) === 3, "Close this pane leaves three");
  await page.click('[data-act="split"]'); await sleep(200);

  // phone width
  await page.setViewportSize({ width: 400, height: 800 }); await page.click('[data-act="split"]'); await sleep(300);
  await page.screenshot({ path: out + "wb-phone.png" });
} catch (e) { console.log("FAIL " + e.message); failed++; await page.screenshot({ path: out + "fail.png" }); }
finally {
  check(errors.length === 0, "no page errors " + errors.join("; "));
  await browser.close(); srv.kill();
  if (!failed && !process.env.KEEP) { // best effort: on Windows a shell that is still closing keeps its folder busy a moment
    for (const d of [dir, out]) try { rmSync(d, { recursive: true, force: true, maxRetries: 5, retryDelay: 300 }); } catch { /* the temp folder is cleaned later */ }
  }
  else console.log("files in " + dir + ", pictures in " + out);
}
process.exit(failed ? 1 : 0);
