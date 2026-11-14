package webdemo

import (
	"encoding/json"
	"fmt"
)

// Steps inside framed pages: `click "Buy" in frame .preview`. A same-origin
// frame is reached through an isolated world of the page's own target; a
// cross-origin one (out of process) through the target Chrome attaches for
// it (Target.setAutoAttach in feature). Clicks and keys still go to the
// page, at page coordinates: Chrome routes them into the frame.

// frameCtx is where an element in a frame lives: the session, and the
// execution context in it (0: the session's main world).
type frameCtx struct {
	session string
	ctx     int
	sel     string // the iframe, a CSS selector in the page
}

// evalIn runs JavaScript in session's context ctx (0: its main world) and
// returns its value as JSON; a thrown exception is an error.
func (r *runner) evalIn(session string, ctx int, js string) (json.RawMessage, error) {
	p := map[string]any{"expression": js, "returnByValue": true, "awaitPromise": true}
	if ctx > 0 {
		p["contextId"] = ctx
	}
	var v struct {
		Result struct {
			Value json.RawMessage `json:"value"`
		} `json:"result"`
		ExceptionDetails *struct {
			Text      string
			Exception struct{ Description string }
		} `json:"exceptionDetails"`
	}
	if err := r.c.CallIn(session, "Runtime.evaluate", p, &v); err != nil {
		return nil, err
	}
	if e := v.ExceptionDetails; e != nil {
		return nil, fmt.Errorf("%s", firstOf(e.Exception.Description, e.Text))
	}
	return v.Result.Value, nil
}

// frameContext finds where the iframe sel's document can be reached.
func (r *runner) frameContext(sel string) (*frameCtx, error) {
	q, _ := json.Marshal(sel)
	var obj struct {
		Result struct {
			ObjectID string `json:"objectId"`
		} `json:"result"`
	}
	if err := r.c.Call("Runtime.evaluate", map[string]any{"expression": "document.querySelector(" + string(q) + ")"}, &obj); err != nil {
		return nil, err
	}
	if obj.Result.ObjectID == "" {
		return nil, nil
	}
	var node struct {
		Node struct {
			FrameID string `json:"frameId"`
		} `json:"node"`
	}
	if err := r.c.Call("DOM.describeNode", map[string]any{"objectId": obj.Result.ObjectID}, &node); err != nil {
		return nil, err
	}
	r.c.Call("Runtime.releaseObject", map[string]any{"objectId": obj.Result.ObjectID}, nil)
	id := node.Node.FrameID
	if id == "" {
		return nil, fmt.Errorf("%s is not a frame", sel)
	}
	var w struct {
		ExecutionContextID int `json:"executionContextId"`
	}
	if err := r.c.Call("Page.createIsolatedWorld", map[string]any{"frameId": id, "worldName": "si-demo"}, &w); err == nil {
		return &frameCtx{r.c.Session, w.ExecutionContextID, sel}, nil
	}
	r.mu.Lock()
	s := r.oopif[id]
	r.mu.Unlock()
	if s == "" {
		return nil, nil // not attached yet: the frame is still coming
	}
	return &frameCtx{s, 0, sel}, nil
}

// locateIn is locate for an element in a frame: the iframe is found (and
// scrolled into view) in the page first, then the element in its document;
// its centre comes back in page coordinates, and the page's window.__siT
// stands for it (a box that follows the iframe), for the notes, the
// spotlight and the camera.
func (r *runner) locateIn(l Locator, want string, scroll bool) (found, error) {
	frame, err := r.locate(Locator{CSS: l.Frame}, "any", scroll)
	if err != nil || !frame.Found || !frame.Visible {
		frame.Found = false
		return frame, err
	}
	fc, err := r.frameContext(l.Frame)
	if err != nil {
		return found{Error: err.Error()}, nil
	}
	if fc == nil {
		return found{}, nil
	}
	inner := l
	inner.Frame = ""
	arg, _ := json.Marshal(map[string]any{"loc": inner, "want": want, "scroll": scroll, "header": "none"})
	v, err := r.evalIn(fc.session, fc.ctx, locateJS+"("+string(arg)+")")
	if err != nil {
		return found{}, nil // its document is being replaced: try again
	}
	var f found
	if err := json.Unmarshal(v, &f); err != nil {
		return f, err
	}
	q, _ := json.Marshal(l.Frame)
	o, err := r.c.Eval(`(function(f){var r=f.getBoundingClientRect(),s=getComputedStyle(f);return [r.left+f.clientLeft+parseFloat(s.paddingLeft),r.top+f.clientTop+parseFloat(s.paddingTop)]})(document.querySelector(` + string(q) + `))`)
	var off [2]float64
	if err != nil || json.Unmarshal(o, &off) != nil {
		return found{}, nil
	}
	f.X, f.Y = f.X+off[0], f.Y+off[1]
	r.fr = fc
	if f.Found {
		r.c.Eval(fmt.Sprintf(`window.__siT={get isConnected(){return !!document.querySelector(%s)},getBoundingClientRect:function(){var f=document.querySelector(%s);if(!f)return {left:0,top:0,width:0,height:0,right:0,bottom:0};var r=f.getBoundingClientRect(),s=getComputedStyle(f),x=r.left+f.clientLeft+parseFloat(s.paddingLeft)+%g,y=r.top+f.clientTop+parseFloat(s.paddingTop)+%g;return {left:x,top:y,width:%g,height:%g,right:x+%g,bottom:y+%g}}}`,
			q, q, f.Box[0], f.Box[1], f.Box[2], f.Box[3], f.Box[2], f.Box[3]))
	}
	return f, nil
}

// closePages closes the feature's pages (session: keep, where the browser
// context outlives the feature).
func (r *runner) closePages() {
	for _, id := range append([]string{r.appTarget}, r.wins...) {
		r.c.CallIn("", "Target.closeTarget", map[string]any{"targetId": id}, nil)
	}
}
