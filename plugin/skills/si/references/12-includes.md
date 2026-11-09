# 12. Includes

```markdown
<!-- include: appendix.md -->
```

The file's slides are spliced in place; its `# H1` becomes a section slide
("Part N"). Its own front matter is ignored. Includes nest; a cycle
(`a.md` → `b.md` → `a.md`) is an error. Paths inside the module stay relative
to the module.

**Recommended layout for a multi-section deck** (and the standard output of a
migration): a thin main deck plus one file per section.

```
deck/
  my-deck.md             front matter, title, agenda, one include per section, closing
  sections/01-intro.md   one file per section — edit these
  sections/02-....md
  images/anim/*.svg       step-animated canvases
  images/diagrams/*.svg   static themed diagrams
```

`si build deck/` builds only `my-deck.md`: `si` skips subdirectories and
non-`.md` files, so the `sections/` fragments never render as separate decks.
A section lives in `sections/`, so its own paths are relative to itself — it
references shared assets as `../images/...`.
