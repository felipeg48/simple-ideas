/* simple-ideas play terminal drawer.
 *
 * Play-only (never shipped in a built deck) and loopback-only: deck.js lazy-
 * loads this file the first time the terminal is toggled, after xterm.js and
 * the fit addon are on the page. Each tab is one PTY-backed shell on the
 * presenter's machine; output streams in over fetch(), keystrokes go back out
 * over POST. See internal/serve/terminal.go.
 *
 * Styles live here (not in the themed layout/theme CSS), so the terminal stays
 * outside the si-* styling contract and a terminal is always a terminal —
 * dark, monospace — regardless of the deck's theme. */
(function () {
  'use strict';
  if (window.siTerm) return;
  var API = '/_si/term/api/';
  var Term = window.Terminal, Fit = window.FitAddon && window.FitAddon.FitAddon;
  if (!Term || !Fit) { console.error('si terminal: xterm.js failed to load'); return; }

  var style = document.createElement('style');
  style.textContent =
    '.si-term{position:fixed;left:0;right:0;bottom:0;height:55vh;z-index:9000;' +
    'display:flex;flex-direction:column;background:#0b0e14;color:#e6e6e6;' +
    'font-family:ui-monospace,SFMono-Regular,Menlo,Consolas,monospace;' +
    'box-shadow:0 -12px 40px rgba(0,0,0,.5);transform:translateY(100%);' +
    'transition:transform .22s ease;border-top:1px solid #1d2536}' +
    '.si-term.is-open{transform:translateY(0)}' +
    '.si-term-bar{display:flex;align-items:stretch;gap:2px;background:#11151f;' +
    'padding:4px 6px;border-bottom:1px solid #1d2536;flex:0 0 auto}' +
    '.si-term-tabs{display:flex;gap:2px;overflow:auto;flex:1 1 auto}' +
    '.si-term-tab{display:flex;align-items:center;gap:8px;padding:4px 10px;' +
    'border-radius:6px 6px 0 0;background:#171c28;color:#9aa6bf;cursor:pointer;' +
    'font-size:13px;white-space:nowrap;border:0}' +
    '.si-term-tab.is-active{background:#0b0e14;color:#e6e6e6}' +
    '.si-term-tab .x{opacity:.5;font-size:15px;line-height:1}' +
    '.si-term-tab .x:hover{opacity:1;color:#ff8a8a}' +
    '.si-term-btn{background:#171c28;color:#9aa6bf;border:0;border-radius:6px;' +
    'cursor:pointer;font-size:15px;padding:4px 12px;line-height:1}' +
    '.si-term-btn:hover{color:#e6e6e6;background:#1d2536}' +
    '.si-term-body{display:flex;flex:1 1 auto;min-height:0}' +
    '.si-term-main{position:relative;flex:1 1 auto;min-width:0;min-height:0}' +
    '.si-term-pane{position:absolute;inset:0;display:none;padding:6px 8px}' +
    '.si-term-pane.is-active{display:block}' +
    '.si-term .xterm{height:100%}' +
    /* runbook drawer */
    '.si-term-btn.is-on{color:#0b0e14;background:#7dd3fc}' +
    '.si-rb{flex:0 0 340px;display:none;flex-direction:column;min-height:0;' +
    'background:#0d1017;border-left:1px solid #1d2536}' +
    '.si-rb.is-open{display:flex}' +
    '.si-rb-head{flex:0 0 auto;padding:6px 8px;border-bottom:1px solid #1d2536}' +
    '.si-rb-find{width:100%;box-sizing:border-box;background:#11151f;color:#e6e6e6;' +
    'border:1px solid #1d2536;border-radius:6px;padding:5px 8px;font:inherit;font-size:13px}' +
    '.si-rb-find::placeholder{color:#5b6680}' +
    '.si-rb-list{flex:1 1 auto;overflow:auto;padding:4px 0}' +
    '.si-rb-cat{font-size:11px;letter-spacing:.08em;text-transform:uppercase;' +
    'color:#7f8bab;padding:10px 10px 4px}' +
    '.si-rb-item{padding:6px 10px;cursor:pointer;border-left:2px solid transparent}' +
    '.si-rb-item:hover{background:#141a26;border-left-color:#7dd3fc}' +
    '.si-rb-item.flash{background:#1b3a2a}' +
    '.si-rb-label{font-family:ui-sans-serif,system-ui,sans-serif;font-size:13px;color:#e6e6e6}' +
    '.si-rb-desc{font-family:ui-sans-serif,system-ui,sans-serif;font-size:11px;color:#7f8bab;margin:1px 0 3px}' +
    '.si-rb-cmd{font-size:12px;white-space:pre-wrap;word-break:break-word;color:#c7d0e0;' +
    'background:#11151f;border-radius:5px;padding:4px 6px}' +
    '.si-rb-hint{font-size:11px;color:#5b6680;padding:6px 10px}' +
    /* highlighter tokens, scoped dark for the drawer (deck theme must not leak in) */
    '.si-rb-cmd .si-tk-k{color:#7dd3fc}.si-rb-cmd .si-tk-s{color:#a5e075}' +
    '.si-rb-cmd .si-tk-n{color:#f5a05a}.si-rb-cmd .si-tk-c{color:#5b6680;font-style:italic}' +
    '.si-rb-cmd .si-tk-m{color:#c792ea}.si-rb-cmd .si-tk-f{color:#f5a05a}' +
    '.si-rb-cmd .si-tk-v{color:#ffd080}.si-rb-cmd .si-tk-y{color:#7dd3fc}';
  document.head.appendChild(style);

  var root = document.createElement('div');
  root.className = 'si-term';
  var hasRunbook = window.SI_RUNBOOK && window.SI_RUNBOOK.categories && window.SI_RUNBOOK.categories.length;
  root.innerHTML =
    '<div class="si-term-bar">' +
    '<div class="si-term-tabs"></div>' +
    (hasRunbook ? '<button class="si-term-btn" data-act="runbook" title="Runbook (cheat-sheet)">⌘ Runbook</button>' : '') +
    '<button class="si-term-btn" data-act="new" title="New tab">+</button>' +
    '<button class="si-term-btn" data-act="hide" title="Hide (`)">✕</button>' +
    '</div><div class="si-term-body"><div class="si-term-main"></div></div>';
  document.body.appendChild(root);
  var tabsEl = root.querySelector('.si-term-tabs');
  var bodyEl = root.querySelector('.si-term-main');

  var tabs = []; // { id, term, fit, pane, tabEl }
  var active = -1;
  var seq = 0;

  function post(path) { return fetch(API + path, { method: 'POST' }).catch(function () {}); }

  function select(i) {
    active = i;
    tabs.forEach(function (t, k) {
      var on = k === i;
      t.pane.classList.toggle('is-active', on);
      t.tabEl.classList.toggle('is-active', on);
      if (on) { try { t.fit.fit(); } catch (e) {} sendResize(t); t.term.focus(); }
    });
  }

  function sendResize(t) {
    if (t.term.cols && t.term.rows) post('resize?id=' + t.id + '&cols=' + t.term.cols + '&rows=' + t.term.rows);
  }

  function stream(t) {
    fetch(API + 'out?id=' + t.id).then(function (res) {
      if (!res.body) return;
      var reader = res.body.getReader();
      (function pump() {
        reader.read().then(function (r) {
          if (r.done) { t.term.write('\r\n\x1b[90m[process exited]\x1b[0m\r\n'); return; }
          t.term.write(r.value);
          pump();
        }).catch(function () {});
      })();
    }).catch(function () {});
  }

  function newTab() {
    var local = ++seq;
    var tabEl = document.createElement('button');
    tabEl.className = 'si-term-tab';
    tabEl.innerHTML = '<span class="label">shell ' + local + '</span><span class="x" title="Close">×</span>';
    tabsEl.appendChild(tabEl);
    var pane = document.createElement('div');
    pane.className = 'si-term-pane';
    bodyEl.appendChild(pane);

    var term = new Term({ cursorBlink: true, fontSize: 14, scrollback: 5000,
      fontFamily: '"MesloLGS NF","MesloLGS Nerd Font","Meslo LG S Nerd Font","FiraCode Nerd Font","Hack Nerd Font","JetBrainsMono Nerd Font","SauceCodePro Nerd Font",ui-monospace,Menlo,Consolas,monospace',
      theme: { background: '#0b0e14', foreground: '#e6e6e6', cursor: '#7dd3fc' } });
    var fit = new Fit();
    term.loadAddon(fit);
    term.open(pane);

    var t = { id: null, term: term, fit: fit, pane: pane, tabEl: tabEl };
    tabs.push(t);
    var idx = tabs.length - 1;
    select(idx);

    fetch(API + 'new', { method: 'POST' }).then(function (r) { return r.text(); }).then(function (id) {
      t.id = id;
      try { fit.fit(); } catch (e) {}
      sendResize(t);
      stream(t);
      term.onData(function (d) { if (t.id) fetch(API + 'in?id=' + t.id, { method: 'POST', body: d }); });
      term.onResize(function () { sendResize(t); });
    }).catch(function () { term.write('\r\n\x1b[31msi: could not start a shell\x1b[0m\r\n'); });

    tabEl.addEventListener('click', function (e) {
      if (e.target.classList.contains('x')) { closeTab(t); return; }
      select(tabs.indexOf(t));
    });
  }

  function closeTab(t) {
    if (t.id) post('close?id=' + t.id);
    try { t.term.dispose(); } catch (e) {}
    t.pane.remove(); t.tabEl.remove();
    var i = tabs.indexOf(t);
    if (i >= 0) tabs.splice(i, 1);
    if (tabs.length === 0) { hide(); return; }
    select(Math.min(active, tabs.length - 1));
  }

  function show() {
    root.classList.add('is-open');
    if (tabs.length === 0) newTab();
    else select(active < 0 ? 0 : active);
  }
  function hide() { root.classList.remove('is-open'); }
  function toggle() { if (root.classList.contains('is-open')) hide(); else show(); }

  root.querySelector('[data-act="new"]').addEventListener('click', newTab);
  root.querySelector('[data-act="hide"]').addEventListener('click', hide);
  window.addEventListener('resize', function () { if (active >= 0 && tabs[active]) { try { tabs[active].fit.fit(); } catch (e) {} sendResize(tabs[active]); } });

  /* ---------- runbook drawer: a command cheat-sheet that types into the shell ---------- */
  var rbPanel = null;
  function sendToShell(text) {
    var t = tabs[active];
    if (!t || !t.id) return; // shell still starting; click again
    fetch(API + 'in?id=' + t.id, { method: 'POST', body: text }).catch(function () {});
    t.term.focus();
  }
  function buildRunbook() {
    if (!hasRunbook) return;
    rbPanel = document.createElement('div');
    rbPanel.className = 'si-rb';
    var open = true;
    try { open = JSON.parse(localStorage.getItem('si-runbook-open') || 'true'); } catch (e) {}
    rbPanel.classList.toggle('is-open', open);
    var btn = root.querySelector('[data-act="runbook"]');
    if (btn) btn.classList.toggle('is-on', open);

    var head = document.createElement('div');
    head.className = 'si-rb-head';
    var find = document.createElement('input');
    find.className = 'si-rb-find'; find.type = 'search'; find.placeholder = 'Filter commands…';
    head.appendChild(find);
    var list = document.createElement('div');
    list.className = 'si-rb-list';
    var hint = document.createElement('div');
    hint.className = 'si-rb-hint';
    hint.textContent = 'Click to type · Shift-click to run';

    window.SI_RUNBOOK.categories.forEach(function (cat) {
      var ch = document.createElement('div');
      ch.className = 'si-rb-cat'; ch.textContent = cat.name;
      list.appendChild(ch);
      cat._catEl = ch; cat._items = [];
      (cat.entries || []).forEach(function (e) {
        var it = document.createElement('div');
        it.className = 'si-rb-item';
        it.innerHTML = '<div class="si-rb-label"></div>' +
          (e.desc ? '<div class="si-rb-desc"></div>' : '') +
          '<div class="si-rb-cmd">' + (e.html || '') + '</div>';
        it.querySelector('.si-rb-label').textContent = e.label || e.cmd;
        if (e.desc) it.querySelector('.si-rb-desc').textContent = e.desc;
        it._hay = ((e.label || '') + ' ' + (e.desc || '') + ' ' + (e.cmd || '')).toLowerCase();
        it.addEventListener('click', function (ev) {
          sendToShell(e.cmd + (ev.shiftKey || ev.metaKey || ev.ctrlKey ? '\r' : ''));
          it.classList.add('flash'); setTimeout(function () { it.classList.remove('flash'); }, 220);
        });
        list.appendChild(it); cat._items.push(it);
      });
    });

    find.addEventListener('input', function () {
      var q = find.value.trim().toLowerCase();
      window.SI_RUNBOOK.categories.forEach(function (cat) {
        var any = false;
        cat._items.forEach(function (it) {
          var ok = !q || it._hay.indexOf(q) >= 0;
          it.style.display = ok ? '' : 'none';
          if (ok) any = true;
        });
        cat._catEl.style.display = any ? '' : 'none';
      });
    });

    rbPanel.appendChild(head); rbPanel.appendChild(list); rbPanel.appendChild(hint);
    root.querySelector('.si-term-body').appendChild(rbPanel);
  }
  function toggleRunbook() {
    if (!rbPanel) return;
    var open = !rbPanel.classList.contains('is-open');
    rbPanel.classList.toggle('is-open', open);
    var btn = root.querySelector('[data-act="runbook"]');
    if (btn) btn.classList.toggle('is-on', open);
    try { localStorage.setItem('si-runbook-open', JSON.stringify(open)); } catch (e) {}
    if (active >= 0 && tabs[active]) { try { tabs[active].fit.fit(); } catch (e) {} sendResize(tabs[active]); }
  }
  buildRunbook();
  var rbBtn = root.querySelector('[data-act="runbook"]');
  if (rbBtn) rbBtn.addEventListener('click', toggleRunbook);

  window.siTerm = { toggle: toggle, show: show, hide: hide, runbook: toggleRunbook };
})();
