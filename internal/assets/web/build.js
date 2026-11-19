/* simple-ideas build animations (animate: type, words, highlight, count,
 * draw, decode, confetti, tilt, explode, flap, flyin, orbit) and the list
 * modes (steps: focus, deal, reorder, carousel), shared by decks and
 * documents as window.siBuild. A deck runs them as their build step arrives
 * (deck/03-motion.js); an article, guide or book plays each once when it is
 * scrolled into view (page(), at the end). Every block ends exactly as
 * written; reduced motion, print and no script show that final state. */
(function () {
  'use strict';
  var EASE = 'cubic-bezier(.22,.8,.2,1)', IN_OUT = 'cubic-bezier(.65,0,.35,1)';
  function each(list, f) { Array.prototype.forEach.call(list, f); }
  var buildRuns = [];
  function buildStop() { var r = buildRuns; buildRuns = []; r.forEach(function (x) { x(); }); }
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
      var sl = el.closest('.si-slide'); /* a page: the burst falls over the window */
      var cv = document.createElement('canvas'); cv.className = 'si-fx'; cv.setAttribute('aria-hidden', 'true');
      if (!sl) cv.style.position = 'fixed';
      fin = buildBegin(el, function () { cancel(); cv.remove(); });
      waapi(el, [{ transform: 'scale(.6)', opacity: 0 }, { transform: 'scale(1.08)', opacity: 1, offset: 0.6 }, { transform: 'scale(1)', opacity: 1 }], { duration: 700, easing: EASE });
      (sl || document.body).appendChild(cv);
      var r = sl ? sl.getBoundingClientRect() : { left: 0, top: 0, width: innerWidth, height: innerHeight }, g = document.createRange(); g.selectNodeContents(el);
      var re = g.getBoundingClientRect(), dpr = window.devicePixelRatio || 1;
      cv.width = r.width * dpr; cv.height = r.height * dpr;
      var x = cv.getContext('2d'); x.scale(dpr, dpr);
      var cs = getComputedStyle(sl || el), cols = ['--si-accent', '--si-accent-2', '--si-ok', '--si-warn'].map(function (t) { return cs.getPropertyValue(t).trim() || '#4b5ff0'; });
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
  function motion3dInit(root) {
    each(root.querySelectorAll('.si-steps-carousel'), carouselSet);
    each(root.querySelectorAll('.si-explode'), function (l) { each(l.children, function (e, k) { e.style.transform = explodeT(k); }); });
    each(root.querySelectorAll('.si-orbit'), function (l) {
      var n = l.children.length - 1;
      each(l.children, function (e, k) { if (k) { var a = Math.round((k - 1) * 360 / n); e.setAttribute('data-si-a', a); e.style.transform = orbitT(a); } });
    });
    each(root.querySelectorAll('[data-si-anim=flap]'), function (el) {
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
      var deck = s.closest('.si-deck') || document;
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

  window.siBuild = { run: buildRun, stop: buildStop, slideStop: slideStop, init: motion3dInit, carouselSet: carouselSet,
    reorderSet: reorderSet, reorderFirst: reorderFirst, reorderPlay: reorderPlay };

  /* ---------- a page (article, guide, book): each one plays once when it is
   * scrolled into view; the lists start from their first look ---------- */
  function page() {
    if (document.querySelector('.si-deck')) return;
    var anims = document.querySelectorAll('[data-si-anim]');
    var lists = document.querySelectorAll('.si-steps-focus,.si-steps-deal,.si-steps-reorder,.si-steps-carousel');
    if (!anims.length && !lists.length) return;
    var items = function (l) { return Array.prototype.slice.call(l.children); };
    var mode = function (l) { return (/\bsi-steps-(focus|deal|reorder|carousel)\b/.exec(l.className) || [])[1]; };
    each(lists, function (l) { if (mode(l) === 'carousel') items(l).forEach(function (e) { e.classList.add('si-step', 'is-visible'); }); });
    motion3dInit(document);
    var mm = function (q) { return !!(window.matchMedia && window.matchMedia(q).matches); };
    if (!window.IntersectionObserver || mm('(prefers-reduced-motion: reduce)') || mm('print')) return;
    var timers = [], later = function (f, ms) { timers.push(setTimeout(f, ms)); };
    var start = {
      deal: function (l) { items(l).forEach(function (e) { e.classList.add('si-step'); }); },
      focus: function (l) { items(l).forEach(function (e) { e.classList.add('si-step', 'is-visible'); }); },
      reorder: function (l) { reorderSet(l, false); },
      carousel: function (l) { items(l).forEach(function (e, i) { e.classList.toggle('is-visible', !i); }); carouselSet(l); }
    };
    var play = {
      deal: function (l) { items(l).forEach(function (e, i) { later(function () { e.classList.add('is-visible'); }, i * 260); }); },
      focus: function (l) {
        var its = items(l), set = function (on) { its.forEach(function (x) { x.classList.toggle('is-latest', x === on); }); };
        its.forEach(function (e, i) { later(function () { set(e); }, i * 900); });
        later(function () { set(null); }, its.length * 900);
      },
      reorder: function (l) { later(function () { var at = reorderFirst(l.parentNode); reorderSet(l, true); reorderPlay(at); }, 500); },
      carousel: function (l) { items(l).forEach(function (e, i) { if (i) later(function () { e.classList.add('is-visible'); carouselSet(l); }, i * 1100); }); }
    };
    each(lists, function (l) { start[mode(l)](l); });
    var io = new IntersectionObserver(function (es) {
      es.forEach(function (e) {
        if (!e.isIntersecting) return;
        io.unobserve(e.target);
        var m = mode(e.target);
        if (m) play[m](e.target); else buildRun(e.target);
      });
    }, { threshold: 0.35 });
    each(anims, function (e) { io.observe(e); });
    each(lists, function (l) { io.observe(l); });
    window.addEventListener('beforeprint', function () { /* printed: where every one ends */
      io.disconnect(); timers.forEach(clearTimeout); buildStop(); slideStop();
      each(lists, function (l) {
        items(l).forEach(function (e) { e.classList.add('is-visible'); e.classList.remove('is-latest'); });
        if (mode(l) === 'reorder') reorderSet(l, true);
        if (mode(l) === 'carousel') carouselSet(l);
      });
    });
  }
  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', page); else page();
})();

