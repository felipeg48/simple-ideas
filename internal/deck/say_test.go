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
				got = append(got, fmt.Sprintf("%s/%s/%s", p.Text, p.Speaker, map[bool]string{true: "k"}[p.Click]))
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
