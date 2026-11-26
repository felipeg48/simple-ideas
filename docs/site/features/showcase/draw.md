---
type: deck
title: Where the retry goes
author: Northwind Logistics
theme: studio
---

<!-- layout: center -->
# Where the retry goes

One order, three services, one place to try again

Notes: Every order crosses three services, and only one of them is allowed to try again.

---

<!-- layout: two-col -->
## Three hand-overs

- **Cart** sends the order
- **Payments** charges the card
- **Warehouse** picks the box

<!-- column -->

The card check fails one time in fifty. Payments retries once, keyed on the
order, so the customer is never charged twice.

Notes: The cart sends the order, payments charges the card, the warehouse picks the box. The card check fails one time in fifty, and payments is the one place that tries again.

---

<!-- layout: big-number -->
## 1 in 50

card checks fail the first time, and pass on the retry

Notes: One check in fifty fails the first time and passes on the retry, thirty seconds later.

---

<!-- layout: center -->
## Retry once, in one place, keyed on the order

Notes: So the rule is short: retry once, in one place, keyed on the order.
