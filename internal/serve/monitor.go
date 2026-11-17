package serve

// The monitor: what si play's terminal monitor (m) and the web monitor, the
// control room at /_si/monitor, show. Both read Stats, so they never
// disagree. The web page also moderates what the audience gets (polls, Q&A,
// word clouds) through the same state and relay messages the teleprompter
// uses. It answers on the loopback listener with the run's key, or on the
// --remote listener behind the presenter key when the monitor role is on.

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"maps"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/felipeg48/simple-ideas/internal/assets"
	"github.com/felipeg48/simple-ideas/internal/config"
	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/felipeg48/simple-ideas/internal/render"
	"github.com/felipeg48/simple-ideas/internal/ui"
)

// MonitorInfo is what the monitor shows that only the command knows.
type MonitorInfo struct {
	Key         string // the run's key of /_si/monitor on the loopback; "" turns the page off
	Target      string // what is served, as typed: examples/
	Open, Phone string // this machine's link, the network's ("" without --addr 0.0.0.0)
	Web         string // the web monitor's own link (with its key)
	Start       time.Time
	Links       []render.RemoteLink // the links with their QR codes (the --remote ones come from RemoteQR)
	Log         func() []ui.LogLine // recent reloads and warnings
}

const monitorCookie = "si_monitor"

// pageRoles are the roles a live-reload stream may name (?role=); anything
// else counts as a page.
var pageRoles = map[string]bool{"deck": true, "teleprompter": true, "tp": true, "draw": true, "clicker": true, "audience": true, "review": true}

func pageRole(r string) string {
	if pageRoles[r] {
		return r
	}
	return "page"
}

// Stats is the monitor's numbers, the same for the terminal and the web:
// CPU and memory are sampled at most about once a second whoever asks.
func (s *Server) Stats() ui.Stats {
	s.monMu.Lock()
	if now := time.Now(); now.Sub(s.sampled) >= 900*time.Millisecond {
		s.smp.Sample(&s.cpu, now, 12)
		s.sampled = now
	}
	st := ui.Stats{CPU: s.cpu.CPU, Mem: s.cpu.Mem, CPUHist: slices.Clone(s.cpu.CPUHist), MemHist: slices.Clone(s.cpu.MemHist)}
	s.monMu.Unlock()
	l := s.Live()
	st.Target, st.Open, st.Phone, st.Web = s.Mon.Target, s.Mon.Open, s.Mon.Phone, s.Mon.Web
	if !s.Mon.Start.IsZero() {
		st.Up = time.Since(s.Mon.Start)
	}
	st.Shells, st.Pages, st.Files, st.Roles = l.Shells, l.Pages, l.Files, l.Roles
	st.Slide, st.Slides, st.Step, st.Steps = l.Slide, l.Slides, l.Step, l.Steps
	if !l.SlideAt.IsZero() {
		st.SlideFor, st.TalkFor = time.Since(l.SlideAt), time.Since(l.TalkAt)
	}
	if s.Mon.Log != nil {
		st.Log = s.Mon.Log()
	}
	return st
}

// monitorRoutes adds the web monitor to mux, each handler behind gate.
func (s *Server) monitorRoutes(mux *http.ServeMux, gate func(http.HandlerFunc) http.HandlerFunc) {
	mux.HandleFunc("GET /_si/monitor", gate(s.monitorPage))
	mux.HandleFunc("GET /_si/monitor/data", gate(s.monitorData))
	mux.HandleFunc("GET /_si/monitor/events", gate(s.monitorEvents))
	mux.HandleFunc("POST /_si/monitor/poll", gate(s.monitorPoll))
	mux.HandleFunc("POST /_si/monitor/qa", gate(s.qaState))
	mux.HandleFunc("POST /_si/monitor/cloud", gate(s.cloudHide))
}

