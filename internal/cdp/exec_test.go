package cdp

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestRunFreshProfileAndRetry: each run gets its own --user-data-dir (gone
// after), a Chrome that exits with an error on start gets one more try, and
// a run killed by its timeout gets none.
func TestRunFreshProfileAndRetry(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sh script")
	}
	dir := t.TempDir()
	fake := filepath.Join(dir, "chrome")
	os.WriteFile(fake, []byte("#!/bin/sh\necho \"$1\" >> "+dir+"/args\nif [ ! -f "+dir+"/once ]; then touch "+dir+"/once; exit 2; fi\n[ \"$3\" = slow ] && sleep 20\necho ok\n"), 0o755)
	out, err := Output(time.Minute, fake, "x")
	if err != nil || strings.TrimSpace(string(out)) != "ok" {
		t.Fatalf("one retry after a crash: %q %v", out, err)
	}
	args, _ := os.ReadFile(filepath.Join(dir, "args"))
	lines := strings.Fields(string(args))
	if len(lines) != 2 || !strings.HasPrefix(lines[0], "--user-data-dir=") || lines[0] == lines[1] {
		t.Fatalf("a fresh profile per try: %q", lines)
	}
	if _, err := os.Stat(strings.TrimPrefix(lines[0], "--user-data-dir=")); !os.IsNotExist(err) {
		t.Error("the profile is removed after the run")
	}
	os.Remove(filepath.Join(dir, "args"))
	start := time.Now()
	if _, err := Run(300*time.Millisecond, fake, "x", "slow"); err == nil || time.Since(start) > 9*time.Second {
		t.Errorf("a timeout is an error, never retried: %v after %v", err, time.Since(start))
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "args")); strings.Count(string(b), "\n") != 1 {
		t.Errorf("tries after a timeout: %q", b)
	}
}

// TestRunTeardownCrash: a Chrome that writes its screenshot (or the whole
// DOM) and then dies on the way out ("Teardown watchdog expired") succeeded.
func TestRunTeardownCrash(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sh script")
	}
	dir := t.TempDir()
	fake := filepath.Join(dir, "chrome")
	shot := filepath.Join(dir, "shot.png")
	os.WriteFile(fake, []byte("#!/bin/sh\necho x >> "+dir+"/tries\ncase \"$2\" in --screenshot=*) echo png > "+shot+";; *) echo '<html></html>';; esac\necho 'Teardown watchdog expired' >&2\nexit 2\n"), 0o755)
	if _, err := Run(time.Minute, fake, "--screenshot="+shot); err != nil {
		t.Errorf("screenshot written, then a crash: %v", err)
	}
	if out, err := Output(time.Minute, fake, "--dump-dom"); err != nil || !strings.Contains(string(out), "</html>") {
		t.Errorf("the DOM out, then a crash: %q %v", out, err)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "tries")); strings.Count(string(b), "\n") != 2 {
		t.Errorf("no second try when the work is done: %q", b)
	}
}
