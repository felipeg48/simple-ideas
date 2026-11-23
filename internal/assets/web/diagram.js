/* si delivery diagrams: every .si-dg on the page (```kanban, ```gantt,
   ```sequence; render/diagram.go). The figure carries its elements (e), their
   states after every step (s[k][i]: x y w h, opacity o, drawn/progress p,
   classes c, text t and u) and where they enter from (in). A step plays as a
   tween: cards glide to their column, bars grow and fill, the today line
   moves, a message draws itself. Every other change (back, a jump, the
   overview, reduced motion, print, thumbnails) sets the state at once. On a
   slide the steps follow its build steps (.si-dg-mark); in a page it plays
   when scrolled into view, with ← → and Play. ?si-ig-film: window.siIG.seek
   draws any moment, for --gif and --mp4. */
(function () {
  'use strict';
  var figs = document.querySelectorAll('.si-dg[data-dg]');
  if (!figs.length) return;
  var reduce = !!(window.matchMedia && matchMedia('(prefers-reduced-motion: reduce)').matches);
  var still = /si-view=thumb|si-handout=/.test(location.search);
  var film = /si-ig-film/.test(location.search);
  var TW = 0.75, ENTER = 0.9, BEAT = 1.7; /* seconds: a step's tween, the entrance, a step in a film or a page's Play */
  function ease(t) { return t < .5 ? 4 * t * t * t : 1 - Math.pow(-2 * t + 2, 3) / 2; }
  function lerp(a, b, t) { return a + (b - a) * t; }
  function num(v) { return v || 0; }
  function printing() { return !!(window.matchMedia && matchMedia('print').matches); }

  function mix(A, B, f) {
    if (f >= 1) return B;
    return B.map(function (b, i) {
      var a = A[i] || b;
      return { x: lerp(num(a.x), num(b.x), f), y: lerp(num(a.y), num(b.y), f), w: lerp(num(a.w), num(b.w), f), h: lerp(num(a.h), num(b.h), f),
        o: lerp(num(a.o), num(b.o), f), p: lerp(num(a.p), num(b.p), f), c: f > 0 ? b.c : a.c, t: f > 0 ? b.t : a.t, u: f > 0 ? b.u : a.u };
    });
  }

  function Dg(fig) {
    var self = this, d = JSON.parse(fig.getAttribute('data-dg'));
    this.fig = fig;
    this.E = d.e;
    this.S = d.s;
    this.IN = d['in'];
    this.L = d.l || [];
    this.n = this.S.length - 1;
    this.g = [];
    Array.prototype.forEach.call(fig.querySelectorAll('[data-e]'), function (g) { self.g[+g.getAttribute('data-e')] = g; });
    this.k = +fig.getAttribute('data-on') || 0;
    this.shown = this.S[this.k];
    this.raf = 0;
    this.slide = fig.closest('.si-slide');
    this.deck = document.querySelector('.si-deck');
    this.marks = Array.prototype.slice.call(fig.querySelectorAll('.si-dg-mark'));
    if (film) { this.draw(this.IN || this.S[0]); return; }
    if (this.slide) this.slideMode(); else this.pageMode();
    var before = this.k;
    window.addEventListener('beforeprint', function () { before = self.k; self.halt(); self.set(self.n); });
    window.addEventListener('afterprint', function () { self.set(self.slide ? self.markK() : before); });
  }
  var P = Dg.prototype;

  P.label = function (txt, s) {
    txt.textContent = s.t || '';
    if (s.u) {
      var u = document.createElementNS('http://www.w3.org/2000/svg', 'tspan');
      u.setAttribute('class', 'u'); u.setAttribute('dx', '6'); u.textContent = s.u;
      txt.appendChild(u);
    }
  };
  P.draw = function (st) {
    var self = this;
    this.shown = st;
    this.E.forEach(function (e, i) {
      var g = self.g[i], s = st[i];
      if (!g || !s) return;
      if (e.k === 'clip') { g.setAttribute('x', num(s.x)); g.setAttribute('y', num(s.y)); g.setAttribute('width', Math.max(0, num(s.w))); g.setAttribute('height', num(s.h)); return; }
      g.setAttribute('transform', 'translate(' + num(s.x) + ' ' + num(s.y) + ')');
      g.setAttribute('opacity', num(s.o));
      g.setAttribute('class', ('si-dg-e ' + (e.c || '') + ' ' + (s.c || '')).trim());
      var r, t;
      if (e.k === 'box') {
        r = g.querySelector('rect'); r.setAttribute('width', Math.max(0, num(s.w))); r.setAttribute('height', Math.max(0, num(s.h)));
        t = g.querySelectorAll('text');
        if (t[0]) { t[0].textContent = s.t || ''; t[0].setAttribute('y', num(s.h) / 2 - (s.u ? 8 : 0)); if (/si-dg-actor|si-dg-mid/.test(e.c)) t[0].setAttribute('x', num(s.w) / 2); }
        if (t[1]) { t[1].textContent = s.u || ''; t[1].setAttribute('y', num(s.h) / 2 + 10); }
      } else if (e.k === 'bar') {
        r = g.querySelectorAll('rect');
        r[0].setAttribute('width', Math.max(0, num(s.w))); r[0].setAttribute('height', num(s.h));
        r[1].setAttribute('width', Math.max(0, num(s.w) * Math.min(1, num(s.p)))); r[1].setAttribute('height', num(s.h));
      } else if (e.k === 'line') {
        var pa = g.querySelector('path'), pp = Math.min(1, num(s.p));
        if (pp >= 1) { pa.removeAttribute('pathLength'); pa.removeAttribute('stroke-dasharray'); pa.removeAttribute('stroke-dashoffset'); } /* whole: its own dash pattern */
        else { pa.setAttribute('pathLength', '1'); pa.setAttribute('stroke-dasharray', '1 1'); pa.setAttribute('stroke-dashoffset', 1 - pp); }
        t = g.querySelector('text'); if (t) self.label(t, s);
      } else if (e.k === 'dot') {
        g.querySelector('circle').setAttribute('r', Math.max(0, num(s.w)));
      } else if (e.k === 'text') {
        t = g.querySelector('text'); if (t) self.label(t, s);
      }
    });
  };
  P.motion = function () { return !reduce && !still && !printing() && (!this.slide || this.slide.classList.contains('si-current')); };
  P.set = function (k) { this.stop(); this.k = k; this.draw(this.S[k]); this.out && this.count(); };
  P.stop = function () { if (this.raf) cancelAnimationFrame(this.raf); this.raf = 0; };
  /* tween from what is shown now to state B over secs */
  P.tween = function (B, secs, done) {
    var self = this, A = this.shown, t0 = null;
    this.stop();
    var tick = function (now) {
      if (t0 === null) t0 = now;
      var f = Math.min(1, (now - t0) / (secs * 1000));
      self.draw(mix(A, B, ease(f)));
      if (f < 1) self.raf = requestAnimationFrame(tick);
      else { self.raf = 0; if (done) done(); }
    };
    this.raf = requestAnimationFrame(tick);
  };
  P.play = function (k, done) { this.k = k; this.tween(this.S[k], TW, done); if (this.out) this.count(); };
  P.enter = function (done) {
    if (!this.IN) { if (done) done(); return; }
    this.draw(this.IN);
    this.tween(this.S[this.k], ENTER, done);
  };

  P.markK = function () {
    var k = 0;
    this.marks.forEach(function (m) { if (m.classList.contains('is-visible')) k++; });
    return k;
  };
  P.slideMode = function () {
    var self = this, slide = this.slide;
    var overview = function () { return self.deck && self.deck.classList.contains('si-overview'); };
    var was = slide.classList.contains('si-current');
    var sync = function () {
      var now = slide.classList.contains('si-current'), k = still || overview() ? self.n : self.markK();
      if (now && !was && self.motion() && !overview()) { self.k = k; self.enter(); }
      else if (now && was && k === self.k + 1 && self.motion()) self.play(k);
      else if (k !== self.k || !now) self.set(k);
      was = now;
    };
    this.set(still ? this.n : this.markK());
    if (was && this.motion()) this.enter();
    var mo = new MutationObserver(sync);
    mo.observe(slide, { attributes: true, attributeFilter: ['class'] });
    this.marks.forEach(function (m) { mo.observe(m, { attributes: true, attributeFilter: ['class'] }); });
    if (this.deck) mo.observe(this.deck, { attributes: true, attributeFilter: ['class'] });
  };
  P.pageMode = function () {
    var self = this;
    if (this.fig.hasAttribute('data-dg-nav')) this.nav();
    this.set(this.n);
    if (!this.motion() || !window.IntersectionObserver) return;
    this.k = 0;
    this.draw(this.IN || this.S[0]);
    var io = new IntersectionObserver(function (es) {
      if (!es.some(function (e) { return e.isIntersecting; })) return;
      io.disconnect();
      self.enter(function () { self.auto(400); });
    }, { threshold: 0.3 });
    io.observe(this.fig);
  };
  P.auto = function (first) {
    var self = this;
    this.halt();
    if (this.playBtn) this.playBtn.setAttribute('aria-pressed', 'true');
    var next = function () {
      if (self.k >= self.n) { self.halt(); return; }
      self.play(self.k + 1, function () { self.timer = setTimeout(next, (BEAT - TW) * 1000); });
    };
    this.timer = setTimeout(next, first || 0);
  };
  P.halt = function () {
    clearTimeout(this.timer);
    this.timer = 0;
    if (this.playBtn) this.playBtn.setAttribute('aria-pressed', 'false');
  };
  P.count = function () { this.out.textContent = this.k ? this.L[this.k - 1] + ' · ' + this.k + '/' + this.n : 'start · 0/' + this.n; };
  P.nav = function () {
    var self = this, nav = document.createElement('div');
    nav.className = 'si-dg-nav';
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
      if (a === 'next' && self.k < self.n) { if (self.motion()) self.play(self.k + 1); else self.set(self.k + 1); }
      if (a === 'prev' && self.k > 0) self.set(self.k - 1);
    });
  };
  /* any moment of the film: the entrance, then a step every BEAT */
  P.at = function (t) {
    var S = this.S;
    if (t < ENTER) { this.draw(this.IN ? mix(this.IN, S[0], ease(t / ENTER)) : S[0]); return; }
    var u = t - ENTER, j = Math.min(this.n, Math.floor(u / BEAT)), f = (u - j * BEAT) / TW;
    if (j >= this.n) { this.draw(S[this.n]); return; }
    this.draw(f >= 1 ? S[j + 1] : mix(S[j], S[j + 1], ease(Math.max(0, f))));
  };
  P.length = function () { return ENTER + this.n * BEAT + 0.6; };

  var all = [];
  function start() {
    Array.prototype.forEach.call(document.querySelectorAll('.si-dg[data-dg]'), function (f) { if (!f.siDiagram) { f.siDiagram = new Dg(f); all.push(f.siDiagram); } });
    var mine = {
      duration: Math.max.apply(null, all.map(function (d) { return d.length(); })),
      seek: function (t) { all.forEach(function (d) { d.at(t); }); },
      final: function () { all.forEach(function (d) { d.halt && d.halt(); d.set ? d.set(d.n) : d.draw(d.S[d.n]); }); }
    };
    var prev = window.siIG; /* an infographic on the same page: both follow one clock */
    window.siIG = !prev ? { duration: mine.duration, loop: false, seek: mine.seek, final: mine.final, replay: function () {} } : {
      duration: Math.max(prev.duration || 0, mine.duration), loop: !!prev.loop,
      seek: function (t) { prev.seek(t); mine.seek(t); }, final: function () { prev.final(); mine.final(); }, replay: prev.replay
    };
  }
  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', start); else start();
})();
