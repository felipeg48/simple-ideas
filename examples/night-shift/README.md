# The Night Shift — a split-structure sample

A small simple-ideas deck that shows the recommended layout for a multi-section
talk: a thin main deck that pulls in one file per section with `include`.

## Run it

```
si play  .                 # live preview, reloads on save
si build . --output out    # writes out/night-shift.html + out/index.html
```

(Run these from inside this `night-shift/` folder.)

## Structure

```
night-shift.md                      main deck: front matter, title, agenda,
                                    one `include` per section, closing
sections/
  01-mix-and-rise.md                one file per section — edit these
  02-shape-and-bake.md
  03-out-the-door.md
images/
  anim/loaf.svg                     an animated canvas (it carries across the
                                    three bake slides, changing state)
  diagrams/night-timeline.svg       a static, theme-coloured diagram
```

## Why it's laid out this way

- `si build .` builds **only** `night-shift.md`. `si` skips subdirectories and
  non-`.md` files, so the `sections/` files never render as separate decks —
  they are spliced in by `<!-- include: sections/NN-*.md -->`.
- A section file lives in `sections/`, so its own paths are relative to itself:
  it references shared art as `../images/...`.
- See section 12 of `si rules` for the include rules.
