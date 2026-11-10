package webdemo

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/felipeg48/simple-ideas/internal/cdp"
)

//go:embed locate.js
var locateJS string

// Options are how a run goes.
type Options struct {
	Chrome  string    // the browser to drive
	Feature string    // run only the feature with this name ("" = all)
	Out     string    // where the screenshots go (the report folder)
	Log     io.Writer // one line per step (nil: quiet)
	SI      string    // si's version, for the report

	// si demo run: film each feature (Video), narrated by Voice
	Video    bool
	Voice    Voice
	Captions bool   // burn the captions into the video
	FFmpeg   string // the ffmpeg to encode with
	Cache    string // where voiced lines are kept ("" = VoiceCache())
}

// failure is a step that did not pass: what it expected, what it found, the
// candidates the locator saw.
type failure struct {
	expected, found string
	candidates      []string
	output          string // a terminal's or a program's last lines
}

func (f *failure) Error() string { return f.found }

type runner struct {
	s       *Script
	o       Options
	c       *cdp.Client
	base    string // the app's URL, without a trailing /
	secrets []string
	mu      sync.Mutex
	errs    []string // console errors and exceptions of this feature
	reqs    []string // failed requests of this feature
	urls    map[string]string
	session string // the page whose events count
	loading int    // frames loading now (a click or a key may start a navigation)
	shotN   int
	tk      *take           // the feature being filmed (run)
	lines   map[*Block]Line // each Say: line voiced (run)
	timeout time.Duration   // how long a step waits for its element

	// terminals and background programs (runterm.go)
	ctx      string // the feature's browser context
	app      string // the app page's session
	home     bool   // the app page has loaded the app
	film     string // the page whose frames the take keeps
	srv      *termServer
	rc       string // si's shell prompt files
	pm       sync.Mutex
	terms    map[string]*term // this feature's terminals
	allTerms []*term
	cur      *term            // the terminal in front (nil: the browser)
	procs    map[string]*proc // programs by name
	all      []*proc
	local    []*proc // this feature's start steps
	quiet    []quietCheck
	down     sync.Once
}

// Run runs the script's features as tests and returns the report. An error
// means the run could not start (no app, no browser), not a failed step.
func Run(s *Script, o Options) (*Report, error) {
	if len(s.Problems) > 0 {
		return nil, fmt.Errorf("%s", s.Problems[0])
	}
	r := &runner{s: s, o: o, timeout: s.Timeout, terms: map[string]*term{}, procs: map[string]*proc{}}
	if o.Log == nil {
		r.o.Log = io.Discard
	}
	defer r.shutdown() // terminals and programs never outlive the run, even on a panic
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	quit := make(chan struct{})
	defer func() { signal.Stop(sig); close(quit) }()
	go func() {
		select {
		case <-sig: // Ctrl-C: stop what the run started, then go
			fmt.Fprintln(r.o.Log, "interrupted: stopping the terminals and programs")
			r.shutdown()
			r.pm.Lock()
			c := r.c
			r.pm.Unlock()
			if c != nil {
				c.Close()
			}
			os.Exit(130)
		case <-quit:
		}
	}()
	if o.Video {
		// a take fails fast: a long wait would be seconds of nothing in the video
		r.timeout = min(s.Timeout, 3*time.Second)
		if err := r.voiceLines(); err != nil {
			return nil, err
		}
	}
	base, stop, err := serveApp(s)
	if err != nil {
		return nil, err
	}
	defer stop()
	r.base = base
	for _, f := range s.Features {
		for _, b := range f.Blocks {
			for _, st := range b.Steps {
				if v, ok := os.LookupEnv(st.Value.Var); ok && st.Value.Var != "" && v != "" {
					r.secrets = append(r.secrets, v)
				}
			}
		}
	}
	r.secrets = append(r.secrets, commandSecrets(s)...)
	if o.Out != "" {
		if err := os.MkdirAll(filepath.Join(o.Out, "shots"), 0o755); err != nil {
			return nil, err
		}
	}
	c, err := cdp.Launch(o.Chrome, s.W, s.H)
	if err != nil {
		return nil, err
	}
	defer c.Close()
	r.pm.Lock()
	r.c = c
	r.pm.Unlock()
	var ver struct{ Product string }
	c.CallIn("", "Browser.getVersion", nil, &ver)
	c.OnEvent(r.event)
	rep := &Report{Script: filepath.Base(s.Path), Title: s.Title, App: r.mask(base), Viewport: fmt.Sprintf("%dx%d", s.W, s.H),
		Chrome: ver.Product, SI: o.SI, Date: time.Now().Format(time.RFC3339)}
	if rep.Title == "" {
		rep.Title = s.Name
	}
	if o.Video && len(r.lines) > 0 {
		rep.Voice = o.Voice.String()
	}
	start := time.Now()
	if len(r.features()) == 0 {
		return nil, fmt.Errorf("no feature named %q in %s", o.Feature, filepath.Base(s.Path))
	}
	// start: programs, for the whole run; when one does not come up, every
	// feature fails with its last lines
	var down *StepResult
	for _, p := range s.Starts {
		fmt.Fprintf(r.o.Log, "start: %s\n", r.mask(p.Cmd))
		t := time.Now()
		if err := r.startProgram(p.Name, p.Cmd, p.Ready, p.Within, false); err != nil {
			sr := StepResult{Line: p.Line, Text: "start: " + p.Cmd, Kind: "start", Status: "failed", Seconds: time.Since(t).Seconds(), Found: r.mask(err.Error())}
			var fe *failure
			if errors.As(err, &fe) {
				sr.Expected, sr.Found, sr.Output = r.mask(fe.expected), r.mask(fe.found), fe.output
			}
			r.line(sr)
			down = &sr
			break
		}
	}
	var clips []*clip
	for _, f := range r.features() {
		if down != nil {
			rep.Features = append(rep.Features, FeatureResult{Name: f.Name, What: f.What, Line: f.Line, Status: "failed", Steps: []StepResult{*down}})
			continue
		}
		fr, cl := r.feature(f)
		if cl != nil {
			clips = append(clips, cl)
			fr.Video = cl.Name + ".mp4"
		}
		rep.Features = append(rep.Features, fr)
	}
	if len(clips) > 0 {
		fmt.Fprintf(r.o.Log, "all.mp4: %d feature(s) with title cards and chapters\n", len(clips))
		if err := r.joinAll(clips); err != nil {
			return nil, err
		}
	}
	rep.Seconds = time.Since(start).Seconds()
	rep.count()
	return rep, nil
}

