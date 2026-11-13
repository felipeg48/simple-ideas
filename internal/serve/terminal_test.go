package serve

import (
	"bufio"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// newTermServer returns an httptest server for a one-deck dir with Terminal set.
func newTermServer(t *testing.T, terminal bool) *httptest.Server {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "talk.md"), []byte("# Talk\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := New(dir, nil, "", "")
	if err != nil {
		t.Fatal(err)
	}
	s.Terminal = terminal
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)
	t.Cleanup(s.CloseTerminals)
	return ts
}

// TestTerminalDisabledByDefault proves the exec surface does not exist unless
// Terminal is set — this is the loopback gate (main.go sets it only on a
// loopback listen address, never on --addr 0.0.0.0).
func TestTerminalDisabledByDefault(t *testing.T) {
	ts := newTermServer(t, false)
	for _, p := range []string{"/_si/term/api/new", "/_si/term/assets/term.js"} {
		res, err := http.Get(ts.URL + p)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != http.StatusNotFound {
			t.Errorf("%s with terminal off: status %d, want 404", p, res.StatusCode)
		}
	}
}

// TestTerminalAssetsServed checks the vendored xterm runtime is reachable.
func TestTerminalAssetsServed(t *testing.T) {
	ts := newTermServer(t, true)
	for _, f := range []string{"term.js", "xterm.js", "xterm.css", "addon-fit.js"} {
		res, err := http.Get(ts.URL + "/_si/term/assets/" + f)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(res.Body)
		res.Body.Close()
		if res.StatusCode != http.StatusOK || len(b) == 0 {
			t.Errorf("%s: status %d len %d", f, res.StatusCode, len(b))
		}
	}
}

