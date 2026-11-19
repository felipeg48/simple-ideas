  /* ---------- motion: the transitions the runtime plays itself (cube, flip,
   * iris, wipe, shatter, depth, blinds, portal, glitch, stack: Web Animations,
   * mirrored going back) and the build animations (animate: on a block,
   * steps: focus|deal|reorder on a list). Off under reduced motion, in print,
   * the overview, thumbnails, handouts and jumps: the page shows the final state. ---------- */
  var EASE = 'cubic-bezier(.22,.8,.2,1)', IN_OUT = 'cubic-bezier(.65,0,.35,1)';
  function motionLive() {
    return !reduceMotion() && role !== 'thumb' && !deck.classList.contains('si-overview') &&
      !root.hasAttribute('data-si-handout') && !(window.matchMedia && window.matchMedia('print').matches);
  }
  /* where the last click or tap landed, as a fraction of the deck (iris, shatter) */
  var lastPoint = null;
  deck.addEventListener('pointerdown', function (e) {
    var r = deck.getBoundingClientRect();
    lastPoint = { x: (e.clientX - r.left) / r.width, y: (e.clientY - r.top) / r.height, t: Date.now() };
  }, true);
  function clickPoint() { return lastPoint && Date.now() - lastPoint.t < 1500 ? lastPoint : { x: 0.5, y: 0.5 }; }
  function pct(v) { return (v * 100).toFixed(1) + '%'; }
  /* a copy of a slide for tiles and strips: no ids, no media playing, not focusable */
  function txClone(s, clip) {
    var c = s.cloneNode(true);
    c.removeAttribute('id');
    c.classList.add('si-tx-tile');
    c.setAttribute('aria-hidden', 'true');
    c.setAttribute('inert', '');
    each(c.querySelectorAll('[id]'), function (e) { e.removeAttribute('id'); });
    each(c.querySelectorAll('video,audio,iframe'), function (e) { e.parentNode.replaceChild(document.createElement('span'), e); });
    c.style.clipPath = clip;
    c.style.visibility = 'visible';
    deck.appendChild(c);
    return c;
  }
  /* each transition: (a, b, ctx) → [element, keyframes, timing] for the forward
     move from slide a to the next slide b; ctx.add puts a temporary element on
     the deck, ctx.hide hides a real slide behind its copies. Going back plays
     the same motion backwards (a mirror), unless the transition says once. */
  var motionTx = {
    cube: { ms: 1000, fx: function (a, b) {
      return [[a, [{ transformOrigin: '100% 50%', transform: 'translateX(0) rotateY(0)', filter: 'brightness(1)' }, { transformOrigin: '100% 50%', transform: 'translateX(-100%) rotateY(-90deg)', filter: 'brightness(.6)' }], { duration: 1000, easing: IN_OUT }],
        [b, [{ transformOrigin: '0 50%', transform: 'translateX(100%) rotateY(90deg)', filter: 'brightness(.6)' }, { transformOrigin: '0 50%', transform: 'translateX(0) rotateY(0)', filter: 'brightness(1)' }], { duration: 1000, easing: IN_OUT }]];
    } },
    flip: { ms: 900, fx: function (a, b) {
      return [[a, [{ transform: 'rotateY(0)' }, { transform: 'rotateY(180deg)' }], { duration: 900, easing: IN_OUT }],
        [b, [{ transform: 'rotateY(-180deg)' }, { transform: 'rotateY(0)' }], { duration: 900, easing: IN_OUT }]];
    } },
    iris: { ms: 900, fx: function (a, b, ctx) {
      var at = ' at ' + pct(ctx.point.x) + ' ' + pct(ctx.point.y);
      return [[a, [{ transform: 'scale(1)', filter: 'brightness(1)' }, { transform: 'scale(.94)', filter: 'brightness(.7)' }], { duration: 900, easing: EASE }],
        [b, [{ clipPath: 'circle(0%' + at + ')' }, { clipPath: 'circle(150%' + at + ')' }], { duration: 900, easing: 'cubic-bezier(.7,0,.25,1)' }]];
    } },
    wipe: { ms: 900, fx: function (a, b, ctx) {
      var bar = ctx.add('si-tx-bar');
      return [[b, [{ clipPath: 'inset(0 100% 0 0)' }, { clipPath: 'inset(0 0 0 0)' }], { duration: 900, easing: IN_OUT }],
        [a, [{ filter: 'blur(0)' }, { filter: 'blur(.25vw)' }], { duration: 900 }],
        [bar, [{ left: '-18%' }, { left: '100%' }], { duration: 900, easing: IN_OUT }]];
    } },
    shatter: { ms: 1300, fx: function (a, b, ctx) {
      var cols = 8, rows = 5, out = [], p = ctx.point;
      ctx.hide(a);
      for (var r = 0; r < rows; r++) for (var c = 0; c < cols; c++) {
        var t = ctx.tile(a, 'inset(' + (r * 100 / rows) + '% ' + (100 - (c + 1) * 100 / cols) + '% ' + (100 - (r + 1) * 100 / rows) + '% ' + (c * 100 / cols) + '%)');
        var dist = Math.hypot(c + 0.5 - cols * p.x, r + 0.5 - rows * p.y), rot = (Math.random() - 0.5) * 70;
        out.push([t, [{ transform: 'none', opacity: 1 }, { transform: 'translate(' + ((c - cols / 2) * 3.6).toFixed(1) + '%,' + (40 + Math.random() * 20).toFixed(1) + '%) rotate(' + rot.toFixed(0) + 'deg) scale(.6)', opacity: 0 }],
          { duration: 900, delay: Math.round(dist * 45), easing: 'cubic-bezier(.5,0,.9,.5)' }]);
      }
      out.push([b, [{ transform: 'scale(1.04)', filter: 'brightness(.85)' }, { transform: 'scale(1)', filter: 'brightness(1)' }], { duration: 1300, easing: EASE }]);
      return out;
    } },
    depth: { ms: 1120, fx: function (a, b) {
      return [[a, [{ transform: 'scale(1)', filter: 'blur(0) brightness(1)', opacity: 1 }, { transform: 'scale(.78)', filter: 'blur(.5vw) brightness(.5)', opacity: 0 }], { duration: 1000, easing: EASE }],
        [b, [{ transform: 'scale(1.3)', filter: 'blur(.6vw)', opacity: 0 }, { transform: 'scale(1)', filter: 'blur(0)', opacity: 1 }], { duration: 1000, delay: 120, easing: EASE }]];
    } },
    blinds: { ms: 1140, fx: function (a, b, ctx) {
      var n = 8, out = [];
      ctx.hide(b);
      for (var i = 0; i < n; i++) {
        var t = ctx.tile(b, 'inset(0 ' + (100 - (i + 1) * 100 / n) + '% 0 ' + (i * 100 / n) + '%)');
        t.style.transformOrigin = ((i + 0.5) * 100 / n) + '% 50%';
        out.push([t, [{ transform: 'rotateY(90deg)' }, { transform: 'rotateY(0)' }], { duration: 650, delay: i * 70, easing: EASE }]);
      }
      return out;
    } },
    portal: { ms: 1100, fx: function (a, b) {
      /* into the block named with <!-- id: --> on slide a, else its first picture */
      var t = a.querySelector('[data-si-morph]') || a.querySelector('.si-figure,.si-image,img,svg,video'), o = '50% 50%', s0 = 0.22;
      if (t) {
        var ra = a.getBoundingClientRect(), rt = t.getBoundingClientRect();
        if (rt.width && ra.width) {
          o = pct((rt.left + rt.width / 2 - ra.left) / ra.width) + ' ' + pct((rt.top + rt.height / 2 - ra.top) / ra.height);
          s0 = Math.min(0.6, Math.max(0.08, rt.width / ra.width));
        }
      }
      var e = 'cubic-bezier(.7,0,.3,1)';
      return [[a, [{ transformOrigin: o, transform: 'scale(1)', opacity: 1 }, { transformOrigin: o, transform: 'scale(' + (1 / s0).toFixed(3) + ')', opacity: 1, offset: 0.75 }, { transformOrigin: o, transform: 'scale(' + (1.1 / s0).toFixed(3) + ')', opacity: 0 }], { duration: 1100, easing: e }],
        [b, [{ transformOrigin: o, transform: 'scale(' + s0.toFixed(3) + ')', opacity: 0, borderRadius: '2em' }, { transformOrigin: o, transform: 'scale(.9)', opacity: 1, offset: 0.75 }, { transformOrigin: o, transform: 'scale(1)', opacity: 1, borderRadius: '0' }], { duration: 1100, easing: e }]];
    } },
    glitch: { ms: 550, once: true, fx: function (a, b) {
      var st = 'steps(2)', m = '#ff2bd6', c = '#00e5ff';
      return [[a, [{ opacity: 1 }, { opacity: 0 }], { duration: 200 }],
        [b, [{ clipPath: 'inset(10% 0 70% 0)', transform: 'translate(-6%,0)', textShadow: '.2em 0 ' + m + ',-.2em 0 ' + c, easing: st },
          { clipPath: 'inset(60% 0 10% 0)', transform: 'translate(5%,0)', easing: st },
          { clipPath: 'inset(30% 0 40% 0)', transform: 'translate(-3%,0)', textShadow: '-.2em 0 ' + m + ',.2em 0 ' + c, easing: st },
          { clipPath: 'inset(0 0 85% 0)', transform: 'translate(2%,0)', easing: st },
          { clipPath: 'inset(80% 0 0 0)', transform: 'translate(-1%,0)', easing: st },
          { clipPath: 'inset(0 0 0 0)', transform: 'none', textShadow: 'none' }], { duration: 550 }]];
    } },
    stack: { ms: 900, fx: function (a, b) {
      return [[a, [{ transform: 'scale(1) translateY(0)', filter: 'brightness(1)' }, { transform: 'scale(.9) translateY(-4%)', filter: 'brightness(.55)' }], { duration: 900, easing: EASE }],
        [b, [{ transform: 'translateY(105%) rotate(4deg)', boxShadow: '0 -1em 3em rgb(0 0 0 / .35)' }, { transform: 'translateY(0) rotate(0)', boxShadow: '0 -1em 3em rgb(0 0 0 / 0)' }], { duration: 900, easing: EASE }]];
    } }
  };
  var txRun = null;
  function txStop() {
    if (!txRun) return;
    var r = txRun; txRun = null;
    r.anims.forEach(function (x) { try { x.cancel(); } catch (e) {} });
    r.clean();
  }
  /* from → to by transition name; back plays the mirror of to → from */
  function txPlay(name, from, to, back) {
    txStop();
    var T = motionTx[name], made = [], hidden = [];
    var once = T.once || !back, a = once ? from : to, b = once ? to : from;
    var v = getComputedStyle(b).getPropertyValue('--si-tx-duration').trim();
    var ms = /ms$/.test(v) ? parseFloat(v) : parseFloat(v) * 1000, k = ms > 0 ? ms / T.ms : 1;
    var ctx = {
      point: clickPoint(),
      add: function (cls) { var e = document.createElement('div'); e.className = cls; e.setAttribute('aria-hidden', 'true'); deck.appendChild(e); made.push(e); return e; },
      tile: function (s, clip) { var t = txClone(s, clip); made.push(t); return t; },
      hide: function (s) { s.style.visibility = 'hidden'; hidden.push(s); }
    };
    deck.classList.add('si-tx-run');
    from.classList.add('si-tx-on'); to.classList.add('si-tx-on');
    a.classList.add('si-tx-a'); b.classList.add('si-tx-b');
    var list = T.fx(a, b, ctx), total = 0;
    list.forEach(function (x) { total = Math.max(total, (x[2].delay || 0) + x[2].duration); });
    var anims = list.map(function (x) {
      var t = x[2], d = t.duration * k, delay = (t.delay || 0) * k;
      var opts = { duration: d, delay: once ? delay : total * k - delay - d, easing: t.easing || 'ease', fill: 'both', direction: once ? 'normal' : 'reverse' };
      return x[0].animate(x[1], opts);
    });
    var run = txRun = { anims: anims, clean: function () {
      made.forEach(function (e) { e.remove(); });
      hidden.forEach(function (s) { s.style.visibility = ''; });
      deck.classList.remove('si-tx-run');
      [from, to].forEach(function (s) { s.classList.remove('si-tx-on', 'si-tx-a', 'si-tx-b'); });
    } };
    run.done = Promise.all(anims.map(function (x) { return x.finished; })).then(function () {
      if (txRun !== run) return;
      txRun = null;
      anims.forEach(function (x) { x.cancel(); });
      run.clean();
    }, function () {});
    return run;
  }

  /* ---------- build animations: web/build.js (window.siBuild, shared with documents) ---------- */
  function buildRun(el) { window.siBuild.run(el); }
  function buildStop() { window.siBuild.stop(); }
  function slideStop() { window.siBuild.slideStop(); }
  function carouselSet(l) { window.siBuild.carouselSet(l); }
  function reorderSet(l, on) { window.siBuild.reorderSet(l, on); }
  function reorderFirst(s) { return window.siBuild.reorderFirst(s); }
  function reorderPlay(at) { window.siBuild.reorderPlay(at); }
  window.siBuild.init(deck);
  /* the build step a block arrives with (0: with the slide) */
  function buildStep(el) { var s = el.closest('.si-step[data-step]'); return s ? +s.getAttribute('data-step') : 0; }
  /* run the animations of slide s that arrive at step k, after the transition when one plays */
  function buildAt(s, k, after) {
    var els = Array.prototype.filter.call(s.querySelectorAll('[data-si-anim]'), function (e) { return buildStep(e) === k; });
    if (!els.length) return;
    var start = function () { if (slides[cur] === s && step === k) els.forEach(buildRun); };
    if (after) after.then(start); else els.forEach(buildRun);
  }
