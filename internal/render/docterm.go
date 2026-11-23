package render

// docTermJS is the terminal drawer on a lab page in si play and on a class
// seat (RULES §37): a Terminal button and the ` key open it (xterm.js and
// term.js load on first use from /_si/term/assets/, as in a deck), and each
// shell code block gets a Run button that types it into the drawer's shell.
// Never in a built page: Options.Terminal is false there.
const docTermJS = `(function () {
  var loaded = false, then = [];
  function load(f) {
    if (window.siTerm) { if (f) f(); return; }
    if (f) then.push(f);
    if (loaded) return;
    loaded = true;
    var css = document.createElement('link'); css.rel = 'stylesheet'; css.href = '/_si/term/assets/xterm.css'; document.head.appendChild(css);
    function add(src, next) { var s = document.createElement('script'); s.src = src; s.onload = next; document.head.appendChild(s); }
    add('/_si/term/assets/xterm.js', function () { add('/_si/term/assets/addon-fit.js', function () { add('/_si/term/assets/term.js', function () {
      if (window.siTerm) then.splice(0).forEach(function (g) { g(); });
    }); }); });
  }
  function toggle() { load(function () { window.siTerm.toggle(); }); }
  var st = document.createElement('style');
  st.textContent = '.si-doc-term{position:fixed;right:1rem;bottom:1rem;z-index:8999;font:600 .85rem/1 system-ui,sans-serif;padding:.65em 1em;border-radius:2em;border:0;background:#0b0e14;color:#e6e6e6;box-shadow:0 4px 16px rgba(0,0,0,.3);cursor:pointer}' +
    '.si-doc-term:hover{background:#1d2536}.si-run{float:right;margin:-.2em -.2em 0 .6em;font:600 .75rem/1 system-ui,sans-serif;padding:.35em .7em;border-radius:.4em;border:1px solid currentColor;background:transparent;color:inherit;opacity:.7;cursor:pointer}.si-run:hover{opacity:1}';
  document.head.appendChild(st);
  var b = document.createElement('button');
  b.type = 'button'; b.className = 'si-doc-term'; b.textContent = '>_ Terminal'; b.title = 'Terminal (\x60)';
  b.addEventListener('click', toggle);
  document.body.appendChild(b);
  document.addEventListener('keydown', function (e) {
    var t = e.target;
    if (e.key !== '\x60' || e.metaKey || e.ctrlKey || e.altKey) return;
    if (t && (t.isContentEditable || /INPUT|TEXTAREA|SELECT/.test(t.tagName) || (t.closest && t.closest('.si-term')))) return;
    e.preventDefault(); toggle();
  });
  try { if ((JSON.parse(sessionStorage.getItem('si-term-state') || '{}').tabs || []).length) load(); } catch (e) {}
  var shells = /^(bash|sh|shell|zsh|console|terminal)$/;
  Array.prototype.forEach.call(document.querySelectorAll('pre[data-lang]'), function (pre) {
    if (!shells.test(pre.getAttribute('data-lang')) || pre.closest('.si-check')) return;
    var code = pre.querySelector('code'); if (!code) return;
    var run = document.createElement('button');
    run.type = 'button'; run.className = 'si-run'; run.textContent = 'Run'; run.title = 'Type it into the terminal';
    run.addEventListener('click', function () {
      var text = code.textContent.split('\n').map(function (l) { return l.replace(/^\$ /, ''); }).join('\n').replace(/\s+$/, '');
      load(function () {
        var api = window.siTerm.api, n = api.names()[0] || 'lab';
        api.open(n).then(function () { return api.run(n, text); });
      });
    });
    pre.insertBefore(run, pre.firstChild);
  });
})();
`
