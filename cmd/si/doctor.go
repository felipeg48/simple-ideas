package main

import (
	"fmt"
	"io"
	"runtime"

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
	if missing == 0 {
		fmt.Fprintln(stdout, "\nAll set: every optional program is here.")
	} else {
		fmt.Fprintf(stdout, "\n%d not found: install only the ones for the features you use.\n", missing)
	}
	return exitOK
}
