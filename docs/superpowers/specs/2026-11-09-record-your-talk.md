# Record your talk: narration, timeline, a narrated video

Status: spec, not started. Order: 3 of 5.

## Why

A rehearsal or a live talk is already a timeline: the teleprompter knows
when each slide and build step began. Add the speaker's voice and si can
build what a course or webinar needs: the deck filmed in real motion
(`cmd/si/motion.go`) with the narration in sync. And the same data says
where the talk ran long.

## What the presenter does

```bash
si play talk.md            # p opens the teleprompter; ● Record in its bar
si build talk.md --mp4 --narration .si/recordings/talk-2026-11-09-1830
```

- **Record** (a button in the teleprompter bar, key `r`): asks once for the
  microphone, then records while you present; every slide and step change is
  stamped. Stop with the button. Pause / resume allowed.
- On stop, si play writes `.si/recordings/<talk>-<date-time>/`:
  `audio.webm` (Opus), `timeline.json` (`[{t: 12.31, slide: 3, step: 1}, …]`,
  seconds from the start), `meta.json` (deck path and hash, si version).
- `si build talk.md --mp4 --narration <dir>` films each slide and step for
  exactly as long as it was on screen in the recording (instead of
  `(N min)` / `--dwell`), with the real transitions, and muxes the audio
  (ffmpeg, as today). `--narration latest` picks the newest recording.

## Pace report

`si rehearse talk.md` (or the teleprompter's Rehearse tab) reads a
recording's timeline and prints, per slide: planned (`Notes: (N min)`),
spoken, and the words per minute the `Notes:` script implies at that time:

```text
slide  title                          planned  spoken   wpm
3      Payment: declined, then paid   1:30     2:41     98   ← long
7      The warehouse falls behind     2:00     1:12     231  ← fast
```

No transcription: wpm is the script's word count over the time spent, which
is what a teleprompter reader actually says.

## How it fits si

- **Teleprompter** (`presenter.html`): `MediaRecorder` on
  `getUserMedia({audio:true})`; chunks POSTed every 2 s to
  `POST /_si/record` (loopback only, `localOnly`, like `/_si/callout`), so a
  crash loses at most two seconds. Step changes it already sees (`state`)
  become timeline entries.
- **Server** (`internal/serve/record.go`): appends chunks, writes the
  timeline and meta on stop; one recording at a time per deck; refuses on the
  remote listener.
- **Video** (`cmd/si/video.go`, `motion.go`): a `narration` pacing source next
  to `--dwell` and `(N min)`; ffmpeg mux `-c:a aac`; the video length follows
  the audio. Decks changed since the recording (meta hash) get a warning, and
  slides added later get the default dwell.
- **Where the mic works**: `http://127.0.0.1` is a secure context, so the
  laptop's teleprompter can record; a phone teleprompter (`--remote`) can
  record only over `--https`.

## Rules check

2. The recording is a file beside the deck, never in a built deck.
3. RULES.md §11 (the Record button), the `--narration` flag, `si rehearse`.
5. `Notes:` drive the wpm estimate; nothing changes in the format.
7. The teleprompter stays a teleprompter: Record is one button in its bar.
8. Stdlib (ffmpeg optional, as for `--mp4` today).

## Privacy

Audio stays on the laptop, in the deck's `.si/recordings/`; nothing is
uploaded; the teleprompter shows a red dot while recording. `.si/` should
not be committed: si does not write a `.gitignore` today (review comments in
`.si/review/` have the same issue), so phase 1 adds `.si/` to the starter
`.gitignore` of `si new` and the Studio's New dialog, and `si play` warns once
when a recording sits in a git work tree that does not ignore it.

## Tests

- Go: chunk append and resume, timeline from a scripted state stream, the
  narration pacing (durations from a fixture timeline), the pace report.
- e2e: Chrome with a fake microphone (`--use-fake-device-for-media-stream`)
  records 10 s across three slides; the build's MP4 length and the slide
  timings checked with ffprobe.

## Phases

1. Record (teleprompter, endpoint, files).
2. `--mp4 --narration` (pacing + mux).
3. `si rehearse` pace report.

## Open questions

- Also record the camera (picture-in-picture)? (Later; the same pipeline.)
- Keep several takes of one slide and pick the best? (Later.)