// features are the features this run performs (--feature, or all).
func (r *runner) features() []*Feature {
	var fs []*Feature
	for _, f := range r.s.Features {
		if r.o.Feature == "" || strings.EqualFold(f.Name, r.o.Feature) {
			fs = append(fs, f)
		}
	}
	return fs
}

// voiceLines voices every Say: line of the features to film, before the
// take: their lengths set its pace.
func (r *runner) voiceLines() error {
	v := r.o.Voice
	v.Style = r.s.Style
	var texts []string
	var blocks []*Block
	for _, f := range r.features() {
		for _, b := range f.Blocks {
			if strings.TrimSpace(b.Say) != "" {
				blocks = append(blocks, b)
				texts = append(texts, Spoken(b.Say, r.s.Pronounce))
			}
		}
	}
	r.lines = map[*Block]Line{}
	if len(texts) == 0 {
		return nil
	}
	if why := v.Ready(); why != "" {
		return fmt.Errorf("%s", why)
	}
	cache := r.o.Cache
	if cache == "" {
		cache = VoiceCache()
	}
	sy := &Synth{FFmpeg: r.o.FFmpeg, Cache: cache, Log: r.o.Log}
	fmt.Fprintf(r.o.Log, "voice: %s, %d line(s)\n", v, len(texts))
	lines, n, err := sy.Voice(v, texts)
	if err != nil {
		return err
	}
	total := 0.0
	for i, b := range blocks {
		r.lines[b] = lines[texts[i]]
		total += lines[texts[i]].Seconds
	}
	fmt.Fprintf(r.o.Log, "voice: %d new, %d from the cache, %.1fs of narration\n", n, len(lines)-n, total)
	return nil
}

// serveApp gives the app's base URL: the app: URL, or a loopback server of
// the app: folder (stopped by the returned func).
func serveApp(s *Script) (string, func(), error) {
	app := s.App
	if strings.Contains(app, "://") {
		return strings.TrimRight(app, "/"), func() {}, nil
	}
	dir := app
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(s.Dir(), dir)
	}
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		return "", nil, fmt.Errorf("app: %s is not a URL or a folder next to the script", app)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", nil, err
	}
	srv := &http.Server{Handler: http.FileServer(http.Dir(dir))}
	go srv.Serve(ln)
	return "http://" + ln.Addr().String(), func() { srv.Close() }, nil
}

