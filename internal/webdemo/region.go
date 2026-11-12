package webdemo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"strconv"
	"strings"
)

// Region capture: si demo run and the screenshot step film the whole viewport;
// a region (--region "<css>" / a feature's Region: line / front matter region:,
// or the fixed --crop x,y,w,h box) crops the output to one element. The cursor,
// clicks and notes are drawn in the page and so stay in the picture; only the
// output frame is cropped. The crop is applied in ffmpeg (video) or on the PNG
// (screenshot); the captions variant shrinks the cropped region into the frame,
// its aspect kept.

// rect is a crop box in device (frame) pixels.
type rect struct{ X, Y, W, H int }

// box is an element's bounding box in CSS pixels, with the picture's pixels per
// CSS pixel (cropFor and cropShot set it from the frame's width over the
// viewport's: 2 in a supersampled take; a fixed --crop box is in CSS pixels).
type box struct {
	X, Y, W, H, DPR float64
}

// clampCrop turns a CSS box (scaled by DPR) into an even, in-frame device-pixel
// rect. frameW, frameH are the frame's size in device pixels. Even offsets and
// sizes keep ffmpeg's crop and yuv420p happy. It returns false when nothing of
// the box lands on the frame.
func clampCrop(b box, frameW, frameH int) (rect, bool) {
	dpr := b.DPR
	if dpr <= 0 {
		dpr = 1
	}
	x0 := clampf(b.X*dpr, 0, float64(frameW))
	y0 := clampf(b.Y*dpr, 0, float64(frameH))
	x1 := clampf((b.X+b.W)*dpr, 0, float64(frameW))
	y1 := clampf((b.Y+b.H)*dpr, 0, float64(frameH))
	x := int(x0) &^ 1
	y := int(y0) &^ 1
	w := (int(x1) - x) &^ 1
	h := (int(y1) - y) &^ 1
	if x+w > frameW {
		w = (frameW - x) &^ 1
	}
	if y+h > frameH {
		h = (frameH - y) &^ 1
	}
	if w <= 0 || h <= 0 {
		return rect{}, false
	}
	return rect{x, y, w, h}, true
}

func clampf(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// ParseCrop reads a fixed "x,y,w,h" box (pixels) for --crop.
func ParseCrop(s string) (box, error) {
	bad := fmt.Errorf("crop %q: write x,y,w,h in pixels (e.g. 40,80,600,400)", s)
	parts := strings.Split(s, ",")
	if len(parts) != 4 {
		return box{}, bad
	}
	var n [4]float64
	for i, p := range parts {
		v, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
		if err != nil {
			return box{}, bad
		}
		n[i] = v
	}
	if n[2] <= 0 || n[3] <= 0 {
		return box{}, fmt.Errorf("crop %q: width and height must be positive", s)
	}
	return box{X: n[0], Y: n[1], W: n[2], H: n[3], DPR: 1}, nil
}

// regionJS measures an element's bounding box and the page's DPR.
const regionJS = `(function(sel){try{var e=document.querySelector(sel);if(!e)return null;var r=e.getBoundingClientRect();return {x:r.left,y:r.top,w:r.width,h:r.height,dpr:window.devicePixelRatio||1}}catch(_){return {bad:1}}})`

// measureRegion reads the box of the element the selector matches, on the page
// in front now.
func (r *runner) measureRegion(sel string) (box, error) {
	arg, _ := json.Marshal(sel)
	v, err := r.c.Eval(regionJS + "(" + string(arg) + ")")
	if err != nil {
		return box{}, err
	}
	if strings.TrimSpace(string(v)) == "null" {
		return box{}, fmt.Errorf("region %q matched no element on the page", sel)
	}
	var b struct {
		X, Y, W, H, DPR float64
		Bad             int
	}
	if err := json.Unmarshal(v, &b); err != nil {
		return box{}, err
	}
	if b.Bad != 0 {
		return box{}, fmt.Errorf("region %q is not a valid CSS selector", sel)
	}
	if b.W <= 0 || b.H <= 0 {
		return box{}, fmt.Errorf("region %q has no size (the element is hidden)", sel)
	}
	return box{b.X, b.Y, b.W, b.H, b.DPR}, nil
}

// regionSpec is the region in force for feature f: a feature's Region: line,
// then --region, then --crop, then the script's region:. Empty sel and crop
// means the whole page.
func (r *runner) regionSpec(f *Feature) (sel, crop string) {
	switch {
	case f != nil && f.Region != "":
		return f.Region, ""
	case r.o.Region != "":
		sel, _ := regionValue(r.o.Region)
		return sel, ""
	case r.o.Crop != "":
		return "", r.o.Crop
	case r.s.Region != "":
		return r.s.Region, ""
	}
	return "", ""
}

// regionZoom: feature f's region is written with · zoom (the clip zooms into
// it instead of starting cropped).
func (r *runner) regionZoom(f *Feature) bool {
	switch {
	case f != nil && f.Region != "":
		return f.Zoom
	case r.o.Region != "":
		_, z := regionValue(r.o.Region)
		return z
	case r.o.Crop != "":
		return false
	}
	return r.s.Zoom
}

// cropFor resolves feature f's region to a device-pixel rect against a
// frameW×frameH frame, or nil when there is no region. A region that is set but
// cannot be measured or lands off-frame is an error (a clear failure, not a
// silent whole-page clip).
func (r *runner) cropFor(f *Feature, frameW, frameH int) (*rect, error) {
	sel, crop := r.regionSpec(f)
	var b box
	var err error
	switch {
	case sel != "":
		if b, err = r.measureRegion(sel); err != nil {
			return nil, err
		}
	case crop != "":
		if b, err = ParseCrop(crop); err != nil {
			return nil, err
		}
	default:
		return nil, nil
	}
	b.DPR = float64(frameW) / float64(r.s.W) // frame pixels per CSS pixel (a supersampled take is 2)
	rc, ok := clampCrop(b, frameW, frameH)
	if !ok {
		return nil, fmt.Errorf("region is outside the %dx%d frame", frameW, frameH)
	}
	return &rc, nil
}

// cropShot crops a screenshot to feature f's region. It is best-effort: a
// missing or off-frame region leaves the full picture (the report still gets a
// shot). The video path (cropFor) is where a bad region fails loudly.
func (r *runner) cropShot(raw []byte, f *Feature) []byte {
	sel, crop := r.regionSpec(f)
	if sel == "" && crop == "" {
		return raw
	}
	img, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		return raw
	}
	var b box
	if sel != "" {
		if b, err = r.measureRegion(sel); err != nil {
			return raw
		}
	} else if b, err = ParseCrop(crop); err != nil {
		return raw
	}
	b.DPR = float64(img.Bounds().Dx()) / float64(r.s.W)
	rc, ok := clampCrop(b, img.Bounds().Dx(), img.Bounds().Dy())
	if !ok {
		return raw
	}
	sub, ok := img.(interface {
		SubImage(image.Rectangle) image.Image
	})
	if !ok {
		return raw
	}
	cropped := sub.SubImage(image.Rect(rc.X, rc.Y, rc.X+rc.W, rc.Y+rc.H))
	var buf bytes.Buffer
	if png.Encode(&buf, cropped) != nil {
		return raw
	}
	return buf.Bytes()
}

