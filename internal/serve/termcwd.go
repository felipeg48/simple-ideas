package serve

import (
	"bytes"
	"net/url"
	"path/filepath"
	"strings"
)

// oscCwd is the last folder a shell's output reports, or "": an OSC 7 mark
// (ESC ] 7 ; file://host/path, ended by BEL or ESC \), which zsh, fish,
// many bash setups and Windows Terminal profiles print at each prompt, or
// si's PowerShell prompts' window title (ESC ] 0 ; si;133;D;<code>;<n>;7;
// <folder> when clean, si;7;<folder> live: webdemo.LiveShell), the one OSC
// ConPTY always passes on, or Windows Terminal's ESC ] 9;9;"<folder>".
// Only whole marks
// count: one cut by a read is found once its end arrives (termSession keeps
// the tail).
func oscCwd(b []byte) string {
	dir := ""
	for {
		i := bytes.Index(b, []byte("\x1b]"))
		if i < 0 {
			return dir
		}
		b = b[i+2:]
		end, n := bytes.IndexByte(b, '\a'), 1
		if st := bytes.Index(b, []byte("\x1b\\")); st >= 0 && (end < 0 || st < end) {
			end, n = st, 2
		}
		if end < 0 {
			return dir
		}
		osc := string(b[:end])
		b = b[end+n:]
		if v, ok := strings.CutPrefix(osc, "7;"); ok {
			if d := fileURLDir(v); d != "" {
				dir = d
			}
			continue
		}
		if v, ok := strings.CutPrefix(osc, "9;9;"); ok { // Windows Terminal's own: ESC ] 9;9;"C:\dir"
			if v = strings.Trim(v, `"`); v != "" {
				dir = v
			}
			continue
		}
		for _, p := range []string{"0;si;", "2;si;"} {
			if v, ok := strings.CutPrefix(osc, p); ok {
				if d, ok := strings.CutPrefix(v, "7;"); ok && d != "" {
					dir = d
				} else if _, d, ok := strings.Cut(v, ";7;"); ok && d != "" {
					dir = d
				}
			}
		}
	}
}

// fileURLDir is the folder of an OSC 7 file:// URL ("" if it is none):
// percent-decoded, a Windows drive path (/C:/Users) without its leading /.
func fileURLDir(v string) string {
	u, err := url.Parse(v)
	if err != nil || u.Scheme != "file" || u.Path == "" {
		return ""
	}
	p := u.Path
	if len(p) >= 3 && p[0] == '/' && p[2] == ':' {
		p = p[1:]
	}
	return filepath.FromSlash(p)
}

// track reads a chunk of a shell's output for the folder it reports (s.mu
// held): OSC marks a read cut wait in oscTail for the rest.
func (s *termSession) track(b []byte) {
	scan := append(s.oscTail, b...)
	if d := oscCwd(scan); d != "" {
		s.dir = d
	}
	s.oscTail = nil
	if i := bytes.LastIndex(scan, []byte("\x1b]")); i >= 0 && len(scan)-i < 4096 {
		if rest := scan[i+2:]; bytes.IndexByte(rest, '\a') < 0 && !bytes.Contains(rest, []byte("\x1b\\")) {
			s.oscTail = append([]byte(nil), scan[i:]...)
		}
	}
}
