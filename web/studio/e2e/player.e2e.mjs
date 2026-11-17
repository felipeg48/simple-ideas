// The video player: a built article (```video fences, inline and a modal
// gallery, captions and chapters found beside the clip) and a page si did not
// build (si --export player, data-video cards), both over http. Play and
// pause, keys, speed, captions, a chapter jump, the dialog (open, gallery,
// close, focus back on the card), the drop-in cards. Headless; needs
// ../../bin/si (make build) and ffmpeg (a short WebM clip is made here).
// SI_SHOTS=<dir> also writes screenshots (desktop light and dark, phone).
// Run: npm run e2e:player
import { chromium } from "playwright";
import { execFileSync } from "node:child_process";
import { mkdtempSync, mkdirSync, writeFileSync, readFileSync, existsSync, rmSync } from "node:fs";
import { createServer } from "node:http";
import { tmpdir } from "node:os";
import { join, extname } from "node:path";

const si = new URL("../../../bin/si", import.meta.url).pathname;
const dir = mkdtempSync(join(tmpdir(), "si-player-"));
const src = join(dir, "src");
mkdirSync(src);
const ff = (...a) => execFileSync("ffmpeg", ["-loglevel", "error", "-y", ...a], { stdio: "ignore" });
ff("-f", "lavfi", "-i", "testsrc2=size=640x360:rate=25:duration=20", "-f", "lavfi", "-i", "sine=frequency=330:duration=20",
  "-c:v", "libvpx", "-b:v", "300k", "-c:a", "libopus", "-shortest", join(src, "clip.webm"));
ff("-i", join(src, "clip.webm"), "-frames:v", "1", join(src, "poster.jpg"));
writeFileSync(join(src, "clip.vtt"), "WEBVTT\n\n00:00.000 --> 00:19.000\nThe order leaves the shop.\n");
writeFileSync(join(src, "clip.chapters.txt"), "0:00 The shop\n0:06 Payments\n0:12 The courier\n");
const modal = (t, cap) => "```video src=\"clip.webm\" poster=\"poster.jpg\" mode=modal title=\"" + t + "\"\n" + cap + "\n```\n";
writeFileSync(join(src, "player.md"), `---
type: article
title: An order's journey
---
# An order's journey

\`\`\`video src="clip.webm" poster="poster.jpg" title="From the shop to the door"
Twenty seconds, three teams.
\`\`\`

## The three legs

${modal("The shop", "Where the order starts.")}
${modal("Payments", "The money moves first.")}
${modal("The courier", "The last mile.")}
`);
execFileSync(si, ["build", join(src, "player.md"), "-o", join(dir, "out")], { stdio: "ignore" });
// the drop-in: si --export player, cards like a hand-built videos page
execFileSync(si, ["--export", "player", join(dir, "out", "drop")], { stdio: "ignore" });
for (const f of ["clip.webm", "clip.vtt", "clip.chapters.txt", "poster.jpg"]) writeFileSync(join(dir, "out", "drop", f), readFileSync(join(src, f)));
const card = (t, extra = "") => `<figure class="video-card" data-video="clip.webm" data-captions="clip.vtt" data-chapters="clip.chapters.txt"${extra}>
  <div class="video-media"><img src="poster.jpg" alt="${t}"></div>
  <figcaption><h3>${t}</h3><p>About ${t.toLowerCase()}.</p></figcaption></figure>`;
writeFileSync(join(dir, "out", "drop", "videos.html"), `<!doctype html><html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1"><title>Videos</title>
<link rel="stylesheet" href="si-player.css"><script src="si-player.js" defer></script>
<style>body{font-family:system-ui;margin:2rem}.grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(14rem,1fr));gap:1rem}
.video-card{margin:0;border:1px solid #ddd;border-radius:.6rem;overflow:hidden}.video-card img{display:block;width:100%}figcaption{padding:.6rem}</style></head>
<body><h1>Videos</h1><div class="grid">${card("Route")}${card("Replicate")}${card("Reason")}</div>
<h2>In place</h2><div style="max-width:30rem">${card("Inline", ' data-mode="inline"')}</div></body></html>`);

