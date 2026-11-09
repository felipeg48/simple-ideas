package render

import (
	"os"
	"testing"
)

// The fetch cache keeps assets forever, keyed by URL; test servers get
// random local ports, so a reused port could serve a stale entry from an
// earlier run (TestFontsInlined failed that way). Every run gets its own.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "si-render-cache-")
	if err == nil {
		os.Setenv("SI_CACHE_DIR", dir)
	}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}
