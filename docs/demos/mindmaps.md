---
type: demo
title: Mind maps
app: http://127.0.0.1:7395
viewport: 1440x900
timeout: 20s
voice: kokoro:af_heart
pronounce: si = ess eye
setup: rm -rf ideas.md talk
---

# Mind maps

A nested Markdown list becomes a mind map: in 2D or 3D, with search, folding,
and a Play that walks it branch by branch.

## A list becomes a map

What it does: write a nested list, and si draws it as a map that grows as you type.

Say: A mind map starts as a list. One command writes a small one.
- terminal "si" · window=mac theme=night backdrop=graphite
- run "si new ideas --type mindmap" · wait
- run "clear; cat ideas.md" · wait
- expect output "First idea"
- note "Topics are list items" at line "- First idea"
- wait 2s

Say: Play draws it as a map.
- start "si play ideas.md --addr 127.0.0.1:7395" as play · wait for port 7395
- browser
- go /ideas.html
- expect text "First idea"
- wait 2s

Say: Add a line to the list, and a new branch grows.
- terminal "si"
- run "clear; printf '%s\n' '- Fourth idea' '  - Its first detail' >> ideas.md; tail -3 ideas.md" · wait
- browser
- expect text "Fourth idea"
- wait 3s

## Explore a bigger map

What it does: a map in 3D with linked maps as floors, search across every topic, and Play for a talk.

Say: Here is a bigger map, in 3D. The linked maps float above it as floors.
- terminal "si" · window=mac theme=night backdrop=graphite
- start "si new talk" as scaffold
- stop scaffold · wait
- start "si play talk --addr 127.0.0.1:7395" as play · wait for port 7395
- browser
- go /mindmap/thursday.html
- expect text "The Thursday Promise"
- wait 3s

Say: Find searches every topic, on every floor.
- click "Find"
- type "refund"
- wait 2s
- press Escape
- wait 1s

Say: The same map in 2D is one click away.
- click "Show the map in 3D or 2D"
- wait 3s

Say: And Play walks the map one branch at a time, for a talk.
- click "Play"
- wait 6s
