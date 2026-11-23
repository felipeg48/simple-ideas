# Sandbox labs: a Docker box per learner, the lab beside it

Status: phase 1 DONE (v3.44.0, RULES §37); phase 2 (classes) is v4.0.0.
As built, the keys are flat like the rest of si's front matter
(`sandbox:`, `sandbox-files:`, `sandbox-setup:`, `sandbox-cleanup:`,
`sandbox-ports:`, `sandbox-limits:`), files are copied by `tar -x` over
exec (the Engine's archive upload fails on sysbox's mounts), the flag is
`--trusted-sandbox`, and the VM sizes are flags of `si sandbox host init`.
Not in phase 1: the Viewer reading files inside the box, `ports:` as
browser windows over the slides, the class pieces.

## The idea

An instructor prepares one lab image (Docker inside Docker: the learner runs
`docker` and `docker compose` in their own box). They open a class on a course;
each learner gets a seat code. A learner opens the code and gets their own
sandbox from that image, with the lesson on one side and a terminal into the
sandbox on the other. The lab's `<!-- check: -->` steps run inside the sandbox
and mark the step done. The instructor sees every seat in the class view,
watches or takes over a terminal, and resets a broken sandbox.

si already has almost every piece except the sandbox itself: the terminal
drawer (PTY + xterm.js, chunked-GET/POST transport), courses (`_course.md`),
`type: lab` with self-checking steps, the live class view, result webhooks,
`--remote` / `--https` / `--tunnel`, the browser-window proxy and a keyed
monitor. The plan adds one thing, a sandbox, and plugs it into those.

## How it reads in Markdown

In `_course.md` (whole course) or a lab's front matter (one lab):

```markdown
---
sandbox:
  image: ghcr.io/acme/mq-lab:1.4     # built by the instructor, pinned by tag or digest
  compose: env/compose.yaml          # copied in, `docker compose up -d` on start
  setup: env/seed.sh                 # runs once the box answers
  cleanup: env/teardown.sh           # runs before the box is removed
  memory: 2g · cpus: 2 · pids: 2048
  ttl: 2h · idle: 30m
  ports: 15672 Console · 8080 App    # opened as browser windows beside the lab
---
```

A lab step stays as it is today; its check runs in the sandbox:

```markdown
## Start three nodes

    docker compose up -d --scale node=3

<!-- check: docker compose ps --status running -q | wc -l · expect 3 -->
```

Code blocks get a **Run** button that types the command into the sandbox
terminal (the lab already has its runbook-to-drawer path).

## Where it runs

| Mode | Command | Who | Network |
|------|---------|-----|---------|
| Solo | `si play course/` | one learner, self-paced, own laptop | loopback only |
| Class | `si serve --class --sandbox` | instructor's host, N learners | `--https` or `--tunnel`, seat codes |
| Built | `si build course/` | static pages | no sandbox; the lab shows `docker run …` to start the image by hand |

## Architecture

```flow
dir: LR
L[Learner browser] -->|/go/CODE| S[si serve --class]
S -->|cookie per seat| P[seat proxy /s/ID/]
P -->|exec + TTY| E[Docker Engine API]
E --> B[Sandbox: dockerd + lab]
S --> C[class view + webhooks]
```

- **Engine client** (`internal/sandbox/engine.go`, stdlib): Engine HTTP API
  over the local socket (`/var/run/docker.sock`, Docker Desktop, Colima,
  OrbStack; a named pipe on Windows). Calls: ping, pull, network create/remove,
  container create/start/inspect/stats/remove, exec create/start. About 400
  lines; no Docker SDK.
- **Terminal over exec**: `POST /exec/{id}/start` with a TTY, hijacked
  (`Upgrade: tcp`) on the raw connection, resized through `/exec/{id}/resize`.
  It feeds the drawer's existing transport, so the learner sees the same
  xterm.js drawer as `si play`, with tabs (one exec each), split view, Viewer
  and text size. No ttyd, no WebSocket dependency, no shell on the host.
