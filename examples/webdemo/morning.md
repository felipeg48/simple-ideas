---
type: demo
title: Northwind's morning, from an empty shop to the first order
app: app
viewport: desktop
timeout: 5s
voice: kokoro:af_heart
pronounce: Northwind = north wind
shell: sh
prepare: mkdir -p .morning && printf '4711\n4712\n' > .morning/orders.txt
step:    echo "seeded $(wc -l < .morning/orders.txt | tr -d ' ') orders" > .morning/seed.log
setup:   cp .morning/orders.txt .morning/today.txt
stop:    echo "the shop closes" >> .morning/seed.log
cleanup: rm -rf .morning
---

# Northwind's morning, from an empty shop to the first order

## Before

- terminal "Ops"
- run "cat .morning/seed.log" · wait
- expect output "seeded 2 orders"

## The day's orders

What it does: the orders seeded before the shop opened are waiting for the first customer.

Say: Two orders came in overnight, and they are already waiting.
- go /
- type "demo" into "Username"
- type $SHOP_PASSWORD into "Password"
- click "Sign in"
- expect text "Order #4711"
- expect text "Order #4712"

Say: The first one still needs paying.
- click "Order #4711"
- expect url /order.html?id=4711
- spotlight [data-testid=pay] · label "Ready to pay"
- wait 1s

## After

- terminal "Ops"
- run "wc -l < .morning/today.txt" · wait
- expect output "2"
