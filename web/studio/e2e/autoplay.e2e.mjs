// Auto-play resumes mid-part: a pause and a resume at the same slide and step
// go on from where the voice stopped (the audio from its position, the
// browser's voice from its sentence); another slide starts its part from the
// beginning. Headless, with a stub speechSynthesis and a served silent WAV.
// Needs ../../bin/si (make build). Run: npm run e2e:autoplay
import { chromium } from "playwright";
import { spawn } from "node:child_process";
import { mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

const si = new URL("../../../bin/si", import.meta.url).pathname;
const dir = mkdtempSync(join(tmpdir(), "si-autoplay-"));
writeFileSync(join(dir, "talk.md"), `# Orders

Notes: One order arrives. Two services answer. Three events follow.

---

## Payments

Notes: Four checks run.
`);
const port = 7391 + Math.floor(Math.random() * 400);
const srv = spawn(si, ["play", join(dir, "talk.md"), "--addr", "127.0.0.1:" + port], { stdio: "ignore" });
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const base = "http://127.0.0.1:" + port + "/talk.html";
for (let i = 0; i < 50; i++) { try { await fetch(base); break; } catch { await sleep(100); } }

// a silent 16-bit mono WAV of `sec` seconds
function wav(sec) {
  const rate = 8000, n = rate * sec, b = Buffer.alloc(44 + n * 2);
  b.write("RIFF", 0); b.writeUInt32LE(36 + n * 2, 4); b.write("WAVEfmt ", 8);
  b.writeUInt32LE(16, 16); b.writeUInt16LE(1, 20); b.writeUInt16LE(1, 22);
  b.writeUInt32LE(rate, 24); b.writeUInt32LE(rate * 2, 28); b.writeUInt16LE(2, 32); b.writeUInt16LE(16, 34);
  b.write("data", 36); b.writeUInt32LE(n * 2, 40);
  return b;
}

const browser = await chromium.launch({ args: ["--autoplay-policy=no-user-gesture-required"] });
let failed = 0;
const check = (ok, what) => { console.log((ok ? "ok   " : "FAIL ") + what); if (!ok) failed++; };
const errors = [];
async function open(audio) {
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  await ctx.addInitScript(() => {
    window.__said = [];
    const syn = {
      getVoices: () => [{ name: "stub" }],
      speak(u) { window.__said.push(u.text); window.__cur = u; },
      cancel() { window.__cur = null; },
    };
    Object.defineProperty(window, "speechSynthesis", { value: syn, configurable: true });
    window.SpeechSynthesisUtterance = function (t) { this.text = t; };
    window.__finish = () => { const u = window.__cur; window.__cur = null; if (u && u.onend) u.onend(); };
    const A = window.Audio;
    window.__audios = [];
    window.Audio = function () { const a = new A(); window.__audios.push(a); return a; };
  });
  const p = await ctx.newPage();
  p.on("pageerror", (e) => errors.push(e.message));
  await p.route("**/_si/voice/**", (r) => audio ? r.fulfill({ status: 200, contentType: "audio/wav", body: wav(6) }) : r.fulfill({ status: 404, body: "" }));
  await p.goto(base + "#1");
  await sleep(400);
  return p;
}
const said = (p) => p.evaluate(() => window.__said.slice());
const finish = (p) => p.evaluate(() => window.__finish());
const pause = (p) => p.mouse.click(700, 450); // a click on the slide takes control back

try {
  // the browser's voice: one sentence at a time, resumed at the sentence
  let p = await open(false);
  await p.keyboard.press("v");
  await sleep(300);
  check((await said(p)).join("|") === "One order arrives.", "speaks the first sentence alone: " + (await said(p)).join("|"));
  await finish(p); await sleep(100);
  check((await said(p)).at(-1) === "Two services answer.", "then the second");
  await pause(p); await sleep(200);
  check(await p.evaluate(() => document.querySelector(".si-autoplay").dataset.state) === "paused", "a click pauses");
  await p.keyboard.press("v"); await sleep(300);
  check((await said(p)).at(-1) === "Two services answer.", "resume repeats the current sentence, not the part: " + (await said(p)).join("|"));
  await finish(p); await sleep(100);
  check((await said(p)).at(-1) === "Three events follow.", "and goes on");
  await finish(p); await sleep(1300);
  check((await said(p)).at(-1) === "Four checks run.", "next slide after the part");
  await pause(p); await sleep(100);
  await p.keyboard.press("ArrowLeft"); await sleep(300);
  await p.keyboard.press("v"); await sleep(300);
  check((await said(p)).at(-1) === "One order arrives.", "another slide starts its part from the beginning");
  await p.context().close();

  // the audio: the same element keeps its position
  p = await open(true);
  await p.keyboard.press("v");
  await sleep(2000);
  const t1 = await p.evaluate(() => window.__audios[0] && window.__audios[0].currentTime);
  await pause(p); await sleep(300);
  const t2 = await p.evaluate(() => window.__audios[0].currentTime);
  await p.keyboard.press("v"); await sleep(800);
  const t3 = await p.evaluate(() => ({ t: window.__audios[0].currentTime, paused: window.__audios[0].paused }));
  if (!(t1 > 0.5)) console.log("skip audio: this Chromium plays no audio here (" + t1 + ")"); // a CI box without an audio sink
  else check(t3.t >= t2 && !t3.paused, `resume keeps the audio position (${t2.toFixed(2)} -> ${t3.t.toFixed(2)})`);
  await p.context().close();
  check(errors.length === 0, "no page errors " + errors.join("; "));
} finally {
  await browser.close();
  srv.kill();
  if (!failed && !process.env.KEEP) rmSync(dir, { recursive: true, force: true });
  else console.log("files in " + dir);
}
process.exit(failed ? 1 : 0);
