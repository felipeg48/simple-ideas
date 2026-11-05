---
title: Start the lab
---

The lab runs the five order services and their queues in Docker.

```bash
docker compose up -d
docker compose ps
```

Open the dashboard on port 8080 and place a test order. It should reach
*dispatched* in under two seconds. Next, [break the payment provider](break.md).
