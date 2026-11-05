---
type: article
title: One retry fixed nine in ten night-time declines
description: Order #4711 failed at 21:03 and was paid at 21:05. Behind those two minutes is the cheapest fix our checkout ever shipped, and a lesson about reading decline codes before blaming fraud.
author: Jonas Berg · Payments team
date: 5 November 2026
theme: studio
footer: Northwind Logistics engineering
year: 2026
copyright: Northwind Logistics
---

# One retry fixed nine in ten night-time declines

For a year we treated every night-time card decline as a fraud signal and
sent the customer a polite "payment failed" email. When we finally split the
decline codes by cause, the picture changed: almost nine in ten were the
bank's fraud check timing out while it waited on a batch job, not a decision
about the card.

| | |
|---|---|
| **Bank timeout** | 87% of night-time declines |
| **Fraud rule** | 9% |
| **Other** | 4% |
| **Sample** | 1,842 declines between 21:00 and 06:00, October 2026 |

## The declines were timeouts

A timeout is not an answer. It is a bank that has not answered yet. Asking
the same question again, a few seconds later, usually gets one.

> A timeout is not a decline. It is a bank that has not answered yet.

The timeouts cluster between 21:00 and 23:30, when the bank runs its nightly
fraud model refresh. During that window its authorisation endpoint takes 20
to 25 seconds to answer, and our client gave up after 15.

## One idempotent retry

The fix is a single retry, thirty seconds after a timeout, using the order id
as the idempotency key. If the first call did go through, the bank sees the
same key and returns the first result instead of charging the card twice.

```java
var key = order.id();
var result = bank.authorize(card, amount, key);
if (result.isTimeout()) {
    sleep(Duration.ofSeconds(30));
    result = bank.authorize(card, amount, key);
}
return result;
```

> [!NOTE]
> Only timeouts are retried. A real decline (insufficient funds, a fraud
> rule) is final, and a second attempt would look like card testing to the
> bank.

Thirty seconds covers the bank's batch window: we measured 21 seconds at the
99th percentile. A second timeout is returned as it is, and the customer sees
"still checking", not "failed".

## What we measured

In the first four weeks, nine in ten night-time timeouts ended in a successful
payment on the retry. Together with the warehouse change of the same month
(a lent picker when the queue passes 500 orders), late orders fell from roughly
one in eleven to about one in eighty.

The customer never saw the declined card. She saw Thursday.

## What we would do again

- Split every failure by its real cause before choosing a fix.
- Make the retry idempotent first; the retry itself is the easy part.
- Measure the window you are waiting for, then wait a little longer.