const types = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".webm": "video/webm", ".vtt": "text/vtt", ".txt": "text/plain", ".jpg": "image/jpeg" };
const server = createServer((req, res) => {
  const p = join(dir, "out", decodeURIComponent(new URL(req.url, "http://x").pathname));
  if (!existsSync(p)) { res.writeHead(404); res.end(); return; }
  const body = readFileSync(p), range = req.headers.range;
  const type = types[extname(p)] || "application/octet-stream";
  if (range) { // videos seek by range
    const [a, b] = range.replace("bytes=", "").split("-");
    const start = +a, end = b ? +b : body.length - 1;
    res.writeHead(206, { "Content-Type": type, "Accept-Ranges": "bytes", "Content-Range": `bytes ${start}-${end}/${body.length}`, "Content-Length": end - start + 1 });
    res.end(body.subarray(start, end + 1));
    return;
  }
  res.writeHead(200, { "Content-Type": type, "Accept-Ranges": "bytes", "Content-Length": body.length });
  res.end(body);
});
await new Promise((r) => server.listen(0, "127.0.0.1", r));
const base = `http://127.0.0.1:${server.address().port}`;

const browser = await chromium.launch({ args: ["--autoplay-policy=no-user-gesture-required"] });
let failed = 0;
const check = (ok, what) => { console.log((ok ? "ok   " : "FAIL ") + what); if (!ok) failed++; };
const shots = process.env.SI_SHOTS;
const near = (a, b, d = 0.6) => Math.abs(a - b) < d;

