  /* ---------- auto-play (v): the deck presents itself. Each slide's Notes:
   * (data.say, split at its click cues by the build) is read aloud part by part, one
   * advance between two parts (the same as →), the next slide after the
   * last; a slide with no notes stays its (N min) share or autoplay.dwell.
   * The audio: the build's files (part.a), else si play's voice
   * (/_si/voice/), else the browser's own voice. Any other navigation, a
   * click on the slide or the pen takes control back (pause). ---------- */
  var say = data.say || [];
  var apBox = role ? null : document.querySelector('.si-autoplay');
  var ap = apBox ? { state: 'off', gen: 0, part: 0, driving: false, audio: null, pre: null, timer: 0 } : null;
  /* #autoplay or ?autoplay in the URL, or si play --autoplay: ready when the page opens */
  var apWanted = !!ap && (!!cfg.autoplay || /(^#|[?&])autoplay(?:&|$)/.test(location.hash) || /[?&]autoplay(?:[=&]|$)/.test(location.search));
  function apEl(sel) { return apBox.querySelector(sel); }
  function apSet(state) {
    ap.state = state;
    apBox.hidden = state === 'off';
    apBox.setAttribute('data-state', state);
    apEl('.si-autoplay-state').textContent = { ready: 'Auto-play', playing: 'Playing', paused: 'Paused', ended: 'The end' }[state] || '';
    apEl('[data-si-ap="toggle"]').textContent = state === 'playing' ? 'Pause' : state === 'paused' ? '▶ Resume' : state === 'ended' ? '▶ Again' : '▶ Start';
    var b = document.querySelector('[data-si-action="autoplay"]');
    if (b) b.setAttribute('aria-pressed', state === 'playing' ? 'true' : 'false');
    if (state !== 'playing') apCaption('');
    announce(); /* the teleprompter shows "auto-play" */
  }
  function apCaption(t) { apEl('.si-autoplay-cap').textContent = t || ''; }
  /* the advances (cues) before part j of a slide */
  function apClicks(parts, j) { var c = 0; for (var x = 1; x <= j && x < parts.length; x++) if (parts[x].k) c++; return c; }
  /* the first part spoken at build step k */
  function apPartAt(parts, k) { for (var j = 0; j < parts.length; j++) if (apClicks(parts, j) >= k) return j; return parts.length; }
  function apDwell(s) {
    var m = parseFloat(s.getAttribute('data-minutes'));
    return m > 0 ? m * 60 / (maxStep(s) + 1) : (+cfg.dwell > 0 ? +cfg.dwell : 3);
  }
  function apHash(t) { var h = 0; for (var i = 0; i < t.length; i++) h = (h * 31 + t.charCodeAt(i)) | 0; return (h >>> 0).toString(36); }
  function apURL(i, j) {
    var p = (say[i] || [])[j];
    if (!p || !p.t) return '';
    if (p.a) return p.a;
    return cfg.voice ? '/_si/voice/' + (i + 1) + '/' + j + '.wav?page=' + encodeURIComponent(location.pathname) + '&h=' + apHash(p.t) : '';
  }
  function apDrive(fn) { ap.driving = true; try { fn(); } finally { ap.driving = false; } }
  /* silence whatever speaks or waits; a stale callback finds a newer gen and stops */
  function apHalt() {
    ap.gen++;
    clearTimeout(ap.timer);
    if (ap.audio) { ap.audio.onended = ap.audio.onerror = null; try { ap.audio.pause(); } catch (e) {} }
    try { if (window.speechSynthesis) speechSynthesis.cancel(); } catch (e) {}
  }
  function apRun() {
    if (ap.state !== 'playing') return;
    clearTimeout(ap.timer);
    var g = ++ap.gen, s = slides[cur], parts = say[cur] || [];
    apEl('.si-autoplay-pos').textContent = (cur + 1) + ' / ' + n + (parts.length > 1 ? ' · ' + Math.min(ap.part + 1, parts.length) + '/' + parts.length : '');
    if (ap.part >= parts.length) { /* said all: the rest of the steps, then the next slide */
      apCaption('');
      if (cur === n - 1 && step >= maxStep(s)) { apSet('ended'); return; }
      ap.timer = setTimeout(function () {
        if (g !== ap.gen) return;
        var at = cur; apDrive(forward);
        if (cur !== at) ap.part = 0;
        apRun();
      }, (parts.length ? 0.7 : apDwell(s)) * 1000);
      return;
    }
    if (step < apClicks(parts, ap.part)) { /* the click cue before this part */
      var at = cur; apDrive(forward);
      if (cur !== at) { ap.part = 0; apRun(); return; }
    }
    var p = parts[ap.part];
    if (!p.t) { ap.part++; apRun(); return; }
    apSpeak(p, g, function () { if (g === ap.gen) { ap.part++; apRun(); } });
  }
  function apSpeak(p, g, done) {
    apCaption(p.t);
    var url = apURL(cur, ap.part);
    if (!url) { apBrowser(p, g, done); return; }
    var a = ap.audio || (ap.audio = new Audio());
    a.onended = done;
    a.onerror = function () { if (g === ap.gen) apBrowser(p, g, done); }; /* no audio for it (404): the browser's voice */
    a.src = url;
    var pr = a.play();
    if (pr && pr.catch) pr.catch(function (e) { if (g === ap.gen && e && e.name === 'NotAllowedError') { apHalt(); apSet('ready'); } });
    /* the next part, so it is voiced (si play) and loaded while this one plays */
    var nx = apURL(cur, ap.part + 1) || apURL(cur + 1, 0);
    if (nx) { if (!ap.pre) { ap.pre = new Audio(); ap.pre.preload = 'auto'; } if (ap.pre.src.indexOf(nx) < 0) ap.pre.src = nx; }
  }
  /* the browser's own voice (Web Speech); a guard moves on where no voice speaks (no voices, headless) */
  function apBrowser(p, g, done) {
    apEl('.si-autoplay-voice').textContent = 'browser voice';
    var words = p.t.split(/\s+/).length;
    var mute = !window.speechSynthesis || !speechSynthesis.getVoices().length; /* no voice here: the caption stays its reading time */
    ap.timer = setTimeout(done, (mute ? words / 2.5 + 1 : words / 1.8 + 4) * 1000);
    try {
      if (!window.speechSynthesis || !window.SpeechSynthesisUtterance) return;
      var u = new SpeechSynthesisUtterance(p.t);
      u.lang = root.getAttribute('lang') || 'en';
      u.onend = function () { if (g === ap.gen) { clearTimeout(ap.timer); done(); } };
      speechSynthesis.speak(u);
    } catch (e) {}
  }
  function apStart() {
    if (!ap || !feat.autoplay) return;
    /* sound needs a gesture first: until then the bar waits with Start */
    if (navigator.userActivation && !navigator.userActivation.hasBeenActive) { apSet('ready'); return; }
    if (deck.classList.contains('si-overview')) overview(false);
    if (drawMode) toggleDraw(false);
    if (ap.state === 'ended') apDrive(function () { go(0, 0); });
    apHalt();
    ap.part = apPartAt(say[cur] || [], step);
    apSet('playing');
    apRun();
  }
  function apPause() { apHalt(); apSet('paused'); }
  function apStop() { if (ap) { apHalt(); apSet('off'); } }
  function apToggle() { if (!ap) return; if (ap.state === 'playing') apPause(); else apStart(); }
  /* someone else moved the deck (a key, a button, the teleprompter, the pen): auto-play lets go */
  function apTouch() { if (ap && ap.state === 'playing' && !ap.driving) apPause(); }
  if (ap) {
    apBox.addEventListener('click', function (e) {
      var b = e.target.closest('[data-si-ap]'); if (!b) return;
      var a = b.getAttribute('data-si-ap');
      if (a === 'toggle') apToggle();
      else if (a === 'stop') apStop();
      else if (a === 'cc') {
        var on = b.getAttribute('aria-pressed') !== 'true';
        b.setAttribute('aria-pressed', on ? 'true' : 'false');
        apBox.setAttribute('data-cc', on ? 'on' : 'off'); save('si-ap-cc', on ? 'on' : 'off');
      }
      b.blur();
    });
    if (load('si-ap-cc') === 'off') { apBox.setAttribute('data-cc', 'off'); apEl('[data-si-ap="cc"]').setAttribute('aria-pressed', 'false'); }
    deck.addEventListener('pointerdown', apTouch);
    if (apWanted) {
      /* the first gesture anywhere starts it (browsers let a page speak only after one) */
      var apFirst = function (e) {
        if (ap.state !== 'ready' || (e.target.closest && e.target.closest('.si-autoplay,.si-bar'))) return;
        document.removeEventListener('pointerdown', apFirst, true); document.removeEventListener('keydown', apFirst, true);
        setTimeout(apStart, 0);
      };
      document.addEventListener('pointerdown', apFirst, true); document.addEventListener('keydown', apFirst, true);
      setTimeout(apStart, 0);
    }
  }
  if (!feat.autoplay || !ap) { var _ba = document.querySelector('[data-si-action="autoplay"]'); if (_ba) _ba.style.display = 'none'; }
