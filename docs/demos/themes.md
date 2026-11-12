---
type: demo
title: How a deck looks
app: http://127.0.0.1:7394
viewport: 1440x900
timeout: 20s
voice: kokoro:af_heart
pronounce: si = ess eye
setup: rm -rf talk
---

# How a deck looks

Nine built-in themes, each in light and dark, and a theme made from your own
logo. A theme changes colours, type and shapes, never the layout, so the same
talk reads well in all of them.

## Nine themes, light and dark

What it does: one line in the file changes the whole look, and every theme comes in light and dark.

Say: si comes with nine themes. Each one is a single word.
- terminal "si" · window=mac theme=night backdrop=graphite
- start "si new talk" as scaffold
- stop scaffold · wait
- terminal "play" · window=mac theme=night backdrop=graphite
- run "si play talk --addr 127.0.0.1:7394"
- wait for output "serving"
- note "Served, with live reload" at line "serving"
- wait 1500ms
- terminal "si"
- run "si theme list" · wait
- expect output "newsprint"
- note "Nine looks" at line "NAME"
- wait 2s

Say: Here is the talk in the default theme, Studio.
- browser
- go /talk.html#5
- expect text "The journey on one screen"
- wait 2s

Say: Change one line, and it turns into Forest.
- terminal "si"
- run "clear; sed -i.bak 's/^theme: studio/theme: forest/' talk/talk.md && grep '^theme:' talk/talk.md" · wait
- browser
- wait 3s

Say: Newsprint, for an editorial feel.
- terminal "si"
- run "sed -i.bak 's/^theme: forest/theme: newsprint/' talk/talk.md && grep '^theme:' talk/talk.md" · wait
- browser
- wait 3s

Say: And every theme has a dark side, one key away.
- press d
- wait 3s
- press d
- wait 1s

## A theme from your logo

What it does: a logo's colours become a theme for light and dark, where every pair of colours is easy to read.

Say: Start from the company logo.
- terminal "si" · window=mac theme=night backdrop=graphite
- start "si new talk" as scaffold
- stop scaffold · wait
- terminal "play" · window=mac theme=night backdrop=graphite
- run "si play talk --addr 127.0.0.1:7394"
- wait for output "serving"
- note "Served, with live reload" at line "serving"
- wait 1500ms
- terminal "si"
- run "si theme from talk/assets/northwind.svg -o talk/northwind.css" · wait
- expect output "northwind.css"
- note "The colours si found" at line "colours"
- wait 2s

Say: Every pair of colours passes the contrast check, in light and in dark.
- run "clear; si theme check talk/northwind.css" · wait
- wait 2s

Say: Point the talk at it, and the deck wears the brand.
- run "sed -i.bak 's/^theme: studio/theme: northwind.css/' talk/talk.md" · wait
- browser
- go /talk.html#5
- expect text "The journey on one screen"
- wait 3s
- press d
- wait 3s
