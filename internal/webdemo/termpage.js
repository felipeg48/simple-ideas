(function (cfg) {
  // A web demo's terminal scene: xterm.js inside a code window, fed by the
  // shell si runs under a pseudo-terminal (loopback, keyed URLs). si types
  // into the shell itself; this page only shows it. It tells si its size,
  // sets window.__siTermReady, and window.__siTermLine(text) points the
  // demo overlay's notes (window.__siT) at the line that shows the text.
  // window.__siTermSplit(dir, cfg) adds a second shell beside (right) or
  // under (down) the first, both shown and filmed; __siTermFocus(n) marks
  // the pane si types into.
  var fig = document.querySelector('.si-cw'), body = fig.querySelector('.si-cw-body'), host = document.getElementById('si-term');
  var cs = getComputedStyle(fig);
  function v(n, d) { return cs.getPropertyValue(n).trim() || d; }
  var fg = v('--si-cw-fg', '#d8dee9'), bg = v('--si-cw-bg', '#0f1420');
  var panes = [];
  // one shell's xterm in el; ready is called once its output streams
  function pane(el, c, ready) {
    var term = new Terminal({
      fontFamily: v('--si-font-mono', 'ui-monospace,SFMono-Regular,Menlo,Consolas,monospace'),
      fontSize: 16, lineHeight: 1.15, cursorBlink: false, scrollback: 5000, disableStdin: true,
      theme: { background: bg, foreground: fg, cursor: v('--si-cw-k', fg), cursorAccent: bg, selectionBackground: v('--si-cw-hl', '#444') }
    });
    var fit = new FitAddon.FitAddon();
    term.loadAddon(fit);
    term.open(el);
    var p = { el: el, term: term, fit: fit, c: c };
    panes.push(p);
    var box = el.getBoundingClientRect();
    if (c.cols) {
      // size=CxR: the largest font that fits that many columns and rows
      var fs = 16, cell;
      term.options.fontSize = fs;
      cell = { w: box.width / Math.max(1, fit.proposeDimensions().cols), h: box.height / Math.max(1, fit.proposeDimensions().rows) };
      fs = Math.max(8, Math.floor(fs * Math.min(box.width / (cell.w * c.cols), box.height / (cell.h * c.rows))));
      term.options.fontSize = fs;
      term.resize(c.cols, c.rows);
    } else {
      term.options.fontSize = c.font || Math.round(Math.max(13, Math.min(innerWidth / 70, innerHeight / 30)));
      fit.fit();
    }
    fetch(c.size + '?cols=' + term.cols + '&rows=' + term.rows, { method: 'POST' }).finally(function () {
      fetch(c.out).then(function (res) {
        var rd = res.body.getReader();
        ready();
        (function pump() {
          rd.read().then(function (r) { if (r.done) return; term.write(r.value); pump(); });
        })();
      });
    });
    return p;
  }
  pane(host, cfg, function () { window.__siTermReady = true; });
  window.__siTermSplit = function (dir, c) {
    if (panes.length > 1) return;
    body.classList.add('is-split');
    body.setAttribute('data-split', dir);
    host.classList.add('si-term-pane');
    var el = document.createElement('div');
    el.className = 'si-term-pane';
    body.appendChild(el);
    // the first pane is half as large now: the same font, fewer columns or rows
    var p1 = panes[0];
    p1.fit.fit();
    fetch(p1.c.size + '?cols=' + p1.term.cols + '&rows=' + p1.term.rows, { method: 'POST' });
    pane(el, { out: c.out, size: c.size, font: p1.term.options.fontSize }, function () { window.__siTermSplitReady = true; });
    window.__siTermFocus(2);
  };
  // each pane's columns and rows (what si told its shell)
  window.__siTermSize = function () { return panes.map(function (p) { return p.term.cols + 'x' + p.term.rows; }).join(' '); };
  window.__siTermFocus = function (n) {
    panes.forEach(function (p, i) { p.el.classList.toggle('is-focus', i === n - 1); });
  };
  // the line showing text, as a box the overlay can follow while it scrolls
  function find(text) {
    var rows = body.querySelectorAll('.xterm-rows > div');
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
  // the middle of the window's body (or of pane n), where the cursor rests
  window.__siTermSpot = function (n) {
    var b = (n && panes[n - 1] ? panes[n - 1].el : host).getBoundingClientRect();
    return [Math.round(b.left + b.width * 0.62), Math.round(b.top + b.height * 0.7)];
  };
})
