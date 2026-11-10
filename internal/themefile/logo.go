package themefile

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	_ "image/gif" // logos may be GIF
	_ "image/jpeg"
	_ "image/png"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// A theme from a logo: the logo's colours (pixels of a PNG, JPEG or GIF; the
// fill, stroke and stop-color values of an SVG), quantised in OKLab and
// weighted by area and chroma, give the accents; the neutrals are tinted
// with the accent's hue; every text/background pair of Pairs is then moved
// in lightness until it passes, in light and in dark. Same logo, same theme.

// Swatch is one colour of a logo's palette; Weight is its share (0..1).
type Swatch struct {
	Hex    string  `json:"hex"`
	Weight float64 `json:"weight"`
}

// LogoPalette reads a logo and returns up to six colours, the weightiest
// first, without near-white, near-black and near-duplicate colours. An SVG
// is told apart from a picture by its text.
func LogoPalette(data []byte) ([]Swatch, error) {
	var bins map[[3]int]*bin
	t := bytes.TrimSpace(data)
	if bytes.HasPrefix(t, []byte("<")) {
		bins = svgBins(string(data))
	} else {
		img, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("not a PNG, JPEG, GIF or SVG logo: %v", err)
		}
		bins = imageBins(img)
	}
	return palette(bins), nil
}

type bin struct{ w, L, A, B float64 }

// add puts one colour, with weight w, into its OKLab cell; near-white and
// near-black are left out (a logo's paper and outline, not its brand).
func add(bins map[[3]int]*bin, r, g, b, w float64) {
	L, A, B := oklab(r, g, b)
	c := math.Hypot(A, B)
	if (L > 0.94 && c < 0.05) || (L < 0.2 && c < 0.05) {
		return
	}
	k := [3]int{int(math.Floor(L / 0.06)), int(math.Floor(A / 0.03)), int(math.Floor(B / 0.03))}
	x := bins[k]
	if x == nil {
		x = &bin{}
		bins[k] = x
	}
	w *= 1 + 4*c // saturated colours count more: they are the brand
	x.w += w
	x.L += L * w
	x.A += A * w
	x.B += B * w
}

func imageBins(img image.Image) map[[3]int]*bin {
	bins := map[[3]int]*bin{}
	r := img.Bounds()
	step := max(1, max(r.Dx(), r.Dy())/128) // ponytail: a 128-sample grid; enough for logos
	for y := r.Min.Y; y < r.Max.Y; y += step {
		for x := r.Min.X; x < r.Max.X; x += step {
			c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			if c.A < 128 {
				continue
			}
			add(bins, float64(c.R)/255, float64(c.G)/255, float64(c.B)/255, 1)
		}
	}
	return bins
}

var (
	svgColRE = regexp.MustCompile(`(?i)(?:fill|stroke|stop-color|color)\s*(?:=\s*["']|:)\s*(#[0-9a-f]{3,8}\b|rgba?\([^)]*\))`)
	numRE    = regexp.MustCompile(`[\d.]+%?`)
)

// svgBins counts each fill, stroke and gradient stop colour once per use.
func svgBins(src string) map[[3]int]*bin {
	bins := map[[3]int]*bin{}
	for _, m := range svgColRE.FindAllStringSubmatch(src, -1) {
		v := strings.ToLower(m[1])
		if strings.HasPrefix(v, "#") {
			if r, g, b, ok := rgb(v); ok {
				add(bins, r, g, b, 1)
			}
			continue
		}
		n := numRE.FindAllString(v, -1)
		if len(n) < 3 {
			continue
		}
		var ch [3]float64
		for i := range ch {
			f, _ := strconv.ParseFloat(strings.TrimSuffix(n[i], "%"), 64)
			if strings.HasSuffix(n[i], "%") {
				f = f * 255 / 100
			}
			ch[i] = math.Max(0, math.Min(255, f)) / 255
		}
		add(bins, ch[0], ch[1], ch[2], 1)
	}
	return bins
}

