(function (a) {
  // Finds the element a person means: by its visible text, label, aria name,
  // placeholder, title or alt (loc.name), a CSS selector (loc.css), narrowed
  // by role (loc.role). Prefers visible, enabled elements, then the closest
  // match, then the smallest. Keeps the pick in window.__siT for the next
  // call; returns its state, its centre after scrolling it into view, and the
  // nearest candidates when nothing matches.
  var loc = a.loc || {}, want = a.want || 'any';
  var norm = function (s) { return String(s || '').replace(/\s+/g, ' ').trim(); };
  var low = function (s) { return norm(s).toLowerCase(); };
  var tag = function (el) { return el.tagName.toLowerCase(); };
  var NOTEXT = { hidden: 1, button: 1, submit: 1, reset: 1, image: 1, checkbox: 1, radio: 1, range: 1, color: 1, file: 1 };
  var role = function (el) {
    var r = el.getAttribute('role');
    if (r) return r;
    var t = tag(el), ty = (el.getAttribute('type') || 'text').toLowerCase();
    if (t === 'button' || t === 'summary' || (t === 'input' && /^(button|submit|reset|image)$/.test(ty))) return 'button';
    if (t === 'a' && el.hasAttribute('href')) return 'link';
    if (t === 'input' && ty === 'checkbox') return 'checkbox';
    if (t === 'input' && ty === 'radio') return 'radio';
    if (t === 'select') return el.multiple ? 'listbox' : 'combobox';
    if (t === 'textarea' || (t === 'input' && !NOTEXT[ty])) return 'textbox';
    if (/^h[1-6]$/.test(t)) return 'heading';
    if (t === 'img') return 'img';
    if (t === 'option') return 'option';
    if (t === 'li') return 'listitem';
    if (t === 'dialog') return 'dialog';
    return '';
  };
  var FIELD = { textbox: 1, combobox: 1, listbox: 1, checkbox: 1, radio: 1, 'switch': 1, searchbox: 1, spinbutton: 1, slider: 1 };
  var CLICK = { button: 1, link: 1, checkbox: 1, radio: 1, 'switch': 1, tab: 1, menuitem: 1, option: 1, combobox: 1, textbox: 1, listbox: 1 };
  var isField = function (el) { return !!FIELD[role(el)] || el.isContentEditable; };
  var visible = function (el) {
    if (!el.isConnected) return false;
    var r = el.getBoundingClientRect();
    if (r.width < 1 || r.height < 1) return false;
    return el.checkVisibility ? el.checkVisibility({ checkOpacity: true, checkVisibilityCSS: true }) : true;
  };
  var enabled = function (el) {
    if (el.disabled || el.getAttribute('aria-disabled') === 'true') return false;
    var f = el.closest('fieldset[disabled]');
    return !f;
  };
  var byIds = function (ids) {
    return ids.split(/\s+/).map(function (id) { var e = document.getElementById(id); return e ? e.innerText || e.textContent : ''; }).join(' ');
  };
  // names: [text, weight] — what a person could call the element
  var names = function (el) {
    var out = [];
    var add = function (s, w) { s = norm(s); if (s) out.push([s, w]); };
    add(el.getAttribute('aria-label'), 3);
    if (el.getAttribute('aria-labelledby')) add(byIds(el.getAttribute('aria-labelledby')), 3);
    if (el.labels) for (var i = 0; i < el.labels.length; i++) add(el.labels[i].innerText, 3);
    add(el.getAttribute('placeholder'), 2);
    add(el.getAttribute('title'), 2);
    add(el.getAttribute('alt'), 2);
    var t = tag(el);
    if (t === 'input' && /^(button|submit|reset)$/i.test(el.type)) add(el.value, 3);
    if (t !== 'select' && t !== 'textarea' && t !== 'input') add(el.innerText, 1);
    return out;
  };
  var desc = function (el) {
    var r = role(el) || tag(el), n = names(el);
    var s = n.length ? n[0][0] : '';
    if (s.length > 60) s = s.slice(0, 57) + '…';
    var id = el.id ? '#' + el.id : '';
    return r + (id ? ' ' + id : '') + (s ? ' "' + s + '"' : '');
  };
  // a text match on a wrapper also matches every ancestor: keep the innermost
  var innermost = function (el, s, exact) {
    for (var c = el.firstElementChild; c; c = c.nextElementSibling) {
      var ct = c.innerText;
      if (ct && (exact ? low(ct) === s : low(ct).indexOf(s) >= 0)) return false;
    }
    return true;
  };
  var all = document.querySelectorAll('body *');
  var hits = [];
  if (loc.css) {
    var found;
    try { found = document.querySelectorAll(loc.css); } catch (e) { return { error: 'not a valid selector: ' + loc.css }; }
    for (var i = 0; i < found.length; i++) hits.push({ el: found[i], score: 4 });
  } else {
    var want0 = low(loc.name);
    for (var i = 0; i < all.length; i++) {
      var el = all[i], t = tag(el);
      if (t === 'script' || t === 'style' || t === 'noscript' || t === 'template') continue;
      if (loc.role && role(el) !== loc.role) continue;
      var best = 0, ns = names(el);
      for (var k = 0; k < ns.length; k++) {
        var n = ns[k][0], w = ns[k][1], ln = n.toLowerCase(), sc = 0;
        if (n === loc.name) sc = 3;
        else if (ln === want0) sc = 2;
        else if (ln.indexOf(want0) >= 0 && n.length <= want0.length * 4 + 20) sc = 1;
        if (!sc) continue;
        if (w === 1 && !loc.role && !innermost(el, want0, sc > 1)) continue;
        sc = sc * 4 + w;
        if (sc > best) best = sc;
      }
      if (best) hits.push({ el: el, score: best });
    }
  }
  // a text inside a button or link means the button or link
  hits.forEach(function (h) {
    if (loc.css || loc.role || isField(h.el)) return;
    var up = h.el.closest('button,a[href],summary,label,[role=button],[role=link],[role=tab],[role=menuitem],[role=option],[role=checkbox],[role=switch]');
    if (up && up !== h.el) h.el = up;
    if (tag(h.el) === 'label' && h.el.control && (want === 'field')) h.el = h.el.control;
  });
  if (want === 'field') {
    var fields = hits.filter(function (h) { return isField(h.el); });
    if (fields.length) hits = fields;
  }
  hits.forEach(function (h) {
    h.vis = visible(h.el); h.en = enabled(h.el);
    var r = h.el.getBoundingClientRect(); h.area = r.width * r.height;
    h.act = CLICK[role(h.el)] ? 1 : 0;
  });
  hits.sort(function (x, y) {
    return (y.vis - x.vis) || (y.en - x.en) || (y.score - x.score) || (y.act - x.act) || (x.area - y.area);
  });
  if (!hits.length) {
    // the nearest things a person could have meant
    var q = low(loc.name || loc.css || ''), seen = {}, cands = [];
    var grams = function (s) { var g = {}; for (var i = 0; i < s.length - 1; i++) g[s.substr(i, 2)] = 1; return g; };
    var gq = grams(q);
    var sim = function (s) {
      var g = grams(s), n = 0, m = 0;
      for (var k in gq) { m++; if (g[k]) n++; }
      for (var k in g) m++;
      return m ? 2 * n / m : 0;
    };
    for (var i = 0; i < all.length; i++) {
      var el = all[i];
      if (!(CLICK[role(el)] || isField(el) || role(el) === 'heading' || tag(el) === 'label') || !visible(el)) continue;
      var d = desc(el);
      if (seen[d]) continue;
      seen[d] = 1;
      var ns = names(el), s = 0;
      for (var k = 0; k < ns.length; k++) s = Math.max(s, sim(ns[k][0].toLowerCase()));
      if (el.id) s = Math.max(s, sim('#' + el.id.toLowerCase()));
      cands.push([s, d]);
    }
    cands.sort(function (x, y) { return y[0] - x[0]; });
    return { found: false, candidates: cands.slice(0, 5).map(function (c) { return c[1]; }) };
  }
  var h = hits[0], el = h.el;
  window.__siT = el;
  var out = { found: true, visible: h.vis, enabled: h.en, desc: desc(el), count: hits.length,
    checked: !!(el.checked || el.getAttribute('aria-checked') === 'true'), field: isField(el), tag: tag(el) };
  // the height a fixed or sticky header covers at the top of the viewport:
  // a number, 'none', or 'auto' (found: a fixed/sticky bar across the top)
  var header = function () {
    if (a.header === 'none' || a.header === 0) return 0;
    if (typeof a.header === 'number') return a.header;
    var W = innerWidth, H = innerHeight, best = 0, xs = [0.06, 0.3, 0.5, 0.7, 0.94];
    for (var i = 0; i < xs.length; i++) {
      var at = document.elementsFromPoint(W * xs[i], 2);
      for (var j = 0; j < at.length; j++) {
        for (var e = at[j]; e && e !== document.body && e !== document.documentElement; e = e.parentElement) {
          var p = getComputedStyle(e).position;
          if (p !== 'fixed' && p !== 'sticky') continue;
          var b = e.getBoundingClientRect();
          if (b.top <= 2 && b.bottom > 0 && b.height < H * 0.4 && b.width >= W * 0.5 && !e.contains(el)) best = Math.max(best, b.bottom);
          break;
        }
      }
    }
    return Math.min(best, H * 0.4);
  };
  // in view: inside the viewport below the header, and its middle is its own
  // (not clipped by a scroller, not under something)
  var inView = function (top) {
    var r = el.getBoundingClientRect();
    if (r.top < top - 1 || r.bottom > innerHeight + 1 || r.left < -1 || r.right > innerWidth + 1) return false;
    var at = document.elementFromPoint(r.left + r.width / 2, r.top + r.height / 2);
    return !!at && (at === el || el.contains(at));
  };
  if (a.scroll && h.vis) {
    var hdr = header();
    // scroll only when it is not in view already: a spotlight on what shows
    // never nudges the page
    if (!inView(hdr)) {
      el.scrollIntoView({ block: 'center', inline: 'center', behavior: 'instant' });
      hdr = Math.max(hdr, header()); // a sticky bar that stuck on the way
      var r0 = el.getBoundingClientRect(), room = innerHeight - hdr, m = 12;
      var want = r0.height > room - 2 * m ? hdr + m : hdr + (room - r0.height) / 2;
      if (Math.abs(r0.top - want) > 1) window.scrollBy({ top: r0.top - want, left: 0, behavior: 'instant' });
    }
    var r = el.getBoundingClientRect();
    out.x = r.left + r.width / 2; out.y = r.top + r.height / 2;
    var top = document.elementFromPoint(out.x, out.y);
    if (top && top !== el && !el.contains(top) && !(el.labels && Array.prototype.indexOf.call(el.labels, top) >= 0)) {
      out.covered = desc(top);
    }
  }
  var rb = el.getBoundingClientRect();
  out.box = [rb.left, rb.top, rb.width, rb.height];
  return out;
})
