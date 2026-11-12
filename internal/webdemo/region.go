package webdemo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
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

// box is an element's bounding box in CSS pixels, with the page's device-pixel
// ratio (a fixed --crop box carries DPR 1: its numbers are pixels as written).
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
