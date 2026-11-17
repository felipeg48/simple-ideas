/* si Studio, the Video view: the preview page with its hooks. A callout on
   the frame is dragged into place (a rectangle also resized by its corner,
   an arrow by its two ends); on release the page tells the Studio the new
   place in source pixels ({type:'si-ve-move', line, to|at|from|rect}) and
   the Studio writes it into that line. A marker, a list row or a callout
   clicked selects its line ('si-ve-select'); the Studio's cursor line comes
   back as 'si-ve-line' and is marked here. A moving callout (a path) is
   moved by its places: handles shown while paused, one dragged sends
   {key, at} (a rectangle's place is its top-left corner). A zoom's or pan's
   rectangle shows while its line is the cursor's and drags like a box.
   Titles stay where the text puts them. */
(function () {
  var P = window.VP, v = document.querySelector('video'), stage = document.querySelector('.ve-stage');
  if (!P || !v || !stage) return;
  document.documentElement.classList.add('ve-edit');
  var send = function (m) { if (parent !== window) parent.postMessage(m, '*'); };
  var round = function (n) { return Math.round(n); };
  function box() { return stage.getBoundingClientRect(); }
  function src(ev) { var r = box(); return [(ev.clientX - r.left) / r.width * P.w, (ev.clientY - r.top) / r.height * P.h]; }
  function pc(el, k) { return parseFloat(el.style[k]) || 0; }
  var RECT = /\bve-(mark|spotlight|hide|camrect)\b/;
  var pcs = function (n) { return n + '%'; };
  // a zoom's or pan's rectangle: shown while its line is the cursor's
  P.marks.forEach(function (m) {
    if (!m.box) return;
    var r = document.createElement('div');
    r.className = 've-c ve-camrect'; r.dataset.line = m.line; r.title = 'line ' + m.line + ': ' + m.label;
    r.style.left = pcs(m.box[0] / P.w * 100); r.style.top = pcs(m.box[1] / P.h * 100);
    r.style.width = pcs(m.box[2] / P.w * 100); r.style.height = pcs(m.box[3] / P.h * 100);
    stage.appendChild(r);
  });
  // a moving callout's places (a rectangle's at its size): handles while paused
  var keyed = [];
  [].forEach.call(stage.querySelectorAll('.ve-c[data-path]'), function (el) {
    var ps = el.dataset.path.split(';').map(function (s) { return s.split(',').map(Number); }), rect = RECT.test(el.className);
    var svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg'), line = document.createElementNS(svg.namespaceURI, 'polyline');
    svg.setAttribute('class', 've-route'); svg.setAttribute('viewBox', '0 0 100 100'); svg.setAttribute('preserveAspectRatio', 'none');
    line.setAttribute('points', ps.map(function (p) { return p[0] + ',' + p[1]; }).join(' '));
    svg.appendChild(line); stage.appendChild(svg);
    var hs = ps.map(function (p, k) {
      var g = document.createElement('i');
      g.className = 've-key' + (rect ? ' ve-key-rect' : ''); g.dataset.grip = 'key'; g.dataset.key = k; g.dataset.line = el.dataset.line;
      g.title = 'line ' + el.dataset.line + ', place ' + (k + 1);
      g.style.left = pcs(p[0]); g.style.top = pcs(p[1]);
      if (rect) { g.style.width = p.length > 2 ? pcs(p[2]) : el.style.width; g.style.height = p.length > 2 ? pcs(p[3]) : el.style.height; }
      stage.appendChild(g);
      return g;
    });
    keyed.push({ el: el, svg: svg, hs: hs });
  });
  (function show() {
    keyed.forEach(function (k) {
      var on = v.paused && (k.el.style.visibility === 'visible' || k.el.classList.contains('ve-sel')) || (drag && drag.key && k.hs.indexOf(drag.target) >= 0);
      k.svg.style.display = on ? '' : 'none'; k.hs.forEach(function (h) { h.hidden = !on; });
    });
    requestAnimationFrame(show);
  })();
  // the handles: a rectangle's corner, an arrow's two ends
  [].forEach.call(stage.querySelectorAll('.ve-c'), function (el) {
    if (el.dataset.path || el.classList.contains('ve-title')) { el.classList.add('ve-fixed'); return; }
    if (RECT.test(el.className)) { var g = document.createElement('i'); g.className = 've-grip'; g.dataset.grip = 'size'; el.appendChild(g); }
    if (el.classList.contains('ve-arrow')) {
      ['tail', 'head'].forEach(function (k) {
        var xy = el.dataset[k].split(',').map(Number), g = document.createElement('i');
        g.className = 've-end'; g.dataset.grip = k;
        g.style.left = (xy[0] / P.w * 100) + '%'; g.style.top = (xy[1] / P.h * 100) + '%';
        el.appendChild(g);
      });
    }
  });
  var drag = null;
  stage.addEventListener('pointerdown', function (ev) {
    if (ev.target.classList && ev.target.classList.contains('ve-key')) { // a place on a path
      var kh = ev.target;
      send({ type: 'si-ve-select', line: +kh.dataset.line });
      ev.preventDefault(); ev.stopPropagation();
      v.pause();
      stage.setPointerCapture(ev.pointerId);
      drag = { el: kh, key: true, grip: 'key', target: kh, at: src(ev), left: pc(kh, 'left'), top: pc(kh, 'top'), moved: false };
      return;
    }
    var el = ev.target.closest && ev.target.closest('.ve-c');
    if (!el) return;
    send({ type: 'si-ve-select', line: +el.dataset.line });
    if (el.classList.contains('ve-fixed')) return;
    var grip = ev.target.dataset && ev.target.dataset.grip;
    if (el.classList.contains('ve-arrow') && !grip) return;
    ev.preventDefault(); ev.stopPropagation();
    v.pause();
    stage.setPointerCapture(ev.pointerId);
    drag = { el: el, grip: grip || '', target: grip ? ev.target : el, at: src(ev), left: pc(grip && grip !== 'size' ? ev.target : el, 'left'), top: pc(grip && grip !== 'size' ? ev.target : el, 'top'),
      w: pc(el, 'width'), h: pc(el, 'height'), moved: false };
    el.classList.add('ve-dragging');
  }, true);
  stage.addEventListener('pointermove', function (ev) {
    if (!drag) return;
    var now = src(ev), dx = (now[0] - drag.at[0]) / P.w * 100, dy = (now[1] - drag.at[1]) / P.h * 100;
    drag.moved = drag.moved || Math.abs(dx) + Math.abs(dy) > 0.2;
    if (drag.grip === 'size') {
      drag.el.style.width = Math.max(0.5, drag.w + dx) + '%'; drag.el.style.height = Math.max(0.5, drag.h + dy) + '%';
    } else {
      drag.target.style.left = (drag.left + dx) + '%'; drag.target.style.top = (drag.top + dy) + '%';
    }
  });
  function end() {
    if (!drag) return;
    var d = drag, el = d.el, m = { type: 'si-ve-move', line: +el.dataset.line };
    drag = null;
    el.classList.remove('ve-dragging');
    if (!d.moved) return;
    var px = function (pctX, pctY) { return [round(pctX / 100 * P.w), round(pctY / 100 * P.h)]; };
    if (d.key) {
      m.key = +el.dataset.key; m.at = px(pc(el, 'left'), pc(el, 'top'));
    } else if (d.grip === 'tail' || d.grip === 'head') {
      m[d.grip === 'tail' ? 'from' : 'to'] = px(pc(d.target, 'left'), pc(d.target, 'top'));
    } else if (RECT.test(el.className)) {
      var a = px(pc(el, 'left'), pc(el, 'top')), z = px(pc(el, 'width'), pc(el, 'height'));
      m.rect = [a[0], a[1], Math.max(1, z[0]), Math.max(1, z[1])];
    } else if (el.classList.contains('ve-bubble')) {
      m.to = px(pc(el, 'left'), pc(el, 'top'));
    } else { // text, keys, a badge, a lower third: its centre
      var r = box(), b = (el.querySelector('.ve-in') || el).getBoundingClientRect();
      m.at = [round((b.left + b.width / 2 - r.left) / r.width * P.w), round((b.top + b.height / 2 - r.top) / r.height * P.h)];
    }
    send(m);
  }
  stage.addEventListener('pointerup', end);
  stage.addEventListener('pointercancel', end);
  // a marker or a row of the list: its line
  document.addEventListener('click', function (ev) {
    var x = ev.target.closest && ev.target.closest('.vp-strip .m, .vp-lines li');
    if (x) send({ type: 'si-ve-select', line: +x.dataset.line });
  }, true);
  var strip = document.querySelector('.vp-strip');
  if (strip) strip.addEventListener('pointerdown', function (ev) {
    var m = ev.target.closest && ev.target.closest('.m');
    if (m) send({ type: 'si-ve-select', line: +m.dataset.line });
  }, true);
  // the Studio's cursor line: marked, and shown when paused
  addEventListener('message', function (ev) {
    var d = ev.data || {};
    if (d.type !== 'si-ve-line') return;
    [].forEach.call(document.querySelectorAll('.ve-sel'), function (e) { e.classList.remove('ve-sel'); });
    [].forEach.call(document.querySelectorAll('[data-line="' + (+d.line) + '"]'), function (e) { e.classList.add('ve-sel'); });
    var mk = P.marks.filter(function (m) { return m.line === +d.line; })[0];
    if (mk && v.paused && d.seek !== false && (v.currentTime < mk.from || v.currentTime >= (mk.range ? mk.to : mk.from + Math.max(mk.dur || 0, 0.5)))) v.currentTime = mk.from + 0.01;
  });
  window.siVideoEdit = { ready: true };
  send({ type: 'si-ve-ready' });
})();
