//go:build darwin

// si Studio for macOS: a native window (Cocoa + WebKit, app_darwin.m) around
// the web Studio. Each folder you open starts the bundled `si --web` on a free
// loopback port; the window shows it, and quitting (or closing the window)
// stops it. Its own module, so the si module keeps no cgo and no new
// dependency.
package main

/*
#cgo CFLAGS: -x objective-c -fobjc-arc -mmacosx-version-min=13.0
#cgo LDFLAGS: -framework Cocoa -framework WebKit -mmacosx-version-min=13.0
#include <stdlib.h>
void siRun(void);
*/
import "C"

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
)

func init() { runtime.LockOSThread() } // Cocoa runs on the main thread

func main() { C.siRun() }

var (
	mu     sync.Mutex
	studio = map[string]*exec.Cmd{} // URL -> its si --web process
	urlRE  = regexp.MustCompile(`http://127\.0\.0\.1:\d+/\?token=[0-9a-f]+`)
)

//export goOpenFolder
func goOpenFolder(path *C.char) *C.char {
	url, err := start(C.GoString(path))
	if err != nil {
		return C.CString("error:" + err.Error())
	}
	return C.CString(url)
}

//export goCloseURL
func goCloseURL(url *C.char) {
	mu.Lock()
	defer mu.Unlock()
	if cmd := studio[C.GoString(url)]; cmd != nil {
		stop(cmd)
		delete(studio, C.GoString(url))
	}
}

//export goShutdown
func goShutdown() {
	mu.Lock()
	defer mu.Unlock()
	for u, cmd := range studio {
		stop(cmd)
		delete(studio, u)
	}
}

// start runs `si --web dir` on a free port and returns the Studio URL it prints.
func start(dir string) (string, error) {
	si, err := siPath()
	if err != nil {
		return "", err
	}
	cmd := exec.Command(si, "--web", dir, "--no-open", "--addr", "127.0.0.1:0")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "PATH="+loginPath())
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} // stop takes its shells with it
	out, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	cmd.Stderr = cmd.Stdout
	if err := cmd.Start(); err != nil {
		return "", err
	}
	found := make(chan string, 1)
	var tail []string
	go func() {
		sc := bufio.NewScanner(out)
		for sc.Scan() { // keep draining: a full pipe would block si
			l := sc.Text()
			if u := urlRE.FindString(l); u != "" {
				select {
				case found <- u:
				default:
				}
			} else if len(tail) < 20 {
				tail = append(tail, l)
			}
		}
		close(found)
	}()
	select {
	case u, ok := <-found:
		if !ok {
			cmd.Wait()
			return "", fmt.Errorf("si --web stopped: %s", strings.Join(tail, " "))
		}
		mu.Lock()
		studio[u] = cmd
		mu.Unlock()
		return u, nil
	case <-time.After(20 * time.Second):
		stop(cmd)
		return "", errors.New("si --web did not start within 20 seconds")
	}
}

func stop(cmd *exec.Cmd) {
	if cmd.Process != nil {
		syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
		go cmd.Wait()
	}
}

// siPath is the si binary inside the app bundle, unless an installed si is
// newer (see pickSi); siVersion and siNote describe the choice.
func siPath() (string, error) {
	bundled := ""
	if exe, err := os.Executable(); err == nil {
		p := filepath.Join(filepath.Dir(exe), "si")
		if _, err := os.Stat(p); err == nil {
			bundled = p
		}
	}
	p, v, note := pickSi(bundled, installedSi(), versionOf)
	if p == "" {
		return "", errors.New("the si binary is missing from the app bundle")
	}
	statusMu.Lock()
	siVersion, siNote = v, note
	statusMu.Unlock()
	return p, nil
}

var (
	statusMu          sync.Mutex
	siVersion, siNote string
)

// goStatus is the si in use and, at most once a day, whether a newer release
// is out: JSON {"si","path","note","update","url"} for the menu, About and banner.
//
//export goStatus
func goStatus() *C.char {
	p, _ := siPath()
	statusMu.Lock()
	s := map[string]string{"si": siVersion, "path": p, "note": siNote}
	statusMu.Unlock()
	if r := latestRelease(releaseCache(), time.Now(), fetchRelease); r.Tag != "" && newer(r.Tag, s["si"]) {
		s["update"], s["url"] = r.Tag, releasePage(r)
	}
	b, _ := json.Marshal(s)
	return C.CString(string(b))
}

// loginPath is the PATH of the user's login shell: an app started from Finder
// gets only /usr/bin:/bin:/usr/sbin:/sbin, and the terminal, Build --pdf and
// demos need docker, brew tools and the like.
func loginPath() string {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/zsh"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	if b, err := exec.CommandContext(ctx, shell, "-l", "-c", "printf %s \"$PATH\"").Output(); err == nil && len(b) > 0 {
		return string(b)
	}
	return "/opt/homebrew/bin:/usr/local/bin:" + os.Getenv("PATH")
}
