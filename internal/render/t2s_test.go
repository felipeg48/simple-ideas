package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestT2SListenBar: feature: t2s carries the Listen runtime and its settings;
// a page without it, or with feature: none, carries neither.
func TestT2SListenBar(t *testing.T) {
	dir := t.TempDir()
	for name, fm := range map[string]string{
		"on.md":   "type: article\ntitle: On\nfeature: t2s\nt2s-highlight: word\nt2s-rate: 1.25",
		"off.md":  "type: article\ntitle: Off",
		"none.md": "type: article\ntitle: None\nfeature: none, t2s",
	} {
		os.WriteFile(filepath.Join(dir, name), []byte("---\n"+fm+"\n---\n\n# Title\n\nOne sentence. Two.\n"), 0o644)
	}
	res, page := renderLab(t, filepath.Join(dir, "on.md"), Options{})
	if len(res.Problems) > 0 {
		t.Fatalf("problems: %v", res.Problems)
	}
	if !strings.Contains(page, `window.SI_T2S = {"highlight":"word","rate":"1.25","voice":""}`) || !strings.Contains(page, "si-t2s-play") {
		t.Error("feature: t2s page has no Listen runtime with its settings")
	}
	for _, f := range []string{"off.md", "none.md"} {
		if _, page := renderLab(t, filepath.Join(dir, f), Options{}); strings.Contains(page, "SI_T2S") {
			t.Errorf("%s carries the Listen runtime", f)
		}
	}
}
