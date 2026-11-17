---
title: Video player
description: A video player with its own controls, captions and chapters, in place or in a dialog over the dimmed page, from one fenced block; and the same player for pages si did not build.
---

# Video player

One fenced block is a player:

````markdown
```video src="img/player/page-load.mp4" title="A page loads, step by step"
From the address bar to the first paint, in sixteen seconds.
```
````

```video src="img/player/page-load.mp4" title="A page loads, step by step"
From the address bar to the first paint, in sixteen seconds.
```

The captions and the chapters come from the files beside the video,
`page-load.vtt` and `page-load.chapters.txt`, found at build time and carried
in the page. The marks on the bar are the chapters; the list button jumps to
one; CC shows the captions. Click the player, then try Space, ← →, J and L,
M, C and F.

## In a dialog, as a gallery

`mode=modal` turns the poster into a button that opens the player large over
the page, which is dimmed and blurred behind it. Every modal player on the
page is one gallery: ‹ › (or ← → while paused) step through them.

````markdown
```video src="img/player/page-load.mp4" mode=modal start=0:03 title="Find the computer"
The name becomes an address.
```
````

```video src="img/player/page-load.mp4" mode=modal start=0:03 width=24rem title="Find the computer"
The name becomes an address.
```

```video src="img/player/page-load.mp4" mode=modal start=0:07 width=24rem title="Ask for the file"
One request, one answer.
```

```video src="img/player/page-load.mp4" mode=modal start=0:13 width=24rem title="Style, lay out, paint"
The page appears.
```

## On any page

`si --export player` writes `si-player.css` and `si-player.js`, the same two
files si puts in its own pages. Load them on a page si did not build and
every element with `data-video` becomes a player; a card's picture is its
poster, its heading the title, and a click opens the dialog:

```html
<figure class="video-card" data-video="clips/01-intro.mp4" data-captions="clips/01-intro.vtt">
  <img src="shots/01-intro.png" alt="The complete film">
  <figcaption><h3>The complete film</h3><p>The whole story, end to end.</p></figcaption>
</figure>
```

Every option, key and attribute is in [the rules](../rules/25-code-windows-and-code-shows.md#video-player).
