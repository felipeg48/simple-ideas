package render

import (
	"fmt"
	"html"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

// quizPage turns the body of a `type: quiz` page into its intro (the text
// before the first ## question, the title included) and one quiz fence.
func quizPage(body, pass string) string {
	lines := strings.Split(body, "\n")
	inFence := ""
	for i, l := range lines {
		t := strings.TrimSpace(l)
		if inFence != "" {
			if strings.HasPrefix(t, inFence) {
				inFence = ""
			}
			continue
		}
		if strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~") {
			inFence = t[:3]
			continue
		}
		if strings.HasPrefix(l, "## ") || strings.HasPrefix(l, "### ") {
			info := "quiz"
			if pass != "" {
				info += " pass=" + pass
			}
			return strings.Join(lines[:i], "\n") + "\n\n`````" + info + "\n" + strings.Join(lines[i:], "\n") + "\n`````\n"
		}
	}
	return body + "\n\n`````quiz\n`````\n" // no questions: the fence says so
}

// quizHTML renders a quiz (a ```quiz fence, or a quiz page): one form, a
// fieldset per question, radio buttons (one right answer) or check boxes
// (more), the explanation hidden until the learner checks; doc.js scores it
// (deck.Score is the same rule) against the pass mark.
func (dr *deckRenderer) quizHTML(src, pass string) string {
	intro, qs, probs := deck.ParseQuiz(src)
	for _, p := range probs {
		dr.problem(nil, "%s", p.Msg)
	}
	mark := 0
	if pass != "" {
		n, ok := deck.ParsePass(pass)
		if !ok {
			dr.problem(nil, "quiz: pass=%s is not a mark from 0%% to 100%%", pass)
		}
		mark = n
	}
	dr.quizN++
	var b strings.Builder
	fmt.Fprintf(&b, `<form class="si-quiz" data-pass="%d" novalidate>`, mark)
	if intro != "" {
		b.WriteString(`<div class="si-quiz-intro">` + dr.toHTML(intro, nil) + `</div>`)
	}
	for i, q := range qs {
		typ, multi := "radio", ""
		if q.Multi() {
			typ, multi = "checkbox", " data-multi"
		}
		fmt.Fprintf(&b, `<fieldset class="si-quiz-q"%s><legend><span class="si-quiz-n">%d</span>%s</legend>`, multi, i+1, dr.inline(q.Title))
		if t := strings.TrimSpace(strings.Join(q.Text, "\n")); t != "" {
			b.WriteString(`<div class="si-quiz-text">` + dr.toHTML(t, nil) + `</div>`)
		}
		if multi != "" {
			b.WriteString(`<p class="si-quiz-hint">Choose every right answer.</p>`)
		}
		for j, o := range q.Options {
			ok := ""
			if o.OK {
				ok = " data-ok"
			}
			fmt.Fprintf(&b, `<label class="si-quiz-opt"><input type="%s" name="si-q%d-%d" value="%d"%s><span>%s</span></label>`, typ, dr.quizN, i+1, j, ok, dr.inline(o.Text))
		}
		if q.Explanation != "" {
			b.WriteString(`<p class="si-quiz-why" hidden><b>Why</b> ` + dr.inline(q.Explanation) + `</p>`)
		}
		b.WriteString(`</fieldset>`)
	}
	passText := ""
	if mark > 0 {
		passText = fmt.Sprintf(`<span class="si-quiz-mark">Pass mark %s %%</span>`, html.EscapeString(fmt.Sprint(mark)))
	}
	b.WriteString(`<div class="si-quiz-foot"><button type="submit" class="si-quiz-check">Check my answers</button><button type="button" class="si-quiz-again" hidden>Try again</button>` + passText + `<p class="si-quiz-score" role="status" aria-live="polite"></p></div></form>` + "\n")
	return b.String()
}
