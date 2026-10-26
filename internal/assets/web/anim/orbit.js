// Built-in timer animation: a constellation of nodes orbiting a centre, with
// links drawn between those that drift close. Contract: a function expression
// (canvas, api) -> stop(). api gives theme tokens, fraction() (0..1 elapsed) /
// remaining() seconds, and reduced (motion). It turns faster near zero.
function (canvas, api) {
  var ctx = canvas.getContext('2d');
  var raf, t = 0, N = 7, nodes = [];
  for (var i = 0; i < N; i++) {
    nodes.push({ r: 0.18 + (i % 3) * 0.11, sp: 0.3 + (i * 13 % 50) / 100, ph: i * 2 * Math.PI / N, sz: 0.6 + (i % 2) });
  }
  function frame() {
    var w = canvas.width, h = canvas.height, R = Math.min(w, h);
    ctx.clearRect(0, 0, w, h);
    ctx.save();
    ctx.translate(w / 2, h / 2);
    t += api.reduced ? 0 : 0.003 + api.fraction() * 0.02; // faster near zero
    var pts = nodes.map(function (nd) {
      var a = nd.ph + t * nd.sp;
      return [Math.cos(a) * nd.r * R, Math.sin(a) * nd.r * R * 0.85, nd.sz];
    });
    ctx.lineWidth = Math.max(1, R * 0.002);
    ctx.strokeStyle = api.tokens.line || api.tokens.muted;
    ctx.globalAlpha = 0.22;
    [0.18, 0.29, 0.4].forEach(function (rr) {
      ctx.beginPath(); ctx.ellipse(0, 0, rr * R, rr * R * 0.85, 0, 0, 2 * Math.PI); ctx.stroke();
    });
    ctx.strokeStyle = api.tokens.accent;
    for (var a = 0; a < pts.length; a++) {
      for (var b = a + 1; b < pts.length; b++) {
        var dx = pts[a][0] - pts[b][0], dy = pts[a][1] - pts[b][1], d = Math.sqrt(dx * dx + dy * dy), lim = R * 0.28;
        if (d < lim) {
          ctx.globalAlpha = 0.35 * (1 - d / lim);
          ctx.beginPath(); ctx.moveTo(pts[a][0], pts[a][1]); ctx.lineTo(pts[b][0], pts[b][1]); ctx.stroke();
        }
      }
    }
    ctx.globalAlpha = 1;
    ctx.fillStyle = api.tokens.accent2 || api.tokens.accent;
    pts.forEach(function (p) { ctx.beginPath(); ctx.arc(p[0], p[1], R * 0.012 * p[2] + R * 0.006, 0, 2 * Math.PI); ctx.fill(); });
    ctx.fillStyle = api.tokens.accent;
    ctx.beginPath(); ctx.arc(0, 0, R * 0.02, 0, 2 * Math.PI); ctx.fill();
    ctx.restore();
    ctx.globalAlpha = 1;
    raf = requestAnimationFrame(frame);
  }
  raf = requestAnimationFrame(frame);
  return function () { cancelAnimationFrame(raf); };
}
