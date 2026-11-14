/* simple-ideas infographics: a block whose content does not fit its cell
   lowers its text (--_fit on the block) step by step, down to a floor of 60%;
   a block that still does not fit is marked data-overflow (si check warns
   about it before: render/infographic.go igFit). Text is in cqi/em, so the fit
   holds at every size and is measured again only when the frame changes. */
(function () {
  var FLOOR = 0.6, STEP = 0.05;
  function spills(e) { return e.clientHeight > 0 && (e.scrollHeight > e.clientHeight + 1 || e.scrollWidth > e.clientWidth + 1); }
  function over(b) {
    if (spills(b)) return true;
    for (var i = 0; i < b.children.length; i++) { /* a child that clips its own content (a tall number's glyphs may poke out: not a clip) */
      var c = b.children[i], o = getComputedStyle(c);
      if ((o.overflowX !== 'visible' || o.overflowY !== 'visible') && spills(c)) return true;
    }
    return false;
  }
  function fit(b) {
    b.style.removeProperty('--_fit');
    var f = 1;
    while (f > FLOOR && over(b)) {
      f = Math.max(FLOOR, Math.round((f - STEP) * 100) / 100);
      b.style.setProperty('--_fit', f);
    }
    if (over(b)) b.setAttribute('data-overflow', ''); else b.removeAttribute('data-overflow');
  }
  function all() { Array.prototype.forEach.call(document.querySelectorAll('.si-ig-block'), fit); }
  var shape = '';
  function onResize() {
    var s = window.innerWidth + 'x' + window.innerHeight;
    if (s !== shape) { shape = s; clearTimeout(onResize.t); onResize.t = setTimeout(all, 120); }
  }
  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', all); else all();
  window.addEventListener('load', all);
  window.addEventListener('resize', onResize);
  if (document.fonts && document.fonts.ready) document.fonts.ready.then(all);
  window.siInfographicFit = all;
})();
/* simple-ideas animated infographics (animate: build | loop): every part
   (the header, each block) carries data-enter, data-at and data-dur, written
   by si (render/igmotion.go), and one clock draws them all: render(S, t) sets
   each part to where it is t seconds in, so playing is a loop of render and
   si build --gif/--mp4 seeks it frame by frame (?si-ig-film: no autoplay,
   window.siIG.seek). A page plays on load, a slide when it becomes the
   current one (again on every visit), a block shown by a build step when it
   is shown. Reduced motion, thumbnails and print get the last frame. */
