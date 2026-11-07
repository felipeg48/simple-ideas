package brand

import (
	"net/url"
	"strings"
	"testing"
)

func TestIssueURL(t *testing.T) {
	u, err := url.Parse(IssueURL(false, "2.5.0", "web Studio", "macOS"))
	if err != nil || !strings.HasPrefix(u.String(), Repo+"/issues/new?") {
		t.Fatalf("bug url %q: %v", u, err)
	}
	q := u.Query()
	for k, want := range map[string]string{"template": "bug.yml", "version": "2.5.0", "mode": "web Studio", "os": "macOS"} {
		if q.Get(k) != want {
			t.Errorf("%s = %q, want %q", k, q.Get(k), want)
		}
	}
	f, _ := url.Parse(IssueURL(true, "2.5.0", "desktop app", "macOS"))
	if f.Query().Get("template") != "feature.yml" || f.Query().Get("mode") != "desktop app" || f.Query().Has("version") {
		t.Errorf("feature url %q", f)
	}
}