try {
  const page = await browser.newPage({ viewport: { width: 1440, height: 900 } });
  const errors = [];
  page.on("pageerror", (e) => errors.push(String(e)));
  await page.goto(base + "/player.html");
  const box = page.locator(".si-vp[data-vp-mode=inline] .si-vp-box");
  const v = box.locator("video");
  const t = () => v.evaluate((x) => x.currentTime);
  await v.evaluate((x) => x.readyState >= 1 || new Promise((r) => x.addEventListener("loadedmetadata", r, { once: true })));
  check(await v.evaluate((x) => !x.controls), "the HUD replaces the native controls");
  check(await box.locator(".si-vp-mark").count() === 2, "two chapter marks on the bar (0:06, 0:12)");
  check(await box.locator(".si-vp-time").textContent() === "0:00 / 0:20", "time reads 0:00 / 0:20");

  await box.locator(".si-vp-big").click();
  await page.waitForFunction(() => !document.querySelector(".si-vp[data-vp-mode=inline] video").paused);
  check(await box.getAttribute("data-state") === "playing", "the big button plays");
  await page.keyboard.press("k");
  check(await v.evaluate((x) => x.paused), "k pauses");
  await page.keyboard.press(" ");
  check(await v.evaluate((x) => !x.paused), "Space plays");
  await page.keyboard.press(" ");
  await v.evaluate((x) => { x.currentTime = 1; });
  await page.keyboard.press("ArrowRight");
  check(near(await t(), 6), "→ seeks 5 s");
  await page.keyboard.press("l");
  check(near(await t(), 16), "L seeks 10 s");
  await page.keyboard.press("j");
  await page.keyboard.press("ArrowLeft");
  check(near(await t(), 1), "J and ← seek back");

  await box.locator(".si-vp-speed").click();
  check(await v.evaluate((x) => x.playbackRate) === 1.25 && (await box.locator(".si-vp-speed").textContent()) === "1.25×", "speed 1.25×");
  await box.locator(".si-vp-speed").click(); await box.locator(".si-vp-speed").click(); await box.locator(".si-vp-speed").click();
  check(await v.evaluate((x) => x.playbackRate) === 1, "speed cycles back to 1×");

  check(await box.locator(".si-vp-cc").isVisible(), "a CC button for the .vtt beside the clip");
  await box.focus();
  await page.keyboard.press("c");
  await page.waitForFunction(() => document.querySelector(".si-vp[data-vp-mode=inline] .si-vp-cue").textContent.includes("leaves the shop"), null, { timeout: 4000 }).catch(() => {});
  check(await box.getAttribute("data-cc") !== null && (await box.locator(".si-vp-cue").textContent()).includes("leaves the shop"), "C shows the captions");
  check(await box.locator(".si-vp-cc").getAttribute("aria-pressed") === "true", "CC is pressed");

  await box.locator(".si-vp-chap").click();
  check(await box.locator(".si-vp-menu").isVisible(), "the chapter list opens");
  await box.locator(".si-vp-mi", { hasText: "Payments" }).click();
  check(near(await t(), 6, 0.3), "a chapter jumps to 0:06");
  check((await box.locator(".si-vp-chname").textContent()) === "Payments", "the HUD names the chapter");
  await v.evaluate((x) => x.pause());
  await page.keyboard.press("m");
  check(await v.evaluate((x) => x.muted), "M mutes");
  if (shots) {
    await v.evaluate((x) => { x.currentTime = 7; });
    await box.hover();
    await page.waitForTimeout(300);
    await page.screenshot({ path: join(shots, "player-inline-1440.png") });
  }

  // the modal gallery
  const opens = page.locator(".si-vp[data-vp-mode=modal] .si-vp-open");
  await opens.nth(0).click();
  const dlg = page.locator("dialog.si-vp-dialog");
  check(await dlg.evaluate((d) => d.open), "the poster opens the dialog");
  check((await dlg.locator(".si-vp-count").textContent()) === "1 / 3", "counter 1 / 3");
  check((await dlg.locator(".si-vp-dtitle").textContent()) === "The shop", "the title under the video");
  check(await page.evaluate(() => document.querySelector("dialog.si-vp-dialog").contains(document.activeElement)), "focus moves into the dialog");
  const dv = dlg.locator("video");
  await page.waitForFunction(() => !document.querySelector("dialog.si-vp-dialog video").paused);
  check(true, "the dialog plays");
  await page.keyboard.press("Shift+ArrowRight");
  check((await dlg.locator(".si-vp-count").textContent()) === "2 / 3", "Shift+→ while playing: the next video");
  await dv.evaluate((x) => x.pause());
  await page.keyboard.press("ArrowRight");
  check((await dlg.locator(".si-vp-count").textContent()) === "3 / 3" && (await dlg.locator(".si-vp-dtitle").textContent()) === "The courier", "→ while paused: the next video");
  await dlg.locator(".si-vp-next").click();
  check((await dlg.locator(".si-vp-count").textContent()) === "1 / 3", "› wraps to the first");
  await dlg.locator(".si-vp-prev").click();
  check((await dlg.locator(".si-vp-count").textContent()) === "3 / 3", "‹ goes back");
  check(await dlg.locator(".si-vp-mark").count() === 2, "the dialog player has the chapters too");
  if (shots) {
    await dv.evaluate((x) => { x.currentTime = 7; });
    await page.waitForTimeout(400);
    await page.screenshot({ path: join(shots, "player-modal-1440.png") });
  }
  await page.keyboard.press("Escape");
  check(!(await dlg.evaluate((d) => d.open)), "Esc closes the dialog");
  check(await opens.nth(0).evaluate((b) => document.activeElement === b), "focus returns to the poster that opened it");
  check(await dv.evaluate((x) => x.paused), "closing pauses");
  await opens.nth(1).click();
  await page.mouse.click(5, 5);
  check(!(await dlg.evaluate((d) => d.open)), "a click on the backdrop closes it");
  check(errors.length === 0, "no page errors" + (errors.length ? ": " + errors.join("; ") : ""));

  if (shots) { // dark mode and a phone
    const dark = await browser.newPage({ viewport: { width: 1440, height: 900 }, colorScheme: "dark" });
    await dark.goto(base + "/player.html");
    await dark.locator(".si-vp[data-vp-mode=modal] .si-vp-open").nth(1).click();
    await dark.waitForTimeout(600);
    await dark.locator("dialog.si-vp-dialog video").evaluate((x) => { x.pause(); x.currentTime = 9; });
    await dark.waitForTimeout(300);
    await dark.screenshot({ path: join(shots, "player-modal-dark-1440.png") });
    await dark.close();
    const phone = await browser.newPage({ viewport: { width: 400, height: 800 }, hasTouch: true, isMobile: true });
    await phone.goto(base + "/player.html");
    await phone.locator(".si-vp[data-vp-mode=inline] .si-vp-big").click();
    await phone.waitForTimeout(500);
    await phone.locator(".si-vp[data-vp-mode=inline] video").evaluate((x) => { x.pause(); });
    await phone.screenshot({ path: join(shots, "player-inline-phone.png") });
    await phone.locator(".si-vp[data-vp-mode=modal] .si-vp-open").nth(0).click();
    await phone.waitForTimeout(500);
    await phone.locator("dialog.si-vp-dialog video").evaluate((x) => { x.pause(); x.currentTime = 4; });
    await phone.waitForTimeout(300);
    await phone.screenshot({ path: join(shots, "player-modal-phone.png") });
    await phone.close();
  }

  // the drop-in: data-video cards on a page si did not build
  const drop = await browser.newPage({ viewport: { width: 1440, height: 900 } });
  const dropErrors = [];
  drop.on("pageerror", (e) => dropErrors.push(String(e)));
  await drop.goto(base + "/drop/videos.html");
  const cards = drop.locator(".video-card");
  check(await cards.nth(0).getAttribute("role") === "button" && (await cards.nth(0).getAttribute("tabindex")) === "0", "a card is a keyboard button");
  await cards.nth(1).click();
  const ddlg = drop.locator("dialog.si-vp-dialog");
  check(await ddlg.evaluate((d) => d.open), "a card opens the dialog");
  check((await ddlg.locator(".si-vp-count").textContent()) === "2 / 3" && (await ddlg.locator(".si-vp-dtitle").textContent()) === "Replicate", "its place and title from the card");
  check((await ddlg.locator(".si-vp-dtext").textContent()) === "About replicate.", "its paragraph under the video");
  await drop.waitForFunction(() => document.querySelectorAll("dialog.si-vp-dialog .si-vp-mark").length === 2, null, { timeout: 4000 }).catch(() => {});
  check(await ddlg.locator(".si-vp-mark").count() === 2, "data-chapters fetched: marks on the bar");
  check(await ddlg.locator(".si-vp-cc").isVisible(), "data-captions: a CC button");
  if (shots) {
    await ddlg.locator("video").evaluate((x) => { x.pause(); x.currentTime = 3; });
    await drop.waitForTimeout(300);
    await drop.screenshot({ path: join(shots, "player-dropin-modal-1440.png") });
  }
  await drop.keyboard.press("Escape");
  check(await cards.nth(1).evaluate((c) => document.activeElement === c), "focus returns to the card");
  await cards.nth(3).click();
  const ib = cards.nth(3).locator(".si-vp-box");
  check(await ib.count() === 1, "data-mode=inline: the player takes the picture's place");
  await drop.waitForFunction(() => { const v = document.querySelector(".video-card .si-vp-box video"); return v && !v.paused; }, null, { timeout: 4000 }).catch(() => {});
  check(await ib.locator("video").evaluate((x) => !x.paused), "and plays");
  check(dropErrors.length === 0, "no page errors on the drop-in page" + (dropErrors.length ? ": " + dropErrors.join("; ") : ""));
  if (shots) {
    await ib.locator("video").evaluate((x) => x.pause());
    await drop.screenshot({ path: join(shots, "player-dropin-1440.png"), fullPage: true });
  }
} finally {
  await browser.close();
  server.close();
  rmSync(dir, { recursive: true, force: true });
}
if (failed) { console.log(`${failed} failed`); process.exit(1); }
console.log("player e2e: all ok");
