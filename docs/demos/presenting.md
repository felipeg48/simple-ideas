---
type: demo
title: Presenting a deck
app: .
viewport: 1440x900
timeout: 15s
voice: kokoro:af_heart
pronounce: si = ess eye
setup: rm -rf talk out
---

# Presenting a deck

The built deck in front of an audience: moving through it, finding a slide,
seeing them all at once, a break, light and dark. Every key works the same in
`si play` and in the built file.

## Move through the talk

What it does: arrows move slide by slide, and a slide with steps reveals them one click at a time.

Say: Here is the talk, built to a single file and opened in a browser.
- start "si new talk && si build talk/talk.md -o out" as prep
- stop prep · wait
- go /out/talk.html
- expect text "The Thursday Promise"
- wait 2s

Say: The arrow keys move on. This slide reveals its points one by one.
- press ArrowRight
- expect text "One promise"
- wait 1s
- press ArrowRight
- wait 1s
- press ArrowRight
- wait 1s
- press ArrowRight
- wait 1s

Say: The picture on the next slides keeps its place, and only what changes moves.
- go /out/talk.html#5
- expect text "The journey on one screen"
- wait 1500ms
- press ArrowRight
- expect text "Payment: declined, then paid"
- wait 2s

## Find any slide

What it does: every slide at a glance, and a search across the whole talk.

Say: The overview shows every slide at once. Pick one, and you are there.
- start "si new talk && si build talk/talk.md -o out" as prep
- stop prep · wait
- go /out/talk.html
- expect text "The Thursday Promise"
- wait 1s
- press o
- wait 3s
- press Escape
- wait 1s

Say: Search finds a word on any slide, and jumps straight to it.
- press /
- wait 1s
- type "retry" into "Search slides or go to a number"
- wait 1500ms
- press Enter
- expect text "retry"
- wait 2s

## Take a break, light or dark

What it does: a break timer on screen, and the same deck in light or dark.

Say: A break? One key puts a timer on the screen.
- start "si new talk && si build talk/talk.md -o out" as prep
- stop prep · wait
- go /out/talk.html#2
- expect text "One promise"
- wait 1s
- press t
- wait 4s
- press Escape
- wait 1s

Say: And one key turns the whole deck dark, or light again.
- press d
- wait 2500ms
- press ArrowRight
- wait 2s
- press d
- wait 1500ms
