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
	"sync"
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

// TestTerminalFilesOps covers the tree's create, rename, delete and stat:
// names taken are 409, names with a / are 400, a link is deleted, not its target.
func TestTerminalFilesOps(t *testing.T) {
	ts := newTermServer(t, true)
	op := func(name string, body map[string]any) (int, string) {
		t.Helper()
		b, _ := json.Marshal(body)
		res, err := http.Post(ts.URL+"/_si/term/api/"+name, "application/json", strings.NewReader(string(b)))
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		out, _ := io.ReadAll(res.Body)
		return res.StatusCode, string(out)
	}
	var ls struct{ Dir string }
	res, _ := http.Get(ts.URL + "/_si/term/api/ls")
	json.NewDecoder(res.Body).Decode(&ls)
	res.Body.Close()
	dir := ls.Dir
	exists := func(p string) bool { _, err := os.Lstat(filepath.Join(dir, p)); return err == nil }

	if code, out := op("create", map[string]any{"name": "notes.md"}); code != 200 || !strings.Contains(out, "notes.md") || !exists("notes.md") {
		t.Fatalf("create file: %d %s", code, out)
	}
	if code, _ := op("create", map[string]any{"name": "notes.md"}); code != http.StatusConflict {
		t.Fatalf("create over a file: %d, want 409", code)
	}
	if code, _ := op("create", map[string]any{"name": "conf", "folder": true}); code != 200 || !exists("conf") {
		t.Fatalf("create folder: %d", code)
	}
	if code, _ := op("create", map[string]any{"name": "conf", "folder": true}); code != http.StatusConflict {
		t.Fatalf("create folder twice: %d, want 409", code)
	}
	if code, _ := op("create", map[string]any{"dir": "conf", "name": "rabbitmq.conf"}); code != 200 || !exists("conf/rabbitmq.conf") {
		t.Fatalf("create in a folder: %d", code)
	}
	for _, bad := range []string{"a/b", "../x", "..", "", "a\\b"} {
		if code, _ := op("create", map[string]any{"name": bad}); code != http.StatusBadRequest {
			t.Errorf("create %q: %d, want 400", bad, code)
		}
		if code, _ := op("rename", map[string]any{"path": "notes.md", "name": bad}); code != http.StatusBadRequest {
			t.Errorf("rename to %q: %d, want 400", bad, code)
		}
	}
	if code, _ := op("rename", map[string]any{"path": "notes.md", "name": "talk.md"}); code != http.StatusConflict || !exists("notes.md") {
		t.Fatalf("rename onto a taken name: %d, want 409", code)
	}
	if code, out := op("rename", map[string]any{"path": "notes.md", "name": "plan.md"}); code != 200 || !exists("plan.md") || exists("notes.md") {
		t.Fatalf("rename: %d %s", code, out)
	}
	if code, _ := op("rename", map[string]any{"path": "gone.md", "name": "x.md"}); code != http.StatusNotFound {
		t.Fatalf("rename a missing file: %d, want 404", code)
	}
	var st struct {
		Dir   bool
		Items int
		Mtime string
	}
	res, _ = http.Get(ts.URL + "/_si/term/api/stat?count=1&path=conf")
	json.NewDecoder(res.Body).Decode(&st)
	res.Body.Close()
	if !st.Dir || st.Items != 1 || st.Mtime == "" {
		t.Fatalf("stat of a folder: %+v", st)
	}
	// a link to a folder: deleting it leaves the folder and its file
	if err := os.Symlink(filepath.Join(dir, "conf"), filepath.Join(dir, "link")); err != nil {
		t.Fatal(err)
	}
	if code, _ := op("delete", map[string]any{"path": "link"}); code != http.StatusNoContent || exists("link") || !exists("conf/rabbitmq.conf") {
		t.Fatalf("delete a link: %d", code)
	}
	if code, _ := op("delete", map[string]any{"path": "plan.md"}); code != http.StatusNoContent || exists("plan.md") {
		t.Fatalf("delete a file: %d", code)
	}
	if code, _ := op("delete", map[string]any{"path": "conf"}); code != http.StatusNoContent || exists("conf") {
		t.Fatalf("delete a folder with a file in it: %d", code)
	}
	if code, _ := op("delete", map[string]any{"path": "conf"}); code != http.StatusNotFound {
		t.Fatalf("delete twice: %d, want 404", code)
	}
	if code, _ := op("delete", map[string]any{"path": "/"}); code != http.StatusBadRequest {
		t.Fatalf("delete /: %d, want 400", code)
	}
}