(function () {
  var figs = Array.prototype.slice.call(document.querySelectorAll('.si-ig[data-animate]'));
  var q = location.search, film = /[?&]si-ig-film\b/.test(q), thumb = /si-view=thumb/.test(q);
  if (film) document.documentElement.classList.add('si-ig-film');
  if (!figs.length) return;
  var reduce = !!(window.matchMedia && matchMedia('(prefers-reduced-motion: reduce)').matches);
  var PKT = 0.7, framed = false;
  try { framed = window.self !== window.top; } catch (e) { framed = true; }
  if (framed) document.documentElement.classList.add('si-ig-framed');
  function clamp(x) { return x < 0 ? 0 : x > 1 ? 1 : x; }
  function ease(x) { x = clamp(x); return 1 - Math.pow(1 - x, 3); }
  function inout(x) { x = clamp(x); return x < 0.5 ? 4 * x * x * x : 1 - Math.pow(-2 * x + 2, 3) / 2; }
  function each(root, sel, f) { Array.prototype.forEach.call(root.querySelectorAll(sel), f); }

  /* counters: the number as written ("1,200", "2.4", "−3") counts up from 0 */
  function counter(el) {
    if (!el._n) {
      var t = el.textContent, c = t.replace(/,/g, '').replace('−', '-'), dot = c.indexOf('.');
      el._n = { t: t, v: parseFloat(c), dec: dot < 0 ? 0 : c.length - dot - 1, comma: t.indexOf(',') >= 0 };
    }
    return el._n;
  }
  function count(el, e) {
    each(el, '.si-ig-cnt', function (c) {
      var n = counter(c);
      if (e >= 1 || isNaN(n.v)) { c.textContent = n.t; return; }
      var s = Math.abs(n.v * e).toFixed(n.dec);
      if (n.comma) { var p = s.split('.'); p[0] = p[0].replace(/\B(?=(\d{3})+(?!\d))/g, ','); s = p.join('.'); }
      c.textContent = (n.v < 0 && n.v * e !== 0 ? '−' : '') + s;
    });
  }
  /* charts grow: bars and progress from the left, a donut sweeps, a line draws */
  function grow(el, p) {
    var rects = el.querySelectorAll('.si-ig-bar rect:not(.si-ig-track)'), n = rects.length;
    Array.prototype.forEach.call(rects, function (r, i) {
      r.style.transform = 'scaleX(' + ease((p - i / Math.max(n, 1) * 0.35) / 0.65).toFixed(4) + ')';
    });
    var before = 0, sweep = ease(p) * 100;
    each(el, '.si-ig-donut circle:not(.si-ig-track)', function (c) {
      if (c._share === undefined) c._share = parseFloat(c.getAttribute('stroke-dasharray')) || 0;
      var v = Math.max(0, Math.min(c._share, sweep - before));
      before += c._share;
      c.style.strokeDasharray = v.toFixed(3) + ' ' + (100 - v).toFixed(3);
    });
    each(el, '.si-ig-plot', function (pl) { pl.style.clipPath = p >= 1 ? '' : 'inset(-50% ' + (100 - 100 * ease(p)).toFixed(2) + '% -50% 0)'; });
  }
  /* steps light one by one */
  function light(el, p) {
    var lis = el.querySelectorAll('.si-ig-steps li'), n = lis.length;
    Array.prototype.forEach.call(lis, function (li, i) {
      var k = ease(p * n - i);
      li.style.opacity = (0.15 + 0.85 * k).toFixed(3);
      var c = li.querySelector('.si-ig-n');
      if (c) c.style.scale = (0.55 + 0.45 * k).toFixed(3);
    });
  }
  /* a flow draws itself: boxes appear, arrows draw, then packets travel */
  function flow(el, p, after) {
    each(el, '.si-igf', function (f) {
      var E = +f.getAttribute('data-events') || 1;
      each(f, '[data-ev]', function (g) {
        var k = ease(p * E - +g.getAttribute('data-ev'));
        if (g.classList.contains('si-igf-node')) {
          g.style.opacity = k.toFixed(3);
          g.style.scale = (0.82 + 0.18 * k).toFixed(3);
        } else {
          var path = g.querySelector('.si-igf-edge');
          path.style.strokeDasharray = '1 1';
          path.style.strokeDashoffset = (1 - k).toFixed(4);
          var o = clamp((k - 0.6) / 0.4).toFixed(3);
          for (var i = 0; i < g.children.length; i++) if (g.children[i] !== path) g.children[i].style.opacity = o;
        }
      });
      var pk = f.querySelectorAll('.si-igf-pkt'), n = pk.length, hot = -1, at = 0;
      if (n && after >= 0 && isFinite(after)) {
        var ph = after % (n * PKT);
        hot = Math.floor(ph / PKT);
        at = inout((ph - hot * PKT) / PKT);
      }
      Array.prototype.forEach.call(pk, function (c, i) {
        var arrow = f.querySelector('.si-igf-arrow[data-e="' + i + '"]');
        if (arrow) arrow.classList.toggle('is-hot', i === hot);
        if (i !== hot) { c.style.opacity = 0; return; }
        var path = arrow.querySelector('.si-igf-edge'), L = path.getTotalLength(), pt = path.getPointAtLength(at * L);
        c.setAttribute('cx', pt.x.toFixed(1));
        c.setAttribute('cy', pt.y.toFixed(1));
        c.style.opacity = Math.min(1, Math.min(at, 1 - at) * 8).toFixed(3);
      });
    });
  }
  /* one part, x seconds after it began (Infinity: in, at rest) */
  function show(P, x, fin) {
    var el = P.el, p = clamp(x / P.dur), e = ease(p), s = el.style;
    s.opacity = ''; s.translate = ''; s.scale = '';
    switch (P.enter) {
      case 'fade': s.opacity = e.toFixed(3); break;
      case 'rise': s.opacity = e.toFixed(3); s.translate = '0 ' + ((1 - e) * 1.4).toFixed(3) + 'em'; break;
      case 'grow':
        if (P.kind === 'chart') { s.opacity = clamp(p * 4).toFixed(3); grow(el, p); }
        else { s.opacity = e.toFixed(3); s.scale = (0.9 + 0.1 * e).toFixed(4); }
        break;
      case 'count': s.opacity = clamp(p * 3).toFixed(3); s.translate = '0 ' + ((1 - ease(p * 2)) * 0.8).toFixed(3) + 'em'; break;
      case 'light': s.opacity = clamp(p * 5).toFixed(3); light(el, p); break;
      case 'draw': s.opacity = clamp(p * 6).toFixed(3); break;
    }
    if (P.enter !== 'grow' && P.kind === 'chart') grow(el, 1);
    count(el, P.enter === 'count' ? e : 1);
    if (P.kind === 'flow') flow(el, P.enter === 'draw' ? p : 1, fin ? -1 : x - P.dur);
  }
  function part(el) {
    return { el: el, at: +el.getAttribute('data-at') || 0, dur: +el.getAttribute('data-dur') || 0.7, enter: el.getAttribute('data-enter'),
      kind: el.getAttribute('data-kind') || '', step: !film && el.classList.contains('si-step'), vis: null };
  }
  /* the figure t seconds in */
  function render(S, t) {
    var fin = !S.loop && t >= S.D;
    S.parts.forEach(function (P) {
      if (P.step) show(P, P.vis === null ? 0 : t - P.vis, fin);
      else show(P, t - P.at, fin);
    });
    var fade = S.loop ? clamp((S.D - t) / 0.5) : 1;
    each(S.fig, '.si-ig-grid,.si-ig-head', function (g) { g.style.opacity = fade < 1 ? fade.toFixed(3) : ''; });
  }
  function final(S) { S.parts.forEach(function (P) { show(P, Infinity, true); }); }
  var all = figs.map(function (fig) {
    var S = { fig: fig, parts: Array.prototype.map.call(fig.querySelectorAll('.si-ig-head[data-enter],.si-ig-block[data-enter]'), part),
      D: +fig.getAttribute('data-duration') || 6, loop: fig.getAttribute('data-animate') === 'loop', t0: null, raf: 0, slide: fig.closest('.si-slide') };
    S.parts.forEach(function (P) { if (P.step) P.el.style.transition = 'none'; });
    return S;
  });
  function now(S) { return (performance.now() - S.t0) / 1000; }
  function tick(S) {
    S.raf = 0;
    if (S.t0 === null) return;
    var t = now(S);
    if (S.loop && t >= S.D) { S.t0 += S.D * 1000 * Math.floor(t / S.D); t = now(S); }
    var busy = S.parts.some(function (P) { return P.step && P.vis !== null && t - P.vis < P.dur; });
    render(S, t);
    if (S.loop || t < S.D || busy) S.raf = requestAnimationFrame(function () { tick(S); });
    label(S);
  }
  function play(S) {
    if (S.raf) cancelAnimationFrame(S.raf);
    S.t0 = performance.now();
    S.parts.forEach(function (P) { if (P.step) P.vis = P.el.classList.contains('is-visible') ? 0 : null; });
    tick(S);
  }
  function rest(S) { /* off screen: back to the start, ready to play again */
    if (S.raf) cancelAnimationFrame(S.raf);
    S.raf = 0; S.t0 = null;
    S.parts.forEach(function (P) { P.vis = null; });
    render(S, 0);
    label(S);
  }
  function label(S) {
    if (S.btn) S.btn.textContent = S.t0 === null ? '▶ Play' : '↺ Replay';
  }
  window.siIG = {
    duration: Math.max.apply(null, all.map(function (S) { return S.D; })),
    loop: all.some(function (S) { return S.loop; }),
    seek: function (t) { all.forEach(function (S) { render(S, S.loop ? t % S.D : Math.min(t, S.D)); }); },
    final: function () { all.forEach(final); },
    replay: function () { all.forEach(function (S) { if (!S.slide || S.slide.classList.contains('si-current')) play(S); }); }
  };
  if (film) { window.siIG.seek(0); return; }
  if (thumb || reduce) { all.forEach(final); return; }
  window.addEventListener('beforeprint', function () { all.forEach(final); });
  all.forEach(function (S) {
    var b = document.createElement('button');
    b.type = 'button'; b.className = 'si-ig-play';
    b.setAttribute('aria-label', 'Play the animation again');
    ['click', 'pointerdown', 'mousedown', 'touchstart'].forEach(function (ev) { b.addEventListener(ev, function (e) { e.stopPropagation(); }); });
    b.addEventListener('click', function () { play(S); });
    S.fig.appendChild(b);
    S.btn = b;
    S.parts.forEach(function (P) {
      if (!P.step) return;
      new MutationObserver(function () {
        var on = P.el.classList.contains('is-visible');
        if (on && P.vis === null && S.t0 !== null) { P.vis = now(S); if (!S.raf) tick(S); }
        else if (!on && P.vis !== null) { P.vis = null; render(S, S.t0 === null ? 0 : now(S)); }
      }).observe(P.el, { attributes: true, attributeFilter: ['class'] });
    });
    if (S.slide) {
      var cur = function () { return S.slide.classList.contains('si-current'); };
      new MutationObserver(function () {
        if (cur() && S.t0 === null) play(S);
        else if (!cur() && S.t0 !== null) rest(S);
      }).observe(S.slide, { attributes: true, attributeFilter: ['class'] });
      if (cur()) play(S); else rest(S);
    } else {
      render(S, 0);
      var go = function () { if (S.t0 === null) play(S); };
      if (document.fonts && document.fonts.ready) document.fonts.ready.then(go); else go();
      setTimeout(go, 400);
    }
  });
})();
