//go:build darwin

package main

// The menu-bar icon lists the si play / si serve sessions running on this
// Mac: each writes <config dir>/si/sessions/<pid>.json while it runs
// (internal/serve/session.go in the si module).

/*
#include <stdlib.h>
*/
import "C"

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
)

type session struct {
	PID     int    `json:"pid"`
	Kind    string `json:"kind"`
	Src     string `json:"src"`
	URL     string `json:"url"`
	Deck    string `json:"deck,omitempty"`
	QR      string `json:"qr,omitempty"`
	Started string `json:"started"`
}

func sessionsDir() string {
	if d := os.Getenv("SI_SESSIONS_DIR"); d != "" {
		return d
	}
	// si v2.43.0+ writes ~/.si/sessions ($SI_HOME/sessions); older si wrote
	// <config dir>/si/sessions
	if h := os.Getenv("SI_HOME"); h != "" {
		return filepath.Join(h, "sessions")
	}
	if h, err := os.UserHomeDir(); err == nil {
		if st, err := os.Stat(filepath.Join(h, ".si", "sessions")); err == nil && st.IsDir() {
			return filepath.Join(h, ".si", "sessions")
		}
	}
	d, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(d, "si", "sessions")
}

// liveSessions reads the session files, oldest first, and deletes the ones
// whose process is gone (a session killed before it could unlist itself).
func liveSessions() []session {
	dir := sessionsDir()
	files, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	var out []session
	for _, f := range files {
		b, err := os.ReadFile(f)
		var s session
		if err != nil || json.Unmarshal(b, &s) != nil || s.PID <= 0 || !strings.HasPrefix(s.URL, "http://127.") && !strings.HasPrefix(s.URL, "http://[::1]") && !strings.HasPrefix(s.URL, "http://localhost") {
			continue // not ours, or not a loopback address
		}
		if syscall.Kill(s.PID, 0) != nil {
			os.Remove(f)
			continue
		}
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Started < out[j].Started })
	return out
}

//export goSessions
func goSessions() *C.char {
	b, _ := json.Marshal(liveSessions())
	return C.CString(string(b))
}

// stopSession asks a listed session to stop the way Ctrl+C does (SIGINT), and
// only one that is still listed: never an arbitrary pid.
func stopSession(pid int) bool {
	for _, s := range liveSessions() {
		if s.PID == pid {
			return syscall.Kill(pid, syscall.SIGINT) == nil
		}
	}
	return false
}

//export goStopSession
func goStopSession(pid C.int) C.int {
	if stopSession(int(pid)) {
		return 1
	}
	return 0
}
