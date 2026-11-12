---
title: The orders broker, live
author: Northwind Engineering
theme: studio
setup:   mkdir -p .live && echo 3 > .live/orders
start:   sh broker.sh as broker · wait for "ready"
setup:   echo 4 > .live/orders
stop:    echo "draining" > .live/state
cleanup: rm -rf .live
---

# The orders broker, live

What happens to an order between "Pay" and the warehouse.

Notes: This talk runs against a real broker on this machine, started a moment before the first slide. Every number you see comes from it.

Points:
- The broker started before the title slide; its log is in the terminal drawer
- Keep it short: three slides, then questions

---

## One queue, many hands

- The shop puts each paid order on the **orders** queue
- The warehouse takes them off, one at a time
- Nothing is lost when the warehouse is slow: orders wait in line

Notes: The shop never talks to the warehouse directly. It puts each paid order on one queue, and the warehouse takes them off at its own pace. [CLICK]

Points:
- Open the terminal drawer and run `tail -f $SI_RUN_LOGS/broker.log` if someone asks
- The count goes from three to four when the second seed runs

---

## When the warehouse is slow

| Orders waiting | What the shop does |
|---|---|
| 0 to 100 | nothing: the warehouse keeps up |
| 100 to 1,000 | warns the on-call engineer |
| over 1,000 | slows the checkout down |

Notes: A slow warehouse is not an emergency. Up to a hundred orders waiting is a normal morning; past a thousand, the shop slows its own checkout so nothing is lost.

Points:
- Thresholds are examples, not Northwind's real numbers
- Likely question: what happens on a broker restart (answer: the queue is durable)

---

# Questions

Notes: That is the whole journey of an order, from the shop to the warehouse. I am happy to take questions.
