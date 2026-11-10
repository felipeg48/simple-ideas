package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSetKey(t *testing.T) {
	p := filepath.Join(t.TempDir(), "si.conf")
	if err := SetKey(p, "theme", "paper"); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(p); string(b) != "theme: paper\n" {
		t.Errorf("new: %q", b)
	}
	os.WriteFile(p, []byte("# theme: old\ntheme: paper\nfeature.annotate: off\n"), 0o644)
	SetKey(p, "theme", "themes/acme.css")
	if b, _ := os.ReadFile(p); string(b) != "# theme: old\ntheme: themes/acme.css\nfeature.annotate: off\n" {
		t.Errorf("%q", b)
	}
}
