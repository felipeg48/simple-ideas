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
