---
type: status
title: Cockroach Education Platform — Status Board
subtitle: Self-hosted Docker lab platform · Go control plane, no k8s
updated: 2026-11-22
meta:
  - git: on main · committed        | ok
  - build: go build ✓ · vet ✓       | ok
  - e2e: verified on Colima         | ok
  - tests: unit 11/11               | plain
tiles:
  - num: "25";  lbl: Sessions / VM;         sub: Phase-1 RAM budget
  - num: "{open}"; lbl: Still open;          sub: {missing} missing · {improve} improve
  - num: 1/1;   lbl: Repos green;           sub: cep-ops builds + vets
  - gate: "Passed ✓ on Colima"; tone: ok; lbl: End-to-end gate; sub: create → terminal → 3-node CRDB → teardown
---

## Control Plane

### Done
- Session API + TTL reaper
- Provisioner over the Docker Engine API (stdlib only)
- Track registry + cohort binding + per-track env
- Token auth — signed per-session cookie
- Readiness gate — box holds `provisioning` until ttyd is up

### Missing
- xterm.js terminal to replace ttyd `P2`

### Improve
- Warm-session pool for instant ad-hoc start `partial`

## Learner Client

### Done
- CU-branded lab page wired to the real API
- Provisioning HUD (title + description) while the cluster comes up
- Run button on code blocks (types into the live terminal)
- Readable syntax colors on dark code blocks `f18de4b` `improvement`
- Editor tab (Monaco) over the file bridge

### Missing
- Browser / Apps tabs (per-session port proxy) `partial`

### Improve
- Self-host remaining assets

## Infra & Deploy

### Done
- host-bootstrap.sh
- Hosting plan — Phase-1 one always-on 64 GB VM, ≤25 sessions
- GKE ruled out (cost + privileged DinD)

### Missing
- GCP VM provisioned `P0`
- LB + IAP on crdb.dev `P0`

### Improve
- Spot + 1-node topology for density

## Shipped
- Lifecycle gate + cleanup phase `482434a`
- Provisioning HUD `d86f934`
- Run button + readable colors `f18de4b`
- Terminal readiness fix + GKE/cost doc `0a6bb10`
- Always-on Phase-1 hosting plan `9fb4bc1`
