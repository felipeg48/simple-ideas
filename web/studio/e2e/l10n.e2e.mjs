// The buttons' language: a deck with lang: es gets Spanish buttons in the
// deck and the teleprompter (static labels, keys kept, labels the runtime
// changes later), slides and notes stay as written, and a deck without
// lang: keeps English (si.conf ui.lang: render's TestUILang). Headless; needs ../../bin/si (make build).
// Run: node e2e/l10n.e2e.mjs [shots-dir]
import { chromium } from "playwright";
import { spawn } from "node:child_process";
import { mkdtempSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

const si = new URL("../../../bin/si", import.meta.url).pathname;
const shots = process.argv[2];
const dir = mkdtempSync(join(tmpdir(), "si-l10n-"));
const talk = "---\ntitle: El viaje\nlang: es\n---\n\n# Draw\n\nNotes: (1 min)\nNext, the order. [CLICK]\n\n---\n\n## Dos\n\nNotes: (1 min)\nPause here.\n";
writeFileSync(join(dir, "talk.md"), talk);
writeFileSync(join(dir, "en.md"), talk.replace("lang: es\n", ""));
const port = 7391 + Math.floor(Math.random() * 400);
const srv = spawn(si, ["play", dir, "--addr", "127.0.0.1:" + port], { stdio: "ignore" });
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const base = "http://127.0.0.1:" + port + "/";
for (let i = 0; i < 50; i++) { try { await fetch(base + "talk.html"); break; } catch { await sleep(100); } }

const browser = await chromium.launch();
let failed = 0;
const check = (ok, what) => { console.log((ok ? "ok   " : "FAIL ") + what); if (!ok) failed++; };
const txt = (p, sel) => p.$eval(sel, (e) => e.textContent.trim());
try {
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  const errors = [];
  const deck = await ctx.newPage(); deck.on("pageerror", (e) => errors.push(e.message));
  await deck.goto(base + "talk.html#1"); await sleep(500);
  check(await txt(deck, '[data-si-action="draw"]') === "Dibujar", "deck: Draw → Dibujar");
  check(await deck.$eval('[data-si-action="overview"]', (e) => e.title) === "Vista general (o)", "deck: a title keeps its key");
  check(await txt(deck, '[data-si-draw="board"]') === "Pizarra", "draw bar: Board → Pizarra");
  check((await txt(deck, ".si-help")).includes("primera / última diapositiva"), "help in Spanish");
  check(await txt(deck, ".si-slide h1") === "Draw", "the slide's own text stays as written");
  await deck.keyboard.press("v"); await sleep(300);
  check(/Pausa|Empezar|Continuar/.test(await txt(deck, '[data-si-ap="toggle"]')), `a label the runtime sets later: ${await txt(deck, '[data-si-ap="toggle"]')}`);
  await deck.keyboard.press("Escape");
  const tp = await ctx.newPage(); tp.on("pageerror", (e) => errors.push(e.message));
  await tp.goto(base + "talk.html?si-view=tp"); await sleep(900);
  check(await txt(tp, "#fit") === "Ajustar al tiempo" && await txt(tp, "#focus") === "Enfoque", "teleprompter: Fit time, Focus");
  check(await txt(tp, "#tab-notes") === "Notas del orador", "teleprompter: tabs");
  check(/^diapositiva 1 \/ 2/.test(await txt(tp, "#pos")), `teleprompter: position (${await txt(tp, "#pos")})`);
  await tp.keyboard.press("a"); await sleep(400);
  check(/^desplazamiento automático/.test(await txt(tp, "#scrollstate")), `teleprompter: a changing label (${await txt(tp, "#scrollstate")})`);
  check((await txt(tp, "#notes")).startsWith("Next, the order."), "teleprompter: the script stays as written");
  check(/^Siguiente/.test(await txt(tp, "#label")), `teleprompter: next label (${await txt(tp, "#label")})`);
  if (shots) { await tp.screenshot({ path: join(shots, "l10n-tp.png") }); await deck.keyboard.press("a"); await deck.screenshot({ path: join(shots, "l10n-deck.png") }); }
  const en = await ctx.newPage(); await en.goto(base + "en.html#1"); await sleep(400);
  check(await txt(en, '[data-si-action="draw"]') === "Draw", "a deck without lang: keeps English");
  check(errors.length === 0, "no page errors " + errors.join("; "));
} finally { await browser.close(); srv.kill(); }
process.exit(failed ? 1 : 0);
