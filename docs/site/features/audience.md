---
title: Your audience
description: The room follows your slides on their phones, votes in live polls and sends you questions, in the room or on a video call. No accounts, no service in the middle.
---

# Your audience

Your talk, in every hand in the room. People scan one code and their phone
follows your slides as you present; a poll on a slide turns into buttons and
the bars grow on the projected screen; a question typed on a phone lands on
your teleprompter, for you alone. Nothing to install, no accounts, nothing
leaves your laptop.

```bash
si play talk.md --audience              # with --remote --https too, for your own phone
```

## Follow along

Put the audience code on your first slide:

```markdown
## Join in

<!-- qr: audience · caption=Follow along on your phone -->
```

Everyone who scans it sees the current slide, step by step: slides only,
never your `Notes:` or `Points:`, and they cannot change the slide or reach
your teleprompter (their key is not yours). Press **q** on the deck and the
code fills the screen. In a built deck the code is simply left out.

## Live polls

```markdown
## Which habit would your team start with?

<!-- poll: first-habit · Follow one order end to end · Retry once · Lend people · hidden -->

Notes:
If your team could start one of these on Monday, which would it be? [CLICK] Here is the room.
```

The first word is the poll's id, then two to six options. On the phones they
become buttons; one vote per phone, changeable. On your screen the bars grow
as the votes arrive.

| Option | What it does |
|---|---|
| `· multiple` | each phone may pick several |
| `· hidden` | the bars wait for your next click; until then only the count of votes shows, so the first votes do not sway the rest |

Without `--audience` (a built deck, a printed handout, an article) the poll
is the plain list of its options: ask for hands.

## Questions from the room

Every phone has an *Ask a question* button (an optional name, 500
characters). Questions go to you only, never to the other phones: a **Q&A**
tab on the teleprompter (key `3`) lists them, each with *Show on screen* (a
band at the foot of the projected slide; Esc takes it off), *Answered* and
*Hide*.

## On a video call

```bash
si play talk.md --tunnel        # once: brew install cloudflared
```

si prints a temporary public link (`https://<words>.trycloudflare.com/a?k=…`):
paste it in the call's chat, or show the `<!-- qr: audience -->` slide, which
now carries it. People anywhere follow, vote and ask exactly like the room.
Only the audience side is on the internet, never your teleprompter or your
controls, and the link is gone when you stop `si play`.

## After the talk

When you stop `si play`, the votes and questions are saved beside your talk
in `.si/audience/<talk>-<date-time>.json`: counts and questions, no
addresses. Keep `.si/` out of git. To keep nothing, put this in `si.conf`:

```
audience.save: off
```

## The rules

- [11. Speaker notes and the teleprompter](../rules/11-speaker-notes-and-the-teleprompter.md)
- [Present from your phone](remote.md): the clicker, the teleprompter and the draw pad on your own devices
