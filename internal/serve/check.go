package serve

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/felipeg48/simple-ideas/internal/config"
	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/felipeg48/simple-ideas/internal/webhook"
)

// Labs that check themselves, and the relay for a course's result events
// (docs/superpowers/specs/2026-11-14-results-webhooks.md).

// checkTimeout is how long a lab check may run.
const checkTimeout = 30 * time.Second

// checkOutputMax is how much of a check's output comes back.
const checkOutputMax = 64 << 10

// checkResult is what POST /_si/check answers.
type checkResult struct {
	Pass     bool   `json:"pass"`
	Exit     int    `json:"exit"`
	Output   string `json:"output"`
	MS       int64  `json:"ms"`
	TimedOut bool   `json:"timedOut"`
}

// capped keeps the first max bytes written to it and drops the rest.
type capped struct {
	b   []byte
	max int
}

func (c *capped) Write(p []byte) (int, error) {
	if room := c.max - len(c.b); room > 0 {
		c.b = append(c.b, p[:min(room, len(p))]...)
	}
	return len(p), nil
}

// runCheck runs a lab check in its own process (sh -c, cmd /C on Windows)
// with dir as the working folder: never in the drawer's shell, so it cannot
// depend on or disturb what the presenter typed there.
func runCheck(dir string, c deck.Check, timeout time.Duration) checkResult {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", c.Cmd)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", c.Cmd)
	}
	cmd.Dir = dir
	cmd.WaitDelay = time.Second // a child that keeps the pipes open does not hold the answer
	out := &capped{max: checkOutputMax}
	cmd.Stdout, cmd.Stderr = out, out
	start := time.Now()
	err := cmd.Run()
	r := checkResult{MS: time.Since(start).Milliseconds()}
	var ee *exec.ExitError
	switch {
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		r.TimedOut, r.Exit = true, -1
	case errors.As(err, &ee):
		r.Exit = ee.ExitCode()
	case err != nil:
		r.Exit = -1
		out.Write([]byte(err.Error()))
	}
	r.Output = string(out.b)
	r.Pass = !r.TimedOut && r.Exit == c.Exit && (c.Expect == "" || strings.Contains(r.Output, c.Expect))
	return r
}

// check is POST /_si/check {page, cmd}: it runs cmd only when it is one of
// that page's own <!-- check: --> commands (read again from the file), so
// the page cannot ask for anything else. Loopback si play only.
func (s *Server) check(w http.ResponseWriter, r *http.Request) {
	var req struct{ Page, Cmd string }
	if err := json.NewDecoder(io.LimitReader(r.Body, 16<<10)).Decode(&req); err != nil {
		http.Error(w, "want {page, cmd}", http.StatusBadRequest)
		return
	}
	md := s.source(strings.TrimSuffix(path.Clean("/"+req.Page), ".html"))
	if md == "" {
		http.Error(w, "no such page", http.StatusNotFound)
		return
	}
	d, err := deck.Parse(md)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, c := range d.Checks() {
		if c.Cmd == req.Cmd {
			res := runCheck(filepath.Dir(md), c, checkTimeout)
			log.Printf("si play: check %q: pass=%v exit=%d (%d ms)", c.Cmd, res.Pass, res.Exit, res.MS)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(res)
			return
		}
	}
	http.Error(w, "not a check of this page", http.StatusForbidden)
}

// hookRelay is POST /_si/relay?page=…: a course page's result event,
// already in its format, forwarded to that course's webhook (webhook: in
// _course.md, else course.webhook in si.conf; never an address the page
// names) and signed with webhook-secret. On a loopback address, or anywhere
// with --relay.
func (s *Server) hookRelay(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	p := path.Clean("/" + r.URL.Query().Get("page"))
	file := filepath.Join(s.dir, filepath.FromSlash(p))
	if md := s.source(strings.TrimSuffix(p, ".html")); md != "" {
		file = md
	}
	g := deck.FindGuide(file, s.dir)
	if g == nil || !g.IsCourse() {
		http.Error(w, "not a course page", http.StatusNotFound)
		return
	}
	cfg := config.Load(s.Conf, filepath.Join(s.dir, "si.conf"))
	to := g.Meta.Webhook
	if to == "" || to == "relay" {
		to = cfg.CourseWebhook
	}
	if to == "" || to == "relay" {
		http.Error(w, "the course has no webhook address (course.webhook in si.conf)", http.StatusNotFound)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 64<<10))
	if err != nil || !json.Valid(body) {
		http.Error(w, "want a JSON event", http.StatusBadRequest)
		return
	}
	format := g.Meta.WebhookFormat
	if format == "" {
		format = "json"
	}
	if err := webhook.Post(to, format, body, webhook.Secret(cfg.WebhookSecret), g.Meta.WebhookAuth); err != nil {
		log.Printf("si: relay: %v", err)
		http.Error(w, "the webhook did not take it", http.StatusBadGateway)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}
