# Video editing from Markdown (`type: video`)

Decided with the founder, 2026-11-17. Phases 1 and 2 ship in v3.28.0;
phase 3 and the "Not now" items (moving callouts, per-word captions, the
highlight cut) in v3.31.0.

A finished screen recording almost always needs a few touches: a callout
that says what to look at, a blurred password, a pause on the result, a
zoom on a small field, the first seconds cut, a broken stretch replaced or
re-filmed. Today that means a video editor and a timeline by hand. si
already films demos (`si demo run`), draws themed cards and lower thirds and
knows ffmpeg and headless Chrome; this spec makes the touches a Markdown
file that sits beside the video, is checked like any other si file, previews
live in `si play` and builds with `si build`.

## The file

```markdown
---
type: video
source: demo.mp4                 # or a list (below)
out: demo.final.mp4              # default: <source stem>.edited.mp4
theme: studio                    # callouts use the theme's tokens and fonts
fonts: heading=Inter:700         # as elsewhere (optional)
voice: kokoro:af_heart           # for say (optional; else si.conf voice:)
captions: on                     # optional: rebuild the captions
script: demo.md                  # optional: the demo script, for refilm
---
# Install a single node

## Timeline

- 0:00 · title "Install a single node" · 3s · fade
- 0:45–0:50 · blur 120,80,400x60
- 1:34 · bubble "Log in as the admin user" · to 820,410 · 4s · pop
- 1:36–1:46 · replace with fixes/curl-ok.mp4 · fade 0.4s
- 1:52 · freeze 2s
- 2:10–2:30 · speed 2x · badge "⏩ 2×"
- 2:40 · zoom to 600,300 900x500 · 3s · ease
- 3:05 · arrow from 200,700 to 640,420 · "This queue" · 3s
- 3:20 · keys "⌘ K" · 1.5s
- 3:30 · say "Now the queue is replicated." · duck 30%
- 4:00–4:04 · cut
- end · card "Next: clustering" · 3s
```

Several sources join in order, each optionally trimmed, with a transition
between them:

```yaml
source:
  - intro.mp4
  - demo.mp4 · from 0:12 · to 5:40
  - outro.mp4
join: fade 0.5s          # between every two sources (default cut)
```

Every timeline line is `- TIME · action · option · option …`. Text goes in
double quotes. Lines outside the `## Timeline` list (prose, notes) are
ignored, so the file can explain itself.

## Times

- `1:34`, `1:34.5`, `1:02:03`, `94s`, `94.5`; ranges `1:36–1:46` (en dash),
  `1:36-1:46`, `1:36 to 1:46`; `end` (the source's end); `start` (0:00).
- `at chapter "Install" + 5s`: the chapter's start (chapters embedded in the
  mp4, read with ffprobe, or a `chapters.txt` beside it in YouTube form,
  `0:42 Install`), plus or minus seconds.
- `at said "management UI" - 1s`: when that phrase starts being said, from
  the `.vtt` or `.srt` beside the source (si demo films write them with
  `--with subtitles`). Within a cue the phrase's start is interpolated by
  its letter position.
- **All times are source times**: the time you see in the source in any
  player. Edits never shift the times of other lines. Durations (`3s` on a
  callout, `freeze 2s`) are seconds of the output.

### The mapping rule

A source time maps to the moment its frame first shows in the output.

- Before an edit point nothing moves.
- After a `cut A–B` everything is B−A earlier; a time inside the cut maps to
  the join (and `si check` warns: the moment is cut).
- In a `speed Nx` range, A+d maps to out(A)+d/N; after it, everything is
  (B−A)(1−1/N) earlier.
- `freeze Ns` at T holds the frame at T: T maps to the freeze's start (a
  callout at T shows during the freeze); after T everything is N later.
- `insert …` at T plays before the frame at T: T maps after the insert.
- `replace with` over A–B: A+d maps to the replacement's start + d (clamped
  to its length); after B everything shifts by (length − (B−A)).
- Transitions do not change times: the outgoing picture is held for the
  transition's length and the next one fades in over it (as `si demo run`
  joins features). Sound is joined without overlap.

Range items (blur, mute, badges of a speed) map both ends; point callouts
start at the mapped time and last their duration in output seconds.

## Positions

`820,410` source pixels; `60%,40%` of the frame; words `top-left`, `top`,
`top-right`, `left`, `center`, `right`, `bottom-left`, `bottom`,
`bottom-right`. A rectangle is `X,Y WxH` (or `X,Y,WxH`; W and H also take
`%`). `si check` reports a position or rectangle outside the frame.

## Actions

Phase 1 (edits):

