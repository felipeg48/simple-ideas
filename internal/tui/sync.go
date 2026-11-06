package tui

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// The presenter and its teleprompter talk over a Unix socket next to the
// temp files (SocketPath): the presenter sends its position as one JSON line
// whenever it moves; the teleprompter sends commands (next, prev, go, hello).

// state is where the presenter is.
type state struct {
	I     int   `json:"i"`     // slide, 0-based
	K     int   `json:"k"`     // build steps shown
	Steps int   `json:"steps"` // build steps of the slide
	N     int   `json:"n"`     // slides
	Start int64 `json:"start"` // when the talk started (unix seconds)
}

// command is a teleprompter's request.
type command struct {
	Cmd string `json:"cmd"`
	I   int    `json:"i,omitempty"`
}

// hub is the presenter's side: every connected teleprompter gets each state.
type hub struct {
	ln    net.Listener
	mu    sync.Mutex
	conns map[net.Conn]bool
	last  []byte
}

func listen(path string, onCommand func(command)) (*hub, error) {
	if c, err := net.Dial("unix", path); err == nil {
		c.Close() // another presenter of this deck is running: leave it be
		return nil, os.ErrExist
	}
	os.Remove(path) // a stale socket from a presenter that crashed
	ln, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}
	h := &hub{ln: ln, conns: map[net.Conn]bool{}}
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			h.mu.Lock()
			h.conns[c] = true
			if h.last != nil {
				c.Write(h.last)
			}
			h.mu.Unlock()
			go func() {
				sc := bufio.NewScanner(c)
				for sc.Scan() {
					var cmd command
					if json.Unmarshal(sc.Bytes(), &cmd) == nil {
						onCommand(cmd)
					}
				}
				h.mu.Lock()
				delete(h.conns, c)
				h.mu.Unlock()
				c.Close()
			}()
		}
	}()
	return h, nil
}

func (h *hub) send(s state) {
	b, _ := json.Marshal(s)
	b = append(b, '\n')
	h.mu.Lock()
	defer h.mu.Unlock()
	h.last = b
	for c := range h.conns {
		c.SetWriteDeadline(time.Now().Add(time.Second))
		if _, err := c.Write(b); err != nil {
			c.Close()
			delete(h.conns, c)
		}
	}
}

func (h *hub) close() {
	h.ln.Close()
	h.mu.Lock()
	for c := range h.conns {
		c.Close()
	}
	h.mu.Unlock()
}

// follower is the teleprompter's side: it keeps reconnecting, delivers each
// state, and sends commands while connected.
type follower struct {
	path    string
	onState func(state, bool) // the state, and whether the presenter is there
	mu      sync.Mutex
	conn    net.Conn
	// a deck in a browser (si play / si serve at base): its page, once it reports
	base, page string
}

func follow(path string, onState func(state, bool)) *follower {
	f := &follower{path: path, onState: onState}
	go func() {
		for {
			c, err := net.Dial("unix", path)
			if err != nil {
				f.mu.Lock()
				browser := f.page != ""
				f.mu.Unlock()
				if !browser { // following a browser deck is being connected too
					onState(state{}, false)
				}
				time.Sleep(time.Second)
				continue
			}
			f.mu.Lock()
			f.conn = c
			f.mu.Unlock()
			f.send(command{Cmd: "hello"})
			sc := bufio.NewScanner(c)
			for sc.Scan() {
				var s state
				if json.Unmarshal(sc.Bytes(), &s) == nil {
					onState(s, true)
				}
			}
			f.mu.Lock()
			f.conn = nil
			f.mu.Unlock()
			c.Close()
			onState(state{}, false)
			time.Sleep(time.Second)
		}
	}()
	return f
}

func (f *follower) send(c command) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.conn == nil && f.page != "" { // the browser deck, through the play server
		body, _ := json.Marshal(map[string]any{"page": f.page, "cmd": c.Cmd, "i": c.I})
		resp, err := http.Post(f.base+"/_si/position/cmd", "application/json", bytes.NewReader(body))
		if err == nil {
			resp.Body.Close()
		}
		return err == nil
	}
	if f.conn == nil {
		return false
	}
	b, _ := json.Marshal(c)
	_, err := f.conn.Write(append(b, '\n'))
	return err == nil
}

// followBrowser follows a deck open in a browser through the si play (or si
// serve) server at addr: the page whose path ends in /name.html, where name is
// the deck file's name.
func (f *follower) followBrowser(addr, deckFile string) {
	f.base = "http://" + addr
	want := "/" + strings.TrimSuffix(filepath.Base(deckFile), filepath.Ext(deckFile)) + ".html"
	go func() {
		for {
			resp, err := http.Get(f.base + "/_si/position/events")
			if err != nil || resp.StatusCode != http.StatusOK {
				if resp != nil {
					resp.Body.Close()
				}
				time.Sleep(2 * time.Second)
				continue
			}
			sc := bufio.NewScanner(resp.Body)
			event := ""
			for sc.Scan() {
				line := sc.Text()
				switch {
				case strings.HasPrefix(line, "event: "):
					event = strings.TrimPrefix(line, "event: ")
				case strings.HasPrefix(line, "data: ") && event == "state":
					var p struct {
						Page        string
						I, K, Steps int
						N           int
					}
					if json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &p) == nil && strings.HasSuffix(p.Page, want) {
						f.mu.Lock()
						f.page = p.Page
						socket := f.conn != nil
						f.mu.Unlock()
						if !socket { // a terminal presenter, when there is one, wins
							f.onState(state{I: p.I, K: p.K, Steps: p.Steps, N: p.N}, true)
						}
					}
				}
			}
			resp.Body.Close()
			time.Sleep(time.Second)
		}
	}()
}
