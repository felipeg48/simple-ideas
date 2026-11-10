# Audience mode: follow along, live polls, Q&A

Status: phase 1 (follow along, `qr: audience`) DONE in v2.16.0; phases 2
(polls) and 3 (Q&A) next. The flag is `--audience` (front matter
`audience:` not added). The audience page also renders without
`Notes:`/`Points:` (`NoSpeakerText`), not only a filtered stream.

## Why

si already puts a phone in the presenter's hand (`--remote`: teleprompter,
clicker, draw pad over a keyed, optionally HTTPS relay). The same relay can
put the deck in the audience's hands: their phone follows the talk, they vote
in a poll and the bars grow on the projected slide, they send a question and
it lands on the teleprompter. Markdown deck tools stop at the slides; none do
this self-contained, with no service in the middle.

## What the presenter writes

```markdown
---
title: The Thursday Promise
audience: on            # or: si play talk.md --remote --audience
---

## Join in

<!-- qr: audience · caption=Follow along, vote, ask -->

---

## Which retry rule would you ship?

<!-- poll: retry · Once, after 30 s · Three times, backing off · Never: tell the customer -->

Notes: (1 min)
Take out your phone. Which one would you ship? [CLICK] Here is how the room voted.
```

- `<!-- qr: audience -->`: the special target `audience` becomes the audience
  link of this run (`https://<lan>:7071/a?k=<audience key>`), so the code on
  the slide is always the live one. In a built deck (no si play) it renders
  nothing and `si check` warns.
- `<!-- poll: id · option · option … -->`: a poll with a stable `id` (results
  are keyed by it) and 2–6 options. Options: `· multiple` (pick several),
  `· hidden` (results stay hidden until the presenter reveals them),
  `· step` (the results reveal on the next click, a build step).
- Q&A needs no markup: it is on whenever audience mode is.

## What the audience sees (role `audience`)

A phone page, light or dark, no install:

- **Follow**: the current slide, rendered like the clicker's preview (the
  same `snapshot` the teleprompter uses), updating on every step; pinch to
  zoom. No Prev/Next: the audience never drives the deck.
- **Vote**: when the current slide has a poll, its options as big buttons; one
  vote per phone per poll (a random id in localStorage; changing the vote
  replaces it). Results show after voting unless `hidden`.
- **Ask**: a text box (500 characters), optional name; "sent" with no echo.
- Nothing else: no notes, no Points:, no draw, no other slides ahead.

## What the presenter sees

- **Projected slide**: the poll's bars grow live (count and percent), in the
  theme's colours; `hidden` shows "Voting…" with a count until revealed.
- **Teleprompter**: a Q&A tab (beside Teleprompter / Speaker Notes): questions
  in arrival order, each with Show on screen, Answered, Hide. Show puts the
  question over the slide (a lower third), Esc clears it.
- **`q` overlay / menu-bar app**: the audience code joins the role codes.

## How it fits si

- **Roles**: `serve.RemoteRoles` gains `audience`. It gets its own key
  (`audienceKey`, separate from the presenter's), so a phone in the room can
  never become the teleprompter or the clicker: the remote gate checks the
  key against the role (today one key opens every role).
- **Relay messages** (`internal/serve/remote.go`): `vote {poll, choice[],
  voter}` and `ask {text, name}` from audience pages; the server keeps the
  tallies and the question list (in memory, per run; `.si/audience/<talk>-<date>.json`
  written at the end, see Privacy); `poll {id, counts, total}` and
  `question {id, text, name, state}` go out. Audience pages receive only
  `state` (slide, step, snapshot) and `poll`: never notes, ink, or questions.
- **Limits**: votes and questions are rate-limited per voter id and per IP
  (e.g. 1 vote change / s, 1 question / 20 s, 200 open questions); bodies
  capped; HTML escaped everywhere (questions are untrusted text).
- **Rendering** (`internal/render/plugins.go`, `deck/`): `poll` is a body
  directive on slides and in documents (in a document: a static list of the
  options, "vote at the talk"); `.si-poll` with one `.si-poll-bar` per option
  sized by a `--_share` custom property the runtime sets; tokens
  `--si-poll-bar`, `--si-poll-track`.
- **HTTPS**: works with and without `--https`; audience mode on a network you
  do not control should use `--https` (the page says so when it is plain HTTP).

## Rules check

1. Fluid: bars and the lower third use `cqi`/`cqb`; the phone page reflows.
2. Self-contained: nothing fetched; built decks show polls as static lists.
3. RULES.md: `audience:`, `poll`, `qr: audience`, the role, with examples;
   the showcase gets a poll slide as content ("which rule would you ship").
4. Content, not instructions, in examples.
5. `Notes:` stays spoken; a poll slide's `[CLICK]` reveals the results.
6. Mechanics in layout.css, looks in theme.css, tokens/hooks in contract.go.
7. The teleprompter stays a teleprompter: Q&A is a tab, never over the script.
8. Stdlib + goldmark only.

## Privacy

- No accounts, nothing leaves the laptop. Voter ids are random per phone.
- Names on questions are optional; the saved results file (`.si/audience/`)
  keeps counts and questions without IPs, and only when the talk ends
  normally. `audience.save: off` in si.conf keeps nothing.

## Tests

- Go: key separation (the audience key opens `audience` only), tallies (one
  vote per voter, changes replace), rate limits, escaping, audience pages
  never receive notes or ink.
- Render: the poll directive (golden), the static document form.
- e2e (Playwright): a deck with a poll, three audience pages voting, the
  projected bars and the teleprompter Q&A tab, light and dark, 4 sizes.
- A real run with phones in the room (the founder).

## Phases

1. Follow along (role, key, snapshot) + `qr: audience`.
2. Polls (directive, tallies, bars, `hidden`, `step`).
3. Q&A (ask, the teleprompter tab, the lower third), saved results.

## Open questions

- Should audience pages show the speaker's `Points:`? (Proposed: never.)
- A cap on the room size? (Proposed: none, but the relay already fans out
  per subscriber; test with 200 pages.)
- Word clouds and free-text polls: later.
