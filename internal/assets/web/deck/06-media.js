  /* ---------- backgrounds: auto overlay from image brightness; videos play on the current slide only ---------- */
  var reduced = window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  function syncVideos() {
    each(document.querySelectorAll('video.si-bg-media'), function (v) {
      var on = !reduced && v.closest('.si-slide') === slides[cur] && !deck.classList.contains('si-overview');
      if (on) { var p = v.play(); if (p && p.catch) p.catch(function () {}); } else if (!v.paused) v.pause();
    });
  }
  function luminance(r, g, b) { return (0.2126 * r + 0.7152 * g + 0.0722 * b) / 255; }
  each(document.querySelectorAll('.si-bg-overlay[data-overlay="auto"]'), function (ov) {
    var img = ov.parentNode.querySelector('img.si-bg-media');
    if (!img) return;
    function measure() {
      try {
        var c = document.createElement('canvas'); c.width = c.height = 24;
        var g = c.getContext('2d'); g.drawImage(img, 0, 0, 24, 24);
        var px = g.getImageData(0, 0, 24, 24).data, sum = 0;
        for (var i = 0; i < px.length; i += 4) sum += luminance(px[i], px[i + 1], px[i + 2]);
        var lum = sum / (px.length / 4);
        /* a faded image mixes with the page background */
        var op = parseFloat(getComputedStyle(img).opacity) || 1;
        var bg = /rgba?\((\d+),\s*(\d+),\s*(\d+)/.exec(getComputedStyle(document.body).backgroundColor);
        if (bg) lum = lum * op + luminance(+bg[1], +bg[2], +bg[3]) * (1 - op);
        var light = lum > 0.62, slide = ov.closest('.si-slide');
        ov.setAttribute('data-overlay', light ? 'light' : 'dark');
        slide.classList.toggle('si-bg-light', light);
        slide.classList.toggle('si-bg-dark', !light);
      } catch (e) { /* unreadable pixels: keep the dark overlay */ }
    }
    if (img.complete && img.naturalWidth) measure(); else img.addEventListener('load', measure);
  });

  /* ---------- SVG packets: dots that travel along paths marked data-packets ---------- */
  var SVGNS = 'http://www.w3.org/2000/svg';
  each(document.querySelectorAll('svg.si-canvas path[data-packets]'), function (p, idx) {
    if (!p.id) p.id = 'si-pk-' + idx;
    var g = document.createElementNS(SVGNS, 'g');
    g.setAttribute('class', 'si-packets');
    g.setAttribute('data-for', p.getAttribute('data-packets'));
    var dur = parseFloat(p.getAttribute('data-packets-dur') || '2.4');
    var count = parseInt(p.getAttribute('data-packets-count') || '3', 10);
    for (var j = 0; j < count; j++) {
      var c = document.createElementNS(SVGNS, 'circle');
      c.setAttribute('class', 'si-packet');
      c.setAttribute('r', p.getAttribute('data-packets-r') || '7');
      var a = document.createElementNS(SVGNS, 'animateMotion');
      a.setAttribute('dur', dur + 's');
      a.setAttribute('repeatCount', 'indefinite');
      a.setAttribute('begin', (-dur * j / count).toFixed(2) + 's');
      var mp = document.createElementNS(SVGNS, 'mpath');
      mp.setAttribute('href', '#' + p.id);
      a.appendChild(mp); c.appendChild(a); g.appendChild(c);
    }
    p.parentNode.insertBefore(g, p.nextSibling);
  });

  /* ---------- Mermaid (only when the deck has a mermaid fence and the script loaded) ---------- */
  var mermaidSrc = null;
  function cssVar(name) { return getComputedStyle(root).getPropertyValue(name).trim(); }
  function renderMermaid(again) {
    if (!window.mermaid) return;
    var nodes = document.querySelectorAll('pre.si-mermaid');
    if (!nodes.length) return;
    if (mermaidSrc === null) mermaidSrc = Array.prototype.map.call(nodes, function (x) { return x.textContent; });
    if (again) each(nodes, function (x, i) { x.removeAttribute('data-processed'); x.textContent = mermaidSrc[i]; });
    var v = { fg: cssVar('--si-fg'), bg: cssVar('--si-bg'), acc: cssVar('--si-accent'), acc2: cssVar('--si-accent-2'), muted: cssVar('--si-muted'),
      on: cssVar('--si-on-accent'), s1: cssVar('--si-surface'), s2: cssVar('--si-surface-2'), line: cssVar('--si-line') };
    try {
      window.mermaid.initialize({ startOnLoad: false, theme: 'base', fontFamily: cssVar('--si-font-body'), themeVariables: {
        primaryColor: v.s1, primaryTextColor: v.fg, primaryBorderColor: v.acc, secondaryColor: v.s2, tertiaryColor: v.s2,
        lineColor: v.muted, textColor: v.fg, mainBkg: v.s1, nodeBorder: v.acc, clusterBkg: v.s2, clusterBorder: v.line,
        titleColor: v.fg, edgeLabelBackground: v.bg, actorBkg: v.s1, actorBorder: v.acc, actorTextColor: v.fg,
        signalColor: v.fg, signalTextColor: v.fg, noteBkgColor: v.s2, noteTextColor: v.fg, noteBorderColor: v.line } });
      var p = window.mermaid.run({ nodes: nodes });
      if (p && p.then) p.then(function () { fit(slides[cur]); });
    } catch (e) { if (window.console) console.warn('mermaid', e); }
  }

  /* ---------- print: every slide in its final state ---------- */
  function printAll() { slides.forEach(function (s) { apply(s, maxStep(s)); }); }
  window.addEventListener('beforeprint', printAll);
  window.addEventListener('afterprint', function () { go(cur, step, true); });
  /* si build --pdf loads the deck at #print; headless Chrome's --print-to-pdf
     does not reliably fire beforeprint, so force every canvas to its final
     state here too. */
  /* si play --audience: on an audience phone, only full screen and light/dark;
     on the presenter's deck, <!-- qr: audience --> shows the live audience link */
  if (role === 'audience') root.setAttribute('data-si-role', 'audience');
  var audQR = window.SI_REMOTE_QR && window.SI_REMOTE_QR.audience;
  each(document.querySelectorAll('.si-qr-audience'), function (f) {
    if (!audQR || role) return;
    var slot = f.querySelector('.si-qr-slot');
    if (slot) slot.innerHTML = '<a href="' + audQR.url.replace(/"/g, '&quot;') + '">' + audQR.svg + '</a>';
    f.hidden = false;
  });

  /* polls (si play --audience): counts arrive as poll messages (to every page);
     an audience phone votes, once per poll, and may change its vote */
  function pollShow(m) {
    each(document.querySelectorAll('.si-poll[data-poll="' + String(m.id).replace(/"/g, '') + '"]'), function (f) {
      var counts = m.counts || [], total = m.total || 0;
      each(f.querySelectorAll('.si-poll-opt'), function (li) {
        var v = counts[+li.getAttribute('data-i')] || 0;
        li.style.setProperty('--_share', total ? Math.round(v * 100 / total) : 0);
        var nEl = li.querySelector('.si-poll-n'); if (nEl) nEl.textContent = v;
      });
      var t = f.querySelector('[data-poll-total]'); if (t) t.textContent = total;
    });
  }
  /* counts and bars only where a room can vote: an audience phone, or the deck of a si play --audience */
  if (role === 'audience' || audQR) each(document.querySelectorAll('.si-poll'), function (f) { f.setAttribute('data-live', ''); });
  if (role === 'audience') {
    var voter = '';
    try { voter = localStorage.getItem('si-voter') || ''; if (!voter) { voter = Math.random().toString(36).slice(2) + Math.random().toString(36).slice(2); localStorage.setItem('si-voter', voter); } } catch (e) { voter = Math.random().toString(36).slice(2) + Math.random().toString(36).slice(2); }
    each(document.querySelectorAll('.si-poll'), function (f) {
      var id = f.getAttribute('data-poll'), multi = f.hasAttribute('data-multiple'), mine = [];
      try { mine = JSON.parse(localStorage.getItem('si-vote-' + id) || '[]'); } catch (e) {}
      var paint = function () {
        each(f.querySelectorAll('.si-poll-opt'), function (li) { li.setAttribute('aria-pressed', mine.indexOf(+li.getAttribute('data-i')) >= 0 ? 'true' : 'false'); });
        if (mine.length) f.setAttribute('data-voted', ''); else f.removeAttribute('data-voted');
      };
      each(f.querySelectorAll('.si-poll-opt'), function (li) {
        li.setAttribute('role', 'button'); li.tabIndex = 0;
        var pick = function () {
          var i = +li.getAttribute('data-i');
          if (multi) { var at = mine.indexOf(i); if (at >= 0) mine.splice(at, 1); else mine.push(i); } else mine = [i];
          try { localStorage.setItem('si-vote-' + id, JSON.stringify(mine)); } catch (e) {}
          paint();
          fetch('/_si/audience/vote', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ poll: id, choices: mine, voter: voter }) }).catch(function () {});
        };
        li.addEventListener('click', pick);
        li.addEventListener('keydown', function (e) { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); pick(); } });
      });
      paint();
    });
  }

  /* an audience phone asks: a button, then a sheet with the question and a name */
  if (role === 'audience') {
    var askBtn = document.createElement('button');
    askBtn.type = 'button'; askBtn.id = 'si-ask-btn'; askBtn.textContent = 'Ask a question';
    askBtn.setAttribute('style', 'position:fixed;left:max(.8rem,env(safe-area-inset-left));bottom:max(.8rem,env(safe-area-inset-bottom));z-index:30;font:600 1rem var(--si-font-body);color:var(--si-on-accent);background:var(--si-accent);border:0;border-radius:2em;padding:.7em 1.1em;box-shadow:var(--si-shadow)');
    var sheet = document.createElement('form');
    sheet.id = 'si-ask'; sheet.hidden = true;
    sheet.setAttribute('style', 'position:fixed;inset:auto 0 0 0;z-index:31;display:grid;gap:.6rem;padding:1rem max(1rem,env(safe-area-inset-right)) max(1rem,env(safe-area-inset-bottom)) max(1rem,env(safe-area-inset-left));background:var(--si-surface);color:var(--si-fg);border-top:1px solid var(--si-line);box-shadow:0 -.5em 2em rgb(0 0 0 / .2);font:1rem var(--si-font-body)');
    sheet.innerHTML = '<label style="font-weight:600" for="si-ask-q">Your question</label>' +
      '<textarea id="si-ask-q" maxlength="500" rows="3" required style="font:inherit;padding:.6em;border:1px solid var(--si-line);border-radius:.5em;background:var(--si-bg);color:var(--si-fg)"></textarea>' +
      '<input id="si-ask-name" maxlength="40" placeholder="Your name (optional)" style="font:inherit;padding:.6em;border:1px solid var(--si-line);border-radius:.5em;background:var(--si-bg);color:var(--si-fg)">' +
      '<div style="display:flex;gap:.6rem;justify-content:flex-end"><button type="button" data-ask="cancel" style="font:inherit;padding:.6em 1em;border:1px solid var(--si-line);border-radius:.5em;background:var(--si-bg);color:var(--si-fg)">Cancel</button>' +
      '<button type="submit" style="font:600 1rem var(--si-font-body);padding:.6em 1.2em;border:0;border-radius:.5em;background:var(--si-accent);color:var(--si-on-accent)">Send</button></div>' +
      '<p data-ask="msg" role="status" style="margin:0;color:var(--si-muted);min-height:1.2em"></p>';
    document.body.appendChild(askBtn); document.body.appendChild(sheet);
    var askMsg = sheet.querySelector('[data-ask="msg"]');
    askBtn.addEventListener('click', function () { sheet.hidden = false; askBtn.hidden = true; askMsg.textContent = ''; sheet.querySelector('textarea').focus(); });
    sheet.querySelector('[data-ask="cancel"]').addEventListener('click', function () { sheet.hidden = true; askBtn.hidden = false; });
    sheet.addEventListener('submit', function (e) {
      e.preventDefault();
      var q = sheet.querySelector('textarea'), name = sheet.querySelector('#si-ask-name');
      fetch('/_si/audience/ask', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ text: q.value, name: name.value, voter: voter }) })
        .then(function (r) {
          if (r.ok) { q.value = ''; askMsg.textContent = 'Sent. Thank you!'; setTimeout(function () { sheet.hidden = true; askBtn.hidden = false; }, 1200); }
          else askMsg.textContent = r.status === 429 ? 'One question every 20 seconds: try again in a moment.' : 'Could not send it.';
        }).catch(function () { askMsg.textContent = 'Could not send it.'; });
    });
  }

  /* Q&A (si play --audience): the question the presenter shows, over the slide;
     the deck passes the list on to a teleprompter window on this computer */
  var qaShown = null, qaBox = null;
  function qaIn(m) {
    if (role) return;
    chans.forEach(function (c) { if (typeof BroadcastChannel !== 'undefined' && c instanceof BroadcastChannel) try { c.postMessage(m); } catch (e) {} });
    var shown = null;
    (m.list || []).forEach(function (q) { if (q.state === 'shown') shown = q; });
    qaShown = shown;
    if (!qaBox) {
      qaBox = document.createElement('aside'); qaBox.className = 'si-qa-show'; qaBox.setAttribute('aria-live', 'polite');
      qaBox.innerHTML = '<p class="si-qa-text"></p><span class="si-qa-name"></span>';
      document.body.appendChild(qaBox);
    }
    qaBox.hidden = !shown;
    if (shown) {
      qaBox.querySelector('.si-qa-text').textContent = shown.text;
      qaBox.querySelector('.si-qa-name').textContent = shown.name || '';
    }
  }
  window.addEventListener('keydown', function (e) {
    if (e.key === 'Escape' && qaShown && !role) {
      try { fetch('/_si/qa', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id: qaShown.id, state: 'open' }) }); } catch (x) {}
    }
  });

  /* si build --handout prints the page with ?si-handout=slides|lines|points|notes
     (and ?si-paper=letter): light, every slide in its final state */
  var handout = /[?&]si-handout=(slides|lines|points|notes)(?:&|$)/.exec(location.search);
  if (handout) {
    root.setAttribute('data-si-handout', handout[1]);
    root.setAttribute('data-si-mode', 'light');
    // portrait paper with margins, after the deck's own landscape @page so it wins
    var paper = document.createElement('style');
    paper.textContent = '@page{size:' + (/[?&]si-paper=letter(?:&|$)/.test(location.search) ? 'letter' : 'A4') + ' portrait;margin:14mm 14mm 16mm}';
    document.head.appendChild(paper);
  }
  if (location.hash === '#print' || handout) printAll();

  var fitTimer = 0;
  window.addEventListener('resize', function () { clearTimeout(fitTimer); fitTimer = setTimeout(function () { fit(slides[cur]); }, 100); });
  if (window.matchMedia) {
    var mq = window.matchMedia('(prefers-color-scheme: dark)');
    if (mq.addEventListener) mq.addEventListener('change', function () { if (!root.getAttribute('data-si-mode')) renderMermaid(true); });
  }
  if (document.fonts && document.fonts.ready) document.fonts.ready.then(function () { fit(slides[cur]); });
  window.addEventListener('load', function () { renderMermaid(false); fit(slides[cur]); });
  fromHash(true);
  if (role === 'thumb') go(cur, 'end', true);
  /* remote clicker: big Next and Prev for a phone, where the deck is, what
     comes next, and the time since the clicker opened */
  var clk = null, clkStart = Date.now();
  function clickerState(m) {
    if (!clk) return;
    clk.pos.textContent = (m.i + 1) + ' / ' + m.n + (m.kmax ? '  \u00b7  step ' + m.k + '/' + m.kmax : '');
    clk.title.textContent = m.title || '';
    clk.next.textContent = m.label || '';
  }
  if (role === 'clicker') {
    var box = document.createElement('div');
    box.id = 'si-clicker';
    box.setAttribute('style', 'position:fixed;inset:0;z-index:2147483647;display:grid;grid-template-rows:auto 1fr 2fr;gap:3vmin;padding:max(4vmin,env(safe-area-inset-top)) 4vmin max(4vmin,env(safe-area-inset-bottom));background:var(--si-bg);color:var(--si-fg);font-family:var(--si-font-body);touch-action:manipulation;user-select:none;-webkit-user-select:none');
    box.innerHTML = '<div style="display:grid;gap:1vmin"><div style="display:flex;justify-content:space-between;color:var(--si-muted);font-variant-numeric:tabular-nums;font-size:4.5vmin"><span data-c="pos">\u2026</span><span data-c="time">00:00</span></div>' +
      '<div data-c="title" style="font:var(--si-heading-weight) 6vmin/1.2 var(--si-font-heading)"></div><div data-c="next" style="color:var(--si-muted);font-size:4vmin"></div></div>' +
      '<button type="button" data-c="prev" style="font:600 6vmin var(--si-font-body);border:2px solid var(--si-line);border-radius:var(--si-radius);background:var(--si-surface);color:var(--si-fg)">\u2039 Prev</button>' +
      '<button type="button" data-c="nextbtn" style="font:700 9vmin var(--si-font-body);border:0;border-radius:var(--si-radius);background:var(--si-accent);color:var(--si-on-accent)">Next \u203a</button>';
    document.body.appendChild(box);
    var q = function (k) { return box.querySelector('[data-c="' + k + '"]'); };
    clk = { pos: q('pos'), title: q('title'), next: q('next') };
    var send = function (c) { post({ type: 'cmd', cmd: c }); try { navigator.vibrate && navigator.vibrate(15); } catch (e) {} };
    q('prev').addEventListener('click', function () { send('prev'); });
    q('nextbtn').addEventListener('click', function () { send('next'); });
    var tm = q('time');
    setInterval(function () { var s = Math.floor((Date.now() - clkStart) / 1000); tm.textContent = (s < 600 ? '0' : '') + Math.floor(s / 60) + ':' + (s % 60 < 10 ? '0' : '') + s % 60; }, 500);
    tm.addEventListener('click', function () { clkStart = Date.now(); });
    post({ type: 'cmd', cmd: 'hello' });
  }
  /* remote draw pad: the pen is always on; prev/next buttons drive the deck */
  if (role === 'draw' && draw) {
    toggleDraw(true);
    if (drawBar) {
      var done = drawBar.querySelector('[data-si-draw="done"]'); if (done) done.style.display = 'none';
      [['prev', '\u2039 Prev', backward], ['next', 'Next \u203a', forward]].forEach(function (b) {
        var el = document.createElement('button'); el.type = 'button'; el.textContent = b[1]; el.setAttribute('data-si-remote', b[0]); el.style.whiteSpace = 'nowrap';
        el.addEventListener('click', function () { b[2](); el.blur(); }); drawBar.appendChild(el);
      });
    }
    post({ type: 'cmd', cmd: 'hello' });
  }
  /* --dev hot reload: after the page's slides are morphed in place, re-apply the
   * current slide (apply() reads the DOM live, so the new content shows) or
   * rebuild the overview grid if it is open. */
  function refresh() {
    if (deck.classList.contains('si-overview')) {
      clearOverview(); buildOverview();
      if (slides[ovSel]) slides[ovSel].classList.add('si-sel');
    } else {
      go(cur, step, true);
    }
  }
  window.SI = { go: go, next: forward, prev: backward, overview: overview, mode: toggleMode, refresh: refresh, run: runCmd,
    /* a command from outside (a terminal teleprompter, through si play's PositionJS) */
    cmd: function (c) { if (!role) onMsg({ data: Object.assign({ type: 'cmd' }, c) }); } };
})();
