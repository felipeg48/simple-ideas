// Package cdp is a small Chrome DevTools Protocol client over
// --remote-debugging-pipe (requests on Chrome's fd 3, replies and events on
// its fd 4, one JSON value per message, NUL-terminated), standard library
// only. si uses it to film decks in real motion (cmd/si/motion.go) and to run
// web demo scripts (internal/webdemo).
package cdp

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sync"
)

// Event is one DevTools event: its method, the session it came from ("" for
// the browser) and its parameters.
type Event struct {
	Method  string
	Session string
	Params  json.RawMessage
}

type reply struct {
	result json.RawMessage
	err    error
}

// Client is one headless Chrome. Call sends to Session (the page attached
// last) unless CallIn names another.
type Client struct {
	cmd     *exec.Cmd
	w       *os.File
	wmu     sync.Mutex
	mu      sync.Mutex
	id      int
	pending map[int]chan reply
	onEvent func(Event)
	dead    error
	Session string
}

// Launch starts headless Chrome with the DevTools pipe, a window of w×h CSS
// pixels and extra flags, without opening a page.
func Launch(chrome string, w, h int, extra ...string) (*Client, error) {
	toChrome, cw, err := os.Pipe() // Chrome reads its fd 3
	if err != nil {
		return nil, err
	}
	cr, fromChrome, err := os.Pipe() // Chrome writes its fd 4
	if err != nil {
		return nil, err
	}
	args := append([]string{"--headless=new", "--use-mock-keychain", "--password-store=basic", "--remote-debugging-pipe", "--hide-scrollbars", "--no-first-run", "--no-default-browser-check",
		fmt.Sprintf("--window-size=%d,%d", w, h)}, extra...)
	cmd := exec.Command(chrome, append(args, "about:blank")...)
	cmd.ExtraFiles = []*os.File{toChrome, fromChrome}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	toChrome.Close()
	fromChrome.Close()
	c := &Client{cmd: cmd, w: cw, pending: map[int]chan reply{}}
	go c.read(bufio.NewReaderSize(cr, 1<<20))
	return c, nil
}

// Start launches Chrome and attaches to a blank page of w×h CSS pixels.
func Start(chrome string, w, h int, extra ...string) (*Client, error) {
	c, err := Launch(chrome, w, h, extra...)
	if err != nil {
		return nil, err
	}
	if _, err := c.OpenPage("", w, h); err != nil {
		c.Close()
		return nil, err
	}
	return c, nil
}

// OpenPage opens a blank page (in the browser context ctx, "" for the
// default one), attaches to it, makes it the Session and sets its viewport to
// w×h CSS pixels. It returns the page's target id.
func (c *Client) OpenPage(ctx string, w, h int) (string, error) {
	p := map[string]any{"url": "about:blank"}
	if ctx != "" {
		p["browserContextId"] = ctx
	}
	var tgt struct{ TargetID string }
	if err := c.CallIn("", "Target.createTarget", p, &tgt); err != nil {
		return "", err
	}
	var att struct{ SessionID string }
	if err := c.CallIn("", "Target.attachToTarget", map[string]any{"targetId": tgt.TargetID, "flatten": true}, &att); err != nil {
		return "", err
	}
	c.Session = att.SessionID
	return tgt.TargetID, c.Call("Emulation.setDeviceMetricsOverride", map[string]any{"width": w, "height": h, "deviceScaleFactor": 1, "mobile": false}, nil)
}

// OnEvent sets the function that receives every event. It runs on the
// reader goroutine: it must not Call.
func (c *Client) OnEvent(f func(Event)) {
	c.mu.Lock()
	c.onEvent = f
	c.mu.Unlock()
}

// Close stops Chrome.
func (c *Client) Close() {
	c.w.Close()
	c.cmd.Process.Kill()
	c.cmd.Wait()
}

