// The docs site's Draw and board film: docs/site/features/img/draw/draw.mp4
// (and the pictures beside it). Draws on a slide, opens the board, sketches,
// changes the paper to chalk and blueprint, copies it.
// Headless; needs ../../bin/si (make build) and ffmpeg.
// Run: node e2e/draw-video.mjs OUTDIR
import { chromium } from "playwright";
import { spawn, execFileSync } from "node:child_process";
import { rmSync } from "node:fs";
import { join } from "node:path";

const si = new URL("../../../bin/si", import.meta.url).pathname;
const deck = new URL("../../../docs/site/features/showcase/draw.md", import.meta.url).pathname;
const out = process.argv[2];
const port = 7391 + Math.floor(Math.random() * 400);
const srv = spawn(si, ["play", deck, "--addr", "127.0.0.1:" + port], { stdio: "ignore" });
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const base = "http://127.0.0.1:" + port + "/draw.html";
for (let i = 0; i < 50; i++) { try { await fetch(base); break; } catch { await sleep(100); } }

const W = 1280, H = 720;
const browser = await chromium.launch();
const ctx = await browser.newContext({ viewport: { width: W, height: H }, colorScheme: "light", recordVideo: { dir: out, size: { width: W, height: H } },
  permissions: ["clipboard-read", "clipboard-write"] });
const p = await ctx.newPage();
const t0 = Date.now();
let mark = 0;
// A hand-speed line through the points.
const line = async (pts, step = 7) => {
  await p.mouse.move(pts[0][0], pts[0][1]); await p.mouse.down();
  for (let i = 1; i < pts.length; i++) {
    const [ax, ay] = pts[i - 1], [bx, by] = pts[i];
    const n = Math.max(3, Math.round(Math.hypot(bx - ax, by - ay) / 14));
    for (let k = 1; k <= n; k++) { await p.mouse.move(ax + (bx - ax) * k / n, ay + (by - ay) * k / n); await sleep(step); }
  }
  await p.mouse.up(); await sleep(120);
};
const box = (x, y, w, h) => line([[x, y], [x + w, y + 2], [x + w - 2, y + h], [x + 1, y + h - 1], [x, y + 3]]);
const arrow = async (x1, y, x2) => { await line([[x1, y], [x2, y + 2]]); await line([[x2 - 18, y - 13], [x2, y + 2], [x2 - 18, y + 16]]); };
const ring = (cx, cy, rx, ry) => line(Array.from({ length: 30 }, (_, i) => [cx + rx * Math.cos(-0.4 + i / 27 * 2 * Math.PI), cy + ry * Math.sin(-0.4 + i / 27 * 2 * Math.PI)]));
const textRect = (s) => p.evaluate((s) => {
  const w = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
  for (let n; (n = w.nextNode());) { const i = n.data.indexOf(s); if (i >= 0) { const r = document.createRange(); r.setStart(n, i); r.setEnd(n, i + s.length); const b = r.getBoundingClientRect(); if (b.width && b.x >= 0 && b.x < innerWidth && b.y >= 0 && b.y < innerHeight) return { x: b.x, y: b.y, w: b.width, h: b.height }; } }
  return null;
}, s);
try {
  await p.goto(base + "#2"); await sleep(1400);
  await p.keyboard.press("a"); await sleep(400);
  const pay = await textRect("Payments");
  if (pay) await ring(pay.x + 125, pay.y + pay.h / 2, 165, pay.h / 2 + 14);
  const fifty = await textRect("one time in fifty");
  if (fifty) { await p.keyboard.press("c"); await line([[fifty.x, fifty.y + fifty.h + 3], [fifty.x + fifty.w * .5, fifty.y + fifty.h + 6], [fifty.x + fifty.w, fifty.y + fifty.h + 2]]); }
  await p.mouse.move(1220, 90); await sleep(2500);
  await p.screenshot({ path: join(out, "draw-slide.png") });
  // The board, on graph paper: the three services, the risky hop ringed.
  await p.keyboard.press("b"); await sleep(300); await p.click('[data-si-pattern="grid"]'); await sleep(500);
  await p.keyboard.press("c"); await p.keyboard.press("c"); await p.keyboard.press("c");
  await box(150, 250, 230, 130); await arrow(395, 315, 515);
  await box(530, 250, 230, 130); await arrow(775, 315, 895);
  await box(910, 250, 220, 130);
  await p.keyboard.press("c"); await ring(645, 315, 175, 125);
  await p.keyboard.press("c"); await line([[460, 520], [640, 524], [830, 518]]); await line([[480, 548], [660, 551], [800, 546]]);
  await p.mouse.move(1220, 90); await sleep(1100);
  await p.click('[data-si-paper="black"]'); await p.mouse.move(1220, 90); await sleep(1800);
  await p.click('[data-si-paper="blue"]'); await p.mouse.move(1220, 90); await sleep(1800);
  await p.click('[data-si-draw="copy"]'); await sleep(1500);
  await p.keyboard.press("Escape"); await sleep(1000);
} finally {
  const v = p.video(); await ctx.close(); await browser.close(); srv.kill();
  const webm = await v.path();
  execFileSync("ffmpeg", ["-y", "-loglevel", "error", "-ss", "0.8", "-i", webm, "-vf", "scale=1280:-2,fps=30", "-c:v", "libx264", "-preset", "slow", "-crf", "26", "-pix_fmt", "yuv420p", "-movflags", "+faststart", "-an", join(out, "draw.mp4")]);
  execFileSync("ffmpeg", ["-y", "-loglevel", "error", "-ss", "0.8", "-i", webm, "-frames:v", "1", "-q:v", "4", join(out, "draw.poster.jpg")]);
  rmSync(webm);
}
