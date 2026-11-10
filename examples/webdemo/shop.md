---
type: demo
title: Northwind Shop, from sign-in to receipt
app: app
viewport: desktop
timeout: 5s
voice: kokoro:af_heart
pronounce: Northwind = north wind
---

# Northwind Shop, from sign-in to receipt

## Sign in

What it does: a returning customer signs in and lands on their orders.

Say: Northwind keeps your orders behind one sign-in.
- go /
- note "Username: demo" at "Username"
- type "demo" into "Username"
- type $SHOP_PASSWORD into "Password"
- click "Sign in"
- expect text "Welcome back, demo"
- expect url /orders.html
- screenshot orders

## Pay for an order

What it does: a customer picks Express shipping and gift wrap, pays with a card and sees "Paid" at once.

Say: Sign in with the demo account.
- type "demo" into "Username"
- type $SHOP_PASSWORD into "Password"
- press Enter
- wait for "Your orders"

Say: Order 4711 is still waiting for payment.
- click "Order #4711"
- expect url /order.html?id=4711
- select "Express" in "Shipping"
- check "Gift wrap"
- expect "Gift wrap" checked
- expect text "Total: €54.90"

Say: Help is one question mark away.
- hover "Help"
- expect "Questions? Call us on 555-0100." visible

Say: Pay sits at the bottom, under the delivery promise.
- scroll to [data-testid=pay]
- click role=button "Pay"
- expect "Confirm" disabled
- type "4242 4242 4242 4242" into "Card number"
- expect "Confirm" enabled

Say: A test card goes through in a second.
- click "Confirm"
- wait for "Paid"
- expect url /receipt.html?order=4711
- expect no errors
- screenshot receipt

## Find an order

What it does: the search box finds an order by its number, and paid orders can be hidden.

Say: Every order is one search away.
- type "demo" into "Username"
- type $SHOP_PASSWORD into "Password"
- click "Sign in"
- press /
- type "4712" into "Order number"
- expect text "Order #4712"
- expect "Order #4711" hidden

Say: Hide what is already paid.
- type "" into "Order number"
- uncheck "Show paid orders"
- expect "Order #4712" hidden
- expect "Order #4711" visible
- wait 0.2s
- expect no errors
