---
type: lab
title: Style your page
description: Give your page a typeface, colours and a comfortable reading width.
---

# Style your page

## Step 1: Link a style sheet

Create `style.css` beside `index.html` and link it from `head`.

```html
<link rel="stylesheet" href="style.css">
```

- [ ] The link is in `head`

## Step 2: Set the typeface and width

```css
body {
  font-family: system-ui, sans-serif;
  line-height: 1.6;
  max-width: 38rem;
  margin: 2rem auto;
  padding: 0 1rem;
}
```

- [ ] The text sits in a column that is easy to read

## Step 3: Add colour

Pick one colour for the headings and one for links.

- [ ] Headings and links have their own colours

<!-- solution -->

```css
h1, h2 { color: #1d3557; }
a { color: #e63946; }
a:hover { text-decoration-thickness: 2px; }
```

<!-- end -->
