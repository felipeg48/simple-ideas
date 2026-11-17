// Headless end-to-end for the Studio's Video view (a type: video edit).
// Run: make build && node e2e/videoedit.e2e.mjs (needs ffmpeg and ffprobe).
// A two-second clip made by ffmpeg, an edit with a bubble, a box and an
// arrow → the Video view shows the preview and its strip → a marker clicked
// selects its line in the text, the cursor's line is marked in the preview →
// the box dragged and resized, the bubble dragged (real pointer): their lines
// change on disk in source pixels → ⌘Z puts the box back → a moving blur's
// places are handles while paused, one dragged rewrites that place (⌘Z
// back) → a zoom's rectangle shows with its line and drags → ⌘B builds and
// the built video plays in the view. Screenshots in e2e/out/.
import { chromium } from "playwright";
import { spawn, execFileSync } from "node:child_process";
import { cpSync, mkdtempSync, readFileSync, mkdirSync, rmSync, writeFileSync, existsSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const si = process.env.SI_BIN || join(here, "../../../bin/si");
const out = join(here, "out");
mkdirSync(out, { recursive: true });
try { execFileSync("ffmpeg", ["-version"], { stdio: "ignore" }); } catch { console.log("skip: no ffmpeg"); process.exit(0); }
const root = mkdtempSync(join(tmpdir(), "si-e2e-video-"));
cpSync(join(here, "fixture"), root, { recursive: true });
// WebM: Playwright's Chromium plays VP8, not H.264
execFileSync("ffmpeg", ["-v", "error", "-f", "lavfi", "-i", "testsrc2=s=640x360:r=30:d=2", "-f", "lavfi", "-i", "sine=f=440:d=2",
  "-c:v", "libvpx", "-b:v", "400k", "-c:a", "libopus", "-shortest", join(root, "clip.webm")]);
const EDIT = "---\ntype: video\nsource: clip.webm\nout: clip.final.mp4\n---\n# Clip\n\n## Timeline\n\n" +
  "- 0:00.2 · bubble \"Look here\" · to 320,180 · 1.6s · out none\n" + // line 10
  "- 0:00.2 · box 40,40 160x90 · 1.6s · out none\n" + // line 11
  "- 0:00.5 · arrow from 600,320 to 420,220 · 1.2s\n" + // line 12
  "- 0:00.2–0:01.9 · blur 40,250 80x60 · to 40,250 → 300,250 120x80 over 1.6s\n" + // line 13: a moving blur that grows
  "- 0:01.5 · zoom to 320,0 320x180 · 0.3s\n"; // line 14
writeFileSync(join(root, "clip.md"), EDIT);

const srv = spawn(si, ["--web", root, "--addr", "127.0.0.1:0", "--no-open"], { stdio: ["ignore", "pipe", "pipe"] });
process.on("exit", () => { srv.kill(); rmSync(root, { recursive: true, force: true }); });
const url = await new Promise((res, rej) => {
  let buf = "";
  const t = setTimeout(() => rej(new Error("no URL from si --web: " + buf)), 10000);
  const on = (d) => { buf += d; const m = buf.match(/at (http:\/\/\S+)/); if (m) { clearTimeout(t); res(m[1]); } };
  srv.stdout.on("data", on); srv.stderr.on("data", on);
});
const browser = await chromium.launch({ headless: true });
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const until = async (fn, ms = 8000, what = "condition") => {
  const end = Date.now() + ms;
  for (;;) {
    try { const v = await fn(); if (v) return v; } catch { /* retry */ }
    if (Date.now() > end) throw new Error("timeout: " + what);
    await sleep(100);
  }
};
let failed = 0;
const check = (ok, what) => { console.log((ok ? "ok   " : "FAIL ") + what); if (!ok) failed++; };
const FILE = join(root, "clip.md");
const disk = () => { try { return readFileSync(FILE, "utf8"); } catch { return ""; } };
const line = (n) => disk().split("\n")[n - 1];

try {
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  await ctx.addInitScript(() => { try { if (!localStorage.getItem("si-studio-theme")) localStorage.setItem("si-studio-theme", "light"); } catch {} });
  const page = await ctx.newPage();
  const errors = [];
  page.on("pageerror", (e) => errors.push(e.message));
  await page.goto(url);
  await page.waitForSelector(".app");
  await page.click('.row-main[title="clip.md"]');
  await page.waitForSelector(".vex iframe");
  check(await page.locator('.seg[aria-label="Video edit view"] button.on').textContent() === "Video", "a type: video file opens in the Video view");
  check(await page.locator(".ed-cm").isVisible(), "the text stays under the preview");
  const f = page.frameLocator(".vex iframe");
  await f.locator(".vp-strip .m").first().waitFor();
  const frame = () => page.frames().find((x) => x !== page.mainFrame());
  await until(() => frame()?.evaluate(() => !!window.siVideoEdit && document.querySelector("video").readyState >= 1), 10000, "preview ready");
  check(await f.locator(".vp-strip .m").count() === 5, "a marker per line on the strip");

  // marker → line; cursor line → marked
  await f.locator('.vp-lines li[data-line="12"] button').click();
  await until(async () => (await page.locator(".cm-activeLine").first().textContent()).includes("arrow from"), 4000, "marker selects its line");
  check(true, "a line clicked under the player selects it in the text");
  await page.locator(".cm-content .cm-line", { hasText: "box 40,40" }).click();
  await until(() => frame().evaluate(() => document.querySelector('.vp-lines li[data-line="11"]').classList.contains("ve-sel")), 4000, "cursor line marked");
  check(true, "the cursor's line is marked in the preview");

  // show 0:01 (every callout on), paused
  await frame().evaluate(() => { const v = document.querySelector("video"); v.pause(); v.currentTime = 1; });
  await sleep(300);
  const fr = await page.locator(".vex iframe").boundingBox();
  const rectOf = async (sel) => { const r = await frame().evaluate((s) => { const b = document.querySelector(s).getBoundingClientRect(); return [b.left, b.top, b.width, b.height]; }, sel); return { x: fr.x + r[0], y: fr.y + r[1], w: r[2], h: r[3] }; };
  const stage = await rectOf(".ve-stage");
  const k = 640 / stage.w; // source px per screen px
  // drag the box by (+100, +50) source px, then its corner by (+40, +20)
  const box = await rectOf('.ve-c[data-line="11"]');
  await page.mouse.move(box.x + box.w / 2, box.y + box.h / 2);
  await page.mouse.down();
  await page.mouse.move(box.x + box.w / 2 + 50 / k, box.y + box.h / 2 + 25 / k, { steps: 6 });
  await page.mouse.move(box.x + box.w / 2 + 100 / k, box.y + box.h / 2 + 50 / k, { steps: 6 });
  await page.mouse.up();
  await until(() => /box 1[34]\d,(8|9)\d 160x90/.test(line(11)), 6000, "box moved on disk: " + line(11));
  check(true, "a box dragged on the frame: " + line(11));
  await until(() => frame()?.evaluate(() => !!window.siVideoEdit), 6000, "preview redrawn");
  await frame().evaluate(() => { const v = document.querySelector("video"); v.pause(); v.currentTime = 1; });
  await sleep(300);
  const grip = await rectOf('.ve-c[data-line="11"] .ve-grip');
  await page.mouse.move(grip.x + grip.w / 2, grip.y + grip.h / 2);
  await page.mouse.down();
  await page.mouse.move(grip.x + grip.w / 2 + 40 / k, grip.y + grip.h / 2 + 20 / k, { steps: 8 });
  await page.mouse.up();
  await until(() => /box 1[34]\d,(8|9)\d (19\d|20\d)x(10\d|11\d)/.test(line(11)), 6000, "box resized on disk: " + line(11));
  check(true, "its corner resizes it: " + line(11));

  // the bubble's tail to about 500,100
  await until(() => frame()?.evaluate(() => !!window.siVideoEdit), 6000, "preview redrawn");
  await frame().evaluate(() => { const v = document.querySelector("video"); v.pause(); v.currentTime = 1; });
  await sleep(300);
  const st = await rectOf(".ve-stage");
  const bub = await rectOf('.ve-c[data-line="10"] .ve-in');
  const tail = { x: st.x + 320 / k, y: st.y + 180 / k }; // the bubble sits above its tail
  await page.mouse.move(bub.x + bub.w / 2, bub.y + bub.h / 2);
  await page.mouse.down();
  await page.mouse.move(bub.x + bub.w / 2 + 180 / k, bub.y + bub.h / 2 - 80 / k, { steps: 8 });
  await page.mouse.up();
  await until(() => /to (49\d|50\d|51\d),(9\d|10\d|11\d) · 1\.6s/.test(line(10)), 6000, "bubble moved: " + line(10));
  check(true, "a bubble dragged writes to X,Y: " + line(10) + " (tail was at " + Math.round(tail.x) + "," + Math.round(tail.y) + ")");
  await page.screenshot({ path: join(out, "videoedit.png") });

  // undo through the editor puts the bubble back
  await page.locator(".vex .insp-head").first().click();
  await page.keyboard.press("ControlOrMeta+z");
  await until(() => line(10).includes("to 320,180"), 6000, "undo: " + line(10));
  check(true, "⌘Z undoes the drag through the editor");

  // a moving blur: its places are handles while paused; the second dragged by (-100, -60) source px
  await until(() => frame()?.evaluate(() => !!window.siVideoEdit), 6000, "preview redrawn");
  await frame().evaluate(() => { const v = document.querySelector("video"); v.pause(); v.currentTime = 1; });
  await until(() => frame().evaluate(() => [...document.querySelectorAll('.ve-key[data-line="13"]')].filter((k) => !k.hidden).length === 2), 4000, "path handles shown");
  check(true, "a moving blur's two places are handles while paused");
  const kf = await rectOf('.ve-key[data-line="13"][data-key="1"]');
  await page.mouse.move(kf.x + kf.w / 2, kf.y + kf.h / 2);
  await page.mouse.down();
  await page.mouse.move(kf.x + kf.w / 2 - 50 / k, kf.y + kf.h / 2 - 30 / k, { steps: 6 });
  await page.mouse.move(kf.x + kf.w / 2 - 100 / k, kf.y + kf.h / 2 - 60 / k, { steps: 6 });
  await page.mouse.up();
  await until(() => /to 40,250 → (19\d|20\d|21\d),(18\d|19\d) 120x80 over 1\.6s/.test(line(13)), 6000, "path place moved: " + line(13));
  check(true, "a place of a path dragged: " + line(13));
  await page.screenshot({ path: join(out, "videoedit-path.png") });
  await page.locator(".vex .insp-head").first().click();
  await page.keyboard.press("ControlOrMeta+z");
  await until(() => line(13).includes("→ 300,250 120x80"), 6000, "undo: " + line(13));
  check(true, "⌘Z puts the place back");

  // the zoom's rectangle: shown with its line, dragged by (-100, +40)
  await page.locator(".cm-content .cm-line", { hasText: "zoom to 320,0" }).click();
  await until(() => frame()?.evaluate(() => { const r = document.querySelector(".ve-camrect.ve-sel"); return r && r.getBoundingClientRect().width > 0; }), 6000, "zoom rectangle shown");
  await frame().evaluate(() => document.querySelector("video").pause());
  await sleep(200);
  const cr = await rectOf(".ve-camrect");
  await page.mouse.move(cr.x + cr.w / 2, cr.y + cr.h / 2);
  await page.mouse.down();
  await page.mouse.move(cr.x + cr.w / 2 - 50 / k, cr.y + cr.h / 2 + 20 / k, { steps: 6 });
  await page.mouse.move(cr.x + cr.w / 2 - 100 / k, cr.y + cr.h / 2 + 40 / k, { steps: 6 });
  await page.mouse.up();
  await until(() => /zoom to (2[01]\d|22\d),(3\d|4\d|5\d) 320x180 · 0\.3s/.test(line(14)), 6000, "zoom moved: " + line(14));
  check(true, "a zoom's rectangle dragged: " + line(14));

  // ⌘B: build, then the result plays in the view
  await page.keyboard.press("ControlOrMeta+b");
  await until(() => page.locator(".vex-result").isVisible(), 120000, "build done: " + (await page.locator(".vex-log").textContent().catch(() => "")));
  check(existsSync(join(root, "clip.final.mp4")), "⌘B built clip.final.mp4");
  const src = await page.locator(".vex-result").getAttribute("src");
  check((await page.evaluate((u) => fetch(u, { headers: { Range: "bytes=0-99" } }).then((r) => r.status), src)) === 206, "the view's player streams the built video (" + src + ")");
  if (await page.locator(".vex-result").evaluate((v) => v.canPlayType('video/mp4; codecs="avc1.42E01E"') !== "")) { // Chrome, not Playwright's Chromium (no H.264)
    const dur = await page.locator(".vex-result").evaluate((v) => new Promise((r) => { if (v.readyState >= 1) r(v.duration); else v.onloadedmetadata = () => r(v.duration); }));
    check(dur > 1.5 && dur < 2.5, "the built video plays in the view (" + dur + " s)");
  }
  await page.screenshot({ path: join(out, "videoedit-built.png") });
  check(errors.length === 0, "no page errors " + JSON.stringify(errors));
} catch (e) {
  console.log("FAIL " + e.message);
  failed++;
  try { await browser.contexts()[0]?.pages()[0]?.screenshot({ path: join(out, "videoedit-fail.png") }); } catch { /* closed */ }
} finally {
  await browser.close();
}
if (failed) { console.log(failed + " failed"); process.exit(1); }
console.log("all passed");
process.exit(0);