// TestTerminalEchoRoundTrip spawns a shell tab, writes a command to its stdin,
// and reads the echoed output back off the streamed PTY output.
func TestTerminalEchoRoundTrip(t *testing.T) {
	ts := newTermServer(t, true)

	res, err := http.Post(ts.URL+"/_si/term/api/new", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusOK || len(id) == 0 {
		t.Fatalf("new tab: status %d id %q", res.StatusCode, id)
	}
	tabID := string(id)

	out, err := http.Get(ts.URL + "/_si/term/api/out?id=" + tabID)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Body.Close()

	found := make(chan bool, 1)
	go func() {
		buf := make([]byte, 4096)
		var acc strings.Builder
		for {
			n, err := out.Body.Read(buf)
			if n > 0 {
				acc.Write(buf[:n])
				if strings.Contains(acc.String(), "roachmark") {
					found <- true
					return
				}
			}
			if err != nil {
				found <- false
				return
			}
		}
	}()

	time.Sleep(150 * time.Millisecond) // let the shell come up
	in, err := http.Post(ts.URL+"/_si/term/api/in?id="+tabID, "application/octet-stream", strings.NewReader("echo roachmark\r"))
	if err != nil {
		t.Fatal(err)
	}
	in.Body.Close()

	select {
	case ok := <-found:
		if !ok {
			t.Fatal("output stream ended before the echo appeared")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for the shell to echo the command")
	}

	// Closing the tab must stop the stream / tear down the shell.
	cl, err := http.Post(ts.URL+"/_si/term/api/close?id="+tabID, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	cl.Body.Close()
	if cl.StatusCode != http.StatusNoContent {
		t.Errorf("close: status %d, want 204", cl.StatusCode)
	}
}

// TestTerminalRefusesOtherSites: another site's page (a blind POST from any tab
// in the browser) and a DNS-rebinding host (evil.example resolving to
// 127.0.0.1) must never start or drive a shell.
func TestTerminalRefusesOtherSites(t *testing.T) {
	ts := newTermServer(t, true)
	h := ts.Config.Handler
	try := func(method, path, host, origin string) int {
		req := httptest.NewRequest(method, path, strings.NewReader("echo hi\r"))
		req.Host = host
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}
	for _, c := range []struct{ method, path, host, origin string }{
		{"POST", "/_si/term/api/new", "127.0.0.1:7070", "https://evil.example"},
		{"POST", "/_si/term/api/in?id=1", "127.0.0.1:7070", "https://evil.example"},
		{"POST", "/_si/term/api/new", "evil.example:7070", "http://evil.example:7070"},
		{"GET", "/_si/term/api/out?id=1", "evil.example:7070", ""},
		{"POST", "/_si/callout", "evil.example:7070", "http://evil.example:7070"},
		{"POST", "/_si/term/api/new", "192.168.1.20:7070", "http://192.168.1.20:7070"},
	} {
		if code := try(c.method, c.path, c.host, c.origin); code != http.StatusForbidden {
			t.Errorf("%s %s Host %s Origin %q: %d, want 403", c.method, c.path, c.host, c.origin, code)
		}
	}
	if code := try("POST", "/_si/term/api/new", "localhost:7070", "http://localhost:7070"); code != http.StatusOK {
		t.Errorf("the drawer's own POST: %d", code)
	}
}

func TestTerminalFilesPanel(t *testing.T) {
	ts := newTermServer(t, true)
	getJSON := func(path string, v any) int {
		t.Helper()
		res, err := http.Get(ts.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode == http.StatusOK {
			json.NewDecoder(res.Body).Decode(v)
		}
		return res.StatusCode
	}
	var ls struct {
		Dir     string
		Entries []struct {
			Name string
			Dir  bool
		}
	}
	if getJSON("/_si/term/api/ls", &ls) != 200 || len(ls.Entries) != 1 || ls.Entries[0].Name != "talk.md" {
		t.Fatalf("ls of the deck folder: %+v", ls)
	}
	var cat struct {
		HTML  string
		Lines int
	}
	if getJSON("/_si/term/api/cat?path=talk.md", &cat) != 200 || !strings.Contains(cat.HTML, "Talk") {
		t.Fatalf("cat: %+v", cat)
	}
	if code := getJSON("/_si/term/api/raw?path=talk.md", &cat); code != http.StatusForbidden {
		t.Fatalf("raw serves pictures only, got %d", code)
	}
	// the active shell's folder follows its cd
	res, _ := http.Post(ts.URL+"/_si/term/api/new", "", nil)
	id, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if out, err := http.Get(ts.URL + "/_si/term/api/out?id=" + string(id)); err == nil { // drain, so the shell never blocks
		defer out.Body.Close()
		go io.Copy(io.Discard, out.Body)
	}
	sub := filepath.Join(ls.Dir, "sub")
	os.Mkdir(sub, 0o755)
	http.Post(ts.URL+"/_si/term/api/in?id="+string(id), "", strings.NewReader("cd sub\n"))
	var cwd struct{ Dir string }
	for i := 0; i < 100; i++ {
		time.Sleep(100 * time.Millisecond)
		getJSON("/_si/term/api/cwd?id="+string(id), &cwd)
		if r, _ := filepath.EvalSymlinks(cwd.Dir); strings.HasSuffix(r, "/sub") {
			return
		}
	}
	t.Fatalf("cwd did not follow cd: %q", cwd.Dir)
}

// TestTerminalFilesSave covers the Viewer's editor: cat ?edit=1 gives the text
// and mtime, save writes it, a stale mtime is a 409 unless force, a folder is refused.
func TestTerminalFilesSave(t *testing.T) {
	ts := newTermServer(t, true)
	type catRes struct {
		Path, Text, Mtime string
	}
	cat := func() catRes {
		t.Helper()
		res, err := http.Get(ts.URL + "/_si/term/api/cat?edit=1&path=talk.md")
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		var c catRes
		json.NewDecoder(res.Body).Decode(&c)
		return c
	}
	save := func(body map[string]any) (int, string) {
		t.Helper()
		b, _ := json.Marshal(body)
		res, err := http.Post(ts.URL+"/_si/term/api/save", "application/json", strings.NewReader(string(b)))
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		out, _ := io.ReadAll(res.Body)
		return res.StatusCode, string(out)
	}
	c := cat()
	if c.Text != "# Talk\n" || c.Mtime == "" {
		t.Fatalf("cat ?edit=1: %+v", c)
	}
	os.Chmod(c.Path, 0o600)
	code, out := save(map[string]any{"path": "talk.md", "text": "# Talk\n\nmore\n", "mtime": c.Mtime})
	if code != 200 {
		t.Fatalf("save: %d %s", code, out)
	}
	if b, _ := os.ReadFile(c.Path); string(b) != "# Talk\n\nmore\n" {
		t.Fatalf("saved text: %q", b)
	}
	if st, _ := os.Stat(c.Path); st.Mode().Perm() != 0o600 {
		t.Fatalf("mode not kept: %v", st.Mode())
	}
	var saved struct{ Mtime string }
	json.Unmarshal([]byte(out), &saved)
	if saved.Mtime != cat().Mtime {
		t.Fatalf("save returned mtime %q, file has %q", saved.Mtime, cat().Mtime)
	}
	// a stale mtime (someone else wrote it): conflict, then force
	if code, out = save(map[string]any{"path": "talk.md", "text": "stale", "mtime": c.Mtime}); code != http.StatusConflict || !strings.Contains(out, "changed on disk") {
		t.Fatalf("stale save: %d %s", code, out)
	}
	if code, _ = save(map[string]any{"path": "talk.md", "text": "forced", "mtime": c.Mtime, "force": true}); code != 200 {
		t.Fatalf("forced save: %d", code)
	}
	if b, _ := os.ReadFile(c.Path); string(b) != "forced" {
		t.Fatalf("forced text: %q", b)
	}
	os.Mkdir(filepath.Join(filepath.Dir(c.Path), "sub"), 0o755)
	if code, _ = save(map[string]any{"path": "sub", "text": "x", "force": true}); code != http.StatusBadRequest {
		t.Fatalf("saving over a folder: %d", code)
	}
	if ents, _ := os.ReadDir(filepath.Dir(c.Path)); len(ents) != 2 {
		t.Fatalf("temp files left behind: %v", ents)
	}
}

// TestTerminalSaveDoesNotReload: a Viewer save in the deck folder is no live
// reload (that would close the drawer); a write from elsewhere still is.
func TestTerminalSaveDoesNotReload(t *testing.T) {
	dir := t.TempDir()
	deck := filepath.Join(dir, "talk.md")
	os.WriteFile(deck, []byte("# Talk\n"), 0o644)
	s, err := New(dir, nil, "", "")
	if err != nil {
		t.Fatal(err)
	}
	s.Terminal = true
	s.Interval = 20 * time.Millisecond
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)
	stop := make(chan struct{})
	defer close(stop)
	go s.Watch(stop)
	res, err := http.Get(ts.URL + "/_si/events")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	reloads := make(chan struct{}, 4)
	go func() {
		sc := bufio.NewScanner(res.Body)
		for sc.Scan() {
			if sc.Text() == "event: reload" {
				reloads <- struct{}{}
			}
		}
	}()
	time.Sleep(100 * time.Millisecond)
	b, _ := json.Marshal(map[string]any{"path": "talk.md", "text": "# Talk\n\nedited\n", "force": true})
	if r, err := http.Post(ts.URL+"/_si/term/api/save", "application/json", strings.NewReader(string(b))); err != nil || r.StatusCode != 200 {
		t.Fatalf("save: %v %v", err, r)
	}
	select {
	case <-reloads:
		t.Fatal("a Viewer save reloaded the page")
	case <-time.After(300 * time.Millisecond):
	}
	os.WriteFile(deck, []byte("# Talk\n\nfrom a shell\n"), 0o644)
	select {
	case <-reloads:
	case <-time.After(3 * time.Second):
		t.Fatal("no reload after a write from outside the Viewer")
	}
}
