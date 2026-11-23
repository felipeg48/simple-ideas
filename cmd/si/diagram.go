package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/felipeg48/simple-ideas/internal/render"
)

// si build of a `type: diagram` page (RULES §38): its frame as <name>.png,
// .jpg (--jpg), pdfs/<name>.pdf (one page), .svg (text kept as text), and
// animated <name>.gif / .mp4 through the infographic film (diagram.js
// answers window.siIG.seek). Landscape frames are 1600×900 CSS pixels,
// portrait ones 1200×1600; every diagram at its last step.
func exportDiagramPage(chrome string, d *deck.Deck, dest, out string, o *options, stdout io.Writer) (pdfRel string, err error) {
	abs, err := filepath.Abs(dest)
	if err != nil {
		return "", err
	}
	portrait := d.Meta.Orientation == "portrait"
	w, h := 1600, 900
	if portrait {
		w, h = 1200, 1600
	}
	if o.png || o.pdf || o.qrJPG {
		shot := pageShot{W: w, H: h, Scale: o.scale, JS: `window.siIG && siIG.final()`}
		pngPath := filepath.Join(out, d.Name+".png")
		if o.png || o.qrJPG {
			shot.PNG = pngPath
		}
		if o.pdf {
			pdfRel = filepath.Join("pdfs", d.Name+".pdf")
			shot.PDF = filepath.Join(out, pdfRel)
			if err := os.MkdirAll(filepath.Dir(shot.PDF), 0o755); err != nil {
				return "", err
			}
		}
		if err := shot.take(chrome, fileURL(abs)+"?si-ig-film"); err != nil {
			return "", err
		}
		if o.png {
			fmt.Fprintf(stdout, "  %-28s (png, %d × %d)\n", d.Name+".png", int(float64(w)*o.scale), int(float64(h)*o.scale))
		}
		if o.qrJPG {
			if err := pngToJPEG(pngPath, filepath.Join(out, d.Name+".jpg")); err != nil {
				return "", err
			}
			if !o.png {
				os.Remove(pngPath)
			}
			fmt.Fprintf(stdout, "  %-28s (jpg, %d × %d)\n", d.Name+".jpg", int(float64(w)*o.scale), int(float64(h)*o.scale))
		}
		if o.pdf {
			fmt.Fprintf(stdout, "  %-28s (pdf, one page)\n", pdfRel)
		}
	}
	if o.svg {
		svg, err := frameSVG(chrome, fileURL(abs)+"?si-view=thumb", ".si-dg-frame", w, h, portrait)
		if err != nil {
			return pdfRel, err
		}
		svg, _ = svgFonts(svg, render.GoogleFamilies(d.Meta.Fonts))
		if err := os.WriteFile(filepath.Join(out, d.Name+".svg"), []byte(svg), 0o644); err != nil {
			return pdfRel, err
		}
		fmt.Fprintf(stdout, "  %-28s (svg, %d KB)\n", d.Name+".svg", (len(svg)+1023)/1024)
	}
	if o.gif || o.mp4 {
		if err := filmInfographic(chrome, igVariant{d: d, html: dest, name: d.Name}, out, o, stdout); err != nil {
			return pdfRel, err
		}
	}
	return filepath.ToSlash(pdfRel), nil
}
