/* The Studio's map editor, inside the map canvas (a mind map page or fence
   rendered by si --web with ?si-edit): select a topic, Tab adds a child, Enter
   a sibling, Delete removes, F2 or typing renames, drag onto a topic to
   reparent (its top or bottom edge: before or after it), Shift-drag moves it
   freely (Alt-drag: its branch), Alt+↑/↓ reorders, Space folds. Every edit
   goes to the Studio as an op (postMessage); the Studio has Go rewrite the
   Markdown and sends the new maps back. The canvas never writes text itself. */
(function () {
  'use strict';
  /* mindmap.js starts its maps on DOMContentLoaded: start after it */
  if (document.readyState === 'loading') { document.addEventListener('DOMContentLoaded', init); return; }
  init();
  function init() {
  var el = document.querySelector('.si-mindmap');
  var mm = el && el.siMindmap;
  if (!mm) return;
  var M = window.SIMindmap.prototype, svg = mm.svg, parent = window.parent;
  var sel = null; // { map: id, id: 'r.0.1' }
  el.classList.add('si-mm-editing');

  function post(m) { parent.postMessage(m, '*'); }
  function pathOf(id) { return id === 'r' ? [] : id.slice(2).split('.').map(Number); }
  function idOf(path) { return path.length ? 'r.' + path.join('.') : 'r'; }
  function mapIndex(id) { return mm.order.indexOf(id); }
  function own() { return mapIndex(mm.st.map) >= 0 && mapIndex(mm.st.map) < (mm.own || mm.order.length); }
  function node(id) { return mm.cur ? mm.findNode(id) : null; }
  function gOf(id) { return svg.querySelector('.si-mm-node[data-id="' + id + '"]'); }

  function mark() {
    svg.querySelectorAll('.si-mm-sel').forEach(function (g) { g.classList.remove('si-mm-sel'); });
    if (sel && sel.map === mm.st.map) { var g = gOf(sel.id); if (g) g.classList.add('si-mm-sel'); }
  }
  var prevPaint = mm.onPaint;
  mm.onPaint = function () { if (prevPaint) prevPaint.call(mm); mark(); };

  function select(id, quiet) {
    sel = id === null ? null : { map: mm.st.map, id: id };
    mark();
    if (sel) {
      var g = gOf(id);
      /* a selection pushed by the Studio (quiet) must not pull the keyboard
         into this frame while the user types elsewhere (the ⌘K palette) */
      if (g && (!quiet || document.hasFocus())) g.focus({ preventScroll: true });
      var n = node(id);
      if (n) { /* keep it on screen */
        var vb = mm.vb, pad = 30;
        if (n.x - n.w / 2 < vb.x + pad || n.x + n.w / 2 > vb.x + vb.w - pad || n.y - n.h / 2 < vb.y + pad || n.y + n.h / 2 > vb.y + vb.h - pad) {
          mm.tween(mm.fitBox(mm.cur.all), 300); /* the whole map again: it grew past the view */
        }
      }
    }
    if (!quiet) post({ type: 'si-mm-select', map: mapIndex(mm.st.map), path: sel ? pathOf(sel.id) : null, readonly: !own() });
  }

  function op(o) {
    if (!own()) { mm.msg('This map comes from another file: open ' + (mm.maps[mm.st.map].file || 'it') + ' to edit it.'); return; }
    o.map = mapIndex(mm.st.map);
    post({ type: 'si-mm-op', op: o });
  }

  /* new maps from the Studio: keep the map, the view and the selection when they still exist */
  M.setData = function (data, opts, select) {
    var self = this, first = !this.edited;
    this.edited = true;
    Object.keys(opts || {}).forEach(function (k) { if (opts[k]) self.el.setAttribute('data-' + k, opts[k]); else self.el.removeAttribute('data-' + k); });
    this.order = data.maps.map(function (m) { return m.id; }); this.own = data.own;
    this.maps = {};
    data.maps.forEach(function (m) {
      (function fix(list) { (list || []).forEach(function (n) {
        if (n.pos) n.pos = { x: n.pos[0], y: n.pos[1] };
        if (n.bpos) n.bpos = { x: n.bpos[0], y: n.bpos[1] };
        fix(n.kids);
      }); })(m.kids);
      m.root = { t: m.title, kids: m.kids };
      self.maps[m.id] = m;
    });
    if (!this.maps[this.st.map]) { this.st.map = data.start; this.st.stack = []; first = true; }
    this.st.stack = this.st.stack.filter(function (e) { return self.maps[e.map]; });
    var lay = this.el.getAttribute('data-layout');
    if (this.lastLayout !== lay) { first = true; this.lastLayout = lay; }
    this.paint(false);
    if (first) this.setVB(this.fitBox(this.cur.all));
  };

  window.addEventListener('message', function (e) {
    var m = e.data || {};
    if (m.type === 'si-mm-data') {
      mm.setData(m.data, m.opts);
      if (m.select) {
        var id = mm.order[m.select.map];
        if (id && id !== mm.st.map && mm.maps[id]) { mm.st.stack = []; mm.show(id); }
        if (m.select.path) select(idOf(m.select.path), true); else select(null, true);
      } else mark();
    } else if (m.type === 'si-mm-show') {
      var to = mm.order[m.map];
      if (to && mm.maps[to]) { mm.st.stack = []; mm.show(to); select('r'); }
    }
  });

  /* ---------- renaming in place ---------- */
  var box = null;
  function rename(id, first) {
    var g = gOf(id), n = node(id);
    if (!g || !n || n.code || !own()) return;
    var r = (g.querySelector('.si-mm-shape, .si-mm-hit') || g).getBoundingClientRect();
    box = document.createElement('input');
    box.className = 'si-mm-rename'; box.value = first !== undefined ? first : n.t;
    box.setAttribute('aria-label', 'Topic text');
    box.style.left = r.left + 'px'; box.style.top = r.top + 'px';
    box.style.width = Math.max(r.width, 160) + 'px'; box.style.height = r.height + 'px';
    document.body.appendChild(box);
    box.focus();
    if (first === undefined) box.select();
    var done = false;
    function end(keep) {
      if (done) return; done = true;
      var v = box.value.trim(); box.remove(); box = null;
      var g2 = gOf(id); if (g2) g2.focus({ preventScroll: true });
      if (keep && v && v !== n.t) op({ kind: 'rename', path: pathOf(id), text: v });
    }
    box.addEventListener('keydown', function (e) {
      e.stopPropagation();
      if (e.key === 'Enter') { e.preventDefault(); end(true); }
      else if (e.key === 'Escape') { e.preventDefault(); end(false); }
      else if (e.key === 'Tab') { e.preventDefault(); end(true); }
    });
    box.addEventListener('blur', function () { end(true); });
  }

  /* ---------- pointer: select, drag to reparent, Shift/Alt-drag to move ---------- */
  var drag = null, lastClick = { t: 0, id: null };
  svg.addEventListener('pointerdown', function (e) {
    if (e.button > 0 || mm.busy) return;
    var g = e.target.closest && e.target.closest('.si-mm-node');
    if (!g) { if (sel) select(null); return; } /* the empty canvas pans, as always */
    e.stopImmediatePropagation(); e.preventDefault();
    var id = g.getAttribute('data-id'), n = node(id);
    var chip = e.target.closest('.si-mm-chip');
    if (chip && chip.getAttribute('data-chip') === 'link') { mm.act(g, chip); return; }
    drag = { id: id, n: n, x: e.clientX, y: e.clientY, moved: false, free: e.shiftKey ? 'pos' : e.altKey ? 'bpos' : null, vb: mm.vb };
    if (drag.free && n && n.d > 0) drag.start = n[drag.free] ? { x: n[drag.free].x, y: n[drag.free].y } : { x: 0, y: 0 };
    svg.setPointerCapture && svg.setPointerCapture(e.pointerId);
  }, true);
  function scale() { return Math.max(mm.vb.w / svg.clientWidth, mm.vb.h / svg.clientHeight); }
  function clearDrop() { svg.querySelectorAll('.si-mm-drop-in,.si-mm-drop-before,.si-mm-drop-after').forEach(function (g) { g.classList.remove('si-mm-drop-in', 'si-mm-drop-before', 'si-mm-drop-after'); }); }
  function dropAt(x, y) {
    var hit = document.elementFromPoint(x, y), g = hit && hit.closest && hit.closest('.si-mm-node');
    if (!g || g.classList.contains('si-mm-ghost')) return null;
    var id = g.getAttribute('data-id');
    if (id === drag.id || id.indexOf(drag.id + '.') === 0) return null; /* not onto itself or its own branch */
    var r = g.getBoundingClientRect(), f = (y - r.top) / Math.max(r.height, 1);
    var zone = id === 'r' ? 'in' : f < 0.28 ? 'before' : f > 0.72 ? 'after' : 'in';
    return { g: g, id: id, zone: zone };
  }
  window.addEventListener('pointermove', function (e) {
    if (!drag) return;
    var dx = e.clientX - drag.x, dy = e.clientY - drag.y;
    if (!drag.moved && Math.hypot(dx, dy) < 4) return;
    if (!own() || !drag.n || drag.n.d === 0) return;
    drag.moved = true;
    var s = scale();
    if (drag.free) {
      drag.n[drag.free] = { x: drag.start.x + dx * s, y: drag.start.y + dy * s };
      mm.paint(false);
      var gm = gOf(drag.id); if (gm) { gm.classList.add('si-mm-moving'); gm.parentNode.appendChild(gm); }
      return;
    }
    if (!drag.ghost) {
      var src = gOf(drag.id);
      if (!src) return;
      drag.ghost = src.cloneNode(true); drag.ghost.classList.add('si-mm-ghost'); drag.ghost.removeAttribute('data-id');
      src.parentNode.appendChild(drag.ghost);
    }
    drag.ghost.setAttribute('transform', 'translate(' + (dx * s) + ' ' + (dy * s) + ')');
    clearDrop();
    var t = dropAt(e.clientX, e.clientY);
    drag.target = t;
    if (t) t.g.classList.add('si-mm-drop-' + t.zone);
  }, true);
  window.addEventListener('pointerup', function (e) {
    var d = drag; drag = null;
    if (!d) return;
    clearDrop();
    if (d.ghost) d.ghost.remove();
    if (d.moved && d.free) {
      var p = d.n[d.free];
      op({ kind: 'pos', path: pathOf(d.id), x: p.x, y: p.y, branch: d.free === 'bpos' });
      return;
    }
    if (d.moved) {
      var t = d.target;
      if (!t) return;
      var tp = pathOf(t.id);
      if (t.zone === 'in') op({ kind: 'move', path: pathOf(d.id), to: tp, index: -1 });
      else op({ kind: 'move', path: pathOf(d.id), to: tp.slice(0, -1), index: tp[tp.length - 1] + (t.zone === 'after' ? 1 : 0) });
      return;
    }
    var now = performance.now(), dbl = lastClick.id === d.id && now - lastClick.t < 400;
    lastClick = { t: now, id: d.id };
    select(d.id);
    if (dbl) rename(d.id);
  }, true);

  /* ---------- keys ---------- */
  function nav(dir) {
    if (!sel) { select('r'); return; }
    var p = pathOf(sel.id), n = node(sel.id);
    if (dir === 'out') { if (p.length) select(idOf(p.slice(0, -1))); return; }
    if (dir === 'in') { if (n && n.kids && n.kids.length && !n.collapsed) select(idOf(p.concat(0))); return; }
    if (!p.length) return;
    var parentNode = node(idOf(p.slice(0, -1))), i = p[p.length - 1] + (dir === 'next' ? 1 : -1);
    if (parentNode && i >= 0 && i < (parentNode.kids || []).length) select(idOf(p.slice(0, -1).concat(i)));
  }
  document.addEventListener('keydown', function (e) {
    if (box || /input|textarea|select/i.test(e.target.tagName)) return;
    var mod = e.metaKey || e.ctrlKey, k = e.key;
    if (mod && (k === 'z' || k === 'Z' || k === 'y')) { e.preventDefault(); post({ type: 'si-mm-undo', redo: k === 'y' || e.shiftKey }); return; }
    if (mod) return;
    if (k === '/' ) return; /* search stays */
    if (!sel && /^(Tab|Enter|Delete|Backspace|F2|Arrow\w+| )$/.test(k)) { e.preventDefault(); select('r'); return; }
    if (!sel) return;
    var p = pathOf(sel.id);
    if (k === 'Tab') { e.preventDefault(); op({ kind: 'add-child', path: p, text: 'New topic' }); }
    else if (k === 'Enter') { e.preventDefault(); op({ kind: p.length ? 'add-sibling' : 'add-child', path: p, text: 'New topic' }); }
    else if ((k === 'Delete' || k === 'Backspace') && p.length) { e.preventDefault(); op({ kind: 'delete', path: p }); }
    else if (k === 'F2') { e.preventDefault(); rename(sel.id); }
    else if (k === ' ') { e.preventDefault(); var n = node(sel.id); if (n && n.kids && n.kids.length && p.length) op({ kind: 'set', path: p, field: 'collapsed', value: n.collapsed ? 'false' : 'true' }); }
    else if (e.altKey && (k === 'ArrowUp' || k === 'ArrowDown') && p.length) { e.preventDefault(); op({ kind: k === 'ArrowUp' ? 'up' : 'down', path: p }); }
    else if (k === 'ArrowLeft' || k === 'ArrowRight' || k === 'ArrowUp' || k === 'ArrowDown') {
      e.preventDefault(); e.stopImmediatePropagation();
      var side = (node(sel.id) || {}).side || 1, lay = mm.lay();
      if (lay === 'org') nav({ ArrowUp: 'out', ArrowDown: 'in', ArrowLeft: 'prev', ArrowRight: 'next' }[k]);
      else nav({ ArrowLeft: side < 0 ? 'in' : 'out', ArrowRight: side < 0 ? 'out' : 'in', ArrowUp: 'prev', ArrowDown: 'next' }[k]);
    }
    else if (k === 'Escape') { select(null); }
    else if (k.length === 1 && !e.altKey) { e.preventDefault(); rename(sel.id, k); }
  }, true);
  /* the runtime's own Enter / arrows on a focused topic (open links, nudge) do not apply here */
  svg.addEventListener('keydown', function (e) { if (sel && /^(Enter| |Arrow\w+)$/.test(e.key)) e.stopImmediatePropagation(); }, true);
  /* Tidy writes the cleared positions back */
  var tools = el.querySelector('.si-mm-tools');
  if (tools) tools.addEventListener('click', function (e) {
    var b = e.target.closest('[data-a="tidy"]');
    if (b) { e.stopImmediatePropagation(); op({ kind: 'tidy', path: [] }); }
  }, true);

  /* the canvas pane changes size with the Studio's panes: keep the whole map in view */
  if (window.ResizeObserver) {
    var lastW = 0, lastH = 0;
    new ResizeObserver(function () {
      var w = el.clientWidth, h = el.clientHeight;
      if (Math.abs(w - lastW) > 8 || Math.abs(h - lastH) > 8) { lastW = w; lastH = h; if (mm.cur) mm.setVB(mm.fitBox(mm.cur.all)); }
    }).observe(el);
  }
  post({ type: 'si-mm-ready' });
  }
})();