| Line | What |
|---|---|
| `A–B · cut` | the range out |
| `T · trim start` / `T · trim end` | everything before / after T out |
| `T · freeze 2s` | hold the frame at T |
| `A–B · speed 2x` | faster (or `0.5x` slower); the sound keeps its pitch (`atempo`), `· mute` silences it; `· badge "⏩ 2×"` shows a badge for the range |
| `A–B · replace with fix.mp4` | the clip, whole, with its own sound, instead of the range; `· keep audio` keeps the original sound and fits the clip to the range (cut, or its last frame held) |
| `T · insert clip x.mp4`, `insert image x.png · 3s`, `insert card "…" · 3s` | played before the frame at T |
| `end · card "…" · 3s` | `card` is `insert card` |
| `· fade [0.4s]`, `dissolve`, `wipe`, `slide` | on replace / insert / card / `join:`: an ffmpeg xfade at both joins |
| `A–B · mute` | the sound off for the range |

Phase 1 (callouts, drawn over the picture):

| Line | What |
|---|---|
| `title "…"` | a large title over a scrim, centred |
| `text "…" · at POS` | text; `· font Inter`, `· size 48` (source px), `· color #fff` or a token (`accent`), `· box`, `· shadow` |
| `bubble "…" · to X,Y` | a speech bubble whose tail points at X,Y; `· round`, `· think` |
| `arrow from X,Y to X,Y · "label"` | an arrow, the label at its tail |
| `box RECT`, `circle RECT`, `underline RECT` | outline a place |
| `spotlight RECT` | darken all but the rectangle |
| `blur RECT`, `pixelate RECT` | hide a place (ffmpeg, not Chrome: no pixel of it leaks) |
| `keys "⌘ K"` | key caps, bottom centre |
| `badge "…"` | a pill, top right |
| `lower-third "Name" "Role"` | the promo kit's lower third |
| `zoom to RECT`, `pan to RECT`, `zoom out` | the camera; `· 3s` holds then returns, without a length it stays until the next camera line; `· ease` (default) or `· linear`; the move takes 0.8 s (`· move 1.2s`) |

Callouts take `· 3s` (default 3 s, a range sets its own), an entrance
`· pop|fade|slide-in|type` (default fade), an exit `· out fade|out none`
(default fade), and `· at POS` where it applies.

Phase 2 (sound and re-filming):

| Line | What |
|---|---|
| `T · say "…" · duck 30%` | new narration with the configured voice (`voice:`, si.conf `voice:`, `--voice`); the original sound drops to 30 % while it speaks (default 30 %) |
| `0:00 · music bed.mp3 · 15%` | music under the whole video, ducked under the speech (the promo kit's sidechain) |
| `captions on` (or `captions: on`) | `<out>.vtt` + `<out>.srt`: the source's captions moved by the edits, plus the `say` lines |
| `A–B · refilm feature "Name"` | re-run that feature of the demo script and splice the new take over the range (`· fade` default 0.3 s) |

## How it is built

1. **Join** (several sources only): each source trimmed, scaled to the first
   one's size, joined with `join:`; kept in `.si/video/<name>.joined.mp4` beside the
   edit (a `.key` file holds the sources' paths, sizes and mtimes). Times refer to this joined source.
2. **Probe** (ffprobe): length, size, frame rate, sound, chapters.
3. **Plan**: the lines become output segments (source pieces at a speed,
   stills, clips, images, cards) and a time map (`internal/videdit/plan.go`).
4. **Callouts**: one page (theme tokens and fonts, the callout CSS, a small
   runtime that draws every callout at time t) opened in headless Chrome at
   the source's size, transparent; one PNG per distinct state (a still
   callout is shot once and linked for every frame it covers, an animated
   entrance is shot per frame) laid out as a 30 fps overlay sequence.
5. **Cards** are the promo kit's cards (`webdemo.CardHTML`), shot at the
   source's size.
6. **Voice**: `say` lines through `webdemo.Synth` (cache, engines).
7. **ffmpeg**, one pass: each segment an input (`-ss/-t`), normalised
   (size, 30 fps, 48 kHz stereo), joined (concat, or xfade with the outgoing
   piece held), then blur/pixelate boxes, the overlay sequence, the camera
   (`scale` with `eval=frame` + `crop`, eased), the mutes and ducks, the said
   lines (`adelay` + `amix`), the chapters moved by the edits. Music is a
   second pass (`webdemo.MusicArgs`).
8. `--with captions` (the captions variant, picture over a band) and
   `--with vertical` reuse the promo kit; `--with highlight` (v3.31.0) joins
   the lines marked `· highlight`, else each chapter's first sentence, with
   the promo kit's highlight join.

### Refilm

The demo script is `script:`, else the source's sibling `<stem>.md` with
`type: demo`, else any `.md` beside it with `type: demo` whose name gives
`<name>.mp4`. si runs only that feature (`webdemo.Run` with `Feature`,
`Video`, `With.Clips`) into a work folder and takes `<feature>.mp4`.

When the edit is refilms only (one source, no callouts, no sound lines), the
rest stays byte-identical: the source is stream-copied up to the last
keyframe before the range (less the fade) and from the first keyframe after
it; only the stretch between is re-encoded (same codec, size, rate), and the
three parts are joined by the concat demuxer with `-c copy` (the head ends
at the keyframe's decode time with a `duration` directive, so no frame of
the next group leaks in). Every packet is copied as it was; keyframes may
gain the parameter sets in band. If the result does not decode cleanly, the
whole video is encoded once instead. Any other edit
re-encodes the whole video once.

## Commands

- `si build edit.md [--with captions,vertical] [--voice …] [-o dir]` writes
  `out:` (relative to the edit file, or into `-o`).
- `si check edit.md`: unknown actions, bad times, positions outside the
  frame, times past the end, missing files, overlapping cuts / replaces /
  speeds / refilms, a callout inside a cut, `music` not at 0:00, refilm
  without a demo script; every problem with its line number. Without
  ffprobe, the checks that need the source's length and size are skipped
  with a note.
- `si play edit.md`: the preview page. The source in a `<video>`, every
  callout as live CSS over it (the same HTML, CSS and runtime the build
  shoots), blur and zoom as CSS, cuts skipped and speeds played at their
  rate while playing, a freeze held; a timeline strip with a marker per line
  (click to seek); live reload on save. Inserts, replaces and refilms are
  markers only: exactness comes from the build.
- A directory build skips `type: video` files with a note (an encode is not
  a page); a directory check checks them.

## Decisions

- **One file, source times.** Editing a line never moves another line's
  time; the build does the arithmetic. Tested in `plan_test.go`.
- **Transitions keep times** (outgoing held), as in `si demo run`.
- **`replace with` takes the clip's own length** (a re-take is rarely the
  same length); `· keep audio` is the same-length fix under the original
  narration.
