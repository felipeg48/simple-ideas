// Two (or more) presenters: one deck under si play, a teleprompter page per
// speaker (?si-view=tp&speaker=…). Each shows its own script and the others
// as cue lines, both show the hand-over banner, speakers.control: turn greys
// the other speaker's Next. Headless; needs ../../bin/si (make build).
// Run: npm run e2e:speakers
import { chromium } from "playwright";
import { spawn } from "node:child_process";
import { mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

const si = new URL("../../../bin/si", import.meta.url).pathname;
const dir = mkdtempSync(join(tmpdir(), "si-speakers-"));
writeFileSync(join(dir, "talk.md"), `---
title: Pay day
speakers: Ana, Felipe, Cleo
---

# Pay day

Notes[Cleo]: (0.5 min)
Welcome, we are three today.

---

## Payment: declined, then paid

<!-- steps: on -->
- the card fails
- the retry goes through

Notes[Ana]: (1 min)
The token reaches payments and turns red. [CLICK]

Notes[Felipe]: (0.5 min)
Here is the fix we shipped: one retry, thirty seconds later. [CLICK]

Points[Felipe]:
- Show the retry log if asked
`);
writeFileSync(join(dir, "si.conf"), "speakers.control: turn\n");
const port = 7391 + Math.floor(Math.random() * 400);
const srv = spawn(si, ["play", join(dir, "talk.md"), "--addr", "127.0.0.1:" + port], { stdio: "ignore" });
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const base = "http://127.0.0.1:" + port + "/talk.html";
for (let i = 0; i < 50; i++) { try { await fetch(base); break; } catch { await sleep(100); } }

const browser = await chromium.launch();
const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } });
const errors = [];
const open = async (u) => { const p = await ctx.newPage(); p.on("pageerror", (e) => errors.push(e.message)); await p.goto(u); return p; };
let failed = 0;
const check = (ok, what) => { console.log((ok ? "ok   " : "FAIL ") + what); if (!ok) failed++; };
try {
  const deck = await open(base + "#2");
  await sleep(500);
  const ana = await open(base + "?si-view=tp&speaker=Ana");
  const felipe = await open(base + "?si-view=tp&speaker=Felipe");
  await sleep(800);
  const view = (p) => p.evaluate(() => ({
    notes: document.getElementById("notes").innerText,
    other: [...document.querySelectorAll("#notes .pv-other")].map((e) => e.innerText),
    banner: document.getElementById("handover").classList.contains("on") ? document.getElementById("handover").innerText : "",
    next: document.getElementById("nextb").disabled,
    mine: document.getElementById("mineclock").hidden ? "" : document.getElementById("mineclock").innerText,
    points: document.getElementById("points").innerText,
    picker: document.getElementById("speaker").value,
  }));
  let a = await view(ana), f = await view(felipe);
  check(a.picker === "Ana" && f.picker === "Felipe", "?speaker= picks the speaker");
  check(a.notes.includes("turns red") && !a.notes.includes("thirty seconds later") && a.other.some((t) => t.startsWith("Felipe 00:30")), "Ana: her script, Felipe as a cue line");
  check(f.notes.includes("thirty seconds later") && f.other.some((t) => t.startsWith("Ana 01:00") && t.includes("The token reaches") && !/click/i.test(t)), "Felipe: his script, Ana's first words as his cue");
  check(!a.next && f.next, "turn: Ana drives, Felipe's Next is greyed");
  check(f.points.includes("retry log") && !a.points.includes("retry log"), "Points[Felipe] only on Felipe's teleprompter");
  check(a.mine.startsWith("Ana") && a.mine.includes("/ 1 min"), "Ana's own clock: " + a.mine);
  await deck.keyboard.press("ArrowRight");
  await sleep(500);
  a = await view(ana); f = await view(felipe);
  check(a.banner.includes("Felipe") && f.banner.includes("Felipe") && f.banner.includes("(you)"), "hand-over banner on both: " + a.banner + " / " + f.banner);
  check(a.next && !f.next, "turn: now Felipe drives");
  await felipe.click("#nextb");
  await sleep(500);
  check((await felipe.evaluate(() => document.getElementById("pos").innerText)).includes("step 2 / 2"), "Felipe's Next drives the deck");
  await sleep(4200);
  a = await view(ana);
  check(a.banner === "", "the banner fades after a few seconds");
  // the picker: Everyone shows every part under its name
  await ana.selectOption("#speaker", "");
  a = await view(ana);
  check(a.notes.includes("Ana") && a.notes.includes("turns red") && a.notes.includes("thirty seconds later") && a.other.length === 0 && !a.next, "Everyone: every part, nothing greyed");
  await ana.screenshot({ path: join(dir, "tp-everyone.png") });
  await felipe.screenshot({ path: join(dir, "tp-felipe.png") });
  check(errors.length === 0, "no page errors " + errors.join("; "));
} finally {
  await browser.close();
  srv.kill();
  if (!failed && !process.env.KEEP) rmSync(dir, { recursive: true, force: true });
  else console.log("files in " + dir);
}
process.exit(failed ? 1 : 0);
