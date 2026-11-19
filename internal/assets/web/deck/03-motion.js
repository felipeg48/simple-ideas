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

  /* ---------- build animations ---------- */
  var buildRuns = [];
  function buildStop() { var r = buildRuns; buildRuns = []; r.forEach(function (x) { x(); }); }
  /* the build step a block arrives with (0: with the slide) */
  function buildStep(el) { var s = el.closest('.si-step[data-step]'); return s ? +s.getAttribute('data-step') : 0; }
  /* run the animations of slide s that arrive at step k, after the transition when one plays */
  function buildAt(s, k, after) {
    var els = Array.prototype.filter.call(s.querySelectorAll('[data-si-anim]'), function (e) { return buildStep(e) === k; });
    if (!els.length) return;
    var start = function () { if (slides[cur] === s && step === k) els.forEach(buildRun); };
    if (after) after.then(start); else els.forEach(buildRun);
  }
  function textNodes(el) {
    var w = document.createTreeWalker(el, NodeFilter.SHOW_TEXT, null), out = [], n;
    while ((n = w.nextNode())) if (!n.parentNode.closest('script,style')) out.push(n);
    return out;
  }
  /* mark el as running: the teleprompter's copy of the slide takes the markup as it was */
  function buildBegin(el, stop) {
    el._siOrig = el.innerHTML;
    el.setAttribute('data-si-live', '');
    var done = false, fin = function () {
      if (done) return;
      done = true;
      stop();
      el.removeAttribute('data-si-live');
      var i = buildRuns.indexOf(fin); if (i >= 0) buildRuns.splice(i, 1);
    };
    buildRuns.push(fin);
    return fin;
  }
  function frames(fn) { /* fn(ms since start) returns true when done */
    var t0 = 0, alive = true;
    requestAnimationFrame(function tick(t) { if (!alive) return; if (!t0) t0 = t; if (!fn(t - t0)) requestAnimationFrame(tick); });
    return function () { alive = false; };
  }
  function buildRun(el) {
    var kind = el.getAttribute('data-si-anim');
    if (run3d[kind]) { run3d[kind](el); return; }
    var ns, orig, anims = [], halt = null, fin;
    var waapi = function (e, kf, t) { t.fill = 'both'; anims.push(e.animate(kf, t)); };
    var finishAll = function () { Promise.all(anims.map(function (x) { return x.finished; })).then(function () { fin(); }, function () {}); };
    var cancel = function () { if (halt) halt(); anims.forEach(function (x) { try { x.cancel(); } catch (e) {} }); };
    if (kind === 'type' || kind === 'decode') {
      ns = textNodes(el); orig = ns.map(function (n) { return n.data; });
      var len = orig.join('').length; if (!len) return;
      var caret = null;
      fin = buildBegin(el, function () { cancel(); ns.forEach(function (n, i) { n.data = orig[i]; }); if (caret) caret.remove(); });
      if (kind === 'type') {
        var per = Math.min(45, 2600 / len);
        caret = document.createElement('span'); caret.className = 'si-caret'; caret.setAttribute('aria-hidden', 'true');
        halt = frames(function (t) {
          var c = Math.floor(t / per), at = null;
          ns.forEach(function (n, i) { var o = orig[i], take = Math.max(0, Math.min(o.length, c)); n.data = o.slice(0, take); c -= o.length; if (take > 0) at = n; });
          if (at && caret.previousSibling !== at) at.parentNode.insertBefore(caret, at.nextSibling);
          if (t / per >= len) { halt = null; setTimeout(fin, 700); return true; }
        });
      } else {
        var glyph = '!<>-_\\/[]{}=+*^?#ABCDEFGHJKLMNPQRSTUVWXYZ0123456789', total = 40;
        halt = frames(function (t) {
          var f = t / 35, upto = (f / total) * len, pos = 0;
          ns.forEach(function (n, i) {
            n.data = orig[i].replace(/[\s\S]/g, function (ch) { var p = pos++; return /\s/.test(ch) || p < upto ? ch : glyph.charAt(Math.floor(Math.random() * glyph.length)); });
          });
          if (f >= total) { halt = null; fin(); return true; }
        });
      }
      return;
    }
    if (kind === 'words' || kind === 'count') {
      var swaps = [];
      fin = buildBegin(el, function () {
        cancel();
        swaps.forEach(function (sw) { sw.made[0].parentNode.insertBefore(sw.node, sw.made[0]); sw.made.forEach(function (m) { m.remove(); }); });
      });
      var parts = [];
      textNodes(el).forEach(function (n) {
        if (kind === 'count' ? !/\d/.test(n.data) : !n.data.trim()) return;
        var made = [];
        (kind === 'count' ? n.data.split(/(\d)/) : n.data.split(/(\s+)/)).forEach(function (w) {
          if (!w) return;
          var m;
          if (kind === 'words' && !/^\s+$/.test(w)) { m = document.createElement('span'); m.className = 'si-word'; m.textContent = w; parts.push(m); }
          else if (kind === 'count' && /^\d$/.test(w)) {
            m = document.createElement('span'); m.className = 'si-odo'; m.setAttribute('aria-hidden', 'true');
            var ph = document.createElement('span'), col = document.createElement('span'), d = +w;
            ph.textContent = w; m.appendChild(ph);
            for (var j = 0; j <= 10 + d; j++) { var c = document.createElement('span'); c.textContent = j % 10; col.appendChild(c); }
            m.appendChild(col); parts.push(m);
          } else m = document.createTextNode(w);
          n.parentNode.insertBefore(m, n);
          made.push(m);
        });
        n.remove();
        swaps.push({ node: n, made: made });
      });
      if (!parts.length) { fin(); return; }
      var gap = kind === 'words' ? Math.min(90, 1800 / parts.length) : 180;
      parts.forEach(function (m, i) {
        if (kind === 'words') waapi(m, [{ opacity: 0, transform: 'translateY(.5em)', filter: 'blur(.35em)' }, { opacity: 1, transform: 'none', filter: 'blur(0)' }], { duration: 650, delay: i * gap, easing: EASE });
        else {
          var col = m.lastChild, h = col.firstChild.getBoundingClientRect().height, steps = col.children.length - 1;
          waapi(col, [{ transform: 'translateY(0)' }, { transform: 'translateY(' + (-steps * h).toFixed(2) + 'px)' }], { duration: 1200 + i * gap, easing: 'cubic-bezier(.2,.7,.1,1)' });
        }
      });
      finishAll();
      return;
    }
    if (kind === 'highlight') {
      var marks = el.matches('.hl,mark') ? [el] : Array.prototype.slice.call(el.querySelectorAll('.hl,mark'));
      if (!marks.length) return;
      fin = buildBegin(el, cancel);
      marks.forEach(function (m, i) {
        var size = getComputedStyle(m).backgroundSize, h = size.split(' ')[1] || 'auto';
        waapi(m, [{ backgroundSize: '0% ' + h }, { backgroundSize: size }], { duration: 800, delay: 300 + i * 250, easing: 'cubic-bezier(.6,0,.2,1)' });
      });
      finishAll();
      return;
    }
    if (kind === 'draw') {
      var shapes = Array.prototype.filter.call(el.querySelectorAll('svg path,svg line,svg polyline,svg polygon,svg rect,svg circle,svg ellipse'), function (p) {
        return !p.closest('defs,marker,symbol,clipPath,mask,pattern') && p.getTotalLength && p.getTotalLength() > 0;
      });
      var texts = el.querySelectorAll('svg text');
      if (!shapes.length) return;
      var dash = shapes.map(function (p) { return p.style.strokeDasharray; });
      fin = buildBegin(el, function () { cancel(); shapes.forEach(function (p, i) { p.style.strokeDasharray = dash[i]; }); });
      var each2 = Math.min(380, 2400 / shapes.length);
      shapes.forEach(function (p, i) {
        var L = p.getTotalLength(), fo = getComputedStyle(p).fillOpacity;
        p.style.strokeDasharray = L + ' ' + L;
        waapi(p, [{ strokeDashoffset: L, fillOpacity: 0 }, { strokeDashoffset: 0, fillOpacity: 0, offset: 0.7 }, { strokeDashoffset: 0, fillOpacity: fo }], { duration: 600, delay: i * each2, easing: 'cubic-bezier(.6,0,.3,1)' });
      });
      each(texts, function (t, i) { waapi(t, [{ opacity: 0 }, { opacity: 1 }], { duration: 400, delay: 300 + i * each2 * 2 }); });
      finishAll();
      return;
    }
    if (kind === 'confetti') {
      var sl = el.closest('.si-slide'); if (!sl) return;
      var cv = document.createElement('canvas'); cv.className = 'si-fx'; cv.setAttribute('aria-hidden', 'true');
      fin = buildBegin(el, function () { cancel(); cv.remove(); });
      waapi(el, [{ transform: 'scale(.6)', opacity: 0 }, { transform: 'scale(1.08)', opacity: 1, offset: 0.6 }, { transform: 'scale(1)', opacity: 1 }], { duration: 700, easing: EASE });
      sl.appendChild(cv);
      var r = sl.getBoundingClientRect(), g = document.createRange(); g.selectNodeContents(el);
      var re = g.getBoundingClientRect(), dpr = window.devicePixelRatio || 1;
      cv.width = r.width * dpr; cv.height = r.height * dpr;
      var x = cv.getContext('2d'); x.scale(dpr, dpr);
      var cs = getComputedStyle(sl), cols = ['--si-accent', '--si-accent-2', '--si-ok', '--si-warn'].map(function (t) { return cs.getPropertyValue(t).trim() || '#4b5ff0'; });
      var u = r.width / 400, z = Math.sqrt(u), P = [], ox = re.left - r.left + re.width / 2, oy = re.top - r.top + re.height / 2;
      for (var i = 0; i < 140; i++) P.push({ x: ox, y: oy, vx: (Math.random() * 8 - 4) * u, vy: -(Math.random() * 7 + 3) * u, w: (4 + Math.random() * 5) * z, h: (6 + Math.random() * 6) * z, a: Math.random() * 6, va: Math.random() * 0.3 - 0.15, c: cols[i % cols.length] });
      halt = frames(function (t) {
        var s = t / 1000, frame = Math.max(1, Math.round(t / 16.7)) - (P.frame || 0);
        P.frame = (P.frame || 0) + frame;
        x.clearRect(0, 0, r.width, r.height);
        P.forEach(function (p) {
          for (var f = 0; f < frame; f++) { p.vy += 0.18 * u; p.vx *= 0.99; p.x += p.vx; p.y += p.vy; p.a += p.va; }
          x.save(); x.translate(p.x, p.y); x.rotate(p.a); x.globalAlpha = Math.max(0, 1 - s / 2.4); x.fillStyle = p.c;
          x.fillRect(-p.w / 2, -p.h / 2, p.w, p.h * Math.abs(Math.cos(p.a * 2))); x.restore();
        });
        if (s >= 2.4) { halt = null; fin(); return true; }
      });
    }
  }

  /* ---------- steps: reorder: the list in source order, then in its new order ---------- */
  function reorderSet(list, on) {
    var items = Array.prototype.slice.call(list.children), want;
    if (on) {
      var order = (list.getAttribute('data-si-order') || '').split(' ');
      want = order.map(function (o) { return items.filter(function (e) { return e.getAttribute('data-si-i') === o; })[0]; }).filter(Boolean);
    } else want = items.slice().sort(function (p, q) { return +p.getAttribute('data-si-i') - +q.getAttribute('data-si-i'); });
    if (want.length !== items.length || want.every(function (e, i) { return e === items[i]; })) return;
    want.forEach(function (e) { list.appendChild(e); });
  }
  /* FLIP: where each reordered item was, then glide it from there */
  function reorderFirst(s) {
    var at = [];
    each(s.querySelectorAll('[data-si-order]>li'), function (e) { at.push([e, e.getBoundingClientRect().top]); });
    return at;
  }
  function reorderPlay(at) {
    var n = 0;
    at.forEach(function (p) {
      var dy = p[1] - p[0].getBoundingClientRect().top;
      if (Math.abs(dy) > 0.5) p[0].animate([{ transform: 'translateY(' + dy.toFixed(1) + 'px)' }, { transform: 'none' }], { duration: 800, delay: (n++) * 40, easing: 'cubic-bezier(.3,1.3,.4,1)', fill: 'backwards' });
    });
  }

  /* ---------- 3D build animations (CSS 3D, no library): perspective sits on
   * each scene's direct parent. tilt and orbit last while the slide is shown;
   * carousel, explode, flap and orbit have a 3D look at rest too ---------- */
  var slideRuns = [];
  function slideStop() { var r = slideRuns; slideRuns = []; r.forEach(function (x) { x(); }); }
  function slideBegin(stop) {
    var done = false, fin = function () { if (done) return; done = true; stop(); var i = slideRuns.indexOf(fin); if (i >= 0) slideRuns.splice(i, 1); };
    slideRuns.push(fin);
    return fin;
  }
  /* steps: carousel: the cards stand in a ring; the last revealed one is at the front */
  function carouselSet(l) {
    var items = l.children, n = items.length, f = 0, any = false;
    each(items, function (e, i) { e.style.setProperty('--_i', i); if (e.classList.contains('is-visible')) { f = i; any = true; } });
    each(items, function (e, i) { e.classList.toggle('si-front', any && i === f); });
    l.style.setProperty('--_n', Math.max(n, 1));
    l.style.setProperty('--_f', f);
    l.style.setProperty('--_r', (n > 2 ? 4.9 / Math.tan(Math.PI / n) : 6).toFixed(2) + 'em');
  }
  var explodeT = function (k) { return 'rotateX(58deg) rotateZ(-36deg) translateZ(' + (k * 2).toFixed(2) + 'em)'; };
  var orbitT = function (a) { return 'rotateX(68deg) rotateZ(' + a + 'deg) translateX(var(--_R)) rotateZ(' + (-a) + 'deg) rotateX(-68deg)'; };
  /* the rest look: plates stacked apart, the orbit's ring, the split-flap tiles */
  function motion3dInit() {
    each(deck.querySelectorAll('.si-steps-carousel'), carouselSet);
    each(deck.querySelectorAll('.si-explode'), function (l) { each(l.children, function (e, k) { e.style.transform = explodeT(k); }); });
    each(deck.querySelectorAll('.si-orbit'), function (l) {
      var n = l.children.length - 1;
      each(l.children, function (e, k) { if (k) { var a = Math.round((k - 1) * 360 / n); e.setAttribute('data-si-a', a); e.style.transform = orbitT(a); } });
    });
    each(deck.querySelectorAll('[data-si-anim=flap]'), function (el) {
      el.classList.add('si-flap');
      textNodes(el).forEach(function (n) {
        var f = document.createDocumentFragment();
        n.data.split('').forEach(function (ch) {
          var t = document.createElement('span'), sp = /\s/.test(ch);
          t.className = sp ? 'si-flap-s' : 'si-flap-c';
          t.textContent = sp ? ' ' : ch;
          f.appendChild(t);
        });
        n.parentNode.replaceChild(f, n);
      });
    });
  }
  motion3dInit();
  var run3d = {
    /* the slide's blocks become layers at different depths; the camera tilts
       once, then follows the pointer while the slide is shown */
    tilt: function (el) {
      var s = el.closest('.si-slide'), inner = s && s.querySelector('.si-inner');
      if (!inner) return;
      s.classList.add('si-tilt');
      el.classList.add('si-tilt-hero');
      s.style.perspective = (s.clientWidth * 1.2).toFixed(0) + 'px';
      var a = inner.animate([{ transform: 'rotateY(-12deg) rotateX(6deg)' }, { transform: 'rotateY(12deg) rotateX(-5deg)' }, { transform: 'rotateY(0) rotateX(0)' }], { duration: 2600, easing: IN_OUT });
      var move = function (e) {
        var r = s.getBoundingClientRect();
        inner.style.transform = 'rotateY(' + (((e.clientX - r.left) / r.width - 0.5) * 16).toFixed(2) + 'deg) rotateX(' + ((0.5 - (e.clientY - r.top) / r.height) * 10).toFixed(2) + 'deg)';
      };
      deck.addEventListener('pointermove', move);
      slideBegin(function () {
        a.cancel(); deck.removeEventListener('pointermove', move);
        s.classList.remove('si-tilt'); el.classList.remove('si-tilt-hero');
        s.style.removeProperty('perspective'); inner.style.removeProperty('transform');
      });
    },
    /* one plate per item, stacked close, pulling apart */
    explode: function (el) {
      var anims = [];
      each(el.children, function (e, k) {
        anims.push(e.animate([{ transform: explodeT(k * 0.15) }, { transform: explodeT(k * 0.15), offset: 0.15 }, { transform: explodeT(k) }], { duration: 1800, easing: 'cubic-bezier(.3,1.2,.4,1)', fill: 'backwards' }));
      });
      var fin = buildBegin(el, function () { anims.forEach(function (x) { x.cancel(); }); });
      Promise.all(anims.map(function (x) { return x.finished; })).then(function () { fin(); }, function () {});
    },
    /* split-flap tiles flip through the alphabet until the word lands */
    flap: function (el) {
      var cells = Array.prototype.slice.call(el.querySelectorAll('.si-flap-c')), want = cells.map(function (c) { return c.textContent; });
      if (!cells.length) return;
      var AB = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ', D = '0123456789', last = -1, halt = null;
      var fin = buildBegin(el, function () { if (halt) halt(); cells.forEach(function (c, i) { c.textContent = want[i]; }); });
      var flip = function (c, ms) { c.animate([{ transform: 'rotateX(-80deg)' }, { transform: 'none' }], { duration: ms }); };
      halt = frames(function (t) {
        var f = Math.floor(t / 85);
        if (f === last) return false;
        last = f;
        var busy = false;
        cells.forEach(function (c, i) {
          var at = 4 + Math.min(i, 24) * 2;
          if (f < at) { busy = true; var set = /\d/.test(want[i]) ? D : AB; c.textContent = set.charAt(Math.floor(Math.random() * set.length)); flip(c, 80); }
          else if (c.textContent !== want[i]) { c.textContent = want[i]; flip(c, 160); }
        });
        if (!busy) { halt = null; fin(); return true; }
      });
    },
    /* words fly in from far behind the slide, out of a blur */
    flyin: function (el) {
      var swaps = [], parts = [], anims = [];
      var fin = buildBegin(el, function () {
        anims.forEach(function (x) { try { x.cancel(); } catch (e) {} });
        swaps.forEach(function (sw) { sw.made[0].parentNode.insertBefore(sw.node, sw.made[0]); sw.made.forEach(function (m) { m.remove(); }); });
      });
      textNodes(el).forEach(function (n) {
        if (!n.data.trim()) return;
        var made = [];
        n.data.split(/(\s+)/).forEach(function (w) {
          if (!w) return;
          var m = /^\s+$/.test(w) ? document.createTextNode(w) : document.createElement('span');
          if (m.nodeType === 1) { m.className = 'si-word'; m.textContent = w; parts.push(m); }
          n.parentNode.insertBefore(m, n); made.push(m);
        });
        n.remove(); swaps.push({ node: n, made: made });
      });
      var gap = Math.min(160, 2400 / Math.max(parts.length, 1));
      parts.forEach(function (m, i) {
        anims.push(m.animate([{ transform: 'perspective(18em) translateZ(-40em) rotateX(30deg)', opacity: 0, filter: 'blur(.5em)' },
          { transform: 'perspective(18em) translateZ(1em)', opacity: 1, filter: 'blur(0)', offset: 0.85 },
          { transform: 'perspective(18em) translateZ(0)', opacity: 1, filter: 'blur(0)' }], { duration: 1000, delay: i * gap, easing: 'cubic-bezier(.2,.8,.3,1)', fill: 'both' }));
      });
      Promise.all(anims.map(function (x) { return x.finished; })).then(function () { fin(); }, function () {});
    },
    /* the items circle the first one in a tilted ring while the slide is shown */
    orbit: function (el) {
      var anims = [];
      each(el.children, function (e, k) {
        if (!k) return;
        var a = +e.getAttribute('data-si-a');
        anims.push(e.animate([{ transform: orbitT(a) }, { transform: orbitT(a + 360) }], { duration: 5200, easing: IN_OUT, iterations: Infinity }));
      });
      slideBegin(function () { anims.forEach(function (x) { x.cancel(); }); });
    }
  };

