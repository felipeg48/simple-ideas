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
tags: payments, reliability, go
canonical: https://northwind.example/engineering/one-retry
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

Sorted by cause, the night-time declines fall into three families, and only
one of them deserves a second try.

```mindmap layout=right lines=orthogonal
# A night-time decline
- Bank timeout {badge=87%}
  - The bank is refreshing its fraud model
  - Retry once, same idempotency key
- Real decline {badge=9%}
  - Insufficient funds
  - Card blocked
  - Ask the customer for another card
- Fraud rule {badge=4%}
  - [!sticky] Never retried: a person reviews it the next morning.
```

## One idempotent retry

The fix is a single retry, thirty seconds after a timeout, using the order id
as the idempotency key. If the first call did go through, the bank sees the
same key and returns the first result instead of charging the card twice.

Three pieces make the retry safe, and none of them works without the others.

```mindmap view=3d kind=bricks
# One safe retry
- Wait thirty seconds {badge=30_s}
  - The bank finishes its refresh
- Same idempotency key
  - The order id, never a new one
- One try only
  - A second timeout goes to a person
```

<!-- tabs -->
```java title="Java" hl="3-6"
var key = order.id();                                  // <1>
var result = bank.authorize(card, amount, key);
if (result.isTimeout()) {
    sleep(Duration.ofSeconds(30));                     // <2>
    result = bank.authorize(card, amount, key);
}
return result;                                         // <3>
```
<1> The order id is the idempotency key: a retry is the same request, never a second charge.
<2> Thirty seconds covers the bank's batch window; we measured 21 seconds at p99.
<3> A second timeout is returned as it is: the customer sees "still checking", not "failed".

```python title="Python" hl="3-5"
key = order.id                                       # <1>
result = bank.authorize(card, amount, key)
if result.is_timeout:
    time.sleep(30)                                   # <2>
    result = bank.authorize(card, amount, key)
return result                                        # <3>
```
<1> The order id is the idempotency key: a retry is the same request, never a second charge.
<2> Thirty seconds covers the bank's batch window; we measured 21 seconds at p99.
<3> A second timeout is returned as it is: the customer sees "still checking", not "failed".

> [!NOTE]
> Only timeouts are retried. A real decline (insufficient funds, a fraud
> rule) is final, and a second attempt would look like card testing to the
> bank.

Thirty seconds covers the bank's batch window: we measured 21 seconds at the
99th percentile. The test that guards the rule runs a hundred times before
every release:

```sh window=terminal title="checkout"
go test ./payments -run TestRetryKeepsTheKey -count=100
ok  	northwind/payments	3.214s
```

A second timeout is returned as it is, and the customer sees
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
