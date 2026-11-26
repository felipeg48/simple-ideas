// The board (b in draw mode): on and off; papers white, black (chalk), blue
// (blueprint), B cycles them; patterns plain, grid, dots, lines, g cycles,
// both remembered; strokes stay on the board across slides, x clears the
// ink but not the pattern; Esc puts it away; PNG saves a picture of it and
// Copy puts one on the clipboard.
// Headless; needs ../../bin/si (make build). Run: node e2e/board.e2e.mjs [shots-dir]
import { chromium } from "playwright";
import { spawn } from "node:child_process";
import { mkdtempSync, readFileSync, writeFileSync } from "node:fs";
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
    const ctx = await browser.newContext({ viewport: { width: w, height: h }, colorScheme: scheme, acceptDownloads: true, permissions: ["clipboard-read", "clipboard-write"] });
    const p = await ctx.newPage(); const errors = []; p.on("pageerror", (e) => errors.push(e.message));
    await p.goto(base + "#1"); await sleep(500);
    const st = () => p.evaluate(() => { const b = document.querySelector(".si-board"); return { kind: b.getAttribute("data-board"), hidden: b.hidden,
      board: b.querySelectorAll(".si-board-ink path").length, slide: document.querySelectorAll(".si-draw path").length, pat: !!b.querySelector(".si-board-paper rect"),
      bg: getComputedStyle(b).backgroundColor, tools: document.querySelector('[data-si-pattern="grid"]').getClientRects().length > 0,
      pressed: document.querySelector('[data-si-pattern][aria-pressed="true"]')?.getAttribute("data-si-pattern") }; });
    await p.keyboard.press("a"); await stroke(p, 60, 200);
    let s = await st(); check(!s.tools, `${w}x${h} board tools hidden while the board is down`);
    await p.keyboard.press("b"); await sleep(100);
    s = await st(); check(!s.hidden && s.kind === "white" && s.bg === "rgb(251, 251, 248)" && s.tools, `${w}x${h} b opens the whiteboard, its tools show`);
    await stroke(p, 60, 300);
    s = await st(); check(s.board === 1 && s.slide === 1, `${w}x${h} the stroke goes on the board (${s.board} board, ${s.slide} slide)`);
    await p.keyboard.press("g"); s = await st(); check(s.pat && s.pressed === "grid", `${w}x${h} g: graph paper`);
    if (shots) await p.screenshot({ path: join(shots, `board-white-${w}.png`) });
    await p.click('[data-si-paper="black"]'); await stroke(p, 60, 420); await sleep(100);
    s = await st(); check(s.kind === "black" && s.board === 2 && s.pat, `${w}x${h} the black paper keeps ink and grid`);
    if (shots) await p.screenshot({ path: join(shots, `board-black-${w}.png`) });
    await p.keyboard.press("B"); await p.keyboard.press("g"); await sleep(100);
    s = await st(); check(s.kind === "blue" && s.pressed === "dots", `${w}x${h} B: blueprint, g: dots`);
    if (shots) await p.screenshot({ path: join(shots, `board-blue-${w}.png`) });
    const [dl] = await Promise.all([p.waitForEvent("download"), p.click('[data-si-draw="png"]')]);
    const png = readFileSync(await dl.path());
    check(dl.suggestedFilename() === "talk-board.png" && png.slice(1, 4).toString() === "PNG" && png.readUInt32BE(16) >= w, `${w}x${h} PNG saves ${dl.suggestedFilename()} (${png.readUInt32BE(16)}x${png.readUInt32BE(20)})`);
    if (shots) writeFileSync(join(shots, `board-export-${w}.png`), png);
    await p.click('[data-si-draw="copy"]'); await sleep(800);
    const clip = await p.evaluate(async () => { const it = await navigator.clipboard.read(); return it[0].types.join(","); }).catch((e) => "error " + e.message);
    check(/image\/png/.test(clip) && /Copied/.test(await p.textContent('[data-si-draw="copy"]')), `${w}x${h} Copy puts a picture on the clipboard (${clip})`);
    await p.keyboard.press("ArrowRight"); await sleep(300);
    s = await st(); check(s.board === 2 && s.slide === 0, `${w}x${h} the next slide clears the slide's ink, not the board`);
    await p.keyboard.press("z"); s = await st(); check(s.board === 1 && s.pat, `${w}x${h} z undoes on the board, the pattern stays`);
    await p.keyboard.press("x"); s = await st(); check(s.board === 0 && s.pat && !!(await p.$(".si-board-ink defs")), `${w}x${h} x clears the ink, pattern and chalk filter stay`);
    await p.keyboard.press("Escape"); s = await st(); check(s.hidden, `${w}x${h} Esc puts the pen and the board away`);
    await p.reload(); await sleep(400); await p.keyboard.press("a"); await p.keyboard.press("b");
    s = await st(); check(s.kind === "blue" && s.pressed === "dots", `${w}x${h} paper and pattern remembered (${s.kind}, ${s.pressed})`);
    check(errors.length === 0, `${w}x${h} no page errors ${errors.join("; ")}`);
    await ctx.close();
  }
} finally { await browser.close(); srv.kill(); }
process.exit(failed ? 1 : 0);
