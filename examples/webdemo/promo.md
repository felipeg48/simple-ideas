---
type: demo
title: Harbor
app: harbor
viewport: desktop
session: keep
header-offset: auto
intro: Harbor · Every order out the door on time
outro: Start free · A hundred orders a month, on us
logo: harbor/logo.svg
url: https://harbor.example
music: harbor/bed.mp3 · 20%
theme: forest
---

# Harbor

## Before

- go /
- click "Accept all"

## One queue for every shop

What it does: orders from every shop land in one queue, sorted by the time they must leave.

Lower-third: Ana Ruiz · Head of Operations, Harbor

Say: Harbor puts the orders from every shop you sell in into one queue. Each one is sorted by the time it must leave the building.
- go /
- spotlight #queue-title · label "One queue"
- scroll to .queue
- note "Leaves first, packed first" at "#1042" · position right

## One plan, priced by what you ship

What it does: start free with a hundred orders a month, then pay the same for every order.

Say: There is one plan. You start free with a hundred orders a month, and above that every order costs the same.
- go /
- scroll to #pricing
- spotlight #pricing h2 · label "Pricing"
- zoom to .plan
- zoom out

## Order from the shop

What it does: a shop framed in the page sends its order to Harbor.

Say: Here is a shop that sends its orders to Harbor. Pick a quantity, buy, and the order is in the queue.
- go /
- scroll to #shop
- type "3" into "Quantity" in frame .preview
- click "Buy" in frame .preview
- expect "Bought 3" visible in frame .preview
- spotlight #done in frame .preview · label "In the queue"
