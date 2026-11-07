package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

// MP4 exports (si code --mp4, si build map.md --mp4): headless Chrome renders
// the frames as PNG, ffmpeg (an optional external tool, like Chrome) encodes
// them: H.264, yuv420p, 30 fps, faststart, so every player and social site
// takes the file.

type videoFrame struct {
	PNG string  // absolute path
	Dur float64 // seconds on screen
}

const ffmpegHint = "MP4 export needs ffmpeg on the PATH (macOS: brew install ffmpeg; Debian/Ubuntu: apt install ffmpeg)"

const videoFPS = 30

// frameRun lays the frames out as a 30 fps image sequence (seq00000.png …),
// each frame repeated round(duration × 30) times as hard links (or copies), so
// every duration is exact; ffmpeg's concat durations drift on the last frame.
func frameRun(frames []videoFrame, dir string) (pattern string, n int, err error) {
	for _, f := range frames {
		reps := int(f.Dur*videoFPS + 0.5)
		if reps < 1 {
			reps = 1
		}
		for r := 0; r < reps; r++ {
			dst := filepath.Join(dir, fmt.Sprintf("seq%05d.png", n))
			if err := os.Link(f.PNG, dst); err != nil {
				raw, rerr := os.ReadFile(f.PNG)
				if rerr != nil {
					return "", 0, rerr
				}
				if err := os.WriteFile(dst, raw, 0o644); err != nil {
					return "", 0, err
				}
			}
			n++
		}
	}
	return filepath.Join(dir, "seq%05d.png"), n, nil
}

// ffmpegArgs encode an image sequence into an MP4 that plays everywhere
// (H.264, even sizes for yuv420p, faststart for the web).
func ffmpegArgs(pattern, out string) []string {
	return []string{"-y", "-hide_banner", "-loglevel", "error", "-framerate", fmt.Sprint(videoFPS), "-i", pattern,
		"-vf", "scale=trunc(iw/2)*2:trunc(ih/2)*2,format=yuv420p",
		"-c:v", "libx264", "-preset", "medium", "-crf", "20", "-r", fmt.Sprint(videoFPS), "-movflags", "+faststart", out}
}

func encodeMP4(frames []videoFrame, out string) error {
	ff, err := exec.LookPath("ffmpeg")
	if err != nil {
		return fmt.Errorf("%s", ffmpegHint)
	}
	if len(frames) == 0 {
		return fmt.Errorf("no frames")
	}
	dir, err := os.MkdirTemp("", "si-seq-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	pattern, _, err := frameRun(frames, dir)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	abs, _ := filepath.Abs(out)
	if msg, err := exec.CommandContext(ctx, ff, ffmpegArgs(pattern, abs)...).CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg: %v %s", err, strings.TrimSpace(string(msg)))
	}
	return nil
}

// mapPlayFrame is one frame of a mind map's Play, as the runtime exports it
// (#si-export-play): the overview, then each branch, with tweened frames between.
type mapPlayFrame struct {
	SVG  string `json:"svg"`
	W    int    `json:"w"`
	H    int    `json:"h"`
	Hold bool   `json:"hold"`
}

var playExportRE = regexp.MustCompile(`(?s)<pre id="si-export-play"[^>]*>(.*?)</pre>`)

