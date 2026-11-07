package serve

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCalloutDragIsSaved(t *testing.T) {
	dir := t.TempDir()
	talk := "# Talk\n\n## Retry\n\n```go window=mac\nfunc send() {\n\tpost()\n}\n```\n<!-- callout: 2 One attempt -->\n"
	os.WriteFile(filepath.Join(dir, "talk.md"), []byte(talk), 0o644)
	os.MkdirAll(filepath.Join(dir, "parts"), 0o755)
	part := "```go window=mac\nfunc b() {}\n```\n<!-- callout: 1 Included note at=1,2 -->\n"
	os.WriteFile(filepath.Join(dir, "parts", "code.md"), []byte(part), 0o644)
	s, err := New(dir, nil, "", "")
	if err != nil {
		t.Fatal(err)
	}
	h := s.Handler()
	post := func(body, origin string) int {
		req := httptest.NewRequest(http.MethodPost, "/_si/callout", strings.NewReader(body))
		req.Host = "127.0.0.1:7070"
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}
	if c := post(`{"page":"/talk.html","line":2,"text":"One attempt","at":[40,-20]}`, "http://127.0.0.1:7070"); c != http.StatusNoContent {
		t.Fatalf("save: %d", c)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "talk.md")); !strings.Contains(string(b), "<!-- callout: 2 One attempt at=40,-20 -->") {
		t.Errorf("talk.md: %s", b)
	}
	if c := post(`{"page":"/talk.html","line":1,"text":"Included note","at":[5,6]}`, ""); c != http.StatusNoContent {
		t.Fatalf("included file: %d", c)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "parts", "code.md")); !strings.Contains(string(b), "Included note at=5,6") {
		t.Errorf("parts/code.md: %s", b)
	}
	if c := post(`{"page":"/talk.html","line":2,"text":"One attempt","at":[1,1]}`, "https://evil.example"); c != http.StatusForbidden {
		t.Errorf("another site's page must be refused: %d", c)
	}
	if c := post(`{"page":"/talk.html","line":9,"text":"nothing like it","at":[1,1]}`, ""); c != http.StatusNotFound {
		t.Errorf("an unknown callout: %d", c)
	}
	if c := post(`{"page":"/../../etc/passwd.html","line":2,"text":"One attempt","at":[3,3]}`, ""); c != http.StatusNoContent {
		t.Errorf("a page outside the folder falls back to the folder's own files: %d", c)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "talk.md")); !strings.Contains(string(b), "at=3,3") {
		t.Errorf("only files under the served folder are written: %s", b)
	}
}
