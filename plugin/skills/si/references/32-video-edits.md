# 32. Video edits

A recording almost always needs a few touches: a callout that says where to
look, a password blurred, a pause on the result, a zoom on a small field,
the first seconds cut, a broken stretch replaced or filmed again. A Markdown
file with `type: video` lists them beside the video; `si check` checks it,
`si play` previews it, `si build` writes the finished video (ffmpeg and
headless Chrome, as for `si demo run`).

```markdown
---
type: video
source: demo.mp4
out: demo.final.mp4
theme: studio
---
# Install a single node

## Timeline

- 0:00 · title "Install a single node" · 3s · fade
- 0:45–0:50 · blur 120,80,400x60
- 1:34 · bubble "Log in as the admin user" · to 820,410 · 4s · pop
- 1:52 · freeze 2s
- 2:10–2:30 · speed 2x · badge "⏩ 2×"
- 2:40 · zoom to 600,300 900x500 · 3s
- 4:00–4:04 · cut
- end · card "Next: clustering" · 3s
```

**Front matter.** `source:` the video (or a list, below); `out:` where the
result goes, beside the file (default `<source>.edited.mp4`; `-o dir` puts
it in `dir`); `theme:` and `fonts:` for the callouts and cards, as elsewhere;
`voice:` for `say`; `captions: on`; `script:` the demo script `refilm` runs;
`join:` the transition between sources.

**Lines.** Every line of the `## Timeline` list is `TIME · action · option ·
…`, text in double quotes; anything outside the list (prose, other lists
under other headings) is ignored, so the file can explain itself.

**Times are source times**: what any player shows for the source. `1:34`,
`1:34.5`, `1:02:03`, `94s`; ranges `1:36–1:46`, `1:36-1:46`, `1:36 to 1:46`;
`start`, `end`. An edit never moves another line's time: si works out where
each moment lands. The rule: a source time maps to the moment its frame
first shows in the output. After a cut everything is earlier; in a speed
range the time runs faster; a freeze at T shows frame T for its length (a
callout at T shows during it); an insert at T plays before frame T; a time
in a cut lands at the join (`si check` warns). Durations (`3s`, `freeze 2s`)
are seconds of the output.

**Anchors** instead of a clock: `at chapter "Install" + 5s` (the chapters in
the mp4, or a `chapters.txt` beside it: `0:42 Install`) and `at said
"management UI" - 1s` (when the words are said, from the `.vtt` or `.srt`
beside the source; `si demo run --with subtitles` writes one).

```markdown
- at chapter "Users" + 2s · keys "⌘ K"
- at said "one message ready" · bubble "Here it is" · to 1210,480
```

**Positions** are source pixels (`820,410`), percentages (`60%,40%`) or a
word: `top-left`, `top`, `top-right`, `left`, `center`, `right`,
`bottom-left`, `bottom`, `bottom-right`. A rectangle is `X,Y WxH` (or
`X,Y,WxH`; `W` and `H` take `%` too). `si check` reports any outside the
frame.

### Edits

| Line | What |
|---|---|
| `A–B · cut` | the range out |
| `T · trim start` · `T · trim end` | everything before / after T out |
| `T · freeze 2s` | hold frame T |
| `A–B · speed 2x` | faster (`0.5x` slower); the sound keeps its pitch; `· mute` silences it; `· badge "⏩ 2×"` shows a badge for the range |
| `A–B · replace with fix.mp4` | the clip, whole, with its own sound, instead of the range; `· keep audio` keeps the original sound and fits the clip to the range |
| `T · insert clip intro.mp4` | a clip before frame T |
| `T · insert image shot.png · 3s` | a picture |
| `T · insert card "Part two" · 3s` (or `card "…"`) | a title card in the theme |
| `A–B · mute` | the sound off |

A replace, insert or card takes a transition, `· fade [0.4s]`, `· dissolve`,
`· wipe`, `· slide`, at both of its joins; the outgoing picture is held while
the next fades in, so transitions never move times.

```markdown
- 0:03 · trim start
- 1:36–1:46 · replace with fixes/curl-ok.mp4 · fade 0.4s
- 2:00 · insert image diagrams/cluster.png · 4s · dissolve
- 2:30–2:34 · mute
- 5:58 · trim end
```

Several sources join in order, each trimmed with `· from` / `· to`:

```markdown
---
type: video
source:
  - intro.mp4
  - demo.mp4 · from 0:12 · to 5:40
  - outro.mp4
join: fade 0.5s
---
```

Times then refer to the joined video (kept in `.si/video/` beside the file
until a source changes).

### Callouts

