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
  /* si play --remote: this page joins the relay; ?role=tp turns it into the
     teleprompter, ?role=draw into a draw pad that follows the deck, ?role=clicker
     into a phone remote, ?role=audience (si play --audience) into a phone in
     the room that follows the talk and never drives it */
  var remoteOn = !!cfg.remote;
  var role = remoteOn ? ((/[?&]role=(tp|draw|clicker|audience)(?:&|$)/.exec(location.search) || [])[1] || '') : '';
  /* ?si-view=tp: this page becomes the teleprompter of the deck open in another
     window or frame of the same browser (si --web Rehearse / Lab, or by hand) */
  if (!role && /[?&]si-view=tp(?:&|$)/.test(location.search)) role = 'tp-local';
  /* ?si-view=thumb: a still picture of one slide (si --web storyboard): every
     build step shown, no sync, no live reload */
  if (!role && /[?&]si-view=thumb(?:&|$)/.test(location.search)) role = 'thumb';
  /* si play --remote on the presenting laptop: q shows the role links as QR codes */
  var remoteQR = !role && window.SI_REMOTE_QR && window.SI_REMOTE_QR.links && window.SI_REMOTE_QR.links.length ? window.SI_REMOTE_QR : null;
  if (remoteQR) cmd.remote = 'q';
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
    if (f.length === 1) return f[0].getAttribute('data-src');
    var cw = s.querySelectorAll('.si-cw[data-cw-id]'); /* a code show across slides morphs like a canvas */
    return !f.length && cw.length === 1 ? 'cw:' + cw[0].getAttribute('data-cw-id') : '';
  }

  /* ---------- fit: when a slide overflows, shrink its figures (canvas, image,
   * diagram) to the room left, then the fluid type a little, then scroll ---------- */
  function fit(s) {
    var inner = s && s.querySelector('.si-inner');
    if (!inner || deck.classList.contains('si-overview')) return;
    var f = 1, g = 1;
    inner.style.setProperty('--_fit', '1');
    inner.style.setProperty('--_figfit', '1');
    var over = function () { return inner.scrollHeight > inner.clientHeight + 1; };
    if (inner.querySelector('.si-figure>svg,.si-image>img,.si-mermaid svg')) {
      for (var u = 0; u < 8 && over() && g > 0.62; u++) {
        g -= 0.05;
        inner.style.setProperty('--_figfit', g.toFixed(2));
      }
    }
    for (var t = 0; t < 8 && over() && f > 0.62; t++) {
      f -= 0.05;
      inner.style.setProperty('--_fit', f.toFixed(2));
    }
  }

