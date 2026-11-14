package deck

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// A Check is a lab step's `<!-- check: <command> · expect "text" · exit N -->`:
// a command that tells whether the learner's step works. si play runs it
// (internal/serve/check.go); every build shows it as "Run this to check".
type Check struct {
	Cmd    string // the shell command
	Expect string // text the output must contain ("" = any)
	Exit   int    // the exit code that passes (0 by default)
}

var checkExpectRE = regexp.MustCompile(`^expect\s*=?\s*"(.*)"$`)
var checkExitRE = regexp.MustCompile(`^exit\s*=?\s*(\d{1,3})$`)

// ParseCheck reads a check directive's value: the command, then
// `· expect "text"` and `· exit N` in any order.
func ParseCheck(val string) (Check, error) {
	parts := strings.Split(val, " · ")
	c := Check{Cmd: strings.TrimSpace(parts[0])}
	if c.Cmd == "" {
		return c, fmt.Errorf(`check: needs a command, <!-- check: curl -fsS localhost:8080/health · expect "ok" -->`)
	}
	for _, p := range parts[1:] {
		p = strings.TrimSpace(p)
		if m := checkExpectRE.FindStringSubmatch(p); m != nil {
			c.Expect = m[1]
		} else if m := checkExitRE.FindStringSubmatch(p); m != nil {
			c.Exit, _ = strconv.Atoi(m[1])
		} else {
			return c, fmt.Errorf(`check: %q is not expect "text" or exit N`, p)
		}
	}
	return c, nil
}

// Checks are the check directives of a parsed page, in order (includes
// resolved, fences skipped), the bad ones left out.
func (d *Deck) Checks() []Check {
	var out []Check
	for _, s := range d.Slides {
		var ft fenceTracker
		for _, l := range strings.Split(s.Body, "\n") {
			if ft.inFence(l) {
				continue
			}
			if name, val, ok := ParseDirective(strings.TrimSpace(l)); ok && name == "check" {
				if c, err := ParseCheck(val); err == nil {
					out = append(out, c)
				}
			}
		}
	}
	return out
}
