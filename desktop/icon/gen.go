//go:build ignore

// gen draws the app icon (1024 px PNG): the si orange squircle with a white
// slide and a play mark. go run icon/gen.go icon.png
package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

func main() {
	const n = 1024
	img := image.NewRGBA(image.Rect(0, 0, n, n))
	a, b := color.RGBA{0xf9, 0x73, 0x16, 0xff}, color.RGBA{0xea, 0x3c, 0x0c, 0xff} // top-left → bottom-right
	inSquircle := func(x, y float64) bool {                                        // macOS icon grid: 824 px body, centred
		cx, cy, r := float64(n)/2, float64(n)/2, 412.0
		dx, dy := math.Abs(x-cx)/r, math.Abs(y-cy)/r
		return math.Pow(dx, 5)+math.Pow(dy, 5) <= 1
	}
	rect := func(x, y, x0, y0, x1, y1, rad float64) bool {
		if x < x0 || x > x1 || y < y0 || y > y1 {
			return false
		}
		cx := math.Max(x0+rad, math.Min(x, x1-rad))
		cy := math.Max(y0+rad, math.Min(y, y1-rad))
		return (x-cx)*(x-cx)+(y-cy)*(y-cy) <= rad*rad
	}
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			fx, fy := float64(x)+.5, float64(y)+.5
			if !inSquircle(fx, fy) {
				continue
			}
			t := (fx + fy) / (2 * n)
			c := color.RGBA{uint8(float64(a.R)*(1-t) + float64(b.R)*t), uint8(float64(a.G)*(1-t) + float64(b.G)*t), uint8(float64(a.B)*(1-t) + float64(b.B)*t), 255}
			// a white slide with a play triangle cut out
			if rect(fx, fy, 252, 312, 772, 640, 40) {
				c = color.RGBA{255, 255, 255, 255}
				px, py := fx-470, fy-476 // play triangle, pointing right
				if px >= 0 && px <= 120 && math.Abs(py) <= 70*(1-px/120) {
					c = b
				}
			}
			if rect(fx, fy, 492, 640, 532, 712, 6) || rect(fx, fy, 392, 704, 632, 736, 16) { // the stand
				c = color.RGBA{255, 255, 255, 255}
			}
			img.Set(x, y, c)
		}
	}
	f, _ := os.Create(os.Args[1])
	png.Encode(f, img)
	f.Close()
}
