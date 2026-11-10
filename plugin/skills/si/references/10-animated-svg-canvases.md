# 10. Animated SVG canvases

One SVG file, many slides: the picture stays, parts appear, fail and recover.

```markdown
## The journey on one screen

<!-- svg: assets/journey.svg placed -->

---

## Payment: declined, then paid

<!-- svg: assets/journey.svg +declined -->
<!-- steps: . -declined,retry,paid -->
- 21:04: the card fails a fraud check
- 21:05: an automatic retry goes through
```

**The directive.** `<!-- svg: PATH words... -->` inlines the SVG (so it can
use the deck's CSS tokens) and sets its state: the root `<svg>` gets
`class="si-canvas"` and `data-state="words"`.

**State words carry across slides.** Plain words replace the state. When all
words start with `+` or `-`, they change the state the previous slide with
the same file ended in (after its steps): `+declined` above means "placed,
plus declined". Steps change it further (see [Build steps](#9-build-steps)).

**The animation.** On two consecutive slides with the same file, the incoming
copy starts in the outgoing state and switches to its own, so every CSS
transition in the SVG runs: parts fade in, move, change colour. The slide
itself does not fade; only the words around the canvas do.

### Authoring contract

1. Give the root `<svg>` a `viewBox` (or `width` and `height`, from which `si`
   derives one) and a class of your own, for example `class="journey"`.
2. Colour with the deck tokens and a fallback, so the file also looks right on
   its own: `fill: var(--si-surface, #fff)`.
3. Show and hide parts with **helper attributes** (no CSS needed):

   | Attribute | When the state contains the word |
   |---|---|
   | `data-show="paid retry"` | visible (hidden otherwise); fades |
   | `data-hide="delivered"` | hidden; fades. `data-hide` wins over `data-show` |
   | `data-alert="declined"` | shapes inside get the `--si-crit` stroke and pulse; a `text` with it turns `--si-crit` |
   | `data-ok="paid"` | shapes inside get the `--si-ok` stroke; a `text` with it turns `--si-ok` |
   | `data-flow="shipped"` | the path gets marching dashes |
   | `data-packets="paid"` | dots travel along the path (`data-packets-count`, `data-packets-dur` seconds, `data-packets-r`) |
   | `data-grow="loaded"` | the element scales up from its base (0 → full height, grown from the bottom) — for bars |
   | `data-move="sent"` | the element travels to the end of its own `offset-path` (set the path in your SVG CSS) |
   | `data-count="s1:0 s2:4200"` | the element's text ticks to the number of the active state word (last match wins) |

   Each attribute takes several words (any of them matches).
4. For anything else write CSS **inside the SVG** with the state selector and
   a transition, and it animates between slides and steps:

   ```css
   .journey .token { transition: transform 1s ease; }
   .journey[data-state~="paid"] .token { transform: translateX(235px); }
   .journey[data-state~="packed"] .token { transform: translateX(470px); }
   ```

   Same specificity, later rule wins: list stages in order so the furthest
   one applies when several words are present.
5. No need to scope animations to the current slide; animations pause on
   hidden slides.
6. Never write a less-than sign inside the SVG `<style>` (not even in a
   comment): inline SVG styles are parsed as markup in HTML.
7. Ids are prefixed per copy (`c3-box`), and `url(#id)` / `href="#id"`
   references with them. `<image href>` files are embedded.
8. With reduced motion, transitions are instant and packets are hidden.

`si new` and `examples/assets/journey.svg` are a complete worked example. To
build one from scratch, run `si new-canvas <name>` for a working, commented
starter canvas plus a deck that drives it, and follow the step-by-step recipe in
`docs/authoring-canvases.md`.

**Frames, drawn in the Studio.** The Studio's Motion view (open an `.svg`,
pick Motion) builds a canvas from frames: draw frame 1, add a frame, drag,
resize, turn, recolour or hide parts, and set how long each move takes. It
writes a normal canvas you can still edit by hand: parts carry
`data-part="name"`, and one block in the `<style>` between
`/* si motion: frame1 frame2 frame3 */` and `/* si motion end */` holds a rule
per frame (coordinates in `viewBox` units, colours as tokens):

```css
/* si motion: frame1 frame2 frame3 */
[data-part]{transform-box:fill-box;transform-origin:center}
[data-state~="frame2"] [data-part="token"]{transform:translate(310px,0px)}
[data-state~="frame3"] *{transition-duration:1.2s;transition-timing-function:cubic-bezier(.3,1.5,.5,1)}
[data-state~="frame3"] [data-part="token"]{transform:translate(620px,40px) scale(1.2);fill:var(--si-ok)}
/* si motion end */
```

Frame 1 stays on; each click adds the next frame and takes the one before off:

```markdown
<!-- svg: anim/token.svg frame1 -->
<!-- step: +frame2 -->
<!-- step: -frame2 +frame3 -->
```

A part's CSS `transform` replaces its own `transform=` attribute, so draw a
moving part without one (or wrap it in a `<g>` and name the `<g>`).
