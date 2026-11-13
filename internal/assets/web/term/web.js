/* simple-ideas play browser windows (key w).
 *
 * Play-only and loopback-only, like the terminal drawer: deck.js lazy-loads
 * this file the first time w is pressed (or on load when windows were open
 * before a live reload). Each window frames a web page through a loopback
 * proxy si play starts for that page's origin (internal/serve/browser.go), so
 * consoles that refuse framing still show. Windows move (drag the title bar),
 * resize (bottom-right corner), minimise to the dock, maximise and close; their
 * places survive a live reload (sessionStorage si-web). Styles live here, dark
 * and outside the deck theme, as the terminal's do. */
(function () {
  'use strict';
  if (window.siWeb) return;
  var LINKS = window.SI_BROWSER || [];
  var KEY = 'si-web', RECENT = 'si-web-recent';
  var style = document.createElement('style');
  style.textContent =
    '.si-web{position:fixed;z-index:9100;display:flex;flex-direction:column;min-width:240px;min-height:140px;' +
    'background:#0b0e14;border:1px solid #2a3550;border-radius:10px;overflow:hidden;resize:both;' +
    'box-shadow:0 18px 50px rgba(0,0,0,.45);font:12px ui-sans-serif,system-ui,sans-serif;color:#e6e6e6}' +
    '.si-web.is-max{left:1vw!important;top:1vh!important;width:98vw!important;height:96vh!important;resize:none}' +
    '.si-web.is-min{display:none}' +
    '.si-web-bar{flex:0 0 auto;display:flex;align-items:center;gap:6px;padding:5px 6px 5px 10px;background:#11151f;' +
    'border-bottom:1px solid #1d2536;cursor:grab;user-select:none}' +
    '.si-web.is-drag .si-web-bar{cursor:grabbing}' +
    '.si-web-name{font-weight:600;white-space:nowrap;max-width:14em;overflow:hidden;text-overflow:ellipsis}' +
    '.si-web-url{flex:1 1 auto;min-width:4em;background:#0b0e14;color:#c8d0e0;border:1px solid #1d2536;border-radius:6px;' +
    'padding:4px 8px;font:12px ui-monospace,Menlo,monospace;cursor:text}' +
    '.si-web-url:focus{outline:none;border-color:#7dd3fc}' +
    '.si-web-b{flex:0 0 auto;width:24px;height:24px;display:grid;place-items:center;background:transparent;color:#c8d0e0;' +
    'border:1px solid transparent;border-radius:6px;font:14px/1 ui-sans-serif,system-ui,sans-serif;cursor:pointer;padding:0}' +
    '.si-web-b:hover{background:#1b2436;border-color:#2a3550;color:#fff}.si-web-b.x:hover{background:#7f1d1d;border-color:#b91c1c}' +
    '.si-web-frame{flex:1 1 auto;border:0;width:100%;background:#fff}' +
    '.si-web.is-drag .si-web-frame,.si-web-busy .si-web-frame{pointer-events:none}' +
    '.si-web-dock{position:fixed;left:12px;bottom:12px;z-index:9150;display:flex;gap:6px;flex-wrap:wrap;max-width:70vw}' +
    '.si-web-pill{background:#11151f;color:#e6e6e6;border:1px solid #2a3550;border-radius:999px;padding:5px 12px;' +
    'font:12px ui-sans-serif,system-ui,sans-serif;cursor:pointer;box-shadow:0 6px 18px rgba(0,0,0,.35)}' +
    '.si-web-pill:hover{border-color:#7dd3fc}' +
    '.si-web-pick{position:fixed;left:50%;top:18vh;transform:translateX(-50%);z-index:9200;width:min(520px,92vw);' +
    'background:#0d1017;border:1px solid #2a3550;border-radius:12px;padding:14px;box-shadow:0 24px 60px rgba(0,0,0,.5);' +
    'font:13px ui-sans-serif,system-ui,sans-serif;color:#e6e6e6;display:flex;flex-direction:column;gap:10px}' +
    '.si-web-pick h3{margin:0;font-size:14px}.si-web-pick form{display:flex;gap:6px}' +
    '.si-web-pick input{flex:1 1 auto;background:#0b0e14;color:#e6e6e6;border:1px solid #2a3550;border-radius:6px;' +
    'padding:7px 9px;font:13px ui-monospace,Menlo,monospace}.si-web-pick input:focus{outline:none;border-color:#7dd3fc}' +
    '.si-web-pick button{background:#1b2436;color:#e6e6e6;border:1px solid #2a3550;border-radius:6px;padding:6px 11px;' +
    'font:13px ui-sans-serif,system-ui,sans-serif;cursor:pointer}.si-web-pick button:hover{border-color:#7dd3fc}' +
    '.si-web-pick button.pri{background:#7dd3fc;color:#0b0e14;border-color:#7dd3fc}' +
    '.si-web-links{display:flex;flex-wrap:wrap;gap:6px}.si-web-links span{color:#7d8aa5;font-size:11px}' +
    '.si-web-hint{color:#7d8aa5;font-size:11px}';
  document.head.appendChild(style);

  var dock = document.createElement('div'); dock.className = 'si-web-dock';
  document.body.appendChild(dock);
  var wins = [], z = 9100, seq = 0;

  function save() {
    var st = wins.map(function (w) {
      var r = w.el.getBoundingClientRect();
      return { name: w.name, url: w.url, x: r.left, y: r.top, w: r.width, h: r.height, min: w.el.classList.contains('is-min'), max: w.el.classList.contains('is-max') };
    });
    try { sessionStorage.setItem(KEY, JSON.stringify(st)); } catch (e) {}
  }
  function recent(url) {
    var list = []; try { list = JSON.parse(localStorage.getItem(RECENT) || '[]'); } catch (e) {}
    list = [url].concat(list.filter(function (u) { return u !== url; })).slice(0, 6);
    try { localStorage.setItem(RECENT, JSON.stringify(list)); } catch (e) {}
  }
  function proxied(url) {
    return fetch('/_si/browser/open', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ url: url }) })
      .then(function (r) { if (!r.ok) return r.text().then(function (t) { throw new Error(t); }); return r.json(); })
      .then(function (j) { return j.url; });
  }
  function norm(u) { u = (u || '').trim(); if (u && !/^[a-z]+:\/\//i.test(u)) u = 'http://' + u; return u; }
  function front(w) { w.el.style.zIndex = ++z; }

  function open(name, url, at) {
    url = norm(url); if (!url) return;
    recent(url);
    var w = { id: ++seq, name: name || url.replace(/^[a-z]+:\/\//i, '').replace(/\/.*$/, ''), url: url };
    var el = w.el = document.createElement('div'); el.className = 'si-web';
    var k = wins.length;
    at = at || { x: innerWidth * 0.45 + k * 28, y: innerHeight * 0.08 + k * 28, w: innerWidth * 0.5, h: innerHeight * 0.66 };
    el.style.left = Math.max(0, Math.min(at.x, innerWidth - 120)) + 'px'; el.style.top = Math.max(0, Math.min(at.y, innerHeight - 40)) + 'px';
    el.style.width = at.w + 'px'; el.style.height = at.h + 'px';
    el.innerHTML = '<div class="si-web-bar"><span class="si-web-name"></span><input class="si-web-url" spellcheck="false" aria-label="Address">' +
      '<button class="si-web-b" data-a="reload" title="Reload">↻</button><button class="si-web-b" data-a="min" title="Minimise">–</button>' +
      '<button class="si-web-b" data-a="max" title="Maximise">□</button><button class="si-web-b x" data-a="close" title="Close">✕</button></div>' +
      '<iframe class="si-web-frame" referrerpolicy="no-referrer"></iframe>';
    el.querySelector('.si-web-name').textContent = w.name;
    var input = el.querySelector('.si-web-url'), frame = el.querySelector('iframe');
    input.value = url;
    function go(u) {
      w.url = norm(u); input.value = w.url; recent(w.url); save();
      proxied(w.url).then(function (p) { frame.src = p; }, function (e) {
        frame.srcdoc = '<body style="font:14px system-ui;padding:2em;color:#333">Cannot open ' + w.url.replace(/</g, '&lt;') + ': ' + String(e.message).replace(/</g, '&lt;') + '</body>';
      });
    }
    // keys typed in the window chrome must not drive the deck
    el.addEventListener('keydown', function (ev) { ev.stopPropagation(); if (ev.key === 'Enter' && ev.target === input) go(input.value); });
    el.addEventListener('pointerdown', function () { front(w); }, true);
    el.querySelector('.si-web-bar').addEventListener('dblclick', function (ev) { if (ev.target.closest('input,button')) return; el.classList.toggle('is-max'); save(); });
    el.querySelectorAll('[data-a]').forEach(function (b) {
      b.addEventListener('click', function () {
        var a = b.getAttribute('data-a');
        if (a === 'reload') { try { frame.contentWindow.location.reload(); } catch (e) { go(w.url); } }
        else if (a === 'min') minimise(w, true);
        else if (a === 'max') { el.classList.toggle('is-max'); save(); }
        else close(w);
      });
    });
    drag(w, el.querySelector('.si-web-bar'));
    new ResizeObserver(function () { save(); }).observe(el);
    document.body.appendChild(el); wins.push(w); front(w);
    go(url);
    return w;
  }
  function drag(w, bar) {
    bar.addEventListener('pointerdown', function (ev) {
      if (ev.button !== 0 || ev.target.closest('input,button') || w.el.classList.contains('is-max')) return;
      var r = w.el.getBoundingClientRect(), dx = ev.clientX - r.left, dy = ev.clientY - r.top;
      w.el.classList.add('is-drag'); document.body.classList.add('si-web-busy'); bar.setPointerCapture(ev.pointerId);
      function mv(e) {
        w.el.style.left = Math.max(-r.width + 80, Math.min(e.clientX - dx, innerWidth - 80)) + 'px';
        w.el.style.top = Math.max(0, Math.min(e.clientY - dy, innerHeight - 30)) + 'px';
      }
      function up() { bar.removeEventListener('pointermove', mv); bar.removeEventListener('pointerup', up); w.el.classList.remove('is-drag'); document.body.classList.remove('si-web-busy'); save(); }
      bar.addEventListener('pointermove', mv); bar.addEventListener('pointerup', up);
    });
  }
  function minimise(w, on) {
    w.el.classList.toggle('is-min', on);
    if (on) {
      var pill = w.pill = document.createElement('button'); pill.className = 'si-web-pill'; pill.textContent = '▢ ' + w.name; pill.title = 'Restore';
      pill.addEventListener('click', function () { minimise(w, false); front(w); });
      dock.appendChild(pill);
    } else if (w.pill) { w.pill.remove(); w.pill = null; }
    save();
  }
  function close(w) {
    w.el.remove(); if (w.pill) w.pill.remove();
    wins = wins.filter(function (x) { return x !== w; }); save();
  }

  /* w: the picker (the deck's browser: links, recent addresses, a typed one) */
  var pick = null;
  function picker() {
    if (pick) { pick.remove(); pick = null; return; }
    pick = document.createElement('div'); pick.className = 'si-web-pick';
    var recentList = []; try { recentList = JSON.parse(localStorage.getItem(RECENT) || '[]'); } catch (e) {}
    pick.innerHTML = '<h3>Open a browser window</h3><div class="si-web-links"></div>' +
      '<form><input placeholder="localhost:15672" aria-label="Address"><button class="pri" type="submit">Open</button></form>' +
      '<div class="si-web-hint">Drag the title bar to move · corner to resize · double-click to maximise · Esc closes this</div>';
    var links = pick.querySelector('.si-web-links');
    function btn(label, name, url) { var b = document.createElement('button'); b.type = 'button'; b.textContent = label; b.title = url; b.addEventListener('click', function () { done(); open(name, url); }); links.appendChild(b); }
    LINKS.forEach(function (l) { btn(l.name, l.name, l.url); });
    recentList.filter(function (u) { return !LINKS.some(function (l) { return norm(l.url) === u; }); }).forEach(function (u) { btn(u.replace(/^[a-z]+:\/\//i, ''), '', u); });
    if (!links.children.length) links.innerHTML = '<span>Add links with browser: in the deck front matter.</span>';
    var input = pick.querySelector('input');
    pick.querySelector('form').addEventListener('submit', function (ev) { ev.preventDefault(); if (input.value.trim()) { var u = input.value; done(); open('', u); } });
    pick.addEventListener('keydown', function (ev) { ev.stopPropagation(); if (ev.key === 'Escape') done(); });
    function done() { if (pick) { pick.remove(); pick = null; } }
    document.body.appendChild(pick); input.focus();
  }

  // restore the windows a live reload closed
  var st = []; try { st = JSON.parse(sessionStorage.getItem(KEY) || '[]'); } catch (e) {}
  st.forEach(function (s) { var w = open(s.name, s.url, { x: s.x, y: s.y, w: s.w, h: s.h }); if (w) { if (s.max) w.el.classList.add('is-max'); if (s.min) minimise(w, true); } });

  window.siWeb = { picker: picker, open: open, restoreOnly: function () {} };
})();
