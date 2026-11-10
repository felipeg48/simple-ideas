package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/felipeg48/simple-ideas/internal/cdp"
)

// Real-motion video: headless Chrome driven over the DevTools pipe
// (--remote-debugging-pipe: requests on fd 3, replies on fd 4, one JSON value
// per message, NUL-terminated), so the export films what a viewer sees move —
// slide transitions, canvas morphs, code glides, 3D fly-ins — not a dissolve
// between stills. The page runs slowed down (CSS and Web Animations at
// motionRate through the Animation domain; performance.now, Date.now,
// requestAnimationFrame and timers through a script injected before the
// page's own), screenshots are taken as fast as Chrome gives them, and each is
// stamped with its time on the page's slowed clock; the 30 fps video then
// takes, for every frame, the last screenshot at or before it.

const motionRate = 0.25 // page time per real second while filming

// motionShim slows the page's JavaScript clock to motionRate, the same rate
// the Animation domain gives CSS, so JS tweens (the 3D map's camera) and CSS
// transitions stay in step.
const motionShim = `(function(){var r=%g;var pn=performance.now.bind(performance),t0=pn();performance.now=function(){return t0+(pn()-t0)*r};
var dn=Date.now,d0=dn();Date.now=function(){return d0+(dn()-d0)*r};
var raf=window.requestAnimationFrame.bind(window);window.requestAnimationFrame=function(cb){return raf(function(){cb(performance.now())})};
var st=window.setTimeout,si=window.setInterval;window.setTimeout=function(f,ms){var a=[].slice.call(arguments,2);return st.apply(window,[f,(ms||0)/r].concat(a))};
window.setInterval=function(f,ms){var a=[].slice.call(arguments,2);return si.apply(window,[f,(ms||0)/r].concat(a))};})();`

