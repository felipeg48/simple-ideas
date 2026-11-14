# Results webhooks: courses that report, labs that check themselves, CI that tells the team

Status: decided with the founder on 2026-11-14. Parts 1, 2, 3 and 5 built
(v3.19.0); part 4 (the live class view) built, unreleased
(`internal/serve/class.go`, `class.html`).

## Why

A course built with si keeps its progress in the learner's browser, and in
an LMS through SCORM. A trainer running a class, a team lead onboarding new
hires, or a company running a course from its own site wants to *see* the
results without an LMS: in Slack, in Teams, in a spreadsheet behind a Zap,
in a learning record store. The same goes for a CI run of web demos: the
team wants a line in its channel, not only a JUnit file.

The rule that shapes everything: **a built course is a static page**. It has
no server, works from `file://`, in a website bundle and inside SCORM. So
the page sends small events itself, the way analytics beacons do, and a
server is only needed for the things a page must never hold (a signing
secret, a command to run).

## Part 1: result events from courses

### Settings (`_course.md`)

```markdown
---
title: A first web page
webhook: https://hooks.example.com/si/class-42
webhook-format: json            # json (default), slack, teams, discord, xapi
webhook-events: quiz.submitted, course.completed   # default: all five
webhook-token: c42-7f3a         # json only: sent as "token" in every event
webhook-learner: ask            # random (default) or ask: the learner types a name
webhook-consent: required       # notice (default) or required
webhook-notice: Your quiz scores go to your trainer, Ana.
---
```

- `webhook:` the address events go to. `https://` (si check warns on any
  other scheme), or `relay` (see Part 3: the page posts to the si server it
  came from, `/_si/relay`, and the URL stays off the page).
- si.conf `course.webhook: https://…` is the default for every course in the
  project; a `_course.md` `webhook:` wins.
- `webhook-auth:` (xapi only) the `Authorization` header value for a learning
  record store, `Basic dXNlcjpwYXNz`. It is in the page: use a write-only LRS
  key, or the relay.
- Unknown `webhook-*` keys are unknown front matter keys (a problem); unknown
  events, formats, learner or consent values are warnings; `webhook:` in a
  page that is not `_course.md` is a warning (lessons inherit the course's).

### Events

| Event | When | `data` |
|---|---|---|
| `lesson.completed` | a lesson is done (read to the end, video at 90 %, lab boxes ticked, quiz passed) | `{}` |
| `quiz.submitted` | Check my answers | `score` (0-100), `pass` (bool), `passMark`, `attempt`, `questions: [{id: "q1", correct: true}]`, never the answers |
| `lab.step.checked` | a lab's ✓ Check ran (si play) | `step` (the check's number), `title` (its `##` step), `pass`, `exit`, `ms` (never the output) |
| `demo.watched` | a lesson video ends, or the page is left after watching some | `percent` (0-100), `seconds` |
| `course.completed` | the last lesson that counts is done | `lessons` |

Each is sent once per occurrence; `lesson.completed` and `course.completed`
once per learner (the same browser).

### The JSON schema (`webhook-format: json`, `version: 1`)

```json
{
  "version": 1,
  "event": "quiz.submitted",
  "id": "e-5f0c9a6b2d41e8f7",
  "time": "2026-11-14T10:02:11.418Z",
  "si": "3.18.0",
  "audience": "learner",
  "course": { "id": "web101", "title": "A first web page", "url": "https://learn.example.com/web101/index.html" },
  "lesson": { "id": "02-html/quiz.html", "title": "Check yourself", "kind": "quiz", "url": "https://learn.example.com/web101/02-html/quiz.html" },
  "learner": { "id": "l-3f9a0c71b2e4", "name": "Ana" },
  "data": { "score": 80, "pass": true, "passMark": 70, "attempt": 2,
            "questions": [ { "id": "q1", "correct": true }, { "id": "q2", "correct": false } ] },
  "token": "c42-7f3a"
}
```

