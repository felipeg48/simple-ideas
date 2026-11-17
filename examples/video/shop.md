---
type: demo
title: Northwind Shop, next-day delivery
app: ../webdemo/app
viewport: 960x600
timeout: 5s
voice: kokoro:af_heart
pronounce: Northwind = north wind
---

# Northwind Shop, next-day delivery

## Sign in

What it does: a returning customer signs in and lands on their orders.

Say: Northwind keeps your orders behind one sign-in.
- type "demo" into "Username"
- type $SHOP_PASSWORD into "Password"
- click "Sign in"
- expect text "Welcome back, demo"

## Ship it express

What it does: an open order moves to Express shipping with gift wrap, and the total follows.

Say: Order 4711 is still open, so its shipping can change.
- type "demo" into "Username"
- type $SHOP_PASSWORD into "Password"
- press Enter
- click "Order #4711"
- select "Express" in "Shipping"

Say: Express arrives the next working day, and gift wrap is one tick.
- check "Gift wrap"
- expect text "Total: €54.90"
