---
description: How the Thursday promise is computed, and when it changes.
---

# Promises and clocks

## The promise tile

The promise is computed when the order is placed and recomputed after every
station: ((promise, computed at checkout)). It only moves later when a station
is late; it never moves earlier.

## Cut-off times

| Order placed before | Leaves on | Delivered |
|---|---|---|
| Monday 23:59 | Wednesday's first wave | Thursday |
| Tuesday 07:10 (packed) | Wednesday's first wave | Thursday |
| Tuesday 07:11 or later | Thursday | Friday |

Table: Cut-off times for a Thursday promise

The second row is the one ((warehouse)) staff watch: an order packed by 07:10
on Tuesday still makes Wednesday's first wave ([](#tbl-cut-off-times-for-a-thursday-promise)).