// monitorLocal lets the loopback page through: a loopback peer and Host, the
// same origin, and the run's key (?k=, then its cookie).
func (s *Server) monitorLocal(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		host, _, _ := net.SplitHostPort(r.RemoteAddr)
		ip := net.ParseIP(host)
		if ip == nil || !ip.IsLoopback() || !loopbackHost(r.Host) || !sameOrigin(r) || s.Mon.Key == "" || !admit(w, r, s.Mon.Key, monitorCookie) {
			http.Error(w, "forbidden: open the monitor link si play printed (it carries the key)", http.StatusForbidden)
			return
		}
		monitorHeaders(w)
		h(w, r)
	}
}

// monitorRemote is the gate on the --remote listener, whose own gate already
// checked the presenter key: the same origin only.
func monitorRemote(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !sameOrigin(r) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		monitorHeaders(w)
		h(w, r)
	}
}

func monitorHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Frame-Options", "DENY") // no other page frames the buttons
}

func (s *Server) monitorPage(w http.ResponseWriter, r *http.Request) {
	theme := s.Theme
	if theme == "" && s.file != "" {
		if d, err := deck.Parse(s.file); err == nil {
			theme = d.Meta.Theme
		}
	}
	if theme == "" {
		theme = config.Load(s.Conf, filepath.Join(s.dir, "si.conf")).Theme
	}
	_, css, err := render.ThemeLayer(theme, s.dir)
	if err != nil {
		log.Printf("si: monitor: %v", err)
	}
	page := strings.Replace(assets.MonitorHTML, "/*SI-THEME*/", strings.ReplaceAll(css, "</", `<\/`), 1)
	title := s.Mon.Target
	if s.file != "" {
		if d, err := deck.Parse(s.file); err == nil && d.Title() != "" {
			title = d.Title()
		}
	}
	page = strings.ReplaceAll(page, "{{TITLE}}", html.EscapeString(title))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	io.WriteString(w, page)
}

// monitorSnapshot is everything the web monitor draws, once.
func (s *Server) monitorSnapshot() []byte {
	st := s.Stats()
	type link struct {
		Name string `json:"name"`
		URL  string `json:"url"`
		SVG  string `json:"svg,omitempty"`
	}
	var links []link
	for _, l := range s.Mon.Links {
		links = append(links, link{l.Name, l.URL, l.SVG})
	}
	if s.RemoteQR != nil {
		for _, l := range s.RemoteQR.Links {
			links = append(links, link{l.Name, l.URL, l.SVG})
		}
	}
	type logLine struct {
		At   string `json:"at"`
		St   string `json:"st"`
		Text string `json:"text"`
	}
	var logs []logLine
	for _, l := range st.Log {
		logs = append(logs, logLine{l.At.Format("15:04:05"), [...]string{"ok", "warn", "fail", "info"}[l.St], l.Text})
	}
	polls, clouds := s.audienceItems()
	s.qa.mu.Lock()
	qs, _ := json.Marshal(s.qa.list)
	s.qa.mu.Unlock()
	b, _ := json.Marshal(map[string]any{
		"target": st.Target, "up": int(st.Up.Seconds()), "cpu": st.CPU, "mem": st.Mem >> 20, "cpuHist": st.CPUHist, "memHist": st.MemHist,
		"shells": st.Shells, "files": st.Files, "pages": st.Pages, "roles": ui.RoleList(st.Roles),
		"slide": st.Slide, "slides": st.Slides, "step": st.Step, "steps": st.Steps,
		"slideFor": int(st.SlideFor.Seconds()), "talkFor": int(st.TalkFor.Seconds()),
		"links": links, "log": logs, "audience": s.AudienceKey != "",
		"polls": polls, "clouds": clouds, "questions": json.RawMessage(qs),
	})
	return b
}

func (s *Server) monitorData(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write(s.monitorSnapshot())
}

// monitorEvents streams the snapshot once a second (a monitor page counts as one).
func (s *Server) monitorEvents(w http.ResponseWriter, r *http.Request) {
	fl, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	sseHeaders(w)
	s.monitors.Add(1)
	defer s.monitors.Add(-1)
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		fmt.Fprintf(w, "data: %s\n\n", s.monitorSnapshot())
		fl.Flush()
		select {
		case <-r.Context().Done():
			return
		case <-t.C:
		}
	}
}

