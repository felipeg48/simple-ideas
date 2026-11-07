package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestFFmpegArgs(t *testing.T) {
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a.png"), filepath.Join(dir, "b.png")
	os.WriteFile(a, []byte("A"), 0o644)
	os.WriteFile(b, []byte("B"), 0o644)
	seq := t.TempDir()
	pattern, n, err := frameRun([]videoFrame{{a, 0.1}, {b, 0.01}}, seq)
	if err != nil || n != 4 || pattern != filepath.Join(seq, "seq%05d.png") {
		t.Fatalf("%q %d %v (0.1 s is 3 frames, a blink still 1)", pattern, n, err)
	}
	if got, _ := os.ReadFile(filepath.Join(seq, "seq00003.png")); string(got) != "B" {
		t.Errorf("last frame %q", got)
	}
	args := strings.Join(ffmpegArgs("p", "out.mp4"), " ")
	for _, w := range []string{"-framerate 30 -i p", "format=yuv420p", "trunc(iw/2)*2", "-c:v libx264", "-movflags +faststart", "out.mp4"} {
		if !strings.Contains(args, w) {
			t.Errorf("ffmpeg args lack %q: %s", w, args)
		}
	}
}

func TestEncodeMP4(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		if err := encodeMP4([]videoFrame{{"x.png", 1}}, "out.mp4"); err == nil || !strings.Contains(err.Error(), "brew install ffmpeg") {
			t.Errorf("without ffmpeg the error should say how to install it: %v", err)
		}
		t.Skip("ffmpeg not installed")
	}
	dir := t.TempDir()
	var frames []videoFrame
	for i, c := range []color.RGBA{{200, 40, 40, 255}, {40, 40, 200, 255}} {
		img := image.NewRGBA(image.Rect(0, 0, 63, 41)) // odd sizes: the encoder must even them
		for p := 0; p < len(img.Pix); p += 4 {
			img.Pix[p], img.Pix[p+1], img.Pix[p+2], img.Pix[p+3] = c.R, c.G, c.B, c.A
		}
		p := filepath.Join(dir, strconv.Itoa(i)+".png")
		f, _ := os.Create(p)
		png.Encode(f, img)
		f.Close()
		frames = append(frames, videoFrame{p, []float64{1, 0.5}[i]})
	}
	out := filepath.Join(dir, "v.mp4")
	if err := encodeMP4(frames, out); err != nil {
		t.Fatal(err)
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		return
	}
	got, err := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration:stream=codec_name,pix_fmt,width", "-of", "default=nw=1", out).Output()
	if err != nil {
		t.Fatal(err)
	}
	s := string(got)
	if !strings.Contains(s, "codec_name=h264") || !strings.Contains(s, "pix_fmt=yuv420p") || !strings.Contains(s, "width=62") {
		t.Errorf("probe: %s", s)
	}
	for _, l := range strings.Split(s, "\n") {
		if d, ok := strings.CutPrefix(l, "duration="); ok {
			if v, _ := strconv.ParseFloat(d, 64); v < 1.3 || v > 1.7 {
				t.Errorf("duration %v, want about 1.5 s", v)
			}
		}
	}
}
