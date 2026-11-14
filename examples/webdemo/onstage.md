---
type: demo
title: Orders wait in a queue, on stage
app: onstage/talk.md
viewport: desktop
timeout: 10s
voice: kokoro:af_heart
shell: sh
start: si serve onstage/console --addr 127.0.0.1:7421 as console · wait for port 7421
---

# Orders wait in a queue, on stage

## The queue fills

What it does: the slide about the queue stays on screen while the presenter's terminal sends three orders and the broker's console shows them waiting.

Say: The shop puts each order in a queue, and the warehouse takes them at its own pace.
- slide 2
- wait 1s

Say: Three orders go out from the terminal, right under the slide.
- embed-terminal "Lab 1"
- run "echo sending 3 orders && sleep 1 && echo sent 3 orders to the queue"
- wait for output "sent 3 orders to the queue"

Say: And here is the broker's console, open over the slide. The queue is holding all three.
- embed-browser "Console" · at 640,70 · size 720x420 · zoom 110%
- click "Queues" in embed-browser "Console"
- expect "orders: 3 messages" visible in embed-browser "Console"
- wait 1500ms

Say: Nothing is lost while the warehouse catches up.
- embed-browser "Console" close
- embed-terminal close
- next
- wait 1s
