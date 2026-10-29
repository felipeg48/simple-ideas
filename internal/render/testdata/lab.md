---
type: lab
title: Ship an order through the pipeline
author: Ada Lovelace
date: 2026-10-30
version: v1.2
year: 2026
copyright: Harbor Labs
footer: Harbor training
---
# Ship an order through the pipeline

You will place an order, follow it through the queue and read it back. The
next lab picks up from here: [Replay the queue](other.md#start).

## Before you start

> [!NOTE]
> You need a terminal and **Go 1.22** or later.

> [!TIP]
> Keep a second terminal open for the logs.

> [!IMPORTANT]
> Every command runs from the repository root.

> [!WARNING]
> The reset step drops the local database.

> [!CAUTION]
> Never point the lab at a production cluster.

> Orders are only as reliable as the slowest hop.

## Set up

### Clone

```bash
git clone https://example.com/harbor/orders.git && cd orders && make bootstrap ENV=lab REGION=us-east-1 WORKERS=4 QUEUE=orders-lab RETRIES=5 TIMEOUT=30s LOG_LEVEL=debug
```

### Clone

Clone the fixtures too.

```
plain text block
```

| Hop | Service | Queue | Timeout | Retries | Owner | Dashboard | Notes |
|---|---|---|---|---|---|---|---|
| 1 | gateway | orders-in | 5s | 3 | edge team | gateway-overview | accepts and validates the order payload |
| 2 | ledger | orders-ledger | 30s | 5 | payments team | ledger-latency | writes the order and reserves stock |

![The pipeline](assets/pic.svg "Orders flow left to right")

---

<!-- include: lab-part.md -->
