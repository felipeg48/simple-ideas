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
| `voice:` | the build's voice, `engine:name` (`kokoro:af_heart`, `openai:nova`) | si.conf `voice:`, else Kokoro |
| `t2s-voice:` | the browser's voice to prefer when there is no build voice (`Samantha`) | the browser's voice for the page's `lang:` |

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

**The voice.** The same voices as auto-play and `si demo`: `si build`
voices every sentence with the page's `voice:`, else si.conf `voice:`, else
Kokoro when it is installed (`si voice install kokoro`), and writes them
beside the page as `<page>.voice/` (m4a, one file per sentence; a rebuild
voices only the sentences that changed). `--voice kokoro:bf_emma` picks one
for the build. `si play` voices each sentence as it is read. The build
needs Chrome (it asks the page for its sentences) and ffmpeg; without them,
or with only the system voice, the browser's own speech reads the page (one
sentence at a time; Safari, Chrome and Edge have voices everywhere). Ship
the `.voice/` folder with the page.

````text
si build guide.md -o site                       # Kokoro, af_heart
si build guide.md -o site --voice kokoro:bm_george
````

With the browser's voice the voice menu lists its voices and `t2s-voice:`
picks one. `t2s-highlight: sentence` and `word` mark the text with CSS highlights (the
page's markup is not touched); `word` follows the voice through the sentence (by its share of the letters
with the build's audio), and a browser
without CSS highlights lights the paragraph. Print and PDF leave the bar
out. On slides, auto-play (RULES §20, `v`) reads `Notes:` instead.
