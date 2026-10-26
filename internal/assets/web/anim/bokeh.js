// Built-in timer animation: soft bokeh dots drifting upward and fading — a calm
// ambient field. Contract: a function expression (canvas, api) -> stop(). api
// gives theme tokens, fraction() (0..1 elapsed) / remaining() seconds, and
// reduced (motion). The drift quickens a little as the break runs down.
function (canvas, api) {
  var ctx = canvas.getContext('2d');
  var raf, dots = [];
  function rnd(a, b) { return a + Math.random() * (b - a); }
  for (var i = 0; i < 26; i++) {
    dots.push({ x: Math.random(), y: Math.random(), r: rnd(0.012, 0.055), v: rnd(0.2, 0.9), a: rnd(0.1, 0.4) });
  }
  function frame() {
    var w = canvas.width, h = canvas.height, R = Math.min(w, h);
    ctx.clearRect(0, 0, w, h);
    var cols = [api.tokens.accent, api.tokens.accent2 || api.tokens.accent];
    var spd = api.reduced ? 0 : 0.0016 + api.fraction() * 0.004;
    for (var i = 0; i < dots.length; i++) {
      var d = dots[i];
      d.y -= d.v * spd;
      if (d.y < -0.1) { d.y = 1.1; d.x = Math.random(); }
      ctx.globalAlpha = d.a;
      ctx.fillStyle = cols[i % 2];
      ctx.beginPath(); ctx.arc(d.x * w, d.y * h, d.r * R, 0, 2 * Math.PI); ctx.fill();
    }
    ctx.globalAlpha = 1;
    raf = requestAnimationFrame(frame);
  }
  raf = requestAnimationFrame(frame);
  return function () { cancelAnimationFrame(raf); };
}
