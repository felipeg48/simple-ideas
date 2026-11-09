# 22. Demo scripts

`type: demo` is the presenter's run sheet for a live part of a session: a
document (built like a lab) **and** the content of the ▶ Demos panel of the
`si play` terminal (a deck lists it with `demos:`, §14).

````markdown
---
type: demo
title: "Live: order #4711 meets a slow bank"
session: The Thursday Promise
slides: 6 (Payment: declined, then paid)
duration: 6 min
---

# Live: order #4711 meets a slow bank

## What this demo must accomplish
1. **A timeout is not a decline.** The room sees the bank answer late.

## Before the session (T-30 min)
```bash
docker compose up -d --wait
```

## The script
### Slide 6, "Payment: declined, then paid" (4 min)
> Say: "I am going to make the bank slow, the way it is every night at nine."
```bash
curl -X POST localhost:8081/admin/latency -d '{"ms": 25000}'
```
```text
{"ms": 25000}
```
Point at the latency while the room waits.

## If it goes wrong
## Reset between rehearsals
````

- The header carries an **INSTRUCTOR ONLY** badge, and `session`, `slides`
  and `duration` fill a facts block under the title.
- **`> Say:` is what the presenter says to the room, word for word** (the
  teleprompter rule): spoken sentences, never directions. It renders as a
  speech callout. Directions to the presenter ("point at the latency") are
  plain text. Any other quote stays a quote.
- Shell blocks (`bash`, `sh`, `zsh`, `console`, `sql`, `powershell`) are
  commands: Copy / Run in the ▶ Demos panel. `text` (or `output`) blocks are
  what the room should see: labelled "expected output". Follow every command
  block with its expected output.
- Script steps are `### Slide N, "title" (M min)` (or any `###` with
  `(M min)`); `si check` warns when they do not add up to `duration:`.
- The usual sections: *What this demo must accomplish*, *Before the session
  (T-30 min)*, *The script*, *If it goes wrong*, *Reset between rehearsals*.
- `si build` builds a demo like a lab (DEMO badge on its index tile);
  `si build --no-instructor` leaves every demo out, for the copy you hand to
  participants. The ▶ Demos panel shows the script without its front matter.

Example: `si --export example --type demo`; the showcase deck lists it, so
`si --export example && si play si-example` shows it in the ▶ Demos panel.
