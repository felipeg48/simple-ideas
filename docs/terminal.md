# Terminal drawer — design note (PINNED, not built yet)

A `si play` presenter terminal: a drawer HUD that slides up from the bottom to
~mid-screen, for running **local** commands live during a demo. Decision on the
exact implementation is **pinned** — this note captures the requirements so we
don't lose them.

## Requirements (from the founder)

- Run local shell commands during a talk without leaving the deck, e.g.
  - `docker ps` — one-shot, prints and exits.
  - `cockroach start-single-node --insecure` — **long-running** server; must keep
    running and stream logs; must be stoppable (Ctrl-C / kill).
  - `cockroach sql --insecure --host …` — **interactive REPL**; needs live stdin.
- **Multiple tabs** per terminal — several concurrent sessions/processes (e.g.
  tab 1 runs the node, tab 2 runs `cockroach sql`, tab 3 runs `docker ps`).
- Local only for now. Nothing remote (yet).
- "we are adding features to that terminal" — treat it as an extensible subsystem.

## How execution works

`si play` is a Go server, so it `fork+exec`s each command as a child process on
the presenter's machine (`exec.Command(name, args…)`), wires stdin/stdout/stderr,
and streams I/O to the drawer in the browser over localhost. A one-shot command
captures output and returns; a long-running/interactive command keeps the child
alive and streams. Each **tab** is its own tracked child process.

This is inherently a **`si play`-only** feature: a built, self-contained `.html`
(rule #2) has no server, so the terminal is inert/hidden there.

## Security (non-negotiable)

- **Loopback only.** The exec endpoint is enabled only when the server is bound
  to a loopback address; it is **disabled on `--addr 0.0.0.0`** (phone/QR mode),
  so it can never become LAN remote-code-execution.
- Likely also an opt-in feature flag (`feature.terminal`) and a dedicated key.

## Decision (founder, locked)

**Option 2: full interactive terminal — xterm.js front-end + a PTY-backed shell.**
This deliberately **relaxes non-negotiable rules #7 (stdlib + goldmark only)** and
**#2 (self-contained)** for this subsystem: it adds `xterm.js` (bundled) and a PTY
Go dependency (e.g. `creack/pty`). Justified because the founder needs a *true*
interactive shell for demos (`cockroach sql` prompt, `start-single-node` logs with
Ctrl-C, full-screen apps). The terminal remains **`si play`-only + loopback-only**,
so built decks stay self-contained and the capability never ships remotely.
When built, update rule #7 in CLAUDE.md to carve out this play-only exception.

## Open decisions (to settle when we build)

1. ~~Front-end: vanilla vs xterm.js~~ → **xterm.js** (decided).
2. ~~PTY: none vs PTY~~ → **PTY** (decided; e.g. `creack/pty`).
3. **Transport:** Go stdlib has no WebSocket. Options: pull in a small WebSocket
   dep (natural now that rule #7 is relaxed) vs SSE (output) + POST (input) vs
   HTTP chunked streaming both ways. Multi-tab = one stream per tab.
4. **Process lifecycle:** leave-running vs auto-kill on tab close / deck close;
   SIGINT (Ctrl-C) handling; working-directory + env persistence per tab.

## Minimal viable first cut (if we go stdlib, no PTY)

- `si play` tracks a map of tab-id → child process; `/_si/term/<tab>` streams
  output, a POST writes a line to the child's stdin, a signal endpoint sends
  SIGINT/kill. Working dir persists per tab.
- Drawer UI in `deck.js`: tab bar (+ new tab, close tab), output pane, input line
  with history; toggled by a key; only enabled when the server injects the
  terminal-enabled flag (loopback play only).
