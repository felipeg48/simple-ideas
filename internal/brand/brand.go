// Package brand holds what si says about itself: the website, the repository,
// the issue links (Report a Problem, Request a Feature), the open-source parts
// it is built with, and the build it runs. The CLI, the Studio and (through
// the Studio) the desktop app all read it from here.
package brand

import (
	"net/url"
	"runtime"
	"runtime/debug"
)

const (
	Name    = "si — simple-ideas"
	Tagline = "Write Markdown or AsciiDoc, present anything."
	About   = "si turns plain text into decks, articles, tutorials and guides, books, mind maps and code shows. " +
		"It runs in the browser, the terminal and the desktop, and builds self-contained pages."
	Site      = "https://simple-ideas.org/" // the docs site; a domain change happens here only
	Repo      = "https://github.com/felipeg48/simple-ideas"
	Copyright = "© 2026 Felipe Gutierrez"
	License   = "Apache License 2.0"
)

// Pages of the docs site the Help menus open.
var Pages = map[string]string{"docs": Site, "rules": Site + "rules/", "manual": Site + "manual.html", "releases": Repo + "/releases"}

// Modes are the values of the issue forms' "mode" dropdown (.github/ISSUE_TEMPLATE).
var Modes = []string{"CLI", "si play (browser)", "web Studio", "desktop app", "terminal", "built page"}

// IssueURL opens a new issue from the bug (or, with feature, the feature)
// form, its fields filled in from the query: GitHub issue forms take a
// field's id as a query parameter.
func IssueURL(feature bool, version, mode, os string) string {
	q := url.Values{}
	if feature {
		q.Set("template", "feature.yml")
	} else {
		q.Set("template", "bug.yml")
	}
	for k, v := range map[string]string{"version": version, "os": os, "mode": mode} {
		if v != "" && (k == "mode" || !feature) {
			q.Set(k, v)
		}
	}
	return Repo + "/issues/new?" + q.Encode()
}

// Credit is an open-source part si is built with.
type Credit struct {
	Name    string `json:"name"`
	License string `json:"license"`
	URL     string `json:"url"`
	Note    string `json:"note,omitempty"`
}

var Credits = []Credit{
	{"goldmark", "MIT", "https://github.com/yuin/goldmark", "Markdown"},
	{"Bubble Tea", "MIT", "https://github.com/charmbracelet/bubbletea", "the terminal front end"},
	{"Lip Gloss", "MIT", "https://github.com/charmbracelet/lipgloss", "the terminal front end"},
	{"creack/pty", "MIT", "https://github.com/creack/pty", "the si play terminal drawer"},
	{"xterm.js", "MIT", "https://github.com/xtermjs/xterm.js", "the si play terminal drawer"},
	{"Preact", "MIT", "https://github.com/preactjs/preact", "the Studio"},
	{"CodeMirror", "MIT", "https://github.com/codemirror/dev", "the Studio editor"},
	{"three.js", "MIT", "https://github.com/mrdoob/three.js", "3D mind maps"},
	{"axe-core", "MPL-2.0", "https://github.com/dequelabs/axe-core", "accessibility tests only, never shipped"},
}

// Build is the commit and date of this binary ("" when built without VCS info).
func Build() (commit, date string) {
	if bi, ok := debug.ReadBuildInfo(); ok {
		for _, s := range bi.Settings {
			switch s.Key {
			case "vcs.revision":
				commit = s.Value
				if len(commit) > 7 {
					commit = commit[:7]
				}
			case "vcs.time":
				date = s.Value
			}
		}
	}
	return
}

// OS is the operating system name the issue form lists.
func OS() string {
	switch runtime.GOOS {
	case "darwin":
		return "macOS"
	case "windows":
		return "Windows"
	case "linux":
		return "Linux"
	}
	return "other"
}
