package main

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

// canvasName allows a safe class + file name.
var canvasName = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*$`)

// canvasStarterSVG is a working, commented animated canvas: a token travels
// through three stages, stage C turns ok and shows a check. CANVAS is the name.
const canvasStarterSVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1000 320" class="CANVAS" aria-label="A starter canvas: an item moves through three stages">
<style>
/* Colour with the deck tokens, plus a fallback so the file also reads on its
   own. NEVER write a less-than sign inside this style block. */
.CANVAS{font-family:var(--si-font-body,system-ui,sans-serif)}
.CANVAS text{fill:var(--si-fg,#16171b)}
.CANVAS .box{fill:var(--si-surface,#fff);stroke:var(--si-line,#d9d6ce);stroke-width:2.5}
.CANVAS .lbl{font-size:24px;font-weight:700;text-anchor:middle}
.CANVAS .token rect{fill:var(--si-accent,#3657d6)}
.CANVAS .token text{fill:var(--si-on-accent,#fff);font-size:18px;font-weight:700;text-anchor:middle}
/* The token slides between stages: ONE css transition, keyed to the state.
   List the stages in order so the furthest match wins when several are set. */
.CANVAS .token{transition:transform .8s cubic-bezier(.3,.1,.2,1)}
.CANVAS[data-state~="s2"] .token{transform:translateX(330px)}
.CANVAS[data-state~="s3"] .token{transform:translateX(660px)}
</style>

<!-- three stages. the last one uses data-ok: its shapes turn --si-ok once the
     state contains "s3" (a helper attribute, no css needed). -->
<g><rect class="box" x="40" y="70" width="240" height="130" rx="16"/><text class="lbl" x="160" y="145">stage A</text></g>
<g><rect class="box" x="380" y="70" width="240" height="130" rx="16"/><text class="lbl" x="500" y="145">stage B</text></g>
<g data-ok="s3"><rect class="box" x="720" y="70" width="240" height="130" rx="16"/><text class="lbl" x="840" y="145">stage C</text></g>

<!-- data-show reveals the check only when the state contains "s3" -->
<path data-show="s3" d="M812 132 l18 20 l34 -40" fill="none" stroke="var(--si-ok,#178a4c)" stroke-width="7" stroke-linecap="round" stroke-linejoin="round"/>

<!-- the item that travels; its transform is animated by the css above -->
<g class="token"><rect x="100" y="238" width="120" height="46" rx="23"/><text x="160" y="269">item</text></g>
</svg>
`

// canvasDemoDeck drives the starter canvas through its states so `si play`
// shows it moving immediately. CANVAS is the name.
const canvasDemoDeck = `---
title: CANVAS canvas
theme: studio
---

# A canvas from scratch

Edit ` + "`CANVAS.svg`" + `, save, and this reloads under ` + "`si play`" + `.

---

## The item moves through three stages

<!-- svg: CANVAS.svg s1 -->
<!-- steps: +s2,-s1 +s3,-s2 -->
- It advances to **stage B**
- It reaches **stage C** — done

Notes: The whole animation lives in CANVAS.svg. The ` + "`steps`" + ` line sets the
canvas state (s1 then s2 then s3); the SVG's own CSS transition slides the
token, ` + "`data-ok`" + ` turns stage C green, and ` + "`data-show`" + ` reveals the check.
Add parts, add state words, drive them from the steps line. Full recipe:
docs/authoring-canvases.md (or run ` + "`si rules`" + ` and read section 10).
`

func scaffoldCanvas(name string, stdout, stderr io.Writer) int {
	if !canvasName.MatchString(name) {
		return usageErr(stderr, fmt.Sprintf("new-canvas: %q is not a valid name (letters, digits, - and _; must start with a letter)", name))
	}
	svg, md := name+".svg", name+".md"
	for _, f := range []string{svg, md} {
		if _, err := os.Stat(f); err == nil {
			fmt.Fprintf(stderr, "si: %s already exists — refusing to overwrite\n", f)
			return exitProblems
		}
	}
	files := map[string]string{
		svg: strings.ReplaceAll(canvasStarterSVG, "CANVAS", name),
		md:  strings.ReplaceAll(canvasDemoDeck, "CANVAS", name),
	}
	for f, body := range files {
		if err := os.WriteFile(f, []byte(body), 0o644); err != nil {
			fmt.Fprintf(stderr, "si: %v\n", err)
			return exitProblems
		}
	}
	fmt.Fprintf(stdout, "si: wrote %s and %s\n", svg, md)
	fmt.Fprintf(stdout, "    try: si play %s   (then edit %s and watch it reload)\n", md, svg)
	return exitOK
}
