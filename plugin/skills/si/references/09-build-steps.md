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
- `?play` before the hash plays the way there: `talk.html?play#5` opens slide 4 at its end and moves on to 5, so slide 5's transition and build animations run (`?play#5.2`: from step 1 to step 2). For a link that shows an effect.

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

**In 3D.** Six more give a slide depth, with CSS 3D only (no library), for
the moments that deserve it:

```markdown
<!-- steps: carousel -->
- Shop
- Payments
- Warehouse
- Courier
- Customer

<!-- animate: explode -->
- Database
- Queue
- API
- Browser

<!-- animate: orbit -->
- Order
- Shop
- Payments
- Courier

<!-- animate: flap -->
DELIVERED
```

| Name | What it does |
|---|---|
| `animate: tilt` | the slide's heading, the marked block and the rest become layers at different depths; the camera tilts once as the slide arrives, then follows the pointer while it is shown |
| `steps: carousel` | the list's items stand as cards in a ring; each step turns the next card to the front and highlights it |
| `animate: explode` | on a list: one plate per item, stacked and tilted, pulling apart so each name can be read (the first item is the bottom plate) |
| `animate: flap` | the text is shown as split-flap tiles; the letters flip through the alphabet until the word lands |
| `animate: flyin` | the words fly in from far behind the slide, out of a blur, one after another |
| `animate: orbit` | on a list: the first item in the middle, the rest on a tilted ring that turns, always facing you; it turns while the slide is shown and rests when you leave |

`carousel`, `explode`, `orbit` and `flap` keep their 3D look at rest, so
print, PDF, the overview and thumbnails show the ring, the plates or the
tiles as they end; `tilt` and the motion of the others never run there.

Every animation ends with the block exactly as written. Going back does
not replay it; reduced motion, print, PDF, the overview, thumbnails,
handouts, the teleprompter previews and a jump (`#5.2`) show the final
state.

**In documents.** In an article, guide or book each `animate:` block plays
once, when it is scrolled into view, and the lists `steps: focus`, `deal`,
`reorder` and `carousel` play through their items the same way (focus moves
down the list, deal turns each item over, reorder re-sorts after a moment,
carousel turns to each card). A `reorder` list is written in its new order,
so the page without its script ends there too. `tilt` belongs to a slide
and does nothing on a page; plain `<!-- steps -->` stays deck-only. Print,
reduced motion and a page without its script show every block as written.

```markdown
<!-- animate: count -->
**1,842** declines between 21:00 and 06:00

<!-- steps: deal -->
- Split every failure by its real cause
- Make the retry idempotent first
- Measure the window, then wait a little longer
```
