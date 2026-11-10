---
type: demo
title: The si Studio
app: http://127.0.0.1:7393/?token=si-demo-studio-token-0123456789abcdef
viewport: 1440x900
timeout: 30s
voice: kokoro:af_heart
pronounce: si = ess eye
start: rm -rf talk; si new talk >/dev/null && SI_STUDIO_TOKEN=si-demo-studio-token-0123456789abcdef si --web talk --no-open --addr 127.0.0.1:7393 · wait for port 7393 · within 60s
stop: rm -rf talk
---

# The si Studio

`si --web` opens the Studio on a folder: files, an editor that knows si, the
slide you are writing beside it, and the problems as you type.

## Write with a live preview

What it does: your files, the Markdown, and the slide it makes, side by side; the slide follows every keystroke.

Say: This is the si Studio. Your files on the left, the text in the middle, and the slide on the right.
- wait 1s
- expect text "talk.md"

Say: Open a deck, and its first slide appears beside the text.
- click "talk.md"
- expect text "The Thursday Promise"
- wait 1s

Say: Change the title, and the preview follows as you type.
- click "# The Thursday Promise"
- press End
- type ", live"
- expect text "The Thursday Promise, live"
- wait 2s

## Problems as you type

What it does: si check runs while you write, and points at the line.

Say: A typo in a directive shows up as a problem at once, with its line.
- click "talk.md"
- expect text "The Thursday Promise"
- click "# The Thursday Promise"
- press End
- press Enter
- press Enter
- type "<!-- layuot: center -->"
- expect text "unknown directive"
- wait 2s

Say: Fix it, and the problem is gone.
- press cmd+z
- wait 1500ms

## A theme from your logo

What it does: si reads a logo's colours and builds a theme for light and dark, where every pair of colours is easy to read.

Say: Every deck can wear a theme. The command palette makes a new one.
- press cmd+k
- type "New theme"
- press Enter
- expect text "New theme"

Say: Start from the company logo. si finds its colours, and builds the theme for light and dark.
- click "From a logo…"
- select "assets/northwind.svg" in "Logo in this folder"
- expect text "In the logo"
- note "The colours si found" at "In the logo"
- wait 2s

Say: Name it, create it, and every pair of colours passes the contrast check.
- type "Northwind" into "Name"
- click "Create"
- expect text "all pass"
- wait 2s
