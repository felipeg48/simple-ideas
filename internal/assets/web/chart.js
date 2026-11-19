/* si charts: every .si-chart on the page (a ```chart fence, render/chart.go).
   The Go side draws every step as a frame, all in one grid cell; the one on
   screen has data-on. Moving to another frame tweens what both frames show
   from the old values to the new ones: a bar's width, a column's height, a
   donut's arcs, a line's points, the numbers written (they count), and rows
   that change place glide there (FLIP). A frame of another kind cross-fades
   in and grows. A chart's first appearance grows from zero: bars from the
   left, a donut around, a line drawn on. On a slide the frames follow the
   slide's build steps (.si-chart-mark) and the same id= on the next slide
   carries on from what this one showed; in a page the chart plays when it
   is scrolled into view, with ← → and Play under it. Reduced motion, print,
   the overview, thumbnails and handouts get the final values at once. */
(function () {
  'use strict';
  var figs = document.querySelectorAll('.si-chart');
  if (!figs.length) return;
  var q = location.search;
  var reduce = !!(window.matchMedia && matchMedia('(prefers-reduced-motion: reduce)').matches);
  var still = /si-view=thumb|si-handout=|si-ig-film/.test(q);
  var MORPH = 900, GROW = 1100;
  var shown = {}; /* id -> the frame the last chart with that id showed */
  var NUM = /-?\d*\.?\d+(?:e[-+]?\d+)?/gi, TXT = /[−-]?\d[\d,]*(?:\.\d+)?/g;
  var SEL = 'rect:not(.si-ig-track),circle[data-key],polyline,polygon,.si-ig-end,.si-ig-last,.si-ig-bv,.si-ig-dv,.si-ig-legend b,.si-chart-at,.si-c3-world,.si-c3-pie,.si-c3-cub,.si-c3-cub b';

  function each(r, sel, f) { Array.prototype.forEach.call(r.querySelectorAll(sel), f); }
  function clamp(x) { return x < 0 ? 0 : x > 1 ? 1 : x; }
  function inout(x) { return x < .5 ? 4 * x * x * x : 1 - Math.pow(-2 * x + 2, 3) / 2; }
  function out3(x) { return 1 - Math.pow(1 - x, 3); }
  function printing() { return !!(window.matchMedia && matchMedia('print').matches); }

  /* numbers in an attribute ("0,10 50,20", "30 70", "45%") move from a's to b's */
  function mixNums(a, b, p) {
    var an = (String(a).match(NUM) || []).map(Number), i = 0;
    if (an.length !== (String(b).match(NUM) || []).length) return b;
    return String(b).replace(NUM, function (m) { var v = an[i] + (Number(m) - an[i]) * p; i++; return String(Math.round(v * 1000) / 1000); });
  }
  /* numbers as written ("1,200", "12.5%", "−3", "2024") count from a's to b's */
  function tnum(m) { return parseFloat(m.replace(/,/g, '').replace('−', '-')); }
  function mixText(a, b, p) {
    var am = String(a).match(TXT) || [], i = 0;
    if (!am.length || am.length !== (String(b).match(TXT) || []).length) return b;
    return String(b).replace(TXT, function (m) {
      var from = tnum(am[i++]), to = tnum(m), dot = m.indexOf('.'), dec = dot < 0 ? 0 : m.length - dot - 1;
      var v = from + (to - from) * p, s = Math.abs(v).toFixed(dec);
      if (m.indexOf(',') >= 0) { var ps = s.split('.'); ps[0] = ps[0].replace(/\B(?=(\d{3})+(?!\d))/g, ','); s = ps.join('.'); }
      return (v < 0 && +s !== 0 ? (m.charAt(0) === '−' ? '−' : '-') : '') + s;
    });
  }

  /* the parts of a frame worth tweening, named so the same part of another
     frame (another step, another slide) is found again: its row's key, its
     class or tag, its place among those */
  function parts(frame) {
    var out = {}, n = {};
    each(frame, SEL, function (e) {
      var row = e.closest('[data-key]');
      var id = (row && frame.contains(row) ? row.getAttribute('data-key') : '') + '|' + (e.getAttribute('class') || e.tagName);
      n[id] = (n[id] || 0) + 1;
      out[id + '|' + n[id]] = e;
    });
    return out;
  }
  function props(e) {
    var t = e.tagName.toLowerCase();
    if (t === 'rect') return ['@x', '@y', '@width', '@height'];
    if (t === 'circle') return ['@stroke-dasharray', '@stroke-dashoffset', '@transform'];
    if (e.classList.contains('si-c3-cub')) return ['--h', '--w', '--d', 'transform'];
    if (e.classList.contains('si-c3-world') || e.classList.contains('si-c3-pie')) return ['transform'];
    if (t === 'polyline' || t === 'polygon') return ['@points'];
    if (e.classList.contains('si-ig-end')) return ['top'];
    if (e.classList.contains('si-ig-last')) return ['top', '#'];
    return ['#'];
  }
  function get(e, p) { return p === '#' ? e.textContent : p.charAt(0) === '@' ? e.getAttribute(p.slice(1)) : p.charAt(0) === '-' ? e.style.getPropertyValue(p) : e.style[p]; }
  function put(e, p, v) {
    if (p === '#') e.textContent = v;
    else if (p.charAt(0) === '-') e.style.setProperty(p, v);
    else if (p.charAt(0) !== '@') e.style[p] = v;
    else if (v === null) e.removeAttribute(p.slice(1));
    else e.setAttribute(p.slice(1), v);
  }
  function rows(frame) { return Array.prototype.slice.call(frame.querySelectorAll('.si-ig-bar[data-key],.si-ig-col[data-key]')); }

  /* what a frame shows, as values: the start of a move to another frame */
  function snap(frame) {
    var ps = parts(frame), vals = {};
    for (var id in ps) { var e = ps[id], v = {}; props(e).forEach(function (p) { v[p] = get(e, p); }); vals[id] = v; }
    return { kind: frame.getAttribute('data-kind'), vals: vals, order: rows(frame).map(function (r) { return r.getAttribute('data-key'); }) };
  }

  /* a value at its start when the part is new (a first appearance) */
  function zero(e, p, v) {
    var col = !!e.closest('.si-ig-col');
    if (p === 'transform' && e.hasAttribute('data-from')) return e.getAttribute('data-from'); /* the camera turns, a layer drops in */
    if (p === '--h' && e.closest('[data-c3="bar3d"]')) return '0em';
    if (p === '@width' && !col) return '0';
    if (p === '@x' && !col) return '0';
    if (p === '@height' && col) return '0';
    if (p === '@y' && col) return '100';
    if (p === '#' && !e.classList.contains('si-chart-at')) return String(v).replace(TXT, '0');
    return v;
  }

  /* the tracks that move frame B from S (null: from nothing) to what B shows */
  function tracks(B, S) {
    var out = [], ps = parts(B), rs = rows(B);
    var keyed = Array.prototype.slice.call(B.querySelectorAll('.si-ig-bar[data-key],.si-ig-col[data-key],.si-c3-cub'));
    var n = Math.max(1, keyed.length), grow = !S;
    var spread = B.getAttribute('data-kind') === 'funnel3d' ? 1.2 : 0.3;
    for (var id in ps) (function (e, from) {
      var rank = Math.max(0, keyed.indexOf(e.closest('[data-key]')));
      if (e.tagName.toLowerCase() === 'circle' && !from) return; /* a donut sweeps round (below) */
      if (/^poly/i.test(e.tagName) && !from) return; /* a line is drawn on (below) */
      props(e).forEach(function (p) {
        var to = get(e, p), a = from ? from[p] : zero(e, p, to);
        if (a === undefined || a === null || to === null || a === to) return;
        var text = p === '#';
        out.push({ delay: grow ? rank / n * spread : 0, run: function (k) {
          put(e, p, k >= 1 ? to : text ? mixText(a, to, k) : mixNums(a, to, k));
        } });
      });
    })(ps[id], S && S.vals[id]);
    if (S) { /* rows that change place glide to it */
      rs.forEach(function (r, i) {
        var was = S.order.indexOf(r.getAttribute('data-key'));
        if (was < 0 || was === i || !rs[was]) return;
        var els = getComputedStyle(r).display === 'contents' ? Array.prototype.slice.call(r.children) : [r];
        var box = function (x) { var c = getComputedStyle(x).display === 'contents' ? x.children[0] : x; return c.getBoundingClientRect(); };
        var a = box(rs[was]), b = box(r), col = r.classList.contains('si-ig-col');
        var d = col ? a.left - b.left : a.top - b.top;
        out.push({ run: function (k) {
          var v = k >= 1 ? '' : col ? (d * (1 - k)).toFixed(2) + 'px 0' : '0 ' + (d * (1 - k)).toFixed(2) + 'px';
          els.forEach(function (x) { x.style.translate = v; });
        } });
      });
    }
    if (!S) each(B, 'svg', function (svg) { /* a donut sweeps round, each layer of a 3D one too */
      var shares = Array.prototype.map.call(svg.querySelectorAll('circle[data-key]'), function (c) { return { c: c, s: parseFloat(c.getAttribute('stroke-dasharray')) || 0, to: c.getAttribute('stroke-dasharray') }; });
      if (shares.length) out.push({ run: function (k) {
        var before = 0, sweep = k * 100;
        shares.forEach(function (x) {
          var v = Math.max(0, Math.min(x.s, sweep - before));
          before += x.s;
          x.c.setAttribute('stroke-dasharray', k >= 1 ? x.to : v.toFixed(3) + ' ' + (100 - v).toFixed(3));
        });
      } });
    });
    each(B, '.si-ig-plot', function (pl) {
      if (S) return;
      out.push({ run: function (k) { pl.style.clipPath = k >= 1 ? '' : 'inset(-50% ' + (100 - 100 * k).toFixed(2) + '% -50% 0)'; } });
    });
    return out;
  }

  function run(list, dur, ease, done) {
    var t0 = 0, raf = 0, over = false;
    var h = { end: function () {
      if (over) return;
      over = true; cancelAnimationFrame(raf);
      list.forEach(function (t) { t.run(1); });
      if (done) done();
    } };
    list.forEach(function (t) { t.run(0); });
    var tick = function (t) {
      if (over) return;
      if (!t0) t0 = t;
      var left = false;
      list.forEach(function (tr) {
        var x = clamp((t - t0 - (tr.delay || 0) * dur) / dur);
        if (x < 1) left = true;
        tr.run(x >= 1 ? 1 : ease(x));
      });
      if (left) raf = requestAnimationFrame(tick); else h.end();
    };
    raf = requestAnimationFrame(tick);
    return h;
  }

  function Chart(fig) {
    var self = this;
    this.fig = fig;
    this.frames = Array.prototype.slice.call(fig.querySelectorAll('.si-chart-frame'));
    this.id = fig.getAttribute('data-chart-id') || '';
    this.slide = fig.closest('.si-slide');
    this.marks = Array.prototype.slice.call(fig.querySelectorAll('.si-chart-mark'));
    this.k = Math.max(0, this.frames.findIndex(function (f) { return f.hasAttribute('data-on'); }));
    if (this.slide) this.slideMode(); else this.pageMode();
    window.addEventListener('beforeprint', function () {
      self.stop();
      if (!self.slide) self.on(self.frames.length - 1);
    });
  }
  var C = Chart.prototype;
  C.on = function (k) {
    this.k = k;
    this.frames.forEach(function (f, i) { f.toggleAttribute('data-on', i === k); });
    if (this.id && (!this.slide || this.slide.classList.contains('si-current'))) shown[this.id] = this.frames[k];
    if (this.out) {
      var at = this.frames[k].querySelector('.si-chart-at');
      this.out.textContent = at ? at.textContent : (k + 1) + ' / ' + this.frames.length;
    }
  };
  C.stop = function () {
    if (this.anim) { var a = this.anim; this.anim = null; a.end(); }
    if (this.fade) { this.fade.cancel(); this.fade = null; }
    each(this.fig, '.si-chart-frame.is-out', function (f) { f.classList.remove('is-out'); });
  };
  C.motion = function () {
    if (reduce || still || printing()) return false;
    var deck = document.querySelector('.si-deck');
    if (deck && deck.classList.contains('si-overview')) return false;
    return !this.slide || this.slide.classList.contains('si-current');
  };
  /* show frame k: from S (a snapshot), from the frame on screen, or growing */
  C.show = function (k, animate, S) {
    k = Math.max(0, Math.min(this.frames.length - 1, k));
    this.stop();
    var A = this.frames[this.k], B = this.frames[k];
    if (animate && S === undefined && A !== B) S = snap(A);
    this.on(k);
    if (!animate) return;
    var same = S && S.kind === B.getAttribute('data-kind');
    if (S && !same && A !== B) { /* another kind: the old frame fades out over the new one growing */
      A.classList.add('is-out');
      this.fade = A.animate([{ opacity: 1 }, { opacity: 0 }], { duration: GROW * 0.5, easing: 'ease' });
      var self = this, fa = this.fade;
      fa.onfinish = function () { A.classList.remove('is-out'); if (self.fade === fa) self.fade = null; };
    }
    this.anim = same ? run(tracks(B, S), MORPH, inout) : run(tracks(B, null), GROW, out3);
  };

  C.markK = function () { var k = 0; this.marks.forEach(function (m) { if (m.classList.contains('is-visible')) k++; }); return k; };
  C.slideMode = function () {
    var self = this, slide = this.slide;
    var wrap = this.fig.parentElement && this.fig.parentElement.closest('.si-step'); /* the chart itself is a build step */
    var seen = function () { return slide.classList.contains('si-current') && (!wrap || wrap.classList.contains('is-visible')); };
    var was = seen();
    var sync = function () {
      var now = seen(), k = self.markK();
      if (now && !was) { /* it arrives: from what the same id showed on the slide before, or growing */
        var prev = self.id && shown[self.id];
        var from = prev && self.frames.indexOf(prev) < 0 ? snap(prev) : null;
        self.show(k, self.motion(), from);
      } else if (k !== self.k) self.show(k, now && self.motion());
      was = now;
    };
    this.on(this.markK());
    var mo = new MutationObserver(sync);
    mo.observe(slide, { attributes: true, attributeFilter: ['class'] });
    this.marks.forEach(function (m) { mo.observe(m, { attributes: true, attributeFilter: ['class'] }); });
    if (wrap) mo.observe(wrap, { attributes: true, attributeFilter: ['class'] });
    if (was) this.show(this.k, this.motion(), null);
  };

  C.pageMode = function () {
    var self = this, n = this.frames.length;
    if (this.fig.hasAttribute('data-chart-nav')) this.nav();
    this.on(this.k);
    if (!this.motion() || !window.IntersectionObserver) return;
    this.on(0);
    var io = new IntersectionObserver(function (es) {
      if (!es.some(function (e) { return e.isIntersecting; })) return;
      io.disconnect();
      self.show(0, true, null);
      if (n > 1) self.play(GROW + 500);
    }, { threshold: 0.25 });
    io.observe(this.fig);
  };
  /* a page: step through to the end, one step every 1.8 s */
  C.play = function (first) {
    var self = this;
    clearTimeout(this.timer);
    if (this.playBtn) this.playBtn.setAttribute('aria-pressed', 'true');
    var next = function () {
      if (self.k >= self.frames.length - 1) { self.halt(); return; }
      self.show(self.k + 1, self.motion());
      self.timer = setTimeout(next, 1800);
    };
    this.timer = setTimeout(next, first || 0);
  };
  C.halt = function () { clearTimeout(this.timer); this.timer = 0; if (this.playBtn) this.playBtn.setAttribute('aria-pressed', 'false'); };
  C.nav = function () {
    var self = this, nav = document.createElement('div');
    nav.className = 'si-chart-nav';
    nav.innerHTML = '<button type="button" data-a="prev" aria-label="Previous step">←</button><output aria-live="polite"></output><button type="button" data-a="next" aria-label="Next step">→</button><button type="button" data-a="play" aria-pressed="false">Play</button>';
    this.fig.appendChild(nav);
    this.out = nav.querySelector('output');
    this.playBtn = nav.querySelector('[data-a="play"]');
    nav.addEventListener('click', function (e) {
      var b = e.target.closest('button'); if (!b) return;
      var a = b.getAttribute('data-a');
      if (a === 'play') {
        if (self.timer) { self.halt(); return; }
        if (self.k >= self.frames.length - 1) self.show(0, self.motion(), null);
        self.play(self.motion() ? GROW : 600);
        return;
      }
      self.halt();
      self.show(self.k + (a === 'next' ? 1 : -1), self.motion());
    });
  };

  function start() { each(document, '.si-chart', function (f) { if (!f.siChart && f.querySelector('.si-chart-frame')) f.siChart = new Chart(f); }); }
  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', start); else start();
})();
