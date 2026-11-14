/* simple-ideas: a vector SVG of a rendered element (si build --svg for
   timelines and infographics). Evaluated in headless Chrome on the built page,
   never shipped in one. It walks the laid-out DOM and writes what it paints:
   backgrounds and borders as rects (rounded), pictures as <image> (data URIs),
   inline SVG (the timeline's line, charts) cloned with its computed styles,
   and text as <text>, one per line as the browser broke it, at its baseline,
   with textLength so it keeps its width in any viewer. Colours are resolved to
   rgb() through a 1×1 canvas (oklch, color-mix and friends included);
   ::before/::after become real elements for the walk so their boxes count.
   ponytail: gradients and box shadows are dropped (the flat colour under a
   gradient is kept); z-index is DOM order. */
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
    'stroke-dasharray', 'stroke-dashoffset', 'vector-effect', 'font-family', 'font-size', 'font-weight', 'text-anchor', 'dominant-baseline', 'display', 'visibility'];
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
      var bg = paint(cs.backgroundColor, 'fill', op);
      if (bg) out += box(r, rx, bg);
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
  return '<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="' + W + '" height="' + H + '" viewBox="0 0 ' + W + ' ' + H + '">' +
    (defs.length ? '<defs>' + defs.join('') + '</defs>' : '') +
    (ground ? '<rect width="100%" height="100%" ' + ground + '/>' : '') + body + '</svg>\n';
};
