package webdemo

import (
	"io"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"
)

// output keeps what a terminal or a background program printed: the bytes
// as they came (secrets masked) for the terminal page, the plain text (ANSI
// stripped) for expect output, and the exit codes a terminal's prompt
// reports (OSC 133;D;<code>, which si's shell prompt prints).
type output struct {
	mu      sync.Mutex
	raw     []byte
	plain   []byte
	codes   []int
	changed chan struct{} // closed and replaced on every write
	closed  bool
	st      stripper
	mask    func(string) string
	log     io.Writer // a copy of the masked bytes (start-<n>.log), or nil
}

func newOutput(mask func(string) string, log io.Writer) *output {
	if mask == nil {
		mask = func(s string) string { return s }
	}
	return &output{changed: make(chan struct{}), mask: mask, log: log}
}

// Write keeps p. ponytail: secrets are masked per write; one split across two
// reads of the pipe would slip through (reads are whole lines in practice).
func (o *output) Write(p []byte) (int, error) {
	b := []byte(o.mask(string(p)))
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.log != nil {
		o.log.Write(b)
	}
	o.raw = append(o.raw, b...)
	o.plain, o.codes = o.st.feed(b, o.plain, o.codes)
	close(o.changed)
	o.changed = make(chan struct{})
	return len(p), nil
}

// close marks the end: the program or the shell is gone.
func (o *output) close() {
	o.mu.Lock()
	defer o.mu.Unlock()
	if !o.closed {
		o.closed = true
		close(o.changed)
		o.changed = make(chan struct{})
	}
}

// from gives the raw bytes after off, a channel closed on the next write,
// and whether the output has ended.
func (o *output) from(off int) ([]byte, <-chan struct{}, bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if off > len(o.raw) {
		off = len(o.raw)
	}
	return o.raw[off:], o.changed, o.closed
}

// mark is where the plain text and the exit codes are now.
func (o *output) mark() (int, int) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return len(o.plain), len(o.codes)
}

// text is the plain text after off.
func (o *output) text(off int) string {
	o.mu.Lock()
	defer o.mu.Unlock()
	if off > len(o.plain) {
		off = len(o.plain)
	}
	return string(o.plain[off:])
}

// code is the n-th exit code the prompt reported (ok false until it has).
func (o *output) code(n int) (int, bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if n < len(o.codes) {
		return o.codes[n], true
	}
	return 0, false
}

// tail is the last n lines of the plain text.
func (o *output) tail(n int) string {
	t := strings.TrimRight(o.text(0), "\n ")
	lines := strings.Split(t, "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}

var spaceRE = regexp.MustCompile(`\s+`)

// shows reports whether text appears in s, any run of white space matching
// any other (a wrapped or padded line still matches).
func shows(s, text string) bool {
	return strings.Contains(spaceRE.ReplaceAllString(s, " "), strings.TrimSpace(spaceRE.ReplaceAllString(text, " ")))
}

// stripper turns a terminal byte stream into plain text, across writes:
// escape sequences (CSI, OSC, charset and single-character escapes) go,
// carriage returns go, a backspace takes back the character before it.
type stripper struct {
	state int // 0 text, 1 after ESC, 2 CSI, 3 OSC, 4 ESC inside OSC, 5 one more byte (ESC ( B)
	osc   []byte
}

func (s *stripper) feed(p, plain []byte, codes []int) ([]byte, []int) {
	for _, c := range p {
		switch s.state {
		case 0:
			switch {
			case c == 0x1b:
				s.state = 1
			case c == '\b':
				if n := len(plain); n > 0 && plain[n-1] != '\n' {
					_, size := utf8.DecodeLastRune(plain)
					plain = plain[:n-size]
				}
			case c == '\n' || c == '\t' || c >= 0x20:
				plain = append(plain, c)
			}
		case 1:
			s.afterEsc(c)
		case 2:
			if c >= 0x40 && c <= 0x7e {
				s.state = 0
			}
		case 3:
			switch c {
			case 0x07:
				codes = s.endOSC(codes)
			case 0x1b:
				s.state = 4
			default:
				if len(s.osc) < 256 {
					s.osc = append(s.osc, c)
				}
			}
		case 4:
			codes = s.endOSC(codes) // ESC \ ends it; any other ESC starts a sequence
			if c != '\\' {
				s.afterEsc(c)
			}
		case 5:
			s.state = 0
		}
	}
	return plain, codes
}

func (s *stripper) afterEsc(c byte) {
	switch c {
	case '[':
		s.state = 2
	case ']':
		s.state, s.osc = 3, s.osc[:0]
	case '(', ')', '*', '+', '#', '%':
		s.state = 5
	default:
		s.state = 0
	}
}

func (s *stripper) endOSC(codes []int) []int {
	s.state = 0
	if v, ok := strings.CutPrefix(string(s.osc), "133;D"); ok {
		n, err := strconv.Atoi(strings.TrimPrefix(v, ";"))
		if err != nil {
			n = 0
		}
		codes = append(codes, n)
	}
	return codes
}
