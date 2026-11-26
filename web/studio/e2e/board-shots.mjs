// Pictures of the board for the docs site's Decks page:
// docs/site/features/img/decks/board-{white,black,blue}.jpg.
// Headless; needs ../../bin/si (make build). Run: node e2e/board-shots.mjs OUTDIR
// (PNG files; the Makefile-free step: sips -s format jpeg -Z 1000 to the site folder).
import { chromium } from "playwright";
import { spawn } from "node:child_process";
import { mkdtempSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

const si = new URL("../../../bin/si", import.meta.url).pathname;
const out = process.argv[2];
const dir = mkdtempSync(join(tmpdir(), "si-board-shots-"));
writeFileSync(join(dir, "talk.md"), "---\ntitle: One order\n---\n\n# One order, six services\n");
const port = 7391 + Math.floor(Math.random() * 400);
const srv = spawn(si, ["play", join(dir, "talk.md"), "--addr", "127.0.0.1:" + port], { stdio: "ignore" });
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const base = "http://127.0.0.1:" + port + "/talk.html";
for (let i = 0; i < 50; i++) { try { await fetch(base); break; } catch { await sleep(100); } }

const line = async (p, pts) => {
  await p.mouse.move(pts[0][0], pts[0][1]); await p.mouse.down();
  for (let i = 1; i < pts.length; i++) {
    const [ax, ay] = pts[i - 1], [bx, by] = pts[i];
    for (let k = 1; k <= 8; k++) await p.mouse.move(ax + (bx - ax) * k / 8 + Math.sin(k) * 1.2, ay + (by - ay) * k / 8 + Math.cos(k) * 1.2);
  }
  await p.mouse.up();
};
const box = (p, x, y, w, h) => line(p, [[x, y], [x + w, y + 2], [x + w - 2, y + h], [x + 1, y + h - 1], [x, y + 4]]);
const arrow = async (p, x1, y, x2) => { await line(p, [[x1, y], [x2, y + 3]]); await line(p, [[x2 - 22, y - 16], [x2, y + 3], [x2 - 22, y + 20]]); };
const ring = (p, cx, cy, r) => line(p, Array.from({ length: 26 }, (_, i) => [cx + r * 1.3 * Math.cos(i / 24 * 2 * Math.PI), cy + r * Math.sin(i / 24 * 2 * Math.PI)]));
const colour = async (p, n) => { for (let i = 0; i < n; i++) await p.keyboard.press("c"); };

// The same sketch on each board: cart → payments → parcel, the risky hop ringed,
// a note underlined.
const sketch = async (p) => {
  await box(p, 170, 330, 260, 150); await arrow(p, 450, 405, 600);
  await box(p, 620, 330, 260, 150); await arrow(p, 900, 405, 1050);
  await box(p, 1070, 330, 220, 150);
  await colour(p, 1); await ring(p, 750, 405, 190);
  await colour(p, 1); await line(p, [[520, 640], [700, 645], [880, 638], [940, 642]]);
  await line(p, [[540, 670], [760, 674], [900, 668]]);
  await colour(p, 2);
};

const browser = await chromium.launch();
try {
  for (const [kind, pattern] of [["white", "grid"], ["black", "plain"], ["blue", "grid"]]) {
    const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 }, colorScheme: "dark" });
    const p = await ctx.newPage();
    await p.goto(base + "#1"); await sleep(500);
    await p.keyboard.press("a"); await p.keyboard.press("b"); await sleep(100);
    await p.click(`[data-si-paper="${kind}"]`); await p.click(`[data-si-pattern="${pattern}"]`);
    await sketch(p); await p.mouse.move(1380, 120); await sleep(300);
    await p.screenshot({ path: join(out, `board-${kind}.png`) });
    await ctx.close();
  }
} finally { await browser.close(); srv.kill(); }
