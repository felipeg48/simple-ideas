package serve

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/felipeg48/simple-ideas/internal/sandbox"
)

// A lab's own Docker box in si play (RULES §37): the drawer's new tabs open
// shells in it (terminal.go), the page's checks run in it, and its
// sandbox-ports: open from links. The box itself is made by si play
// (cmd/si/sandbox.go) before the pages are served.

// box is si play's sandbox, or nil.
func (s *Server) box() *sandbox.Box {
	if s.Sandbox == nil {
		return nil
	}
	return s.Sandbox()
}

// pageBox is the sandbox a page's checks run in: si play's box, when the
// page asks for a sandbox (or the box is its course's).
func (s *Server) pageBox(d *deck.Deck) *sandbox.Box {
	if _, ok, _ := d.Meta.SandboxSpec(); !ok && !s.SandboxAll {
		return nil
	}
	return s.box()
}

// runBoxCheck runs a lab check with sh -c in the box's /lab.
func runBoxCheck(b *sandbox.Box, c deck.Check, timeout time.Duration) checkResult {
	start := time.Now()
	out, code, err := b.Exec(context.Background(), c.Cmd, timeout)
	r := checkResult{MS: time.Since(start).Milliseconds(), Exit: code, Output: out}
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		r.TimedOut, r.Exit = true, -1
	case err != nil:
		r.Exit, r.Output = -1, out+err.Error()
	}
	if len(r.Output) > checkOutputMax {
		r.Output = r.Output[:checkOutputMax]
	}
	r.Pass = !r.TimedOut && r.Exit == c.Exit && (c.Expect == "" || strings.Contains(r.Output, c.Expect))
	return r
}

// sandboxState is GET /_si/sandbox: {running, image, unprivileged, ports}.
func (s *Server) sandboxState(w http.ResponseWriter, r *http.Request) {
	type port struct {
		Port  int    `json:"port"`
		Label string `json:"label"`
		Addr  string `json:"addr"`
	}
	out := struct {
		Running      bool   `json:"running"`
		Image        string `json:"image,omitempty"`
		Unprivileged bool   `json:"unprivileged"`
		Ports        []port `json:"ports"`
	}{Ports: []port{}}
	if b := s.box(); b != nil {
		out.Running, out.Image, out.Unprivileged = true, b.Spec.Image, b.Runtime != ""
		for _, p := range b.Spec.Ports {
			out.Ports = append(out.Ports, port{p.Port, p.Label, b.Ports[p.Port].Addr})
		}
	}
	writeJSON(w, out)
}

// sandboxPort is GET /_si/sandbox/port/{port}/…: a redirect to where the box
// publishes that sandbox-ports: port, the rest of the path kept.
func (s *Server) sandboxPort(w http.ResponseWriter, r *http.Request) {
	b := s.box()
	n, _ := strconv.Atoi(r.PathValue("port"))
	if b == nil {
		http.Error(w, "no sandbox is running (si play starts the lab's box)", http.StatusNotFound)
		return
	}
	hp, ok := b.Ports[n]
	if !ok {
		http.Error(w, "the lab's sandbox-ports: has no port "+strconv.Itoa(n), http.StatusNotFound)
		return
	}
	rest := strings.TrimPrefix(r.URL.Path, "/_si/sandbox/port/"+strconv.Itoa(n))
	to := "http://" + hp.Addr + rest
	if r.URL.RawQuery != "" {
		to += "?" + r.URL.RawQuery
	}
	http.Redirect(w, r, to, http.StatusFound)
}
