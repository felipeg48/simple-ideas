package webdemo

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestEmbedStepParse(t *testing.T) {
	cases := []struct {
		in   string
		want Step // Kind "" = a parse error
	}{
		{`embed-terminal "Lab 1"`, Step{Kind: "embed-terminal", Name: "Lab 1", Opts: map[string]string{}}},
		{`embed-terminal "Lab 2" · split right`, Step{Kind: "embed-terminal", Name: "Lab 2", Value: Value{Text: "right"}, Opts: map[string]string{}}},
		{`embed-terminal "Lab 2" · split down`, Step{Kind: "embed-terminal", Name: "Lab 2", Value: Value{Text: "down"}, Opts: map[string]string{}}},
		{`embed-terminal "Ops" · opacity 80%`, Step{Kind: "embed-terminal", Name: "Ops", Opts: map[string]string{"opacity": "80"}}},
		{`embed-terminal "Ops" · split right · opacity 70%`, Step{Kind: "embed-terminal", Name: "Ops", Value: Value{Text: "right"}, Opts: map[string]string{"opacity": "70"}}},
		{`embed-terminal "Ops" · opacity 30%`, Step{}},
		{`embed-terminal close`, Step{Kind: "embed-terminal", State: "close"}},
		{`embed-terminal min`, Step{Kind: "embed-terminal", State: "min"}},
		{`embed-terminal max`, Step{Kind: "embed-terminal", State: "max"}},
		{`embed-terminal "Lab" · split left`, Step{}},
		{`embed-terminal "Lab" close`, Step{}},
		{`embed-terminal Lab`, Step{}},
		{`embed-browser "RabbitMQ"`, Step{Kind: "embed-browser", Name: "RabbitMQ", Opts: map[string]string{}}},
		{`embed-browser "http://localhost:15701/#/queues"`, Step{Kind: "embed-browser", Name: "http://localhost:15701/#/queues", Opts: map[string]string{}}},
		{`embed-browser "RabbitMQ" · at 80,60 · size 1100x700 · zoom 125%`, Step{Kind: "embed-browser", Name: "RabbitMQ",
			Opts: map[string]string{"x": "80", "y": "60", "w": "1100", "h": "700", "zoom": "125"}}},
		{`embed-browser "RabbitMQ" close`, Step{Kind: "embed-browser", Name: "RabbitMQ", State: "close"}},
		{`embed-browser "RabbitMQ" min`, Step{Kind: "embed-browser", Name: "RabbitMQ", State: "min"}},
		{`embed-browser "RabbitMQ" · zoom 900%`, Step{}},
		{`embed-browser "RabbitMQ" · at here`, Step{}},
		{`embed-browser close`, Step{}},
		{`slide 4`, Step{Kind: "slide", Code: 4}},
		{`slide 0`, Step{}},
		{`next`, Step{Kind: "next"}},
		{`prev`, Step{Kind: "prev"}},
		{`next 2`, Step{}},
		{`click "Queues" in embed-browser "RabbitMQ"`, Step{Kind: "click", Target: Locator{Name: "Queues", Frame: `iframe[data-si-window="RabbitMQ"]`}}},
		{`click .nav a in embed-browser "Rabbit MQ"`, Step{Kind: "click", Target: Locator{CSS: ".nav a", Frame: `iframe[data-si-window="Rabbit MQ"]`}}},
		{`expect "orders" visible in embed-browser "RabbitMQ"`, Step{Kind: "expect-state", State: "visible", Target: Locator{Name: "orders", Frame: `iframe[data-si-window="RabbitMQ"]`}}},
	}
	for _, tc := range cases {
		st, err := ParseStep(tc.in)
		if tc.want.Kind == "" {
			if err == nil {
				t.Errorf("%s: want an error, got %+v", tc.in, st)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: %v", tc.in, err)
			continue
		}
		st.Text, tc.want.Text = "", ""
		if !reflect.DeepEqual(*st, tc.want) {
			t.Errorf("%s\n got %+v\nwant %+v", tc.in, *st, tc.want)
		}
	}
}

func TestEmbedScenes(t *testing.T) {
	for _, tc := range []struct{ app, steps, want string }{
		{"talk.md", "- embed-terminal \"Lab\"\n- run \"make up\"\n- expect output \"up\"\n- press ctrl+c\n- type \"ls\"\n- embed-browser \"Console\"\n- slide 3", ""},
		{"http://127.0.0.1:7401/talk.html", "- next\n- prev", ""},
		{"none", "- terminal \"T\"\n- slide 2", "app: none has no si play page"},
		{"app", "- embed-terminal \"Lab\"", "not a folder"},
		{"talk.md", "- run \"ls\"", "needs a terminal"},
		{"talk.md", "- embed-terminal \"Lab\"\n- embed-terminal min\n- run \"ls\"", "needs a terminal"},
		{"talk.md", "- embed-terminal \"Lab\"\n- run \"make\" · wait", "follow the run with wait for output"},
		{"talk.md", "- embed-terminal \"Lab\"\n- expect no output \"ERR\"", "not embed-terminal"},
		{"talk.md", "- embed-terminal \"Lab\"\n- press cmd+k", "a terminal takes"},
		{"talk.md", "- terminal \"T\"\n- slide 2", "terminal \"T\" is in front"},
	} {
		s := Parse(filepath.Join(t.TempDir(), "d.md"), "---\ntype: demo\napp: "+tc.app+"\n---\n\n## F\n\n"+tc.steps+"\n")
		var got []string
		for _, p := range s.Problems {
			got = append(got, p.Msg)
		}
		all := strings.Join(got, "\n")
		if (tc.want == "") != (all == "") || !strings.Contains(all, tc.want) {
			t.Errorf("app %s, %q:\n got %q\nwant %q", tc.app, tc.steps, all, tc.want)
		}
	}
}

func TestSinceRun(t *testing.T) {
	before := "$ echo old\nold\n$"
	out := before + " echo sent 3 && echo done\nsent 3\ndone\n$"
	got := sinceRun(out, [2]string{before, "echo sent 3 && echo done"})
	if strings.Contains(got, "echo") || !strings.Contains(got, "sent 3") {
		t.Errorf("sinceRun = %q", got)
	}
	if sinceRun("abc", [2]string{}) != "abc" {
		t.Error("no run: all of it")
	}
}

func TestPlayProgram(t *testing.T) {
	p := playProgram("/usr/local/bin/si", "/a b/talk.md", 7411, "/bin/bash")
	if p.Cmd != "SHELL='/bin/bash' SI_TERM_PROMPT=clean exec '/usr/local/bin/si' play '/a b/talk.md' --addr 127.0.0.1:7411" && p.Cmd != "SHELL=/bin/bash SI_TERM_PROMPT=clean exec /usr/local/bin/si play '/a b/talk.md' --addr 127.0.0.1:7411" {
		t.Errorf("cmd = %s", p.Cmd)
	}
	if p.Ready.Port != 7411 || !p.Ready.Gate || p.Within != 10*time.Minute || p.Name != "si-play" {
		t.Errorf("program = %+v", p)
	}
	if !deckApp("../m/talk.md") || deckApp("http://x/talk.md") || deckApp("app") {
		t.Error("deckApp: a .md path only")
	}
}

// fakeGate serves si play's /_si/ready: state gives ready and failed.
func fakeGate(t *testing.T, state func() (bool, string)) int {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/_si/ready" {
			http.NotFound(w, r)
			return
		}
		ready, failed := state()
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"ready":%t,"failed":%q,"steps":[]}`, ready, failed)
	}))
	t.Cleanup(srv.Close)
	return srv.Listener.Addr().(*net.TCPAddr).Port
}

// TestDeckGate (#18): with app: a deck, si play is ready once it serves the
// talk, past its getting-ready page, so the demo's prepare: runs after the
// deck's setup:, never beside it; a failed setup: fails the run with its
// error; a deck with no setup: (no gate) is ready at once.
func TestDeckGate(t *testing.T) {
	noWindows(t)
	dir := t.TempDir()
	marker := filepath.Join(dir, "set-up")
	polls := 0
	port := fakeGate(t, func() (bool, string) { // the deck's setup: takes three polls, then leaves its mark
		polls++
		if polls < 3 {
			return false, ""
		}
		os.WriteFile(marker, nil, 0o644)
		return true, ""
	})
	path := filepath.Join(dir, "d.md")
	os.WriteFile(path, []byte("---\ntype: demo\napp: none\nprepare: test -f set-up\n---\n## F\n- terminal \"T\"\n"), 0o644)
	s, _ := ParseFile(path)
	play := &Program{Name: "si-play", Cmd: "sleep 30", Ready: Ready{Port: port, Gate: true}, Within: 10 * time.Second, Text: "si play talk.md (app:)"}
	s.Prelude = append([]Prep{{Start: play}}, s.Prelude...) // as playDeck does
	r := &runner{s: s, o: Options{Log: io.Discard}, procs: map[string]*proc{}, terms: map[string]*term{}, abort: make(chan struct{})}
	if f := r.prelude(); f != nil {
		t.Errorf("prepare: after the deck's setup: %+v", f)
	}
	r.shutdown()

	same := func(s string) string { return s }
	p, err := startProc("si-play", "sleep 30", dir, "", same)
	if err != nil {
		t.Fatal(err)
	}
	defer p.stop()
	failed := fakeGate(t, func() (bool, string) { return false, "setup: docker compose up: exit status 1" })
	if err := p.ready(Ready{Port: failed, Gate: true}, 5*time.Second); err == nil || !strings.Contains(err.(*failure).found, "getting ready failed: setup: docker compose up") {
		t.Errorf("a failed setup: %v", err)
	}
	talk := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "<!doctype html><title>talk</title>") }))
	defer talk.Close()
	if err := p.ready(Ready{Port: talk.Listener.Addr().(*net.TCPAddr).Port, Gate: true}, 5*time.Second); err != nil {
		t.Errorf("no gate: %v", err)
	}
	// si play exits on a failed setup: its last words say why
	q, err := startProc("si-play", "echo 'si play: getting ready failed: setup: make: exit status 2'; exit 1", dir, "", same)
	if err != nil {
		t.Fatal(err)
	}
	if err := q.ready(Ready{Port: 1, Gate: true}, 5*time.Second); err == nil || err.(*failure).found != "getting ready failed: setup: make: exit status 2" {
		t.Errorf("si play exited: %v", err)
	}
}
