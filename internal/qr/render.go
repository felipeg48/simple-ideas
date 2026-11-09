package qr

import (
	"bytes"
	"cmp"
	"compress/zlib"
	"encoding/base64"
	"fmt"
	"html"
	"image"
	"image/color"
	"math"
	"strings"
)

// Style is how a code is drawn. The zero value is black squares on white with
// the standard four-module quiet zone.
type Style struct {
	FG, BG color.NRGBA // BG.A == 0: transparent background
	Dots   bool        // round data modules, rounded finder squares (alignment and timing stay square)
	Quiet  int         // quiet zone in modules; 0 means 4
	Logo   *Logo
	Label  string // the SVG's accessible name (aria-label)
	Class  string // SVG only: fill the modules with currentColor and set this class, for CSS colours
}

// Logo is a picture in the middle of the code. Image is set for PNG and JPEG
// pictures (every output can draw them); an SVG picture (SVG set) only goes
// into SVG output.
type Logo struct {
	Image image.Image
	SVG   []byte
	Mime  string // image/png, image/jpeg or image/svg+xml
	Raw   []byte // the file, for the SVG output's data URI
	Href  string // SVG only: the picture's URL (a data URI or a link); wins over Raw
	// the hole in modules, [X0, X1) x [Y0, Y1): the picture plus a light margin
	X0, Y0, X1, Y1 int
}

// Hole places a centred hole for a picture taking frac of the code's width
// (plus a one-module margin), and checks the code still reads.
func (c *Code) Hole(frac float64, budget float64) (x0, y0, x1, y1 int, ok bool, worst float64) {
	side := int(math.Round(frac*float64(c.Size))) + 2
	if (c.Size-side)%2 != 0 {
		side++
	}
	x0 = (c.Size - side) / 2
	x1 = x0 + side
	ok, worst = c.Cover(x0, x0, x1, x1, budget)
	return x0, x0, x1, x1, ok, worst
}

func (s Style) quiet() int {
	if s.Quiet <= 0 {
		return 4
	}
	return s.Quiet
}

func (s Style) fg() color.NRGBA {
	if s.FG == (color.NRGBA{}) {
		return color.NRGBA{0, 0, 0, 255}
	}
	return s.FG
}

func (s Style) bg() color.NRGBA {
	if s.BG == (color.NRGBA{}) {
		return color.NRGBA{255, 255, 255, 255}
	}
	return s.BG
}

// Transparent is the background colour that leaves the background clear.
var Transparent = color.NRGBA{0, 0, 0, 1} // A=1 is never a real colour here; see hasBG

func (s Style) hasBG() bool { return s.BG != Transparent }

func (s Style) inHole(x, y int) bool {
	l := s.Logo
	return l != nil && x >= l.X0 && x < l.X1 && y >= l.Y0 && y < l.Y1
}

