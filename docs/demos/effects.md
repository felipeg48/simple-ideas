---
type: demo
title: Demos edited like a video
app: ../../examples/webdemo/app
viewport: 1440x900
timeout: 10s
voice: kokoro:af_heart
pronounce: si = ess eye; Northwind = north wind
transition: fade 0.5s
---

# Demos edited like a video

The Northwind shop, filmed the way an editor would cut it: scenes that move
into each other, a camera that leans in, and a spotlight on what matters.

## Scenes that move into each other

What it does: every change of page or window is a transition, a fade unless the script asks for another.

Say: The shop opens on its sign-in page.
- go /
- type "demo" into "Username"
- type "shop-demo" into "Password"
- click "Sign in"
- expect text "Your orders"
- wait 1s

Say: Each change of page moves into the next, here with a slide.
- go /order.html?id=4711 · slide
- expect text "Order #4711"
- wait 1500ms

Say: Back to the list with a wipe, and to the next order with a pop.
- go /orders.html · wipe
- expect text "Your orders"
- wait 1s
- go /order.html?id=4712 · pop
- expect text "Order #4712"
- wait 1500ms

## A camera that leans in

What it does: the picture glides close to what the narration is about, slides along, and pulls back.

Say: The total sits in a corner of the page. The camera moves in on it.
- go /
- type "demo" into "Username"
- type "shop-demo" into "Password"
- click "Sign in"
- go /order.html?id=4711
- select "Express" in "Shipping"
- zoom to #total · 1s
- wait 2s

Say: It slides over to the shipping choice, at the same distance.
- pan to "Shipping"
- wait 2s

Say: And it pulls back to the whole order.
- zoom out
- wait 1500ms

## A spotlight on what matters

What it does: everything but one element dims, with a short label, so the eye goes straight there.

Say: Paying takes one tap.
- go /
- type "demo" into "Username"
- type "shop-demo" into "Password"
- click "Sign in"
- go /order.html?id=4711
- scroll to [data-testid=pay]
- spotlight [data-testid=pay] · circle · label "One tap to pay"
- wait 3s

Say: The total, with the rest of the order out of focus.
- spotlight #total · blur · label "Shipping and gift wrap included"
- wait 3s
- spotlight off
- wait 1s

## One panel, from wide to close

Region: main · zoom

What it does: the clip opens on the whole page, then settles on the panel that matters.

Say: The orders page opens wide, then the camera settles on the list itself.
- go /
- type "demo" into "Username"
- type "shop-demo" into "Password"
- click "Sign in"
- expect text "Your orders"
- wait 3s
