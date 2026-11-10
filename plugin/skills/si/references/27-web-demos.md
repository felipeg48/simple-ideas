# 27. Web demos

A demo script (§22) with an `app:` in its front matter is a **web demo**: si
opens the web app in headless Chrome and performs each feature the way a
person would, step by step, checking what the page shows. `si demo test`
runs it as a test, with a report and an exit code for CI; `si demo run`
performs the same steps on film and narrates them: one video per feature, one
with chapters, captions and the report.

```markdown
---
type: demo
title: Northwind Shop, from sign-in to receipt
app: app                # a URL, or the app's folder next to the script
viewport: desktop       # 1440x900; also phone (400x800), tablet (820x1180), WxH
timeout: 5s             # how long a step waits for its element
setup: ./seed.sh        # optional: run before each feature, in the script's folder
voice: kokoro:af_heart  # si demo run: engine:voice (see Voices below)
pronounce: si = ess eye # what the voice says for a word (captions keep the word)
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

- Each `##` heading is a **feature**: one test case, run in a fresh browser
  (no cookies or storage from the one before), after `setup:` and with the
  app's first page open.
- `What it does:` describes the feature (the report's summary). Each `Say:`
  line is the narration of the steps under it, up to the next `Say:`.
  `Notes:` and `Points:` keep their usual meaning and are not steps.
- Steps are list items. A step waits up to `timeout:` for its element or
  condition, then fails with what it expected, what it found and the closest
  elements it saw.

| Step | What it does |
|---|---|
| `go /path` (or a full URL) | open a page of the app and wait for it to load |
| `click "Pay"` | a real mouse click at the element's centre |
| `type "text" into "Field"` · `type $VAR into "Field"` | replace the field's text; `$VAR` comes from the environment and is never printed (`••••` in the report) |
| `press Enter` · `press cmd+k` | a key or a shortcut (alt, ctrl, cmd, shift) |
| `select "Express" in "Shipping"` | an option of a `<select>` (or a listbox) |
| `check "Gift wrap"` · `uncheck …` | a checkbox or switch |
| `hover "Help"` · `scroll to "Footer"` | move the mouse over it; scroll it into view |
| `wait 1s` · `wait for "Saved"` | a pause; or until something appears |
| `note "Username: demo" at "Username"` | a callout in the video: a bubble pointing at the element, ringed, on screen to the end of its `Say:` block (`· for 3s` for less; `· position top\|right\|bottom\|left`, else where it fits; no `at` = a note at the top); a test checks its element is there |
| `expect text "Paid"` | the page shows these words |
| `expect url /orders/4711/receipt` | the address (a path, or a full URL; a `?…` or `#…` after it is fine) |
| `expect "Pay" disabled` | `visible`, `hidden`, `enabled`, `disabled`, `checked`, `unchecked` |
| `expect no errors` | no console errors and no failed requests (HTTP 400 and above) so far |
| `screenshot receipt` | a named picture in the report |

**Elements** are found the way a person names them: `"Pay"` matches visible
text, a `<label>`, an `aria-label`, a placeholder, a title or an alt text
(exact first, then any case, then a part). `#id` and `[data-testid=pay]` are
CSS selectors; `role=button "Pay"` narrows by role. Visible, enabled
elements come first; text inside a button means the button.

```bash
SHOP_PASSWORD=shop-demo si demo test shop.md
si demo test shop.md --feature "Pay for an order" --junit junit.xml -o out/
```

```text
Pay for an order
  ✓ go /  0.06s
  ✓ type "demo" into "Username"  0.01s
  ✓ type $SHOP_PASSWORD into "Password"  0.02s
  ✗ expect text "10% off"  5.07s
      expected: the page shows "10% off"
      found:    the page shows: Northwind Shop ? Order #4711 … That coupon is not valid. …
si demo: 0 passed, 1 failed in 6.9s; report: demo-out/shop/report.html
```

- `si demo test` writes `report.html` (one page, screenshots inlined) and
  `report.json` into `demo-out/<script>/`, exits 1 when a feature failed,
  and with `--junit file.xml` writes JUnit XML (one test case per feature).
  `--viewport phone|tablet|WxH` overrides the script's size (both commands).
- A failed step stops its feature; the report shows expected and found, the
  closest elements, a screenshot, and the page's console errors and failed
  requests.
- `si check` reports a bad step with its line, and warns about a `$VAR` that
  is not set. The script can also be AsciiDoc, or JSON for tools:
  `{"type": "demo", "app": "…", "features": [{"name": "…", "what": "…",
  "blocks": [{"say": "…", "steps": ["go /", "click \"Pay\""]}]}]}`.
- Needs Google Chrome or Chromium (`si doctor`).

### The videos: `si demo run`

```bash
SHOP_PASSWORD=shop-demo si demo run shop.md
si demo run shop.md --feature "Pay for an order" --captions --viewport phone
si demo run shop.md --voice system:Samantha -o videos/
```

The same steps, filmed: a drawn cursor glides to each element and clicks
ripple, `note` steps point at elements, text is typed at a person's pace, and a
field typed from a `$VAR` is blurred. **The narration sets the pace:** every
`Say:` line is voiced first and measured, the block's steps are spread over its
words, and a block whose steps need longer than its words waits for them. A
failing step ends that feature's clip there (in a run a step waits at most 3 s)
and goes into the report; the other features still run.

It writes into `demo-out/<script>/` (or `-o dir`):

| File | What |
|---|---|
| `<feature>.mp4` | the feature, video and voice (H.264 + AAC, 30 fps) |
| `<feature>.video.mp4` | the same video, silent |
| `<feature>.voice.wav` · `.voice.m4a` | the narration alone, to edit or re-voice |
| `<feature>.srt` | captions of the `Say:` lines, timed (always; `--captions` also burns them into the video, bottom, two lines at most) |
| `all.mp4` | every feature, each after a title card (the feature and its `What it does:`), chapters embedded |
| `chapters.txt` | YouTube chapter lines: `0:00 Sign in` |
| `report.html` · `report.json` | the run, each feature linked to its clip |
| `README.md` · `README.html` | the folder: every file, each feature with its video (linked, not inlined), the script, how it was made |

`<feature>` is the heading in lower case with dashes (`pay-for-an-order`).

**Voices.** `voice: engine:name` in front matter, or `--voice`:

| Engine | What | Default voice |
|---|---|---|
| `kokoro` | open source (Apache 2.0), local, natural: si's own voice | `af_heart` |
| `system` | the computer's own: `say` on macOS, espeak-ng on Linux, Windows' voices | the system's (`system:Samantha`) |
| `elevenlabs` | cloud, `ELEVENLABS_API_KEY` in the environment | a voice id (`elevenlabs:21m00Tcm4TlvDq8ikWAM`) |
| `openai` | cloud, `OPENAI_API_KEY`; `gpt-4o-mini-tts`, a tone from `voice-style: calm, friendly` | `alloy` |

Without `voice:`, si uses `kokoro:af_heart` when Kokoro is installed, else
`system`. Kokoro runs as a program: `python3 -m pip install kokoro-onnx` and the
model files `kokoro-v1.0.onnx` and `voices-v1.0.bin` in the folder `si doctor`
names (or `SI_KOKORO_DIR`; `SI_KOKORO_PYTHON` picks the Python). Each line is
cached by its engine, voice, style and words (in the user cache folder), so a
re-run only voices the lines that changed; a cloud call sends nothing but the
`Say:` text.

`pronounce: word = how to say it` (several lines, or pairs split by `;`)
changes what the voice says, whole words in any case; the captions keep the
written word:

```markdown
pronounce: si = ess eye
pronounce: kubectl = cube control; YAML = yammel
```

`si demo run` needs Chrome and ffmpeg, and the voice's engine (`si doctor`).

### Terminals and programs

Many apps are a terminal and a web page together: start a broker, send it a
message, see it in the console. A web demo can start programs in the
background, type commands into a terminal the viewer sees, check what they
print, and switch between the terminal and the browser.

```markdown
---
type: demo
title: A message broker, from the terminal to its console
app: http://127.0.0.1:7310
shell: bash
start: go run ./broker serve --port 7310 as broker · wait for port 7310 · within 120s
setup: go run ./broker reset --port 7310
stop: go run ./broker reset --port 7310
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

## Watch a workload fill the queue

What it does: a load generator sends thirty messages while the console counts them.

Say: A load generator sends thirty messages.
- start "go run ./broker load --count 30" as load · wait for "sending"
- expect load running
- wait for "30 messages waiting"
- stop load · wait
- expect load exited 0
- expect output of load "sent 30 messages"
```

| Front matter | What it does |
|---|---|
| `start: <command> · wait for "text"` | runs the command in the background, in the script's folder, before the first feature, and waits until its output shows the text (or `· wait for port 5672`: a TCP port on 127.0.0.1 accepts; or `· wait 5s`); `· within 60s` gives it longer than 30 s. Several `start:` lines start in order. `as NAME` (after the command) names it for the steps below; else it is `start-1`, `start-2` … Its output goes to `<name>.log` beside the report. When one does not come up, every feature fails with its last lines. Not filmed. |
| `stop: <command>` | a tear-down command run at the end of the run (passed, failed or interrupted), before the programs are stopped, e.g. `make demo-down` |
| `shell: bash` | the shell of the terminals: `sh`, `bash` or `zsh` (default `$SHELL`, else `sh`; `cmd` on Windows) |
| `app: none` | a terminal-only demo: no browser; every feature begins with `terminal` |

| Step | What it does |
|---|---|
| `terminal "Name"` | opens a terminal named Name (or brings it back): a real shell in the script's folder, filmed as a code window (§25). Options after `·`: `window=terminal\|mac\|windows\|none` (default `terminal`), `title="…"` (default the name), `theme=night\|dusk\|paper\|auto` (default `night`), `backdrop=graphite\|ocean\|dusk\|mint\|none` (default `graphite`), `size=100x30` (columns x rows; default: as many as fit) |
| `run "command"` | types the command into the terminal in front (at a person's pace in `si demo run`, at once in `si demo test`) and presses Enter. It does not wait: a broker never ends; the next `expect output` is the wait |
| `run "command" · wait` | the same, and waits until it ends (up to a minute, `· within 5m` for longer); fails when it exits non-zero |
| `run "command" · exit 1` · `run "command" · fails` | a failure on purpose: waits until it ends, and passes only when it exits with that status (`· exit 2`) or with any non-zero one (`· fails`); exit 0 is then the failure. Show what the error says with `expect output "…"` after it |
| `press ctrl+c` | in a terminal, sends the key to the shell: `ctrl+<letter>`, Enter, Tab, Escape, Backspace, arrows, one character |
| `expect output "text"` | the terminal's output since its last `run` shows the text, within `timeout:` (`· within 30s` for longer); white space matches any white space, colours do not count |
| `expect no output "text"` | the text never appears from there to the end of the `Say:` block |
| `wait for output "text"` | like `expect output`, as a pause before what comes next |
| `browser` | brings the app back in front |
| `note "…" at line "text"` | in a terminal, a callout under the line that shows the text, ringed |
| `start "command" as NAME` | runs a program in the background for the rest of the feature (not filmed); `· wait for "text"`, `· wait for port N` or `· wait 3s` before going on |
| `stop NAME` | stops it now; `· wait` waits for it to end on its own instead (a workload of a fixed length) |
| `expect output of NAME "text"` | a background program's output shows the text |
| `expect NAME running` · `expect NAME exited 0` | its state |

**Three lifetimes.** A product demo needs traffic on screen: a load generator
running while the console's charts move. Programs live for the whole run, one
feature, or some blocks:

| Lifetime | How | Example |
|---|---|---|
| the whole run | front matter `start:` (and `stop:` to tear down) | the broker, the cluster |
| one feature | `setup:` (must exit) for state; a `start` step at the top of the feature for a program, stopped when the feature ends | a seed, a steady workload |
| some blocks | `start "…" as NAME` … `stop NAME` | a burst, a second workload |

A program can also be shown: `terminal "Load"` then `run "…"` (no `wait`)
puts its scrolling output on camera, and `browser` goes back to the console
while it keeps running.

- **Stopped, always.** Each program runs in its own process group, so
  stopping it stops what it started too: SIGTERM, then SIGKILL after 5 s.
  That happens when its feature (a `start` step) or the run (a `start:`)
  ends, passed or failed, and on Ctrl-C. A terminal hangs up its shell and
  everything running in it when its feature ends.
- **The terminal scene** is a page si serves on 127.0.0.1 with a random key:
  xterm.js inside the code window frame. It opens in its own tab, and the
  video follows the tab in front; the cursor, clicks, notes and captions of
  `si demo run` work on it as on the app. `si demo test` opens it too
  (nothing is filmed), so `expect output` is a real check either way.
- **si's prompt.** The shell starts without your own rc files: the prompt is
  the folder's name and `$`, history expansion (`!`) is off, and job control
  is off (a program run in a terminal stops with it). Commands are typed as
  written: a `$VAR` is expanded by the shell, so its value is never typed on
  screen, and the values of the `$VAR`s a command uses are masked (`••••`) in
  the logs and the report.
- On a failure the report and the JUnit case show the terminal's (or the
  program's) last 40 lines.
- `si check` follows the scenes: a browser step while a terminal is in front
  (write `- browser` first), a terminal step with no terminal, and a program
  name no `start` gave are problems, with their line.
- On Windows a terminal is `cmd` on pipes (no colours); `run … · wait` and the
  rest work the same.

Examples: `examples/webdemo/` (a small shop, `shop.md`, and `broken.md`,
whose coupon does not work; `broker.md`, a tiny broker in Go started with
`go run`, filmed in a terminal and in its console).