func (r *runner) mask(s string) string {
	for _, v := range r.secrets {
		s = strings.ReplaceAll(s, v, "••••")
	}
	return s
}

// event collects the page's console errors, exceptions and failed requests.
func (r *runner) event(e cdp.Event) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if e.Method == "Page.screencastFrame" { // the scene in front: the app or a terminal
		if e.Session == r.film {
			r.onFrame(e.Params)
		}
		return
	}
	if e.Session != r.session {
		return
	}
	switch e.Method {
	case "Page.frameStartedLoading":
		r.loading++
	case "Page.frameStoppedLoading":
		r.loading--
	case "Runtime.exceptionThrown":
		var p struct {
			ExceptionDetails struct {
				Text      string
				Exception struct{ Description string }
			}
		}
		json.Unmarshal(e.Params, &p)
		msg := p.ExceptionDetails.Exception.Description
		if msg == "" {
			msg = p.ExceptionDetails.Text
		}
		r.errs = append(r.errs, firstLine(msg))
	case "Runtime.consoleAPICalled":
		var p struct {
			Type string
			Args []struct {
				Value       any
				Description string
			}
		}
		json.Unmarshal(e.Params, &p)
		if p.Type != "error" && p.Type != "assert" {
			return
		}
		var parts []string
		for _, a := range p.Args {
			if s, ok := a.Value.(string); ok {
				parts = append(parts, s)
			} else if a.Description != "" {
				parts = append(parts, firstLine(a.Description))
			} else if a.Value != nil {
				parts = append(parts, fmt.Sprint(a.Value))
			}
		}
		r.errs = append(r.errs, "console.error: "+strings.Join(parts, " "))
	case "Network.requestWillBeSent":
		var p struct {
			RequestID string
			Request   struct{ URL string }
		}
		json.Unmarshal(e.Params, &p)
		r.urls[p.RequestID] = p.Request.URL
	case "Network.responseReceived":
		var p struct {
			Response struct {
				URL        string
				Status     int
				StatusText string
			}
		}
		json.Unmarshal(e.Params, &p)
		if p.Response.Status >= 400 && !ignoredURL(p.Response.URL) {
			r.reqs = append(r.reqs, fmt.Sprintf("%d %s", p.Response.Status, p.Response.URL))
		}
	case "Network.loadingFailed":
		var p struct {
			RequestID string
			ErrorText string
			Canceled  bool
		}
		json.Unmarshal(e.Params, &p)
		if u := r.urls[p.RequestID]; !p.Canceled && !ignoredURL(u) {
			r.reqs = append(r.reqs, p.ErrorText+" "+u)
		}
	}
}

// ignoredURL: browsers ask for /favicon.ico on their own.
func ignoredURL(u string) bool { return strings.HasSuffix(u, "/favicon.ico") }

func firstLine(s string) string {
	s, _, _ = strings.Cut(s, "\n")
	return s
}

