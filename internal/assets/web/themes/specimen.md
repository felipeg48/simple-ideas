---
title: Bread by Six
author: Mara Quint, Head Baker
date: Autumn 2026
transition: fade
---

# Bread by Six

How a small bakery gets two hundred loaves out of the oven before the first customer.

---

<!-- eyebrow: The morning · 02:00 to 06:00 -->
## Four hours, ==one oven==

- The dough is mixed at **two**, shaped at four, baked at five
- Every batch has a card: flour, water, salt, and the [time it went in](#)
- A loaf that misses its slot waits for the `second bake`
- *Nothing* leaves the shop warm before six

---

## What changed this year

<!-- layout: two-col -->

### Before

- One long bake for every loaf
- The rye proofed beside the oven door

<!-- column -->

### After

- Two short bakes, rye first
- A cool shelf for the slow doughs

---

## The schedule

| Time | Dough | Loaves |
|---|---|---|
| 02:00 | Country white | 80 |
| 03:15 | Rye | 40 |
| 04:30 | Seeded | 60 |
| 05:10 | Baguette | 20 |

> [!TIP]
> Rye goes in first: it is the slowest to cool.

---

<!-- layout: split -->
## The timer card

- One line per batch
- The oven beeps; the card says what comes next

```go window=mac title="timer.go" hl=3
// the next batch is the first one not yet baked
func next(batches []Batch, now time.Time) Batch {
	for _, b := range batches {
		if !b.Baked && b.At.Before(now) {
			return b
		}
	}
	return Batch{}
}
```

---

<!-- layout: big-number -->

**200**

loaves on the shelf by six, up from 140 last spring.

---

<!-- layout: quote -->

> People queue for the smell, then stay for the rye.

A regular, Tuesday morning

---

<!-- layout: center -->
## What makes a loaf late

```mindmap layout=right
# A late loaf
- Dough
  - Cold water
  - Slow starter
- Oven
  - One tray too many
- People
  - A late shift
```

---

# Thank you

Fresh at six, every day but Monday.

<!-- qr: https://example.com/bakery · caption=Our hours · size=small -->
