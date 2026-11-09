package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/qr"
)

// printQR prints a terminal QR code for u, or says why there is none, so a
// missing code is never a silent one.
func printQR(w io.Writer, u string) {
	c, err := qr.Encode([]byte(u), qr.Options{Level: qr.M})
	if err != nil {
		fmt.Fprintf(w, "  (no QR code: %v; type the link instead)\n", err)
		return
	}
	fmt.Fprint(w, c.ANSI(2))
}

// runQR is `si qr`: a QR code for a URL, text, a Wi-Fi network or a contact
// card, written as PNG (default), JPG, SVG and/or PDF.
func runQR(rest []string, o *options, stdout, stderr io.Writer) int {
	text, err := qrPayload(rest, o)
	if err != nil {
		return qrUsage(stderr, err.Error())
	}
	level, err := qr.ParseLevel(o.qrEC)
	if err != nil {
		return qrUsage(stderr, err.Error())
	}
	if o.qrLogo != "" && !o.qrECSet {
		level = qr.H // a picture hides modules: the most error correction
	}
	c, err := qr.Encode([]byte(text), qr.Options{Level: level})
	if err != nil {
		return fail(stderr, err)
	}
	st := qr.Style{Dots: o.qrStyle == "dots", Quiet: o.qrQuiet}
	switch o.qrStyle {
	case "square", "dots":
	default:
		return qrUsage(stderr, fmt.Sprintf("--style %q: use square or dots", o.qrStyle))
	}
	if st.FG, err = qrColor(o.qrFG, color.NRGBA{0, 0, 0, 255}); err != nil {
		return qrUsage(stderr, "--fg: "+err.Error())
	}
	if st.FG == qr.Transparent {
		return qrUsage(stderr, "--fg cannot be none: the code needs a colour")
	}
	if st.BG, err = qrColor(o.qrBG, color.NRGBA{255, 255, 255, 255}); err != nil {
		return qrUsage(stderr, "--bg: "+err.Error())
	}
	formats, base, err := qrFormats(o)
	if err != nil {
		return qrUsage(stderr, err.Error())
	}
	if st.BG == qr.Transparent && contains(formats, "jpg") {
		return qrUsage(stderr, "a JPG has no transparent background: use --bg with a colour, or PNG / SVG / PDF")
	}
	if st.BG != qr.Transparent {
		if r := contrast(st.FG, st.BG); r < 3 {
			fmt.Fprintf(stderr, "si qr: warning: --fg and --bg differ too little (contrast %.1f:1); many readers need 3:1 or more\n", r)
		}
		if luminance(st.FG) > luminance(st.BG) {
			fmt.Fprintln(stderr, "si qr: warning: a light code on a dark background is inverted; some readers do not read it")
		}
	}
	if o.qrLogo != "" {
		logo, err := qrLogo(c, o, formats)
		if err != nil {
			return fail(stderr, err)
		}
		st.Logo = logo
	}
	px, pt, width, err := qrSize(o.size)
	if err != nil {
		return qrUsage(stderr, err.Error())
	}
	for _, f := range formats {
		name := base + "." + f
		var data []byte
		switch f {
		case "svg":
			data = []byte(c.SVG(st, width, "", "") + "\n")
		case "pdf":
			data = c.PDF(st, pt)
		case "png", "jpg":
			img := c.Image(st, px)
			var buf bytes.Buffer
			if f == "png" {
				err = png.Encode(&buf, img)
			} else {
				flat := image.NewRGBA(img.Bounds())
				draw.Draw(flat, flat.Bounds(), img, image.Point{}, draw.Src)
				err = jpeg.Encode(&buf, flat, &jpeg.Options{Quality: 95})
			}
			if err != nil {
				return fail(stderr, err)
			}
			data = buf.Bytes()
		}
		if err := os.WriteFile(name, data, 0o644); err != nil {
			return fail(stderr, err)
		}
		fmt.Fprintf(stdout, "si qr: %s (version %d, %dx%d modules, error correction %s)\n", name, c.Version, c.Size, c.Size, c.Level)
	}
	if o.terminal {
		fmt.Fprint(stdout, c.ANSI(2))
	}
	return exitOK
}

// qrUsage reports a wrong option in one line (not the whole usage text).
func qrUsage(stderr io.Writer, msg string) int {
	fmt.Fprintf(stderr, "si qr: %s (si --help)\n", msg)
	return exitUsage
}

