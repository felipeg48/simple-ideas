/* si scenes: every .si-scene on the page (a ```scene fence, render/scene.go).
   The figure carries, as JSON, where every object is after every step
   (s[k][i]: x, y, size, corner, turn, shown, lit, label) and how each step
   moves (p[k-1]: the verb, its objects, its places). The next step plays as
   motion: a parcel flies along its trail, a token hops and lights the blocks,
   a block morphs, splits, merges, swaps, opens; tiles assemble or scatter; a
   queue fills and its consumer takes from the front; a route draws itself
   and drops pins; a parcel drops into a box. Every other change (a step
   back, a #N.k jump, the overview, reduced motion, print, thumbnails,
   handouts) sets that step's state at once. On a slide the steps follow the
   slide's build steps (.si-scene-mark); in a page the scene plays once when
   it is scrolled into view, with ← → and Play under it. */
(function () {
  'use strict';
  var figs = document.querySelectorAll('.si-scene[data-scene]');
  if (!figs.length) return;
  var NS = 'http://www.w3.org/2000/svg';
  var reduce = !!(window.matchMedia && matchMedia('(prefers-reduced-motion: reduce)').matches);
  var still = /si-view=thumb|si-handout=|si-ig-film/.test(location.search);
  /* about how long each verb plays at its own pace (ms), so · 2s can set it */
  var BASE = { flies: 2100, hops: 3000, morphs: 2600, splits: 1650, merge: 1700, swap: 1100, assemble: 1700, scatter: 1400,
    queue: 3500, takes: 2400, 'fans out': 1100, opens: 1700, route: 2400, 'drops into': 1100 };

  function printing() { return !!(window.matchMedia && matchMedia('print').matches); }
  function lerp(a, b, t) { return a + (b - a) * t; }
  function lin(t) { return t; }
  function inout(t) { return t < .5 ? 4 * t * t * t : 1 - Math.pow(-2 * t + 2, 3) / 2; }
  function out3(t) { return 1 - Math.pow(1 - t, 3); }
  function back(t) { var c = 2.55; return 1 + (c + 1) * Math.pow(t - 1, 3) + c * Math.pow(t - 1, 2); }
  function snap(t) { return 1 - Math.pow(1 - t, 3) * Math.cos(t * 4.5); }
  function seeded(s) { s = s % 2147483647 || 1; return function () { s = s * 16807 % 2147483647; return (s - 1) / 2147483646; }; }
  function num(v, d) { return v == null ? d : v; }

  function Scene(fig) {
    var self = this, d = JSON.parse(fig.getAttribute('data-scene'));
    this.fig = fig;
    this.S = d.s;
    this.P = d.p || [];
    this.n = this.P.length;
    this.svg = fig.querySelector('svg');
    this.fx = fig.querySelector('.si-scene-fx');
    this.els = [];
    this.kinds = [];
    Array.prototype.forEach.call(fig.querySelectorAll('[data-o]'), function (g) {
      var i = +g.getAttribute('data-o');
      self.els[i] = g;
      self.kinds[i] = (/si-scene-(?!o\b)([a-z]+)/.exec(g.getAttribute('class')) || [])[1];
    });
    this.k = +fig.getAttribute('data-on') || 0;
    this.jobs = [];
    this.raf = 0;
    this.slide = fig.closest('.si-slide');
    this.deck = document.querySelector('.si-deck');
    this.marks = Array.prototype.slice.call(fig.querySelectorAll('.si-scene-mark'));
    if (this.slide) this.slideMode(); else this.pageMode();
    var before = this.k;
    window.addEventListener('beforeprint', function () { before = self.k; self.halt(); self.set(self.n); });
    window.addEventListener('afterprint', function () { self.set(self.slide ? self.markK() : before); });
  }
  var C = Scene.prototype;

  /* draw object i in state st, with o overriding: x y rot sx sy op on w h r a l ly */
  C.draw = function (i, st, o) {
    var g = this.els[i], kind = this.kinds[i];
    if (!g || !st) return;
    o = o || {};
    if (kind !== 'trail' && kind !== 'route') {
      var t = 'translate(' + num(o.x, st.x) + ' ' + num(o.y, st.y) + ')';
      if (o.rot) t += ' rotate(' + o.rot + ')';
      if (o.sx != null) t += ' scale(' + o.sx + ' ' + num(o.sy, o.sx) + ')';
      g.setAttribute('transform', t);
    }
    g.setAttribute('opacity', num(o.op, st.v ? 1 : 0));
    g.classList.toggle('is-on', !!num(o.on, st.on));
    if (kind === 'box' || kind === 'tile' || kind === 'piece') {
      var r = g.firstChild, w = num(o.w, st.w || 0), h = num(o.h, st.h || 0);
      r.setAttribute('x', -w / 2); r.setAttribute('y', -h / 2);
      r.setAttribute('width', w); r.setAttribute('height', h);
      r.setAttribute('rx', num(o.r, st.r || 0));
      if (kind === 'box') {
        r.setAttribute('transform', 'rotate(' + num(o.a, st.a || 0) + ')');
        var tx = g.lastChild, l = num(o.l, st.l || '');
        if (tx.textContent !== l) tx.textContent = l;
        tx.setAttribute('y', num(o.ly, st.ly || 0));
      }
    }
  };
  /* step k at once */
  C.set = function (k) {
    this.stop();
    this.k = k;
    var self = this, st = this.S[k];
    st.forEach(function (s, i) { self.draw(i, s); });
    Array.prototype.forEach.call(this.fig.querySelectorAll('.si-scene-trail path,.si-scene-route path'), function (p) {
      p.style.strokeDasharray = ''; p.style.strokeDashoffset = '';
    });
    this.label();
  };
  C.label = function () { if (this.out) this.out.textContent = this.k + ' / ' + this.n; };

  /* the clock: jobs run from the frame they are first seen */
  C.tw = function (delay, dur, fn, ease, done) {
    this.jobs.push({ t0: 0, delay: delay || 0, dur: dur, fn: fn, ease: ease || inout, done: done });
    if (!this.raf) { var self = this; this.raf = requestAnimationFrame(function (t) { self.tick(t); }); }
  };
  C.wait = function (ms, fn) { this.tw(ms, 0, null, lin, fn); };
  C.tick = function (now) {
    var self = this, over = [];
    this.raf = 0;
    this.jobs.slice().forEach(function (j) {
      if (!j.t0) j.t0 = now;
      var t = j.dur > 0 ? (now - j.t0 - j.delay) / j.dur : (now - j.t0 >= j.delay ? 1 : -1);
      if (t < 0) return;
      t = Math.min(1, t);
      if (j.fn) j.fn(j.ease(t), t);
      if (t >= 1) over.push(j);
    });
    this.jobs = this.jobs.filter(function (j) { return over.indexOf(j) < 0; });
    over.forEach(function (j) { if (j.done) j.done(); });
    if (this.jobs.length) this.raf = requestAnimationFrame(function (t) { self.tick(t); });
    else if (this.idle) { var f = this.idle; this.idle = null; f(); }
  };
  C.stop = function () {
    this.jobs = [];
    this.idle = null;
    if (this.raf) cancelAnimationFrame(this.raf);
    this.raf = 0;
    if (this.fx) this.fx.textContent = '';
  };

  /* play step k (from step k-1); done runs when it has settled */
  C.play = function (k, done) {
    var self = this, p = this.P[k - 1];
    this.set(k - 1);
    this.k = k;
    this.label();
    var f = p.d ? p.d / (BASE[p.v] || 1500) : (p.sp || 1);
    if (V[p.v]) V[p.v].call(this, p, this.S[k - 1], this.S[k], f);
    if (!this.jobs.length) { this.set(k); if (done) done(); return; }
    this.idle = function () { self.set(k); if (done) done(); };
  };
  C.motion = function () {
    if (reduce || still || printing()) return false;
    if (this.deck && this.deck.classList.contains('si-overview')) return false;
    return !this.slide || this.slide.classList.contains('si-current');
  };

  var V = {};
  V.flies = function (p, A, B, f) {
    var self = this, o = p.o[0], tr = p.o[1], path = this.els[tr].querySelector('path'), L = path.getTotalLength();
    this.draw(tr, B[tr]);
    path.style.strokeDasharray = L; path.style.strokeDashoffset = L;
    this.draw(o, B[o], { x: p.from[0], y: p.from[1] });
    this.tw(0, 1600 * f, function (e) {
      path.style.strokeDashoffset = L * (1 - e);
      var a = path.getPointAtLength(e * L), b = path.getPointAtLength(Math.min(L, e * L + 1));
      self.draw(o, B[o], { x: a.x, y: a.y, rot: e < 1 ? Math.atan2(b.y - a.y, b.x - a.x) * 180 / Math.PI : 0 });
    }, inout, function () {
      path.style.strokeDasharray = ''; path.style.strokeDashoffset = '';
      self.tw(0, 500 * f, function (e) { var k = 1 + .25 * Math.sin(e * Math.PI); self.draw(o, B[o], { sx: k, sy: 2 - k }); }, lin);
    });
  };
  V.hops = function (p, A, B, f) {
    var self = this, o = p.o[0], pts = p.pts, L = p.lights, i = 0;
    this.draw(L[0], A[L[0]], { on: true });
    this.draw(o, B[o], { x: pts[0][0], y: pts[0][1] });
    var hop = function () {
      if (i >= pts.length - 1) return;
      var a = pts[i], b = pts[i + 1], bi = L[++i];
      self.tw(0, 560 * f, function (e) { self.draw(o, B[o], { x: lerp(a[0], b[0], e), y: lerp(a[1], b[1], e) - Math.sin(e * Math.PI) * 34 }); }, inout, function () {
        self.tw(0, 300 * f, function (e) { self.draw(bi, A[bi], { on: true, sx: 1 + .12 * Math.sin(e * Math.PI) }); }, lin, function () { self.wait(120 * f, hop); });
      });
    };
    hop();
  };
  V.morphs = function (p, A, B, f) {
    var self = this, o = p.o[0], cur = A[o], j = 0;
    var next = function () {
      if (j >= p.st.length) return;
      var to = p.st[j++], from = cur;
      self.wait(400 * f, function () {
        self.tw(0, 900 * f, function (e) {
          self.draw(o, A[o], { w: lerp(from.w, to.w, e), h: lerp(from.h, to.h, e), r: lerp(from.r || 0, to.r || 0, e), a: lerp(from.a || 0, to.a || 0, e), l: e < .5 ? from.l : to.l });
        }, inout, function () { cur = to; next(); });
      });
    };
    next();
  };
  V.splits = V['fans out'] = function (p, A, B, f) {
    var self = this, n = p.o.length, mid = (n - 1) / 2, fr = p.from, fan = p.v === 'fans out';
    (p.hide || []).forEach(function (i) { self.tw(0, 250 * f, function (e) { self.draw(i, A[i], { op: 1 - e, sx: 1 - e * .5 }); }, lin); });
    if (p.src >= 0) self.tw(0, 500 * f, function (e) { self.draw(p.src, B[p.src], { sx: 1 + .12 * Math.sin(e * Math.PI) }); }, lin);
    p.o.forEach(function (i, j) {
      var b = B[i];
      self.draw(i, b, { x: fr[0], y: fr[1], sx: 0 });
      if (fan) self.tw(250 * f, 800 * f, function (e) { self.draw(i, b, { x: lerp(fr[0], b.x, e), y: lerp(fr[1], b.y, 1 - Math.pow(1 - e, 2)), sx: Math.min(1, e * 5) }); }, lin);
      else self.tw((350 + j * 140) * f, 1000 * f, function (e) {
        self.draw(i, b, { x: lerp(fr[0], b.x, e), y: lerp(fr[1], b.y, e) - Math.sin(e * Math.PI) * (j === Math.round(mid) ? 28 : 12), rot: (1 - e) * (j - mid) * 40, sx: Math.min(1, e * 4) });
      });
    });
  };
  V.merge = function (p, A, B, f) {
    var self = this, t = p.t, n = p.o.length, mid = (n - 1) / 2, left = n;
    p.o.forEach(function (i, j) {
      var a = A[i], tx = B[t].x, ty = B[t].y;
      self.tw(j * 80 * f, 1100 * f, function (e) {
        self.draw(i, a, { x: lerp(a.x, tx, e), y: lerp(a.y, ty, e) - Math.sin(e * Math.PI) * 20, rot: (1 - e) * 90 * (j - mid), sx: lerp(1, .4, e), op: 1 });
      }, inout, function () {
        if (--left) return;
        p.o.forEach(function (x) { self.draw(x, B[x]); });
        self.tw(0, 500 * f, function (e) { self.draw(t, B[t], { sx: 1 + .18 * (1 - e) }); }, out3);
      });
    });
  };
  V.swap = function (p, A, B, f) {
    var self = this, a = p.o[0], b = p.o[1];
    this.tw(0, 1100 * f, function (e) {
      var s = Math.sin(e * Math.PI) * 48;
      self.draw(a, B[a], { x: lerp(A[a].x, B[a].x, e), y: lerp(A[a].y, B[a].y, e) - s });
      self.draw(b, B[b], { x: lerp(A[b].x, B[b].x, e), y: lerp(A[b].y, B[b].y, e) + s });
    });
  };
  V.assemble = function (p, A, B, f) {
    var self = this, rnd = seeded(p.o.length * 7919), W = this.svg.viewBox.baseVal.width, H = this.svg.viewBox.baseVal.height, left = p.o.length;
    p.o.forEach(function (i, j) {
      var b = B[i], sx = rnd() * W, sy = rnd() < .5 ? -30 : H + 30, r = (rnd() - .5) * 360;
      self.draw(i, b, { x: sx, y: sy, rot: r, sx: .4, op: 1 });
      self.tw(j * 90 * f, 900 * f, function (e) { self.draw(i, b, { x: lerp(sx, b.x, e), y: lerp(sy, b.y, e), rot: r * (1 - e), sx: lerp(.4, 1, e), op: 1 }); }, snap, function () {
        if (--left || p.t < 0) return;
        p.o.forEach(function (x) { self.tw(0, 350 * f, function (e) { self.draw(x, B[x], { op: 1 - e }); }, lin); });
        self.tw(0, 450 * f, function (e) { self.draw(p.t, B[p.t], { sx: lerp(.7, 1, e), op: Math.min(1, e * 2) }); }, back);
      });
    });
  };
  V.scatter = function (p, A, B, f) {
    var self = this, c = p.from, rnd = seeded(p.o.length * 104729);
    if (p.src >= 0) this.draw(p.src, B[p.src]);
    p.o.forEach(function (i) {
      var s = A[i].v ? A[i] : B[i], x = s.x, y = s.y;
      var a = Math.atan2(y - c[1], x - c[0]) + (rnd() - .5) * .6, d = 120 + rnd() * 80, r = (rnd() - .5) * 400;
      self.draw(i, s, { op: 1 });
      self.tw((150 + rnd() * 120) * f, 1100 * f, function (e) {
        self.draw(i, s, { x: x + Math.cos(a) * d * e, y: y + Math.sin(a) * d * e + 60 * e * e, rot: r * e, sx: 1 - e * .5, op: 1 - e });
      }, function (t) { return 1 - Math.pow(1 - t, 2); });
    });
  };
  V.queue = V.takes = function (p, A, B, f) {
    var self = this, lane = p.lane || [], slots = p.slots || [], fill = p.fill || [], take = p.take || 0, c = p.t, fr = p.from, y = fr[1], t = 0;
    fill.forEach(function (i, j) {
      var k = lane.indexOf(i);
      self.draw(i, B[i], { x: fr[0], y: y, op: 0 });
      self.tw(j * 160 * f, 500 * f, function (e) { self.draw(i, B[i], { x: lerp(fr[0], slots[k], e), y: y, op: Math.min(1, e * 3) }); });
    });
    if (!take) return;
    var one = function () {
      if (t >= take) return;
      var i = lane[t];
      self.tw(0, 500 * f, function (e) {
        self.draw(i, B[i], { x: lerp(slots[0], B[c].x, e), y: lerp(y, B[c].y, e) - Math.sin(e * Math.PI) * 30, sx: lerp(1, .5, e), op: 1 });
      }, inout, function () {
        self.draw(i, B[i]);
        self.draw(c, B[c], { on: true });
        self.wait(250 * f, function () { self.draw(c, B[c]); });
      });
      for (var m = t + 1; m < lane.length; m++) (function (el, from) {
        self.tw(0, 500 * f, function (e) { self.draw(el, B[el], { x: lerp(slots[from], slots[from - 1], e), y: y, op: 1 }); });
      })(lane[m], m - t);
      t++;
      self.wait(800 * f, one);
    };
    this.wait(fill.length ? (fill.length * 160 + 600) * f : 0, one);
  };
  V.opens = function (p, A, B, f) {
    var self = this, b = p.src, a = A[b], z = B[b];
    (p.hide || []).forEach(function (i) { self.tw(0, 400 * f, function (e) { self.draw(i, A[i], { op: 1 - e }); }, lin); });
    p.o.forEach(function (i) { self.draw(i, B[i], { op: 0 }); });
    this.tw(0, 900 * f, function (e) {
      self.draw(b, z, { x: lerp(a.x, z.x, e), y: lerp(a.y, z.y, e), w: lerp(a.w, z.w, e), h: lerp(a.h, z.h, e), ly: lerp(a.ly || 0, z.ly || 0, e), a: lerp(a.a || 0, 0, e), r: lerp(a.r || 0, z.r || 0, e) });
    }, inout, function () {
      p.o.forEach(function (i, j) {
        self.tw(j * 160 * f, 450 * f, function (e) { self.draw(i, B[i], { sx: lerp(.3, 1, e), op: Math.min(1, e * 2) }); }, back);
      });
    });
  };
  V.route = function (p, A, B, f) {
    var self = this, ro = p.o[0], rider = p.o[1], path = this.els[ro].querySelector('path'), L = path.getTotalLength(), pins = p.lights || [], at = p.f || [], n = 0;
    this.draw(ro, B[ro]);
    path.style.strokeDasharray = L; path.style.strokeDashoffset = L;
    pins.forEach(function (i) { self.draw(i, B[i], { op: 0 }); });
    this.tw(0, 2400 * f, function (e) {
      path.style.strokeDashoffset = L * (1 - e);
      var pt = path.getPointAtLength(e * L);
      self.draw(rider, B[rider], { x: pt.x, y: pt.y });
      while (n < pins.length && e >= at[n] - .001) (function (i) {
        n++;
        self.tw(0, 350 * f, function (k) { self.draw(i, B[i], { y: B[i].y - 20 * (1 - k), op: Math.min(1, k * 2) }); }, back);
      })(pins[n]);
    }, lin);
  };
  V['drops into'] = function (p, A, B, f) {
    var self = this, o = p.o[0], z = B[o], box = B[p.t], floor = box.y + box.h / 2, w = box.w, y0 = z.y - 110;
    this.draw(p.t, box);
    this.tw(0, 650 * f, function (e) { self.draw(o, z, { y: lerp(y0, z.y, e), sx: 1 - .08 * e, sy: 1 + .12 * e }); }, function (t) { return t * t; }, function () {
      self.tw(0, 420 * f, function (e) { var k = Math.sin(e * Math.PI) * Math.exp(-2 * e); self.draw(o, z, { y: z.y + k * 5, sx: 1 + .25 * k, sy: 1 - .22 * k }); }, lin);
      for (var i = 0; i < 6; i++) (function (c, dir) {
        self.fx.appendChild(c);
        self.tw(0, 600 * f, function (e) {
          c.setAttribute('cx', z.x + dir * (w * .3 + e * w * .4)); c.setAttribute('cy', floor - e * 10);
          c.setAttribute('r', 2 + e * 3); c.setAttribute('opacity', 1 - e);
        }, out3);
      })(document.createElementNS(NS, 'circle'), i < 3 ? -1 - i * .3 : 1 + (i - 3) * .3);
    });
  };

  C.markK = function () {
    var k = 0;
    this.marks.forEach(function (m) { if (m.classList.contains('is-visible')) k++; });
    return k;
  };
  C.slideMode = function () {
    var self = this, slide = this.slide;
    var overview = function () { return self.deck && self.deck.classList.contains('si-overview'); };
    var was = slide.classList.contains('si-current');
    var sync = function () {
      var now = slide.classList.contains('si-current'), k = still || overview() ? self.n : self.markK();
      if (now && was && k === self.k + 1 && self.motion()) self.play(k);
      else if (k !== self.k || !now) { if (k !== self.k) self.set(k); else self.stop(); }
      was = now;
    };
    this.set(still ? this.n : this.markK());
    var mo = new MutationObserver(sync);
    mo.observe(slide, { attributes: true, attributeFilter: ['class'] });
    this.marks.forEach(function (m) { mo.observe(m, { attributes: true, attributeFilter: ['class'] }); });
    if (this.deck) mo.observe(this.deck, { attributes: true, attributeFilter: ['class'] });
  };
  C.pageMode = function () {
    var self = this;
    if (this.fig.hasAttribute('data-scene-nav')) this.nav();
    this.set(this.n);
    if (!this.n || !this.motion() || !window.IntersectionObserver) return;
    this.set(0);
    var io = new IntersectionObserver(function (es) {
      if (!es.some(function (e) { return e.isIntersecting; })) return;
      io.disconnect();
      self.auto(300);
    }, { threshold: 0.35 });
    io.observe(this.fig);
  };
  /* a page: every step in turn, a pause between them */
  C.auto = function (first) {
    var self = this;
    this.halt();
    if (this.playBtn) this.playBtn.setAttribute('aria-pressed', 'true');
    var next = function () {
      if (self.k >= self.n) { self.halt(); return; }
      self.go(self.k + 1, function () { self.timer = setTimeout(next, 700); });
    };
    this.timer = setTimeout(next, first || 0);
  };
  C.go = function (k, done) {
    if (this.motion()) this.play(k, done);
    else { this.set(k); if (done) this.timer = setTimeout(done, 500); }
  };
  C.halt = function () {
    clearTimeout(this.timer);
    this.timer = 0;
    if (this.playBtn) this.playBtn.setAttribute('aria-pressed', 'false');
  };
  C.nav = function () {
    var self = this, nav = document.createElement('div');
    nav.className = 'si-scene-nav';
    nav.innerHTML = '<button type="button" data-a="prev" aria-label="Previous step">←</button><output aria-live="polite"></output><button type="button" data-a="next" aria-label="Next step">→</button><button type="button" data-a="play" aria-pressed="false">Play</button>';
    this.fig.appendChild(nav);
    this.out = nav.querySelector('output');
    this.playBtn = nav.querySelector('[data-a="play"]');
    nav.addEventListener('click', function (e) {
      var b = e.target.closest('button');
      if (!b) return;
      var a = b.getAttribute('data-a');
      if (a === 'play') {
        if (self.timer) { self.halt(); return; }
        if (self.k >= self.n) self.set(0);
        self.auto(200);
        return;
      }
      self.halt();
      if (a === 'next' && self.k < self.n) self.go(self.k + 1);
      if (a === 'prev' && self.k > 0) self.set(self.k - 1);
    });
  };

  function start() {
    Array.prototype.forEach.call(document.querySelectorAll('.si-scene[data-scene]'), function (f) { if (!f.siScene) f.siScene = new Scene(f); });
  }
  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', start); else start();
})();
