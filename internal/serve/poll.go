package serve

import (
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Polls (si play --audience): an audience phone POSTs its choices to
// /_si/audience/vote; the server keeps one ballot per phone per poll and
// sends the counts to every page as a "poll" message (the projected slide
// grows its bars, phones show results after voting). In memory, per run.

var (
	pollIDRE = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,39}$`) // as render.pollIDRE
	voterRE  = regexp.MustCompile(`^[a-z0-9]{8,40}$`)
)

const (
	maxPolls       = 100
	maxVotersPoll  = 5000
	maxOptions     = 6
	voteMinBetween = 300 * time.Millisecond // one change per phone at a time
)

type polls struct {
	mu       sync.Mutex
	ballot   map[string]map[string][]int // poll -> voter -> choices
	at       map[string]time.Time        // poll+voter -> last change
	closed   map[string]bool             // the monitor closed it: no more votes
	revealed map[string]bool             // the monitor showed a · hidden poll's results
	round    map[string]int              // the monitor's resets: a phone drops a vote from an earlier round
}

func newPolls() *polls {
	return &polls{ballot: map[string]map[string][]int{}, at: map[string]time.Time{},
		closed: map[string]bool{}, revealed: map[string]bool{}, round: map[string]int{}}
}

// message is a poll's counts and state for every page (callers hold mu).
func (p *polls) message(poll string) []byte {
	b := p.ballot[poll]
	counts := make([]int, maxOptions)
	for _, cs := range b {
		for _, c := range cs {
			counts[c]++
		}
	}
	msg, _ := json.Marshal(map[string]any{"type": "poll", "id": poll, "counts": counts, "total": len(b),
		"closed": p.closed[poll], "revealed": p.revealed[poll], "round": p.round[poll]})
	return msg
}

// control is the monitor's poll buttons: open, close, reveal, conceal, reset.
func (p *polls) control(poll, action string) ([]byte, bool) {
	if !pollIDRE.MatchString(poll) {
		return nil, false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.ballot[poll]; !ok && len(p.ballot) >= maxPolls {
		return nil, false
	}
	switch action {
	case "open", "close":
		p.closed[poll] = action == "close"
	case "reveal", "conceal":
		p.revealed[poll] = action == "reveal"
	case "reset":
		p.round[poll]++
		delete(p.ballot, poll)
		for k := range p.at {
			if strings.HasPrefix(k, poll+"/") {
				delete(p.at, k)
			}
		}
	default:
		return nil, false
	}
	if p.ballot[poll] == nil {
		p.ballot[poll] = map[string][]int{}
	}
	return p.message(poll), true
}

// vote applies one ballot and returns the poll message to send, or an HTTP
// status for a refused ballot.
func (p *polls) vote(poll, voter string, choices []int, now time.Time) ([]byte, int) {
	if !pollIDRE.MatchString(poll) || !voterRE.MatchString(voter) || len(choices) > maxOptions {
		return nil, http.StatusBadRequest
	}
	seen := map[int]bool{}
	for _, c := range choices {
		if c < 0 || c >= maxOptions || seen[c] {
			return nil, http.StatusBadRequest
		}
		seen[c] = true
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed[poll] {
		return nil, http.StatusConflict
	}
	if t, ok := p.at[poll+"/"+voter]; ok && now.Sub(t) < voteMinBetween {
		return nil, http.StatusTooManyRequests
	}
	b, ok := p.ballot[poll]
	if !ok {
		if len(p.ballot) >= maxPolls {
			return nil, http.StatusTooManyRequests
		}
		b = map[string][]int{}
		p.ballot[poll] = b
	}
	if _, known := b[voter]; !known && len(b) >= maxVotersPoll && len(choices) > 0 {
		return nil, http.StatusTooManyRequests
	}
	if len(choices) == 0 {
		delete(b, voter) // every choice taken back: no vote
	} else {
		b[voter] = append([]int(nil), choices...)
	}
	p.at[poll+"/"+voter] = now
	return p.message(poll), http.StatusNoContent
}

// vote is POST /_si/audience/vote {poll, choices, voter} (audience key only).
func (s *Server) vote(w http.ResponseWriter, r *http.Request) {
	var v struct {
		Poll    string `json:"poll"`
		Voter   string `json:"voter"`
		Choices []int  `json:"choices"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 2048)).Decode(&v); err != nil {
		http.Error(w, "want {poll, choices, voter}", http.StatusBadRequest)
		return
	}
	msg, code := s.polls.vote(v.Poll, v.Voter, v.Choices, time.Now())
	if msg == nil {
		http.Error(w, http.StatusText(code), code)
		return
	}
	s.relay.publish(msg, "poll:"+v.Poll)
	w.WriteHeader(code)
}
