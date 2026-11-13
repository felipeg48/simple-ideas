package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/felipeg48/simple-ideas/internal/cdp"
	"github.com/felipeg48/simple-ideas/internal/deck"
)

// exportInfographic writes <name>.png and pdfs/<name>.pdf of a built
// infographic: a landscape one as its slide at its aspect, every block
// shown; a portrait one as the whole poster on one tall page. scale is the
// PNG's pixels per CSS pixel (--scale, 2 by default).
func exportInfographic(chrome, htmlPath, out string, d *deck.Deck, png, pdf bool, scale float64, stdout io.Writer) (pdfRel string, err error) {
	abs, err := filepath.Abs(htmlPath)
	if err != nil {
		return "", err
	}
	w, h, hash := 1600, 900, "#1.999" // the slide, its last build step
	switch {
	case d.Meta.IGPortrait():
		w, h, hash = 1080, 1528, ""
	case d.Meta.Aspect == "4:3":
		h = 1200
	}
	shot := pageShot{W: w, H: h, Scale: scale, Fit: d.Meta.IGPortrait()}
	if png {
		shot.PNG = filepath.Join(out, d.Name+".png")
	}
	if pdf {
		pdfRel = filepath.Join("pdfs", d.Name+".pdf")
		shot.PDF = filepath.Join(out, pdfRel)
		if err := os.MkdirAll(filepath.Dir(shot.PDF), 0o755); err != nil {
			return "", err
		}
	}
	if err := shot.take(chrome, "file://"+filepath.ToSlash(abs)+hash); err != nil {
		return "", err
	}
	if png {
		fmt.Fprintf(stdout, "  %-28s (png, %d × %d)\n", d.Name+".png", int(float64(shot.W)*scale), int(float64(shot.H)*scale))
	}
	if pdf {
		fmt.Fprintf(stdout, "  %-28s (pdf, one page)\n", pdfRel)
	}
	return filepath.ToSlash(pdfRel), nil
}

// pageShot is a picture of a page taken in headless Chrome: a PNG and/or a
// one-page PDF of W×H CSS pixels (Fit: H grows to the page's whole height).
type pageShot struct {
	W, H     int
	Scale    float64
	Fit      bool
	PNG, PDF string
}

// the deck's own controls never belong in a picture of it
const hideChrome = `(function(){var s=document.createElement('style');s.textContent='.si-bar,.si-progress,.si-help,.si-problems{display:none!important}';document.head.appendChild(s)})()`

func (p *pageShot) take(chrome, url string) error {
	c, err := cdp.Launch(chrome, p.W, p.H)
	if err != nil {
		return err
	}
	defer c.Close()
	c.Scale = p.Scale
	if _, err := c.OpenPage("", p.W, p.H); err != nil {
		return err
	}
	if err := c.Call("Page.navigate", map[string]any{"url": url}, nil); err != nil {
		return err
	}
	for i := 0; i < 200; i++ {
		if v, _ := c.Eval("document.readyState"); string(v) == `"complete"` {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if _, err := c.Eval(hideChrome); err != nil {
		return err
	}
	time.Sleep(900 * time.Millisecond) // build steps fade in
	if p.Fit {
		v, err := c.Eval(`Math.ceil(Math.max(document.documentElement.scrollHeight,document.body.scrollHeight,(document.querySelector('main')||document.body).getBoundingClientRect().bottom))`)
		if err != nil {
			return err
		}
		if n, err := strconv.Atoi(string(v)); err == nil && n > p.H {
			p.H = n
		}
		if err := c.Call("Emulation.setDeviceMetricsOverride", map[string]any{"width": p.W, "height": p.H, "deviceScaleFactor": max(p.Scale, 1), "mobile": false}, nil); err != nil {
			return err
		}
		time.Sleep(300 * time.Millisecond)
	}
	if p.PNG != "" {
		raw, err := c.Screenshot()
		if err != nil {
			return err
		}
		if err := os.WriteFile(p.PNG, raw, 0o644); err != nil {
			return err
		}
	}
	if p.PDF != "" {
		// printed as seen: the screen layout on one page of the picture's size
		if err := c.Call("Emulation.setEmulatedMedia", map[string]any{"media": "screen"}, nil); err != nil {
			return err
		}
		page := fmt.Sprintf(`(function(){var s=document.createElement('style');s.textContent='@page{size:%dpx %dpx;margin:0}';document.head.appendChild(s)})()`, p.W, p.H)
		if _, err := c.Eval(page); err != nil {
			return err
		}
		var r struct{ Data string }
		if err := c.Call("Page.printToPDF", map[string]any{"printBackground": true, "preferCSSPageSize": true}, &r); err != nil {
			return err
		}
		raw, err := base64.StdEncoding.DecodeString(r.Data)
		if err != nil {
			return err
		}
		return os.WriteFile(p.PDF, raw, 0o644)
	}
	return nil
}
