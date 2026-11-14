---
title: Orders wait in a queue
browser: Console=http://127.0.0.1:7421/broker.html
---

# Orders wait in a queue

Notes: (0.5 min)
An order does not go straight to the warehouse. It waits in a queue first,
and that small wait is what keeps the shop up on a busy day. [CLICK]

---

## The queue holds the rush

- The shop puts each order in the queue
- The warehouse takes them at its own pace
- Nothing is lost while it catches up

Notes: (1 min)
The shop puts each order in the queue and moves on. The warehouse takes them
at its own pace, and nothing is lost while it catches up. Let's see it in
action. [TERM Lab 1: echo "sending 3 orders" && sleep 1 && echo "sent 3 orders to the queue"]
Three orders go out from the terminal. [BROWSER Console] And here is the
broker's console: the queue is holding them. [TERM close] [BROWSER close] [CLICK]

Points:
- The queue decouples the shop from the warehouse
- The drawer and the console open on their own here; they close before the next slide
- Likely question: what if the warehouse is down for an hour? The queue keeps the orders

---

## One wait, many calm days

Notes: (0.5 min)
A queue costs a few milliseconds per order, and buys a shop that stays up
when the warehouse is slow.
