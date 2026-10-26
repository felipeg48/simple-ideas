// Built-in timer animation: ranges (dots) distributing and rebalancing across
// nodes (columns). Contract: (canvas, api) -> stop(). A range periodically moves
// from the fullest node to the emptiest — a gentle, continuous rebalance.
function (canvas, api) {
  var ctx = canvas.getContext('2d');
  var raf, tick = 0;
  var N = 5, ranges = [];
  for (var i = 0; i < 24; i++) ranges.push({ node: i % N, t: 1 });
  function counts() { var c = []; for (var i = 0; i < N; i++) c.push(0); ranges.forEach(function (r) { c[r.node]++; }); return c; }
  function frame() {
    var w = canvas.width, h = canvas.height;
    ctx.clearRect(0, 0, w, h);
    var pad = w * 0.08, gap = w * 0.03, nw = (w - pad * 2 - gap * (N - 1)) / N;
    var baseY = h * 0.82, colH = h * 0.6, dot = Math.max(5, nw * 0.09);
    tick++;
    if (!api.reduced && tick % Math.max(20, 70 - Math.floor(api.fraction() * 50)) === 0) {
      var c = counts(), hi = 0, lo = 0;
      for (var i = 1; i < N; i++) { if (c[i] > c[hi]) hi = i; if (c[i] < c[lo]) lo = i; }
      for (var j = 0; j < ranges.length; j++) { if (ranges[j].node === hi) { ranges[j].node = lo; ranges[j].t = 0; break; } }
    }
    var nx = function (n) { return pad + n * (nw + gap); };
    ctx.fillStyle = api.tokens.surface || api.tokens.bg; ctx.strokeStyle = api.tokens.line; ctx.lineWidth = 2;
    for (var n = 0; n < N; n++) { ctx.beginPath(); ctx.roundRect(nx(n), baseY - colH, nw, colH, 10); ctx.fill(); ctx.stroke(); }
    var c2 = [0, 0, 0, 0, 0];
    ranges.forEach(function (r) {
      if (r.t < 1) r.t = Math.min(1, r.t + 0.05);
      var slot = c2[r.node]++; var col = slot % 3, row = Math.floor(slot / 3);
      var x = nx(r.node) + nw * (0.25 + col * 0.25), y = baseY - dot * 1.6 - row * dot * 2.4;
      ctx.globalAlpha = 0.4 + 0.6 * r.t;
      ctx.fillStyle = (r.t < 1 ? (api.tokens.accent2 || api.tokens.accent) : api.tokens.accent);
      ctx.beginPath(); ctx.arc(x, y, dot, 0, 2 * Math.PI); ctx.fill();
    });
    ctx.globalAlpha = 1;
    raf = requestAnimationFrame(frame);
  }
  raf = requestAnimationFrame(frame);
  return function () { cancelAnimationFrame(raf); };
}
