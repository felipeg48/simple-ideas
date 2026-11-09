//go:build darwin

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// The menu lists live loopback sessions only, deletes the files of sessions
// whose process is gone, and Stop never signals a pid that is not listed.
func TestSessions(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SI_SESSIONS_DIR", dir)
	write := func(name string, s session) {
		b, _ := json.Marshal(s)
		os.WriteFile(filepath.Join(dir, name), b, 0o600)
	}
	write("live.json", session{PID: os.Getpid(), Kind: "play", Src: "/talks/talk.md", URL: "http://127.0.0.1:7070/", Started: "2026-11-09T10:00:00Z"})
	write("dead.json", session{PID: 999999, Kind: "play", URL: "http://127.0.0.1:7071/"})
	write("lan.json", session{PID: os.Getpid(), Kind: "play", URL: "http://192.168.1.5:7070/"})
	got := liveSessions()
	if len(got) != 1 || got[0].Src != "/talks/talk.md" {
		t.Fatalf("live sessions: %+v", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "dead.json")); err == nil {
		t.Error("a dead session's file should be deleted")
	}
	if stopSession(1) {
		t.Error("Stop must refuse a pid that is not a listed session")
	}
}
