package serve

import (
	"encoding/json"
	"io"
	"net/http"
	"regexp"
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
	mu     sync.Mutex
	ballot map[string]map[string][]int // poll -> voter -> choices
	at     map[string]time.Time        // poll+voter -> last change
}

func newPolls() *polls {
	return &polls{ballot: map[string]map[string][]int{}, at: map[string]time.Time{}}
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
	counts := make([]int, maxOptions)
	for _, cs := range b {
		for _, c := range cs {
			counts[c]++
		}
	}
	msg, _ := json.Marshal(map[string]any{"type": "poll", "id": poll, "counts": counts, "total": len(b)})
	return msg, http.StatusNoContent
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