// DefaultScale is the device pixels per CSS pixel si demo run films at: 2,
// supersampling. Every picture of a take (frames, title cards, captions,
// badges) is twice the viewport's size; the videos are scaled back down to it
// at the end (lanczos), so text stays crisp, under a zoom most of all.
// --scale 1 (or scale: 1) films at the viewport's size: faster, smaller files.
const DefaultScale = 2

// ParseScale reads --scale / scale: 1, 2 or 3 (a trailing x allowed).
func ParseScale(s string) (int, error) {
	k, err := strconv.Atoi(strings.TrimSuffix(strings.TrimSpace(s), "x"))
	if err != nil || k < 1 || k > 3 {
		return 0, fmt.Errorf("scale %q: write 1, 2 or 3 (device pixels per CSS pixel when filming; 2 is the default)", s)
	}
	return k, nil
}

func firstPositive(a ...int) int {
	for _, v := range a {
		if v > 0 {
			return v
		}
	}
	return 1
}

// screenshot is the page in front as a PNG of the viewport's size: a
// supersampled page's picture is scaled down (shrink).
func (r *runner) screenshot() ([]byte, error) {
	raw, err := r.c.Screenshot()
	if err != nil || r.k <= 1 {
		return raw, err
	}
	m, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		return raw, nil
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, shrink(m, r.k)); err != nil {
		return raw, nil
	}
	return buf.Bytes(), nil
}

// shrink scales m down k times, each pixel the mean of a k×k block (a box
// filter: exact for a whole factor).
func shrink(m image.Image, k int) *image.NRGBA {
	b := m.Bounds()
	w, h := b.Dx()/k, b.Dy()/k
	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	n := uint32(k * k)
	for y := range h {
		for x := range w {
			var sr, sg, sb, sa uint32
			for j := range k {
				for i := range k {
					cr, cg, cb, ca := m.At(b.Min.X+x*k+i, b.Min.Y+y*k+j).RGBA()
					sr, sg, sb, sa = sr+cr, sg+cg, sb+cb, sa+ca
				}
			}
			out.Set(x, y, color.RGBA64{uint16(sr / n), uint16(sg / n), uint16(sb / n), uint16(sa / n)})
		}
	}
	return out
}
