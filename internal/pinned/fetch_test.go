package pinned

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	good := map[string]string{
		"gist:abc123@4f2e9c1":                  "gist:abc123@4f2e9c1",
		"gist:abc123@4f2e9c1/seed.sh":          "gist:abc123@4f2e9c1/seed.sh",
		"github:acme/ops@v1.2/scripts/seed.sh": "github:acme/ops@v1.2/scripts/seed.sh",
		"github:acme/ops@v1.2/scripts/":        "github:acme/ops@v1.2/scripts/",
		"github:acme/ops@0123abc":              "github:acme/ops@0123abc/",
	}
	for in, key := range good {
		s, err := Parse(in, "")
		if err != nil || s.Key() != key {
			t.Errorf("%s: %v %v", in, s, err)
		}
	}
	u, err := Parse("https://example.com/x/seed.sh", strings.Repeat("ab", 32))
	if err != nil || u.Kind != "url" || u.Name() != "seed" {
		t.Errorf("url: %+v %v", u, err)
	}
	for in, sum := range map[string]string{
		"gist:abc123":                     "",
		"github:acme/ops/scripts/seed.sh": "",
		"https://example.com/seed.sh":     "",
		"https://example.com/x.sh":        "123",
		"github:acme/ops@v1/../etc":       "",
		"gist:abc@1/a/b":                  "",
	} {
		if _, err := Parse(in, sum); err == nil {
			t.Errorf("%s: no error", in)
		}
	}
	if _, err := Parse("github:acme/ops/seed.sh", ""); err == nil || !strings.Contains(err.Error(), "not pinned") {
		t.Errorf("unpinned: %v", err)
	}
}

func TestParseTarget(t *testing.T) {
	s, ok, err := ParseTarget("github:felipeg48/simple-ideas@7492470/docs/demos/")
	if !ok || err != nil || !s.Folder() || s.Repo != "simple-ideas" {
		t.Errorf("%+v %v %v", s, ok, err)
	}
	if _, ok, _ := ParseTarget("docs/demos/"); ok {
		t.Error("a local path")
	}
	sum := strings.Repeat("0f", 32)
	s, ok, err = ParseTarget("https://x.org/talk.md · sha256 " + sum)
	if !ok || err != nil || s.Sum != sum {
		t.Errorf("%+v %v", s, err)
	}
	if got := JoinTarget([]string{"https://x.org/a.md", "·", "sha256", sum, "--x"}); len(got) != 2 || got[0] != "https://x.org/a.md · sha256 "+sum {
		t.Errorf("join: %q", got)
	}
}

// fakeGitHub serves gists, contents, tarballs and plain files.
func fakeGitHub(t *testing.T, tarball []byte) (*httptest.Server, *int) {
	hits := new(int)
	mux := http.NewServeMux()
	mux.HandleFunc("/gists/g1/r1", func(w http.ResponseWriter, r *http.Request) {
		*hits++
		json.NewEncoder(w).Encode(map[string]any{"files": map[string]any{"seed.sh": map[string]any{"filename": "seed.sh", "content": "echo seeded\n"}}})
	})
	mux.HandleFunc("/gists/g2/r1", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"files": map[string]any{"a.sh": map[string]any{"content": "a"}, "b.sh": map[string]any{"content": "b"}}})
	})
	mux.HandleFunc("/repos/acme/ops/contents/", func(w http.ResponseWriter, r *http.Request) {
		*hits++
		if r.URL.Query().Get("ref") != "v1" {
			http.Error(w, `{"message":"No commit found for the ref"}`, 404)
			return
		}
		switch strings.TrimPrefix(r.URL.Path, "/repos/acme/ops/contents/") {
		case "scripts":
			json.NewEncoder(w).Encode([]map[string]string{{"name": "b.sh", "path": "scripts/b.sh", "type": "file"}, {"name": "README.md", "path": "scripts/README.md", "type": "file"},
				{"name": "a.sh", "path": "scripts/a.sh", "type": "file"}, {"name": "lib", "path": "scripts/lib", "type": "dir"}})
		case "scripts/a.sh":
			w.Write([]byte("echo a\n"))
		case "scripts/b.sh":
			w.Write([]byte("echo b\n"))
		default:
			http.NotFound(w, r)
		}
	})
	mux.HandleFunc("/repos/acme/talks/tarball/v2", func(w http.ResponseWriter, r *http.Request) {
		*hits++
		w.Write(tarball)
	})
	mux.HandleFunc("/files/seed.sh", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("echo from the web\n")) })
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, hits
}

func testClient(t *testing.T, api string, tr *Trust) *Client {
	if tr == nil {
		tr = &Trust{Store: filepath.Join(t.TempDir(), "trusted"), Yes: true}
	}
	return &Client{API: api, Cache: t.TempDir(), Trust: tr, GH: "-"}
}

