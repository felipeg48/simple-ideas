---
title: Place an order
description: Send order #4711 to the checkout and watch it appear on the customer's tracking page.
---

# Place an order

| | |
|---|---|
| **Time** | 5 minutes |
| **You need** | The lab from [Install the lab](../install.md#step-2-start-the-stack), running |

## Step 1: Send the order

```bash
curl -X POST localhost:8080/orders \
  -H 'content-type: application/json' \
  -d '{"id": 4711, "items": [{"sku": "LAMP-02", "qty": 1}], "promise": "thursday"}'
```

The checkout answers with the order and its first status:

```text
{"id": 4711, "status": "placed", "promise": "Thursday"}
```

## Step 2: Open the tracking page

Open [localhost:8085/orders/4711](http://localhost:8085/orders/4711). The
customer sees one green tile, **Placed**, and four grey ones. The promise
under them reads *Delivered by Thursday*.

> [!NOTE]
> The tracking page reads the same events the stations write. It never asks a
> station directly, so a slow station never makes the page slow.
