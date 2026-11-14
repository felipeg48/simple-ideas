/* simple-ideas timelines: in a frame (data-frame: snake and line, eras and
   cards when landscape) every card (eras: every era column) has a fixed room;
   the text shrinks (--_fit on the track) until the fullest one fits. Text is in cqi, so the fit holds at every size and is
   only measured again when the frame changes shape. */
(function () {
  function fit(tl) {
    var track = tl.querySelector('.si-tl-track');
    if (!track) return;
    var cards = tl.querySelectorAll(tl.getAttribute('data-layout') === 'eras' ? '.si-tl-era' : '.si-tl-card');
    track.style.setProperty('--_fit', 1);
    if (!tl.hasAttribute('data-frame') || getComputedStyle(tl).aspectRatio === 'auto') return; /* plain flow */
    var f = 1;
    for (var pass = 0; pass < 8; pass++) {
      var worst = 1;
      for (var i = 0; i < cards.length; i++) {
        var c = cards[i];
        if (c.clientHeight > 0) worst = Math.max(worst, c.scrollHeight / c.clientHeight);
      }
      if (worst <= 1.01) break;
      f = Math.max(0.5, f * Math.min(0.97, 1 / Math.sqrt(worst)));
      track.style.setProperty('--_fit', f.toFixed(3));
      if (f === 0.5) break;
    }
  }
  function all() { Array.prototype.forEach.call(document.querySelectorAll('.si-tl'), fit); }
  var shape = '';
  function onResize() {
    var w = window.innerWidth, h = window.innerHeight;
    var s = (w > h * 0.8 && w > 640) + ':' + Math.round(w / h * 20);
    if (s !== shape) { shape = s; all(); }
  }
  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', all); else all();
  window.addEventListener('load', all);
  window.addEventListener('resize', onResize);
  if (document.fonts && document.fonts.ready) document.fonts.ready.then(all);
  window.siTimelineFit = all;
})();
