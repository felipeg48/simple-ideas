# 6. Transitions

`<!-- transition: fade|slide|zoom|none -->` sets how a slide arrives; the
front matter `transition:` sets the default (`fade`).

When two consecutive slides show **the same SVG file** with `svg:`, the slide
transition is replaced by the canvas transition: the picture stays put, its
state animates, and the words around it fade in. See
[SVG canvases](#10-animated-svg-canvases).

With `prefers-reduced-motion: reduce` every transition and animation is off.
