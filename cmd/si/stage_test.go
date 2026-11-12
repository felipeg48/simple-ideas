//go:build !windows

package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// syncBuf is a buffer play may write while the test reads it.
type syncBuf struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuf) Write(p []byte) (int, error) { s.mu.Lock(); defer s.mu.Unlock(); return s.b.Write(p) }
func (s *syncBuf) String() string              { s.mu.Lock(); defer s.mu.Unlock(); return s.b.String() }

// TestPlayStage: si play runs a deck's setup: and start: before the talk (a
// getting-ready page meanwhile), keeps the program running while it
// serves, and runs stop:, stops the program and runs cleanup: when it quits
// (Ctrl-C); si check reports an unpinned source; si build runs nothing.
func TestPlayStage(t *testing.T) {
	if runtime.GOOS == "windows" || testing.Short() {
		t.Skip("sh scripts, a signal")
	}
	t.Setenv("SI_HOME", t.TempDir())
	dir := t.TempDir()
	log := filepath.Join(dir, "log")
	os.WriteFile(filepath.Join(dir, "rec"), []byte("#!/bin/sh\necho \"$@\" >> \""+log+"\"\n"), 0o755)
	os.WriteFile(filepath.Join(dir, "prog"), []byte("#!/bin/sh\ntrap 'echo stopped >> \""+log+"\"; exit 0' TERM\necho ready\nwhile :; do sleep 0.05; done\n"), 0o755)
	talk := filepath.Join(dir, "talk.md")
	os.WriteFile(talk, []byte("---\ntitle: Live\nsetup: ./rec install; sleep 1\nstart: ./prog as broker · wait for \"ready\"\nstop: ./rec stop\ncleanup: ./rec clean\n---\n# Live\n\nThe broker, live.\n"), 0o644)

	var so syncBuf
	var se bytes.Buffer
	done := make(chan int)
	go func() { done <- run([]string{"play", talk, "--addr", "127.0.0.1:0"}, &so, &se) }()
	var url string
	for i := 0; i < 100 && url == ""; i++ {
		time.Sleep(20 * time.Millisecond)
		if _, rest, ok := strings.Cut(so.String(), " at http://"); ok {
			url = "http://" + strings.Fields(rest)[0]
		}
	}
	if url == "" {
		t.Fatalf("no URL: %s %s", so.String(), se.String())
	}
	res, err := http.Get(url + "talk.html")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 503 || !strings.Contains(string(b), "Getting ready") || !strings.Contains(string(b), "setup: ./rec install; sleep 1") {
		t.Errorf("getting ready: %d %s", res.StatusCode, b)
	}
	var ready bool
	for i := 0; i < 200 && !ready; i++ {
		time.Sleep(25 * time.Millisecond)
		if r, err := http.Get(url + "_si/ready"); err == nil {
			var st struct{ Ready bool }
			json.NewDecoder(r.Body).Decode(&st)
			r.Body.Close()
			ready = st.Ready
		}
	}
	if !ready {
		t.Fatalf("never ready: %s", so.String())
	}
	if res, err := http.Get(url + "talk.html"); err != nil || res.StatusCode != 200 {
		t.Errorf("the talk: %v %v", res, err)
	} else {
		b, _ := io.ReadAll(res.Body)
		res.Body.Close()
		if !strings.Contains(string(b), "The broker, live.") {
			t.Error("the talk's page")
		}
	}
	if got := strings.Fields(readFile(log)); strings.Join(got, " ") != "install" {
		t.Errorf("while serving: %q", got)
	}
	syscall.Kill(os.Getpid(), syscall.SIGINT)
	select {
	case code := <-done:
		if code != exitOK {
			t.Errorf("exit %d: %s", code, se.String())
		}
	case <-time.After(20 * time.Second):
		t.Fatal("si play did not stop")
	}
	if got := strings.Join(strings.Fields(readFile(log)), " "); got != "install stop stopped clean" {
		t.Errorf("on quit: %q\n%s", got, so.String())
	}
	// si build runs nothing; si check reads the lines and their pins
	os.Remove(log)
	var bo, be bytes.Buffer
	if code := run([]string{"build", talk, "-o", t.TempDir()}, &bo, &be); code != exitOK || readFile(log) != "" {
		t.Errorf("build: %d %s %q", code, be.String(), readFile(log))
	}
	os.WriteFile(talk, []byte("---\ntitle: Live\nsetup: github:acme/ops/install.sh\n---\n# Live\n"), 0o644)
	be.Reset()
	if code := run([]string{"check", talk}, &bo, &be); code != exitProblems || !strings.Contains(be.String(), "talk.md:3: setup: github:acme/ops/install.sh is not pinned") {
		t.Errorf("check: %d %s", code, be.String())
	}
}

func readFile(p string) string {
	b, _ := os.ReadFile(p)
	return string(b)
}
