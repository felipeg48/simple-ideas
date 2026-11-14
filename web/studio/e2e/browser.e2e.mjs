// Browser windows across sites: a framed page's link to another origin and a
// redirect to another origin both stay proxied (each origin its own proxy),
// the address bar shows the real address, back and forward cross the sites;
// a script that sends the page to another site (location.assign, replace,
// location.href =, window.open, a form) stays proxied too.
// Two local origins, si play, headless. Needs ../../bin/si (make build).
// Run: npm run e2e:browser
import { chromium } from "playwright";
import { spawn } from "node:child_process";
import { createServer } from "node:http";
import { mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

const si = new URL("../../../bin/si", import.meta.url).pathname;
const dir = mkdtempSync(join(tmpdir(), "si-browser-"));
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const listen = (h) => new Promise((ok) => { const s = createServer(h); s.listen(0, "127.0.0.1", () => ok(s)); });
const page = (title, body) => `<!doctype html><html><head><title>${title}</title></head><body><h1>${title}</h1>${body}</body></html>`;
const two = await listen((q, r) => {
  // framing refused, as consoles do: only the proxy shows it
  r.writeHead(200, { "Content-Type": "text/html", "X-Frame-Options": "DENY" });
  r.end(page("Two " + q.url, ""));
});
const twoURL = "http://localhost:" + two.address().port;
const one = await listen((q, r) => {
  if (q.url === "/hop") { r.writeHead(302, { Location: twoURL + "/hopped" }); r.end(); return; }
  r.writeHead(200, { "Content-Type": "text/html" });
  r.end(page("One", `<a id="cross" href="${twoURL}/linked">to two</a> <a id="hop" href="/hop">hop</a>
    <button id="assign" onclick="location.assign('${twoURL}/assigned')">assign</button>
    <button id="replace" onclick="location.replace('${twoURL}/replaced')">replace</button>
    <button id="href" onclick="location.href='${twoURL}/href'">href</button>
    <button id="open" onclick="window.open('${twoURL}/opened')">open</button>
    <form action="${twoURL}/form" method="get"><input name="q" value="orders"><button id="form">form</button></form>`));
});
const oneURL = "http://127.0.0.1:" + one.address().port;

writeFileSync(join(dir, "talk.md"), `# Consoles\n\nNotes: Two consoles, one window.\n`);
const port = 7391 + Math.floor(Math.random() * 400);
const srv = spawn(si, ["play", join(dir, "talk.md"), "--addr", "127.0.0.1:" + port], { stdio: "ignore" });
const base = "http://127.0.0.1:" + port + "/talk.html";
for (let i = 0; i < 50; i++) { try { await fetch(base); break; } catch { await sleep(100); } }

const browser = await chromium.launch();
let failed = 0;
const check = (ok, what) => { console.log((ok ? "ok   " : "FAIL ") + what); if (!ok) failed++; };
const errors = [];
try {
  const p = await (await browser.newContext({ viewport: { width: 1440, height: 900 } })).newPage();
  p.on("pageerror", (e) => errors.push(e.message));
  await p.goto(base);
  await p.evaluate(() => new Promise((ok) => { const s = document.createElement("script"); s.src = "/_si/term/assets/web.js"; s.onload = ok; document.head.appendChild(s); }));
  await p.evaluate((u) => window.siWeb.open("one", u), oneURL + "/");
  const frame = () => p.frames().find((f) => f.parentFrame() && f.url().startsWith("http"));
  const addr = () => p.inputValue(".si-web-url");
  const waitText = async (t) => { for (let i = 0; i < 40; i++) { const f = frame(); if (f && (await f.title().catch(() => "")).startsWith(t)) return f; await sleep(100); } return null; };
  let f = await waitText("One");
  check(!!f, "the first site opens in the window");
  check((await addr()) === oneURL + "/", "address bar: " + (await addr()));
  await f.click("#cross");
  f = await waitText("Two /linked");
  check(!!f && !f.url().startsWith(twoURL), "a link to another site opens through its own proxy: " + (f && f.url()));
  await sleep(300);
  check((await addr()) === twoURL + "/linked", "address bar shows the real address: " + (await addr()));
  await p.click('[data-a="back"]');
  f = await waitText("One");
  check(!!f, "back crosses to the first site");
  await sleep(300);
  check((await addr()) === oneURL + "/", "address bar after back: " + (await addr()));
  await p.click('[data-a="fwd"]');
  f = await waitText("Two /linked");
  check(!!f, "forward crosses again");
  await p.click('[data-a="back"]');
  f = await waitText("One");
  await f.click("#hop");
  f = await waitText("Two /hopped");
  check(!!f && !f.url().startsWith(twoURL), "a redirect to another site stays proxied: " + (f && f.url()));
  await sleep(300);
  check((await addr()) === twoURL + "/hopped", "address bar after the redirect: " + (await addr()));
  // script navigations to the other site
  for (const [id, path] of [["assign", "/assigned"], ["href", "/href"], ["open", "/opened"], ["form", "/form?q=orders"], ["replace", "/replaced"]]) {
    await p.click('[data-a="back"]');
    f = await waitText("One");
    if (!f) { check(false, "back to the first site before " + id); continue; }
    await f.click("#" + id);
    f = await waitText("Two " + path);
    check(!!f && !f.url().startsWith(twoURL), id + " to another site stays proxied: " + (f && f.url()));
    await sleep(200);
    check((await addr()) === twoURL + path, id + ": address bar " + (await addr()));
  }
  await p.screenshot({ path: join(dir, "browser.png") });
  check(errors.length === 0, "no page errors " + errors.join("; "));
} finally {
  await browser.close();
  srv.kill(); one.close(); two.close();
  if (!failed && !process.env.KEEP) rmSync(dir, { recursive: true, force: true });
  else console.log("files in " + dir);
}
process.exit(failed ? 1 : 0);
