package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felipeg48/simple-ideas/internal/sandbox"
)

func TestClassCommands(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	os.MkdirAll("lab", 0o755)
	os.WriteFile("lab/lab.md", []byte("---\ntype: lab\ntitle: L\nsandbox: docker:28-dind\n---\n# L\n"), 0o644)
	os.WriteFile("notes.md", []byte("# no sandbox\n"), 0o644)
	os.WriteFile("people.csv", []byte("name,email\nAda,ada@example.com\n"), 0o644)
	var out, errb bytes.Buffer
	if code := runClass([]string{"new", "lab", "--name", "Docker basics", "--roster", "people.csv", "--seats", "2", "--ttl", "3h"}, &out, &errb); code != exitOK {
		t.Fatalf("new: %d %s", code, errb.String())
	}
	if !strings.Contains(out.String(), "docker-basics, 3 seats") || !strings.Contains(out.String(), "Ada <ada@example.com>") {
		t.Errorf("new printed %q", out.String())
	}
	c, err := sandbox.LoadClass(root, "docker-basics")
	if err != nil || len(c.Seats) != 3 || c.Course != "lab" || c.TTL != "3h" {
		t.Fatalf("class %+v %v", c, err)
	}
	errb.Reset()
	if code := runClass([]string{"new", "notes.md", "--seats", "1"}, &out, &errb); code == exitOK || !strings.Contains(errb.String(), "no sandbox:") {
		t.Errorf("a course without sandbox: %d %q", code, errb.String())
	}
	if code := runClass([]string{"new", "lab"}, &out, &errb); code == exitOK {
		t.Error("a class without seats")
	}
	out.Reset()
	runClass([]string{"list"}, &out, &errb)
	if !strings.Contains(out.String(), "docker-basics") || !strings.Contains(out.String(), "3 seats, 0 joined") {
		t.Errorf("list %q", out.String())
	}
	out.Reset()
	runClass([]string{"codes", "docker-basics"}, &out, &errb)
	if !strings.Contains(out.String(), sandbox.ShowCode(c.Seats[0].Code)) {
		t.Errorf("codes %q", out.String())
	}

	p := &planner{root: root, key: strings.Repeat("k", 32), out: &out}
	h := p.handler()
	do := func(method, path, cookie, host string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, nil)
		r.Host, r.RemoteAddr = host, "127.0.0.1:5000"
		if cookie != "" {
			r.Header.Set("Cookie", cookie)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	if w := do("GET", "/api/state", "", "127.0.0.1:9"); w.Code != http.StatusForbidden {
		t.Errorf("planner without its key: %d", w.Code)
	}
	if w := do("GET", "/?k="+p.key, "", "evil.example"); w.Code != http.StatusForbidden {
		t.Errorf("planner under another host name: %d", w.Code)
	}
	w := do("GET", "/?k="+p.key, "", "127.0.0.1:9")
	if w.Code != http.StatusFound || len(w.Result().Cookies()) != 1 {
		t.Fatalf("key to cookie: %d", w.Code)
	}
	ck := "si_planner=" + p.key
	var st struct {
		Classes []struct {
			ID    string
			Seats []struct{ Show string }
		}
		Courses []courseChoice
	}
	if err := json.Unmarshal(do("GET", "/api/state", ck, "127.0.0.1:9").Body.Bytes(), &st); err != nil {
		t.Fatal(err)
	}
	if len(st.Classes) != 1 || len(st.Classes[0].Seats) != 3 || len(st.Courses) != 1 || st.Courses[0].Path != "lab" || st.Courses[0].Image != "docker:28-dind" {
		t.Errorf("state %+v", st)
	}
	if w := do("GET", "/sheet/docker-basics", ck, "127.0.0.1:9"); !strings.Contains(w.Body.String(), st.Classes[0].Seats[0].Show) {
		t.Error("codes sheet")
	}
	if w := do("POST", "/api/class/docker-basics/delete", ck, "127.0.0.1:9"); w.Code != 200 || fileExists(filepath.Join(sandbox.ClassDir(root), "docker-basics.json")) {
		t.Errorf("delete %d", w.Code)
	}
}
