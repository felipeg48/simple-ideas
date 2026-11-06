# Code windows and code shows (v2.1.0)

Approved by the founder on 2026-11-06 from the mockup next to this file
(`2026-11-06-code-windows.html`).

## What

- A code block becomes a window: macOS, Windows, a terminal or a plain card,
  optionally on a gradient backdrop, with line numbers and highlighted lines.
- Callouts are placed by hand: a numbered dot at the end of a code line, a
  dashed wire, a note you drag; the place is written back as `at=dx,dy`.
- A run of blocks with the same `id=` is a show: one window whose lines glide
  from step to step (morph), or fade, or slide.
- It goes wherever a fence goes: documents (← → Play), slides (one click per
  extra step; the same id across consecutive slides morphs with the slide
  change), mind maps (a code topic with a title bar).
- Exports: PNG and SVG per step, an animated GIF of the show (`si code`).
- The Studio has a Window view, like the mind map editor: look, steps,
  callouts (add, edit, drag), exports, every change as one undoable edit.

## Markdown

See RULES.md §25 and `examples/code/retry.md`.

## Decisions

- The window is drawn by the runtime (`web/codewin.js`) from per-step JSON
  (highlighted lines + plain text); without JavaScript every step is a plain
  code block with its callouts as a list.
- The morph matches lines by their text and occurrence; unchanged lines keep
  their element and move, new ones fade in after the move.
- Exports reuse the mind map approach: headless Chrome loads an article of only
  the windows at `#si-code-export`, the runtime writes SVG frames
  (foreignObject, the window's CSS and tokens, data-URI fonts), Chrome
  rasterises, Go encodes the GIF (stdlib `image/gif`, a popularity palette with
  no dithering, which kept a four-step show under 1 MB; five morph frames between steps).
- Callout drags write back only in the Studio (postMessage from the preview to
  the Window view); in `si play` and built pages a drag is local.

## Next

- A show's steps as builds in `si play --terminal` (stacked today).
- Editing a step's code inside the Window view (it is edited in the text).
- Completion for fence options in the editor.
- MP4 export.
