# 6. Transitions

`<!-- transition: NAME -->` sets how a slide arrives; the front matter
`transition:` sets the default (`fade`). `fade`, `slide`, `zoom` and `none`
are CSS; `morph` is below; the ten motion transitions are played by the
page itself:

| Name | What the room sees | Length |
|---|---|---|
| `cube` | the slide turns on the edge of a cube; the next face swings into view | 1s |
| `flip` | the slide flips over like a card; the next slide is on its back | 0.9s |
| `iris` | a circle opens from where you clicked (else the centre) and grows into the next slide | 0.9s |
| `wipe` | a soft band of light sweeps across and leaves the next slide behind it | 0.9s |
| `shatter` | the slide breaks into tiles that fall away, nearest the click first | 1.3s |
| `depth` | the slide sinks back and blurs; the next one arrives from in front of the camera | 1.1s |
| `blinds` | vertical strips turn one after another, each showing a strip of the next slide | 1.1s |
| `portal` | the camera flies into a picture on the slide, and that picture becomes the next slide | 1.1s |
| `glitch` | a short digital tear: colour channels split, slices jump, the next slide settles | 0.55s |
| `stack` | the next slide slides over this one like a card on a pile; this one sinks under it | 0.9s |

```markdown
<!-- transition: cube -->
## Payment authorised

---

<!-- transition: portal 1.4s -->
## The receipt
```

A length after the name (`cube 1.2s`, `iris 600ms`) sets it for that slide;
only `morph` and the motion transitions take one. A motion transition
belongs to the slide it arrives at: going back from that slide plays the
same motion in reverse (the mirror), going back to it fades. `portal` flies
into the block named with `<!-- id: name -->` on the slide before, else that
slide's first picture or diagram. They run between neighbours only; a jump
(`#7`, the overview, find) fades, and reduced motion, print, PDF, the
overview, thumbnails and handouts show each slide as it ends. Two slides
that show the same SVG keep the canvas transition, and `morph` keeps its
own rules.

**Morph.** With `transition: morph`, what the slide shares with the slide
before it glides from its old place and size to its new one (colour and
opacity too); everything else fades out and in. Two things match when they
have the same name, the same heading text, the same picture, or the same
words, in that order. Name a block with `<!-- id: name -->` on the line
before it when its words change between the slides:

```markdown
<!-- transition: morph -->
## Five teams

- shop, payments, warehouse, courier
- Nobody owns the promise from end to end

<!-- id: promise -->
Two days, door to door.

---

<!-- transition: morph 0.9s -->
<!-- layout: center -->
## Five teams

**Nobody owns the promise from end to end**

<!-- id: promise -->
Two days, **door to door**.
```

The heading and the line in bold match by their words, the last paragraph by
its name. Words keep their shape as they move (they scale by font size, not
by box); pictures, code and tables scale to their new box. `morph 0.9s` sets
the length for that slide; `--si-morph-duration` (`.7s`) and
`--si-morph-ease` set it for the deck. The morph runs between neighbours
only, forward and back (going back from a morph slide morphs too); a jump,
the overview and print show the slide as it ends. Build steps reveal after
the morph, and only what is already shown takes part. Two slides that show
the same SVG keep the canvas transition below.

When two consecutive slides show **the same SVG file** with `svg:`, the slide
transition is replaced by the canvas transition: the picture stays put, its
state animates, and the words around it fade in. See
[SVG canvases](#10-animated-svg-canvases).

With `prefers-reduced-motion: reduce` every transition and animation is off.