// exportMindmapMP4 writes <name>.mp4: the page's first map played branch by
// branch (2D, like the other exports).
func exportMindmapMP4(chrome, htmlPath, out, name string, scale float64, stdout io.Writer) error {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return fmt.Errorf("%s", ffmpegHint)
	}
	abs, err := filepath.Abs(htmlPath)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	dom, err := exec.CommandContext(ctx, chrome, "--headless=new", "--disable-gpu", "--virtual-time-budget=6000",
		"--window-size=1600,1000", "--dump-dom", "file://"+filepath.ToSlash(abs)+"#si-export-play").Output()
	if err != nil {
		return fmt.Errorf("chrome: %v", err)
	}
	m := playExportRE.FindSubmatch(dom)
	if m == nil {
		return fmt.Errorf("the page did not export its Play")
	}
	var frames []mapPlayFrame
	if err := json.Unmarshal([]byte(html.UnescapeString(string(m[1]))), &frames); err != nil {
		return err
	}
	tmp, err := os.MkdirTemp("", "si-play-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	var video []videoFrame
	for i, f := range frames {
		page := filepath.Join(tmp, "frame.html")
		doc := `<!doctype html><meta charset="utf-8"><style>html,body{margin:0;background:#fff}svg{display:block}</style>` + f.SVG
		if err := os.WriteFile(page, []byte(doc), 0o644); err != nil {
			return err
		}
		p := filepath.Join(tmp, fmt.Sprintf("f%03d.png", i))
		if err := chromeRun(chrome, "--screenshot="+p, fmt.Sprintf("--window-size=%d,%d", f.W, f.H),
			fmt.Sprintf("--force-device-scale-factor=%g", scale), "--hide-scrollbars", "file://"+filepath.ToSlash(page)); err != nil {
			return fmt.Errorf("frame %d: %v", i, err)
		}
		dur := 1.0 / 30 * 2
		if f.Hold {
			dur = 2.5
		}
		video = append(video, videoFrame{PNG: p, Dur: dur})
	}
	dest := filepath.Join(out, name+".mp4")
	if err := encodeMP4(video, dest); err != nil {
		return err
	}
	st, _ := os.Stat(dest)
	var size int64
	if st != nil {
		size = st.Size()
	}
	fmt.Fprintf(stdout, "  %-28s (mp4, %d frames, %d KB)\n", name+".mp4", len(video), size/1024)
	return nil
}

// crossfade adds n frames (each one frame long at 30 fps) blending a into b,
// so a cut between two screenshots becomes a short dissolve.
func crossfade(a, b, dir string, n int, tag string) ([]videoFrame, error) {
	ia, err := readPNG(a)
	if err != nil {
		return nil, err
	}
	ib, err := readPNG(b)
	if err != nil {
		return nil, err
	}
	r := ia.Bounds()
	if ib.Bounds() != r {
		return nil, nil // sizes differ: a cut
	}
	enc := png.Encoder{CompressionLevel: png.BestSpeed}
	var out []videoFrame
	for i := 1; i < n; i++ {
		dst := image.NewRGBA(r)
		draw.Draw(dst, r, ia, r.Min, draw.Src)
		draw.DrawMask(dst, r, ib, r.Min, image.NewUniform(color.Alpha{uint8(255 * i / n)}), image.Point{}, draw.Over)
		p := filepath.Join(dir, fmt.Sprintf("%s-x%02d.png", tag, i))
		f, err := os.Create(p)
		if err != nil {
			return nil, err
		}
		err = enc.Encode(f, dst)
		f.Close()
		if err != nil {
			return nil, err
		}
		out = append(out, videoFrame{PNG: p, Dur: 1.0 / videoFPS})
	}
	return out, nil
}

// withFades holds each shot for its time and dissolves into the next one.
func withFades(shots []videoFrame, dir string) ([]videoFrame, error) {
	var out []videoFrame
	for i, s := range shots {
		out = append(out, s)
		if i+1 < len(shots) {
			x, err := crossfade(s.PNG, shots[i+1].PNG, dir, 12, fmt.Sprintf("s%03d", i))
			if err != nil {
				return nil, err
			}
			out = append(out, x...)
		}
	}
	return out, nil
}

var (
	slideSplitRE = regexp.MustCompile(`<section class="si-slide`)
	dataStepRE   = regexp.MustCompile(`data-step="(\d+)"`)
)

// slideSteps is the number of build steps of every slide of a built deck.
func slideSteps(page string) []int {
	parts := slideSplitRE.Split(page, -1)
	var steps []int
	for _, p := range parts[1:] {
		n := 0
		for _, m := range dataStepRE.FindAllStringSubmatch(p, -1) {
			var k int
			fmt.Sscan(m[1], &k)
			n = max(n, k)
		}
		steps = append(steps, n)
	}
	return steps
}

// stepHold is how long one view of a slide stays: dwell when given, else the
// slide's (N min) shared by its views, else 3 s.
func stepHold(minutes float64, views int, dwell float64) float64 {
	switch {
	case dwell > 0:
		return dwell
	case minutes > 0:
		return max(1, minutes*60/float64(views))
	}
	return 3
}

