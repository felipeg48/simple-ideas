---
description: Every part of a page is an element, written as tags with attributes.
---

# Tags, elements and attributes

HTML marks up text. A **tag** is a word between angle brackets; most come in
pairs, an opening tag and a closing tag with a slash. Together with the text
between them they make an **element**.

```html
<p>A paragraph of text.</p>
<a href="https://example.com">A link</a>
```

## Attributes

An attribute adds information to an element. It sits inside the opening tag
as a name, an equals sign and a value in quotes. The link above has one:
`href`, the address it leads to.

## Nesting

Elements go inside other elements, and they must close in the reverse order
they opened. An emphasised word inside a paragraph closes before the
paragraph does.

```html
<p>This is <em>really</em> important.</p>
```

## A quick check

```quiz
## Which part of `<a href="/about">About us</a>` is the attribute?
- [ ] `a`
- [x] `href="/about"`
- [ ] `About us`
> Explanation: the attribute sits inside the opening tag, as a name and a value.
```
