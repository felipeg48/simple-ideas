---
type: architecture
title: Dockyard Labs — Architecture & Design
subtitle: Self-hosted Docker lab sandboxes · one Go binary, no Kubernetes
---

## The control plane is the scheduler

Dockyard runs no Kubernetes and no virtual machine per learner. One Go binary drives the
Docker Engine API on a single large host, and that binary *is* the scheduler.

- `invariant` **One binary** — `dockyard serve` creates sandboxes, proxies the terminal,
  serves the lesson pages and runs the step checks.
- `invariant` **Budget, not count** — a new sandbox is admitted against a memory budget,
  never a fixed number of containers.
- `safe nesting` **Docker inside the sandbox** — each learner runs their own containers
  inside their own sandbox; the host's Docker socket is never shared.

> Honest read: privileged Docker-in-Docker is fine for a trusted class; an unprivileged
> runtime comes before strangers get a seat. The host is Linux only.

## Request & networking flow

How a learner's browser reaches their own sandbox: one origin, routed by path.

```flow
dir: LR
B[Browser] -->|HTTPS + WS| G[Gateway] -->|/s/id/term| P[Session proxy] -->|docker exec| T[Shell in sandbox]
dot B..T 2.4s
```

## Capacity & the first phase

Measured per learner: one sandbox running a small compose project of one to five
containers. Memory follows the number of containers, so a host is sized by RAM.

| Host | Memory | 1 container | 3 containers | 5 containers |
|------|--------|-------------|--------------|--------------|
| 8 vCPU | 64 GB | ~50 | ~25 | ~16 |
| 16 vCPU | 128 GB | ~110 | ~55 | ~36 |

> First phase: one always-on 64 GB host, a 50 GB budget at 2 GB per sandbox: 25 learners
> at a time, behind the company's single sign-on.

## Phases

1. One host · privileged Docker-in-Docker · up to 25 learners. (now)
2. Unprivileged runtime · a bigger host · smaller topologies per lab.
3. Several hosts; the control plane still places every sandbox.
