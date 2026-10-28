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
Canvas notes.

---

## Canvas again

<!-- svg: assets/canvas.svg +busy -->

---

<!-- layout: two-col -->
<!-- background: #101820 -->
<!-- class: extra -->
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

# The end
