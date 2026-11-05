---
title: Break the payment provider
---

Slow the payment stub down to five seconds per call and watch the retry rate.

```bash
curl -X POST localhost:8090/stub/delay -d 'ms=5000'
```

The retry rate climbs first and the error rate follows about ninety seconds
later, as described in [Retries without the stampede](../blog/retries.md).
