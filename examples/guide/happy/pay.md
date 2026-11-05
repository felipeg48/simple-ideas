---
title: Pay for the order
description: Send order #4711 through the payment service and watch the card authorised.
---

# Pay for the order

| | |
|---|---|
| **Time** | 5 minutes |
| **You need** | Order #4711 [placed](place.md) |

## Step 1: Pay

```bash
curl -X POST localhost:8082/orders/4711/pay
```

```text
{"order": 4711, "status": "paid", "attempts": 1, "elapsed_ms": 212}
```

## Step 2: Read the payment record

```bash
curl -s localhost:8082/orders/4711/payment | jq '{status, attempts, key}'
```

The `key` is the order id. The bank uses it to recognise the same request if
it arrives twice; the next part of the guide depends on it. See
[A declined card](../breaks/declined.md#step-2-read-the-retry).
