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
  var cmd = { overview: 'o', presenter: 'p', draw: 'a', timer: 't', mode: 'd', fullscreen: 'f', find: '/', help: '?' };
  var termEnabled = !!cfg.terminal; /* si play on a loopback address only; never in a built deck */
  if (termEnabled) cmd.terminal = '`';
  if (cfg.keys) for (var _k in cfg.keys) if (cmd.hasOwnProperty(_k) && cfg.keys[_k]) cmd[_k] = cfg.keys[_k];
  var feat = { draw: true, timer: true, presenter: true, overview: true, find: true };
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
  var hudDial = hud && hud.querySelector('.si-hud-dial');
  var tcfg = cfg.timer || {};
  var hudUnit = tcfg.unit === 'seconds' ? 'seconds' : 'minutes';
  var hudMul = hudUnit === 'seconds' ? 1000 : 60000;
  var hudMins = (tcfg.time && tcfg.time.length) ? tcfg.time : [5, 10, 15, 20, 30, 45, 60];
  var hudAnims = (tcfg.animations && tcfg.animations.length) ? tcfg.animations : [];
  var animKinds = tcfg.animKinds || {};
  var savedAnim = load('si-anim');
  var animCurrent = savedAnim === 'none' ? 'none' : (hudAnims.indexOf(savedAnim) >= 0 ? savedAnim : (hudAnims[0] || ''));
  var savedFace = load('si-face');
  var faceCurrent = (savedFace === 'analog' || savedFace === 'digital') ? savedFace : (tcfg.face === 'analog' ? 'analog' : 'digital');
  var hudEnd = 0, hudLeft = 0, hudTotal = 0, hudPaused = false, hudTimer = 0;
  function hudPad(x) { return (x < 10 ? '0' : '') + x; }
  function hudFmt(ms) { if (ms < 0) ms = 0; var s = Math.round(ms / 1000); return Math.floor(s / 60) + ':' + hudPad(s % 60); }
  var dialArc = hudDial && hudDial.querySelectorAll('circle')[1];
  var dialHand = hudDial && hudDial.querySelector('line');
  var dialText = hudDial && hudDial.querySelector('text');
  var DIAL_C = 2 * Math.PI * 44; // circumference of the r=44 ring
  if (dialArc) dialArc.style.strokeDasharray = DIAL_C;
  function applyFace() {
    var analog = faceCurrent === 'analog' && !!hudDial;
    if (hudTime) hudTime.hidden = analog;
    // hudDial is an <svg>: the `hidden` IDL property is not reflected on SVG
    // elements, so toggle the content attribute directly.
    if (hudDial) { if (analog) hudDial.removeAttribute('hidden'); else hudDial.setAttribute('hidden', ''); }
  }
  function dialUpdate() {
    if (!dialArc) return;
    var elapsed = hudTotal > 0 ? Math.max(0, Math.min(1, 1 - hudLeft / hudTotal)) : 0;
    dialArc.style.strokeDashoffset = DIAL_C * elapsed; // ring empties as time runs down
    if (dialHand) dialHand.setAttribute('transform', 'rotate(' + (elapsed * 360) + ' 50 50)');
    if (dialText) dialText.textContent = hudFmt(hudLeft);
  }
  function hudShow(on) {
    if (!hud) return;
    if (on === undefined) on = !hud.classList.contains('is-open');
    hud.classList.toggle('is-open', on);
    hud.setAttribute('aria-hidden', on ? 'false' : 'true');
    if (on) { applyFace(); dialUpdate(); hudApplyChrome(); startAnim(); } // runs continuously while the HUD is open, independent of the countdown
    else { clearInterval(hudTimer); hudTimer = 0; hud.classList.remove('is-over', 'is-focus'); stopAnim(); }
  }
  // is-focus hides the presets/toggles/hint so only the clock (and the animation)
  // show. It is tied to the countdown: focus while running, chrome back while
  // paused or not started — so Space (pause) reveals the controls, resume hides them.
  function hudApplyChrome() { if (hud) hud.classList.toggle('is-focus', !!hudTimer && !hudPaused); }
  function hudStart(m) {
    hudTotal = m * hudMul;
    hudEnd = Date.now() + hudTotal; hudLeft = hudTotal; hudPaused = false; hud.classList.remove('is-over');
    clearInterval(hudTimer); hudTimer = setInterval(hudTick, 250); hudTick();
    hudApplyChrome();
    if (!animLayer) startAnim(); // keep the already-running loop; only start if not already
  }

  /* ---------- timer animation layer: SLIDE | ANIMATION | TIMER (behind the HUD) ---------- */
  var animLayer = null, animStop = null;
  function animTokens() {
    function v(n) { return getComputedStyle(root).getPropertyValue(n).trim(); }
    return {
      accent: v('--si-accent'), accent2: v('--si-accent-2'), fg: v('--si-fg'), bg: v('--si-bg'),
      muted: v('--si-muted'), ok: v('--si-ok'), warn: v('--si-warn'), crit: v('--si-crit'),
      surface: v('--si-surface'), line: v('--si-line')
    };
  }
  function sizeAnimCanvas(cv) {
    var dpr = window.devicePixelRatio || 1;
    cv.width = Math.max(1, Math.round(window.innerWidth * dpr));
    cv.height = Math.max(1, Math.round(window.innerHeight * dpr));
  }
  function stopAnim() {
    if (animStop) { try { animStop(); } catch (e) {} animStop = null; }
    if (animLayer) { animLayer.remove(); animLayer = null; }
    if (hud) hud.classList.remove('has-anim');
  }
  function startAnim() {
    stopAnim();
    if (!animCurrent || animCurrent === 'none') return;
    animLayer = document.createElement('div');
    animLayer.className = 'si-anim';
    animLayer.setAttribute('aria-hidden', 'true');
    if (animKinds[animCurrent] === 'svg') {
      var tpl = document.getElementById('si-anim-svg-' + animCurrent);
      if (tpl) animLayer.innerHTML = tpl.innerHTML;
      deck.appendChild(animLayer);
    } else if (window.__siAnims && typeof window.__siAnims[animCurrent] === 'function') {
      var cv = document.createElement('canvas');
      animLayer.appendChild(cv); deck.appendChild(animLayer);
      sizeAnimCanvas(cv);
      animLayer._cv = cv;
      var api = {
        tokens: animTokens(), reduced: reduced,
        fraction: function () { return hudTotal > 0 ? Math.max(0, Math.min(1, 1 - hudLeft / hudTotal)) : 0; },
        remaining: function () { return Math.max(0, hudLeft / 1000); }
      };
      try { animStop = window.__siAnims[animCurrent](cv, api); } catch (e) { if (window.console) console.warn('si-anim', e); }
    }
    if (hud) hud.classList.add('has-anim');
  }
  window.addEventListener('resize', function () { if (animLayer && animLayer._cv) sizeAnimCanvas(animLayer._cv); });
  function hudPause() { if (!hudTimer) return; hudPaused = !hudPaused; if (!hudPaused) hudEnd = Date.now() + hudLeft; hudApplyChrome(); }
  function hudTick() {
    if (!hudPaused) hudLeft = hudEnd - Date.now();
    if (hudTime) hudTime.textContent = hudFmt(hudLeft);
    dialUpdate();
    if (hudLeft <= 0) { hudLeft = 0; if (hudTime) hudTime.textContent = '0:00'; dialUpdate(); hud.classList.add('is-over'); clearInterval(hudTimer); hudTimer = 0; hudApplyChrome(); }
  }

  /* ---------- find slide (/) : jump by number or search titles + body ---------- */
  var find = document.querySelector('.si-find');
  var findInput = find && find.querySelector('.si-find-input');
  var findList = find && find.querySelector('.si-find-list');
  var findIndex = null, findResults = [], findSel = 0;
  function escHTML(s) { return s.replace(/[&<>]/g, function (c) { return { '&': '&amp;', '<': '&lt;', '>': '&gt;' }[c]; }); }
  function slideTitle(i) { return slides[i].getAttribute('data-title') || ('Slide ' + (i + 1)); }
  function buildIndex() {
    findIndex = [];
    for (var i = 0; i < n; i++) {
      var inner = slides[i].querySelector('.si-inner');
      var text = (inner ? inner.textContent : slides[i].textContent) || '';
      findIndex.push(text.replace(/\s+/g, ' ').trim());
    }
  }
  function snippet(text, at, len) {
    var s = Math.max(0, at - 30);
    return (s > 0 ? '…' : '') + escHTML(text.slice(s, at)) +
      '<mark>' + escHTML(text.slice(at, at + len)) + '</mark>' +
      escHTML(text.slice(at + len, at + len + 46)) + (at + len + 46 < text.length ? '…' : '');
  }
  function findSearch(q) {
    q = q.trim(); findResults = [];
    if (!q) { for (var i = 0; i < n; i++) findResults.push({ i: i }); return; }
    if (/^\d+$/.test(q)) {
      var num = Math.max(1, Math.min(n, parseInt(q, 10)));
      findResults.push({ i: num - 1, jump: num });
      return;
    }
    var ql = q.toLowerCase(), hits = [];
    for (var j = 0; j < n; j++) {
      var inT = slideTitle(j).toLowerCase().indexOf(ql);
      if (inT >= 0) { hits.push({ i: j, pri: 0, snip: snippet(slideTitle(j), inT, q.length) }); continue; }
      var body = findIndex[j], inB = body.toLowerCase().indexOf(ql);
      if (inB >= 0) hits.push({ i: j, pri: 1, snip: snippet(body, inB, q.length) });
    }
    hits.sort(function (a, b) { return a.pri - b.pri || a.i - b.i; });
    findResults = hits.slice(0, 12);
  }
  function findRender() {
    if (!findList) return;
    if (!findResults.length) { findList.innerHTML = '<li aria-disabled="true">No matches</li>'; return; }
    findList.innerHTML = '';
    findResults.forEach(function (r, idx) {
      var li = document.createElement('li');
      li.setAttribute('data-i', r.i);
      li.setAttribute('aria-selected', idx === findSel ? 'true' : 'false');
      var extra = r.jump ? '<small>go to slide ' + r.jump + '</small>' : (r.snip ? '<small>' + r.snip + '</small>' : '');
      li.innerHTML = '<b>' + (r.i + 1) + '</b> <span>' + escHTML(slideTitle(r.i)) + '</span> ' + extra;
      li.addEventListener('click', function () { findGo(r.i); });
      li.addEventListener('mousemove', function () { if (findSel !== idx) { findSel = idx; markSel(); } });
      findList.appendChild(li);
    });
  }
  function markSel() {
    if (!findList) return;
    each(findList.children, function (li, idx) { li.setAttribute('aria-selected', idx === findSel ? 'true' : 'false'); });
    var sel = findList.children[findSel];
    if (sel && sel.scrollIntoView) sel.scrollIntoView({ block: 'nearest' });
  }
  function findMove(d) {
    if (!findResults.length) return;
    findSel = Math.max(0, Math.min(findResults.length - 1, findSel + d));
    markSel();
  }
  function findGo(i) { findShow(false); go(i, 0); }
  function findShow(on) {
    if (!find || !feat.find) return;
    if (on === undefined) on = !find.classList.contains('is-open');
    find.classList.toggle('is-open', on);
    find.setAttribute('aria-hidden', on ? 'false' : 'true');
    if (on) {
      if (!findIndex) buildIndex();
      findInput.value = ''; findSearch(''); findSel = 0; findRender();
      setTimeout(function () { findInput.focus(); }, 0);
    } else if (findInput) { findInput.blur(); }
  }
  if (findInput) {
    findInput.addEventListener('input', function () { findSearch(findInput.value); findSel = 0; findRender(); });
    findInput.addEventListener('keydown', function (e) {
      // stopPropagation so the key never reaches the global nav handler — notably
      // Enter, which (after findGo blurs the input) would otherwise advance a slide.
      if (e.key === 'Escape') { e.preventDefault(); e.stopPropagation(); findShow(false); }
      else if (e.key === 'ArrowDown') { e.preventDefault(); e.stopPropagation(); findMove(1); }
      else if (e.key === 'ArrowUp') { e.preventDefault(); e.stopPropagation(); findMove(-1); }
      else if (e.key === 'Enter') { e.preventDefault(); e.stopPropagation(); if (findResults.length) findGo(findResults[findSel].i); }
    });
    find.addEventListener('click', function (e) { if (e.target === find) findShow(false); });
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
  var penSize = +load('si-pen-size') || 5;
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
    each(drawBar.querySelectorAll('[data-si-size]'), function (b) { b.classList.toggle('is-on', +b.getAttribute('data-si-size') === penSize); });
    var er = drawBar.querySelector('[data-si-draw="erase"]'); if (er) er.classList.toggle('is-on', erasing);
  }
  function dStart(e) {
    if (!drawMode) return;
    e.preventDefault(); drawing = true;
    if (erasing) { eraseAt(e); return; }
    curPath = document.createElementNS('http://www.w3.org/2000/svg', 'path');
    curPath.setAttribute('d', 'M ' + e.clientX + ' ' + e.clientY);
    // inline style (not a presentation attribute) so the CSS var() resolves
    curPath.style.stroke = penColor;
    var w = (e.pointerType === 'pen' && e.pressure) ? penSize * (0.5 + e.pressure) : penSize;
    curPath.setAttribute('stroke-width', w.toFixed(1));
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
    each(drawBar.querySelectorAll('[data-si-size]'), function (b) { b.addEventListener('click', function () { penSize = +b.getAttribute('data-si-size'); save('si-pen-size', penSize); syncPenUI(); b.blur(); }); });
    each(drawBar.querySelectorAll('[data-si-draw]'), function (b) {
      b.addEventListener('click', function () {
        var a = b.getAttribute('data-si-draw');
        if (a === 'erase') toggleEraser(); else if (a === 'undo') undoStroke(); else if (a === 'clear') clearStrokes(); else if (a === 'done') toggleDraw(false);
        b.blur();
      });
    });
    syncPenUI();
  }

  /* Play terminal drawer: lazy-load the runtime (xterm.js + term.js) on first
   * toggle, so a page that never opens the terminal pays nothing for it, and a
   * built deck (termEnabled false) never references it at all. */
  var termLoaded = false;
  function toggleTerminal() {
    if (!termEnabled) return;
    if (window.siTerm) { window.siTerm.toggle(); return; }
    if (termLoaded) return; // load in flight
    termLoaded = true;
    var css = document.createElement('link');
    css.rel = 'stylesheet'; css.href = '/_si/term/assets/xterm.css';
    document.head.appendChild(css);
    function add(src, next) { var s = document.createElement('script'); s.src = src; s.onload = next; document.head.appendChild(s); }
    add('/_si/term/assets/xterm.js', function () {
      add('/_si/term/assets/addon-fit.js', function () {
        add('/_si/term/assets/term.js', function () { if (window.siTerm) window.siTerm.toggle(); });
      });
    });
  }

  function runCmd(a) {
    if (a === 'overview') { if (feat.overview) overview(); }
    else if (a === 'presenter') { if (feat.presenter) openPresenter(); }
    else if (a === 'draw') { if (feat.draw) toggleDraw(); }
    else if (a === 'timer') { if (feat.timer) hudShow(); }
    else if (a === 'mode') toggleMode();
    else if (a === 'fullscreen') fullscreen();
    else if (a === 'find') { if (feat.find) findShow(); }
    else if (a === 'help') toggleHelp();
    else if (a === 'terminal') toggleTerminal();
  }
  var actions = { next: forward, prev: backward, overview: function () { runCmd('overview'); }, draw: function () { runCmd('draw'); }, presenter: function () { runCmd('presenter'); }, find: function () { runCmd('find'); }, mode: toggleMode, help: function () { toggleHelp(); }, fullscreen: fullscreen };
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
