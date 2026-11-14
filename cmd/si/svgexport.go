package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/felipeg48/simple-ideas/internal/assets"
	"github.com/felipeg48/simple-ideas/internal/cdp"
	"github.com/felipeg48/simple-ideas/internal/deck"
)

// si build --svg for timelines and infographics: <name>.svg, a vector picture
// of the frame as shown. The built page is opened in headless Chrome at the
// PNG's size and assets.SVGExportJS walks the laid-out page into SVG (text as
// <text>, pictures as data URIs, theme colours resolved), so the SVG is the
// browser's own layout, not a second one.

// exportFrameSVG writes <name>.svg of a built timeline or infographic (dest).
func exportFrameSVG(chrome, dest, out string, d *deck.Deck, stdout io.Writer) error {
	abs, err := filepath.Abs(dest)
	if err != nil {
		return err
	}
	w, h, url, sel := 1600, 900, "file://"+filepath.ToSlash(abs), ".si-tl"
	poster := d.Meta.Orientation == "portrait"
	if d.IsInfographic() {
		sel, poster = ".si-ig-frame", d.Meta.IGPortrait()
		if d.Meta.Aspect == "4:3" {
			h = 1200
		}
	} else if a, b, ok := cutRatio(d.Meta.Aspect); ok && !poster {
		h = int(float64(w) * b / a)
	}
	if poster {
		w, h = 1100, 1400
		if d.IsInfographic() {
			w, h = 1080, 1528
		}
	} else if d.IsInfographic() {
		url += "#1.999" // the slide, its last build step
	} else {
		url += "?si-view=thumb#1" // every era shown
	}
	svg, err := frameSVG(chrome, url, sel, w, h, poster)
	if err != nil {
		return err
	}
	p := filepath.Join(out, d.Name+".svg")
	if err := os.WriteFile(p, []byte(svg), 0o644); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "  %-28s (svg, %d KB)\n", d.Name+".svg", (len(svg)+1023)/1024)
	return nil
}

func cutRatio(s string) (float64, float64, bool) {
	var a, b float64
	if _, err := fmt.Sscanf(s, "%g:%g", &a, &b); err != nil || a <= 0 || b <= 0 {
		return 0, 0, false
	}
	return a, b, true
}

// frameSVG opens url at w×h (tall: the page's whole height) and returns the
// SVG of the element sel.
func frameSVG(chrome, url, sel string, w, h int, tall bool) (string, error) {
	q, _ := json.Marshal(sel)
	raw, err := pageRun(chrome, url, w, h, tall, assets.SVGExportJS+";window.siSVGExport("+string(q)+")")
	if err != nil {
		return "", err
	}
	var svg string
	if err := json.Unmarshal(raw, &svg); err != nil || svg == "" {
		return "", fmt.Errorf("svg: no %s on the page", sel)
	}
	return svg, nil
}

// frameEval is the value of js on a page at 1600×900, as JSON.
func frameEval(chrome, url, js string) (string, error) {
	raw, err := pageRun(chrome, url, 1600, 900, false, js)
	return string(raw), err
}

// pageRun opens url once it has settled (fonts, build steps, fit passes) and
// evaluates js.
func pageRun(chrome, url string, w, h int, tall bool, js string) (json.RawMessage, error) {
	c, err := cdp.Launch(chrome, w, h)
	if err != nil {
		return nil, err
	}
	defer c.Close()
	if _, err := c.OpenPage("", w, h); err != nil {
		return nil, err
	}
	if err := c.Call("Page.navigate", map[string]any{"url": url}, nil); err != nil {
		return nil, err
	}
	for i := 0; i < 200; i++ {
		if v, _ := c.Eval("document.readyState"); string(v) == `"complete"` {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if _, err := c.Eval(`document.fonts?document.fonts.ready.then(function(){return 1}):1`); err != nil {
		return nil, err
	}
	time.Sleep(900 * time.Millisecond) // build steps fade in, the fit passes run
	if tall {
		v, err := c.Eval(`Math.ceil(document.documentElement.scrollHeight)`)
		if err != nil {
			return nil, err
		}
		if n, _ := strconv.Atoi(string(v)); n > h {
			if err := c.Call("Emulation.setDeviceMetricsOverride", map[string]any{"width": w, "height": n, "deviceScaleFactor": 1, "mobile": false}, nil); err != nil {
				return nil, err
			}
			time.Sleep(400 * time.Millisecond)
		}
	}
	return c.Eval(js)
}
