package deck

import (
	"strings"
	"testing"
)

func TestT2SKeys(t *testing.T) {
	for _, c := range []struct{ fm, problem string }{
		{"feature: t2s\nt2s-highlight: paragraph\nt2s-rate: 0.75\nt2s-voice: Samantha", ""},
		{"features: t2s", ""},
		{"feature: tts", `unknown feature "tts"`},
		{"t2s-highlight: line", `unknown t2s-highlight "line"`},
		{"t2s-rate: 3", "t2s-rate \"3\" is not a speed"},
	} {
		d, err := ParseSource("/tmp/p.md", "---\ntype: article\ntitle: P\n"+c.fm+"\n---\n\n# P\n", Meta{})
		if err != nil {
			t.Fatal(err)
		}
		got := ""
		for _, p := range d.Problems {
			got += p.Msg + ";"
		}
		if (c.problem == "") != (got == "") || !strings.Contains(got, c.problem) {
			t.Errorf("%q: problems %q, want %q", c.fm, got, c.problem)
		}
		if c.problem == "" && !d.Meta.HasFeature("t2s") {
			t.Errorf("%q: HasFeature(t2s) = false", c.fm)
		}
	}
	if (Meta{Feature: "t2s, none"}).HasFeature("t2s") {
		t.Error("none should turn t2s off")
	}
}
