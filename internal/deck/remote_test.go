package deck

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func gitRepo(t *testing.T) (dir, first, second string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	dir = t.TempDir()
	run := func(args ...string) string {
		cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.email=t@example.com", "-c", "user.name=T"}, args...)...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	run("init", "-q")
	os.MkdirAll(filepath.Join(dir, "src"), 0o755)
	os.WriteFile(filepath.Join(dir, "src", "Retry.java"), []byte("class Retry {\n  // tag::retry[]\n  int pay() { return 1; }\n  // end::retry[]\n}\n"), 0o644)
	run("add", ".")
	run("commit", "-qm", "first")
	first = run("rev-parse", "HEAD")
	os.WriteFile(filepath.Join(dir, "src", "Retry.java"), []byte("class Retry {\n  // tag::retry[]\n  int pay() { return retry(1); }\n  // end::retry[]\n}\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("x\n"), 0o644)
	run("add", ".")
	run("commit", "-qm", "second")
	second = run("rev-parse", "HEAD")
	return dir, first, second
}

func TestCodeAndDiffFromALocalClone(t *testing.T) {
	repo, first, second := gitRepo(t)
	dir := writeTree(t, map[string]string{
		"d.md": "---\nrepo: " + repo + "\n---\n# D\n\n<!-- code: path=src/Retry.java ref=" + first[:10] + " tag=retry -->\n\n" +
			"<!-- diff: ref=" + first[:10] + ".." + second[:10] + " path=src/Retry.java -->\n\n<!-- diff: ref=" + second[:10] + " path=src -->\n",
	})
	d, err := Parse(filepath.Join(dir, "d.md"))
	if err != nil || len(d.Problems) > 0 {
		t.Fatalf("%v %v", err, d.Problems)
	}
	body := d.Slides[0].Body
	for _, want := range []string{"```java title=\"Retry.java\" linenos start=3\nint pay() { return 1; }\n```",
		"````diff title=\"src/Retry.java\"", "````diff title=\"src\"", "-  int pay() { return 1; }", "+  int pay() { return retry(1); }"} {
		if !strings.Contains(body, want) {
			t.Errorf("body lacks %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, "README.md") {
		t.Error("path= filters the diff to src/")
	}
	if len(d.Warnings) > 0 {
		t.Errorf("a local clone has no moving-ref warning: %v", d.Warnings)
	}
	bad := writeTree(t, map[string]string{"b.md": "# B\n\n<!-- code: path=src/Retry.java -->\n<!-- code: repo=" + repo + " path=nope.java -->\n<!-- diff: https://example.com/x -->\n"})
	b, _ := Parse(filepath.Join(bad, "b.md"))
	var msgs []string
	for _, p := range b.Problems {
		msgs = append(msgs, p.Msg)
	}
	all := strings.Join(msgs, "\n")
	for _, want := range []string{"no repo:", "git show", "want a GitHub commit or compare URL"} {
		if !strings.Contains(all, want) {
			t.Errorf("problems lack %q:\n%s", want, all)
		}
	}
}

func TestGitHubPermalinkOffline(t *testing.T) {
	Offline = true
	defer func() { Offline = false }()
	os.Setenv("XDG_CACHE_HOME", t.TempDir()) // an empty cache (Linux); on macOS UserCacheDir ignores it, the key is unique anyway
	dir := writeTree(t, map[string]string{"d.md": "# D\n\n<!-- code: https://github.com/example-owner/no-such-repo-7f3a/blob/main/src/App.java#L5-L10 -->\n"})
	d, _ := Parse(filepath.Join(dir, "d.md"))
	if len(d.Problems) != 1 || !strings.Contains(d.Problems[0].Msg, "offline and not in the cache") {
		t.Errorf("offline without a cache entry: %v", d.Problems)
	}
	if len(d.Warnings) != 1 || !strings.Contains(d.Warnings[0].Msg, `ref "main" can move`) {
		t.Errorf("a branch ref warns: %v", d.Warnings)
	}
}

func TestFilterDiff(t *testing.T) {
	d := "diff --git a/a.go b/a.go\n+x\ndiff --git a/src/b.go b/src/b.go\n+y\n"
	if got := string(filterDiff([]byte(d), "src")); got != "diff --git a/src/b.go b/src/b.go\n+y\n" {
		t.Errorf("%q", got)
	}
}
