---
type: lab
title: Build your first page
description: Write a page about a topic you like, with a heading, paragraphs, a list and a link.
---

# Build your first page

| | |
|---|---|
| **Time** | 20 minutes |
| **You need** | a text editor and a browser |

## Step 1: Create the file

Make a folder called `my-page` and, inside it, a file called `index.html`.
Open it in your editor and in your browser.

- [ ] I can see an empty page in the browser

## Step 2: Add the skeleton

Every page has the same frame around its content.

```html
<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <title>My page</title>
  </head>
  <body>
  </body>
</html>
```

- [ ] The tab shows "My page"

## Step 3: Write the content

Inside `body`, add a heading, two paragraphs, a list of three things and a
link to a page you like.

- [ ] The heading, the paragraphs, the list and the link all show

<!-- solution -->

```html
<body>
  <h1>Sourdough at home</h1>
  <p>Bread needs four things: flour, water, salt and time.</p>
  <p>The starter does the rest, if you feed it every day.</p>
  <ul>
    <li>500 g flour</li>
    <li>350 g water</li>
    <li>10 g salt</li>
  </ul>
  <a href="https://en.wikipedia.org/wiki/Sourdough">Read more about sourdough</a>
</body>
```

<!-- end -->
