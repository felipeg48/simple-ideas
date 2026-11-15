package deck

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func TestParts(t *testing.T) {
	tests := []struct {
		name, front, notes string
		want               string // text/speaker/click|…
	}{
		{"no notes", "", "", ""},
		{"one part", "", "Notes: (2 min) Hello **there**.", "Hello there.//"},
		{"cues split", "", "Notes: One. [CLICK] Two.\n[click] Three.", "One.//|Two.//k|Three.//k"},
		{"trailing cue", "", "Notes: One. [CLICK]", "One.//|//k"},
		{"leading cue", "", "Notes: [CLICK] After.", "//|After.//k"},
		{"markdown plain", "", "Notes: See [the docs](http://x) and `si play`, _really_.\n- a list item\n", "See the docs and si play, really. a list item//"},
		{"snake_case kept", "", "Notes: the max_step value", "the max_step value//"},
		{"two speakers", "speakers: Ana, Ravi\n", "Notes[Ana]: Hi. [CLICK] More.\n\nNotes[Ravi]: Mine.", "Hi./Ana/|More./Ana/k|Mine./Ravi/"},
		{"tool cues split, never said", "", "Notes: Let's see. [TERM Lab 1: docker compose up -d] Up.\n[BROWSER RabbitMQ] The queue. [CLICK]",
			"Let's see.//|Up.//+term:Lab 1:docker compose up -d|The queue.//+browser:RabbitMQ:|//k"},
		{"cue after a click", "", "Notes: One. [CLICK] [TERM close] Two.", "One.//|Two.//k+term::"},
		{"malformed cue dropped", "", "Notes: A [TERM] B", "A//|B//"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := write(t, map[string]string{"d.md": "---\n" + tc.front + "---\n# Talk\n\n---\n\n## Slide\n\nbody\n\n" + tc.notes + "\n"})
			d, err := Parse(filepath.Join(dir, "d.md"))
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			for _, p := range d.Parts()[1] {
				g := fmt.Sprintf("%s/%s/%s", p.Text, p.Speaker, map[bool]string{true: "k"}[p.Click])
				for _, a := range p.Acts {
					g += "+" + a.Tool + ":" + a.Name + ":" + a.Cmd
				}
				got = append(got, g)
			}
			if g := strings.Join(got, "|"); g != tc.want {
				t.Errorf("parts = %q, want %q", g, tc.want)
			}
		})
	}
}

func TestVoicesMeta(t *testing.T) {
	dir := write(t, map[string]string{"d.md": "---\nspeakers: Ana, Ravi\nvoices: Ana=af_heart, Ravi=kokoro:am_michael, Bob=x\n---\n# T\n\nNotes[Ana]: a\n\nNotes[Ravi]: b\n"})
	d, err := Parse(filepath.Join(dir, "d.md"))
	if err != nil {
		t.Fatal(err)
	}
	if m := d.Meta.VoiceMap(); m["Ana"] != "af_heart" || m["Ravi"] != "kokoro:am_michael" {
		t.Errorf("VoiceMap = %v", m)
	}
	if len(d.Warnings) != 1 || !strings.Contains(d.Warnings[0].Msg, "Bob") {
		t.Errorf("warnings = %v, want one about Bob", d.Warnings)
	}
	dir = write(t, map[string]string{"d.md": "---\nvoices: Ana\n---\n# T\n"})
	if d, _ := Parse(filepath.Join(dir, "d.md")); len(d.Problems) != 1 || !strings.Contains(d.Problems[0].Msg, "Name=voice") {
		t.Errorf("bad voices: problems = %v", d.Problems)
	}
}

func TestParseAct(t *testing.T) {
	for _, tc := range []struct{ tool, body, want, err string }{
		{"TERM", " Lab 1: echo hi", "term|Lab 1|echo hi|", ""},
		{"TERM", " Lab 1", "term|Lab 1||", ""},
		{"TERM", " max", "term|||max", ""},
		{"TERM", " http://x:1", "term|http://x:1||", ""},
		{"TERM", "", "", "needs a tab name"},
		{"TERM", " : ls", "", "needs a tab name"},
		{"TERM", " Lab:", "", "empty command"},
		{"BROWSER", " RabbitMQ", "browser|RabbitMQ||", ""},
		{"BROWSER", " http://localhost:15672/#/queues", "browser|http://localhost:15672/#/queues||", ""},
		{"BROWSER", " close", "browser|||close", ""},
		{"BROWSER", "", "", "needs a browser: link"},
	} {
		a, err := ParseAct(tc.tool, tc.body)
		if tc.err != "" {
			if err == nil || !strings.Contains(err.Error(), tc.err) {
				t.Errorf("%s%q: err = %v, want %q", tc.tool, tc.body, err, tc.err)
			}
			continue
		}
		if got := a.Tool + "|" + a.Name + "|" + a.Cmd + "|" + a.State; err != nil || got != tc.want {
			t.Errorf("%s%q = %q, %v; want %q", tc.tool, tc.body, got, err, tc.want)
		}
	}
	// a window's place, size and zoom; Name: URL takes the window Name there
	for _, tc := range []struct{ body, want, err string }{
		{" RabbitMQ · at 560,40 · size 1010x640 · zoom 125%", "RabbitMQ| map[h:640 w:1010 x:560 y:40 zoom:1.25]", ""},
		{" RabbitMQ: http://localhost:15701/#/queues", "RabbitMQ|http://localhost:15701/#/queues map[]", ""},
		{" RabbitMQ:http://localhost:15701/", "RabbitMQ|http://localhost:15701/ map[]", ""},
		{" RabbitMQ: #/queues · zoom 80%", "RabbitMQ|#/queues map[zoom:0.8]", ""},
		{" localhost:15672", "localhost:15672| map[]", ""},
		{" http://localhost:15672/ · at -10,0", "http://localhost:15672/| map[x:-10 y:0]", ""},
		{" RabbitMQ · zoom 500%", "", "30% to 300%"},
		{" RabbitMQ · size big", "", "· size WxH"},
		{" RabbitMQ · left", "", "· at X,Y"},
		{" close · at 1,2", "", "close takes no options"},
		{" RabbitMQ: ", "", "empty address"},
	} {
		a, err := ParseAct("BROWSER", tc.body)
		if tc.err != "" {
			if err == nil || !strings.Contains(err.Error(), tc.err) {
				t.Errorf("BROWSER%q: err = %v, want %q", tc.body, err, tc.err)
			}
			continue
		}
		if got := a.Name + "|" + a.URL + " " + fmt.Sprint(a.Place); err != nil || got != tc.want {
			t.Errorf("BROWSER%q = %q, %v; want %q", tc.body, got, err, tc.want)
		}
	}
	if acts, errs := Acts("x [TERM a: b] [BROWSER] [TERMINAL no] [CLICK]"); len(acts) != 1 || len(errs) != 1 {
		t.Errorf("Acts = %v, %v", acts, errs)
	}
	if p := Plain("Up [TERM a: b] now"); p != "Up now" {
		t.Errorf("Plain = %q", p)
	}
}
