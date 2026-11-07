  /* ---------- theme mode (d) ---------- */
  function effectiveDark() {
    var m = root.getAttribute('data-si-mode');
    if (m) return m === 'dark';
    return window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches;
  }
  function toggleMode() {
    var mode = effectiveDark() ? 'light' : 'dark';
    root.setAttribute('data-si-mode', mode);
    save('si-mode', mode);
    renderMermaid(true);
    announce();
  }

  /* ---------- break timer HUD (t): a full-screen countdown to project during a break ---------- */
  var hud = document.querySelector('.si-hud'), hudTime = hud && hud.querySelector('.si-hud-time');
  var hudDial = hud && hud.querySelector('.si-hud-dial');
  var tcfg = cfg.timer || {};
  var hudUnit = tcfg.unit === 'seconds' ? 'seconds' : 'minutes';
  var hudMul = hudUnit === 'seconds' ? 1000 : 60000;
  var hudMins = (tcfg.time && tcfg.time.length) ? tcfg.time : [5, 10, 15, 20, 30, 45, 60];
  var hudAnims = (tcfg.animations && tcfg.animations.length) ? tcfg.animations : [];
  var animKinds = tcfg.animKinds || {};
  var savedAnim = load('si-anim');
  var animCurrent = savedAnim === 'none' ? 'none' : (hudAnims.indexOf(savedAnim) >= 0 ? savedAnim : (hudAnims[0] || ''));
  var savedFace = load('si-face');
  var faceCurrent = (savedFace === 'analog' || savedFace === 'digital') ? savedFace : (tcfg.face === 'analog' ? 'analog' : 'digital');
  var hudEnd = 0, hudLeft = 0, hudTotal = 0, hudPaused = false, hudTimer = 0;
  function hudPad(x) { return (x < 10 ? '0' : '') + x; }
  function hudFmt(ms) { if (ms < 0) ms = 0; var s = Math.round(ms / 1000); return Math.floor(s / 60) + ':' + hudPad(s % 60); }
  var dialArc = hudDial && hudDial.querySelectorAll('circle')[1];
  var dialHand = hudDial && hudDial.querySelector('line');
  var dialText = hudDial && hudDial.querySelector('text');
  var DIAL_C = 2 * Math.PI * 44; // circumference of the r=44 ring
  if (dialArc) dialArc.style.strokeDasharray = DIAL_C;
  function applyFace() {
    var analog = faceCurrent === 'analog' && !!hudDial;
    if (hudTime) hudTime.hidden = analog;
    // hudDial is an <svg>: the `hidden` IDL property is not reflected on SVG
    // elements, so toggle the content attribute directly.
    if (hudDial) { if (analog) hudDial.removeAttribute('hidden'); else hudDial.setAttribute('hidden', ''); }
  }
  function dialUpdate() {
    if (!dialArc) return;
    var elapsed = hudTotal > 0 ? Math.max(0, Math.min(1, 1 - hudLeft / hudTotal)) : 0;
    dialArc.style.strokeDashoffset = DIAL_C * elapsed; // ring empties as time runs down
    if (dialHand) dialHand.setAttribute('transform', 'rotate(' + (elapsed * 360) + ' 50 50)');
    if (dialText) dialText.textContent = hudFmt(hudLeft);
  }
  function hudShow(on) {
    if (!hud) return;
    if (on === undefined) on = !hud.classList.contains('is-open');
    hud.classList.toggle('is-open', on);
    hud.setAttribute('aria-hidden', on ? 'false' : 'true');
    if (on) { applyFace(); dialUpdate(); hudApplyChrome(); startAnim(); } // runs continuously while the HUD is open, independent of the countdown
    else { clearInterval(hudTimer); hudTimer = 0; hud.classList.remove('is-over', 'is-focus'); stopAnim(); }
  }
  // is-focus hides the presets/toggles/hint so only the clock (and the animation)
  // show. It is tied to the countdown: focus while running, chrome back while
  // paused or not started — so Space (pause) reveals the controls, resume hides them.
  function hudApplyChrome() { if (hud) hud.classList.toggle('is-focus', !!hudTimer && !hudPaused); }
  function hudStart(m) {
    hudTotal = m * hudMul;
    hudEnd = Date.now() + hudTotal; hudLeft = hudTotal; hudPaused = false; hud.classList.remove('is-over');
    clearInterval(hudTimer); hudTimer = setInterval(hudTick, 250); hudTick();
    hudApplyChrome();
    if (!animLayer) startAnim(); // keep the already-running loop; only start if not already
  }

  /* ---------- timer animation layer: SLIDE | ANIMATION | TIMER (behind the HUD) ---------- */
  var animLayer = null, animStop = null;
  function animTokens() {
    function v(n) { return getComputedStyle(root).getPropertyValue(n).trim(); }
    return {
      accent: v('--si-accent'), accent2: v('--si-accent-2'), fg: v('--si-fg'), bg: v('--si-bg'),
      muted: v('--si-muted'), ok: v('--si-ok'), warn: v('--si-warn'), crit: v('--si-crit'),
      surface: v('--si-surface'), line: v('--si-line')
    };
  }
  function sizeAnimCanvas(cv) {
    var dpr = window.devicePixelRatio || 1;
    cv.width = Math.max(1, Math.round(window.innerWidth * dpr));
    cv.height = Math.max(1, Math.round(window.innerHeight * dpr));
  }
  function stopAnim() {
    if (animStop) { try { animStop(); } catch (e) {} animStop = null; }
    if (animLayer) { animLayer.remove(); animLayer = null; }
    if (hud) hud.classList.remove('has-anim');
  }
  function startAnim() {
    stopAnim();
    if (!animCurrent || animCurrent === 'none') return;
    animLayer = document.createElement('div');
    animLayer.className = 'si-anim';
    animLayer.setAttribute('aria-hidden', 'true');
    if (animKinds[animCurrent] === 'svg') {
      var tpl = document.getElementById('si-anim-svg-' + animCurrent);
      if (tpl) animLayer.innerHTML = tpl.innerHTML;
      deck.appendChild(animLayer);
    } else if (window.__siAnims && typeof window.__siAnims[animCurrent] === 'function') {
      var cv = document.createElement('canvas');
      animLayer.appendChild(cv); deck.appendChild(animLayer);
      sizeAnimCanvas(cv);
      animLayer._cv = cv;
      var api = {
        tokens: animTokens(), reduced: reduced,
        fraction: function () { return hudTotal > 0 ? Math.max(0, Math.min(1, 1 - hudLeft / hudTotal)) : 0; },
        remaining: function () { return Math.max(0, hudLeft / 1000); }
      };
      try { animStop = window.__siAnims[animCurrent](cv, api); } catch (e) { if (window.console) console.warn('si-anim', e); }
    }
    if (hud) hud.classList.add('has-anim');
  }
  window.addEventListener('resize', function () { if (animLayer && animLayer._cv) sizeAnimCanvas(animLayer._cv); });
  function hudPause() { if (!hudTimer) return; hudPaused = !hudPaused; if (!hudPaused) hudEnd = Date.now() + hudLeft; hudApplyChrome(); }
  function hudTick() {
    if (!hudPaused) hudLeft = hudEnd - Date.now();
    if (hudTime) hudTime.textContent = hudFmt(hudLeft);
    dialUpdate();
    if (hudLeft <= 0) { hudLeft = 0; if (hudTime) hudTime.textContent = '0:00'; dialUpdate(); hud.classList.add('is-over'); clearInterval(hudTimer); hudTimer = 0; hudApplyChrome(); }
  }

  /* ---------- find slide (/) : jump by number or search titles + body ---------- */
  var find = document.querySelector('.si-find');
  var findInput = find && find.querySelector('.si-find-input');
  var findList = find && find.querySelector('.si-find-list');
  var findIndex = null, findResults = [], findSel = 0;
  function escHTML(s) { return s.replace(/[&<>]/g, function (c) { return { '&': '&amp;', '<': '&lt;', '>': '&gt;' }[c]; }); }
  function slideTitle(i) { return slides[i].getAttribute('data-title') || ('Slide ' + (i + 1)); }
  function buildIndex() {
    findIndex = [];
    for (var i = 0; i < n; i++) {
      var inner = slides[i].querySelector('.si-inner');
      var text = (inner ? inner.textContent : slides[i].textContent) || '';
      findIndex.push(text.replace(/\s+/g, ' ').trim());
    }
  }
  function snippet(text, at, len) {
    var s = Math.max(0, at - 30);
    return (s > 0 ? '…' : '') + escHTML(text.slice(s, at)) +
      '<mark>' + escHTML(text.slice(at, at + len)) + '</mark>' +
      escHTML(text.slice(at + len, at + len + 46)) + (at + len + 46 < text.length ? '…' : '');
  }
  function findSearch(q) {
    q = q.trim(); findResults = [];
    if (!q) { for (var i = 0; i < n; i++) findResults.push({ i: i }); return; }
    if (/^\d+$/.test(q)) {
      var num = Math.max(1, Math.min(n, parseInt(q, 10)));
      findResults.push({ i: num - 1, jump: num });
      return;
    }
    var ql = q.toLowerCase(), hits = [];
    for (var j = 0; j < n; j++) {
      var inT = slideTitle(j).toLowerCase().indexOf(ql);
      if (inT >= 0) { hits.push({ i: j, pri: 0, snip: snippet(slideTitle(j), inT, q.length) }); continue; }
      var body = findIndex[j], inB = body.toLowerCase().indexOf(ql);
      if (inB >= 0) hits.push({ i: j, pri: 1, snip: snippet(body, inB, q.length) });
    }
    hits.sort(function (a, b) { return a.pri - b.pri || a.i - b.i; });
    findResults = hits.slice(0, 12).concat(pageHits(ql));
  }
  /* other pages of the build (SI_CONFIG.pages: index, decks, labs, PDFs, other
     .html, si.conf links), matched fuzzily: "idx" and "inde" find index.html */
  var pages = (cfg.pages || []).filter(function (p) { return p.href !== data.name + '.html'; });
  function fuzzy(q, s) {
    var t = s.toLowerCase(), at = [], score = 0, last = -1;
    for (var i = 0, j = 0; i < q.length; i++) {
      if (q.charAt(i) === ' ') continue;
      j = t.indexOf(q.charAt(i), last + 1);
      if (j < 0) return null;
      score += 1 + (j === last + 1 ? 2 : 0) + (j === 0 || /[^a-z0-9]/.test(t.charAt(j - 1)) ? 3 : 0) - Math.min(2, (j - last - 1) * 0.1);
      at.push(j); last = j;
    }
    return { score: score, at: at };
  }
  function marked(s, at) {
    var out = '';
    for (var i = 0; i < s.length; i++) out += at.indexOf(i) >= 0 ? '<mark>' + escHTML(s.charAt(i)) + '</mark>' : escHTML(s.charAt(i));
    return out;
  }
  function pageHits(ql) {
    var hits = [];
    pages.forEach(function (p) {
      var name = p.href.replace(/^.*\//, '').replace(/\.html$/, '');
      var a = fuzzy(ql, p.title), b = fuzzy(ql, name);
      if (!a && !b) return;
      var byName = b && (!a || b.score > a.score);
      hits.push({ page: p, score: byName ? b.score : a.score, title: byName ? escHTML(p.title) : marked(p.title, a.at), href: byName ? marked(p.href, b.at.map(function (k) { return k + p.href.length - p.href.replace(/^.*\//, '').length; })) : escHTML(p.href) });
    });
    hits.sort(function (x, y) { return y.score - x.score; });
    return hits.slice(0, 6);
  }
  function findRender() {
    if (!findList) return;
    if (!findResults.length) { findList.innerHTML = '<li aria-disabled="true">No matches</li>'; return; }
    findList.innerHTML = '';
    findResults.forEach(function (r, idx) {
      var li = document.createElement('li');
      li.setAttribute('aria-selected', idx === findSel ? 'true' : 'false');
      if (r.page) {
        li.setAttribute('data-page', r.page.kind);
        li.innerHTML = '<b>\u2197</b> <span>' + r.title + '</span> <small>' + r.page.kind + ' \u00b7 ' + r.href + '</small>';
      } else {
        li.setAttribute('data-i', r.i);
        var extra = r.jump ? '<small>go to slide ' + r.jump + '</small>' : (r.snip ? '<small>' + r.snip + '</small>' : '');
        li.innerHTML = '<b>' + (r.i + 1) + '</b> <span>' + escHTML(slideTitle(r.i)) + '</span> ' + extra;
      }
      li.addEventListener('click', function () { findPick(r); });
      li.addEventListener('mousemove', function () { if (findSel !== idx) { findSel = idx; markSel(); } });
      findList.appendChild(li);
    });
  }
  function markSel() {
    if (!findList) return;
    each(findList.children, function (li, idx) { li.setAttribute('aria-selected', idx === findSel ? 'true' : 'false'); });
    var sel = findList.children[findSel];
    if (sel && sel.scrollIntoView) sel.scrollIntoView({ block: 'nearest' });
  }
  function findMove(d) {
    if (!findResults.length) return;
    findSel = Math.max(0, Math.min(findResults.length - 1, findSel + d));
    markSel();
  }
  function findGo(i) { findShow(false); go(i, 0); }
  function findPick(r) { if (r.page) { findShow(false); location.href = r.page.href; } else findGo(r.i); }
  function findShow(on) {
    if (!find || !feat.find) return;
    if (on === undefined) on = !find.classList.contains('is-open');
    find.classList.toggle('is-open', on);
    find.setAttribute('aria-hidden', on ? 'false' : 'true');
    if (on) {
      if (!findIndex) buildIndex();
      findInput.value = ''; findSearch(''); findSel = 0; findRender();
      setTimeout(function () { findInput.focus(); }, 0);
    } else if (findInput) { findInput.blur(); }
  }
  if (findInput) {
    findInput.addEventListener('input', function () { findSearch(findInput.value); findSel = 0; findRender(); });
    findInput.addEventListener('keydown', function (e) {
      // stopPropagation so the key never reaches the global nav handler — notably
      // Enter, which (after findGo blurs the input) would otherwise advance a slide.
      if (e.key === 'Escape') { e.preventDefault(); e.stopPropagation(); findShow(false); }
      else if (e.key === 'ArrowDown') { e.preventDefault(); e.stopPropagation(); findMove(1); }
      else if (e.key === 'ArrowUp') { e.preventDefault(); e.stopPropagation(); findMove(-1); }
      else if (e.key === 'Enter') { e.preventDefault(); e.stopPropagation(); if (findResults.length) findPick(findResults[findSel]); }
    });
    find.addEventListener('click', function (e) { if (e.target === find) findShow(false); });
  }

