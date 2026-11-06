/* si mind maps in 3D: three.js (r128, inlined before this file only on pages
   with a 3D map) draws the maps of a .si-mindmap as pills on a floor. Topics
   are rounded blocks whose height says how close they are to the centre; lines
   are cables on the floor with flowing dots; linked maps hover as glass floors
   above the topic that opens them. mindmap.js owns the data, the chrome
   (breadcrumb, tools, search box, Play position) and the 2D view; it creates
   one Mindmap3D per map when the map starts in 3D or the viewer presses 3D.
   Colours and fonts come from the map's --si-mm-* tokens. Three kinds share
   the layout, search, Play and drill-down: blocks (pills), bricks (studded
   bricks on a baseplate, labels printed on the front) and toy (cartoon shapes
   with outlines on a floating island). */
(function () {
  'use strict';
  if (!window.THREE || !THREE.OrbitControls) return;
  var reduce = window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  var S = 200; /* texture pixels per world unit */
  var LV = [{ fs: 60, fw: 700, d: 1.4, h: 0.95, pad: 46 }, { fs: 46, fw: 600, d: 1.02, h: 0.62, pad: 34 }, { fs: 38, fw: 500, d: 0.82, h: 0.36, pad: 28 }];
  var HG = 1.25, GAP = 0.42, FH = 20;
  var ISO = new THREE.Vector3(1, 1, 1).normalize(), PERSP = new THREE.Vector3(0.75, 0.95, 1.25).normalize();
  var meas = document.createElement('canvas').getContext('2d');
  var probe = document.createElement('canvas'); probe.width = probe.height = 1;
  var pctx = probe.getContext('2d');
  var images = {};

  function lv(d) { return LV[Math.min(d, 2)]; }
  function vis(n) { return n.collapsed ? [] : (n.kids || []); }
  function rr(c, x, y, w, h, r) { c.beginPath(); c.moveTo(x + r, y); c.arcTo(x + w, y, x + w, y + h, r); c.arcTo(x + w, y + h, x, y + h, r); c.arcTo(x, y + h, x, y, r); c.arcTo(x, y, x + w, y, r); c.closePath(); }
  /* colours are written in sRGB; three.js lights in linear space */
  function col(h) { return new THREE.Color(h).convertSRGBToLinear(); }
  function mix(a, b, t) { return '#' + new THREE.Color(a).lerp(new THREE.Color(b), t).getHexString(); }
  function lum(h) { var c = new THREE.Color(h); return 0.2126 * c.r + 0.7152 * c.g + 0.0722 * c.b; }
  function ease(p) { return p < 0.5 ? 4 * p * p * p : 1 - Math.pow(-2 * p + 2, 3) / 2; }
  /* any CSS colour (var(), color-mix(), rgb) as #rrggbb, by painting one pixel */
  function colorOf(el, expr) {
    var s = document.createElement('span');
    s.style.color = expr; s.style.display = 'none'; el.appendChild(s);
    var c = getComputedStyle(s).color; el.removeChild(s);
    pctx.clearRect(0, 0, 1, 1); pctx.fillStyle = '#808080'; pctx.fillStyle = c; pctx.fillRect(0, 0, 1, 1);
    var d = pctx.getImageData(0, 0, 1, 1).data;
    return '#' + [d[0], d[1], d[2]].map(function (x) { return (x + 256).toString(16).slice(1); }).join('');
  }
  function image(src, then) {
    var im = images[src];
    if (!im) { im = images[src] = new Image(); im.decoding = 'async'; im.src = src; }
    if (im.complete && im.naturalWidth) return im;
    im.addEventListener('load', then, { once: true });
    return null;
  }
  /* the highlighted HTML of a code topic as lines of runs {t, k} (k: si-tk-* kind) */
  function codeRuns(html) {
    var d = document.createElement('div'), lines = [[]];
    d.innerHTML = html;
    (function walk(node, k) {
      node.childNodes.forEach(function (c) {
        if (c.nodeType === 3) {
          c.nodeValue.replace(/\t/g, '    ').split('\n').forEach(function (p, i) { if (i) lines.push([]); if (p) lines[lines.length - 1].push({ t: p, k: k }); });
        } else if (c.nodeType === 1) walk(c, (c.className || '').replace('si-tk-', ''));
      });
    })(d, '');
    return lines;
  }

  function Mindmap3D(mm) {
    var self = this;
    this.mm = mm; this.el = mm.el; this.maps = mm.maps; this.st = mm.st;
    var gl = document.createElement('canvas');
    if (!window.WebGLRenderingContext || !(gl.getContext('webgl') || gl.getContext('experimental-webgl'))) { this.ok = false; return; }
    try { this.renderer = new THREE.WebGLRenderer({ antialias: true }); } catch (e) { this.ok = false; return; }
    this.ok = true;
    var r = this.renderer;
    r.setPixelRatio(Math.min(window.devicePixelRatio || 1, 2));
    r.outputEncoding = THREE.sRGBEncoding;
    r.shadowMap.enabled = true; r.shadowMap.type = THREE.PCFSoftShadowMap;
    this.box = document.createElement('div'); this.box.className = 'si-mm-3d'; this.box.hidden = true;
    this.box.appendChild(r.domElement);
    this.el.insertBefore(this.box, mm.crumbs);
    this.scene = new THREE.Scene();
    this.ortho = new THREE.OrthographicCamera(-FH, FH, FH / 2, -FH / 2, 0.1, 400);
    this.persp = new THREE.PerspectiveCamera(32, 1, 0.1, 600);
    this.cam = mm.opt('camera') === 'perspective' ? 'persp' : 'iso';
    this.camera = this.cam === 'iso' ? this.ortho : this.persp;
    this.floors = mm.opt('floors') !== 'off';
    this.kind = mm.opt('kind') || 'blocks';
    this.deco = new THREE.Group(); this.clouds = []; /* toy scenery: left out of Fit */
    this.hemi = new THREE.HemisphereLight(0xffffff, 0x9aa6b8, 0.95); this.scene.add(this.hemi);
    var sun = this.sun = new THREE.DirectionalLight(0xffffff, 0.55);
    sun.castShadow = true; sun.shadow.mapSize.set(2048, 2048); sun.shadow.bias = -0.0004;
    this.scene.add(sun); this.scene.add(sun.target);
    this.ground = new THREE.Mesh(new THREE.PlaneGeometry(600, 600), new THREE.ShadowMaterial({ opacity: 0.13 }));
    this.ground.rotation.x = -Math.PI / 2; this.ground.receiveShadow = true; this.scene.add(this.ground);
    this.world = new THREE.Group(); this.scene.add(this.world); this.scene.add(this.deco);
    this.picks = []; this.flows = []; this.blocks = []; this.anims = []; this.tweens = [];
    this.seen = true; this.on = false; this.tour = null;
    /* an embedded map (a slide, an article): the wheel scrolls the page; pinch or Ctrl zooms */
    r.domElement.addEventListener('wheel', function (e) { if (!mm.full && !e.ctrlKey && !e.metaKey) e.stopImmediatePropagation(); }, { capture: true });
    this.makeControls();
    this.card = document.createElement('div'); this.card.className = 'si-mm-card'; this.card.hidden = true;
    this.card.setAttribute('role', 'dialog'); this.card.setAttribute('aria-label', 'Topic');
    this.el.appendChild(this.card);
    this.pointer();
    if (window.ResizeObserver) new ResizeObserver(function () { self.resize(); }).observe(this.box);
    if (window.IntersectionObserver) new IntersectionObserver(function (es) { self.seen = es[es.length - 1].isIntersecting; }).observe(this.el);
    if (document.fonts && document.fonts.ready) document.fonts.ready.then(function () { if (self.on) self.rebuild(false); });
    var restyle = function () { if (self.on) self.rebuild(false); };
    new MutationObserver(restyle).observe(document.documentElement, { attributes: true, attributeFilter: ['data-si-mode', 'data-si-theme'] });
    if (window.matchMedia) { var mq = window.matchMedia('(prefers-color-scheme: dark)'); if (mq.addEventListener) mq.addEventListener('change', restyle); }
    requestAnimationFrame(function frame(now) { requestAnimationFrame(frame); self.frame(now); });
  }
  var P = Mindmap3D.prototype;

  P.makeControls = function () {
    if (this.controls) this.controls.dispose();
    var c = this.controls = new THREE.OrbitControls(this.camera, this.renderer.domElement);
    c.enableDamping = true; c.dampingFactor = 0.08; c.screenSpacePanning = true; c.maxPolarAngle = 1.32;
    c.minZoom = 0.25; c.maxZoom = 8;
  };
  P.resize = function () {
    var w = this.box.clientWidth, h = this.box.clientHeight;
    if (!w || !h) return;
    var a = w / h;
    this.renderer.setSize(w, h, false);
    this.ortho.left = -FH * a / 2; this.ortho.right = FH * a / 2; this.ortho.top = FH / 2; this.ortho.bottom = -FH / 2; this.ortho.updateProjectionMatrix();
    this.persp.aspect = a; this.persp.updateProjectionMatrix();
  };

  /* the map's look, from its --si-mm-* tokens (the deck theme, dark mode, or a map theme) */
  P.theme = function () {
    var el = this.el, css = function (k) { return getComputedStyle(el).getPropertyValue(k).trim(); };
    var c = function (v, f) { return colorOf(el, 'var(' + v + ', ' + f + ')'); };
    var T = {
      bg: c('--si-mm-bg', '#ffffff'), text: c('--si-mm-fg', '#141821'), muted: c('--si-mm-muted', '#5b6474'), top: c('--si-mm-surface', '#ffffff'),
      accent: c('--si-mm-accent', '#2f5bd3'), on: c('--si-mm-on-accent', '#ffffff'), sticky: c('--si-mm-sticky', '#ffe27a'), stickyFg: c('--si-mm-sticky-fg', '#3d3000'),
      b: [1, 2, 3, 4, 5].map(function (i) { return c('--si-mm-' + i, '#2f5bd3'); }),
      font: css('--si-mm-font') || 'sans-serif', mono: css('--si-mm-mono') || 'monospace', hand: css('--si-mm-hand') || 'cursive'
    };
    T.glow = lum(T.bg) < 0.35 ? 0.5 : 0;
    return T;
  };

  /* ---------- sizes and the text on the top faces ---------- */
  P.size = function (n, d, T) {
    if (n.sticky) {
      meas.font = '600 52px ' + T.hand;
      var line = '', out = [];
      n.t.split(' ').forEach(function (w) { var t = line ? line + ' ' + w : w; if (meas.measureText(t).width > 400 && line) { out.push(line); line = w; } else line = t; });
      if (line) out.push(line);
      n.lines = out; n.w = 2.3; n.d = 0.42 + out.length * 0.27; n.h = 0.05; return;
    }
    if (n.code) {
      meas.font = '30px ' + T.mono;
      n.runs = codeRuns(n.code.html);
      var mw = 0; n.runs.forEach(function (l) { mw = Math.max(mw, meas.measureText(l.map(function (r) { return r.t; }).join('')).width); });
      n.w = Math.max(1.6, mw / S + 0.42); n.d = n.runs.length * 0.2 + 0.46; n.h = 0.2; return;
    }
    var L = lv(d);
    meas.font = L.fw + ' ' + L.fs + 'px ' + T.font;
    n.tw = meas.measureText(n.t).width;
    n.chips = [];
    if (n.badge) n.chips.push({ k: 'badge', t: n.badge });
    if (n.href) n.chips.push({ k: 'href', t: n.chip || 'link' });
    if (n.link) n.chips.push({ k: 'link', t: 'map ›' });
    if (n.collapsed && n.kids && n.kids.length) n.chips.push({ k: 'more', t: '+' + n.kids.length });
    meas.font = '600 26px ' + T.font;
    var cw = 0; n.chips.forEach(function (c) { c.w = meas.measureText(c.t).width + 30; cw += c.w + 14; });
    n.w = (n.tw + L.pad * 2 + cw) / S + (n.img ? 1.05 : 0) + L.d * 0.55; n.d = L.d; n.h = L.h;
  };
  P.topTexture = function (n, k, T) {
    var W = Math.round(n.w * S), H = Math.round(n.d * S), cv = document.createElement('canvas');
    cv.width = W; cv.height = H;
    var tex = new THREE.CanvasTexture(cv);
    tex.encoding = THREE.sRGBEncoding; tex.anisotropy = this.renderer.capabilities.getMaxAnisotropy();
    tex.repeat.set(1 / n.w, 1 / n.d); tex.offset.set(0.5, 0.5);
    var draw = function () {
      var c = cv.getContext('2d');
      c.fillStyle = k.top; c.fillRect(0, 0, W, H);
      if (n.sticky) {
        c.fillStyle = T.stickyFg; c.font = '600 52px ' + T.hand; c.textBaseline = 'middle';
        n.lines.forEach(function (l, i) { c.fillText(l, 34, 44 + i * 54); });
      } else if (n.code) {
        c.textBaseline = 'middle';
        var tone = { k: T.accent, m: T.accent, s: T.b[2], f: T.b[2], y: T.b[2], n: T.b[1], v: T.b[1], c: T.muted };
        n.runs.forEach(function (line, i) {
          var x = 40, y = 50 + i * 40;
          line.forEach(function (r) {
            c.font = (r.k === 'c' ? 'italic ' : r.k === 'k' || r.k === 'm' ? '600 ' : '') + '30px ' + T.mono;
            c.fillStyle = tone[r.k] || T.text; c.fillText(r.t, x, y); x += c.measureText(r.t).width;
          });
        });
        c.fillStyle = T.muted; c.font = '600 20px ' + T.mono; c.textAlign = 'right'; c.fillText((n.code.lang || 'code').toUpperCase(), W - 24, 26); c.textAlign = 'left';
      } else {
        var L = lv(n.dd), x = L.pad + L.d * 0.275 * S;
        if (n.img) {
          var im = image(n.img, function () { draw(); tex.needsUpdate = true; });
          if (im) {
            c.save(); rr(c, x - 24, H / 2 - 66, 180, 132, 66); c.clip();
            var s = Math.max(180 / im.naturalWidth, 132 / im.naturalHeight), iw = im.naturalWidth * s, ih = im.naturalHeight * s;
            c.drawImage(im, x - 24 + (180 - iw) / 2, H / 2 - 66 + (132 - ih) / 2, iw, ih); c.restore();
          }
          x += 176;
        }
        c.fillStyle = k.text; c.font = L.fw + ' ' + L.fs + 'px ' + T.font; c.textBaseline = 'middle';
        c.fillText(n.t, x, H / 2 + 2);
        x += n.tw + 16;
        c.font = '600 26px ' + T.font;
        n.chips.forEach(function (ch) {
          var y = H / 2 - 22;
          if (ch.k === 'link') { c.fillStyle = T.accent; rr(c, x, y, ch.w, 44, 22); c.fill(); c.fillStyle = T.on; }
          else if (ch.k === 'href') { c.strokeStyle = k.edge; c.lineWidth = 3; rr(c, x, y, ch.w, 44, 22); c.stroke(); c.fillStyle = k.edge; }
          else if (ch.k === 'more') { c.fillStyle = k.edge; rr(c, x, y, ch.w, 44, 22); c.fill(); c.fillStyle = T.bg; }
          else { c.fillStyle = k.badge; rr(c, x, y, ch.w, 44, 22); c.fill(); c.fillStyle = k.text; }
          c.textAlign = 'center'; c.fillText(ch.t, x + ch.w / 2, H / 2 + 1); c.textAlign = 'left';
          x += ch.w + 14;
        });
      }
    };
    draw();
    return tex;
  };

  /* ---------- layout: branches to both sides, on the floor (x, z) ---------- */
  function spanZ(n) { var k = vis(n); if (!k.length) return (n.sz = n.d); var s = 0; k.forEach(function (c) { s += spanZ(c); }); s += GAP * (k.length - 1); return (n.sz = Math.max(n.d, s)); }
  function kidsSpan(k) { var s = 0; k.forEach(function (c) { s += c.sz; }); return s + GAP * Math.max(0, k.length - 1); }
  function place(n, x, top, side) {
    n.x = x; n.z = top + n.sz / 2; n.side = side;
    var k = vis(n), z = top + (n.sz - kidsSpan(k)) / 2;
    k.forEach(function (c) { place(c, x + side * (n.w / 2 + HG + c.w / 2), z, side); z += c.sz + GAP; });
  }
  function layout(root) {
    root.x = 0; root.z = 0; root.side = 1;
    var k = vis(root); k.forEach(spanZ);
    var total = kidsSpan(k), acc = 0, R = [], Lf = [];
    k.forEach(function (c) { (acc + c.sz / 2 < total / 2 ? R : Lf).push(c); acc += c.sz + GAP; });
    if (!Lf.length && R.length > 1) Lf.push(R.pop());
    [[R, 1], [Lf, -1]].forEach(function (g) { var z = -kidsSpan(g[0]) / 2; g[0].forEach(function (c) { place(c, g[1] * (root.w / 2 + HG + c.w / 2), z, g[1]); z += c.sz + GAP; }); });
  }

  /* ---------- one map as blocks and cables ---------- */
  function pill(n) {
    var w = n.w, d = n.d, r = n.sticky || n.code ? Math.min(0.12, d * 0.2) : d / 2 - 0.001, sh = new THREE.Shape();
    sh.moveTo(-w / 2 + r, -d / 2); sh.lineTo(w / 2 - r, -d / 2); sh.quadraticCurveTo(w / 2, -d / 2, w / 2, -d / 2 + r);
    sh.lineTo(w / 2, d / 2 - r); sh.quadraticCurveTo(w / 2, d / 2, w / 2 - r, d / 2); sh.lineTo(-w / 2 + r, d / 2);
    sh.quadraticCurveTo(-w / 2, d / 2, -w / 2, d / 2 - r); sh.lineTo(-w / 2, -d / 2 + r); sh.quadraticCurveTo(-w / 2, -d / 2, -w / 2 + r, -d / 2);
    var g = new THREE.ExtrudeGeometry(sh, { depth: n.h, bevelEnabled: true, bevelThickness: 0.07, bevelSize: 0.06, bevelSegments: 5, curveSegments: 18 });
    g.rotateX(-Math.PI / 2);
    return g;
  }
  P.curve = function (a, b) {
    var V = THREE.Vector3, lines = this.mm.opt('lines');
    if (lines === 'straight') return new THREE.LineCurve3(a, b);
    if (lines === 'orthogonal') {
      var mx = (a.x + b.x) / 2, p = new THREE.CurvePath(), r = Math.min(0.3, Math.abs(b.z - a.z) / 2, Math.abs(mx - a.x)), s = Math.sign(b.x - a.x), sz = Math.sign(b.z - a.z);
      if (Math.abs(b.z - a.z) < 0.01) { p.add(new THREE.LineCurve3(a, b)); return p; }
      p.add(new THREE.LineCurve3(a, new V(mx - s * r, a.y, a.z)));
      p.add(new THREE.QuadraticBezierCurve3(new V(mx - s * r, a.y, a.z), new V(mx, a.y, a.z), new V(mx, a.y, a.z + sz * r)));
      p.add(new THREE.LineCurve3(new V(mx, a.y, a.z + sz * r), new V(mx, a.y, b.z - sz * r)));
      p.add(new THREE.QuadraticBezierCurve3(new V(mx, a.y, b.z - sz * r), new V(mx, a.y, b.z), new V(mx + s * r, a.y, b.z)));
      p.add(new THREE.LineCurve3(new V(mx + s * r, a.y, b.z), b));
      return p;
    }
    var m = (b.x - a.x) / 2; /* curved; tapered draws curved in 3D */
    return new THREE.CubicBezierCurve3(a, new V(a.x + m, a.y, a.z), new V(b.x - m, b.y, b.z), b);
  };
  /* ---------- bricks: a stud pitch, studded bodies, printed fronts, a baseplate ---------- */
  var PITCH = 0.4, studGeo = null, tileGeo = null;
  function snap(n, d) {
    n.w = Math.ceil((n.w + 0.1) / PITCH) * PITCH; n.d = Math.max(2 * PITCH, Math.ceil(n.d / PITCH) * PITCH);
    n.tile = !!(n.sticky || n.code); n.h = n.tile ? 0.16 : [1.0, 0.8, 0.6][Math.min(d, 2)];
  }
  function plastic(hex, T) { return new THREE.MeshStandardMaterial({ color: col(hex), roughness: 0.34, metalness: 0, emissive: col(hex), emissiveIntensity: T.glow * 0.12 }); }
  function inkOn(hex) { return lum(hex) > 0.45 ? '#1b1f27' : '#ffffff'; }
  P.frontTexture = function (n, body, T) {
    var W = Math.round(n.w * S), H = Math.round(n.h * S), cv = document.createElement('canvas'); cv.width = W; cv.height = H;
    var tex = new THREE.CanvasTexture(cv); tex.encoding = THREE.sRGBEncoding; tex.anisotropy = this.renderer.capabilities.getMaxAnisotropy();
    var draw = function () {
      var c = cv.getContext('2d'), L = lv(n.dd), fs = Math.min(L.fs * 1.15, H * 0.5), ink = inkOn(body), x = W * 0.08;
      c.fillStyle = body; c.fillRect(0, 0, W, H);
      if (n.img) {
        var im = image(n.img, function () { draw(); tex.needsUpdate = true; }), ih = H * 0.64, iw = ih * 1.4;
        if (im) { c.save(); rr(c, x, H * 0.18, iw, ih, 10); c.clip(); var s = Math.max(iw / im.naturalWidth, ih / im.naturalHeight); c.drawImage(im, x + (iw - im.naturalWidth * s) / 2, H * 0.18 + (ih - im.naturalHeight * s) / 2, im.naturalWidth * s, im.naturalHeight * s); c.restore(); }
        x += iw + 18;
      }
      c.fillStyle = ink; c.font = L.fw + ' ' + fs + 'px ' + T.font; c.textBaseline = 'middle'; c.fillText(n.t, x, H / 2 + 2);
      x += c.measureText(n.t).width + 14;
      c.font = '700 ' + Math.round(fs * 0.55) + 'px ' + T.font;
      (n.chips || []).forEach(function (ch) {
        var cw = c.measureText(ch.t).width + 24, chh = fs * 0.85;
        c.fillStyle = ch.k === 'link' ? ink : 'rgba(255,255,255,.3)'; rr(c, x, H / 2 - chh / 2, cw, chh, chh / 2); c.fill();
        c.fillStyle = ch.k === 'link' ? body : ink; c.textAlign = 'center'; c.fillText(ch.t, x + cw / 2, H / 2 + 1); c.textAlign = 'left'; x += cw + 10;
      });
    };
    draw();
    return tex;
  };
  P.brick = function (n, k, T) {
    if (!studGeo) { studGeo = new THREE.CylinderGeometry(PITCH * 0.3, PITCH * 0.3, PITCH * 0.18, 28); tileGeo = new THREE.CylinderGeometry(0.11, 0.11, 0.09, 24); }
    var b = T.b[n.branch], body = n.dd === 0 ? T.accent : n.sticky ? T.sticky : n.code ? mix(T.top, T.text, 0.12) : n.dd === 1 ? b : mix(b, '#ffffff', T.glow ? 0.25 : 0.42);
    var mat = plastic(body, T), mats;
    if (n.tile) { var tex = this.topTexture(n, k, T); tex.repeat.set(1, 1); tex.offset.set(0, 0); mats = [mat, mat, new THREE.MeshStandardMaterial({ map: tex, roughness: 0.4 }), mat, mat, mat]; }
    else mats = [mat, mat, mat, mat, new THREE.MeshStandardMaterial({ map: this.frontTexture(n, body, T), roughness: 0.34 }), mat];
    var mesh = new THREE.Mesh(new THREE.BoxGeometry(n.w - 0.02, n.h, n.d - 0.02).translate(0, n.h / 2, 0), mats);
    if (!n.tile) { /* the studs share the body's material, so Play dims them with it */
      var cols = Math.round(n.w / PITCH), rows = Math.round(n.d / PITCH), st = new THREE.InstancedMesh(studGeo, mat, cols * rows), m4 = new THREE.Matrix4(), i = 0;
      for (var x = 0; x < cols; x++) for (var z = 0; z < rows; z++) { m4.makeTranslation(-n.w / 2 + PITCH / 2 + x * PITCH, n.h + PITCH * 0.09, -n.d / 2 + PITCH / 2 + z * PITCH); st.setMatrixAt(i++, m4); }
      st.castShadow = true; mesh.add(st);
    }
    return mesh;
  };
  function studPath(curve, hex, T) {
    var count = Math.max(2, Math.floor(curve.getLength() / 0.27)), im = new THREE.InstancedMesh(tileGeo, plastic(hex, T), count), m4 = new THREE.Matrix4();
    for (var i = 0; i < count; i++) { var p = curve.getPointAt((i + 0.5) / count); m4.makeTranslation(p.x, 0.045, p.z); im.setMatrixAt(i, m4); }
    im.castShadow = true; im.userData.total = count;
    return im;
  }
  function baseplate(w, d, hex, k) {
    k = k || 1;
    var cols = Math.round(w / (PITCH * k)), rows = Math.round(d / (PITCH * k)), g = new THREE.Group(), mat = new THREE.MeshStandardMaterial({ color: col(hex), roughness: 0.5 });
    w = cols * PITCH * k; d = rows * PITCH * k;
    var plate = new THREE.Mesh(new THREE.BoxGeometry(w, 0.14 * k, d), mat); plate.position.y = -0.07 * k; plate.receiveShadow = true; g.add(plate);
    var st = new THREE.InstancedMesh(studGeo, mat, cols * rows), m4 = new THREE.Matrix4(), sc = new THREE.Matrix4().makeScale(k, k, k), i = 0;
    for (var x = 0; x < cols; x++) for (var z = 0; z < rows; z++) { m4.makeTranslation(-w / 2 + PITCH * k / 2 + x * PITCH * k, PITCH * 0.09 * k, -d / 2 + PITCH * k / 2 + z * PITCH * k).multiply(sc); st.setMatrixAt(i++, m4); }
    st.receiveShadow = true; g.add(st);
    return g;
  }

  /* ---------- toy: three-tone cartoon shading, outlines, candy colours, an island ---------- */
  var TOY = {
    day: { b: ['#ff5d73', '#ffb627', '#1fc8a0', '#4d8dff', '#b26bff'], root: '#ffcf3f', rootInk: '#2a2140', ink: '#22263a', paper: '#ffffff', outline: '#23263a',
      sky: ['#9fdcff', '#fff4dc'], grass: '#a6d98a', dirt: '#c98a52', leaf: ['#3fae5a', '#66c96b'], trunk: '#8a5a35', cloud: '#ffffff' },
    night: { b: ['#ff7a8e', '#ffc95c', '#3ee0b8', '#7aa8ff', '#c798ff'], root: '#ffd45c', rootInk: '#2a2140', ink: '#22263a', paper: '#2a3150', outline: '#05070f',
      sky: ['#14193a', '#3a2f63'], grass: '#3f7a52', dirt: '#5b3d2a', leaf: ['#2e6b48', '#3f8a5a'], trunk: '#4a3322', cloud: '#c9cfee' }
  };
  var toonGrad = null;
  function toon(hex, map) {
    if (!toonGrad) { toonGrad = new THREE.DataTexture(new Uint8Array([110, 110, 110, 190, 190, 190, 255, 255, 255]), 3, 1, THREE.RGBFormat); toonGrad.minFilter = toonGrad.magFilter = THREE.NearestFilter; toonGrad.needsUpdate = true; }
    return new THREE.MeshToonMaterial({ color: col(hex), map: map || null, gradientMap: toonGrad });
  }
  /* an ink outline: the shape again, a little bigger, drawn from the inside */
  function outlined(mesh, pad, hex) {
    var bb = new THREE.Box3().setFromBufferAttribute(mesh.geometry.attributes.position), sz = bb.getSize(new THREE.Vector3());
    var o = new THREE.Mesh(mesh.geometry, new THREE.MeshBasicMaterial({ color: col(hex), side: THREE.BackSide }));
    o.scale.set(1 + 2 * pad / Math.max(sz.x, 0.01), 1 + pad / Math.max(sz.y, 0.01), 1 + 2 * pad / Math.max(sz.z, 0.01));
    o.position.y = -pad * 0.5; mesh.add(o); mesh.userData.outline = o;
    return mesh;
  }
  function toySize(n, d) {
    n.shape = n.sticky || n.code ? 'note' : d === 0 ? 'drum' : n.link ? 'hex' : 'pill';
    if (n.shape === 'drum') { n.w *= 0.92; n.d = n.w; n.h = 1.05; }
    else if (n.shape === 'hex') { n.w *= 1.08; n.d = n.w * 0.866; n.h = 0.62; }
    else if (n.shape === 'pill') n.h = d === 1 ? 0.72 : 0.42;
  }
  P.roundTexture = function (n, k, T) {
    var R = Math.round(n.w * S), cv = document.createElement('canvas'); cv.width = cv.height = R;
    var c = cv.getContext('2d'), L = lv(n.dd);
    c.fillStyle = k.top; c.fillRect(0, 0, R, R);
    c.fillStyle = k.text; c.font = L.fw + ' ' + L.fs + 'px ' + T.font; c.textAlign = 'center'; c.textBaseline = 'middle';
    c.fillText(n.t, R / 2, R / 2 - (n.link ? 18 : 0));
    if (n.link) { c.font = '700 28px ' + T.font; var cw = c.measureText('map ›').width + 34; rr(c, R / 2 - cw / 2, R / 2 + 22, cw, 46, 23); c.fill(); c.fillStyle = k.top; c.fillText('map ›', R / 2, R / 2 + 46); }
    var tex = new THREE.CanvasTexture(cv); tex.encoding = THREE.sRGBEncoding; tex.anisotropy = this.renderer.capabilities.getMaxAnisotropy();
    /* the cap's texture runs along the world axes; turn it to read upright from the camera (hexagons are turned 30° more) */
    tex.center.set(0.5, 0.5); tex.rotation = 3 * Math.PI / 4 + (n.shape === 'hex' ? -Math.PI / 6 : 0);
    return tex;
  };
  P.toyMesh = function (n, T) {
    var K = T.glow ? TOY.night : TOY.day, b = K.b[n.branch];
    var body = n.dd === 0 ? K.root : n.sticky ? T.sticky : n.code ? K.paper : n.dd === 1 ? b : mix(b, '#ffffff', 0.55);
    var k = { top: n.dd === 0 ? mix(K.root, '#ffffff', 0.25) : n.sticky ? mix(T.sticky, '#ffffff', 0.3) : n.code ? K.paper : n.dd === 1 ? mix(b, '#ffffff', 0.62) : mix(b, '#ffffff', 0.8),
      text: n.dd === 0 ? K.rootInk : K.ink, edge: b, badge: mix(b, '#ffffff', 0.45) };
    var TT = Object.create(T); TT.accent = K.ink; TT.on = '#ffffff'; TT.text = n.code ? (T.glow ? '#e8ecff' : K.ink) : K.ink;
    var mesh;
    if (n.shape === 'drum' || n.shape === 'hex') {
      var rad = n.w / 2, g = new THREE.CylinderGeometry(rad, rad * 1.02, n.h, n.shape === 'hex' ? 6 : 64, 1).translate(0, n.h / 2, 0);
      if (n.shape === 'hex') g.rotateY(Math.PI / 6);
      mesh = new THREE.Mesh(g, [toon(body), toon('#ffffff', this.roundTexture(n, k, TT)), toon(body)]);
    } else mesh = new THREE.Mesh(pill(n), [toon('#ffffff', this.topTexture(n, k, TT)), toon(body)]);
    return outlined(mesh, 0.045, K.outline);
  };
  function island(rx, rz, K, k) {
    k = k || 1;
    var g = new THREE.Group(), land = new THREE.Mesh(new THREE.CylinderGeometry(1, 0.86, 0.7 * k, 72, 1), [toon(K.dirt), toon(K.grass), toon(K.dirt)]);
    land.scale.set(rx, 1, rz); land.position.y = -0.35 * k; land.receiveShadow = true; g.add(land);
    var rim = new THREE.Mesh(new THREE.CylinderGeometry(1.012, 1.012, 0.08 * k, 72, 1, true), toon(mix(K.grass, '#000000', 0.18)));
    rim.scale.set(rx, 1, rz); rim.position.y = -0.04 * k; g.add(rim);
    return g;
  }
  function tree(x, z, K, s) {
    var t = new THREE.Group();
    [[new THREE.CylinderGeometry(0.07, 0.09, 0.42, 10).translate(0, 0.21, 0), K.trunk, 0.02], [new THREE.ConeGeometry(0.42, 0.75, 12).translate(0, 0.72, 0), K.leaf[0], 0.03], [new THREE.ConeGeometry(0.32, 0.6, 12).translate(0, 1.08, 0), K.leaf[1], 0.03]]
      .forEach(function (p) { var m = outlined(new THREE.Mesh(p[0], toon(p[1])), p[2], K.outline); m.castShadow = true; t.add(m); });
    t.position.set(x, 0, z); t.scale.setScalar(s);
    return t;
  }
  P.cloud = function (x, y, z, s, K) {
    var g = new THREE.Group();
    [[0, 0, 0, 0.55], [0.55, -0.08, 0.1, 0.42], [-0.55, -0.1, -0.05, 0.4], [0.2, 0.25, -0.1, 0.38]].forEach(function (p) {
      var m = outlined(new THREE.Mesh(new THREE.SphereGeometry(p[3], 20, 14), toon(K.cloud)), 0.025, K.outline); m.position.set(p[0], p[1], p[2]); g.add(m);
    });
    g.position.set(x, y, z); g.scale.setScalar(s); g.userData = { bob: Math.random() * 6, y: y }; this.clouds.push(g);
    return g;
  };
  function sky(a, b) {
    var cv = document.createElement('canvas'); cv.width = 4; cv.height = 256;
    var c = cv.getContext('2d'), gr = c.createLinearGradient(0, 0, 0, 256); gr.addColorStop(0, a); gr.addColorStop(1, b); c.fillStyle = gr; c.fillRect(0, 0, 4, 256);
    return new THREE.CanvasTexture(cv);
  }
  P.scenery = function (T) {
    var self = this, d = this.deco, K = T.glow ? TOY.night : TOY.day, all = this.main.userData.all;
    var tb = new THREE.Box3().setFromObject(this.main), ts = tb.getSize(new THREE.Vector3()), tc = tb.getCenter(new THREE.Vector3());
    if (this.kind === 'bricks') { var bp = baseplate(ts.x + 1.6, ts.z + 1.6, T.glow ? mix(T.bg, '#ffffff', 0.08) : mix(T.bg, '#000000', 0.06)); bp.position.set(tc.x, 0, tc.z); d.add(bp); return; }
    if (this.kind !== 'toy') return;
    var rx = ts.x / 2 + 2.4, rz = ts.z / 2 + 2.4, isl = island(rx, rz, K); isl.position.set(tc.x, 0, tc.z); d.add(isl);
    var taken = all.map(function (n) { return new THREE.Box3(new THREE.Vector3(n.x - n.w / 2 - 0.6, -1, n.z - n.d / 2 - 0.6), new THREE.Vector3(n.x + n.w / 2 + 0.6, 2, n.z + n.d / 2 + 0.6)); });
    for (var q = 0; q < 22; q++) { /* trees around the rim, never on a topic */
      var an = q / 22 * Math.PI * 2 + 0.2, p = new THREE.Vector3(tc.x + Math.cos(an) * rx * 0.9, 0, tc.z + Math.sin(an) * rz * 0.87);
      if (!taken.some(function (b) { return b.containsPoint(p); })) d.add(tree(p.x, p.z, K, 0.8 + (q % 3) * 0.18));
    }
    [[-1.3, 6, -1.2, 1.3], [0.5, 8.5, -1.6, 1], [1.4, 5, -0.7, 1.1], [-1.5, 4.5, -0.3, 0.9]] /* the far side: above the map on screen */.forEach(function (c) { d.add(self.cloud(tc.x + c[0] * rx, c[1], tc.z + c[2] * rz, c[3], K)); });
    this.scene.background = sky(K.sky[0], K.sky[1]);
  };
  P.setKind = function (kind) { this.kind = kind; this.closeCard(); if (this.tour) this.tourStop(true); this.rebuild(true); this.fit(700); };

  P.buildMap = function (id, T, anim) {
    var self = this, m = this.maps[id], root = m.root, all = [], group = new THREE.Group();
    (function prep(n, d, b, top, idp, path) {
      n.id = idp; n.dd = d; n.branch = b; n.top = top; n.path = path; self.size(n, d, T); all.push(n);
      if (self.kind === 'bricks') snap(n, d); else if (self.kind === 'toy') toySize(n, d);
      vis(n).forEach(function (c, i) { prep(c, d + 1, d === 0 ? i % 5 : b, d === 0 ? i : top, idp + '.' + i, path.concat(n.t)); });
    })(root, 0, 0, -1, 'r', []);
    layout(root);
    all.slice().sort(function (a, b) { return a.dd - b.dd; }).forEach(function (n, i) { n.i = i; });
    all.forEach(function (n) {
      var branch = T.b[n.branch], k = {
        top: n.dd === 0 ? T.accent : n.sticky ? T.sticky : n.code ? mix(T.top, T.text, 0.06) : n.dd === 1 ? mix(T.top, branch, T.glow ? 0.28 : 0.3) : mix(T.top, branch, T.glow ? 0.1 : 0.09),
        text: n.dd === 0 ? T.on : T.text, edge: branch, badge: mix(T.top, branch, 0.28)
      };
      var side = n.dd === 0 ? mix(T.accent, '#000000', 0.18) : n.sticky ? mix(T.sticky, '#000000', 0.08) : n.code ? mix(T.top, T.text, 0.25) : n.dd === 1 ? mix(branch, T.top, 0.25) : mix(T.top, branch, 0.38);
      var mesh;
      if (self.kind === 'bricks') mesh = self.brick(n, k, T);
      else if (self.kind === 'toy') mesh = self.toyMesh(n, T);
      else {
        var topMat = new THREE.MeshLambertMaterial({ map: self.topTexture(n, k, T) });
        var sideMat = new THREE.MeshLambertMaterial({ color: col(side), emissive: col(side), emissiveIntensity: T.glow * (n.dd <= 1 ? 0.35 : 0.15) });
        mesh = new THREE.Mesh(pill(n), [topMat, sideMat]);
      }
      mesh.castShadow = true; mesh.receiveShadow = true;
      var holder = new THREE.Group(); holder.position.set(n.x, 0, n.z);
      if (n.sticky) holder.rotation.y = 0.07;
      holder.add(mesh); group.add(holder);
      mesh.userData = { node: n, map: id, holder: holder, lift: 0, outline: mesh.userData.outline };
      n.mesh = mesh; n.holder = holder;
      self.picks.push(mesh); self.blocks.push(mesh);
      if (anim) { holder.scale.set(1, 0.001, 1); self.anims.push({ obj: holder, delay: n.i * 55, kind: 'grow' }); }
    });
    all.forEach(function (p) {
      vis(p).forEach(function (c) {
        var s = Math.sign(c.x - p.x) || 1, y = 0.05;
        var a = new THREE.Vector3(p.x + s * p.w / 2, y, p.z), b = new THREE.Vector3(c.x - s * c.w / 2, y, c.z);
        var curve = self.curve(a, b), colr = col(c.sticky ? T.muted : T.b[c.branch]), mesh, tube;
        if (self.kind === 'bricks') {
          mesh = studPath(curve, c.sticky ? T.muted : T.b[c.branch], T);
          if (anim) { mesh.count = 0; self.anims.push({ inst: mesh, delay: c.i * 55 - 80, kind: 'inst' }); }
        } else {
          var K = T.glow ? TOY.night : TOY.day;
          tube = new THREE.TubeGeometry(curve, 96, self.kind === 'toy' ? (c.sticky ? 0.03 : p.dd === 0 ? 0.13 : 0.09) : (c.sticky ? 0.02 : p.dd === 0 ? 0.075 : 0.05), 14, false);
          mesh = new THREE.Mesh(tube, self.kind === 'toy' ? toon(c.sticky ? mix(K.ink, K.paper, 0.5) : mix(K.b[c.branch], '#ffffff', 0.3)) : new THREE.MeshLambertMaterial({ color: colr, emissive: colr, emissiveIntensity: T.glow * 0.8 }));
          if (anim) { tube.setDrawRange(0, 0); self.anims.push({ geo: tube, delay: c.i * 55 - 80, kind: 'draw' }); }
        }
        mesh.castShadow = true; group.add(mesh);
        c.edge = mesh;
        if (!c.sticky && !reduce) for (var j = 0; j < 2; j++) {
          var dot = new THREE.Mesh(new THREE.SphereGeometry(p.dd === 0 ? 0.085 : 0.065, 16, 12), new THREE.MeshLambertMaterial({ color: col('#ffffff'), emissive: colr, emissiveIntensity: 0.9 }));
          dot.position.y = self.dotY(); group.add(dot);
          self.flows.push({ dot: dot, curve: curve, off: j / 2 + Math.random() * 0.1, speed: 0.22 + (p.dd === 0 ? 0.06 : 0), of: c });
        }
      });
    });
    group.userData = { all: all, root: root, id: id };
    return group;
  };

  P.dotY = function () { return this.kind === 'bricks' ? 0.14 : this.kind === 'toy' ? 0.2 : 0.05; };

  /* ---------- floors: linked maps hover above the topic that opens them ---------- */
  P.addFloors = function (main, T, anim) {
    var self = this, level = 0, seen = {};
    main.userData.all.forEach(function (n) {
      if (!n.link || n.link === main.userData.id || seen[n.link]) return;
      seen[n.link] = true;
      var sub = self.buildMap(n.link, T, anim), box = new THREE.Box3().setFromObject(sub), sz = box.getSize(new THREE.Vector3()), c = box.getCenter(new THREE.Vector3());
      var k = 0.5, h = 4.2 + level * 2.9, side = n.side || 1; level++;
      var pw = sz.x * k + 0.9, pd = sz.z * k + 0.9, fx = n.x + side * (pw / 2 - 0.45);
      sub.scale.setScalar(k); sub.position.set(fx - c.x * k, h, n.z - c.z * k);
      sub.traverse(function (o) { if (o.isMesh) o.castShadow = false; });
      var plat, edges = new THREE.Group();
      if (self.kind === 'bricks') { plat = baseplate(pw, pd, T.glow ? mix(T.bg, '#ffffff', 0.12) : mix(T.bg, '#000000', 0.1), k); plat.position.set(fx, h, n.z); }
      else if (self.kind === 'toy') { plat = island(pw / 2 + 0.5, pd / 2 + 0.5, T.glow ? TOY.night : TOY.day, k); plat.position.set(fx, h, n.z); }
      else {
        plat = new THREE.Mesh(new THREE.BoxGeometry(pw, 0.06, pd), new THREE.MeshPhysicalMaterial({ color: col(T.b[0]), transparent: true, opacity: 0.2, roughness: 0.15, clearcoat: 1 }));
        plat.position.set(fx, h - 0.05, n.z); plat.receiveShadow = true;
        edges = new THREE.LineSegments(new THREE.EdgesGeometry(plat.geometry), new THREE.LineBasicMaterial({ color: col(T.b[n.branch]), transparent: true, opacity: 0.7 }));
        edges.position.copy(plat.position);
      }
      var beam = new THREE.Line(new THREE.BufferGeometry().setFromPoints([new THREE.Vector3(n.x, n.h + 0.08, n.z), new THREE.Vector3(n.x, h - 0.08, n.z)]),
        new THREE.LineDashedMaterial({ color: col(T.b[n.branch]), dashSize: 0.14, gapSize: 0.1 }));
      beam.computeLineDistances();
      var label = self.textPlane(self.maps[n.link].title + ' ›', T.b[n.branch], '700 54px ' + T.font, 3.2);
      label.position.set(fx - pw / 2 + 0.05, h + 0.02, n.z - pd / 2 - 0.02);
      [sub, plat, edges, beam, label].forEach(function (o) { self.world.add(o); });
      if (anim) [plat, edges, label].forEach(function (o) { o.visible = false; self.anims.push({ show: o, delay: 600 }); });
    });
  };
  /* flat text on the floor (floor names, the watermark) */
  P.textPlane = function (text, color, font, width, alpha) {
    meas.font = font;
    var cv = document.createElement('canvas'), tw = Math.ceil(meas.measureText(text).width) + 16;
    cv.width = tw; cv.height = 96;
    var c = cv.getContext('2d'); c.font = font; c.fillStyle = color; c.textBaseline = 'middle'; c.globalAlpha = alpha || 1;
    c.fillText(text, 8, 50);
    var tex = new THREE.CanvasTexture(cv); tex.encoding = THREE.sRGBEncoding;
    var w = width || tw / S, m = new THREE.Mesh(new THREE.PlaneGeometry(w, w * 96 / tw), new THREE.MeshBasicMaterial({ map: tex, transparent: true, depthWrite: false }));
    m.rotation.x = -Math.PI / 2; m.geometry.translate(w / 2, w * 48 / tw, 0);
    return m;
  };

  /* ---------- rebuild the scene for the current map ---------- */
  function dispose(o) { o.traverse(function (x) { if (x.geometry) x.geometry.dispose(); if (x.material) [].concat(x.material).forEach(function (m) { if (m.map) m.map.dispose(); m.dispose(); }); }); }
  P.rebuild = function (anim) {
    var T = this.T = this.theme(), w = this.world;
    while (w.children.length) { var ch = w.children[0]; w.remove(ch); dispose(ch); }
    while (this.deco.children.length) { var dc = this.deco.children[0]; this.deco.remove(dc); dispose(dc); }
    this.clouds = [];
    this.picks = []; this.flows = []; this.blocks = []; this.anims = [];
    anim = anim && !reduce;
    this.scene.background = new THREE.Color(T.bg); /* the clear colour is not converted: give it as is */
    this.scene.fog = this.cam === 'persp' ? new THREE.Fog(col(T.bg), 40, 120) : null;
    this.ground.material.opacity = T.glow ? 0.32 : 0.13;
    this.hemi.intensity = T.glow ? 0.7 : 0.95; this.sun.intensity = T.glow ? 0.5 : 0.55;
    this.main = this.buildMap(this.st.map, T, anim);
    w.add(this.main);
    this.ground.visible = this.kind !== 'toy';
    this.scenery(T);
    if (this.floors) this.addFloors(this.main, T, anim);
    var wm = this.mm.opt('wm'), mb = new THREE.Box3().setFromObject(this.main);
    if (wm) {
      var across = this.mm.opt('wm-pos') === 'across';
      var t = this.textPlane((across ? '' : '© ') + wm, T.text, (across ? '700 64px ' : '500 40px ') + T.font, across ? Math.max(4, (mb.max.x - mb.min.x) * 0.6) : 2.6, across ? 0.07 : 0.5);
      if (across) { t.rotation.z = 0.38; t.position.set((mb.min.x + mb.max.x) / 2 - (mb.max.x - mb.min.x) * 0.28, 0.01, (mb.min.z + mb.max.z) / 2 + 0.6); }
      else t.position.set(mb.max.x - 2.6, 0.01, mb.max.z + 0.7);
      w.add(t);
    }
    var box = new THREE.Box3().setFromObject(w), c = box.getCenter(new THREE.Vector3()), s = box.getSize(new THREE.Vector3());
    var R = Math.max(s.x, s.z) * 0.7 + 6, sc = this.sun.shadow.camera;
    this.sun.position.set(c.x - 8, 26, c.z + 12); this.sun.target.position.copy(c);
    sc.left = -R; sc.right = R; sc.top = R; sc.bottom = -R; sc.near = 1; sc.far = 80; sc.updateProjectionMatrix();
    this.animStart = performance.now();
    this.built = this.st.map;
    this.mm.drawCrumbs();
    if (this.revealK !== undefined) this.revealBranches(this.revealK);
  };
  /* ```mindmap steps: branches from the k-th on stay hidden until their build step */
  P.revealBranches = function (k) {
    this.revealK = k;
    if (!this.main) return;
    this.main.userData.all.forEach(function (n) {
      var hide = n.dd > 0 && n.top >= k;
      if (n.holder) n.holder.visible = !hide;
      if (n.edge) n.edge.visible = !hide;
    });
    this.flows.forEach(function (f) { f.hidden = f.of.top >= k; });
  };

  /* ---------- camera ---------- */
  P.tween = function (ms, fn, done) { this.tweens.push({ t0: performance.now(), ms: reduce ? 1 : ms, fn: fn, done: done }); };
  P.dir = function () { return this.cam === 'iso' ? ISO : PERSP; };
  P.fly = function (box, ms, done) {
    var self = this, cam = this.camera, dir = this.dir(), target = box.getCenter(new THREE.Vector3()), corners = [];
    for (var i = 0; i < 8; i++) corners.push(new THREE.Vector3(i & 1 ? box.max.x : box.min.x, i & 2 ? box.max.y : box.min.y, i & 4 ? box.max.z : box.min.z));
    var probeCam = cam.clone(); probeCam.position.copy(target).addScaledVector(dir, 60); probeCam.lookAt(target); probeCam.updateMatrixWorld();
    var inv = probeCam.matrixWorldInverse, x0 = Infinity, x1 = -Infinity, y0 = Infinity, y1 = -Infinity;
    corners.forEach(function (p) { var q = p.clone().applyMatrix4(inv); x0 = Math.min(x0, q.x); x1 = Math.max(x1, q.x); y0 = Math.min(y0, q.y); y1 = Math.max(y1, q.y); });
    var a = (this.box.clientWidth || 1) / (this.box.clientHeight || 1), need = Math.max(y1 - y0, (x1 - x0) / a) * 1.08;
    var fromT = this.controls.target.clone(), fromP = cam.position.clone(), fromZ = cam.zoom, toP, toZ = 1;
    if (cam === this.ortho) { toP = target.clone().addScaledVector(dir, 80); toZ = FH / need; }
    else toP = target.clone().addScaledVector(dir, (need / 2) / Math.tan(THREE.MathUtils.degToRad(this.persp.fov / 2)) + 2);
    this.tween(ms, function (p) {
      var e = ease(p);
      self.controls.target.lerpVectors(fromT, target, e); cam.position.lerpVectors(fromP, toP, e);
      if (cam === self.ortho) { cam.zoom = fromZ + (toZ - fromZ) * e; cam.updateProjectionMatrix(); }
    }, done);
  };
  function boxOf(objs) { var b = new THREE.Box3(); objs.forEach(function (o) { b.expandByObject(o); }); return b; }
  /* a portrait screen (a phone) frames the main map only, a little tighter, floors left above it */
  P.fit = function (ms) {
    var portrait = (this.box.clientWidth || 1) < (this.box.clientHeight || 1) * 0.8;
    this.fly(portrait && this.main ? boxOf([this.main]).expandByScalar(-0.4) : boxOf([this.world]), ms === undefined ? 700 : ms);
  };
  P.zoom = function (f) {
    if (this.camera === this.ortho) { this.camera.zoom = Math.max(0.2, Math.min(8, this.camera.zoom / f)); this.camera.updateProjectionMatrix(); }
    else this.camera.position.sub(this.controls.target).multiplyScalar(f).add(this.controls.target);
  };
  P.setCam = function (kind) {
    this.cam = kind;
    this.camera = kind === 'iso' ? this.ortho : this.persp;
    this.camera.position.copy(this.controls.target).addScaledVector(this.dir(), 60);
    this.makeControls();
    this.scene.fog = kind === 'persp' && this.T ? new THREE.Fog(col(this.T.bg), 40, 120) : null;
    this.fit(0); this.controls.update();
  };

  /* ---------- in and out of the 3D view ---------- */
  P.enter = function () {
    this.on = true; this.box.hidden = false; this.resize();
    if (this.built !== this.st.map || !this.main) this.rebuild(true);
    else this.mm.drawCrumbs();
    this.camera.position.copy(this.controls.target).addScaledVector(this.dir(), 60);
    this.fit(0); this.controls.update();
  };
  P.leave = function () { this.on = false; this.box.hidden = true; this.closeCard(); if (this.tour) this.tourStop(true); };

  /* ---------- picking: hover lifts a block, click opens its card ---------- */
  P.hit = function (e) {
    var r = this.renderer.domElement.getBoundingClientRect(), ray = new THREE.Raycaster();
    ray.setFromCamera(new THREE.Vector2(((e.clientX - r.left) / r.width) * 2 - 1, -((e.clientY - r.top) / r.height) * 2 + 1), this.camera);
    var h = ray.intersectObjects(this.picks, false)[0];
    return h ? h.object : null;
  };
  P.pointer = function () {
    var self = this, cv = this.renderer.domElement, down = null, last = { t: 0, m: null };
    cv.addEventListener('pointermove', function (e) {
      if (down) return;
      var m = self.hit(e);
      if (m !== self.hovered) {
        if (self.hovered) self.hovered.userData.lift = 0;
        self.hovered = m; if (m) m.userData.lift = 0.14;
        cv.style.cursor = m ? 'pointer' : 'grab';
      }
    });
    cv.addEventListener('pointerdown', function (e) { e.stopPropagation(); down = { x: e.clientX, y: e.clientY }; });
    cv.addEventListener('pointerup', function (e) {
      var d = down; down = null;
      if (!d || Math.hypot(e.clientX - d.x, e.clientY - d.y) > 5) return;
      var m = self.hit(e);
      if (!m) { self.closeCard(); return; }
      var now = performance.now(), dbl = last.m === m && now - last.t < 380, u = m.userData;
      last = { t: now, m: m };
      if (dbl) { if (u.node.link) self.drill(u.map, u.node); else if (u.map !== self.st.map) self.drillTo(u.map); return; }
      self.openCard(u.map, u.node);
    });
    cv.addEventListener('click', function (e) { e.stopPropagation(); });
  };
  P.openCard = function (mapId, n) {
    var self = this, card = this.card, kind = n.dd === 0 ? 'Map' : n.sticky ? 'Sticky note' : n.code ? 'Code' : 'Topic';
    card.innerHTML = '<button type="button" class="si-mm-x" aria-label="Close">×</button><small></small><strong></strong><small></small><div></div>';
    card.children[1].textContent = kind + (mapId !== this.st.map ? ' · floor above' : '');
    card.children[2].textContent = n.code ? (n.code.lang || 'code') + ' code' : n.t;
    card.children[3].textContent = n.path.concat(n.dd === 0 || n.code ? [] : [n.t]).join(' › ') || this.maps[mapId].title;
    var row = card.children[4];
    function btn(label, fn, primary) { var b = document.createElement('button'); b.type = 'button'; b.textContent = label; if (primary) b.setAttribute('data-primary', ''); b.onclick = fn; row.appendChild(b); }
    if (n.link) btn('Open map ›', function () { self.drill(mapId, n); }, true);
    if (mapId !== this.st.map && !n.link) btn('Go to this floor', function () { self.drillTo(mapId); }, true);
    if (n.href) btn(n.chip ? n.chip.charAt(0).toUpperCase() + n.chip.slice(1) : 'Open', function () { if (/^[a-z]+:/i.test(n.href)) window.open(n.href, '_blank', 'noopener'); else location.href = n.href; });
    if (n.kids && n.kids.length && n.dd > 0 && mapId === this.st.map) btn(n.collapsed ? 'Unfold' : 'Fold', function () { n.collapsed = !n.collapsed; self.closeCard(); self.rebuild(false); });
    btn('Focus', function () { self.fly(boxOf([n.holder]).expandByScalar(1.6), 700); });
    card.firstChild.onclick = function () { self.closeCard(); };
    card.hidden = false;
  };
  P.closeCard = function () { this.card.hidden = true; };

  /* ---------- drill-down: fly into the topic, the next map rises ---------- */
  P.drill = function (fromMap, n) {
    var self = this;
    if (this.busy) return;
    if (fromMap !== this.st.map) { this.drillTo(fromMap, function () { var m = self.main.userData.all.filter(function (x) { return x.t === n.t; })[0]; if (m) self.drill(self.st.map, m); }); return; }
    this.busy = true; this.closeCard(); if (this.tour) this.tourStop(true);
    this.st.stack.push({ map: this.st.map, via: n.id });
    this.fly(boxOf([n.holder]).expandByScalar(0.4), 650, function () {
      self.st.map = n.link; self.rebuild(true);
      var r = self.main.userData.root;
      self.camera.position.sub(self.controls.target); self.controls.target.set(r.x, 0, r.z); self.camera.position.add(self.controls.target);
      self.fly(boxOf([r.holder]).expandByScalar(2.2), 0);
      setTimeout(function () { self.fit(1100); self.busy = false; }, 60);
    });
  };
  P.drillTo = function (mapId, then) {
    var self = this;
    if (this.busy) return;
    this.busy = true; this.closeCard(); if (this.tour) this.tourStop(true);
    this.st.stack.push({ map: this.st.map, via: '' });
    this.st.map = mapId; this.rebuild(true); this.fit(1000);
    setTimeout(function () { self.busy = false; if (then) then(); }, 1050);
  };
  P.back = function (k) {
    var self = this;
    if (this.busy || k >= this.st.stack.length) return;
    this.busy = true; this.closeCard(); if (this.tour) this.tourStop(true);
    var e = this.st.stack[k]; this.st.stack = this.st.stack.slice(0, k);
    this.st.map = e.map; this.rebuild(true); this.fit(1000);
    setTimeout(function () { self.busy = false; }, 1000);
  };
  /* a search hit: mindmap.js has unfolded the path and set the breadcrumb */
  P.reveal = function (mapId, obj) {
    var self = this, changed = mapId !== this.built;
    this.closeCard(); if (this.tour) this.tourStop(true);
    this.st.map = mapId; this.rebuild(changed);
    this.fit(changed ? 900 : 0);
    if (!obj) return;
    setTimeout(function () {
      if (!obj.mesh || !obj.holder) return;
      self.fly(boxOf([obj.holder]).expandByScalar(2.6), 800);
      var u = obj.mesh.userData, side = [].concat(obj.mesh.material)[1];
      u.lift = 0.4;
      if (side && !reduce) { var i0 = side.emissiveIntensity; self.tween(1600, function (p) { side.emissiveIntensity = i0 + Math.sin(p * Math.PI * 3) * 0.6; }, function () { side.emissiveIntensity = i0; }); }
      setTimeout(function () { u.lift = 0; }, 1600);
    }, changed && !reduce ? 950 : 50);
  };

  /* ---------- Play: one branch per step, the rest dimmed ---------- */
  P.steps = function () { return [null].concat(vis(this.main.userData.root).filter(function (n) { return !n.sticky; })); };
  P.setDim = function (top) {
    this.main.userData.all.forEach(function (n) {
      var dim = top !== null && n.dd > 0 && n.top !== top;
      var o = n.mesh.userData.outline;
      [].concat(n.mesh.material).concat(n.edge ? [n.edge.material] : []).concat(o ? [o.material] : []).forEach(function (m) { m.transparent = true; m.opacity = dim ? 0.16 : 1; m.depthWrite = !dim; });
    });
    this.flows.forEach(function (f) { f.dim = top !== null && f.of.dd > 0 && f.of.top !== top; });
  };
  P.tourGo = function (k) {
    var steps = this.steps(); k = Math.max(0, Math.min(steps.length - 1, k)); this.tour.k = k;
    var t = steps[k];
    this.setDim(t ? t.top : null);
    if (t) this.fly(boxOf(this.main.userData.all.filter(function (n) { return n.dd > 0 && n.top === t.top; }).map(function (n) { return n.holder; })).expandByScalar(0.6), 1100);
    else this.fit(1100);
    if (this.mm.tourEl) this.mm.tourEl.textContent = (k + 1) + ' / ' + steps.length + ' · ' + (t ? t.t : this.maps[this.st.map].title);
  };
  P.tourStart = function () {
    var self = this;
    this.closeCard(); this.tour = { k: 0 };
    if (this.mm.tourEl) this.mm.tourEl.hidden = false;
    if (this.mm.playBtn) { this.mm.playBtn.setAttribute('aria-pressed', 'true'); this.mm.playBtn.textContent = 'Stop'; }
    this.tourGo(0);
    this.tour.timer = setInterval(function () { if (self.tour.k >= self.steps().length - 1) { self.tourStop(); return; } self.tourGo(self.tour.k + 1); }, 3400);
  };
  P.tourStop = function (quiet) {
    if (!this.tour) return; clearInterval(this.tour.timer); this.tour = null;
    if (this.mm.tourEl) this.mm.tourEl.hidden = true;
    if (this.mm.playBtn) { this.mm.playBtn.setAttribute('aria-pressed', 'false'); this.mm.playBtn.textContent = 'Play'; }
    this.setDim(null);
    if (!quiet) this.fit(900);
  };

  /* ---------- the frame ---------- */
  P.frame = function (now) {
    if (!this.on || !this.seen || !this.box.clientWidth) return;
    this.tweens = this.tweens.filter(function (t) { var p = Math.min(1, (now - t.t0) / t.ms); t.fn(p); if (p >= 1) { if (t.done) t.done(); return false; } return true; });
    var el = now - this.animStart;
    this.anims.forEach(function (a) {
      var p = Math.max(0, Math.min(1, (el - a.delay) / 520));
      if (a.kind === 'grow') a.obj.scale.y = Math.max(0.001, p === 1 ? 1 : 1 + 2.2 * Math.pow(p - 1, 3) + 1.2 * Math.pow(p - 1, 2));
      else if (a.kind === 'draw') a.geo.setDrawRange(0, Math.floor(a.geo.index.count * p / 6) * 6);
      else if (a.kind === 'inst') a.inst.count = Math.floor(a.inst.userData.total * p);
      else if (a.show && el > a.delay) a.show.visible = true;
    });
    this.blocks.forEach(function (m) { var h = m.userData.holder; h.position.y += ((m.userData.lift || 0) - h.position.y) * 0.18; });
    var t = now / 1000, dy = this.dotY(), flowOn = this.mm.opt('flow') !== 'off';
    this.flows.forEach(function (f) {
      f.dot.visible = !f.dim && !f.hidden && flowOn;
      var pt = f.curve.getPointAt((t * f.speed + f.off) % 1);
      f.dot.position.set(pt.x, dy, pt.z);
    });
    if (!reduce) this.clouds.forEach(function (c) { c.position.y = c.userData.y + Math.sin(now / 1400 + c.userData.bob) * 0.18; });
    this.controls.update();
    this.renderer.render(this.scene, this.camera);
  };

  window.SIMindmap3D = Mindmap3D;
})();
