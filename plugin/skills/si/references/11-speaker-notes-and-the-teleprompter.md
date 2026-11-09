# 11. Speaker notes and the teleprompter

```markdown
Notes: (1.5 min)
The token reaches payments and turns red: the bank declined the card.
Thirty seconds later one retry goes through, and the order is paid.

Points: What to cover, not a script
- Payments goes red: the retry kicks in after 30s
- Pause after the red tile; this is the moment to remember
```

- A line starting with `Notes:` starts the slide's **verbatim teleprompter
  script**: the words the speaker says to the audience, word for word, about
  what is on the slide. Write it as speech, never as instructions to the
  speaker: no "pause here", "mention X", "tell them", "press next", "this slide
  shows". Everything after it, to the end of the slide (or a `Points:` line),
  is the script (markdown).
- **`[CLICK]` cues (recommended on slides with build steps or animations).**
  Write `[CLICK]` in the script where the speaker advances: the next build
  step, a canvas or stage change, the next slide. It is the one mark in
  `Notes:` that is not read aloud: the teleprompter shows it as a ▶ CLICK
  badge, so the speaker knows when to press. Put one per build step, and at
  most one more for moving on to the next slide; `si check` warns when a slide
  has fewer cues than steps, or more than steps + 1.

  ```markdown
  Notes: (1 min)
  The token reaches payments and turns red. [CLICK] Thirty seconds later one
  retry goes through. [CLICK] And the payment tile turns green.
  ```
- A line starting with `Points:` starts the slide's **speaker-notes talking
  points**: the main points of the topic as short bullets, to glance at if
  you'd rather talk than read. Reminders for the speaker (pace, emphasis,
  likely questions, what to skip) belong here, not in `Notes:`. Both
  blocks are optional and may appear in either order; neither swallows the other.
- `(N min)` on the `Notes:` line is the slide's planned time; the deck total is
  the sum, shown in the index, the build summary and the teleprompter.
- Both are hidden on the slides.

**Teleprompter (`p` or `Shift+S`).** A second window, synced with the
audience window over `BroadcastChannel`; it works in `si play` and in the
static HTML (`file://`), and either window drives both. It has two tabs:
**Teleprompter** (the verbatim `Notes:` script, with auto-scroll) and
**Speaker Notes** (the `Points:` talking points).

| In the teleprompter | |
|---|---|
| Tabs | `1` Teleprompter (script), `2` Speaker Notes (points); choice remembered per browser |
| Notes | large type for reading at a distance; a guide line at reading height (teleprompter tab only) |
| Auto-scroll | `a` on/off, `+` / `-` speed 1 to 10 (lines per ten seconds, follows the text size) |
| Manual scroll | ↑ ↓, mouse wheel, trackpad; `r` back to the top (scrolls whichever tab is showing) |
| Text size | `[` / `]` (both tabs) |
| Mirror | `m` flips the script for teleprompter glass |
| Notes only | `x` hides the previews, the prompt fills the window |
| Previews | the current slide (with its step state) and the next step or slide |
| Clocks | time on this slide against its `(N min)` (amber when over, red at 1.5×), total against the deck plan, ahead/behind pace, wall clock |
| Rehearsal | the time you spent on each slide is recorded per run; next to the slide clock it shows `last M:SS` — your previous run's time for this slide, to pace against; and the whole talk against it: `vs last run 0:40 behind` (works without any `(N min)` plan) |
| Timer | `t` pause/resume, `0` reset (promotes this run to the `last` reference and starts fresh) |
| Navigation | → ← Space PgDn PgUp Home End, buttons |

Settings (tab, size, speed, mirror, notes only) are remembered per browser.

**On another device (`si play --remote`).** Put the teleprompter on an iPad
(or any tablet or phone) and project from the laptop. `--remote` opens a
second listener on your network, next to the loopback one, and prints one
keyed URL and QR code per role:

```bash
si play talk.md --remote                    # every role
si play talk.md --remote=tp                 # teleprompter only
si play talk.md --remote --remote-addr 0.0.0.0:9000
```

Press `q` on the deck (the laptop's window, the one you project) to show the
same links as QR codes, big enough to scan from the room; `q` or Esc closes
them (`keys.remote` in si.conf remaps it). When the laptop is on more than
one network (Wi-Fi and Ethernet, or a VPN such as Tailscale), the other
addresses are listed under the codes.

`si play talk.md --trackpad` (macOS) makes the Mac's trackpad a pen: the pad
maps to the whole display the deck is on, a finger shows a dot, and holding
⌥ Option while a finger moves draws (the cursor stays put meanwhile). It
needs the `si-trackpad` helper next to `si`; see `docs/remote.md`.

`--https` serves the remote pages over HTTPS, so the phone keeps its screen
on and the notes are private on any network. si makes its own certificate
authority once (it can only vouch for local addresses) and each phone trusts
it once: scan the "first time on this phone" code, install the certificate,
and turn on full trust (iPhone: Settings › General › About › Certificate
Trust Settings; without that switch Safari still warns). `si ca trust` does
the same on the laptop. With Tailscale on the laptop and the phone,
`--https=tailscale` uses a real certificate for the machine's `ts.net` name
instead, with nothing to install. See `docs/remote.md`.

| Role | On the device | Drives the deck |
|---|---|---|
| `tp` | the teleprompter above: script, tabs, auto-scroll, clocks, previews, mirror; on a phone the script fills the screen (no previews, one clock line, big Prev / Next) | Prev / Next, keys |
| `draw` | the current slide with the pen always on (Apple Pencil pressure works); Prev / Next in the pen bar | the strokes appear on the projected slide live; Prev / Next |
| `clicker` | a phone remote: big Next and Prev, the slide and step (`2 / 22 · step 1/4`), its title, what comes next, and a stopwatch (tap it to reset) | Prev / Next |

- The deck on the laptop stays on `127.0.0.1`, so the terminal drawer keeps
  working. The remote listener (default: every interface, the play port + 1)
  serves the decks, their files and the sync, never the terminal.
- Pages sync through the server (server-sent events and small POSTs), the
  deck stays the source of truth, and a device that connects late jumps to
  the current slide. Ink is shared both ways in slide coordinates and clears
  when the slide changes, like the local pen (`a`). A phone that slept or
  left the browser reconnects when it is shown again and jumps to the
  current slide.
- A device page keeps its screen on (the browser's wake lock) where the
  browser allows it: on `localhost` or HTTPS. Over plain HTTP on the LAN
  (an iPhone or iPad at `http://192.168.…`) it cannot, so the page says so
  for a few seconds: set Auto-Lock to Never for the talk.
- The URL carries a random per-run key; the first visit stores it as a
  cookie, and anything without it gets 403. It is plain HTTP: use it on a
  network you trust.
- With a folder of decks the QR code opens a picker (deck × role).
- Play only: a built deck never contains any of it. See `docs/remote.md`.


**Teleprompter in a tab or frame.** Opening a deck page with `?si-view=tp`
(for example `talk.html?si-view=tp`) turns that page into the teleprompter of
the same deck open in another tab, window or frame of the same browser,
synced like the `p` window. `si --web` uses it for its Rehearse and Lab modes.
`?si-view=thumb` shows one slide as a still picture (every build step shown,
no sync, no live reload); the Studio storyboard uses it for its thumbnails.
