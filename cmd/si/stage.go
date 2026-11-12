package main

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/felipeg48/simple-ideas/internal/pinned"
	"github.com/felipeg48/simple-ideas/internal/webdemo"
)

// deckStages are the setup:, start:, stop: and cleanup: lines of what si
// play or si serve serves (a document, or a folder's top-level documents;
// web demo scripts have their own), their remote sources fetched and
// trusted. None: nothing to set up.
func deckStages(src string, fetcher func() *pinned.Client) ([]*webdemo.Script, error) {
	if deck.IsGuideDir(src) {
		return nil, nil
	}
	files, err := deck.List(src)
	if err != nil {
		return nil, nil // si play reports a bad path itself
	}
	var out []*webdemo.Script
	for _, f := range files {
		d, err := deck.Parse(f)
		if err != nil || len(d.Meta.Lifecycle) == 0 || (d.IsDemo() && d.Meta.App != "") {
			continue
		}
		s := webdemo.ParseStage(f, d.Meta.Lifecycle)
		if len(s.Problems) > 0 {
			return nil, fmt.Errorf("%s", s.Problems[0])
		}
		if err := fetchSources(s, fetcher); err != nil {
			return nil, fmt.Errorf("%s: %v", f, err)
		}
		if !s.Empty() {
			out = append(out, s)
		}
	}
	return out, nil
}

// stageProblems are a deck's setup:, start:, stop: and cleanup: problems (si
// check): a bad line, a source without its pin.
func stageProblems(f string, d *deck.Deck) []webdemo.Problem {
	if len(d.Meta.Lifecycle) == 0 || (d.IsDemo() && d.Meta.App != "") {
		return nil
	}
	return webdemo.ParseStage(f, d.Meta.Lifecycle).Problems
}

// readyGate serves a "getting ready" page, with each setup step's progress,
// until the decks' setup: and start: lines have run; then the talk.
type readyGate struct {
	next  http.Handler
	mu    sync.Mutex
	open  bool
	fail  string
	steps []gateStep
}

type gateStep struct {
	Text    string  `json:"text"`
	Status  string  `json:"status"` // waiting, running, passed, failed
	Seconds float64 `json:"seconds"`
}

func newGate(next http.Handler, stages []*webdemo.Script) *readyGate {
	g := &readyGate{next: next}
	for _, s := range stages {
		for _, p := range s.Prelude {
			t := ""
			if p.Start != nil {
				t = "start: " + p.Start.Text
			} else {
				t = p.Cmd.Key + ": " + p.Cmd.Text
			}
			g.steps = append(g.steps, gateStep{Text: t, Status: "waiting"})
		}
	}
	if len(g.steps) > 0 {
		g.steps[0].Status = "running"
	}
	return g
}

// done marks the next step with its result and the one after it running.
func (g *readyGate) done(sr webdemo.StepResult) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for i := range g.steps {
		if g.steps[i].Status == "running" || g.steps[i].Status == "waiting" {
			g.steps[i].Status, g.steps[i].Seconds = sr.Status, sr.Seconds
			if i+1 < len(g.steps) && sr.Status == "passed" {
				g.steps[i+1].Status = "running"
			}
			return
		}
	}
}

func (g *readyGate) ready() {
	g.mu.Lock()
	g.open = true
	g.mu.Unlock()
}

func (g *readyGate) failed(msg string) {
	g.mu.Lock()
	g.fail = msg
	g.mu.Unlock()
}

