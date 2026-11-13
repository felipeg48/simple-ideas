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
	"bytes"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/creack/pty"
)

// ringMax is how much recent output a tab keeps to replay when a page
// attaches again (a live reload rebuilds the page; the shell lives on).
const ringMax = 256 << 10

// termIdle is how long a shell with no page attached lives on (a closed
// browser tab, a reload that never came back) before it is closed.
const termIdle = 10 * time.Minute

// termSession is one terminal tab: a shell running under a PTY master. One
// goroutine (pump) reads the PTY into a ring buffer and forwards to the page
// attached now, if any; a newer attach replaces the older stream.
type termSession struct {
	pty     *os.File
	cmd     *exec.Cmd
	title   string
	created time.Time
	exited  chan struct{} // closed when the PTY reads end

	mu       sync.Mutex
	ring     []byte
	cut      bool // the ring dropped older output (replay starts at a line)
	sub      *termSub
	attached bool      // a page has attached at least once
	idle     time.Time // when the last page detached (or the tab was made)
}

// termSub is one attached out stream.
type termSub struct {
	ch       chan []byte
	replaced chan struct{} // closed when a newer stream takes over
	gone     chan struct{} // closed when this stream's handler returns
}

// termHub tracks the open terminal tabs. A real shell runs per tab, so the
// backend stays tiny: Ctrl-C, arrow keys, REPLs and full-screen apps all
// travel as raw bytes through the PTY master; we never parse commands.
type termHub struct {
	dir  string
	next atomic.Int64
	mu   sync.Mutex
	tabs map[string]*termSession
	// quiet runs a Viewer save so the live-reload watcher does not see it: a
	// reload would close the drawer, its shells and the editor.
	quiet   func(write func())
	idle    time.Duration // detached shells are closed after this
	janitor sync.Once
}

func newTermHub(dir string) *termHub {
	return &termHub{dir: dir, tabs: map[string]*termSession{}, quiet: func(f func()) { f() }, idle: termIdle}
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
	now := time.Now()
	s := &termSession{pty: f, cmd: cmd, title: filepath.Base(shell), created: now, idle: now, exited: make(chan struct{})}
	h.mu.Lock()
	h.tabs[id] = s
	h.mu.Unlock()
	go s.pump()
	go func() { // reap when the shell exits, after its last output is read
		cmd.Wait()
		select {
		case <-s.exited:
		case <-time.After(time.Second): // a background child still holds the PTY
		}
		h.drop(id)
	}()
	h.janitor.Do(func() {
		go func() {
			for range time.Tick(time.Minute) {
				h.reap(time.Now())
			}
		}()
	})
	w.Header().Set("Content-Type", "text/plain")
	io.WriteString(w, id)
}

// pump reads the PTY until it closes: into the ring, and to the attached page.
func (s *termSession) pump() {
	defer close(s.exited)
	buf := make([]byte, 32*1024)
	for {
		n, err := s.pty.Read(buf)
		if n > 0 {
			b := append([]byte(nil), buf[:n]...)
			s.mu.Lock()
			s.ring = append(s.ring, b...)
			if len(s.ring) > 2*ringMax { // trim now and then, not on every read
				s.ring = append([]byte(nil), s.ring[len(s.ring)-ringMax:]...)
				s.cut = true
			}
			sub := s.sub
			s.mu.Unlock()
			if sub != nil {
				select {
				case sub.ch <- b:
				case <-sub.gone:
				}
			}
		}
		if err != nil {
			return // PTY closed (shell exited) or read error
		}
	}
}

// attach makes sub the tab's one stream and returns the output to replay:
// the ring as it is now, so nothing is shown twice or lost in between.
func (s *termSession) attach(sub *termSub) (replay []byte, again bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sub != nil {
		close(s.sub.replaced)
	}
	s.sub = sub
	again, s.attached = s.attached, true
	r := s.ring
	if len(r) > ringMax {
		r = r[len(r)-ringMax:]
	}
	if s.cut || len(r) < len(s.ring) {
		if i := bytes.IndexByte(r, '\n'); i >= 0 { // start on a whole line
			r = r[i+1:]
		}
	}
	return append([]byte(nil), r...), again
}

// detach ends sub; the tab goes idle unless a newer stream took over.
func (s *termSession) detach(sub *termSub) {
	s.mu.Lock()
	if s.sub == sub {
		s.sub = nil
		s.idle = time.Now()
	}
	s.mu.Unlock()
	close(sub.gone)
}

// out streams the PTY output to the browser as a chunked byte stream; the
// client reads it with fetch().body.getReader() and feeds xterm.write(). It
// starts with the recent output (the screen a reloaded page lost); on a
// reattach X-Si-Replay says how many bytes that is, so the page does not
// answer old terminal queries in it. A newer out for the same tab ends this one.
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
	sub := &termSub{ch: make(chan []byte, 64), replaced: make(chan struct{}), gone: make(chan struct{})}
	replay, again := s.attach(sub)
	defer s.detach(sub)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if again {
		w.Header().Set("X-Si-Replay", strconv.Itoa(len(replay)))
	}
	w.WriteHeader(http.StatusOK)
	if len(replay) > 0 {
		w.Write(replay)
	}
	fl.Flush() // send headers so the client's fetch() resolves before data
	for {
		select {
		case b := <-sub.ch:
			if _, err := w.Write(b); err != nil {
				return
			}
			fl.Flush()
		case <-s.exited:
			for { // what the pump sent before it ended
				select {
				case b := <-sub.ch:
					w.Write(b)
				default:
					fl.Flush()
					return
				}
			}
		case <-sub.replaced:
			return
		case <-r.Context().Done():
			return
		}
	}
}

// list: GET -> [{id, title, created}], the shells alive now, oldest first, so
// a reloaded page can attach to them again.
func (h *termHub) list(w http.ResponseWriter, r *http.Request) {
	type tab struct {
		ID      string `json:"id"`
		Title   string `json:"title"`
		Created int64  `json:"created"`
	}
	h.mu.Lock()
	out := make([]tab, 0, len(h.tabs))
	for id, s := range h.tabs {
		out = append(out, tab{id, s.title, s.created.UnixMilli()})
	}
	h.mu.Unlock()
	sort.Slice(out, func(i, j int) bool {
		a, _ := strconv.Atoi(out[i].ID)
		b, _ := strconv.Atoi(out[j].ID)
		return a < b
	})
	writeJSON(w, out)
}

// reap closes the shells no page has been attached to for h.idle.
func (h *termHub) reap(now time.Time) {
	var ids []string
	h.mu.Lock()
	for id, s := range h.tabs {
		s.mu.Lock()
		if s.sub == nil && now.Sub(s.idle) >= h.idle {
			ids = append(ids, id)
		}
		s.mu.Unlock()
	}
	h.mu.Unlock()
	for _, id := range ids {
		h.drop(id)
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
