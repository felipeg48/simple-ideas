---
type: lab
title: Trace an Order Through the Hand-offs
author: Ada Lindqvist, Operations
theme: studio
accent: "#3657d6"
---

# Trace an Order Through the Hand-offs

A hands-on companion to [The Thursday Promise](showcase.md). You will follow
order #4711 from checkout to a doorstep, find the hand-off that eats the
slack, and fix it.

> [!NOTE]
> You need read access to the tracking API and a shell with `curl` and `jq`.
> Everything here is read-only until the last step.

## Before you start

Pick an order that was promised for Thursday and arrived on Friday. Export its
id and the API base URL:

```bash
export ORDER=4711
export API=https://tracking.northwind.example/v2
```

> [!TIP]
> The late orders from the last seven days are listed at
> `$API/orders?promise=thursday&state=late`.

## Step 1: read the timeline

Every hand-off writes one event. Fetch them in order and keep the timestamps:

```bash
curl -s "$API/orders/$ORDER/events" | jq -r '.[] | [.at, .team, .state] | @tsv'
```

You should see five teams. The columns that matter are the hand-off times and
the budget each team was given:

| Team | Hand-off | Budget | Owner | Escalation channel | Dashboard | Last review |
|------|----------|--------|-------|--------------------|-----------|-------------|
| Checkout | Order placed | 2 min | Mara | #checkout-oncall | https://dash.northwind.example/checkout | 2026-03-02 |
| Payment | Card captured | 5 min | Jonas | #payments-oncall | https://dash.northwind.example/payment | 2026-03-09 |
| Warehouse | Parcel packed | 4 h | Ines | #warehouse-floor | https://dash.northwind.example/warehouse | 2026-02-23 |
| Courier | Parcel collected | 18 h | Tomas | #courier-desk | https://dash.northwind.example/courier | 2026-03-16 |
| Delivery | Left at door | 6 h | Priya | #last-mile | https://dash.northwind.example/delivery | 2026-03-16 |

### Spot the slack

Subtract each budget from the time the team actually took. The team with the
largest overrun is where the promise broke:

```bash
curl -s "$API/orders/$ORDER/events" | jq -r 'sort_by(.at) | [.[] | .at] | . as $t | [range(1; length)] | map(($t[.] | fromdate) - ($t[. - 1] | fromdate)) | @csv' | tee /tmp/gaps.csv | column -s, -t
```

> [!IMPORTANT]
> Compare against the budget in the table, not against the average. One slow
> team can hide behind a fast one.

## Step 2: find the cause

Payment declines are the usual suspect. Count them for the order:

```sql
SELECT count(*) AS declines, min(at) AS first_decline, max(at) AS retried
FROM payment_attempts
WHERE order_id = 4711 AND status = 'declined';
```

> [!WARNING]
> A decline followed by a retry within a minute is normal. A retry hours later
> means the customer was asked to re-enter their card, and the clock kept
> running.

## Step 3: apply the fix

Move the retry to the same session so the customer is never asked twice:

```bash
curl -s -X PATCH "$API/policies/payment-retry" -H 'content-type: application/json' -d '{"mode":"same-session","max_attempts":3}'
```

> [!CAUTION]
> This changes the policy for every order, not just #4711. Run it in the
> staging API first and compare the late rate for a day.

> A promise is a hand-off with a deadline, and every hand-off needs an owner.

## What you should see

The next late-order report shows the Payment overrun gone, and the Thursday
rate back above 95 percent. The floor plan below shows where each hand-off
happens:

![The warehouse floor](assets/floor.svg)
