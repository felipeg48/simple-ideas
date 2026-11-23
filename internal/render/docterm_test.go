package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

// TestLabDrawer: a lab page carries the drawer's loader and Run buttons only
// when it is served with the terminal (si play, a class seat), never built.
func TestLabDrawer(t *testing.T) {
	f := filepath.Join(t.TempDir(), "lab.md")
	os.WriteFile(f, []byte("---\ntype: lab\ntitle: L\n---\n# L\n\n```bash\ndocker ps\n```\n"), 0o644)
	d, err := deck.Parse(f)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(Deck(d, Options{Terminal: true}).HTML), "si-doc-term") {
		t.Error("served lab: no drawer")
	}
	if strings.Contains(string(Deck(d, Options{}).HTML), "/_si/term/") {
		t.Error("a built lab references the drawer")
	}
}
