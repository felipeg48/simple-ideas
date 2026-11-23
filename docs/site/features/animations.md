---
title: Build animations
description: Blocks that arrive with a click in an si deck. Text that types itself, numbers that roll, markers that paint, lists that focus, deal or re-sort, and six in 3D (carousel, explode, orbit, split-flap, fly-in, tilt). One comment line each.
---

# Build animations

A comment before a block makes it arrive: with its click when it is a build
step, else when the slide arrives. Click the deck, then → to see each one.

<!-- slides: showcase/animations.md -->

## Written as

```markdown
<!-- animate: type -->
> Your shoes arrive Thursday.

<!-- step -->
<!-- animate: highlight -->
We kept it for ==97.4%== of orders last quarter.

---

<!-- layout: big-number -->
## On time

<!-- animate: count -->
**12,480** orders on time in November
```

| Line | What it does |
|---|---|
| `<!-- animate: type -->` | the text types itself out with a caret |
| `<!-- animate: words -->` | words rise one by one out of a soft blur |
| `<!-- animate: highlight -->` | each `==mark==` paints in, left to right |
| `<!-- animate: count -->` | the digits roll into place like an odometer |
| `<!-- animate: decode -->` | the text arrives scrambled and decodes letter by letter |
| `<!-- animate: confetti -->` | the block pops in and confetti falls away |
| `<!-- animate: draw -->` | a canvas or a diagram draws its lines like a pen |
| `<!-- steps: focus -->` | each click brings the next item into focus, the rest blur |
| `<!-- steps: deal -->` | each item flips in like a dealt card |
| `<!-- steps: reorder -->` | one click re-sorts the list by its numbers; each item glides |

## In 3D

CSS 3D, no library, for the moments that deserve it:

```markdown
<!-- steps: carousel -->
- Shop
- Payments
- Warehouse
- Courier
- Customer

---

<!-- animate: flap -->
DELIVERED
```

| Line | What it does |
|---|---|
| `<!-- steps: carousel -->` | the items stand as cards in a ring; each click turns the next to the front |
| `<!-- animate: explode -->` | a list as stacked plates that pull apart |
| `<!-- animate: orbit -->` | the first item in the middle, the rest on a turning ring |
| `<!-- animate: flap -->` | split-flap tiles flip through the alphabet until the word lands |
| `<!-- animate: flyin -->` | the words fly in from behind the slide |
| `<!-- animate: tilt -->` | the slide becomes layers; the camera tilts and follows the pointer |

Each ends with the block exactly as written; print, PDF, reduced motion and
the overview show it at rest. In an article or a guide the same lines play
once as the block scrolls into view. Every rule: [§9 Build
steps](../rules/09-build-steps.md). More: [Transitions](transitions.md) ·
[Motion](motion.md) · [Decks](decks.md).
