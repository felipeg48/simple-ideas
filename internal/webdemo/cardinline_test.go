package webdemo

import "testing"

func TestCardInline(t *testing.T) {
	cases := []struct{ in, want string }{
		{"hello", "hello"},
		{"run `acemq top` now", "run <code>acemq top</code> now"},
		{"`acemq migrate assess` reads it", "<code>acemq migrate assess</code> reads it"},
		{"join #channel", "join #channel"},                  // # is not markdown
		{"a *star* b", "a *star* b"},                        // * is not markdown
		{"x & y < z", "x &amp; y &lt; z"},                   // HTML still escaped
		{"use \\`ticks\\` here", "use `ticks` here"},        // escaped backticks stay literal
		{"a backslash \\\\ here", "a backslash \\ here"},    // escaped backslash
		{"unterminated `code", "unterminated `code"},        // lone backtick is literal
		{"`a \\` b`", "<code>a ` b</code>"},                 // escaped backtick inside a span
	}
	for _, c := range cases {
		if got := cardInline(c.in); got != c.want {
			t.Errorf("cardInline(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