// palette merges cells closer than a just-noticeable step, heaviest first.
func palette(bins map[[3]int]*bin) []Swatch {
	type col struct{ w, L, A, B float64 }
	var cs []col
	total := 0.0
	for _, b := range bins {
		cs = append(cs, col{b.w, b.L / b.w, b.A / b.w, b.B / b.w})
		total += b.w
	}
	sort.Slice(cs, func(i, j int) bool { // ties broken by colour: deterministic
		if cs[i].w != cs[j].w {
			return cs[i].w > cs[j].w
		}
		if cs[i].L != cs[j].L {
			return cs[i].L < cs[j].L
		}
		if cs[i].A != cs[j].A {
			return cs[i].A < cs[j].A
		}
		return cs[i].B < cs[j].B
	})
	var keep []col
	for _, c := range cs {
		merged := false
		for i := range keep {
			k := &keep[i]
			if math.Sqrt((k.L-c.L)*(k.L-c.L)+(k.A-c.A)*(k.A-c.A)+(k.B-c.B)*(k.B-c.B)) < 0.1 {
				k.w += c.w
				merged = true
				break
			}
		}
		if !merged {
			keep = append(keep, c)
		}
	}
	sort.SliceStable(keep, func(i, j int) bool { return keep[i].w > keep[j].w })
	var out []Swatch
	for _, k := range keep {
		if k.w/total < 0.01 || len(out) == 6 { // specks: anti-aliasing, a stray pixel
			continue
		}
		out = append(out, Swatch{hexOf(fromOklab(k.L, k.A, k.B)), math.Round(k.w/total*1000) / 1000})
	}
	return out
}

// lch reads a hex colour as OKLab lightness, chroma and hue (radians).
func lch(hex string) (L, C, H float64) {
	r, g, b, _ := rgb(hex)
	L, A, B := oklab(r, g, b)
	return L, math.Hypot(A, B), math.Atan2(B, A)
}

func fromLCH(L, C, H float64) string {
	// lower the chroma until the colour is inside sRGB
	for ; C > 0; C -= 0.005 {
		r, g, b := fromOklab(L, C*math.Cos(H), C*math.Sin(H))
		if r >= -1e-4 && r <= 1+1e-4 && g >= -1e-4 && g <= 1+1e-4 && b >= -1e-4 && b <= 1+1e-4 {
			return hexOf(r, g, b)
		}
	}
	return hexOf(fromOklab(L, 0, 0))
}

