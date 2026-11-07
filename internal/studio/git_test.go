package studio

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestHistory(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git")
	}
	for k, v := range map[string]string{"GIT_AUTHOR_NAME": "A", "GIT_AUTHOR_EMAIL": "a@x", "GIT_COMMITTER_NAME": "A", "GIT_COMMITTER_EMAIL": "a@x", "GIT_CONFIG_GLOBAL": os.DevNull} {
		t.Setenv(k, v)
	}
	f := newFx(t)
	if code, m := f.do("GET", "git/status", nil); code != 200 || m["repo"] != false {
		t.Fatalf("not a repo yet: %d %v", code, m)
	}
	if code, m := f.do("POST", "git/init", map[string]any{}); code != 200 {
		t.Fatalf("init: %d %v", code, m)
	}
	f.put("talk.md", "# Talk\n\nOne.\n")
	code, m := f.do("GET", "git/status", nil)
	files := m["files"].([]any)
	if code != 200 || len(files) != 1 || files[0].(map[string]any)["status"] != "untracked" {
		t.Fatalf("status: %d %v", code, m)
	}
	if _, m := f.do("GET", "git/diff?path=talk.md", nil); !strings.Contains(m["diff"].(string), "+One.") {
		t.Fatalf("a new file's diff is all of it: %v", m)
	}
	if code, m := f.do("POST", "git/commit", map[string]any{"message": "First talk", "paths": []string{"talk.md"}}); code != 200 || m["commit"] == "" {
		t.Fatalf("commit: %d %v", code, m)
	}
	first := ""
	if _, m := f.do("GET", "git/log?path=talk.md", nil); len(m["commits"].([]any)) != 1 {
		t.Fatalf("log: %v", m)
	} else {
		first = m["commits"].([]any)[0].(map[string]any)["hash"].(string)
	}
	f.put("talk.md", "# Talk\n\nTwo.\n")
	if _, m := f.do("GET", "git/diff?path=talk.md", nil); !strings.Contains(m["diff"].(string), "-One.") || !strings.Contains(m["diff"].(string), "+Two.") {
		t.Fatalf("diff: %v", m)
	}
	if _, m := f.do("GET", "git/show?path=talk.md&rev="+first, nil); m["text"] != "# Talk\n\nOne.\n" {
		t.Fatalf("show: %v", m)
	}
	if code, _ := f.do("POST", "git/restore", map[string]any{"path": "talk.md", "rev": first}); code != 200 {
		t.Fatal("restore")
	}
	if raw, _ := os.ReadFile(filepath.Join(f.root, "talk.md")); string(raw) != "# Talk\n\nOne.\n" {
		t.Fatalf("restored: %q", raw)
	}
	// refused: a revision that is not a hash, a path outside, an empty message
	if code, _ := f.do("GET", "git/show?path=talk.md&rev=HEAD;rm", nil); code != 400 {
		t.Errorf("rev must be a hash: %d", code)
	}
	if code, _ := f.do("GET", "git/diff?path=../x", nil); code != 403 {
		t.Errorf("outside the workspace: %d", code)
	}
	if code, _ := f.do("POST", "git/commit", map[string]any{"message": " ", "paths": []string{"talk.md"}}); code != 400 {
		t.Errorf("empty message: %d", code)
	}
}