// feature runs one feature in a fresh browser context, after setup:; in a
// run it films it too and returns its clip.
func (r *runner) feature(f *Feature) (fr FeatureResult, cl *clip) {
	fr = FeatureResult{Name: f.Name, What: f.What, Line: f.Line, Status: "passed"}
	fmt.Fprintf(r.o.Log, "%s\n", f.Name)
	start := time.Now()
	defer func() { fr.Seconds = time.Since(start).Seconds() }()
	fail := func(sr StepResult) (FeatureResult, *clip) {
		fr.Steps = append(fr.Steps, sr)
		fr.Status = "failed"
		return fr, nil
	}
	if r.s.Setup != "" {
		t := time.Now()
		out, err := shell(r.s.Setup, r.s.Dir())
		sr := StepResult{Text: "setup: " + r.s.Setup, Status: "passed", Seconds: time.Since(t).Seconds()}
		r.line(sr)
		if err != nil {
			sr.Status, sr.Expected, sr.Found = "failed", "setup: exits 0", r.mask(strings.TrimSpace(err.Error()+"\n"+out))
			r.line(sr)
			return fail(sr)
		}
	}
	var ctx struct{ BrowserContextID string }
	if err := r.c.CallIn("", "Target.createBrowserContext", map[string]any{"disposeOnDetach": true}, &ctx); err != nil {
		return fail(StepResult{Text: "open a browser context", Status: "failed", Found: err.Error()})
	}
	defer r.c.CallIn("", "Target.disposeBrowserContext", map[string]any{"browserContextId": ctx.BrowserContextID}, nil)
	defer r.endFeature() // its terminals and start steps' programs, before its context goes
	r.ctx = ctx.BrowserContextID
	r.mu.Lock()
	r.errs, r.reqs, r.urls, r.loading = nil, nil, map[string]string{}, 0
	r.mu.Unlock()
	if _, err := r.c.OpenPage(ctx.BrowserContextID, r.s.W, r.s.H); err != nil {
		return fail(StepResult{Text: "open a page", Status: "failed", Found: err.Error()})
	}
	r.mu.Lock()
	r.session = r.c.Session
	r.mu.Unlock()
	r.app, r.home = r.c.Session, false
	for _, m := range []string{"Page.enable", "Runtime.enable", "Network.enable"} {
		if err := r.c.Call(m, nil, nil); err != nil {
			return fail(StepResult{Text: m, Status: "failed", Found: err.Error()})
		}
	}
	if r.o.Video {
		if err := r.startTake(); err != nil {
			return fail(StepResult{Text: "film the page", Status: "failed", Found: err.Error()})
		}
	}
	// a feature that begins in a terminal loads the app at its first browser
	// step (the app may be what the terminal starts)
	if first := firstStep(f); first != nil && first.Kind == "terminal" {
		if err := r.terminal(first); err != nil { // opened (and filmed) before the first Say:
			return fail(StepResult{Line: first.Line, Text: first.Text, Status: "failed", Found: r.mask(err.Error())})
		}
	} else {
		r.home = true
		if err := r.navigate(r.base + "/"); err != nil {
			return fail(StepResult{Text: "open " + r.base, Status: "failed", Found: err.Error()})
		}
		if r.tk != nil {
			if err := r.roll(); err != nil {
				return fail(StepResult{Text: "film the page", Status: "failed", Found: err.Error()})
			}
			time.Sleep(800 * time.Millisecond) // the page, before anything moves
		}
	}
	failed := false
	for _, b := range f.Blocks {
		if r.tk != nil && !failed {
			r.beginBlock(b)
		}
		for i, st := range b.Steps {
			sr := StepResult{Line: st.Line, Text: st.Text, Kind: st.Kind, Status: "passed"}
			if failed {
				sr.Status = "skipped"
				fr.Steps = append(fr.Steps, sr)
				continue
			}
			if r.tk != nil {
				r.pace(i, len(b.Steps))
			}
			t := time.Now()
			shot, err := r.step(st)
			sr.Seconds = time.Since(t).Seconds()
			if r.tk != nil {
				r.claim()
				r.expire()
				r.tk.last = time.Now()
			}
			if shot != "" {
				sr.Shot = shot
			}
			if st.Kind == "note" {
				sr.Note = st.Value.String()
			}
			if err != nil {
				failed = true
				fr.Status = "failed"
				sr.Status = "failed"
				var fe *failure
				if errors.As(err, &fe) {
					sr.Expected, sr.Found, sr.Candidates, sr.Output = r.mask(fe.expected), r.mask(fe.found), fe.candidates, fe.output
				} else {
					sr.Found = r.mask(err.Error())
				}
				if sr.Output == "" && r.cur != nil {
					sr.Output = r.cur.out.tail(40)
				}
				if png, err := r.c.Screenshot(); err == nil {
					sr.Shot = r.save(f.Name, "failed", png)
				}
			}
			r.line(sr)
			fr.Steps = append(fr.Steps, sr)
			if st.Kind == "expect-no-output" && sr.Status == "passed" {
				r.quiet[len(r.quiet)-1].idx = len(fr.Steps) - 1
			}
		}
		if i, fe := r.checkQuiet(); fe != nil && !failed {
			failed, fr.Status = true, "failed"
			sr := &fr.Steps[i]
			sr.Status, sr.Expected, sr.Found, sr.Output = "failed", fe.expected, fe.found, fe.output
			r.line(*sr)
		}
		if r.tk != nil && !failed {
			r.endBlock()
		}
	}
	r.c.Session = r.app // the cards and captions are drawn in the app's tab
	if r.tk != nil {
		time.Sleep(time.Second) // the end (or the failure), held
		d := r.cut()
		var err error
		if cl, err = r.finishTake(f, d); err != nil {
			fr.Status = "failed"
			fr.Steps = append(fr.Steps, StepResult{Text: "make the video", Status: "failed", Found: err.Error()})
			cl = nil
		} else {
			fmt.Fprintf(r.o.Log, "  ▶ %s.mp4  %.1fs\n", cl.Name, cl.Seconds)
		}
	}
	r.mu.Lock()
	fr.Errors = append([]string(nil), r.errs...)
	fr.Requests = append([]string(nil), r.reqs...)
	r.mu.Unlock()
	for i := range fr.Errors {
		fr.Errors[i] = r.mask(fr.Errors[i])
	}
	for i := range fr.Requests {
		fr.Requests[i] = r.mask(fr.Requests[i])
	}
	return fr, cl
}