// ANSI draws the code for a terminal: two spaces per module, each painting
// its own background, so it reads on light and dark terminal themes alike.
func (c *Code) ANSI(quiet int) string {
	const light, dark = "\x1b[107m  \x1b[0m", "\x1b[40m  \x1b[0m"
	var b strings.Builder
	for y := -quiet; y < c.Size+quiet; y++ {
		for x := -quiet; x < c.Size+quiet; x++ {
			if x >= 0 && y >= 0 && x < c.Size && y < c.Size && c.Dark(x, y) {
				b.WriteString(dark)
			} else {
				b.WriteString(light)
			}
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// finder corners (top-left module) of the three finder squares
func (c *Code) finders() [3][2]int {
	return [3][2]int{{0, 0}, {c.Size - 7, 0}, {0, c.Size - 7}}
}

// shapes walks what to paint, in module units: rounded rectangles (finder
// rings and eyes in the dots style, r > 0) and the modules.
type shapes struct {
	rects   [][5]float64 // x, y, w, h, r: filled rounded rectangle
	rings   [][2]float64 // finder ring at (x, y): 7x7 r=2 less 5x5 r=1.5 (dots style)
	circles [][2]float64 // module centres (dots style)
}

func (c *Code) shapes(s Style) shapes {
	var sh shapes
	for y := 0; y < c.Size; y++ {
		for x := 0; x < c.Size; x++ {
			if !c.Dark(x, y) || s.inHole(x, y) || (s.Dots && c.Finder(x, y)) {
				continue
			}
			if s.Dots && !c.Function(x, y) { // alignment and timing stay square: big codes need them crisp
				sh.circles = append(sh.circles, [2]float64{float64(x) + .5, float64(y) + .5})
			} else {
				sh.rects = append(sh.rects, [5]float64{float64(x), float64(y), 1, 1, 0})
			}
		}
	}
	if s.Dots {
		for _, f := range c.finders() {
			fx, fy := float64(f[0]), float64(f[1])
			sh.rings = append(sh.rings, [2]float64{fx, fy})
			sh.rects = append(sh.rects, [5]float64{fx + 2, fy + 2, 3, 3, 1})
		}
	}
	return sh
}

// dotR is the dot radius in modules. Touching dots (0.5) read at every
// version; 0.45 dots stopped reading above version 20 (Vision, TestVision).
const dotR = 0.5

// dark reports whether the point (px, py), in module units from the code's
// top-left corner, is painted. Used by the raster output.
func (sh shapes) dark(c *Code, s Style, px, py float64) bool {
	x, y := int(math.Floor(px)), int(math.Floor(py))
	if x < 0 || y < 0 || x >= c.Size || y >= c.Size || s.inHole(x, y) {
		return false
	}
	if s.Dots && c.Finder(x, y) {
		for _, f := range c.finders() {
			lx, ly := px-float64(f[0]), py-float64(f[1])
			if lx < 0 || ly < 0 || lx >= 7 || ly >= 7 {
				continue
			}
			if inRound(lx, ly, 2, 2, 3, 3, 1) {
				return true
			}
			return inRound(lx, ly, 0, 0, 7, 7, 2) && !inRound(lx, ly, 1, 1, 5, 5, 1.5)
		}
	}
	if !c.Dark(x, y) {
		return false
	}
	if s.Dots && !c.Function(x, y) {
		dx, dy := px-float64(x)-.5, py-float64(y)-.5
		return dx*dx+dy*dy <= dotR*dotR
	}
	return true
}

func inRound(px, py, x, y, w, h, r float64) bool {
	if px < x || py < y || px > x+w || py > y+h {
		return false
	}
	cx := math.Max(x+r, math.Min(px, x+w-r))
	cy := math.Max(y+r, math.Min(py, y+h-r))
	dx, dy := px-cx, py-cy
	return dx*dx+dy*dy <= r*r
}

func hex(c color.NRGBA) string { return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B) }

func roundPath(b *strings.Builder, x, y, w, h, r float64) {
	if r == 0 {
		fmt.Fprintf(b, "M%g %gh%gv%gh%gz", x, y, w, h, -w)
		return
	}
	fmt.Fprintf(b, "M%g %gh%ga%g %g 0 0 1 %g %gv%ga%g %g 0 0 1 %g %gh%ga%g %g 0 0 1 %g %gv%ga%g %g 0 0 1 %g %gz",
		x+r, y, w-2*r, r, r, r, r, h-2*r, r, r, -r, r, -(w - 2*r), r, r, -r, -r, -(h - 2*r), r, r, r, -r)
}

// SVG draws the code as a standalone SVG, sized by its viewBox (it scales to
// its container). width, if not empty, sets the width attribute (e.g. "5cm").
// fg and bg may be CSS values (currentColor, var(--si-fg)); empty uses Style.
func (c *Code) SVG(s Style, width, fg, bg string) string {
	q := float64(s.quiet())
	n := float64(c.Size) + 2*q
	if fg == "" {
		fg = hex(s.fg())
	}
	if bg == "" && s.hasBG() {
		bg = hex(s.bg())
	}
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %g %g"`, n, n)
	if width != "" {
		fmt.Fprintf(&b, ` width="%s" height="%s"`, width, width)
	}
	if !s.Dots {
		b.WriteString(` shape-rendering="crispEdges"`)
	}
	if s.Class != "" {
		fmt.Fprintf(&b, ` class="%s"`, s.Class)
		fg = "currentColor"
	}
	fmt.Fprintf(&b, ` role="img" aria-label="%s">`, html.EscapeString(cmp.Or(s.Label, "QR code")))
	if bg != "" {
		fmt.Fprintf(&b, `<rect width="%g" height="%g" fill="%s"/>`, n, n, bg)
	}
	sh := c.shapes(s)
	fmt.Fprintf(&b, `<path transform="translate(%g %g)" fill="%s" fill-rule="evenodd" d="`, q, q, fg)
	for _, r := range sh.rects {
		roundPath(&b, r[0], r[1], r[2], r[3], r[4])
	}
	for _, r := range sh.rings {
		roundPath(&b, r[0], r[1], 7, 7, 2)
		roundPath(&b, r[0]+1, r[1]+1, 5, 5, 1.5)
	}
	b.WriteString(`"/>`)
	if len(sh.circles) > 0 {
		fmt.Fprintf(&b, `<g transform="translate(%g %g)" fill="%s">`, q, q, fg)
		for _, d := range sh.circles {
			fmt.Fprintf(&b, `<circle cx="%g" cy="%g" r="%g"/>`, d[0], d[1], dotR)
		}
		b.WriteString(`</g>`)
	}
	if l := s.Logo; l != nil {
		x, y, w := q+float64(l.X0)+1, q+float64(l.Y0)+1, float64(l.X1-l.X0-2)
		href := l.Href
		if href == "" {
			href = "data:" + l.Mime + ";base64," + base64.StdEncoding.EncodeToString(l.Raw)
		}
		fmt.Fprintf(&b, `<image x="%g" y="%g" width="%g" height="%g" preserveAspectRatio="xMidYMid meet" href="%s"/>`, x, y, w, w, html.EscapeString(href))
	}
	b.WriteString(`</svg>`)
	return b.String()
}

// Image draws the code into a px x px picture (3x3 samples per pixel, so dots
// have smooth edges), with the logo, if any, scaled into its hole.
func (c *Code) Image(s Style, px int) *image.NRGBA {
	q := float64(s.quiet())
	n := float64(c.Size) + 2*q
	img := image.NewNRGBA(image.Rect(0, 0, px, px))
	sh := c.shapes(s)
	fg, bg := s.fg(), s.bg()
	if !s.hasBG() {
		bg = color.NRGBA{fg.R, fg.G, fg.B, 0}
	}
	scale := n / float64(px)
	for py := 0; py < px; py++ {
		for pxx := 0; pxx < px; pxx++ {
			hits := 0
			for sy := 0; sy < 3; sy++ {
				for sx := 0; sx < 3; sx++ {
					mx := (float64(pxx)+(float64(sx)+.5)/3)*scale - q
					my := (float64(py)+(float64(sy)+.5)/3)*scale - q
					if sh.dark(c, s, mx, my) {
						hits++
					}
				}
			}
			img.SetNRGBA(pxx, py, mix(bg, fg, float64(hits)/9))
		}
	}
	if l := s.Logo; l != nil && l.Image != nil {
		x0 := int((q + float64(l.X0) + 1) / scale)
		w := int(float64(l.X1-l.X0-2) / scale)
		drawScaled(img, l.Image, x0, int((q+float64(l.Y0)+1)/scale), w)
	}
	return img
}

func mix(a, b color.NRGBA, t float64) color.NRGBA {
	l := func(x, y uint8) uint8 { return uint8(math.Round(float64(x) + (float64(y)-float64(x))*t)) }
	return color.NRGBA{l(a.R, b.R), l(a.G, b.G), l(a.B, b.B), l(a.A, b.A)}
}

// drawScaled draws src fitted into the w x w square at (x0, y0), keeping its
// aspect ratio, with box sampling and alpha over what is there.
func drawScaled(dst *image.NRGBA, src image.Image, x0, y0, w int) {
	sb := src.Bounds()
	sw, shh := float64(sb.Dx()), float64(sb.Dy())
	k := math.Min(float64(w)/sw, float64(w)/shh)
	dw, dh := int(sw*k), int(shh*k)
	ox, oy := x0+(w-dw)/2, y0+(w-dh)/2
	for y := 0; y < dh; y++ {
		for x := 0; x < dw; x++ {
			// average the source pixels this destination pixel covers
			sx0, sx1 := int(float64(x)/k), max(int(float64(x+1)/k), int(float64(x)/k)+1)
			sy0, sy1 := int(float64(y)/k), max(int(float64(y+1)/k), int(float64(y)/k)+1)
			var r, g, b, a, cnt float64
			for yy := sy0; yy < sy1 && yy < sb.Dy(); yy++ {
				for xx := sx0; xx < sx1 && xx < sb.Dx(); xx++ {
					c := color.NRGBAModel.Convert(src.At(sb.Min.X+xx, sb.Min.Y+yy)).(color.NRGBA)
					al := float64(c.A)
					r, g, b, a = r+float64(c.R)*al, g+float64(c.G)*al, b+float64(c.B)*al, a+al
					cnt++
				}
			}
			if a == 0 {
				continue
			}
			top := color.NRGBA{uint8(r / a), uint8(g / a), uint8(b / a), uint8(a / cnt)}
			under := dst.NRGBAAt(ox+x, oy+y)
			t := float64(top.A) / 255
			out := mix(under, top, t)
			out.A = uint8(math.Min(255, float64(under.A)+float64(top.A)*(1-float64(under.A)/255)))
			dst.SetNRGBA(ox+x, oy+y, out)
		}
	}
}

// PDF draws the code on one page of side pt points, as vector shapes, with
// the logo (a PNG or JPEG picture) as an image.
func (c *Code) PDF(s Style, pt float64) []byte {
	q := float64(s.quiet())
	n := float64(c.Size) + 2*q
	k := pt / n
	var g strings.Builder                                     // page content, in module units after the cm below
	fmt.Fprintf(&g, "q %.4f 0 0 %.4f 0 %.4f cm\n", k, -k, pt) // y down, like the grid
	if s.hasBG() {
		bg := s.bg()
		fmt.Fprintf(&g, "%.3f %.3f %.3f rg 0 0 %g %g re f\n", f(bg.R), f(bg.G), f(bg.B), n, n)
	}
	fg := s.fg()
	fmt.Fprintf(&g, "%.3f %.3f %.3f rg %g 0 0 %g %g %g cm\n", f(fg.R), f(fg.G), f(fg.B), 1.0, 1.0, q, q)
	sh := c.shapes(s)
	for _, r := range sh.rects {
		pdfRound(&g, r[0], r[1], r[2], r[3], r[4])
		g.WriteString("f\n")
	}
	for _, r := range sh.rings {
		pdfRound(&g, r[0], r[1], 7, 7, 2)
		pdfRound(&g, r[0]+1, r[1]+1, 5, 5, 1.5)
		g.WriteString("f*\n")
	}
	for _, d := range sh.circles {
		pdfCircle(&g, d[0], d[1], dotR)
		g.WriteString("f\n")
	}
	g.WriteString("Q\n")
	var objs []string
	resources := "<< >>"
	if l := s.Logo; l != nil && l.Image != nil {
		x := (q + float64(l.X0) + 1) * k
		w := float64(l.X1-l.X0-2) * k
		img, smask := pdfImage(l.Image)
		b := l.Image.Bounds()
		iw, ih := float64(b.Dx()), float64(b.Dy())
		sc := math.Min(w/iw, w/ih)
		dw, dh := iw*sc, ih*sc
		ix := x + (w-dw)/2
		iy := pt - (q+float64(l.Y0)+1)*k - (w-dh)/2 - dh
		fmt.Fprintf(&g, "q %.4f 0 0 %.4f %.4f %.4f cm /Im1 Do Q\n", dw, dh, ix, iy)
		objs = append(objs, smask, strings.Replace(img, "%SMASK%", "6 0 R", 1))
		resources = "<< /XObject << /Im1 7 0 R >> >>"
	}
	content := g.String()
	pdfObjs := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %.2f %.2f] /Contents 4 0 R /Resources 5 0 R >>", pt, pt),
		fmt.Sprintf("<< /Length %d >>\nstream\n%sendstream", len(content), content),
		resources,
	}
	pdfObjs = append(pdfObjs, objs...)
	var out bytes.Buffer
	out.WriteString("%PDF-1.4\n%\xe2\xe3\xcf\xd3\n")
	offs := make([]int, len(pdfObjs))
	for i, o := range pdfObjs {
		offs[i] = out.Len()
		fmt.Fprintf(&out, "%d 0 obj\n%s\nendobj\n", i+1, o)
	}
	xref := out.Len()
	fmt.Fprintf(&out, "xref\n0 %d\n0000000000 65535 f \n", len(pdfObjs)+1)
	for _, o := range offs {
		fmt.Fprintf(&out, "%010d 00000 n \n", o)
	}
	fmt.Fprintf(&out, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(pdfObjs)+1, xref)
	return out.Bytes()
}

