---
title: Try it, as the room does
description: What a phone in the audience shows during a talk given with si play --audience: the slide, a poll, a word cloud and a question box. A demo that stays on this page.
---

# Try it, as the room does

This is what a phone in the room shows while you present with
`si play --audience`: the current slide, a poll, a box for a few words and
*Ask a question*. Here it only plays on this page; at a talk every vote and
word reaches the projected slide.

<div class="si-try">

<p class="si-try-slide"><b>Which habit would your team start with?</b></p>

<figure class="si-poll" data-poll="try" data-live>
<ol class="si-poll-opts">
<li class="si-poll-opt" data-i="0" style="--_share:0" role="button" tabindex="0"><span class="si-poll-label">Follow one order end to end</span><span class="si-poll-n">0</span><span class="si-poll-bar" aria-hidden="true"></span></li>
<li class="si-poll-opt" data-i="1" style="--_share:0" role="button" tabindex="0"><span class="si-poll-label">Retry once, then tell the customer</span><span class="si-poll-n">0</span><span class="si-poll-bar" aria-hidden="true"></span></li>
<li class="si-poll-opt" data-i="2" style="--_share:0" role="button" tabindex="0"><span class="si-poll-label">Lend people before adding systems</span><span class="si-poll-n">0</span><span class="si-poll-bar" aria-hidden="true"></span></li>
</ol>
<figcaption class="si-poll-total"><span data-poll-total>0</span> votes</figcaption>
</figure>

<p class="si-try-slide"><b>In one word, what makes a delivery late?</b></p>

<figure class="si-cloud" data-live>
<ul class="si-cloud-words"></ul>
<form class="si-cloud-form"><input maxlength="30" autocomplete="off" aria-label="Your answer, a few words" placeholder="A few words"><button type="submit">Send</button></form>
</figure>

<form class="si-try-ask"><label>Ask a question <textarea maxlength="500" rows="2" placeholder="Your question for the speaker"></textarea></label><button type="submit">Send</button><span class="si-try-sent" hidden>Sent to the speaker ✓ (here, nowhere)</span></form>

</div>

[← Back to Your audience](audience.md)

<style>
.si-try{max-width:26em;margin:1em auto;padding:1em 1.1em;border:1px solid var(--si-line);border-radius:1.4em;background:var(--si-bg)}
.si-try-slide{margin:.4em 0 .2em}
.si-try .si-poll-opts{list-style:none;margin:0;padding:0;display:grid;gap:.45em}
.si-try .si-poll-opt{position:relative;display:flex;justify-content:space-between;align-items:center;gap:1em;padding:.45em .8em;overflow:hidden;isolation:isolate;background:var(--si-surface-2);border:1px solid transparent;border-radius:var(--si-radius)}
.si-try .si-poll-opt[aria-pressed="true"]{border-color:var(--si-accent)}
.si-try .si-poll-bar{position:absolute;inset:0 auto 0 0;width:calc(var(--_share,0) * 1%);z-index:-1;transition:width .6s ease;background:color-mix(in srgb,var(--si-accent) 30%,transparent)}
.si-try .si-poll-n{display:inline;font-variant-numeric:tabular-nums;font-weight:700}
.si-try .si-poll-bar{display:block}
.si-try .si-poll:not([data-voted]) :is(.si-poll-n,.si-poll-bar){visibility:hidden}
.si-try .si-poll-total{font-size:.8em;color:var(--si-muted);margin-top:.4em}
.si-try .si-cloud-words{list-style:none;margin:0;padding:.4em 0;display:flex;flex-wrap:wrap;justify-content:center;align-items:baseline;gap:.15em .7em}
.si-try .si-cloud-word{font-size:calc(1em * (.8 + var(--_w,1) * .45));line-height:1.1;font-weight:calc(400 + var(--_w,1) * 60)}
.si-try .si-cloud-word:nth-child(3n+1){color:var(--si-accent)}
.si-try .si-cloud-word:nth-child(3n+2){color:var(--si-accent-2,var(--si-accent))}
.si-try .si-cloud-form{display:flex;gap:.4em;margin-top:.5em}
.si-try .si-cloud-form input{flex:1;min-width:0;font:inherit;padding:.4em .6em;border:1px solid var(--si-line);border-radius:var(--si-radius);background:var(--si-bg);color:var(--si-fg)}
.si-try .si-cloud-form button{font:inherit;padding:.4em .9em;border:0;border-radius:var(--si-radius);background:var(--si-accent);color:var(--si-bg)}
.si-try .si-poll-opt{cursor:pointer}
.si-try-ask{display:grid;gap:.4em;margin-top:1em}
.si-try-ask textarea{width:100%;font:inherit;padding:.4em .6em;border:1px solid var(--si-line);border-radius:var(--si-radius);background:var(--si-bg);color:var(--si-fg)}
.si-try-ask button{justify-self:start;font:inherit;padding:.4em .9em;border:0;border-radius:var(--si-radius);background:var(--si-accent);color:var(--si-bg)}
</style>

<script>
(function () {
  var poll = document.querySelector('.si-try .si-poll'), counts = [7, 4, 9], mine = -1;
  function paint() {
    var total = counts.reduce(function (a, b) { return a + b; }, 0);
    poll.querySelectorAll('.si-poll-opt').forEach(function (li, i) {
      li.style.setProperty('--_share', Math.round(counts[i] * 100 / total));
      li.querySelector('.si-poll-n').textContent = counts[i];
      li.setAttribute('aria-pressed', i === mine ? 'true' : 'false');
    });
    poll.querySelector('[data-poll-total]').textContent = total;
  }
  poll.querySelectorAll('.si-poll-opt').forEach(function (li, i) {
    function pick() { if (mine >= 0) counts[mine]--; mine = i; counts[i]++; poll.setAttribute('data-voted', ''); paint(); }
    li.addEventListener('click', pick);
    li.addEventListener('keydown', function (e) { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); pick(); } });
  });
  var words = { weather: ['Weather', 5], staffing: ['Staffing', 3], payments: ['Payments', 2] }, ul = document.querySelector('.si-try .si-cloud-words');
  function cloud() {
    var list = Object.keys(words).map(function (k) { return words[k]; }).sort(function (a, b) { return b[1] - a[1]; }), max = list[0][1];
    ul.textContent = '';
    list.forEach(function (w) {
      var li = document.createElement('li'); li.className = 'si-cloud-word'; li.textContent = w[0]; li.title = w[1];
      li.style.setProperty('--_w', 1 + Math.round(4 * (w[1] - 1) / Math.max(1, max - 1))); ul.appendChild(li);
    });
  }
  document.querySelector('.si-try .si-cloud-form').addEventListener('submit', function (e) {
    e.preventDefault();
    var input = this.querySelector('input'), t = input.value.trim(), k = t.toLowerCase();
    if (!t) return;
    words[k] = words[k] ? [words[k][0], words[k][1] + 1] : [t, 1];
    input.value = ''; cloud();
  });
  document.querySelector('.si-try-ask').addEventListener('submit', function (e) {
    e.preventDefault(); this.querySelector('textarea').value = ''; this.querySelector('.si-try-sent').hidden = false;
  });
  paint(); cloud();
})();
</script>
