# 29. Timelines

A timeline is a one-page infographic: eras with their colours, dated
milestones with a short description and a picture, a closing line and the
sources. It is a page of its own (`type: timeline`) or a ```` ```timeline ````
fence inside a slide or a document.

### A timeline page

```markdown
---
type: timeline
title: Hitos en Algoritmos y Complejidad Computacional
subtitle: Del algoritmo de Euclides (300 a.C.) a la inteligencia artificial
lang: es-MX
layout: snake
closing: La pregunta P vs NP, planteada en 1971, sigue sin respuesta.
require: 20 dates, 15 images
---

## Antigüedad · color=#b3541e

### c. 300 a.C. · Algoritmo de Euclides
En "Los Elementos" (libro VII) Euclides describe el método para calcular el
máximo común divisor.
![Retrato de Euclides](img/euclides.jpg "Wellcome Collection, CC BY 4.0")

### c. 240 a.C. · Criba de Eratóstenes
Método sistemático para encontrar todos los números primos hasta un límite dado.

## Edad Media

### c. 825 · Al-Juarismi
Su nombre latinizado ("Algoritmi") da origen a la palabra "algoritmo".
![Al-Juarismi](https://upload.wikimedia.org/…/al-khwarizmi.jpg "Dominio público")

## Fuentes · sources
- Cormen, T. H., et al. (2022). Introduction to algorithms (4.ª ed.). MIT Press.

Notes: Empezamos en la Antigüedad. [CLICK] En la Edad Media…
```

- `## Era` is an era: a band of colour on the line and an entry in the
  legend. `· color=` takes any CSS colour; without it the era gets the next
  colour of a palette turned around the theme's accent (`--si-tl-base`),
  readable in light and dark.
- `### date · title` is a milestone. The lines under it are its description
  (one or two lines read best; `si check` warns above 220 characters).
  `![caption](src "credit")` is its picture, drawn as a medallion on the line,
  with the caption and the credit under the description. `src` is a file
  beside the page or an `https://` address, fetched once at build time
  (cached) and inlined, so the page stays one file.
- Dates are shown as written. si reads them only to check the order and for
  `scale: time`: `300 a.C.`, `300 BC`, `BCE`, `c.` / `ca.`, years, `1936-05`,
  ranges `2022–2024` (the start counts).
- `## Anything · sources` is the sources block, shown at the end.
- On a landscape page the timeline is one slide: `si play` presents it,
  `Notes:` and `Points:` at the end are its teleprompter script and speaker
  points, and each era after the first is a build step (`steps: off` shows
  them all at once). Text outside a milestone is not shown (`si check` says
  so).

| Front matter | Values | Default |
|---|---|---|
| `type` | `timeline` | |
| `title`, `subtitle` | the header | |
| `layout` | `snake` (the line wraps across rows: 30 milestones fit one 16:9 page), `line` (one horizontal line, two levels a side), `vertical` (one line down the page, cards left and right) | `snake`; `vertical` when portrait |
| `orientation` | `landscape` (a slide: a frame of `aspect:` that scales like a picture), `portrait` (a tall scrolling poster, a document page) | `landscape` |
| `aspect` | the landscape frame: `16:9`, `16:10`, `4:3`, `1:1` | `16:9` |
| `alternate` | `on` (milestones above and below the line, or left and right), `off` (one side) | `on` |
| `scale` | `even` (equal gaps), `time` (gaps follow the years, held between 0.6 and 1.8 of the usual gap; a shortened long gap gets a ≈ mark) | `even` |
| `closing` | a line at the end of the timeline | none |
| `steps` | `eras` (one build step per era on the slide), `off` | `eras` |
| `require` | a rubric for `si check`: `20 dates, 15 images, 7 eras, 4 sources` | none |
| `lang`, `theme` | as on any page; every built-in theme works, light and dark | |

On a phone (narrow or portrait screen) every layout becomes `vertical`, with
the line on the left.

### In a slide or a document

````markdown
## Sorting, briefly

```timeline layout=line steps
## Cards
### 1890 · Hollerith tabulator
Punched cards sorted by machine for the US census.
## Libraries
### 2002 · Timsort
Tim Peters writes the sort Python still uses.
```
````

The fence takes the same Markdown (a `# Title` line inside it is the title)
and the options `layout=`, `alternate=`, `scale=`, `aspect=`,
`orientation=`, `title=`, `subtitle=`, `closing=` and `steps` (one build
step per era after the first; slides only). On a slide it fills the slide; in
a document it is `vertical` unless `layout=` says otherwise.

### Check and export

- `si check` prints the counts (`timeline: 30 milestones, 30 pictures, 7
  eras, 5 sources`) and reports missing pictures, milestones out of
  chronological order, long or empty descriptions, and an unmet `require:`.
- `si build timeline.md --png` writes `timeline.png`: the landscape frame with
  every era shown, or the whole portrait poster; `--scale 3` for more pixels
  (default 2: a 16:9 timeline is 3200 × 1800). `--pdf` writes
  `pdfs/timeline.pdf` (one landscape page, or the printed poster). Both need
  Chrome.
- `si new history --type timeline` writes an example timeline with its
  pictures; `si --export example --type timeline` the same.
