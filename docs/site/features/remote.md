---
title: Present from your phone
description: Your phone or tablet as the clicker, the teleprompter or a pen pad, while the laptop projects the deck. Over HTTPS after a one-time setup, so the phone stays awake.
---

# Present from your phone

The laptop projects the deck. Your phone, in your hand, is the **clicker**
(big Next and Prev, where you are, what comes next, a stopwatch). An iPad on
the lectern is the **teleprompter** (your script at your pace, timers,
previews). A tablet with a pen is the **draw pad** (what you draw shows on the
projected slide). They all follow the deck, and each can drive it.

```bash
si play talk.md --remote --https
```

`si play` prints a QR code per role. Press **q** on the projected deck and
the same codes fill the screen, big enough to scan from the room.

## The one-time setup

Do this once per laptop and once per phone or tablet. After that, every talk
is: run `si play talk.md --remote --https`, scan the code.

**On the laptop: nothing.** The first `--https` run makes a certificate
authority for this laptop (in `~/Library/Application Support/si/ca/` on a
Mac). It lasts ten years, its key never leaves the laptop, and it can only
vouch for local addresses: it can never be used to fake a public website.
Every run signs a fresh certificate for whatever addresses the laptop has
that day, so a new venue, Wi-Fi or address needs nothing redone.

**On each phone or tablet, once:**

1. Press **q** on the deck and scan **First time on this phone: trust si**
   (it is also printed in the terminal).
2. Tap **Download the certificate** and allow it.
3. iPhone and iPad: **Settings › Profile Downloaded › Install**.
   Android: open the downloaded file, or **Settings › Security ›
   Encryption & credentials › Install a certificate › CA certificate**.
4. iPhone and iPad only, and easy to miss: **Settings › General › About ›
   Certificate Trust Settings** › turn on **si local CA (your laptop)**.
   Without this switch Safari still says *This Connection Is Not Private*.

Before step 4, compare the fingerprint the phone shows with the one the
terminal prints (`si ca` prints it any time). They must match.

**Optional, on the laptop:** `si ca trust` adds the same authority to the
laptop's own trusted certificates (macOS asks for your password), so the
laptop's browser shows the phones' `https://` links as secure too. You do not
need it to present: the deck on the laptop is `http://127.0.0.1:7070`, which
browsers already treat as secure.

Each laptop has its own authority, so a phone that trusts your laptop trusts
no one else's. Present from a second laptop, or set up a new phone, and you
do the one-time setup there once.

## Every talk

```bash
si play talk.md --remote --https        # every role
si play talk.md --remote=tp,clicker --https
```

1. Open the deck on the laptop (the URL `si play` prints) and project it.
2. Press **q**, scan the Teleprompter or Clicker code, press **q** again.
3. Present. The phone keeps its screen on, and the notes travel encrypted.

The code is new every run: the link carries a key made for that run, and a
page without it gets nothing.

## Your audience too

`--audience` puts the talk on the room's phones as well: they follow the
slides, vote in polls and send you questions; `--tunnel` does the same on a
video call. See [Your audience](audience.md).

## Draw with the cursor

Press `a` (draw), then **hold ⌥ Option and move**: the cursor draws where it
goes, no click needed, so a trackpad draws with one finger. Let go of Option
to end the stroke. Another key suits you better? Put it in `si.conf` next to
the deck:

```text
draw.hold: shift      # alt (⌥ Option, the default), shift, ctrl, meta (⌘ Command), or off
```
 It only works in draw mode, in any browser and any deck
(`si play`, a built deck, offline), and the strokes travel to the
teleprompter and a phone's draw pad like any other.

## With Tailscale

If the laptop and the phone are both on [Tailscale](https://tailscale.com),
there is nothing to install on the phone:

```bash
si play talk.md --remote --https=tailscale
```

si asks Tailscale for a real certificate for the laptop's name
(`laptop.your-tailnet.ts.net`). Your tailnet needs MagicDNS and HTTPS
certificates turned on (admin console › DNS).

## Without HTTPS

`si play talk.md --remote` works without `--https`, over plain HTTP: fine
on your own hotspot or a network you trust. A phone then cannot keep its own
screen on, so the page reminds you to set **Auto-Lock** to **Never** for the
talk, and anyone who can read the network can read the notes.

The same `--https` (or `--https=tailscale`) also covers the audience link
(`--audience`) and a review link (`si serve . --review --https`), where
colleagues leave comments on your slides.

## When something is off

| You see | Why, and what to do |
|---|---|
| *This Connection Is Not Private* on the iPhone | the trust switch (step 4) is off; turn it on, close the tab, scan again |
| the camera reads the code, the page never loads | the phone is on another network, or the Wi-Fi keeps devices apart (guest and office networks often do); use your own hotspot, or the other address listed under the codes |
| *ERR_SSL_PROTOCOL_ERROR* at `https://localhost:7070` | the laptop's own port is plain HTTP on purpose: open `http://127.0.0.1:7070` |
| the laptop's browser says *Not secure* on the phone links | run `si ca trust` once (Firefox: import the certificate in its settings) |
| a new phone, a reset phone, another laptop | the one-time setup, there, once |

To remove it: on the iPhone, **Settings › General › VPN & Device
Management**; on Android, **Trusted credentials › User**; on the laptop,
`si ca untrust`. Delete the `ca` folder to make a new authority.

## The rules

- [11. Speaker notes and the teleprompter](../rules/11-speaker-notes-and-the-teleprompter.md)
- [14. Presenting: keys, touch, overview, print](../rules/14-presenting-keys-touch-overview-print.md)
