package render

import (
	"strings"
	"testing"
)

func TestHighlightMark(t *testing.T) {
	h := string(renderDoc(t, "## Five things ==to keep==\n\nA ==**bold** idea== and `a == b` and x = y = z and ===x===.\n").HTML)
	for _, want := range []string{
		`Five things <span class="hl">to keep</span>`,
		`A <span class="hl"><strong>bold</strong> idea</span>`,
		"<code>a == b</code>",
		"x = y = z",
	} {
		if !strings.Contains(h, want) {
			t.Errorf("lacks %q", want)
		}
	}
	if strings.Contains(h, `<span class="hl">x</span>`) {
		t.Error("three = is not a highlight")
	}
}
