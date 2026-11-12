# Timelines and infographics: `type: timeline`, `type: infographic`

Status: idea from the founder, 2026-11-12; not built. Do after the courses
work (HANDOFF "NEXT TASK"). Start with design mockups (several layouts ×
themes, as with courses) and get them approved before building.

## Why

A one-page visual for a class, a report or a slide: a timeline with dates,
eras, pictures and sources, at 16:9, with a wow factor. The founder's sample is
a university activity (Spanish, Mexico): "Hitos en Algoritmos y Complejidad
Computacional", 30 milestones from Euclid (c. 300 a.C.) to 2024, 7 eras each
with its colour, at least 15 pictures with captions, a closing line and a
"Fuentes" section. Graded by rubric, so nothing may be dropped or invented:
every date, title, description and source in the Markdown must appear.

## The Markdown

```markdown
---
type: timeline
title: Hitos en Algoritmos y Complejidad Computacional
subtitle: Del algoritmo de Euclides (300 a.C.) a la inteligencia artificial
lang: es-MX
layout: snake            # line | vertical | snake | eras | cards
alternate: on            # milestones above/below (left/right) the line
scale: even              # even | time (gaps by real distance, long gaps broken)
aspect: 16:9             # the page keeps this ratio and scales (cqi), never px
theme: studio
closing: La pregunta P vs NP, planteada en 1971, sigue sin respuesta.
---

## Antigüedad · color=#b3541e

### c. 300 a.C. · Algoritmo de Euclides
En "Los Elementos" (libro VII) Euclides describe el método para calcular el
máximo común divisor; es el algoritmo no trivial más antiguo que se sigue usando.
![Euclides](img/euclides.jpg)

### c. 240 a.C. · Criba de Eratóstenes
Método sistemático para encontrar todos los números primos hasta un límite dado.
![Criba de Eratóstenes](img/criba.svg)

## Edad Media

### c. 825 · Al-Juarismi
…

## Fuentes · sources
- Cormen, T. H., Leiserson, C. E., Rivest, R. L., y Stein, C. (2022). …
```

- `## Era` = an era (a band of colour; `· color=` optional, else a palette
  from the theme with enough contrast in light and dark).
- `### date · title` = a milestone; the paragraph under it is the
  description (1–2 lines; `si check` warns when longer); `![caption](src)`
  is its picture with the alt text as the caption.
- Dates understood: `300 a.C.` / `300 BC` / `BCE`, `c.`/`ca.`, years, ranges
  `2022–2024`, months `1936-05`; kept exactly as written on the page, parsed
  only for order and `scale: time`. `si check` warns when out of order.
- `## … · sources` = the sources block at the end; `closing:` = the line at
  the end of the timeline.
- Pictures: local files, or `https://` (e.g. Wikimedia Commons) fetched and
  inlined at build time like web fonts, with the credit line shown when given
  (`![Ada Lovelace](https://… "Wikimedia Commons, dominio público")`).
- `si check`: counts (milestones, pictures, eras), missing pictures, order,
  description length; `--budget` style rules optional (`require: 20 dates, 15
  images`) so a rubric can be checked.

## Layouts and looks

- `line`: one horizontal line, milestones alternating above/below.
- `vertical`: one vertical line, alternating left/right (long pages, phones).
- `snake`: the line wraps across rows (fits 30 milestones on one 16:9 page).
- `eras`: one column per era, coloured headers.
- `cards`: a grid of picture cards on a thin dated rail.
- Every built-in theme, light and dark; era colours from the theme or the
  Markdown. Fluid (rule #1): the 16:9 frame is an `aspect-ratio` box with
  container units; on a phone `vertical` is used.
- Motion (optional): milestones appear era by era; on a slide each era is a
  build step.

## Where it runs

- A page: `si build timeline.md` (HTML), `--png`, `--pdf`, `--svg` through
  headless Chrome as mind maps do (`cmd/si/mindmap.go`); high-resolution PNG
  for a document (`--scale 2`).
- On a slide: a ```` ```timeline ```` fence or `<!-- timeline: file.md -->`,
  filling the slide (like the ```` ```mindmap ```` fence).
- No new command: `si build`, `si play`, `si check` know the type; `si new
  <name> --type timeline`.

## `type: infographic` (second phase)

The same idea for other one-page visuals: big numbers, comparisons
(before/after, A vs B), steps, icon rows, bars and donuts drawn as SVG from
Markdown tables, a sources footer; same layouts machinery, same exports, same
16:9 / portrait aspect choice. Spec it after timelines ship.

## Docs

RULES.md section, manual, agent rules + skill, FEATURES, CHANGELOG, README,
docs site feature page with the founder's sample as the live example
(`examples/timeline/`), HANDOFF, CLAUDE.md roadmap.
