---
title: Decks
description: Talks that fill any screen, written in Markdown: layouts, build steps, animated canvases, a teleprompter, a remote on your phone, live polls and questions from the room.
---

# Decks

A talk is a Markdown file cut into slides by `---` lines. Every slide reflows
to the screen it is on, from a phone to a 4K projector, and `si` gives you
everything you need on the day: a teleprompter, a remote on your phone, and
the room voting on theirs.

```bash
si new talk --type deck       # talk.md, a starter deck
si play talk.md --open        # present it, live reload: p opens the teleprompter
si build talk.md -o html/     # one self-contained HTML file, ready to send
```

<!-- embed: ../live/showcase.md 1-4 -->

## Write it in Markdown

```markdown
---
title: The Thursday Promise
---

# The Thursday Promise

How one order crosses five teams and still arrives on time.

---

## One promise, ==five hand-offs==

- The shop promises **Thursday** the moment the customer pays
- Payments, warehouse and courier each own one hand-off

Notes: (1 min)
Every order you place carries a promise: Thursday.
```

## Layouts

One line on a slide picks its layout; on a phone or a portrait screen the
columns stack by themselves.

<div class="dk-grid">
<figure><img src="img/decks/two-col.jpg" alt="Two columns: before and after" loading="lazy"><figcaption><code>layout: two-col</code></figcaption></figure>
<figure><img src="img/decks/split.jpg" alt="Text beside code" loading="lazy"><figcaption><code>layout: split</code>, text beside code</figcaption></figure>
<figure><img src="img/decks/big-number.jpg" alt="A big number" loading="lazy"><figcaption><code>layout: big-number</code></figcaption></figure>
<figure><img src="img/decks/quote.jpg" alt="A quote" loading="lazy"><figcaption><code>layout: quote</code></figcaption></figure>
<figure><img src="img/decks/image.jpg" alt="A full-bleed picture" loading="lazy"><figcaption><code>layout: image</code>, or <code>background:</code></figcaption></figure>
<figure><img src="img/decks/cards.jpg" alt="Three cards" loading="lazy"><figcaption><code>cards: 3</code></figcaption></figure>
</div>

## Pictures that move

One SVG canvas can change state slide after slide: parts appear, fail,
recover and travel as the story goes, and `si` animates every change.

<div class="dk-grid dk-3">
<figure><img src="img/decks/canvas-1.jpg" alt="The order is placed" loading="lazy"><figcaption><code>svg: journey.svg placed</code></figcaption></figure>
<figure><img src="img/decks/canvas-2.jpg" alt="The payment is declined, then paid" loading="lazy"><figcaption><code>+declined</code>, then <code>+paid</code></figcaption></figure>
<figure><img src="img/decks/canvas-3.jpg" alt="The parcel is delivered" loading="lazy"><figcaption><code>+delivered</code></figcaption></figure>
</div>

Or draw it in text: a `stage` flow, a Mermaid diagram, a mind map on a slide.

