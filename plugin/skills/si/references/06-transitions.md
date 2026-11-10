# 6. Transitions

`<!-- transition: fade|slide|zoom|none|morph -->` sets how a slide arrives; the
front matter `transition:` sets the default (`fade`).

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
