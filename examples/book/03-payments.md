---
description: Why night-time card declines were timeouts, and the retry that fixed them.
---

# Payments that time out

## The declines were timeouts

Nine in ten night-time declines were the bank's fraud check timing out, not a
decision about the card. A ((timeout)) is not a decline: it is a bank that has
not answered yet.

## One idempotent retry

```java title="Retry.java" hl="3-6"
var key = order.id();                                  // <1>
var result = bank.authorize(card, amount, key);
if (result.isTimeout()) {
    sleep(Duration.ofSeconds(30));                     // <2>
    result = bank.authorize(card, amount, key);
}
return result;
```
<1> The order id is the ((idempotency key)): a retry is the same request.
<2> One retry only, thirty seconds later.

The retry changes nothing for the stations after payment; see
[](01-journey.md#hand-offs). The (((payment, retry))) retry is logged with both
attempts.
