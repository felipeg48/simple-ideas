# A theme from a logo

Status: spec, not started. Order: 4 of 5.

## Why

The first thing anyone does with a new deck tool is make it look like their
company. si has themes as CSS token files (`theme: brand.css`), but writing
one means knowing the tokens. A logo already carries the brand's colours.

## What the author runs

```bash
si theme from logo.png                 # writes brand.css next to it, prints a preview path
si theme from logo.svg -o themes/acme.css --name "Acme"
si theme from logo.png --accent '#e4572e'   # keep one colour you know, pick the rest
```

Then `theme: themes/acme.css` in front matter or si.conf, and the theme
gallery (`si --export themes`) shows it beside the built-ins.

## What it picks

1. **Colours from the picture.** PNG/JPEG decoded with `image/*`; SVG read
   as text for `fill`/`stroke`/`stop-color` values. Pixels quantised
   (k-means in OKLab, k = 6, transparent and near-white/near-black ignored),
   weighted by area and saturation.
2. **Roles.** The most saturated frequent colour → `--si-accent`; the next,
   distinct in hue → `--si-accent-2`; surfaces and lines are tinted neutrals
   derived from the accent's hue (not grey): `--si-bg`, `--si-surface`,
   `--si-surface-2`, `--si-line`, `--si-fg`, `--si-muted`.
3. **Light and dark.** Both palettes, the dark one under
   `@media (prefers-color-scheme: dark)` and `[data-si-mode=dark]`, as the
   built-in themes do.
4. **Contrast, checked.** Every text/background pair meets WCAG AA (4.5:1
   body, 3:1 large and UI); an accent that fails as text is darkened (light)
   or lightened (dark) in OKLab until it passes, and the CSS says so in a
   comment. `--si-on-accent` is black or white, whichever contrasts more.
5. **Fonts** are left to the author (a comment shows `fonts:` examples).
6. **The logo itself**: `--si-brand-logo` set to the image (inlined at build
   like any theme url()), so `<!-- logo: × Partner -->` shows it.

## Output

A commented CSS file in `@layer si.theme`, tokens only (rule #6), plus a
preview deck path (`brand.preview.md` with title, two-col, code, quote, a
mind map, a QR code) the command offers to open with `si play`.

## How it fits si

`internal/themegen` (new, stdlib: `image`, `image/png`, `image/jpeg`,
`encoding/xml`), `cmd/si` command `theme from`. Token names from
`internal/assets/contract.go`, so the generated file can never use a token
that does not exist (a test checks every `--si-*` it writes is in
`Tokens`).

## Tests

- Palette extraction on fixture logos (flat SVG, PNG with transparency, a
  photo-like JPEG) with expected accent hues (± tolerance).
- Every generated text pair meets its contrast ratio, light and dark.
- The generated theme renders the showcase with no `si check` problems, and
  the a11y suite (axe) passes with it.

## Open questions

- Gradients in the logo: use their stops as accent candidates (proposed: yes).
- A `--from-url` that takes a website's favicon/OG image? (Later; fetching.)
