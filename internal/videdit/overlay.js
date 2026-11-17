/* si video edit: draws every callout as it is at time t (seconds).
   window.VE.at(t) is called per frame: by the preview with the video's time,
   by the build (headless Chrome) once per shot. Deterministic: the same t
   always draws the same picture. */
(function () {
  var cs = [].slice.call(document.querySelectorAll('.ve-c'));
  function clamp(x) { return x < 0 ? 0 : x > 1 ? 1 : x; }
  function out3(p) { return 1 - Math.pow(1 - p, 3); }
  function back(p) { var c = 1.70158; return 1 + (c + 1) * Math.pow(p - 1, 3) + c * Math.pow(p - 1, 2); }
  function smooth(p) { return p * p * (3 - 2 * p); }
  /* a moving callout: its place along the keyframes (percentages of the
     stage) at l seconds, the same length per unit of distance */
  function place(el, l) {
    var ps = el.dataset.path.split(';').map(function (s) { return s.split(',').map(Number); });
    var p = clamp(l / (+el.dataset.pdur || 1)); if (el.dataset.ease !== 'linear') p = smooth(p);
    var lens = [], all = 0;
    for (var i = 1; i < ps.length; i++) { var n = Math.hypot(ps[i][0] - ps[i - 1][0], ps[i][1] - ps[i - 1][1]); lens.push(n); all += n; }
    var want = p * all, k = 0;
    while (k < lens.length - 1 && want > lens[k]) { want -= lens[k]; k++; }
    var f = lens[k] ? clamp(want / lens[k]) : 1, a = ps[k], b = ps[k + 1];
    var x = a[0] + (b[0] - a[0]) * f, y = a[1] + (b[1] - a[1]) * f;
    if (el.classList.contains('ve-arrow')) { arrowTo(el, x, y); return; }
    el.style.left = x.toFixed(3) + '%'; el.style.top = y.toFixed(3) + '%';
  }
  /* an arrow whose head moves: the line and head drawn again (as callout.go draws them) */
  function arrowTo(el, xp, yp) {
    var svg = el.querySelector('svg'), vb = svg.viewBox.baseVal, tl = el.dataset.tail.split(',').map(Number), sw = +el.dataset.sw;
    var x1 = tl[0], y1 = tl[1], x2 = xp / 100 * vb.width, y2 = yp / 100 * vb.height;
    var ang = Math.atan2(y2 - y1, x2 - x1), hl = sw * 4.2, bx = x2 - hl * Math.cos(ang), by = y2 - hl * Math.sin(ang);
    var px = -Math.sin(ang) * hl * 0.6, py = Math.cos(ang) * hl * 0.6, r = function (v) { return v.toFixed(1); };
    el.querySelector('.ve-path').setAttribute('d', 'M' + r(x1) + ' ' + r(y1) + 'L' + r(bx) + ' ' + r(by));
    el.querySelector('.ve-head').setAttribute('d', 'M' + r(x2) + ' ' + r(y2) + 'L' + r(bx + px) + ' ' + r(by + py) + 'L' + r(bx - px) + ' ' + r(by - py) + 'Z');
  }
  function at(t) {
    cs.forEach(function (el) {
      var s = +el.dataset.start, d = +el.dataset.dur, l = t - s;
      if (!(l >= 0 && l < d)) { el.style.visibility = 'hidden'; return; }
      el.style.visibility = 'visible';
      if (el.dataset.path) place(el, l);
      var pin = +el.dataset.in, pout = +el.dataset.out;
      var e = pin > 0 ? clamp(l / pin) : 1, x = pout > 0 ? clamp((l - (d - pout)) / pout) : 0;
      var en = el.dataset.enter, inn = el.querySelector('.ve-in') || el, op = 1, tf = '';
      if (en === 'fade') op = out3(e);
      else if (en === 'pop') { op = clamp(e * 3); tf = 'scale(' + (0.55 + 0.45 * back(e)).toFixed(4) + ')'; }
      else if (en === 'slide-in') { op = out3(e); tf = 'translateY(' + ((1 - out3(e)) * 6).toFixed(3) + 'cqmin)'; }
      else if (en === 'type') {
        var tx = el.querySelector('.ve-type');
        if (tx) { var full = Array.from(tx.dataset.text); tx.textContent = full.slice(0, Math.round(full.length * e)).join(''); }
      } else if (en === 'draw') {
        var p = el.querySelector('.ve-path'); if (p) p.style.strokeDashoffset = (1 - out3(e)).toFixed(4);
        var h = el.querySelector('.ve-head'); if (h) h.style.opacity = e >= 0.92 ? 1 : 0;
        var lb = el.querySelector('.ve-label'); if (lb) lb.style.opacity = clamp(e * 2);
      }
      if (el.dataset.exit === 'fade') op *= 1 - x;
      inn.style.opacity = op.toFixed(4);
      inn.style.transform = tf;
    });
  }
  window.VE = { at: at };
})();
