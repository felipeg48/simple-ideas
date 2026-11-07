package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
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

type cdp struct {
	cmd     *exec.Cmd
	w       *os.File
	r       *bufio.Reader
	mu      sync.Mutex
	id      int
	session string
}

// startCDP launches headless Chrome with the DevTools pipe and attaches to a
// blank page of w×h CSS pixels.
func startCDP(chrome string, w, h int, extra ...string) (*cdp, error) {
	toChrome, cw, err := os.Pipe() // Chrome reads its fd 3
	if err != nil {
		return nil, err
	}
	cr, fromChrome, err := os.Pipe() // Chrome writes its fd 4
	if err != nil {
		return nil, err
	}
	args := append([]string{"--headless=new", "--remote-debugging-pipe", "--hide-scrollbars", "--no-first-run", "--no-default-browser-check",
		fmt.Sprintf("--window-size=%d,%d", w, h)}, extra...)
	cmd := exec.Command(chrome, append(args, "about:blank")...)
	cmd.ExtraFiles = []*os.File{toChrome, fromChrome}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	toChrome.Close()
	fromChrome.Close()
	c := &cdp{cmd: cmd, w: cw, r: bufio.NewReaderSize(cr, 1<<20)}
	var tgt struct{ TargetID string }
	if err := c.call("Target.createTarget", map[string]any{"url": "about:blank"}, &tgt); err != nil {
		c.close()
		return nil, err
	}
	var att struct{ SessionID string }
	if err := c.call("Target.attachToTarget", map[string]any{"targetId": tgt.TargetID, "flatten": true}, &att); err != nil {
		c.close()
		return nil, err
	}
	c.session = att.SessionID
	if err := c.call("Emulation.setDeviceMetricsOverride", map[string]any{"width": w, "height": h, "deviceScaleFactor": 1, "mobile": false}, nil); err != nil {
		c.close()
		return nil, err
	}
	return c, nil
}

func (c *cdp) close() {
	c.w.Close()
	c.cmd.Process.Kill()
	c.cmd.Wait()
}

// call sends one command (to the page session once attached) and waits for
// its reply, skipping events.
func (c *cdp) call(method string, params any, out any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.id++
	msg := map[string]any{"id": c.id, "method": method, "params": params}
	if c.session != "" {
		msg["sessionId"] = c.session
	}
	b, _ := json.Marshal(msg)
	if _, err := c.w.Write(append(b, 0)); err != nil {
		return err
	}
	for {
		raw, err := c.r.ReadBytes(0)
		if err != nil {
			return fmt.Errorf("chrome: %v", err)
		}
		var rep struct {
			ID     int             `json:"id"`
			Result json.RawMessage `json:"result"`
			Error  *struct{ Message string }
		}
		if json.Unmarshal(raw[:len(raw)-1], &rep) != nil || rep.ID != c.id {
			continue // an event, or a reply to someone else
		}
		if rep.Error != nil {
			return fmt.Errorf("%s: %s", method, rep.Error.Message)
		}
		if out != nil {
			return json.Unmarshal(rep.Result, out)
		}
		return nil
	}
}

// eval runs JavaScript in the page and returns its value as JSON.
func (c *cdp) eval(js string) (json.RawMessage, error) {
	var r struct {
		Result struct {
			Value json.RawMessage `json:"value"`
		} `json:"result"`
	}
	err := c.call("Runtime.evaluate", map[string]any{"expression": js, "returnByValue": true, "awaitPromise": true}, &r)
	return r.Result.Value, err
}

