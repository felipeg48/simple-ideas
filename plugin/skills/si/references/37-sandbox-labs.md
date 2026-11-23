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

`si play` runs one box: pages that name the same image share it; two
different images in what one `si play` serves is a problem. `si serve`
and built pages have no box: the page shows the commands, and a
`sandbox:` link has nowhere to go. Boxes left by a `si` that was killed are
removed the next time one starts.
