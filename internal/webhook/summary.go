package webhook

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Failure is one failed step of a demo test run.
type Failure struct {
	Script  string `json:"script"`
	Feature string `json:"feature"`
	Step    string `json:"step"`
}

// Summary is a si demo test run, the demo.test event.
type Summary struct {
	Version  int       `json:"version"`
	Event    string    `json:"event"`
	Title    string    `json:"title"`
	Passed   int       `json:"passed"`
	Failed   int       `json:"failed"`
	Skipped  int       `json:"skipped"`
	Seconds  float64   `json:"seconds"`
	OK       bool      `json:"ok"`
	Report   string    `json:"report"`
	Failures []Failure `json:"failures"`
	SI       string    `json:"si"`
	Time     string    `json:"time"`
}

// Sentence is the summary in one line, for chat formats.
func (s Summary) Sentence() string {
	t := fmt.Sprintf("si demo test %s: %d passed, %d failed", s.Title, s.Passed, s.Failed)
	if s.Skipped > 0 {
		t += fmt.Sprintf(", %d skipped", s.Skipped)
	}
	t += fmt.Sprintf(" in %.1f s.", s.Seconds)
	if len(s.Failures) > 0 {
		var f []string
		for i, x := range s.Failures {
			if i == 5 {
				f = append(f, fmt.Sprintf("and %d more", len(s.Failures)-5))
				break
			}
			f = append(f, x.Script+" › "+x.Feature+" › "+x.Step)
		}
		t += " Failed: " + strings.Join(f, "; ") + "."
	}
	if s.Report != "" {
		t += " Report: " + s.Report
	}
	return t
}

// Body is the summary as format's payload (json, slack, teams, discord).
func (s Summary) Body(format string) ([]byte, error) {
	if s.Failures == nil {
		s.Failures = []Failure{}
	}
	s.Version, s.Event = 1, "demo.test"
	if format == "" || format == "json" {
		return json.Marshal(s)
	}
	if b := Chat(format, s.Sentence()); b != nil {
		return b, nil
	}
	return nil, fmt.Errorf("--webhook-format %s: want one of %s", format, strings.Join(ChatFormats, ", "))
}
