package webdemo

import (
	"math"
	"strings"
	"testing"
)

// Per-word captions: the words share the caption's time by letters; the
// VTT stamps each word after the first; one picture per word, its word
// marked, on screen until the next word starts.
func TestWordCaptions(t *testing.T) {
	c := caption{Text: "one two three", From: 1, To: 2.4}
	ws := wordTimes(c)
	if len(ws) != 3 || ws[0].At != 1 || ws[1].At != 1.4 || math.Abs(ws[2].At-1.8) > 1e-9 {
		t.Fatalf("times: %+v", ws)
	}
	v := vttWords([]caption{c})
	if !strings.Contains(v, "00:00:01.000 --> 00:00:02.400\none <00:00:01.400>two <00:00:01.800>three\n") {
		t.Errorf("vtt:\n%s", v)
	}
	var pages []string
	ps, err := wordPics([]caption{c}, func(page string, i int) (string, error) {
		pages = append(pages, page)
		return "w" + string(rune('0'+i)), nil
	})
	if err != nil || len(ps) != 3 || ps[1].PNG != "w1" || ps[1].From != 1.4 || math.Abs(ps[1].To-1.8) > 1e-9 || ps[2].To != 2.4 {
		t.Fatalf("pics: %+v %v", ps, err)
	}
	if !strings.Contains(pages[1], "one <mark>two</mark> three") || strings.Contains(pages[0], "<mark>two") {
		t.Errorf("page: %s", pages[1])
	}
}
