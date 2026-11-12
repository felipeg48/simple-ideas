package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestMain keeps the tests off the real user's si.conf.
func TestMain(m *testing.M) {
	d, _ := os.MkdirTemp("", "si-config-home-")
	os.Setenv("HOME", d)
	os.Setenv("XDG_CONFIG_HOME", filepath.Join(d, ".config"))
	os.Setenv("AppData", filepath.Join(d, "AppData"))
	code := m.Run()
	os.RemoveAll(d)
	os.Exit(code)
}

func writeUser(t *testing.T, text string, mode os.FileMode) string {
	t.Helper()
	p := UserPath()
	os.MkdirAll(filepath.Dir(p), 0o700)
	if err := os.WriteFile(p, []byte(text), mode); err != nil {
		t.Fatal(err)
	}
	os.Chmod(p, mode)
	t.Cleanup(func() { os.Remove(p) })
	return p
}

func TestDemoWith(t *testing.T) {
	writeUser(t, "demo.with: subtitles, chapters\n", 0o600)
	if c := Load(""); c.DemoWith != "subtitles, chapters" {
		t.Errorf("user: %q", c.DemoWith)
	}
	proj := filepath.Join(t.TempDir(), "si.conf")
	os.WriteFile(proj, []byte("demo.with: captions\n"), 0o644)
	if c := Load("", proj); c.DemoWith != "captions" || len(c.Warnings) > 0 {
		t.Errorf("project over user: %q %v", c.DemoWith, c.Warnings)
	}
}

func TestUserAndProject(t *testing.T) {
	user := writeUser(t, "theme: paper\nvoice: openai:nova\nvoice.style: calm\nvoice.openai.key: sk-user-1234567890\n", 0o600)
	proj := filepath.Join(t.TempDir(), "si.conf")
	os.WriteFile(proj, []byte("voice: kokoro:af_bella\nvoice.elevenlabs.key: leaked-key-abcdef\n"), 0o644)

	c := Load("", proj)
	if c.Theme != "paper" || c.Voice != "kokoro:af_bella" || c.VoiceStyle != "calm" {
		t.Errorf("project values win, user values stay: theme %q voice %q style %q", c.Theme, c.Voice, c.VoiceStyle)
	}
	if c.OpenAIKey != "sk-user-1234567890" || c.ElevenLabsKey != "" {
		t.Errorf("keys: openai %q (from the user file), elevenlabs %q (project: ignored)", c.OpenAIKey, c.ElevenLabsKey)
	}
	if len(c.Warnings) != 1 || !strings.Contains(c.Warnings[0], proj+": line 2: voice.elevenlabs.key is ignored") || strings.Contains(c.Warnings[0], "leaked") {
		t.Errorf("warnings: %v", c.Warnings)
	}
	if c.Path != proj {
		t.Errorf("Path = %q", c.Path)
	}

	if k, from := APIKey("openai"); k != "sk-user-1234567890" || from != user {
		t.Errorf("APIKey(openai) = %q from %q", k, from)
	}
	t.Setenv("OPENAI_API_KEY", "sk-env")
	if k, from := APIKey("openai"); k != "sk-env" || from != "OPENAI_API_KEY" {
		t.Errorf("the environment wins: %q from %q", k, from)
	}
	if k, _ := APIKey("elevenlabs"); k != "" {
		t.Errorf("a project key never counts: %q", k)
	}
}

func TestUserFileReadableByOthers(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix modes")
	}
	writeUser(t, "voice.openai.key: sk-abc\n", 0o644)
	if w := LoadUser().Warnings; len(w) != 1 || !strings.Contains(w[0], "chmod 600") || strings.Contains(w[0], "sk-abc") {
		t.Errorf("warnings: %v", w)
	}
	writeUser(t, "theme: paper\n", 0o644) // no key: nothing to hide
	if w := LoadUser().Warnings; len(w) != 0 {
		t.Errorf("warnings: %v", w)
	}
}

func TestUnsetKeyAndMode(t *testing.T) {
	p := filepath.Join(t.TempDir(), "si.conf")
	os.WriteFile(p, []byte("theme: paper\nvoice.openai.key: a\n# voice.openai.key: old\n"), 0o600)
	SetKey(p, "voice.openai.key", "b")
	if b, _ := os.ReadFile(p); string(b) != "theme: paper\nvoice.openai.key: b\n# voice.openai.key: old\n" {
		t.Errorf("set: %q", b)
	}
	UnsetKey(p, "voice.openai.key")
	if b, _ := os.ReadFile(p); string(b) != "theme: paper\n# voice.openai.key: old\n" {
		t.Errorf("unset: %q", b)
	}
	if st, _ := os.Stat(p); runtime.GOOS != "windows" && st.Mode().Perm() != 0o600 {
		t.Errorf("mode %v, want 0600 kept", st.Mode().Perm())
	}
	if err := UnsetKey(filepath.Join(t.TempDir(), "none.conf"), "x"); err != nil {
		t.Error(err)
	}
	if Mask("sk-1234567890") != "****7890" || Mask("short") != "****" {
		t.Error("mask")
	}
}
