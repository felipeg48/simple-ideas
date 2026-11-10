# Motion: first frame, last frame, si does the rest

Status: spec, not started. Order: 7. Requested by the founder, 2026-11-09:
"more animations: create visually one frame, then the final frame, and do the
motion".

## Why

si animates today in three ways, each tied to one kind of content:

- **Canvases** (RULES §10): one SVG whose parts change state slide by slide
  (`<!-- svg: journey.svg +paid -->`); the motion is CSS transitions written
  by hand in the SVG's `<style>`.
- **Code shows** (§25): lines glide between steps (`transition=morph`).
- **Slide transitions** (§6): fade, slide, zoom, none.

What is missing is the way designers think, as in Keynote's Magic Move: make
the first picture, make the last picture, and let the tool move everything in
between. Two parts:

1. **Morph between slides** (any content, no authoring): two slides in a row,
   the same things on both, in new places, sizes or colours; si glides them.
2. **The Motion view in the Studio** (visual authoring): draw or import a
   picture, set frame 1, set the last frame by dragging, scaling, recolouring;
   tune timing; si writes the canvas and its steps.

## 1. Morph between slides

```markdown
<!-- transition: morph -->
## Five teams

![](assets/shop.svg){#shop} ![](assets/warehouse.svg){#wh}

---

<!-- transition: morph -->
## One promise

![](assets/warehouse.svg){#wh .big}
```

- `transition: morph` on a slide (or in front matter for the whole deck):
  on arrival from the previous slide, elements that match glide from their
  old box to their new one (position, size, colour, opacity), unmatched ones
  fade out / in.
- **Matching**, in order: an explicit id (`{#wh}` on images, headings, list
  items, figures, `<!-- id: wh -->` before a block), then the same heading
  text, then the same image file, then the same text block.
- **How**: FLIP in the runtime (measure first, measure last, invert with a
  transform, play with the Web Animations API), so it stays fluid at any
  size; transitions of `color`, `background`, `opacity`; text that changes
  cross-fades. Duration and easing from tokens (`--si-morph-duration`,
  `--si-morph-ease`), per slide `<!-- transition: morph 0.8s -->`.
- Works with build steps: the morph runs on arrival; steps reveal after.
- Reduced motion: a plain fade. Print and PDF: the last frame. MP4: filmed in
  real motion (`motion.go`), like canvas morphs today.
- Canvases already morph between consecutive slides of the same SVG
  (`canvasKey`); `morph` generalises that to every element and keeps the
  canvas path for SVGs.

## 2. The Motion view (Studio, web and desktop)

For a picture that moves inside one slide, in steps.

1. **Start**: Studio › New… › Animation, or open an existing canvas SVG in
   Motion view. Start from a blank board, an SVG file, or a stage diagram.
2. **Frame 1**: place shapes (rectangles, circles, lines, arrows, text,
   images), choose colours from the theme's tokens (`var(--si-accent)` …,
   so the animation follows light/dark and the theme), name the parts.
3. **Add frame**: duplicates the last frame; drag, resize, rotate, recolour,
   hide or show parts. Frames appear as a filmstrip; any frame can be edited.
4. **Motion between frames**: per frame, duration and easing (ease, linear,
   spring, bounce), a stagger for groups, and per part "move along a path"
   (draws the path the part follows: the `data-move` idea on the roadmap).
5. **Preview**: plays frame to frame in the real renderer, light and dark,
   four sizes; a scrubber.
6. **Use**: "Insert in this slide" writes
   `<!-- svg: anim/journey.svg frame1 -->` and one `<!-- step: +frame2 -->`
   per frame (the existing canvas contract), or "Spread over slides" puts
   frame N on slide N for the slide-to-slide canvas morph.

**What it writes** is a normal si canvas, readable and editable by hand:

```svg
<svg viewBox="0 0 800 450" data-states="frame1 frame2 frame3">
  <style>
    .part{transition:transform .6s var(--si-ease,ease),fill .6s,opacity .6s}
    [data-state~="frame2"] #token{transform:translate(310px,0)}
    [data-state~="frame3"] #token{transform:translate(620px,40px) scale(1.2);fill:var(--si-ok)}
  </style>
  <rect id="token" class="part" … fill="var(--si-accent)"/>
</svg>
```

Coordinates are in the SVG's own `viewBox` units, so the picture scales with
the slide (rule #1); the view never writes screen pixels.

## How it fits si

- Runtime: `deck/02-navigation.js` gains the morph transition (a matcher and
  FLIP player; `data-si-tx="morph"`); render emits ids for `{#id}` and
  `<!-- id: -->`.
- Studio: `web/studio/src/motion-editor.tsx` (an SVG board with handles; the
  Map and Window views' patterns for undo and text sync);
  `internal/studio/motion.go`: read a canvas into frames (states and their
  CSS) and write it back, keeping unknown SVG and CSS verbatim.
- Model: `internal/motion` (stdlib): the frame model, the SVG/CSS writer and
  reader, used by the Studio, the MCP tool and tests.
- MCP / skill: `si_motion` (write or update a canvas from frames given as
  data) and a skill reference "animating with si" (when to morph, when to
  use a canvas, how many frames, timing that reads well).

## Rules check

1. Fluid: FLIP uses measured boxes at the current size; canvases use viewBox
   units; no fixed canvas.
2. Self-contained: CSS and the existing runtime, no animation library.
3. RULES.md: `transition: morph`, `{#id}` / `<!-- id: -->`, the Motion view,
   with examples; the showcase uses a morph as content.
6. Durations and easings are tokens; mechanics in layout.css.
8. Stdlib; the Studio's existing bundle.

## Tests

- Unit: the matcher (ids, headings, images, text), the frame model round trip
  (an existing hand-written canvas survives open → edit one frame → save).
- e2e: two morph slides → the matched element's box at the start of the
  animation equals its old box and at the end its new one (measured), at four
  sizes; reduced motion gives a fade; the Motion view makes a 3-frame canvas,
  inserts it, and the steps play.
- Visual: screenshots mid-morph, light and dark.

## Phases

1. `transition: morph` between slides (the most value per line of code).
2. The frame model and canvas writer (`internal/motion`) + MCP `si_motion`.
3. The Motion view: board, frames, timing, preview, insert.
4. Paths (`data-move`), springs, stagger; counters that tick between frames.

## Open questions

- Morph across non-consecutive slides (going back two)? Proposed: only
  consecutive, like canvases.
- Lottie import? Proposed: no (a runtime and a format to carry); SVG covers it.
