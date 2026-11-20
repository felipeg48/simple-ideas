package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/felipeg48/simple-ideas/internal/assets"
	"github.com/felipeg48/simple-ideas/internal/cdp"
	"github.com/felipeg48/simple-ideas/internal/render"
)

// si build --png / --svg on a deck or a document: a picture of every ```chart
// fence, numbered in page order: <page>-chart-<n>.png / .svg of its final
// step, and with more than one step <page>-chart-<n>-step-<k>.png / .svg of
// each. The built page is opened once in headless Chrome (reduced motion: no
// animation), each chart is set to the step (window.siChart's own show) and
// its figure is cut out of the page; the SVG is the laid-out figure walked by
// assets.SVGExportJS, text kept as text. A 3D chart (CSS 3D) is PNG only:
// the walk draws boxes, not depth.

// chartFile is the name of chart n's picture (1-based): step k (1-based), or
// k 0 for its final step.
func chartFile(page string, n, k int, ext string) string {
	return pictureFile("chart", page, n, k, ext)
}

// pictureFile names chart or scene n's picture: <page>-scene-<n>.png, …
func pictureFile(what, page string, n, k int, ext string) string {
	if k == 0 {
		return fmt.Sprintf("%s-%s-%d.%s", page, what, n, ext)
	}
	return fmt.Sprintf("%s-%s-%d-step-%d.%s", page, what, n, k, ext)
}

// hasScenes reports whether a built page has a ```scene fence.
func hasScenes(page []byte) bool { return strings.Contains(string(page), `class="si-figure si-scene"`) }

// the scenes of the page: one picture per step (after it), or one of a
// scene that has no steps
const sceneListJS = `(function(){var sl=[].slice.call(document.querySelectorAll('.si-slide'));
return [].slice.call(document.querySelectorAll('.si-scene')).filter(function(f){return f.siScene}).map(function(f){var s=f.closest('.si-slide');
var k=[];for(var i=0;i<Math.max(1,f.siScene.n);i++)k.push('scene');return {slide:s?sl.indexOf(s):-1,kinds:k}})})()`

const sceneShowJS = `(function(n,k){var f=[].slice.call(document.querySelectorAll('.si-scene')).filter(function(f){return f.siScene})[n];
[].forEach.call(document.querySelectorAll('[data-si-shot]'),function(e){e.removeAttribute('data-si-shot')});
f.setAttribute('data-si-shot','');f.siScene.halt();f.siScene.set(f.siScene.n?k+1:0);f.scrollIntoView({block:'center'});
var r=f.getBoundingClientRect();return [r.left+scrollX,r.top+scrollY,r.width,r.height]})(%d,%d)`

// hasCharts reports whether a built page has a ```chart fence.
func hasCharts(page []byte) bool { return strings.Contains(string(page), `class="si-figure si-chart"`) }

// the charts of the page, in order: the slide each is on (-1: a document) and
// the kind of each step's frame
const chartListJS = `(function(){var sl=[].slice.call(document.querySelectorAll('.si-slide'));
return [].slice.call(document.querySelectorAll('.si-chart')).filter(function(f){return f.siChart}).map(function(f){var s=f.closest('.si-slide');
return {slide:s?sl.indexOf(s):-1,kinds:f.siChart.frames.map(function(x){return x.getAttribute('data-kind')})}})})()`

// chart n on step k, marked [data-si-shot]: its box in page pixels
const chartShowJS = `(function(n,k){var f=[].slice.call(document.querySelectorAll('.si-chart')).filter(function(f){return f.siChart})[n];
[].forEach.call(document.querySelectorAll('[data-si-shot]'),function(e){e.removeAttribute('data-si-shot')});
f.setAttribute('data-si-shot','');if(f.siChart.halt)f.siChart.halt();f.siChart.show(k,false);f.scrollIntoView({block:'center'});
var r=f.getBoundingClientRect();return [r.left+scrollX,r.top+scrollY,r.width,r.height]})(%d,%d)`

// exportCharts writes the pictures of every chart on a built page (dest).
func exportCharts(chrome, dest, out, page, fonts string, png, svg, doc bool, scale float64, stdout io.Writer) error {
	return exportPictures("chart", chrome, dest, out, page, fonts, png, svg, doc, scale, stdout)
}

