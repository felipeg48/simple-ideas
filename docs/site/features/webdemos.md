---
title: Web demos
description: One Markdown script performs each feature of your web app in a browser, the way a person would, and reports what passed and what broke.
---

# Web demos

A web demo is a demo script with an `app:`. Each `##` is a feature of your
web app; under it, the steps a person would take, and what the page should
show. `si demo test` performs them in headless Chrome and tells you, step by
step, what passed and what broke; `si demo run` performs them on film and
narrates them, one video per feature and one with chapters.

```markdown
---
type: demo
title: Northwind Shop, from sign-in to receipt
app: app                # a URL, or the app's folder next to the script
viewport: desktop
---

## Pay for an order

What it does: a customer picks Express shipping and pays with a card.

Say: Sign in with the demo account.
- go /
- type "demo" into "Username"
- type $SHOP_PASSWORD into "Password"
- click "Sign in"
- expect text "Welcome back, demo"

Say: A test card goes through in a second.
- click "Order #4711"
- select "Express" in "Shipping"
- check "Gift wrap"
- click role=button "Pay"
- type "4242 4242 4242 4242" into "Card number"
- click "Confirm"
- wait for "Paid"
- expect url /receipt.html?order=4711
- expect no errors
```

Steps name elements the way you would say them: `click "Sign in"` finds the
button that reads Sign in, `type … into "Username"` the field labelled
Username. When the text is not enough, `#id`, `[data-testid=pay]` and
`role=button "Pay"` are there too. Clicks are real mouse clicks at the
element's centre, typing goes through the keyboard, and every step waits a
few seconds for the page to catch up before it gives up.

Secrets stay out of the script: `$SHOP_PASSWORD` comes from the environment
and shows as `$SHOP_PASSWORD` (or `••••`) in every line si prints.

## Run it

```bash
SHOP_PASSWORD=shop-demo si demo test shop.md
si demo test shop.md --feature "Pay for an order" --junit junit.xml
```

```text
Pay for an order
  ✓ type "demo" into "Username"  0.01s
  ✓ type $SHOP_PASSWORD into "Password"  0.01s
  ✓ press Enter  0.06s
  ✓ click "Order #4711"  0.07s
  ✓ select "Express" in "Shipping"  0.00s
  ✓ check "Gift wrap"  0.01s
  ✓ expect text "Total: €54.90"  0.00s
  ✓ click "Confirm"  0.07s
  ✓ wait for "Paid"  0.61s
  ✓ expect url /receipt.html?order=4711  0.00s
  ✓ expect no errors  0.00s
si demo: 3 passed, 0 failed in 4.6s; report: demo-out/shop/report.html
```

Every feature runs in a fresh browser, after the script's `setup:` command if
it has one, so one feature never leans on what the last one left behind.
`si demo test` exits 1 when anything failed, and `--junit` writes the JUnit
XML that GitHub Actions, GitLab and Jenkins read.

## When a step fails

The feature stops there, and the report says what the step expected, what
it found, the elements that came closest, and shows the page at that moment,
with the console errors and failed requests up to then.

