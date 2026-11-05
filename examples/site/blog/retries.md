---
type: article
title: Retries without the stampede
description: Why our payment retries took the provider down at 21:40, and the three lines of backoff that stopped it.
author: Ada Lindqvist
date: 2026-10-14
---

At 21:40 on a Friday the payment provider slowed down for eleven seconds.
Our checkout service retried every failed call immediately, so eleven slow
seconds became four minutes of outage.

## What happened

Each checkout pod retried a failed charge three times with no delay. With
120 pods and a queue of waiting orders, the provider saw its normal load
multiplied by four at exactly the moment it was weakest.

## What we changed

```go title="retry.go"
delay := base << attempt                        // 200 ms, 400 ms, 800 ms
delay = time.Duration(rand.Int63n(int64(delay))) // full jitter
time.Sleep(delay)
```

Exponential backoff spreads the retries over time; full jitter spreads them
across pods, so they stop arriving in waves.

## What we measure now

We alert on the retry rate, not only on the error rate. A rising retry rate
was visible ninety seconds before the outage, and nobody was looking at it.