| Line | What |
|---|---|
| `title "…"` | a large title over a darkened picture |
| `text "…" · at top-right` | text; `· font Inter`, `· size 48` (source pixels), `· color #fff` or `accent`, `fg`, `bg`, `muted`, `surface`; `· box` (on the theme's surface), `· shadow` |
| `bubble "…" · to 820,410` | a speech bubble whose tail points at the place; `· round`, `· think` |
| `arrow from 200,700 to 640,420 · "This queue"` | an arrow, its label at the tail |
| `box RECT` · `circle RECT` · `underline RECT` | outline a place |
| `spotlight RECT` | darken everything else |
| `blur RECT` · `pixelate RECT` | hide a place (done by ffmpeg: the pixels are gone, not covered) |
| `keys "⌘ K"` | key caps, bottom centre |
| `badge "Live"` | a pill, top right |
| `lower-third "Ada Lovelace" "Engineer"` | a name and a role, bottom left |

A callout shows 3 s (`· 4s`, or a range); it comes in with `· fade` (the
default), `· pop`, `· slide-in` or `· type` (the text typed; an arrow draws
itself), and leaves with a fade (`· out none`: at once). `· at POS` moves
`text`, `keys`, `badge` and `lower-third`. Callouts are drawn in the theme's
colours and fonts and scale with the frame, a phone-shaped recording too.

```markdown
- 0:02 · lower-third "Ada Lovelace" "Platform engineer" · 4s · slide-in
- 0:20 · text "Three nodes, one queue" · at top · box · type
- 0:31 · bubble "Leader moved" · to 62%,48% · think · 3s
- 0:40 · arrow from 200,700 to 640,420 · "This queue" · 3s
- 0:44 · box 120,80 400x60 · 2s · out none
- 0:44 · circle 900,500 160x160 · pop
- 0:47 · underline 120,140 400x8
- 0:50–0:56 · spotlight 600,300 500x200
- 1:02–1:10 · pixelate 40,20 300x40
- 1:20 · keys "⌘ ⇧ P" · 1.5s
- 1:25 · badge "Live" · at top-left · 5s
```

### The camera

`zoom to RECT` moves the camera onto the rectangle (its shape fitted to the
frame's) in 0.8 s (`· move 1.2s`), eased (`· linear` for a steady move);
with `· 3s` it holds that long and comes back, without a length it stays
until `pan to RECT` or `zoom out`. Callouts sit in source pixels, so they
zoom with the picture.

```markdown
- 2:40 · zoom to 600,300 900x500
- 2:46 · pan to 100,300 900x500 · move 1.2s
- 2:52 · zoom out
```

### Sound, captions and a new take

| Line | What |
|---|---|
| `T · say "…" · duck 30%` | new narration in the configured voice (`voice:`, `--voice`, si.conf `voice:`); the original sound drops to 30 % while it speaks |
| `0:00 · music bed.mp3 · 15%` | music under the whole video, ducked under the speech |
| `captions on` (or `captions: on`) | `<out>.vtt` and `<out>.srt`: the source's captions moved by the edits, plus the `say` lines |
| `A–B · refilm feature "Name"` | run that feature of the demo script again and splice the new take over the range |

```markdown
- 3:30 · say "Now the queue is replicated." · duck 20%
- 0:00 · music bed.mp3 · 12%
- 1:36–1:46 · refilm feature "Ops terminal" · fade
- captions on
```

`refilm` finds the script as `script:`, else the source's sibling
`<name>.md` with `type: demo`, else the one demo script beside it, and runs
only that feature, filmed as `si demo run` would. When the refilm is the
only line, the rest of the video is copied as it was (stream copy up to the
keyframe before the range and from the one after it); only the stretch
between is encoded again.

### Commands

- `si check edit.md`: unknown actions, bad times, anchors not found, times
  past the end, positions outside the frame, missing files, overlapping
  cuts, replaces, speeds and refilms, callouts inside a cut, `music` not at
  0:00, a refilm without its script or feature; every problem with its line.
- `si play edit.md`: the source playing with the callouts, blur and the
  camera drawn live over it, cuts skipped, speeds and freezes played, a
  strip with a marker per line (click to jump); it reloads when you save.
  Inserted and replaced clips, cards and sound are markers; the build is
  exact.
- `si build edit.md [--with captions,vertical] [--voice engine:name] [-o dir]`:
  the video, and with `--with` the captions variant (`.captions.mp4`, the
  picture over a band) and the 9:16 cut (`.vertical.mp4`).
- A folder build skips video edits (`si build edit.md` makes one); a folder
  check checks them.
