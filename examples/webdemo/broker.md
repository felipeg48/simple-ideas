---
type: demo
title: A message broker, from the terminal to its console
app: http://127.0.0.1:7310
viewport: desktop
timeout: 10s
voice: kokoro:af_heart
shell: bash
start: go run ./broker serve --port 7310 as broker · wait for port 7310 · within 120s
setup: go run ./broker reset --port 7310
stop: go run ./broker reset --port 7310
with: subtitles, chapters
---

# A message broker, from the terminal to its console

## Send a message from the terminal

What it does: a client sends one message to the broker, and the broker's console shows it waiting in the queue.

Say: The broker runs on this machine. From a terminal, we send it one message.
- terminal "Client" · backdrop=ocean
- run "go run ./broker send hello from the terminal" · wait
- expect output "sent 1 message"
- note "One message, sent" at line "sent 1 message"

Say: The broker's console shows it, waiting in the queue for a consumer.
- browser
- expect text "1 message waiting"
- note "Waiting for a consumer" at "1 message waiting"
- expect no errors

## Watch a workload fill the queue

What it does: a load generator sends thirty messages while the console counts them.

Say: A load generator sends thirty messages, one every tenth of a second.
- start "go run ./broker load --count 30 --every 100ms" as load · wait for "sending"
- expect load running
- wait for "30 messages waiting"

Say: When it is done, it says so, and the console agrees.
- stop load · wait
- expect load exited 0
- expect output of load "sent 30 messages"
- expect output of broker "received #30"
- expect text "30 messages waiting"

## Stop a broker on camera

What it does: a second broker starts in a terminal, says it is ready, and stops cleanly on Control C.

Say: A second broker starts in a terminal of its own, on port 7311.
- terminal "Broker" · window=mac theme=dusk backdrop=dusk title="broker on 7311" size=90x22
- run "go run ./broker serve --port 7311"
- expect output "broker ready on 127.0.0.1:7311" · within 60s
- note "Ready for messages" at line "broker ready"

Say: Control C stops it, and it says goodbye.
- press ctrl+c
- expect output "broker stopped"
- expect no output "panic"
