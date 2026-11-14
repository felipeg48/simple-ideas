package serve

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFuzzyScore(t *testing.T) {
	if fuzzyScore("xyz", "internal/serve/files.go") != 0 {
		t.Error("a non-subsequence scored")
	}
	if fuzzyScore("fgo", "internal/serve/files.go") == 0 {
		t.Error("a subsequence did not score")
	}
	// a run in the file name beats letters scattered over the path
	if a, b := fuzzyScore("term", "internal/serve/terminal.go"), fuzzyScore("term", "axtyezrqm/a.go"); a <= b {
		t.Errorf("run in name %d <= scattered %d", a, b)
	}
	// shorter path wins a tie
	if a, b := fuzzyScore("a", "a.go"), fuzzyScore("a", "a/b/c/d/e.go"); a <= b {
		t.Errorf("short %d <= long %d", a, b)
	}
}

// searchGet calls find or grep on a hub over dir.
func searchGet(t *testing.T, dir, api string, q url.Values, v any) int {
	t.Helper()
	h := newTermHub(dir)
	q.Set("dir", dir)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/_si/term/api/"+api+"?"+q.Encode(), nil)
	if api == "find" {
		h.find(rec, req)
	} else {
		h.grep(rec, req)
	}
	if rec.Code == http.StatusOK {
		if err := json.Unmarshal(rec.Body.Bytes(), v); err != nil {
			t.Fatal(err)
		}
	}
	return rec.Code
}

func writeFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for p, s := range files {
		p = filepath.Join(dir, p)
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

type findRes struct {
	Files []struct{ Path, Rel string }
	More  bool
}

func TestFindSkipsAndRanks(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"src/server.go": "", "src/serve_test.go": "", "docs/s/e/r/v/e.md": "",
		"node_modules/server.js": "", ".git/server": "", "vendor/server.go": "", "dist/server.js": "",
		"build/server.js": "", ".hidden/server.go": "", ".env": "",
	})
	var res findRes
	searchGet(t, dir, "find", url.Values{"q": {"server"}}, &res)
	if len(res.Files) == 0 || res.Files[0].Rel != "src/server.go" {
		t.Fatalf("best match: %+v", res.Files)
	}
	for _, f := range res.Files {
		for _, bad := range []string{"node_modules", ".git", "vendor", "dist", "build", ".hidden"} {
			if strings.HasPrefix(f.Rel, bad+"/") {
				t.Errorf("not skipped: %s", f.Rel)
			}
		}
	}
	// a query starting with a dot looks in dot-folders (never .git)
	res = findRes{}
	searchGet(t, dir, "find", url.Values{"q": {".hid"}}, &res)
	if len(res.Files) != 1 || res.Files[0].Rel != ".hidden/server.go" {
		t.Errorf(".hid: %+v", res.Files)
	}
}

func TestFindCap(t *testing.T) {
	dir := t.TempDir()
	m := map[string]string{}
	for i := 0; i < 70; i++ {
		m[fmt.Sprintf("f%02d.txt", i)] = ""
	}
	writeFiles(t, dir, m)
	var res findRes
	searchGet(t, dir, "find", url.Values{"q": {"txt"}}, &res)
	if len(res.Files) != findMaxResults || !res.More {
		t.Errorf("got %d more=%v", len(res.Files), res.More)
	}
}

type grepRes struct {
	Lines []struct {
		Rel  string
		Line int
		Text string
		Hits [][2]int
	}
	Hits int
	More bool
}

func TestGrep(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"a.txt":              "one Hello\ntwo\nhello hello\n",
		"b.bin":              "Hello\x00world",
		"node_modules/x.txt": "Hello",
		"é.txt":              "ééé Hello",
	})
	var res grepRes
	searchGet(t, dir, "grep", url.Values{"q": {"hello"}}, &res)
	if res.Hits != 4 || len(res.Lines) != 3 {
		t.Fatalf("case-insensitive: %+v", res)
	}
	for _, l := range res.Lines {
		if l.Rel == "b.bin" || strings.HasPrefix(l.Rel, "node_modules") {
			t.Errorf("not skipped: %s", l.Rel)
		}
		if l.Rel == "é.txt" && (l.Hits[0] != [2]int{4, 9}) { // UTF-16 offsets
			t.Errorf("offsets: %v", l.Hits)
		}
	}
	res = grepRes{}
	searchGet(t, dir, "grep", url.Values{"q": {"Hello"}, "case": {"1"}}, &res)
	if res.Hits != 2 {
		t.Errorf("case-sensitive hits %d", res.Hits)
	}
	res = grepRes{}
	searchGet(t, dir, "grep", url.Values{"q": {`^t\w+$`}, "re": {"1"}}, &res)
	if len(res.Lines) != 1 || res.Lines[0].Line != 2 || res.Lines[0].Text != "two" {
		t.Errorf("regex: %+v", res.Lines)
	}
	// a regex is plain text unless re=1
	res = grepRes{}
	searchGet(t, dir, "grep", url.Values{"q": {`^t\w+$`}}, &res)
	if len(res.Lines) != 0 {
		t.Errorf("plain text matched as regex: %+v", res.Lines)
	}
	if code := searchGet(t, dir, "grep", url.Values{"q": {"("}, "re": {"1"}}, &res); code != http.StatusBadRequest {
		t.Errorf("bad regex: %d", code)
	}
}

func TestGrepCapAndSize(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"many.txt": strings.Repeat("x\n", 1500),
		"big.txt":  "needle\n" + strings.Repeat("a", filesMaxBytes),
	})
	var res grepRes
	searchGet(t, dir, "grep", url.Values{"q": {"x"}}, &res)
	if res.Hits != grepMaxHits || !res.More {
		t.Errorf("cap: hits %d more %v", res.Hits, res.More)
	}
	res = grepRes{}
	searchGet(t, dir, "grep", url.Values{"q": {"needle"}}, &res)
	if len(res.Lines) != 0 {
		t.Errorf("a file over 1 MB was searched")
	}
}
