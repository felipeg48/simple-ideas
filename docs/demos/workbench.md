---
type: demo
title: The presenter's workbench
app: http://127.0.0.1:7401/talk.html#2
viewport: 1440x900
timeout: 15s
voice: kokoro:af_heart
pronounce: si = ess eye; rabbitmq.conf = rabbit M Q dot conf; app.log = app dot log
transition: fade
prepare: sh workbench-files.sh
start: python3 -m http.server 7402 --bind 127.0.0.1 --directory broker-dash as dashboard · wait for port 7402
start: SHELL=/bin/bash HOME="$PWD/broker-home" si play broker-talk --addr 127.0.0.1:7401 as talk · wait for port 7401 · within 60s
start: sh broker-dash/logger.sh as logger · wait 1s
cleanup: rm -rf broker-talk broker-dash broker-home
---

# The presenter's workbench

`si play` is more than slides: a drawer of real shells under the deck, a
runbook and demo scripts beside them, files to read, follow and edit, and
browser windows over the slides, all still there after the deck reloads.

## A terminal under the slides

What it does: real shells in a drawer over the deck, as tabs, each with its own text size; the drawer sizes, minimises and fills the window.

Say: Here is the talk, served by si play. One key opens a drawer of real shells, right under the slides.
- expect text "Ship it in one afternoon"
- wait 1s
- press `
- expect .si-term visible
- wait 1500ms

Say: It is the presenter's own machine. The broker's compose file is one command away.
- type "cat docker-compose.yml"
- press Enter
- wait 2s

Say: Another tab is another shell, here following the service log as it grows.
- click [data-act=new]
- wait 1s
- type "tail -f app.log"
- press Enter
- wait 2500ms

Say: Each tab keeps its own text size, large enough for the back row.
- click [data-act=zin]
- wait 400ms
- click [data-act=zin]
- wait 400ms
- click [data-act=zin]
- spotlight .si-term-zoom · label "This tab only"
- wait 2500ms
- spotlight off

Say: The drawer is as tall as the demo needs.
- drag from .si-term-vgrip to 720,180
- wait 1500ms

Say: It fills the whole window for a long command, and minimises out of the way, still running.
- click [data-act=max]
- wait 2s
- click [data-act=max]
- wait 1s
- click [data-act=min]
- expect text "Ship it in one afternoon"
- wait 2s
- press `
- wait 1500ms

## Runbook and demos at hand

What it does: the tools bar on the right holds the runbook and the demo scripts; every command runs in the shell with one click.

Say: On the right edge, a tools bar, like an editor's. The runbook holds the commands for this talk.
- press `
- expect .si-term visible
- wait 1s
- spotlight .si-term-acts · label "Tools"
- wait 2s
- spotlight off
- expect text "Deliveries so far"
- wait 1s

Say: One click runs a command in the shell. No typing on stage, no typos.
- zoom to .si-rb · 0.8s
- click .si-rb-acts .run
- wait 2500ms
- zoom out

Say: The demo scripts sit in the next panel, only their commands, ready to run.
- click [data-act=demos]
- expect text "Smoke test the broker"
- wait 1500ms
- zoom to .si-term-body · 0.8s
- click .si-dm-acts .run
- wait 3s
- zoom out

## Files to read, follow and edit

What it does: Viewer tabs show a folder and a file with syntax colours, follow a growing log, and edit and save in place.

Say: The Viewer opens a folder in a tab, beside the shells.
- press `
- expect .si-term visible
- click [data-act=runbook]
- click [data-act=files]
- expect text "rabbitmq.conf"
- click [data-act=zin]
- click [data-act=zin]
- wait 1500ms

Say: The broker's config, coloured: keys, sizes, durations and ports, with line numbers.
- click .si-fs-row[title$="/rabbitmq.conf"]
- expect text "vm_memory_high_watermark"
- zoom to .si-fs-split · 0.8s
- wait 3s
- zoom out

Say: A log can be followed as it grows, like tail, line by line.
- click .si-fs-row[title$="/app.log"]
- expect text "delivered message"
- click [data-f=follow]
- zoom to .si-fs-split · 0.8s
- wait 4s
- zoom out

Say: And a file can be changed on the spot. Edit, raise the memory limit, and save.
- click .si-fs-row[title$="/rabbitmq.conf"]
- wait 1s
- click [data-f=edit]
- zoom to .si-fs-split · 0.8s
- wait 1000ms
- click "512MB"
- press End
- press Backspace
- press Backspace
- press Backspace
- press Backspace
- press Backspace
- type "1GB"
- wait 1s
- click [data-f=save]
- expect text "Saved"
- wait 2s
- zoom out

Say: New files and folders, renames and deletes, all from the tree.
- click [data-f=done]
- click [data-f=newfile]
- wait 800ms
- type "release-notes.md"
- press Enter
- wait 2s

## Browser windows over the slides

What it does: the broker's dashboard in floating windows over the deck, moved, resized and zoomed one by one, with back and forward, minimised to a dock.

Say: The w key opens a web page in a window, over the slides. The deck lists its links.
- press w
- expect text "Open a browser window"
- wait 1500ms
- click "Queues"
- expect .si-web visible
- wait 2500ms

Say: The window moves by its title bar, and resizes from any edge.
- drag from .si-web-name to 300,120
- wait 800ms
- drag from .si-web-h[data-e=se] to 1150,760
- wait 1500ms

Say: Each window zooms its own page, for a dashboard the whole room can read.
- click .si-web-b[data-a=in]
- wait 600ms
- click .si-web-b[data-a=in]
- wait 2s

Say: Links work inside the page, and back and forward follow them.
- type "127.0.0.1:7402/consumers.html" into .si-web-url
- press Enter
- expect .si-web-b[data-a=back] enabled
- wait 2s
- click .si-web-b[data-a=back]
- wait 2s
- click .si-web-b[data-a=fwd]
- wait 2s

Say: And it minimises to a dock in the corner, a click away.
- click .si-web-b[data-a=min]
- expect .si-web-pill visible
- wait 2s
- click .si-web-pill
- wait 1500ms

## It all comes back after a reload

What it does: save the deck and the page reloads, with every shell still running, every Viewer on its file and every window in its place.

Say: The dashboard in a window, a shell following the log, and the config in a Viewer.
- press w
- click "Consumers"
- expect .si-web visible
- drag from .si-web-h[data-e=sw] to 900,430
- press `
- expect .si-term visible
- type "tail -f app.log"
- press Enter
- click [data-act=files]
- click .si-fs-row[title$="/rabbitmq.conf"]
- wait 2s

Say: Now the deck itself changes, from a second shell, and the page reloads.
- click [data-act=new]
- wait 1s
- type "sed -i.bak 's/^## Ship it in one afternoon$/&, live/' talk.md && rm talk.md.bak"
- press Enter
- wait for "Ship it in one afternoon, live" · within 20s
- wait 1s

Say: Everything comes back: the drawer and its tabs, the shell still running, the Viewer on its file, the window where it was.
- expect .si-web visible
- wait 1500ms
- click .si-term-tab:first-child
- wait 2500ms
- click .si-term-tab:nth-child(2)
- expect text "vm_memory_high_watermark"
- wait 2500ms
