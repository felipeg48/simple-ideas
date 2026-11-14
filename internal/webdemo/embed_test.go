package webdemo

import (
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
	if p.Cmd != "SHELL='/bin/bash' exec '/usr/local/bin/si' play '/a b/talk.md' --addr 127.0.0.1:7411" && p.Cmd != "SHELL=/bin/bash exec /usr/local/bin/si play '/a b/talk.md' --addr 127.0.0.1:7411" {
		t.Errorf("cmd = %s", p.Cmd)
	}
	if p.Ready.Port != 7411 || p.Within != time.Minute || p.Name != "si-play" {
		t.Errorf("program = %+v", p)
	}
	if !deckApp("../m/talk.md") || deckApp("http://x/talk.md") || deckApp("app") {
		t.Error("deckApp: a .md path only")
	}
}
