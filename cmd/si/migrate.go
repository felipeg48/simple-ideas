package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// migratePromptTmpl is the editorial-migration playbook handed to an AI agent.
// SRC and DEST are substituted. It mirrors the "Migrating an existing deck into
// si" guidance in CLAUDE.md and the agent prompts used to migrate the courses.
const migratePromptTmpl = `You are migrating an existing slide deck / course at:
  SRC
into an ` + "`si`" + ` animated slide deck written to:
  DEST

This is an EDITORIAL migration, not a mechanical format conversion. Work with
full autonomy: pick sensible defaults and proceed.

First, learn the target format (read these, or run the si commands):
- RULES.md (run ` + "`si rules`" + `) — every directive, layout, the stage fence, canvas helpers.
- docs/stage-diagrams.md — the ` + "```stage" + ` flow-diagram DSL (chains, fan-out, merges,
  node/edge labels, dot/flow, and step [highlight|ok|alert|show|hide] reveals).
- docs/authoring-canvases.md — hand-authored animated SVG canvases and the helper
  attributes (data-show/hide/ok/alert/flow/packets/grow/move/count).

Then do the editorial pass:
1. Read the whole source and understand the narrative; decide what each slide is for.
2. Prune ruthlessly: merge/cut near-duplicate slides, restated headings, filler and
   course cross-sell. Preserve EVERY substantive teaching point, code snippet and
   setting/flag name.
3. Replace static images with NATIVE visuals (imported images ignore the theme and
   dark mode). Prefer, in order: a stronger layout (two-col, big-number, quote), an
   inline stage flow diagram, a hand-authored <!-- svg: --> canvas coloured with
   --si-* tokens, or a highlighted code block. No slide should be a wall of text.
   Add as many diagrams/animations as genuinely help.
4. Ship a SPLIT deck: a thin main deck (front matter, title, agenda, one
   <!-- include: sections/NN-*.md --> per section, closing) plus one file per section
   under sections/, with images/anim/ (animated canvases) and images/diagrams/
   (static diagrams). Section files reference assets as ../images/....
5. Verify: ` + "`si check`" + ` must pass with zero problems, then screenshot slides with
   headless Chrome and LOOK — a check-passing SVG can still be visually wrong. Fix
   and re-shoot.

Deliver the built deck and report: source→migrated slide counts and what you pruned,
the section breakdown, and how many stage diagrams / canvases / other SVGs you created.
`

// migratePrompt fills the playbook with concrete source and destination paths.
func migratePrompt(src, dest string) string {
	p := strings.ReplaceAll(migratePromptTmpl, "SRC", src)
	return strings.ReplaceAll(p, "DEST", dest)
}

// runMigrate prints the migration prompt, or (with --ai) pipes it to an agent CLI
// on stdin. The prompt generation is the dependency-free core; --ai is a thin,
// best-effort convenience around whatever agent command the user names.
func runMigrate(src string, o *options, stdout, stderr io.Writer) int {
	if _, err := os.Stat(src); err != nil {
		return usageErr(stderr, fmt.Sprintf("migrate: cannot read %s", src))
	}
	dest := o.output
	if dest == "" {
		dest = filepath.Base(filepath.Clean(src)) + "-si"
	}
	prompt := migratePrompt(src, dest)
	if o.ai == "" {
		fmt.Fprint(stdout, prompt)
		fmt.Fprintf(stderr, "si: migration prompt for %s -> %s; pipe it to your agent, or re-run with --ai \"<command>\"\n", src, dest)
		return exitOK
	}
	fields := strings.Fields(o.ai)
	cmd := exec.Command(fields[0], fields[1:]...)
	cmd.Stdin = strings.NewReader(prompt)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	fmt.Fprintf(stderr, "si: running %q with the migration prompt on stdin...\n", o.ai)
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(stderr, "si: agent failed: %v\n", err)
		return exitProblems
	}
	return exitOK
}
