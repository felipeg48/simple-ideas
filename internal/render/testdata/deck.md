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

# The end