// exportDeckMP4 writes <name>.mp4: every slide and build step as the audience
// sees it (1920×1080), each held for stepHold, dissolving into the next.
func exportDeckMP4(chrome, htmlPath, out string, d *deck.Deck, dwell float64, stdout io.Writer) error {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return fmt.Errorf("%s", ffmpegHint)
	}
	page, err := os.ReadFile(htmlPath)
	if err != nil {
		return err
	}
	abs, _ := filepath.Abs(htmlPath)
	steps := slideSteps(string(page))
	tmp, err := os.MkdirTemp("", "si-deckvid-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	var shots []videoFrame
	for i, n := range steps {
		minutes := 0.0
		if i < len(d.Slides) {
			minutes = d.Slides[i].Minutes
		}
		for k := 0; k <= n; k++ {
			p := filepath.Join(tmp, fmt.Sprintf("shot%03d-%02d.png", i, k))
			if err := chromeRun(chrome, "--screenshot="+p, "--window-size=1920,1080", "--hide-scrollbars", "--virtual-time-budget=2500",
				fmt.Sprintf("file://%s#%d.%d", filepath.ToSlash(abs), i+1, k)); err != nil {
				return fmt.Errorf("slide %d.%d: %v", i+1, k, err)
			}
			shots = append(shots, videoFrame{PNG: p, Dur: stepHold(minutes, n+1, dwell)})
		}
	}
	video, err := withFades(shots, tmp)
	if err != nil {
		return err
	}
	dest := filepath.Join(out, d.Name+".mp4")
	if err := encodeMP4(video, dest); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "  %-28s (mp4, %d views, %d KB)\n", d.Name+".mp4", len(shots), fileKB(dest))
	return nil
}

func fileKB(p string) int64 {
	if st, err := os.Stat(p); err == nil {
		return st.Size() / 1024
	}
	return 0
}

var play3DRE = regexp.MustCompile(`<pre[^>]*id="si-play3d"[^>]*>(\d+)</pre>`)

// exportMindmap3DMP4 writes <name>.mp4 for a map that starts in 3D: its Play
// (the overview, then each branch with the rest dimmed), each view rendered by
// Chrome's software WebGL, held 2.5 s and dissolving into the next.
func exportMindmap3DMP4(chrome, htmlPath, out, name string, stdout io.Writer) error {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return fmt.Errorf("%s", ffmpegHint)
	}
	abs, _ := filepath.Abs(htmlPath)
	gl := []string{"--headless=new", "--use-gl=angle", "--use-angle=swiftshader", "--enable-unsafe-swiftshader", "--hide-scrollbars", "--window-size=1600,1000"}
	url := func(k int) string { return fmt.Sprintf("file://%s#si-play3d-%d", filepath.ToSlash(abs), k) }
	run := func(args ...string) ([]byte, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		return exec.CommandContext(ctx, chrome, append(append([]string{}, gl...), args...)...).CombinedOutput()
	}
	dom, err := run("--virtual-time-budget=8000", "--dump-dom", url(0))
	if err != nil {
		return fmt.Errorf("chrome: %v", err)
	}
	m := play3DRE.FindSubmatch(dom)
	if m == nil {
		return fmt.Errorf("the page did not start its 3D Play (no WebGL?)")
	}
	var n int
	fmt.Sscan(string(m[1]), &n)
	tmp, err := os.MkdirTemp("", "si-play3d-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	var shots []videoFrame
	for k := 0; k < n; k++ {
		p := filepath.Join(tmp, fmt.Sprintf("v%02d.png", k))
		if msg, err := run("--virtual-time-budget=8000", "--screenshot="+p, url(k)); err != nil {
			return fmt.Errorf("view %d: %v %s", k, err, strings.TrimSpace(string(msg)))
		}
		shots = append(shots, videoFrame{PNG: p, Dur: 2.5})
	}
	video, err := withFades(shots, tmp)
	if err != nil {
		return err
	}
	dest := filepath.Join(out, name+".mp4")
	if err := encodeMP4(video, dest); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "  %-28s (mp4, 3D Play, %d views, %d KB)\n", name+".mp4", n, fileKB(dest))
	return nil
}
