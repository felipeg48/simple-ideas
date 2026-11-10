  /* ---------- presenter view (p): a second window synced over BroadcastChannel,
     and with si play --remote also over the relay (other devices) ---------- */
  var chans = [];
  try { if ('BroadcastChannel' in window && role !== 'tp' && role !== 'tp-local' && role !== 'thumb') chans.push(new BroadcastChannel('si:' + data.name)); } catch (e) {}
  /* window.SI_RELAY is injected by si play --remote only (never in a built deck) */
  if (remoteOn && role !== 'tp' && window.SI_RELAY) chans.push(window.SI_RELAY());
  /* a page on the same origin hears a message twice (BroadcastChannel and relay): mid dedupes */
  var seen = [];
  function post(m) { m.name = data.name; m.mid = Math.random().toString(36).slice(2); chans.forEach(function (c) { try { c.postMessage(m); } catch (e) {} }); }
  function snapshot(s, k) {
    var c = s.cloneNode(true);
    c.classList.remove('si-past', 'si-future');
    c.classList.add('si-current');
    c.removeAttribute('id');
    c.removeAttribute('aria-hidden');
    var nt = c.querySelector('.si-notes'); if (nt) nt.parentNode.removeChild(nt);
    var pt = c.querySelector('.si-points'); if (pt) pt.parentNode.removeChild(pt);
    var inner = c.querySelector('.si-inner'); if (inner) { inner.style.removeProperty('--_fit'); inner.style.removeProperty('--_figfit'); }
    if (k !== undefined) apply(c, k);
    return c.outerHTML;
  }
  function announce() {
    /* si play / si serve: tell the server where we are, for a terminal teleprompter
       (si play deck.md --terminal --notes); a built deck has no cfg.live */
    if (cfg.live && !role && cur >= 0) {
      try {
        fetch('/_si/position', { method: 'POST', keepalive: true, headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ page: location.pathname, i: cur, k: step, steps: maxStep(slides[cur]), n: n }) });
      } catch (e) {}
    }
    if (!chans.length || cur < 0 || role) return; /* only the deck itself speaks for the deck */
    var s = slides[cur], max = maxStep(s), nx = null, label = 'End of deck';
    if (step < max) { nx = snapshot(s, step + 1); label = 'Next step ' + (step + 1) + '/' + max; }
    else if (cur + 1 < n) { nx = snapshot(slides[cur + 1], 0); label = 'Next: ' + (slides[cur + 1].getAttribute('data-title') || 'slide ' + (cur + 2)); }
    var notes = s.querySelector('.si-notes');
    var points = s.querySelector('.si-points');
    post({
        type: 'state', i: cur, k: step, kmax: max, n: n, title: s.getAttribute('data-title') || '',
        minutes: s.getAttribute('data-minutes') || '', total: data.minutes, plan: data.plan, notes: notes ? notes.innerHTML : '',
        points: points ? points.innerHTML : '',
        cur: snapshot(s), next: nx, label: label, dark: effectiveDark()
    });
  }
  function onMsg(e) {
    var m = e.data || {};
    if (m.name && m.name !== data.name) return;
    if (m.mid) { if (seen.indexOf(m.mid) >= 0) return; seen.push(m.mid); if (seen.length > 64) seen.shift(); }
    if (m.type === 'ink' || m.type === 'erase' || m.type === 'clear') { inkIn(m); return; }
    if (m.type === 'poll') { pollShow(m); return; }
    if (m.type === 'cloud') { cloudShow(m); return; }
    if (m.type === 'questions') { qaIn(m); return; }
    if (m.type === 'state' && role === 'clicker') { clickerState(m); return; }
    if (m.type === 'state' && (role === 'draw' || role === 'audience')) { if (m.i !== cur || m.k !== step) go(m.i, m.k); return; }
    if (m.type !== 'cmd' || role) return;
    if (m.cmd === 'next') forward();
    else if (m.cmd === 'prev') backward();
    else if (m.cmd === 'first') go(0, 0);
    else if (m.cmd === 'last') go(n - 1, 'end');
    else if (m.cmd === 'hello') announce();
    else if (m.cmd === 'go' && typeof m.i === 'number') go(m.i, 0);
  }
  chans.forEach(function (c) { c.onmessage = onMsg; });
  function presenterDoc(relay) {
    var css = Array.prototype.map.call(document.querySelectorAll('style'), function (s) { return s.textContent; }).join('\n');
    var rl = relay && document.getElementById('si-relay');
    return window.SI_PRESENTER
      .split('</head>').join((rl ? rl.outerHTML : '') + '</head>')
      .split('__CSS__').join(css)
      .split('__TITLE__').join(data.title.replace(/[<&]/g, ''))
      .split('__DATA__').join(JSON.stringify({ name: data.name, title: data.title, aspect: data.aspect, relay: relay, path: location.protocol === 'file:' ? '' : location.pathname }).replace(/</g, '\\u003c'));
  }
  function openPresenter() {
    var w = window.open('', 'si-presenter-' + data.name, 'width=1200,height=760');
    if (!w) return;
    w.document.open(); w.document.write(presenterDoc(false)); w.document.close();
    setTimeout(announce, 150);
  }
  /* remote teleprompter: this page becomes the presenter view, synced over the relay */
  if (role === 'tp' || role === 'tp-local') {
    var becomeTp = function () { var doc = presenterDoc(role === 'tp'); document.open(); document.write(doc); document.close(); };
    if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', becomeTp); else setTimeout(becomeTp, 0);
    return;
  }

  /* ---------- code tabs on a slide (<!-- tabs -->): a click shows that tab ---------- */
  document.addEventListener('click', function (e) {
    var b = e.target.closest && e.target.closest('.si-tabs-bar [role="tab"]');
    if (!b) return;
    var group = b.closest('.si-tabs'), label = b.getAttribute('data-label');
    each(group.querySelectorAll(':scope > .si-tabs-bar [role="tab"]'), function (x) { x.setAttribute('aria-selected', x === b ? 'true' : 'false'); });
    each(group.querySelectorAll(':scope > .si-tab'), function (t) { t.hidden = t.getAttribute('data-label') !== label; });
    b.blur(); e.stopPropagation();
  });

  /* ---------- keys, touch, controls ---------- */
  function toggleHelp(on) { if (help) help.classList.toggle('is-open', on === undefined ? !help.classList.contains('is-open') : on); }
  function fullscreen() {
    if (document.fullscreenElement) document.exitFullscreen();
    else if (root.requestFullscreen) root.requestFullscreen();
  }
