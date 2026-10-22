---
title: The Thursday Promise, appendix
background: linear-gradient(160deg, #f6f5f1 0%, #e9eefb 100%)
---

# Appendix: the numbers

Where the time goes, and what each change bought.

Notes: (0.5 min)
Three slides of evidence. Skip them in a short slot; they are here for the questions.

---

## Where an order spends its time

```mermaid
flowchart LR
  shop[Shop] --> pay[Payments]
  pay -- declined --> retry{{Retry in 30 s}}
  retry --> pay
  pay --> wh[Warehouse]
  wh --> courier[Courier]
  courier --> door[Doorstep]
```

Notes: (1 min)
The only loop in the journey is the payment retry. Everything else moves forward or waits.

---

## Hand-off times, median

| Hand-off | Before | After |
|---|---:|---:|
| Paid to picked | 9 h 40 min | 3 h 05 min |
| Picked to packed | 35 min | 22 min |
| Packed to on a van | 14 h | 13 h 30 min |
| Declined payments recovered | 0% | 61% |

Notes: (1 min)
The big win is paid to picked: the lent picker. Packed to van barely moves because routes leave at fixed times.

---

## Two views of route 7

<!-- layout: two-col -->

<!-- image: ratio=4:3 fit=cover position=left-bottom -->
![Route 7 at dusk, left](assets/dusk.jpg "Depot side")

<!-- column -->

<!-- image: ratio=4:3 fit=cover position=right-bottom -->
![Route 7 at dusk, right](assets/dusk.jpg "Residential side")

Notes: (0.5 min)
Same morning, two ends of the route. On a narrow screen the two pictures stack.
