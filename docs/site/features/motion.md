---
title: Motion
description: Pictures that change instead of slides that change, in an si deck. One SVG canvas whose parts light up, fail and recover click by click, scenes where parcels fly and tokens hop between blocks, charts whose bars race, and morph.
---

# Motion

The deck below keeps one picture on screen while the story moves through
it: the canvas changes state, a parcel splits into three, a token hops
down the line of teams, the bars race. Click it, then →.

<!-- slides: showcase/motion.md -->

## A canvas that changes state

Two slides that show the same SVG keep it in place: only its state
changes, and every CSS transition in it runs. Words after the file add or
remove states; `steps:` gives each click its own.

```markdown
## Payment: declined, then paid

<!-- svg: journey.svg +declined -->
<!-- steps: . -declined,retry,paid -->
- 21:04: the card fails a fraud check
- 21:05: a retry thirty seconds later goes through
```

The SVG says what each state shows with plain attributes
(`data-show="paid"`, `data-alert="declined"`), no script. See [§10 Animated
SVG canvases](../rules/10-animated-svg-canvases.md).

## Scenes

A ```` ```scene ```` fence names a few blocks, then each line is one click
of movement, in plain words:

````markdown
```scene
box Order €84
column Berlin, Lyon, Milan
Order €84 splits → Berlin, Lyon, Milan
parcels merge → Lyon · slow
```
````

Parcels fly, tokens hop, blocks morph, split, merge, swap, queue up, fan
out and open on their parts. See [§34
Scenes](../rules/34-scenes-objects-that-move.md).

## Charts that move

````markdown
```chart bar race unit=%
| Cause     | Q1 | Q2 | Q3 |
|-----------|----|----|----|
| Carrier   | 34 | 22 | 12 |
| Warehouse | 18 | 25 | 9  |
| Payment   | 9  | 14 | 21 |
| Address   | 25 | 11 | 6  |
```
````

Each number column is a click: bars grow, shrink and change places. See
[§33 Animated charts](../rules/33-animated-charts.md).

More: [Transitions](transitions.md) · [Build animations](animations.md) ·
[Decks](decks.md).