// exportPictures is exportCharts for charts or (what "scene") scenes.
func exportPictures(what, chrome, dest, out, page, fonts string, png, svg, doc bool, scale float64, stdout io.Writer) error {
	listJS, showJS := chartListJS, chartShowJS
	if what == "scene" {
		listJS, showJS = sceneListJS, sceneShowJS
	}
	abs, err := filepath.Abs(dest)
	if err != nil {
		return err
	}
	w, h := 1600, 900
	if doc {
		w = 1280
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
	// reduced motion: every step shown as it ends, nothing growing
	c.Call("Emulation.setEmulatedMedia", map[string]any{"features": []map[string]string{{"name": "prefers-reduced-motion", "value": "reduce"}}}, nil)
	if err := c.Call("Page.navigate", map[string]any{"url": fileURL(abs)}, nil); err != nil {
		return err
	}
	for i := 0; i < 200; i++ {
		if v, _ := c.Eval("document.readyState"); string(v) == `"complete"` {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if _, err := c.Eval(`document.fonts?document.fonts.ready.then(function(){return 1}):1`); err != nil {
		return err
	}
	if _, err := c.Eval(hideChrome + `;(function(){var s=document.createElement('style');s.textContent='.si-chart-nav,.si-scene-nav,.si-doc-find,.si-doc-head .si-mode,[data-si-shot] .si-chart-frame:not([data-on]){display:none!important}';document.head.appendChild(s)})()`); err != nil {
		return err
	}
	raw, err := c.Eval(listJS)
	if err != nil {
		return err
	}
	var charts []struct {
		Slide int
		Kinds []string
	}
	if err := json.Unmarshal(raw, &charts); err != nil {
		return err
	}
	if svg {
		if _, err := c.Eval(assets.SVGExportJS); err != nil {
			return err
		}
	}
	flat := false // a 3D step was left out of the SVGs
	for n, ch := range charts {
		if ch.Slide >= 0 {
			if _, err := c.Eval(fmt.Sprintf(`window.SI&&SI.go(%d,'end',true)`, ch.Slide)); err != nil {
				return err
			}
			time.Sleep(400 * time.Millisecond) // the slide settles, its build steps shown
		}
		steps := len(ch.Kinds)
		for k := 0; k < steps; k++ {
			raw, err := c.Eval(fmt.Sprintf(showJS, n, k))
			if err != nil {
				return err
			}
			var box [4]float64
			if err := json.Unmarshal(raw, &box); err != nil {
				return err
			}
			time.Sleep(250 * time.Millisecond)
			// the last step is the chart's own picture; with more, each step too
			names := []int{}
			if steps > 1 {
				names = append(names, k+1)
			}
			if k == steps-1 {
				names = append(names, 0)
			}
			if png {
				shot, err := clipShot(c, box)
				if err != nil {
					return err
				}
				for _, s := range names {
					f := pictureFile(what, page, n+1, s, "png")
					if err := os.WriteFile(filepath.Join(out, f), shot, 0o644); err != nil {
						return err
					}
					fmt.Fprintf(stdout, "  %-28s (png, %s %d%s)\n", f, what, n+1, stepNote(s, steps))
				}
			}
			if svg && strings.HasSuffix(ch.Kinds[k], "3d") {
				flat = true
			} else if svg {
				raw, err := c.Eval(`window.siSVGExport('[data-si-shot]')`)
				if err != nil {
					return err
				}
				var pic string
				if err := json.Unmarshal(raw, &pic); err != nil || pic == "" {
					return fmt.Errorf("svg: %s %d drew nothing", what, n+1)
				}
				pic, _ = svgFonts(pic, render.GoogleFamilies(fonts))
				for _, s := range names {
					f := pictureFile(what, page, n+1, s, "svg")
					if err := os.WriteFile(filepath.Join(out, f), []byte(pic), 0o644); err != nil {
						return err
					}
					fmt.Fprintf(stdout, "  %-28s (svg, %s %d%s)\n", f, what, n+1, stepNote(s, steps))
				}
			}
		}
	}
	if flat {
		fmt.Fprintln(stdout, "  (no SVG of a 3D chart: CSS 3D is raster only; --png pictures it)")
	}
	return nil
}

func stepNote(k, steps int) string {
	if k == 0 {
		if steps > 1 {
			return ", final step"
		}
		return ""
	}
	return fmt.Sprintf(", step %d of %d", k, steps)
}

// clipShot is a PNG of one box of the page (page pixels), with a margin.
func clipShot(c *cdp.Client, box [4]float64) ([]byte, error) {
	const pad = 16
	x, y := max(0, box[0]-pad), max(0, box[1]-pad)
	clip := map[string]any{"x": x, "y": y, "width": box[2] + box[0] - x + pad, "height": box[3] + box[1] - y + pad, "scale": 1}
	var shot struct{ Data string }
	if err := c.Call("Page.captureScreenshot", map[string]any{"format": "png", "clip": clip, "captureBeyondViewport": true}, &shot); err != nil {
		return nil, err
	}
	return base64.StdEncoding.DecodeString(shot.Data)
}
