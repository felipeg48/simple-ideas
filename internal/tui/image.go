package tui

import (
	"fmt"
	"image"
	_ "image/gif" // decoders for image.Decode
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
)

// image draws a picture as half-block pixels: each character is two pixels,
// the upper one in the foreground colour of ▀ and the lower one in its
// background, so a cell holds two square pixels. Formats Go cannot decode
// (SVG, WebP, video) get a placeholder with their name.
func (r *textRenderer) image(src, alt string) []string {
	if strings.Contains(src, "://") || strings.HasPrefix(src, "data:") {
		return r.placeholder("▣ "+orName(alt, "image"), []string{src})
	}
	p := src
	if !filepath.IsAbs(p) {
		p = filepath.Join(r.dir, src)
	}
	f, err := os.Open(p)
	if err != nil {
		return r.placeholder("▣ "+orName(alt, filepath.Base(src)), []string{"missing: " + src})
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return r.placeholder("▣ "+orName(alt, filepath.Base(src)), []string{filepath.Base(src) + " (shown in the browser)"})
	}
	b := img.Bounds()
	cols := min(r.w, 72)
	ph := cols * b.Dy() / max(1, b.Dx()) // pixel rows at that width
	if maxRows := max(4, r.maxH); (ph+1)/2 > maxRows {
		ph = maxRows * 2
		cols = max(4, ph*b.Dx()/max(1, b.Dy()))
	}
	rows := (ph + 1) / 2
	if r.pics != nil && r.pics.Proto != "" { // the terminal draws the real picture into this block
		out := r.pics.reserve(img, cols, rows)
		if alt != "" {
			out = append(out, r.st().Foreground(r.pal.Muted).Italic(true).Render(alt))
		}
		return out
	}
	// box-sample the source into cols × ph pixels
	sample := func(x, y int) (uint32, uint32, uint32) {
		x0, x1 := b.Min.X+x*b.Dx()/cols, b.Min.X+(x+1)*b.Dx()/cols
		y0, y1 := b.Min.Y+y*b.Dy()/ph, b.Min.Y+(y+1)*b.Dy()/ph
		x1, y1 = max(x1, x0+1), max(y1, y0+1)
		var rs, gs, bs, n uint32
		step := max(1, (x1-x0)/4) // at most ~4×4 samples a pixel
		for yy := y0; yy < y1; yy += max(1, (y1-y0)/4) {
			for xx := x0; xx < x1; xx += step {
				cr, cg, cb, _ := img.At(xx, yy).RGBA()
				rs, gs, bs, n = rs+cr>>8, gs+cg>>8, bs+cb>>8, n+1
			}
		}
		return rs / n, gs / n, bs / n
	}
	out := make([]string, 0, rows)
	for row := 0; row < rows; row++ {
		var s strings.Builder
		for x := 0; x < cols; x++ {
			tr, tg, tb := sample(x, row*2)
			br, bg, bb := tr, tg, tb
			if row*2+1 < ph {
				br, bg, bb = sample(x, row*2+1)
			}
			fmt.Fprintf(&s, "\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm▀", tr, tg, tb, br, bg, bb)
		}
		s.WriteString("\x1b[m")
		out = append(out, s.String())
	}
	if alt != "" {
		out = append(out, r.st().Foreground(r.pal.Muted).Italic(true).Render(alt))
	}
	return out
}

func orName(alt, name string) string {
	if alt != "" {
		return alt
	}
	return name
}
