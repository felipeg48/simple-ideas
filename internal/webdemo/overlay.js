(function (css) {
  // si demo run's layer over the page under test: a drawn cursor that glides
  // to each target, a ripple on clicks, notes pointing at elements and a blur
  // over fields typed from secrets. It lives in a closed shadow root on <html>
  // (out of the page's text and selectors) and never takes a pointer event.
  // Injected before the page's own scripts on every document; si restores the
  // cursor and the notes after a navigation (fresh is true until it does).
  if (window.__siDemo) return;
  var d = window.__siDemo = { fresh: true };
  var host, root, cur, notes = {}, blurs = [], at = null;
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
  function box(e, r) { e.style.left = r.left + 'px'; e.style.top = r.top + 'px'; e.style.width = r.width + 'px'; e.style.height = r.height + 'px'; }
  function tick(once) {
    var W = innerWidth, H = innerHeight, gap = 14;
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
