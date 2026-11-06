package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestMindmapTree(t *testing.T) {
	r := &textRenderer{pal: PaletteFor("studio", "dark"), w: 80, k: 1 << 30}
	src := "# Order {#order}\n- Checkout\n  - Payment → payments.md\n  - Date {badge=Thu}\n- Delivery\n  - [!sticky] Call ops\n  ```go\n  x := 1\n  ```\n"
	got := ansi.Strip(strings.Join(r.mindmap(src), "\n"))
	for _, want := range []string{"✲ Order\n", "├─ Checkout", "│  ├─ Payment → payments.md", "│  └─ Date [Thu]", "└─ Delivery", "   ├─ ✎ Call ops", "   └─ ‹go›"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in\n%s", want, got)
		}
	}
}
