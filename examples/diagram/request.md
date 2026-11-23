---
type: diagram
title: One order, one request
description: What happens between the click on Pay and the receipt
orientation: portrait
theme: studio
---

```sequence
Browser → Shop: POST /orders
Shop → Payments: authorise €84 · 380 ms
Payments --> Shop: ok
note Shop: stock reserved
Shop → Warehouse: pick order #4711
Shop --> Browser: 201 Created · 520 ms
```
