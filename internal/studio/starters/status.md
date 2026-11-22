---
type: status
title: {{title}}
subtitle: Where the project stands
updated: {{date}}
meta:
  - build: green | ok
  - release: next week | warn
tiles:
  - num: "{pct}"; lbl: Done; sub: done of done + missing
  - num: "{open}"; lbl: Still open; sub: {missing} missing · {improve} improve
  - gate: Load test; lbl: The one gate left; sub: 10k orders a minute
---

## Checkout

### Done
- Cart and payment `v1.2 · 3f9c2a1`
- Receipts by email

### Missing
- Saved cards `P1`

### Improve
- Faster address lookup `partial`

## Warehouse

### Done
- Pick lists

### Missing
- Returns `P0`
- Stock alerts `gated`

### Improve
- Batch printing

## Shipped
- Receipts by email `3f9c2a1`
- Pick lists `8d01b77`
