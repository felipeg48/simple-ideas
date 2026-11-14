(function () {
  // si demo draft: what one page says, read from its rendered DOM: its title,
  // its hero (the first heading, its key sentence, its call to action), its
  // sections (a heading each, a sentence, a call to action), a cookie
  // banner's accept button and the same-origin links to follow. Every element
  // comes with a CSS selector that matches it alone, so si demo test finds it.
  var norm = function (s) { return String(s || '').replace(/\s+/g, ' ').trim(); };
  var visible = function (el) {
    if (!el || !el.isConnected) return false;
    var r = el.getBoundingClientRect();
    if (r.width < 2 || r.height < 2) return false;
    return el.checkVisibility ? el.checkVisibility({ checkOpacity: true, checkVisibilityCSS: true }) : true;
  };
  var okID = function (id) { return /^[A-Za-z][\w-]*$/.test(id || '') && document.querySelectorAll('#' + id).length === 1; };
  var sel = function (el) {
    if (!el) return '';
    if (okID(el.id)) return '#' + el.id;
    var path = [], e = el;
    while (e && e !== document.body && e !== document.documentElement) {
      if (e !== el && okID(e.id)) { path.unshift('#' + e.id); break; }
      var t = e.tagName.toLowerCase(), i = 1, n = 0, s = e;
      while ((s = s.previousElementSibling)) if (s.tagName === e.tagName) i++;
      for (var c = e.parentElement ? e.parentElement.firstElementChild : null; c; c = c.nextElementSibling) if (c.tagName === e.tagName) n++;
      path.unshift(n > 1 ? t + ':nth-of-type(' + i + ')' : t);
      e = e.parentElement;
    }
    if (!path.length || path[0].charAt(0) !== '#') path.unshift('body');
    var css = path.join(' > ');
    try { return document.querySelectorAll(css).length === 1 ? css : ''; } catch (x) { return ''; }
  };
  // the key sentence: the first sentence of the first paragraph of some length
  var sentence = function (root, after) {
    var ps = root.querySelectorAll('p, [class*="lead"], [class*="sub"], li');
    for (var i = 0; i < ps.length; i++) {
      var p = ps[i];
      if (after && (after.compareDocumentPosition(p) & Node.DOCUMENT_POSITION_PRECEDING)) continue;
      if (!visible(p) || p.closest('nav, footer, form, button, a')) continue;
      var t = norm(p.innerText);
      if (t.length < 30) continue;
      var m = t.match(/^.{20,}?[.!?](?=\s|$)/);
      t = m ? m[0] : t;
      if (t.length > 240) t = t.slice(0, 240).replace(/\s+\S*$/, '') + '…';
      return t;
    }
    return '';
  };
  var cta = function (root) {
    var as = root.querySelectorAll('a[href], button, [role=button]');
    for (var i = 0; i < as.length; i++) {
      var a = as[i], t = norm(a.innerText || a.getAttribute('aria-label'));
      if (!visible(a) || !t || t.length > 40 || a.closest('nav, form')) continue;
      var s = sel(a);
      if (s) return { text: t, sel: s };
    }
    return null;
  };
  var origin = location.origin, out = { title: norm(document.title), sections: [], links: [] };
  var main = document.querySelector('main') || document.body;
  var h1 = [].filter.call(document.querySelectorAll('h1'), visible)[0];
  if (h1) {
    var box = h1.closest('header, section, [class*="hero"], main > div, article') || h1.parentElement;
    out.hero = { heading: norm(h1.innerText), sel: sel(h1), sentence: sentence(box, h1) || sentence(main, h1), cta: cta(box) };
  }
  // the sections: blocks with a heading of their own, outermost first
  var seen = [];
  var blocks = main.querySelectorAll('section, article, main > div, [id]');
  for (var i = 0; i < blocks.length && out.sections.length < 8; i++) {
    var b = blocks[i];
    if (!visible(b) || b.closest('nav, footer, header, form, dialog, [role=dialog]')) continue;
    if (h1 && b.contains(h1)) continue;
    if (seen.some(function (s) { return s.contains(b) || b.contains(s); })) continue;
    var h = [].filter.call(b.querySelectorAll('h2, h3'), visible)[0];
    if (!h || !sel(h)) continue;
    seen.push(b);
    out.sections.push({ heading: norm(h.innerText), sel: sel(h), sentence: sentence(b, h), cta: cta(b) });
  }
  // a cookie or consent banner: its accept button
  var ACCEPT = /^(accept( all)?( cookies)?|allow( all)?( cookies)?|agree|i agree|got it|ok|okay)$/i;
  var btns = document.querySelectorAll('button, a, [role=button]');
  for (var k = 0; k < btns.length; k++) {
    var bt = btns[k], bx = norm(bt.innerText);
    if (!visible(bt) || !ACCEPT.test(bx)) continue;
    for (var p = bt; p && p !== document.body; p = p.parentElement) {
      if (/cookie|consent|gdpr|privacy/i.test((p.id || '') + ' ' + (typeof p.className === 'string' ? p.className : '') + ' ' + (p.getAttribute('aria-label') || ''))) {
        out.cookie = { text: bx, sel: sel(bt) };
        break;
      }
    }
    if (out.cookie) break;
  }
  // the links to follow: the navigation's first, then the page's
  var robots = (document.querySelector('meta[name=robots]') || {}).content || '';
  if (!/nofollow/i.test(robots)) {
    var add = function (a) {
      if (!/\bnofollow\b/i.test(a.rel || '') && a.href.indexOf(origin) === 0 && !a.hasAttribute('download')) {
        out.links.push({ href: a.href.split('#')[0], text: norm(a.innerText || a.getAttribute('aria-label')) });
      }
    };
    [].forEach.call(document.querySelectorAll('header a[href], nav a[href]'), add);
    [].forEach.call(document.querySelectorAll('main a[href]'), add);
  }
  return out;
})()
