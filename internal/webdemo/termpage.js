(function (cfg) {
  // A web demo's terminal scene: xterm.js inside a code window, fed by the
  // shell si runs under a pseudo-terminal (loopback, keyed URLs). si types
  // into the shell itself; this page only shows it. It tells si its size,
  // sets window.__siTermReady, and window.__siTermLine(text) points the
  // demo overlay's notes (window.__siT) at the line that shows the text.
  var fig = document.querySelector('.si-cw'), host = document.getElementById('si-term');
  var cs = getComputedStyle(fig);
  function v(n, d) { return cs.getPropertyValue(n).trim() || d; }
  var fg = v('--si-cw-fg', '#d8dee9'), bg = v('--si-cw-bg', '#0f1420');
  var term = new Terminal({
    fontFamily: v('--si-font-mono', 'ui-monospace,SFMono-Regular,Menlo,Consolas,monospace'),
    fontSize: 16, lineHeight: 1.15, cursorBlink: false, scrollback: 5000, disableStdin: true,
    theme: { background: bg, foreground: fg, cursor: v('--si-cw-k', fg), cursorAccent: bg, selectionBackground: v('--si-cw-hl', '#444') }
  });
  var fit = new FitAddon.FitAddon();
  term.loadAddon(fit);
  term.open(host);
  var box = host.getBoundingClientRect();
  if (cfg.cols) {
    // size=CxR: the largest font that fits that many columns and rows
    var fs = 16, cell;
    term.options.fontSize = fs;
    cell = { w: box.width / Math.max(1, fit.proposeDimensions().cols), h: box.height / Math.max(1, fit.proposeDimensions().rows) };
    fs = Math.max(8, Math.floor(fs * Math.min(box.width / (cell.w * cfg.cols), box.height / (cell.h * cfg.rows))));
    term.options.fontSize = fs;
    term.resize(cfg.cols, cfg.rows);
  } else {
    term.options.fontSize = Math.round(Math.max(13, Math.min(innerWidth / 70, innerHeight / 30)));
    fit.fit();
  }
  fetch(cfg.size + '?cols=' + term.cols + '&rows=' + term.rows, { method: 'POST' }).finally(function () {
    fetch(cfg.out).then(function (res) {
      var rd = res.body.getReader();
      window.__siTermReady = true;
      (function pump() {
        rd.read().then(function (r) { if (r.done) return; term.write(r.value); pump(); });
      })();
    });
  });
  // the line showing text, as a box the overlay can follow while it scrolls
  function find(text) {
    var rows = host.querySelectorAll('.xterm-rows > div');
    for (var i = rows.length - 1; i >= 0; i--) {
      var t = rows[i].textContent.replace(/\u00a0/g, ' '), k = t.indexOf(text);
      if (k < 0) continue;
      var rg = document.createRange(), walk = document.createTreeWalker(rows[i], NodeFilter.SHOW_TEXT), n, pos = 0, set = 0;
      while ((n = walk.nextNode())) {
        var len = n.data.length;
        if (!set && k < pos + len) { rg.setStart(n, k - pos); set = 1; }
        if (set && k + text.length <= pos + len) { rg.setEnd(n, k + text.length - pos); return rg.getBoundingClientRect(); }
        pos += len;
      }
      return rows[i].getBoundingClientRect();
    }
    return null;
  }
  window.__siTermLine = function (text) {
    var r = find(text);
    if (!r) return false;
    window.__siT = {
      get isConnected() { return !!find(text); },
      getBoundingClientRect: function () { return find(text) || r; }
    };
    return true;
  };
  // the middle of the window's body, where the cursor rests
  window.__siTermSpot = function () {
    var b = host.getBoundingClientRect();
    return [Math.round(b.left + b.width * 0.62), Math.round(b.top + b.height * 0.7)];
  };
})