// slowOpen prepares the page clock (motionRate), loads url and waits for it
// to settle (settle seconds of page time).
func slowOpen(c *cdp.Client, url string, settle float64) error {
	for _, step := range []struct {
		m string
		p any
	}{
		{"Page.enable", map[string]any{}},
		{"Runtime.enable", map[string]any{}},
		{"Animation.enable", map[string]any{}},
		{"Page.addScriptToEvaluateOnNewDocument", map[string]any{"source": fmt.Sprintf(motionShim, motionRate)}},
	} {
		if err := c.Call(step.m, step.p, nil); err != nil {
			return err
		}
	}
	if err := c.Call("Page.navigate", map[string]any{"url": url}, nil); err != nil {
		return err
	}
	if err := c.Call("Animation.setPlaybackRate", map[string]any{"playbackRate": motionRate}, nil); err != nil {
		return err
	}
	for i := 0; i < 200; i++ {
		v, _ := c.Eval("document.readyState")
		if string(v) == `"complete"` {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	time.Sleep(time.Duration(settle / motionRate * float64(time.Second)))
	return nil
}

type stamped struct {
	png string
	at  float64 // page seconds since the start of the clip
}

// film screenshots the page for d seconds of page time, after run (a key
// press, a click) starts the motion; files go to dir as tag-NNNN.png.
func film(c *cdp.Client, d float64, dir, tag string, run func() error) ([]stamped, error) {
	start := time.Now()
	if run != nil {
		if err := run(); err != nil {
			return nil, err
		}
	}
	var out []stamped
	for i := 0; ; i++ {
		at := time.Since(start).Seconds() * motionRate
		raw, err := c.Screenshot()
		if err != nil {
			return nil, err
		}
		p := filepath.Join(dir, fmt.Sprintf("%s-%04d.png", tag, i))
		if err := os.WriteFile(p, raw, 0o644); err != nil {
			return nil, err
		}
		out = append(out, stamped{p, at})
		if at >= d {
			return out, nil
		}
	}
}

// on30 turns stamped screenshots into 30 fps frames: frame k shows the last
// screenshot taken at or before k/30 s.
func on30(shots []stamped, d float64) []videoFrame {
	n := int(d*videoFPS + .5)
	var out []videoFrame
	j := 0
	for k := 0; k < n; k++ {
		t := float64(k) / videoFPS
		for j+1 < len(shots) && shots[j+1].at <= t {
			j++
		}
		if len(shots) > 0 {
			out = append(out, videoFrame{PNG: shots[j].png, Dur: 1.0 / videoFPS})
		}
	}
	return out
}

// exportDeckMotion writes <name>.mp4 with the deck's real motion: each view
// held for stepHold, then → pressed and the transition (slide, build step,
// canvas morph) filmed as it plays.
func exportDeckMotion(chrome, htmlPath, out string, d *deckInfo, dwell float64, nar *narration) (views int, err error) {
	tmp, err := os.MkdirTemp("", "si-deckmotion-")
	if err != nil {
		return 0, err
	}
	defer os.RemoveAll(tmp)
	c, err := cdp.Start(chrome, 1920, 1080)
	if err != nil {
		return 0, err
	}
	defer c.Close()
	abs, _ := filepath.Abs(htmlPath)
	if err := slowOpen(c, "file://"+filepath.ToSlash(abs)+"#1", 1.5); err != nil {
		return 0, err
	}
	var video []videoFrame
	const move = 1.3 // page seconds filmed after each →: the longest transition and morph, with room
	for i, n := range d.steps {
		for k := 0; k <= n; k++ {
			views++
			held, err := film(c, 0, tmp, fmt.Sprintf("h%03d-%02d", i, k), nil)
			if err != nil {
				return views, err
			}
			dur := stepHold(d.minutes(i), n+1, dwell)
			if h, ok := nar.hold(i, k, move, views == 1); ok {
				dur = h
			}
			video = append(video, videoFrame{PNG: held[len(held)-1].png, Dur: dur})
			if i == len(d.steps)-1 && k == n {
				break
			}
			span := move
			if k == n && d.morph != nil { // arriving at a morph slide: film the whole glide
				span = max(span, d.morph(i+1)+0.4)
			}
			shots, err := film(c, span, tmp, fmt.Sprintf("m%03d-%02d", i, k), func() error { return c.Key("ArrowRight", 39) })
			if err != nil {
				return views, err
			}
			video = append(video, on30(shots, span)...)
		}
	}
	return views, encodeMP4(video, filepath.Join(out, d.name+".mp4"))
}

type deckInfo struct {
	name    string
	steps   []int
	minutes func(i int) float64
	morph   func(i int) float64 // seconds slide i's morph takes to arrive, 0 if it does not morph
}

// morphSeconds is a morph length (`transition: morph 0.8s`, 600ms) in
// seconds; the default token (.7s) when none is given.
func morphSeconds(v string) float64 {
	if ms, ok := strings.CutSuffix(v, "ms"); ok {
		f, _ := strconv.ParseFloat(ms, 64)
		return f / 1000
	}
	if f, err := strconv.ParseFloat(strings.TrimSuffix(v, "s"), 64); err == nil {
		return f
	}
	return 0.7
}

// exportMindmap3DMotion films a 3D map's Play as it moves: each view held,
// then the camera's flight to the next branch.
func exportMindmap3DMotion(chrome, htmlPath, out, name string) (views int, err error) {
	tmp, err := os.MkdirTemp("", "si-map3dmotion-")
	if err != nil {
		return 0, err
	}
	defer os.RemoveAll(tmp)
	c, err := cdp.Start(chrome, 1600, 1000, "--use-gl=angle", "--use-angle=swiftshader", "--enable-unsafe-swiftshader")
	if err != nil {
		return 0, err
	}
	defer c.Close()
	abs, _ := filepath.Abs(htmlPath)
	if err := slowOpen(c, "file://"+filepath.ToSlash(abs)+"#si-film3d", 2.5); err != nil {
		return 0, err
	}
	const ready = `(function(){var el=document.querySelector('.si-mindmap[data-full]'),m=el&&el.siMindmap,d=m&&m.is3d&&m.d3;if(!d)return -1;
el.querySelectorAll('.si-mm-tools').forEach(function(t){t.style.display='none'});if(!d.tour)d.tour={k:0};d.tourGo(0);return d.steps().length})()`
	var n int
	for try := 0; try < 40 && n <= 0; try++ {
		v, err := c.Eval(ready)
		if err != nil {
			return 0, err
		}
		json.Unmarshal(v, &n)
		if n <= 0 {
			time.Sleep(250 * time.Millisecond)
		}
	}
	if n <= 0 {
		return 0, fmt.Errorf("the page did not start its 3D Play (no WebGL?)")
	}
	time.Sleep(time.Duration(1.5 / motionRate * float64(time.Second)))
	var video []videoFrame
	const fly = 1.6 // page seconds of a camera flight
	for k := 0; k < n; k++ {
		held, err := film(c, 0, tmp, fmt.Sprintf("h%02d", k), nil)
		if err != nil {
			return k, err
		}
		video = append(video, videoFrame{PNG: held[len(held)-1].png, Dur: 2.5})
		if k == n-1 {
			break
		}
		next := fmt.Sprintf(`document.querySelector('.si-mindmap[data-full]').siMindmap.d3.tourGo(%d)`, k+1)
		shots, err := film(c, fly, tmp, fmt.Sprintf("f%02d", k), func() error { _, err := c.Eval(next); return err })
		if err != nil {
			return k, err
		}
		video = append(video, on30(shots, fly)...)
	}
	return n, encodeMP4(video, filepath.Join(out, name+".mp4"))
}
