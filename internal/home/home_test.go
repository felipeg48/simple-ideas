package home

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDir(t *testing.T) {
	t.Setenv("SI_HOME", "/x/si-home")
	if Dir() != "/x/si-home" || Path("si.conf", Config) != filepath.Join("/x/si-home", "si.conf") {
		t.Errorf("SI_HOME not used: %s", Path("si.conf", Config))
	}
}

// The first ask moves the old copy (<config dir>/si/<rel>) into ~/.si.
func TestPathMovesTheOldCopy(t *testing.T) {
	h, cfg := t.TempDir(), t.TempDir()
	t.Setenv("SI_HOME", "")
	t.Setenv("HOME", h)
	t.Setenv("XDG_CONFIG_HOME", cfg) // os.UserConfigDir on Linux
	t.Setenv("AppData", cfg)         // on Windows
	old, err := os.UserConfigDir()
	if err != nil {
		t.Skip(err)
	}
	os.MkdirAll(filepath.Join(old, "si"), 0o700)
	os.WriteFile(filepath.Join(old, "si", "si.conf"), []byte("voice: kokoro:af_heart\n"), 0o600)
	p := Path("si.conf", Config)
	if p != filepath.Join(h, ".si", "si.conf") {
		t.Fatalf("path %s", p)
	}
	if b, err := os.ReadFile(p); err != nil || string(b) != "voice: kokoro:af_heart\n" {
		t.Errorf("not moved: %q %v", b, err)
	}
	if _, err := os.Stat(filepath.Join(old, "si", "si.conf")); err == nil {
		t.Error("old copy left behind")
	}
}
