/* si video player: one source for si pages and the drop-in (si --export player).
   It turns into players:
   - .si-vp figures (a ```video fence): inline and facade play in place; modal
     opens the shared dialog;
   - video[data-si-player] (![](x.mp4) in documents, course video lessons);
   - any element with data-video="clip.mp4" (pages si did not build): optional
     data-poster (else the first <img> inside), data-title, data-caption,
     data-captions (.vtt), data-chapters (.txt or .vtt), data-mode=inline|modal
     (modal by default), data-group, data-start.
   The HUD: play/pause, scrub bar (buffered, hover time, chapter marks), time,
   volume and mute, speed, captions, chapters, picture in picture, fullscreen.
   Keys while a player has focus: Space/K, <- -> (5 s), J/L (10 s), M, F, C,
   Home/End, Esc. Touch: tap shows or hides the HUD, a double tap on a side
   seeks 10 s. Every modal-mode player on a page (or with the same group) shares
   one dialog: < > and <- -> while paused (Shift+<- -> while playing) step
   through them. window.siPlayer.init(root) wires players added later. */
(function () {
  'use strict';
  if (window.siPlayer) { return; }
  var SPEEDS = [1, 1.25, 1.5, 2];
  var ICON = {
    play: '<path d="M8 5v14l11-7z"/>',
    pause: '<path d="M6 5h4v14H6zM14 5h4v14h-4z"/>',
    vol: '<path d="M4 9v6h4l5 4V5L8 9zm12.5 3A4.5 4.5 0 0 0 14 8v8a4.5 4.5 0 0 0 2.5-4z"/>',
    mute: '<path d="M4 9v6h4l5 4V5L8 9zm11.6.4 1.4-1.4 2 2 2-2 1.4 1.4-2 2 2 2-1.4 1.4-2-2-2 2-1.4-1.4 2-2z"/>',
    list: '<path d="M4 6h2v2H4zm4 0h12v2H8zm-4 5h2v2H4zm4 0h12v2H8zm-4 5h2v2H4zm4 0h12v2H8z"/>',
    full: '<path d="M4 4h6v2H6v4H4zm10 0h6v6h-2V6h-4zM4 14h2v4h4v2H4zm14 0h2v6h-6v-2h4z"/>',
    pip: '<path fill-rule="evenodd" d="M3 5h18v14H3zm2 2v10h14V7zm7 5h6v4h-6z"/>',
    prev: '<path d="M15.4 5.4 14 4l-8 8 8 8 1.4-1.4L8.8 12z"/>',
    next: '<path d="M8.6 5.4 10 4l8 8-8 8-1.4-1.4 6.6-6.6z"/>',
    close: '<path d="M6.4 5 12 10.6 17.6 5 19 6.4 13.4 12l5.6 5.6-1.4 1.4-5.6-5.6L6.4 19 5 17.6l5.6-5.6L5 6.4z"/>'
  };
  function svg(n) { return '<svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">' + ICON[n] + '</svg>'; }
  function el(tag, cls, html) { var e = document.createElement(tag); if (cls) e.className = cls; if (html != null) e.innerHTML = html; return e; }
  function btn(cls, label, icon) { var b = el('button', 'si-vp-b ' + cls, icon ? svg(icon) : ''); b.type = 'button'; b.setAttribute('aria-label', label); b.title = label; return b; }
  function fmt(t) {
    if (!isFinite(t) || t < 0) t = 0;
    t = Math.floor(t);
    var h = Math.floor(t / 3600), m = Math.floor(t / 60) % 60, s = t % 60;
    return (h ? h + ':' + (m < 10 ? '0' : '') : '') + m + ':' + (s < 10 ? '0' : '') + s;
  }
  function clock(s) { /* 1:02:03, 2:31, 12.5, 00:01:02.500 */
    var p = String(s).trim().split(':'), t = 0;
    for (var i = 0; i < p.length; i++) { var n = parseFloat(p[i].replace(',', '.')); if (isNaN(n)) return NaN; t = t * 60 + n; }
    return t;
  }
  /* chapters from text: "0:00 Intro" lines (YouTube style, si's chapters.txt)
     or a WebVTT file whose cues are the chapters */
  function parseChapters(txt) {
    var out = [], lines = String(txt).replace(/\r/g, '').split('\n');
    if (/^WEBVTT/.test(lines[0])) {
      for (var i = 0; i < lines.length; i++) {
        var m = /^\s*([\d:.,]+)\s+-->\s+/.exec(lines[i]);
        if (m && lines[i + 1] && lines[i + 1].trim()) out.push([clock(m[1]), lines[i + 1].trim()]);
      }
    } else {
      lines.forEach(function (l) {
        var m = /^\s*\(?([\d:.]+)\)?\s+[-–—]?\s*(.+)$/.exec(l);
        if (m && !isNaN(clock(m[1]))) out.push([clock(m[1]), m[2].trim()]);
      });
    }
    return out.filter(function (c) { return !isNaN(c[0]); });
  }
  function play(v) { try { var p = v.play(); if (p && p.catch) p.catch(function () {}); } catch (e) {} }

  /* ---------- one player: a <video> wrapped in a box with the HUD ---------- */
  function Player(v, o) {
    o = o || {};
    var P = this, box = el('div', 'si-vp-box');
    P.v = v; P.o = o; P.box = box;
    v.parentNode.insertBefore(box, v);
    box.appendChild(v);
    v.controls = false;
    v.removeAttribute('controls');
    v.setAttribute('playsinline', '');
    v._siVp = P;
    box.tabIndex = -1;
    box.setAttribute('role', 'group');
    box.dataset.state = 'paused';

    var big = btn('si-vp-big', 'Play', 'play');
    var cue = el('div', 'si-vp-cue'); cue.setAttribute('aria-hidden', 'true'); cue.hidden = true;
    var flash = el('div', 'si-vp-flash'); flash.setAttribute('aria-hidden', 'true');
    var live = el('div', 'si-vp-live'); live.setAttribute('aria-live', 'polite');
    var hud = el('div', 'si-vp-hud');
    var bar = el('div', 'si-vp-bar'); bar.tabIndex = 0; bar.setAttribute('role', 'slider'); bar.setAttribute('aria-label', 'Seek'); bar.setAttribute('aria-valuemin', '0');
    var track = el('div', 'si-vp-track'), buf = el('div', 'si-vp-buf'), fill = el('div', 'si-vp-fill'), marks = el('div', 'si-vp-marks'), tip = el('div', 'si-vp-tip');
    track.appendChild(buf); track.appendChild(fill); track.appendChild(marks); bar.appendChild(track); bar.appendChild(tip);
    var row = el('div', 'si-vp-row');
    var pp = btn('si-vp-pp', 'Play (k)', 'play');
    var mute = btn('si-vp-mute', 'Mute (m)', 'vol');
    var vol = el('input', 'si-vp-vol'); vol.type = 'range'; vol.min = '0'; vol.max = '1'; vol.step = '0.05'; vol.setAttribute('aria-label', 'Volume');
    var time = el('span', 'si-vp-time', '0:00 / 0:00');
    var chname = el('span', 'si-vp-chname');
    var gap = el('span', 'si-vp-gap');
    var chap = btn('si-vp-chap', 'Chapters', 'list'); chap.setAttribute('aria-haspopup', 'true'); chap.setAttribute('aria-expanded', 'false');
    var menu = el('div', 'si-vp-menu'); menu.hidden = true; menu.setAttribute('role', 'menu'); menu.setAttribute('aria-label', 'Chapters');
    var cc = btn('si-vp-cc', 'Captions (c)'); cc.innerHTML = '<span aria-hidden="true">CC</span>'; cc.setAttribute('aria-pressed', 'false');
    var speed = btn('si-vp-speed', 'Speed'); speed.textContent = '1×';
    var pip = btn('si-vp-pip', 'Picture in picture', 'pip');
    var full = btn('si-vp-full', 'Fullscreen (f)', 'full');
    [pp, mute, vol, time, chname, gap, chap, cc, speed, pip, full].forEach(function (x) { row.appendChild(x); });
    hud.appendChild(bar); hud.appendChild(row); hud.appendChild(menu);
    [big, cue, flash, live, hud].forEach(function (x) { box.appendChild(x); });
    pip.hidden = !(document.pictureInPictureEnabled && !v.disablePictureInPicture);
    var canFull = !!(box.requestFullscreen || box.webkitRequestFullscreen || v.webkitEnterFullscreen);
    full.hidden = !canFull;

    P.chapters = [];
    function dur() { return isFinite(v.duration) ? v.duration : 0; }
    function say(t) { live.textContent = t; }
    function showFlash(t) {
      flash.textContent = t; flash.classList.remove('is-on'); void flash.offsetWidth; flash.classList.add('is-on');
    }
    function setLabel(t) { box.setAttribute('aria-label', t ? 'Video player: ' + t : 'Video player'); big.setAttribute('aria-label', t ? 'Play: ' + t : 'Play'); }
    setLabel(o.title || v.getAttribute('aria-label') || '');

    P.toggle = function () { if (v.paused || v.ended) P.play(); else v.pause(); };
    P.play = function () {
      if (!v.getAttribute('src') && v.dataset.vpSrc) { v.src = v.dataset.vpSrc; }
      play(v);
    };
    P.seek = function (t) { var d = dur(); if (!d) return; v.currentTime = Math.max(0, Math.min(d - 0.05, t)); render(); };
    P.skip = function (dt) { P.seek(v.currentTime + dt); showFlash((dt > 0 ? '+' : '−') + Math.abs(dt) + ' s'); wake(); };
    function textTrack() {
      for (var i = 0; i < v.textTracks.length; i++) { var k = v.textTracks[i].kind; if (k === 'captions' || k === 'subtitles') return v.textTracks[i]; }
      return null;
    }
    P.captions = function (on) {
      var t = textTrack(); if (!t) return;
      if (on == null) on = !box.hasAttribute('data-cc');
      t.mode = 'hidden'; /* we draw the cue ourselves, above the HUD */
      box.toggleAttribute('data-cc', on);
      cc.setAttribute('aria-pressed', String(on));
      drawCue(); say(on ? 'Captions on' : 'Captions off');
    };
    function drawCue() {
      var t = textTrack(), txt = '';
      if (t && box.hasAttribute('data-cc') && t.activeCues) {
        for (var i = 0; i < t.activeCues.length; i++) txt += (txt ? '\n' : '') + t.activeCues[i].text.replace(/<[^>]+>/g, '');
      }
      cue.textContent = txt; cue.hidden = !txt;
    }
    function wireTracks() {
      cc.hidden = !textTrack();
      for (var i = 0; i < v.textTracks.length; i++) {
        var t = v.textTracks[i];
        if (t.mode === 'disabled') t.mode = 'hidden';
        t.oncuechange = drawCue;
      }
      var def = v.querySelector('track[default]');
      if (textTrack()) P.captions(!!def || box.hasAttribute('data-cc'));
    }
    v.textTracks.addEventListener && v.textTracks.addEventListener('addtrack', wireTracks);
    P.setChapters = function (list) {
      P.chapters = (list || []).slice().sort(function (a, b) { return a[0] - b[0]; });
      chap.hidden = !P.chapters.length;
      menu.innerHTML = '';
      P.chapters.forEach(function (c, i) {
        var b = el('button', 'si-vp-mi'); b.type = 'button'; b.setAttribute('role', 'menuitem');
        b.innerHTML = '<span class="si-vp-mt"></span><span class="si-vp-mn"></span>';
        b.firstChild.textContent = fmt(c[0]); b.lastChild.textContent = c[1];
        b.addEventListener('click', function () { P.seek(c[0]); openMenu(false); P.play(); bar.focus(); });
        b.dataset.i = i;
        menu.appendChild(b);
      });
      drawMarks();
    };
    function drawMarks() {
      marks.innerHTML = ''; var d = dur(); if (!d) return;
      P.chapters.forEach(function (c) {
        if (c[0] <= 0) return;
        var m = el('span', 'si-vp-mark'); m.style.insetInlineStart = (100 * c[0] / d) + '%'; marks.appendChild(m);
      });
    }
    function chapterAt(t) { var r = -1; P.chapters.forEach(function (c, i) { if (c[0] <= t + 0.01) r = i; }); return r; }
    function openMenu(on) {
      menu.hidden = !on; chap.setAttribute('aria-expanded', String(on));
      if (on) { var cur = menu.children[Math.max(0, chapterAt(v.currentTime))]; if (cur) cur.focus(); }
    }

    function render() {
      var d = dur(), t = v.currentTime;
      fill.style.inlineSize = d ? (100 * t / d) + '%' : '0';
      time.textContent = fmt(t) + ' / ' + fmt(d);
      bar.setAttribute('aria-valuemax', String(Math.round(d)));
      bar.setAttribute('aria-valuenow', String(Math.round(t)));
      bar.setAttribute('aria-valuetext', fmt(t) + ' of ' + fmt(d));
      var ci = chapterAt(t);
      chname.textContent = ci >= 0 ? P.chapters[ci][1] : '';
      for (var i = 0; i < menu.children.length; i++) menu.children[i].toggleAttribute('aria-current', i === ci);
    }
    function buffered() {
      var d = dur(), b = v.buffered, end = 0;
      for (var i = 0; i < b.length; i++) if (b.start(i) <= v.currentTime + 0.5) end = Math.max(end, b.end(i));
      buf.style.inlineSize = d ? (100 * end / d) + '%' : '0';
    }
    function state() {
      var playing = !v.paused && !v.ended;
      box.dataset.state = playing ? 'playing' : 'paused';
      pp.innerHTML = svg(playing ? 'pause' : 'play');
      pp.setAttribute('aria-label', playing ? 'Pause (k)' : 'Play (k)'); pp.title = pp.getAttribute('aria-label');
      if (playing) box.setAttribute('data-started', '');
      box.toggleAttribute('data-ended', v.ended);
      wake();
    }
    function volume() {
      var m = v.muted || v.volume === 0;
      mute.innerHTML = svg(m ? 'mute' : 'vol');
      mute.setAttribute('aria-label', m ? 'Unmute (m)' : 'Mute (m)'); mute.title = mute.getAttribute('aria-label');
      vol.value = String(v.muted ? 0 : v.volume);
    }
    P.speed = function (r) {
      v.playbackRate = r; speed.textContent = r + '×';
      speed.setAttribute('aria-label', 'Speed ' + r + '×'); say('Speed ' + r);
    };
    P.fullscreen = function () {
      var d = document;
      if (d.fullscreenElement || d.webkitFullscreenElement) { (d.exitFullscreen || d.webkitExitFullscreen).call(d); return; }
      if (box.requestFullscreen) box.requestFullscreen().catch(function () {});
      else if (box.webkitRequestFullscreen) box.webkitRequestFullscreen();
      else if (v.webkitEnterFullscreen) v.webkitEnterFullscreen();
    };

    /* the HUD hides itself while playing, after a pause in pointer moves */
    var idle = 0;
    function wake() {
      box.removeAttribute('data-idle'); clearTimeout(idle);
      if (!v.paused && menu.hidden) idle = setTimeout(function () { if (!v.paused && menu.hidden && !hud.contains(document.activeElement)) box.setAttribute('data-idle', ''); }, 2600);
    }
    P.wake = wake;

    v.addEventListener('play', state); v.addEventListener('pause', state); v.addEventListener('ended', state);
    v.addEventListener('timeupdate', render);
    v.addEventListener('progress', buffered);
    v.addEventListener('durationchange', function () { render(); drawMarks(); });
    v.addEventListener('loadedmetadata', function () {
      if (P.start > 0 && !P.started) { P.started = true; v.currentTime = P.start; }
      render(); drawMarks(); buffered();
    });
    v.addEventListener('volumechange', volume);
    v.addEventListener('ratechange', function () { speed.textContent = v.playbackRate + '×'; });

    big.addEventListener('click', function () { P.play(); box.focus({ preventScroll: true }); });
    pp.addEventListener('click', P.toggle);
    mute.addEventListener('click', function () { v.muted = !v.muted; if (!v.muted && v.volume === 0) v.volume = 0.6; });
    vol.addEventListener('input', function () { v.volume = +vol.value; v.muted = +vol.value === 0; });
    cc.addEventListener('click', function () { P.captions(); });
    speed.addEventListener('click', function () { var i = SPEEDS.indexOf(v.playbackRate); P.speed(SPEEDS[(i + 1) % SPEEDS.length]); });
    chap.addEventListener('click', function () { openMenu(menu.hidden); });
    pip.addEventListener('click', function () {
      if (document.pictureInPictureElement === v) document.exitPictureInPicture().catch(function () {});
      else if (v.requestPictureInPicture) v.requestPictureInPicture().catch(function () {});
    });
    full.addEventListener('click', P.fullscreen);
    document.addEventListener('fullscreenchange', function () { box.toggleAttribute('data-full', document.fullscreenElement === box); });

    /* the scrub bar: drag to seek, hover for the time (and the chapter) */
    function timeAt(e) { var r = track.getBoundingClientRect(); var x = Math.max(0, Math.min(1, (e.clientX - r.left) / r.width)); return { x: x, t: x * dur() }; }
    bar.addEventListener('pointerdown', function (e) { if (e.button) return; bar.setPointerCapture(e.pointerId); bar.dataset.drag = ''; P.seek(timeAt(e).t); });
    bar.addEventListener('pointermove', function (e) {
      var a = timeAt(e); if (bar.hasAttribute('data-drag')) P.seek(a.t);
      var ci = chapterAt(a.t);
      tip.textContent = fmt(a.t) + (ci >= 0 ? ' · ' + P.chapters[ci][1] : '');
      tip.style.insetInlineStart = (a.x * 100) + '%'; tip.classList.add('is-on');
    });
    bar.addEventListener('pointerup', function () { bar.removeAttribute('data-drag'); });
    bar.addEventListener('pointerleave', function () { tip.classList.remove('is-on'); });

    /* mouse: a click plays or pauses; touch: a tap shows or hides the HUD,
       a double tap on the left or right third seeks 10 s */
    var lastTap = 0, tapTimer = 0;
    v.addEventListener('pointerup', function (e) {
      if (e.pointerType === 'mouse') { P.toggle(); box.focus({ preventScroll: true }); return; }
      var now = Date.now(), r = v.getBoundingClientRect(), x = (e.clientX - r.left) / r.width;
      if (now - lastTap < 320) {
        clearTimeout(tapTimer); lastTap = 0;
        if (x < 1 / 3) P.skip(-10); else if (x > 2 / 3) P.skip(10); else P.toggle();
        return;
      }
      lastTap = now;
      tapTimer = setTimeout(function () {
        if (!box.hasAttribute('data-started')) { P.play(); return; }
        if (box.hasAttribute('data-idle') || v.paused) wake(); else { clearTimeout(idle); box.setAttribute('data-idle', ''); }
      }, 330);
    });
    box.addEventListener('pointermove', function (e) { if (e.pointerType === 'mouse') wake(); });
    box.addEventListener('focusin', wake);

    box.addEventListener('keydown', function (e) {
      if (e.ctrlKey || e.metaKey || e.altKey) return;
      var k = e.key, tag = e.target.tagName, onRange = e.target === vol;
      var done = true;
      if (!menu.hidden && k === 'Escape') { openMenu(false); chap.focus(); }
      else if (!menu.hidden && (k === 'ArrowDown' || k === 'ArrowUp')) {
        var items = Array.prototype.slice.call(menu.children), i = items.indexOf(document.activeElement);
        items[(i + (k === 'ArrowDown' ? 1 : items.length - 1)) % items.length].focus();
      }
      else if (k === ' ' && (tag === 'BUTTON' || onRange)) done = false;
      else if (k === ' ' || k === 'k' || k === 'K') P.toggle();
      else if ((k === 'ArrowLeft' || k === 'ArrowRight') && !onRange) {
        var dir = k === 'ArrowLeft' ? -1 : 1;
        if (o.nav && (e.shiftKey || v.paused)) o.nav(dir); else P.skip(5 * dir);
      }
      else if (k === 'j' || k === 'J') P.skip(-10);
      else if (k === 'l' || k === 'L') P.skip(10);
      else if (k === 'm' || k === 'M') v.muted = !v.muted;
      else if (k === 'f' || k === 'F') P.fullscreen();
      else if (k === 'c' || k === 'C') P.captions();
      else if (k === 'Home') P.seek(0);
      else if (k === 'End') P.seek(dur());
      else if (k === 'Escape' && !o.nav && !document.fullscreenElement) { box.blur(); done = false; }
      else done = false;
      if (done) { e.preventDefault(); e.stopPropagation(); wake(); }
    });

    /* load another clip into this player (the shared dialog) */
    P.load = function (it) {
      v.pause();
      Array.prototype.slice.call(v.querySelectorAll('track')).forEach(function (t) { t.remove(); });
      box.removeAttribute('data-started'); box.removeAttribute('data-ended'); box.removeAttribute('data-cc');
      if (it.poster) v.poster = it.poster; else v.removeAttribute('poster');
      v.loop = !!it.loop;
      P.start = it.start || 0; P.started = false;
      (it.tracks || []).forEach(function (t) {
        var tr = document.createElement('track');
        tr.kind = t.kind || 'captions'; tr.src = t.src; tr.srclang = t.srclang || 'en'; tr.label = t.label || 'Captions';
        if (t.default) tr.default = true;
        v.appendChild(tr);
      });
      v.src = it.src;
      v.load();
      setLabel(it.title);
      P.setChapters(it.chapters || []);
      if (it.chaptersURL) loadChapters(P, it.chaptersURL);
      cc.hidden = !(it.tracks && it.tracks.length);
      setTimeout(wireTracks, 0);
      render();
    };

    P.start = +(v.dataset.vpStart || 0);
    try { P.setChapters(JSON.parse(v.dataset.vpChapters || '[]')); } catch (e) { P.setChapters([]); }
    if (v.dataset.vpChaptersUrl) loadChapters(P, v.dataset.vpChaptersUrl);
    wireTracks();
    volume(); render(); state();
    if (v.readyState >= 1) { render(); drawMarks(); if (P.start) { P.started = true; v.currentTime = P.start; } }
  }
  function loadChapters(P, url) {
    if (!window.fetch) return;
    fetch(url).then(function (r) { return r.ok ? r.text() : ''; }).then(function (t) { if (t) P.setChapters(parseChapters(t)); }).catch(function () {});
  }

  /* ---------- the shared dialog (modal mode and galleries) ---------- */
  var M = null;
  function dialog() {
    if (M) return M;
    var d = el('dialog', 'si-vp-dialog');
    d.setAttribute('aria-label', 'Video');
    var top = el('div', 'si-vp-dtop'), count = el('span', 'si-vp-count'), close = btn('si-vp-close', 'Close (Esc)', 'close');
    top.appendChild(count); top.appendChild(close);
    var stage = el('div', 'si-vp-dstage'), prev = btn('si-vp-prev', 'Previous video', 'prev'), next = btn('si-vp-next', 'Next video', 'next');
    var holder = el('div', 'si-vp-dplayer'), v = document.createElement('video');
    v.preload = 'metadata'; holder.appendChild(v);
    stage.appendChild(prev); stage.appendChild(holder); stage.appendChild(next);
    var cap = el('div', 'si-vp-dcap'), title = el('p', 'si-vp-dtitle'), text = el('div', 'si-vp-dtext');
    cap.appendChild(title); cap.appendChild(text);
    d.appendChild(top); d.appendChild(stage); d.appendChild(cap);
    document.body.appendChild(d);
    M = { d: d, items: [], i: 0, count: count, title: title, text: text, prev: prev, next: next, opener: null };
    M.player = new Player(v, { nav: function (dir) { go(dir); } });
    close.addEventListener('click', function () { d.close(); });
    prev.addEventListener('click', function () { go(-1); });
    next.addEventListener('click', function () { go(1); });
    d.addEventListener('click', function (e) { if (e.target === d) d.close(); }); /* the backdrop */
    d.addEventListener('close', function () {
      M.player.v.pause();
      if (document.pictureInPictureElement === M.player.v) document.exitPictureInPicture().catch(function () {});
      document.documentElement.removeAttribute('data-si-vp-open');
      if (M.opener && M.opener.focus) M.opener.focus({ preventScroll: true });
    });
    return M;
  }
  function show(i, autoplay) {
    var n = M.items.length;
    M.i = (i + n) % n;
    var it = M.items[M.i];
    M.player.load(it);
    M.d.setAttribute('aria-label', it.title || 'Video');
    M.count.textContent = n > 1 ? (M.i + 1) + ' / ' + n : '';
    M.prev.hidden = M.next.hidden = n < 2;
    M.title.textContent = it.title || ''; M.title.hidden = !it.title;
    M.text.innerHTML = it.caption || ''; M.text.hidden = !it.caption;
    if (autoplay) M.player.play();
  }
  function go(dir) { show(M.i + dir, !M.player.v.paused); M.player.box.focus({ preventScroll: true }); }
  function open(items, i, opener) {
    dialog();
    M.items = items; M.opener = opener;
    show(i, true);
    if (!M.d.open) { if (M.d.showModal) M.d.showModal(); else M.d.setAttribute('open', ''); }
    document.documentElement.setAttribute('data-si-vp-open', '');
    M.player.box.focus({ preventScroll: true });
  }

  /* ---------- finding players ---------- */
  var groups = {};
  function register(it, opener) {
    var g = groups[it.group || ''] || (groups[it.group || ''] = []);
    g.push(it);
    var fire = function (e) { if (e) e.preventDefault(); open(g, g.indexOf(it), opener); };
    opener.addEventListener('click', fire);
    if (!/^(BUTTON|A)$/.test(opener.tagName)) {
      opener.tabIndex = 0; opener.setAttribute('role', 'button');
      opener.setAttribute('aria-label', 'Play video: ' + (it.title || 'video'));
      opener.addEventListener('keydown', function (e) { if (e.key === 'Enter' || e.key === ' ') fire(e); });
    }
  }
  function tracksOf(v) {
    return Array.prototype.map.call(v.querySelectorAll('track'), function (t) {
      return { src: t.getAttribute('src'), kind: t.getAttribute('kind'), srclang: t.getAttribute('srclang'), label: t.getAttribute('label'), default: t.hasAttribute('default') };
    });
  }
  function siFigure(fig) {
    var v = fig.querySelector('video.si-vp-video'), mode = fig.dataset.vpMode || 'inline', hud = fig.dataset.vpHud || 'on';
    if (!v) return;
    if (mode === 'modal') {
      var cap = fig.querySelector('.si-vp-cap'), img = fig.querySelector('.si-vp-poster'), chs = [];
      try { chs = JSON.parse(v.dataset.vpChapters || '[]'); } catch (e) {}
      register({
        src: v.getAttribute('src') || v.dataset.vpSrc, poster: img ? img.getAttribute('src') : '', title: v.dataset.vpTitle || '',
        caption: cap ? cap.innerHTML : '', tracks: tracksOf(v), chapters: chs, chaptersURL: v.dataset.vpChaptersUrl || '', start: +(v.dataset.vpStart || 0),
        loop: v.loop, group: fig.dataset.vpGroup || ''
      }, fig.querySelector('.si-vp-open'));
      return;
    }
    if (hud === 'on') new Player(v, { title: v.dataset.vpTitle });
    else if (hud === 'off') v.addEventListener('click', function () { if (v.paused) play(v); else v.pause(); });
    else if (!v.getAttribute('src') && v.dataset.vpSrc) v.src = v.dataset.vpSrc;
  }
  function dropIn(e) {
    var d = e.dataset, img = e.querySelector('img');
    var heading = e.querySelector('h1,h2,h3,h4,h5,h6') || e.querySelector('figcaption');
    var para = e.querySelector('figcaption p, p');
    var it = {
      src: d.video, poster: d.poster || (img && (img.currentSrc || img.src)) || '',
      title: d.title || (heading && heading.textContent.trim()) || (img && img.alt) || '',
      caption: d.caption != null ? d.caption.replace(/[<>&]/g, function (c) { return { '<': '&lt;', '>': '&gt;', '&': '&amp;' }[c]; }) : (para && para.innerHTML) || '',
      tracks: d.captions ? [{ src: d.captions, kind: 'captions', srclang: d.captionsLang || document.documentElement.lang || 'en', label: 'Captions' }] : [],
      chaptersURL: d.chapters || '', start: d.start ? clock(d.start) : 0, group: d.group || '', loop: d.loop != null
    };
    e.classList.add('si-vp-card');
    if ((d.mode || 'modal') !== 'inline') { register(it, e); return; }
    /* inline: the card's picture becomes the player on the first click */
    var spot = img && img.parentNode !== e ? img.parentNode : (img || e);
    var start = function (ev) {
      if (ev) ev.preventDefault();
      e.removeEventListener('click', start);
      var v = document.createElement('video');
      v.preload = 'metadata'; if (it.poster) v.poster = it.poster; v.loop = it.loop;
      it.tracks.forEach(function (t) { var tr = document.createElement('track'); tr.kind = t.kind; tr.src = t.src; tr.srclang = t.srclang; tr.label = t.label; v.appendChild(tr); });
      v.src = it.src;
      var holder = el('div', 'si-vp si-vp-inline');
      if (spot === e) { e.innerHTML = ''; e.appendChild(holder); } else spot.replaceWith(holder);
      holder.appendChild(v);
      var P = new Player(v, { title: it.title });
      P.start = it.start;
      if (it.chaptersURL) loadChapters(P, it.chaptersURL);
      e.removeAttribute('role'); e.removeAttribute('tabindex');
      P.play(); P.box.focus({ preventScroll: true });
    };
    e.addEventListener('click', start);
    if (!/^(BUTTON|A)$/.test(e.tagName)) {
      e.tabIndex = 0; e.setAttribute('role', 'button'); e.setAttribute('aria-label', 'Play video: ' + (it.title || 'video'));
      e.addEventListener('keydown', function k(ev) { if ((ev.key === 'Enter' || ev.key === ' ') && e.getAttribute('role')) { e.removeEventListener('keydown', k); start(ev); } });
    }
  }
  function init(root) {
    root = root || document;
    root.querySelectorAll('.si-vp[data-vp-mode]').forEach(function (f) { if (!f._siVp) { f._siVp = 1; siFigure(f); } });
    root.querySelectorAll('video[data-si-player]').forEach(function (v) { if (!v._siVp) new Player(v, {}); });
    root.querySelectorAll('[data-video]').forEach(function (e) { if (e.dataset.video && !e._siVp && e.tagName !== 'VIDEO') { e._siVp = 1; dropIn(e); } });
  }
  window.siPlayer = { init: init, open: function (items, i) { open(items, i || 0, document.activeElement); }, parseChapters: parseChapters };
  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', function () { init(); });
  else init();
})();