func fail(stderr io.Writer, err error) int {
	fmt.Fprintf(stderr, "si qr: %v\n", err)
	return exitProblems
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

// qrPayload is the text to encode: the argument (or --url), "-" for stdin,
// a Wi-Fi network (--wifi) or a contact card file (--vcard).
func qrPayload(rest []string, o *options) (string, error) {
	n := 0
	for _, set := range []bool{len(rest) > 0, o.url != "", o.qrWifi != "", o.qrVCard != ""} {
		if set {
			n++
		}
	}
	switch {
	case n == 0:
		return "", fmt.Errorf("qr needs a <text or URL>, --url, --wifi or --vcard")
	case n > 1 || len(rest) > 1:
		return "", fmt.Errorf("qr takes one thing to encode (quote text with spaces)")
	case o.qrWifi != "":
		sec := strings.ToUpper(o.qrSecurity)
		if o.qrPassword == "" {
			sec = "nopass"
		} else if sec != "WPA" && sec != "WEP" {
			return "", fmt.Errorf("--security %q: use WPA or WEP", o.qrSecurity)
		}
		esc := strings.NewReplacer(`\`, `\\`, `;`, `\;`, `,`, `\,`, `:`, `\:`, `"`, `\"`)
		s := "WIFI:T:" + sec + ";S:" + esc.Replace(o.qrWifi) + ";"
		if o.qrPassword != "" {
			s += "P:" + esc.Replace(o.qrPassword) + ";"
		}
		return s + ";", nil
	case o.qrVCard != "":
		b, err := os.ReadFile(o.qrVCard)
		if err != nil {
			return "", err
		}
		if !strings.Contains(strings.ToUpper(string(b)), "BEGIN:VCARD") {
			return "", fmt.Errorf("%s is not a vCard (no BEGIN:VCARD)", o.qrVCard)
		}
		return strings.TrimSpace(string(b)), nil
	case o.url != "":
		return o.url, nil
	case rest[0] == "-":
		b, err := io.ReadAll(os.Stdin)
		return strings.TrimRight(string(b), "\n"), err
	}
	return rest[0], nil
}

// qrFormats reads the formats from --png --jpg --svg --pdf, --type a,b or the
// -o extension (default png), and the file name without extension.
func qrFormats(o *options) ([]string, string, error) {
	var fs []string
	add := func(f string) {
		if f == "jpeg" {
			f = "jpg"
		}
		if !contains(fs, f) {
			fs = append(fs, f)
		}
	}
	for f, on := range map[string]bool{"png": o.png, "jpg": o.qrJPG, "svg": o.svg, "pdf": o.pdf} {
		if on {
			add(f)
		}
	}
	if o.kind != "" {
		for _, f := range strings.Split(o.kind, ",") {
			add(strings.ToLower(strings.TrimSpace(f)))
		}
	}
	base := o.output
	if base == "" {
		base = o.qrName
	}
	if base == "" {
		base = "qr"
	}
	if ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(base), ".")); ext == "png" || ext == "jpg" || ext == "jpeg" || ext == "svg" || ext == "pdf" {
		base = strings.TrimSuffix(base, filepath.Ext(base))
		add(ext)
	}
	if len(fs) == 0 {
		add("png")
	}
	for _, f := range fs {
		if f != "png" && f != "jpg" && f != "svg" && f != "pdf" {
			return nil, "", fmt.Errorf("format %q: use png, jpg, svg or pdf", f)
		}
	}
	// a stable order: the order given is a map's for the flags
	order := map[string]int{"png": 0, "jpg": 1, "svg": 2, "pdf": 3}
	for i := range fs {
		for j := i + 1; j < len(fs); j++ {
			if order[fs[j]] < order[fs[i]] {
				fs[i], fs[j] = fs[j], fs[i]
			}
		}
	}
	return fs, base, nil
}

