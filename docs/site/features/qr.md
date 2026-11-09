---
title: QR codes
description: A QR code for a link, any text, a Wi-Fi network or a contact, on a slide or as a PNG, JPG, SVG or PDF file, with your logo in the middle.
---

# QR codes

On a slide or a page, one line:

```markdown
<!-- qr: https://simple-ideas.org/ · caption=simple-ideas.org · dots -->
```

<!-- qr: https://simple-ideas.org/ · caption=simple-ideas.org · dots -->

It is drawn at build time as an SVG (nothing is fetched), scales with the
slide like any figure, stays dark on light in every theme (an inverted code
reads badly), and is also a link for whoever reads the page on a laptop. Put
one on the last slide and the room leaves with your slides.

| Write | What it does |
|---|---|
| `caption=text` | a line under the code |
| `size=small\|medium\|large` | the width (default medium) |
| `dots` | round modules, rounded corner squares |
| `logo=assets/mark.svg` | your logo in the middle (error correction H) |
| `ec=L\|M\|Q\|H` | error correction (default M) |

## As a file

```bash
si qr https://simple-ideas.org                       # qr.png, 1024 px
si qr https://simple-ideas.org --type png,svg,pdf --name poster
si qr https://simple-ideas.org --logo mark.png --style dots --fg '#1f3a93'
si qr https://simple-ideas.org --pdf --size 5cm     # a print size, vector
si qr --wifi "Office" --password 'secret'          # join a network
si qr --vcard me.vcf                                 # a contact card
echo "any text" | si qr - --terminal                 # from stdin, also in the terminal
```

| Option | What it does |
|---|---|
| `-o name.png`, `--name name` | the file name; the extension picks the format |
| `--png --jpg --svg --pdf`, `--type a,b` | one or more formats (default PNG) |
| `--logo picture.png` | a picture in the middle; PNG or JPEG for every format, SVG for SVG output |
| `--logo-size 20` | its share of the width, in percent |
| `--style square\|dots` | the look of the modules |
| `--fg '#rrggbb'`, `--bg '#rrggbb'\|none` | colours; `none` is a transparent background (not for JPG) |
| `--ec L\|M\|Q\|H` | error correction: about 7, 15, 25 or 30 % of the code can be lost |
| `--size 1024`, `--size 5cm` | pixels for PNG and JPG; a print size for PDF and the SVG width |
| `--quiet 4` | the white margin, in modules |

## A logo that still scans

A logo hides part of the code, and the error correction rebuilds what it
hides. si knows which bytes the logo covers, so it checks before it writes
the file: a logo too big for the code to read past is refused, with the
largest size that works.

```text
si qr: --logo-size 45 is too big for this code (error correction H): at most 28
```

It also warns when the colours are too close, or when a light code sits on
a dark background (some phones cannot read an inverted code).

## Checked by a real reader

si makes its QR codes itself (no library, no network), every size from
version 1 to 40 at every error correction level. Each of them, square and
dotted, with and without a logo, was decoded by Apple's barcode reader, and
the test suite pins those codes module for module.

## The rules

- [23. Sites (the qr plugin)](../rules/23-sites.md)
