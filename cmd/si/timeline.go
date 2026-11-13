package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/felipeg48/simple-ideas/internal/cdp"
	"github.com/felipeg48/simple-ideas/internal/deck"
)

// Timelines (type: timeline): `si build x.md --png` writes the timeline as
// one picture (landscape: the 16:9 frame with every era shown; portrait: the
// whole poster), --scale sets its pixels per CSS pixel; `--pdf` is the deck
// PDF (landscape, one page) or the printed page (portrait).

// exportTimeline writes <name>.png beside the built page (dest).
func exportTimeline(chrome, dest, out string, d *deck.Deck, scale float64, stdout io.Writer) error {
	abs, err := filepath.Abs(dest)
	if err != nil {
		return err
	}
	w, h := 1600, 900
	if a, b, ok := strings.Cut(d.Meta.Aspect, ":"); ok && d.Meta.Timeline.Orientation != "portrait" {
		x, _ := strconv.ParseFloat(a, 64)
		y, _ := strconv.ParseFloat(b, 64)
		if x > 0 && y > 0 {
			h = int(float64(w) * y / x)
		}
	}
	poster := d.Meta.Timeline.Orientation == "portrait"
	if poster {
		w, h = 1100, 1400
	}
	c, err := cdp.Launch(chrome, w, h)
	if err != nil {
		return err
	}
	defer c.Close()
	c.Scale = scale
	if _, err := c.OpenPage("", w, h); err != nil {
		return err
	}
	url := "file://" + filepath.ToSlash(abs)
	if !poster {
		url += "?si-view=thumb#1" // every build step shown, no sync
	}
	if err := c.Call("Page.navigate", map[string]any{"url": url}, nil); err != nil {
		return err
	}
	for i := 0; i < 200; i++ {
		v, _ := c.Eval("document.readyState")
		if string(v) == `"complete"` {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	// the picture is the timeline alone: no counter, progress bar or buttons
	if _, err := c.Eval(`(function(){var s=document.createElement('style');s.textContent='.si-deck>:not(.si-slide),body:has(.si-deck)>:not(.si-deck){display:none!important}';document.head.appendChild(s);window.siTimelineFit&&window.siTimelineFit();return document.fonts?document.fonts.ready.then(function(){return 1}):1})()`); err != nil {
		return err
	}
	time.Sleep(600 * time.Millisecond)
	if poster {
		v, err := c.Eval(`Math.ceil(document.documentElement.scrollHeight)`)
		if err != nil {
			return err
		}
		if n, _ := strconv.Atoi(string(v)); n > 0 {
			h = n
		}
		if err := c.Call("Emulation.setDeviceMetricsOverride", map[string]any{"width": w, "height": h, "deviceScaleFactor": max(scale, 1), "mobile": false}, nil); err != nil {
			return err
		}
		time.Sleep(400 * time.Millisecond)
	}
	png, err := c.Screenshot()
	if err != nil {
		return err
	}
	p := filepath.Join(out, d.Name+".png")
	if err := os.WriteFile(p, png, 0o644); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "  %-28s (png, %d × %d)\n", d.Name+".png", int(float64(w)*scale), int(float64(h)*scale))
	return nil
}

// newTimeline is `si new <name> --type timeline`: the example timeline (the
// founder's sample, its pictures) in a folder.
func newTimeline(name string, stdout, stderr io.Writer) int {
	dir := strings.TrimSuffix(name, "/")
	if _, err := os.Lstat(dir); err == nil {
		fmt.Fprintf(stderr, "si: %s already exists; si new never overwrites\n", dir)
		return exitProblems
	}
	if err := writeExampleKind(dir, "timeline"); err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	fmt.Fprintf(stdout, "si: created %s/ (a timeline: eras, 30 milestones, pictures, sources)\n  si play %s/timeline.md --open\n  si build %s/timeline.md --png --pdf\n", dir, dir, dir)
	return exitOK
}

// timelineCheck is what si check says about a timeline page: its counts.
func timelineCheck(f string, d *deck.Deck, stdout io.Writer) {
	if d.Timeline != nil {
		fmt.Fprintf(stdout, "%s: timeline: %s\n", f, d.Timeline.Summary())
	}
}
