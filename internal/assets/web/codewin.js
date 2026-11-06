/* si code windows: every .si-cw on the page (a fence with window= or id=).
   The Go renderer gives each its steps as JSON (highlighted lines, the same
   lines as text, highlighted line numbers, callouts); this draws the lines,
   morphs between steps (lines that stay glide to their new place, new ones
   fade in, removed ones fade out), places the callouts (a dot at the line's
   end, a dashed wire, a note you can drag), and steps through a show: ← →
   and Play under it in a document, build steps and slide changes on a
   slide. #si-code-export writes every show as SVG frames for si code. */
(function () {
  'use strict';
  var reduce = window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  var shown = {}; /* id -> the step the last window with that id showed (a slide carries it to the next) */
  var all = [];

  function esc(s) { return String(s).replace(/[&<>"]/g, function (c) { return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]; }); }

  function Win(fig, index) {
    var self = this;
    this.fig = fig; this.index = index;
    this.data = JSON.parse(fig.querySelector('.si-cw-data').textContent);
    this.steps = this.data.steps;
    this.id = fig.getAttribute('data-cw-id') || '';
    this.stage = fig.querySelector('.si-cw-stage');
    this.win = fig.querySelector('.si-cw-win');
    this.body = fig.querySelector('.si-cw-body');
    this.code = fig.querySelector('.si-cw-code');
    this.cols = 1;
    this.steps.forEach(function (s) { s.text.forEach(function (t) { self.cols = Math.max(self.cols, t.replace(/\t/g, '    ').length); }); });
    this.code.style.width = this.cols + 'ch';
    if (fig.hasAttribute('data-numbers')) { this.gutter = document.createElement('div'); this.gutter.className = 'si-cw-gutter'; this.code.appendChild(this.gutter); }
    this.layer = document.createElement('div'); this.layer.className = 'si-cw-notes-layer';
    this.layer.innerHTML = '<svg class="si-cw-wires" aria-hidden="true"></svg>';
    this.stage.appendChild(this.layer);
    this.list = document.createElement('ol'); this.list.className = 'si-cw-list';
    fig.insertBefore(this.list, fig.querySelector('.si-cw-steps'));
    this.live = {}; this.k = 0;
    fig.classList.add('si-cw-on');
    if (fig.hasAttribute('data-cw-nav') && this.steps.length > 1) this.nav();
    this.show(0, false);
    var marks = fig.querySelectorAll('.si-cw-mark');
    if (marks.length && window.MutationObserver) { /* build steps on a slide: one per extra step */
      var mo = new MutationObserver(function () {
        var k = 0; marks.forEach(function (m) { if (m.classList.contains('is-visible')) k++; });
        if (k !== self.k) self.show(k, true);
      });
      marks.forEach(function (m) { mo.observe(m, { attributes: true, attributeFilter: ['class'] }); });
    }
    var slide = fig.closest('.si-slide');
    if (slide && window.MutationObserver) { /* the same id on the slide before: start from what it showed, then morph */
      var was = slide.classList.contains('si-current');
      new MutationObserver(function () {
        var now = slide.classList.contains('si-current');
        if (now && !was) {
          var prev = self.id && shown[self.id];
          if (prev && prev.win !== self) { self.paint(prev.step, false); requestAnimationFrame(function () { self.show(self.k, true); }); }
          else { self.place(); self.remember(); }
        }
        was = now;
      }).observe(slide, { attributes: true, attributeFilter: ['class'] });
    }
    if (window.ResizeObserver) new ResizeObserver(function () { self.place(); }).observe(this.stage);
  }
  var W = Win.prototype;

  /* what a slide carries to the next: only the window on screen (or one not on a slide) speaks for its id */
  W.remember = function () {
    var slide = this.fig.closest('.si-slide');
    if (this.id && (!slide || slide.classList.contains('si-current'))) shown[this.id] = { win: this, step: this.cur || this.steps[this.k] };
  };
  W.lh = function () { return parseFloat(getComputedStyle(this.body).lineHeight) || 24; };
  W.tx = function () { return this.fig.getAttribute('data-transition') || 'morph'; };

  /* show step k, morphing from what is on screen */
  W.show = function (k, animate) {
    this.k = Math.max(0, Math.min(this.steps.length - 1, k));
    this.paint(this.steps[this.k], animate && !reduce);
    this.remember();
    if (this.out) this.out.textContent = (this.k + 1) + ' / ' + this.steps.length + (this.steps[this.k].name ? ' · ' + this.steps[this.k].name : '');
  };

  W.paint = function (st, animate) {
    var self = this, lh = this.lh(), keep = {}, seen = {}, hl = {}, tx = animate ? this.tx() : 'none';
    (st.hl || []).forEach(function (n) { hl[n] = true; });
    if (tx === 'fade' || tx === 'slide') {
      Object.keys(this.live).forEach(function (key) {
        var el = self.live[key]; el.style.opacity = 0;
        if (tx === 'slide') el.style.transform += ' translateX(-1.5em)';
        setTimeout(function () { el.remove(); }, 360);
      });
      this.live = {};
    }
    st.text.forEach(function (text, i) {
      var key = text + '\u0001' + (seen[text] = (seen[text] || 0) + 1), el = self.live[key];
      var y = 'translateY(' + i * lh + 'px)';
      if (el) { el.style.transitionDelay = '0s'; el.style.transform = y; el.style.opacity = 1; }
      else {
        el = document.createElement('div'); el.className = 'si-cw-l'; el.innerHTML = st.lines[i] || ' ';
        el.style.transform = y + (tx === 'slide' ? ' translateX(1.5em)' : '');
        el.style.opacity = tx === 'none' ? 1 : 0;
        if (tx === 'none') el.style.transition = 'none';
        self.code.appendChild(el);
        if (tx !== 'none') (function (e, d) {
          requestAnimationFrame(function () { requestAnimationFrame(function () { e.style.transitionDelay = d; e.style.opacity = 1; e.style.transform = y; }); });
        })(el, tx === 'morph' ? '.32s' : '.18s');
        else requestAnimationFrame(function () { el.style.transition = ''; });
      }
      el.classList.toggle('si-cw-hl', !!hl[i + 1]);
      el.style.height = lh + 'px';
      keep[key] = el;
    });
    Object.keys(this.live).forEach(function (key) {
      if (keep[key]) return;
      var el = self.live[key]; el.style.transitionDelay = '0s'; el.style.opacity = 0;
      setTimeout(function () { el.remove(); }, tx === 'none' ? 0 : 360);
    });
    this.live = keep;
    this.code.style.height = st.text.length * lh + 'px';
    if (this.gutter) this.gutter.innerHTML = st.text.map(function (_, i) { return '<div style="height:' + lh + 'px">' + (i + 1) + '</div>'; }).join('');
    this.cur = st;
    this.notes(animate ? 620 : 0);
  };

  /* ---------- callouts ---------- */
  W.notes = function (delay) {
    var self = this;
    this.layer.querySelectorAll('.si-cw-bubble,.si-cw-dot').forEach(function (e) { e.style.opacity = 0; setTimeout(function () { e.remove(); }, delay ? 250 : 0); });
    this.layer.querySelector('svg').innerHTML = '';
    this.list.innerHTML = (this.cur.notes || []).map(function (n) { return '<li>' + n.html + '</li>'; }).join('');
    clearTimeout(this.noteT);
    if (delay) this.noteT = setTimeout(function () { self.place(); }, delay); else this.place();
  };
  W.anchor = function (n) {
    var sr = this.stage.getBoundingClientRect(), cr = this.code.getBoundingClientRect(), lh = this.lh();
    var ch = cr.width / this.cols, t = (this.cur.text[n.line - 1] || '').replace(/\t/g, '    ').replace(/\s+$/, '');
    return { x: Math.min(cr.left - sr.left + (t.length + 1.2) * ch, this.body.getBoundingClientRect().right - sr.left - 6), y: cr.top - sr.top + (n.line - 0.5) * lh };
  };
  W.place = function () {
    var self = this, notes = this.cur && this.cur.notes || [], wires = this.layer.querySelector('svg');
    if (!this.stage.offsetWidth) return;
    this.layer.querySelectorAll('.si-cw-bubble,.si-cw-dot').forEach(function (e) { e.remove(); });
    wires.innerHTML = '';
    this.fig.classList.remove('si-cw-listmode'); /* measure the notes as they would show */
    var sr = this.stage.getBoundingClientRect(), wr = this.win.getBoundingClientRect().right - sr.left;
    var listMode = sr.width < 34 * parseFloat(getComputedStyle(this.fig).fontSize); /* a phone: the notes go under the window */
    notes.forEach(function (n, k) {
      var a = self.anchor(n);
      var dot = document.createElement('div'); dot.className = 'si-cw-dot'; dot.textContent = k + 1;
      dot.style.left = a.x + 'px'; dot.style.top = a.y + 'px'; self.layer.appendChild(dot);
      var b = document.createElement('div'); b.className = 'si-cw-bubble'; b.setAttribute('data-n', k + 1); b.innerHTML = n.html; b.title = 'Drag to place';
      self.layer.appendChild(b);
      var path = document.createElementNS('http://www.w3.org/2000/svg', 'path'); wires.appendChild(path);
      b.style.left = '0px'; var bw = b.offsetWidth; b.style.width = bw + 'px'; var bh = b.offsetHeight;
      if (!n.at && wr + 24 + bw > sr.width) listMode = true;
      function put() {
        var at = n.at || [wr + 24 - a.x, (k % 2 ? 1 : -1) * Math.max(bh, 30)];
        var x = Math.max(4, Math.min(sr.width - bw - 4, a.x + at[0])), y = Math.max(4, Math.min(sr.height - bh - 4, a.y + at[1] - bh / 2));
        b.style.left = x + 'px'; b.style.top = y + 'px';
        var ex = x < a.x ? x + bw : x, ey = y + bh / 2;
        path.setAttribute('d', 'M' + (a.x + 9) + ' ' + a.y + ' C' + ((a.x + ex) / 2) + ' ' + a.y + ' ' + ((a.x + ex) / 2) + ' ' + ey + ' ' + ex + ' ' + ey);
      }
      put();
      var drag = null;
      b.addEventListener('pointerdown', function (e) { e.stopPropagation(); drag = { x: e.clientX, y: e.clientY, at: (n.at || [wr + 24 - a.x, (k % 2 ? 1 : -1) * Math.max(bh, 30)]).slice() }; b.setPointerCapture(e.pointerId); });
      b.addEventListener('pointermove', function (e) { if (!drag) return; n.at = [Math.round(drag.at[0] + e.clientX - drag.x), Math.round(drag.at[1] + e.clientY - drag.y)]; put(); });
      b.addEventListener('pointerup', function () {
        if (!drag) return; drag = null;
        /* the Studio's Window view writes the place back as at= */
        if (window.parent !== window) window.parent.postMessage({ type: 'si-cw-callout', show: self.index, step: self.k, index: k, at: n.at }, '*');
      });
      b.addEventListener('click', function (e) { e.stopPropagation(); });
    });
    this.fig.classList.toggle('si-cw-listmode', listMode);
  };

  /* ---------- ← → and Play in a document ---------- */
  W.nav = function () {
    var self = this, nav = document.createElement('div');
    nav.className = 'si-cw-nav';
    nav.innerHTML = '<button type="button" data-a="prev" aria-label="Previous step">←</button><output aria-live="polite"></output><button type="button" data-a="next" aria-label="Next step">→</button><button type="button" data-a="play" aria-pressed="false">Play</button>';
    this.fig.appendChild(nav);
    this.out = nav.querySelector('output');
    var play = nav.querySelector('[data-a="play"]');
    nav.addEventListener('click', function (e) {
      var b = e.target.closest('button'); if (!b) return;
      var a = b.getAttribute('data-a');
      if (a === 'play') {
        if (self.timer) { clearInterval(self.timer); self.timer = null; play.setAttribute('aria-pressed', 'false'); play.textContent = 'Play'; return; }
        play.setAttribute('aria-pressed', 'true'); play.textContent = 'Pause';
        self.timer = setInterval(function () { self.show((self.k + 1) % self.steps.length, true); }, 2600);
        return;
      }
      if (self.timer) play.click();
      self.show(self.k + (a === 'next' ? 1 : -1), true);
    });
  };

  /* ---------- #si-code-export: every show as SVG frames for si code ---------- */
  var TOKENS = ['--si-cw-bg', '--si-cw-bar', '--si-cw-rule', '--si-cw-fg', '--si-cw-muted', '--si-cw-k', '--si-cw-s', '--si-cw-c', '--si-cw-n', '--si-cw-f', '--si-cw-t', '--si-cw-hl',
    '--si-cw-backdrop', '--si-cw-pad', '--si-cw-radius', '--si-cw-shadow', '--si-cw-pin', '--si-cw-pin-fg', '--si-cw-note', '--si-cw-note-fg', '--si-cw-wire',
    '--si-font-mono', '--si-font-body', '--si-cw-size'];
  function exportCSS() {
    var css = '', own = document.getElementById('si-cw-css');
    Array.prototype.forEach.call(document.styleSheets, function (sh) {
      try { Array.prototype.forEach.call(sh.cssRules, function (r) { if (r.type === 5 && /url\(\s*["']?data:/.test(r.cssText)) css += r.cssText + '\n'; }); } catch (e) {}
    });
    return css + (own ? own.textContent : '');
  }
  W.frame = function (from, to, t) { /* the window between two steps, t in 0..1, without transitions */
    var self = this, lh = this.lh(), tx = this.tx(), fy = {}, seen = {}, out = [];
    from.text.forEach(function (text, i) { fy[text + '\u0001' + (seen[text] = (seen[text] || 0) + 1)] = i; });
    seen = {};
    var e = t < 0.5 ? 4 * t * t * t : 1 - Math.pow(-2 * t + 2, 3) / 2, hl = {};
    (to.hl || []).forEach(function (n) { hl[n] = true; });
    var used = {};
    to.text.forEach(function (text, i) {
      var key = text + '\u0001' + (seen[text] = (seen[text] || 0) + 1), j = fy[key], y, op = 1;
      if (tx === 'morph' && j !== undefined) { y = (j + (i - j) * e) * lh; used[key] = true; }
      else { y = i * lh; op = tx === 'morph' ? Math.max(0, (t - 0.45) / 0.55) : Math.max(0, (t - 0.5) * 2); }
      out.push({ html: to.lines[i], y: y, op: op, hl: hl[i + 1] });
    });
    seen = {};
    from.text.forEach(function (text, i) {
      var key = text + '\u0001' + (seen[text] = (seen[text] || 0) + 1);
      if (used[key] || t >= 0.5 && tx !== 'morph') return;
      out.push({ html: from.lines[i], y: i * lh, op: Math.max(0, 1 - t * (tx === 'morph' ? 2.5 : 2)), hl: false });
    });
    this.code.innerHTML = '';
    if (this.gutter) this.code.appendChild(this.gutter);
    out.forEach(function (l) {
      var el = document.createElement('div'); el.className = 'si-cw-l' + (l.hl ? ' si-cw-hl' : ''); el.innerHTML = l.html || ' ';
      el.style.transition = 'none'; el.style.transform = 'translateY(' + l.y + 'px)'; el.style.opacity = l.op; el.style.height = lh + 'px';
      self.code.appendChild(el);
    });
    var rows = Math.max(from.text.length + (to.text.length - from.text.length) * e, 1);
    this.code.style.height = rows * lh + 'px';
    var cur = t < 1 ? from : to;
    if (this.gutter) this.gutter.innerHTML = to.text.map(function (_, i) { return '<div style="height:' + lh + 'px">' + (i + 1) + '</div>'; }).join('');
    this.cur = t >= 1 ? to : { text: cur.text, notes: [] };
    this.live = {};
    this.place();
  };
  W.svg = function (css) {
    var fig = this.fig, cs = getComputedStyle(fig), r = this.stage.getBoundingClientRect(), vars = '';
    TOKENS.forEach(function (k) { var v = cs.getPropertyValue(k); if (v) vars += k + ':' + v.trim() + ';'; });
    var clone = this.stage.cloneNode(true);
    var attrs = ['data-window', 'data-cw-theme', 'data-backdrop', 'data-numbers'].map(function (a) { return fig.hasAttribute(a) ? ' ' + a + '="' + esc(fig.getAttribute(a)) + '"' : ''; }).join('');
    var w = Math.ceil(r.width), h = Math.ceil(r.height);
    var xhtml = new XMLSerializer().serializeToString(clone);
    return '<svg xmlns="http://www.w3.org/2000/svg" width="' + w + '" height="' + h + '" viewBox="0 0 ' + w + ' ' + h + '"><style>' + css.replace(/</g, '\\3c ') + '</style>' +
      '<foreignObject x="0" y="0" width="' + w + '" height="' + h + '"><div xmlns="http://www.w3.org/1999/xhtml" class="si-cw si-cw-on"' + attrs + ' style="margin:0;' + esc(vars) + 'font-size:' + cs.fontSize + ';width:' + w + 'px">' +
      xhtml + '</div></foreignObject></svg>';
  };
  function exportAll() {
    var css = exportCSS(), out = [];
    all.forEach(function (w, n) {
      var frames = [], k = 6, mw = 0, mh = 0;
      /* one size for every frame: the widest and tallest step, the window centred on the backdrop */
      w.fig.style.width = '1700px'; w.fig.style.maxWidth = 'none'; /* room to measure: the window takes its natural width */
      var pad = parseFloat(getComputedStyle(w.stage).paddingLeft) || 0;
      w.steps.forEach(function (st) { w.frame(st, st, 1); mw = Math.max(mw, w.win.getBoundingClientRect().width + 2 * pad); mh = Math.max(mh, w.stage.getBoundingClientRect().height); });
      if (w.steps.some(function (st) { return (st.notes || []).length; })) mw += 2 * 18 * parseFloat(getComputedStyle(w.fig).fontSize); /* room for the notes on the right */
      w.stage.style.width = Math.ceil(mw) + 'px'; w.stage.style.minHeight = Math.ceil(mh) + 'px';
      w.steps.forEach(function (st, i) {
        w.frame(st, st, 1); frames.push({ step: i + 1, hold: true, svg: w.svg(css), w: Math.ceil(w.stage.getBoundingClientRect().width), h: Math.ceil(w.stage.getBoundingClientRect().height) });
        if (i + 1 < w.steps.length) for (var f = 1; f < k; f++) {
          w.frame(st, w.steps[i + 1], f / k);
          frames.push({ step: i + 1, hold: false, svg: w.svg(css), w: Math.ceil(w.stage.getBoundingClientRect().width), h: Math.ceil(w.stage.getBoundingClientRect().height) });
        }
      });
      out.push({ id: w.id || 'window-' + (n + 1), title: w.fig.getAttribute('aria-label') || '', steps: w.steps.length, frames: frames });
    });
    var pre = document.createElement('pre'); pre.id = 'si-code-export'; pre.hidden = true; pre.textContent = JSON.stringify(out);
    document.body.appendChild(pre);
  }

  function start() {
    document.querySelectorAll('.si-cw').forEach(function (fig, i) { if (!fig.siCw) { fig.siCw = new Win(fig, i); all.push(fig.siCw); } });
    if (location.hash === '#si-code-export') {
      var go = function () { try { exportAll(); } catch (e) { var p = document.createElement('pre'); p.id = 'si-code-export-error'; p.textContent = String(e && e.stack || e); document.body.appendChild(p); } };
      if (document.fonts && document.fonts.ready) document.fonts.ready.then(function () { setTimeout(go, 50); }); else setTimeout(go, 50);
    }
  }
  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', start); else start();
})();