// monitorPoll is POST /_si/monitor/poll {poll, action}: open, close, reveal,
// conceal or reset a poll; every page gets the new state.
func (s *Server) monitorPoll(w http.ResponseWriter, r *http.Request) {
	var v struct{ Poll, Action string }
	if err := json.NewDecoder(io.LimitReader(r.Body, 1024)).Decode(&v); err != nil {
		http.Error(w, "want {poll, action}", http.StatusBadRequest)
		return
	}
	msg, ok := s.polls.control(v.Poll, v.Action)
	if !ok {
		http.Error(w, "no such poll or action (open, close, reveal, conceal, reset)", http.StatusBadRequest)
		return
	}
	s.relay.publish(msg, "poll:"+v.Poll)
	w.WriteHeader(http.StatusNoContent)
}

var audienceItemRE = regexp.MustCompile(`<!--\s*(poll|cloud):\s*([\s\S]*?)-->`)

type monPoll struct {
	ID       string   `json:"id"`
	Options  []string `json:"options"`
	Hidden   bool     `json:"hidden"` // · hidden: results on a click (or the monitor's Reveal)
	Counts   []int    `json:"counts"`
	Total    int      `json:"total"`
	Closed   bool     `json:"closed"`
	Revealed bool     `json:"revealed"`
}

type monCloud struct {
	ID     string      `json:"id"`
	Words  []cloudWord `json:"words"`
	Hidden []cloudWord `json:"hidden"`
	Total  int         `json:"total"`
}

// audienceItems is the served talks' polls and clouds with their state: the
// ones in the Markdown (so a poll shows before its first vote), then any
// other the server has ballots for.
// ponytail: reads the decks on every snapshot (once a second per monitor); fine for a talk.
func (s *Server) audienceItems() ([]monPoll, []monCloud) {
	var polls []monPoll
	var cloudIDs []string
	seen := map[string]bool{}
	for _, p := range s.RemoteDecks() {
		md := filepath.Join(s.dir, filepath.FromSlash(strings.TrimSuffix(strings.TrimPrefix(p, "/"), ".html")+".md"))
		files := []string{md}
		if d, err := deck.Parse(md); err == nil {
			files = append(files, d.Includes...)
		}
		for _, f := range files {
			b, err := os.ReadFile(f)
			if err != nil {
				continue
			}
			for _, m := range audienceItemRE.FindAllStringSubmatch(string(b), -1) {
				parts := strings.Split(m[2], "·")
				id := strings.TrimSpace(parts[0])
				if !pollIDRE.MatchString(id) || seen[m[1]+id] {
					continue
				}
				seen[m[1]+id] = true
				if m[1] == "cloud" {
					cloudIDs = append(cloudIDs, id)
					continue
				}
				mp := monPoll{ID: id}
				for _, o := range parts[1:] {
					switch o = strings.TrimSpace(o); o {
					case "hidden":
						mp.Hidden = true
					case "multiple", "":
					default:
						mp.Options = append(mp.Options, o)
					}
				}
				polls = append(polls, mp)
			}
		}
	}
	s.polls.mu.Lock()
	for _, id := range slices.Sorted(maps.Keys(s.polls.ballot)) {
		if !seen["poll"+id] {
			polls = append(polls, monPoll{ID: id})
		}
	}
	for i := range polls {
		var m struct {
			Counts           []int
			Total            int
			Closed, Revealed bool
		}
		json.Unmarshal(s.polls.message(polls[i].ID), &m)
		polls[i].Counts, polls[i].Total, polls[i].Closed, polls[i].Revealed = m.Counts, m.Total, m.Closed, m.Revealed
	}
	s.polls.mu.Unlock()
	var clouds []monCloud
	s.clouds.mu.Lock()
	for _, id := range slices.Sorted(maps.Keys(s.clouds.ans)) {
		if !seen["cloud"+id] {
			cloudIDs = append(cloudIDs, id)
		}
	}
	for _, id := range cloudIDs {
		shown, hidden, n := s.clouds.count(id)
		clouds = append(clouds, monCloud{id, shown, hidden, n})
	}
	s.clouds.mu.Unlock()
	return polls, clouds
}