func TestScripts(t *testing.T) {
	srv, hits := fakeGitHub(t, nil)
	c := testClient(t, srv.URL, nil)
	src, _ := Parse("gist:g1@r1", "")
	ps, err := c.Scripts(src)
	if err != nil || len(ps) != 1 {
		t.Fatal(ps, err)
	}
	if b, _ := os.ReadFile(ps[0]); string(b) != "echo seeded\n" {
		t.Errorf("gist: %q", b)
	}
	if _, err := c.Scripts(src); err != nil || *hits != 1 {
		t.Errorf("cached by pin: %d hits, %v", *hits, err)
	}
	two, _ := Parse("gist:g2@r1", "")
	if _, err := c.Scripts(two); err == nil || !strings.Contains(err.Error(), "name one") {
		t.Errorf("two files: %v", err)
	}
	dir, _ := Parse("github:acme/ops@v1/scripts/", "")
	ps, err = c.Scripts(dir)
	if err != nil || len(ps) != 2 || filepath.Base(ps[0]) != "a.sh" || filepath.Base(ps[1]) != "b.sh" {
		t.Errorf("folder: %v %v", ps, err)
	}
	one, _ := Parse("github:acme/ops@v1/scripts/b.sh", "")
	if ps, err := c.Scripts(one); err != nil || len(ps) != 1 {
		t.Errorf("file: %v %v", ps, err)
	}
	bad, _ := Parse("github:acme/ops@v9/scripts/b.sh", "")
	if _, err := c.Scripts(bad); err == nil || !strings.Contains(err.Error(), "404") {
		t.Errorf("a ref that is not there: %v", err)
	}
	good := sum([]byte("echo from the web\n"))
	u, _ := Parse(srv.URL+"/files/seed.sh", good)
	if ps, err := c.Scripts(u); err != nil || filepath.Base(ps[0]) != "seed.sh" {
		t.Errorf("url: %v %v", ps, err)
	}
	wrong, _ := Parse(srv.URL+"/files/seed.sh", strings.Repeat("0", 64))
	if _, err := c.Scripts(wrong); err == nil || !strings.Contains(err.Error(), "pin no longer matches") {
		t.Errorf("a wrong sha256: %v", err)
	}
}

func TestTrust(t *testing.T) {
	srv, _ := fakeGitHub(t, nil)
	store := filepath.Join(t.TempDir(), "trusted")
	src, _ := Parse("gist:g1@r1", "")
	// no terminal, no --trust: refused, with the reason
	c := testClient(t, srv.URL, &Trust{Store: store})
	if _, err := c.Scripts(src); err == nil || !strings.Contains(err.Error(), "--trust") {
		t.Fatalf("no terminal: %v", err)
	}
	// --trust: yes for the run, remembered nowhere
	c.Trust = &Trust{Store: store, Yes: true}
	if _, err := c.Scripts(src); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(store); err == nil {
		t.Error("--trust wrote the store")
	}
	// asked: shown the source, a no refuses
	var shown bytes.Buffer
	c.Trust = &Trust{Store: store, Ask: true, In: strings.NewReader("n\n"), Out: &shown}
	if _, err := c.Scripts(src); err == nil || !strings.Contains(shown.String(), "echo seeded") || !strings.Contains(shown.String(), "gist g1 at revision r1") {
		t.Fatalf("no: %v\n%s", err, shown.String())
	}
	// a yes is remembered for the pin: asked once
	c.Trust = &Trust{Store: store, Ask: true, In: strings.NewReader("y\n"), Out: &shown}
	if _, err := c.Scripts(src); err != nil {
		t.Fatal(err)
	}
	c.Trust = &Trust{Store: store}
	if _, err := c.Scripts(src); err != nil {
		t.Errorf("trusted before: %v", err)
	}
	// the same pin with other contents: the pin no longer matches, even with --trust
	dir := filepath.Join(c.Cache, "scripts", keyDir(src.Key()))
	os.WriteFile(filepath.Join(dir, "seed.sh"), []byte("rm -rf /\n"), 0o755)
	c.Trust = &Trust{Store: store, Yes: true}
	if _, err := c.Scripts(src); err == nil || !strings.Contains(err.Error(), "changed since it was trusted") {
		t.Errorf("changed: %v", err)
	}
}

func tarGz(t *testing.T, files map[string]string) []byte {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(zw)
	tw.WriteHeader(&tar.Header{Name: "acme-talks-abc/", Typeflag: tar.TypeDir, Mode: 0o755})
	for n, body := range files {
		tw.WriteHeader(&tar.Header{Name: "acme-talks-abc/" + n, Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len(body))})
		tw.Write([]byte(body))
	}
	tw.Close()
	zw.Close()
	return buf.Bytes()
}

func TestTree(t *testing.T) {
	tgz := tarGz(t, map[string]string{"talks/talk.md": "# Talk\n", "examples/app/index.html": "<p>hi</p>"})
	srv, hits := fakeGitHub(t, tgz)
	c := testClient(t, srv.URL, nil)
	src, _, _ := ParseTarget("github:acme/talks@v2/talks/talk.md")
	p, err := c.Tree(src, nil)
	if err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(p); string(b) != "# Talk\n" {
		t.Errorf("talk: %q", b)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(p), "..", "examples", "app", "index.html")); err != nil {
		t.Error("the repo's layout is kept")
	}
	if _, err := c.Tree(src, nil); err != nil || *hits != 1 {
		t.Errorf("cached by pin: %d, %v", *hits, err)
	}
	missing, _, _ := ParseTarget("github:acme/talks@v2/nope.md")
	if _, err := c.Tree(missing, nil); err == nil {
		t.Error("a path not in the repo")
	}
	evil := tarGz(t, map[string]string{"../../evil": "x"})
	if _, err := extract(evil, filepath.Join(t.TempDir(), "r")); err == nil {
		t.Error("a path out of the repo")
	}
}
