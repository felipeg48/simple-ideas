package render

import (
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// A runbook is an optional command cheat-sheet shown in the `si play` terminal
// drawer (play-only, loopback-only, like the terminal itself — a built deck
// never loads it). The on-disk JSON shape matches the cockroach-engineering-tool
// VS Code extension's runbook.json, so an exported runbook drops in unchanged:
//
//	{ "categories": [ { "name": "...", "icon": "...", "entries": [
//	    { "label": "...", "description": "...", "command": "SHOW ...;", "target": "sql" } ] } ] }
//
// `command` is canonical; `sql` is accepted as an alias for drop-in compat.
// `target` ("sql" | "bash", default "sql") selects the syntax highlighter.

type runbookFile struct {
	Name        string            `json:"name,omitempty"`
	Description string            `json:"description,omitempty"` // accepted, unused by the drawer
	Categories  []runbookCategory `json:"categories"`
}

type runbookCategory struct {
	Name    string         `json:"name"`
	Icon    string         `json:"icon,omitempty"`
	Entries []runbookEntry `json:"entries"`
}

type runbookEntry struct {
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	Command     string `json:"command,omitempty"` // canonical command text
	SQL         string `json:"sql,omitempty"`     // alias (extension compat)
	Target      string `json:"target,omitempty"`  // "sql" | "bash"; default "sql"
}

// Placeholder is a `{{name}}` or `{{name:default}}` slot in a runbook command;
// the drawer asks for a value before typing the command.
type Placeholder struct{ Name, Default string }

var placeholderRe = regexp.MustCompile(`\{\{([A-Za-z_][A-Za-z0-9_-]*)(?::([^}]*))?\}\}`)

// templateWords are Go-template keywords that fit the placeholder syntax
// (`{{end}}`, `{{else}}`) but belong to a go-template/--format argument; a
// match with one of these names is left as plain text (same rule in
// web/studio/src/runbook.ts and term.js).
var templateWords = map[string]bool{"end": true, "else": true, "break": true, "continue": true, "nil": true, "true": true, "false": true}

// Placeholders lists the distinct placeholders of cmd in document order; the
// first occurrence (and its default) wins.
func Placeholders(cmd string) []Placeholder {
	var out []Placeholder
	seen := map[string]bool{}
	for _, m := range placeholderRe.FindAllStringSubmatch(cmd, -1) {
		if !seen[m[1]] && !templateWords[m[1]] {
			seen[m[1]] = true
			out = append(out, Placeholder{m[1], m[2]})
		}
	}
	return out
}

// highlightWithVars highlights cmd with each placeholder occurrence swapped for
// a private-use sentinel (no letters, digits, quotes or comment characters, so
// no token rule can split it), then swaps the sentinels for si-rb-var spans.
// The spans therefore always nest properly inside the token spans.
func highlightWithVars(cmd, target string) string {
	var toks []string
	masked := placeholderRe.ReplaceAllStringFunc(cmd, func(m string) string {
		if templateWords[placeholderRe.FindStringSubmatch(m)[1]] {
			return m
		}
		toks = append(toks, m)
		return "\uE000" + string(rune(0xE100+len(toks)-1)) + "\uE001"
	})
	out := highlight(masked, target)
	for i, t := range toks {
		out = strings.Replace(out, "\uE000"+string(rune(0xE100+i))+"\uE001",
			`<span class="si-rb-var">`+html.EscapeString(t)+`</span>`, 1)
	}
	return out
}

// runbookForPage parses a runbook JSON file and returns the JSON injected as
// window.SI_RUNBOOK: the same categories, but each entry carries the raw command
// (`cmd`, sent to the shell on click) plus pre-highlighted HTML (`html`, shown
// in the drawer). Highlighting is done here, server-side, with the same
// highlighter the code fences use — no JS port. Entries with no command and
// categories left empty are dropped.
func runbookForPage(raw []byte) (string, error) {
	var rf runbookFile
	if err := json.Unmarshal(raw, &rf); err != nil {
		return "", fmt.Errorf("runbook: invalid JSON: %w", err)
	}
	type outVar struct {
		Name    string `json:"name"`
		Default string `json:"default"`
	}
	type outEntry struct {
		Label  string   `json:"label"`
		Desc   string   `json:"desc,omitempty"`
		Cmd    string   `json:"cmd"`
		HTML   string   `json:"html"`
		Target string   `json:"target"`
		Vars   []outVar `json:"vars,omitempty"`
	}
	type outCat struct {
		Name    string     `json:"name"`
		Icon    string     `json:"icon,omitempty"`
		Entries []outEntry `json:"entries"`
	}
	var cats []outCat
	for _, c := range rf.Categories {
		oc := outCat{Name: c.Name, Icon: c.Icon}
		for _, e := range c.Entries {
			cmd := e.Command
			if cmd == "" {
				cmd = e.SQL
			}
			if cmd == "" {
				continue
			}
			target := e.Target
			if target == "" {
				target = "sql"
			}
			var vars []outVar
			for _, p := range Placeholders(cmd) {
				vars = append(vars, outVar{p.Name, p.Default})
			}
			oc.Entries = append(oc.Entries, outEntry{
				Label: e.Label, Desc: e.Description, Cmd: cmd,
				HTML:   highlightWithVars(cmd, target),
				Target: target, Vars: vars,
			})
		}
		if len(oc.Entries) > 0 {
			cats = append(cats, oc)
		}
	}
	b, err := json.Marshal(struct {
		Name       string   `json:"name,omitempty"`
		Categories []outCat `json:"categories"`
	}{rf.Name, cats})
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// resolveRunbook loads the deck's `runbook:` file (if any) into dr.runbookJSON.
// It runs for every render so `si check` validates the path and JSON, but the
// result is only injected into the page by si play (see page, gated on Terminal).
func (dr *deckRenderer) resolveRunbook() {
	rb := dr.d.Meta.Runbook
	if rb == "" {
		return
	}
	path := rb
	if !filepath.IsAbs(path) {
		path = filepath.Join(filepath.Dir(dr.d.Path), rb)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		dr.problem(nil, "runbook: cannot read %s", rb)
		return
	}
	js, err := runbookForPage(raw)
	if err != nil {
		dr.problem(nil, "%v", err)
		return
	}
	dr.runbookJSON = js
}
