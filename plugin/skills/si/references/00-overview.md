# simple-ideas markdown rules

Everything `si` understands, with an example for each rule. `si rules` prints
this file; `examples/showcase.md` uses every rule in a real talk.

> **AsciiDoc too.** A deck can be `*.adoc` / `*.asciidoc` instead of `*.md`; si
> converts the AsciiDoc to this same format and everything below applies. See
> `docs/asciidoc.md` for the supported subset. On the docs site every example
> has an **AsciiDoc** tab beside the Markdown one.

> **The layout rule.** Slides are fluid and responsive, never fixed. There is
> no 1280×720 canvas scaled by a transform: every slide fills the viewport and
> reflows, type follows the slide size (container units and `clamp()`), two
> columns stack on narrow or portrait screens, SVGs scale by their `viewBox`.
> A deck must read well from a ~400 px phone to a 4K display, in portrait and
> landscape. Image widths take `%`, `em` or `cqi`; `px` sizes are refused.

## Contents

1. [Decks and files](#1-decks-and-files)
2. [Front matter](#2-front-matter)
3. [Slides and headings](#3-slides-and-headings)
4. [Directives](#4-directives)
5. [Layouts and columns](#5-layouts-and-columns)
6. [Transitions](#6-transitions)
7. [Backgrounds](#7-backgrounds)
8. [Images](#8-images)
9. [Build steps](#9-build-steps)
10. [Animated SVG canvases](#10-animated-svg-canvases)
11. [Speaker notes and the teleprompter](#11-speaker-notes-and-the-teleprompter)
12. [Includes](#12-includes)
13. [Code, Mermaid, tables, links](#13-code-mermaid-tables-links)
14. [Presenting: keys, touch, overview, print](#14-presenting-keys-touch-overview-print)
15. [Styling: themes and your own CSS](#15-styling-themes-and-your-own-css)
16. [Configuring keys and features (si.conf)](#16-configuring-keys-and-features-siconf)
17. [What si check reports](#17-what-si-check-reports)
18. [Lab documents](#18-lab-documents)
19. [Articles](#19-articles)
20. [Guides (tutorials)](#20-guides-tutorials)
21. [Books (references)](#21-books-references)
22. [Demo scripts](#22-demo-scripts)
23. [Sites](#23-sites)
24. [Mind maps](#24-mind-maps)
25. [Code windows and code shows](#25-code-windows-and-code-shows)
26. [Publishing](#26-publishing)
27. [Web demos](#27-web-demos)
28. [Courses](#28-courses)
29. [Timelines](#29-timelines)
30. [Infographics](#30-infographics)
