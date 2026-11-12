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
