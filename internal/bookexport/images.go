package bookexport

import (
	"bytes"
	"encoding/xml"
	"image"
	_ "image/gif" // image.DecodeConfig reads GIF sizes
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Options are what the writers need from outside: a way to turn an SVG into
// a PNG for formats that cannot show SVG (Word, RTF), and the paper size.
type Options struct {
	// Raster draws an SVG file as a PNG: its bytes and its size in CSS
	// pixels. Nil, or an error, leaves a placeholder in the .docx and .rtf.
	Raster func(svgPath string) (png []byte, w, h int, err error)
}

// picture is an image's bytes, kind (png, jpeg, gif, svg) and size in CSS pixels.
type picture struct {
	data []byte
	ext  string
	w, h int
}

// load reads an image file; raster true turns an SVG into a PNG (or fails).
func (o Options) load(src string, raster bool) (picture, bool) {
	if strings.Contains(src, "://") || strings.HasPrefix(src, "data:") {
		return picture{}, false
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return picture{}, false
	}
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(src), "."))
	if ext == "jpg" {
		ext = "jpeg"
	}
	if ext == "svg" {
		if !raster {
			w, h := svgSize(data)
			return picture{data, "svg", w, h}, true
		}
		if o.Raster == nil {
			return picture{}, false
		}
		png, w, h, err := o.Raster(src)
		if err != nil || len(png) == 0 {
			return picture{}, false
		}
		return picture{png, "png", w, h}, true
	}
	cfg, kind, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return picture{}, false
	}
	return picture{data, kind, cfg.Width, cfg.Height}, true
}

var svgNumRE = regexp.MustCompile(`^[\d.]+`)

// svgSize is an SVG's size from its width/height or viewBox (600×400 if none).
func svgSize(data []byte) (int, int) {
	d := xml.NewDecoder(bytes.NewReader(data))
	for {
		t, err := d.Token()
		if err != nil {
			return 600, 400
		}
		se, ok := t.(xml.StartElement)
		if !ok || se.Name.Local != "svg" {
			continue
		}
		var w, h float64
		var vb []float64
		for _, a := range se.Attr {
			switch a.Name.Local {
			case "width":
				if !strings.HasSuffix(a.Value, "%") {
					w, _ = strconv.ParseFloat(svgNumRE.FindString(a.Value), 64)
				}
			case "height":
				if !strings.HasSuffix(a.Value, "%") {
					h, _ = strconv.ParseFloat(svgNumRE.FindString(a.Value), 64)
				}
			case "viewBox":
				for _, f := range strings.Fields(strings.ReplaceAll(a.Value, ",", " ")) {
					v, _ := strconv.ParseFloat(f, 64)
					vb = append(vb, v)
				}
			}
		}
		if (w == 0 || h == 0) && len(vb) == 4 {
			w, h = vb[2], vb[3]
		}
		if w == 0 || h == 0 {
			return 600, 400
		}
		return int(w + .5), int(h + .5)
	}
}

// SVGSize is an SVG file's size in CSS pixels (from width/height or viewBox).
func SVGSize(path string) (int, int) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 600, 400
	}
	return svgSize(data)
}

// fit scales w×h (CSS pixels) to at most maxW points wide, in points.
func fit(w, h int, maxW float64) (float64, float64) {
	pw, ph := float64(w)*0.75, float64(h)*0.75
	if pw > maxW {
		ph, pw = ph*maxW/pw, maxW
	}
	return pw, ph
}

// names gives every local image of the book a unique file name, in order.
func names(b *Book) (order []string, name map[string]string) {
	name = map[string]string{}
	taken := map[string]bool{}
	add := func(src string) {
		if src == "" || name[src] != "" || strings.Contains(src, "://") || strings.HasPrefix(src, "data:") {
			return
		}
		base := filepath.Base(src)
		n := base
		for i := 2; taken[strings.ToLower(n)]; i++ {
			n = strings.TrimSuffix(base, filepath.Ext(base)) + "-" + strconv.Itoa(i) + filepath.Ext(base)
		}
		taken[strings.ToLower(n)] = true
		name[src] = n
		order = append(order, src)
	}
	var walk func([]Block)
	walk = func(bs []Block) {
		for _, bl := range bs {
			if bl.Kind == Image {
				add(bl.Src)
			}
			for _, it := range bl.Items {
				walk(it)
			}
			walk(bl.Blocks)
		}
	}
	walk(b.Colophon)
	for _, c := range b.Chapters {
		walk(c.Blocks)
	}
	return order, name
}
