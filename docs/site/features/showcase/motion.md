---
type: deck
title: Where order #4711 went
author: Northwind Logistics
theme: studio
---

<!-- layout: center -->
# Where order #4711 went

Monday night to Thursday afternoon

---

## The journey on one screen

<!-- svg: journey.svg placed -->

- Five stations, one order, three tiles the customer sees

---

## Payment: declined, then paid

<!-- svg: journey.svg +declined -->
<!-- steps: . -declined,retry,paid -->
- 21:04: the card fails a fraud check
- 21:05: a retry thirty seconds later goes through

---

## On the road

<!-- svg: journey.svg +shipped -->
<!-- step: delivered -->

---

## One order, three parcels

```scene
box Order €84
column Berlin, Lyon, Milan
Order €84 splits → Berlin, Lyon, Milan
parcels merge → Lyon · slow
```

---

## Tuesday to Thursday

```scene
row Shop, Payments, Warehouse, Courier
token hops Shop → Payments → Warehouse → Courier
Courier morphs → Delivered circle
```

---

## Late orders by cause

```chart bar race unit=%
| Cause     | Q1 | Q2 | Q3 |
|-----------|----|----|----|
| Carrier   | 34 | 22 | 12 |
| Warehouse | 18 | 25 | 9  |
| Payment   | 9  | 14 | 21 |
| Address   | 25 | 11 | 6  |
```

---

<!-- transition: morph -->
## Four teams

- shop, payments, warehouse, courier

<!-- id: owner -->
Nobody owns the promise.

---

<!-- transition: morph -->
<!-- layout: center -->
## Four teams

<!-- id: owner -->
**One team owns the promise.**
