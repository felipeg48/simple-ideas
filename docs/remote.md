# Remote: teleprompter, draw pad and clicker on another device

`si play --remote` lets a tablet or phone on the same network act as the
teleprompter (`tp`), as a pen pad (`draw`) or as a clicker (`clicker`) for the
deck you project from the laptop.

```bash
si play talk.md --remote            # tp + draw + clicker, LAN port = play port + 1
si play talk.md --remote=tp         # only the teleprompter
si play decks/ --remote --remote-addr 0.0.0.0:9000
```

`si play` prints one URL and QR code per role (for a folder: one picker URL),
and `q` on the presenting deck shows the same codes full screen (the links go
only into the loopback page: they carry the key). When the machine has more
than one usable address (Wi-Fi, Ethernet, Tailscale's 100.64/10), the first is
used and the others are printed, and listed under the codes.
The role links are short (`http://<LAN address>:7071/clicker?k=<key>`, which
redirects to the deck in that role) so the QR code stays small enough for a
phone camera to read from the terminal.
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

## The audience (`--audience`)

`si play talk.md --audience` adds a second key to the remote listener (it
starts the listener when `--remote` is not given). `RemoteHandler` admits the
presenter key (cookie `si_remote`) to the presenter routes as before, and the
audience key (cookie `si_audience`) to `audienceHandler` only: `/a` (a 302 to
the deck with `?role=audience`, or a picker for a folder), the deck rendered
with `NoSpeakerText` (no `Notes:` / `Points:` asides), static files, and
`GET /_si/remote/events` through `audienceView`, which passes `state`
messages cut down to slide, step, count and title, and drops everything else
(notes, points, previews, ink, commands). An audience page cannot POST to the
relay. The runtime role `audience` follows `state` like the draw pad, ignores
navigation and every command but full screen and light/dark, and gets no
wake-lock hint. `<!-- qr: audience -->` is an empty, hidden figure that the
presenter's loopback page fills from `SI_REMOTE_QR.audience`.

### Polls

`<!-- poll: id · a · b … -->` renders `.si-poll` (one `.si-poll-opt` per
option, `--_share` 0–100 sizing its `.si-poll-bar`). An audience phone POSTs
`{poll, choices, voter}` to `/_si/audience/vote` (audience mux only;
`serve/poll.go`): ids and voter ids are checked, at most 6 distinct choices,
one change per voter per 300 ms, 5000 voters per poll, 100 polls; an empty
ballot withdraws. Each ballot publishes `{type: poll, id, counts, total}` to
every page through the relay (kept as `poll:<id>` for late joiners);
`audienceView` passes it. Counts and bars show only where a room can vote
(`data-live`: an audience phone, or the deck of `si play --audience`).

### Remote audiences (`--tunnel`)

`si play --tunnel` (implies `--audience`; `cmd/si/tunnel.go`) serves
`Server.AudienceOnlyHandler` (the audience key and routes only) on a loopback
port and runs `cloudflared tunnel --no-autoupdate --url http://127.0.0.1:<port>`,
reading the `https://<words>.trycloudflare.com` address from its output
(40 s timeout); the printed link, the `q` overlay and `<!-- qr: audience -->`
use it. Quick tunnels do not carry server-sent events: audience pages that
get nothing on the stream within 4 s poll `GET /_si/audience/now` (the last
state and poll messages through `audienceView`) every 1.2 s. Streams send
`Cache-Control: no-store, no-transform`, `X-Accel-Buffering: no` and a
heartbeat comment every 20 s. `si play` stops cloudflared on exit
(SIGINT, then kill after 3 s). A new quick-tunnel name can take ~10 s to
resolve in DNS.

### Questions

`POST /_si/audience/ask {text, name, voter}` (audience mux; `serve/qa.go`):
text tidied and cut to 500 characters, name to 40, one question per voter
per 20 s, 200 open questions at most. The list goes out as `{type:
questions, list}` (kept as `questions` for late presenter pages);
`audienceView` drops it, so phones never see it. `POST /_si/qa {id, state}`
(open, shown, answered, hidden; one shown at a time) is on the loopback
handler (`localOnly`) and the presenter mux, never the audience's. The deck
passes the list on over BroadcastChannel to a teleprompter window on the
same computer, shows the `shown` one as `.si-qa-show`, and Esc sets it back
to open. On exit `SaveAudience` writes the counts and questions to
`<served dir>/.si/audience/<talk>-<date-time>.json`.

## Keeping the device awake

`RelayJS` also asks for a screen wake lock (`navigator.wakeLock`) on a role
page (`?role=tp|draw|clicker`), and takes it again whenever the page is shown
or touched. Browsers grant it only in a secure context (`localhost` or
HTTPS); a phone at `http://<LAN address>` has none, so the page shows a short
hint instead: Settings › Display & Brightness › Auto-Lock › Never. With
`--https` (below) the phone gets the lock.

## HTTPS (`--https`)

```bash
si play talk.md --remote --https             # si's own certificate authority
si play talk.md --remote --https=tailscale   # a real certificate for machine.tailnet.ts.net
```

**`--https`** (`internal/serve/tls.go`). si keeps a certificate authority of
its own in `<config dir>/si/ca/` (macOS: `~/Library/Application
Support/si/ca/`; `$SI_CA_DIR` overrides), made on first use: an ECDSA P-256
key (file mode 0600) and a 10-year certificate whose **name constraints**
allow only private and Tailscale addresses (10/8, 172.16/12, 192.168/16,
100.64/10, 127/8, 169.254/16) and `.local` names. It cannot vouch for a
public site, so even a leaked key cannot fake your bank. Each run signs a
7-day certificate for the machine's current addresses (the shape Apple
requires: names in the SAN, serverAuth, well under 825 days).

