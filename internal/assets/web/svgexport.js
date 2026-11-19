/* simple-ideas: a vector SVG of a rendered element (si build --svg for
   timelines and infographics). Evaluated in headless Chrome on the built page,
   never shipped in one. It walks the laid-out DOM and writes what it paints:
   backgrounds and borders as rects (rounded), pictures as <image> (data URIs),
   inline SVG (the timeline's line, charts) cloned with its computed styles,
   and text as <text>, one per line as the browser broke it, at its baseline,
   with textLength so it keeps its width in any viewer. Colours are resolved to
   rgb() through a 1×1 canvas (oklch, color-mix and friends included);
   ::before/::after become real elements for the walk so their boxes count.
   Linear and radial background gradients become SVG gradients, conic ones
   180 thin wedges of their colours; outer box shadows blurred rects (an
   feGaussianBlur filter) under their box, inset ones a blurred frame clipped
   to the box, text shadows blurred copies of the text under it; the
   @font-face rules of the families used go in with their files as data URIs
   (si cuts them to the text's glyphs afterwards: cmd/si/svgfonts.go).
   ponytail: no repeating gradients; z-index is DOM order. */
window.siSVGExport = function (sel) {
  var root = document.querySelector(sel);
  if (!root) return '';
  var cv = document.createElement('canvas');
  cv.width = cv.height = 1;
  var cx = cv.getContext('2d', { willReadFrequently: true });
  function rgba(c) {
    if (!c || c === 'none' || c === 'transparent' || /^url\(/.test(c)) return null;
    cx.clearRect(0, 0, 1, 1);
    cx.fillStyle = '#000';
    cx.fillStyle = c;
    cx.fillRect(0, 0, 1, 1);
    var d = cx.getImageData(0, 0, 1, 1).data;
    return d[3] ? d : null;
  }
  function n(v) { return Math.round(v * 100) / 100; }
  function esc(s) { return String(s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;'); }
  function paint(c, name, op) {
    var d = rgba(c);
    if (!d) return null;
    var a = d[3] / 255 * op, s = name + '="rgb(' + d[0] + ',' + d[1] + ',' + d[2] + ')"';
    if (a < 0.995) s += ' ' + name + '-opacity="' + n(a) + '"';
    return s;
  }

  // ::before / ::after as real elements, so they have boxes to read
  var added = [], marks = [];
  var hide = document.createElement('style');
  hide.textContent = '[data-si-pb]::before{content:none!important}[data-si-pa]::after{content:none!important}';
  document.head.appendChild(hide); // a copied pseudo-element replaces its original at once
  Array.prototype.forEach.call(root.querySelectorAll('*'), function (el) {
    if (el.closest('svg')) return;
    ['before', 'after'].forEach(function (w) {
      var cs = getComputedStyle(el, '::' + w);
      if (!cs.content || cs.content === 'none' || cs.content === 'normal' || cs.display === 'none') return;
      var s = document.createElement('si-px');
      for (var i = 0; i < cs.length; i++) s.style.setProperty(cs[i], cs.getPropertyValue(cs[i]));
      var m = /^"([\s\S]*)"$/.exec(cs.content);
      s.textContent = m ? m[1].replace(/\\(.)/g, '$1') : '';
      var was = el.getBoundingClientRect();
      if (w === 'before') el.insertBefore(s, el.firstChild); else el.appendChild(s);
      el.setAttribute(w === 'before' ? 'data-si-pb' : 'data-si-pa', '');
      // a stretched pseudo-element reports its used size: copied, it can push
      // its parent bigger; then let it size itself again
      [['height', 'block-size'], ['width', 'inline-size']].forEach(function (ps) {
        var now = el.getBoundingClientRect();
        if (Math.abs(now.height - was.height) > 0.5 || Math.abs(now.width - was.width) > 0.5) ps.forEach(function (p) { s.style.removeProperty(p); });
      });
      added.push(s);
      marks.push(el);
    });
  });

  var R = root.getBoundingClientRect(), ox = R.left, oy = R.top, defs = [], id = 0;
  function radius(cs, w, h) {
    var r = cs.borderTopLeftRadius.split(' ')[0];
    var v = parseFloat(r) || 0;
    if (/%$/.test(r)) v = v / 100 * Math.min(w, h) * 2;
    return Math.min(v, w / 2, h / 2);
  }
  function box(r, rx, attrs) {
    return '<rect x="' + n(r.left - ox) + '" y="' + n(r.top - oy) + '" width="' + n(r.width) + '" height="' + n(r.height) + '"' +
      (rx ? ' rx="' + n(rx) + '"' : '') + ' ' + attrs + '/>';
  }
  /* any CSS colour as [r, g, b, a] (transparent too), null when v is none */
  function color4(v) {
    if (!v || !CSS.supports("color", v)) return null;
    return rgba(v) || [0, 0, 0, 0];
  }
  /* top-level comma (or space) split, parentheses kept whole */
  function split(s, sep) {
    var out = [], depth = 0, cur = '';
    for (var i = 0; i < s.length; i++) {
      var ch = s[i];
      if (ch === '(') depth++;
      if (ch === ')') depth--;
      if (!depth && (sep === ' ' ? /\s/.test(ch) : ch === sep)) { if (cur.trim()) out.push(cur.trim()); cur = ''; continue; }
      cur += ch;
    }
    if (cur.trim()) out.push(cur.trim());
    return out;
  }
  function stopsOf(parts, len) {
    return stopList(parts, len).map(function (s) {
      return '<stop offset="' + n(s.o) + '" stop-color="rgb(' + s.c[0] + ',' + s.c[1] + ',' + s.c[2] + ')"' + (s.c[3] < 255 ? ' stop-opacity="' + n(s.c[3] / 255) + '"' : '') + '/>';
    }).join('');
  }
  function stopList(parts, len) {
    var st = [];
    parts.forEach(function (p) {
      var w = split(p, ' '), d = color4(w[0]);
      if (!d) return;
      var pos = w.slice(1).map(function (v) { var o = /%$/.test(v) ? parseFloat(v) / 100 : parseFloat(v) / (len || 1); return isFinite(o) ? o : null; });
      if (!pos.length) pos = [null];
      pos.forEach(function (o) { st.push({ c: d, o: o }); });
    });
    if (st.length && st[0].o === null) st[0].o = 0;
    if (st.length && st[st.length - 1].o === null) st[st.length - 1].o = 1;
    for (var i = 1; i < st.length; i++) { // missing positions: spread evenly, never backwards
      if (st[i].o !== null) { st[i].o = Math.max(st[i].o, st[i - 1].o); continue; }
      var j = i; while (st[j].o === null) j++;
      for (var k = i; k < j; k++) st[k].o = st[i - 1].o + (st[j].o - st[i - 1].o) * (k - i + 1) / (j - i + 1);
      i = j - 1;
    }
    return st;
  }
  /* the colour of a stop list at offset o (0..1), straight RGBA interpolation */
  function colorAt(st, o) {
    if (o <= st[0].o) return st[0].c;
    for (var i = 1; i < st.length; i++) {
      if (o <= st[i].o) {
        var a = st[i - 1], b = st[i], t = b.o > a.o ? (o - a.o) / (b.o - a.o) : 1;
        return [0, 1, 2, 3].map(function (k) { return Math.round(a.c[k] + (b.c[k] - a.c[k]) * t); });
      }
    }
    return st[st.length - 1].c;
  }
  var units = { deg: 1 / 360, turn: 1, rad: 1 / (2 * Math.PI), grad: 1 / 400 };
  /* a rounded rect as path data (for holes and clips) */
  function rrect(x, y, w, h, r) {
    r = Math.max(0, Math.min(r, w / 2, h / 2));
    if (!r) return 'M' + n(x) + ' ' + n(y) + 'h' + n(w) + 'v' + n(h) + 'h' + n(-w) + 'Z';
    return 'M' + n(x + r) + ' ' + n(y) + 'h' + n(w - 2 * r) + 'a' + n(r) + ' ' + n(r) + ' 0 0 1 ' + n(r) + ' ' + n(r) + 'v' + n(h - 2 * r) + 'a' + n(r) + ' ' + n(r) + ' 0 0 1 ' + n(-r) + ' ' + n(r) +
      'h' + n(2 * r - w) + 'a' + n(r) + ' ' + n(r) + ' 0 0 1 ' + n(-r) + ' ' + n(-r) + 'v' + n(2 * r - h) + 'a' + n(r) + ' ' + n(r) + ' 0 0 1 ' + n(r) + ' ' + n(-r) + 'Z';
  }
  function clipTo(r, rx) {
    var cid = 'si-c' + (++id);
    defs.push('<clipPath id="' + cid + '">' + box(r, rx, '') + '</clipPath>');
    return ' clip-path="url(#' + cid + ')"';
  }
  /* a conic gradient: thin wedges of its colours around its centre, clipped
     to the box (SVG has no conic gradient) */
  function conic(args, r, rx, op) {
    var x = r.left - ox, y = r.top - oy, w = r.width, h = r.height, from = 0, px = w / 2, py = h / 2, head = args[0];
    if (!color4(split(head, ' ')[0])) {
      args = args.slice(1);
      var f = /from\s+(-?[\d.]+)(deg|turn|rad|grad)/.exec(head);
      if (f) from = parseFloat(f[1]) * units[f[2]];
      var at = /at\s+(.+)$/.exec(head);
      if (at) {
        var ps = at[1].trim().split(/\s+/), pv = function (v, d) { return v === 'left' || v === 'top' ? 0 : v === 'right' || v === 'bottom' ? d : v === 'center' ? d / 2 : /%$/.test(v) ? parseFloat(v) / 100 * d : parseFloat(v); };
        px = pv(ps[0], w); py = pv(ps[1] || 'center', h);
      }
    }
    var st = stopList(args.map(function (p) {
      return p.replace(/(-?[\d.]+)(deg|turn|rad|grad)\b/g, function (_, v, u) { return parseFloat(v) * units[u] * 100 + '%'; });
    }), 1);
    if (!st.length) return '';
    var cx0 = x + px, cy0 = y + py, R = Math.hypot(Math.max(px, w - px), Math.max(py, h - py)) + 1, N = 180, out = '';
    for (var i = 0; i < N; i++) {
      var c = colorAt(st, (i + 0.5) / N), a0 = (from + i / N) * 2 * Math.PI, a1 = (from + Math.min(i + 2, N + 0.02) / N) * 2 * Math.PI; /* each wedge runs under the next: no seams */
      if (!c[3]) continue;
      out += '<path d="M' + n(cx0) + ' ' + n(cy0) + 'L' + n(cx0 + R * Math.sin(a0)) + ' ' + n(cy0 - R * Math.cos(a0)) + 'L' + n(cx0 + R * Math.sin(a1)) + ' ' + n(cy0 - R * Math.cos(a1)) +
        'Z" fill="rgb(' + c[0] + ',' + c[1] + ',' + c[2] + ')"' + (c[3] < 255 ? ' fill-opacity="' + n(c[3] / 255) + '"' : '') + '/>';
    }
    return '<g' + clipTo(r, rx) + (op < 0.995 ? ' opacity="' + n(op) + '"' : '') + '>' + out + '</g>';
  }
  var sides = { top: [0, -1], bottom: [0, 1], left: [-1, 0], right: [1, 0] };
  /* background-image gradients as SVG gradients over the box, bottom layer first */
  function gradients(cs, r, rx, op) {
    var img = cs.backgroundImage;
    if (!img || img === 'none') return '';
    var out = '', sizes = split(cs.backgroundSize, ','), poss = split(cs.backgroundPosition, ','), reps = split(cs.backgroundRepeat, ',');
    split(img, ',').map(function (l, i) { return [l, i]; }).reverse().forEach(function (li) {
      var i = li[1], t = tile(r, sizes[i % sizes.length], poss[i % poss.length], reps[i % reps.length]);
      if (t === r) { out += layerSVG(li[0], r, rx, op); return; }
      var s = layerSVG(li[0], t, 0, op); /* a sized, unrepeated layer (a drawn line): its own rect, cut to the box */
      if (s) out += '<g' + clipTo(r, rx) + '>' + s + '</g>';
    });
    return out;
  }
  /* the rect a background layer covers: the box, or for a no-repeat layer
     with a size its own tile at its position */
  function tile(r, size, pos, rep) {
    var sz = split(size || 'auto', ' ');
    if (rep !== 'no-repeat' || /^(auto|cover|contain)$/.test(sz[0])) return r;
    var len = function (v, d) { return /%$/.test(v) ? parseFloat(v) / 100 * d : parseFloat(v); };
    var tw = len(sz[0], r.width), th = sz[1] && sz[1] !== 'auto' ? len(sz[1], r.height) : r.height;
    var ps = split(pos || '0% 0%', ' '), px = /%$/.test(ps[0]) ? (r.width - tw) * parseFloat(ps[0]) / 100 : parseFloat(ps[0]) || 0,
      py = /%$/.test(ps[1] || '') ? (r.height - th) * parseFloat(ps[1]) / 100 : parseFloat(ps[1]) || 0;
    if (!(tw > 0 && th > 0)) return r;
    return { left: r.left + px, top: r.top + py, width: tw, height: th };
  }
  function layerSVG(layer, r, rx, op) {
    var x = r.left - ox, y = r.top - oy, w = r.width, h = r.height, out = '';
    var m = /^(linear|radial|conic)-gradient\(([\s\S]*)\)$/.exec(layer);
    if (!m) return '';
    var args = split(m[2], ','), gid = 'si-g' + (++id), g;
    if (m[1] === 'conic') return conic(args, r, rx, op);
    if (m[1] === 'linear') {
      var a = 180, first = args[0];
      if (/^-?[\d.]+(deg|turn|rad|grad)$/.test(first)) {
        a = parseFloat(first) * ({ deg: 1, turn: 360, rad: 180 / Math.PI, grad: 0.9 })[/[a-z]+$/.exec(first)[0]];
        args.shift();
      } else if (/^to /.test(first)) {
        var v = [0, 0];
        first.slice(3).split(/\s+/).forEach(function (k) { if (sides[k]) { v[0] += sides[k][0] * h; v[1] += sides[k][1] * w; } });
        a = Math.atan2(v[0], -v[1]) * 180 / Math.PI;
        args.shift();
      }
      var t = a * Math.PI / 180, sx = Math.sin(t), sy = -Math.cos(t), L = Math.abs(w * sx) + Math.abs(h * sy), cxp = x + w / 2, cyp = y + h / 2;
      g = '<linearGradient id="' + gid + '" gradientUnits="userSpaceOnUse" x1="' + n(cxp - sx * L / 2) + '" y1="' + n(cyp - sy * L / 2) + '" x2="' + n(cxp + sx * L / 2) + '" y2="' + n(cyp + sy * L / 2) + '">' + stopsOf(args, L) + '</linearGradient>';
    } else {
      var shape = 'ellipse', size = 'farthest-corner', px = w / 2, py = h / 2, head = args[0];
      if (!color4(split(head, " ")[0])) {
        args.shift();
        var at = head.split(/\s+at\s+|^at\s+/), wds = (at[0] || '').split(/\s+/).filter(Boolean), lens = [];
        wds.forEach(function (k) { if (k === 'circle' || k === 'ellipse') shape = k; else if (/side|corner/.test(k)) size = k; else lens.push(k); });
        if (at[1]) {
          var ps = at[1].split(/\s+/), pv = function (v, d) { return v === 'left' || v === 'top' ? 0 : v === 'right' || v === 'bottom' ? d : v === 'center' ? d / 2 : /%$/.test(v) ? parseFloat(v) / 100 * d : parseFloat(v); };
          px = pv(ps[0], w); py = pv(ps[1] || 'center', h);
        }
        if (lens.length) size = lens;
      }
      var dx = [px, w - px], dy = [py, h - py], rX, rY;
      if (Array.isArray(size)) {
        rX = /%$/.test(size[0]) ? parseFloat(size[0]) / 100 * w : parseFloat(size[0]);
        rY = size[1] ? (/%$/.test(size[1]) ? parseFloat(size[1]) / 100 * h : parseFloat(size[1])) : rX;
      } else {
        var far = /farthest/.test(size), side = /side/.test(size), ex = far ? Math.max.apply(0, dx) : Math.min.apply(0, dx), ey = far ? Math.max.apply(0, dy) : Math.min.apply(0, dy);
        if (shape === 'circle') { rX = rY = side ? (far ? Math.max(ex, ey) : Math.min(ex, ey)) : Math.hypot(ex, ey); }
        else { rX = side ? ex : ex * Math.SQRT2; rY = side ? ey : ey * Math.SQRT2; }
      }
      rX = rX || 1; rY = rY || 1;
      var gcx = x + px, gcy = y + py;
      g = '<radialGradient id="' + gid + '" gradientUnits="userSpaceOnUse" cx="' + n(gcx) + '" cy="' + n(gcy) + '" r="' + n(rY) + '" gradientTransform="translate(' + n(gcx) + ' ' + n(gcy) + ') scale(' + n(rX / rY) + ' 1) translate(' + n(-gcx) + ' ' + n(-gcy) + ')">' + stopsOf(args, rY) + '</radialGradient>';
    }
    defs.push(g);
    out += box(r, rx, 'fill="url(#' + gid + ')"' + (op < 0.995 ? ' opacity="' + n(op) + '"' : ''));
    return out;
  }
  /* outer box shadows: the box's shape, offset, spread and blurred by a filter, under it */
  function shadows(cs, r, rx, op) {
    var bs = cs.boxShadow;
    if (!bs || bs === 'none') return '';
    var out = '';
    split(bs, ',').reverse().forEach(function (s) {
      var w = split(s, ' ');
      if (w.indexOf('inset') >= 0) return;
      var col = null, l = [];
      w.forEach(function (v) { if (/^-?[\d.]+(px)?$/.test(v)) l.push(parseFloat(v)); else col = v; });
      var f = paint(col || cs.color, 'fill', op);
      if (!f || l.length < 2) return;
      var blur = l[2] || 0, sp = l[3] || 0, b = { left: r.left + l[0] - sp, top: r.top + l[1] - sp, width: r.width + 2 * sp, height: r.height + 2 * sp };
      if (b.width <= 0 || b.height <= 0) return;
      var filt = '';
      if (blur > 0) {
        var fid = 'si-s' + (++id), pad = blur * 1.5;
        defs.push('<filter id="' + fid + '" filterUnits="userSpaceOnUse" x="' + n(b.left - ox - pad) + '" y="' + n(b.top - oy - pad) + '" width="' + n(b.width + 2 * pad) + '" height="' + n(b.height + 2 * pad) + '"><feGaussianBlur stdDeviation="' + n(blur / 2) + '"/></filter>');
        filt = ' filter="url(#' + fid + ')"';
      }
      out += box(b, rx ? rx + sp : 0, f + filt);
    });
    return out;
  }
  /* inset box shadows: a frame (a big rect with the box's shape, offset and
     shrunk by the spread, cut out) blurred, clipped to the box, over the
     background */
  function insetShadows(cs, r, rx, op) {
    var bs = cs.boxShadow;
    if (!bs || bs === 'none' || bs.indexOf('inset') < 0) return '';
    var out = '';
    split(bs, ',').reverse().forEach(function (s) {
      var w = split(s, ' ');
      if (w.indexOf('inset') < 0) return;
      var col = null, l = [];
      w.forEach(function (v) { if (/^-?[\d.]+(px)?$/.test(v)) l.push(parseFloat(v)); else if (v !== 'inset') col = v; });
      var f = paint(col || cs.color, 'fill', op);
      if (!f || l.length < 2) return;
      var blur = l[2] || 0, sp = l[3] || 0, x = r.left - ox, y = r.top - oy, pad = blur * 2 + Math.abs(l[0]) + Math.abs(l[1]) + 2;
      var hole = rrect(x + l[0] + sp, y + l[1] + sp, Math.max(0, r.width - 2 * sp), Math.max(0, r.height - 2 * sp), Math.max(0, rx - sp));
      var filt = '';
      if (blur > 0) {
        var fid = 'si-s' + (++id);
        defs.push('<filter id="' + fid + '" filterUnits="userSpaceOnUse" x="' + n(x - pad) + '" y="' + n(y - pad) + '" width="' + n(r.width + 2 * pad) + '" height="' + n(r.height + 2 * pad) + '"><feGaussianBlur stdDeviation="' + n(blur / 2) + '"/></filter>');
        filt = ' filter="url(#' + fid + ')"';
      }
      out += '<g' + clipTo(r, rx) + '><path fill-rule="evenodd" d="' + rrect(x - pad, y - pad, r.width + 2 * pad, r.height + 2 * pad, 0) + hole + '" ' + f + filt + '/></g>';
    });
    return out;
  }
  /* text-shadow: copies of the text in the shadow colour, offset and blurred,
     under it (the first shadow on top, as in CSS) */
  var blurs = {};
  function textShadows(cs, op) {
    var ts = cs.textShadow;
    if (!ts || ts === 'none') return [];
    var out = [];
    split(ts, ',').reverse().forEach(function (s) {
      var col = null, l = [];
      split(s, ' ').forEach(function (v) { if (/^-?[\d.]+(px)?$/.test(v)) l.push(parseFloat(v)); else col = v; });
      var f = paint(col || cs.color, 'fill', op);
      if (!f || l.length < 2) return;
      var blur = l[2] || 0, filt = '';
      if (blur > 0) {
        if (!blurs[blur]) {
          blurs[blur] = 'si-t' + (++id);
          defs.push('<filter id="' + blurs[blur] + '" x="-20%" y="-60%" width="140%" height="220%"><feGaussianBlur stdDeviation="' + n(blur / 2) + '"/></filter>');
        }
        filt = ' filter="url(#' + blurs[blur] + ')"';
      }
      out.push({ dx: l[0], dy: l[1], attrs: f + filt });
    });
    return out;
  }
  /* the @font-face rules of the families the text uses, with their sources
     inlined (data: URIs as built; a font file the page can read, read now) */
  var used = {};
  function fontFaces() {
    var out = [];
    function rules(list) {
      Array.prototype.forEach.call(list || [], function (ru) {
        if (ru.cssRules && ru.type !== 5) { rules(ru.cssRules); return; }
        if (ru.type !== 5) return; // CSSRule.FONT_FACE_RULE
        var fam = ru.style.getPropertyValue('font-family').replace(/^["']|["']$/g, '').toLowerCase();
        if (!used[fam]) return;
        var css = ru.cssText.replace(/url\(\s*["']?([^"')]+)["']?\s*\)/g, function (all, u) {
          if (/^data:/.test(u)) return all;
          try {
            var x = new XMLHttpRequest();
            x.open('GET', new URL(u, ru.parentStyleSheet && ru.parentStyleSheet.href || location.href).href, false);
            x.overrideMimeType('text/plain; charset=x-user-defined');
            x.send();
            if (x.status && x.status !== 200) return all;
            var bin = '', t = x.responseText;
            for (var i = 0; i < t.length; i++) bin += String.fromCharCode(t.charCodeAt(i) & 255);
            return 'url("data:font/' + (/\.(woff2?|ttf|otf)(\?|$)/.exec(u) || [0, 'woff2'])[1] + ';base64,' + btoa(bin) + '")';
          } catch (e) { return all; }
        });
        out.push(css);
      });
    }
    Array.prototype.forEach.call(document.styleSheets, function (sh) { try { rules(sh.cssRules); } catch (e) { /* another origin's sheet */ } });
    return out.length ? '<style>' + out.join('\n').replace(/<\//g, '<\\/') + '</style>' : '';
  }
  function transform(s, tt) {
    if (tt === 'uppercase') return s.toUpperCase();
    if (tt === 'lowercase') return s.toLowerCase();
    if (tt === 'capitalize') return s.replace(/(^|\s)(\S)/g, function (_, a, b) { return a + b.toUpperCase(); });
    return s;
  }
  function text(node, cs, op) {
    var t = node.nodeValue;
    if (!/\S/.test(t)) return '';
    var rg = document.createRange(), lines = [], cur = null, re = /\S+/g, m;
    function add(a, b, r) {
      if (!r.width) return;
      if (cur && Math.abs(r.top - cur.top) < 2 && r.left >= cur.right - 1) { cur.b = b; cur.right = r.right; return; }
      cur = { a: a, b: b, left: r.left, right: r.right, top: r.top, h: r.height };
      lines.push(cur);
    }
    function lineCount(rs) {
      var tops = {};
      for (var i = 0; i < rs.length; i++) tops[Math.round(rs[i].top)] = 1;
      return Object.keys(tops).length;
    }
    while ((m = re.exec(t))) {
      rg.setStart(node, m.index);
      rg.setEnd(node, m.index + m[0].length);
      var rs = rg.getClientRects();
      if (lineCount(rs) <= 1) { if (rs.length) add(m.index, m.index + m[0].length, rs[0]); continue; }
      var segs = [], from = 0, cnt = 1; // a word broken across lines: where its range gains a line
      for (var k = 1; k < m[0].length; k++) {
        rg.setStart(node, m.index);
        rg.setEnd(node, m.index + k + 1);
        var c = lineCount(rg.getClientRects()); // a drawn hyphen is a rect of its own, on the same line
        if (c > cnt) { segs.push([from, k]); from = k; cnt = c; }
      }
      segs.push([from, m[0].length]);
      segs.forEach(function (sg, i) {
        rg.setStart(node, m.index + sg[0]);
        rg.setEnd(node, m.index + sg[1]);
        var rs1 = rg.getClientRects();
        if (!rs1.length) return;
        var pick = i ? rs1[rs1.length - 1] : rs1[0], top = Math.round(pick.top), l = Infinity, r = -Infinity;
        for (var q = 0; q < rs1.length; q++) if (Math.round(rs1[q].top) === top) { l = Math.min(l, rs1[q].left); r = Math.max(r, rs1[q].right); }
        add(m.index + sg[0], m.index + sg[1], { left: l, right: r, width: r - l, top: pick.top, height: pick.height });
        if (i < segs.length - 1 && cs.hyphens === 'auto') cur.hy = true; // the browser drew a hyphen
      });
    }
    if (!lines.length) return '';
    var fill = paint(cs.color, 'fill', op);
    if (!fill) return '';
    cs.fontFamily.split(",").forEach(function (f) { used[f.trim().replace(/^["']|["']$/g, "").toLowerCase()] = 1; });
    cx.font = cs.fontStyle + ' ' + cs.fontWeight + ' ' + cs.fontSize + ' ' + cs.fontFamily;
    var mt = cx.measureText('Hg'), asc = mt.fontBoundingBoxAscent, desc = mt.fontBoundingBoxDescent;
    var font = ' font-family="' + esc(cs.fontFamily) + '" font-size="' + cs.fontSize + '" font-weight="' + cs.fontWeight + '"' +
      (cs.fontStyle !== 'normal' ? ' font-style="' + cs.fontStyle + '"' : '') +
      (cs.letterSpacing !== 'normal' && parseFloat(cs.letterSpacing) ? ' letter-spacing="' + cs.letterSpacing + '"' : '');
    var sh = textShadows(cs, op);
    return lines.map(function (l) {
      var s = transform(t.slice(l.a, l.b).replace(/\s+/g, ' '), cs.textTransform) + (l.hy ? '-' : '');
      var y = l.top + (l.h - asc - desc) / 2 + asc;
      function one(dx, dy, paintAttrs) {
        return '<text x="' + n(l.left - ox + dx) + '" y="' + n(y - oy + dy) + '" textLength="' + n(l.right - l.left) + '" lengthAdjust="spacingAndGlyphs"' +
          font + ' ' + paintAttrs + '>' + esc(s) + '</text>';
      }
      return sh.map(function (d) { return one(d.dx, d.dy, d.attrs); }).join('') + one(0, 0, fill);
    }).join('');
  }
  var svgProps = ['fill', 'stroke', 'stroke-width', 'stroke-opacity', 'fill-opacity', 'opacity', 'stroke-linecap', 'stroke-linejoin',
    'stroke-dasharray', 'stroke-dashoffset', 'vector-effect', 'paint-order', 'font-family', 'font-size', 'font-weight', 'text-anchor', 'dominant-baseline', 'display', 'visibility'];
  function svgClone(el, op) {
    var r = el.getBoundingClientRect();
    if (!r.width || !r.height) return '';
    var c = el.cloneNode(true), src = [el].concat(Array.prototype.slice.call(el.querySelectorAll('*'))),
      dst = [c].concat(Array.prototype.slice.call(c.querySelectorAll('*')));
    src.forEach(function (s, i) {
      var cs = getComputedStyle(s), d = dst[i];
      svgProps.forEach(function (p) {
        var v = cs.getPropertyValue(p);
        if (!v) return;
        if (p === 'fill' || p === 'stroke') {
          var rgb = rgba(v);
          if (rgb) {
            d.setAttribute(p, 'rgb(' + rgb[0] + ',' + rgb[1] + ',' + rgb[2] + ')');
            if (rgb[3] < 255) d.setAttribute(p + '-opacity', n(rgb[3] / 255 * (parseFloat(cs.getPropertyValue(p + '-opacity')) || 1)));
            return;
          }
          if (v !== 'none') return; // url(#…): kept as written
        }
        if (p === "font-family") v.split(",").forEach(function (f) { used[f.trim().replace(/^["']|["']$/g, "").toLowerCase()] = 1; });
        if ((p === 'stroke-opacity' || p === 'fill-opacity') && d.hasAttribute(p)) return;
        d.setAttribute(p, v);
      });
      d.removeAttribute('class');
      d.removeAttribute('style');
    });
    c.setAttribute('x', n(r.left - ox));
    c.setAttribute('y', n(r.top - oy));
    c.setAttribute('width', n(r.width));
    c.setAttribute('height', n(r.height));
    c.setAttribute('overflow', 'visible');
    if (op < 0.995) c.setAttribute('opacity', n(op));
    return new XMLSerializer().serializeToString(c).replace(/ xmlns="http:\/\/www\.w3\.org\/2000\/svg"/, '');
  }
  function image(el, op) {
    var r = el.getBoundingClientRect();
    if (!r.width || !r.height) return '';
    var src = el.currentSrc || el.src;
    if (!/^data:/.test(src)) {
      try {
        var c = document.createElement('canvas');
        c.width = el.naturalWidth; c.height = el.naturalHeight;
        c.getContext('2d').drawImage(el, 0, 0);
        src = c.toDataURL('image/png');
      } catch (e) { /* a tainted picture keeps its URL */ }
    }
    var fit = getComputedStyle(el).objectFit, par = fit === 'cover' ? 'xMidYMid slice' : fit === 'contain' || fit === 'scale-down' ? 'xMidYMid meet' : 'none';
    return '<image x="' + n(r.left - ox) + '" y="' + n(r.top - oy) + '" width="' + n(r.width) + '" height="' + n(r.height) + '" preserveAspectRatio="' + par + '"' +
      (op < 0.995 ? ' opacity="' + n(op) + '"' : '') + ' href="' + esc(src) + '"/>';
  }
  function walk(el, op) {
    var cs = getComputedStyle(el);
    if (cs.display === 'none') return '';
    op *= parseFloat(cs.opacity);
    if (op < 0.01) return '';
    var tag = el.tagName.toLowerCase();
    if (tag === 'svg') return cs.visibility === 'hidden' ? '' : svgClone(el, op);
    if (tag === 'script' || tag === 'style' || tag === 'template' || el.classList.contains('si-c3')) return ''; // CSS 3D: raster only (--png)
    var out = '', r = el.getBoundingClientRect(), rx = r.width ? radius(cs, r.width, r.height) : 0;
    if (cs.visibility !== 'hidden' && cs.display !== 'contents' && r.width && r.height) {
      out += shadows(cs, r, rx, op);
      var bg = paint(cs.backgroundColor, 'fill', op);
      if (bg) out += box(r, rx, bg);
      out += gradients(cs, r, rx, op);
      out += insetShadows(cs, r, rx, op);
      var sides = ['Top', 'Right', 'Bottom', 'Left'].map(function (s) {
        return { w: cs.borderStyle !== 'none' && cs['border' + s + 'Style'] !== 'none' ? parseFloat(cs['border' + s + 'Width']) || 0 : 0, c: cs['border' + s + 'Color'] };
      });
      var same = sides.every(function (s) { return s.w === sides[0].w && s.c === sides[0].c; });
      if (same && sides[0].w) {
        var st = paint(sides[0].c, 'stroke', op), w = sides[0].w;
        if (st) out += box({ left: r.left + w / 2, top: r.top + w / 2, width: r.width - w, height: r.height - w }, Math.max(0, rx - w / 2), 'fill="none" stroke-width="' + n(w) + '" ' + st);
      } else {
        sides.forEach(function (s, i) {
          var f = s.w && paint(s.c, 'fill', op);
          if (!f) return;
          var b = [{ left: r.left, top: r.top, width: r.width, height: s.w }, { left: r.right - s.w, top: r.top, width: s.w, height: r.height },
            { left: r.left, top: r.bottom - s.w, width: r.width, height: s.w }, { left: r.left, top: r.top, width: s.w, height: r.height }][i];
          out += box(b, 0, f);
        });
      }
      if (tag === 'img') return out + image(el, op);
    }
    var kids = '';
    for (var c = el.firstChild; c; c = c.nextSibling) {
      if (c.nodeType === 1) kids += walk(c, op);
      else if (c.nodeType === 3 && cs.visibility !== 'hidden') kids += text(c, cs, op);
    }
    if (kids && cs.display !== 'contents' && (cs.overflowX !== 'visible' || cs.overflowY !== 'visible') && el !== root) {
      var cid = 'si-c' + (++id), bw = function (s) { return parseFloat(cs['border' + s + 'Width']) || 0; };
      defs.push('<clipPath id="' + cid + '">' + box({ left: r.left + bw('Left'), top: r.top + bw('Top'), width: r.width - bw('Left') - bw('Right'), height: r.height - bw('Top') - bw('Bottom') },
        Math.max(0, rx - bw('Left')), '') + '</clipPath>');
      kids = '<g clip-path="url(#' + cid + ')">' + kids + '</g>';
    }
    return out + kids;
  }
  var ground = '';
  for (var a = root; a && !ground; a = a.parentElement) ground = paint(getComputedStyle(a).backgroundColor, 'fill', 1) || '';
  var body = walk(root, 1);
  // put the page back
  added.forEach(function (s) { s.remove(); });
  marks.forEach(function (el) { el.removeAttribute('data-si-pb'); el.removeAttribute('data-si-pa'); });
  hide.remove();
  var W = n(R.width), H = n(R.height);
  var fonts = fontFaces();
  return '<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="' + W + '" height="' + H + '" viewBox="0 0 ' + W + ' ' + H + '">' +
    (defs.length || fonts ? "<defs>" + fonts + defs.join("") + "</defs>" : "") +
    (ground ? '<rect width="100%" height="100%" ' + ground + '/>' : '') + body + '</svg>\n';
};
