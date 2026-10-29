package studio

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fx struct {
	t    *testing.T
	root string
	mux  *http.ServeMux
}

func newFx(t *testing.T) *fx {
	t.Helper()
	w, err := OpenWorkspace(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	(&api{ws: w}).routes(mux)
	return &fx{t, w.Root, mux}
}

func (f *fx) do(method, url string, body any) (int, map[string]any) {
	f.t.Helper()
	var rd strings.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = *strings.NewReader(string(b))
	}
	r := httptest.NewRequest(method, "/_si/studio/api/"+url, &rd)
	rec := httptest.NewRecorder()
	f.mux.ServeHTTP(rec, r)
	m := map[string]any{}
	_ = json.Unmarshal(rec.Body.Bytes(), &m)
	return rec.Code, m
}

func (f *fx) put(rel, s string) {
	f.t.Helper()
	p := filepath.Join(f.root, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		f.t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
		f.t.Fatal(err)
	}
}

func (f *fx) read(rel string) string {
	b, err := os.ReadFile(filepath.Join(f.root, rel))
	if err != nil {
		return "<missing>"
	}
	return string(b)
}

func TestReadWriteRoundTrip(t *testing.T) {
	f := newFx(t)
	f.put("a.md", "# one\n")
	if err := os.Chmod(filepath.Join(f.root, "a.md"), 0o600); err != nil {
		t.Fatal(err)
	}
	code, m := f.do("GET", "file?path=a.md", nil)
	if code != 200 || m["content"] != "# one\n" || m["path"] != "a.md" {
		t.Fatalf("get: %d %v", code, m)
	}
	code, m2 := f.do("PUT", "file?path=a.md", map[string]any{"content": "# two\n", "etag": m["etag"]})
	if code != 200 || m2["etag"] == "" || m2["etag"] == m["etag"] {
		t.Fatalf("put: %d %v", code, m2)
	}
	_, m3 := f.do("GET", "file?path=a.md", nil)
	if m3["content"] != "# two\n" || m3["etag"] != m2["etag"] {
		t.Fatalf("get2: %v", m3)
	}
	if fi, _ := os.Stat(filepath.Join(f.root, "a.md")); fi.Mode().Perm() != 0o600 {
		t.Errorf("mode = %v", fi.Mode())
	}
	if code, _ := f.do("GET", "file?path=nope.md", nil); code != 404 {
		t.Errorf("missing = %d", code)
	}
	// empty etag creates only if missing
	if code, _ := f.do("PUT", "file?path=new.md", map[string]any{"content": "x", "etag": ""}); code != 200 || f.read("new.md") != "x" {
		t.Errorf("create-only failed: %d", code)
	}
	if code, _ := f.do("PUT", "file?path=new.md", map[string]any{"content": "y", "etag": ""}); code != 409 || f.read("new.md") != "x" {
		t.Errorf("create-only over existing = %d", code)
	}
}

func TestWriteConflict(t *testing.T) {
	f := newFx(t)
	f.put("a.md", "v1")
	_, m := f.do("GET", "file?path=a.md", nil)
	f.put("a.md", "external")
	code, c := f.do("PUT", "file?path=a.md", map[string]any{"content": "mine", "etag": m["etag"]})
	if code != 409 || c["error"] != "conflict" || c["content"] != "external" || c["etag"] == "" {
		t.Fatalf("conflict: %d %v", code, c)
	}
	if f.read("a.md") != "external" {
		t.Error("file was overwritten")
	}
}

func TestReadRefusesBinaryAndHuge(t *testing.T) {
	f := newFx(t)
	f.put("p.png", "\x89PNG\r\n\x1a\n\x00\x00")
	f.put("bad.txt", "ok\xff\xfe")
	f.put("big.md", strings.Repeat("a", 6<<20))
	for _, c := range []struct {
		p    string
		want int
	}{{"p.png", 415}, {"bad.txt", 415}, {"big.md", 413}} {
		if code, _ := f.do("GET", "file?path="+c.p, nil); code != c.want {
			t.Errorf("%s = %d, want %d", c.p, code, c.want)
		}
	}
}

