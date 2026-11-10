package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/felipeg48/simple-ideas/internal/tools"
	"github.com/felipeg48/simple-ideas/internal/webdemo"
)

const demoUsage = `Usage:
  si demo test <script.md|.adoc|.json> [--feature "Name"] [--junit report.xml] [-o dir]
        run a web demo script (type: demo with app:) in headless Chrome as a test: one line
        per step, report.html + report.json in demo-out/<script>/ (or -o dir), exit 1 on a failure;
        --junit also writes JUnit XML (one test case per feature) for CI
  si demo run <script>   the narrated videos come in a later version; for now it runs the test
`

// runDemo is `si demo test|run <script>`. Its flags are its own: --feature
// takes a name here, unlike si report's --feature.
func runDemo(argv []string, stdout, stderr io.Writer) int {
	fl := flag.NewFlagSet("si demo", flag.ContinueOnError)
	fl.SetOutput(io.Discard)
	var feature, junit, out string
	var help bool
	fl.StringVar(&feature, "feature", "", "")
	fl.StringVar(&junit, "junit", "", "")
	fl.StringVar(&out, "output", "", "")
	fl.StringVar(&out, "o", "", "")
	fl.BoolVar(&help, "help", false, "")
	fl.BoolVar(&help, "h", false, "")
	var args []string
	for {
		if err := fl.Parse(argv); err != nil {
			fmt.Fprintf(stderr, "si demo: %v\n\n%s", err, demoUsage)
			return exitUsage
		}
		argv = fl.Args()
		if len(argv) == 0 {
			break
		}
		args, argv = append(args, argv[0]), argv[1:]
	}
	if help {
		fmt.Fprint(stdout, demoUsage)
		return exitOK
	}
	if len(args) != 2 || (args[0] != "test" && args[0] != "run") {
		fmt.Fprintf(stderr, "si demo: test or run, and one script\n\n%s", demoUsage)
		return exitUsage
	}
	if args[0] == "run" {
		fmt.Fprintln(stderr, "si demo run: the narrated videos come in a later version of si; running the test now")
	}
	s, err := webdemo.ParseFile(args[1])
	if err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	for _, w := range s.Warnings {
		fmt.Fprintf(stderr, "warning: %s\n", w)
	}
	if len(s.Problems) > 0 {
		for _, p := range s.Problems {
			fmt.Fprintln(stderr, p)
		}
		return exitProblems
	}
	chrome := findChrome()
	if chrome == "" {
		fmt.Fprintf(stderr, "si: %s\n", tools.Need("chrome", "si demo"))
		return exitProblems
	}
	if out == "" {
		out = filepath.Join("demo-out", s.Name)
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	rep, err := webdemo.Run(s, webdemo.Options{Chrome: chrome, Feature: feature, Out: out, Log: stdout, SI: version})
	if err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	if err := rep.Write(out); err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	if junit != "" {
		b, err := rep.JUnit()
		if err == nil {
			err = os.WriteFile(junit, b, 0o644)
		}
		if err != nil {
			fmt.Fprintf(stderr, "si: %v\n", err)
			return exitProblems
		}
	}
	fmt.Fprintf(stdout, "si demo: %d passed, %d failed in %.1fs; report: %s\n", rep.Passed, rep.Failed, rep.Seconds, filepath.Join(out, "report.html"))
	if !rep.OK() {
		return exitProblems
	}
	return exitOK
}

// webDemoProblems prints a web demo script's problems (a bad step, with its
// line) and warnings (an unset $VAR) and returns the number of problems.
func webDemoProblems(path string, stderr io.Writer) int {
	s, err := webdemo.ParseFile(path)
	if err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return 1
	}
	for _, p := range s.Problems {
		fmt.Fprintln(stderr, p)
	}
	for _, w := range s.Warnings {
		fmt.Fprintf(stderr, "warning: %s\n", w)
	}
	return len(s.Problems)
}

// checkWebDemo is si check on a JSON web demo script.
func checkWebDemo(path string, stdout, stderr io.Writer) int {
	if n := webDemoProblems(path, stderr); n > 0 {
		fmt.Fprintf(stderr, "si check: %d problem(s) in %s\n", n, path)
		return exitProblems
	}
	fmt.Fprintf(stdout, "si check: %s OK\n", path)
	return exitOK
}
