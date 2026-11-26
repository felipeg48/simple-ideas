// The board (b in draw mode): whiteboard, then blackboard (chalk), then off;
// strokes on the board stay on the board, are kept across slides, x clears
// the board only, and putting the pen away puts the board away.
// Headless; needs ../../bin/si (make build). Run: node e2e/board.e2e.mjs [shots-dir]
import { chromium } from "playwright";
import { spawn } from "node:child_process";
import { mkdtempSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

const si = new URL("../../../bin/si", import.meta.url).pathname;
const shots = process.argv[2];
const dir = mkdtempSync(join(tmpdir(), "si-board-"));
writeFileSync(join(dir, "talk.md"), "---\ntitle: Board\n---\n\n# One order\n\n---\n\n## Six services\n\n- cart\n- payments\n");
const port = 7391 + Math.floor(Math.random() * 400);
const srv = spawn(si, ["play", join(dir, "talk.md"), "--addr", "127.0.0.1:" + port], { stdio: "ignore" });
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const base = "http://127.0.0.1:" + port + "/talk.html";
for (let i = 0; i < 50; i++) { try { await fetch(base); break; } catch { await sleep(100); } }

const browser = await chromium.launch();
let failed = 0;
const check = (ok, what) => { console.log((ok ? "ok   " : "FAIL ") + what); if (!ok) failed++; };
const stroke = async (p, x, y) => { await p.mouse.move(x, y); await p.mouse.down(); for (let i = 1; i <= 12; i++) await p.mouse.move(x + i * 25, y + Math.sin(i / 2) * 40); await p.mouse.up(); };
try {
  for (const [w, h, scheme] of [[1440, 900, "dark"], [400, 800, "light"]]) {
    const ctx = await browser.newContext({ viewport: { width: w, height: h }, colorScheme: scheme });
    const p = await ctx.newPage(); const errors = []; p.on("pageerror", (e) => errors.push(e.message));
    await p.goto(base + "#1"); await sleep(500);
    const st = () => p.evaluate(() => ({ kind: document.querySelector(".si-board").getAttribute("data-board"), hidden: document.querySelector(".si-board").hidden,
      board: document.querySelectorAll(".si-board path").length, slide: document.querySelectorAll(".si-draw path").length,
      bg: getComputedStyle(document.querySelector(".si-board")).backgroundColor + getComputedStyle(document.querySelector(".si-board")).backgroundImage }));
    await p.keyboard.press("a"); await stroke(p, 60, 200);
    await p.keyboard.press("b"); await sleep(100);
    let s = await st();
    check(!s.hidden && s.kind === "white" && /251, 251, 248/.test(s.bg), `${w}x${h} b opens the whiteboard`);
    await stroke(p, 60, 300);
    s = await st(); check(s.board === 1 && s.slide === 1, `${w}x${h} the stroke goes on the board (${s.board} board, ${s.slide} slide)`);
    if (shots) await p.screenshot({ path: join(shots, `board-white-${w}.png`) });
    await p.keyboard.press("b"); await stroke(p, 60, 420); await sleep(100);
    s = await st(); check(s.kind === "black" && /gradient/.test(s.bg) && s.board === 2, `${w}x${h} b again: the blackboard, ink kept`);
    if (shots) await p.screenshot({ path: join(shots, `board-black-${w}.png`) });
    await p.keyboard.press("ArrowRight"); await sleep(300);
    s = await st(); check(s.board === 2 && s.slide === 0, `${w}x${h} the next slide clears the slide's ink, not the board`);
    await p.keyboard.press("z"); s = await st(); check(s.board === 1, `${w}x${h} z undoes on the board`);
    await p.keyboard.press("x"); s = await st(); check(s.board === 0 && !!(await p.$(".si-board defs")), `${w}x${h} x clears the board, the chalk filter stays`);
    await p.keyboard.press("Escape"); s = await st(); check(s.hidden, `${w}x${h} Esc puts the pen and the board away`);
    check(errors.length === 0, `${w}x${h} no page errors ${errors.join("; ")}`);
    await ctx.close();
  }
} finally { await browser.close(); srv.kill(); }
process.exit(failed ? 1 : 0);
