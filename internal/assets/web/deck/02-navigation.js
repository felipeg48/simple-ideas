  /* ---------- morph: what both slides show glides from its old box to its
   * new one (FLIP with the Web Animations API), the rest cross-fades ---------- */
  var morphAnims = [];
  function morphStop() {
    morphAnims.forEach(function (a) { try { a.cancel(); } catch (e) {} });
    morphAnims = [];
    each(deck.querySelectorAll('.si-morph-from'), function (s) { s.classList.remove('si-morph-from'); });
  }
  /* the key an element is matched by, in order: its name (<!-- id: -->), a
     heading's text, a picture's file, a text block's text */
  var morphSel = '[data-si-morph],h1,h2,h3,h4,img,.si-figure[data-src],p,li,blockquote,pre,table';
  function morphKey(e) {
    var t = (e.textContent || '').replace(/\s+/g, ' ').trim();
    if (e.hasAttribute('data-si-morph')) return '0' + e.getAttribute('data-si-morph');
    if (/^H[1-4]$/.test(e.tagName)) return t ? '1' + t : '';
    if (e.tagName === 'IMG') return '2' + e.getAttribute('src');
    if (e.hasAttribute('data-src') && e.classList.contains('si-figure')) return '2' + e.getAttribute('data-src');
    return t ? '3' + t : '';
  }
  /* a picture or block scales to its new box; words keep their shape: they
     scale by font size from where the text itself starts */
  function morphBox(e) {
    var r = e.getBoundingClientRect(), box = r, cs = getComputedStyle(e);
    if (!/^(IMG|PRE|TABLE|svg)$/.test(e.tagName) && !e.classList.contains('si-figure') && !e.querySelector('img,svg,pre,table')) {
      var g = document.createRange(); g.selectNodeContents(e);
      var t = g.getBoundingClientRect();
      if (t.width && t.height) box = t;
    }
    return { e: e, r: r, box: box, text: box !== r, fs: parseFloat(cs.fontSize), c: cs.color, bg: cs.backgroundColor, o: cs.opacity };
  }
  function morphList(s) {
    var out = [];
    each(s.querySelectorAll(morphSel), function (e) {
      if (e.closest('.si-step:not(.is-visible)')) return;
      var r = e.getBoundingClientRect(), k = morphKey(e);
      if (k && r.width && r.height) { var m = morphBox(e); m.k = k; out.push(m); }
    });
    return out;
  }
  function morphTime(s) {
    var cs = getComputedStyle(s);
    var v = cs.getPropertyValue('--si-morph-duration').trim();
    var ms = /ms$/.test(v) ? parseFloat(v) : parseFloat(v) * 1000;
    return { duration: ms >= 0 ? ms : 700, easing: cs.getPropertyValue('--si-morph-ease').trim() || 'ease' };
  }
  /* the blocks of a slide that hold no matched element: they fade */
  function morphRest(s, matched) {
    var out = [];
    (function walk(el) {
      each(el.children, function (c) {
        if (matched.indexOf(c) >= 0) return;
        if (matched.some(function (m) { return c.contains(m); })) walk(c); else out.push(c);
      });
    })(s);
    return out;
  }
  function morphPlay(first, from, to) {
    var t = morphTime(to), anim = function (el, kf) { morphAnims.push(el.animate(kf, t)); };
    var last = morphList(to), pairs = [], used = [];
    var nested = function (e, list) { return list.some(function (x) { return x === e || x.contains(e) || e.contains(x); }); };
    ['0', '1', '2', '3'].forEach(function (kind) {
      first.forEach(function (a) {
        if (a.k.charAt(0) !== kind || nested(a.e, pairs.map(function (p) { return p.a.e; }))) return;
        for (var j = 0; j < last.length; j++) {
          var b = last[j];
          if (b.k === a.k && !nested(b.e, used)) { pairs.push({ a: a, b: b }); used.push(b.e); return; }
        }
      });
    });
    from.classList.add('si-morph-from');
    pairs.forEach(function (p) {
      var a = p.a, z = morphBox(p.b.e), b = z.e, cs = getComputedStyle(b);
      var end = cs.transform === 'none' ? '' : ' ' + cs.transform;
      var text = a.text && z.text, sx = text ? a.fs / z.fs : a.box.width / z.box.width, sy = text ? sx : a.box.height / z.box.height;
      var origin = (z.box.left - z.r.left) + 'px ' + (z.box.top - z.r.top) + 'px';
      var k0 = { transformOrigin: origin, transform: 'translate(' + (a.box.left - z.box.left) + 'px,' + (a.box.top - z.box.top) + 'px) scale(' + sx + ',' + sy + ')' + end };
      var k1 = { transformOrigin: origin, transform: end.trim() || 'none' };
      if (a.c !== z.c) { k0.color = a.c; k1.color = z.c; }
      if (a.bg !== z.bg) { k0.backgroundColor = a.bg; k1.backgroundColor = z.bg; }
      if (a.o !== z.o) { k0.opacity = a.o; k1.opacity = z.o; }
      anim(b, [k0, k1]);
      anim(a.e, [{ opacity: 0 }, { opacity: 0 }]);
    });
    morphRest(to, used).forEach(function (e) { anim(e, [{ opacity: 0 }, { opacity: getComputedStyle(e).opacity }]); });
    morphRest(from, pairs.map(function (p) { return p.a.e; })).forEach(function (e) { anim(e, [{ opacity: getComputedStyle(e).opacity }, { opacity: 0 }]); });
    var mine = morphAnims;
    Promise.all(mine.map(function (a) { return a.finished; })).then(function () { if (mine === morphAnims) morphStop(); }, function () {});
    return pairs.length;
  }

  /* ---------- navigation ---------- */
  function go(i, k, instant) {
    i = Math.max(0, Math.min(n - 1, i));
    var next = slides[i], prev = cur >= 0 ? slides[cur] : null;
    var max = maxStep(next);
    if (k === 'end') k = max;
    k = Math.max(0, Math.min(max, k || 0));
    if (i !== cur || k !== step) apTouch(); /* a move auto-play did not make pauses it */
    if (i !== cur) {
      clearStrokes();
      var tx = next.getAttribute('data-transition') || data.transition || 'fade';
      var key = canvasKey(next);
      var carry = !instant && prev && Math.abs(i - cur) === 1 && key !== '' && key === canvasKey(prev);
      /* morph: arriving at a morph slide, or going back from one, between neighbours */
      if (prev && i < cur && prev.getAttribute('data-transition') === 'morph') tx = 'morph';
      morphStop();
      var first = tx === 'morph' && prev && !carry && !instant && Math.abs(i - cur) === 1 && !reduceMotion() &&
        !deck.classList.contains('si-overview') && !(window.matchMedia && window.matchMedia('print').matches) ? morphList(prev) : null;
      if (tx === 'morph' && !first) tx = 'fade';
      if (instant) tx = 'none'; else if (carry) tx = 'canvas';
      deck.setAttribute('data-si-tx', tx);
      if (carry) {
        /* one canvas across slides: start the incoming copy in the outgoing state */
        var from = prev.querySelector('svg.si-canvas'), to = next.querySelector('svg.si-canvas');
        if (from && to) to.setAttribute('data-state', from.getAttribute('data-state') || '');
      }
      slides.forEach(function (s, j) {
        s.classList.toggle('si-past', j < i);
        s.classList.toggle('si-future', j > i);
        s.classList.toggle('si-current', j === i);
        s.setAttribute('aria-hidden', j === i ? 'false' : 'true');
      });
      cur = i;
      applySteps(next, k);
      if (carry) void next.getBoundingClientRect(); /* flush styles so the state change below transitions */
      applyCanvas(next, k);
      applyCounts(next);
      fit(next);
      if (first) morphPlay(first, prev, next);
    } else {
      apply(next, k);
    }
    step = k;
    update();
  }
  function forward() {
    if (role === 'audience') return; /* the room follows; only the presenter moves the talk */
    if (role === 'draw') { post({ type: 'cmd', cmd: 'next' }); return; }
    if (step < maxStep(slides[cur])) go(cur, step + 1);
    else if (cur < n - 1) go(cur + 1, 0);
  }
  function backward() {
    if (role === 'audience') return;
    if (role === 'draw') { post({ type: 'cmd', cmd: 'prev' }); return; }
    if (step > 0) go(cur, step - 1);
    else if (cur > 0) go(cur - 1, 'end');
  }
  function update() {
    var max = maxStep(slides[cur]);
    if (counter) counter.textContent = (cur + 1) + ' / ' + n + (max ? '  ·  ' + step + '/' + max : '');
    if (progress) progress.style.width = ((cur + 1) / n * 100) + '%';
    var h = '#' + (cur + 1) + (step ? '.' + step : '');
    if (location.hash !== h) { try { history.replaceState(null, '', location.pathname + location.search + h); } catch (e) {} } /* an embedded deck (about:srcdoc) has no URL to update */
    document.title = (slides[cur].getAttribute('data-title') || data.title) + (cur ? ' · ' + data.title : '');
    syncVideos();
    announce();
  }
  function fromHash(instant) {
    var m = /^#(\d+)(?:\.(\d+))?$/.exec(location.hash);
    go(m ? +m[1] - 1 : 0, m && m[2] ? +m[2] : 0, instant);
  }
  window.addEventListener('hashchange', function () {
    var m = /^#(\d+)(?:\.(\d+))?$/.exec(location.hash);
    if (m && (+m[1] - 1 !== cur || +(m[2] || 0) !== step)) fromHash(false);
  });

  /* ---------- overview ---------- */
  /* The overview lays slides out in one column per topic (the section a slide
   * belongs to, data-topic). Arrow keys move a selection cursor; Enter opens it. */
  var ovCols = [];   // columns: each an array of slide indices, in deck order
  var ovPos = {};    // slide index -> [columnIndex, rowIndex]
  var ovSel = 0;     // the keyboard-selected slide index while in overview
  var ovHeads = [];  // the topic header elements (removed on close)

  function buildOverview() {
    ovCols = []; ovPos = {}; ovHeads = [];
    var colByTopic = {};
    slides.forEach(function (s) { colByTopic[s.getAttribute('data-topic') || '0'] = true; });
    // One topic (no sections, no modules): a plain grid, row by row, as wide as
    // the window allows; each grid column is still a column for the arrow keys.
    var flat = Object.keys(colByTopic).length === 1;
    var perRow = flat ? Math.max(1, Math.floor(deck.clientWidth / (14 * parseFloat(getComputedStyle(root).fontSize)))) : 0;
    colByTopic = {};
    slides.forEach(function (s, i) {
      var t = flat ? String(i % perRow) : (s.getAttribute('data-topic') || '0');
      var ci = colByTopic[t];
      if (ci === undefined) { ci = ovCols.length; colByTopic[t] = ci; ovCols.push([]); }
      var ri = ovCols[ci].length;
      ovCols[ci].push(i);
      ovPos[i] = [ci, ri];
      s.style.gridColumn = (ci + 1);
      s.style.gridRow = (ri + 2); // row 1 holds the topic header
    });
    deck.style.setProperty('--si-ov-cols', ovCols.length);
    var heads = flat ? [[0, '1 / -1', data.title || '']] : ovCols.map(function (col, ci) {
      var first = slides[col[0]];
      var label = first.getAttribute('data-topic') === '0' ? (data.title || 'Intro') : first.getAttribute('data-title');
      return [ci, ci + 1, label];
    });
    heads.forEach(function (hd) {
      var h = document.createElement('div');
      h.className = 'si-ov-head';
      h.textContent = hd[2] || '';
      h.style.gridColumn = hd[1];
      h.style.gridRow = 1;
      deck.appendChild(h);
      ovHeads.push(h);
    });
  }
  function clearOverview() {
    slides.forEach(function (s) { s.style.removeProperty('grid-column'); s.style.removeProperty('grid-row'); s.classList.remove('si-sel'); });
    ovHeads.forEach(function (h) { h.remove(); });
    ovHeads = []; deck.style.removeProperty('--si-ov-cols');
  }
  function setSel(i) {
    if (i == null || i < 0 || i >= slides.length) return;
    slides[ovSel].classList.remove('si-sel');
    ovSel = i;
    slides[ovSel].classList.add('si-sel');
    slides[ovSel].scrollIntoView({ block: 'nearest', inline: 'nearest' });
  }
  function moveSel(dx, dy) {
    var p = ovPos[ovSel]; if (!p) return;
    var c = p[0], r = p[1];
    if (dy) { r += dy; var col = ovCols[c]; if (r < 0) r = 0; if (r >= col.length) r = col.length - 1; setSel(col[r]); return; }
    if (dx) { c += dx; if (c < 0) c = 0; if (c >= ovCols.length) c = ovCols.length - 1; var cc = ovCols[c]; if (r >= cc.length) r = cc.length - 1; setSel(cc[r]); }
  }
  function overview(on) {
    if (on === undefined) on = !deck.classList.contains('si-overview');
    deck.classList.toggle('si-overview', on);
    if (on) {
      slides.forEach(function (s, j) { if (j !== cur) apply(s, maxStep(s)); });
      buildOverview();
      ovSel = cur; slides[ovSel].classList.add('si-sel');
      slides[ovSel].scrollIntoView({ block: 'center', inline: 'center' });
    } else {
      clearOverview();
      slides.forEach(function (s) { var inner = s.querySelector('.si-inner'); if (inner) { inner.style.removeProperty('--_fit'); inner.style.removeProperty('--_figfit'); } });
      go(cur, step, true);
    }
  }
  deck.addEventListener('click', function (e) {
    if (!deck.classList.contains('si-overview')) return;
    var s = e.target.closest('.si-slide');
    if (!s) return;
    var i = slides.indexOf(s);
    overview(false);
    go(i, 0, true);
  });

