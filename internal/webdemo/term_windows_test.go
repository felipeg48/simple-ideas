//go:build windows

package webdemo

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/felipeg48/simple-ideas/internal/tools"
)

// TestTerminalWindowsShells: PowerShell runs under ConPTY and its prompt
// reports each command's exit status (in the title mark); cmd runs on pipes
// and si asks for the status; both show the folder's name and $.
func TestTerminalWindowsShells(t *testing.T) {
	for _, sh := range []string{"pwsh", "powershell", "cmd"} {
		path, err := exec.LookPath(sh)
		if err != nil {
			continue
		}
		t.Run(sh, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "orders")
			os.Mkdir(dir, 0o755)
			tm, err := openShell("T", path, dir, t.TempDir(), nil)
			if err != nil {
				t.Fatal(err)
			}
			defer tm.close()
			if tm.pipes != (sh == "cmd") {
				t.Fatalf("pipes = %v for %s", tm.pipes, sh)
			}
			if err := tm.prompt(60 * time.Second); err != nil {
				t.Fatal(err)
			}
			at, n := tm.out.mark()
			tm.runAt, tm.ran = at, true
			r := &runner{}
			line := `cmd /c "echo hi there& exit 3"`
			if sh == "cmd" {
				line = `echo hi there& cmd /c exit 3`
			}
			if err := r.typeIn(tm, line); err != nil {
				t.Fatal(err)
			}
			if tm.pipes {
				tm.send("echo \x1b]133;D;%errorlevel%\x07\r\n")
			}
			for i := 0; i < 400; i++ {
				if _, ok := tm.out.code(n); ok {
					break
				}
				time.Sleep(50 * time.Millisecond)
			}
			if c, ok := tm.out.code(n); !ok || c != 3 {
				t.Fatalf("exit code %d %v\n%q", c, ok, tm.out.text(0))
			}
			if s := tm.since(); !strings.Contains(s, "hi there") {
				t.Errorf("since the run: %q", s)
			}
			if s := tm.out.text(0); !strings.Contains(s, "orders $") {
				t.Errorf("no clean prompt: %q", s)
			}
		})
	}
}

// TestTerminalSplitWindows: a web demo's split terminal under ConPTY (and
// cmd on pipes): terminal split right and down, run … in 2, expect output …
// in 1 and in 2, focus, each pane its own shell.
func TestTerminalSplitWindows(t *testing.T) {
	chrome := tools.Find("chrome")
	if chrome == "" || testing.Short() {
		t.Skip("needs Chrome")
	}
	for _, sh := range []string{"pwsh", "powershell", "cmd"} {
		if _, err := exec.LookPath(sh); err != nil {
			continue
		}
		t.Run(sh, func(t *testing.T) {
			dir := t.TempDir()
			src := "---\ntype: demo\napp: none\nshell: " + sh + "\n---\n" +
				"## Right\n- terminal \"T\"\n- run \"echo left-side\" · wait\n- terminal split right\n- run \"echo right-side\" · wait\n" +
				"- expect output \"right-side\"\n- expect output \"left-side\" in 1\n- run \"echo again-left\" in 1 · wait\n" +
				"- expect output \"again-left\" in 1\n- focus 1\n- expect output \"again-left\"\n- expect no output \"right-side\"\n\n" +
				"## Down\n- terminal \"D\"\n- run \"echo top-pane\" · wait\n- terminal split down\n- run \"echo bottom-pane\" in 2 · wait\n" +
				"- expect output \"bottom-pane\" in 2\n- expect output \"top-pane\" in 1\n- expect no output \"bottom-pane\" in 1\n"
			path := filepath.Join(dir, "split.md")
			os.WriteFile(path, []byte(src), 0o644)
			s, err := ParseFile(path)
			if err != nil || len(s.Problems) > 0 {
				t.Fatal(err, s.Problems)
			}
			var log strings.Builder
			rep, err := Run(s, Options{Chrome: chrome, Log: &log})
			if err != nil || !rep.OK() {
				msg := ""
				if rep != nil {
					for _, f := range rep.Features {
						for _, st := range f.Steps {
							msg += fmt.Sprintf("%s: %s %s %s %s\n", f.Name, st.Status, st.Expected, st.Found, st.Output)
						}
					}
				}
				t.Errorf("%v\n%s\n%s", err, msg, log.String())
			}
		})
	}
}
