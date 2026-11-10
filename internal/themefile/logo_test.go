package themefile

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"testing"
)

// logoPNG draws a w×h logo: a background, then each colour as a band
// covering share of the width.
func logoPNG(t *testing.T, bg color.Color, bands []color.NRGBA, shares []float64) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 200, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 200; x++ {
			img.Set(x, y, bg)
		}
	}
	x0 := 0
	for i, c := range bands {
		w := int(shares[i] * 200)
		for y := 20; y < 80; y++ {
			for x := x0; x < x0+w; x++ {
				img.SetNRGBA(x, y, c)
			}
		}
		x0 += w
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func hueDeg(hex string) float64 {
	_, _, h := lch(hex)
	d := h * 180 / math.Pi
	if d < 0 {
		d += 360
	}
	return d
}

func TestLogoPalette(t *testing.T) {
	orange := color.NRGBA{0xe4, 0x57, 0x2e, 255}
	navy := color.NRGBA{0x1d, 0x3a, 0x8a, 255}
	cases := []struct {
		name   string
		logo   []byte
		accent string // the accent's hex: its hue must be near
		n      int    // colours in the palette
	}{
		{"png on white", logoPNG(t, color.White, []color.NRGBA{orange, navy}, []float64{.5, .2}), "#e4572e", 2},
		{"png, transparent", logoPNG(t, color.Transparent, []color.NRGBA{navy, orange}, []float64{.6, .1}), "#1d3a8a", 2},
		{"png, black outline only", logoPNG(t, color.Black, []color.NRGBA{orange}, []float64{.3}), "#e4572e", 1},
		{"svg fills", []byte(`<svg viewBox="0 0 10 10"><rect fill="#fff" width="10" height="10"/><path fill="#2a9d8f" d="M0 0"/><path fill="#2a9d8f"/><circle style="fill:#e9c46a;stroke:#000"/></svg>`), "#2a9d8f", 2},
		{"svg gradient stops", []byte(`<svg><linearGradient><stop offset="0" stop-color="rgb(124, 58, 237)"/><stop offset="1" stop-color="#7c3aed"/></linearGradient><path fill="url(#g)"/></svg>`), "#7c3aed", 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			pal, err := LogoPalette(c.logo)
			if err != nil {
				t.Fatal(err)
			}
			if len(pal) != c.n {
				t.Fatalf("palette %v, want %d colours", pal, c.n)
			}
			f, err := FromLogo("Acme", pal, "")
			if err != nil {
				t.Fatal(err)
			}
			if d := math.Abs(hueDeg(f.Light["--si-accent"]) - hueDeg(c.accent)); d > 12 && d < 348 {
				t.Errorf("accent %s, want the hue of %s (palette %v)", f.Light["--si-accent"], c.accent, pal)
			}
			if issues := f.Check(f.String()); len(issues) != 0 {
				t.Errorf("issues: %+v", issues)
			}
			again, _ := LogoPalette(c.logo)
			g, _ := FromLogo("Acme", again, "")
			if g.String() != f.String() {
				t.Error("same logo, different theme")
			}
		})
	}
}

func TestLogoNoColour(t *testing.T) {
	pal, err := LogoPalette(logoPNG(t, color.White, []color.NRGBA{{0, 0, 0, 255}}, []float64{.5}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := FromLogo("x", pal, ""); err == nil {
		t.Error("a black and white logo gave a theme without --accent")
	}
	f, err := FromLogo("x", pal, "#E4572E")
	if err != nil || f.Light["--si-accent"] != "#e4572e" {
		t.Errorf("--accent: %v %v", f, err)
	}
	if _, err := LogoPalette([]byte("not an image")); err == nil {
		t.Error("garbage read as a logo")
	}
}

// Every hue at every lightness gives a theme that passes si theme check.
func TestFromLogoAlwaysPasses(t *testing.T) {
	for h := 0; h < 360; h += 15 {
		for _, L := range []float64{0.3, 0.5, 0.65, 0.8, 0.92} {
			for _, C := range []float64{0.05, 0.12, 0.25} {
				hex := fromLCH(L, C, float64(h)*math.Pi/180)
				f, err := FromLogo("t", []Swatch{{hex, 1}}, "")
				if err != nil {
					t.Fatal(err)
				}
				text := f.String()
				if issues := Parse(text).Check(text); len(issues) != 0 {
					t.Fatalf("%s: %v", hex, fmt.Sprint(issues))
				}
			}
		}
	}
}
