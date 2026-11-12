/* simple-ideas document runtime (lab, article, guide): copy buttons, d
 * (light/dark), the table of contents highlight, the guide nav (pages read),
 * Mermaid. No dependencies; works from file://. */
(function () {
  'use strict';
  var root = document.documentElement;

  /* ---------- copy buttons ---------- */
  function copyText(text) {
    if (navigator.clipboard && window.isSecureContext) return navigator.clipboard.writeText(text);
    var ta = document.createElement('textarea'); // file:// has no async clipboard
    ta.value = text; ta.setAttribute('readonly', ''); ta.style.position = 'fixed'; ta.style.opacity = '0';
    document.body.appendChild(ta); ta.select();
    try { document.execCommand('copy'); } finally { ta.remove(); }
    return Promise.resolve();
  }
  document.addEventListener('click', function (e) {
    var t = e.target.closest ? e.target : e.target.parentElement;
    var btn = t && t.closest('.si-copy');
    if (btn) {
      var pre = btn.closest('.si-codeblock').querySelector('pre');
      copyText(pre.textContent).then(function () {
        btn.textContent = 'Copied'; btn.setAttribute('data-done', '');
        setTimeout(function () { btn.textContent = 'Copy'; btn.removeAttribute('data-done'); }, 1600);
      });
      return;
    }
    if (t && t.closest('[data-si-action="mode"]')) toggleMode();
  });

  /* ---------- light / dark (d) ---------- */
  function effectiveDark() {
    var m = root.getAttribute('data-si-mode');
    if (m) return m === 'dark';
    return window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches;
  }
  function toggleMode() {
    var mode = effectiveDark() ? 'light' : 'dark';
    root.setAttribute('data-si-mode', mode);
    try { localStorage.setItem('si-mode', mode); } catch (e) {}
    renderMermaid(true);
  }
  document.addEventListener('keydown', function (e) {
    if (e.ctrlKey || e.metaKey || e.altKey || e.key !== 'd') return;
    var t = e.target;
    if (t && (t.isContentEditable || /^(INPUT|TEXTAREA|SELECT)$/.test(t.tagName))) return;
    toggleMode();
  });

  /* ---------- table of contents: folded on narrow screens, the section in view highlighted ---------- */
  var toc = document.querySelector('.si-doc-toc');
  if (toc) {
    var det = toc.querySelector('details');
    if (det && window.matchMedia && window.matchMedia('(width < 56rem)').matches) det.open = false;
    var links = {};
    Array.prototype.forEach.call(toc.querySelectorAll('a[href^="#"]'), function (a) { links[decodeURIComponent(a.hash.slice(1))] = a; });
    var heads = Array.prototype.filter.call(document.querySelectorAll('.si-doc-body h2[id], .si-doc-body h3[id]'), function (h) { return links[h.id]; });
    var current = null;
    var mark = function () {
      var y = window.innerHeight * 0.25, pick = heads[0];
      for (var i = 0; i < heads.length; i++) { if (heads[i].getBoundingClientRect().top <= y) pick = heads[i]; else break; }
      if (!pick || pick === current) return;
      if (current) links[current.id].removeAttribute('aria-current');
      current = pick;
      links[pick.id].setAttribute('aria-current', 'true');
    };
    var queued = false;
    window.addEventListener('scroll', function () {
      if (queued) return;
      queued = true;
      requestAnimationFrame(function () { queued = false; mark(); });
    }, { passive: true });
    mark();
  }

  /* ---------- code tabs (<!-- tabs -->): one choice switches every group with that
     label on the page and is remembered for the next page; callout badges light
     up with their list item ---------- */
  function tabSelect(group, label) {
    var hit = false;
    Array.prototype.forEach.call(group.querySelectorAll(':scope > .si-tabs-bar [role="tab"]'), function (b) {
      var on = b.getAttribute('data-label') === label; b.setAttribute('aria-selected', on ? 'true' : 'false'); if (on) hit = true;
    });
    if (!hit) return false;
    Array.prototype.forEach.call(group.querySelectorAll(':scope > .si-tab'), function (t) { t.hidden = t.getAttribute('data-label') !== label; });
    return true;
  }
  function tabAll(label) { Array.prototype.forEach.call(document.querySelectorAll('.si-tabs'), function (g) { tabSelect(g, label); }); }
  var savedTab = null;
  try { savedTab = localStorage.getItem('si-tab'); } catch (e) {}
  if (savedTab) tabAll(savedTab);
  document.addEventListener('click', function (e) {
    var b = e.target.closest && e.target.closest('.si-tabs-bar [role="tab"]');
    if (!b) return;
    var label = b.getAttribute('data-label');
    tabAll(label);
    try { localStorage.setItem('si-tab', label); } catch (x) {}
  });
  document.addEventListener('keydown', function (e) {
    var b = e.target.closest && e.target.closest('.si-tabs-bar [role="tab"]');
    if (!b || (e.key !== 'ArrowRight' && e.key !== 'ArrowLeft')) return;
    var all = Array.prototype.slice.call(b.parentNode.querySelectorAll('[role="tab"]'));
    var n = all[(all.indexOf(b) + (e.key === 'ArrowRight' ? 1 : -1) + all.length) % all.length];
    n.focus(); n.click(); e.preventDefault(); e.stopPropagation();
  });
  function coToggle(el, on) {
    var box = el.closest('.si-tab') || el.closest('.si-doc-body') || document.body;
    var n = el.getAttribute('data-co');
    Array.prototype.forEach.call(box.querySelectorAll('.si-co[data-co="' + n + '"], .si-callouts li[data-co="' + n + '"]'), function (x) { x.classList.toggle('is-on', on); });
  }
  document.addEventListener('mouseover', function (e) { var c = e.target.closest && e.target.closest('.si-co, .si-callouts li'); if (c) coToggle(c, true); });
  document.addEventListener('mouseout', function (e) { var c = e.target.closest && e.target.closest('.si-co, .si-callouts li'); if (c) coToggle(c, false); });

  /* ---------- search (/): this page from the DOM, the other pages (and their
     sections, and a book's index terms) from the #si-search data ---------- */
  var finder = document.querySelector('.si-doc-find');
  if (finder) {
    var fInput = finder.querySelector('input'), fList = finder.querySelector('.si-doc-find-list');
    var others = [];
    try { others = JSON.parse((document.getElementById('si-search') || {}).textContent || '[]') || []; } catch (e) { others = []; }
    var local = null, results = [], sel = 0;
    var esc = function (t) { return String(t).replace(/[&<>"]/g, function (c) { return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]; }); };
    function buildLocal() {
      local = [];
      var cur = { id: '', title: document.title, text: '' };
      local.push(cur);
      Array.prototype.forEach.call(document.querySelectorAll('.si-doc-body > *'), function (el) {
        if (/^H[23]$/.test(el.tagName) && el.id) {
          var c = el.cloneNode(true), a = c.querySelector('.si-anchor'); if (a) a.remove();
          Array.prototype.forEach.call(c.querySelectorAll('.si-step-num, .si-sec-num'), function (n) { n.textContent = n.textContent + ' '; });
          cur = { id: el.id, title: c.textContent.replace(/\s+/g, ' ').trim(), text: '' };
          local.push(cur);
        } else if (!el.matches('pre, .si-codeblock, .si-tabs, .si-doc-pager, .si-doc-edit')) {
          cur.text += ' ' + el.textContent;
        }
      });
      local.forEach(function (s) { s.text = s.text.replace(/\s+/g, ' ').trim(); });
    }
    function fuzzy(q, t) {
      t = t.toLowerCase(); var last = -1, score = 0;
      for (var i = 0; i < q.length; i++) {
        if (q.charAt(i) === ' ') continue;
        var j = t.indexOf(q.charAt(i), last + 1);
        if (j < 0) return -1;
        score += 1 + (j === last + 1 ? 2 : 0) + (j === 0 || /[^a-z0-9]/.test(t.charAt(j - 1)) ? 3 : 0);
        last = j;
      }
      return score;
    }
    function snip(t, at, n) {
      var a = Math.max(0, at - 40);
      return (a ? '…' : '') + esc(t.slice(a, at)) + '<mark>' + esc(t.slice(at, at + n)) + '</mark>' + esc(t.slice(at + n, at + n + 70)) + '…';
    }
    function search(q) {
      q = q.trim().toLowerCase(); results = [];
      if (!q) return;
      local.forEach(function (s) {
        var ti = s.title.toLowerCase().indexOf(q), xi = s.text.toLowerCase().indexOf(q);
        if (s.id && ti >= 0) results.push({ pri: 0, href: '#' + s.id, title: s.title, where: 'On this page' });
        else if (xi >= 0) results.push({ pri: 1, href: s.id ? '#' + s.id : '#', title: s.id ? s.title : 'This page', where: 'On this page', snip: snip(s.text, xi, q.length) });
      });
      var outer = [];
      others.forEach(function (o) {
        var sc = fuzzy(q, o.t), tx = o.x ? o.x.toLowerCase().indexOf(q) : -1;
        if (sc < 0 && tx < 0) return;
        var where = o.k === 'page' ? 'Page' : (o.k === 'term' ? 'Index · ' + (o.p || '') : o.p || '');
        outer.push({ pri: 2, score: sc < 0 ? 0 : sc + (o.k === 'page' ? 2 : 0), href: o.h, title: o.t, where: where, snip: tx >= 0 ? snip(o.x, tx, q.length) : '' });
      });
      outer.sort(function (a, b) { return b.score - a.score; });
      results = results.slice(0, 10).concat(outer.slice(0, 14));
    }
    function draw() {
      sel = Math.min(sel, Math.max(0, results.length - 1));
      fList.innerHTML = results.length ? results.map(function (r, i) {
        return '<li role="option" aria-selected="' + (i === sel) + '" data-i="' + i + '"><span class="si-doc-find-t">' + esc(r.title) + '</span><span class="si-doc-find-w">' + esc(r.where) + '</span>' +
          (r.snip ? '<span class="si-doc-find-x">' + r.snip + '</span>' : '') + '</li>';
      }).join('') : (fInput.value.trim() ? '<li class="si-doc-find-none">No matches</li>' : '');
      var on = fList.querySelector('[aria-selected="true"]'); if (on && on.scrollIntoView) on.scrollIntoView({ block: 'nearest' });
    }
    function open(on) {
      finder.hidden = !on;
      if (on) { if (!local) buildLocal(); fInput.value = ''; results = []; sel = 0; draw(); setTimeout(function () { fInput.focus(); }, 0); }
    }
    function go(r) { if (!r) return; open(false); location.href = r.href; }
    fInput.addEventListener('input', function () { sel = 0; search(fInput.value); draw(); });
    fInput.addEventListener('keydown', function (e) {
      e.stopPropagation();
      if (e.key === 'Escape') { e.preventDefault(); open(false); }
      else if (e.key === 'ArrowDown') { e.preventDefault(); sel = Math.min(results.length - 1, sel + 1); draw(); }
      else if (e.key === 'ArrowUp') { e.preventDefault(); sel = Math.max(0, sel - 1); draw(); }
      else if (e.key === 'Enter') { e.preventDefault(); go(results[sel]); }
    });
    fList.addEventListener('click', function (e) { var li = e.target.closest('li[data-i]'); if (li) go(results[+li.getAttribute('data-i')]); });
    finder.addEventListener('click', function (e) { if (e.target === finder) open(false); });
    document.addEventListener('click', function (e) { var t = e.target.closest && e.target.closest('[data-si-action="search"]'); if (t) { e.preventDefault(); open(true); } });
    document.addEventListener('keydown', function (e) {
      if (e.key !== '/' || e.ctrlKey || e.metaKey || e.altKey || !finder.hidden) return;
      var t = e.target; if (t && (t.isContentEditable || /^(INPUT|TEXTAREA|SELECT)$/.test(t.tagName))) return;
      e.preventDefault(); open(true);
    });
  }

  /* ---------- plugins: alert, modal, carousel, events ---------- */
  var today = new Date().toISOString().slice(0, 10);
  Array.prototype.forEach.call(document.querySelectorAll('.si-alert'), function (a) {
    var key = 'si-alert:' + a.getAttribute('data-id'), until = a.getAttribute('data-until');
    var gone = false; try { gone = localStorage.getItem(key) === '1'; } catch (e) {}
    if (gone || (until && today > until)) { a.hidden = true; return; }
    a.querySelector('.si-alert-x').addEventListener('click', function () { a.hidden = true; try { localStorage.setItem(key, '1'); } catch (e) {} });
  });
  function openModal(id) {
    var d = id && document.getElementById(id);
    if (d && d.classList.contains('si-modal') && d.showModal && !d.open) { d.showModal(); return true; }
    return false;
  }
  document.addEventListener('click', function (e) {
    var a = e.target.closest && e.target.closest('a[href^="#"]');
    if (a && openModal(decodeURIComponent(a.getAttribute('href').slice(1)))) { e.preventDefault(); return; }
    var dlg = e.target.classList && e.target.classList.contains('si-modal') ? e.target : null;
    if (dlg) dlg.close(); /* a click on the backdrop */
  });
  if (location.hash) openModal(decodeURIComponent(location.hash.slice(1)));
  Array.prototype.forEach.call(document.querySelectorAll('.si-carousel'), function (car) {
    var track = car.querySelector('.si-carousel-track'), slides = track.children, n = slides.length, i = 0, timer = 0;
    if (n < 2) return;
    var ctl = document.createElement('div'); ctl.className = 'si-carousel-ctl';
    ctl.innerHTML = '<button type="button" aria-label="Previous image">\u2190</button><button type="button" aria-label="Next image">\u2192</button>';
    var dots = document.createElement('div'); dots.className = 'si-carousel-dots';
    for (var k = 0; k < n; k++) dots.appendChild(document.createElement('i'));
    car.appendChild(ctl); car.appendChild(dots); car.tabIndex = 0;
    function go(d) {
      i = (i + d + n) % n; track.style.transform = 'translateX(' + (-100 * i) + '%)';
      Array.prototype.forEach.call(dots.children, function (x, j) { x.classList.toggle('is-on', j === i); });
      Array.prototype.forEach.call(slides, function (x, j) { x.setAttribute('aria-hidden', j === i ? 'false' : 'true'); });
    }
    ctl.children[0].addEventListener('click', function () { go(-1); restart(); });
    ctl.children[1].addEventListener('click', function () { go(1); restart(); });
    car.addEventListener('keydown', function (e) { if (e.key === 'ArrowLeft') { go(-1); restart(); } if (e.key === 'ArrowRight') { go(1); restart(); } });
    var x0 = 0; car.addEventListener('touchstart', function (e) { x0 = e.touches[0].clientX; }, { passive: true });
    car.addEventListener('touchend', function (e) { var dx = e.changedTouches[0].clientX - x0; if (Math.abs(dx) > 40) { go(dx < 0 ? 1 : -1); restart(); } }, { passive: true });
    var auto = parseFloat(car.getAttribute('data-auto')) || 0, still = window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    function restart() { clearInterval(timer); if (auto && !still) timer = setInterval(function () { go(1); }, auto * 1000); }
    go(0); restart();
  });
  var events = document.querySelectorAll('template.si-event');
  if (events.length) {
    var hud = document.createElement('div'); hud.className = 'si-hud'; hud.setAttribute('aria-live', 'polite'); document.body.appendChild(hud);
    Array.prototype.forEach.call(events, function (t) {
      var at = new Date(t.getAttribute('data-at')), left = at - Date.now();
      if (isNaN(at) || left < 0 || left > 2 * 864e5) return; /* shown from two days before until it starts */
      var el = document.createElement('div'); el.className = 'si-toast';
      el.innerHTML = '<span class="si-toast-dot"></span><div><b></b><span class="si-toast-when"></span></div><button type="button" aria-label="Dismiss">\u00d7</button>';
      el.querySelector('b').textContent = t.getAttribute('data-text');
      var href = t.getAttribute('data-href'), label = t.getAttribute('data-label');
      if (href && label) { var a = document.createElement('a'); a.href = href; a.textContent = label; el.children[1].appendChild(a); }
      el.querySelector('button').addEventListener('click', function () { el.remove(); });
      var when = el.querySelector('.si-toast-when');
      (function tick() {
        var ms = at - Date.now(); if (ms <= 0) { el.remove(); return; }
        var h = Math.floor(ms / 36e5), m = Math.floor(ms % 36e5 / 6e4), sec = Math.floor(ms % 6e4 / 1e3);
        when.textContent = 'starts in ' + (h ? h + ' h ' : '') + m + ' min ' + (h ? '' : sec + ' s');
        setTimeout(tick, 1000);
      })();
      hud.appendChild(el);
    });
  }

  /* ---------- site navigator: the menu is open on wide screens, folded behind Menu on narrow ones ---------- */
  var siteMenu = document.querySelector('.si-site-menu');
  if (siteMenu && window.matchMedia) {
    var smq = window.matchMedia('(width >= 56rem)');
    var setMenu = function () { siteMenu.open = smq.matches; };
    setMenu();
    if (smq.addEventListener) smq.addEventListener('change', setMenu);
  }

  /* ---------- guide nav: folded on narrow screens; pages already read get a mark ---------- */
  var nav = document.querySelector('.si-doc-nav');
  if (nav) {
    var nd = nav.querySelector('details');
    if (nd && window.matchMedia && window.matchMedia('(width < 56rem)').matches) nd.open = false;
    var key = 'si-read:' + (document.body.getAttribute('data-si-guide') || ''), page = document.body.getAttribute('data-si-page');
    var read = {};
    try { read = JSON.parse(localStorage.getItem(key) || '{}') || {}; } catch (e) {}
    if (page) { read[page] = 1; try { localStorage.setItem(key, JSON.stringify(read)); } catch (e) {} }
    var pageLinks = nav.querySelectorAll('a[data-page]'), n = 0;
    Array.prototype.forEach.call(pageLinks, function (a) { if (read[a.getAttribute('data-page')]) { a.classList.add('is-read'); n++; } });
    var prog = nav.querySelector('.si-doc-progress');
    if (prog) prog.textContent = n + ' of ' + pageLinks.length + ' read';
    var curLink = nav.querySelector('a[aria-current="page"]');
    if (curLink && curLink.scrollIntoView && nd && nd.open) curLink.scrollIntoView({ block: 'nearest' });
  }

  /* ---------- guide nav: nav: accordion / collapsed fold the sections; the current page's stays open ---------- */
  var gnav = document.querySelector('.si-doc-nav[data-accordion], .si-doc-nav[data-collapsed]');
  if (gnav) {
    var secs = gnav.querySelectorAll('li.si-nav-sec');
    Array.prototype.forEach.call(secs, function (li) {
      var head = li.querySelector(':scope > span');
      if (!head) return;
      head.setAttribute('role', 'button'); head.setAttribute('tabindex', '0');
      var open = !!li.querySelector('a[aria-current="page"]');
      li.classList.toggle('is-open', open); head.setAttribute('aria-expanded', String(open));
      var flip = function () {
        var to = !li.classList.contains('is-open');
        if (gnav.hasAttribute('data-accordion')) Array.prototype.forEach.call(secs, function (o) { if (o !== li) { o.classList.remove('is-open'); var h = o.querySelector(':scope > span'); if (h) h.setAttribute('aria-expanded', 'false'); } });
        li.classList.toggle('is-open', to); head.setAttribute('aria-expanded', String(to));
      };
      head.addEventListener('click', flip);
      head.addEventListener('keydown', function (e) { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); flip(); } });
    });
  }

  /* ---------- quizzes: check the answers, show why, score against the pass mark (deck.Score is the same rule) ---------- */
  Array.prototype.forEach.call(document.querySelectorAll('form.si-quiz'), function (f) {
    var pass = +f.getAttribute('data-pass') || 0;
    var checkBtn = f.querySelector('.si-quiz-check'), again = f.querySelector('.si-quiz-again'), score = f.querySelector('.si-quiz-score');
    var qs = f.querySelectorAll('.si-quiz-q');
    f.addEventListener('submit', function (e) {
      e.preventDefault();
      var missing = Array.prototype.filter.call(qs, function (q) { return !q.querySelector('input:checked'); }).length;
      if (missing) { score.textContent = missing === 1 ? 'Answer the last question first.' : 'Answer every question first (' + missing + ' to go).'; return; }
      var right = 0;
      Array.prototype.forEach.call(qs, function (q) {
        var good = true;
        Array.prototype.forEach.call(q.querySelectorAll('input'), function (i) {
          var ok = i.hasAttribute('data-ok'), l = i.closest('label');
          if (ok !== i.checked) good = false;
          l.classList.toggle('is-ok', ok); l.classList.toggle('is-wrong', i.checked && !ok);
          i.disabled = true;
        });
        q.classList.toggle('is-right', good); q.classList.toggle('is-wrong', !good);
        var why = q.querySelector('.si-quiz-why'); if (why) why.hidden = false;
        if (good) right++;
      });
      var pct = qs.length ? Math.round(100 * right / qs.length) : 0, passed = pct >= pass;
      score.textContent = right + ' of ' + qs.length + ' right, ' + pct + ' %' + (pass ? (passed ? '. Passed.' : '. The pass mark is ' + pass + ' %.') : '.');
      f.classList.toggle('is-passed', passed); f.classList.toggle('is-failed', !passed);
      checkBtn.hidden = true; again.hidden = false;
      if (passed) {
        f.setAttribute('data-passed', '');
        try { document.dispatchEvent(new CustomEvent('si-quiz-pass', { detail: { score: pct } })); } catch (x) {}
      }
    });
    again.addEventListener('click', function () {
      Array.prototype.forEach.call(f.querySelectorAll('input'), function (i) { i.disabled = false; i.checked = false; var l = i.closest('label'); l.classList.remove('is-ok', 'is-wrong'); });
      Array.prototype.forEach.call(qs, function (q) { q.classList.remove('is-right', 'is-wrong'); var why = q.querySelector('.si-quiz-why'); if (why) why.hidden = true; });
      f.classList.remove('is-passed', 'is-failed'); score.textContent = ''; checkBtn.hidden = false; again.hidden = true;
    });
  });

  /* ---------- Mermaid (only when the lab has a mermaid fence and the script loaded) ---------- */
  var mermaidSrc = null;
  function cssVar(name) { return getComputedStyle(root).getPropertyValue(name).trim(); }
  function renderMermaid(again) {
    if (!window.mermaid) return;
    var nodes = document.querySelectorAll('pre.si-mermaid');
    if (!nodes.length) return;
    if (mermaidSrc === null) mermaidSrc = Array.prototype.map.call(nodes, function (x) { return x.textContent; });
    if (again) Array.prototype.forEach.call(nodes, function (x, i) { x.removeAttribute('data-processed'); x.textContent = mermaidSrc[i]; });
    var v = { fg: cssVar('--si-fg'), bg: cssVar('--si-bg'), acc: cssVar('--si-accent'), muted: cssVar('--si-muted'),
      s1: cssVar('--si-surface'), s2: cssVar('--si-surface-2'), line: cssVar('--si-line') };
    try {
      window.mermaid.initialize({ startOnLoad: false, theme: 'base', fontFamily: cssVar('--si-font-body'), themeVariables: {
        primaryColor: v.s1, primaryTextColor: v.fg, primaryBorderColor: v.acc, secondaryColor: v.s2, tertiaryColor: v.s2,
        lineColor: v.muted, textColor: v.fg, mainBkg: v.s1, nodeBorder: v.acc, clusterBkg: v.s2, clusterBorder: v.line,
        titleColor: v.fg, edgeLabelBackground: v.bg, actorBkg: v.s1, actorBorder: v.acc, actorTextColor: v.fg,
        signalColor: v.fg, signalTextColor: v.fg, noteBkgColor: v.s2, noteTextColor: v.fg, noteBorderColor: v.line } });
      window.mermaid.run({ nodes: nodes });
    } catch (e) { if (window.console) console.warn('mermaid', e); }
  }
  renderMermaid(false);
  if (window.matchMedia) {
    var mq = window.matchMedia('(prefers-color-scheme: dark)');
    if (mq.addEventListener) mq.addEventListener('change', function () { if (!root.getAttribute('data-si-mode')) renderMermaid(true); });
  }
})();