<div class="dk-grid dk-3">
<figure><img src="img/decks/stage.jpg" alt="A stage flow with a moving dot" loading="lazy"><figcaption><code>```stage</code></figcaption></figure>
<figure><img src="img/decks/mermaid.jpg" alt="A Mermaid flowchart" loading="lazy"><figcaption><code>```mermaid</code></figcaption></figure>
<figure><img src="img/decks/mindmap.jpg" alt="A mind map on a slide" loading="lazy"><figcaption><code>```mindmap</code></figcaption></figure>
</div>

`<!-- step -->` reveals the next block on a click, and `transition:` picks
fade, slide or zoom.

## While you present

Every deck si builds answers these keys, in `si play` or opened from a file.

<div class="dk-grid">
<figure><img src="img/decks/overview.jpg" alt="The overview: every slide, one column per section" loading="lazy"><figcaption><kbd>o</kbd> <b>Overview</b>: the whole talk, one column per section; arrows move, Enter opens</figcaption></figure>
<figure><img src="img/decks/find.jpg" alt="Find: slides matching retry" loading="lazy"><figcaption><kbd>/</kbd> <b>Find</b>: type a word, or a number to jump to that slide</figcaption></figure>
<figure><img src="img/decks/draw.jpg" alt="A ring and an underline drawn on a slide" loading="lazy"><figcaption><kbd>a</kbd> <b>Draw</b>: drag, or hold <kbd>⌥ Option</kbd> and move, so one finger on the trackpad draws; <kbd>c</kbd> colour, <kbd>z</kbd> undo, <kbd>x</kbd> clear</figcaption></figure>
<figure><img src="img/decks/timer.jpg" alt="A break timer counting down" loading="lazy"><figcaption><kbd>t</kbd> <b>Break timer</b>: pick the minutes, project the countdown</figcaption></figure>
</div>

<kbd>d</kbd> light or dark · <kbd>f</kbd> full screen · <kbd>?</kbd> every key.
Any key can be changed in `si.conf` (`keys.draw: w`), and the key that draws
with the cursor too (`draw.hold: shift`).

## The teleprompter

<kbd>p</kbd> opens it in a second window: your `Notes:` script in large type,
`[CLICK]` cues where you advance, auto-scroll at your pace, mirror for
teleprompter glass, the clock for this slide and the whole talk, and the slide
now and next.

![The teleprompter with two speakers' parts](img/decks/teleprompter.jpg)

**Two presenters?** `speakers: Ada, Ravi` in front matter and `Notes[Ravi]:`
for his part: each opens a teleprompter for their own script, sees the other's
lines as cues and a banner when the turn passes, as in the picture above.

**Record your talk.** **● Record** in the teleprompter keeps your voice and
when each slide began, beside your talk. Then:

```bash
si build talk.md --narration latest   # talk.mp4: the deck, paced and voiced as you gave it
si rehearse talk.md                   # per slide: time planned, time spoken, words per minute
```

## Present from your phone

```bash
si play talk.md --remote --https
```

The laptop projects; your phone is the **clicker**, a tablet on the lectern
the **teleprompter**, and a tablet with a pen a **draw pad**. Press <kbd>q</kbd>
on the deck for their codes. [Presenting →](remote.md)

<div class="dk-grid dk-phones">
<figure><img src="img/decks/clicker.jpg" alt="The phone clicker: Prev and a big Next" loading="lazy"><figcaption>The clicker on a phone</figcaption></figure>
<figure><img src="img/decks/tablet.jpg" alt="The teleprompter on a tablet" loading="lazy"><figcaption>The teleprompter on a tablet</figcaption></figure>
</div>

## Your audience: polls and questions

```bash
si play talk.md --audience     # the room, in the room
si play talk.md --tunnel       # the room, on a video call
```

**Follow along.** Put the audience code on your first slide; everyone who
scans it sees your slides on their phone as you present, never your notes.

![A title slide with the audience QR code](img/decks/qr.jpg)

```md
<!-- qr: audience · caption=Follow along on your phone -->
```

**Live polls.** A slide asks, the phones turn the options into buttons, and
the bars grow on the projected slide. `· hidden` keeps them for your next
click, so the first votes do not sway the rest.

<div class="dk-grid dk-wide">
<figure><img src="img/decks/poll.jpg" alt="Poll results on the projected slide" loading="lazy"><figcaption>On the screen</figcaption></figure>
<figure><img src="img/decks/phone-poll.jpg" alt="The same poll on a phone" loading="lazy"><figcaption>On a phone</figcaption></figure>
</div>

```md
<!-- poll: first-habit · Follow one order end to end · Retry once · Lend people · hidden -->
```

**Word clouds.** Each phone sends a few words; the most given word is the
largest. Your teleprompter can hide any word from the screen.

<div class="dk-grid dk-wide">
<figure><img src="img/decks/cloud.jpg" alt="A word cloud on the projected slide" loading="lazy"><figcaption>On the screen</figcaption></figure>
<figure><img src="img/decks/phone-cloud.jpg" alt="The word box on a phone" loading="lazy"><figcaption>On a phone</figcaption></figure>
</div>

```md
<!-- cloud: late -->
```

**Questions.** Every phone has *Ask a question*. Questions reach only you, in
the teleprompter's **Q&A** tab; *Show on screen* puts one on the slide.

<div class="dk-grid dk-3">
<figure><img src="img/decks/phone-ask.jpg" alt="Asking a question on a phone" loading="lazy"><figcaption>Asked on a phone</figcaption></figure>
<figure><img src="img/decks/qa-tab.jpg" alt="The Q&A tab in the teleprompter" loading="lazy"><figcaption>In your Q&A tab</figcaption></figure>
<figure><img src="img/decks/qa-screen.jpg" alt="The question shown on the slide" loading="lazy"><figcaption>On the screen</figcaption></figure>
</div>

[Try it, as the room does](audience-demo.md) · [Audience →](audience.md)

## Share it

```bash
si build talk.md -o html/            # one HTML file: open it anywhere, offline
si build talk.md --pdf               # a PDF of every slide
si build talk.md --handout           # a handout for the room: slides, or =lines, =points, =notes
si build talk.md --mp4               # a video of every slide, with the real motion
si publish talk.md --to linkedin     # a document post people swipe through (a dry run first)
```

In a terminal, over SSH: `si play talk.md --terminal`
([Terminal](terminal.md)). In a window with a live preview: the
[Studio](studio.md).

<!-- si-deps: chrome ffmpeg cloudflared tailscale gh -->

## The rules

- [3. Slides and headings](../rules/03-slides-and-headings.md)
- [5. Layouts and columns](../rules/05-layouts-and-columns.md)
- [9. Build steps](../rules/09-build-steps.md)
- [10. Animated SVG canvases](../rules/10-animated-svg-canvases.md)
- [11. Speaker notes and the teleprompter](../rules/11-speaker-notes-and-the-teleprompter.md)
- [14. Presenting: keys, touch, overview, print](../rules/14-presenting-keys-touch-overview-print.md)

<style>
.dk-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(min(100%,18em),1fr));gap:1.2em;margin:1.2em 0}
.dk-grid.dk-3{grid-template-columns:repeat(auto-fit,minmax(min(100%,14em),1fr))}
.dk-grid.dk-wide{grid-template-columns:minmax(0,2.6fr) minmax(0,1fr);align-items:start}
.dk-grid.dk-phones{grid-template-columns:minmax(0,1fr) minmax(0,1.6fr);align-items:start}
@container (max-width:40em){.dk-grid.dk-wide,.dk-grid.dk-phones{grid-template-columns:1fr}}
@media (max-width:40em){.dk-grid.dk-wide,.dk-grid.dk-phones{grid-template-columns:1fr}}
.dk-grid figure{margin:0}
.dk-wide figure:last-child img,.dk-phones figure:first-child img,.dk-3 img[src*="phone-"]{max-width:16em;margin-inline:auto}
.dk-grid img{display:block;width:100%;height:auto;border:1px solid var(--si-line);border-radius:var(--si-radius);box-shadow:0 .4em 1.2em -.6em rgb(0 0 0 / .25)}
.dk-grid figcaption{font-size:.85em;color:var(--si-muted);margin-top:.5em;line-height:1.45}
kbd{font-family:var(--si-font-mono);font-size:.85em;padding:.05em .4em;border:1px solid var(--si-line);border-bottom-width:2px;border-radius:.3em;background:var(--si-surface)}
</style>
