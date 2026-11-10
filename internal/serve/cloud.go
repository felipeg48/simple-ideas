package serve

import (
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// Word clouds (si play --audience): an audience phone sends a short answer to
// /_si/audience/word; the server counts the answers of every phone and sends
// a "cloud" message (the projected slide sizes each word by its count). Free
// text reaches the screen, so the presenter can hide a word (/_si/cloud): it
// leaves every cloud page at once. In memory, per run.

const (
	maxWord       = 30 // characters
	wordsPerPhone = 3  // a fourth answer replaces the oldest
	maxClouds     = 100
	maxCloudWords = 60 // the most frequent, on screen
	wordMinGap    = time.Second
)

type clouds struct {
	mu   sync.Mutex
	ans  map[string]map[string][]string // cloud -> voter -> answers
	hide map[string]map[string]bool     // cloud -> hidden words (lower case)
	at   map[string]time.Time           // cloud/voter -> last answer
}

func newClouds() *clouds {
	return &clouds{ans: map[string]map[string][]string{}, hide: map[string]map[string]bool{}, at: map[string]time.Time{}}
}

func (c *clouds) add(id, voter, text string, now time.Time) ([]byte, int) {
	text = clean(text, maxWord, false)
	if !pollIDRE.MatchString(id) || !voterRE.MatchString(voter) || text == "" {
		return nil, http.StatusBadRequest
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if t, ok := c.at[id+"/"+voter]; ok && now.Sub(t) < wordMinGap {
		return nil, http.StatusTooManyRequests
	}
	a, ok := c.ans[id]
	if !ok {
		if len(c.ans) >= maxClouds {
			return nil, http.StatusTooManyRequests
		}
		a = map[string][]string{}
		c.ans[id] = a
	}
	if _, known := a[voter]; !known && len(a) >= maxVotersPoll {
		return nil, http.StatusTooManyRequests
	}
	mine := append(a[voter], text)
	if len(mine) > wordsPerPhone {
		mine = mine[len(mine)-wordsPerPhone:]
	}
	a[voter] = mine
	c.at[id+"/"+voter] = now
	return c.message(id), http.StatusNoContent
}

func (c *clouds) setHidden(id, word string, hidden bool) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.ans[id]; !ok {
		return nil, false
	}
	if c.hide[id] == nil {
		c.hide[id] = map[string]bool{}
	}
	key := strings.ToLower(strings.TrimSpace(word))
	if hidden {
		c.hide[id][key] = true
	} else {
		delete(c.hide[id], key)
	}
	return c.message(id), true
}

type cloudWord struct {
	Word string `json:"w"`
	N    int    `json:"n"`
}

// count is the cloud's words, most frequent first, the same word in any case
// counted once; hidden words apart (callers hold mu).
func (c *clouds) count(id string) (shown, hidden []cloudWord, voters int) {
	n, first := map[string]int{}, map[string]string{}
	for _, ws := range c.ans[id] {
		for _, w := range ws {
			k := strings.ToLower(w)
			if f, ok := first[k]; !ok || w < f { // one spelling, the same on every run
				first[k] = w
			}
			n[k]++
		}
	}
	for k, v := range n {
		cw := cloudWord{first[k], v}
		if c.hide[id][k] {
			hidden = append(hidden, cw)
		} else {
			shown = append(shown, cw)
		}
	}
	sort.Slice(shown, func(i, j int) bool {
		if shown[i].N != shown[j].N {
			return shown[i].N > shown[j].N
		}
		return strings.ToLower(shown[i].Word) < strings.ToLower(shown[j].Word)
	})
	if len(shown) > maxCloudWords {
		shown = shown[:maxCloudWords]
	}
	return shown, hidden, len(c.ans[id])
}

// message is the cloud for every page; "hidden" is for the presenter's pages
// (audienceView drops it).
func (c *clouds) message(id string) []byte {
	shown, hidden, voters := c.count(id)
	if shown == nil {
		shown = []cloudWord{}
	}
	b, _ := json.Marshal(map[string]any{"type": "cloud", "id": id, "words": shown, "hidden": hidden, "total": voters})
	return b
}

// word is POST /_si/audience/word {cloud, text, voter} (audience key only).
func (s *Server) word(w http.ResponseWriter, r *http.Request) {
	var v struct{ Cloud, Text, Voter string }
	if err := json.NewDecoder(io.LimitReader(r.Body, 1024)).Decode(&v); err != nil {
		http.Error(w, "want {cloud, text, voter}", http.StatusBadRequest)
		return
	}
	msg, code := s.clouds.add(v.Cloud, v.Voter, v.Text, time.Now())
	if msg == nil {
		http.Error(w, http.StatusText(code), code)
		return
	}
	s.relay.publish(msg, "cloud:"+v.Cloud)
	w.WriteHeader(code)
}

// cloudHide is POST /_si/cloud {cloud, word, hide} from the presenter's pages.
func (s *Server) cloudHide(w http.ResponseWriter, r *http.Request) {
	var v struct {
		Cloud string `json:"cloud"`
		Word  string `json:"word"`
		Hide  bool   `json:"hide"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1024)).Decode(&v); err != nil {
		http.Error(w, "want {cloud, word, hide}", http.StatusBadRequest)
		return
	}
	msg, ok := s.clouds.setHidden(v.Cloud, v.Word, v.Hide)
	if !ok {
		http.Error(w, "no such cloud", http.StatusBadRequest)
		return
	}
	s.relay.publish(msg, "cloud:"+v.Cloud)
	w.WriteHeader(http.StatusNoContent)
}
