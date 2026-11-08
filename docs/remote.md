# Remote: teleprompter, draw pad and clicker on another device

`si play --remote` lets a tablet or phone on the same network act as the
teleprompter (`tp`), as a pen pad (`draw`) or as a clicker (`clicker`) for the
deck you project from the laptop.

```bash
si play talk.md --remote            # tp + draw + clicker, LAN port = play port + 1
si play talk.md --remote=tp         # only the teleprompter
si play decks/ --remote --remote-addr 0.0.0.0:9000
```

`si play` prints one URL and QR code per role (for a folder: one picker URL).
Scan it on the iPad; open the deck on the laptop as usual
(`http://127.0.0.1:7070/`) and project that window.

## How it fits together

```
 laptop (loopback :7070)                 iPad (LAN :7071, keyed)
 ┌───────────────────────┐   relay       ┌─────────────────────────┐
 │ deck (audience)       │◀──────────────│ ?role=tp  teleprompter  │
 │  posts state          │──────────────▶│ ?role=draw pen pad      │
 │  obeys cmd, draws ink │  SSE + POST   └─────────────────────────┘
 └───────────────────────┘
```

- **Relay** (`internal/serve/remote.go`): `POST /_si/remote/send` takes one
  JSON message; `GET /_si/remote/events` streams everyone else's (the sender
  is skipped by its client id). The last `state` per deck is replayed to a
  new subscriber, so a late device jumps to the current slide. A page that
  was hidden for more than 5 s (a phone that slept) reopens its stream when
  shown again, and so gets that state too. Both listeners
  share it; the loopback one needs no key.
- **Messages**: `state` (deck → all: slide, step, notes, previews, mode),
  `cmd` (next, prev, first, last, hello → deck), `ink` / `erase` / `clear`
  (any page → all; points in 0–1 slide coordinates, width as a fraction of
  the slide width, batched every 40 ms). Each carries the deck name and a
  message id (a page on the deck's own origin hears BroadcastChannel and the
  relay; the id dedupes).
- **Pages**: the same deck HTML. `?role=tp` replaces the page with the
  teleprompter (the same `presenter.html`, with the relay instead of
  BroadcastChannel; on a phone, portrait or landscape, the script fills the
  screen: no previews, a compact clock line, big Prev / Next first in a
  scrolling button row); `?role=draw` follows the deck's state, keeps the pen on
  and turns Prev / Next into commands; `?role=clicker` covers the slide with
  big Next / Prev buttons that send commands and shows the deck's state
  (position, title, the next label) and a stopwatch.
- **Transport code** (`window.SI_RELAY`, `render.RelayJS`) is injected only by
  `si play --remote`, like the live-reload client. A built deck contains none
  of it, and the local presenter window keeps using BroadcastChannel (so it
  still works from `file://`).

## Keeping the device awake

`RelayJS` also asks for a screen wake lock (`navigator.wakeLock`) on a role
page (`?role=tp|draw|clicker`), and takes it again whenever the page is shown
or touched. Browsers grant it only in a secure context (`localhost` or
HTTPS); a phone at `http://<LAN address>` has none, so the page shows a short
hint instead: Settings › Display & Brightness › Auto-Lock › Never.

## Security

- The remote listener requires a random 128-bit key per run: the printed URL
  carries `?k=`, the first response sets it as an `HttpOnly`, `SameSite=Strict`
  cookie, every other request without it gets 403.
- It serves the decks (always without the terminal drawer), files under the
  deck directory (no dotfiles), the relay and live reload, and the role
  picker. Roles not enabled with `--remote=` return 403.
- Plain HTTP: anyone who can read your network traffic can read the key and
  the notes. Use a network you trust (or your own hotspot).
- The deck window stays on loopback, so the terminal drawer stays available
  and is never exposed.
