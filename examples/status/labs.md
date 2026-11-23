---
type: status
title: Dockyard Labs — Status Board
subtitle: Self-hosted Docker lab sandboxes · one Go binary, no Kubernetes
updated: 2026-11-22
meta:
  - git: on main · committed        | ok
  - build: go build ✓ · vet ✓       | ok
  - e2e: verified on a laptop VM    | ok
  - tests: unit 11/11               | plain
tiles:
  - num: "25";  lbl: Learners / host;       sub: first-phase memory budget
  - num: "{open}"; lbl: Still open;          sub: {missing} missing · {improve} improve
  - num: 1/1;   lbl: Repos green;           sub: dockyard builds + vets
  - gate: "Passed ✓ on a laptop VM"; tone: ok; lbl: End-to-end gate; sub: create → terminal → 3 containers → teardown
---

## Control Plane

### Done
- Session API + time-to-live reaper
- Sandboxes over the Docker Engine API (standard library only)
- Lab registry + class codes + per-lab environment
- Signed per-session cookie
- Readiness gate — a sandbox stays `provisioning` until its shell answers

### Missing
- Terminal over `docker exec` instead of a separate terminal server `P2`

### Improve
- A pool of warm sandboxes for instant starts `partial`

## Learner Page

### Done
- Lab page wired to the real API
- Progress banner while the sandbox comes up
- Run button on code blocks (types into the live terminal)
- Readable colours on dark code blocks `f18de4b` `improvement`
- Editor tab over the file bridge

### Missing
- Browser tab for the sandbox's own web ports `partial`

### Improve
- Serve every asset from the binary

## Hosting

### Done
- Host bootstrap script
- Hosting plan — one always-on 64 GB host, up to 25 learners
- Kubernetes ruled out (cost and privileged containers)

### Missing
- Production host provisioned `P0`
- Single sign-on in front of the gateway `P0`

### Improve
- Spot hosts and smaller topologies for density

## Shipped
- Lifecycle gate + cleanup phase `482434a`
- Progress banner `d86f934`
- Run button + readable colours `f18de4b`
- Terminal readiness fix + hosting cost notes `0a6bb10`
- Always-on first-phase hosting plan `9fb4bc1`
