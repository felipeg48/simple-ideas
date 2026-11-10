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
