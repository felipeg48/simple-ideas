package config

import (
	"strings"
	"testing"
)

func TestDefaultParsesClean(t *testing.T) {
	c := Parse(strings.NewReader(Default))
	if len(c.Warnings) != 0 {
		t.Fatalf("Default should parse with no warnings, got %v", c.Warnings)
	}
	if c.Theme != "studio" {
		t.Errorf("Default theme = %q, want studio", c.Theme)
	}
	for _, a := range Actions {
		if c.Keys[a] == "" {
			t.Errorf("Default is missing keys.%s", a)
		}
	}
	for _, f := range Features {
		if !c.Features[f] {
			t.Errorf("Default feature.%s should be on", f)
		}
	}
}

func TestParseFlatAndWarnings(t *testing.T) {
	src := `
theme: crdb
keys.draw: w
feature.timer: off
keys.bogus: x
feature.nope: on
feature.timer: maybe
no-colon-line
`
	c := Parse(strings.NewReader(src))
	if c.Keys["draw"] != "w" {
		t.Errorf("keys.draw = %q, want w", c.Keys["draw"])
	}
	if v, ok := c.Features["timer"]; !ok || v {
		t.Errorf("feature.timer = %v,%v, want false,true", v, ok)
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
