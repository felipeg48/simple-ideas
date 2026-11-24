/* ---------- text to speech (feature: t2s): a Listen bar under the page
 * title reads the page aloud with the browser's own voice (speechSynthesis:
 * no network, works from file://). The page is read in order (title, lede,
 * headings, paragraphs, list items, quotes, captions); code blocks, tables,
 * figures, navigation and anything marked .si-t2s-skip are left out. One
 * utterance per sentence, so pause, skip and speed changes are exact and
 * long pages do not hit the engines' length limits. What is being read is
 * lit by window.SI_T2S.highlight: sentence (default), word, paragraph or
 * off; sentence and word use the CSS Custom Highlight API (no change to
 * the page's markup), else the paragraph gets .si-t2s-on. While it plays,
 * a click on a paragraph reads from there; l plays or pauses. ---------- */
(function () {
  var cfg = window.SI_T2S || {};
  var synth = window.speechSynthesis;
  if (!synth || !window.SpeechSynthesisUtterance) return;
  var main = document.querySelector('.si-doc-body') || document.querySelector('main');
  if (!main) return;
  var mode = cfg.highlight || 'sentence';
  var lang = document.documentElement.lang || '';
  var HL = !!(window.CSS && CSS.highlights && window.Highlight);
  var BLOCK = 'h1,h2,h3,h4,h5,h6,p,li,dt,dd,blockquote,figcaption,td,th,summary,.si-lede';
  var SKIP = 'pre,svg,math,script,style,noscript,nav,figure,table,button,input,select,textarea,kbd,.si-doc-toc,.si-doc-pager,.si-doc-edit,.si-t2s,.si-copy,.si-anchor,.si-doc-meta,.si-book-chap,.si-doc-crumb,.si-t2s-skip,[aria-hidden="true"],[hidden]';
  function store(k, v) { try { if (v === undefined) return localStorage.getItem('si-t2s-' + k); localStorage.setItem('si-t2s-' + k, v); } catch (e) { return null; } }

  /* the page as units (one block's run of text) and sentences in them */
  function collect() {
    var title = document.querySelector('.si-doc-title');
    var roots = [];
    if (title && !main.contains(title)) {
      roots.push(title);
      var lede = document.querySelector('.si-lede');
      if (lede && !main.contains(lede)) roots.push(lede);
    }
    roots.push(main);
    var units = [], unit = null, seen = new Map();
    roots.forEach(function (root) {
      var w = document.createTreeWalker(root, NodeFilter.SHOW_TEXT);
      for (var t = w.nextNode(); t; t = w.nextNode()) {
        var p = t.parentElement;
        if (!p || !t.nodeValue || p.closest(SKIP)) continue;
        var el = p.closest(BLOCK) || p;
        if (!root.contains(el) && el !== root) el = root;
        if (!seen.has(el)) seen.set(el, el.getClientRects().length > 0); /* closed details, display:none */
        if (!seen.get(el)) continue;
        if (!unit || unit.el !== el || unit.root !== root) { unit = { el: el, root: root, nodes: [], text: '' }; units.push(unit); }
        var num = p.closest('[class$="-num"]'); /* 1.1 / step 1: a section or step number joins its heading */
        if (unit.num && unit.num !== num && !/\s$/.test(unit.text)) unit.text += ' ';
        unit.num = num;
        unit.nodes.push({ node: t, start: unit.text.length });
        unit.text += t.nodeValue;
      }
      unit = null;
    });
    var seg = window.Intl && Intl.Segmenter ? new Intl.Segmenter(lang || undefined, { granularity: 'sentence' }) : null;
    var out = [];
    units.forEach(function (u) {
      var parts = [];
      if (seg) { for (var s of seg.segment(u.text)) parts.push([s.index, s.segment]); }
      else { var re = /[^.!?…]+(?:[.!?…]+["'”’)\]]*|$)\s*/g, m; while ((m = re.exec(u.text)) && m[0]) parts.push([m.index, m[0]]); }
      parts.forEach(function (p) {
        var lead = p[1].length - p[1].replace(/^\s+/, '').length, txt = p[1].trim();
        if (/[\p{L}\p{N}]/u.test(txt)) out.push({ u: u, start: p[0] + lead, end: p[0] + lead + txt.length, text: txt.replace(/\s+/g, ' ') });
      });
    });
    return out;
  }
  function point(u, off) {
    for (var i = u.nodes.length - 1; i >= 0; i--) {
      var n = u.nodes[i];
      if (off >= n.start) return [n.node, Math.min(off - n.start, n.node.nodeValue.length)];
    }
    return [u.nodes[0].node, 0];
  }
  function range(u, a, b) {
    var r = document.createRange(), s = point(u, a), e = point(u, b);
    try { r.setStart(s[0], s[1]); r.setEnd(e[0], e[1]); } catch (x) { return null; }
    return r;
  }

  var sents = collect();
  if (!sents.length) return;
  var words = 0;
  sents.forEach(function (s) { words += s.text.split(' ').length; });

  /* the bar, under the title (and its lede and byline) */
  var bar = document.createElement('div');
  bar.className = 'si-t2s';
  bar.setAttribute('role', 'region');
  bar.setAttribute('aria-label', 'Listen to this page');
  bar.innerHTML = '<button type="button" class="si-t2s-play" aria-label="Listen (l)" title="Listen (l)"><span class="si-t2s-icon" aria-hidden="true">▶</span><span class="si-t2s-label">Listen</span></button>' +
    '<span class="si-t2s-time"></span>' +
    '<span class="si-t2s-tools" hidden>' +
    '<button type="button" data-t2s-do="prev" aria-label="Previous sentence" title="Previous sentence">⏮</button>' +
    '<button type="button" data-t2s-do="stop" aria-label="Stop" title="Stop">⏹</button>' +
    '<button type="button" data-t2s-do="next" aria-label="Next sentence" title="Next sentence">⏭</button>' +
    '<select class="si-t2s-rate" aria-label="Speed">' + [0.75, 1, 1.25, 1.5, 1.75, 2].map(function (r) { return '<option value="' + r + '">' + r + '×</option>'; }).join('') + '</select>' +
    '<select class="si-t2s-voice" aria-label="Voice"></select>' +
    '</span>';
  var title = document.querySelector('.si-doc-title');
  var at = title;
  while (at && at.nextElementSibling && at.nextElementSibling.matches('.si-lede,.si-doc-meta')) at = at.nextElementSibling;
  if (at) at.after(bar); else main.prepend(bar);
  function q(s) { return bar.querySelector(s); }
  var rateSel = q('.si-t2s-rate'), voiceSel = q('.si-t2s-voice');
  var rate = parseFloat(store('rate')) || parseFloat(cfg.rate) || 1;
  if (![].some.call(rateSel.options, function (o) { return +o.value === rate; })) rateSel.add(new Option(rate + '×', rate), 0);
  rateSel.value = String(rate);

  var voices = [], voice = null;
  function loadVoices() {
    var all = synth.getVoices();
    if (!all.length) return;
    var base = lang.split('-')[0].toLowerCase();
    voices = all.filter(function (v) { return !base || v.lang.toLowerCase().split(/[-_]/)[0] === base; });
    if (!voices.length) voices = all;
    var want = store('voice') || cfg.voice || '';
    voice = voices.filter(function (v) { return v.name === want; })[0] ||
      (want && voices.filter(function (v) { return v.name.toLowerCase().indexOf(want.toLowerCase()) >= 0; })[0]) ||
      voices.filter(function (v) { return v.default; })[0] || voices.filter(function (v) { return v.localService; })[0] || voices[0];
    voiceSel.innerHTML = '';
    voices.forEach(function (v) { voiceSel.add(new Option(v.name + (v.localService ? '' : ' ☁'), v.name, false, v === voice)); });
    voiceSel.hidden = voices.length < 2;
  }
  loadVoices();
  if (synth.addEventListener) synth.addEventListener('voiceschanged', loadVoices);

  function minutes() { return Math.max(1, Math.round(words / (165 * rate))); }
  var st = { state: 'off', i: 0, gen: 0 };
  function show() {
    var playing = st.state === 'playing';
    bar.setAttribute('data-state', st.state);
    q('.si-t2s-icon').textContent = playing ? '❚❚' : '▶';
    q('.si-t2s-label').textContent = playing ? 'Pause' : st.state === 'paused' ? 'Resume' : 'Listen';
    q('.si-t2s-play').setAttribute('aria-pressed', playing ? 'true' : 'false');
    q('.si-t2s-tools').hidden = st.state === 'off';
    q('.si-t2s-time').textContent = st.state === 'off' ? minutes() + ' min listen' : (st.i + 1) + ' / ' + sents.length;
  }

  var lit = null;
  function light(s, w) {
    if (lit) lit.classList.remove('si-t2s-on');
    lit = null;
    if (HL) { CSS.highlights.delete('si-t2s'); CSS.highlights.delete('si-t2s-word'); }
    if (!s || mode === 'off') return;
    if (mode === 'paragraph' || !HL) { lit = s.u.el; lit.classList.add('si-t2s-on'); }
    else {
      var r = range(s.u, s.start, s.end);
      if (r) CSS.highlights.set('si-t2s', new Highlight(r));
      if (w) { var wr = range(s.u, w[0], w[1]); if (wr) CSS.highlights.set('si-t2s-word', new Highlight(wr)); }
    }
  }
  var userScroll = 0;
  addEventListener('wheel', function () { userScroll = Date.now(); }, { passive: true });
  addEventListener('touchmove', function () { userScroll = Date.now(); }, { passive: true });
  function follow(s) {
    if (Date.now() - userScroll < 4000) return; /* the reader is looking elsewhere */
    var r = s.u.el.getBoundingClientRect();
    if (r.top < 0 || r.bottom > innerHeight) {
      var calm = matchMedia('(prefers-reduced-motion: reduce)').matches;
      s.u.el.scrollIntoView({ block: 'center', behavior: calm ? 'auto' : 'smooth' });
    }
  }

  function speak() {
    var g = ++st.gen;
    synth.cancel();
    if (st.i >= sents.length) { stop(); return; }
    var s = sents[st.i], u = new SpeechSynthesisUtterance(s.text);
    u.rate = rate;
    if (voice) { u.voice = voice; u.lang = voice.lang; } else if (lang) u.lang = lang;
    u.onstart = function () { if (g === st.gen) { light(s); follow(s); show(); } };
    if (mode === 'word') u.onboundary = function (e) {
      if (g !== st.gen || e.name !== 'word') return;
      var len = e.charLength || ((/^[^\s]+/.exec(s.text.slice(e.charIndex)) || [''])[0].length);
      var a = s.start + map(s, e.charIndex), b = s.start + map(s, e.charIndex + len);
      light(s, [a, b]);
    };
    u.onend = function () { if (g === st.gen && st.state === 'playing') { st.i++; speak(); } };
    u.onerror = function (e) { if (g === st.gen && e.error !== 'interrupted' && e.error !== 'canceled') { console.warn('si t2s:', e.error); stop(); } };
    synth.speak(u);
    if (synth.paused) synth.resume(); /* Chrome can start paused after an earlier pause */
  }
  /* an offset in the spoken text (whitespace collapsed) to one in the unit's text */
  function map(s, off) {
    var raw = s.u.text.slice(s.start, s.end), k = 0, j = 0;
    while (j < raw.length && k < off) {
      if (/\s/.test(raw[j])) { while (j + 1 < raw.length && /\s/.test(raw[j + 1])) j++; }
      j++; k++;
    }
    return j;
  }
  function play(i) {
    if (typeof i === 'number') st.i = Math.max(0, Math.min(sents.length - 1, i));
    st.state = 'playing';
    show();
    speak();
  }
  /* pause is a cancel that remembers the sentence: synth.pause() is unreliable across engines */
  function pause() { st.state = 'paused'; st.gen++; synth.cancel(); show(); }
  function stop() { st.state = 'off'; st.gen++; synth.cancel(); st.i = 0; light(null); show(); }
  function toggle() { if (st.state === 'playing') pause(); else play(); }

  q('.si-t2s-play').addEventListener('click', toggle);
  bar.addEventListener('click', function (e) {
    var b = e.target.closest('[data-t2s-do]');
    if (!b) return;
    var d = b.getAttribute('data-t2s-do');
    if (d === 'stop') stop();
    else play(st.i + (d === 'next' ? 1 : -1));
  });
  rateSel.addEventListener('change', function () {
    rate = parseFloat(rateSel.value) || 1;
    store('rate', rate);
    if (st.state === 'playing') speak(); else show();
  });
  voiceSel.addEventListener('change', function () {
    voice = voices.filter(function (v) { return v.name === voiceSel.value; })[0] || voice;
    store('voice', voiceSel.value);
    if (st.state === 'playing') speak();
  });
  /* while it reads: a click on a paragraph (not a link or a control) reads from there */
  document.addEventListener('click', function (e) {
    if (st.state === 'off' || e.defaultPrevented || e.target.closest('a,button,input,select,textarea,summary,.si-t2s')) return;
    if (String(getSelection()).length) return;
    for (var i = 0; i < sents.length; i++) {
      if (sents[i].u.el === e.target || sents[i].u.el.contains(e.target)) { play(i); return; }
    }
  });
  document.addEventListener('keydown', function (e) {
    if (e.key !== 'l' || e.metaKey || e.ctrlKey || e.altKey || (e.target.closest && e.target.closest('input,textarea,select,[contenteditable]'))) return;
    e.preventDefault();
    toggle();
  });
  addEventListener('pagehide', function () { synth.cancel(); }); /* some engines keep talking after the page is gone */
  show();
})();
