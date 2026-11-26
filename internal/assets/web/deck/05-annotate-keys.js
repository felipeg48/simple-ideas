  /* ---------- annotate (a): a transient pen layer over the slides; cleared on slide change ---------- */
  var draw = document.querySelector('.si-draw'), drawBar = document.querySelector('.si-draw-bar');
  var penColors = ['var(--si-accent)', 'var(--si-crit)', 'var(--si-ok)', 'var(--si-fg)'];
  var drawMode = false, erasing = false, drawing = false, penColor = penColors[0], curPath = null;
  var penSize = +load('si-pen-size') || 5;
  /* the board (b in draw mode): an opaque page over the slides with its own ink, kept across
     slides until cleared, put away with the pen. Its paper: white, black (chalk) or blue
     (blueprint), B cycles; its pattern: plain, grid, dots or lines, g cycles; both remembered.
     The pattern is its own layer under the ink, so undo and clear never touch it. */
  var board = document.querySelector('.si-board'), boardInk = board && board.querySelector('.si-board-ink'),
    boardPaper = board && board.querySelector('.si-board-paper'), boardKind = '';
  var papers = ['white', 'black', 'blue'], patterns = ['plain', 'grid', 'dots', 'lines'];
  var paper = papers.indexOf(load('si-board-paper')) >= 0 ? load('si-board-paper') : 'white';
  var pattern = patterns.indexOf(load('si-board-pattern')) >= 0 ? load('si-board-pattern') : 'plain';
  function ink() { return boardKind ? boardInk : draw; }
  function setBoard(k) {
    if (!board) return;
    boardKind = k; board.hidden = !k;
    if (k) { paper = k; save('si-board-paper', k); board.setAttribute('data-board', k); document.body.setAttribute('data-si-board', k); drawPaper(); }
    else document.body.removeAttribute('data-si-board');
    syncPenUI();
  }
  function toggleBoard() { setBoard(boardKind ? '' : paper); }
  function cyclePaper() { setBoard(papers[(papers.indexOf(boardKind || paper) + 1) % papers.length]); }
  function setPattern(p) { pattern = p; save('si-board-pattern', p); drawPaper(); syncPenUI(); }
  function cyclePattern() { setPattern(patterns[(patterns.indexOf(pattern) + 1) % patterns.length]); }
  /* the pattern, in screen pixels: a cell is 1/22 of the shorter side; the colour is the paper's --_grid */
  function drawPaper() {
    if (!boardPaper || !boardKind) return;
    var c = Math.max(12, Math.round(Math.min(innerWidth, innerHeight) / 22)), col = getComputedStyle(board).getPropertyValue('--_grid').trim() || '#c8d6e8';
    var w = c, h = c, body = '';
    if (pattern === 'grid') {
      w = h = 5 * c;
      for (var i = 1; i < 5; i++) body += '<path d="M' + i * c + ' 0V' + h + 'M0 ' + i * c + 'H' + w + '" stroke="' + col + '" stroke-opacity=".55" stroke-width="1"/>';
      body += '<path d="M0 0V' + h + 'M0 0H' + w + '" stroke="' + col + '" stroke-width="1.5"/>';
    } else if (pattern === 'dots') body = '<circle cx="' + c / 2 + '" cy="' + c / 2 + '" r="' + Math.max(1.3, c / 16).toFixed(1) + '" fill="' + col + '"/>';
    else if (pattern === 'lines') { h = Math.round(c * 1.2); body = '<path d="M0 ' + (h - .5) + 'H' + w + '" stroke="' + col + '" stroke-width="1.2"/>'; }
    boardPaper.innerHTML = pattern === 'plain' ? '' : '<defs><pattern id="si-board-pat" patternUnits="userSpaceOnUse" width="' + w + '" height="' + h + '">' + body +
      '</pattern></defs><rect width="100%" height="100%" fill="url(#si-board-pat)"/>';
  }
  if (board) window.addEventListener('resize', drawPaper);
  /* the board as a picture: paper colour, pattern and ink, with the colours the theme gave them
     written in, drawn at up to 2x on a canvas */
  function boardShot() {
    return new Promise(function (ok, fail) {
      var W = board.clientWidth, H = board.clientHeight, s = Math.min(2, window.devicePixelRatio || 1), cs = getComputedStyle(board);
      var inkCopy = boardInk.cloneNode(true), src = boardInk.querySelectorAll('path'), dst = inkCopy.querySelectorAll('path');
      each(dst, function (d, i) {
        var c = getComputedStyle(src[i]);
        d.removeAttribute('style');
        d.setAttribute('stroke', c.stroke); d.setAttribute('fill', 'none'); d.setAttribute('stroke-linecap', 'round'); d.setAttribute('stroke-linejoin', 'round');
        if (c.opacity !== '1') d.setAttribute('opacity', c.opacity);
        if (/url\(/.test(c.filter)) d.setAttribute('filter', 'url(#si-chalk)');
      });
      var svg = '<svg xmlns="http://www.w3.org/2000/svg" width="' + W + '" height="' + H + '" viewBox="0 0 ' + W + ' ' + H + '">' +
        '<rect width="100%" height="100%" fill="' + cs.backgroundColor + '"/>' + boardPaper.innerHTML + inkCopy.innerHTML + '</svg>';
      var img = new Image(), url = URL.createObjectURL(new Blob([svg], { type: 'image/svg+xml' }));
      img.onload = function () {
        var cv = document.createElement('canvas'); cv.width = Math.round(W * s); cv.height = Math.round(H * s);
        var g = cv.getContext('2d'); g.scale(s, s); g.drawImage(img, 0, 0, W, H); URL.revokeObjectURL(url);
        cv.toBlob(function (b) { if (b) ok(b); else fail(new Error('no picture')); }, 'image/png');
      };
      img.onerror = function () { URL.revokeObjectURL(url); fail(new Error('no picture')); };
      img.src = url;
    });
  }
  function boardSave() {
    return boardShot().then(function (b) {
      var a = document.createElement('a'); a.download = (data.name || 'board') + '-board.png'; a.href = URL.createObjectURL(b);
      document.body.appendChild(a); a.click(); a.remove(); setTimeout(function () { URL.revokeObjectURL(a.href); }, 4000);
    });
  }
  function flash(btn, text) { if (!btn) return; var t = btn.textContent; btn.textContent = text; setTimeout(function () { btn.textContent = t; }, 1600); }
  /* Copy: the picture on the clipboard (https, localhost and file pages); elsewhere it is saved instead */
  function boardCopy(btn) {
    var saved = function () { boardSave().then(function () { flash(btn, 'Saved ✓'); }); };
    if (!navigator.clipboard || !window.ClipboardItem || !window.isSecureContext) { saved(); return; }
    navigator.clipboard.write([new ClipboardItem({ 'image/png': boardShot() })]).then(function () { flash(btn, 'Copied ✓'); }, saved);
  }
  function toggleDraw(on) {
    if (!draw) return;
    if (on === undefined) on = !drawMode;
    drawMode = on; if (!on) { drawing = false; erasing = false; setBoard(''); } else apTouch(); /* the pen takes over from auto-play */
    document.body.classList.toggle('si-drawing', on);
    draw.setAttribute('aria-hidden', on ? 'false' : 'true');
    syncPenUI();
  }
  /* clearStrokes() empties the slide's ink (a new slide); the x key and Clear empty the layer in use */
  function clearStrokes(share, layer) {
    layer = layer || draw;
    if (layer) each(layer.querySelectorAll(':scope > path'), function (p) { layer.removeChild(p); });
    if (share) post({ type: 'clear', i: cur });
  }
  function undoStroke() { var l = ink(); if (l && l.lastChild && l.lastChild.nodeName === 'path') dropStroke(l.lastChild); }
  function dropStroke(el) { el.parentNode.removeChild(el); if (el.id) post({ type: 'erase', i: cur, id: el.id }); }
  /* shared ink: points travel in 0..1 slide coordinates (and the width as a
     fraction of the slide width), batched every 40 ms while a stroke grows */
  var inkSeq = 0, inkBuf = [], inkTimer = 0, inkW = 0;
  function slideBox() { return slides[cur].getBoundingClientRect(); }
  function inkPush(x, y) {
    if (!chans.length || !curPath) return;
    var r = slideBox(); inkBuf.push([+((x - r.left) / r.width).toFixed(4), +((y - r.top) / r.height).toFixed(4)]);
    if (!inkTimer) inkTimer = setTimeout(inkFlush, 40);
  }
  function inkFlush() {
    inkTimer = 0;
    if (!inkBuf.length || !curPath) { inkBuf = []; return; }
    post({ type: 'ink', i: cur, id: curPath.id, c: curPath.style.stroke, w: +(inkW / slideBox().width).toFixed(5), p: inkBuf });
    inkBuf = [];
  }
  function inkIn(m) {
    if (!draw || m.i !== cur) return;
    if (m.type === 'clear') { clearStrokes(false, ink()); return; }
    var el = m.id && document.getElementById(m.id);
    if (m.type === 'erase') { if (el && (el.parentNode === draw || el.parentNode === boardInk)) el.parentNode.removeChild(el); return; }
    var r = slideBox(), d = '';
    (m.p || []).forEach(function (pt) { var x = r.left + pt[0] * r.width, y = r.top + pt[1] * r.height; d += (d || el ? ' L ' : 'M ') + x.toFixed(1) + ' ' + y.toFixed(1); });
    if (!el) {
      el = document.createElementNS('http://www.w3.org/2000/svg', 'path');
      el.id = m.id; el.style.stroke = m.c; el.setAttribute('stroke-width', Math.max(1, m.w * r.width).toFixed(1));
      el.setAttribute('d', d); ink().appendChild(el);
    } else el.setAttribute('d', el.getAttribute('d') + d);
  }
  function cycleColor() { penColor = penColors[(penColors.indexOf(penColor) + 1) % penColors.length]; erasing = false; syncPenUI(); }
  function toggleEraser() { erasing = !erasing; syncPenUI(); }
  function syncPenUI() {
    if (!drawBar) return;
    each(drawBar.querySelectorAll('[data-si-pen]'), function (b) { b.classList.toggle('is-on', !erasing && b.getAttribute('data-si-pen') === penColor); });
    each(drawBar.querySelectorAll('[data-si-size]'), function (b) { b.classList.toggle('is-on', +b.getAttribute('data-si-size') === penSize); });
    var er = drawBar.querySelector('[data-si-draw="erase"]'); if (er) er.classList.toggle('is-on', erasing);
    var bd = drawBar.querySelector('[data-si-draw="board"]'); if (bd) bd.classList.toggle('is-on', !!boardKind);
    each(drawBar.querySelectorAll('[data-si-paper]'), function (b) { b.classList.toggle('is-on', b.getAttribute('data-si-paper') === boardKind); });
    each(drawBar.querySelectorAll('[data-si-pattern]'), function (b) { b.setAttribute('aria-pressed', b.getAttribute('data-si-pattern') === pattern ? 'true' : 'false'); });
  }
  function dStart(e) {
    if (!drawMode) return;
    e.preventDefault(); drawing = true;
    if (erasing) { eraseAt(e); return; }
    curPath = document.createElementNS('http://www.w3.org/2000/svg', 'path');
    curPath.id = 'si-ink-' + Math.random().toString(36).slice(2, 7) + (++inkSeq);
    curPath.setAttribute('d', 'M ' + e.clientX + ' ' + e.clientY);
    // inline style (not a presentation attribute) so the CSS var() resolves
    curPath.style.stroke = penColor;
    var w = (e.pointerType === 'pen' && e.pressure) ? penSize * (0.5 + e.pressure) : penSize;
    curPath.setAttribute('stroke-width', w.toFixed(1));
    ink().appendChild(curPath);
    inkW = w; inkPush(e.clientX, e.clientY);
  }
  function dMove(e) {
    if (!drawMode || !drawing) return;
    if (erasing) { eraseAt(e); return; }
    if (curPath) { curPath.setAttribute('d', curPath.getAttribute('d') + ' L ' + e.clientX + ' ' + e.clientY); inkPush(e.clientX, e.clientY); }
  }
  function dEnd() { if (curPath) { clearTimeout(inkTimer); inkFlush(); } drawing = false; curPath = null; }
  function eraseAt(e) { var el = document.elementFromPoint(e.clientX, e.clientY); if (el && el.nodeName === 'path' && el.parentNode === ink()) dropStroke(el); }
  if (draw) {
    draw.addEventListener('pointerdown', dStart);
    if (board) board.addEventListener('pointerdown', dStart);
    window.addEventListener('pointermove', dMove);
    window.addEventListener('pointerup', dEnd);
    window.addEventListener('pointercancel', dEnd);
    /* in draw mode, holding Option (Alt) draws where the cursor goes, no
       click: a trackpad draws with one finger; letting go of it ends the
       stroke. si.conf draw.hold picks the key: alt, shift, ctrl, meta, off */
    var hold = { alt: ['altKey', 'Alt'], shift: ['shiftKey', 'Shift'], ctrl: ['ctrlKey', 'Control'], meta: ['metaKey', 'Meta'] }[cfg.drawHold || 'alt'];
    var held = function (e) { return !!hold && !!e[hold[0]]; };
    var altStroke = false, lastX = -1, lastY = -1;
    window.addEventListener('pointermove', function (e) {
      if (e.pointerType === 'touch') return;
      lastX = e.clientX; lastY = e.clientY;
      if (!drawMode) return;
      if (held(e) && !drawing) { altStroke = true; dStart(e); }
      else if (!held(e) && altStroke) { altStroke = false; dEnd(); }
    });
    // the stroke starts where the cursor is the moment Option goes down
    window.addEventListener('keydown', function (e) {
      if (!hold || e.key !== hold[1] || !drawMode || drawing || lastX < 0) return;
      altStroke = true;
      dStart({ clientX: lastX, clientY: lastY, pointerType: 'mouse', preventDefault: function () {} });
    });
    window.addEventListener('keyup', function (e) { if (hold && e.key === hold[1] && altStroke) { altStroke = false; dEnd(); } });
    window.addEventListener('blur', function () { if (altStroke) { altStroke = false; dEnd(); } });
  }
  if (drawBar) {
    each(drawBar.querySelectorAll('[data-si-pen]'), function (b) { b.addEventListener('click', function () { penColor = b.getAttribute('data-si-pen'); erasing = false; syncPenUI(); b.blur(); }); });
    each(drawBar.querySelectorAll('[data-si-size]'), function (b) { b.addEventListener('click', function () { penSize = +b.getAttribute('data-si-size'); save('si-pen-size', penSize); syncPenUI(); b.blur(); }); });
    each(drawBar.querySelectorAll('[data-si-paper]'), function (b) { b.addEventListener('click', function () { setBoard(b.getAttribute('data-si-paper')); b.blur(); }); });
    each(drawBar.querySelectorAll('[data-si-pattern]'), function (b) { b.addEventListener('click', function () { setPattern(b.getAttribute('data-si-pattern')); b.blur(); }); });
    each(drawBar.querySelectorAll('[data-si-draw]'), function (b) {
      b.addEventListener('click', function () {
        var a = b.getAttribute('data-si-draw');
        if (a === 'erase') toggleEraser(); else if (a === 'undo') undoStroke(); else if (a === 'clear') clearStrokes(true, ink()); else if (a === 'board') toggleBoard(); else if (a === 'copy') boardCopy(b); else if (a === 'png') boardSave(); else if (a === 'done') toggleDraw(false);
        b.blur();
      });
    });
    syncPenUI();
  }

  /* Play terminal drawer: lazy-load the runtime (xterm.js + term.js) on first
   * toggle, so a page that never opens the terminal pays nothing for it, and a
   * built deck (termEnabled false) never references it at all. */
  var termLoaded = false, termThen = [];
  function loadTerm(then) {
    if (then) termThen.push(then);
    if (termLoaded) return; // load in flight: then runs when it is in
    termLoaded = true;
    var css = document.createElement('link');
    css.rel = 'stylesheet'; css.href = '/_si/term/assets/xterm.css';
    document.head.appendChild(css);
    function add(src, next) { var s = document.createElement('script'); s.src = src; s.onload = next; document.head.appendChild(s); }
    add('/_si/term/assets/xterm.js', function () {
      add('/_si/term/assets/addon-fit.js', function () {
        add('/_si/term/assets/term.js', function () { if (window.siTerm) termThen.splice(0).forEach(function (f) { f(); }); });
      });
    });
  }
  function toggleTerminal() {
    if (!termEnabled) return;
    if (window.siTerm) { window.siTerm.toggle(); return; }
    loadTerm(function () { window.siTerm.toggle(); });
  }

  /* Floating browser windows (w, si play only): lazy-load web.js; reopen the
   * windows a live reload closed (sessionStorage si-web) without the picker. */
  var webLoaded = false, webThen = [];
  function loadWeb(then) {
    if (window.siWeb) { if (then) then(); return; }
    if (then) webThen.push(then);
    if (webLoaded) return;
    webLoaded = true;
    var s = document.createElement('script'); s.src = '/_si/term/assets/web.js';
    s.onload = function () { webThen.splice(0).forEach(function (f) { f(); }); };
    document.head.appendChild(s);
  }
  function toggleBrowser() { if (termEnabled && !role) loadWeb(function () { window.siWeb.picker(); }); }
  try { if (termEnabled && !role && JSON.parse(sessionStorage.getItem('si-web') || '[]').length) loadWeb(); } catch (e) {}
  /* the drawer's shells and Viewers outlive a live reload: term.js brings them back (sessionStorage si-term-state) */
  try { if (termEnabled && !role && (JSON.parse(sessionStorage.getItem('si-term-state') || '{}').tabs || []).length) loadTerm(); } catch (e) {}

  function runCmd(a) {
    if (role === 'audience' && a !== 'fullscreen' && a !== 'mode' && a !== 'help') return; /* follow only */
    if (a === 'overview') { if (feat.overview) overview(); }
    else if (a === 'presenter') { if (feat.presenter) openPresenter(); }
    else if (a === 'draw') { if (feat.draw) toggleDraw(); }
    else if (a === 'timer') { if (feat.timer) hudShow(); }
    else if (a === 'mode') toggleMode();
    else if (a === 'fullscreen') fullscreen();
    else if (a === 'find') { if (feat.find) findShow(); }
    else if (a === 'help') toggleHelp();
    else if (a === 'terminal') toggleTerminal();
    else if (a === 'browser') toggleBrowser();
    else if (a === 'remote') toggleRemoteQR();
    else if (a === 'autoplay') { if (feat.autoplay) apToggle(); }
  }
  /* the remote links as QR codes, for a phone or tablet in the room (si play --remote) */
  var rqBox = null;
  function toggleRemoteQR(on) {
    if (!remoteQR) return;
    if (!rqBox) {
      rqBox = document.createElement('div');
      rqBox.id = 'si-remote-qr';
      rqBox.setAttribute('role', 'dialog');
      rqBox.setAttribute('aria-label', 'Remote links');
      rqBox.setAttribute('style', 'position:fixed;inset:0;z-index:2147483646;display:flex;flex-wrap:wrap;align-content:center;justify-content:center;gap:clamp(1rem,4vmin,3rem);padding:4vmin;overflow:auto;background:color-mix(in srgb,var(--si-bg) 97%,transparent);color:var(--si-fg);font-family:var(--si-font-body)');
      var esc = function (t) { return String(t).replace(/[&<>"]/g, function (c) { return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]; }); };
      var html = '';
      remoteQR.links.forEach(function (l) {
        html += '<figure style="margin:0;display:flex;flex-direction:column;align-items:center;gap:.6em;width:min(30vmin,80vw);min-width:12rem">' +
          '<div style="width:100%;background:#fff;border-radius:.8em;padding:.4em;box-sizing:border-box">' + l.svg + '</div>' +
          '<figcaption style="text-align:center;font-weight:600">' + esc(l.name) + '<br><span style="font:400 .72em/1.3 var(--si-font-mono);color:var(--si-muted);word-break:break-all">' + esc(l.url) + '</span></figcaption></figure>';
      });
      if (remoteQR.also && remoteQR.also.length) html += '<p style="flex-basis:100%;text-align:center;color:var(--si-muted);margin:0">Not on this network? The same links work with ' + esc(remoteQR.also.join(', ')) + '.</p>';
      html += '<p style="flex-basis:100%;text-align:center;color:var(--si-muted);margin:0;font-size:.85em">q or Esc closes</p>';
      rqBox.innerHTML = html;
      rqBox.addEventListener('click', function () { toggleRemoteQR(false); });
      document.body.appendChild(rqBox);
      rqBox.hidden = true;
    }
    rqBox.hidden = on === undefined ? !rqBox.hidden : !on;
  }
  if (remoteQR && help) {
    var _ht = help.querySelector('table');
    if (_ht) { var _tr = document.createElement('tr'); _tr.innerHTML = '<td>' + cmd.remote + '</td><td>remote: the phone and tablet links as QR codes</td>'; _ht.appendChild(_tr); }
  }
  var actions = { next: forward, prev: backward, overview: function () { runCmd('overview'); }, draw: function () { runCmd('draw'); }, presenter: function () { runCmd('presenter'); }, find: function () { runCmd('find'); }, mode: toggleMode, help: function () { toggleHelp(); }, fullscreen: fullscreen, autoplay: function () { runCmd('autoplay'); } };
  /* a disabled feature hides its toolbar button */
  /* si play on loopback: Terminal and Window buttons beside Auto-play (a
     built deck never has them; termEnabled is false there) */
  if (termEnabled && !role) {
    actions.terminal = function () { runCmd('terminal'); };
    actions.browser = function () { runCmd('browser'); };
    var _ba = document.querySelector('[data-si-action="autoplay"]');
    if (_ba) [['browser', 'Window', 'Browser window over the slides (' + cmd.browser + ')'], ['terminal', 'Terminal', 'Terminal (' + cmd.terminal + ')']].forEach(function (x) {
      var b = document.createElement('button'); b.type = 'button'; b.setAttribute('data-si-action', x[0]); b.title = x[2]; b.textContent = x[1];
      _ba.parentNode.insertBefore(b, _ba.nextSibling);
    });
  }
  if (!feat.overview) { var _bo = document.querySelector('[data-si-action="overview"]'); if (_bo) _bo.style.display = 'none'; }
  if (!feat.presenter) { var _bp = document.querySelector('[data-si-action="presenter"]'); if (_bp) _bp.style.display = 'none'; }
  if (!feat.draw) { var _bd = document.querySelector('[data-si-action="draw"]'); if (_bd) _bd.style.display = 'none'; }
  document.addEventListener('keydown', function (e) {
    if (e.metaKey || e.ctrlKey || e.altKey) return;
    var tag = (document.activeElement && document.activeElement.tagName) || '';
    if (/INPUT|SELECT|TEXTAREA/.test(tag)) return;
    if (rqBox && !rqBox.hidden && (e.key === 'Escape' || e.key === cmd.remote)) { e.preventDefault(); toggleRemoteQR(false); return; }
    if (deck.classList.contains('si-overview')) {
      if (e.key === 'Escape' || e.key === cmd.overview) { e.preventDefault(); overview(false); return; }
      if (e.key === 'Enter') { e.preventDefault(); var t = ovSel; overview(false); go(t, 0); return; }
      if (e.key === 'ArrowLeft') { e.preventDefault(); moveSel(-1, 0); return; }
      if (e.key === 'ArrowRight') { e.preventDefault(); moveSel(1, 0); return; }
      if (e.key === 'ArrowUp') { e.preventDefault(); moveSel(0, -1); return; }
      if (e.key === 'ArrowDown') { e.preventDefault(); moveSel(0, 1); return; }
      if (e.key === 'Home') { e.preventDefault(); setSel(0); return; }
      if (e.key === 'End') { e.preventDefault(); setSel(slides.length - 1); return; }
      return; // swallow other keys while navigating the overview
    }
    if (hud && hud.classList.contains('is-open')) {
      if (e.key === 'Escape' || e.key === cmd.timer) { e.preventDefault(); hudShow(false); return; }
      if (e.key === ' ') { e.preventDefault(); hudPause(); return; } // pause reveals the controls; resume hides them
      var pi = '123456789'.indexOf(e.key); if (pi >= 0 && pi < hudMins.length) { e.preventDefault(); hudStart(hudMins[pi]); return; }
      if (/^(Arrow|Page|Home|End|Enter)/.test(e.key)) e.preventDefault();
      return;
    }
    if (drawMode) {
      if (e.key === 'Escape' || e.key === cmd.draw) { e.preventDefault(); toggleDraw(false); return; }
      if (e.key === 'c') { e.preventDefault(); cycleColor(); return; }
      if (e.key === 'e') { e.preventDefault(); toggleEraser(); return; }
      if (e.key === 'z') { e.preventDefault(); undoStroke(); return; }
      if (e.key === 'x') { e.preventDefault(); clearStrokes(true, ink()); return; }
      if (e.key === 'b') { e.preventDefault(); toggleBoard(); return; }
      if (e.key === 'B') { e.preventDefault(); cyclePaper(); return; }
      if (e.key === 'g' && boardKind) { e.preventDefault(); cyclePattern(); return; }
    }
    switch (e.key) {
      case 'ArrowRight': case 'ArrowDown': case 'PageDown': case 'Enter': case 'j': case 'n':
        e.preventDefault(); forward(); break;
      case ' ':
        e.preventDefault(); if (e.shiftKey) backward(); else forward(); break;
      case 'ArrowLeft': case 'ArrowUp': case 'PageUp': case 'Backspace': case 'k':
        e.preventDefault(); backward(); break;
      case 'Home': e.preventDefault(); go(0, 0); break;
      case 'End': e.preventDefault(); go(n - 1, 'end'); break;
      case 'Escape': toggleHelp(false); if (ap && ap.state !== 'off') apStop(); if (deck.classList.contains('si-overview')) overview(false); break;
      case 'S': runCmd('presenter'); break;
      case 'h': runCmd('help'); break;
      default: { var act = byKey[e.key]; if (act) { e.preventDefault(); runCmd(act); } }
    }
  });
  each(document.querySelectorAll('[data-si-action]'), function (b) {
    b.addEventListener('click', function () { var f = actions[b.getAttribute('data-si-action')]; if (f) f(); b.blur(); });
  });
  /* rebuild the HUD preset buttons + hint from the configured times/unit */
  (function () {
    if (!hud) return;
    hud.addEventListener('click', function (e) { if (!e.target.closest('button') && hudTimer) hudPause(); }); // click the clock to pause (reveal) / resume (hide)
    var presets = hud.querySelector('.si-hud-presets'), hint = hud.querySelector('.si-hud-hint');
    if (presets) {
      presets.textContent = '';
      hudMins.forEach(function (m) {
        var btn = document.createElement('button');
        btn.type = 'button'; btn.setAttribute('data-si-min', m); btn.textContent = m;
        btn.addEventListener('click', function () { hudStart(m); btn.blur(); });
        presets.appendChild(btn);
      });
    }
    if (hint) {
      var keyspan = hudMins.length > 1 ? '1-' + Math.min(9, hudMins.length) : '1';
      hint.textContent = keyspan + ' pick ' + hudUnit + ' · space pause · ' + (cmd.timer || 't') + ' / Esc close';
    }
    /* toggle row: pick the animation (when several) and the face (digital/analog) */
    var toggle = hud.querySelector('.si-hud-toggle');
    if (toggle) {
      toggle.textContent = '';
      function seg(labels, values, getCur, onPick) {
        if (values.length < 2) return;
        var grp = document.createElement('div');
        grp.className = 'si-hud-seg';
        values.forEach(function (val, i) {
          var b = document.createElement('button');
          b.type = 'button'; b.textContent = labels[i];
          b.setAttribute('aria-pressed', getCur() === val ? 'true' : 'false');
          b.addEventListener('click', function () {
            onPick(val);
            each(grp.querySelectorAll('button'), function (x, j) { x.setAttribute('aria-pressed', values[j] === val ? 'true' : 'false'); });
            b.blur();
          });
          grp.appendChild(b);
        });
        toggle.appendChild(grp);
      }
      if (hudAnims.length) { // always offer "None" to turn the animation off
        seg(hudAnims.concat(['None']), hudAnims.concat(['none']), function () { return animCurrent || 'none'; }, function (v) { animCurrent = v; save('si-anim', v); if (hud.classList.contains('is-open')) startAnim(); });
      }
      if (hudDial) seg(['Digital', 'Analog'], ['digital', 'analog'], function () { return faceCurrent; }, function (v) { faceCurrent = v; save('si-face', v); applyFace(); dialUpdate(); });
    }
  })();
  if (help) help.addEventListener('click', function () { toggleHelp(false); });
  var tx0 = 0, ty0 = 0, t0 = 0;
  document.addEventListener('touchstart', function (e) { var t = e.changedTouches[0]; tx0 = t.clientX; ty0 = t.clientY; t0 = Date.now(); }, { passive: true });
  document.addEventListener('touchend', function (e) {
    if (drawMode) return; /* a stroke is not a swipe */
    if (hud && hud.classList.contains('is-open')) return;
    if (deck.classList.contains('si-overview')) return;
    var t = e.changedTouches[0], dx = t.clientX - tx0, dy = t.clientY - ty0;
    if (Date.now() - t0 < 700 && Math.abs(dx) > 50 && Math.abs(dx) > Math.abs(dy) * 1.4) { if (dx < 0) forward(); else backward(); }
  }, { passive: true });
  var uiTimer = 0;
  document.addEventListener('mousemove', function () {
    document.body.classList.add('si-ui');
    clearTimeout(uiTimer);
    uiTimer = setTimeout(function () { document.body.classList.remove('si-ui'); }, 1600);
  });

