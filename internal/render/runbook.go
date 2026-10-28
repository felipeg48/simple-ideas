package render

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
	Categories []runbookCategory `json:"categories"`
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
	type outEntry struct {
		Label  string `json:"label"`
		Desc   string `json:"desc,omitempty"`
		Cmd    string `json:"cmd"`
		HTML   string `json:"html"`
		Target string `json:"target"`
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
			oc.Entries = append(oc.Entries, outEntry{
				Label: e.Label, Desc: e.Description, Cmd: cmd,
				HTML: highlight(cmd, target), Target: target,
			})
		}
		if len(oc.Entries) > 0 {
			cats = append(cats, oc)
		}
	}
	b, err := json.Marshal(map[string]any{"categories": cats})
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
