# 39. Listen: text to speech

`feature: t2s` in a document's front matter puts a **Listen** bar under the
title: the page is read aloud, top to bottom, and what is being read is lit.
It works in guides and tutorials, books, articles, labs and site pages. Put
it in `_nav.md` (or `_site.md`, `_course.md`) to turn it on for every page;
a page that should stay silent says `feature: none`.

````markdown
---
type: book
title: The Northwind Operations Reference
feature: t2s
t2s-highlight: word
---
````

| Key | Values | Default |
|---|---|---|
| `feature:` | `t2s`, or `none` to turn off an inherited one (a list, `t2s, …`) | off |
| `t2s-highlight:` | `sentence`, `word`, `paragraph`, `off` | `sentence` |
| `t2s-rate:` | the starting speed, `0.5` to `2` | `1` |
| `t2s-voice:` | part of a voice's name to prefer (`Samantha`, `Daniel`) | the browser's voice for the page's `lang:` |

**The bar.** ▶ Listen shows the listening time; while it plays: Pause /
Resume, ⏮ ⏭ a sentence back or on, ⏹ stop, the speed (0.75× to 2×) and,
when the browser has more than one voice for the page's language, a voice
menu. The reader's speed and voice are remembered in their browser. `l`
plays and pauses; while it plays, a click on a paragraph reads from there.
The page follows the reading unless the reader has just scrolled away.

**What is read.** The title and the description, then the headings (with
their numbers), paragraphs, list items, quotes and callouts, in page order.
Code blocks, tables, figures, keyboard keys, navigation and anything closed
or hidden are left out. Put `class="si-t2s-skip"` on an element to leave it
out too.

**The voice.** The browser's own speech (no network, no files, works from
`file://` and in a built page), one sentence at a time: Safari, Chrome and
Edge on every system have voices; a browser without speech shows no bar.
`t2s-highlight: sentence` and `word` mark the text with CSS highlights (the
page's markup is not touched); `word` needs a voice that reports words (the
system voices do; a cloud voice may only light the sentence), and a browser
without CSS highlights lights the paragraph. Print and PDF leave the bar
out. On slides, auto-play (RULES §20, `v`) reads `Notes:` instead.
