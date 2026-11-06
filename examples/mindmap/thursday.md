---
type: mindmap
title: The Thursday Promise
description: Every hand-off between "Buy now" on Monday and the doorstep on Thursday, and what we do when one of them breaks.
layout: map
lines: curved
nodes: box
theme: studio
watermark: Northwind Logistics
---
# The Thursday Promise

- Checkout
  - Cart & address
  - Payment → payments.md
  - Promise date {badge=Thu}
- Warehouse → warehouse.md
  - Picking
  - Packing
  - Label printed → slides: ../showcase.md#7
- Delivery
  - Dusk departure ![The first vans at dusk](../assets/dusk.jpg) → slides: ../showcase.md#13
  - Route 7
  - Doorstep photo
  - [!sticky] Drivers asked for a later cut-off on Thursdays. Check with ops.
- When it breaks
  - Retry with backoff
    ```go
    for try := 1; try <= 5; try++ {
        if err := send(order); err == nil {
            return nil // delivered
        }
        time.Sleep(backoff(try))
    }
    ```
  - Dead-letter queue
  - Apology email {badge=auto}
