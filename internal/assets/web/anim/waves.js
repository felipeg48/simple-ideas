// Built-in timer animation: layered sine waves drifting across the screen.
// Contract: a function expression (canvas, api) -> stop(). api gives theme
// tokens, fraction() (0..1 elapsed) / remaining() seconds, and reduced (motion).
// The waves drift faster as the break runs down.
function (canvas, api) {
  var ctx = canvas.getContext('2d');
  var raf, t = 0;
  function frame() {
    var w = canvas.width, h = canvas.height;
    ctx.clearRect(0, 0, w, h);
    t += api.reduced ? 0 : 0.01 + api.fraction() * 0.04; // faster near zero
    var cols = [api.tokens.accent, api.tokens.accent2 || api.tokens.accent, api.tokens.muted || api.tokens.line];
    var step = Math.max(4, w / 240);
    for (var L = 0; L < 3; L++) {
      var amp = h * 0.06 * (1 + L * 0.5), base = h * (0.52 + L * 0.12);
      var k = (2 + L) * Math.PI / w, sp = t * (1 + L * 0.4);
      ctx.beginPath();
      ctx.moveTo(0, base);
      for (var x = 0; x <= w; x += step) ctx.lineTo(x, base + Math.sin(x * k + sp) * amp);
      ctx.lineTo(w, h); ctx.lineTo(0, h); ctx.closePath();
      ctx.globalAlpha = 0.18 - L * 0.04;
      ctx.fillStyle = cols[L];
      ctx.fill();
    }
    ctx.globalAlpha = 1;
    raf = requestAnimationFrame(frame);
  }
  raf = requestAnimationFrame(frame);
  return function () { cancelAnimationFrame(raf); };
}
