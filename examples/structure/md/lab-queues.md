---
type: lab
title: Declare a Queue and Consume From It
author: Ada Lindqvist · Platform team
---

# Declare a Queue and Consume From It

The hands-on part of [RabbitMQ in One Hour](intro.md): declare a durable
queue, publish to it, and acknowledge what you consume.

## Declare the queue

```bash
rabbitmqadmin declare queue name=orders durable=true
```

## Publish and consume

```bash
rabbitmqadmin publish routing_key=orders payload="order 4711"
rabbitmqadmin get queue=orders ackmode=ack_requeue_false
```

> [!NOTE]
> `ack_requeue_false` acknowledges the message, so it leaves the queue.
