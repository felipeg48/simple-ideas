(function (css) {
  // si demo run's layer over the page under test: a drawn cursor that glides
  // to each target, a ripple on clicks, notes pointing at elements and a blur
  // over fields typed from secrets. It lives in a closed shadow root on <html>
  // (out of the page's text and selectors) and never takes a pointer event.
  // Injected before the page's own scripts on every document; si restores the
  // cursor and the notes after a navigation (fresh is true until it does).
  if (window.__siDemo) return;
  var d = window.__siDemo = { fresh: true };
  var host, root, cur, notes = {}, blurs = [], at = null, spot = null;
  var ARROW = '<svg viewBox="0 0 24 24" width="28" height="28"><path d="M4 2.5v17.2l4.6-4.3 3 6.9 3.1-1.4-3-6.7h6.4z" fill="#111" stroke="#fff" stroke-width="1.6" stroke-linejoin="round"/></svg>';
  function el(tag, cls, html) { var e = document.createElement(tag); e.className = cls; if (html) e.innerHTML = html; return e; }
  function ensure() {
    if (host && host.isConnected) return true;
    if (!document.documentElement) return false;
    host = document.createElement('si-demo-overlay');
    host.style.cssText = 'all:initial;position:fixed;inset:0;z-index:2147483647;pointer-events:none;display:block;contain:strict';
    root = host.attachShadow({ mode: 'closed' });
    var st = document.createElement('style'); st.textContent = css; root.appendChild(st);
    cur = el('div', 'cursor', ARROW); cur.hidden = true; root.appendChild(cur);
    document.documentElement.appendChild(host);
    if (at) place(at[0], at[1], 0);
    return true;
  }
  function place(x, y, ms) {
    cur.style.transition = ms > 0 ? 'transform ' + ms + 'ms cubic-bezier(.3,.1,.2,1)' : 'none';
    cur.style.transform = 'translate(' + (x - 4) + 'px,' + (y - 2) + 'px)';
    cur.hidden = false;
  }
  d.move = function (x, y, ms) { if (!ensure()) return; var first = !at; at = [x, y]; place(x, y, first ? 0 : ms); };
  d.ripple = function (x, y) {
    if (!ensure()) return;
    var r = el('div', 'ripple'); r.style.left = x + 'px'; r.style.top = y + 'px';
    root.insertBefore(r, cur); setTimeout(function () { r.remove(); }, 700);
  };
  // note: a bubble with text pointing at the element locate.js picked last
  // (window.__siT) when anchored; pos top/right/bottom/left ("" = where it fits)
  d.note = function (id, text, pos, ms, anchored) {
    if (!ensure()) return;
    d.unnote(id);
    var b = el('div', 'note'); b.textContent = text;
    var ring = el('div', 'ring');
    root.insertBefore(ring, cur); root.insertBefore(b, cur);
    notes[id] = { b: b, ring: ring, t: anchored ? window.__siT : null, pos: pos };
    if (ms > 0) setTimeout(function () { d.unnote(id); }, ms);
    tick(true);
  };
  d.unnote = function (id) { var n = notes[id]; if (n) { n.b.remove(); n.ring.remove(); delete notes[id]; } };
  d.clearNotes = function () { for (var k in notes) d.unnote(k); };
  d.blur = function () {
    if (!ensure() || !window.__siT) return;
    var b = el('div', 'blur'); root.insertBefore(b, cur);
    blurs.push({ b: b, t: window.__siT }); tick(true);
  };
  // spotlight: the page dimmed (or blurred) around the element locate.js
  // picked last; the cut-out shrinks from the whole screen onto it, then
  // follows it. Under the notes and the cursor.
  d.spot = function (label, shape, look, ms, again) {
    if (!ensure() || !window.__siT) return;
    d.unspot(true);
    var l = el('div', 'spot' + (look === 'blur' ? ' blurred' : '')), ring = el('div', 'spot-ring' + (shape === 'circle' ? ' round' : ''));
    var tag = label ? el('div', 'note spot-label') : null;
    if (tag) tag.textContent = label;
    var first = root.children[1] || cur;
    root.insertBefore(l, first); root.insertBefore(ring, first); if (tag) root.insertBefore(tag, first);
    spot = { l: l, ring: ring, tag: tag, t: window.__siT, circle: shape === 'circle', t0: again ? 0 : performance.now() };
    if (again) l.classList.add('on'); else requestAnimationFrame(function () { l.classList.add('on'); });
    if (ms > 0) { var mine = spot; setTimeout(function () { if (spot === mine) d.unspot(); }, ms); }
    tick(true);
  };
  d.unspot = function (now) {
    if (!spot) return;
    var s = spot; spot = null;
    s.ring.remove(); if (s.tag) s.tag.remove();
    if (now) { s.l.remove(); return; }
    s.l.classList.remove('on'); setTimeout(function () { s.l.remove(); }, 450);
  };
  function hole(W, H) {
    var r = spot.t.getBoundingClientRect(), m = 10, x = r.left - m, y = r.top - m, w = r.width + 2 * m, h = r.height + 2 * m;
    if (spot.circle) { var c = Math.max(w, h); x -= (c - w) / 2; y -= (c - h) / 2; w = h = c; }
    var p = spot.t0 ? Math.min(1, (performance.now() - spot.t0) / 700) : 1, e = 1 - Math.pow(1 - p, 3);
    // from a hole as big as the screen (just outside it) onto the element
    var R = Math.hypot(W, H), fx = W / 2 - R, fy = H / 2 - R;
    return { x: fx + (x - fx) * e, y: fy + (y - fy) * e, w: 2 * R + (w - 2 * R) * e, h: 2 * R + (h - 2 * R) * e, p: p };
  }
  function drawSpot(W, H) {
    if (!spot.t.isConnected) { spot.l.hidden = spot.ring.hidden = true; if (spot.tag) spot.tag.hidden = true; return; }
    spot.l.hidden = spot.ring.hidden = false;
    var o = hole(W, H), rad = spot.circle ? Math.min(o.w, o.h) / 2 : 12, x1 = o.x + o.w, y1 = o.y + o.h;
    var rr = Math.min(rad, o.w / 2, o.h / 2);
    var path = 'M0 0H' + W + 'V' + H + 'H0Z' +
      'M' + (o.x + rr) + ' ' + o.y + 'H' + (x1 - rr) + 'A' + rr + ' ' + rr + ' 0 0 1 ' + x1 + ' ' + (o.y + rr) +
      'V' + (y1 - rr) + 'A' + rr + ' ' + rr + ' 0 0 1 ' + (x1 - rr) + ' ' + y1 + 'H' + (o.x + rr) +
      'A' + rr + ' ' + rr + ' 0 0 1 ' + o.x + ' ' + (y1 - rr) + 'V' + (o.y + rr) + 'A' + rr + ' ' + rr + ' 0 0 1 ' + (o.x + rr) + ' ' + o.y + 'Z';
    spot.l.style.clipPath = "path(evenodd,'" + path + "')";
    box(spot.ring, { left: o.x, top: o.y, width: o.w, height: o.h });
    spot.ring.style.borderRadius = rr + 'px';
    spot.ring.style.opacity = o.p;
    if (spot.tag) {
      var t = spot.tag, th = t.offsetHeight, ty = o.y - th - 10 < 8 ? y1 + 10 : o.y - th - 10;
      t.style.left = Math.max(8, Math.min(W - t.offsetWidth - 8, o.x)) + 'px'; t.style.top = ty + 'px';
      t.style.opacity = o.p;
    }
  }
  function box(e, r) { e.style.left = r.left + 'px'; e.style.top = r.top + 'px'; e.style.width = r.width + 'px'; e.style.height = r.height + 'px'; }
  function tick(once) {
    var W = innerWidth, H = innerHeight, gap = 14;
    if (spot) drawSpot(W, H);
    for (var k in notes) {
      var n = notes[k], b = n.b, bw = b.offsetWidth, bh = b.offsetHeight, x, y, p = n.pos;
      if (!n.t) { b.className = 'note free'; n.ring.hidden = true; b.style.left = (W - bw) / 2 + 'px'; b.style.top = gap * 2 + 'px'; continue; }
      if (!n.t.isConnected) { b.hidden = n.ring.hidden = true; continue; }
      b.hidden = n.ring.hidden = false;
      var r = n.t.getBoundingClientRect();
      box(n.ring, { left: r.left - 5, top: r.top - 5, width: r.width + 10, height: r.height + 10 });
      if (!p) p = r.right + gap + bw < W ? 'right' : r.left - gap - bw > 0 ? 'left' : r.bottom + gap + bh < H ? 'bottom' : 'top';
      if (p === 'right') { x = r.right + gap; y = r.top + r.height / 2 - bh / 2; }
      else if (p === 'left') { x = r.left - gap - bw; y = r.top + r.height / 2 - bh / 2; }
      else if (p === 'bottom') { x = r.left + r.width / 2 - bw / 2; y = r.bottom + gap; }
      else { x = r.left + r.width / 2 - bw / 2; y = r.top - gap - bh; }
      b.className = 'note at-' + p;
      b.style.left = Math.max(8, Math.min(W - bw - 8, x)) + 'px';
      b.style.top = Math.max(8, Math.min(H - bh - 8, y)) + 'px';
    }
    blurs = blurs.filter(function (z) {
      if (!z.t.isConnected) { z.b.remove(); return false; }
      var r = z.t.getBoundingClientRect(); box(z.b, r); return true;
    });
    if (!once) requestAnimationFrame(function () { tick(); });
  }
  if (document.documentElement) ensure();
  document.addEventListener('DOMContentLoaded', ensure);
  requestAnimationFrame(function () { tick(); });
})
