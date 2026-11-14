---
title: RabbitMQ in One Hour
author: Ada Lindqvist · Platform team
---

# RabbitMQ in One Hour

Producers, queues and the acknowledgement that closes the loop.

Notes: In the next hour we follow one message from the service that sends it to the service that finishes the work.

Points:
- one message, start to finish
- the lab comes after the talk

---

## A message waits in a queue

<!-- svg: assets/queue.svg -->

<!-- steps: . +sent -->
- The producer hands the message over
- The queue keeps it until a consumer is free

Notes: The producer does not wait for the consumer. [CLICK] It hands the message to a queue. [CLICK] The queue keeps it safe until someone is ready to work on it.

Points:
- decoupling in time
- ask who has lost a message before

---

## The consumer says it is done

<!-- svg: assets/queue.svg +acked -->

Nothing leaves the queue for good until the consumer acknowledges it.

Notes: The message stays in the queue until the consumer says it finished the work. Only that acknowledgement removes it for good.

Points:
- at-least-once delivery
- redelivery after a crash

---

## The exchange decides where it goes

![An exchange routes by key](assets/exchange.png)

Notes: Producers never write to a queue directly. They publish to an exchange, and the exchange decides which queues get a copy.

Points:
- direct, topic, fanout

---

## One publish, end to end

![A publish, start to finish](assets/publish.mp4)

Notes: Here is one publish from start to finish, as the management screen shows it.

Points:
- keep it short, the lab repeats it