func (c *Client) read(r *bufio.Reader) {
	for {
		raw, err := r.ReadBytes(0)
		if err != nil {
			c.mu.Lock()
			c.dead = fmt.Errorf("chrome: %v", err)
			for id, ch := range c.pending {
				ch <- reply{err: c.dead}
				delete(c.pending, id)
			}
			c.mu.Unlock()
			return
		}
		var m struct {
			ID        int             `json:"id"`
			Method    string          `json:"method"`
			SessionID string          `json:"sessionId"`
			Params    json.RawMessage `json:"params"`
			Result    json.RawMessage `json:"result"`
			Error     *struct{ Message string }
		}
		if json.Unmarshal(raw[:len(raw)-1], &m) != nil {
			continue
		}
		c.mu.Lock()
		if m.ID == 0 {
			f := c.onEvent
			c.mu.Unlock()
			if f != nil && m.Method != "" {
				f(Event{m.Method, m.SessionID, m.Params})
			}
			continue
		}
		ch := c.pending[m.ID]
		delete(c.pending, m.ID)
		c.mu.Unlock()
		if ch == nil {
			continue
		}
		if m.Error != nil {
			ch <- reply{err: fmt.Errorf("%s", m.Error.Message)}
		} else {
			ch <- reply{result: m.Result}
		}
	}
}

// Call sends one command to the page Session and waits for its reply.
func (c *Client) Call(method string, params, out any) error {
	return c.CallIn(c.Session, method, params, out)
}

// CallIn sends one command to session ("" for the browser) and waits for its
// reply, decoded into out when out is not nil.
func (c *Client) CallIn(session, method string, params, out any) error {
	if params == nil {
		params = map[string]any{}
	}
	ch := make(chan reply, 1)
	c.mu.Lock()
	if c.dead != nil {
		c.mu.Unlock()
		return c.dead
	}
	c.id++
	id := c.id
	c.pending[id] = ch
	c.mu.Unlock()
	msg := map[string]any{"id": id, "method": method, "params": params}
	if session != "" {
		msg["sessionId"] = session
	}
	b, _ := json.Marshal(msg)
	c.wmu.Lock()
	_, err := c.w.Write(append(b, 0))
	c.wmu.Unlock()
	if err != nil {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return err
	}
	rep := <-ch
	if rep.err != nil {
		return fmt.Errorf("%s: %v", method, rep.err)
	}
	if out != nil {
		return json.Unmarshal(rep.result, out)
	}
	return nil
}

// Eval runs JavaScript in the page (awaiting a promise) and returns its value
// as JSON. A thrown exception is an error.
func (c *Client) Eval(js string) (json.RawMessage, error) {
	var r struct {
		Result struct {
			Value json.RawMessage `json:"value"`
		} `json:"result"`
		ExceptionDetails *struct {
			Text      string
			Exception struct{ Description string }
		} `json:"exceptionDetails"`
	}
	if err := c.Call("Runtime.evaluate", map[string]any{"expression": js, "returnByValue": true, "awaitPromise": true}, &r); err != nil {
		return nil, err
	}
	if e := r.ExceptionDetails; e != nil {
		if e.Exception.Description != "" {
			return nil, fmt.Errorf("%s", e.Exception.Description)
		}
		return nil, fmt.Errorf("%s", e.Text)
	}
	return r.Result.Value, nil
}

// Key presses a key in the page (down, then up).
func (c *Client) Key(name string, code int) error {
	for _, t := range []string{"keyDown", "keyUp"} {
		if err := c.Call("Input.dispatchKeyEvent", map[string]any{"type": t, "key": name, "code": name, "windowsVirtualKeyCode": code, "nativeVirtualKeyCode": code}, nil); err != nil {
			return err
		}
	}
	return nil
}

// Screenshot is the page as a PNG.
func (c *Client) Screenshot() ([]byte, error) {
	var shot struct{ Data string }
	if err := c.Call("Page.captureScreenshot", map[string]any{"format": "png"}, &shot); err != nil {
		return nil, err
	}
	return base64.StdEncoding.DecodeString(shot.Data)
}
