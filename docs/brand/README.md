# simple-ideas brand

The mark is the **Stack**: three isometric slabs, the deck, the article and
the book that si makes from one Markdown file.

| File | Use |
|---|---|
| `mark.svg` | the mark in colour, on light or dark backgrounds |
| `mark-mono.svg` | one colour (`currentColor`), for monochrome places |
| `favicon.svg` | simplified for 16 px: three faces, no sides |
| `app-light.svg`, `app-dark.svg` | the mark on an app tile |
| `wordmark.svg`, `wordmark-dark.svg` | mark + "simple-ideas" (system fonts) |
| `social-card.svg` / `.png` | 1200×630 card for links (og:image) |
| `apple-touch-icon.png` | 180 px home-screen icon |

The desktop icon (`desktop/AppIcon.icns`) is drawn by `desktop/icon/gen.go`
from the same geometry: `make desktop-icon`.

## Colours

| Name | Hex | Where |
|---|---|---|
| Indigo | `#5b4bdb` | top slab |
| Teal | `#1fc8a0` | the two lower slabs, at 85% and 55% |
| Ink | `#15171c` | wordmark on light |
| Night | `#0d0f15`–`#1b1e2b` | dark tile and card backdrop |

Side faces use the slab colour at 75% (left) and 55% (right) of its opacity.

## Use

- Keep clear space of a quarter of the mark's width around it.
- Do not recolour the slabs except in `mark-mono.svg`; do not rotate or skew it.
- si never puts this mark on pages people build with it: built decks and
  documents carry their author's branding only.
