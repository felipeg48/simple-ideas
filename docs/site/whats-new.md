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
