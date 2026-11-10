package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/felipeg48/simple-ideas/internal/config"
	"github.com/felipeg48/simple-ideas/internal/webdemo"
)

// TestMain keeps every test here off the real user's si.conf and caches.
func TestMain(m *testing.M) {
	d, _ := os.MkdirTemp("", "si-cmd-home-")
	for k, v := range map[string]string{"HOME": d, "XDG_CONFIG_HOME": filepath.Join(d, ".config"), "XDG_CACHE_HOME": filepath.Join(d, ".cache"),
		"AppData": filepath.Join(d, "AppData"), "LocalAppData": filepath.Join(d, "LocalAppData"), "ELEVENLABS_API_KEY": "", "OPENAI_API_KEY": ""} {
		os.Setenv(k, v)
	}
	code := m.Run()
	os.RemoveAll(d)
	os.Exit(code)
}

func TestVoiceLoginLogout(t *testing.T) {
	voiceStdin = strings.NewReader("sk-test-1234567890\n")
	defer func() { voiceStdin = os.Stdin }()
	var out, errb bytes.Buffer
	if code := run([]string{"voice", "login", "openai"}, &out, &errb); code != exitOK {
		t.Fatalf("login: %d %s", code, errb.String())
	}
	if strings.Contains(out.String()+errb.String(), "sk-test-1234567890") || !strings.Contains(out.String(), "****7890") {
		t.Errorf("the key must never print in full: %q %q", out.String(), errb.String())
	}
	p := config.UserPath()
	st, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && st.Mode().Perm() != 0o600 {
		t.Errorf("mode %v, want 0600", st.Mode().Perm())
	}
	if k, from := config.APIKey("openai"); k != "sk-test-1234567890" || from != p {
		t.Errorf("APIKey = %q from %q", k, from)
	}
	out.Reset()
	if code := run([]string{"voice", "logout", "openai"}, &out, &errb); code != exitOK {
		t.Fatalf("logout: %d", code)
	}
	if k, _ := config.APIKey("openai"); k != "" {
		t.Errorf("still a key after logout: %q", k)
	}
	voiceStdin = strings.NewReader("\n")
	if code := run([]string{"voice", "login", "elevenlabs"}, &out, &errb); code != exitUsage {
		t.Errorf("an empty key stores nothing: %d", code)
	}
	if code := run([]string{"voice", "login", "kokoro"}, &out, &errb); code != exitUsage {
		t.Errorf("kokoro has no key: %d", code)
	}
}

func TestChooseVoice(t *testing.T) {
	for _, c := range []struct {
		specs []string
		want  string
	}{
		{[]string{"openai:nova", "kokoro:af_bella", "system"}, "openai:nova"},
		{[]string{"", "kokoro:af_bella", "system"}, "kokoro:af_bella"},
		{[]string{"", "", "elevenlabs"}, "elevenlabs:21m00Tcm4TlvDq8ikWAM"},
	} {
		v, err := chooseVoice(c.specs...)
		if err != nil || v.String() != c.want {
			t.Errorf("%q: %v %v, want %s", c.specs, v, err, c.want)
		}
	}
	if v, _ := chooseVoice("", ""); v != webdemo.DefaultVoice() {
		t.Errorf("nothing set: %v", v)
	}
	if _, err := chooseVoice("robot:x"); err == nil {
		t.Error("an unknown engine is an error")
	}
}

func TestPlayerFor(t *testing.T) {
	has := func(names ...string) func(string) (string, error) {
		return func(n string) (string, error) {
			for _, x := range names {
				if x == n {
					return "/usr/bin/" + n, nil
				}
			}
			return "", errors.New("no")
		}
	}
	for _, c := range []struct {
		goos string
		look func(string) (string, error)
		want string
	}{
		{"darwin", has(), "afplay a.wav"},
		{"linux", has("aplay", "paplay"), "paplay a.wav"},
		{"linux", has("aplay", "ffplay"), "aplay -q a.wav"},
		{"linux", has("ffplay"), "ffplay -nodisp -autoexit -loglevel quiet a.wav"},
		{"linux", has(), ""},
	} {
		if got := strings.Join(playerFor(c.goos, "a.wav", c.look), " "); got != c.want {
			t.Errorf("%s: %q, want %q", c.goos, got, c.want)
		}
	}
	if w := playerFor("windows", `C:\it's.wav`, has()); !strings.Contains(w[4], `SoundPlayer 'C:\it''s.wav'`) {
		t.Errorf("windows: %q", w)
	}
}