- **Callouts in Chrome, hiding in ffmpeg.** Callouts are HTML so they follow
  the theme and fonts and match the preview exactly; blur and pixelate are
  ffmpeg filters so the hidden pixels are really gone.
- **Positions in source pixels**, the preview uses percentages and container
  units of the stage, so the page stays fluid (rule #1).
- **The camera after the callouts**: callouts are placed in source pixels
  and zoom with the picture.
- **Music at 0:00 only** (whole video, sidechain-ducked): the promo kit's
  pass, not a second mixer.
- **Done in v3.31.0** (were "not now"): the `highlight` cut for edits
  (`· highlight` lines, else each chapter's first sentence); keyframed
  (moving) callouts (`· to A → B over 2s`, `· path A B C`, eased or
  `linear`, the same speed along every stretch; overlay.js places them per
  frame, the build shoots every frame while they move); per-word captions
  (`captions words`: word stamps in the .vtt, the captions variant a picture
  per word; word times share a caption's time by letters, as `at said`
  does). **Still not now:** ProRes/WebM alpha intermediates (the PNG overlay
  sequence with hard links is smaller and simpler); a moving blur.

## Phases

1. Phase 1 (v3.28.0): the file, times, anchors, every edit, transition and
   callout above, `si build`, `si check`, `si play`, `--with captions,vertical`.
2. Phase 2 (v3.28.0): `say` with ducking, `music`, `captions on`, `refilm
   feature` with the stream-copy splice.
3. Phase 3 (DONE v3.31.0): the Studio's Video view: the preview with the
   timeline strip above the text, callouts dragged into place (writing
   `to X,Y` / `at X,Y` / `from A to B` / rectangles back into the line,
   rectangles resized by their corner; `videdit.Move`, `edit.js`,
   `internal/studio/video.go`, `web/studio/src/video-editor.tsx`), a line
   selected from its marker and the cursor's line marked, ⌘B to build and
   play the result; e2e `web/studio/e2e/videoedit.e2e.mjs`.

## Code

`internal/videdit/` (`parse.go`, `plan.go`, `anchors.go`, `probe.go`,
`graph.go`, `callout.go` + `overlay.js` / `overlay.css`, `build.go`,
`refilm.go`, `preview.go`, `move.go` + `edit.js` / `edit.css` (the Studio's
Video view), `highlight.go`), `cmd/si/videoedit.go`; per-word captions in
`internal/webdemo/words.go`; the promo kit's pieces
exported in `internal/webdemo/videdit.go`.
