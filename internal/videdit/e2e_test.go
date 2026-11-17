package videdit

import (
	"bytes"
	"fmt"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felipeg48/simple-ideas/internal/tools"
	"github.com/felipeg48/simple-ideas/internal/webdemo"
)

// End to end on short generated clips: needs ffmpeg, ffprobe and Chrome
// (skipped without them, and in -short).

func needTools(t *testing.T) (chrome, ff, fp string) {
	t.Helper()
	chrome, ff, fp = tools.Find("chrome"), FFmpeg(), FFprobe()
	if chrome == "" || ff == "" || fp == "" || testing.Short() {
		t.Skip("needs Chrome, ffmpeg and ffprobe")
	}
	return
}

// colours writes a clip of 2.5 s per colour (w×h, 30 fps) with a tone.
func colours(t *testing.T, ff, file string, w, h int, cs ...string) {
	t.Helper()
	var a []string
	var g strings.Builder
	for i, c := range cs {
		a = append(a, "-f", "lavfi", "-i", fmt.Sprintf("color=c=%s:s=%dx%d:r=30:d=2.5", c, w, h))
		fmt.Fprintf(&g, "[%d:v]", i)
	}
	a = append(a, "-f", "lavfi", "-i", fmt.Sprintf("sine=f=440:d=%g", 2.5*float64(len(cs))))
	fmt.Fprintf(&g, "concat=n=%d:v=1:a=0,format=yuv420p[v]", len(cs))
	a = append(a, "-filter_complex", g.String(), "-map", "[v]", "-map", fmt.Sprintf("%d:a", len(cs)), "-c:v", "libx264", "-preset", "medium", "-crf", "20", "-r", "30", "-g", "30",
		"-c:a", "aac", "-ar", "48000", "-ac", "2", "-shortest", file)
	if out, err := exec.Command(ff, append([]string{"-nostdin", "-y", "-hide_banner", "-loglevel", "error"}, a...)...).CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
}

// pixel is the colour at x,y of the frame at t seconds.
func pixel(t *testing.T, ff, file string, at float64, x, y int) (r, g, b int) {
	t.Helper()
	out, err := exec.Command(ff, "-nostdin", "-hide_banner", "-loglevel", "error", "-ss", fmt.Sprint(at), "-i", file, "-frames:v", "1", "-f", "image2pipe", "-vcodec", "png", "-").Output()
	if err != nil {
		t.Fatal(err)
	}
	m, err := png.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
	R, G, B, _ := m.At(x, y).RGBA()
	return int(R >> 8), int(G >> 8), int(B >> 8)
}

func is(r, g, b int, want string) bool {
	switch want {
	case "red":
		return r > 200 && g < 60 && b < 60
	case "green":
		return r < 60 && g > 100 && b < 60
	case "blue":
		return r < 60 && g < 60 && b > 200
	case "yellow":
		return r > 200 && g > 200 && b < 60
	}
	return false
}

func duration(t *testing.T, fp, file string) float64 {
	t.Helper()
	in, err := Probe(fp, file)
	if err != nil {
		t.Fatal(err)
	}
	return in.D
}

// fakeVoice voices any text as a 1 s tone (no voice engine in tests).
type fakeVoice struct{ ff, dir string }

func (f fakeVoice) Voice(v webdemo.Voice, texts []string) (map[string]webdemo.Line, int, error) {
	out := map[string]webdemo.Line{}
	for i, s := range texts {
		w := filepath.Join(f.dir, fmt.Sprintf("say%d.wav", i))
		if b, err := exec.Command(f.ff, "-nostdin", "-y", "-loglevel", "error", "-f", "lavfi", "-i", "sine=f=880:d=1", "-ar", "24000", "-ac", "1", w).CombinedOutput(); err != nil {
			return nil, 0, fmt.Errorf("%v %s", err, b)
		}
		out[s] = webdemo.Line{WAV: w, Seconds: 1}
	}
	return out, len(texts), nil
}

