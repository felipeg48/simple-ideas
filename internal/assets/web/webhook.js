/* simple-ideas course result events. The build sets window.SI_WEBHOOK from
 * _course.md (webhook:, webhook-format:, webhook-events: …) and puts this
 * file before course.js. Events: lesson.completed, quiz.submitted,
 * lab.step.checked, demo.watched, course.completed (schema: RULES.md §28),
 * as json, slack, teams, discord or an xAPI 1.0.3 statement. json, slack and
 * the si relay go by navigator.sendBeacon with a text/plain body (no CORS
 * preflight, leaves as the page closes); teams, discord and xapi by fetch
 * with keepalive. What cannot leave waits in localStorage and goes with the
 * next page. A notice bar tells the learner; with webhook-consent: required
 * nothing leaves before OK. No answers, no lab output, no names unless the
 * learner types one. No dependencies; works from file:// and inside SCORM. */
(function () {
  'use strict';
  var V = 'http://adlnet.gov/expapi/verbs/', A = 'http://adlnet.gov/expapi/activities/';

  /* ---------- formats: pure, so node can test them (webhook_test.go) ---------- */
  function who(ev) { return (ev.learner && ev.learner.name) || 'Learner ' + String(ev.learner && ev.learner.id || '').replace(/^l-/, '').slice(0, 4); }
  function sentence(ev) {
    var d = ev.data || {}, c = ev.course.title, l = ev.lesson ? '“' + ev.lesson.title + '”' : '', w = who(ev), s;
    switch (ev.event) {
      case 'lesson.completed': s = w + ' finished ' + l + ' in ' + c + '.'; break;
      case 'quiz.submitted':
        s = w + ' ' + (d.passMark ? (d.pass ? 'passed' : 'did not pass') : 'took') + ' the quiz ' + l + ' in ' + c + ': ' + d.score + ' % (attempt ' + d.attempt + ').';
        break;
      case 'lab.step.checked': s = w + ' checked step ' + d.step + ' of ' + l + ' in ' + c + ': ' + (d.pass ? 'passed' : 'failed') + '.'; break;
      case 'demo.watched': s = w + ' watched ' + d.percent + ' % of ' + l + ' in ' + c + '.'; break;
      case 'course.completed': s = w + ' completed the course ' + c + '.'; break;
      default: s = w + ': ' + ev.event + ' in ' + c + '.';
    }
    return ev.audience === 'instructor' ? s + ' (instructor build)' : s;
  }
  function statement(ev) {
    var d = ev.data || {}, verb = 'completed', result, obj;
    var act = function (id, name, type) { return { objectType: 'Activity', id: id, definition: { name: { 'en-US': name }, type: type } }; };
    var course = act(ev.course.url, ev.course.title, A + 'course');
    var lesson = function (type) { return act(ev.lesson.url, ev.lesson.title, type); };
    switch (ev.event) {
      case 'course.completed': obj = course; result = { completion: true }; break;
      case 'quiz.submitted':
        verb = d.pass ? 'passed' : 'failed'; obj = lesson(A + 'assessment');
        result = { score: { scaled: d.score / 100, raw: d.score, min: 0, max: 100 }, success: !!d.pass, completion: true };
        break;
      case 'lab.step.checked':
        verb = 'answered'; obj = act(ev.lesson.url + '#check-' + d.step, d.title || 'Check ' + d.step, A + 'interaction');
        result = { success: !!d.pass };
        break;
      case 'demo.watched':
        verb = 'experienced'; obj = lesson('https://w3id.org/xapi/video/activity-type/video');
        result = { completion: d.percent >= 90, extensions: { 'https://w3id.org/xapi/video/extensions/progress': d.percent / 100 } };
        break;
      default: obj = lesson(A + 'lesson'); result = { completion: true };
    }
    var actor = { objectType: 'Agent', account: { homePage: ev.course.url, name: ev.learner.id } };
    if (ev.learner.name) actor.name = ev.learner.name;
    var ctx = { platform: 'si ' + ev.si };
    if (ev.event !== 'course.completed') ctx.contextActivities = { parent: [course] };
    return { id: ev.id, actor: actor, verb: { id: V + verb, display: { 'en-US': verb } }, object: obj, result: result, context: ctx, timestamp: ev.time };
  }
  // format is the body of an event in a webhook format.
  function format(ev, f) {
    var t;
    switch (f) {
      case 'slack': return JSON.stringify({ text: sentence(ev) });
      case 'discord':
        t = sentence(ev);
        return JSON.stringify({ content: t.length > 2000 ? t.slice(0, 1999) + '…' : t });
      case 'teams':
        return JSON.stringify({ type: 'message', attachments: [{ contentType: 'application/vnd.microsoft.card.adaptive',
          content: { $schema: 'http://adaptivecards.io/schemas/adaptive-card.json', type: 'AdaptiveCard', version: '1.4',
            body: [{ type: 'TextBlock', text: sentence(ev), wrap: true }] } }] });
      case 'xapi': return JSON.stringify(statement(ev));
      default: return JSON.stringify(ev);
    }
  }
  window.siWebhook = { format: format, sentence: sentence, statement: statement };

  var C = window.SI_WEBHOOK;
  if (!C || typeof document === 'undefined') return;

  /* ---------- the learner, the page ---------- */
  var get = function (k) { try { return localStorage.getItem(k); } catch (e) { return null; } };
  var put = function (k, v) { try { localStorage.setItem(k, v); } catch (e) {} };
  function hex(n) {
    var a = new Uint8Array(n), s = '';
    try { crypto.getRandomValues(a); } catch (e) { for (var i = 0; i < n; i++) a[i] = Math.random() * 256; }
    for (var j = 0; j < n; j++) s += ('0' + a[j].toString(16)).slice(-2);
    return s;
  }
  function uuid() { var h = hex(16); return h.slice(0, 8) + '-' + h.slice(8, 12) + '-4' + h.slice(13, 16) + '-' + '89ab'[parseInt(h[16], 16) & 3] + h.slice(17, 20) + '-' + h.slice(20); }
  var learner = get('si-learner');
  if (!/^l-[0-9a-f]{12}$/.test(learner || '')) { learner = 'l-' + hex(6); put('si-learner', learner); }
  var body = document.body, page = body.getAttribute('data-si-page') || '';
  var okKey = 'si-webhook-ok:' + C.course.id, doneKey = 'si-webhook-done:' + C.course.id, QK = 'si-webhook-queue';
  function abs(href) { try { return new URL(href, location.href).href.split('#')[0]; } catch (e) { return href; } }
  var brand = document.querySelector('.si-course-brand');
  var home = brand ? abs(brand.getAttribute('href')) : abs('index.html');
  var cur = document.querySelector('.si-course-mods a[aria-current="page"]');
  var view = (document.querySelector('.si-course') || body).getAttribute('data-view');

  function make(type, data) {
    var ev = { version: 1, event: type, id: uuid(), time: new Date().toISOString(), si: C.si || '', audience: C.audience || 'learner',
      course: { id: C.course.id, title: C.course.title, url: home } };
    if (type !== 'course.completed' && view === 'lesson') {
      var h = document.querySelector('.si-course-lesson .si-doc-title');
      ev.lesson = { id: page, title: (h ? h.textContent : document.title).trim(), kind: cur ? cur.getAttribute('data-kind') || '' : '', url: abs(location.href) };
    }
    ev.learner = { id: learner };
    var name = get('si-learner-name');
    if (C.learner === 'ask' && name) ev.learner.name = name;
    ev.data = data || {};
    if (C.token) ev.token = C.token;
    return ev;
  }

  /* ---------- sending, and the queue ---------- */
  function allowed() { return C.consent !== 'required' || get(okKey) === 'yes'; }
  function load() { try { var q = JSON.parse(get(QK) || '[]'); return Array.isArray(q) ? q : []; } catch (e) { return []; } }
  function store(q) { put(QK, JSON.stringify(q.slice(-100))); } // ponytail: the newest 100; a long offline stretch loses the oldest
  function later(it) { var q = load(); q.push(it); store(q); }
  function beacon(u, b) { try { return navigator.sendBeacon(u, new Blob([b], { type: 'text/plain;charset=UTF-8' })); } catch (e) { return false; } }
  function deliver(it) { // true when it left (a failed fetch queues it again)
    if (navigator.onLine === false) return false;
    if (it.to.charAt(0) === '/') return beacon(it.to + '?page=' + encodeURIComponent(it.page || location.pathname), it.b);
    if (it.f === 'json' || it.f === 'slack') return beacon(it.to, it.b);
    var h = { 'Content-Type': 'application/json' };
    if (it.f === 'xapi') h['X-Experience-API-Version'] = '1.0.3';
    if (it.a) h.Authorization = it.a;
    try {
      fetch(it.to, { method: 'POST', keepalive: true, headers: h, body: it.b, credentials: 'omit' })
        .then(function (r) { if (!r.ok && r.status >= 500) later(it); }, function () { later(it); });
      return true;
    } catch (e) { return false; }
  }
  function flush() {
    if (!allowed()) return;
    var q = load(), keep = [];
    store([]);
    q.forEach(function (it) { // another course's events wait for its own pages (its own consent)
      if (it.course !== C.course.id || !deliver(it)) keep.push(it);
    });
    if (keep.length) store(load().concat(keep));
  }
  function emit(type, data) {
    if ((C.events || []).indexOf(type) < 0 || get(okKey) === 'no') return;
    if (type === 'course.completed') { if (get(doneKey)) return; put(doneKey, '1'); }
    var ev = make(type, data), its = [];
    // the webhook (direct or through the relay), with its own event list when the class view asks for all
    if ((C.relay || C.url) && (!C.hookEvents || C.hookEvents.indexOf(type) >= 0))
      its.push({ to: C.relay || C.url, f: C.format || 'json', a: C.auth || '', course: C.course.id, page: location.pathname, b: format(ev, C.format) });
    // si serve --class: the live class view, always the JSON event
    if (C['class']) its.push({ to: C['class'], f: 'json', a: '', course: C.course.id, page: location.pathname, b: JSON.stringify(ev) });
    its.forEach(function (it) { if (!allowed() || !deliver(it)) later(it); });
  }
  window.siWebhook.emit = emit;
  window.addEventListener('online', flush);

  /* ---------- the notice ---------- */
  if (get(okKey) !== 'yes' && get(okKey) !== 'no') {
    var bar = document.createElement('div');
    bar.className = 'si-webhook-notice';
    bar.setAttribute('role', 'region'); bar.setAttribute('aria-label', 'Results');
    var p = document.createElement('p');
    p.textContent = C.notice || 'This course sends your progress (lessons done, quiz scores) to its organizers.';
    bar.appendChild(p);
    var input = null;
    if (C.learner === 'ask') {
      input = document.createElement('input');
      input.type = 'text'; input.className = 'si-webhook-name'; input.placeholder = 'Your name'; input.setAttribute('aria-label', 'Your name, sent with your results');
      input.value = get('si-learner-name') || '';
      bar.appendChild(input);
    }
    var ok = document.createElement('button');
    ok.type = 'button'; ok.className = 'si-webhook-ok'; ok.textContent = 'OK';
    bar.appendChild(ok);
    ok.addEventListener('click', function () {
      if (input && input.value.trim()) put('si-learner-name', input.value.trim().slice(0, 80));
      put(okKey, 'yes'); bar.remove(); flush();
    });
    if (C.consent === 'required') {
      var no = document.createElement('button');
      no.type = 'button'; no.className = 'si-webhook-no'; no.textContent = 'No thanks';
      bar.appendChild(no);
      no.addEventListener('click', function () {
        put(okKey, 'no'); bar.remove();
        store(load().filter(function (it) { return it.course !== C.course.id; }));
      });
    }
    body.appendChild(bar);
  }

  /* ---------- the events this file hears itself ---------- */
  document.addEventListener('si-quiz-score', function (e) { // doc.js, after Check my answers
    var d = e.detail || {}, k = 'si-quiz-attempts:' + C.course.id + '|' + page, n = (+get(k) || 0) + 1;
    put(k, String(n));
    emit('quiz.submitted', { score: d.score, pass: !!d.passed, passMark: d.pass || 0, attempt: n, questions: d.questions || [] });
  });
  document.addEventListener('si-lab-check', function (e) { // doc.js, after ✓ Check
    var d = e.detail || {};
    emit('lab.step.checked', { step: d.step, title: d.title || '', pass: !!d.pass, exit: d.exit, ms: d.ms });
  });
  var video = document.querySelector('.si-course-video video');
  if (video) {
    var most = 0, told = false;
    var tell = function () {
      if (told || !most) return;
      told = true;
      emit('demo.watched', { percent: most, seconds: Math.round(video.currentTime || 0) });
    };
    video.addEventListener('timeupdate', function () { if (video.duration) most = Math.max(most, Math.round(100 * video.currentTime / video.duration)); });
    video.addEventListener('ended', function () { most = 100; tell(); });
    window.addEventListener('pagehide', tell);
  }
  flush();
})();