// qrSize reads --size: pixels for PNG/JPG ("1024", "1024px"), a print size
// for PDF and the SVG's width ("5cm", "50mm", "2in", "144pt"). The default
// is 1024 px, 5 cm and an SVG that fills its container.
func qrSize(s string) (px int, pt float64, width string, err error) {
	px, pt = 1024, 5/2.54*72
	if s == "" || s == "100x30" { // 100x30 is the terminal --dump default, not a QR size
		return px, pt, "", nil
	}
	num := strings.TrimRight(s, "abcdefghijklmnopqrstuvwxyz")
	unit := s[len(num):]
	v, err := strconv.ParseFloat(num, 64)
	if err != nil || v <= 0 {
		return 0, 0, "", fmt.Errorf("--size %q: a number with px, cm, mm, in or pt", s)
	}
	per := map[string]float64{"cm": 72 / 2.54, "mm": 72 / 25.4, "in": 72, "pt": 1}
	switch {
	case unit == "" || unit == "px":
		if v > 8000 {
			return 0, 0, "", fmt.Errorf("--size %q: at most 8000 px", s)
		}
		return int(v), v * 0.75, s, nil
	case per[unit] > 0:
		return int(math.Round(v * per[unit] / 72 * 300)), v * per[unit], s, nil // print sizes rasterise at 300 dpi
	}
	return 0, 0, "", fmt.Errorf("--size %q: unit %q, use px, cm, mm, in or pt", s, unit)
}

func qrColor(s string, def color.NRGBA) (color.NRGBA, error) {
	switch strings.ToLower(s) {
	case "":
		return def, nil
	case "none", "transparent":
		return qr.Transparent, nil
	case "black":
		return color.NRGBA{0, 0, 0, 255}, nil
	case "white":
		return color.NRGBA{255, 255, 255, 255}, nil
	}
	h := strings.TrimPrefix(s, "#")
	if len(h) == 3 {
		h = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]})
	}
	v, err := strconv.ParseUint(h, 16, 32)
	if len(h) != 6 || err != nil {
		return color.NRGBA{}, fmt.Errorf("%q: use #rrggbb, #rgb, black, white or none", s)
	}
	return color.NRGBA{uint8(v >> 16), uint8(v >> 8), uint8(v), 255}, nil
}

func luminance(c color.NRGBA) float64 {
	ch := func(v uint8) float64 {
		x := float64(v) / 255
		if x <= 0.03928 {
			return x / 12.92
		}
		return math.Pow((x+0.055)/1.055, 2.4)
	}
	return 0.2126*ch(c.R) + 0.7152*ch(c.G) + 0.0722*ch(c.B)
}

func contrast(a, b color.NRGBA) float64 {
	la, lb := luminance(a), luminance(b)
	return (math.Max(la, lb) + 0.05) / (math.Min(la, lb) + 0.05)
}

// qrLogo loads the picture and cuts its hole, refusing a size the error
// correction cannot carry (and saying what fits).
func qrLogo(c *qr.Code, o *options, formats []string) (*qr.Logo, error) {
	raw, err := os.ReadFile(o.qrLogo)
	if err != nil {
		return nil, err
	}
	l := &qr.Logo{Raw: raw}
	switch strings.ToLower(filepath.Ext(o.qrLogo)) {
	case ".png":
		l.Mime = "image/png"
		l.Image, err = png.Decode(bytes.NewReader(raw))
	case ".jpg", ".jpeg":
		l.Mime = "image/jpeg"
		l.Image, err = jpeg.Decode(bytes.NewReader(raw))
	case ".svg":
		l.Mime = "image/svg+xml"
		for _, f := range formats {
			if f != "svg" {
				return nil, fmt.Errorf("an SVG --logo only goes into SVG output; for %s use a PNG or JPEG picture", strings.ToUpper(f))
			}
		}
	default:
		return nil, fmt.Errorf("--logo %s: use a PNG, JPEG or SVG picture", o.qrLogo)
	}
	if err != nil {
		return nil, fmt.Errorf("--logo %s: %v", o.qrLogo, err)
	}
	frac := o.qrLogoSize / 100
	if frac <= 0 || frac > 0.5 {
		return nil, fmt.Errorf("--logo-size %g: a share of the width between 1 and 50 (percent)", o.qrLogoSize)
	}
	const budget = 0.8 // of what the error correction can rebuild: room for a real camera's misreads
	x0, y0, x1, y1, ok, _ := c.Hole(frac, budget)
	if !ok {
		best := 0.0
		for f := frac - 0.01; f >= 0.05; f -= 0.01 {
			if _, _, _, _, ok, _ := c.Hole(f, budget); ok {
				best = f
				break
			}
		}
		if best == 0 {
			return nil, fmt.Errorf("no room for a picture in this code (error correction %s); try --ec H or less text", c.Level)
		}
		return nil, fmt.Errorf("--logo-size %g is too big for this code (error correction %s): at most %.0f", o.qrLogoSize, c.Level, best*100)
	}
	l.X0, l.Y0, l.X1, l.Y1 = x0, y0, x1, y1
	return l, nil
}
