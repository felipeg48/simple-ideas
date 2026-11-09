# 17. What si check reports

`si check <dir|file.md>` exits 1 on any of:

- a deck that does not start with a `# H1` title slide, or is empty
- an unknown directive, layout, transition, theme, mode, aspect or front
  matter key; an accent that is not a colour
- a bad background or image option (`opacity=2`, `width=300px`)
- a missing file: image, SVG, background, poster, include, image inside an SVG
- an include cycle
- a `steps:` list with more state tokens than items
- an `image` directive not followed by a paragraph holding one image
- a second `# H1` inside a slide
- a `type` that is not `deck` or `lab`
- a malformed `stage` block (unknown node, impossible `dot`/`flow` route, bad `dir`)
- an authored canvas SVG with a `<` inside its `<style>`, or with no `viewBox`
  and no `width`+`height` to derive one

Warnings (exit 0): inlined files above 2 MB; slide or body directives in a lab.

**Size budgets.** `si check --budget` also fails a page whose built HTML is
over its budget: 1 MB for a page, 1.5 MB for a page that draws a 3D mind map
(three.js is inlined). `budget.*` in `si.conf` sets the limits and turns the
check on without the flag:

```
budget.page: 900 KB     # any page
budget.3d: 1.5 MB       # a page with a 3D mind map
```

```bash
si check talk/ --budget   # talk.md: the built page is 1340 KB, over its budget of 1024 KB
```

Building with `--no-inline-media` (pictures and video beside the page) is the
usual fix.
