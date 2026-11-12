package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/felipeg48/simple-ideas/internal/tools"
)

func needChrome(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("sh scripts")
	}
	if tools.Find("chrome") == "" || testing.Short() {
		t.Skip("needs Chrome")
	}
}

const termDemo = "---\ntype: demo\napp: none\nshell: sh\ntitle: %s\n---\n## It runs\n- terminal \"T\"\n- run \"echo %s\" · wait\n- expect output \"%s\"\n"

func demoFile(t *testing.T, dir, name, word string) {
	t.Helper()
	os.MkdirAll(dir, 0o755)
	src := strings.ReplaceAll(termDemo, "%s", word)
	if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestDemoFolder: si demo test <folder> runs every demo script in name
// order, each in its own folder, with one combined report and one JUnit
// file, a test suite per script.
func TestDemoFolder(t *testing.T) {
	needChrome(t)
	dir := t.TempDir()
	demoFile(t, dir, "b.md", "bravo")
	demoFile(t, dir, "a.md", "alpha")
	os.WriteFile(filepath.Join(dir, "notes.md"), []byte("# Not a demo\n"), 0o644)
	out := filepath.Join(t.TempDir(), "out")
	junit := filepath.Join(out, "junit.xml")
	var so, se bytes.Buffer
	if code := run([]string{"demo", "test", dir, "-o", out, "--junit", junit}, &so, &se); code != exitOK {
		t.Fatalf("exit %d\n%s\n%s", code, so.String(), se.String())
	}
	if !strings.Contains(so.String(), "── a.md") || strings.Index(so.String(), "── a.md") > strings.Index(so.String(), "── b.md") {
		t.Errorf("name order:\n%s", so.String())
	}
	for _, p := range []string{"report.html", "report.json", "a/report.json", "b/report.html"} {
		if _, err := os.Stat(filepath.Join(out, p)); err != nil {
			t.Errorf("%s: %v", p, err)
		}
	}
	var suite struct {
		Passed  int
		Scripts []struct{ Script string }
	}
	b, _ := os.ReadFile(filepath.Join(out, "report.json"))
	json.Unmarshal(b, &suite)
	if suite.Passed != 2 || len(suite.Scripts) != 2 || suite.Scripts[0].Script != "a.md" {
		t.Errorf("report.json: %s", b)
	}
	html, _ := os.ReadFile(filepath.Join(out, "report.html"))
	if !strings.Contains(string(html), "alpha") || !strings.Contains(string(html), "bravo") {
		t.Error("report.html: both scripts")
	}
	var x struct {
		Suites []struct {
			Name string `xml:"name,attr"`
		} `xml:"testsuite"`
	}
	j, _ := os.ReadFile(junit)
	if xml.Unmarshal(j, &x); len(x.Suites) != 2 || x.Suites[0].Name != "alpha" {
		t.Errorf("junit: %s", j)
	}
}

func tgz(t *testing.T, top string, files map[string]string) []byte {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(zw)
	names := make([]string, 0, len(files))
	for n := range files {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		tw.WriteHeader(&tar.Header{Name: top + "/" + n, Typeflag: tar.TypeReg, Mode: 0o755, Size: int64(len(files[n]))})
		tw.Write([]byte(files[n]))
	}
	tw.Close()
	zw.Close()
	return buf.Bytes()
}

// fakeHub is GitHub's API for the tests: a repo's tarball at v1 and a gist.
func fakeHub(t *testing.T, files map[string]string) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/acme/demos/tarball/v1", func(w http.ResponseWriter, r *http.Request) { w.Write(tgz(t, "acme-demos-1234567", files)) })
	mux.HandleFunc("/gists/g1/r1", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"files": map[string]any{"mark.sh": map[string]any{"content": "#!/bin/sh\necho \"$@\" > \"$MARK\"\n"}}})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	t.Setenv("SI_GITHUB_API", srv.URL)
	t.Setenv("SI_HOME", t.TempDir())
	return srv
}

// TestDemoRemote: a folder of demos from a repo at a pin (its layout kept),
// a gist: setup source; refused without a terminal and without --trust; a
// pin that is not there fails.
func TestDemoRemote(t *testing.T) {
	needChrome(t)
	mark := filepath.Join(t.TempDir(), "mark")
	t.Setenv("MARK", mark)
	demo := strings.ReplaceAll(termDemo, "%s", "remote") + "\n"
	demo = strings.Replace(demo, "shell: sh\n", "shell: sh\nprepare: gist:g1@r1 from-the-gist\nsetup: cat ../shared/data.txt\n", 1)
	fakeHub(t, map[string]string{"demos/one.md": demo, "shared/data.txt": "shared", "README.md": "# x"})
	out := filepath.Join(t.TempDir(), "out")
	var so, se bytes.Buffer
	// stdin is not a terminal here: without --trust, refused with the reason
	if code := run([]string{"demo", "test", "github:acme/demos@v1/demos/", "-o", out}, &so, &se); code != exitProblems || !strings.Contains(se.String(), "pass --trust") {
		t.Fatalf("no --trust: %d\n%s", code, se.String())
	}
	so.Reset()
	se.Reset()
	if code := run([]string{"demo", "test", "github:acme/demos@v1/demos/", "-o", out, "--trust"}, &so, &se); code != exitOK {
		t.Fatalf("--trust: %d\n%s\n%s", code, so.String(), se.String())
	}
	if b, _ := os.ReadFile(mark); strings.TrimSpace(string(b)) != "from-the-gist" {
		t.Errorf("the gist's script: %q", b)
	}
	if _, err := os.Stat(filepath.Join(out, "one", "report.json")); err != nil {
		t.Error(err)
	}
	// one script of the repo, default output demo-out/<repo>/<script>/
	wd, _ := os.Getwd()
	os.Chdir(t.TempDir())
	defer os.Chdir(wd)
	so.Reset()
	if code := run([]string{"demo", "test", "github:acme/demos@v1/demos/one.md", "--trust"}, &so, &se); code != exitOK {
		t.Fatalf("one: %d\n%s", code, se.String())
	}
	if _, err := os.Stat(filepath.Join("demo-out", "demos", "one", "report.html")); err != nil {
		t.Errorf("demo-out/<repo>/<script>: %v", err)
	}
	se.Reset()
	if code := run([]string{"demo", "test", "github:acme/demos@v9/demos/", "--trust"}, &so, &se); code != exitProblems || !strings.Contains(se.String(), "404") {
		t.Errorf("a pin that is not there: %d %s", code, se.String())
	}
	se.Reset()
	if code := run([]string{"demo", "test", "github:acme/demos/demos/", "--trust"}, &so, &se); code != exitProblems || !strings.Contains(se.String(), "not pinned") {
		t.Errorf("unpinned: %d %s", code, se.String())
	}
}
