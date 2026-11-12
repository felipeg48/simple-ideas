---
type: deck
title: From address to pixels
author: Mira Okafor
---

# From address to pixels

What happens between pressing Enter and seeing a page

Notes: (1 min) You type an address and press Enter, and a moment later a page appears. In these few slides we follow that moment, one step at a time. [CLICK]

Points:
- Four steps: find, ask, read, draw
- Keep it light: no protocol details

---

## Find the computer

The address names a computer. A lookup turns the name into a number the
network can route to.

Notes: (1 min) An address like example.com is a name for people. The network needs a number, so the browser first asks a directory which number belongs to that name. [CLICK]

Points:
- DNS, without the acronym
- The answer is cached, so the second visit skips it

---

## Ask for the file

<!-- steps -->
- The browser sends a request: "the page at this path, please"
- The server answers with a status and the file
- 200 means here it is; 404 means there is no such page

Notes: (1 min) With the number in hand, the browser sends a short request. [CLICK] The server answers with a status code and, if all is well, the file itself. [CLICK] You have met the status codes already: two hundred means here it is, and four hundred and four means there is no such page. [CLICK]

Points:
- One request per file: the page, then each picture and style sheet

---

## Read, then draw

The browser reads the HTML into a tree of elements, applies the CSS to each
one, works out where everything goes, and paints it.

Notes: (1 min) Now the browser reads the file into a tree of elements, applies the styles to each element, works out where every box goes, and paints the result. Change one line of CSS and those last three steps run again, which is why a save and a reload are all it takes. [CLICK]

Points:
- Tree, styles, layout, paint
- Lead into the video next
