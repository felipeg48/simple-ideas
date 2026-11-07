//go:build ignore

// gen draws the app icon (1024 px PNG): the simple-ideas Stack (three
// isometric slabs, see docs/brand) on a dark macOS squircle.
// go run icon/gen.go icon.png (make desktop-icon turns it into AppIcon.icns)
package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

type pt struct{ x, y float64 }

// inside reports whether p is inside the convex polygon q (either winding).
func inside(p pt, q []pt) bool {
	pos, neg := false, false
	for i := range q {
		a, b := q[i], q[(i+1)%len(q)]
		c := (b.x-a.x)*(p.y-a.y) - (b.y-a.y)*(p.x-a.x)
		pos, neg = pos || c > 0, neg || c < 0
	}
	return !(pos && neg)
}

type face struct {
	poly  []pt
	c     color.RGBA
	alpha float64
}

func main() {
	const n, ss = 1024, 3 // ss×ss samples per pixel
	indigo, teal := color.RGBA{0x5b, 0x4b, 0xdb, 255}, color.RGBA{0x1f, 0xc8, 0xa0, 255}
	top, bot := color.RGBA{0x1b, 0x1e, 0x2b, 255}, color.RGBA{0x0d, 0x0f, 0x15, 255}
	// the mark in 64 units, placed like docs/brand/app-dark.svg on the 824 px icon body
	u := 824.0 / 64
	at := func(x, y float64) pt { return pt{100 + (9.6+.7*x)*u, 100 + (10.6+.7*y)*u} }
	var faces []face
	slab := func(y float64, c color.RGBA, o float64) {
		faces = append(faces,
			face{[]pt{at(32, y), at(54, y+11), at(32, y+22), at(10, y+11)}, c, o},
			face{[]pt{at(10, y+11), at(10, y+16), at(32, y+27), at(32, y+22)}, c, o * .75},
			face{[]pt{at(54, y+11), at(54, y+16), at(32, y+27), at(32, y+22)}, c, o * .55})
	}
	slab(30, teal, .55)
	slab(19, teal, .85)
	slab(8, indigo, 1)
	inSquircle := func(x, y float64) bool { // macOS icon grid: 824 px body, centred
		dx, dy := math.Abs(x-n/2)/412, math.Abs(y-n/2)/412
		return math.Pow(dx, 5)+math.Pow(dy, 5) <= 1
	}
	img := image.NewRGBA(image.Rect(0, 0, n, n))
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			var r, g, b, a float64
			for sy := 0; sy < ss; sy++ {
				for sx := 0; sx < ss; sx++ {
					p := pt{float64(x) + (float64(sx)+.5)/ss, float64(y) + (float64(sy)+.5)/ss}
					if !inSquircle(p.x, p.y) {
						continue
					}
					t := (p.x + p.y) / (2 * n)
					cr, cg, cb := float64(top.R)*(1-t)+float64(bot.R)*t, float64(top.G)*(1-t)+float64(bot.G)*t, float64(top.B)*(1-t)+float64(bot.B)*t
					for _, f := range faces {
						if inside(p, f.poly) {
							cr = cr*(1-f.alpha) + float64(f.c.R)*f.alpha
							cg = cg*(1-f.alpha) + float64(f.c.G)*f.alpha
							cb = cb*(1-f.alpha) + float64(f.c.B)*f.alpha
						}
					}
					r, g, b, a = r+cr, g+cg, b+cb, a+1
				}
			}
			if a == 0 {
				continue
			}
			k := float64(ss * ss)
			// premultiplied: colour sums over covered samples, coverage a/k
			img.Set(x, y, color.RGBA{uint8(r / k), uint8(g / k), uint8(b / k), uint8(255 * a / k)})
		}
	}
	f, _ := os.Create(os.Args[1])
	png.Encode(f, img)
	f.Close()
}
