package docsite

import (
	"regexp"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/tools"
)

// The External dependencies section of a feature page, made from the same
// table as `si doctor` (internal/tools), so the site cannot drift from what si
// checks: the programs the feature uses, what for, and macOS | Linux |
// Windows tabs with the install commands. A hand-written site page asks for
// it with a <!-- si-deps: chrome ffmpeg --> line (none: <!-- si-deps: -->).

// featureDeps are the programs each generated feature page lists.
var featureDeps = map[string][]string{
	"articles": {"gh"},
	"guides":   {"gh"},
	"books":    {"chrome", "gh"},
	"mindmaps": {"chrome", "ffmpeg"},
	"code":     {"chrome", "ffmpeg", "gh"},
}

var depsLineRE = regexp.MustCompile(`(?m)^<!-- si-deps:([^>]*)-->$`)

// withDeps replaces a page's <!-- si-deps: … --> line with the section.
func withDeps(md string) string {
	return depsLineRE.ReplaceAllStringFunc(md, func(l string) string {
		return depsSection(strings.Fields(depsLineRE.FindStringSubmatch(l)[1]))
	})
}

// depsSection is the Markdown of the section for these tools.
func depsSection(names []string) string {
	var b strings.Builder
	b.WriteString("## External dependencies\n\n")
	if len(names) == 0 {
		b.WriteString("None: everything here is in si itself. `si doctor` lists the programs other features use.\n")
		return b.String()
	}
	b.WriteString("si itself needs nothing installed. These parts use a program you install once:\n\n| Program | Used for |\n|---|---|\n")
	var ts []tools.Tool
	for _, n := range names {
		if t, ok := tools.Get(n); ok {
			ts = append(ts, t)
			b.WriteString("| [" + t.Label + "](" + t.URL + ") | " + t.For + " |\n")
		}
	}
	lines := func(platform string) string {
		var out []string
		for _, t := range ts {
			out = append(out, cmdLine(t.Install[platform]))
		}
		return strings.Join(out, "\n")
	}
	b.WriteString("\n<!-- tabs -->\n```bash title=macOS\n" + lines("mac") + "\n```\n```bash title=Linux\n")
	for i, fam := range []struct{ key, name string }{{"debian", "Debian, Ubuntu"}, {"fedora", "Fedora"}, {"arch", "Arch"}} {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString("# " + fam.name + "\n" + lines(fam.key) + "\n")
	}
	b.WriteString("```\n```powershell title=Windows\n" + lines("windows") + "\n```\n\n")
	b.WriteString("Not sure what you have? `si doctor` lists every program, found or not, with the command for your computer.\n")
	return b.String()
}

// cmdLine keeps a command as it is and turns advice into a comment.
func cmdLine(s string) string {
	for _, c := range []string{"brew ", "sudo ", "winget ", "xcode-select", "curl ", "wsl "} {
		if strings.HasPrefix(s, c) {
			return s
		}
	}
	return "# " + s
}
