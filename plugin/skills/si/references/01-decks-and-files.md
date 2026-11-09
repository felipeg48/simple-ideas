# 1. Decks and files

- Every `*.md` file in the directory you give `si` is a deck, except
  `README.md` and files whose name starts with `_` or `.` (use `_module.md`
  for a part that only exists to be included).
- One deck becomes one HTML file with the same name: `talk.md` → `talk.html`.
- Paths (images, SVGs, backgrounds, includes) are relative to the markdown
  file that mentions them.
