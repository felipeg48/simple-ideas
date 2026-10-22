package config

import "strings"

import "testing"

func TestParse(t *testing.T) {
	src := `
# a comment
theme: crdb
keys.annotate: w
keys.timer: T
feature.annotate: off
feature.timer: on
keys.bogus: x
feature.nope: on
feature.timer: maybe
no-colon-line
`
	c := Parse(strings.NewReader(src))
	if c.Theme != "crdb" {
		t.Errorf("theme = %q, want crdb", c.Theme)
	}
	if c.Keys["annotate"] != "w" || c.Keys["timer"] != "T" {
		t.Errorf("keys = %v, want annotate=w timer=T", c.Keys)
	}
	if v, ok := c.Features["annotate"]; !ok || v {
		t.Errorf("feature.annotate = %v,%v, want false,true", v, ok)
	}
	if v, ok := c.Features["timer"]; !ok || !v {
		t.Errorf("feature.timer = %v,%v, want true,true", v, ok)
	}
	// unknown action, unknown feature, bad bool, and the no-colon line → 4 warnings
	if len(c.Warnings) != 4 {
		t.Errorf("warnings = %d (%v), want 4", len(c.Warnings), c.Warnings)
	}
	if c.Empty() {
		t.Error("config should not be Empty")
	}
}

func TestLoadMissing(t *testing.T) {
	c := Load("/no/such/file.conf", "")
	if !c.Empty() {
		t.Errorf("missing files should yield an empty config, got %+v", c)
	}
}
