---
title: A declined card
description: Make the bank slow, watch the first payment time out, and see the retry turn it into a paid order.
---

# A declined card

At 21:03 the real order #4711 was declined: the bank's fraud check timed out
during its nightly batch. Here you make the fake bank just as slow.

| | |
|---|---|
| **Time** | 15 minutes |
| **You need** | The lab running, a fresh order (run [Reset the lab](../reset.adoc) first if #4711 is already paid) |

## Step 1: Make the bank slow

```bash
curl -X POST localhost:8081/admin/latency -d '{"ms": 25000}'
```

The fake bank now takes 25 seconds to answer, longer than the 15 seconds the
payment service waits.

## Step 2: Read the retry

This is the part of the payment service that saves the order, in the three
languages the lab ships. Pick yours; the choice sticks on every page.

<!-- tabs -->
<!-- include: ../code/Retry.java tag=retry hl=3-6 title=Java -->
<1> The order id is the idempotency key, the same on both calls.
<2> One retry only, thirty seconds later.
<3> A second timeout is returned as it is: the customer sees "still checking".

<!-- include: ../code/retry.py tag=retry hl=3-5 title=Python -->
<1> The order id is the idempotency key, the same on both calls.
<2> One retry only, thirty seconds later.
<3> A second timeout is returned as it is: the customer sees "still checking".

<!-- include: ../code/retry.go tag=retry hl=3-6 title=Go -->
<1> The order id is the idempotency key, the same on both calls.
<2> One retry only, thirty seconds later.
<3> A second timeout is returned as it is: the customer sees "still checking".

The same key goes with both calls, so the bank cannot charge the card twice.

## Step 3: Pay and wait

```bash
time curl -X POST localhost:8082/orders/4711/pay
```

```text
{"order": 4711, "status": "paid", "attempts": 2, "elapsed_ms": 30412}
real    0m30.4s
```

The tracking page showed *Still checking your payment* for thirty seconds,
then turned the payment tile green.

> [!WARNING]
> Set the bank back to normal before the next page:
> `curl -X POST localhost:8081/admin/latency -d '{"ms": 200}'`.
