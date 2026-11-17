/* si video edit: draws every callout as it is at time t (seconds).
   window.VE.at(t) is called per frame: by the preview with the video's time,
   by the build (headless Chrome) once per shot. Deterministic: the same t
   always draws the same picture. */
(function () {
  var cs = [].slice.call(document.querySelectorAll('.ve-c'));
  function clamp(x) { return x < 0 ? 0 : x > 1 ? 1 : x; }
  function out3(p) { return 1 - Math.pow(1 - p, 3); }
  function back(p) { var c = 1.70158; return 1 + (c + 1) * Math.pow(p - 1, 3) + c * Math.pow(p - 1, 2); }
  function at(t) {
    cs.forEach(function (el) {
      var s = +el.dataset.start, d = +el.dataset.dur, l = t - s;
      if (!(l >= 0 && l < d)) { el.style.visibility = 'hidden'; return; }
      el.style.visibility = 'visible';
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
