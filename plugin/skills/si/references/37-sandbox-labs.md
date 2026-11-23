# 37. Sandbox labs

A lab can bring its own machine: `sandbox:` names a Docker image, and `si
play` makes a box from it before the page opens. The box has Docker inside
it (the learner runs `docker` and `docker compose` there), the lab's files
in `/lab`, and nothing of your own computer. The terminal drawer's tabs open
shells in the box, **✓ Check** runs the page's checks in the box, and links
written `sandbox:PORT` open the box's ports. When `si play` stops, the box
goes.

```markdown
---
type: lab
title: Put a Proxy in Front of the Shop
sandbox: docker:28-dind
sandbox-files: env
sandbox-setup: docker compose pull -q
sandbox-cleanup: docker compose down
sandbox-ports: 8080 Shop
sandbox-limits: memory=2g cpus=2
---

# Put a Proxy in Front of the Shop

## Start the shop

    docker compose up -d

<!-- check: docker compose ps --services --status running · expect "proxy" -->

## Look at it

Open [the shop front](sandbox:8080).
```

| Key | What it does |
|---|---|
| `sandbox:` | the image, pinned: `name:tag` or `name@sha256:…`. A Docker-in-Docker image (`docker:28-dind`, or your own built from it) gives the learner Docker; any other image gives a plain box. `si check` warns about an image with no tag or `:latest`. |
| `sandbox-files:` | a folder beside the page, copied into `/lab` (files and folders; links are skipped) |
| `sandbox-setup:` | a command run in `/lab` once Docker in the box answers (pull images, seed data); a failure stops `si play` with its output |
| `sandbox-cleanup:` | a command run in the box before it is removed (30 s at most) |
| `sandbox-ports:` | the box's ports the learner opens: `15672 Console, 8080 App` |
| `sandbox-limits:` | `memory=2g cpus=2 pids=2048` (these are the defaults) |

**In the page.** `[Console](sandbox:15672)` (and `sandbox:15672/path`)
becomes a link to `/_si/sandbox/port/15672/`, which `si play` sends on to
where the box publishes that port. `GET /_si/sandbox` says what runs:
the image, whether it runs unprivileged and its ports. A page's checks run
with `sh -c` in the box's `/lab`; a page without `sandbox:` keeps running its
checks on this computer.

**The sandbox host.** Boxes do not run on your own Docker. `si sandbox host
init` makes a Colima VM, `si-sandbox`, for them:

```text
si sandbox host init                 # once: Docker, sysbox, no host folder mounted
si sandbox host status               # VM, socket, mounts, sysbox, room for boxes
si sandbox host start | stop | rm
```

- The VM mounts one empty folder, `~/.si/sandbox/share`, never your home
  folder (Colima mounts the home folder by default); `init` and `status`
  check that the VM sees nothing else.
- sysbox (a pinned, checksummed release) runs every box unprivileged: root in
  the box is not root in the VM. A host without sysbox refuses boxes unless
  `si play --trusted-sandbox`, which runs them privileged: only for an image
  and people you trust.
- Your own `docker` context is left as it is.
- `init` sizes the VM to half the cores, three quarters of the memory and
  100 GiB of disk (`--cpu`, `--memory`, `--disk` to choose); a box takes 2 GiB
  unless `sandbox-limits:` says otherwise.

`sandbox.host:` in si.conf picks another host: `docker` (your own Docker:
`$DOCKER_HOST`, else `/var/run/docker.sock`) or a socket path;
`$SI_SANDBOX_HOST` wins over it.

**Your own image.** `si new web-lab --type sandbox` writes a lab with its
`env/` folder and a `Dockerfile` from `docker:28-dind`. Build it into the
sandbox host and name it in `sandbox:`:

```text
si sandbox build web-lab -t northwind/web-lab:1.0
si sandbox try web-lab/lab.md        # the box as si play makes it, a shell here
```