func firstStep(f *Feature) *Step {
	for _, b := range f.Blocks {
		if len(b.Steps) > 0 {
			return b.Steps[0]
		}
	}
	return nil
}

func (r *runner) line(sr StepResult) {
	mark := map[string]string{"passed": "✓", "failed": "✗", "skipped": "-"}[sr.Status]
	fmt.Fprintf(r.o.Log, "  %s %s  %.2fs\n", mark, sr.Text, sr.Seconds)
	if sr.Status != "failed" {
		return
	}
	if sr.Expected != "" {
		fmt.Fprintf(r.o.Log, "      expected: %s\n", sr.Expected)
	}
	fmt.Fprintf(r.o.Log, "      found:    %s\n", sr.Found)
	if len(sr.Candidates) > 0 {
		fmt.Fprintf(r.o.Log, "      closest:  %s\n", strings.Join(sr.Candidates, ", "))
	}
}

// shell runs setup: and stop: in dir.
func shell(cmd, dir string) (string, error) {
	out, err := shellCmd(cmd, dir).CombinedOutput()
	return string(out), err
}

var slugRE = regexp.MustCompile(`[^a-z0-9]+`)

func slug(s string) string {
	return strings.Trim(slugRE.ReplaceAllString(strings.ToLower(s), "-"), "-")
}

// save writes a screenshot into the report folder and returns its path
// relative to it ("" when there is no folder).
func (r *runner) save(feature, name string, png []byte) string {
	if r.o.Out == "" {
		return ""
	}
	r.shotN++
	rel := fmt.Sprintf("shots/%02d-%s-%s.png", r.shotN, slug(feature), slug(name))
	if os.WriteFile(filepath.Join(r.o.Out, rel), png, 0o644) != nil {
		return ""
	}
	return rel
}

