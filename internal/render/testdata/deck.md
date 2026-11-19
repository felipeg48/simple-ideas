---
title: Golden deck
author: Test Author
date: 2026
theme: paper
accent: "#e4572e"
transition: slide
footer: Internal
footer-position: bottom-left
watermark: Draft
watermark-opacity: 0.08
speakers: Ana, Felipe
---

# Golden **deck**

<!-- footer: off -->

A subtitle with `code`.

Notes: (0.5 min)
Title notes.

---

<!-- steps: on . bad -->
- first
- second
- third

---

## Canvas

<!-- svg: assets/canvas.svg on -->
<!-- step: busy -->
Step with state.
<!-- step: -on,bad -->

Notes: (1 min)
Canvas notes. [CLICK]

Notes[Felipe]: (0.5 min)
Felipe takes the second step. [CLICK]

Points[Felipe]:
- the bad state is on purpose

Points: Canvas talking point.
- second point

---

## Canvas again

<!-- svg: assets/canvas.svg +busy -->

---

<!-- layout: two-col -->
<!-- background: #101820 -->
<!-- class: extra -->
<!-- eyebrow: Layouts & more -->
## Columns

Left.

<!-- column -->

```go
func main() { fmt.Println("hi") } // done
```

---

<!-- layout: split -->
## Split

- text

<!-- image: width=40% align=right ratio=4:3 -->
![A picture](assets/pic.svg "The caption")

---

<!-- include: part.md -->

---

<!-- layout: big-number -->
<!-- transition: none -->

**42**

the answer

---

<!-- labs: [Create a Cluster](https://lab.example.com/one), [Scale Out](https://lab.example.com/two) -->

---

## Components, ==highlighted==

<!-- logo: × Partner · alt=Brand -->

<!-- badge: Prepared for Partner · Session 1 of 5 -->

<!-- qr: https://example.com/slides · caption=The slides · size=small · dots -->

<!-- poll: next-step · Ship it · Test more · multiple · hidden -->

Body with a ==phrase==.

---

## A mind map

```mindmap layout=right lines=orthogonal
# Order
- Checkout {badge=Thu}
  - Payment
- [!sticky] Ask ops.
```

---

## An infographic

```infographic animate=build
## 2.4 s · stat · accent
Median load before the fix.

## Before and after
| Before | After |
|---|---|
| 2.4 s | 0.6 s |

## Traffic · chart=donut
| Source | % |
|---|---|
| Search | 54 |
| Direct | 46 |

## How · steps · wide
1. Measure
2. Fix
```

---

## A chart race

```chart bar race unit=%
| Cause | Q1 | Q2 |
|---|---|---|
| Carrier | 34 | 12 |
| Payment | 9 | 21 |
```

```chart donut3d explode=Late title="Orders"
| Status | Share |
|---|---|
| On time | 85 |
| Late | 15 |
```

---

## A code show

```go id=send window=mac title="send.go" numbers backdrop=ocean step="Once"
func send() error { return post() }
```
<!-- callout: 1 one try at=40,-20 -->

```go id=send window=mac title="send.go" numbers backdrop=ocean hl=2
func send() error {
	return retry(post)
}
```

---

## Code from files

<!-- include: code/compose.yaml window=mac numbers hl=3 theme=night -->
<!-- callout: 4 the port the browser opens -->

<!-- include: code/start-v1.sh id=start window=terminal as=cat step="Up" -->

<!-- include: code/start-v2.sh id=start window=terminal as=cat step="Pull first" hl=3 -->
<!-- callout: 4 waits for the health checks -->

---

<!-- transition: morph -->
## One promise

<!-- id: promise -->
Two days, door to door.

<!-- id: code -->
```sh
si play
```

---

<!-- transition: morph 0.9s -->
## One promise

<!-- id: promise -->
**Two days, door to door.**

---

<!-- transition: cube 1.2s -->
## Late orders

<!-- animate: words -->
Nobody owns the promise.

<!-- steps: reorder -->
- Address wrong **12%**
- Stock drift **61%**

<!-- steps: focus -->
- Shop
- Payments

<!-- animate: orbit -->
- Order
- Shop

<!-- step -->
<!-- animate: count -->
**12,480** orders
---

# The end
