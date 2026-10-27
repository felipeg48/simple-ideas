package serve

import (
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