// resolve turns a step's path into a URL of the app.
func (r *runner) resolve(p string) string {
	if strings.Contains(p, "://") {
		return p
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return r.base + p
}

// navigate loads u and waits for the page to settle.
func (r *runner) navigate(u string) error {
	var nav struct{ ErrorText string }
	if err := r.c.Call("Page.navigate", map[string]any{"url": u}, &nav); err != nil {
		return err
	}
	if nav.ErrorText != "" {
		return &failure{expected: "the page loads: " + u, found: nav.ErrorText}
	}
	return r.settle()
}

// settle waits until the document is complete (up to the timeout).
func (r *runner) settle() error {
	end := time.Now().Add(r.s.Timeout)
	for {
		v, err := r.c.Eval("document.readyState")
		if err == nil && string(v) == `"complete"` {
			time.Sleep(50 * time.Millisecond) // a frame for scripts that run on load
			return nil
		}
		if time.Now().After(end) {
			return &failure{expected: "the page finishes loading", found: "still " + string(v)}
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// loaded waits, after a click or a key, for a navigation it started to
// finish, so the next step does not look at the page being left.
func (r *runner) loaded() error {
	time.Sleep(60 * time.Millisecond)
	end := time.Now().Add(r.s.Timeout)
	for {
		r.mu.Lock()
		n := r.loading
		r.mu.Unlock()
		if n <= 0 {
			return nil
		}
		if time.Now().After(end) {
			return &failure{expected: "the page finishes loading", found: "still loading"}
		}
		time.Sleep(30 * time.Millisecond)
	}
}

// found is the locator's answer (locate.js).
type found struct {
	Found      bool
	Visible    bool
	Enabled    bool
	Checked    bool
	Field      bool
	Tag        string
	Desc       string
	Count      int
	X, Y       float64
	Covered    string
	Candidates []string
	Error      string
}

func (r *runner) locate(l Locator, want string, scroll bool) (found, error) {
	arg, _ := json.Marshal(map[string]any{"loc": l, "want": want, "scroll": scroll})
	v, err := r.c.Eval(locateJS + "(" + string(arg) + ")")
	var f found
	if err != nil {
		return f, err
	}
	err = json.Unmarshal(v, &f)
	return f, err
}

// until retries try every 100 ms up to the timeout; try returns done, or the
// failure to report when time runs out.
func (r *runner) until(try func() (bool, error)) error { return r.untilFor(r.timeout, try) }

// untilFor is until with its own timeout.
func (r *runner) untilFor(d time.Duration, try func() (bool, error)) error {
	end := time.Now().Add(d)
	for {
		done, err := try()
		if done {
			return err
		}
		if time.Now().After(end) {
			return err
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// element waits for a visible, enabled (when act) element, scrolled into
// view and not covered, and returns where it is.
func (r *runner) element(l Locator, want string, act bool) (found, error) {
	var last found
	err := r.until(func() (bool, error) {
		f, err := r.locate(l, want, true)
		if err != nil {
			return false, &failure{expected: "an element " + l.String(), found: err.Error()}
		}
		last = f
		switch {
		case f.Error != "":
			return true, &failure{expected: "an element " + l.String(), found: f.Error}
		case !f.Found:
			return false, &failure{expected: "an element " + l.String(), found: "nothing matches", candidates: f.Candidates}
		case !f.Visible:
			return false, &failure{expected: l.String() + " visible", found: f.Desc + ", hidden"}
		case act && !f.Enabled:
			return false, &failure{expected: l.String() + " enabled", found: f.Desc + ", disabled"}
		case act && f.Covered != "":
			return false, &failure{expected: l.String() + " clickable", found: f.Desc + " is covered by " + f.Covered}
		}
		return true, nil
	})
	return last, err
}

func (r *runner) mouse(typ string, x, y float64, extra map[string]any) error {
	p := map[string]any{"type": typ, "x": x, "y": y}
	for k, v := range extra {
		p[k] = v
	}
	return r.c.Call("Input.dispatchMouseEvent", p, nil)
}

// click is a real click at the element's centre (in a take, the drawn
// cursor glides there first and ripples).
func (r *runner) click(f found) error {
	r.glide(f.X, f.Y)
	r.ripple(f.X, f.Y)
	if err := r.mouse("mouseMoved", f.X, f.Y, nil); err != nil {
		return err
	}
	if err := r.mouse("mousePressed", f.X, f.Y, map[string]any{"button": "left", "buttons": 1, "clickCount": 1}); err != nil {
		return err
	}
	return r.mouse("mouseReleased", f.X, f.Y, map[string]any{"button": "left", "buttons": 0, "clickCount": 1})
}

func (r *runner) value(v Value) (string, error) {
	if v.Var == "" {
		return v.Text, nil
	}
	s, ok := os.LookupEnv(v.Var)
	if !ok {
		return "", &failure{expected: "$" + v.Var + " set in the environment", found: "it is not set"}
	}
	return s, nil
}

// step runs one step; it returns a screenshot's path when it took one.
func (r *runner) step(st *Step) (string, error) {
	switch st.Kind {
	case "terminal", "browser", "run", "expect-output", "expect-no-output", "waitfor-output", "start", "stop", "expect-proc":
		return "", r.termStep(st)
	case "press", "note":
		if r.cur != nil && (st.Kind == "press" || st.AtLine) {
			return "", r.termStep(st)
		}
	}
	switch st.Kind {
	case "go":
		return "", r.navigate(r.resolve(st.Value.Text))
	case "click":
		f, err := r.element(st.Target, "click", true)
		if err != nil {
			return "", err
		}
		if err := r.click(f); err != nil {
			return "", err
		}
		return "", r.loaded()
	case "hover":
		f, err := r.element(st.Target, "click", false)
		if err != nil {
			return "", err
		}
		r.glide(f.X, f.Y)
		return "", r.mouse("mouseMoved", f.X, f.Y, nil)
	case "scroll", "waitfor", "note":
		var err error
		if st.Kind != "note" || st.Target != (Locator{}) {
			_, err = r.element(st.Target, "any", false)
		}
		if err == nil && st.Kind == "note" {
			r.note(st)
		}
		return "", err
	case "type":
		text, err := r.value(st.Value)
		if err != nil {
			return "", err
		}
		f, err := r.element(st.Target, "field", true)
		if err != nil {
			return "", err
		}
		if err := r.click(f); err != nil {
			return "", err
		}
		if st.Value.Var != "" { // a secret: blurred in the video
			r.js(`__siDemo.blur()`)
		}
		// replace what the field holds, as a person selecting it all would
		if _, err := r.c.Eval(`(function(e){if(document.activeElement!==e)e.focus();if(e.select)e.select();else{var s=getSelection(),g=document.createRange();g.selectNodeContents(e);s.removeAllRanges();s.addRange(g)}})(window.__siT)`); err != nil {
			return "", err
		}
		if text == "" { // type "" clears the field
			return "", r.press(Chord{Key: "Backspace"})
		}
		if r.tk != nil {
			return "", r.typeSlowly(text)
		}
		return "", r.c.Call("Input.insertText", map[string]any{"text": text}, nil)
	case "select":
		text, err := r.value(st.Value)
		if err != nil {
			return "", err
		}
		f, err := r.element(st.Target, "field", true)
		if err != nil {
			return "", err
		}
		if f.Tag != "select" { // a listbox: open it, then click the option
			if err := r.click(f); err != nil {
				return "", err
			}
			o, err := r.element(Locator{Name: text}, "click", true)
			if err != nil {
				return "", err
			}
			return "", r.click(o)
		}
		arg, _ := json.Marshal(text)
		v, err := r.c.Eval(`(function(e,t){var o=[].find.call(e.options,function(o){return o.text.trim()===t||o.value===t||o.label===t});
if(!o)return [].map.call(e.options,function(o){return o.text.trim()});e.value=o.value;e.dispatchEvent(new Event('input',{bubbles:true}));e.dispatchEvent(new Event('change',{bubbles:true}));return true})(window.__siT,` + string(arg) + `)`)
		if err != nil {
			return "", err
		}
		if string(v) != "true" {
			var opts []string
			json.Unmarshal(v, &opts)
			return "", &failure{expected: "an option " + st.Value.String() + " in " + f.Desc, found: "no such option", candidates: opts}
		}
		return "", nil
	case "check", "uncheck":
		want := st.Kind == "check"
		f, err := r.element(st.Target, "field", true)
		if err != nil {
			return "", err
		}
		if f.Checked != want {
			if err := r.click(f); err != nil {
				return "", err
			}
		}
		return "", r.until(func() (bool, error) {
			v, err := r.c.Eval(`(function(e){return !!(e.checked||e.getAttribute('aria-checked')==='true')})(window.__siT)`)
			if err == nil && (string(v) == "true") == want {
				return true, nil
			}
			return false, &failure{expected: f.Desc + " " + st.Kind + "ed", found: "it did not change"}
		})
	case "press":
		ch, _ := keyChord(st.Value.Text)
		if err := r.press(ch); err != nil {
			return "", err
		}
		return "", r.loaded()
	case "wait":
		time.Sleep(st.Wait)
		return "", nil
	case "screenshot":
		png, err := r.c.Screenshot()
		if err != nil {
			return "", err
		}
		return r.save(r.currentFeature(st), st.Value.Text, png), nil
	case "expect-text":
		want, err := r.value(st.Value)
		if err != nil {
			return "", err
		}
		arg, _ := json.Marshal(want)
		return "", r.until(func() (bool, error) {
			v, err := r.c.Eval(`(function(t){var b=document.body?document.body.innerText.replace(/\s+/g,' '):'';return b.indexOf(t.replace(/\s+/g,' '))>=0?'':b.slice(0,300)})(` + string(arg) + `)`)
			var page string
			json.Unmarshal(v, &page)
			if err == nil && string(v) == `""` {
				return true, nil
			}
			return false, &failure{expected: "the page shows " + st.Value.String(), found: "the page shows: " + strings.TrimSpace(page)}
		})
	case "expect-url":
		want := st.Value.Text
		return "", r.until(func() (bool, error) {
			v, err := r.c.Eval("location.href")
			var href string
			json.Unmarshal(v, &href)
			if err == nil && urlMatches(href, want) {
				return true, nil
			}
			return false, &failure{expected: "the URL is " + want, found: href}
		})
	case "expect-state":
		return "", r.until(func() (bool, error) {
			f, err := r.locate(st.Target, "any", false)
			if err != nil {
				return false, &failure{expected: st.Target.String() + " " + st.State, found: err.Error()}
			}
			got := ""
			switch {
			case !f.Found:
				got = "nothing matches"
			case !f.Visible:
				got = f.Desc + ", hidden"
			}
			ok := false
			switch st.State {
			case "visible":
				ok = f.Found && f.Visible
			case "hidden":
				ok = !f.Found || !f.Visible
				if f.Found && f.Visible {
					got = f.Desc + ", visible"
				}
			case "disabled", "enabled":
				ok = f.Found && f.Enabled == (st.State == "enabled")
				if f.Found {
					got = f.Desc + map[bool]string{true: ", enabled", false: ", disabled"}[f.Enabled]
				}
			case "checked", "unchecked":
				ok = f.Found && f.Checked == (st.State == "checked")
				if f.Found {
					got = f.Desc + map[bool]string{true: ", checked", false: ", unchecked"}[f.Checked]
				}
			}
			if ok {
				return true, nil
			}
			return false, &failure{expected: st.Target.String() + " " + st.State, found: got, candidates: f.Candidates}
		})
	case "expect-errors":
		r.mu.Lock()
		all := append(append([]string(nil), r.errs...), r.reqs...)
		r.mu.Unlock()
		if len(all) > 0 {
			return "", &failure{expected: "no console errors and no failed requests", found: strings.Join(all, "; ")}
		}
		return "", nil
	}
	return "", fmt.Errorf("step %q is not known to the runner", st.Kind)
}

// currentFeature names a screenshot after the feature holding st.
func (r *runner) currentFeature(st *Step) string {
	for _, f := range r.s.Features {
		for _, b := range f.Blocks {
			for _, s := range b.Steps {
				if s == st {
					return f.Name
				}
			}
		}
	}
	return r.s.Name
}

// urlMatches: a /path matches the page's path (with its query and hash), a
// full URL the whole address; either may be followed by ?… or #….
func urlMatches(href, want string) bool {
	got := href
	if strings.HasPrefix(want, "/") {
		u, err := url.Parse(href)
		if err != nil {
			return false
		}
		got = u.RequestURI()
		if u.Fragment != "" {
			got += "#" + u.Fragment
		}
	}
	return got == want || strings.HasPrefix(got, want+"?") || strings.HasPrefix(got, want+"#")
}

// press sends a key with its modifiers.
func (r *runner) press(ch Chord) error {
	k, named := namedKey(ch.Key)
	if !named {
		up := strings.ToUpper(ch.Key)
		k = keyDef{key: ch.Key, text: ch.Key, vk: int(up[0])}
		if c := up[0]; c >= 'A' && c <= 'Z' {
			k.code = "Key" + up
		} else if c >= '0' && c <= '9' {
			k.code = "Digit" + up
		}
	}
	text := k.text
	if ch.Mods&(1|2|4) != 0 {
		text = "" // a shortcut types nothing
	}
	down := map[string]any{"type": "rawKeyDown", "key": k.key, "code": k.code, "windowsVirtualKeyCode": k.vk, "modifiers": ch.Mods}
	if text != "" {
		down["type"], down["text"] = "keyDown", text
	}
	if ch.Mods&4 != 0 && runtime.GOOS == "darwin" {
		// macOS Chrome runs editing shortcuts (cmd+a, cmd+c …) as commands
		cmds := map[string]string{"a": "selectAll", "c": "copy", "x": "cut", "v": "paste", "z": "undo"}
		if c, ok := cmds[strings.ToLower(ch.Key)]; ok {
			down["commands"] = []string{c}
		}
	}
	if err := r.c.Call("Input.dispatchKeyEvent", down, nil); err != nil {
		return err
	}
	return r.c.Call("Input.dispatchKeyEvent", map[string]any{"type": "keyUp", "key": k.key, "code": k.code, "windowsVirtualKeyCode": k.vk, "modifiers": ch.Mods}, nil)
}
