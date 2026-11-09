# Two (or more) presenters

Status: spec, not started. Order: 5 of 5.

## Why

Many talks have two voices: a host and a demo driver, a pair of teachers.
Each needs only their own script on their teleprompter, to know when the
other one takes over, and control of the deck when it is their turn.

## What the author writes

```markdown
---
title: The Thursday Promise
speakers: Ana, Felipe
---

## Payment: declined, then paid

Notes[Ana]: (1 min)
The token reaches payments and turns red. [CLICK]

Notes[Felipe]: (0.5 min)
Here is the fix we shipped: one retry, thirty seconds later. [CLICK]

Points[Felipe]:
- Show the retry log if asked
```

- `speakers:` declares the names (one deck, many voices).
- `Notes[Name]:` / `Points[Name]:` belong to that speaker; plain `Notes:`
  belongs to the first speaker (so every existing deck still works).
- Order on a slide is speaking order: Ana, then Felipe.
- `si check`: a name not in `speakers:`, a speaker with no lines in the whole
  deck, `[CLICK]` counts per slide summed across speakers.

## What each presenter sees

- **Teleprompter** with `?speaker=Ana` (or a picker in the teleprompter bar):
  only Ana's Notes/Points; where Felipe speaks, a quiet line "Felipe: 0:30"
  with his first words, so Ana knows her cue and when she is next.
- **Hand-over**: when the script moves to the other speaker, both
  teleprompters show a small "→ Felipe" banner for a few seconds.
- **Remote**: `si play talk.md --remote --speakers` prints one teleprompter
  code per speaker (`/tp?speaker=Ana&k=…`); the `q` overlay too.
- **Timers**: per speaker and total; the pace report (record spec) splits by
  speaker.
- **Control**: every presenter's clicker/teleprompter can drive the deck
  (today's behaviour); an optional `speakers.control: turn` in si.conf lets
  only the speaker whose turn it is advance (the other's Next is greyed).

## How it fits si

- Parser (`internal/deck`): `Slide.Notes` becomes a list of
  `{Speaker, Minutes, Text}` (one entry for decks without speakers, so
  render and the terminal keep working); `Points` likewise.
- Teleprompter (`presenter.html`): a `speaker` filter over the same state
  messages; the snapshot already carries notes HTML, it gains per-speaker
  parts.
- Terminal (`--notes`): `--speaker Ana`.
- Studio: the inspector shows notes per speaker; completion knows `speakers:`.

## Rules check

5. `Notes[Name]:` is still the spoken script, verbatim, `[CLICK]` included;
   `Points[Name]:` the speaker's own list. The agent guide and RULES.md §11
   get the syntax with an example.
7. The teleprompter stays a teleprompter: one speaker's script, others as cues.

## Tests

- Parser table cases (`Notes[Ana]:`, plain `Notes:` with speakers, unknown
  names, durations per speaker), golden render, `si check` click counts.
- e2e: two teleprompter pages (Ana, Felipe) on one deck: each shows its own
  script, the cue line and the hand-over banner at the right step.

## Open questions

- Colour per speaker in the teleprompter? (Proposed: the speaker name in the
  accent, no extra colours.)
- More than two speakers: supported by design; test with three.