- `id` is random per event: a receiver drops duplicates by it (the offline
  queue may send twice).
- `course.id` is the course folder's name; `lesson.id` the lesson's page
  path inside the course. `course.completed` has no `lesson`.
- `learner.id` is random (`l-` and 12 hex digits), made once and kept in
  localStorage (`si-learner`), so one learner is one id across the courses of
  a site; `name` only with `webhook-learner: ask`.
- `audience` is `instructor` for the instructor build (`si play`,
  `--instructor`), so a receiver can ignore the trainer's own clicks.
- `token` only when `webhook-token:` is set. It is visible in the page: it
  tells your receiver which class an event belongs to, it is not a secret.
- A new field may be added in version 1; a field changes or goes only with
  version 2.

### Sending

- `json` and `slack`: `navigator.sendBeacon` with a `text/plain` JSON body,
  so there is no CORS preflight and the event leaves even as the page
  closes. Slack's incoming webhooks read a text/plain JSON body.
- `teams`, `discord` and `xapi`: `fetch(…, {keepalive: true})` with
  `application/json` (and for xapi `X-Experience-API-Version: 1.0.3` and
  `Authorization` from `webhook-auth:`). These need the receiver to allow
  CORS; when it does not, use `webhook: relay` and `si serve`.
- A small queue in localStorage (`si-webhook-queue`, at most 100 events)
  keeps what could not leave (offline, a failed fetch, a refused beacon) and
  sends it again on the next page, when the browser comes back online.
- In a SCORM package the events go out as well as the LMS calls.

### The notice and consent

A short bar at the bottom of the course: "This course sends your progress
(lessons done, quiz scores) to its organizers." (`webhook-notice:` replaces
the text), with **OK**. With `webhook-learner: ask` it has a name field.

- `webhook-consent: notice` (default): events are sent; the bar stays until
  **OK**.
- `webhook-consent: required`: nothing is sent until **OK**; events wait in
  the queue. **No thanks** drops them and sends nothing, ever, from that
  browser for that course.

Privacy, stated in RULES: no answers, no lab output, no names unless the
learner types one, no IP or browser data beyond what any request carries;
the learner id is random.

## Part 2: formats

- `json`: the schema above.
- `slack`: `{"text": "Ana passed the quiz “Check yourself” in A first web page: 80 % (attempt 2)."}`
- `teams`: a message with one Adaptive Card (`TextBlock` with the same
  sentence), what Teams Workflows (and the older connectors) accept.
- `discord`: `{"content": "<the sentence>"}`.
- `xapi`: one xAPI 1.0.3 statement per event:
  - `actor`: `{"objectType": "Agent", "account": {"homePage": <course url>, "name": <learner id>}, "name": <name>?}`
  - `verb` (ADL vocabulary, `http://adlnet.gov/expapi/verbs/…`):
    lesson.completed and course.completed → `completed`; quiz.submitted →
    `passed` or `failed`; lab.step.checked → `answered`; demo.watched →
    `experienced`.
  - `object`: an Activity whose `id` is the lesson's (or course's) URL, a
    lab check `…#check-N`; `definition.type` lesson, course, assessment,
    interaction or media from the ADL activity types.
  - `result`: `score.scaled` (0-1), `score.raw/min/max`, `success`,
    `completion`; demo.watched the video profile's progress extension.
  - `context.contextActivities.parent` the course, `context.platform`
    `si <version>`, `timestamp`, a UUID `id`.
  - The URL is the LRS statements endpoint (`https://lrs.example.com/xapi/statements`).

The sentences: "Ana finished “Tags” in A first web page." · "Ana passed
the quiz …: 80 % (attempt 2)." · "Ana checked step 2 of “Build your first
page” in …: passed." · "Ana watched 100 % of “A page loads” in …." · "Ana
completed the course A first web page." A learner without a name is
"Learner 3f9a".

## Part 3: labs that check themselves

