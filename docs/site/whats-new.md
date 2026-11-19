---
title: What's new in si 3
description: si 3 finishes web demos. Decks present themselves, and demos film themselves like edited videos, with a voice, transitions, camera moves and a spotlight.
---

# What's new in si 3

si 3 brings together a month of work on one idea: **decks that present
themselves, demos that film themselves.** One Markdown script drives a web
app, a terminal or a deck, checks every step as a test, and comes out as a
narrated video you would publish.

<!-- si-video: presenting -->

## Charts and motion, finished (3.38)

A chart show whose kind changes now folds: each bar bends round the ring
into its own slice of the donut, and unrolls back into a bar the other
way. `si build talk.md --png --svg` writes every chart as a picture,
`talk-chart-1.png` for its last step and `talk-chart-1-step-2.png` for each
step, the SVG with real text. Infographic blocks take the 3D charts
(`chart=donut3d`). In an article, guide or book, `animate:` blocks and the
`steps: deal`, `focus`, `reorder` and `carousel` lists play once as you
scroll to them. A jump to another slide, from a link or the overview, plays
the transition of the slide it lands on.

```chart bar id=region title="Orders by region"
| Region | Orders |
|--------|--------|
| North  | 1,200  |
| South  | 860    |
| East   | 1,540  |
```

```chart donut id=region
| Region | Orders |
|--------|--------|
| North  | 1,200  |
| South  | 860    |
| East   | 1,540  |
```

## Charts that move (3.37)

A ```` ```chart ```` fence draws a Markdown table as bars, columns, a line,
an area, a donut, a pie, progress bars, or solid 3D bars, a 3D donut and a
3D funnel, on the theme's colours. With `steps` each number column is one
click: bars grow and shrink, arcs sweep, lines bend and the numbers count.
`race` keeps the bars in order of value, so they glide past each other.
Charts with the same `id=` in a row are one show: a click turns the bars
into a donut, and the same `id=` on the next slide carries on.

```chart bar race unit=% title="Late orders by cause"
| Cause     | Q1 | Q2 | Q3 |
|-----------|----|----|----|
| Carrier   | 34 | 22 | 12 |
| Warehouse | 18 | 25 | 9  |
| Payment   | 9  | 14 | 21 |
| Address   | 25 | 11 | 6  |
```

[The rules for charts →](rules/33-animated-charts.md)

## Motion (3.36)

Ten new slide transitions, played by the page itself with no library:
`cube`, `flip`, `iris` (it opens from where you clicked), `wipe`, `shatter`,
`depth`, `blinds`, `portal` (the camera flies into a picture on the slide,
and the picture becomes the next slide), `glitch` and `stack`. Each one
plays in reverse when you go back, and a length sets it per slide:

```markdown
<!-- transition: portal 1.4s -->
## The receipt
```

Build animations run as their step arrives: `type`, `words`, `highlight`
(the marker of a `==mark==` paints in), `count` (an odometer), `draw` (an SVG
canvas or stage diagram draws itself on), `decode` and `confetti`. A list
can arrive in focus (`steps: focus`), dealt like cards (`steps: deal`), or
shown as written and then re-sorted by its numbers (`steps: reorder`).

Since 3.36.1, six more give a slide depth with CSS 3D: `tilt` turns the slide into layers
that follow the pointer, `steps: carousel` stands a list's items in a ring
and turns one to the front per click, `explode` pulls a list apart into
plates, `orbit` circles the first item with the rest, `flap` shows a word
on split-flap tiles, and `flyin` brings words from far behind the slide.

Reduced motion, print, PDF, the overview and the teleprompter previews show
the final state.

## The control room (3.34)

`si play` prints a keyed link to a **web monitor** at `/_si/monitor`: uptime,
CPU and memory, who is watching by role (the deck, the teleprompter, the
remote teleprompter, draw pad and clicker, the audience's phones, review and
class pages), the slide with its timers, the links with their QR codes, and
the recent reloads. Under `--audience` it runs the room: close, reopen or
reset a poll and reveal a hidden poll's results; approve the questions,
put one on screen, mark it answered or hide it; hide a word from a cloud or
clear it. It answers on this machine only, with the run's key; `--remote`
offers it to a tablet as the `monitor` role. In the terminal monitor, `p`
shows the network and audience links as QR codes and `w` opens the page.

## Terminal looks (3.32)

`si play`, `si serve`, `si build` and `si check` print in colour on a
terminal, in one of four looks: **ledger** (the default), **badges**,
**checklist** or **steps**. Pick one with `--view`, `SI_VIEW` or `ui.view:` in
si.conf. In `si play`, press `m` for a live monitor: CPU, memory, shells,
links, open pages, the current slide and the recent reloads. Piped output and
`NO_COLOR=1` stay plain.

```text
$ si play talk.md --view checklist

🎬  si play 3.32.0  serving talk.md · theme studio

✅  Deck     The Thursday Promise  (26 slides)
✅  Live     live reload on  (163 files watched)

👉  Open     http://127.0.0.1:7070/talk.html

m monitor · p codes · w web monitor · o open · q quit
```

## Demos edited like a video

- **Transitions** at every scene change: `fade` by default, or `slide`,
  `wipe`, `pop`, `zoom`, `circle`, per script or per step.
- **Camera moves:** `zoom to "Agents panel"`, `pan to`, `zoom out`, and a
  region that opens wide and zooms in.
- **Spotlight** (`focus`): everything but one element dims, with a label.
- **Windows and dragging:** film a popup such as the teleprompter, and draw
  on a slide with the mouse.

```markdown
Say: The script is in large type, with the clocks and the next slide beside it.
- press p
- window "si-presenter" · pop
- spotlight #prompt · label "What you say, word for word"
```

## A voice, and decks that present themselves

- **Kokoro `af_heart`**, an open-source voice that runs on your computer, is
  si's voice: `si voice install kokoro` sets it up in one command and makes it
  the default in `~/.si/si.conf`. The computer's own voice and cloud voices
  work too.
- **Auto-play** (`v` or ▶ Auto-play): a deck reads its `Notes:` aloud and
  clicks at every `[CLICK]`. `si build --mp4 --voice af_heart` turns the same
  notes into a narrated video.

## Demos of anything on screen

- **Web apps** (`app:`), **terminals** (`terminal`, `run`, `expect output`)
  and programs in the background (`start:`, `start … as`), in one script.
- **A test as well as a video:** `si demo test` checks every step, writes a
  report and JUnit for CI; `si demo run` films the same steps.
- **One file by default** (`<script>.mp4`), the rest with `--with captions,
  subtitles, transcript, clips, report…`; crop to a region with `--region`.

## si's own videos

Every video on this site is a demo script in `docs/demos/`, tested on every
push and filmed with Kokoro on every release.

<!-- si-video: studio -->

[Web demos, in full](features/webdemos.md) · [Decks](features/decks.md) ·
[The changelog](https://github.com/felipeg48/simple-ideas/blob/main/CHANGELOG.md)