// silentWAV is half a second of 24 kHz mono 16-bit silence.
func silentWAV(t *testing.T, path string) {
	t.Helper()
	pcm := make([]byte, 24000)
	var b bytes.Buffer
	b.WriteString("RIFF")
	binary.Write(&b, binary.LittleEndian, uint32(36+len(pcm)))
	b.WriteString("WAVEfmt ")
	for _, v := range []any{uint32(16), uint16(1), uint16(1), uint32(24000), uint32(48000), uint16(2), uint16(16)} {
		binary.Write(&b, binary.LittleEndian, v)
	}
	b.WriteString("data")
	binary.Write(&b, binary.LittleEndian, uint32(len(pcm)))
	b.Write(pcm)
	os.MkdirAll(filepath.Dir(path), 0o755)
	if err := os.WriteFile(path, b.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestVoiceSayPlaysFromCache(t *testing.T) {
	var played [][]string
	defer func(f func([]string) error) { runPlayer = f }(runPlayer)
	runPlayer = func(argv []string) error { played = append(played, argv); return nil }
	t.Setenv("OPENAI_API_KEY", "sk-env")
	sy := &webdemo.Synth{Cache: webdemo.VoiceCache()}
	v, _ := webdemo.ParseVoice("openai:nova")
	silentWAV(t, sy.Path(v, "Ideas finished"))

	var out, errb bytes.Buffer
	if code := run([]string{"voice", "say", "Ideas", "finished", "--voice", "openai:nova"}, &out, &errb); code != exitOK {
		t.Fatalf("say: %d %s", code, errb.String())
	}
	if len(played) != 1 || !strings.Contains(strings.Join(played[0], " "), sy.Path(v, "Ideas finished")) {
		t.Errorf("played %q", played)
	}
	// -o saves instead of playing; --play does both
	wav := filepath.Join(t.TempDir(), "out.wav")
	played = nil
	if code := run([]string{"voice", "say", "Ideas finished", "--voice", "openai:nova", "-o", wav}, &out, &errb); code != exitOK || len(played) != 0 {
		t.Fatalf("-o: %d, played %q, %s", code, played, errb.String())
	}
	if st, err := os.Stat(wav); err != nil || st.Size() < 1000 {
		t.Errorf("saved: %v", err)
	}
	if run([]string{"voice", "say", "Ideas finished", "--voice", "openai:nova", "-o", wav, "--play"}, &out, &errb); len(played) != 1 {
		t.Errorf("--play: played %q", played)
	}
	// voice: in the user's si.conf, no flag
	os.MkdirAll(filepath.Dir(config.UserPath()), 0o700)
	os.WriteFile(config.UserPath(), []byte("voice: openai:nova\n"), 0o600)
	defer os.Remove(config.UserPath())
	played = nil
	if code := run([]string{"voice", "say", "Ideas finished"}, &out, &errb); code != exitOK || len(played) != 1 {
		t.Errorf("si.conf voice: %d, played %q, %s", code, played, errb.String())
	}
}

func TestVoiceSayFallsBackToSystem(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("system voice here is say")
	}
	var played [][]string
	defer func(f func([]string) error) { runPlayer = f }(runPlayer)
	runPlayer = func(argv []string) error { played = append(played, argv); return nil }
	sy := &webdemo.Synth{Cache: webdemo.VoiceCache()}
	silentWAV(t, sy.Path(webdemo.Voice{Engine: "system"}, "hello"))
	var out, errb bytes.Buffer
	if code := run([]string{"voice", "say", "hello", "--voice", "elevenlabs"}, &out, &errb); code != exitOK || len(played) != 1 {
		t.Fatalf("fallback: %d %q %s", code, played, errb.String())
	}
	if n := strings.Count(errb.String(), "\n"); n != 1 || !strings.Contains(errb.String(), "speaking with the system voice") {
		t.Errorf("one line of note: %q", errb.String())
	}
}

func TestVoiceList(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run([]string{"voice", "list", "kokoro"}, &out, &errb); code != exitOK || !strings.Contains(out.String(), "af_heart") || !strings.Contains(out.String(), "si voice install kokoro") {
		t.Errorf("list kokoro: %d %s", code, out.String())
	}
	out.Reset()
	if code := run([]string{"voice", "list", "openai"}, &out, &errb); code != exitOK || !strings.Contains(out.String(), "alloy") || !strings.Contains(out.String(), "si voice login openai") {
		t.Errorf("list openai: %d %s", code, out.String())
	}
	if code := run([]string{"voice", "install", "nope"}, &out, &errb); code != exitUsage {
		t.Errorf("install nope: %d", code)
	}
	if code := run([]string{"voice"}, &out, &errb); code != exitUsage {
		t.Errorf("no command: %d", code)
	}
}

func TestCheckWarnsProjectKey(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "talk.md"), []byte("---\ntitle: T\n---\n\n# T\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "si.conf"), []byte("theme: paper\nvoice.openai.key: sk-leak-123456789\n"), 0o644)
	var out, errb bytes.Buffer
	run([]string{"check", dir}, &out, &errb)
	if !strings.Contains(errb.String(), filepath.Join(dir, "si.conf")+": line 2: voice.openai.key is ignored") || strings.Contains(errb.String(), "sk-leak") {
		t.Errorf("stderr: %s", errb.String())
	}
}
