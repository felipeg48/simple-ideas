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
