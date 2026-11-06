package render

import (
	"strings"
	"testing"
)

func TestShareMedia(t *testing.T) {
	big := "data:image/png;base64," + strings.Repeat("A", 2000)
	small := "data:image/png;base64,AAAA"
	in := `<img src="` + big + `"><video src="v.mp4" poster="` + big + `"></video><img src="` + small + `"><img src="` + small + `">`
	out, script := shareMedia(in)
	if strings.Count(out+script, big) != 1 {
		t.Fatalf("big URI should appear once, got %d", strings.Count(out+script, big))
	}
	for _, want := range []string{`<img data-si-m-src="0">`, `poster`, `data-si-m-poster="0"`, `<img src="` + small + `">`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in %s", want, out)
		}
	}
	if out, script := shareMedia(`<img src="` + big + `">`); script != "" || !strings.Contains(out, big) {
		t.Error("a URI used once must stay in place")
	}
}
