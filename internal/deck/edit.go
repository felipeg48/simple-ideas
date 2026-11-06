package deck

import (
	"regexp"
	"strings"
)

// SetDirective sets <!-- name: value --> inside the slide that spans lines
// start..end (1-based, inclusive) of text: it replaces the directive if the
// slide has it, removes it when value is empty, or adds it under the slide's
// heading (or at its first line when it has none). The Studio inspectors (web
// and terminal) write slide properties with it.
func SetDirective(text string, start, end int, name, value string) string {
	lines := strings.Split(text, "\n")
	lo, hi := max(start-1, 0), min(end-1, len(lines)-1)
	re := regexp.MustCompile(`^\s*<!--\s*` + regexp.QuoteMeta(name) + `\s*:.*-->\s*$`)
	value = strings.TrimSpace(value)
	line := "<!-- " + name + ": " + value + " -->"
	for i := lo; i <= hi; i++ {
		if !re.MatchString(lines[i]) {
			continue
		}
		if value != "" {
			lines[i] = line
		} else {
			// drop the line, and one blank line around it so no double gap is left
			blankAfter := i+1 < len(lines) && strings.TrimSpace(lines[i+1]) == ""
			blankBefore := i > 0 && strings.TrimSpace(lines[i-1]) == ""
			n := 1
			if blankAfter && blankBefore {
				n = 2
			}
			lines = append(lines[:i], lines[i+n:]...)
		}
		return strings.Join(lines, "\n")
	}
	if value == "" {
		return text
	}
	at := lo
	for at <= hi && strings.TrimSpace(lines[at]) == "" {
		at++
	}
	var ins []string
	if at <= hi && headingLineRE.MatchString(lines[at]) {
		at++
		ins = []string{"", line}
	} else {
		ins = []string{line, ""}
	}
	lines = append(lines[:at], append(ins, lines[at:]...)...)
	return strings.Join(lines, "\n")
}

var headingLineRE = regexp.MustCompile(`^#{1,6}\s`)
