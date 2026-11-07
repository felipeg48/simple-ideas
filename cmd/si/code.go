package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/felipeg48/simple-ideas/internal/codeshow"
	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/felipeg48/simple-ideas/internal/render"
)

// si code exports the code windows of a file: every show (a fence with
// window= or id=; same-id fences, on one page or across slides, are one
// show) as PNG and SVG per step and as an animated GIF. Headless Chrome
// loads a page with only the windows at #si-code-export, where codewin.js
// writes every frame as SVG (the window on its backdrop, callouts, fonts
// inside); Chrome then rasterises the frames and Go encodes the GIF.

type codeFrame struct {
	Step int    `json:"step"`
	Hold bool   `json:"hold"`
	SVG  string `json:"svg"`
	W    int    `json:"w"`
	H    int    `json:"h"`
}

type codeExport struct {
	ID     string      `json:"id"`
	Title  string      `json:"title"`
	Steps  int         `json:"steps"`
	Frames []codeFrame `json:"frames"`
}

var codeExportRE = regexp.MustCompile(`(?s)<pre id="si-code-export"[^>]*>(.*?)</pre>`)

func runCode(path string, o *options, stdout, stderr io.Writer) int {
	if !o.png && !o.svg && !o.gif && !o.mp4 {
		return usageErr(stderr, "code needs --png, --svg, --gif or --mp4")
	}
	if o.mp4 {
		if _, err := exec.LookPath("ffmpeg"); err != nil {
			fmt.Fprintf(stderr, "si: %s\n", ffmpegHint)
			return exitProblems
		}
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	d0, err := deck.Parse(path)
	if err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	m := d0.Meta
	src, shows := codeshow.Source(string(raw), map[string]string{"theme": m.Theme, "fonts": m.Fonts, "accent": m.Accent, "mode": m.Mode}, o.showID)
	if shows == 0 {
		fmt.Fprintf(stderr, "si: %s has no code window%s (a fence with window= or id=)\n", path, map[bool]string{true: " " + o.showID, false: ""}[o.showID != ""])
		return exitProblems
	}
	chrome := findChrome()
	if chrome == "" {
		fmt.Fprintln(stderr, "si: code exports need Google Chrome or Chromium (headless)")
		return exitProblems
	}
	d, err := deck.ParseSource(path, src, deck.Meta{Type: "article"})
	if err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	res := render.Document(d, render.Options{Media: render.MediaInline})
	tmp, err := os.MkdirTemp("", "si-code-")
	if err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	defer os.RemoveAll(tmp)
	page := filepath.Join(tmp, "windows.html")
	if err := os.WriteFile(page, res.HTML, 0o644); err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	exports, err := codeFrames(chrome, page)
	if err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	out := o.output
	if out == "" {
		out = filepath.Dir(path)
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	for _, ex := range exports {
		base := name + "-" + ex.ID
		var frames []image.Image
		var delays []int
		var video []videoFrame
		for i, f := range ex.Frames {
			if f.Hold && o.svg {
				p := filepath.Join(out, fmt.Sprintf("%s-%d.svg", base, f.Step))
				if err := os.WriteFile(p, []byte(f.SVG), 0o644); err != nil {
					fmt.Fprintf(stderr, "si: %v\n", err)
					return exitProblems
				}
				fmt.Fprintf(stdout, "  %-32s (svg, %d × %d)\n", filepath.Base(p), f.W, f.H)
			}
			if f.Hold && o.png {
				p, _ := filepath.Abs(filepath.Join(out, fmt.Sprintf("%s-%d.png", base, f.Step)))
				if err := svgToPNG(chrome, tmp, f, o.scale, p); err != nil {
					fmt.Fprintf(stderr, "si: png %s: %v\n", base, err)
					return exitProblems
				}
				fmt.Fprintf(stdout, "  %-32s (png, %d × %d)\n", filepath.Base(p), int(float64(f.W)*o.scale), int(float64(f.H)*o.scale))
			}
			if o.mp4 {
				p := filepath.Join(tmp, fmt.Sprintf("v%03d.png", i))
				if err := svgToPNG(chrome, tmp, f, o.scale, p); err != nil {
					fmt.Fprintf(stderr, "si: mp4 frame: %v\n", err)
					return exitProblems
				}
				video = append(video, videoFrame{PNG: p, Dur: map[bool]float64{true: 2.2, false: 0.06}[f.Hold]})
			}
			if o.gif {
				p := filepath.Join(tmp, fmt.Sprintf("f%d.png", i))
				if err := svgToPNG(chrome, tmp, f, 1, p); err != nil {
					fmt.Fprintf(stderr, "si: gif frame: %v\n", err)
					return exitProblems
				}
				img, err := readPNG(p)
				if err != nil {
					fmt.Fprintf(stderr, "si: %v\n", err)
					return exitProblems
				}
				frames = append(frames, img)
				delays = append(delays, map[bool]int{true: 220, false: 6}[f.Hold])
			}
		}
		if o.mp4 {
			p := filepath.Join(out, base+".mp4")
			if err := encodeMP4(video, p); err != nil {
				fmt.Fprintf(stderr, "si: mp4: %v\n", err)
				return exitProblems
			}
			st, _ := os.Stat(p)
			fmt.Fprintf(stdout, "  %-32s (mp4, %d steps, %d frames, %d KB)\n", base+".mp4", ex.Steps, len(video), st.Size()/1024)
		}
		if o.gif {
			data, err := encodeGIF(frames, delays)
			if err != nil {
				fmt.Fprintf(stderr, "si: gif: %v\n", err)
				return exitProblems
			}
			p := filepath.Join(out, base+".gif")
			if err := os.WriteFile(p, data, 0o644); err != nil {
				fmt.Fprintf(stderr, "si: %v\n", err)
				return exitProblems
			}
			fmt.Fprintf(stdout, "  %-32s (gif, %d steps, %d frames, %d KB)\n", base+".gif", ex.Steps, len(frames), len(data)/1024)
		}
	}
	return exitOK
}

// codeFrames has headless Chrome load the windows page at #si-code-export.
func codeFrames(chrome, page string) ([]codeExport, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, chrome, "--headless=new", "--disable-gpu", "--virtual-time-budget=5000",
		"--window-size=1800,1200", "--dump-dom", "file://"+filepath.ToSlash(page)+"#si-code-export").Output()
	if err != nil {
		return nil, fmt.Errorf("chrome: %v", err)
	}
	m := codeExportRE.FindSubmatch(out)
	if m == nil {
		return nil, fmt.Errorf("the page did not export its windows")
	}
	var ex []codeExport
	err = json.Unmarshal([]byte(html.UnescapeString(string(m[1]))), &ex)
	return ex, err
}

func svgToPNG(chrome, tmp string, f codeFrame, scale float64, dest string) error {
	page := filepath.Join(tmp, "frame.html")
	doc := `<!doctype html><meta charset="utf-8"><style>html,body{margin:0;background:transparent}svg{display:block}</style>` + f.SVG
	if err := os.WriteFile(page, []byte(doc), 0o644); err != nil {
		return err
	}
	return chromeRun(chrome, "--screenshot="+dest, fmt.Sprintf("--window-size=%d,%d", f.W, f.H),
		fmt.Sprintf("--force-device-scale-factor=%g", scale), "--hide-scrollbars", "--default-background-color=00000000", "file://"+filepath.ToSlash(page))
}

func readPNG(p string) (image.Image, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return png.Decode(f)
}

// encodeGIF quantises the frames to one shared palette (the 256 most used
// colours, 4 bits a channel) without dithering: dithered gradients made a
// four-step show 5.8 MB, plain ones 0.9 MB.
// ponytail: a popularity palette, not median cut; good for windows on a
// gradient, upgrade if photos ever land in code windows.
func encodeGIF(frames []image.Image, delays []int) ([]byte, error) {
	if len(frames) == 0 {
		return nil, fmt.Errorf("no frames")
	}
	b := frames[0].Bounds()
	count := map[[3]uint8]int{}
	for _, f := range frames {
		fb := f.Bounds()
		for y := fb.Min.Y; y < fb.Max.Y; y += 2 {
			for x := fb.Min.X; x < fb.Max.X; x += 2 {
				r, g, bl, _ := f.At(x, y).RGBA()
				count[[3]uint8{uint8(r>>12) << 4, uint8(g>>12) << 4, uint8(bl>>12) << 4}]++
			}
		}
	}
	keys := make([][3]uint8, 0, len(count))
	for k := range count {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return count[keys[i]] > count[keys[j]] })
	if len(keys) > 256 {
		keys = keys[:256]
	}
	pal := make(color.Palette, len(keys))
	for i, k := range keys {
		pal[i] = color.RGBA{k[0] | k[0]>>4, k[1] | k[1]>>4, k[2] | k[2]>>4, 255}
	}
	g := &gif.GIF{LoopCount: 0}
	for i, f := range frames {
		p := image.NewPaletted(b, pal)
		draw.Draw(p, b, f, f.Bounds().Min, draw.Src)
		g.Image = append(g.Image, p)
		g.Delay = append(g.Delay, delays[i])
	}
	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, g); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
