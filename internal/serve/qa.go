package serve

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// Q&A (si play --audience): an audience phone sends a question to
// /_si/audience/ask; the presenter's pages get the list as a "questions"
// message (the audience stream never carries it) and set a question's state
// through /_si/qa: shown (over the projected slide), answered, hidden, open.

const (
	maxQuestion     = 500 // characters
	maxQuestionName = 40
	maxQuestions    = 200
	askMinBetween   = 20 * time.Second
)

type question struct {
	ID    int       `json:"id"`
	Text  string    `json:"text"`
	Name  string    `json:"name,omitempty"`
	State string    `json:"state"` // open, shown, answered, hidden
	At    time.Time `json:"at"`
}

type qa struct {
	mu   sync.Mutex
	list []*question
	last map[string]time.Time // voter -> last question
}

func newQA() *qa { return &qa{last: map[string]time.Time{}} }

// clean trims a phone's text to one tidy line set: no control characters but
// newlines, at most n characters.
func clean(s string, n int, newlines bool) string {
	s = strings.Map(func(r rune) rune {
		if r == '\n' && newlines {
			return r
		}
		if r < 0x20 || r == 0x7f {
			return ' '
		}
		return r
	}, strings.TrimSpace(s))
	if utf8.RuneCountInString(s) > n {
		s = string([]rune(s)[:n])
	}
	return strings.TrimSpace(s)
}

func (q *qa) ask(text, name, voter string, now time.Time) ([]byte, int) {
	text, name = clean(text, maxQuestion, true), clean(name, maxQuestionName, false)
	if text == "" || !voterRE.MatchString(voter) {
		return nil, http.StatusBadRequest
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	if t, ok := q.last[voter]; ok && now.Sub(t) < askMinBetween {
		return nil, http.StatusTooManyRequests
	}
	open := 0
	for _, x := range q.list {
		if x.State == "open" || x.State == "shown" {
			open++
		}
	}
	if open >= maxQuestions {
		return nil, http.StatusTooManyRequests
	}
	q.last[voter] = now
	q.list = append(q.list, &question{ID: len(q.list) + 1, Text: text, Name: name, State: "open", At: now})
	return q.message(), http.StatusNoContent
}

// set changes a question's state; showing one puts any other shown one back
// to open (one question over the slide at a time).
func (q *qa) set(id int, state string) ([]byte, bool) {
	switch state {
	case "open", "shown", "answered", "hidden":
	default:
		return nil, false
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	var hit *question
	for _, x := range q.list {
		if x.ID == id {
			hit = x
		}
	}
	if hit == nil {
		return nil, false
	}
	if state == "shown" {
		for _, x := range q.list {
			if x.State == "shown" {
				x.State = "open"
			}
		}
	}
	hit.State = state
	return q.message(), true
}

// message is the whole list for the presenter's pages (callers hold mu).
func (q *qa) message() []byte {
	b, _ := json.Marshal(map[string]any{"type": "questions", "list": q.list})
	return b
}

// ask is POST /_si/audience/ask {text, name, voter} (audience key only).
func (s *Server) ask(w http.ResponseWriter, r *http.Request) {
	var v struct{ Text, Name, Voter string }
	if err := json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&v); err != nil {
		http.Error(w, "want {text, name, voter}", http.StatusBadRequest)
		return
	}
	msg, code := s.qa.ask(v.Text, v.Name, v.Voter, time.Now())
	if msg == nil {
		http.Error(w, http.StatusText(code), code)
		return
	}
	s.relay.publish(msg, "questions")
	w.WriteHeader(code)
}

// qaState is POST /_si/qa {id, state} from the presenter's pages (the
// loopback deck and teleprompter, or the presenter key on the remote port).
func (s *Server) qaState(w http.ResponseWriter, r *http.Request) {
	var v struct {
		ID    int    `json:"id"`
		State string `json:"state"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1024)).Decode(&v); err != nil {
		http.Error(w, "want {id, state}", http.StatusBadRequest)
		return
	}
	msg, ok := s.qa.set(v.ID, v.State)
	if !ok {
		http.Error(w, "no such question or state", http.StatusBadRequest)
		return
	}
	s.relay.publish(msg, "questions")
	w.WriteHeader(http.StatusNoContent)
}

// SaveAudience writes the run's poll counts and questions beside the served
// folder, in .si/audience/<name>-<date-time>.json, when there were any; it
// returns the path ("" when there was nothing to keep).
func (s *Server) SaveAudience(name string, now time.Time) (string, error) {
	s.polls.mu.Lock()
	results := map[string]any{}
	for id, b := range s.polls.ballot {
		counts := make([]int, maxOptions)
		for _, cs := range b {
			for _, c := range cs {
				counts[c]++
			}
		}
		results[id] = map[string]any{"counts": counts, "voters": len(b)}
	}
	s.polls.mu.Unlock()
	s.qa.mu.Lock()
	questions := append([]*question(nil), s.qa.list...)
	s.qa.mu.Unlock()
	if len(results) == 0 && len(questions) == 0 {
		return "", nil
	}
	dir := filepath.Join(s.dir, ".si", "audience")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	b, _ := json.MarshalIndent(map[string]any{"talk": name, "ended": now.Format(time.RFC3339), "polls": results, "questions": questions}, "", "  ")
	p := filepath.Join(dir, name+"-"+now.Format("2006-01-02-1504")+".json")
	return p, os.WriteFile(p, b, 0o644)
}