![The report of a failed run: six steps passed, then expect text "10% off" failed; it shows the page's words, and a screenshot of the order with "That coupon is not valid."](img/webdemos/report.jpg)

`report.html` is one page with the screenshots inside, so it travels as an
attachment or a CI artifact; `report.json` has the same for your own tools.

## The script

- Markdown, AsciiDoc, or JSON for tools (the same features, blocks and
  steps).
- `app:` is a URL, or a folder: si serves the folder itself on this computer
  only, so a static app needs no server at all.
- `viewport:` `desktop` (1440×900), `tablet`, `phone` or `1280x720`;
  `timeout:` how long a step waits (5 s by default).
- `si check` knows every step, points at a bad one by its line, and warns
  when a `$VAR` is not set.

The example is a small shop in `examples/webdemo/`: `shop.md` signs in, pays
for an order and searches the order list; `broken.md` tries a spring coupon
that the shop does not know.

## On film

```bash
SHOP_PASSWORD=shop-demo si demo run shop.md --captions
```

The same steps, filmed in a browser you never see. A drawn cursor glides to
each button and a ripple marks the click; `note "Username: demo" at
"Username"` puts a bubble beside the field until the narration moves on; text
is typed at a person's pace, and a field typed from a `$VAR` is blurred.

![A frame of the sign-in video: the Username field ringed, a note "Username: demo" beside it, and the caption "Northwind keeps your orders behind one sign-in."](img/webdemos/film.jpg)

**The words set the pace.** Every `Say:` line is voiced before the take and
measured; the steps under it are spread over its words, and a block whose
steps need longer waits for them, so there is no dead air and no voice
talking over the wrong screen. A step that fails ends that feature's clip
there and goes into the report.

In `demo-out/shop/`:

| File | What |
|---|---|
| `pay-for-an-order.mp4` | the feature, video and voice |
| `pay-for-an-order.video.mp4` | the same, silent |
| `pay-for-an-order.voice.m4a` · `.wav` | the narration alone |
| `pay-for-an-order.srt` | captions, timed |
| `all.mp4` | every feature after a title card, chapters inside |
| `chapters.txt` | `0:00 Sign in` lines for a YouTube description |
| `report.html` · `report.json` | the run, linked to the clips |
| `README.md` · `README.html` | every file, each video in the page, the script, how it was made |

`--captions` burns the captions into the video as well (bottom, two lines at
most); `--viewport phone` films the phone layout; `--feature "Pay for an
order"` films one feature.

## With a terminal

Many apps are a terminal and a web page together: start a broker, send it a
message, see the message in its console. The script can start programs in
the background, type commands into a real terminal on camera, check what
they print, and go back to the browser.

```markdown
---
type: demo
title: A message broker, from the terminal to its console
app: http://127.0.0.1:7310
start: go run ./broker serve --port 7310 as broker · wait for port 7310 · within 120s
---

## Send a message from the terminal

What it does: a client sends one message, and the console shows it waiting.

Say: From a terminal, we send the broker one message.
- terminal "Client" · backdrop=ocean
- run "go run ./broker send hello from the terminal" · wait
- expect output "sent 1 message"
- note "One message, sent" at line "sent 1 message"

Say: The broker's console shows it, waiting in the queue.
- browser
- expect text "1 message waiting"
```

`start:` runs the broker before the first feature and waits until its port
answers (or until its output shows a text, `· wait for "ready"`). `terminal
"Client"` opens a shell in the script's folder, framed like a code window:
`window=mac`, `theme=dusk`, `backdrop=ocean`, `title="…"`, `size=100x30`.
`run` types the command at a person's pace and presses Enter; `· wait` waits
for it to end and fails on a non-zero exit. `expect output` checks what it
printed since, colours aside; `press ctrl+c` stops a program on camera;
`note "…" at line "…"` points at a line.

Sometimes the failure is the point: `run "si check bad.md" · exit 1` passes
only when the command exits 1 (`· fails`: any non-zero status), and an
`expect output "problem"` after it shows the error the viewer should read.
A command that succeeds there fails the step.

![A frame of a terminal scene: a Mac-style window on a dusk backdrop, "go run ./broker serve --port 7311" typed, the line "broker ready" ringed with a note "Ready for messages" under it, and the caption "A second broker starts in a terminal of its own, on port 7311."](img/webdemos/terminal.jpg)

A workload for one feature is `start "go run ./broker load" as load · wait
for "sending"`; `stop load · wait` lets it finish, `expect load exited 0` and
`expect output of load "sent 30 messages"` check it, and the console's
counter climbs on screen meanwhile. Programs live for the whole run
(`start:`, with `stop:` to tear down), one feature (a `start` step), or a few
blocks (`start` … `stop`).

Nothing is left behind: every program runs in its own process group and is
stopped (SIGTERM, then SIGKILL after 5 s) when its feature or the run ends,
passed or failed, and on Ctrl-C. Its output goes to `<name>.log` beside the
report, secrets masked; a failure shows the terminal's last 40 lines in the
report and in JUnit. The example `examples/webdemo/broker.md` starts a tiny
broker written in Go with `go run`, so CI runs it too.

## Voices

```markdown
voice: kokoro:af_heart
pronounce: si = ess eye
```

| Engine | What |
|---|---|
| `kokoro` | open source and local, natural: si's own voice, `af_heart` (the default when it is installed) |
| `system` | the computer's own: `say` on macOS, espeak-ng on Linux, the voices Windows has |
| `elevenlabs` | cloud, `ELEVENLABS_API_KEY`; `elevenlabs:<voice id>` |
| `openai` | cloud, `OPENAI_API_KEY`; `gpt-4o-mini-tts`, with `voice-style: calm, friendly` |

`pronounce:` changes what the voice says (Kokoro reads "si" as "see"); the
captions keep the written word. Every line is cached by its words and voice,
so a re-run only voices the lines that changed, and a cloud voice is sent
nothing but the `Say:` text. To install Kokoro: `python3 -m pip install
kokoro-onnx`, then its two model files in the folder `si doctor` names.

<!-- si-deps: chrome ffmpeg kokoro espeak-ng -->

## The rules

- [27. Web demos](../rules/27-web-demos.md)
- [22. Demo scripts](../rules/22-demo-scripts.md)
