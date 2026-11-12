# Web demos: ordered steps, cleanup, and scripts from anywhere

Status: spec, decided with the founder on 2026-11-12; all three phases built
(unreleased: `internal/pinned`, `internal/webdemo/lifecycle.go`,
`cmd/si/stage.go`). Not built: the relative files of a script fetched from a
plain URL (its `app:` must be a URL or `none`).

## Why

A real demo needs more than one program and one setup command: start a
broker, create its queues, load data, start a consumer, and clean all of it
up afterwards, every time, even when a step fails. And a product's demos
should be shareable: run them from a folder, a repo or a URL, with the
scripts they need fetched safely.

## Phase 1: ordered steps and cleanup

### Front matter, in the order written

```markdown
---
start:   make demo-single-insecure · wait for port 22310   # keeps running
prepare: ./scripts/create-queues.sh                         # must finish, exit 0
step:    acemq-workloads run -f seed.yaml                   # alias of prepare:
start:   ./consumer · wait for "listening"                  # a second program, after the seed
setup:   acemq queue purge orders                           # before EACH feature
stop:    make demo-down                                     # while the programs still run
cleanup: rm -rf data/ tmp/                                  # always last
---
```

- `start:`, `prepare:` and `step:` may repeat and run **interleaved in the
  order written**, once, before the first feature. `prepare:`/`step:` must
  exit 0 within `timeout:` (`· within 5m` for longer); `start:` as today.
- `setup:` may repeat: in order, before each feature.
- `stop:` may repeat: in order, after the last feature, while the programs
  still run.
- `cleanup:` (new) may repeat: in order, after si has stopped every program.
- Order of a run: start/prepare/step (as written) → for each feature: setup,
  the feature → stop → programs stopped → cleanup.
- `stop:` and `cleanup:` always run: after a pass, a failure, a failed
  `prepare:`, Ctrl-C. Their output goes to the report and the logs, never on
  film.
- A failed `prepare:` fails every feature with its output (as a `start:`
  that never comes up does today).

### Sections, with steps and checks

```markdown
## Before
- start "make demo-single-insecure" · wait for port 22310
- run "acemq queue create orders" · wait
- expect output "created"

## After
- run "make demo-down" · wait
```

- `## Before` and `## After` are not features: no clip, no chapter, no
  narration, no title card, never filmed, not in `--feature` lists. Any step
  a terminal or the browser takes works there (`run`, `start … as`, `expect
  output`, `go`, `click`, `expect text`…).
- Order: front-matter start/prepare/step, then `## Before`; after the last
  feature, `## After`, then front-matter stop, programs stopped, cleanup.
- A failed check in `## Before` fails the run (every feature skipped, the
  report shows why). `## After` always runs; its failures are reported but do
  not change the features' results (the exit code says 1 if it failed).
- Programs started in `## Before` with `start … as NAME` live for the whole
  run.

### Many scripts at once

- `si demo test docs/demos/` and `si demo run docs/demos/`: every demo script
  in the folder (`type: demo` with `app:`, name order), one after another,
  each in its own output folder; one combined `report.html`/`report.json`
  and one JUnit file (`--junit`), a test suite per script.

## Phase 2: scripts from anywhere, safely

### Sources for start / prepare / step / setup / stop / cleanup

| Write | Means |
|---|---|
| `./seed.sh` (any command) | as today: run in the script's folder |
| `gist:abc123@4f2e9c1` | a GitHub gist at that revision (its one file, or `gist:abc123@rev/file.sh`) |
| `github:owner/repo@v1.2/scripts/seed.sh` | one file from a repo at a tag or commit |
| `github:owner/repo@v1.2/scripts/` | every script in that folder, in name order |
| `https://…/seed.sh · sha256 9f2a…` | a plain URL, with its fingerprint |

- Arguments follow the source: `prepare: github:acme/ops@v1/seed.sh --orders 50`.
- **Pinned only:** a remote source without `@tag`/`@commit`, or a URL without
  `· sha256`, is a `si check` problem. A pin that no longer matches (the
  file changed) stops the run.
- **Asked the first time:** before running remote code si shows each source
  (where it comes from, its size, its first lines) and asks once; the answer
  is remembered per pin in `~/.si/trusted`. `--trust` answers yes for this
  run (CI). `si demo test` without a terminal and without `--trust` refuses
  and says why.
- Downloads cached by pin in `~/.si/cache/scripts`; `github:` uses the GitHub
  API or `gh` when present (private repos); shell scripts run with the
  script's folder as their working directory (`· in <dir>` to change it).

### Demos from anywhere

| Write | Means |
|---|---|
| `si demo test github:owner/repo@v2/demos/` | clone (shallow) at that pin into `~/.si/cache/repos`, run every demo there |
| `si demo test github:owner/repo@v2/demos/shop.md` | one script from the repo, its folder alongside |
| `si demo test https://…/shop.md · sha256 …` | one script from the web (its relative files fetched next to it, pinned the same way) |

Same trust rules; the output folder is local (`demo-out/<repo>/<script>/`).

## Phase 3: decks from anywhere, with their own setup

The founder (2026-11-12): present a talk from a repo or a URL, with the
environment it needs set up first.

- `si play`, `si build` and `si serve` take the same places as `si demo test`:
  a folder (as today), `github:owner/repo@REF/talks/` or `…/talk.md`, a
  pinned URL (`https://…/talk.md · sha256 …`; its relative files fetched
  alongside, pinned). Cloned or fetched into `~/.si/cache/repos`, served from
  there; the same trust rules (asked once per pin, `--trust`).
- A deck (any document `si play` serves) may set up what the talk shows live:
  front matter `setup:` (must finish), `start:` (keeps running, `· wait for
  …`), `stop:` and `cleanup:`, each repeatable, in the order written, with the
  same sources (`./script.sh`, `gist:`, `github:`, a pinned URL).
  ```markdown
  ---
  title: The orders broker, live
  setup: github:acme/ops@v1.4/scripts/install-broker.sh
  start: ./bin/broker --port 5672 · wait for "ready"
  stop:  ./bin/broker stop
  ---
  ```
- When: `si play` runs them before it serves the first page (a "getting
  ready" page shows meanwhile with each step's progress), keeps `start:`
  programs running while it serves, and runs `stop:`/`cleanup:` when it quits
  (Ctrl-C included). The terminal drawer and the runbook see the started
  programs. `si build` never runs them (a built deck is a file); `si check`
  checks their sources and pins.
- Remote setup asks before running, every time it changed (pin), never
  silently; a deck from a repo that sets up nothing just plays.

## Tests

Parser cases (repeat keys, order, aliases, sections, sources and pins);
lifecycle order and always-run on failure and Ctrl-C (fake commands writing
to a log); folder runs (combined report, JUnit suites); phase 2 against
`httptest` fakes for gist/GitHub/URL (never the network), the trust store,
pin mismatch, `--trust`.

## Docs

RULES.md §27 (front matter table, a "Before and after" section, sources and
trust), the manual, agent rules + skill, FEATURES, CHANGELOG, the site's Web
Demos page, HANDOFF, CLAUDE.md roadmap; an example under `examples/webdemo/`
using `prepare:`, `## Before`/`## After` and `cleanup:`.
