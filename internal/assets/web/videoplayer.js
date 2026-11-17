/* si video player: every .si-vp on the page (a video fence).
   inline needs no JS; modal opens a native dialog lightbox (Esc, the close
   button and a backdrop click all close it, and closing pauses playback; the
   dialog focus-traps on its own); facade swaps its poster for a player element
   on the first click (lazy, nothing loads until then). prefers-reduced-motion
   is kept by the stylesheet. */
(function () {
  'use strict';
  function play(v) { try { var p = v.play(); if (p && p.catch) p.catch(function () {}); } catch (e) {} }

  document.querySelectorAll('.si-vp[data-vp-mode="modal"]').forEach(function (fig) {
    var open = fig.querySelector('.si-vp-open');
    var dlg = fig.querySelector('.si-vp-modal');
    var vid = dlg && dlg.querySelector('.si-vp-video');
    var close = dlg && dlg.querySelector('.si-vp-close');
    if (!open || !dlg || !vid) return;
    function shut() { if (dlg.open) dlg.close(); }
    open.addEventListener('click', function () {
      if (typeof dlg.showModal === 'function') dlg.showModal(); else dlg.setAttribute('open', '');
      play(vid);
    });
    if (close) close.addEventListener('click', shut);
    dlg.addEventListener('click', function (e) { if (e.target === dlg) shut(); }); /* backdrop */
    dlg.addEventListener('close', function () { vid.pause(); });
    dlg.addEventListener('cancel', function () { vid.pause(); }); /* Esc, for the setAttribute fallback */
  });

  document.querySelectorAll('.si-vp[data-vp-mode="facade"]').forEach(function (fig) {
    var open = fig.querySelector('.si-vp-open');
    var src = open && open.getAttribute('data-si-vp-src');
    if (!open || !src) return;
    open.addEventListener('click', function () {
      var v = document.createElement('video');
      v.className = 'si-vp-video';
      v.controls = true;
      v.setAttribute('playsinline', '');
      v.preload = 'metadata';
      var poster = fig.querySelector('.si-vp-poster');
      if (poster) v.poster = poster.currentSrc || poster.src;
      if (fig.hasAttribute('data-vp-loop')) v.loop = true;
      v.src = src;
      open.replaceWith(v);
      play(v);
    });
  });
})();
