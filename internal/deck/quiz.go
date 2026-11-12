package deck

import (
	"fmt"
	"math"
	"regexp"
	"strings"
)

// A quiz is Markdown: a `## Question` heading, any text under it, the answers
// as a task list (`- [x]` marks a right one; more than one makes it multiple
// choice) and an optional `> Explanation: …` shown after answering. It is the
// body of a `type: quiz` page and of a ```` ```quiz ```` fence.

// Question is one question of a quiz.
type Question struct {
	Title       string   // the heading (inline Markdown)
	Text        []string // lines between the heading and the answers (Markdown)
	Options     []Option // the answers, in order
	Explanation string   // shown after answering (inline Markdown)
	Line        int      // 1-based line of the heading in the quiz source
}

// Option is one answer of a question.
type Option struct {
	Text string // inline Markdown
	OK   bool   // a right answer
}

// Multi reports whether the question has more than one right answer (check
// boxes instead of radio buttons).
func (q Question) Multi() bool {
	n := 0
	for _, o := range q.Options {
		if o.OK {
			n++
		}
	}
	return n > 1
}

var (
	quizQRE   = regexp.MustCompile(`^ {0,3}#{2,3}\s+(.+?)\s*#*\s*$`)
	quizOptRE = regexp.MustCompile(`^ {0,3}[-*+]\s+\[([ xX])\]\s+(.*\S)\s*$`)
	quizWhyRE = regexp.MustCompile(`^ {0,3}>\s?(.*)$`)
)

// ParseQuiz reads a quiz. intro is the Markdown before the first question;
// problems carry 1-based lines. A quiz with no heading at all is one
// question whose first line of text is its title.
func ParseQuiz(src string) (intro string, qs []Question, problems []Problem) {
	lines := strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n")
	hasHeading := false
	var ft fenceTracker
	for _, l := range lines {
		if !ft.inFence(l) && quizQRE.MatchString(l) {
			hasHeading = true
			break
		}
	}
	var introLines []string
	var q *Question
	flush := func() {
		if q != nil {
			qs = append(qs, *q)
		}
		q = nil
	}
	ft = fenceTracker{}
	for i, l := range lines {
		fenced := ft.inFence(l)
		if !fenced {
			if m := quizQRE.FindStringSubmatch(l); m != nil {
				flush()
				q = &Question{Title: m[1], Line: i + 1}
				continue
			}
		}
		if q == nil {
			if hasHeading || strings.TrimSpace(l) == "" {
				introLines = append(introLines, l)
				continue
			}
			q = &Question{Title: strings.TrimSpace(l), Line: i + 1} // no headings: the first line asks
			continue
		}
		if !fenced {
			if m := quizOptRE.FindStringSubmatch(l); m != nil {
				q.Options = append(q.Options, Option{Text: m[2], OK: m[1] != " "})
				continue
			}
			if m := quizWhyRE.FindStringSubmatch(l); m != nil {
				t := strings.TrimSpace(m[1])
				if q.Explanation == "" {
					t = strings.TrimSpace(strings.TrimPrefix(t, "Explanation:"))
				}
				q.Explanation = strings.TrimSpace(q.Explanation + " " + t)
				continue
			}
		}
		if len(q.Options) == 0 {
			q.Text = append(q.Text, l)
		}
	}
	flush()
	intro = strings.TrimSpace(strings.Join(introLines, "\n"))
	if len(qs) == 0 {
		problems = append(problems, Problem{Line: 1, Msg: "quiz: no questions (## Question, then - [ ] and - [x] answers)"})
	}
	for _, q := range qs {
		ok := 0
		for _, o := range q.Options {
			if o.OK {
				ok++
			}
		}
		switch {
		case len(q.Options) < 2:
			problems = append(problems, Problem{Line: q.Line, Msg: fmt.Sprintf("quiz: %q wants at least two answers (- [ ] and - [x])", q.Title)})
		case ok == 0:
			problems = append(problems, Problem{Line: q.Line, Msg: fmt.Sprintf("quiz: %q marks no right answer (- [x])", q.Title)})
		}
	}
	return intro, qs, problems
}

// Score is the percentage of questions answered right (rounded): picks[i]
// holds the indexes of the answers chosen for question i, and a question is
// right when exactly its right answers are chosen. The page runtime
// (doc.js) scores the same way.
func Score(qs []Question, picks [][]int) int {
	if len(qs) == 0 {
		return 0
	}
	right := 0
	for i, q := range qs {
		chosen := map[int]bool{}
		if i < len(picks) {
			for _, p := range picks[i] {
				chosen[p] = true
			}
		}
		good := true
		for j, o := range q.Options {
			if o.OK != chosen[j] {
				good = false
			}
		}
		if good {
			right++
		}
	}
	return int(math.Round(100 * float64(right) / float64(len(qs))))
}
