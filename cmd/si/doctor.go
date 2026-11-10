package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/felipeg48/simple-ideas/internal/config"
	"github.com/felipeg48/simple-ideas/internal/tools"
)

// doctor is `si doctor`: si itself, then every optional program it can use,
// found or not, with the install command for this computer.
func doctor(stdout io.Writer) int {
	fmt.Fprintf(stdout, "si %s on %s/%s (%s)\n\n", version, runtime.GOOS, runtime.GOARCH, tools.Platform())
	fmt.Fprintln(stdout, "Everything else in si needs nothing installed. These programs add features:")
	missing := 0
	for _, t := range tools.All {
		fmt.Fprintln(stdout)
		if p := tools.Find(t.Name); p != "" {
			fmt.Fprintf(stdout, "  ✓ %s  %s\n      for %s\n", t.Label, p, t.For)
			continue
		}
		missing++
		fmt.Fprintf(stdout, "  ✗ %s  not found\n      for %s\n      install: %s\n      more:    %s\n", t.Label, t.For, t.How(), t.URL)
	}
	voiceDoctor(stdout)
	if missing == 0 {
		fmt.Fprintln(stdout, "\nAll set: every optional program is here.")
	} else {
		fmt.Fprintf(stdout, "\n%d not found: install only the ones for the features you use.\n", missing)
	}
	return exitOK
}

// voiceDoctor shows what each voice engine has: Kokoro's venv and model
// files, and whether the cloud engines have a key (never the key).
func voiceDoctor(stdout io.Writer) {
	mark := map[bool]string{true: "✓", false: "✗"}
	dir := tools.KokoroDir()
	fmt.Fprintf(stdout, "\nVoices (si demo run, si voice say):\n  kokoro      %s\n", dir)
	py := tools.KokoroVenvPython(dir)
	_, err := os.Stat(py)
	fmt.Fprintf(stdout, "      %s venv  %s\n", mark[err == nil], py)
	for _, f := range tools.KokoroFiles {
		_, err := os.Stat(filepath.Join(dir, f))
		fmt.Fprintf(stdout, "      %s %s\n", mark[err == nil], f)
	}
	for _, e := range []string{"elevenlabs", "openai"} {
		if _, from := config.APIKey(e); from != "" {
			fmt.Fprintf(stdout, "  %-11s ✓ key set (%s)\n", e, from)
		} else {
			fmt.Fprintf(stdout, "  %-11s ✗ no key: si voice login %s\n", e, e)
		}
	}
}