func f(v uint8) float64 { return float64(v) / 255 }

const kappa = 0.5523 // Bézier control distance for a quarter circle

func pdfRound(b *strings.Builder, x, y, w, h, r float64) {
	if r == 0 {
		fmt.Fprintf(b, "%g %g %g %g re\n", x, y, w, h)
		return
	}
	c := r * kappa
	fmt.Fprintf(b, "%g %g m %g %g l %g %g %g %g %g %g c %g %g l %g %g %g %g %g %g c %g %g l %g %g %g %g %g %g c %g %g l %g %g %g %g %g %g c h\n",
		x+r, y, x+w-r, y, x+w-r+c, y, x+w, y+r-c, x+w, y+r,
		x+w, y+h-r, x+w, y+h-r+c, x+w-r+c, y+h, x+w-r, y+h,
		x+r, y+h, x+r-c, y+h, x, y+h-r+c, x, y+h-r,
		x, y+r, x, y+r-c, x+r-c, y, x+r, y)
}

func pdfCircle(b *strings.Builder, cx, cy, r float64) {
	pdfRound(b, cx-r, cy-r, 2*r, 2*r, r)
}

// pdfImage returns the image XObject (RGB, Flate) and its alpha soft mask.
func pdfImage(img image.Image) (obj, smask string) {
	b := img.Bounds()
	var rgb, alpha bytes.Buffer
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			rgb.Write([]byte{c.R, c.G, c.B})
			alpha.WriteByte(c.A)
		}
	}
	z := func(p []byte) string {
		var o bytes.Buffer
		w := zlib.NewWriter(&o)
		w.Write(p)
		w.Close()
		return o.String()
	}
	zr, za := z(rgb.Bytes()), z(alpha.Bytes())
	smask = fmt.Sprintf("<< /Type /XObject /Subtype /Image /Width %d /Height %d /ColorSpace /DeviceGray /BitsPerComponent 8 /Filter /FlateDecode /Length %d >>\nstream\n%s\nendstream",
		b.Dx(), b.Dy(), len(za), za)
	obj = fmt.Sprintf("<< /Type /XObject /Subtype /Image /Width %d /Height %d /ColorSpace /DeviceRGB /BitsPerComponent 8 /Filter /FlateDecode /SMask %%SMASK%% /Length %d >>\nstream\n%s\nendstream",
		b.Dx(), b.Dy(), len(zr), zr)
	return obj, smask
}
