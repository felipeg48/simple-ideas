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
    '.si-term.is-max{height:100vh;transition:transform .22s ease,height .22s ease}' +
    '.si-term-win{display:flex;gap:2px;flex:0 0 auto;margin-right:6px;padding-right:6px;border-right:1px solid #1d2536}' +
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
    '.si-term-acts{flex:0 0 42px;display:flex;flex-direction:column;align-items:center;gap:4px;padding:6px 0;background:#11151f;border-left:1px solid #1d2536;order:9}' +
    '.si-term-act{position:relative;width:34px;height:34px;display:grid;place-items:center;background:transparent;color:#7d8aa5;border:0;border-radius:6px;cursor:pointer}' +
    '.si-term-act:hover{color:#e6e6e6;background:#1d2536}.si-term-act:focus-visible{outline:2px solid #7dd3fc}' +
    '.si-term-act.is-on{color:#e6e6e6}.si-term-act.is-on::before{content:"";position:absolute;right:-4px;top:6px;bottom:6px;width:2px;border-radius:2px;background:#7dd3fc}' +
    /* side panels resize from their left edge */
    '.si-rb,.si-dm{position:relative}.si-term-grip{position:absolute;left:-3px;top:0;bottom:0;width:6px;cursor:ew-resize;z-index:3}' +
    '.si-term-grip:hover,.si-term-grip.is-drag{background:#7dd3fc55}.si-term.is-resizing .xterm{pointer-events:none}' +
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
    /* files panel: a read-only tree of a folder and a viewer for one file */
    '.si-fs-pane{padding:0}.si-fs{display:flex;flex-direction:column;height:100%;min-height:0;background:#0d1017;' +
    'font:12px ui-sans-serif,system-ui,sans-serif;color:#c8d0e0}' +
    '.si-fs-head{flex:0 0 auto;display:flex;gap:4px;padding:6px 8px;border-bottom:1px solid #1d2536}' +
    '.si-fs-head input{flex:1 1 auto;min-width:0;background:#11151f;color:#e6e6e6;border:1px solid #1d2536;border-radius:6px;padding:5px 8px;font:12px ui-monospace,Menlo,monospace}' +
    '.si-fs-head input:focus{outline:none;border-color:#7dd3fc}' +
    '.si-fs-head button,.si-fs-vh button{flex:0 0 auto;background:#11151f;color:#e6e6e6;border:1px solid #1d2536;border-radius:6px;padding:4px 8px;font:12px ui-sans-serif,system-ui,sans-serif;cursor:pointer}' +
    '.si-fs-head button:hover,.si-fs-vh button:hover{border-color:#7dd3fc}' +
    '.si-fs-split{flex:1 1 auto;display:flex;min-height:0}' +
    '.si-fs-side{flex:0 0 min(340px,32%);display:flex;flex-direction:column;min-height:0;border-right:1px solid #1d2536}' +
    '.si-fs-tree{flex:1 1 auto;overflow:auto;padding:4px 0}.si-fs-tree:focus{outline:none}' +
    '.si-fs-tb{flex:0 0 auto;display:flex;gap:2px;padding:3px 6px;border-bottom:1px solid #1d2536}' +
    '.si-fs-tb button{width:26px;height:24px;display:grid;place-items:center;background:transparent;color:#7d8aa5;border:0;border-radius:4px;cursor:pointer;padding:0}' +
    '.si-fs-tb button:hover{color:#e6e6e6;background:#1d2536}.si-fs-tb button:disabled{opacity:.3;cursor:default;background:transparent;color:#7d8aa5}' +
    '.si-fs-tb button:focus-visible{outline:2px solid #7dd3fc}' +
    '.si-fs-row input{flex:1 1 auto;min-width:0;background:#0b0e14;color:#e6e6e6;border:1px solid #7dd3fc;border-radius:3px;padding:0 4px;font:inherit;outline:none}' +
    '.si-fs-row input.bad{border-color:#ff8a8a}' +
    '.si-fs-menu{position:fixed;z-index:9100;min-width:170px;padding:4px;background:#11151f;border:1px solid #2a3550;border-radius:6px;' +
    'box-shadow:0 8px 24px rgba(0,0,0,.55);font:12px ui-sans-serif,system-ui,sans-serif;color-scheme:dark}' +
    '.si-fs-menu button{display:block;width:100%;text-align:left;background:transparent;color:#c8d0e0;border:0;border-radius:4px;padding:5px 10px;cursor:pointer;font:inherit}' +
    '.si-fs-menu button:hover,.si-fs-menu button:focus{background:#1b2436;color:#fff;outline:none}.si-fs-menu button.danger{color:#ff8a8a}' +
    '.si-fs-menu hr{border:0;border-top:1px solid #1d2536;margin:4px 2px}' +
    '.si-fs-ask button.danger{background:#5a1d1d;border-color:#a33a3a;color:#ffd6d6}.si-fs-ask .p{font:12px ui-monospace,Menlo,monospace;user-select:all;color:#e6e6e6}' +
    '.si-fs-vh button.is-on{background:#14532d;border-color:#3fb47a;color:#d9fbe5}' +
    '.si-fs-row{display:flex;align-items:center;gap:5px;padding:2px 8px;white-space:nowrap;cursor:pointer;color:#c8d0e0}' +
    '.si-fs-row:hover{background:#151b27}.si-fs-row.is-on{background:#1b2436;color:#fff}' +
    '.si-fs-row .ic{flex:0 0 1em;color:#7d8aa5;text-align:center}.si-fs-row.dot{opacity:.6}' +
    '.si-fs-row .nm{overflow:hidden;text-overflow:ellipsis}.si-fs-row .sz{margin-left:auto;padding-left:8px;color:#5b6680;font-size:11px}' +
    '.si-fs-view{flex:1 1 auto;display:flex;flex-direction:column;min-width:0}' +
    '.si-fs-vh{flex:0 0 auto;display:flex;align-items:center;gap:6px;padding:5px 8px;border-bottom:1px solid #1d2536;color:#9aa7c0}' +
    '.si-fs-vh .nm{flex:1 1 auto;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;color:#e6e6e6;font-weight:600}' +
    '.si-fs-body{flex:1 1 auto;overflow:auto;margin:0}' +
    '.si-fs-code{margin:0;padding:6px 0;font:12px/1.5 ui-monospace,Menlo,monospace;color:#c7d0e0;counter-reset:ln;white-space:pre;tab-size:4}' +
    '.si-fs-code .ln{display:block;padding:0 10px 0 0}.si-fs-code .ln::before{counter-increment:ln;content:counter(ln);display:inline-block;width:3.2em;margin-right:10px;text-align:right;color:#3d4760;user-select:none}' +
    '.si-fs-vh{flex-wrap:wrap}.si-fs-vh .nm{min-width:6em}.si-fs-vh .in{white-space:nowrap}' +
    '.si-fs-vh button.pri{background:#7dd3fc;color:#0b0e14;border-color:#7dd3fc}.si-fs-vh button[hidden]{display:none}' +
    '.si-fs-vh .ok{color:#3fb47a;font-weight:600}.si-fs-ed{height:100%}.si-fs-body.is-edit{overflow:hidden}' +
    '.si-fs-ask{flex:0 0 auto;display:flex;flex-wrap:wrap;align-items:center;gap:6px;padding:6px 8px;background:#2a2410;color:#ffd080;border-bottom:1px solid #4a3d14}' +
    '.si-fs-ask span{flex:1 1 auto}.si-fs-ask button{background:#1b2436;color:#e6e6e6;border:1px solid #2a3550;border-radius:4px;padding:3px 10px;font:12px ui-sans-serif,system-ui,sans-serif;cursor:pointer}' +
    '.si-fs-ask button.pri{background:#7dd3fc;color:#0b0e14;border-color:#7dd3fc}.si-fs-ask button:hover{border-color:#7dd3fc}' +
    '.si-fs-msg{padding:14px;color:#7d8aa5}.si-fs-img{max-width:100%;display:block;margin:10px auto;background:#fff}' +
    ':is(.si-fs-code) .si-tk-k{color:#7dd3fc}:is(.si-fs-code) .si-tk-s{color:#a5e075}:is(.si-fs-code) .si-tk-n{color:#f5a05a}' +
    ':is(.si-fs-code) .si-tk-c{color:#5b6680;font-style:italic}:is(.si-fs-code) .si-tk-m{color:#c792ea}:is(.si-fs-code) .si-tk-f{color:#f5a05a}' +
    ':is(.si-fs-code) .si-tk-v{color:#ffd080}:is(.si-fs-code) .si-tk-y{color:#7dd3fc}' +
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
    '<span class="si-term-win"><button class="si-term-btn" data-act="min" title="Minimise (` brings it back)">–</button>' +
    '<button class="si-term-btn" data-act="max" title="Maximise: the whole window (double-click the bar too)">□</button></span>' +
    '<div class="si-term-tabs"></div>' +
    '<button class="si-term-btn" data-act="new" title="New shell tab">+</button>' +
    '<button class="si-term-btn" data-act="hide" title="Hide (`)">✕</button>' +
    '</div><div class="si-term-body"><div class="si-term-main"></div>' +
    /* the activity bar on the right: one icon per tool, as in an editor */
    '<div class="si-term-acts" role="toolbar" aria-label="Terminal tools" aria-orientation="vertical">' +
    (hasDemos ? '<button class="si-term-act" data-act="demos" title="Demos: the demo scripts"><svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linejoin="round"><path d="M7 4.5v15l12-7.5z"/></svg></button>' : '') +
    (hasRunbook ? '<button class="si-term-act" data-act="runbook" title="Runbook: the command cheat-sheet"><svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M5 4h11a3 3 0 0 1 3 3v13H8a3 3 0 0 1-3-3z"/><path d="M5 17a3 3 0 0 1 3-3h11"/><path d="M9 8h6"/></svg></button>' : '') +
    '<button class="si-term-act" data-act="files" title="Viewer: a new tab with a folder\'s files, to read and edit"><svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M14 3H7a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V8z"/><path d="M14 3v5h5"/><path d="M9 13h6M9 17h4"/></svg></button>' +
    '</div></div>';
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
      if (on && !t.viewer) { lastShell = t; try { t.fit.fit(); } catch (e) {} sendResize(t); t.term.focus(); }
      else if (on && t.editor) t.editor.focus();
    });
  }

  function sendResize(t) {
    if (t.viewer) return;
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
    if (t.leave && !t.leave(function () { t.leave = null; closeTab(t); })) return; // a Viewer with unsaved edits asks first
    if (t.editor) try { t.editor.destroy(); } catch (e) {}
    if (t.stop) t.stop();
    if (t.id) post('close?id=' + t.id);
    if (t.term) try { t.term.dispose(); } catch (e) {}
    t.pane.remove(); t.tabEl.remove();
    var i = tabs.indexOf(t);
    if (i >= 0) tabs.splice(i, 1);
    if (tabs.length === 0) { hide(); return; }
    select(Math.min(active, tabs.length - 1));
  }

  function show(noShell) {
    root.classList.add('is-open');
    if (noShell === true) return;
    if (tabs.length === 0) newTab();
    else select(active < 0 ? 0 : active);
  }
  function hide() { root.classList.remove('is-open'); }
  function toggle() { if (root.classList.contains('is-open')) hide(); else show(); }

  root.querySelector('[data-act="new"]').addEventListener('click', newTab);
  root.querySelector('[data-act="hide"]').addEventListener('click', hide);
  root.querySelector('[data-act="min"]').addEventListener('click', hide);
  /* maximise: the drawer takes the whole window; remembered */
  function refit() { if (active >= 0 && tabs[active] && !tabs[active].viewer) { try { tabs[active].fit.fit(); } catch (e) {} sendResize(tabs[active]); } }
  function setMax(on) {
    root.classList.toggle('is-max', on);
    var b = root.querySelector('[data-act="max"]'); b.textContent = on ? '❐' : '□'; b.title = on ? 'Restore the drawer size' : 'Maximise: the whole window (double-click the bar too)';
    try { localStorage.setItem('si-term-max', on ? '1' : ''); } catch (e) {}
    setTimeout(refit, 260);
  }
  root.querySelector('[data-act="max"]').addEventListener('click', function () { setMax(!root.classList.contains('is-max')); });
  root.querySelector('.si-term-bar').addEventListener('dblclick', function (ev) { if (ev.target.closest('button,.si-term-tab')) return; setMax(!root.classList.contains('is-max')); });
  try { if (localStorage.getItem('si-term-max')) setMax(true); } catch (e) {}
  window.addEventListener('resize', function () { if (active >= 0 && tabs[active] && !tabs[active].viewer) { try { tabs[active].fit.fit(); } catch (e) {} sendResize(tabs[active]); } });

  /* one width for the side panels, dragged from their left edge, remembered */
  var panelW = 0; try { panelW = +localStorage.getItem('si-panel-w') || 0; } catch (e) {}
  function sizePanel(p) { if (panelW) p.style.flex = '0 0 ' + panelW + 'px'; }
  function addGrip(p) {
    sizePanel(p);
    var g = document.createElement('div'); g.className = 'si-term-grip'; g.title = 'Drag to resize';
    p.appendChild(g);
    g.addEventListener('pointerdown', function (ev) {
      if (ev.button !== 0) return;
      ev.preventDefault(); g.setPointerCapture(ev.pointerId); g.classList.add('is-drag'); root.classList.add('is-resizing');
      var right = p.getBoundingClientRect().right, body = root.querySelector('.si-term-body').getBoundingClientRect();
      function mv(e) {
        panelW = Math.round(Math.max(240, Math.min(body.width * 0.8, right - e.clientX)));
        [rbPanel, dmPanel].forEach(function (x) { if (x) sizePanel(x); });
      }
      function up() {
        g.removeEventListener('pointermove', mv); g.removeEventListener('pointerup', up); g.classList.remove('is-drag'); root.classList.remove('is-resizing');
        try { localStorage.setItem('si-panel-w', String(panelW)); } catch (e) {}
        if (active >= 0 && tabs[active] && !tabs[active].viewer) { try { tabs[active].fit.fit(); } catch (e) {} sendResize(tabs[active]); }
      }
      g.addEventListener('pointermove', mv); g.addEventListener('pointerup', up);
    });
  }

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
    if (t && t.viewer) { t = lastShell && tabs.indexOf(lastShell) >= 0 ? lastShell : null; if (t) select(tabs.indexOf(t)); }
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
      if (ev.key === 'Escape') { ev.preventDefault(); closeForm(); if (active >= 0 && tabs[active] && tabs[active].term) tabs[active].term.focus(); }
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
    addGrip(rbPanel); root.querySelector('.si-term-body').appendChild(rbPanel);
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
    addGrip(dmPanel); root.querySelector('.si-term-body').appendChild(dmPanel);
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
  /* ---------- Viewer tabs: a folder as a tree, one file read-only or in the editor, as many as you like ---------- */
  var viewerSeq = 0, lastShell = null;
  function size(n) { return n < 1024 ? n + ' B' : n < 1048576 ? (n / 1024).toFixed(n < 10240 ? 1 : 0) + ' KB' : (n / 1048576).toFixed(1) + ' MB'; }
  function join(d, n) { return d.replace(/\/+$/, '') + '/' + n; }
  function tb(f, title, path) {
    return '<button type="button" data-f="' + f + '" title="' + title + '" aria-label="' + title + '"><svg viewBox="0 0 24 24" width="15" height="15" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">' + path + '</svg></button>';
  }
  /* one context menu at a time, on the page (the drawer's transform would
     shift a fixed menu inside it); items: [label, fn, class] or null for a rule */
  var menuEl = null;
  function closeMenu() { if (menuEl) { menuEl.remove(); menuEl = null; } }
  function showMenu(x, y, items) {
    closeMenu();
    var m = menuEl = document.createElement('div'); m.className = 'si-fs-menu'; m.setAttribute('role', 'menu');
    items.forEach(function (it) {
      if (!it) { m.appendChild(document.createElement('hr')); return; }
      var b = document.createElement('button'); b.type = 'button'; b.setAttribute('role', 'menuitem'); b.textContent = it[0]; if (it[2]) b.className = it[2];
      b.addEventListener('click', function () { closeMenu(); it[1](); });
      m.appendChild(b);
    });
    document.body.appendChild(m);
    m.style.left = Math.max(4, Math.min(x, innerWidth - m.offsetWidth - 4)) + 'px';
    m.style.top = Math.max(4, Math.min(y, innerHeight - m.offsetHeight - 4)) + 'px';
    m.addEventListener('keydown', function (ev) {
      ev.stopPropagation();
      var bs = Array.prototype.slice.call(m.querySelectorAll('button')), i = bs.indexOf(document.activeElement);
      if (ev.key === 'Escape') { ev.preventDefault(); closeMenu(); }
      else if (ev.key === 'ArrowDown' || ev.key === 'ArrowUp') { ev.preventDefault(); bs[(i + (ev.key === 'ArrowDown' ? 1 : bs.length - 1)) % bs.length].focus(); }
    });
    m.querySelector('button').focus();
  }
  document.addEventListener('pointerdown', function (ev) { if (menuEl && !menuEl.contains(ev.target)) closeMenu(); }, true);
  window.addEventListener('blur', closeMenu);
  var editorLoad = null; // editor.js (CodeMirror), fetched the first time Edit is pressed
  function loadEditor() {
    if (window.SiEditor) return Promise.resolve();
    return editorLoad || (editorLoad = new Promise(function (ok, fail) {
      var s = document.createElement('script'); s.src = '/_si/term/assets/editor.js';
      s.onload = function () { window.SiEditor ? ok() : fail(); }; s.onerror = function () { editorLoad = null; fail(); };
      document.head.appendChild(s);
    }));
  }
  function fsList(dir) { return fetch(API + 'ls?dir=' + encodeURIComponent(dir)).then(function (r) { if (!r.ok) return r.text().then(function (t) { throw new Error(t); }); return r.json(); }); }
  function newViewer() {
    var start = ''; try { start = localStorage.getItem('si-files-dir') || ''; } catch (e) {}
    var n = ++viewerSeq;
    var tabEl = document.createElement('button'); tabEl.className = 'si-term-tab';
    tabEl.innerHTML = '<span class="label">Viewer ' + n + '</span><span class="x" title="Close">×</span>';
    tabsEl.appendChild(tabEl);
    var pane = document.createElement('div'); pane.className = 'si-term-pane si-fs-pane';
    bodyEl.appendChild(pane);
    var t = { viewer: true, pane: pane, tabEl: tabEl };
    tabs.push(t);
    tabEl.addEventListener('click', function (e) { if (e.target.classList.contains('x')) { closeTab(t); return; } select(tabs.indexOf(t)); });

    var root0 = '', cur = '';
    pane.innerHTML = '<div class="si-fs"><div class="si-fs-head"><button type="button" data-f="up" title="Parent folder">↑</button>' +
      '<input spellcheck="false" aria-label="Folder" placeholder="Folder"><button type="button" data-f="shell" title="The folder the last shell you used is in">⌂ Shell</button>' +
      '<button type="button" data-f="reload" title="Reload">↻</button></div>' +
      '<div class="si-fs-split"><div class="si-fs-side"><div class="si-fs-tb" role="toolbar" aria-label="Files">' +
      tb('newfile', 'New file', '<path d="M14 3H7a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V8z"/><path d="M14 3v5h5M12 11v6M9 14h6"/>') +
      tb('newdir', 'New folder', '<path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/><path d="M12 10v6M9 13h6"/>') +
      tb('rename', 'Rename (F2)', '<path d="M4 20h4L19 9l-4-4L4 16z"/><path d="M13.5 6.5l4 4"/>') +
      tb('delete', 'Delete', '<path d="M4 7h16M10 11v6M14 11v6M6 7l1 13h10l1-13M9 7V4h6v3"/>') +
      tb('copysel', 'Copy path', '<rect x="9" y="9" width="11" height="11" rx="2"/><path d="M5 15V5a2 2 0 0 1 2-2h8"/>') +
      '</div><div class="si-fs-tree" tabindex="-1"></div></div><div class="si-fs-view"><div class="si-fs-vh"><span class="nm">No file open</span><span class="in"></span>' +
      '<span class="ok"></span><button type="button" data-f="follow" title="Follow: show the file as it changes on disk, scrolled to the end (for logs)" hidden>Follow</button>' +
      '<button type="button" data-f="edit" title="Edit this file" hidden>Edit</button>' +
      '<button type="button" data-f="save" class="pri" title="Save (⌘S)" hidden>Save</button><button type="button" data-f="revert" title="Back to the file as saved" hidden>Revert</button>' +
      '<button type="button" data-f="done" title="Back to the read-only view" hidden>Done</button>' +
      '<button type="button" data-f="copy" title="Copy the path">Copy path</button></div><div class="si-fs-ask" hidden></div><div class="si-fs-body"><div class="si-fs-msg">Pick a file in the tree. Double-click a folder to make it the top.</div></div></div></div></div>';
    var pathEl = pane.querySelector('input'), tree = pane.querySelector('.si-fs-tree');
    var vName = pane.querySelector('.si-fs-vh .nm'), vInfo = pane.querySelector('.si-fs-vh .in'), body = pane.querySelector('.si-fs-body');
    var label = tabEl.querySelector('.label'), askEl = pane.querySelector('.si-fs-ask'), okEl = pane.querySelector('.si-fs-vh .ok');
    var btn = function (f) { return pane.querySelector('[data-f="' + f + '"]'); };
    var file = null, dirty = false, curName = ''; // file: the cat answer for cur (text, mtime)
    var following = false, noticed = '', polling = false; // Follow, and the changed-on-disk notice while editing
    function setDirty(on) { dirty = on; label.textContent = 'Viewer · ' + curName + (on ? ' ●' : ''); }
    function editing() { return !!t.editor; }
    function buttons() {
      btn('edit').hidden = editing() || !file || typeof file.text !== 'string';
      btn('follow').hidden = editing() || !file || !!file.image || !!file.binary;
      btn('follow').classList.toggle('is-on', following);
      ['save', 'revert', 'done'].forEach(function (f) { btn(f).hidden = !editing(); });
    }
    /* a question inside the panel (no confirm()): [label, fn, primary] choices */
    function ask(msg, choices) {
      askEl.innerHTML = '<span></span>'; askEl.firstChild.textContent = msg;
      choices.forEach(function (c) {
        var b = document.createElement('button'); b.type = 'button'; b.textContent = c[0]; if (c[2]) b.className = c[2] === true ? 'pri' : c[2];
        b.addEventListener('click', function () { askEl.hidden = true; c[1](); });
        askEl.appendChild(b);
      });
      askEl.hidden = false;
      var first = askEl.querySelector('button.pri') || askEl.querySelector('button'); if (first) first.focus();
    }
    /* unsaved edits: Save / Discard / Keep editing; then() runs once they are saved or dropped */
    t.leave = function (then) {
      if (!editing() || !dirty) return true;
      ask(curName + ' has unsaved changes.', [
        ['Save', function () { save(false, then); }, true],
        ['Discard', function () { setDirty(false); then(); }],
        ['Keep editing', function () { t.editor.focus(); }],
      ]);
      return false;
    };
    function stopEditing() { if (t.editor) { try { t.editor.destroy(); } catch (e) {} t.editor = null; } body.classList.remove('is-edit'); askEl.hidden = true; setDirty(false); }
    function startEditing() {
      if (!file || typeof file.text !== 'string') return;
      loadEditor().then(function () {
        if (editing() || !file) return;
        body.innerHTML = ''; body.classList.add('is-edit');
        var holder = document.createElement('div'); holder.className = 'si-fs-ed'; body.appendChild(holder);
        t.editor = window.SiEditor.create(holder, { doc: file.text, file: cur,
          onChange: function () { var d = t.editor && t.editor.getText() !== file.text; if (d !== dirty) setDirty(d); },
          onSave: function () { save(false); } });
        buttons(); t.editor.focus();
      }, function () { ask('The editor did not load.', [['OK', function () {}]]); });
    }
    function save(force, then) {
      if (!t.editor) return;
      var text = t.editor.getText(), p = cur;
      fetch(API + 'save', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ path: p, text: text, mtime: file.mtime, force: !!force }) }).then(function (r) {
        if (r.status === 409) {
          ask(curName + ' changed on disk since you opened it.', [
            ['Reload', function () { reload(); }],
            ['Overwrite', function () { save(true, then); }, true],
            ['Keep editing', function () { t.editor && t.editor.focus(); }],
          ]);
          return;
        }
        if (!r.ok) return r.text().then(function (m) { ask('Not saved: ' + m.trim(), [['OK', function () { t.editor && t.editor.focus(); }]]); });
        return r.json().then(function (j) {
          if (cur !== p) return;
          file.mtime = j.mtime; file.text = text; file.size = j.size; askEl.hidden = true;
          setDirty(t.editor ? t.editor.getText() !== text : false);
          vInfo.textContent = size(j.size) + ' · ' + (text.split('\n').length) + ' lines';
          okEl.textContent = 'Saved'; clearTimeout(okEl._t); okEl._t = setTimeout(function () { okEl.textContent = ''; }, 1500);
          if (then) then();
        });
      }, function (err) { ask('Not saved: ' + err.message, [['OK', function () {}]]); });
    }
    /* the file as it is on disk now, into the editor */
    function reload() {
      fetchFile(cur).then(function (j) {
        if (typeof j.text !== 'string') { stopEditing(); openFile(cur, curName); return; }
        file = j; if (t.editor) t.editor.setText(j.text); setDirty(false); askEl.hidden = true;
      });
    }
    /* the tree: holders[dir] is the element a listed folder's rows are in, opened
       the folders shown open (kept across a refresh), sel the row picked last */
    var holders = {}, opened = {}, expanders = {}, sel = null;
    function tbState() {
      ['rename', 'delete', 'copysel'].forEach(function (f) { btn(f).disabled = !sel; });
    }
    function pick(row, p, isDir) {
      Array.prototype.forEach.call(tree.querySelectorAll('.is-on'), function (x) { x.classList.remove('is-on'); });
      if (row) row.classList.add('is-on');
      sel = row ? { path: p, dir: isDir, row: row } : null; tbState();
    }
    function rows(holder, dir, entries, depth) {
      holders[dir] = { el: holder, depth: depth };
      holder.innerHTML = '';
      entries.forEach(function (e) {
        var row = document.createElement('div'); row.className = 'si-fs-row' + (e.name.charAt(0) === '.' ? ' dot' : '');
        row.style.paddingLeft = (8 + depth * 14) + 'px';
        var p = join(dir, e.name); row.title = p;
        row.innerHTML = '<span class="ic">' + (e.dir ? '▸' : '·') + '</span><span class="nm"></span>' + (e.dir ? '' : '<span class="sz">' + size(e.size) + '</span>');
        row.querySelector('.nm').textContent = e.name + (e.dir ? '/' : '');
        holder.appendChild(row);
        row.addEventListener('contextmenu', function (ev) {
          ev.preventDefault(); ev.stopPropagation(); pick(row, p, e.dir); treeMenu(ev.clientX, ev.clientY, true);
        });
        if (e.dir) {
          var kids = document.createElement('div'); kids.hidden = true; holder.appendChild(kids);
          var expand = expanders[p] = function (open) {
            kids.hidden = !open; row.querySelector('.ic').textContent = open ? '▾' : '▸';
            if (open) opened[p] = true; else delete opened[p];
            if (!open || holders[p] && holders[p].el === kids && kids.childNodes.length) return Promise.resolve();
            return fsList(p).then(function (j) { rows(kids, j.dir, j.entries, depth + 1); }, function (err) { kids.textContent = String(err.message); });
          };
          row.addEventListener('click', function () { if (row.querySelector('input')) return; pick(row, p, true); expand(kids.hidden); });
          row.addEventListener('dblclick', function () { go(p); });
          if (opened[p]) expand(true);
        } else {
          row.addEventListener('click', function () { if (row.querySelector('input')) return; pick(row, p, false); openFile(p, e.name); });
        }
        if (sel && sel.path === p) pick(row, p, e.dir);
      });
    }
    function refresh(dir) {
      var h = holders[dir];
      if (!h || dir === root0) return go(root0);
      return fsList(dir).then(function (j) { rows(h.el, j.dir, j.entries, h.depth); });
    }
    function go(dir) {
      return fsList(dir).then(function (j) {
        if (j.dir !== root0) { opened = {}; sel = null; }
        root0 = j.dir; pathEl.value = j.dir; holders = {}; expanders = {};
        try { localStorage.setItem('si-files-dir', j.dir); } catch (e) {}
        rows(tree, j.dir, j.entries, 0); tbState();
        if (j.more) { var m = document.createElement('div'); m.className = 'si-fs-msg'; m.textContent = 'Showing the first 2000 entries.'; tree.appendChild(m); }
      }, function (err) { tree.innerHTML = '<div class="si-fs-msg"></div>'; tree.firstChild.textContent = 'Cannot open ' + dir + ': ' + err.message; });
    }
    /* file operations: POST a JSON body; an error's text is the message */
    function op(name, body) {
      return fetch(API + name, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) }).then(function (r) {
        if (!r.ok) return r.text().then(function (m) { throw new Error(m.trim()); });
        return r.status === 204 ? {} : r.json();
      });
    }
    function parentOf(p) { return p.replace(/\/[^\/]+\/?$/, '') || '/'; }
    function targetDir() { return !sel ? root0 : sel.dir ? sel.path : parentOf(sel.path); }
    /* a name typed in place: Enter commits (commit returns a promise; a refusal
       keeps the field, marked), Escape or leaving the field cancels */
    function nameInput(row, value, commit, cancel) {
      var inp = document.createElement('input'); inp.spellcheck = false; inp.value = value; inp.setAttribute('aria-label', 'Name');
      var nm = row.querySelector('.nm'); if (nm) nm.hidden = true;
      row.insertBefore(inp, nm ? nm.nextSibling : null);
      var done = false;
      function end() { done = true; inp.remove(); if (nm) nm.hidden = false; }
      inp.addEventListener('keydown', function (ev) {
        if (ev.key === 'Escape') { ev.preventDefault(); end(); cancel(); tree.focus(); }
        else if (ev.key === 'Enter') {
          ev.preventDefault();
          var v = inp.value.trim();
          if (!v || v === value) { end(); cancel(); return; }
          done = true;
          commit(v).then(function () { inp.remove(); if (nm) nm.hidden = false; }, function (err) {
            done = false; inp.classList.add('bad'); inp.title = err.message;
            ask(err.message, [['OK', function () { inp.focus(); }, true]]);
          });
        } else inp.classList.remove('bad');
      });
      inp.addEventListener('blur', function () { setTimeout(function () { if (!done && document.activeElement !== inp && askEl.hidden) { end(); cancel(); } }, 0); });
      inp.focus();
      var dot = value.lastIndexOf('.'); inp.setSelectionRange(0, dot > 0 ? dot : value.length);
    }
    function newItem(folder) {
      var d = targetDir();
      (d === root0 || !expanders[d] ? Promise.resolve() : expanders[d](true)).then(function () {
        var h = holders[d] || holders[root0]; if (!h) return;
        var row = document.createElement('div'); row.className = 'si-fs-row'; row.style.paddingLeft = (8 + h.depth * 14) + 'px';
        row.innerHTML = '<span class="ic">' + (folder ? '▸' : '·') + '</span>';
        h.el.insertBefore(row, h.el.firstChild);
        nameInput(row, '', function (name) {
          return op('create', { dir: d, name: name, folder: folder }).then(function (j) {
            row.remove(); sel = { path: j.path, dir: folder };
            return refresh(d).then(function () { if (!folder) openFile(j.path, name, true); });
          });
        }, function () { row.remove(); });
      });
    }
    function renameSel() {
      if (!sel || !sel.row) return;
      var old = sel.path, isDir = sel.dir, base = old.split('/').pop();
      nameInput(sel.row, base, function (name) {
        return op('rename', { path: old, name: name }).then(function (j) {
          if (opened[old]) { opened[j.path] = true; delete opened[old]; }
          if (cur === old || cur.indexOf(old + '/') === 0) { // the open file moved with it
            cur = j.path + cur.slice(old.length); if (cur === j.path) curName = name;
            vName.textContent = curName; vName.title = cur; tabEl.title = cur; setDirty(dirty);
          }
          sel = { path: j.path, dir: isDir };
          return refresh(parentOf(old));
        });
      }, function () {});
    }
    function deleteSel() {
      if (!sel) return;
      var p = sel.path, isDir = sel.dir;
      fetch(API + 'stat?count=1&path=' + encodeURIComponent(p)).then(function (r) { return r.ok ? r.json() : {}; }).then(function (s) {
        var what = s.link ? 'the link ' + p + ' (what it points to stays)' : s.dir || isDir ?
          'the folder ' + p + (s.items ? ' and the ' + s.items + (s.items === 1 ? ' item' : ' items') + ' in it' : ' (empty)') : p;
        ask('Delete ' + what + '? This cannot be undone.', [
          ['Cancel', function () { tree.focus(); }],
          ['Delete', function () {
            op('delete', { path: p }).then(function () {
              delete opened[p]; if (sel && sel.path === p) sel = null;
              if (cur === p || cur.indexOf(p + '/') === 0) { // the open file went with it
                stopEditing(); cur = ''; curName = ''; file = null; following = false; buttons();
                vName.textContent = 'No file open'; vInfo.textContent = ''; label.textContent = 'Viewer ' + n;
                body.innerHTML = '<div class="si-fs-msg"></div>'; body.firstChild.textContent = 'Deleted ' + p + '.';
              }
              return refresh(parentOf(p));
            }).catch(function (err) { ask('Not deleted: ' + err.message, [['OK', function () {}]]); });
          }, 'danger'],
        ]);
      });
    }
    function copySel() { var p = sel ? sel.path : cur; if (p && navigator.clipboard) navigator.clipboard.writeText(p).catch(function () {}); }
    function revealSel() {
      var p = sel ? sel.path : cur; if (!p) return;
      ask('', [['Copy path', copySel], ['OK', function () {}, true]]);
      var sp = askEl.firstChild; sp.innerHTML = '<span class="p"></span>'; sp.firstChild.textContent = p;
    }
    function treeMenu(x, y, onRow) {
      var items = [['New file…', function () { newItem(false); }], ['New folder…', function () { newItem(true); }]];
      if (onRow) items.push(null, ['Rename…', renameSel], ['Copy path', copySel], ['Reveal path', revealSel], null, ['Delete…', deleteSel, 'danger']);
      showMenu(x, y, items);
    }
    tree.addEventListener('contextmenu', function (ev) { ev.preventDefault(); pick(null); treeMenu(ev.clientX, ev.clientY, false); });
    tree.addEventListener('keydown', function (ev) {
      if (ev.target.tagName === 'INPUT') return;
      if (ev.key === 'F2') { ev.preventDefault(); renameSel(); }
      else if ((ev.key === 'Delete' || (ev.key === 'Backspace' && ev.metaKey)) && sel) { ev.preventDefault(); deleteSel(); }
    });
    btn('newfile').addEventListener('click', function () { newItem(false); });
    btn('newdir').addEventListener('click', function () { newItem(true); });
    btn('rename').addEventListener('click', renameSel);
    btn('delete').addEventListener('click', deleteSel);
    btn('copysel').addEventListener('click', copySel);
    function fetchFile(p, tail) { return fetch(API + 'cat?edit=1' + (tail ? '&tail=1' : '') + '&path=' + encodeURIComponent(p)).then(function (r) { if (!r.ok) return r.text().then(function (t) { throw new Error(t); }); return r.json(); }); }
    function atEnd() { return body.scrollHeight - body.scrollTop - body.clientHeight < 24; }
    /* a text file into the read-only view; scroll: 'end', 'keep' or the top */
    function showText(j, scroll) {
      vInfo.textContent = size(j.size) + (j.lines ? ' · ' + j.lines + ' lines' : '') + (j.truncated ? (j.tail ? ' · last 1 MB' : ' · first 1 MB') : '');
      if (j.image) { body.innerHTML = '<img class="si-fs-img" alt="">'; body.firstChild.src = API + 'raw?path=' + encodeURIComponent(j.path) + '&m=' + j.mtime; return; }
      if (j.binary) { body.innerHTML = '<div class="si-fs-msg">A binary file: nothing to show as text.</div>'; return; }
      var top = body.scrollTop, pre = document.createElement('pre'); pre.className = 'si-fs-code';
      pre.innerHTML = j.html.split('\n').map(function (l) { return '<span class="ln">' + (l || ' ') + '</span>'; }).join('');
      body.innerHTML = ''; body.appendChild(pre);
      body.scrollTop = scroll === 'end' ? body.scrollHeight : scroll === 'keep' ? top : 0;
    }
    function openFile(p, name, edit) {
      if (!t.leave(function () { openFile(p, name, edit); })) return;
      stopEditing();
      cur = p; curName = name; file = null; buttons(); vName.textContent = name; vName.title = p; vInfo.textContent = ''; setDirty(false); tabEl.title = p;
      body.innerHTML = '<div class="si-fs-msg">Loading…</div>';
      fetchFile(p, following).then(function (j) {
        if (cur !== p) return;
        file = j; buttons(); showText(j, following ? 'end' : 0);
        if (edit) startEditing();
      }, function (err) { body.innerHTML = '<div class="si-fs-msg"></div>'; body.firstChild.textContent = String(err.message); });
    }
    /* Follow: once a second (while this tab shows), a cheap stat; when the file
       changed, the read-only view is read again and stays at the end if it was
       there. The editor is never refreshed under you: a notice instead. */
    function poll() {
      if (polling || !cur || !file || document.hidden || !root.classList.contains('is-open') || !pane.classList.contains('is-active')) return;
      if (!editing() && !following) return;
      polling = true;
      var p = cur, done = function () { polling = false; };
      fetch(API + 'stat?path=' + encodeURIComponent(p)).then(function (r) { return r.ok ? r.json() : null; }).then(function (s) {
        if (!s || cur !== p || !file || (s.mtime === file.mtime && s.size === file.size)) return;
        if (editing()) {
          if (noticed !== s.mtime) {
            noticed = s.mtime;
            ask(curName + ' changed on disk.', [['Reload', function () { reload(); }], ['Keep editing', function () { t.editor && t.editor.focus(); }, true]]);
          }
          return;
        }
        return fetchFile(p, true).then(function (j) {
          if (cur !== p || editing() || !following) return;
          var end = atEnd(); file = j; buttons(); showText(j, end ? 'end' : 'keep');
        });
      }).then(done, done);
    }
    var timer = setInterval(poll, 1000);
    t.stop = function () { clearInterval(timer); };
    btn('follow').addEventListener('click', function () {
      following = !following; buttons();
      if (!following || !file) return;
      if (file.truncated && !file.tail) fetchFile(cur, true).then(function (j) { if (following && !editing()) { file = j; buttons(); showText(j, 'end'); } });
      else body.scrollTop = body.scrollHeight;
    });
    pathEl.addEventListener('keydown', function (ev) { if (ev.key === 'Enter') go(pathEl.value); });
    pane.addEventListener('keydown', function (ev) {
      ev.stopPropagation(); // keys typed here never drive the deck
      if ((ev.metaKey || ev.ctrlKey) && (ev.key === 's' || ev.key === 'S') && editing()) { ev.preventDefault(); save(false); }
    });
    btn('edit').addEventListener('click', startEditing);
    btn('save').addEventListener('click', function () { save(false); });
    btn('revert').addEventListener('click', function () { if (t.editor && file) { t.editor.setText(file.text); setDirty(false); askEl.hidden = true; t.editor.focus(); } });
    btn('done').addEventListener('click', function () { if (t.leave(function () { openFile(cur, curName); })) openFile(cur, curName); });
    pane.querySelector('[data-f="up"]').addEventListener('click', function () { go(root0.replace(/\/[^\/]+\/?$/, '') || '/'); });
    pane.querySelector('[data-f="reload"]').addEventListener('click', function () { go(root0); if (cur && !editing()) openFile(cur, curName); });
    pane.querySelector('[data-f="shell"]').addEventListener('click', function () {
      var sh = lastShell && tabs.indexOf(lastShell) >= 0 ? lastShell : null; if (!sh || !sh.id) return;
      fetch(API + 'cwd?id=' + sh.id).then(function (r) { return r.json(); }).then(function (j) { go(j.dir); });
    });
    pane.querySelector('[data-f="copy"]').addEventListener('click', function () { if (cur && navigator.clipboard) navigator.clipboard.writeText(cur); });
    go(start);
    select(tabs.length - 1);
  }
  buildRunbook();
  buildDemos();
  root.querySelector('[data-act="files"]').addEventListener('click', function () { show(true); newViewer(); });
  var rbBtn = root.querySelector('[data-act="runbook"]');
  if (rbBtn) rbBtn.addEventListener('click', toggleRunbook);
  var dmBtn = root.querySelector('[data-act="demos"]');
  if (dmBtn) dmBtn.addEventListener('click', toggleDemos);

  window.siTerm = { toggle: toggle, show: show, hide: hide, runbook: toggleRunbook, demos: toggleDemos, viewer: function () { show(true); newViewer(); } };
})();
