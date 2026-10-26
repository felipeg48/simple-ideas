/* simple-ideas deck runtime: navigation, build steps, SVG canvas states,
 * overview, presenter sync, theme mode, mermaid, print. No dependencies. */
(function () {
  'use strict';
  var root = document.documentElement;
  var deck = document.querySelector('.si-deck');
  if (!deck) return;
  var data = JSON.parse(document.getElementById('si-data').textContent);
  /* si.conf (optional): remap command keys and turn features on/off, over the defaults */
  var cfg = window.SI_CONFIG || {};
  var cmd = { overview: 'o', presenter: 'p', draw: 'a', timer: 't', mode: 'd', fullscreen: 'f', help: '?' };
  if (cfg.keys) for (var _k in cfg.keys) if (cmd.hasOwnProperty(_k) && cfg.keys[_k]) cmd[_k] = cfg.keys[_k];
  var feat = { draw: true, timer: true, presenter: true, overview: true };
  if (cfg.features) for (var _f in cfg.features) if (feat.hasOwnProperty(_f)) feat[_f] = !!cfg.features[_f];
  var byKey = {}; for (var _a in cmd) byKey[cmd[_a]] = _a;
  var slides = Array.prototype.slice.call(deck.children).filter(function (e) { return e.classList.contains('si-slide'); });
  var n = slides.length, cur = -1, step = 0;
  var counter = document.querySelector('.si-counter');
  var progress = document.querySelector('.si-progress');
  var help = document.querySelector('.si-help');
  var each = function (list, fn) { Array.prototype.forEach.call(list, fn); };

  function load(k) { try { return localStorage.getItem(k); } catch (e) { return null; } }
  function save(k, v) { try { localStorage.setItem(k, v); } catch (e) {} }

  /* ---------- build steps and canvas state ---------- */
  function stepEls(s) { return s.querySelectorAll('[data-step]'); }
  function maxStep(s) { var m = 0; each(stepEls(s), function (e) { m = Math.max(m, +e.getAttribute('data-step')); }); return m; }
  /* state of a canvas after step k: its base words, then each reached step's ops
     in order (word or +word adds, -word removes, . does nothing) */
  function stateAt(s, svg, k) {
    var words = (svg.getAttribute('data-base') || '').split(/\s+/).filter(Boolean);
    var ops = [];
    each(stepEls(s), function (e) {
      var st = +e.getAttribute('data-step');
      if (st <= k && e.hasAttribute('data-ops')) ops.push([st, e.getAttribute('data-ops')]);
    });
    ops.sort(function (a, b) { return a[0] - b[0]; });
    ops.forEach(function (o) {
      o[1].split(/[\s,]+/).forEach(function (w) {
        if (!w || w === '.') return;
        if (w.charAt(0) === '-') { w = w.slice(1); words = words.filter(function (x) { return x !== w; }); return; }
        if (w.charAt(0) === '+') w = w.slice(1);
        if (words.indexOf(w) < 0) words.push(w);
      });
    });
    return words.join(' ');
  }
  function applySteps(s, k) {
    each(stepEls(s), function (e) {
      var st = +e.getAttribute('data-step');
      e.classList.toggle('is-visible', st <= k);
      e.classList.toggle('is-latest', st === k);
    });
  }
  function applyCanvas(s, k) { each(s.querySelectorAll('svg.si-canvas'), function (svg) { svg.setAttribute('data-state', stateAt(s, svg, k)); }); }
  /* data-count="word:number ..." ticks the element's text to the number of the
     active state word (last match wins); animated on the current slide only. */
  function fmtCount(n) { return String(Math.round(n)); }
  function reduceMotion() { return window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches; }
  function tweenCount(el, from, to) {
    var dur = 650, t0 = 0;
    function fr(t) { if (!t0) t0 = t; var p = Math.min(1, (t - t0) / dur); var e = p < .5 ? 2 * p * p : 1 - Math.pow(-2 * p + 2, 2) / 2; el.textContent = fmtCount(from + (to - from) * e); if (p < 1) requestAnimationFrame(fr); }
    requestAnimationFrame(fr);
  }
  function applyCounts(s) {
    var printing = window.matchMedia && window.matchMedia('print').matches;
    var animate = s.classList.contains('si-current') && !reduceMotion() && !printing;
    each(s.querySelectorAll('svg.si-canvas [data-count]'), function (el) {
      var svg = el.closest('svg.si-canvas'); if (!svg) return;
      var state = (svg.getAttribute('data-state') || '').split(/\s+/), target = null;
      el.getAttribute('data-count').split(/\s+/).forEach(function (pair) {
        var p = pair.split(':');
        if (p.length === 2 && state.indexOf(p[0]) >= 0) target = parseFloat(p[1]);
      });
      if (target === null) return;
      var from = parseFloat(el.getAttribute('data-count-now'));
      if (isNaN(from)) from = parseFloat((el.textContent || '').replace(/[^0-9.\-]/g, '')) || 0;
      el.setAttribute('data-count-now', target);
      if (!animate || from === target) { el.textContent = fmtCount(target); return; }
      tweenCount(el, from, target);
    });
  }
  function apply(s, k) { applySteps(s, k); applyCanvas(s, k); applyCounts(s); }
  function canvasKey(s) {
    var f = s.querySelectorAll('.si-figure[data-src]');
    return f.length === 1 ? f[0].getAttribute('data-src') : '';
  }

  /* ---------- fit: shrink the fluid type a little when a slide overflows, then scroll ---------- */
  function fit(s) {
    var inner = s && s.querySelector('.si-inner');
    if (!inner || deck.classList.contains('si-overview')) return;
    var f = 1;
    inner.style.setProperty('--_fit', '1');
    for (var t = 0; t < 8 && inner.scrollHeight > inner.clientHeight + 1 && f > 0.62; t++) {
      f -= 0.05;
      inner.style.setProperty('--_fit', f.toFixed(2));
    }
  }

  /* ---------- navigation ---------- */
  function go(i, k, instant) {
    i = Math.max(0, Math.min(n - 1, i));
    var next = slides[i], prev = cur >= 0 ? slides[cur] : null;
    var max = maxStep(next);
    if (k === 'end') k = max;
    k = Math.max(0, Math.min(max, k || 0));
    if (i !== cur) {
      clearStrokes();
      var tx = next.getAttribute('data-transition') || data.transition || 'fade';
      var key = canvasKey(next);
      var carry = !instant && prev && Math.abs(i - cur) === 1 && key !== '' && key === canvasKey(prev);
      if (instant) tx = 'none'; else if (carry) tx = 'canvas';
      deck.setAttribute('data-si-tx', tx);
      if (carry) {
        /* one canvas across slides: start the incoming copy in the outgoing state */
        var from = prev.querySelector('svg.si-canvas'), to = next.querySelector('svg.si-canvas');
        if (from && to) to.setAttribute('data-state', from.getAttribute('data-state') || '');
      }
      slides.forEach(function (s, j) {
        s.classList.toggle('si-past', j < i);
        s.classList.toggle('si-future', j > i);
        s.classList.toggle('si-current', j === i);
        s.setAttribute('aria-hidden', j === i ? 'false' : 'true');
      });
      cur = i;
      applySteps(next, k);
      if (carry) void next.getBoundingClientRect(); /* flush styles so the state change below transitions */
      applyCanvas(next, k);
      applyCounts(next);
      fit(next);
    } else {
      apply(next, k);
    }
    step = k;
    update();
  }
  function forward() {
    if (step < maxStep(slides[cur])) go(cur, step + 1);
    else if (cur < n - 1) go(cur + 1, 0);
  }
  function backward() {
    if (step > 0) go(cur, step - 1);
    else if (cur > 0) go(cur - 1, 'end');
  }
  function update() {
    var max = maxStep(slides[cur]);
    if (counter) counter.textContent = (cur + 1) + ' / ' + n + (max ? '  ·  ' + step + '/' + max : '');
    if (progress) progress.style.width = ((cur + 1) / n * 100) + '%';
    var h = '#' + (cur + 1) + (step ? '.' + step : '');
    if (location.hash !== h) history.replaceState(null, '', location.pathname + location.search + h);
    document.title = (slides[cur].getAttribute('data-title') || data.title) + (cur ? ' · ' + data.title : '');
    syncVideos();
    announce();
  }
  function fromHash(instant) {
    var m = /^#(\d+)(?:\.(\d+))?$/.exec(location.hash);
    go(m ? +m[1] - 1 : 0, m && m[2] ? +m[2] : 0, instant);
  }
  window.addEventListener('hashchange', function () {
    var m = /^#(\d+)(?:\.(\d+))?$/.exec(location.hash);
    if (m && (+m[1] - 1 !== cur || +(m[2] || 0) !== step)) fromHash(false);
  });

  /* ---------- overview ---------- */
  function overview(on) {
    if (on === undefined) on = !deck.classList.contains('si-overview');
    deck.classList.toggle('si-overview', on);
    if (on) {
      slides.forEach(function (s, j) { if (j !== cur) apply(s, maxStep(s)); });
      slides[cur].scrollIntoView({ block: 'center' });
    } else {
      slides.forEach(function (s) { var inner = s.querySelector('.si-inner'); if (inner) inner.style.removeProperty('--_fit'); });
      go(cur, step, true);
    }
  }
  deck.addEventListener('click', function (e) {
    if (!deck.classList.contains('si-overview')) return;
    var s = e.target.closest('.si-slide');
    if (!s) return;
    var i = slides.indexOf(s);
    deck.classList.remove('si-overview');
    go(i, 0, true);
  });

  /* ---------- theme mode (d) ---------- */
  function effectiveDark() {
    var m = root.getAttribute('data-si-mode');
    if (m) return m === 'dark';
    return window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches;
  }
  function toggleMode() {
    var mode = effectiveDark() ? 'light' : 'dark';
    root.setAttribute('data-si-mode', mode);
    save('si-mode', mode);
    renderMermaid(true);
    announce();
  }

  /* ---------- break timer HUD (t): a full-screen countdown to project during a break ---------- */
  var hud = document.querySelector('.si-hud'), hudTime = hud && hud.querySelector('.si-hud-time');
  var hudMins = [5, 10, 15, 20, 30, 45, 60], hudEnd = 0, hudLeft = 0, hudPaused = false, hudTimer = 0;
  function hudPad(x) { return (x < 10 ? '0' : '') + x; }
  function hudFmt(ms) { if (ms < 0) ms = 0; var s = Math.round(ms / 1000); return Math.floor(s / 60) + ':' + hudPad(s % 60); }
  function hudShow(on) {
    if (!hud) return;
    if (on === undefined) on = !hud.classList.contains('is-open');
    hud.classList.toggle('is-open', on);
    hud.setAttribute('aria-hidden', on ? 'false' : 'true');
    if (!on) { clearInterval(hudTimer); hudTimer = 0; hud.classList.remove('is-over'); }
  }
  function hudStart(m) {
    hudEnd = Date.now() + m * 60000; hudPaused = false; hud.classList.remove('is-over');
    clearInterval(hudTimer); hudTick(); hudTimer = setInterval(hudTick, 250);
  }
  function hudPause() { if (!hudTimer) return; hudPaused = !hudPaused; if (!hudPaused) hudEnd = Date.now() + hudLeft; }
  function hudTick() {
    if (!hudPaused) hudLeft = hudEnd - Date.now();
    if (hudTime) hudTime.textContent = hudFmt(hudLeft);
    if (hudLeft <= 0) { hudLeft = 0; if (hudTime) hudTime.textContent = '0:00'; hud.classList.add('is-over'); clearInterval(hudTimer); hudTimer = 0; }
  }

  /* ---------- presenter view (p): a second window synced over BroadcastChannel ---------- */
  var chan = null;
  try { if ('BroadcastChannel' in window) chan = new BroadcastChannel('si:' + data.name); } catch (e) { chan = null; }
  function snapshot(s, k) {
    var c = s.cloneNode(true);
    c.classList.remove('si-past', 'si-future');
    c.classList.add('si-current');
    c.removeAttribute('id');
    c.removeAttribute('aria-hidden');
    var nt = c.querySelector('.si-notes'); if (nt) nt.parentNode.removeChild(nt);
    var inner = c.querySelector('.si-inner'); if (inner) inner.style.removeProperty('--_fit');
    if (k !== undefined) apply(c, k);
    return c.outerHTML;
  }
  function announce() {
    if (!chan || cur < 0) return;
    var s = slides[cur], max = maxStep(s), nx = null, label = 'End of deck';
    if (step < max) { nx = snapshot(s, step + 1); label = 'Next step ' + (step + 1) + '/' + max; }
    else if (cur + 1 < n) { nx = snapshot(slides[cur + 1], 0); label = 'Next: ' + (slides[cur + 1].getAttribute('data-title') || 'slide ' + (cur + 2)); }
    var notes = s.querySelector('.si-notes');
    try {
      chan.postMessage({
        type: 'state', i: cur, k: step, kmax: max, n: n, title: s.getAttribute('data-title') || '',
        minutes: s.getAttribute('data-minutes') || '', total: data.minutes, plan: data.plan, notes: notes ? notes.innerHTML : '',
        cur: snapshot(s), next: nx, label: label, dark: effectiveDark()
      });
    } catch (e) {}
  }
  if (chan) chan.onmessage = function (e) {
    var m = e.data || {};
    if (m.type !== 'cmd') return;
    if (m.cmd === 'next') forward();
    else if (m.cmd === 'prev') backward();
    else if (m.cmd === 'first') go(0, 0);
    else if (m.cmd === 'last') go(n - 1, 'end');
    else if (m.cmd === 'hello') announce();
  };
  function openPresenter() {
    var w = window.open('', 'si-presenter-' + data.name, 'width=1200,height=760');
    if (!w) return;
    var css = Array.prototype.map.call(document.querySelectorAll('style'), function (s) { return s.textContent; }).join('\n');
    var doc = window.SI_PRESENTER
      .split('__CSS__').join(css)
      .split('__TITLE__').join(data.title.replace(/[<&]/g, ''))
      .split('__DATA__').join(JSON.stringify({ name: data.name, title: data.title, aspect: data.aspect }).replace(/</g, '\\u003c'));
    w.document.open(); w.document.write(doc); w.document.close();
    setTimeout(announce, 150);
  }

  /* ---------- keys, touch, controls ---------- */
  function toggleHelp(on) { if (help) help.classList.toggle('is-open', on === undefined ? !help.classList.contains('is-open') : on); }
  function fullscreen() {
    if (document.fullscreenElement) document.exitFullscreen();
    else if (root.requestFullscreen) root.requestFullscreen();
  }
  /* ---------- annotate (a): a transient pen layer over the slides; cleared on slide change ---------- */
  var draw = document.querySelector('.si-draw'), drawBar = document.querySelector('.si-draw-bar');
  var penColors = ['var(--si-accent)', 'var(--si-crit)', 'var(--si-ok)', 'var(--si-fg)'];
  var drawMode = false, erasing = false, drawing = false, penColor = penColors[0], curPath = null;
  function toggleDraw(on) {
    if (!draw) return;
    if (on === undefined) on = !drawMode;
    drawMode = on; if (!on) { drawing = false; erasing = false; }
    document.body.classList.toggle('si-drawing', on);
    draw.setAttribute('aria-hidden', on ? 'false' : 'true');
    syncPenUI();
  }
  function clearStrokes() { if (draw) while (draw.firstChild) draw.removeChild(draw.firstChild); }
  function undoStroke() { if (draw && draw.lastChild) draw.removeChild(draw.lastChild); }
  function cycleColor() { penColor = penColors[(penColors.indexOf(penColor) + 1) % penColors.length]; erasing = false; syncPenUI(); }
  function toggleEraser() { erasing = !erasing; syncPenUI(); }
  function syncPenUI() {
    if (!drawBar) return;
    each(drawBar.querySelectorAll('[data-si-pen]'), function (b) { b.classList.toggle('is-on', !erasing && b.getAttribute('data-si-pen') === penColor); });
    var er = drawBar.querySelector('[data-si-draw="erase"]'); if (er) er.classList.toggle('is-on', erasing);
  }
  function dStart(e) {
    if (!drawMode) return;
    e.preventDefault(); drawing = true;
    if (erasing) { eraseAt(e); return; }
    curPath = document.createElementNS('http://www.w3.org/2000/svg', 'path');
    curPath.setAttribute('d', 'M ' + e.clientX + ' ' + e.clientY);
    curPath.setAttribute('stroke', penColor);
    curPath.setAttribute('stroke-width', ((e.pointerType === 'pen' && e.pressure) ? 1 + e.pressure * 7 : 4).toFixed(1));
    draw.appendChild(curPath);
  }
  function dMove(e) {
    if (!drawMode || !drawing) return;
    if (erasing) { eraseAt(e); return; }
    if (curPath) curPath.setAttribute('d', curPath.getAttribute('d') + ' L ' + e.clientX + ' ' + e.clientY);
  }
  function dEnd() { drawing = false; curPath = null; }
  function eraseAt(e) { var el = document.elementFromPoint(e.clientX, e.clientY); if (el && el.parentNode === draw) draw.removeChild(el); }
  if (draw) {
    draw.addEventListener('pointerdown', dStart);
    window.addEventListener('pointermove', dMove);
    window.addEventListener('pointerup', dEnd);
    window.addEventListener('pointercancel', dEnd);
  }
  if (drawBar) {
    each(drawBar.querySelectorAll('[data-si-pen]'), function (b) { b.addEventListener('click', function () { penColor = b.getAttribute('data-si-pen'); erasing = false; syncPenUI(); b.blur(); }); });
    each(drawBar.querySelectorAll('[data-si-draw]'), function (b) {
      b.addEventListener('click', function () {
        var a = b.getAttribute('data-si-draw');
        if (a === 'erase') toggleEraser(); else if (a === 'undo') undoStroke(); else if (a === 'clear') clearStrokes(); else if (a === 'done') toggleDraw(false);
        b.blur();
      });
    });
    syncPenUI();
  }

  function runCmd(a) {
    if (a === 'overview') { if (feat.overview) overview(); }
    else if (a === 'presenter') { if (feat.presenter) openPresenter(); }
    else if (a === 'draw') { if (feat.draw) toggleDraw(); }
    else if (a === 'timer') { if (feat.timer) hudShow(); }
    else if (a === 'mode') toggleMode();
    else if (a === 'fullscreen') fullscreen();
    else if (a === 'help') toggleHelp();
  }
  var actions = { next: forward, prev: backward, overview: function () { runCmd('overview'); }, draw: function () { runCmd('draw'); }, presenter: function () { runCmd('presenter'); }, mode: toggleMode, help: function () { toggleHelp(); }, fullscreen: fullscreen };
  /* a disabled feature hides its toolbar button */
  if (!feat.overview) { var _bo = document.querySelector('[data-si-action="overview"]'); if (_bo) _bo.style.display = 'none'; }
  if (!feat.presenter) { var _bp = document.querySelector('[data-si-action="presenter"]'); if (_bp) _bp.style.display = 'none'; }
  if (!feat.draw) { var _bd = document.querySelector('[data-si-action="draw"]'); if (_bd) _bd.style.display = 'none'; }
  document.addEventListener('keydown', function (e) {
    if (e.metaKey || e.ctrlKey || e.altKey) return;
    var tag = (document.activeElement && document.activeElement.tagName) || '';
    if (/INPUT|SELECT|TEXTAREA/.test(tag)) return;
    if (hud && hud.classList.contains('is-open')) {
      if (e.key === 'Escape' || e.key === cmd.timer) { e.preventDefault(); hudShow(false); return; }
      if (e.key === ' ') { e.preventDefault(); hudPause(); return; }
      var pi = '1234567'.indexOf(e.key); if (pi >= 0) { e.preventDefault(); hudStart(hudMins[pi]); return; }
      if (/^(Arrow|Page|Home|End|Enter)/.test(e.key)) e.preventDefault();
      return;
    }
    if (drawMode) {
      if (e.key === 'Escape' || e.key === cmd.draw) { e.preventDefault(); toggleDraw(false); return; }
      if (e.key === 'c') { e.preventDefault(); cycleColor(); return; }
      if (e.key === 'e') { e.preventDefault(); toggleEraser(); return; }
      if (e.key === 'z') { e.preventDefault(); undoStroke(); return; }
      if (e.key === 'x') { e.preventDefault(); clearStrokes(); return; }
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
      case 'Escape': toggleHelp(false); if (deck.classList.contains('si-overview')) overview(false); break;
      case 'S': runCmd('presenter'); break;
      case 'h': runCmd('help'); break;
      default: { var act = byKey[e.key]; if (act) runCmd(act); }
    }
  });
  each(document.querySelectorAll('[data-si-action]'), function (b) {
    b.addEventListener('click', function () { var f = actions[b.getAttribute('data-si-action')]; if (f) f(); b.blur(); });
  });
  each(hud ? hud.querySelectorAll('[data-si-min]') : [], function (x) {
    x.addEventListener('click', function () { hudStart(+x.getAttribute('data-si-min')); x.blur(); });
  });
  if (help) help.addEventListener('click', function () { toggleHelp(false); });
  var tx0 = 0, ty0 = 0, t0 = 0;
  document.addEventListener('touchstart', function (e) { var t = e.changedTouches[0]; tx0 = t.clientX; ty0 = t.clientY; t0 = Date.now(); }, { passive: true });
  document.addEventListener('touchend', function (e) {
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

  /* ---------- backgrounds: auto overlay from image brightness; videos play on the current slide only ---------- */
  var reduced = window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  function syncVideos() {
    each(document.querySelectorAll('video.si-bg-media'), function (v) {
      var on = !reduced && v.closest('.si-slide') === slides[cur] && !deck.classList.contains('si-overview');
      if (on) { var p = v.play(); if (p && p.catch) p.catch(function () {}); } else if (!v.paused) v.pause();
    });
  }
  function luminance(r, g, b) { return (0.2126 * r + 0.7152 * g + 0.0722 * b) / 255; }
  each(document.querySelectorAll('.si-bg-overlay[data-overlay="auto"]'), function (ov) {
    var img = ov.parentNode.querySelector('img.si-bg-media');
    if (!img) return;
    function measure() {
      try {
        var c = document.createElement('canvas'); c.width = c.height = 24;
        var g = c.getContext('2d'); g.drawImage(img, 0, 0, 24, 24);
        var px = g.getImageData(0, 0, 24, 24).data, sum = 0;
        for (var i = 0; i < px.length; i += 4) sum += luminance(px[i], px[i + 1], px[i + 2]);
        var lum = sum / (px.length / 4);
        /* a faded image mixes with the page background */
        var op = parseFloat(getComputedStyle(img).opacity) || 1;
        var bg = /rgba?\((\d+),\s*(\d+),\s*(\d+)/.exec(getComputedStyle(document.body).backgroundColor);
        if (bg) lum = lum * op + luminance(+bg[1], +bg[2], +bg[3]) * (1 - op);
        var light = lum > 0.62, slide = ov.closest('.si-slide');
        ov.setAttribute('data-overlay', light ? 'light' : 'dark');
        slide.classList.toggle('si-bg-light', light);
        slide.classList.toggle('si-bg-dark', !light);
      } catch (e) { /* unreadable pixels: keep the dark overlay */ }
    }
    if (img.complete && img.naturalWidth) measure(); else img.addEventListener('load', measure);
  });

  /* ---------- SVG packets: dots that travel along paths marked data-packets ---------- */
  var SVGNS = 'http://www.w3.org/2000/svg';
  each(document.querySelectorAll('svg.si-canvas path[data-packets]'), function (p, idx) {
    if (!p.id) p.id = 'si-pk-' + idx;
    var g = document.createElementNS(SVGNS, 'g');
    g.setAttribute('class', 'si-packets');
    g.setAttribute('data-for', p.getAttribute('data-packets'));
    var dur = parseFloat(p.getAttribute('data-packets-dur') || '2.4');
    var count = parseInt(p.getAttribute('data-packets-count') || '3', 10);
    for (var j = 0; j < count; j++) {
      var c = document.createElementNS(SVGNS, 'circle');
      c.setAttribute('class', 'si-packet');
      c.setAttribute('r', p.getAttribute('data-packets-r') || '7');
      var a = document.createElementNS(SVGNS, 'animateMotion');
      a.setAttribute('dur', dur + 's');
      a.setAttribute('repeatCount', 'indefinite');
      a.setAttribute('begin', (-dur * j / count).toFixed(2) + 's');
      var mp = document.createElementNS(SVGNS, 'mpath');
      mp.setAttribute('href', '#' + p.id);
      a.appendChild(mp); c.appendChild(a); g.appendChild(c);
    }
    p.parentNode.insertBefore(g, p.nextSibling);
  });

  /* ---------- Mermaid (only when the deck has a mermaid fence and the script loaded) ---------- */
  var mermaidSrc = null;
  function cssVar(name) { return getComputedStyle(root).getPropertyValue(name).trim(); }
  function renderMermaid(again) {
    if (!window.mermaid) return;
    var nodes = document.querySelectorAll('pre.si-mermaid');
    if (!nodes.length) return;
    if (mermaidSrc === null) mermaidSrc = Array.prototype.map.call(nodes, function (x) { return x.textContent; });
    if (again) each(nodes, function (x, i) { x.removeAttribute('data-processed'); x.textContent = mermaidSrc[i]; });
    var v = { fg: cssVar('--si-fg'), bg: cssVar('--si-bg'), acc: cssVar('--si-accent'), acc2: cssVar('--si-accent-2'), muted: cssVar('--si-muted'),
      on: cssVar('--si-on-accent'), s1: cssVar('--si-surface'), s2: cssVar('--si-surface-2'), line: cssVar('--si-line') };
    try {
      window.mermaid.initialize({ startOnLoad: false, theme: 'base', fontFamily: cssVar('--si-font-body'), themeVariables: {
        primaryColor: v.s1, primaryTextColor: v.fg, primaryBorderColor: v.acc, secondaryColor: v.s2, tertiaryColor: v.s2,
        lineColor: v.muted, textColor: v.fg, mainBkg: v.s1, nodeBorder: v.acc, clusterBkg: v.s2, clusterBorder: v.line,
        titleColor: v.fg, edgeLabelBackground: v.bg, actorBkg: v.s1, actorBorder: v.acc, actorTextColor: v.fg,
        signalColor: v.fg, signalTextColor: v.fg, noteBkgColor: v.s2, noteTextColor: v.fg, noteBorderColor: v.line } });
      var p = window.mermaid.run({ nodes: nodes });
      if (p && p.then) p.then(function () { fit(slides[cur]); });
    } catch (e) { if (window.console) console.warn('mermaid', e); }
  }

  /* ---------- print: every slide in its final state ---------- */
  function printAll() { slides.forEach(function (s) { apply(s, maxStep(s)); }); }
  window.addEventListener('beforeprint', printAll);
  window.addEventListener('afterprint', function () { go(cur, step, true); });
  /* si build --pdf loads the deck at #print; headless Chrome's --print-to-pdf
     does not reliably fire beforeprint, so force every canvas to its final
     state here too. */
  if (location.hash === '#print') printAll();

  var fitTimer = 0;
  window.addEventListener('resize', function () { clearTimeout(fitTimer); fitTimer = setTimeout(function () { fit(slides[cur]); }, 100); });
  if (window.matchMedia) {
    var mq = window.matchMedia('(prefers-color-scheme: dark)');
    if (mq.addEventListener) mq.addEventListener('change', function () { if (!root.getAttribute('data-si-mode')) renderMermaid(true); });
  }
  if (document.fonts && document.fonts.ready) document.fonts.ready.then(function () { fit(slides[cur]); });
  window.addEventListener('load', function () { renderMermaid(false); fit(slides[cur]); });
  fromHash(true);
  window.SI = { go: go, next: forward, prev: backward, overview: overview, mode: toggleMode };
})();
