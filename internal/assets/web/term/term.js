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
    '.si-rb-item{position:relative;padding:6px 10px;cursor:pointer;border-left:2px solid transparent}' +
    '.si-rb-acts{float:right;display:flex;gap:4px;margin-left:6px}' +
    '.si-rb-acts button,.si-dm-acts button{background:#1b2436;color:#c8d0e0;border:1px solid #2a3550;border-radius:4px;' +
    'padding:1px 8px;font:600 11px ui-sans-serif,system-ui,sans-serif;cursor:pointer}' +
    '.si-rb-acts button:hover,.si-dm-acts button:hover{border-color:#7dd3fc;color:#fff}' +
    '.si-rb-acts button.run,.si-dm-acts button.run{background:#14532d;border-color:#1f7a44;color:#d9fbe5}' +
    '.si-rb-acts button.done,.si-dm-acts button.done{background:#1b3a2a;border-color:#3fb47a;color:#d9fbe5}' +
    /* demos panel: a drop-down of demo scripts, the chosen one as a document */
    '.si-dm{flex:0 0 min(620px,50%);display:none;flex-direction:column;min-height:0;' +
    'background:#0d1017;border-left:1px solid #1d2536}' +
    '.si-dm.is-open{display:flex}' +
    '.si-dm-head{flex:0 0 auto;display:flex;gap:6px;padding:6px 8px;border-bottom:1px solid #1d2536}' +
    '.si-dm-pick{flex:1 1 auto;min-width:0;background:#11151f;color:#e6e6e6;border:1px solid #1d2536;' +
    'border-radius:6px;padding:5px 8px;font:13px ui-sans-serif,system-ui,sans-serif}' +
    '.si-dm-doc{flex:1 1 auto;overflow:auto;padding:4px 14px 40px;font:13px/1.5 ui-sans-serif,system-ui,sans-serif;color:#c8d0e0}' +
    '.si-dm-doc h1{font-size:17px;color:#fff;margin:12px 0 8px}.si-dm-doc h2{font-size:15px;color:#fff;margin:18px 0 6px;' +
    'padding-top:10px;border-top:1px solid #1d2536}.si-dm-doc h3{font-size:13px;color:#e6e6e6;margin:14px 0 4px}' +
    '.si-dm-doc p,.si-dm-doc li{margin:4px 0}.si-dm-doc ul,.si-dm-doc ol{padding-left:20px;margin:4px 0}' +
    '.si-dm-doc a{color:#7dd3fc}.si-dm-doc strong{color:#fff}' +
    '.si-dm-doc :not(pre)>code{font:12px ui-monospace,Menlo,monospace;background:#171c28;border:0;border-radius:3px;padding:0 3px;color:#e6e6e6}' +
    '.si-dm-doc blockquote{margin:8px 0;padding:4px 10px;border-left:3px solid #f59e0b;background:#17140c;color:#f1e3c4}' +
    '.si-dm-doc table{border-collapse:collapse;margin:6px 0;font-size:12px}' +
    '.si-dm-doc td,.si-dm-doc th{border:1px solid #1d2536;padding:3px 6px;text-align:left;vertical-align:top}' +
    '.si-dm-view{flex:0 0 auto;background:#11151f;color:#e6e6e6;border:1px solid #1d2536;border-radius:6px;' +
    'padding:5px 9px;font:12px ui-sans-serif,system-ui,sans-serif;cursor:pointer}.si-dm-view:hover{border-color:#7dd3fc}' +
    /* Commands view (the default): headings and runnable blocks only; the full script is one click away */
    '.si-dm-doc.si-dm-cmds>:not(h1,h2,h3,h4,h5,h6,.si-dm-block),.si-dm-doc.si-dm-cmds>.si-dm-bare{display:none}' +
    '.si-dm-block{position:relative;margin:6px 0}' +
    '.si-dm-acts{position:absolute;top:4px;right:4px;display:flex;gap:4px}' +
    '.si-dm-code{margin:0;font:12px/1.45 ui-monospace,Menlo,monospace;white-space:pre-wrap;word-break:break-word;' +
    'background:#11151f;border:1px solid #1d2536;border-radius:5px;padding:6px 8px;color:#c7d0e0}' +
    '.si-dm-code.si-dm-run{border-left:3px solid #3fb47a;padding-right:110px}' +
    '.si-dm-code code{all:unset;font:inherit;white-space:inherit;color:inherit}' +
    '.si-dm-code:not(.si-dm-run){color:#8f9bb3;background:#0f131b}' +
    '.si-rb-item:hover{background:#141a26;border-left-color:#7dd3fc}' +
    '.si-rb-item.flash{background:#1b3a2a}' +
    '.si-rb-label{font-family:ui-sans-serif,system-ui,sans-serif;font-size:13px;color:#e6e6e6}' +
    '.si-rb-desc{font-family:ui-sans-serif,system-ui,sans-serif;font-size:11px;color:#7f8bab;margin:1px 0 3px}' +
    '.si-rb-cmd{font-size:12px;white-space:pre-wrap;word-break:break-word;color:#c7d0e0;' +
    'background:#11151f;border-radius:5px;padding:4px 6px}' +
    '.si-rb-title{padding:6px 10px 0;font:600 12px ui-sans-serif,system-ui,sans-serif;color:#9aa7c0}' +
    '.si-rb-var{background:#2a2410;color:#ffd080;border-radius:3px;padding:0 3px}' +
    '.si-rb-form{flex:0 0 auto;padding:8px 10px;border-bottom:1px solid #1d2536;background:#11151f;' +
    'font-family:ui-sans-serif,system-ui,sans-serif;font-size:12px;color:#c8d0e0}' +
    '.si-rb-form label{display:block;margin:4px 0 2px;color:#9aa7c0}' +
    '.si-rb-form input{width:100%;box-sizing:border-box;background:#0b0e14;color:#e6e6e6;' +
    'border:1px solid #2a3550;border-radius:4px;padding:4px 6px;font:12px ui-monospace,Menlo,monospace}' +
    '.si-rb-form input:focus{outline:none;border-color:#7dd3fc}' +
    '.si-rb-form .si-rb-fcmd{margin:6px 0;font:11px ui-monospace,Menlo,monospace;color:#7d8aa5;word-break:break-word}' +
    '.si-rb-form button{background:#1b2436;color:#e6e6e6;border:1px solid #2a3550;border-radius:4px;' +
    'padding:3px 10px;margin-right:6px;font-size:12px;cursor:pointer}' +
    '.si-rb-form button.pri{background:#7dd3fc;color:#0b0e14;border-color:#7dd3fc}' +
    '.si-rb-hint{font-size:11px;color:#5b6680;padding:6px 10px}' +
    /* highlighter tokens, scoped dark for the drawer (deck theme must not leak in) */
    ':is(.si-rb-cmd,.si-dm-code) .si-tk-k{color:#7dd3fc}:is(.si-rb-cmd,.si-dm-code) .si-tk-s{color:#a5e075}' +
    ':is(.si-rb-cmd,.si-dm-code) .si-tk-n{color:#f5a05a}:is(.si-rb-cmd,.si-dm-code) .si-tk-c{color:#5b6680;font-style:italic}' +
    ':is(.si-rb-cmd,.si-dm-code) .si-tk-m{color:#c792ea}:is(.si-rb-cmd,.si-dm-code) .si-tk-f{color:#f5a05a}' +
    ':is(.si-rb-cmd,.si-dm-code) .si-tk-v{color:#ffd080}:is(.si-rb-cmd,.si-dm-code) .si-tk-y{color:#7dd3fc}';
  document.head.appendChild(style);

  var root = document.createElement('div');
  root.className = 'si-term';
  var hasRunbook = window.SI_RUNBOOK && window.SI_RUNBOOK.categories && window.SI_RUNBOOK.categories.length;
  var hasDemos = window.SI_DEMOS && window.SI_DEMOS.length;
  root.innerHTML =
    '<div class="si-term-bar">' +
    '<div class="si-term-tabs"></div>' +
    (hasDemos ? '<button class="si-term-btn" data-act="demos" title="Demo scripts">▶ Demos</button>' : '') +
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
          t.lastOut = Date.now();
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

    var t = { id: null, term: term, fit: fit, pane: pane, tabEl: tabEl, born: Date.now() };
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
  /* Put a command at the prompt (run: also press Enter); Copy also puts it on
     the clipboard. The ready signal is bracketed-paste mode: zsh and bash 5.1+
     turn it on when the prompt is ready to edit (and off while a command runs),
     so the text goes in as a paste, shown whole (multi-line included) and run
     only on Enter. Typing earlier (while .zshrc loads) would echo it above the
     prompt and run a multi-line block line by line; a heavy .zshrc can take
     several seconds and print nothing meanwhile. So: prompt ready → paste now;
     the tab was ready before but a command runs now → type into that command;
     still starting → wait (up to 20 s; after 15 s a shell that never turns
     bracketed paste on, like sh or macOS bash 3.2, gets plain keystrokes once
     quiet). done() runs when the text has been sent. */
  function sendToShell(text, run, done) {
    text = String(text).replace(/\n+$/, '');
    if (!run && navigator.clipboard) navigator.clipboard.writeText(text).catch(function () {});
    var t = tabs[active];
    if (!t) return;
    var start = Date.now();
    (function wait() {
      var bp = !!(t.term.modes && t.term.modes.bracketedPasteMode), now = Date.now();
      if (bp) t.everBP = true;
      var quiet = function (ms) { return t.lastOut && now - t.lastOut > ms; };
      var ready = t.id && (bp ? quiet(80) : t.everBP || (now - t.born > 15000 && quiet(250)));
      if (!ready && now - start < 20000) { setTimeout(wait, 60); return; }
      if (!t.id) return; // the shell never started
      var body = (bp ? '\x1b[200~' + text + '\x1b[201~' : text) + (run ? '\r' : '');
      fetch(API + 'in?id=' + t.id, { method: 'POST', body: body }).catch(function () {});
      if (done) done();
    })();
    t.term.focus();
  }
  function flash(b, label) {
    if (!b._label) b._label = b.textContent;
    b.textContent = label; b.classList.add('done');
    clearTimeout(b._t); b._t = setTimeout(function () { b.textContent = b._label; b.classList.remove('done'); }, 900);
  }
  function actButtons(cls, getText, before) {
    var box = document.createElement('div'); box.className = cls;
    [['Copy', false], ['Run', true]].forEach(function (a) {
      var b = document.createElement('button'); b.type = 'button'; b.textContent = a[0];
      b.title = a[1] ? 'Put it at the prompt and press Enter' : 'Put it at the prompt (not run) and on the clipboard';
      if (a[1]) b.className = 'run';
      b.addEventListener('click', function (ev) {
        ev.stopPropagation();
        if (before && before(a[1])) return;
        if (!b._label) b._label = b.textContent;
        b.textContent = 'Waiting…';
        sendToShell(getText(), a[1], function () { flash(b, a[1] ? 'Sent' : 'Copied'); }); b.blur();
      });
      box.appendChild(b);
    });
    return box;
  }
  var RB_VAR = /\{\{([A-Za-z_][A-Za-z0-9_-]*)(?::([^}]*))?\}\}/g; // same as Go
  var RB_TPL = { end: 1, else: 1, break: 1, continue: 1, nil: 1, true: 1, false: 1 }; // Go-template keywords, not placeholders (same as Go)
  function rbSaved() { try { return JSON.parse(sessionStorage.getItem('si-rb-vars') || '{}') || {}; } catch (e) { return {}; } }
  function rbFill(cmd, vals) {
    return cmd.replace(RB_VAR, function (m, n) { return !RB_TPL[n] && Object.prototype.hasOwnProperty.call(vals, n) ? vals[n] : m; });
  }
  var rbForm = null;
  function closeForm() { if (rbForm) { rbForm.remove(); rbForm = null; } }
  function openForm(e, ref) {
    closeForm();
    var saved = rbSaved(), f = document.createElement('div');
    f.className = 'si-rb-form';
    var inputs = [];
    e.vars.forEach(function (v) {
      var l = document.createElement('label'), i = document.createElement('input');
      l.textContent = v.name; i.type = 'text'; i.spellcheck = false;
      i.value = Object.prototype.hasOwnProperty.call(saved, v.name) ? saved[v.name] : (v.default || '');
      l.appendChild(i); f.appendChild(l); inputs.push([v.name, i]);
    });
    var pv = document.createElement('div'); pv.className = 'si-rb-fcmd'; f.appendChild(pv);
    function vals() { var o = {}; inputs.forEach(function (p) { o[p[0]] = p[1].value; }); return o; }
    function upd() { pv.textContent = rbFill(e.cmd, vals()); }
    function go(run) {
      var o = vals(), all = rbSaved();
      for (var k in o) all[k] = o[k];
      try { sessionStorage.setItem('si-rb-vars', JSON.stringify(all)); } catch (x) {}
      sendToShell(rbFill(e.cmd, o), run);
      closeForm();
    }
    function btn(txt, cls, fn) {
      var b = document.createElement('button'); b.type = 'button'; b.textContent = txt;
      if (cls) b.className = cls; b.addEventListener('click', fn); return b;
    }
    var row = document.createElement('div');
    row.appendChild(btn('Copy', 'pri', function () { go(false); }));
    row.appendChild(btn('Run', '', function () { go(true); }));
    row.appendChild(btn('Cancel', '', closeForm));
    f.appendChild(row);
    f.addEventListener('input', upd);
    f.addEventListener('keydown', function (ev) {
      ev.stopPropagation();
      if (ev.key === 'Escape') { ev.preventDefault(); closeForm(); if (active >= 0 && tabs[active]) tabs[active].term.focus(); }
      // Enter submits from a field only; on a button it does that button's own click
      else if (ev.key === 'Enter' && ev.target.tagName === 'INPUT') { ev.preventDefault(); go(ev.shiftKey || ev.metaKey || ev.ctrlKey); }
    });
    upd();
    rbPanel.insertBefore(f, rbPanel.querySelector('.si-rb-list'));
    rbForm = f;
    inputs[0][1].focus(); inputs[0][1].select();
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
    hint.textContent = 'Copy puts a command at the prompt (and on the clipboard) · Run also presses Enter';

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
        it.insertBefore(actButtons('si-rb-acts', function () { return e.cmd; }, function (run) {
          if (e.vars && e.vars.length) { openForm(e, it); return true; } // the form has its own Copy / Run
          return false;
        }), it.firstChild);
        if (e.desc) it.querySelector('.si-rb-desc').textContent = e.desc;
        it._hay = ((e.label || '') + ' ' + (e.desc || '') + ' ' + (e.cmd || '')).toLowerCase();
        it.addEventListener('click', function (ev) {
          if (e.vars && e.vars.length) { openForm(e, it); return; }
          sendToShell(e.cmd, ev.shiftKey || ev.metaKey || ev.ctrlKey); // the row itself: Copy (Shift: Run)
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

    if (window.SI_RUNBOOK.name) {
      var ttl = document.createElement('div');
      ttl.className = 'si-rb-title'; ttl.textContent = window.SI_RUNBOOK.name;
      rbPanel.appendChild(ttl);
    }
    rbPanel.appendChild(head); rbPanel.appendChild(list); rbPanel.appendChild(hint);
    root.querySelector('.si-term-body').appendChild(rbPanel);
  }
  /* ---------- demos panel: pick a demo script, read it, Copy / Run its commands ---------- */
  var dmPanel = null;
  function buildDemos() {
    if (!hasDemos) return;
    dmPanel = document.createElement('div'); dmPanel.className = 'si-dm';
    var head = document.createElement('div'); head.className = 'si-dm-head';
    var pick = document.createElement('select'); pick.className = 'si-dm-pick'; pick.setAttribute('aria-label', 'Demo');
    window.SI_DEMOS.forEach(function (d, i) { var o = document.createElement('option'); o.value = i; o.textContent = d.title; pick.appendChild(o); });
    head.appendChild(pick);
    var doc = document.createElement('div'); doc.className = 'si-dm-doc';
    // Commands (default) shows only headings and runnable blocks, so the Say
    // lines and notes stay in the speaker's own window; Full shows the script.
    var full = false; try { full = localStorage.getItem('si-demos-view') === 'full'; } catch (e) {}
    var view = document.createElement('button'); view.type = 'button'; view.className = 'si-dm-view';
    function setView() {
      doc.classList.toggle('si-dm-cmds', !full);
      view.textContent = full ? 'Commands only' : 'Full script';
      view.title = full ? 'Show only the commands' : 'Show the whole script, with its notes';
    }
    view.addEventListener('click', function () {
      full = !full; setView();
      try { localStorage.setItem('si-demos-view', full ? 'full' : 'cmds'); } catch (e) {}
    });
    head.appendChild(view);
    function showDemo(i) {
      var d = window.SI_DEMOS[i] || window.SI_DEMOS[0];
      doc.innerHTML = d.html; doc.scrollTop = 0;
      Array.prototype.forEach.call(doc.querySelectorAll('pre.si-dm-run'), function (pre) {
        var wrap = document.createElement('div'); wrap.className = 'si-dm-block';
        pre.parentNode.insertBefore(wrap, pre); wrap.appendChild(pre);
        wrap.appendChild(actButtons('si-dm-acts', function () { return pre.textContent; }));
      });
      // a heading whose section (up to the next heading of its level or higher)
      // has no command is hidden in the Commands view
      var kids = Array.prototype.slice.call(doc.children), lvl = function (el) { var m = /^H(\d)$/.exec(el.tagName); return m ? +m[1] : 0; };
      kids.forEach(function (el, k) {
        var n = lvl(el); if (!n) return;
        var has = false;
        for (var j = k + 1; j < kids.length; j++) {
          var m = lvl(kids[j]); if (m && m <= n) break;
          if (kids[j].classList.contains('si-dm-block')) { has = true; break; }
        }
        if (!has) el.classList.add('si-dm-bare');
      });
      setView();
      try { localStorage.setItem('si-demo-pick', d.file); } catch (e) {}
    }
    var saved = null; try { saved = localStorage.getItem('si-demo-pick'); } catch (e) {}
    var at = 0; window.SI_DEMOS.forEach(function (d, i) { if (d.file === saved) at = i; });
    pick.value = at; showDemo(at);
    pick.addEventListener('change', function () { showDemo(+pick.value); });
    // keys typed in the panel (the drop-down, find-in-page) must not drive the deck
    dmPanel.addEventListener('keydown', function (ev) { ev.stopPropagation(); });
    dmPanel.appendChild(head); dmPanel.appendChild(doc);
    var open = false; try { open = JSON.parse(localStorage.getItem('si-demos-open') || 'false'); } catch (e) {}
    root.querySelector('.si-term-body').appendChild(dmPanel);
    if (open) setPanel('demos', true);
  }
  /* one side panel at a time: demos or runbook */
  function setPanel(which, open) {
    [['demos', dmPanel, 'si-demos-open'], ['runbook', rbPanel, 'si-runbook-open']].forEach(function (p) {
      if (!p[1]) return;
      var on = p[0] === which ? open : (open ? false : p[1].classList.contains('is-open'));
      p[1].classList.toggle('is-open', on);
      var b = root.querySelector('[data-act="' + p[0] + '"]'); if (b) b.classList.toggle('is-on', on);
      try { localStorage.setItem(p[2], JSON.stringify(on)); } catch (e) {}
    });
    if (active >= 0 && tabs[active]) { try { tabs[active].fit.fit(); } catch (e) {} sendResize(tabs[active]); }
  }
  function toggleDemos() { if (dmPanel) setPanel('demos', !dmPanel.classList.contains('is-open')); }
  function toggleRunbook() {
    if (!rbPanel) return;
    if (dmPanel) { setPanel('runbook', !rbPanel.classList.contains('is-open')); return; }
    var open = !rbPanel.classList.contains('is-open');
    rbPanel.classList.toggle('is-open', open);
    var btn = root.querySelector('[data-act="runbook"]');
    if (btn) btn.classList.toggle('is-on', open);
    try { localStorage.setItem('si-runbook-open', JSON.stringify(open)); } catch (e) {}
    if (active >= 0 && tabs[active]) { try { tabs[active].fit.fit(); } catch (e) {} sendResize(tabs[active]); }
  }
  buildRunbook();
  buildDemos();
  var rbBtn = root.querySelector('[data-act="runbook"]');
  if (rbBtn) rbBtn.addEventListener('click', toggleRunbook);
  var dmBtn = root.querySelector('[data-act="demos"]');
  if (dmBtn) dmBtn.addEventListener('click', toggleDemos);

  window.siTerm = { toggle: toggle, show: show, hide: hide, runbook: toggleRunbook, demos: toggleDemos };
})();
