package serve

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
)

// Where each deck open in a browser is (si play / si serve): the deck POSTs
// its position, a terminal teleprompter (si play deck.md --terminal --notes)
// follows it over server-sent events and sends next / prev / go back, which
// the deck receives on the same stream.

type position struct {
	Page  string `json:"page"`
	I     int    `json:"i"`
	K     int    `json:"k"`
	Steps int    `json:"steps"`
	N     int    `json:"n"`
}

type posEvent struct {
	kind, page string
	data       []byte
}

type positions struct {
	mu   sync.Mutex
	last map[string][]byte // page -> its latest position
	subs map[chan posEvent]string
}

func newPositions() *positions {
	return &positions{last: map[string][]byte{}, subs: map[chan posEvent]string{}}
}

func (p *positions) publish(e posEvent) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if e.kind == "state" {
		p.last[e.page] = e.data
	}
	for c, page := range p.subs {
		if page == "" || page == e.page {
			select {
			case c <- e:
			default: // a slow reader misses a step, never blocks the deck
			}
		}
	}
}

// sameOrigin refuses a POST that a page from another site sends to the loopback server.
func sameOrigin(r *http.Request) bool {
	o := r.Header.Get("Origin")
	return o == "" || o == "http://"+r.Host || o == "https://"+r.Host
}

func (p *positions) post(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || !sameOrigin(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	b, _ := io.ReadAll(io.LimitReader(r.Body, 4096))
	var pos position
	if json.Unmarshal(b, &pos) != nil || pos.Page == "" {
		http.Error(w, "bad position", http.StatusBadRequest)
		return
	}
	data, _ := json.Marshal(pos)
	p.publish(posEvent{"state", pos.Page, data})
	w.WriteHeader(http.StatusNoContent)
}

func (p *positions) command(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || !sameOrigin(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	b, _ := io.ReadAll(io.LimitReader(r.Body, 1024))
	var c struct {
		Page string `json:"page"`
		Cmd  string `json:"cmd"`
		I    *int   `json:"i,omitempty"`
	}
	if json.Unmarshal(b, &c) != nil || c.Page == "" || (c.Cmd != "next" && c.Cmd != "prev" && c.Cmd != "go" && c.Cmd != "hello") {
		http.Error(w, "bad command", http.StatusBadRequest)
		return
	}
	data, _ := json.Marshal(map[string]any{"cmd": c.Cmd, "i": c.I})
	p.publish(posEvent{"cmd", c.Page, data})
	w.WriteHeader(http.StatusNoContent)
}

func (p *positions) events(w http.ResponseWriter, r *http.Request) {
	fl, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	page := r.URL.Query().Get("page")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	c := make(chan posEvent, 16)
	p.mu.Lock()
	p.subs[c] = page
	last := p.last[page]
	p.mu.Unlock()
	defer func() {
		p.mu.Lock()
		delete(p.subs, c)
		p.mu.Unlock()
	}()
	fmt.Fprint(w, ": hello\n\n")
	if last != nil {
		fmt.Fprintf(w, "event: state\ndata: %s\n\n", last)
	}
	fl.Flush()
	for {
		select {
		case <-r.Context().Done():
			return
		case e := <-c:
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", e.kind, e.data)
			fl.Flush()
		}
	}
}
