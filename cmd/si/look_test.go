package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felipeg48/simple-ideas/internal/ui"
)

// TestViewPrecedence: --view, then SI_VIEW, then the project's si.conf, then
// the user's, then ledger.
func TestViewPrecedence(t *testing.T) {
	home := t.TempDir()
	t.Setenv("SI_HOME", home)
	proj, bare := t.TempDir(), t.TempDir()
	os.WriteFile(filepath.Join(proj, "si.conf"), []byte("ui.view: steps\n"), 0o644)
	for _, tc := range []struct {
		name, flag, env, target, user string
		want                          ui.View
	}{
		{"default", "", "", bare, "", ui.Ledger},
		{"user conf", "", "", bare, "badges", ui.Badges},
		{"project over user", "", "", proj, "badges", ui.Steps},
		{"env over conf", "", "checklist", proj, "badges", ui.Checklist},
		{"flag over env", "badges", "checklist", proj, "", ui.Badges},
		{"a file's folder", "", "", filepath.Join(proj, "talk.md"), "", ui.Steps},
	} {
		os.WriteFile(filepath.Join(home, "si.conf"), []byte("ui.view: "+tc.user+"\n"), 0o644)
		if tc.user == "" {
			os.Remove(filepath.Join(home, "si.conf"))
		}
		if tc.name == "a file's folder" {
			os.WriteFile(tc.target, []byte("# t\n"), 0o644)
		}
		got, err := resolveView(tc.flag, tc.env, tc.target, "")
		if err != nil || got != tc.want {
			t.Errorf("%s: %q, %v; want %q", tc.name, got, err, tc.want)
		}
	}
	for _, bad := range [][2]string{{"fancy", ""}, {"", "fancy"}} {
		if _, err := resolveView(bad[0], bad[1], bare, ""); err == nil || !strings.Contains(err.Error(), "ledger, badges, checklist or steps") {
			t.Errorf("%v: %v", bad, err)
		}
	}
}

func TestViewFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"check", "../../examples/showcase.md", "--view", "rainbow"}, &stdout, &stderr); code != exitUsage {
		t.Fatalf("a bad --view: exit %d", code)
	}
	if !strings.Contains(stderr.String(), "ledger, badges, checklist or steps") {
		t.Errorf("the error does not name the looks: %s", stderr.String())
	}
	// not a terminal: the plain lines, whatever the look
	var plain, steps bytes.Buffer
	run([]string{"check", "../../examples/showcase.md"}, &plain, &stderr)
	run([]string{"check", "../../examples/showcase.md", "--view", "steps"}, &steps, &stderr)
	if plain.String() != steps.String() || strings.Contains(steps.String(), "\x1b") {
		t.Errorf("piped output changed with --view:\n%q\n%q", plain.String(), steps.String())
	}
}
