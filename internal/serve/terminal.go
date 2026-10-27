// This file is the `si play` presenter terminal backend: one PTY-backed shell
// per tab, streamed to the xterm.js drawer in the browser over localhost.
//
// It is a deliberate, documented exception to the project's non-negotiable
// rules #2 (self-contained) and #7 (stdlib + goldmark only): it adds the
// github.com/creack/pty dependency and the drawer bundles xterm.js. The
// exception is contained two ways — it is **si play only** (a built, static
// .html has no server, so none of this exists there) and **loopback only**
// (Server.Terminal is set from the listen address; it is false on
// --addr 0.0.0.0, so the exec surface can never become LAN remote-code-exec).
package serve

import (
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/creack/pty"
)

// termSession is one terminal tab: a shell running under a PTY master.
type termSession struct {
	pty *os.File
	cmd *exec.Cmd
}

// termHub tracks the open terminal tabs. A real shell runs per tab, so the
// backend stays tiny: Ctrl-C, arrow keys, REPLs and full-screen apps all
// travel as raw bytes through the PTY master; we never parse commands.
type termHub struct {
	dir  string
	next atomic.Int64
	mu   sync.Mutex
	tabs map[string]*termSession
}

func newTermHub(dir string) *termHub {
	return &termHub{dir: dir, tabs: map[string]*termSession{}}
}

// newTab starts a shell in a fresh PTY and returns its tab id.
func (h *termHub) newTab(w http.ResponseWriter, r *http.Request) {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	cmd := exec.Command(shell)
	cmd.Dir = h.dir
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	f, err := pty.Start(cmd)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	id := strconv.FormatInt(h.next.Add(1), 10)
	h.mu.Lock()
	h.tabs[id] = &termSession{pty: f, cmd: cmd}
	h.mu.Unlock()
	go func() { cmd.Wait(); h.drop(id) }() // reap when the shell exits
	w.Header().Set("Content-Type", "text/plain")
	io.WriteString(w, id)
}

// out streams the PTY output to the browser as a chunked byte stream; the
// client reads it with fetch().body.getReader() and feeds xterm.write().
func (h *termHub) out(w http.ResponseWriter, r *http.Request) {
	s := h.get(r)
	if s == nil {
		http.NotFound(w, r)
		return
	}
	fl, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	fl.Flush() // send headers so the client's fetch() resolves before data
	buf := make([]byte, 32*1024)
	for {
		n, err := s.pty.Read(buf)
		if n > 0 {
			if _, werr := w.Write(buf[:n]); werr != nil {
				return
			}
			fl.Flush()
		}
		if err != nil {
			return // PTY closed (shell exited) or read error
		}
		select {
		case <-r.Context().Done():
			return
		default:
		}
	}
}

// in writes the request body (keystrokes, pastes, Ctrl-C's 0x03, ...) to the
// PTY master.
func (h *termHub) in(w http.ResponseWriter, r *http.Request) {
	s := h.get(r)
	if s == nil {
		http.NotFound(w, r)
		return
	}
	io.Copy(s.pty, r.Body)
	w.WriteHeader(http.StatusNoContent)
}

// resize applies the terminal's cols/rows so full-screen apps lay out right.
func (h *termHub) resize(w http.ResponseWriter, r *http.Request) {
	s := h.get(r)
	if s == nil {
		http.NotFound(w, r)
		return
	}
	cols, _ := strconv.Atoi(r.URL.Query().Get("cols"))
	rows, _ := strconv.Atoi(r.URL.Query().Get("rows"))
	if cols > 0 && rows > 0 {
		pty.Setsize(s.pty, &pty.Winsize{Rows: uint16(rows), Cols: uint16(cols)})
	}
	w.WriteHeader(http.StatusNoContent)
}

// close kills the tab's shell and drops the session.
func (h *termHub) close(w http.ResponseWriter, r *http.Request) {
	h.drop(r.URL.Query().Get("id"))
	w.WriteHeader(http.StatusNoContent)
}

func (h *termHub) get(r *http.Request) *termSession {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.tabs[r.URL.Query().Get("id")]
}

// drop removes a tab and tears down its shell. Safe to call twice (the close
// endpoint and the reaper goroutine race benignly).
func (h *termHub) drop(id string) {
	h.mu.Lock()
	s := h.tabs[id]
	delete(h.tabs, id)
	h.mu.Unlock()
	if s == nil {
		return
	}
	if s.cmd.Process != nil {
		s.cmd.Process.Kill()
	}
	s.pty.Close()
}

// closeAll tears down every open tab (called on server shutdown).
func (h *termHub) closeAll() {
	h.mu.Lock()
	ids := make([]string, 0, len(h.tabs))
	for id := range h.tabs {
		ids = append(ids, id)
	}
	h.mu.Unlock()
	for _, id := range ids {
		h.drop(id)
	}
}
