package webdemo

import (
	"image"
	"image/color"
	"math"
	"strings"
	"testing"
)

// A supersampled take (issue #6): frames are k times the viewport, every
// rectangle is in frame pixels, the clip is scaled down k times at the end.

func TestParseScale(t *testing.T) {
	for in, want := range map[string]int{"1": 1, "2": 2, " 3 ": 3, "2x": 2} {
		if k, err := ParseScale(in); err != nil || k != want {
			t.Errorf("ParseScale(%q) = %d, %v", in, k, err)
		}
	}
	for _, in := range []string{"0", "4", "1.5", "big", ""} {
		if _, err := ParseScale(in); err == nil {
			t.Errorf("ParseScale(%q) accepted", in)
		}
	}
	if firstPositive(0, 0, DefaultScale) != 2 || firstPositive(1, 0, 2) != 1 || firstPositive(0, 3, 2) != 3 {
		t.Error("firstPositive: --scale, then scale:, then the default")
	}
	s := Parse("/x/a.md", "---\ntype: demo\napp: http://x\nscale: 1\n---\n\n## F\n\n- go /\n")
	if s.Scale != 1 || len(s.Problems) > 0 {
		t.Errorf("scale: front matter = %d %v", s.Scale, s.Problems)
	}
	if s = Parse("/x/a.md", "---\ntype: demo\napp: http://x\nscale: 5\n---\n\n## F\n\n- go /\n"); len(s.Problems) != 1 {
		t.Errorf("scale: 5 = %v", s.Problems)
	}
}

func TestScaledCameraIsTheSameShot(t *testing.T) {
	// the same zoom on a 1x and on a 2x frame frames the same part of the page
	mv := []move{{At: 1, Dur: 1, Kind: "zoom", B: box{X: 350, Y: 200, W: 20, H: 10}}}
	one := resolveMoves(mv, nil, rectF{0, 0, 1440, 900}, 1)
	two := resolveMoves(mv, nil, rectF{0, 0, 2880, 1800}, 2)
	a, b := one[0].To, two[0].To
	if b != (rectF{a.X * 2, a.Y * 2, a.W * 2, a.H * 2}) {
		t.Errorf("1x %+v, 2x %+v", a, b)
	}
	// the 3x zoom crops a third of a 2x frame: 960 frame pixels for 1440 out,
	// so it is never more than 1.5x magnified (3x at scale 1)
	if math.Abs(b.W-2880/maxZoom) > 1e-9 || 1440/b.W > 1.5 {
		t.Errorf("max zoom at 2x: %+v", b)
	}
}

func TestScaledRegionCrop(t *testing.T) {
	// a CSS box on a 2x frame: twice the numbers, still even
	rc, ok := clampCrop(box{X: 50, Y: 30, W: 300, H: 200, DPR: 2}, 2880, 1800)
	if !ok || rc != (rect{100, 60, 600, 400}) {
		t.Errorf("2x crop: %+v", rc)
	}
}

func TestVideoArgsScaledDown(t *testing.T) {
	// k 2: crop in frame pixels, then lanczos down to half (the region's CSS size)
	cr := &rect{200, 120, 1200, 800}
	a := strings.Join(videoArgs("/t/seq%05d.jpg", 2880, 1800, 2, cr, nil, nil, "", "/o/a.mp4"), " ")
	if !strings.Contains(a, "-vf crop=1200:800:200:120,scale=trunc(iw/2/2)*2:trunc(ih/2/2)*2:flags=lanczos") {
		t.Errorf("2x crop: %s", a)
	}
	// badges (drawn at frame size) cropped like the frame; captions over the
	// 2880x1800 band, then the whole is scaled down to 1440x900
	badges, caps := []pic{{"/t/ff6.png", 4, 6}}, []pic{{"/t/c0.png", 1, 2}}
	a = strings.Join(videoArgs("/t/seq%05d.jpg", 2880, 1800, 2, cr, badges, caps, "/o/v.wav", "/o/a.captions.mp4"), " ")
	for _, want := range []string{"[1:v]crop=1200:800:200:120[p1]", "pad=2880:1800:", "[band][2:v]overlay=0:0", "scale=trunc(iw/2/2)*2:trunc(ih/2/2)*2:flags=lanczos"} {
		if !strings.Contains(a, want) {
			t.Errorf("no %q in %s", want, a)
		}
	}
	// k 1: the size is kept
	a = strings.Join(videoArgs("/t/seq%05d.jpg", 1440, 900, 1, nil, nil, nil, "", "/o/a.mp4"), " ")
	if !strings.Contains(a, "scale=trunc(iw/1/2)*2:trunc(ih/1/2)*2") {
		t.Errorf("1x: %s", a)
	}
}

func TestShrink(t *testing.T) {
	m := image.NewNRGBA(image.Rect(0, 0, 4, 2))
	for x := range 4 {
		for y := range 2 {
			v := uint8(0)
			if x%2 == 0 {
				v = 200
			}
			m.Set(x, y, color.NRGBA{v, v, v, 255})
		}
	}
	s := shrink(m, 2)
	if s.Bounds().Dx() != 2 || s.Bounds().Dy() != 1 {
		t.Fatalf("size %v", s.Bounds())
	}
	if c := s.NRGBAAt(1, 0); c.R != 100 || c.A != 255 {
		t.Errorf("mean of a 2x2 block: %+v", c)
	}
}