```markdown
## Step 2: Start the server

<!-- check: curl -fsS http://localhost:8080/health · expect "ok" -->
```

- `<!-- check: <command> -->`, optional `· expect "text"` (the output must
  contain it) and `· exit N` (default 0). Shown in every build as **Run this
  to check** with the command and what passes.
- In `si play` on a loopback address with the terminal on (the drawer's
  rule), the box gets a **✓ Check** button. It posts the page and the
  command to `POST /_si/check`; the server runs it **only if that exact
  command is a check of that page** (re-read from the file, includes
  resolved), as `sh -c` (`cmd /C` on Windows) in a separate process, the
  page's folder as cwd, 30 s timeout, output capped at 64 KB. Not in the
  shell of the drawer: a check must not depend on, or disturb, what the
  presenter typed. The answer (pass, exit, output, ms, timed out) shows
  under the button; the page sends `lab.step.checked`.
- The relay: `POST /_si/relay` on `si play` and `si serve`, on a loopback
  address or with `--relay` (on any address). The page posts its formatted
  event and its own path; the server finds that page's course, forwards the
  body to the course's `webhook:` (`webhook: relay` means: the URL is only
  on the server, from `course.webhook` in si.conf), and signs it:
  `X-Si-Signature: sha256=<hex HMAC-SHA256 of the body>` with
  `webhook-secret:` from the user's si.conf or `SI_WEBHOOK_SECRET` (never a
  project si.conf: it goes to git; never the page). Forwarding only ever
  goes to a course's configured address, so the relay is not an open proxy.
  A page served this way sends every event through the relay.

## Part 4: the live class view (built)

`si serve course/ --class` (and `si play … --class`): course pages served
this way send every event, as the JSON schema, to `POST /_si/class/event`
(same-origin, the page's course found by `?page=` like the relay), beside
the webhook (which keeps its own `webhook-events:` and format; no webhook is
needed). The server checks and trims each event (version 1, a known event, a
random learner id, bounded numbers and titles, question ids only), drops
duplicate ids, caps 200 000 events a run and 5 000 per learner, keeps them in
memory and appends them to `.si/class/<date>.jsonl` beside the served folder
(a restart loads today's file). `/_si/class` (dashboard), `/data` (JSON),
`/events` (SSE) and `/csv` open on the loopback (address and Host) or with
the run's random key (`?k=`, printed with a QR code when the server is
reachable). The dashboard: learners × lessons grid (✓, quiz score and
attempts, a dot per check, video percent), who's stuck (a check failed twice
in a row; no progress for N = 5/10/15/30 minutes on the lesson of the last
event), per-lesson and per-question summaries, a module filter, a course
picker, CSV (formula-safe), light/dark, the course's theme tokens.
`--class` makes `webhook-learner: ask` the default. Decided: a keyed link
and the loopback both; files are kept until the trainer deletes them (no
expiry).

## Part 5: CI results

```text
si demo test docs/demos/ --webhook https://hooks.slack.com/services/… --webhook-format slack
```

At the end of the run (one script or a folder) si posts one summary:
`{"version": 1, "event": "demo.test", "title", "passed", "failed",
"skipped", "seconds", "ok", "report", "failures": [{"script", "feature",
"step"}], "si", "time"}` for json, or a sentence for slack, teams and
discord ("si demo test docs/demos: 5 passed, 1 failed in 42.0 s. Failed:
studio › Open a file › click "Open""). `report` is the report path, or
`$SI_REPORT_URL` when CI sets one (a link to the job's artifacts).
`webhook-secret` signs it too. xapi is refused (a test run is not a
learning record). A failed post is a warning: it never changes the exit
code.

## Left out (for now)

- A per-course xAPI activity base (`file://` builds get file URLs as ids).
- Retries with back-off inside one page (the queue retries on the next).
- Timestamps in the signature (replay protection): receivers can drop
  duplicate `id`s.