func (g *readyGate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	g.mu.Lock()
	open := g.open
	state := struct {
		Ready bool       `json:"ready"`
		Fail  string     `json:"failed,omitempty"`
		Steps []gateStep `json:"steps"`
	}{g.open, g.fail, append([]gateStep(nil), g.steps...)}
	g.mu.Unlock()
	if r.URL.Path == "/_si/ready" {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		json.NewEncoder(w).Encode(state)
		return
	}
	if open {
		g.next.ServeHTTP(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusServiceUnavailable)
	var b strings.Builder
	for _, s := range state.Steps {
		fmt.Fprintf(&b, `<li class="%s"><span></span><code>%s</code></li>`, s.Status, html.EscapeString(s.Text))
	}
	io.WriteString(w, strings.Replace(readyPage, "<!--steps-->", b.String(), 1))
}

// readyPage is the getting-ready page: the steps, polled from /_si/ready;
// it loads the talk once they are done.
const readyPage = `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>Getting ready…</title>
<style>
:root{--bg:#f6f5f1;--fg:#1d1f24;--muted:#5d6370;--ok:#1f7a4d;--bad:#b3261e;--run:#1f4fa3;color-scheme:light dark}
@media (prefers-color-scheme:dark){:root{--bg:#15171c;--fg:#e8e8e3;--muted:#a3a8b3;--ok:#5fcf95;--bad:#ff8a80;--run:#8ab4ff}}
body{margin:0;min-height:100vh;display:grid;place-items:center;background:var(--bg);color:var(--fg);font:clamp(1rem,.9rem + .4vw,1.25rem)/1.5 system-ui,-apple-system,"Segoe UI",sans-serif;padding:1rem;box-sizing:border-box}
main{width:min(40rem,100%)}h1{font-size:clamp(1.6rem,1.2rem + 2vw,2.6rem);margin:0 0 .2em}p{color:var(--muted);margin:0 0 1.2em}
ol{list-style:none;padding:0;margin:0}li{display:flex;gap:.7em;align-items:baseline;padding:.4em 0;border-top:1px solid color-mix(in srgb,var(--fg) 15%,transparent)}
li span{flex:none;width:1.2em;font-weight:700}li.waiting{color:var(--muted)}li.waiting span::before{content:"·"}
li.running span::before{content:"◌";color:var(--run)}li.passed span::before{content:"✓";color:var(--ok)}li.failed span::before{content:"✗";color:var(--bad)}
code{font:.9em ui-monospace,SFMono-Regular,Menlo,monospace;overflow-wrap:anywhere}pre{white-space:pre-wrap;color:var(--bad);font:.85em ui-monospace,Menlo,monospace}
</style></head><body><main><h1>Getting ready…</h1><p>The talk sets up what it shows live; it opens when this is done.</p>
<ol id="steps"><!--steps--></ol><pre id="fail"></pre></main>
<script>
(function(){var t=setInterval(function(){fetch('/_si/ready',{cache:'no-store'}).then(function(r){return r.json()}).then(function(s){
if(s.ready){clearInterval(t);location.reload();return}
var ol=document.getElementById('steps');ol.innerHTML='';s.steps.forEach(function(x){var li=document.createElement('li');li.className=x.status;
li.appendChild(document.createElement('span'));var c=document.createElement('code');c.textContent=x.text;li.appendChild(c);ol.appendChild(li)});
if(s.failed){clearInterval(t);document.getElementById('fail').textContent=s.failed}
}).catch(function(){clearInterval(t);document.getElementById('fail').textContent='si play has stopped.'})},500)})();
</script></body></html>`

// runStages runs every stage's setup: and start: lines (in the background:
// the gate shows them meanwhile); on a failure it calls fail. The returned
// func runs stop:, stops the programs and runs cleanup:, last stage first.
func runStages(stages []*webdemo.Script, g *readyGate, stdout io.Writer, fail func()) func() {
	logs, _ := os.MkdirTemp("", "si-play-run-")
	os.Setenv("SI_RUN_LOGS", logs) // the terminal drawer's shells see the programs' logs
	var sts []*webdemo.Stage
	var mu sync.Mutex
	for _, s := range stages {
		sts = append(sts, webdemo.NewStage(s, logs, stdout, g.done))
	}
	go func() {
		for _, st := range sts {
			mu.Lock()
			err := st.Up()
			mu.Unlock()
			if err != nil {
				fmt.Fprintf(stdout, "si play: getting ready failed: %v\n", err)
				g.failed(err.Error())
				fail()
				return
			}
		}
		for _, st := range sts {
			for name, log := range st.Programs() {
				fmt.Fprintf(stdout, "  %s is running (log: %s)\n", name, log)
			}
		}
		fmt.Fprintf(stdout, "si play: ready (programs' logs in $SI_RUN_LOGS of the terminal drawer: %s)\n", logs)
		g.ready()
	}()
	var once sync.Once
	return func() {
		once.Do(func() {
			for i := len(sts) - 1; i >= 0; i-- {
				sts[i].Down()
			}
		})
	}
}

// deckTarget is what si play, build and serve take: a <dir|file.md>, or
// github:owner/repo@REF/path and a pinned URL, fetched into the cache and
// trusted once per pin. "" (and the exit code) when it cannot be had.
func deckTarget(t string, o *options, stderr io.Writer) (string, int) {
	local, _, err := fetchTarget(t, func() *pinned.Client { return newFetcher(o.trust, stderr) }, func(p string) []string {
		fs, _ := deck.List(p)
		return fs
	})
	if err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return "", exitProblems
	}
	return local, exitOK
}
