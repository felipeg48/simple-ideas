---
type: demo
title: "Live: order #4711 meets a slow bank"
session: The Thursday Promise
slides: 6 (Payment: declined, then paid)
duration: 6 min
---

# Live: order #4711 meets a slow bank

## What this demo must accomplish

1. **A timeout is not a decline.** The room sees the bank answer late, not
   say no.
2. **The retry is safe.** The second call carries the same key, and the card
   is charged once.

## Before the session (T-30 min)

Start the order journey lab and check every station answers.

```bash
docker compose up -d --wait
for p in 8080 8081 8082; do curl -s -o /dev/null -w "$p %{http_code}\n" localhost:$p/health; done
```

```text
8080 200
8081 200
8082 200
```

## The script

### Slide 6, "Payment: declined, then paid" (4 min)

> Say: "This is the same order you saw on the slide, on a laptop. I am going to make the bank slow, the way it is every night at nine."

```bash
curl -X POST localhost:8081/admin/latency -d '{"ms": 25000}'
time curl -X POST localhost:8082/orders/4711/pay
```

```text
{"order": 4711, "status": "paid", "attempts": 2, "elapsed_ms": 30412}
real    0m30.4s
```

Point at `"attempts": 2` while the room waits for the thirty seconds.

> Say: "Two attempts, one payment. The bank saw the same key twice and charged the card once."

### The tracking page (2 min)

Open `localhost:8085/orders/4711`: the payment tile is green.

> Say: "And the customer never saw a failure. She saw Thursday."

## If it goes wrong

- **The pay call answers at once with `"attempts": 1`:** the latency did not
  apply; run the first command again and check it prints `{"ms": 25000}`.
- **A port answers `000`:** the stack is still starting; wait ten seconds.

## Reset between rehearsals

```bash
curl -X POST localhost:8081/admin/latency -d '{"ms": 200}'
curl -X POST localhost:8080/admin/reset
```