// FromLogo is a theme named name from a logo's palette. accent, when set
// (a hex colour), is kept as the first accent and the logo gives the rest.
// The theme extends studio and sets every colour token in light and dark;
// the header says which colours were moved for contrast.
func FromLogo(name string, pal []Swatch, accent string) (*File, error) {
	if accent != "" {
		if _, _, _, ok := rgb(accent); !ok {
			return nil, fmt.Errorf("--accent %s is not a hex colour", accent)
		}
		pal = append([]Swatch{{Hex: strings.ToLower(accent), Weight: 1}}, pal...)
	}
	if len(pal) == 0 {
		return nil, errors.New("no brand colour in the logo (only white, black or transparent); give one with --accent")
	}
	// the accent: the weightiest colour with some chroma, or the first
	a1 := pal[0].Hex
	if accent == "" {
		best := -1.0
		for _, s := range pal {
			_, c, _ := lch(s.Hex)
			if sc := s.Weight * math.Min(c, 0.2); c >= 0.04 && sc > best {
				best, a1 = sc, s.Hex
			}
		}
	}
	L1, C1, H1 := lch(a1)
	// the second accent: the next colour at least 40° away in hue, else a
	// split complement of the first
	a2 := ""
	for _, s := range pal {
		_, c, h := lch(s.Hex)
		if s.Hex != a1 && c >= 0.04 && hueDist(h, H1) > 40*math.Pi/180 {
			a2 = s.Hex
			break
		}
	}
	if a2 == "" {
		a2 = fromLCH(L1, math.Max(C1, 0.1), H1+150*math.Pi/180)
	}
	f := New(strings.TrimSpace(strings.NewReplacer("*/", "", "\n", " ").Replace(name)), "studio")
	tint := func(L, C float64) string { return fromLCH(L, math.Min(C, C1), H1) }
	f.Light = map[string]string{
		"--si-bg": tint(0.985, 0.006), "--si-surface": tint(0.958, 0.012), "--si-surface-2": tint(0.93, 0.016),
		"--si-line": tint(0.86, 0.02), "--si-fg": tint(0.24, 0.02), "--si-muted": tint(0.47, 0.03),
		"--si-accent": a1, "--si-accent-2": a2,
		"--si-ok": "#1f8a4c", "--si-warn": "#b45309", "--si-crit": "#c62828",
	}
	_, C2, H2 := lch(a2)
	f.Dark = map[string]string{
		"--si-bg": tint(0.18, 0.012), "--si-surface": tint(0.225, 0.016), "--si-surface-2": tint(0.27, 0.018),
		"--si-line": tint(0.35, 0.02), "--si-fg": tint(0.95, 0.008), "--si-muted": tint(0.76, 0.02),
		"--si-accent": fromLCH(math.Max(L1, 0.72), C1, H1), "--si-accent-2": fromLCH(math.Max(lOf(a2), 0.75), C2, H2),
		"--si-ok": "#4ade80", "--si-warn": "#fbbf24", "--si-crit": "#f87171",
	}
	var notes []string
	for _, m := range []struct {
		name string
		v    map[string]string
	}{{"light", f.Light}, {"dark", f.Dark}} {
		for pass := 0; pass < 2; pass++ { // muted is measured on two grounds
			for _, p := range Pairs {
				if p.Text == "--si-on-accent" {
					continue
				}
				if r, ok := Contrast(m.v[p.Text], m.v[p.Ground]); ok && r < p.Min {
					was := m.v[p.Text]
					m.v[p.Text] = FixContrast(was, m.v[p.Ground], p.Min)
					if p.Text == "--si-accent" || p.Text == "--si-accent-2" {
						notes = append(notes, fmt.Sprintf("%s %s → %s in %s mode, for %.1f:1 on %s", p.Text, was, m.v[p.Text], m.name, p.Min, p.Ground))
					}
				}
			}
		}
		m.v["--si-on-accent"] = "#ffffff"
		if w, _ := Contrast("#ffffff", m.v["--si-accent"]); w < 4.5 {
			if b, _ := Contrast("#000000", m.v["--si-accent"]); b > w {
				m.v["--si-on-accent"] = "#000000"
			}
		}
	}
	var hb strings.Builder
	hb.WriteString("\n   made by si theme from a logo; its colours:")
	for _, s := range pal {
		hb.WriteString(" " + s.Hex)
	}
	for _, n := range notes {
		hb.WriteString("\n   contrast: " + n)
	}
	hb.WriteString("\n   fonts are yours to pick: fonts: heading=Fraunces:600, body=Inter:400;700 (front matter), then --si-font-body / --si-font-heading here")
	hb.WriteString("\n   tokens and looks only; edit here or in the Theme view of si Studio ")
	f.HeaderRest = hb.String()
	return f, nil
}

// MaxLogo is the largest logo si reads (the CLI, the Studio, si mcp).
const MaxLogo = 8 << 20

var slugRE = regexp.MustCompile(`[^a-z0-9]+`)

// Slug is a theme name as a file name: "Acme Corp" → acme-corp ("brand"
// when nothing is left).
func Slug(name string) string {
	if s := strings.Trim(slugRE.ReplaceAllString(strings.ToLower(name), "-"), "-"); s != "" {
		return s
	}
	return "brand"
}

// LogoName is a theme name from a logo's file name: acme-logo.svg → Acme.
func LogoName(path string) string {
	n := path[strings.LastIndexAny(path, `/\`)+1:]
	if i := strings.LastIndex(n, "."); i > 0 {
		n = n[:i]
	}
	n = strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(n, "-logo"), "_logo"), ".logo")
	if n == "" {
		return "Brand"
	}
	return strings.ToUpper(n[:1]) + n[1:]
}

func lOf(hex string) float64 { L, _, _ := lch(hex); return L }

func hueDist(a, b float64) float64 {
	d := math.Mod(math.Abs(a-b), 2*math.Pi)
	return math.Min(d, 2*math.Pi-d)
}