- **Lifecycle** (`internal/sandbox/session.go`): per seat a network
  `si-s-<id>` and a container `si-box-<id>` labelled `si.managed=1`,
  `si.class`, `si.seat`; limits from `sandbox:`; ready when `docker info`
  answers inside; then compose up, then `setup:`. A reaper removes boxes past
  `ttl:` or `idle:`; at start si reconciles labelled leftovers; on exit it tears
  down (or keeps them with `--keep`). New boxes are admitted against a memory
  budget (`--budget 48g`), never a container count.
- **Seats** (`internal/sandbox/class.go`): `si class new course/ --seats 30
  --expires 2026-12-05` writes `.si/class/<id>.json` (atomic rename, 0600) with
  one 8-character code per seat (base32, any case). `/go/<code>` binds the seat
  on first use, sets an HMAC cookie scoped to `/s/<id>/` (secret kept in
  `~/.si`, so a restart keeps learners signed in) and opens the course on that
  seat. The same code re-enters the same box, or a fresh one if it was reaped.
  A name is asked once, so the class view shows people, not codes.
- **Checks**: the lab's `<!-- check: -->` runner gets a sandbox executor
  (`docker exec` argv, exit code and output matched as today). Results go
  through the existing course events, so the class view and webhooks need no
  new path.
- **Ports**: `ports:` open through the browser-window proxy, path-routed
  `/s/<id>/port/15672/`, never published on the host.
- **Files**: the Viewer reads and writes inside the box (exec-based, confined
  to the lab folder, size-capped), so editing a config file needs no editor in
  the image.

## Instructor

- Class view: a sandbox column per seat (provisioning, ready, stopped; CPU,
  memory), step progress as today, "stuck" from the time on one step.
- **Watch**: a read-only copy of a seat's terminal stream; **Take over**:
  type into it (the learner sees a badge); **Show**: put it on the deck screen
  as a drawer tab.
- **Reset**: cleanup, remove, create again from the image; **Reset all**
  before the next exercise.
- All instructor routes behind the monitor key, as `/_si/monitor` is today.

## The lab image

