---
title: Transitions
description: How a slide arrives in an si deck. Fade, slide and zoom, ten transitions the page plays itself (cube, flip, iris, wipe, depth, blinds, stack, portal, glitch, shatter) and morph, where what two slides share glides into place.
---

# Transitions

One line before a slide's heading sets how it arrives. Click the deck
below, then step through it with → and ←: every slide arrives in its own
way, and going back plays each motion in reverse.

<!-- slides: showcase/transitions.md -->

## Written as

```markdown
<!-- transition: cube -->
<!-- layout: big-number -->
## 21:03

Order placed: running shoes, size 42, Berlin

---

<!-- transition: portal 1.4s -->
<!-- layout: center -->
## Delivered

Two days, door to door.
```

`transition:` in the front matter sets the deck's default (`fade`). A length
after the name (`cube 1.2s`) sets it for one slide.

| Name | What the room sees |
|---|---|
| `fade`, `slide`, `zoom`, `none` | the plain ones, in CSS |
| `cube` | the slide turns on the edge of a cube |
| `flip` | the slide flips over like a card; the next is on its back |
| `iris` | a circle opens from where you clicked |
| `wipe` | a band of light sweeps across and leaves the next slide |
| `depth` | the slide sinks back and blurs; the next comes from the camera |
| `blinds` | vertical strips turn one after another |
| `stack` | the next slide slides over this one like a card on a pile |
| `portal` | the camera flies into a block on the slide (`<!-- id: door -->`), and it becomes the next slide |
| `glitch` | a short digital tear, then the next slide settles |
| `shatter` | the slide breaks into tiles that fall away |
| `morph` | what the two slides share (a heading, a named block, a picture) glides to its new place and size |

## Morph

With `transition: morph`, the heading and anything named the same on both
slides move instead of fading: the last two slides above turn a list into
one sentence.

```markdown
<!-- transition: morph -->
## Five teams

- shop, payments, warehouse, courier, support

<!-- id: promise -->
Two days, door to door.

---

<!-- transition: morph -->
<!-- layout: center -->
## Five teams

<!-- id: promise -->
**One promise: two days, door to door.**
```

Reduced motion, print, PDF and the overview show each slide as it ends.
Every rule, with the details: [§6 Transitions](../rules/06-transitions.md).
More: [Build animations](animations.md) · [Motion](motion.md) ·
[Decks](decks.md).