The remote port speaks both: a connection that opens with a TLS handshake
gets HTTPS (`SplitTLS` peeks at the first byte), any other gets plain HTTP,
which serves only the one-time setup (`/ca`: the steps, `/ca.crt`: the
certificate) and moves everything else to HTTPS. One port, one set of codes.

On each phone, once: scan the "first time" code (also on the `q` overlay),
download the certificate, then:

- **iPhone / iPad:** Settings › Profile Downloaded › Install, then Settings ›
  General › About › Certificate Trust Settings › turn on full trust.
- **Android:** open the file, or Settings › Security › Encryption &
  credentials › Install a certificate › CA certificate.

Installing the profile is not enough on an iPhone: until full trust is on
(the second step), Safari still says "This Connection Is Not Private". Apple's
trust engine accepts si's certificates (checked with `security verify-cert`,
the same evaluation iOS runs), so that warning means the switch is off.

On the laptop itself, `si ca trust` adds the authority to the login keychain
(macOS asks for your password; Safari and Chrome read it, Firefox keeps its
own list), so the remote pages open as secure there too; `si ca untrust`
takes it out, and `si ca` shows where it is and its fingerprint. On Windows it
runs `certutil -user`; on Linux it prints the commands (they need root).

Compare the SHA-256 fingerprint the phone shows with the one `si play`
prints before trusting it: the download is plain HTTP, so on a network you
do not trust, that comparison is what makes it safe. Remove it later in
Settings › General › VPN & Device Management (iPhone) or Trusted credentials
› User (Android); delete the folder to make a new authority.

**`--https=tailscale`** (`cmd/si/tailscale.go`): with Tailscale running on
the laptop and the phone, si asks `tailscale cert` for this machine's
MagicDNS name (`machine.tailnet.ts.net`, a Let's Encrypt certificate through
Tailscale; cached in `<cache dir>/si/tailscale/`). Nothing to install on the
phone; the tailnet needs MagicDNS and HTTPS certificates on (admin console ›
DNS).

## Security

- The remote listener requires a random 128-bit key per run: the printed URL
  carries `?k=`, the first response sets it as an `HttpOnly`, `SameSite=Strict`
  cookie, every other request without it gets 403.
- It serves the decks (always without the terminal drawer), files under the
  deck directory (no dotfiles), the relay and live reload, and the role
  picker. Roles not enabled with `--remote=` return 403.
- Plain HTTP (no `--https`): anyone who can read your network traffic can
  read the key and the notes. Use a network you trust, your own hotspot, or
  `--https`.
- The deck window stays on loopback, so the terminal drawer stays available
  and is never exposed.