`si new --type sandbox` scaffolds a `Dockerfile` (from `docker:dind`, plus
bash, git, curl, the lab's tools) and `env/` with an example compose file.
`si sandbox build` wraps `docker build`, `si sandbox try` starts one box locally
with the terminal, so the instructor tests the image the way a learner gets it.
`si check` warns on an unpinned image tag and on a missing `setup:` file.

## Security

- Privileged Docker-in-Docker gives root on the host to anyone who escapes the
  box. Default: `--sandbox` refuses privileged boxes unless the instructor
  passes `--trusted-class`; with `runtime: sysbox-runc` on the host (Linux),
  boxes run unprivileged and that flag is not needed.
- The host's Docker socket is never mounted into a box. One network per seat.
  Memory, CPU and pids are always capped; disk is capped where the storage
  driver allows (`storage-opt size`), and `si check` says where it does not.
- Class mode needs `--https` or `--tunnel`; codes are rate-limited; cookies
  are HttpOnly, SameSite=Lax, Secure on HTTPS.
- Every instructor and API route is keyed; nothing that creates or deletes a
  box is reachable without the key (the gap the reference control plane has).

## Phases

1. **Solo sandbox** (`si play`, loopback only, no rule change): engine
   client, `sandbox:` key, one box, exec terminal in the drawer, checks in the
   box, ports through the proxy, cleanup on exit. `si sandbox try`, `si new
   --type sandbox`. RULES §37, an example course lab (an invented message-broker
   cluster lab), tests against an `httptest` fake of the Engine API plus one
   real-Docker test behind `SI_DOCKER=1`.
2. **Classes**: `si class new`, seat codes, `/go/<code>`, cookie, per-seat
   boxes, budget, reaper, reconcile, reset; class view columns. Needs the
   rule #8 change below.
3. **Instructor live**: watch, take over, show on screen, reset all, idle
   warnings.
4. **Hardening**: sysbox by default where present, a small warm pool for
   instant starts, Windows host check. Several hosts stays on hold until a
   real class needs more than one machine.

## Decided with the founder (2026-11-22)

1. **Two terminals, two audiences.** The drawer terminal stays the
   presenter's own, loopback only, unchanged. Learners reach only sandbox
   shells (exec into their box), through `--tunnel` or `--https`. Rule #8 gains
   that exception; a learner never gets a shell on the host.
2. **Isolation.** Boxes never run privileged on the presenter's own Docker by
   default. A privileged box on Docker Desktop / OrbStack / Colima can mount the
   VM's disks, and that VM shares the user's home folder, so an escape reads the
   presenter's files. Default: a dedicated sandbox host on **Colima** (the
   founder's choice), so boxes run unprivileged. `--trusted-class` allows
   privileged boxes on any Docker, with a warning, for people the presenter
   knows.

## The sandbox host (Colima)

`si sandbox host init|start|stop|status|rm` drives one Colima profile,
`si-sandbox`, apart from the presenter's own Docker:

- `colima start si-sandbox --runtime docker --cpu N --memory M --disk D
  --mount ~/.si/sandbox/share` (sizes from flags or si.conf `sandbox.cpu`,
  `sandbox.memory`, `sandbox.disk`). Colima mounts the home folder when no
  mount is given, so si always gives exactly one: an empty share folder. Phase 1
  verifies inside the VM that `/Users` is not reachable, and `status` checks it
  on every start.
- A `provision:` script in the profile's `colima.yaml` (system mode) installs
  sysbox-ce on the Ubuntu VM and restarts dockerd; si creates boxes with
  `HostConfig.Runtime: sysbox-runc`, never `Privileged`.
- si talks to `~/.colima/si-sandbox/docker.sock` only; the presenter's
  `docker` context is never touched. Missing Colima: `status` says how to
  install it (`brew install colima`); Linux hosts skip the VM and use sysbox
  on the local Docker.
- `si sandbox host status` prints the VM, sysbox, socket, free memory and how
  many boxes fit the budget.
3. **Hosting.** The person running si hosts. No hosted si service.
4. **Seats planned ahead.** A roster CSV (`name,email[,seat]`) or seats typed
   in a web planner; codes and QR codes (internal/qr) printed or copied per
   learner; seats can also be added on the day.

## Class planner (web)

`si class` opens a keyed loopback web page (the Studio's style, like the
control room), not a new `si remote`: `--remote` already names the tablet
listener. It plans and runs classes:

- **Images**: local `docker images`, pull by name, `si sandbox try` per image,
  pinned digest saved with the class.
- **Class**: course folder, image, limits, TTL, dates, seats (CSV upload or
  typed), codes and QR sheet, the share link (`--tunnel` or LAN HTTPS).
- **Live**: start/stop the class, per-seat box state, CPU and memory, reset,
  watch / take over (the class view and control room pieces).
- CLI twins for scripts: `si class new|list|start|stop|reset|codes`.

## Versions

Phase 1 (solo sandbox, loopback) ships as a 3.x minor. **v4.0.0** is phase 2
plus the planner: learners on the presenter's machine is a new hosting model
and a new rule #8 exception, worth the major number.

## Open risks

- Cloudflare quick tunnels buffer streams (the audience page already falls
  back to polling). The terminal stream must be tested through one; a named
  tunnel or LAN HTTPS may be required for typing to feel live.
- The host's RAM bounds the class, at about 2 GB per box. The founder's Mac
  has 128 GB: `si sandbox host init` defaults to a 96 GB VM with an 80 GB box
  budget, leaving 32 GB for macOS, si and the presenter's own Docker. That fits
  about 40 learners at 2 GB each, or about 26 at 3 GB. On smaller machines the
  default is three quarters of RAM for the VM.
