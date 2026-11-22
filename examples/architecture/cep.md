---
type: architecture
title: Cockroach Education Platform — Architecture & Design
subtitle: Self-hosted Docker lab platform · Go control plane, no k8s
---

## The control plane is the scheduler

CEP runs no Kubernetes and no per-session VMs. A Go control plane drives the Docker Engine
API directly on one (later a few) beefy host(s) — it *is* the scheduler.

- `invariant` **One binary** — `cep ops` provisions boxes, proxies the terminal, serves the
  si-rendered content, and runs step checks.
- `invariant` **Budget, not count** — admission is by a RAM budget, never a container cap.
- `safe nesting` **Nested-Docker box** — the student deploys CockroachDB *inside* their own
  sandbox; the host Docker socket is never exposed.

> Honest read: privileged Docker-in-Docker for the demo; `sysbox-runc` (unprivileged) before
> untrusted users. Linux-only host.

## Request & networking flow

How a learner's browser reaches their private box — one origin, path-routed by the gateway.

```flow
client "Browser" -> edge "Gateway (cep ops)" : HTTPS + WebSocket
edge -> ctrl "Session proxy" : /s/{id}/term
ctrl -> node "ttyd in box" : nested docker
```

## Capacity, cost & the Phase-1 plan

Measured per session — one DinD box running a nested compose of 1–9 CRDB nodes (multi-region
means more nodes in the *same* box, not more VMs). Data is tiny (~10M small rows, on disk), so
RAM tracks node count.

| Host (always-on) | approx $/mo | 1-node | 3-node | 9-node |
|------------------|------------|--------|--------|--------|
| e2/n2-highmem-8 · 64 GB (Phase 1) | ~$250–300 | ~50 | ~25 | ~16 |
| n2-highmem-16 · 128 GB | ~$570 | ~110 | ~55 | ~36 |

> Phase 1: one always-on 64 GB VM, `--mem-budget-gb 50 --mem-mb 2048` → 25 sessions, internal
> via LB + IAP. GKE and scheduled start/stop are out (instant availability is required).

## Phases

1. Single host · privileged DinD · internal ≤25 sessions. (now)
2. Sysbox runtime + bigger VM + 1-node topology per track.
3. Multi-host placement (MIG); the control plane stays the scheduler.
