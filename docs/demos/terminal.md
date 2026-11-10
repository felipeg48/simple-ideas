---
type: demo
title: si without a browser
app: none
viewport: 1440x900
timeout: 20s
voice: kokoro:af_heart
pronounce: si = ess eye
setup: rm -rf talk
---

# si without a browser

The same talk in a terminal: slides drawn in text and colour, and the
Studio as a keyboard app. Useful over SSH, on a server, or when a browser is
one window too many.

## Present in the terminal

What it does: the deck drawn in the terminal, with its layouts, builds and pictures, moved with the arrow keys.

Say: The same talk plays in a terminal. Titles are drawn in block letters.
- terminal "si" · window=mac theme=night backdrop=graphite size=120x34
- start "si new talk" as scaffold
- stop scaffold · wait
- run "si play talk/talk.md --terminal"
- wait for output "Operations review"
- wait 3s

Say: The arrow keys move on, one build at a time.
- press ArrowRight
- wait 2s
- press ArrowRight
- wait 2s
- press ArrowRight
- wait 2s

Say: The overview shows every slide, and q leaves.
- press o
- wait 3s
- press Escape
- wait 1s
- press q
- wait 1s

## The Studio, in the terminal

What it does: files, a live preview and the problems, all from the keyboard.

Say: The Studio has a terminal version too: files on the left, the slide beside them.
- terminal "si" · window=mac theme=night backdrop=graphite size=120x34
- start "si new talk" as scaffold
- stop scaffold · wait
- run "si --terminal talk"
- wait for output "talk.md"
- wait 3s

Say: The storyboard lists every slide, with its minutes and its builds.
- press s
- wait 4s
- press Escape
- wait 1s

Say: And q quits, back to the shell.
- press q
- wait 1s
