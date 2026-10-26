// Built-in timer animation: a rotating wireframe globe with orbiting nodes.
// Contract: a function expression (canvas, api) -> stop(). api gives theme
// tokens, fraction() (0..1 elapsed) / remaining() seconds, and reduced (motion).
// The globe spins faster as the break runs down.
function (canvas, api) {
  var ctx = canvas.getContext('2d');
  var raf, t = 0;
  function frame() {
    var w = canvas.width, h = canvas.height, R = Math.min(w, h) * 0.3;
    ctx.clearRect(0, 0, w, h);
    ctx.save();
    ctx.translate(w / 2, h / 2);
    t += api.reduced ? 0 : 0.004 + api.fraction() * 0.03; // faster near zero
    ctx.lineWidth = Math.max(1, R * 0.012);
    ctx.strokeStyle = api.tokens.accent;
    ctx.globalAlpha = 0.55;
    ctx.beginPath(); ctx.arc(0, 0, R, 0, 2 * Math.PI); ctx.stroke();
    for (var i = 0; i < 6; i++) {
      var rx = Math.abs(Math.cos(t + i * Math.PI / 6)) * R;
      ctx.beginPath(); ctx.ellipse(0, 0, rx, R, 0, 0, 2 * Math.PI); ctx.stroke();
    }
    for (var j = 1; j < 4; j++) {
      var yy = R * j / 4, rr = Math.sqrt(R * R - yy * yy);
      ctx.beginPath(); ctx.ellipse(0, yy, rr, rr * 0.33, 0, 0, 2 * Math.PI); ctx.stroke();
      ctx.beginPath(); ctx.ellipse(0, -yy, rr, rr * 0.33, 0, 0, 2 * Math.PI); ctx.stroke();
    }
    ctx.globalAlpha = 1;
    ctx.fillStyle = api.tokens.accent2 || api.tokens.accent;
    for (var k = 0; k < 5; k++) {
      var a = t * 1.5 + k * 2 * Math.PI / 5;
      ctx.beginPath(); ctx.arc(Math.cos(a) * R, Math.sin(a) * R * 0.5, R * 0.045, 0, 2 * Math.PI); ctx.fill();
    }
    ctx.restore();
    raf = requestAnimationFrame(frame);
  }
  raf = requestAnimationFrame(frame);
  return function () { cancelAnimationFrame(raf); };
}
