package main

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"testing"
)

func TestEncodeAnimGIF(t *testing.T) {
	var frames []image.Image
	var delays []int
	for k := 0; k < 6; k++ {
		m := image.NewRGBA(image.Rect(0, 0, 40, 30))
		for i := range m.Pix {
			m.Pix[i] = 240
		}
		if k < 4 { // a dot that moves, then stops: frames 4 and 5 equal frame 3
			m.Set(5+k*5, 10, color.RGBA{200, 30, 30, 255})
		} else {
			m.Set(20, 10, color.RGBA{200, 30, 30, 255})
		}
		frames = append(frames, m)
		delays = append(delays, gifDelay(k, 15))
	}
	for _, loop := range []bool{true, false} {
		data, err := encodeAnimGIF(frames, delays, loop)
		if err != nil {
			t.Fatal(err)
		}
		g, err := gif.DecodeAll(bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		if len(g.Image) != 4 { // the still frames merged into the last move
			t.Errorf("frames %d, want 4", len(g.Image))
		}
		if want := map[bool]int{true: 0, false: -1}[loop]; g.LoopCount != want {
			t.Errorf("loop %v: LoopCount %d", loop, g.LoopCount)
		}
		if b := g.Image[1].Bounds(); b.Dx() > 6 || b.Dy() > 1 { // only the changed box
			t.Errorf("frame 2 covers %v", b)
		}
		sum := 0
		for _, d := range g.Delay {
			sum += d
		}
		if sum != 40 { // 6 frames at 15 fps
			t.Errorf("delays add to %d", sum)
		}
	}
}

func TestIGFilmSizes(t *testing.T) {
	if w, h := igFilmCSS(9, 19.5); w != 540 || h != 1170 {
		t.Errorf("phone %d×%d", w, h)
	}
	if w, h := igFilmCSS(16, 9); w != 1280 || h != 720 {
		t.Errorf("landscape %d×%d", w, h)
	}
	if igVideoWidth("phone", 9, 19.5) != 1080 || igVideoWidth("landscape", 16, 9) != 1920 || igVideoWidth("1080x1350", 1080, 1350) != 1080 {
		t.Error("video widths")
	}
	if igGIFWidth != 720 {
		t.Error("GIF width")
	}
}