// TestTerminalCatTail: ?tail=1 shows a big file's last MB, from a line start.
func TestTerminalCatTail(t *testing.T) {
	ts := newTermServer(t, true)
	var ls struct{ Dir string }
	res, _ := http.Get(ts.URL + "/_si/term/api/ls")
	json.NewDecoder(res.Body).Decode(&ls)
	res.Body.Close()
	var b strings.Builder
	for i := 0; b.Len() < 1<<20+5000; i++ {
		b.WriteString("line " + strings.Repeat("x", 40) + "\n")
	}
	b.WriteString("the last line\n")
	os.WriteFile(filepath.Join(ls.Dir, "big.log"), []byte(b.String()), 0o644)
	get := func(q string) (c struct {
		HTML            string
		Tail, Truncated bool
	}) {
		res, err := http.Get(ts.URL + "/_si/term/api/cat?path=big.log" + q)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		json.NewDecoder(res.Body).Decode(&c)
		return
	}
	if c := get(""); c.Tail || !c.Truncated || strings.Contains(c.HTML, "the last line") {
		t.Fatalf("without tail: the first MB, got tail=%v", c.Tail)
	}
	c := get("&tail=1")
	if !c.Tail || !c.Truncated || !strings.Contains(c.HTML, "the last line") || !strings.HasPrefix(c.HTML, "line ") {
		t.Fatalf("tail: tail=%v starts %q", c.Tail, c.HTML[:min(20, len(c.HTML))])
	}
}

// readUntil reads r until want shows up (true) or it ends / times out.
func readUntil(r io.Reader, want string, d time.Duration) (string, bool) {
	got := make(chan bool, 1)
	var acc strings.Builder
	var mu sync.Mutex
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := r.Read(buf)
			mu.Lock()
			acc.Write(buf[:n])
			ok := strings.Contains(acc.String(), want)
			mu.Unlock()
			if ok {
				got <- true
				return
			}
			if err != nil {
				got <- false
				return
			}
		}
	}()
	var ok bool
	select {
	case ok = <-got:
	case <-time.After(d):
	}
	mu.Lock()
	defer mu.Unlock()
	return acc.String(), ok
}

// TestTerminalReattach: a shell outlives its page. A second out stream (the
// reloaded page) gets the earlier output replayed and says so in
// X-Si-Replay; the older stream ends (latest wins, no output twice); list
// shows the shell.
func TestTerminalReattach(t *testing.T) {
	t.Setenv("SHELL", "/bin/sh") // quick to start
	ts := newTermServer(t, true)
	res, err := http.Post(ts.URL+"/_si/term/api/new", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(res.Body)
	res.Body.Close()
	id := string(b)

	first, err := http.Get(ts.URL + "/_si/term/api/out?id=" + id)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Body.Close()
	if first.Header.Get("X-Si-Replay") != "" {
		t.Error("the first attach is not a replay")
	}
	time.Sleep(150 * time.Millisecond)
	in, _ := http.Post(ts.URL+"/_si/term/api/in?id="+id, "", strings.NewReader("echo reload$((40+2))\r"))
	in.Body.Close()
	if _, ok := readUntil(first.Body, "reload42", 5*time.Second); !ok {
		t.Fatal("first stream: no echo")
	}

	second, err := http.Get(ts.URL + "/_si/term/api/out?id=" + id)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Body.Close()
	if second.Header.Get("X-Si-Replay") == "" {
		t.Error("reattach: no X-Si-Replay header")
	}
	if _, ok := readUntil(second.Body, "reload42", 5*time.Second); !ok {
		t.Fatal("reattach did not replay the earlier output")
	}
	// the first stream has ended: reading it returns EOF quickly
	done := make(chan struct{})
	go func() { io.Copy(io.Discard, first.Body); close(done) }()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Error("the older stream did not end when a newer one attached")
	}

	lr, err := http.Get(ts.URL + "/_si/term/api/list")
	if err != nil {
		t.Fatal(err)
	}
	var list []struct {
		ID      string `json:"id"`
		Title   string `json:"title"`
		Created int64  `json:"created"`
	}
	json.NewDecoder(lr.Body).Decode(&list)
	lr.Body.Close()
	if len(list) != 1 || list[0].ID != id || list[0].Title == "" || list[0].Created == 0 {
		t.Errorf("list = %+v", list)
	}
}

// TestTerminalIdleReap: a shell no page is attached to is closed after the
// idle time; an attached one is kept.
func TestTerminalIdleReap(t *testing.T) {
	h := newTermHub(t.TempDir())
	t.Cleanup(h.closeAll)
	h.idle = time.Minute
	newID := func() string {
		w := httptest.NewRecorder()
		h.newTab(w, httptest.NewRequest("POST", "/_si/term/api/new", nil))
		return w.Body.String()
	}
	idle, kept := newID(), newID()
	h.tabs[kept].attach(&termSub{ch: make(chan []byte, 64), replaced: make(chan struct{}), gone: make(chan struct{})})
	h.reap(time.Now()) // not yet idle long enough
	if len(h.tabs) != 2 {
		t.Fatalf("reaped too early: %d tabs", len(h.tabs))
	}
	h.reap(time.Now().Add(2 * time.Minute))
	h.mu.Lock()
	_, a := h.tabs[idle]
	_, b := h.tabs[kept]
	h.mu.Unlock()
	if a || !b {
		t.Errorf("after the idle time: idle tab alive %v, attached tab alive %v", a, b)
	}
}
