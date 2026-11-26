// The teleprompter's Focus: the line on the reading line and one line either
// side stay bright, the rest dims; f and the Focus button toggle it, it is
// remembered, and the script's first line starts on the reading line.
// Fit time (k): auto-scroll brings the last line to the reading line as the
// slide's Notes: (N min) run out.
// Headless; needs ../../bin/si (make build). Run: node e2e/focus.e2e.mjs [shots-dir]
import { chromium } from "playwright";
import { spawn } from "node:child_process";
import { mkdtempSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

const si = new URL("../../../bin/si", import.meta.url).pathname;
const shots = process.argv[2];
const dir = mkdtempSync(join(tmpdir(), "si-focus-"));
const para = "Every order starts as a click on a phone, and from that moment it travels through six services before it reaches a door. ";
writeFileSync(join(dir, "talk.md"), `---\ntitle: The journey\n---\n\n# The journey\n\nNotes: (2 min)\n${para.repeat(4)}\n\n${para.repeat(4)}\n\n---\n\n## Fit\n\nNotes: (0.05 min)\n${para.repeat(6)}\n`);
const port = 7391 + Math.floor(Math.random() * 400);
const srv = spawn(si, ["play", join(dir, "talk.md"), "--addr", "127.0.0.1:" + port], { stdio: "ignore" });
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const base = "http://127.0.0.1:" + port + "/talk.html";
for (let i = 0; i < 50; i++) { try { await fetch(base); break; } catch { await sleep(100); } }

const browser = await chromium.launch();
let failed = 0;
const check = (ok, what) => { console.log((ok ? "ok   " : "FAIL ") + what); if (!ok) failed++; };
try {
  for (const [w, h, scheme] of [[1440, 900, "light"], [400, 800, "dark"], [820, 1180, "light"]]) {
    const ctx = await browser.newContext({ viewport: { width: w, height: h }, colorScheme: scheme });
    const errors = [];
    const deck = await ctx.newPage(); await deck.goto(base + "#1");
    const tp = await ctx.newPage(); tp.on("pageerror", (e) => errors.push(e.message));
    await tp.goto(base + "?si-view=tp"); await sleep(800);
    await tp.keyboard.press("f"); await sleep(200);
    const s = await tp.evaluate(() => {
      const pv = document.getElementById("pv"), n = document.getElementById("notes"), pr = document.getElementById("prompt");
      const line = pr.getBoundingClientRect().top + parseFloat(getComputedStyle(pr).getPropertyValue("--pv-line"));
      const r = document.createRange(), t = n.querySelector("p").firstChild; r.setStart(t, 0); r.setEnd(t, 5); const first = r.getClientRects()[0];
      return { on: pv.classList.contains("focus"), pressed: document.getElementById("focus").getAttribute("aria-pressed"),
        mask: getComputedStyle(n).maskImage || getComputedStyle(n).webkitMaskImage, line, mid: (first.top + first.bottom) / 2, lh: first.height };
    });
    check(s.on && s.pressed === "true", `${w}x${h} f turns Focus on`);
    check(/gradient/.test(s.mask), `${w}x${h} the script is masked`);
    check(Math.abs(s.mid - s.line) < s.lh * 0.75, `${w}x${h} first line on the reading line (${Math.round(s.mid)} vs ${Math.round(s.line)})`);
    if (shots) await tp.screenshot({ path: join(shots, `focus-${w}x${h}.png`) });
    await tp.reload(); await sleep(600);
    check(await tp.evaluate(() => document.getElementById("pv").classList.contains("focus")), `${w}x${h} Focus remembered`);
    await tp.click("#focus"); await sleep(100);
    check(!(await tp.evaluate(() => document.getElementById("pv").classList.contains("focus"))), `${w}x${h} the button turns it off`);
    check(errors.length === 0, `${w}x${h} no page errors ${errors.join("; ")}`);
    await ctx.close();
  }
  // fit time: 3 s of notes, the last line on the reading line when they end
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  const deck = await ctx.newPage(); await deck.goto(base + "#1");
  const tp = await ctx.newPage(); await tp.goto(base + "?si-view=tp"); await sleep(600);
  await tp.keyboard.press("ArrowRight"); await sleep(100);
  await tp.keyboard.press("k");
  const last = () => tp.evaluate(() => {
    const n = document.getElementById("notes"), pr = document.getElementById("prompt"), ps = n.querySelectorAll("p"), t = ps[ps.length - 1].lastChild;
    const r = document.createRange(); r.setStart(t, t.length - 3); r.setEnd(t, t.length);
    const b = r.getClientRects()[0], line = pr.getBoundingClientRect().top + parseFloat(getComputedStyle(pr).getPropertyValue("--pv-line"));
    return { off: (b.top + b.bottom) / 2 - line, lh: b.height, state: document.getElementById("scrollstate").textContent, auto: document.getElementById("auto").getAttribute("aria-pressed") };
  });
  const a = await last();
  check(a.auto === "true" && /fits 0.05 min/.test(a.state), `k turns auto-scroll on, fitted (${a.state})`);
  await sleep(1500);
  const mid = await last();
  check(mid.off > a.lh && mid.off < a.off * 0.75, `halfway the script is partway (${Math.round(mid.off)} of ${Math.round(a.off)} px left)`);
  await sleep(1700);
  const end = await last();
  check(Math.abs(end.off) < end.lh, `when the time is up the last line is on the reading line (${Math.round(end.off)} px)`);
  check(/time is up/.test(end.state), `then it says so (${end.state})`);
  await tp.keyboard.press("k");
  await ctx.close();
} finally { await browser.close(); srv.kill(); }
process.exit(failed ? 1 : 0);
