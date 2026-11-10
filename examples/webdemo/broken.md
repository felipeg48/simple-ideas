---
type: demo
title: Northwind Shop, a spring coupon
app: app
viewport: desktop
---

# Northwind Shop, a spring coupon

## Take 10% off with a coupon

What it does: a customer types the spring coupon on an order and the total drops by ten percent.

Say: Sign in with the demo account.
- type "demo" into "Username"
- type $SHOP_PASSWORD into "Password"
- click "Sign in"

Say: Spring brings ten percent off every order.
- click "Order #4711"
- type "SPRING10" into "Coupon"
- click "Apply"
- expect text "10% off"
- expect text "Total: €42.66"
- expect no errors
