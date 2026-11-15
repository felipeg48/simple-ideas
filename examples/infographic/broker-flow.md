---
type: infographic
title: How a message travels through a broker
subtitle: From the app that sends it to the one that handles it, in four hops
aspect: phone
animate: build
theme: studio
sources: Typical figures for a single broker on one machine
---

## The path · flow · dir=tb
[server] Producer -[publish]-> [api] Exchange -[route]-> [queue] Queue -[deliver]-> [function] Consumer
Consumer -[ack]-> Queue

## 50,000 · stat · accent
messages a second through one queue on an ordinary laptop

## 2 ms · stat
from publish to deliver when nobody waits in the queue

## What happens on the way · steps
1. **Publish**: the producer sends the message with a routing key
2. **Route**: the exchange matches the key to one or more queues
3. **Store**: the queue keeps it, on disk if it is durable
4. **Deliver**: the consumer takes it and acks when done

Notes: A message leaves the producer, the exchange picks its queue, the queue keeps it, and the consumer acknowledges it once the work is done.
