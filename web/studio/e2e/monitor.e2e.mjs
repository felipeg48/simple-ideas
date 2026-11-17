// Headless end-to-end for si play's web monitor, the control room. Run: make build && node e2e/monitor.e2e.mjs
// si play --audience on a talk with a hidden poll and a word cloud → the deck,
// an audience phone and the monitor open → the monitor counts them by role →
// the phone votes and asks → the monitor reveals the poll and approves and
// shows the question → both arrive on the deck. Screenshots of the monitor at
// four sizes, light and dark, in e2e/out/.
import { chromium } from "playwright";
import { spawn } from "node:child_process";
import { mkdtempSync, mkdirSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const si = join(here, "../../../bin/si");
const out = join(here, "out");
mkdirSync(out, { recursive: true });
const dir = mkdtempSync(join(tmpdir(), "si-e2e-monitor-"));
writeFileSync(join(dir, "talk.md"), `---
title: Retries that help
---

# Retries that help

---

## How often should a client retry?

<!-- poll: retry · Once · Three times · Never · hidden -->

Notes:
Pick one on your phone. [CLICK] Here is what the room said.

---

## One word for a retry storm

<!-- cloud: storm -->

---

## Your questions

<!-- qr: audience -->
`);

const srv = spawn(si, ["play", join(dir, "talk.md"), "--audience", "--addr", "127.0.0.1:0", "--remote-addr", "127.0.0.1:0"], { stdio: ["ignore", "pipe", "pipe"] });
process.on("exit", () => { srv.kill(); try { rmSync(dir, { recursive: true, force: true }); } catch {} }); // si may still be saving .si/audience
const links = await new Promise((res, rej) => {
  let buf = "";
  const t = setTimeout(() => rej(new Error("no links from si play: " + buf)), 15000);
  const on = (d) => {
    buf += d;
    const deck = buf.match(/serving \S+ at (http:\/\/\S+)/), mon = buf.match(/monitor [^:]*: (http:\/\/\S+)/), aud = buf.match(/Audience \(follow along\): (http:\/\/\S+)/);
    if (deck && mon && aud) { clearTimeout(t); res({ deck: deck[1] + "talk.html", mon: mon[1], aud: aud[1] }); }
  };
  srv.stdout.on("data", on); srv.stderr.on("data", on);
});
let failed = 0; const check = (ok, what) => { console.log((ok ? "ok   " : "FAIL ") + what); if (!ok) failed++; };
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const browser = await chromium.launch({ headless: true });
const errors = [];
const page = async (url, opts) => {
  const p = await (await browser.newContext(opts)).newPage();
  p.on("pageerror", (e) => errors.push(e.message));
  await p.goto(url);
  return p;
};
let mon;
try {
  // the key is the gate: no key, no page
  const bare = await fetch(links.mon.replace(/\?k=.*/, ""));
  check(bare.status === 403, "the monitor without its key: " + bare.status);

  const deck = await page(links.deck, { viewport: { width: 1280, height: 720 } });
  await deck.waitForSelector(".si-slide.si-current");
  const phone = await page(links.aud, { viewport: { width: 400, height: 800 }, hasTouch: true });
  await phone.waitForSelector(".si-slide.si-current");
  mon = await page(links.mon, { viewport: { width: 1440, height: 900 } });
  await mon.waitForSelector("#mon-live[data-on]");
  check(!/k=/.test(mon.url()), "the key leaves the address bar: " + mon.url());

  const role = (r) => mon.locator(`#mon-v [data-role="${r}"] b`);
  await role("audience").waitFor({ timeout: 8000 });
  check(await role("deck").textContent() === "1", "one deck counted");
  check(await role("audience").textContent() === "1", "one audience phone counted");
  check(await role("monitor").textContent() === "1", "the monitor counts itself");

  // the deck moves to the poll; the phone follows and votes
  await deck.keyboard.press("ArrowRight");
  await phone.waitForSelector('.si-slide.si-current .si-poll[data-poll="retry"]', { timeout: 8000 });
  await phone.locator('.si-slide.si-current .si-poll-opt[data-i="1"]').click();
  const poll = mon.locator('[data-poll="retry"]');
  await poll.getByText("1 vote", { exact: true }).waitFor({ timeout: 8000 });
  check(true, "the vote reaches the monitor");
  check(await mon.locator("#mon-tiles").getByText("2 / 4").count() === 1, "the monitor shows the slide");
  const hiddenOnDeck = () => deck.locator('.si-slide.si-current .si-poll .si-poll-n').first().evaluate((e) => getComputedStyle(e).visibility);
  check(await hiddenOnDeck() === "hidden", "a hidden poll's counts are hidden on the deck");
  await poll.getByRole("button", { name: "Reveal results" }).click();
  await deck.waitForSelector('.si-slide.si-current .si-poll[data-revealed]', { timeout: 8000 });
  check(await hiddenOnDeck() === "visible", "Reveal results shows the counts on the deck");

  // the phone asks; the monitor approves, then shows it over the slide
  await phone.locator("#si-ask-btn").click();
  await phone.locator("#si-ask-q").fill("Should a retry ever be immediate?");
  await phone.locator("#si-ask-name").fill("Ana");
  await phone.locator('#si-ask button[type="submit"]').click();
  const q = mon.locator(".mon-q", { hasText: "Should a retry ever be immediate?" });
  await q.waitFor({ timeout: 8000 });
  await q.getByRole("button", { name: "Approve" }).click();
  await q.locator('.mon-tag[data-k="approved"]').waitFor({ timeout: 8000 });
  check(true, "Approve marks the question");
  await q.getByRole("button", { name: "Show on screen" }).click();
  await deck.waitForSelector(".si-qa-show:not([hidden])", { timeout: 8000 });
  check(/immediate/.test(await deck.locator(".si-qa-show .si-qa-text").textContent()), "the question shows over the deck");

  // the room's link carries no presenter key
  const tag = await mon.locator("#mon-l").textContent();
  check(/Audience/.test(tag), "the links list the audience link");

  // the monitor at four sizes, light and dark (rule #1)
  for (const scheme of ["light", "dark"]) {
    for (const [w, h] of [[1440, 900], [400, 800], [820, 1180], [3840, 2160]]) {
      const ctx = await browser.newContext({ viewport: { width: w, height: h }, colorScheme: scheme });
      const p = await ctx.newPage();
      p.on("pageerror", (e) => errors.push(e.message));
      await p.goto(links.mon);
      await p.waitForSelector("#mon-live[data-on]");
      await p.locator(".mon-q").first().waitFor();
      const wide = await p.evaluate(() => document.documentElement.scrollWidth - innerWidth);
      check(wide <= 0, `no sideways scroll at ${w}x${h} ${scheme}`);
      await p.screenshot({ path: join(out, `monitor-${w}x${h}-${scheme}.png`), fullPage: w === 400 });
      await ctx.close();
    }
  }
  check(errors.length === 0, "no page errors " + errors.join("; "));
} catch (e) {
  console.log("FAIL " + e.message);
  failed++;
  if (mon) await mon.screenshot({ path: join(out, "monitor-fail.png") }).catch(() => {});
}
await browser.close();
process.exit(failed ? 1 : 0);
