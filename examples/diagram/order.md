---
type: diagram
title: The life of an order
description: Every state an order can be in, and the path order #4711 took
theme: studio
---

```state
Placed → Paid on authorised
Placed → Cancelled on timeout
Paid → Shipped on picked
Paid → Refunded on cancelled
Shipped → Delivered on scanned
Shipped → Returned on refused
event authorised
event picked
event scanned
```
