# 4. Directives

Directives are HTML comments on a line of their own:

```markdown
<!-- name: value -->
<!-- name -->
```

They stay invisible on GitHub and in other markdown viewers. A comment whose
first word is followed by anything other than `:` or `-->` is an ordinary
comment (`<!-- note to self -->`, `<!-- TODO -->`); a lowercase single word
that is not a directive (`<!-- wobble -->`) is an error.

| Directive | Scope | Section |
|---|---|---|
| `layout: center` | slide | [Layouts](#5-layouts-and-columns) |
| `column` | body | [Layouts](#5-layouts-and-columns) |
| `transition: zoom`, `transition: morph 0.9s` | slide | [Transitions](#6-transitions) |
| `id: promise` | next block | names the block so `transition: morph` matches it on the next slide ([Transitions](#6-transitions)) |
| `background: assets/city.jpg opacity=0.4` | slide | [Backgrounds](#7-backgrounds) |
| `footer: off` | slide | overrides the deck `footer` on this slide (`off` hides it, e.g. the title) |
| `labs: [Create a Cluster](https://…), [Scale out](https://…)` | slide | turns the slide into a lab slide: a centred beaker icon, the `labs-label` heading, and a button per lab (opens in a new tab). A bare URL works too and shows a default label. |
| `class: dense dark-code` | slide | adds classes to the slide for your CSS |
| `eyebrow: Order #4711 · the promise` | slide | a small context label above the slide heading (`.si-kicker`); it is not part of the slide title, so find, the overview and the teleprompter show the heading alone. On a section slide it replaces the "Part N" kicker. |
| `image: width=40% align=right` | next image | [Images](#8-images) |
| `steps`, `step`, `steps: a b`, `step: a` | next block | [Build steps](#9-build-steps) |
| `svg: assets/canvas.svg state words` | body | [SVG canvases](#10-animated-svg-canvases) |
| `include: other.md` | deck | [Includes](#12-includes) |

Body directives (`column`, `id`, `image`, `step`, `steps`, `svg`) work on the
top-level blocks of a slide, not inside lists or quotes.
