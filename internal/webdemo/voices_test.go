package webdemo

import (
	"archive/zip"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestKokoroVoices(t *testing.T) {
	dir := t.TempDir()
	names, fromFile := KokoroVoices(dir)
	if fromFile || len(names) < 20 || names[3] != "af_heart" {
		t.Errorf("built-in: %v %v", fromFile, names)
	}
	f, _ := os.Create(filepath.Join(dir, "voices-v1.0.bin"))
	z := zip.NewWriter(f)
	for _, n := range []string{"bm_lewis.npy", "af_heart.npy", "jf_alpha.npy", "readme.txt"} {
		z.Create(n)
	}
	z.Close()
	f.Close()
	if names, fromFile := KokoroVoices(dir); !fromFile || !reflect.DeepEqual(names, []string{"af_heart", "bm_lewis"}) {
		t.Errorf("from the file: %v %v (English only)", fromFile, names)
	}
}

func TestParseVoiceBareKokoroName(t *testing.T) {
	for in, want := range map[string]string{"af_heart": "kokoro:af_heart", "am_michael": "kokoro:am_michael", "kokoro:bf_emma": "kokoro:bf_emma", "system": "system"} {
		v, err := ParseVoice(in)
		if err != nil || v.String() != want && !(in == "system" && v.Engine == "system") {
			t.Errorf("ParseVoice(%q) = %v, %v; want %s", in, v, err, want)
		}
	}
	if _, err := ParseVoice("heart"); err == nil {
		t.Error("a word that is no engine and no Kokoro name must fail")
	}
}

// A rule whose words contain the word itself must not loop.
func TestSpokenSelfReplacing(t *testing.T) {
	ps := []Pronounce{{Word: "VEX", Say: "vex"}, {Word: "a", Say: "a b"}, {Word: "si", Say: "ess eye"}}
	if got := Spoken("VEX and a, si si", ps); got != "vex and a b b b, ess eye ess eye" {
		t.Errorf("Spoken = %q", got)
	}
}
