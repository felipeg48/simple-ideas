//go:build darwin

package main

import (
	"net/http"
	"net/http/cookiejar"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestStartAndStop runs the real si --web the way the app does (no window).
func TestStartAndStop(t *testing.T) {
	bin := t.TempDir()
	build := exec.Command("go", "build", "-o", filepath.Join(bin, "si"), "./cmd/si")
	build.Dir = ".."
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build si: %v\n%s", err, out)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "talk.md"), []byte("# Talk\n"), 0o644)
	url, err := start(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(url, "http://127.0.0.1:") || !strings.Contains(url, "?token=") {
		t.Fatalf("url %q", url)
	}
	jar, _ := cookiejar.New(nil)
	resp, err := (&http.Client{Jar: jar}).Get(url) // the token is swapped for a cookie and the Studio page served
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("GET studio: %d", resp.StatusCode)
	}
	goShutdown()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := http.Get(url); err != nil {
			return // stopped
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Error("si --web still answers after goShutdown")
}

func TestStartReportsAFailure(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // no si anywhere
	installed := installedSi
	installedSi = func() []string { return nil }
	defer func() { installedSi = installed }()
	if _, err := start(t.TempDir()); err == nil || !strings.Contains(err.Error(), "si binary") {
		t.Errorf("want a clear error without si: %v", err)
	}
}

func TestHelpJS(t *testing.T) {
	if got := helpJS("gallery"); got != `if(window.siStudio){siStudio.gallery()}else{location.hash="gallery"}` {
		t.Errorf("gallery: %s", got)
	}
	if !strings.Contains(helpJS("tour"), "siStudio.tour()") {
		t.Error("tour")
	}
	if got := helpJS("report"); got != `if(window.siStudio){siStudio.help("report",true)}` {
		t.Errorf("report: %s", got)
	}
	if !strings.Contains(helpJS("about"), "siStudio.about({") {
		t.Errorf("about: %s", helpJS("about"))
	}
	if helpJS("alert(1)") != "" {
		t.Error("only known commands")
	}
}
