  /* window.siPlay: the public API of si play's presenter tools (si play on a
   * loopback address only, never in a built deck). Web demos (embed-terminal,
   * embed-browser, slide N), a deck's [TERM …] / [BROWSER …] cues and
   * auto-play drive the drawer, the browser windows and the deck through it,
   * never through their markup. It is a contract: docs/terminal.md lists it,
   * internal/webdemo/embed_e2e_test.go pins it. Calls that open or type return
   * a Promise; output, frame and at answer at once. */
  if (termEnabled && !role) {
    var tApi = function () { return new Promise(function (ok) { if (window.siTerm) ok(window.siTerm.api); else loadTerm(function () { ok(window.siTerm.api); }); }); };
    var wApi = function () { return new Promise(function (ok) { loadWeb(function () { ok(window.siWeb.api); }); }); };
    var siPlay = window.siPlay = {
      version: 1,
      terminal: {
        /* the drawer open on the shell tab name (created if needed); o.split 'right'|'down': beside the tab in front; o.opacity: as opacity() */
        open: function (name, o) { return tApi().then(function (a) { if (o && o.opacity) a.opacity(o.opacity); return a.open(name, o && o.split); }); },
        /* a command run in tab name (pasted; o.typing: ms per character, typed) */
        run: function (name, cmd, o) { return tApi().then(function (a) { return a.run(name, cmd, o && o.typing); }); },
        /* keys sent as they are ('\r' Enter, '\x03' ctrl+c) */
        type: function (name, text, o) { return tApi().then(function (a) { return a.input(name, text, o && o.typing); }); },
        /* the text on the tab's screen and scrollback ('' for no such tab) */
        output: function (name) { return window.siTerm ? window.siTerm.api.text(name) : ''; },
        names: function () { return window.siTerm ? window.siTerm.api.names() : []; },
        /* the drawer: 'min' (hidden, shells run on), 'max' (the whole window), 'restore', 'close' */
        state: function (s) { return tApi().then(function (a) { return a.state(s); }); },
        /* the drawer's opacity in % (40-100, remembered; the slide shows through, the text stays opaque); no n: the current one */
        opacity: function (n) { return tApi().then(function (a) { return a.opacity(n); }); }
      },
      browser: {
        /* a window over the slides: a browser: link's name or a URL; o {x, y, w, h} CSS pixels, zoom (1.25 = 125 %) */
        open: function (name, o) { return wApi().then(function (a) { return a.open(name, o); }); },
        /* the open window name to url (relative to its page: '#/queues'); resolves when it loads */
        go: function (name, url) { return wApi().then(function (a) { return a.go(name, url); }); },
        close: function (name) { return wApi().then(function (a) { a.close(name); }); }, /* no name: every window */
        min: function (name) { return wApi().then(function (a) { a.min(name); }); },
        /* the window's <iframe> (also iframe[data-si-window="name"]), or null */
        frame: function (name) { return window.siWeb ? window.siWeb.api.frame(name) : null; },
        names: function () { return window.siWeb ? window.siWeb.api.names() : []; }
      },
      /* the deck: slide k (1-based) at build step s; next / prev as → / ← */
      slide: function (k, s) { go(k - 1, s || 0); return Promise.resolve(siPlay.at()); },
      next: function () { forward(); return Promise.resolve(siPlay.at()); },
      prev: function () { backward(); return Promise.resolve(siPlay.at()); },
      at: function () { return { slide: cur + 1, step: step, slides: n }; },
      /* a Notes: script's cues, in order (deck.Act: {tool, name, cmd, state, url, o}) */
      act: function (acts) {
        return (acts || []).reduce(function (p, a) {
          return p.then(function () {
            if (a.tool === 'term') {
              if (a.opacity) siPlay.terminal.opacity(a.opacity);
              if (a.state && !a.name) return siPlay.terminal.state(a.state);
              /* · split right|down, then · max|min, then the command (· typing slow|fast: ms per character) */
              return siPlay.terminal.open(a.name, { split: a.split }).then(function () {
                return a.state && siPlay.terminal.state(a.state);
              }).then(function () {
                if (a.cmd) return siPlay.terminal.run(a.name, a.cmd, { typing: a.typing || 45 });
              });
            }
            if (a.tool === 'browser') {
              if (a.state) return siPlay.browser.close();
              var w = siPlay.browser.open(a.name, a.o); /* opened, or raised and placed */
              return a.url ? w.then(function () { return siPlay.browser.go(a.name, a.url); }) : w;
            }
          });
        }, Promise.resolve());
      }
    };
  }
