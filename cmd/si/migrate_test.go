package main

import (
	"strings"
	"testing"
)

func TestMigratePrompt(t *testing.T) {
	p := migratePrompt("/courses/foo", "foo-si")
	for _, want := range []string{
		"/courses/foo", "foo-si",
		"EDITORIAL migration", "Prune", "SPLIT deck", "si check",
		"stage", "si rules",
	} {
		if !strings.Contains(p, want) {
			t.Errorf("prompt missing %q", want)
		}
	}
	if strings.Contains(p, "SRC") || strings.Contains(p, "DEST") {
		t.Error("placeholders not substituted")
	}
}