// slowOpen prepares the page clock (motionRate), loads url and waits for it
// to settle (settle seconds of page time).
func (c *cdp) slowOpen(url string, settle float64) error {
	for _, step := range []struct {
		m string
		p any
	}{
		{"Page.enable", map[string]any{}},
		{"Runtime.enable", map[string]any{}},
		{"Animation.enable", map[string]any{}},
		{"Page.addScriptToEvaluateOnNewDocument", map[string]any{"source": fmt.Sprintf(motionShim, motionRate)}},
	} {
		if err := c.call(step.m, step.p, nil); err != nil {
			return err
		}
	}
	if err := c.call("Page.navigate", map[string]any{"url": url}, nil); err != nil {
		return err
	}
	if err := c.call("Animation.setPlaybackRate", map[string]any{"playbackRate": motionRate}, nil); err != nil {
		return err
	}
	for i := 0; i < 200; i++ {
		v, _ := c.eval("document.readyState")
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
func (c *cdp) film(d float64, dir, tag string, run func() error) ([]stamped, error) {
	start := time.Now()
	if run != nil {
		if err := run(); err != nil {
			return nil, err
		}
	}
	var out []stamped
	for i := 0; ; i++ {
		at := time.Since(start).Seconds() * motionRate
		var shot struct{ Data string }
		if err := c.call("Page.captureScreenshot", map[string]any{"format": "png"}, &shot); err != nil {
			return nil, err
		}
		raw, err := base64.StdEncoding.DecodeString(shot.Data)
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

// key presses a key in the page (deck.js and code windows listen for keys).
func (c *cdp) key(name string, code int) error {
	for _, t := range []string{"keyDown", "keyUp"} {
		if err := c.call("Input.dispatchKeyEvent", map[string]any{"type": t, "key": name, "code": name, "windowsVirtualKeyCode": code, "nativeVirtualKeyCode": code}, nil); err != nil {
			return err
		}
	}
	return nil
}

// exportDeckMotion writes <name>.mp4 with the deck's real motion: each view
// held for stepHold, then → pressed and the transition (slide, build step,
// canvas morph) filmed as it plays.
func exportDeckMotion(chrome, htmlPath, out string, d *deckInfo, dwell float64) (views int, err error) {
	tmp, err := os.MkdirTemp("", "si-deckmotion-")
	if err != nil {
		return 0, err
	}
	defer os.RemoveAll(tmp)
	c, err := startCDP(chrome, 1920, 1080)
	if err != nil {
		return 0, err
	}
	defer c.close()
	abs, _ := filepath.Abs(htmlPath)
	if err := c.slowOpen("file://"+filepath.ToSlash(abs)+"#1", 1.5); err != nil {
		return 0, err
	}
	var video []videoFrame
	const move = 1.3 // page seconds filmed after each →: the longest transition and morph, with room
	for i, n := range d.steps {
		for k := 0; k <= n; k++ {
			views++
			held, err := c.film(0, tmp, fmt.Sprintf("h%03d-%02d", i, k), nil)
			if err != nil {
				return views, err
			}
			video = append(video, videoFrame{PNG: held[len(held)-1].png, Dur: stepHold(d.minutes(i), n+1, dwell)})
			if i == len(d.steps)-1 && k == n {
				break
			}
			shots, err := c.film(move, tmp, fmt.Sprintf("m%03d-%02d", i, k), func() error { return c.key("ArrowRight", 39) })
			if err != nil {
				return views, err
			}
			video = append(video, on30(shots, move)...)
		}
	}
	return views, encodeMP4(video, filepath.Join(out, d.name+".mp4"))
}

type deckInfo struct {
	name    string
	steps   []int
	minutes func(i int) float64
}

// exportMindmap3DMotion films a 3D map's Play as it moves: each view held,
// then the camera's flight to the next branch.
func exportMindmap3DMotion(chrome, htmlPath, out, name string) (views int, err error) {
	tmp, err := os.MkdirTemp("", "si-map3dmotion-")
	if err != nil {
		return 0, err
	}
	defer os.RemoveAll(tmp)
	c, err := startCDP(chrome, 1600, 1000, "--use-gl=angle", "--use-angle=swiftshader", "--enable-unsafe-swiftshader")
	if err != nil {
		return 0, err
	}
	defer c.close()
	abs, _ := filepath.Abs(htmlPath)
	if err := c.slowOpen("file://"+filepath.ToSlash(abs)+"#si-film3d", 2.5); err != nil {
		return 0, err
	}
	const ready = `(function(){var el=document.querySelector('.si-mindmap[data-full]'),m=el&&el.siMindmap,d=m&&m.is3d&&m.d3;if(!d)return -1;
el.querySelectorAll('.si-mm-tools').forEach(function(t){t.style.display='none'});if(!d.tour)d.tour={k:0};d.tourGo(0);return d.steps().length})()`
	var n int
	for try := 0; try < 40 && n <= 0; try++ {
		v, err := c.eval(ready)
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
		held, err := c.film(0, tmp, fmt.Sprintf("h%02d", k), nil)
		if err != nil {
			return k, err
		}
		video = append(video, videoFrame{PNG: held[len(held)-1].png, Dur: 2.5})
		if k == n-1 {
			break
		}
		next := fmt.Sprintf(`document.querySelector('.si-mindmap[data-full]').siMindmap.d3.tourGo(%d)`, k+1)
		shots, err := c.film(fly, tmp, fmt.Sprintf("f%02d", k), func() error { _, err := c.eval(next); return err })
		if err != nil {
			return k, err
		}
		video = append(video, on30(shots, fly)...)
	}
	return n, encodeMP4(video, filepath.Join(out, name+".mp4"))
}