func TestCreateRenameDelete(t *testing.T) {
	f := newFx(t)
	if code, m := f.do("POST", "create", map[string]any{"path": "a/b.md", "content": "hi"}); code != 201 || m["etag"] == "" || f.read("a/b.md") != "hi" {
		t.Fatalf("create: %d %v", code, m)
	}
	if code, _ := f.do("POST", "create", map[string]any{"path": "a/b.md"}); code != 409 {
		t.Errorf("create again = %d", code)
	}
	if code, _ := f.do("POST", "create", map[string]any{"path": "d/e", "dir": true}); code != 201 {
		t.Errorf("mkdir = %d", code)
	}
	if code, _ := f.do("POST", "rename", map[string]any{"from": "a/b.md", "to": "a/c.md"}); code != 200 || f.read("a/c.md") != "hi" {
		t.Errorf("rename = %d", code)
	}
	f.put("a/x.md", "x")
	if code, _ := f.do("POST", "rename", map[string]any{"from": "a/c.md", "to": "a/x.md"}); code != 409 || f.read("a/x.md") != "x" {
		t.Errorf("rename onto existing = %d", code)
	}
	if code, _ := f.do("POST", "rename", map[string]any{"from": "zz", "to": "yy"}); code != 404 {
		t.Errorf("rename missing = %d", code)
	}
	if code, _ := f.do("DELETE", "file?path=a", nil); code != 409 {
		t.Errorf("delete non-empty dir = %d", code)
	}
	if code, _ := f.do("DELETE", "file?path=a/c.md", nil); code != 204 {
		t.Errorf("delete file = %d", code)
	}
	if code, _ := f.do("DELETE", "file?path=d/e", nil); code != 204 {
		t.Errorf("delete empty dir = %d", code)
	}
	if code, _ := f.do("DELETE", "file?path=gone", nil); code != 404 {
		t.Errorf("delete missing = %d", code)
	}
}

func TestFileAPIRejectsEscape(t *testing.T) {
	f := newFx(t)
	if code, _ := f.do("GET", "file?path=../x", nil); code != 403 {
		t.Errorf("escape = %d", code)
	}
	if code, _ := f.do("POST", "rename", map[string]any{"from": "a", "to": "../b"}); code != 403 {
		t.Errorf("rename escape = %d", code)
	}
}

func TestFileAPIRejectsDotSegments(t *testing.T) {
	f := newFx(t)
	f.put(".git/config", "x")
	f.put("a/.hidden", "x")
	f.put("ok.md", "x")
	for _, p := range []string{".git/config", ".superpowers/x", "a/.hidden"} {
		if code, _ := f.do("GET", "file?path="+p, nil); code != 403 {
			t.Errorf("GET %s = %d", p, code)
		}
		if code, _ := f.do("PUT", "file?path="+p, map[string]any{"content": "y"}); code != 403 {
			t.Errorf("PUT %s = %d", p, code)
		}
		if code, _ := f.do("POST", "create", map[string]any{"path": p}); code != 403 {
			t.Errorf("create %s = %d", p, code)
		}
		if code, _ := f.do("DELETE", "file?path="+p, nil); code != 403 {
			t.Errorf("DELETE %s = %d", p, code)
		}
		if code, _ := f.do("POST", "rename", map[string]any{"from": p, "to": "z"}); code != 403 {
			t.Errorf("rename from %s = %d", p, code)
		}
		if code, _ := f.do("POST", "rename", map[string]any{"from": "ok.md", "to": p}); code != 403 {
			t.Errorf("rename to %s = %d", p, code)
		}
	}
	if f.read(".git/config") != "x" || f.read("ok.md") != "x" {
		t.Error("files touched")
	}
}

func TestTreeRoute(t *testing.T) {
	f := newFx(t)
	f.put("a.md", "# T\n")
	code, m := f.do("GET", "tree", nil)
	if code != 200 || m["kind"] == nil {
		t.Fatalf("tree: %d %v", code, m)
	}
}

// doAny is do for endpoints answering a JSON array.
func (f *fx) doAny(method, url string) []any {
	f.t.Helper()
	rec := httptest.NewRecorder()
	f.mux.ServeHTTP(rec, httptest.NewRequest(method, "/_si/studio/api/"+url, nil))
	var v []any
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		f.t.Fatalf("%v: %s", err, rec.Body)
	}
	return v
}

// JSON would replace an invalid byte past the sniff window with U+FFFD while
// the etag hashes the raw bytes; the next save would corrupt the file.
func TestReadRefusesInvalidUTF8AfterSniffWindow(t *testing.T) {
	f := newFx(t)
	f.put("late.md", strings.Repeat("a", sniffLen+100)+"\xff tail")
	if code, _ := f.do("GET", "file?path=late.md", nil); code != 415 {
		t.Fatalf("got %d, want 415", code)
	}
}
