//go:build windows

package webdemo

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
