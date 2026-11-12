package webdemo

import (
	"fmt"
	"strings"
	"testing"
)

func TestParseCrop(t *testing.T) {
	b, err := ParseCrop("40,80,600,400")
	if err != nil || b.X != 40 || b.Y != 80 || b.W != 600 || b.H != 400 || b.DPR != 1 {
		t.Fatalf("ParseCrop: %+v %v", b, err)
	}
	for _, bad := range []string{"", "1,2,3", "a,b,c,d", "1,2,3,4,5", "0,0,0,100", "0,0,100,-1"} {
		if _, err := ParseCrop(bad); err == nil {
			t.Errorf("ParseCrop(%q) = no error, want error", bad)
		}
	}
}

func TestClampCrop(t *testing.T) {
	// a box inside the frame, DPR 1: even, unchanged but for rounding
	rc, ok := clampCrop(box{X: 40, Y: 80, W: 600, H: 400, DPR: 1}, 1440, 900)
	if !ok || rc != (rect{40, 80, 600, 400}) {
		t.Fatalf("inside: %+v %v", rc, ok)
	}
	// DPR 2 doubles every number into device pixels
	rc, ok = clampCrop(box{X: 10, Y: 20, W: 100, H: 50, DPR: 2}, 2880, 1800)
	if !ok || rc != (rect{20, 40, 200, 100}) {
		t.Fatalf("dpr2: %+v %v", rc, ok)
	}
	// a box running off the right/bottom is clamped to the frame
	rc, ok = clampCrop(box{X: 1400, Y: 880, W: 200, H: 200, DPR: 1}, 1440, 900)
	if !ok || rc.X+rc.W > 1440 || rc.Y+rc.H > 900 || rc.W <= 0 || rc.H <= 0 {
		t.Fatalf("clamp: %+v %v", rc, ok)
	}
	// odd numbers round to even (crop and yuv420p need it)
	rc, _ = clampCrop(box{X: 11, Y: 13, W: 101, H: 103, DPR: 1}, 1440, 900)
	if rc.X%2 != 0 || rc.Y%2 != 0 || rc.W%2 != 0 || rc.H%2 != 0 {
		t.Fatalf("odd: %+v", rc)
	}
	// a box wholly off the frame is rejected
	if _, ok := clampCrop(box{X: 2000, Y: 2000, W: 50, H: 50, DPR: 1}, 1440, 900); ok {
		t.Fatal("off-frame box accepted")
	}
}

func TestVideoArgsCrop(t *testing.T) {
	cr := &rect{100, 60, 600, 400}
	// no overlays: a -vf with crop before the even-size scale, output is the region
	a := strings.Join(videoArgs("/t/seq%05d.jpg", 1440, 900, cr, nil, nil, "", "/o/a.video.mp4"), " ")
	if !strings.Contains(a, "-vf crop=600:400:100:60,scale=") {
		t.Errorf("crop vf: %s", a)
	}
	// with a badge: the frame and the badge are both cropped, then overlaid
	badges := []pic{{"/t/ff6.png", 4, 6}}
	a = strings.Join(videoArgs("/t/seq%05d.jpg", 1440, 900, cr, badges, nil, "", "/o/a.video.mp4"), " ")
	if !strings.Contains(a, "[0:v]crop=600:400:100:60[c0];") || !strings.Contains(a, "[1:v]crop=600:400:100:60[p1];[c0][p1]overlay=0:0") {
		t.Errorf("crop badge: %s", a)
	}
	// captions variant: the cropped region (600x400) shrinks into the 1440x900
	// band, its aspect kept; the captions are NOT cropped (frame-sized)
	caps := []pic{{"/t/c0.png", 1, 2.5}}
	a = strings.Join(videoArgs("/t/seq%05d.jpg", 1440, 900, cr, nil, caps, "/o/a.voice.wav", "/o/a.captions.mp4"), " ")
	if !strings.Contains(a, "[c0]scale=") || !strings.Contains(a, "pad=1440:900:") {
		t.Errorf("crop band: %s", a)
	}
	if strings.Contains(a, "[1:v]crop=") { // the caption is overlaid whole
		t.Errorf("caption was cropped: %s", a)
	}
}

func TestBandAspectKept(t *testing.T) {
	// a 600x400 region into a 1440x900 frame: scaled to picScale of the height
	// (765), width from the region's 3:2 aspect, centered, padded to 1440x900
	var sw, sh, W, H, x, y int
	if _, err := fmt.Sscanf(band(600, 400, 1440, 900), "scale=%d:%d,pad=%d:%d:%d:%d", &sw, &sh, &W, &H, &x, &y); err != nil {
		t.Fatal(err)
	}
	if W != 1440 || H != 900 {
		t.Fatalf("pad target %dx%d", W, H)
	}
	if ratio := float64(sw) / float64(sh); ratio < 1.49 || ratio > 1.51 {
		t.Fatalf("aspect not kept: %dx%d = %.3f", sw, sh, ratio)
	}
	if x*2+sw != W {
		t.Fatalf("not centered: x=%d sw=%d W=%d", x, sw, W)
	}
}

func TestScriptRegion(t *testing.T) {
	src := `---
type: demo
app: app
region: "#console"
---

## Agents panel

What it does: the agents panel fills up.
Region: ".agents-lane"

Say: Agents arrive.
- go /
- expect text "ready"

## Whole page

What it does: nothing special.

Say: The whole page.
- go /
`
	s := Parse("t.md", src)
	if len(s.Problems) > 0 {
		t.Fatalf("problems: %v", s.Problems)
	}
	if s.Region != "#console" {
		t.Errorf("front-matter region = %q, want #console", s.Region)
	}
	if s.Features[0].Region != ".agents-lane" {
		t.Errorf("feature region = %q, want .agents-lane", s.Features[0].Region)
	}
	if s.Features[1].Region != "" {
		t.Errorf("feature 2 region = %q, want empty (falls back to the script's)", s.Features[1].Region)
	}
}

func TestRegionSpecPrecedence(t *testing.T) {
	r := &runner{s: &Script{Region: "#front"}, o: Options{}}
	feat := &Feature{Region: "#feature"}
	// feature line wins
	if sel, _ := r.regionSpec(feat); sel != "#feature" {
		t.Errorf("feature should win: %q", sel)
	}
	// --region beats the script's front matter
	r.o.Region = "#cli"
	if sel, _ := r.regionSpec(&Feature{}); sel != "#cli" {
		t.Errorf("--region should win over front matter: %q", sel)
	}
	// --crop beats front matter when no --region
	r.o.Region, r.o.Crop = "", "0,0,10,10"
	if sel, crop := r.regionSpec(&Feature{}); sel != "" || crop != "0,0,10,10" {
		t.Errorf("--crop: sel=%q crop=%q", sel, crop)
	}
	// nothing set at the top levels: the script's region: is the default
	r.o.Crop = ""
	if sel, _ := r.regionSpec(&Feature{}); sel != "#front" {
		t.Errorf("front-matter default: %q", sel)
	}
}
