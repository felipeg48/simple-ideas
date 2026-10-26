package assets

import "testing"

func TestAnim(t *testing.T) {
	for _, n := range []string{"globe", "ranges", "roaches"} {
		if code, kind, ok := Animation(n); !ok || kind != "js" || len(code) < 50 {
			t.Errorf("%s: ok=%v kind=%s len=%d", n, ok, kind, len(code))
		}
	}
	if _, _, ok := Animation("nope"); ok {
		t.Error("nope should not exist")
	}
	if len(AnimationNames()) != 3 {
		t.Errorf("names=%v", AnimationNames())
	}
}