`si play` runs one box, for the file it serves or a folder's top-level
pages: pages that name the same image share it; two different images are a
problem. Pages in subfolders never start one (serving a folder of examples
starts no Docker). `sandbox:` in a course's `_course.md` gives the whole
course one box, and every lesson's checks run in it. `si serve`
and built pages have no box: the page shows the commands, and a
`sandbox:` link has nowhere to go. Boxes left by a `si` that was killed are
removed the next time one starts.

**The drawer on the lab page.** A lab page served by `si play` (or to a
class seat) has a **>_ Terminal** button (and the `` ` `` key) for the
drawer, and every ```` ```bash ```` (`sh`, `shell`, `zsh`, `console`) block a
**Run** button that types it into the drawer's shell, `$ ` prompts left out.
A built page has neither.

### Classes

`si class` serves a course with a `sandbox:` to a class from your computer:
each learner gets a seat code, and their own box from the course's image
when they first open the course. Learners reach only shells in their own
box: never a shell, a file or a page of si's on your computer.

```text
si class new docker-course --name "Docker basics" --roster people.csv --seats 5
si class start docker-basics --tunnel     # a link per seat; Ctrl+C removes every box
si class                                  # the planner: classes, seats, codes, live boxes
```

| Command | What it does |
|---|---|
| `si class` | the planner, a keyed page on this computer only: plan a class (course, image, memory and CPUs per box, how long a box lives, when it is removed idle, the last day, seats from a roster or a count), a printable sheet of codes with a QR code per seat, start and stop it, each seat's box live (starting, ready, its memory), **Reset** (a fresh box from the image), seats added or removed on the day, the images on the sandbox host and **Pull** |
| `si class new <course>` | plan a class of a course folder (its `_course.md` or a top-level lab has `sandbox:`) or of one lab page: `--name`, `--seats N`, `--roster file.csv` (rows `name,email[,code]`, a header row skipped), `--image` (instead of the course's), `--memory 2g`, `--cpus 2`, `--ttl 3h`, `--idle 30m`, `--expires YYYY-MM-DD` |
| `si class list` / `codes <id>` / `rm <id>` | the classes planned here, a class's seats and codes, forget a class (its codes stop working) |
| `si class start <id>` | serve the class and open the planner: `--tunnel` (a public HTTPS address, cloudflared; the default when it is installed), `--https` (this network, si's own certificate: a learner's device trusts it once at `http://<address>/ca`), `--local` (this computer, to try it); `--budget 48g`, `--addr`, `--trusted-sandbox` |

A class is a file, `.si/classes/<id>.json` (0600: it holds the codes). A
code is 8 letters and digits, printed `ABCD-EFGH`, typed in any case;
`<link>/go/ABCD-EFGH` (or `<link>/go` and typing it) gives the learner's
browser the seat (an HttpOnly cookie, Secure over HTTPS) and opens the
course. The same code comes back to the same box; a box removed for its
`--ttl` or `--idle` comes back fresh. Wrong codes are slowed down (60 a
minute for the whole class), and a class past its last day lets nobody in.

**What a learner reaches**: the course's pages, the drawer (each tab a
shell in their box; the Viewer and file search, which read your folder, are
not there), **✓ Check** run in their box, their box's `sandbox-ports:`
(proxied: the box publishes them on your computer only), and the course's
results for the class view (**Progress** in the planner: the live class
view of §28). The page sources, `si.conf`, `.si/` and si's own pages are not
served. Through a quick tunnel, which holds a stream back, the drawer reads
its shells by long poll: a key comes back in about 150 ms.

**The budget.** Every box takes its memory (`sandbox-limits:` or
`--memory`) from the budget: 80% of the sandbox host's memory unless
`--budget` says otherwise. A seat whose box would go past it waits until a
box goes (Reset, `--ttl`, `--idle`, Stop). `si sandbox host init` sizes the
VM to three quarters of your memory: on a 128 GB Mac a 96 GB VM, about 77
GB of budget, about 38 learners at 2 GB.
