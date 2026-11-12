# 16. Configuring keys and features (si.conf)

An optional `si.conf` next to the deck (or `--conf <path>`) sets project-wide
defaults without editing each deck. It uses the same `key: value` style as
front matter — no new syntax. Blank lines and `#` comments are ignored.

| Key | Effect |
|---|---|
| `theme: <name>` | default theme when a deck sets no `theme:` and there is no `--theme` |
| `keys.<action>: <key>` | remap a command key |
| `feature.<name>: off` | turn a feature off (its key stops working and its toolbar button is hidden) |
| `handout.paper: letter` | the paper of `si build --handout` PDFs: `a4` (default) or `letter` |
| `draw.hold: shift` | the key you hold in draw mode to draw with the cursor, no click: `alt` (⌥ Option, the default), `shift`, `ctrl`, `meta` (⌘ Command), or `off` |
| `links.<label>: <url>` | a page outside the build that find (`/`) offers, e.g. `links.Team wiki: https://wiki.example.com`; relative paths work too |
| `timer.time: 10,15,30` | break-timer presets (default `5,10,15,20,30,45,60`) |
| `timer.unit: seconds` | preset unit: `minutes` (default) or `seconds` |
| `timer.face: analog` | countdown face: `digital` (default) or `analog` (sport-clock dial) |
| `timer.animation: waves,orbit` | animation(s) behind the timer HUD: `waves`, `orbit`, `bokeh`, or custom names; a comma list gives a toggle |
| `timer.animation.path: ./anim` | directory of custom `<name>.js` / `<name>.svg` animations |
| `voice: kokoro:af_heart` | the voice of `si demo run` and `si voice say` when the script and `--voice` set none (§27) |
| `voice.style: calm, friendly` | the tone, for engines that take one (`openai`), when the script sets no `voice-style:` |
| `demo.with: subtitles, chapters` | what `si demo run` writes besides `<script>.mp4` when `--with` and the script's `with:` say nothing (§27) |
| `voice.Ada: kokoro:af_heart` | auto-play: one speaker's voice in a deck with `speakers:` (front matter `voices:` wins) |
| `autoplay.dwell: 5` | auto-play: seconds on a slide with no `Notes:` and no `(N min)` (default 3) |

Nesting is written as dotted keys (`timer.animation.path`), not YAML.

**Your own si.conf.** Besides the project's, si reads one si.conf of yours in
the user config folder: `~/Library/Application Support/si/si.conf` on macOS,
`~/.config/si/si.conf` on Linux, `%AppData%\si\si.conf` on Windows. It is
read first, so the project's file wins key by key; put there what follows you
from project to project (a theme you like, `voice:`).

It is also the only place for **API keys**: `voice.elevenlabs.key:` and
`voice.openai.key:`. A project si.conf goes to git with the decks, so a key in
it is ignored, and `si check` warns with the file and line. Do not write them
by hand: `si voice login openai` asks for the key without echo and writes it
there with the file readable by you only (si warns when others can read it);
`si voice logout openai` removes it. A key in the environment
(`OPENAI_API_KEY`, `ELEVENLABS_API_KEY`) wins over the file; si never prints
one.

```
# <config dir>/si/si.conf — yours, not the project's
theme: slate
voice: kokoro:bf_emma
voice.openai.key: sk-…        # written by si voice login openai
```

Remappable **actions**: `overview`, `presenter`, `draw`, `timer`, `find`,
`mode`, `fullscreen`, `help`, `terminal`, `remote`, `autoplay`. Toggleable **features**: `draw`, `timer`,
`presenter`, `overview`, `find`, `autoplay`.

```
# si.conf — flat key: value, same style as front matter (not YAML)
theme: slate
keys.draw: w          # draw with w instead of a
draw.hold: shift      # in draw mode, hold Shift (not Option) to draw with the cursor
feature.timer: off    # no break-timer HUD for these decks
```

Run `si --export conf` to print a starter `si.conf` with every key and feature
at its default. Theme precedence: `--theme` flag > a deck's `theme:` front
matter > `si.conf` > the built-in default (`studio`). `si play` re-reads
`si.conf` on every request, so edits show up on reload. An unknown key or a bad
value is a warning printed to the terminal (with the file and line), never a
failed build.
