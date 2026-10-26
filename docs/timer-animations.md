# Timer animations

The break timer (`t`) can run a themed animation layer behind its translucent
HUD — the layering is **SLIDE │ ANIMATION │ TIMER**. Turn it on in `si.conf`:

```
timer.animation: globe        # built-in: globe | ranges | roaches
```

Three built-ins ship with si:

| name | what it is |
|---|---|
| `globe` | a rotating wireframe globe with orbiting nodes; spins faster as time runs down |
| `ranges` | range replicas distributing and rebalancing across nodes |
| `roaches` | cockroaches wandering the screen; they speed up and scatter near zero |

All three are drawn with the deck's theme tokens (so they match `crdb`, `paper`,
your custom theme, dark mode …) and pause under `prefers-reduced-motion`.

## Custom animations

Point `timer.animation.path` at a directory and `timer.animation` (or
`timer.animation.name`) at a file stem:

```
timer.animation.path: ./anim
timer.animation.name: orbit
```

si loads `./anim/orbit.js` or `./anim/orbit.svg` (by extension) and inlines it,
so the built HTML stays self-contained. The path is resolved relative to the
current directory, then the deck. A missing animation is a `si check` problem.

### The `.js` format

A `.js` animation is a single **function expression** `(canvas, api) → stop`.
si runs it when the timer starts and calls the returned `stop()` when the HUD
closes. The canvas fills the viewport (sized for the device pixel ratio).

```js
function (canvas, api) {
  var ctx = canvas.getContext('2d'), raf;
  function frame() {
    ctx.clearRect(0, 0, canvas.width, canvas.height);
    // api.tokens.accent / .fg / .bg / .accent2 / .ok / .warn / .crit / ...
    // api.fraction()  -> 0 at the start, 1 at zero (react to the countdown)
    // api.remaining() -> seconds left
    // api.reduced     -> true under prefers-reduced-motion (draw calm/static)
    raf = requestAnimationFrame(frame);
  }
  raf = requestAnimationFrame(frame);
  return function () { cancelAnimationFrame(raf); };
}
```

`api.fraction()` is the hook for "animate *with* the timer": make something
intensify, drain, or resolve as the break runs out. Read theme colours from
`api.tokens` (resolved `--si-*` values) rather than hard-coding, so the
animation re-themes for free. The built-ins in `internal/assets/web/anim/` are
the reference — `globe.js` is the smallest.

### The `.svg` format

A `.svg` animation is a self-contained SVG (CSS/SMIL animated), like an si
canvas — theme it with `var(--si-*)` tokens and it scales to fill the layer. Use
this for looping, decorative motion that doesn't need the live countdown; a `.js`
module is the choice when the animation should react to `api.fraction()`.

## Notes

- The HUD keeps its translucency over the animation; the backdrop blur is
  lightened while an animation plays so it shows through.
- Animations run only while the break timer is counting, never during the talk.
- Keep them small: the code is inlined into every deck that uses it.
