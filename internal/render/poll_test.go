package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

// A · hidden poll on a slide keeps its results for the next click (a build
// step); in a document it is the plain list.
func TestPollHidden(t *testing.T) {
	p := filepath.Join(t.TempDir(), "d.md")
	os.WriteFile(p, []byte("# T\n\n---\n\n## Vote\n\n<!-- poll: pick · A · B · hidden -->\n"), 0o644)
	d, err := deck.Parse(p)
	if err != nil {
		t.Fatal(err)
	}
	h := string(Deck(d, Options{}).HTML)
	if !strings.Contains(h, `data-poll="pick" data-hidden>`) || !strings.Contains(h, `<span class="si-step si-poll-reveal" data-step="1" hidden></span>`) {
		t.Errorf("hidden poll on a slide: no data-hidden or reveal step")
	}
	if strings.Contains(h, ">hidden<") {
		t.Errorf("hidden became an option")
	}
	doc := string(renderDoc(t, "<!-- poll: pick · A · B · hidden -->\n").HTML)
	if strings.Contains(doc, "data-hidden") || strings.Contains(doc, "si-poll-reveal") {
		t.Errorf("hidden poll in a document should be the plain list")
	}
}
