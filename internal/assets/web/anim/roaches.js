// Built-in timer animation: cockroaches wandering the screen; they speed up and
// scatter as the break runs down. Contract: (canvas, api) -> stop().
function (canvas, api) {
  var ctx = canvas.getContext('2d');
  var raf, bugs = [];
  function rnd(a, b) { return a + Math.random() * (b - a); }
  for (var i = 0; i < 9; i++) {
    bugs.push({ x: rnd(0.1, 0.9), y: rnd(0.1, 0.9), a: rnd(0, 6.28), v: rnd(0.4, 0.9), turn: 0 });
  }
  function roach(b, s) {
    ctx.save();
    ctx.translate(b.x * canvas.width, b.y * canvas.height);
    ctx.rotate(b.a + Math.PI / 2);
    ctx.strokeStyle = api.tokens.fg; ctx.lineWidth = Math.max(1, s * 0.07); ctx.lineCap = 'round';
    for (var side = -1; side <= 1; side += 2) {
      for (var L = 0; L < 3; L++) {
        var ly = -s * 0.3 + L * s * 0.35, sw = Math.sin(Date.now() / 90 + L + (side > 0 ? 1.5 : 0)) * s * 0.18;
        ctx.beginPath(); ctx.moveTo(side * s * 0.25, ly); ctx.lineTo(side * s * 0.6, ly + sw); ctx.stroke();
      }
    }
    ctx.fillStyle = api.tokens.fg;
    ctx.beginPath(); ctx.ellipse(0, 0, s * 0.3, s * 0.52, 0, 0, 2 * Math.PI); ctx.fill();
    ctx.fillStyle = api.tokens.accent;
    ctx.beginPath(); ctx.ellipse(0, -s * 0.38, s * 0.22, s * 0.26, 0, 0, 2 * Math.PI); ctx.fill();
    ctx.restore();
  }
  function frame() {
    ctx.clearRect(0, 0, canvas.width, canvas.height);
    var s = Math.min(canvas.width, canvas.height) * 0.05;
    var speed = (api.reduced ? 0 : 1) * (1 + api.fraction() * 2.5) * 0.004; // scatter near zero
    bugs.forEach(function (b) {
      if (Math.random() < 0.02) b.turn = rnd(-0.08, 0.08);
      b.a += b.turn;
      b.x += Math.cos(b.a) * b.v * speed; b.y += Math.sin(b.a) * b.v * speed;
      if (b.x < 0.04 || b.x > 0.96) { b.a = Math.PI - b.a; b.x = Math.max(0.04, Math.min(0.96, b.x)); }
      if (b.y < 0.04 || b.y > 0.96) { b.a = -b.a; b.y = Math.max(0.04, Math.min(0.96, b.y)); }
      roach(b, s);
    });
    raf = requestAnimationFrame(frame);
  }
  raf = requestAnimationFrame(frame);
  return function () { cancelAnimationFrame(raf); };
}
