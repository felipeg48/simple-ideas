/* simple-ideas course runtime: the navigator (accordion, tabs, the Menu and
 * Outline panel) and the learner's progress, kept in localStorage per
 * course: ticks in the menu, a ring per module, the overall bar, Continue on
 * the home page, the completion page and its certificate. A lesson is done
 * when read to the end, its video watched to 90 %, its lab's boxes ticked,
 * its quiz passed. No dependencies; works from file://. */
(function () {
  'use strict';
  var root = document.querySelector('.si-course');
  if (!root) return;
  var each = function (list, fn) { Array.prototype.forEach.call(list, fn); };
  var body = document.body;
  var key = 'si-course:' + (body.getAttribute('data-si-guide') || '');
  var page = body.getAttribute('data-si-page') || '';
  var view = root.getAttribute('data-view');
  var st = { done: {}, last: '', ticks: {}, name: '' };
  try {
    var saved = JSON.parse(localStorage.getItem(key) || 'null');
    if (saved && typeof saved === 'object') {
      st.done = saved.done || {}; st.last = saved.last || ''; st.ticks = saved.ticks || {}; st.name = saved.name || '';
    }
  } catch (e) {}
  function save() { try { localStorage.setItem(key, JSON.stringify(st)); } catch (e) {} }
  var nav = root.querySelector('.si-course-nav');
  var mods = nav ? nav.querySelectorAll('.si-course-mod') : [];
  var tabrow = root.querySelector('.si-course-tabrow');
  var cur = nav && nav.querySelector('.si-course-mods a[aria-current="page"]');
  var kind = cur ? cur.getAttribute('data-kind') : '';
  if (view === 'lesson' && page) { st.last = page; save(); }

  /* ---------- the navigator ---------- */
  function setOpen(m, open) {
    m.classList.toggle('is-open', open);
    var b = m.querySelector('.si-course-mod-btn');
    if (b) b.setAttribute('aria-expanded', String(open));
  }
  each(mods, function (m) {
    var b = m.querySelector('.si-course-mod-btn');
    b.addEventListener('click', function () {
      var was = m.classList.contains('is-open');
      var tabs = root.hasAttribute('data-tabs') && !root.hasAttribute('data-menu-open');
      if (tabs || root.hasAttribute('data-accordion')) each(mods, function (x) { if (x !== m) setOpen(x, false); });
      setOpen(m, tabs ? true : !was);
      if (tabs && tabrow) { tabrow.innerHTML = m.querySelector('.si-course-lessons').innerHTML; paint(); }
    });
  });
  function menu(open) {
    if (open) root.setAttribute('data-menu-open', ''); else root.removeAttribute('data-menu-open');
    each(root.querySelectorAll('.si-course-menu,.si-course-outline'), function (b) { b.setAttribute('aria-expanded', String(open)); });
    if (open && cur && cur.scrollIntoView) cur.scrollIntoView({ block: 'nearest' });
  }
  each(root.querySelectorAll('.si-course-menu,.si-course-outline'), function (b) {
    b.addEventListener('click', function () { menu(!root.hasAttribute('data-menu-open')); });
  });
  document.addEventListener('keydown', function (e) { if (e.key === 'Escape' && root.hasAttribute('data-menu-open')) menu(false); });
  if (cur && cur.scrollIntoView && window.matchMedia && window.matchMedia('(width >= 50rem)').matches && !root.hasAttribute('data-tabs')) {
    var box = nav.querySelector('.si-course-nav-in');
    if (box && box.scrollHeight > box.clientHeight) cur.scrollIntoView({ block: 'nearest' });
  }

  /* ---------- progress ---------- */
  var C = 2 * Math.PI * 7;
  function lessons() { return nav ? nav.querySelectorAll('.si-course-mods a[data-page]') : []; }
  function next() { // the lesson to continue with: the first not done from the last one opened
    var ls = Array.prototype.filter.call(lessons(), function (a) { return a.hasAttribute('data-count'); });
    var from = 0;
    ls.forEach(function (a, i) { if (a.getAttribute('data-page') === st.last) from = i; });
    for (var i = 0; i < ls.length; i++) {
      var a = ls[(from + i) % ls.length];
      if (!st.done[a.getAttribute('data-page')]) return a;
    }
    return null;
  }
  function paint() {
    var total = 0, done = 0, todo = [];
    each(root.querySelectorAll('.si-course-nav a[data-page]'), function (a) { a.classList.toggle('is-done', !!st.done[a.getAttribute('data-page')]); });
    each(mods, function (m) {
      var ls = m.querySelectorAll('a[data-count]'), n = 0;
      each(ls, function (a) {
        if (st.done[a.getAttribute('data-page')]) n++;
        else todo.push(a);
      });
      total += ls.length; done += n;
      var fg = m.querySelector('.si-course-ring-fg');
      if (fg) fg.setAttribute('stroke-dasharray', (ls.length ? C * n / ls.length : 0).toFixed(2) + ' ' + C.toFixed(2));
      var pct = ls.length ? Math.round(100 * n / ls.length) : 0;
      var ring = m.querySelector('.si-course-ring');
      if (ring) ring.setAttribute('aria-label', pct + ' % done');
      var card = root.querySelector('.si-course-card[data-module="' + m.getAttribute('data-module') + '"]');
      if (card) {
        card.querySelector('.si-course-card-pct').textContent = pct + ' %';
        card.setAttribute('data-done', ls.length && n === ls.length ? 'all' : String(n));
      }
    });
    var all = total ? Math.round(100 * done / total) : 0, finished = total > 0 && done === total;
    var pctEl = root.querySelector('.si-course-pct'), ov = root.querySelector('.si-course-overall');
    if (pctEl) pctEl.textContent = all + ' %';
    if (ov) ov.style.setProperty('--_pct', all + '%');
    var go = root.querySelector('.si-course-continue');
    if (go) {
      var n = next(), where = root.querySelector('.si-course-cta-where');
      var started = st.last || done > 0;
      if (finished) {
        go.textContent = 'Review the course';
        if (where) where.textContent = 'All ' + total + ' lessons done';
      } else if (n && started) {
        go.textContent = 'Continue: ' + n.querySelector('.si-course-lt').textContent;
        go.setAttribute('href', n.getAttribute('href'));
        var m = n.closest('.si-course-mod'), ls = m.querySelectorAll('a[data-page]');
        var k = Array.prototype.indexOf.call(ls, n);
        if (where) where.textContent = 'Module ' + m.getAttribute('data-module') + ' · lesson ' + (k + 1) + ' of ' + ls.length;
        if (view === 'home' && root.hasAttribute('data-accordion')) each(mods, function (x) { setOpen(x, x === m); });
      }
    }
    var fin = root.querySelector('.si-course-finished');
    if (fin) fin.hidden = !finished;
    if (view === 'complete') complete(finished, done, total, todo);
  }
  function mark(p) {
    if (!p || st.done[p]) return;
    st.done[p] = 1; save(); paint();
  }

  /* ---------- the completion page and its certificate ---------- */
  function complete(finished, done, total, todo) {
    var status = root.querySelector('.si-course-status'), list = root.querySelector('.si-course-todo');
    status.textContent = finished ? 'You finished all ' + total + ' lessons. Well done.'
      : 'You have finished ' + done + ' of ' + total + ' lessons. Still to do:';
    list.innerHTML = '';
    if (!finished) todo.forEach(function (a) {
      var li = document.createElement('li'), l = document.createElement('a');
      l.href = a.getAttribute('href'); l.textContent = a.querySelector('.si-course-lt').textContent;
      li.appendChild(l); list.appendChild(li);
    });
    var form = root.querySelector('.si-course-cert-form'), cert = root.querySelector('.si-course-cert');
    if (!form || !cert) return;
    form.hidden = cert.hidden = !finished;
    var input = form.querySelector('.si-course-name'), name = cert.querySelector('.si-course-cert-name');
    input.value = st.name;
    name.textContent = st.name || 'Your name';
    cert.querySelector('.si-course-cert-date').textContent = new Date().toLocaleDateString(undefined, { year: 'numeric', month: 'long', day: 'numeric' });
    input.oninput = function () { st.name = input.value.trim(); save(); name.textContent = st.name || 'Your name'; };
    form.querySelector('.si-course-print').onclick = function () { window.print(); };
  }

  /* ---------- when the lesson on this page is done ---------- */
  var main = root.querySelector('.si-course-lesson');
  function whenRead() {
    var end = root.querySelector('.si-course-end');
    if (!end) return;
    if (!('IntersectionObserver' in window)) { mark(page); return; }
    var io = new IntersectionObserver(function (es) {
      es.forEach(function (e) { if (e.isIntersecting) { mark(page); io.disconnect(); } });
    });
    io.observe(end);
  }
  if (view === 'lesson' && main) {
    var video = main.querySelector('.si-course-video video');
    var boxes = Array.prototype.filter.call(main.querySelectorAll('input[type="checkbox"]'), function (b) { return !b.closest('.si-quiz'); });
    if (video) {
      var seen = function () { if (video.duration && video.currentTime / video.duration >= 0.9) mark(page); };
      video.addEventListener('timeupdate', seen);
      video.addEventListener('ended', function () { mark(page); });
    } else if (kind === 'quiz' || (main.querySelector('.si-quiz') && kind !== 'lab')) {
      document.addEventListener('si-quiz-pass', function () {
        if (Array.prototype.every.call(main.querySelectorAll('.si-quiz'), function (f) { return f.hasAttribute('data-passed'); })) mark(page);
      });
      if (kind !== 'quiz') whenRead();
    } else if (kind === 'lab' && boxes.length) { // a lab with a task list: done when every box is ticked
      var ticks = st.ticks[page] || [];
      var check = function () {
        st.ticks[page] = boxes.map(function (b) { return b.checked ? 1 : 0; });
        save();
        if (boxes.every(function (b) { return b.checked; })) mark(page);
      };
      boxes.forEach(function (b, i) { b.disabled = false; if (ticks[i]) b.checked = true; b.addEventListener('change', check); });
    } else {
      whenRead();
    }
  }
  paint();
})();
