/* simple-ideas: a vector SVG of a rendered element (si build --svg for
   timelines and infographics). Evaluated in headless Chrome on the built page,
   never shipped in one. It walks the laid-out DOM and writes what it paints:
   backgrounds and borders as rects (rounded), pictures as <image> (data URIs),
   inline SVG (the timeline's line, charts) cloned with its computed styles,
   and text as <text>, one per line as the browser broke it, at its baseline,
   with textLength so it keeps its width in any viewer. Colours are resolved to
   rgb() through a 1×1 canvas (oklch, color-mix and friends included);
   ::before/::after become real elements for the walk so their boxes count.
   Linear and radial background gradients become SVG gradients, outer box
   shadows blurred rects (an feGaussianBlur filter) under their box, and the
   @font-face rules of the families used go in with their files as data URIs.
   ponytail: no font subsetting (whole files), no conic or repeating gradients,
   no inset or text shadows; z-index is DOM order. */
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
      // a stretched pseudo-element reports its used size: copied, it can push
      // its parent bigger; then let it size itself again
      [['height', 'block-size'], ['width', 'inline-size']].forEach(function (ps) {
        var now = el.getBoundingClientRect();
        if (Math.abs(now.height - was.height) > 0.5 || Math.abs(now.width - was.width) > 0.5) ps.forEach(function (p) { s.style.removeProperty(p); });
      });
      el.setAttribute(w === 'before' ? 'data-si-pb' : 'data-si-pa', '');
      added.push(s);
      marks.push(el);
    });
  });
  document.head.appendChild(hide);

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
    return st.map(function (s) {
      return '<stop offset="' + n(s.o) + '" stop-color="rgb(' + s.c[0] + ',' + s.c[1] + ',' + s.c[2] + ')"' + (s.c[3] < 255 ? ' stop-opacity="' + n(s.c[3] / 255) + '"' : '') + '/>';
    }).join('');
  }
  var sides = { top: [0, -1], bottom: [0, 1], left: [-1, 0], right: [1, 0] };
  /* background-image gradients as SVG gradients over the box, bottom layer first */
  function gradients(cs, r, rx, op) {
    var img = cs.backgroundImage;
    if (!img || img === 'none') return '';
    var out = '', x = r.left - ox, y = r.top - oy, w = r.width, h = r.height;
    split(img, ',').reverse().forEach(function (layer) {
      var m = /^(linear|radial)-gradient\(([\s\S]*)\)$/.exec(layer);
      if (!m) return;
      var args = split(m[2], ','), gid = 'si-g' + (++id), g;
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
    });
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
    return lines.map(function (l) {
      var s = transform(t.slice(l.a, l.b).replace(/\s+/g, ' '), cs.textTransform) + (l.hy ? '-' : '');
      var y = l.top + (l.h - asc - desc) / 2 + asc;
      return '<text x="' + n(l.left - ox) + '" y="' + n(y - oy) + '" textLength="' + n(l.right - l.left) + '" lengthAdjust="spacingAndGlyphs"' +
        font + ' ' + fill + '>' + esc(s) + '</text>';
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
    if (tag === 'script' || tag === 'style' || tag === 'template') return '';
    var out = '', r = el.getBoundingClientRect(), rx = r.width ? radius(cs, r.width, r.height) : 0;
    if (cs.visibility !== 'hidden' && cs.display !== 'contents' && r.width && r.height) {
      out += shadows(cs, r, rx, op);
      var bg = paint(cs.backgroundColor, 'fill', op);
      if (bg) out += box(r, rx, bg);
      out += gradients(cs, r, rx, op);
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
