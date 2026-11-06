/* si mind maps: every .si-mindmap on the page (a type: mindmap page, or a
   ```mindmap fence in a slide or a document). The Go renderer gives each one
   its maps as JSON; this lays them out (map, right, org, radial), draws the
   lines (curved, straight, orthogonal, tapered) and the topics (box, pill,
   underline; pictures, badges, links, sticky notes, code), and handles
   drill-down between maps, folding, moving topics, pan and zoom, and on a
   full page `/` search and Play. Without JavaScript the maps stay the nested
   lists they were written as. */
(function () {
  'use strict';
  var NS = 'http://www.w3.org/2000/svg', seq = 0;
  var reduce = window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  var LV = [{ fs: 22, fw: 700, px: 22, py: 14 }, { fs: 16, fw: 600, px: 15, py: 10 }, { fs: 14, fw: 450, px: 11, py: 7 }];
  var HG = 58, VG = 56, GAP = 14;
  var meas = document.createElement('canvas').getContext('2d');

  function esc(s) { return String(s).replace(/[&<>"]/g, function (c) { return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]; }); }
  function f1(v) { return Math.round(v * 10) / 10; }
  function P(p) { return f1(p.x) + ' ' + f1(p.y); }
  function unit(x, y) { var l = Math.hypot(x, y); return l ? { x: x / l, y: y / l } : null; }
  function lv(d) { return LV[Math.min(d, 2)]; }
  function vis(n) { return n.collapsed ? [] : (n.kids || []); }

  function Mindmap(el) {
    var self = this;
    this.el = el; this.uid = 'mm' + (++seq);
    var data = JSON.parse(el.querySelector('.si-mm-data').textContent);
    this.maps = {};
    data.maps.forEach(function (m) {
      (function fix(list) { (list || []).forEach(function (n) {
        if (n.pos) n.pos = { x: n.pos[0], y: n.pos[1] };
        if (n.bpos) n.bpos = { x: n.bpos[0], y: n.bpos[1] };
        fix(n.kids);
      }); })(m.kids);
      m.root = { t: m.title, kids: m.kids };
      self.maps[m.id] = m;
    });
    this.full = el.hasAttribute('data-full');
    this.st = { map: data.start, stack: [] };
    var h = this.full && location.hash.slice(1);
    if (h && this.maps[decodeURIComponent(h)]) this.st.map = decodeURIComponent(h);
    this.vb = { x: 0, y: 0, w: 100, h: 100 }; this.busy = false; this.tweenId = 0; this.tour = null;
    el.classList.add('si-mm-on');
    this.build();
    this.show(this.st.map);
    if (window.SIMindmap3D && this.opt('view') === '3d') this.set3d(true);
    if (document.fonts && document.fonts.ready) document.fonts.ready.then(function () { self.paint(false); self.setVB(self.fitBox(self.cur.all)); });
    if (window.ResizeObserver) {
      var lastLay = this.lay();
      new ResizeObserver(function () {
        var l = self.lay();
        if (l !== lastLay) { lastLay = l; self.paint(false); self.setVB(self.fitBox(self.cur.all)); }
      }).observe(el);
    }
    var slide = el.closest('.si-slide');
    if (slide && window.MutationObserver) { /* grow in again whenever the slide comes on screen */
      var was = slide.classList.contains('si-current');
      new MutationObserver(function () {
        var now = slide.classList.contains('si-current');
        if (now && !was) { self.paint(true); self.setVB(self.fitBox(self.cur.all)); }
        was = now;
      }).observe(slide, { attributes: true, attributeFilter: ['class'] });
    }
  }

  var M = Mindmap.prototype;
  M.opt = function (k) { return this.el.getAttribute('data-' + k) || ''; };
  M.css = function (k) { return getComputedStyle(this.el).getPropertyValue(k).trim(); };
  /* the layout in use: a narrow portrait screen (a phone) gets the tall right layout */
  M.lay = function () {
    var l = this.opt('layout'), w = this.el.clientWidth, h = this.el.clientHeight;
    return w && w < 640 && h > w ? 'right' : l;
  };

  /* ---------- chrome: crumbs, tools, search, Play ---------- */
  M.build = function () {
    var self = this, el = this.el;
    var svg = this.svg = document.createElementNS(NS, 'svg');
    svg.setAttribute('class', 'si-mm-svg');
    svg.setAttribute('role', 'group');
    el.appendChild(svg);
    this.crumbs = div('si-mm-crumbs'); this.crumbs.setAttribute('role', 'navigation'); this.crumbs.setAttribute('aria-label', 'Maps');
    el.appendChild(this.crumbs);
    var tools = div('si-mm-tools');
    var has3d = !!window.SIMindmap3D, persp = this.opt('camera') === 'perspective';
    tools.innerHTML = '<button type="button" data-a="find" title="Find' + (this.full ? ' (/)' : '') + '">Find</button>' +
      (this.full ? '<button type="button" data-a="play" aria-pressed="false" title="Play: one branch at a time">Play</button>' : '') +
      (has3d ? '<button type="button" data-a="view" aria-pressed="false" title="Show the map in 3D or 2D">3D</button><button type="button" data-a="cam" title="Switch the 3D camera">' + (persp ? 'Isometric' : 'Perspective') + '</button>' : '') +
      '<button type="button" data-a="in" aria-label="Zoom in">+</button><button type="button" data-a="out" aria-label="Zoom out">−</button><button type="button" data-a="fit">Fit</button><button type="button" data-a="tidy" title="Put moved topics back">Tidy</button>';
    el.appendChild(tools);
    this.playBtn = tools.querySelector('[data-a="play"]');
    this.viewBtn = tools.querySelector('[data-a="view"]'); this.camBtn = tools.querySelector('[data-a="cam"]');
    tools.addEventListener('click', function (e) {
      var b = e.target.closest('button'); if (!b) return;
      var a = b.getAttribute('data-a'), w = svg.clientWidth, hh = svg.clientHeight, d3 = self.is3d && self.d3;
      if (a === 'view') self.set3d(!self.is3d);
      else if (a === 'cam' && self.d3) { self.d3.setCam(self.d3.cam === 'iso' ? 'persp' : 'iso'); b.textContent = self.d3.cam === 'iso' ? 'Perspective' : 'Isometric'; }
      else if (d3 && (a === 'in' || a === 'out')) d3.zoom(a === 'in' ? 0.8 : 1.25);
      else if (d3 && a === 'fit') d3.fit();
      else if (d3 && a === 'play') { if (d3.tour) d3.tourStop(); else d3.tourStart(); }
      else if (a === 'in') self.zoomAt(0.8, w / 2, hh / 2);
      else if (a === 'out') self.zoomAt(1.25, w / 2, hh / 2);
      else if (a === 'fit') self.tween(self.fitBox(self.cur.all), 420);
      else if (a === 'tidy') self.tidy();
      else if (a === 'find') self.openFind();
      else if (a === 'play') { if (self.tour) self.tourStop(); else self.tourStart(); }
    });
    this.find = div('si-mm-find'); this.find.hidden = true;
    {
      this.find.innerHTML = '<input type="search" placeholder="Find a topic, a note or code in every map" aria-label="Find" autocomplete="off" spellcheck="false"><ol role="listbox"></ol>';
      el.appendChild(this.find);
      this.findQ = this.find.querySelector('input'); this.findList = this.find.querySelector('ol');
      this.findQ.addEventListener('input', function () { self.listHits(); });
      this.findQ.addEventListener('keydown', function (e) {
        if (e.key === 'ArrowDown' || e.key === 'ArrowUp') { e.preventDefault(); self.sel = (self.sel + (e.key === 'ArrowDown' ? 1 : self.hits.length - 1)) % Math.max(1, self.hits.length); self.drawHits(); }
        else if (e.key === 'Enter') { e.preventDefault(); self.goHit(self.hits[self.sel]); }
        else if (e.key === 'Escape') { e.preventDefault(); self.closeFind(); }
      });
      this.findList.addEventListener('click', function (e) { var li = e.target.closest('[data-k]'); if (li) self.goHit(self.hits[+li.getAttribute('data-k')]); });
    }
    if (this.full) {
      this.tourEl = div('si-mm-tour'); this.tourEl.hidden = true; this.tourEl.setAttribute('role', 'status');
      el.appendChild(this.tourEl);
      document.addEventListener('keydown', function (e) {
        var typing = /input|textarea|select/i.test(e.target.tagName) || e.target.isContentEditable;
        if (e.key === '/' && !typing && !e.metaKey && !e.ctrlKey) { e.preventDefault(); self.openFind(); return; }
        var t = self.active();
        if (!t.tour || typing) { if (e.key === 'Escape' && self.d3) self.d3.closeCard(); return; }
        if (e.key === 'ArrowRight' || e.key === ' ' || e.key === 'PageDown') { e.preventDefault(); clearInterval(t.tour.timer); t.tourGo(t.tour.k + 1); }
        else if (e.key === 'ArrowLeft' || e.key === 'PageUp') { e.preventDefault(); clearInterval(t.tour.timer); t.tourGo(t.tour.k - 1); }
        else if (e.key === 'Escape') t.tourStop();
      });
    }
    this.crumbs.addEventListener('click', function (e) { var b = e.target.closest('[data-back]'); if (b) self.active().back(+b.getAttribute('data-back')); });
    this.pointer();
  };
  function div(c) { var d = document.createElement('div'); d.className = c; return d; }

  /* ---------- 2D or 3D (mindmap3d.js, when the page carries three.js) ---------- */
  M.active = function () { return this.is3d ? this.d3 : this; };
  M.set3d = function (on) {
    var t = this.active(); if (t.tour) t.tourStop(true);
    if (on && !this.d3) {
      try { this.d3 = new window.SIMindmap3D(this); } catch (e) { this.d3 = null; }
      if (!this.d3 || !this.d3.ok) { this.d3 = null; this.msg('3D needs WebGL, which this browser does not offer: the map is shown in 2D.'); return; }
    }
    this.is3d = !!(on && this.d3);
    this.el.classList.toggle('si-mm-is3d', this.is3d);
    if (this.viewBtn) { this.viewBtn.textContent = this.is3d ? '2D' : '3D'; this.viewBtn.setAttribute('aria-pressed', String(this.is3d)); }
    if (this.is3d) this.d3.enter();
    else { if (this.d3) this.d3.leave(); this.paint(false); this.setVB(this.fitBox(this.cur.all)); }
  };
  M.msg = function (text) {
    var m = this.note || (this.note = this.el.appendChild(div('si-mm-msg')));
    m.textContent = text; m.hidden = false;
    clearTimeout(this.noteT); this.noteT = setTimeout(function () { m.hidden = true; }, 6000);
  };

  /* ---------- sizes ---------- */
  M.size = function (n, d, f) {
    if (n.code) {
      meas.font = '12.5px ' + this.css('--si-mm-mono');
      var cw = meas.measureText('M').width;
      n.w = Math.max(160, n.code.cols * cw + 34); n.h = n.code.lines * 18 + 38; n.chips = []; n.d = d; n.tw = 0; return;
    }
    if (n.sticky) {
      meas.font = '600 17px ' + this.css('--si-mm-hand');
      var line = '', out = [];
      n.t.split(' ').forEach(function (w) { var t = line ? line + ' ' + w : w; if (meas.measureText(t).width > 170 && line) { out.push(line); line = w; } else line = t; });
      if (line) out.push(line);
      n.lines = out; n.w = 200; n.h = out.length * 21 + 26; n.chips = []; n.d = d; n.tw = 0; return;
    }
    var L = lv(d);
    meas.font = L.fw + ' ' + L.fs + 'px ' + f;
    n.tw = meas.measureText(n.t).width;
    var w = n.tw + L.px * 2, h = L.fs * 1.25 + L.py * 2;
    n.chips = [];
    if (n.badge) n.chips.push({ k: 'badge', t: n.badge });
    if (n.href) n.chips.push({ k: 'href', t: n.chip || 'link' });
    if (n.link) n.chips.push({ k: 'link', t: 'map ›' });
    if (n.collapsed && n.kids) n.chips.push({ k: 'more', t: '+' + n.kids.length });
    meas.font = '600 11px ' + f;
    n.chips.forEach(function (c) { c.w = meas.measureText(c.t).width + 14; w += c.w + 6; });
    if (n.img) { w += 62; h = Math.max(h, 48); }
    n.w = w; n.h = h; n.d = d;
  };

  /* ---------- layouts: every topic gets a centre (x, y) ---------- */
  function spanH(n) { var k = vis(n); if (!k.length) return (n.sh = n.h); var s = 0; k.forEach(function (c) { s += spanH(c); }); s += GAP * (k.length - 1); return (n.sh = Math.max(n.h, s)); }
  function kidsSpan(k) { var s = 0; k.forEach(function (c) { s += c.sh; }); return s + GAP * Math.max(0, k.length - 1); }
  function placeH(n, x, top, side) {
    n.x = x; n.y = top + n.sh / 2; n.side = side;
    var k = vis(n), y = top + (n.sh - kidsSpan(k)) / 2;
    k.forEach(function (c) { placeH(c, x + side * (n.w / 2 + HG + c.w / 2), y, side); y += c.sh + GAP; });
  }
  function stacked(n) { var k = vis(n); return n.d >= 1 && k.length && k.every(function (c) { return !vis(c).length; }); }
  function spanW(n) {
    var k = vis(n);
    if (!k.length) return (n.sw = n.w);
    if (stacked(n)) { var m = 0; k.forEach(function (c) { m = Math.max(m, c.w); }); return (n.sw = Math.max(n.w, 22 + m)); }
    var s = 0; k.forEach(function (c) { s += spanW(c); }); s += 22 * (k.length - 1);
    return (n.sw = Math.max(n.w, s));
  }
  function placeV(n, y, left) {
    n.x = left + n.sw / 2; n.y = y;
    var k = vis(n);
    if (!k.length) return;
    if (stacked(n)) {
      var yy = y + n.h / 2 + 14;
      if (n.sw > n.w) n.x = left + n.w / 2;
      k.forEach(function (c) { c.stack = true; c.x = n.x - n.w / 2 + 22 + c.w / 2; c.y = yy + c.h / 2; yy += c.h + 10; });
      return;
    }
    var s = 0; k.forEach(function (c) { s += c.sw; }); s += 22 * (k.length - 1);
    var x = left + (n.sw - s) / 2;
    k.forEach(function (c) { placeV(c, y + n.h / 2 + VG + c.h / 2, x); x += c.sw + 22; });
  }
  function leaves(n) { var k = vis(n); return (n.lc = k.length ? k.reduce(function (a, c) { return a + leaves(c); }, 0) : 1); }
  function placeR(n, a0, a1, d, R) {
    var k = vis(n), tot = 0, a = a0;
    k.forEach(function (c) { tot += c.lc; });
    k.forEach(function (c) {
      var span = (a1 - a0) * c.lc / tot, mid = a + span / 2, r = R(d + 1);
      c.x = r * Math.cos(mid); c.y = r * Math.sin(mid);
      placeR(c, a, a + span, d + 1, R); a += span;
    });
  }
  M.layout = function (root) {
    var all = [], lay = this.lay();
    (function walk(n) { all.push(n); vis(n).forEach(walk); })(root);
    all.forEach(function (n) { n.stack = false; n.side = 1; });
    if (lay === 'right') { spanH(root); placeH(root, 0, -root.sh / 2, 1); }
    else if (lay === 'org') { spanW(root); placeV(root, 0, -root.sw / 2); }
    else if (lay === 'radial') {
      leaves(root); root.x = 0; root.y = 0;
      var maxW = [0, 0, 0, 0, 0];
      all.forEach(function (n) { var i = Math.min(n.d, 4); maxW[i] = Math.max(maxW[i], n.w); });
      var R = function (d) { if (!d) return 0; var r = root.w / 2 + 70 + maxW[1] / 2; for (var i = 2; i <= d; i++) r += maxW[i - 1] / 2 + 80 + maxW[Math.min(i, 4)] / 2; return r; };
      placeR(root, -Math.PI / 2 - 0.35, Math.PI * 1.5 - 0.35, 0, R);
    } else {
      root.x = 0; root.y = 0;
      var k = vis(root); k.forEach(spanH);
      var total = kidsSpan(k), acc = 0, Rt = [], Lf = [];
      k.forEach(function (c) { (acc + c.sh / 2 < total / 2 ? Rt : Lf).push(c); acc += c.sh + GAP; });
      if (!Lf.length && Rt.length > 1) Lf.push(Rt.pop());
      [[Rt, 1], [Lf, -1]].forEach(function (g) {
        var y = -kidsSpan(g[0]) / 2;
        g[0].forEach(function (c) { placeH(c, g[1] * (root.w / 2 + HG + c.w / 2), y, g[1]); y += c.sh + GAP; });
      });
    }
    /* moved by hand: a branch offset carries its children, a topic offset moves only that topic */
    (function shift(n, ox, oy) {
      if (n.bpos) { ox += n.bpos.x; oy += n.bpos.y; }
      n.x += ox; n.y += oy;
      vis(n).forEach(function (c) { shift(c, ox, oy); });
      if (n.pos) { n.x += n.pos.x; n.y += n.pos.y; }
    })(root, 0, 0);
    return all;
  };

  /* ---------- lines ---------- */
  function rectEdge(n, tx, ty) {
    var dx = tx - n.x, dy = ty - n.y;
    var s = Math.min(dx ? (n.w / 2) / Math.abs(dx) : Infinity, dy ? (n.h / 2) / Math.abs(dy) : Infinity);
    if (!isFinite(s)) s = 0;
    return { x: n.x + dx * s, y: n.y + dy * s };
  }
  M.ul = function (n) { return this.opt('nodes') === 'underline' && n.d >= 2 && !n.code && !n.sticky; };
  M.ends = function (p, c) {
    var lay = this.lay();
    if (c.stack) return { dir: 'stack', a: { x: p.x - p.w / 2 + 11, y: p.y + p.h / 2 }, b: { x: c.x - c.w / 2, y: this.ul(c) ? c.y + c.h / 2 : c.y } };
    if (lay === 'org') return { dir: 'v', a: { x: p.x, y: p.y + p.h / 2 }, b: { x: c.x, y: c.y - c.h / 2 } };
    if (lay === 'radial') return { dir: 'r', a: rectEdge(p, c.x, c.y), b: rectEdge(c, p.x, p.y) };
    var s = Math.sign(c.x - p.x) || c.side;
    return { dir: 'h', a: { x: p.x + s * p.w / 2, y: this.ul(p) ? p.y + p.h / 2 : p.y }, b: { x: c.x - s * c.w / 2, y: this.ul(c) ? c.y + c.h / 2 : c.y } };
  };
  function ctrl(e, p, c) {
    var a = e.a, b = e.b;
    if (e.dir === 'h') { var m = (b.x - a.x) / 2; return [{ x: a.x + m, y: a.y }, { x: b.x - m, y: b.y }]; }
    if (e.dir === 'v') { var mv = (b.y - a.y) / 2; return [{ x: a.x, y: a.y + mv }, { x: b.x, y: b.y - mv }]; }
    if (e.dir === 'stack') return [{ x: a.x, y: a.y + (b.y - a.y) * 0.8 }, { x: a.x + (b.x - a.x) * 0.2, y: b.y }];
    var L = Math.hypot(b.x - a.x, b.y - a.y) * 0.42;
    var u1 = unit(a.x - p.x, a.y - p.y) || unit(b.x - a.x, b.y - a.y) || { x: 1, y: 0 }, u2 = unit(c.x - p.x, c.y - p.y) || { x: 1, y: 0 };
    return [{ x: a.x + u1.x * L, y: a.y + u1.y * L }, { x: b.x - u2.x * L, y: b.y - u2.y * L }];
  }
  function orth(e) {
    var a = e.a, b = e.b, dir = e.dir;
    if (dir === 'stack') { var r0 = Math.min(10, Math.abs(b.y - a.y)); return 'M' + P(a) + 'V' + f1(b.y - r0) + 'Q' + f1(a.x) + ' ' + f1(b.y) + ' ' + f1(a.x + r0) + ' ' + f1(b.y) + 'H' + f1(b.x); }
    if (dir === 'r') dir = Math.abs(b.x - a.x) > Math.abs(b.y - a.y) ? 'h' : 'v';
    if (dir === 'h') {
      if (Math.abs(b.y - a.y) < 1) return 'M' + P(a) + 'H' + f1(b.x);
      var mx = a.x + (b.x - a.x) / 2, s = Math.sign(b.x - a.x), sy = Math.sign(b.y - a.y), r = Math.min(12, Math.abs(b.y - a.y) / 2, Math.abs(mx - a.x));
      return 'M' + P(a) + 'H' + f1(mx - s * r) + 'Q' + f1(mx) + ' ' + f1(a.y) + ' ' + f1(mx) + ' ' + f1(a.y + sy * r) + 'V' + f1(b.y - sy * r) + 'Q' + f1(mx) + ' ' + f1(b.y) + ' ' + f1(mx + s * r) + ' ' + f1(b.y) + 'H' + f1(b.x);
    }
    if (Math.abs(b.x - a.x) < 1) return 'M' + P(a) + 'V' + f1(b.y);
    var my = a.y + (b.y - a.y) / 2, t = Math.sign(b.y - a.y), sx = Math.sign(b.x - a.x), q = Math.min(12, Math.abs(b.x - a.x) / 2, Math.abs(my - a.y));
    return 'M' + P(a) + 'V' + f1(my - t * q) + 'Q' + f1(a.x) + ' ' + f1(my) + ' ' + f1(a.x + sx * q) + ' ' + f1(my) + 'H' + f1(b.x - sx * q) + 'Q' + f1(b.x) + ' ' + f1(my) + ' ' + f1(b.x) + ' ' + f1(my + t * q) + 'V' + f1(b.y);
  }
  function bez(a, c1, c2, b, t) {
    var u = 1 - t;
    return { x: u * u * u * a.x + 3 * u * u * t * c1.x + 3 * u * t * t * c2.x + t * t * t * b.x, y: u * u * u * a.y + 3 * u * u * t * c1.y + 3 * u * t * t * c2.y + t * t * t * b.y };
  }
  function taper(a, c1, c2, b, w0, w1) {
    var N = 28, L = [], R = [];
    for (var i = 0; i <= N; i++) {
      var t = i / N, p = bez(a, c1, c2, b, t), q = bez(a, c1, c2, b, Math.min(1, t + 0.01)), o = bez(a, c1, c2, b, Math.max(0, t - 0.01));
      var n = unit(-(q.y - o.y), q.x - o.x) || { x: 0, y: 1 }, w = (w0 + (w1 - w0) * t) / 2;
      L.push({ x: p.x + n.x * w, y: p.y + n.y * w }); R.push({ x: p.x - n.x * w, y: p.y - n.y * w });
    }
    return 'M' + L.map(P).join('L') + 'L' + R.reverse().map(P).join('L') + 'Z';
  }
  M.edgeSVG = function (p, c) {
    var e = this.ends(p, c), cc = ctrl(e, p, c), lines = this.opt('lines'), center;
    var curve = 'M' + P(e.a) + 'C' + P(cc[0]) + ' ' + P(cc[1]) + ' ' + P(e.b);
    if (lines === 'straight') center = 'M' + P(e.a) + 'L' + P(e.b);
    else if (lines === 'orthogonal') center = orth(e);
    else center = curve;
    var at = 'style="--si-mm-c:var(--si-mm-' + (c.branch + 1) + ');--i:' + c.i + '" data-top="' + c.top + '"';
    if (c.sticky) return '<path class="si-mm-edge si-mm-note" ' + at + ' d="' + (lines === 'tapered' ? curve : center) + '"/>';
    var body = lines === 'tapered'
      ? '<path class="si-mm-edge si-mm-tap" ' + at + ' d="' + taper(e.a, cc[0], cc[1], e.b, p.d === 0 ? 10 : 6, 1.6) + '"/>'
      : '<path class="si-mm-edge" pathLength="1" ' + at + ' stroke-width="' + (p.d === 0 ? 3.2 : 2) + '" d="' + center + '"/>';
    return body + '<path class="si-mm-flow" d="' + center + '"/>';
  };

  /* ---------- topics ---------- */
  M.nodeSVG = function (n) {
    var L = lv(n.d), x = n.x - n.w / 2, y = n.y - n.h / 2, cls = n.d === 0 ? 'si-mm-root' : 'si-mm-lv' + Math.min(n.d, 2), self = this;
    var label = n.code ? (n.code.lang || 'code') + ' code' : n.t;
    if (n.link) label += ', opens the ' + this.maps[n.link].title + ' map';
    else if (n.kids && n.kids.length) label += n.collapsed ? ', folded' : ', open';
    var s = '<g class="si-mm-node ' + cls + (n.code ? ' si-mm-code' : '') + (n.sticky ? ' si-mm-sticky' : '') + '" data-id="' + n.id + '"' + (n.d > 0 ? ' data-top="' + n.top + '"' : '') +
      ' tabindex="0" role="button" aria-label="' + esc(label) + '" style="--si-mm-c:var(--si-mm-' + (n.branch + 1) + ');--i:' + n.i + '">';
    if (n.code) {
      s += '<rect class="si-mm-shape" x="' + f1(x) + '" y="' + f1(y) + '" width="' + f1(n.w) + '" height="' + f1(n.h) + '" rx="9"/>';
      s += '<text class="si-mm-lang" x="' + f1(x + n.w - 12) + '" y="' + f1(y + 13) + '" text-anchor="end" dominant-baseline="central">' + esc(n.code.lang || 'code') + '</text>';
      s += '<foreignObject x="' + f1(x + 12) + '" y="' + f1(y + 20) + '" width="' + f1(n.w - 20) + '" height="' + f1(n.h - 24) + '"><pre xmlns="http://www.w3.org/1999/xhtml" class="si-mm-pre"><code>' + n.code.html + '</code></pre></foreignObject>';
      return s + '</g>';
    }
    if (n.sticky) {
      s += '<g transform="rotate(-2.5 ' + f1(n.x) + ' ' + f1(n.y) + ')"><rect class="si-mm-shape" x="' + f1(x) + '" y="' + f1(y) + '" width="' + f1(n.w) + '" height="' + f1(n.h) + '" rx="3"/>';
      n.lines.forEach(function (l, k) { s += '<text x="' + f1(x + 14) + '" y="' + f1(y + 23 + k * 21) + '" dominant-baseline="central">' + esc(l) + '</text>'; });
      return s + '</g></g>';
    }
    var rx = n.d === 0 || this.opt('nodes') === 'pill' ? n.h / 2 : (n.d === 1 ? 11 : 8);
    if (this.ul(n)) s += '<rect class="si-mm-hit" x="' + f1(x) + '" y="' + f1(y) + '" width="' + f1(n.w) + '" height="' + f1(n.h) + '"/><line class="si-mm-uline" x1="' + f1(x) + '" y1="' + f1(y + n.h) + '" x2="' + f1(x + n.w) + '" y2="' + f1(y + n.h) + '"/>';
    else s += '<rect class="si-mm-shape" x="' + f1(x) + '" y="' + f1(y) + '" width="' + f1(n.w) + '" height="' + f1(n.h) + '" rx="' + f1(rx) + '"/>';
    var tx = x + L.px;
    if (n.img) {
      var cid = this.uid + 'c' + n.id.replace(/\./g, '_');
      s += '<clipPath id="' + cid + '"><rect x="' + f1(x + 7) + '" y="' + f1(n.y - 17) + '" width="54" height="34" rx="6"/></clipPath>';
      s += '<image href="' + esc(n.img) + '" x="' + f1(x + 7) + '" y="' + f1(n.y - 17) + '" width="54" height="34" preserveAspectRatio="xMidYMid slice" clip-path="url(#' + cid + ')"/>';
      tx += 60;
    }
    s += '<text x="' + f1(tx) + '" y="' + f1(n.y) + '" dominant-baseline="central" font-size="' + L.fs + '" font-weight="' + L.fw + '">' + esc(n.t) + '</text>';
    var cx = tx + n.tw + 8;
    n.chips.forEach(function (c) {
      s += '<g class="si-mm-chip si-mm-chip-' + c.k + '" data-chip="' + c.k + '"><rect x="' + f1(cx) + '" y="' + f1(n.y - 9) + '" width="' + f1(c.w) + '" height="18" rx="8"/><text x="' + f1(cx + c.w / 2) + '" y="' + f1(n.y) + '" text-anchor="middle" dominant-baseline="central">' + esc(c.t) + '</text></g>';
      cx += c.w + 6;
    });
    void self;
    return s + '</g>';
  };

  /* ---------- render ---------- */
  M.render = function () {
    var self = this, m = this.maps[this.st.map], f = this.css('--si-mm-font') || 'sans-serif', root = m.root;
    var all = [];
    (function prep(n, d, b, id, top) { n.id = id; n.top = top; self.size(n, d, f); n.branch = b; all.push(n); vis(n).forEach(function (c, i) { prep(c, d + 1, d === 0 ? i % 5 : b, id + '.' + i, d === 0 ? i : top); }); })(root, 0, 0, 'r', -1);
    all.slice().sort(function (a, b) { return a.d - b.d; }).forEach(function (n, i) { n.i = i; });
    this.layout(root);
    var edges = '', nodes = '';
    all.forEach(function (n) { vis(n).forEach(function (c) { edges += self.edgeSVG(n, c); }); });
    all.forEach(function (n) { nodes += self.nodeSVG(n); });
    var fb = this.fitBox(all), under = '', over = '', wm = this.opt('wm');
    if (wm && this.opt('wm-pos') === 'across') under = '<defs><pattern id="' + this.uid + 'wm" width="340" height="130" patternUnits="userSpaceOnUse" patternTransform="rotate(-22)"><text class="si-mm-wm" x="10" y="70">' + esc(wm) + '</text></pattern></defs><rect x="' + f1(fb.x) + '" y="' + f1(fb.y) + '" width="' + f1(fb.w) + '" height="' + f1(fb.h) + '" fill="url(#' + this.uid + 'wm)"/>';
    else if (wm) over = '<text class="si-mm-wm si-mm-wm-corner" x="' + f1(fb.x + fb.w - 20) + '" y="' + f1(fb.y + fb.h - 18) + '" text-anchor="end">© ' + esc(wm) + '</text>';
    return { root: root, all: all, html: under + '<g class="si-mm-edges">' + edges + '</g><g class="si-mm-nodes">' + nodes + '</g>' + over };
  };
  M.fitBox = function (all) {
    var x0 = Infinity, y0 = Infinity, x1 = -Infinity, y1 = -Infinity;
    all.forEach(function (n) { x0 = Math.min(x0, n.x - n.w / 2); y0 = Math.min(y0, n.y - n.h / 2); x1 = Math.max(x1, n.x + n.w / 2); y1 = Math.max(y1, n.y + n.h / 2); });
    var pt = 64, pb = 56, px = 40;
    return { x: x0 - px, y: y0 - pt, w: x1 - x0 + px * 2, h: y1 - y0 + pt + pb };
  };
  function around(n, k) { var w = Math.max(n.w * k, 220), h = Math.max(n.h * k * 2.2, 150); return { x: n.x - w / 2, y: n.y - h / 2, w: w, h: h }; }
  M.setVB = function (b) { this.vb = b; this.svg.setAttribute('viewBox', [b.x, b.y, b.w, b.h].map(f1).join(' ')); };
  M.tween = function (to, ms, done) {
    var self = this, from = this.vb, id = ++this.tweenId, t0 = 0;
    if (reduce || !ms) { this.setVB(to); if (done) done(); return; }
    function fr(t) {
      if (id !== self.tweenId) return;
      if (!t0) t0 = t;
      var p = Math.min(1, (t - t0) / ms), e = p < 0.5 ? 4 * p * p * p : 1 - Math.pow(-2 * p + 2, 3) / 2;
      self.setVB({ x: from.x + (to.x - from.x) * e, y: from.y + (to.y - from.y) * e, w: from.w + (to.w - from.w) * e, h: from.h + (to.h - from.h) * e });
      if (p < 1) requestAnimationFrame(fr); else if (done) done();
    }
    requestAnimationFrame(fr);
  };
  M.paint = function (anim) {
    this.cur = this.render();
    this.svg.innerHTML = this.cur.html;
    this.svg.setAttribute('aria-label', this.maps[this.st.map].title);
    this.svg.classList.remove('si-mm-anim');
    if (anim && !reduce) { void this.svg.getBoundingClientRect(); this.svg.classList.add('si-mm-anim'); }
    this.drawCrumbs();
  };
  M.show = function (id, opt) {
    var self = this;
    opt = opt || {};
    this.st.map = id;
    this.paint(true);
    var full = this.fitBox(this.cur.all);
    if (opt.fromRoot) this.setVB(around(this.cur.root, 1.6));
    else if (opt.fromId) { var n = this.findNode(opt.fromId); this.setVB(n ? around(n, 1.6) : full); }
    else this.setVB(full);
    this.svg.style.opacity = 1;
    this.tween(full, opt.fromRoot || opt.fromId ? 620 : 0, function () { self.busy = false; });
    if (!opt.fromRoot && !opt.fromId) this.busy = false;
  };
  M.findNode = function (id) { for (var i = 0; i < this.cur.all.length; i++) if (this.cur.all[i].id === id) return this.cur.all[i]; return null; };
  M.relayout = function () { this.paint(false); this.tween(this.fitBox(this.cur.all), 420); };
  M.tidy = function () {
    (function clear(n) { delete n.pos; delete n.bpos; (n.kids || []).forEach(clear); })(this.maps[this.st.map].root);
    this.paint(true); this.tween(this.fitBox(this.cur.all), 420);
  };

  /* ---------- drill-down ---------- */
  M.drill = function (n) {
    var self = this;
    if (this.busy) return; this.busy = true;
    if (this.tour) this.tourStop(true);
    this.st.stack.push({ map: this.st.map, via: n.id });
    this.tween(around(n, 1.4), 420, function () { self.svg.style.opacity = 0; setTimeout(function () { self.show(n.link, { fromRoot: true }); }, reduce ? 0 : 140); });
  };
  M.back = function (k) {
    var self = this;
    if (this.busy || k >= this.st.stack.length) return; this.busy = true;
    if (this.tour) this.tourStop(true);
    var e = this.st.stack[k], vb = this.vb; this.st.stack = this.st.stack.slice(0, k);
    var c = { x: vb.x + vb.w / 2, y: vb.y + vb.h / 2 }, w = vb.w * 2.4, h = vb.h * 2.4;
    this.tween({ x: c.x - w / 2, y: c.y - h / 2, w: w, h: h }, 380, function () { self.svg.style.opacity = 0; setTimeout(function () { self.show(e.map, { fromId: e.via }); }, reduce ? 0 : 140); });
  };
  M.drawCrumbs = function () {
    var self = this, h = '';
    this.st.stack.forEach(function (e, k) { h += '<button type="button" data-back="' + k + '">' + esc(self.maps[e.map].title) + '</button><span aria-hidden="true">›</span>'; });
    this.crumbs.innerHTML = h + '<span class="si-mm-here">' + esc(this.maps[this.st.map].title) + '</span>';
  };

  /* ---------- pointer: pan the canvas, move a topic (Shift: its branch), click ---------- */
  M.act = function (g, chipEl) {
    var n = this.findNode(g.getAttribute('data-id')); if (!n) return;
    var chip = chipEl && chipEl.getAttribute('data-chip');
    if (n.href && (chip === 'href' || (!n.kids && !n.link))) { if (/^[a-z]+:/i.test(n.href)) window.open(n.href, '_blank', 'noopener'); else location.href = n.href; return; }
    if (n.link && (chip === 'link' || !n.kids)) { this.drill(n); return; }
    if (n.kids && n.kids.length && n.d > 0) { n.collapsed = !n.collapsed; this.relayout(); }
  };
  M.pointer = function () {
    var self = this, svg = this.svg, drag = null, moved = false;
    svg.addEventListener('pointerdown', function (e) {
      if (self.busy || e.button > 0) return;
      e.stopPropagation();
      var g = e.target.closest && e.target.closest('.si-mm-node'), n = g && self.findNode(g.getAttribute('data-id'));
      drag = { x: e.clientX, y: e.clientY, vb: self.vb };
      if (n && n.d > 0) { drag.node = n; drag.key = e.shiftKey ? 'bpos' : 'pos'; drag.start = n[drag.key] ? { x: n[drag.key].x, y: n[drag.key].y } : { x: 0, y: 0 }; }
      moved = false;
    });
    window.addEventListener('pointermove', function (e) {
      if (!drag) return;
      var dx = e.clientX - drag.x, dy = e.clientY - drag.y;
      if (!moved && Math.hypot(dx, dy) < 4) return;
      moved = true; svg.classList.add('si-mm-dragging'); self.tweenId++;
      var s = Math.max(drag.vb.w / svg.clientWidth, drag.vb.h / svg.clientHeight);
      if (drag.node) {
        drag.node[drag.key] = { x: drag.start.x + dx * s, y: drag.start.y + dy * s };
        self.paint(false);
        var el = svg.querySelector('[data-id="' + drag.node.id + '"]'); if (el) { el.classList.add('si-mm-moving'); el.parentNode.appendChild(el); }
        return;
      }
      self.setVB({ x: drag.vb.x - dx * s, y: drag.vb.y - dy * s, w: drag.vb.w, h: drag.vb.h });
    });
    window.addEventListener('pointerup', function (e) {
      var was = drag; drag = null; svg.classList.remove('si-mm-dragging');
      if (was && was.node && moved) { self.paint(false); return; }
      if (!was || moved) return;
      var g = e.target.closest && e.target.closest('.si-mm-node');
      if (g && svg.contains(g)) self.act(g, e.target.closest('.si-mm-chip'));
    });
    svg.addEventListener('click', function (e) { e.stopPropagation(); });
    svg.addEventListener('keydown', function (e) {
      var g = e.target.closest('.si-mm-node'); if (!g || self.tour) return;
      var step = { ArrowLeft: [-12, 0], ArrowRight: [12, 0], ArrowUp: [0, -12], ArrowDown: [0, 12] }[e.key];
      if (step) {
        var m = self.findNode(g.getAttribute('data-id')); if (!m || m.d === 0) return;
        e.preventDefault(); e.stopPropagation();
        var key = e.shiftKey ? 'bpos' : 'pos', o = m[key] || { x: 0, y: 0 };
        m[key] = { x: o.x + step[0], y: o.y + step[1] };
        self.paint(false);
        var again = svg.querySelector('[data-id="' + m.id + '"]'); if (again) again.focus();
        return;
      }
      if (e.key !== 'Enter' && e.key !== ' ') return;
      e.preventDefault(); e.stopPropagation();
      var n = self.findNode(g.getAttribute('data-id'));
      self.act(g, n && n.link ? g.querySelector('[data-chip="link"]') : n && n.href ? g.querySelector('[data-chip="href"]') : null);
    });
    svg.addEventListener('wheel', function (e) {
      if (!self.full && !e.ctrlKey && !e.metaKey) return; /* in a page or a slide the wheel scrolls; pinch or Ctrl zooms */
      e.preventDefault();
      var r = svg.getBoundingClientRect();
      self.zoomAt(Math.exp(e.deltaY * (e.ctrlKey ? 0.01 : 0.0015)), e.clientX - r.left, e.clientY - r.top);
    }, { passive: false });
  };
  M.zoomAt = function (f, px, py) {
    var svg = this.svg, vb = this.vb;
    var s = Math.max(vb.w / svg.clientWidth, vb.h / svg.clientHeight);
    var ox = (svg.clientWidth - vb.w / s) / 2, oy = (svg.clientHeight - vb.h / s) / 2;
    var X = vb.x + (px - ox) * s, Y = vb.y + (py - oy) * s;
    this.tweenId++;
    this.setVB({ x: X - (X - vb.x) * f, y: Y - (Y - vb.y) * f, w: vb.w * f, h: vb.h * f });
  };

  /* ---------- / search: every topic, note and code in every map ---------- */
  M.openFind = function () { this.find.hidden = false; this.findQ.value = ''; this.listHits(); this.findQ.focus(); };
  M.closeFind = function () { this.find.hidden = true; };
  M.listHits = function () {
    var self = this, q = this.findQ.value.trim().toLowerCase(), hits = [];
    Object.keys(this.maps).forEach(function (k) {
      var m = self.maps[k];
      if (!q || m.title.toLowerCase().indexOf(q) >= 0) hits.push({ map: k, obj: null, text: m.title, kind: 'map' });
      (function walk(list) { (list || []).forEach(function (n) {
        var hay = (n.t + ' ' + (n.code ? n.code.text : '') + ' ' + (n.badge || '')).toLowerCase();
        if (q && hay.indexOf(q) >= 0) hits.push({ map: k, obj: n, text: n.code ? n.t + ': ' + n.code.text.split('\n')[0] : n.t, kind: n.sticky ? 'note' : n.code ? 'code' : 'topic' });
        walk(n.kids);
      }); })(m.kids);
    });
    this.hits = hits.slice(0, 8); this.sel = 0; this.drawHits();
  };
  M.drawHits = function () {
    var self = this;
    this.findList.innerHTML = this.hits.length ? this.hits.map(function (h, i) { return '<li role="option" data-k="' + i + '" aria-selected="' + (i === self.sel) + '"><b>' + esc(h.text) + '</b><small>' + h.kind + ' · ' + esc(self.maps[h.map].title) + '</small></li>'; }).join('') : '<li aria-disabled="true">No matches</li>';
  };
  M.parentOf = function () {
    var self = this, out = {};
    Object.keys(this.maps).forEach(function (k) { (function walk(list) { (list || []).forEach(function (n) { if (n.link && !out[n.link] && n.link !== self.st0) out[n.link] = { map: k, obj: n }; walk(n.kids); }); })(self.maps[k].kids); });
    return out;
  };
  function unfold(list, obj) { return (list || []).some(function (n) { if (n === obj) return true; if (unfold(n.kids, obj)) { n.collapsed = false; return true; } return false; }); }
  function idIn(list, obj, id) { for (var i = 0; i < list.length; i++) { var me = id + '.' + i; if (list[i] === obj) return me; if (list[i].kids) { var r = idIn(list[i].kids, obj, me); if (r) return r; } } return null; }
  /* the breadcrumb to a map: the maps that link to it, from the first one; their links unfold */
  M.chainTo = function (map) {
    var self = this;
    this.st0 = this.st0 || Object.keys(this.maps)[0];
    var par = this.parentOf(), chain = [], m = map, guard = 0;
    while (par[m] && guard++ < 50) { chain.unshift(par[m]); m = par[m].map; }
    chain.forEach(function (c) { unfold(self.maps[c.map].kids, c.obj); });
    return chain.map(function (c) { return { map: c.map, via: idIn(self.maps[c.map].kids, c.obj, 'r') }; });
  };
  M.pulse = function (obj) {
    var n = null; this.cur.all.forEach(function (m) { if (m === obj) n = m; }); if (!n) return;
    this.tween(around(n, 3), 520);
    var el = this.svg.querySelector('[data-id="' + n.id + '"]');
    if (el) { el.classList.remove('si-mm-pulse'); void el.getBoundingClientRect(); el.classList.add('si-mm-pulse'); el.focus({ preventScroll: true }); }
  };
  M.goHit = function (h) {
    var self = this;
    if (!h) return; this.closeFind(); var t = this.active(); if (t.tour) t.tourStop(true);
    if (h.obj) unfold(this.maps[h.map].kids, h.obj);
    if (this.is3d) {
      if (h.map !== this.st.map) this.st.stack = this.chainTo(h.map);
      this.d3.reveal(h.map, h.obj);
      return;
    }
    if (h.map === this.st.map) { this.paint(false); if (h.obj) this.pulse(h.obj); else this.tween(this.fitBox(this.cur.all), 420); return; }
    this.st.stack = this.chainTo(h.map);
    this.busy = true; this.svg.style.opacity = 0;
    setTimeout(function () { self.show(h.map, { fromRoot: true }); if (h.obj) setTimeout(function () { self.pulse(h.obj); }, reduce ? 0 : 680); }, reduce ? 0 : 160);
  };

  /* ---------- Play: the overview, then one branch per step ---------- */
  M.tourSteps = function () { return [null].concat(vis(this.cur.root).filter(function (n) { return !n.sticky; })); };
  M.tourGo = function (k) {
    var steps = this.tourSteps(); k = Math.max(0, Math.min(steps.length - 1, k)); this.tour.k = k;
    var t = steps[k];
    this.svg.querySelectorAll('[data-top]').forEach(function (el) { el.classList.toggle('si-mm-dim', !!t && el.getAttribute('data-top') !== String(t.top)); });
    this.tween(t ? this.fitBox(this.cur.all.filter(function (n) { return n.d > 0 && n.top === t.top; })) : this.fitBox(this.cur.all), 700);
    this.tourEl.textContent = (k + 1) + ' / ' + steps.length + ' · ' + (t ? t.t : this.maps[this.st.map].title);
  };
  M.tourStart = function () {
    var self = this;
    this.tour = { k: 0 }; this.tourEl.hidden = false;
    this.playBtn.setAttribute('aria-pressed', 'true'); this.playBtn.textContent = 'Stop';
    this.tourGo(0);
    this.tour.timer = setInterval(function () { if (self.tour.k >= self.tourSteps().length - 1) { self.tourStop(); return; } self.tourGo(self.tour.k + 1); }, 3200);
  };
  M.tourStop = function (quiet) {
    if (!this.tour) return; clearInterval(this.tour.timer); this.tour = null;
    this.tourEl.hidden = true;
    this.playBtn.setAttribute('aria-pressed', 'false'); this.playBtn.textContent = 'Play';
    this.svg.querySelectorAll('.si-mm-dim').forEach(function (el) { el.classList.remove('si-mm-dim'); });
    if (!quiet) this.tween(this.fitBox(this.cur.all), 600);
  };

  function start() { document.querySelectorAll('.si-mindmap').forEach(function (el) { if (!el.siMindmap) el.siMindmap = new Mindmap(el); }); }
  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', start); else start();
})();
