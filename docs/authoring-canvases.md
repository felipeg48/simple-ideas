# Authoring an animated canvas from scratch

RULES.md §10 is the *contract* — what the markup compiles to. This is the
*recipe* — how to get from a blank page to a working animation. An si canvas is
**one inline SVG whose parts change as the state word changes**; the deck only
flips `data-state`, and the SVG's own helpers and CSS do the rest. There is no
visual editor — the loop is **edit the SVG, `si play` reloads, look**.

## 0. Start from the scaffold

```
si new-canvas pipeline      # writes pipeline.svg + pipeline.md
si play pipeline.md         # open it; edit pipeline.svg and it reloads
```

`pipeline.svg` is a complete, commented example (a token moving through three
stages). The fastest way to learn is to change it and watch. The steps below
are what that scaffold is made of.

## 1. Write the states down first

Before drawing, list the states and what each shows. This table *is* your
animation:

| state word | what the viewer sees |
|---|---|
| `s1` | item at stage A |
| `s2` | item at stage B |
| `s3` | item at stage C, stage C green, a check |

Keep it to a handful of words. A word is just a token in `data-state`; parts
react to it.

## 2. Draw the static picture

Author plain SVG for the "everything visible" layout. Rules that matter:

- Root: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 W H" class="pipeline" aria-label="…">`.
- Colour only with deck tokens + a fallback: `fill="var(--si-surface,#fff)"`,
  `stroke="var(--si-line,#d9d6ce)"`, text in `var(--si-fg,#16171b)`.
- Colour `<text>` with a **CSS class**, never a `fill=` attribute — a
  `.pipeline text{fill:…}` rule overrides presentation attributes on `<text>`.
- Never write a `<` inside the `<style>` block (it's parsed as HTML markup).

## 3. Reveal / recolour parts with helper attributes (no CSS)

Put these on a shape or group; they react to the current state (RULES §10 has
the full table):

- `data-show="s3"` — visible only when the state has `s3` (hidden otherwise).
- `data-hide="s3"` — hidden when the state has `s3` (`data-hide` wins).
- `data-ok="s3"` / `data-alert="s1"` — shapes turn `--si-ok` / `--si-crit`
  (and `data-alert` pulses); a `<text>` with it recolours too.
- `data-flow="…"` (marching dashes) and `data-packets="…"` (travelling dots)
  for paths.
- `data-grow="…"` — a bar scales up from its base (0 → full) when the word is
  set; good for a value growing over steps.
- `data-move="…"` — the element travels to the end of its own `offset-path`
  (you set `offset-path: path('…')` in the SVG's CSS) when the word is set.
- `data-count="s1:0 s2:4200"` — the element's text ticks to the number of the
  active state word (the last matching word wins); animated on the current slide,
  instant in print/PDF and under reduced motion.

Each takes several words; any match fires it.

## 4. Move / transform parts with CSS keyed to the state

For motion, write CSS **inside the SVG**, one transition plus a rule per state:

```css
.pipeline .token { transition: transform .8s cubic-bezier(.3,.1,.2,1); }
.pipeline[data-state~="s2"] .token { transform: translateX(330px); }
.pipeline[data-state~="s3"] .token { transform: translateX(660px); }
```

Same specificity → **later rule wins**, so list states in order (the furthest
one applies when several words are present). Use `transform-box: fill-box` when
scaling/rotating a shape around its own centre.

## 5. Drive it from the deck

```markdown
## The item moves through three stages

<!-- svg: pipeline.svg s1 -->
<!-- steps: +s2,-s1 +s3,-s2 -->
- It advances to stage B
- It reaches stage C
```

- `<!-- svg: file words -->` inlines the SVG and sets the base state. Plain
  words replace the state; `+word` / `-word` change the state the *previous*
  slide with the same file left (so a canvas can animate across slides, not
  only across steps).
- `<!-- steps: g1 g2 … -->` advances the state per keypress; one group per
  bullet. `+s2,-s1` means "add s2, remove s1".

## 6. Preview and iterate

`si play pipeline.md`, then press → to walk the states. Edit the SVG, save, the
browser reloads. For a quiet check: `si check`, and a headless screenshot of a
specific step (`…/pipeline.html#2.2`).

## Checklist before you ship a canvas

- [ ] root has a `viewBox` and a `class`
- [ ] colours are `var(--si-*, fallback)`; `<text>` colour is via a class
- [ ] no `<` anywhere inside `<style>`
- [ ] every state word in the steps line is handled by a helper or a CSS rule
- [ ] the final step lands on the "resolved" state (earlier error words removed
      if they shouldn't linger)
- [ ] looks right in light and dark, and at phone / 4K sizes
