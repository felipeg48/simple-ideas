package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCanvasMotionHelpers checks data-grow/data-move generate state-keyed CSS
// and data-count is recognised (handled by deck.js, no CSS emitted for it).
func TestCanvasMotionHelpers(t *testing.T) {
	dir := t.TempDir()
	svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100" class="m">` +
		`<rect data-grow="g" x="0" y="0" width="10" height="90"/>` +
		`<circle data-move="mv" r="5" style="offset-path:path('M0 0 L90 0')"/>` +
		`<text data-count="c:100">0</text></svg>`
	if err := os.WriteFile(filepath.Join(dir, "m.svg"), []byte(svg), 0o644); err != nil {
		t.Fatal(err)
	}
	res := renderMD(t, dir, "---\ntitle: t\n---\n# M\n\n<!-- svg: m.svg g mv c -->\n")
	if len(res.Problems) != 0 {
		t.Fatalf("problems: %v", res.Problems)
	}
	h := string(res.HTML)
	for _, want := range []string{
		`[data-grow~="g"]{transform:scaleY(1)}`,
		`[data-move~="mv"]{offset-distance:100%}`,
	} {
		if !strings.Contains(h, want) {
			t.Errorf("missing generated CSS %q", want)
		}
	}
	// data-count has no generated CSS (deck.js drives it)
	if strings.Contains(h, `data-count~="c"]{`) {
		t.Error("data-count should not emit CSS")
	}
}