func TestBuildEndToEnd(t *testing.T) {
	chrome, ff, fp := needTools(t)
	dir := t.TempDir()
	colours(t, ff, filepath.Join(dir, "src.mp4"), 320, 180, "red", "green", "blue", "yellow")
	os.WriteFile(filepath.Join(dir, "src.vtt"), []byte("WEBVTT\n\n00:00:00.200 --> 00:00:00.900\nRed first.\n\n00:00:03.000 --> 00:00:04.000\nNow green.\n\n00:00:08.000 --> 00:00:09.000\nYellow, fast.\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "chapters.txt"), []byte("0:00 Red\n0:05 Blue\n"), 0o644)
	md := filepath.Join(dir, "edit.md")
	os.WriteFile(md, []byte(`---
type: video
source: src.mp4
out: out/final.mp4
captions: on
---
# Colours

## Timeline

- 0:01–0:02 · cut
- 0:06 · freeze 1s
- 0:08–0:10 · speed 2x · badge "2×"
- 0:00.5 · box 10,10 100x60 · 2s · out none
- 0:03 · say "Green now." · duck 20%
- at said "now green" · bubble "Look" · to 200,120 · 1s
- at chapter "Blue" + 2s · keys "⌘ K" · 1s
- end · card "Done" · 1s · fade
`), 0o644)
	e, err := ParseFile(md)
	if err != nil || len(e.Problems) > 0 {
		t.Fatal(err, e.Problems)
	}
	if note := e.Check(fp); note != "" || len(e.Problems) > 0 {
		t.Fatal(note, e.Problems)
	}
	e, _ = ParseFile(md)
	res, err := Build(e, Options{Chrome: chrome, FFmpeg: ff, FFprobe: fp, Synth: fakeVoice{ff, dir}, Captions: true})
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out", "final.mp4")
	if res.Out != out || len(res.Files) != 3 {
		t.Fatalf("result %+v", res)
	}
	// 10 s: -1 (cut) +1 (freeze) -1 (2x over 2 s) +1 (card)
	if d := duration(t, fp, out); d < 9.9 || d > 10.15 {
		t.Errorf("duration %g, want 10", d)
	}
	for _, c := range []struct {
		at   float64
		want string
	}{{0.2, "red"}, {1.2, "red"}, {2.0, "green"}, {5.5, "blue"}, {7.8, "yellow"}, {8.6, "yellow"}} {
		if r, g, b := pixel(t, ff, out, c.at, 160, 90); !is(r, g, b, c.want) {
			t.Errorf("at %g: %d,%d,%d, want %s", c.at, r, g, b, c.want)
		}
	}
	// the box's left edge (accent blue) over red at 1.2 s, not at 0.2 s
	if r, g, b := pixel(t, ff, out, 1.2, 11, 40); is(r, g, b, "red") {
		t.Errorf("no box at 1.2 s: %d,%d,%d", r, g, b)
	}
	if r, g, b := pixel(t, ff, out, 0.2, 11, 40); !is(r, g, b, "red") {
		t.Errorf("a box before its time: %d,%d,%d", r, g, b)
	}
	// the card is the theme's background, not a colour of the source
	if r, g, b := pixel(t, ff, out, 9.6, 10, 10); r < 200 || g < 200 || b < 200 {
		t.Errorf("card: %d,%d,%d", r, g, b)
	}
	// captions moved: red first at 0.2, now green (3 s) at 2, yellow fast (8 s) at 8; the said line at 2
	vtt, _ := os.ReadFile(filepath.Join(dir, "out", "final.vtt"))
	for _, w := range []string{"00:00:00.200 --> 00:00:00.900\nRed first.", "00:00:02.000 --> 00:00:03.000\nGreen now.", "00:00:02.000 --> 00:00:03.000\nNow green.", "00:00:08.000 --> 00:00:08.500\nYellow, fast."} {
		if !strings.Contains(string(vtt), w) {
			t.Errorf("vtt lacks %q:\n%s", w, vtt)
		}
	}
	// chapters moved (Blue at 5 s is 4 s in)
	in, _ := Probe(fp, out)
	if len(in.Chapters) != 2 || in.Chapters[1].Title != "Blue" || in.Chapters[1].Start < 3.9 || in.Chapters[1].Start > 4.1 {
		t.Errorf("chapters %+v", in.Chapters)
	}
	if !fileExists(filepath.Join(dir, "out", "final.captions.mp4")) {
		t.Error("no captions variant")
	}
}

func TestBuildZoomBlurAndJoin(t *testing.T) {
	chrome, ff, fp := needTools(t)
	dir := t.TempDir()
	// a blue frame with a red top-left quarter
	src := filepath.Join(dir, "quad.mp4")
	if out, err := exec.Command(ff, "-nostdin", "-y", "-loglevel", "error", "-f", "lavfi", "-i", "color=c=blue:s=320x180:r=30:d=4", "-f", "lavfi", "-i", "color=c=red:s=160x90:r=30:d=4",
		"-f", "lavfi", "-i", "color=c=green:s=160x90:r=30:d=4", "-filter_complex", "[0][1]overlay=0:0[q];[q][2]overlay=160:90,format=yuv420p", "-an", "-c:v", "libx264", src).CombinedOutput(); err != nil {
		t.Fatal(err, string(out))
	}
	colours(t, ff, filepath.Join(dir, "green.mp4"), 640, 360, "green")
	md := filepath.Join(dir, "edit.md")
	os.WriteFile(md, []byte("---\ntype: video\nsource:\n  - quad.mp4\n  - green.mp4 · from 0:01\njoin: fade 0.5s\nout: z.mp4\n---\n- 0:00.5 · zoom to 0,0 160x90 · 1s · move 0.3s\n- 0:02.3 · zoom to 160,90 160x90 · 0.4s · move 0.2s\n- 0:03.3–0:03.8 · pixelate 125,50 80x80\n"), 0o644)
	e, _ := ParseFile(md)
	if note := e.Check(fp); note != "" || len(e.Problems) > 0 {
		t.Fatal(note, e.Problems)
	}
	e, _ = ParseFile(md)
	if _, err := Build(e, Options{Chrome: chrome, FFmpeg: ff, FFprobe: fp}); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "z.mp4")
	if d := duration(t, fp, out); d < 5.4 || d > 5.6 { // 4 + 1.5, the fade keeps times
		t.Errorf("duration %g", d)
	}
	if r, g, b := pixel(t, ff, out, 0.2, 300, 40); !is(r, g, b, "blue") {
		t.Errorf("before the zoom: %d,%d,%d", r, g, b)
	}
	if r, g, b := pixel(t, ff, out, 1.2, 300, 170); !is(r, g, b, "red") { // zoomed into the red quarter
		t.Errorf("zoomed: %d,%d,%d", r, g, b)
	}
	if r, g, b := pixel(t, ff, out, 2.15, 300, 40); !is(r, g, b, "blue") {
		t.Errorf("after the zoom: %d,%d,%d", r, g, b)
	}
	if r, g, b := pixel(t, ff, out, 2.7, 10, 10); !is(r, g, b, "green") { // zoomed into the green bottom-right quarter
		t.Errorf("zoomed bottom-right: %d,%d,%d", r, g, b)
	}
	if r, g, b := pixel(t, ff, out, 5.0, 160, 90); !is(r, g, b, "green") { // the second source
		t.Errorf("joined: %d,%d,%d", r, g, b)
	}
	// pixelated: at the red/blue edge (x 160) a block mixes both
	if r, g, b := pixel(t, ff, out, 3.2, 162, 60); !is(r, g, b, "blue") {
		t.Errorf("before: %d,%d,%d", r, g, b)
	}
	if r, g, b := pixel(t, ff, out, 3.55, 162, 60); is(r, g, b, "blue") {
		t.Errorf("not pixelated: %d,%d,%d", r, g, b)
	}
	if !fileExists(filepath.Join(dir, ".si", "video", "edit.joined.mp4")) {
		t.Error("the join is kept in .si/video")
	}
}

// refilm: a tiny demo against the example app, its feature spliced over a
// range of a "film" whose head stays byte for byte.
func TestRefilm(t *testing.T) {
	chrome, ff, fp := needTools(t)
	dir := t.TempDir()
	colours(t, ff, filepath.Join(dir, "film.mp4"), 640, 480, "red", "red", "green", "blue", "blue", "yellow")
	app, _ := filepath.Abs("../../examples/webdemo/app")
	os.WriteFile(filepath.Join(dir, "film.md"), []byte("---\ntype: demo\ntitle: Film\napp: "+app+"\nviewport: 640x480\n---\n\n## Sign in\n\n- type \"demo\" into \"Username\"\n- click \"Sign in\"\n"), 0o644)
	md := filepath.Join(dir, "fix.md")
	os.WriteFile(md, []byte("---\ntype: video\nsource: film.mp4\nout: fixed.mp4\n---\n- 0:07.5–0:10 · refilm feature \"Sign in\" · fade 0.3s\n"), 0o644)
	e, _ := ParseFile(md)
	if e.Check(fp); len(e.Problems) > 0 {
		t.Fatal(e.Problems)
	}
	bad := Parse(md, "---\ntype: video\nsource: film.mp4\n---\n- 0:01–0:02 · refilm feature \"Nope\"\n")
	if bad.Check(fp); len(bad.Problems) != 1 || !strings.Contains(bad.Problems[0].Msg, `no feature "Nope" in film.md ("Sign in")`) {
		t.Errorf("problems %v", bad.Problems)
	}
	e, _ = ParseFile(md)
	var log strings.Builder
	res, err := Build(e, Options{Chrome: chrome, FFmpeg: ff, FFprobe: fp, Log: &log})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Fast {
		t.Log(log.String())
		t.Error("a refilm alone keeps the rest by stream copy")
	}
	out := filepath.Join(dir, "fixed.mp4")
	if r, g, b := pixel(t, ff, out, 1, 320, 240); !is(r, g, b, "red") {
		t.Errorf("head: %d,%d,%d", r, g, b)
	}
	d := duration(t, fp, out)
	if d < 12 {
		t.Errorf("duration %g", d)
	}
	if r, g, b := pixel(t, ff, out, d-0.5, 320, 240); !is(r, g, b, "yellow") {
		t.Errorf("tail: %d,%d,%d", r, g, b)
	}
	// the head's frames are the source's packets (a keyframe may carry the
	// parameter sets in band: only the others are compared)
	hash := func(f string) []string {
		raw, err := exec.Command(fp, "-v", "error", "-select_streams", "v:0", "-show_entries", "packet=flags,data_hash", "-show_data_hash", "md5", "-of", "csv=p=0", f).Output()
		if err != nil {
			t.Fatal(err)
		}
		var hs []string
		for _, l := range strings.Fields(string(raw)) {
			if fs := strings.Split(l, ","); len(fs) == 2 && !strings.Contains(fs[0], "K") {
				hs = append(hs, fs[1])
			}
		}
		return hs
	}
	a, b := hash(filepath.Join(dir, "film.mp4")), hash(out)
	if len(a) < 150 || len(b) < 150 || strings.Join(a[:150], " ") != strings.Join(b[:150], " ") {
		t.Error("the head was re-encoded")
	}
}

// The example edit si ships: checks clean and builds.
func TestExamples(t *testing.T) {
	chrome, ff, fp := needTools(t)
	md := filepath.Join("..", "..", "examples", "video", "edit.md")
	e, err := ParseFile(md)
	if err != nil {
		t.Fatal(err)
	}
	if note := e.Check(fp); note != "" || len(e.Problems) > 0 {
		t.Fatal(note, e.Problems)
	}
	e, _ = ParseFile(md)
	out := t.TempDir()
	res, err := Build(e, Options{Chrome: chrome, FFmpeg: ff, FFprobe: fp, OutDir: out})
	if err != nil {
		t.Fatal(err)
	}
	if d := duration(t, fp, res.Out); d < 21 || d > 23 || len(res.Files) != 2 {
		t.Errorf("%s: %g s, %v", res.Out, d, res.Files)
	}
}
