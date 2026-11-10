# Web demos with a terminal: start, run, film

Status: spec, decided with the founder on 2026-11-10; not built. Planned as
v2.36.0, after `si demo run` (video and voice, v2.35.0). The founder's most
wanted part of web demos.

## Why

Many apps are a terminal and a web page together: start a broker (RabbitMQ,
CockroachDB, Kafka) in a terminal, send a message, then look at it in the web
console. Today a demo script can only drive the browser, and `setup:` runs a
command that must exit (a broker never does). The demo needs to:

1. start long-running programs in the background and wait until they are ready;
2. run commands in a terminal the viewer sees, as a step, with real output;
3. check that output (`expect output`), so the same script is the test;
4. film the terminal as a framed window (the code show's terminal look) and
   switch between it and the browser.

## The script

```markdown
---
type: demo
title: The orders broker
app: http://localhost:15672
start: ./bin/broker --port 5672 · wait for "Server startup complete"
---

## Start the broker and watch a queue

What it does: start the broker, publish a message, see it in the console.

Say: First we start the broker.
- terminal "Broker" · window=terminal backdrop=graphite
- run "./bin/broker --port 5672 --node demo"
- expect output "Server startup complete" · within 30s

Say: Then we publish one message.
- terminal "Client"
- run "./bin/send orders hello"
- expect output "sent 1"

Say: The console shows it waiting in the queue.
- browser
- go /#/queues/orders
- expect text "1 message"
```

### Front matter

| Key | What it does |
|---|---|
| `start: <command> · wait for "<text>"` | runs the command in the background before the first feature (in the script's folder), waits until its output shows the text (or `· wait for port 5672`, or `· wait 5s`), and stops it (SIGTERM, then SIGKILL after 5 s) when the run ends, passed or failed. Several `start:` lines start in order. Its output goes to `start-<n>.log` beside the report and into the report on a failure. Not shown in the video. |
| `setup:` | unchanged: a command that must exit 0, before each feature. |
| `shell:` | the shell for `run` steps (default `$SHELL`, else `sh`; `cmd` / PowerShell on Windows). |

### Steps

| Step | What it does |
|---|---|
| `terminal "Name"` | opens (or switches to) a terminal named Name: a real shell in the script's folder. The video shows it as a code-show window; options after `·`: `window=terminal\|mac\|windows\|none`, `title=` (default the name), `theme=`, `backdrop=`, `size=` (columns×rows, default fits the viewport). |
| `run "command"` | types the command into the current terminal at a human pace (instantly in `si demo test`) and presses Enter. Does not wait for it to finish (a broker never does); the next `expect output` or `wait for` is the wait. `$VAR`s are expanded and blurred like `type $VAR`. |
| `run "command" · wait` | the same, and waits until the command exits (the prompt comes back); fails if it exits non-zero. |
| `press ctrl+c` | in a terminal, sends the key to the shell (stop a program on camera). |
| `expect output "text"` | the current terminal's output (since its last `run`) shows the text, within the timeout (`· within 30s`). On a failure the report shows the terminal's last lines. |
| `expect no output "text"` | the text never appears (e.g. `ERROR`), checked at the end of the block. |
| `wait for output "text"` | like `expect output`, but a pause, not a check. |
| `browser` | switches back to the browser (the app). |
| `split` / `split "Broker"` | the terminal and the browser side by side (terminal left, 40 %); `browser` or `terminal` returns to one scene. Phase 2. |

`note "…" at "Element"` works in a terminal too: `at line "Server startup"`
points at the line that shows that text.

## How it works

- **The terminal scene is a page.** si serves a small page (loopback only, a
  random key, like the `si play` drawer) with the vendored xterm.js
  (`internal/assets/web/term/`) inside a code-window frame drawn by
  `codewin.css` (chrome, title, backdrop, theme). Chrome opens it in its own
  tab; `terminal`/`browser` switch which tab is filmed. The cursor, ripple,
  callouts and captions overlay of `si demo run` apply as on any page.
- **A real shell under a PTY** per terminal, reusing `internal/serve`'s
  terminal hub (`creack/pty`, already a dependency for `si play`, exception
  #8 in CLAUDE.md); the runner keeps a copy of each terminal's output as plain
  text (ANSI stripped) for `expect output`. On Windows, ConPTY if available,
  else a pipe (no colours).
- **Typing** is xterm.js input events at the narration's pace in `run`; in
  `test` it writes the command at once.
- **`start:`** is `exec.Command` in its own process group, output to a log
  file and a buffer scanned for the `wait for` text; stopped by group on exit,
  on Ctrl-C, and on a panic (a deferred stop), so no broker is left running.
- **Test mode** (`si demo test`) runs terminals headless the same way (the
  page still exists, nothing is filmed): `expect output` is a real check, the
  JUnit case gets the terminal's last 40 lines on failure.

## Safety

- Commands come only from the script the user runs, like `setup:` today.
- Terminal pages listen on 127.0.0.1 with a key; never with `--addr 0.0.0.0`.
- Secrets typed from `$VAR` are never written to logs or the report, and are
  blurred in the video; `start:` logs mask them too.

## Phases

1. `start:` (background, wait, stop) and `terminal` / `run` / `expect output`
   / `browser` in `si demo test`, with the report and JUnit.
2. Filmed in `si demo run`: the framed terminal scene, typing at the voice's
   pace, switching scenes, callouts `at line`.
3. `split` (side by side), and a terminal-only demo (no `app:`).

## Docs

RULES.md §27 (the keys and steps above, with the broker example), the
manual, agent rules and the skill, FEATURES.md, the site's Web Demos page
(an example and a picture of the framed terminal scene), CHANGELOG, and an
example script under `examples/webdemo/` that starts a tiny Go "broker" (a
test helper that prints a ready line and serves a page) so CI can run it.
