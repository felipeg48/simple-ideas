package deck

import "testing"

func TestSetDirective(t *testing.T) {
	deck := "# Title\n\n---\n\n## Two\n\nBody.\n\n---\n\n## Three\n"
	for _, c := range []struct {
		name, text     string
		start, end     int
		key, val, want string
	}{
		{"adds under the heading", deck, 5, 8, "layout", "two-col", "# Title\n\n---\n\n## Two\n\n<!-- layout: two-col -->\n\nBody.\n\n---\n\n## Three\n"},
		{"replaces in that slide only", "## A\n\n<!-- layout: quote -->\n\n---\n\n## B\n\n<!-- layout: quote -->\n", 1, 4, "layout", "big-number",
			"## A\n\n<!-- layout: big-number -->\n\n---\n\n## B\n\n<!-- layout: quote -->\n"},
		{"removes without a double gap", "## A\n\n<!-- layout: quote -->\n\nBody\n", 1, 5, "layout", "", "## A\n\nBody\n"},
		{"first in a slide without a heading", "**98%**\n\nkept\n", 1, 3, "layout", "big-number", "<!-- layout: big-number -->\n\n**98%**\n\nkept\n"},
		{"nothing to remove", deck, 5, 8, "class", "", deck},
	} {
		if got := SetDirective(c.text, c.start, c.end, c.key, c.val); got != c.want {
			t.Errorf("%s:\n got %q\nwant %q", c.name, got, c.want)
		}
	}
}
