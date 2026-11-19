# 9. Build steps

Steps reveal content one press at a time. → ↓ Space PgDn go through the
steps of a slide before moving to the next slide; ← ↑ PgUp go back.

```markdown
<!-- steps -->
- revealed first
- then this
- then this

<!-- step -->
A paragraph (or any block) revealed as one step.
```

- `<!-- steps -->` before a list: each item is one step.
- `<!-- step -->` before any block: that block is one step.
- Steps can **drive an SVG canvas**: words after the colon change the state of
  the slide's canvas when the step is reached.
  - `<!-- step: paid -->` reveals the next block and adds `paid`.
  - `<!-- steps: placed paid -declined,retry . -->` gives each list item one
    token: item 1 adds `placed`, item 2 adds `paid`, item 3 removes `declined`
    and adds `retry`, item 4 changes nothing (`.`). Commas join several words
    in one token.
  - A `step:` with no block after it (the last line of the slide, or followed
    by another directive) is an invisible step that only changes the canvas.
- Going back undoes the state change. Overview and print show every step.
- The URL hash keeps the step: `#5.2` is slide 5 at step 2.

**How a list arrives.** A first word after `steps:` changes how the items
come in; canvas tokens may follow it as above.

```markdown
<!-- steps: focus -->
- Shop
- Payments
- Warehouse
- Courier

<!-- steps: deal -->
- Shop
- Payments

<!-- steps: reorder -->
- Address wrong 12%
- Stock drift 61%
- Courier no-show 23%
```

- `steps: focus`: every item is shown; each step brings the next one into
  focus and blurs and dims the rest, like a camera changing focus.
- `steps: deal`: each item flips in from its top edge like a dealt card.
- `steps: reorder`: the list is shown as written; one build step re-sorts
  it and each item glides to its new place. The new order is by the last
  number in each item, highest first (`61%`, `23%`, `12%`), or the order you
  give as item numbers counted from 1: `<!-- steps: reorder 2 3 1 -->` puts
  item 2 first, then 3, then 1. Going back puts it as written again.

**Build animations.** `<!-- animate: NAME -->` animates the next block as it
arrives: with its build step when it has one (`<!-- step -->` before or
after it), else when the slide arrives.

```markdown
<!-- animate: type -->
> Delivered. Two days, door to door.

<!-- step -->
<!-- animate: count -->
**12,480** orders on time

<!-- animate: highlight -->
Shoes at the door in ==two days==.
```

| Name | What it does |
|---|---|
| `type` | the text types itself out with a blinking caret |
| `words` | words rise one by one out of a soft blur |
| `highlight` | the marker of each `==mark==` in the block paints in, left to right |
| `count` | the digits roll into place like an odometer, each column at its own speed |
| `draw` | an `svg:` canvas or a stage diagram draws its lines on like a pen, then fills |
| `decode` | the text arrives scrambled and decodes letter by letter |
| `confetti` | the block pops in and a burst of confetti falls away (then it is removed) |

Every animation ends with the block exactly as written. Going back does
not replay it; reduced motion, print, PDF, the overview, thumbnails,
handouts, the teleprompter previews and a jump (`#5.2`) show the final
state. In an article, guide or book `animate:` has no effect (`si check`
warns): documents show the final state.
