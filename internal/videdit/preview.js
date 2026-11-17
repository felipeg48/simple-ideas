/* si play: the video edit preview. Source time drives everything (the
   build is exact; this plays the source with the edits approximated). */
(function () {
  var P = window.VP, v = document.querySelector('video'), cam = document.querySelector('.vp-cam');
  var strip = document.querySelector('.vp-strip'), head = document.querySelector('.vp-head');
  var btn = document.querySelector('.vp-play'), time = document.querySelector('.vp-time');
  var say = document.querySelector('.vp-say'), list = document.querySelector('.vp-lines');
  if (!v) return;
  function stamp(t) { t = Math.max(0, t); var m = Math.floor(t / 60), s = t - m * 60; return m + ':' + (s < 10 ? '0' : '') + s.toFixed(1); }
  var full = [P.w / 2, P.h / 2, P.w], moves = [], cur = full, free = 0, lanes = [];
  P.marks.forEach(function (m, i) {
    var el = document.createElement('div');
    el.className = 'm ' + m.kind + (m.approx ? ' approx' : '');
    var to = m.range ? m.to : m.from + Math.max(m.dur || 0, 0);
    var a = Math.min(m.from, P.d), z = Math.min(to, P.d);
    el.style.left = (a / P.d * 100) + '%';
    el.style.width = (Math.max(0, z - a) / P.d * 100) + '%';
    if (z - a < P.d / 200 && a > P.d * 0.98) el.style.transform = 'translateX(-100%)'; // at the end: inside the strip
    var lane = 0, end = Math.max(z, a + P.d / 60); // lines at the same time: one lane each, none hidden
    while (lanes[lane] > a) lane++;
    lanes[lane] = end; m.lane = lane; m.el = el;
    el.title = 'line ' + m.line + ': ' + m.label; el.dataset.line = m.line;
    strip.appendChild(el);
    var li = document.createElement('li'), b = document.createElement('button');
    b.type = 'button';
    b.innerHTML = '<b></b>';
    b.firstChild.textContent = stamp(m.from) + (m.approx ? ' ∗' : '');
    b.appendChild(document.createTextNode(m.label));
    b.onclick = function () { v.currentTime = m.from; draw(); };
    li.dataset.line = m.line; li.appendChild(b); list.appendChild(li); m.li = li;
    if (m.kind === 'zoom' || m.kind === 'pan' || m.kind === 'zoom-out') {
      var t0 = Math.max(m.from, free), target = m.kind === 'zoom-out' ? full : m.Rect;
      moves.push({ t0: t0, t1: t0 + m.move, a: cur, b: target, lin: m.linear });
      cur = target; free = t0 + m.move;
      if (m.hold && m.kind !== 'zoom-out') { var bk = free + m.dur; moves.push({ t0: bk, t1: bk + m.move, a: target, b: full, lin: m.linear }); cur = full; free = bk + m.move; }
    }
  });
  if (lanes.length > 1) {
    strip.style.height = (0.7 + lanes.length * 0.9) + 'rem';
    P.marks.forEach(function (m) { m.el.style.top = (0.35 + m.lane * 0.9) + 'rem'; m.el.style.height = '0.75rem'; });
  }
  function view(t) {
    var r = full;
    for (var i = 0; i < moves.length; i++) {
      var m = moves[i];
      if (t < m.t0) return i === 0 ? full : moves[i - 1].b;
      if (t < m.t1) { var p = (t - m.t0) / (m.t1 - m.t0); if (!m.lin) p = p * p * (3 - 2 * p); return [0, 1, 2].map(function (k) { return m.a[k] + (m.b[k] - m.a[k]) * p; }); }
      r = m.b;
    }
    return r;
  }
  var holdUntil = 0, last = 0;
  function draw() {
    var t = v.currentTime;
    if (window.VE) VE.at(t);
    var c = view(t), z = P.w / c[2];
    cam.style.transform = 'scale(' + z + ') translate(' + (-(c[0] - c[2] / 2) / P.w * 100) + '%,' + (-(c[1] - c[2] * P.h / P.w / 2) / P.h * 100) + '%)';
    head.style.left = (t / P.d * 100) + '%';
    time.textContent = stamp(t) + ' / ' + stamp(P.d);
    var says = [];
    P.marks.forEach(function (m) {
      var on = t >= m.from && t < (m.range ? m.to : m.from + Math.max(m.dur || 0, 0.5));
      m.li.classList.toggle('on', on);
      if (on && m.kind === 'say') says.push(m.label);
    });
    say.textContent = says.join(' ');
  }
  function tick() {
    var t = v.currentTime, now = performance.now();
    if (!v.paused) {
      var rate = 1;
      P.marks.forEach(function (m) {
        if ((m.kind === 'cut' || m.kind === 'trim-end') && t >= m.from && t < (m.kind === 'trim-end' ? P.d : m.to)) { if (m.kind === 'trim-end') v.pause(); else v.currentTime = m.to; }
        if (m.kind === 'speed' && t >= m.from && t < m.to) rate = m.speed;
        if (m.kind === 'freeze' && last < m.from && t >= m.from && t - last < 1) { v.pause(); holdUntil = now + m.dur * 1000; v.currentTime = m.from; }
      });
      if (v.playbackRate !== rate) v.playbackRate = rate;
    } else if (holdUntil && now >= holdUntil) { holdUntil = 0; v.play(); }
    last = t;
    draw();
    requestAnimationFrame(tick);
  }
  function seek(ev) {
    var r = strip.getBoundingClientRect();
    v.currentTime = Math.min(P.d, Math.max(0, (ev.clientX - r.left) / r.width * P.d));
    draw();
  }
  strip.addEventListener('pointerdown', function (ev) { strip.setPointerCapture(ev.pointerId); seek(ev); });
  strip.addEventListener('pointermove', function (ev) { if (ev.buttons) seek(ev); });
  strip.addEventListener('keydown', function (ev) {
    if (ev.key === 'ArrowRight') v.currentTime = Math.min(P.d, v.currentTime + 1);
    else if (ev.key === 'ArrowLeft') v.currentTime = Math.max(0, v.currentTime - 1);
    else return;
    ev.preventDefault(); draw();
  });
  function toggle() {
    holdUntil = 0;
    if (v.paused) {
      var start = P.marks.filter(function (m) { return m.kind === 'trim-start'; })[0];
      if (start && v.currentTime < start.from) v.currentTime = start.from;
      v.play();
    } else v.pause();
  }
  btn.onclick = toggle;
  v.addEventListener('play', function () { btn.textContent = '❚❚'; btn.setAttribute('aria-label', 'Pause'); });
  v.addEventListener('pause', function () { btn.textContent = '▶'; btn.setAttribute('aria-label', 'Play'); });
  document.addEventListener('keydown', function (ev) { if (ev.key === ' ' && ev.target === document.body) { ev.preventDefault(); toggle(); } });
  v.addEventListener('loadedmetadata', draw);
  requestAnimationFrame(tick);
})();
