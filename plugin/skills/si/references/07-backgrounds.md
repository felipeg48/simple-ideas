# 7. Backgrounds

```markdown
<!-- background: VALUE [fit=cover|contain] [position=center|left-top|...]
                        [opacity=0..1] [blur=N] [overlay=auto|dark|light|none]
                        [poster=image] -->
```

(Write it on one line; it is wrapped here for reading.) `VALUE` is one of:

| Value | Example |
|---|---|
| a colour | `<!-- background: #101820 -->` |
| a CSS gradient | `<!-- background: linear-gradient(135deg, #1d2b64, #0b8a7a) -->` |
| an image | `<!-- background: assets/dusk.jpg opacity=0.35 blur=4 -->` |
| a video (mp4, webm, mov, m4v, ogv) | `<!-- background: assets/loop.mp4 poster=assets/still.jpg -->` |
| `none` | turns off the deck default on one slide |

Options (images and videos; `overlay` also for colours):

| Option | Meaning |
|---|---|
| `fit=cover` (default) / `contain` | `object-fit` of the media |
| `position=center` | `object-position`; use `-` for spaces: `left-top`, `center-bottom` |
| `opacity=0.35` | fade the media into the page background |
| `blur=4` | blur radius in tenths of a percent of the slide width (scales with the slide) |
| `overlay=auto` (default for media) | `auto` measures the image and picks `dark` or `light`; `dark` darkens and switches text to light; `light` brightens and switches text to dark; `none` leaves it alone |
| `poster=assets/still.jpg` | video poster; shown with reduced motion |

Readable text is automatic: dark backgrounds (colours, gradients by their
first hex colour, media by measurement) get the `si-bg-dark` class, which
switches the slide tokens to a light-on-dark palette; light ones get
`si-bg-light`. Videos are muted, loop, play only on the current slide, and
stay on their poster or first frame with reduced motion.

A deck-wide default goes in front matter: `background: linear-gradient(...)`.
