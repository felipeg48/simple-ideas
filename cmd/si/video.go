package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
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
