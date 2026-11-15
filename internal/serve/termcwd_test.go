package serve

import (
	"path/filepath"
	"testing"
)

// TestOscCwd: the folder a shell's output reports, from OSC 7 (BEL or ST),
// si's PowerShell titles and Windows Terminal's OSC 9;9; the last one wins,
// and a mark cut by a read counts once its end arrives.
func TestOscCwd(t *testing.T) {
	for in, want := range map[string]string{
		"\x1b]7;file://mac.local/Users/ana/my%20orders\x07$ ":         "/Users/ana/my orders",
		"\x1b]7;file:///tmp/a\x1b\\x \x1b]7;file:///tmp/b\x1b\\":      "/tmp/b",
		"\x1b]7;file://host/C:/Users/ana\x07":                         "C:/Users/ana",
		"\x1b]0;si;133;D;0;3;7;C:\\Users\\ana\\sub\x07sub $ ":         `C:\Users\ana\sub`,
		"\x1b]2;si;7;D:\\work\x07PS D:\\work> ":                       `D:\work`,
		"\x1b]9;9;\"C:\\repo\"\x07":                                   `C:\repo`,
		"\x1b]0;si;133;D;0;1\x07\x1b]0;pwsh\x07\x1b]7;http://x/y\x07": "",
		"\x1b]7;file:///tmp/cut":                                      "",
	} {
		w := want
		if want != "" && want[0] == '/' || want == "C:/Users/ana" {
			w = filepath.FromSlash(want)
		}
		if got := oscCwd([]byte(in)); got != w {
			t.Errorf("oscCwd(%q) = %q, want %q", in, got, w)
		}
	}
	s := &termSession{}
	for _, chunk := range []string{"ls\r\n\x1b]7;file:///tm", "p/one\x07$ ", "\x1b]0;si;7;/tmp/two", "\x07"} {
		s.track([]byte(chunk))
	}
	if s.dir != filepath.FromSlash("/tmp/two") || s.oscTail != nil {
		t.Errorf("across reads: dir %q, tail %q", s.dir, s.oscTail)
	}
}
