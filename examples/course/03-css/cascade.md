---
description: A rule picks elements with a selector and sets their properties; when rules disagree, the cascade decides.
instructor: Ask the room to guess the colour of the second heading before you show the answer; most guess the first rule.
---

# Selectors and the cascade

A CSS rule has two parts: a **selector** that picks elements, and a block of
**declarations** that set their properties.

```css
h1 {
  font-family: Georgia, serif;
  color: #1d3557;
}
```

## Three selectors to start with

| | |
|---|---|
| `p` | every paragraph |
| `.note` | every element with `class="note"` |
| `#intro` | the one element with `id="intro"` |

## When rules disagree

Two rules can set the same property on the same element. The browser picks
one by **specificity** first (an id beats a class, a class beats a tag) and,
when that is equal, by **order**: the rule written later wins.

```css
h2 { color: teal; }
h2 { color: crimson; }  /* later, so every h2 is crimson */
```

> [!NOTE]
> That is the cascade in "Cascading Style Sheets": rules flow down in order,
> and the more specific ones take precedence.
