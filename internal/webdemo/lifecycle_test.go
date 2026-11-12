package webdemo

import (
	"encoding/xml"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/felipeg48/simple-ideas/internal/pinned"
	"github.com/felipeg48/simple-ideas/internal/tools"
)

const lifeSample = `---
type: demo
app: none
timeout: 3s
start:   ./broker · wait for port 22310   # keeps running
prepare: ./scripts/create-queues.sh
step:    seed run -f seed.yaml · within 5m
start:   ./consumer as consumer · wait for "listening"
setup:   purge orders
setup:   echo two
stop:    make demo-down
cleanup: rm -rf data/ tmp/
cleanup: echo "#not a comment" # a comment
prepare: github:acme/ops@v1.4/scripts/seed.sh --orders 50 · in ops
setup:   gist:abc123@4f2e9c1
stop:    https://example.com/down.sh · sha256 ` + "9f2a9f2a9f2a9f2a9f2a9f2a9f2a9f2a9f2a9f2a9f2a9f2a9f2a9f2a9f2a9f2a" + `
---

## Before
- terminal "T"
- start "./worker" as worker · wait 1s
- run "echo ready" · wait

## One
- terminal "T"
- expect worker running

## after
- terminal "T"
- run "echo bye" · wait
`

func TestParseLifecycle(t *testing.T) {
	s := Parse("/x/life.md", lifeSample)
	if len(s.Problems) > 0 {
		t.Fatalf("problems: %v", s.Problems)
	}
	var order []string
	for _, p := range s.Prelude {
		if p.Start != nil {
			order = append(order, "start "+p.Start.Name)
		} else {
			order = append(order, p.Cmd.Key+" "+p.Cmd.Text)
		}
	}
	want := "start start-1|prepare ./scripts/create-queues.sh|step seed run -f seed.yaml · within 5m|start consumer|prepare github:acme/ops@v1.4/scripts/seed.sh --orders 50 · in ops"
	if strings.Join(order, "|") != want {
		t.Errorf("prelude:\n%s\n%s", strings.Join(order, "|"), want)
	}
	if c := s.Prelude[2].Cmd; c.Within != 5*time.Minute || c.Cmds[0] != "seed run -f seed.yaml" {
		t.Errorf("step: %+v", c)
	}
	if c := s.Prelude[4].Cmd; c.Source == nil || c.Source.Path != "scripts/seed.sh" || c.Args != "--orders 50" || c.Dir != "ops" || c.Cmds != nil {
		t.Errorf("a github source: %+v", c)
	}
	if len(s.Setups) != 3 || s.Setups[1].Cmds[0] != "echo two" || s.Setups[2].Source.Kind != "gist" {
		t.Errorf("setups: %+v", s.Setups)
	}
	if len(s.Stops) != 2 || s.Stops[1].Source.Kind != "url" || len(s.Cleanups) != 2 || s.Cleanups[1].Cmds[0] != `echo "#not a comment"` {
		t.Errorf("stops %+v, cleanups %+v", s.Stops, s.Cleanups)
	}
	if s.Before == nil || s.After == nil || len(s.Features) != 1 || s.Features[0].Name != "One" {
		t.Fatalf("sections: %+v %+v %d", s.Before, s.After, len(s.Features))
	}
	if len(s.Sources()) != 3 || s.unresolved() == "" {
		t.Errorf("sources: %v", s.Sources())
	}
	// fetched: each script quoted, the arguments after it
	err := s.Resolve(func(src *pinned.Source) ([]string, error) { return []string{"/cache/" + src.Kind + ".sh"}, nil })
	if err != nil || s.unresolved() != "" || s.Prelude[4].Cmd.Cmds[0] != "'/cache/github.sh' --orders 50" {
		t.Errorf("resolve: %v %q", err, s.Prelude[4].Cmd.Cmds)
	}
	// problems: unpinned, bad clauses, two Befores, a program from ## Before is known
	for src, msg := range map[string]string{
		"prepare: github:acme/ops/seed.sh":         "not pinned",
		"setup: https://x.org/a.sh":                "not pinned",
		"cleanup: rm x · sha256 abc":               "fingerprint of a URL",
		"stop: make down · soon":                   "after ·",
		"start: github:acme/ops@v1/bin/ · wait 1s": "a program is one script",
		"prepare: ./x · within forever":            "within 5m",
	} {
		s := Parse("/x/p.md", "---\ntype: demo\napp: none\n"+src+"\n---\n## F\n- terminal \"T\"\n")
		if len(s.Problems) == 0 || !strings.Contains(s.Problems[0].Msg, msg) || s.Problems[0].Line != 4 {
			t.Errorf("%s: %v", src, s.Problems)
		}
	}
	two := Parse("/x/p.md", "---\ntype: demo\napp: none\n---\n## Before\n- terminal \"T\"\n## Before\n- terminal \"T\"\n## F\n- terminal \"T\"\n")
	if len(two.Problems) != 1 || !strings.Contains(two.Problems[0].Msg, "second ## Before") {
		t.Errorf("two Befores: %v", two.Problems)
	}
	only := Parse("/x/p.md", "---\ntype: demo\napp: none\n---\n## Before\n- terminal \"T\"\n")
	if len(only.Problems) == 0 || !strings.Contains(only.Problems[0].Msg, "no features") {
		t.Errorf("only a Before: %v", only.Problems)
	}
	// JSON: setup as a string or a list, prepare and cleanup, a Before
	j, err := ParseJSON("/x/b.json", []byte(`{"type":"demo","app":"none","setup":"a","prepare":["b","c"],"cleanup":["d"],"stop":["e","f"],
"features":[{"name":"Before","blocks":[{"steps":["terminal \"T\""]}]},{"name":"F","blocks":[{"steps":["terminal \"T\""]}]}]}`))
	if err != nil || len(j.Problems) > 0 || len(j.Setups) != 1 || len(j.Prelude) != 2 || len(j.Cleanups) != 1 || len(j.Stops) != 2 || j.Before == nil || len(j.Features) != 1 {
		t.Errorf("json: %v %+v", err, j)
	}
}

// fake writes a script that appends its name (and its arguments) to the log.
func fake(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
}

func readLog(t *testing.T, p string) string {
	b, _ := os.ReadFile(p)
	return strings.Join(strings.Fields(string(b)), " ")
}

// TestLifecycleOrder: start/prepare/step in the order written, stop: while
// the programs run, the programs stopped, then cleanup:; and stop: and
// cleanup: run after a failed prepare: too.
func TestLifecycleOrder(t *testing.T) {
	noWindows(t)
	dir := t.TempDir()
	log := filepath.Join(dir, "log")
	fake(t, dir, "rec", `echo "$@" >> "`+log+`"`)
	fake(t, dir, "prog", `trap 'echo stopped-$1 >> "`+log+`"; exit 0' TERM; echo started-$1 >> "`+log+`"; echo up; while :; do sleep 0.05; done`)
	src := "---\ntype: demo\napp: none\nstart: ./prog a · wait for \"up\"\nprepare: ./rec prepare1\nstep: ./rec step2\nstart: ./prog b · wait for \"up\"\n" +
		"stop: ./rec stop1\nstop: ./rec stop2\ncleanup: ./rec cleanup1\ncleanup: ./rec cleanup2\n---\n## F\n- terminal \"T\"\n"
	path := filepath.Join(dir, "d.md")
	os.WriteFile(path, []byte(src), 0o644)
	s, _ := ParseFile(path)
	r := &runner{s: s, o: Options{Log: io.Discard, Out: dir}, procs: map[string]*proc{}, terms: map[string]*term{}, abort: make(chan struct{})}
	if f := r.prelude(); f != nil {
		t.Fatalf("prelude: %+v", f)
	}
	r.shutdown()
	got := readLog(t, log)
	want := "started-a prepare1 step2 started-b stop1 stop2 stopped-"
	if !strings.HasPrefix(got, want) || !strings.HasSuffix(got, "cleanup1 cleanup2") || !strings.Contains(got, "stopped-a") || !strings.Contains(got, "stopped-b") {
		t.Errorf("order:\n%s", got)
	}
	if len(r.post) != 4 || r.post[3].Text != "cleanup: ./rec cleanup2" {
		t.Errorf("post: %+v", r.post)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "lifecycle.log")); !strings.Contains(string(b), "prepare: ./rec prepare1") {
		t.Errorf("lifecycle.log: %s", b)
	}
	// a failed prepare: stops the prelude; stop: and cleanup: still run
	os.Remove(log)
	src = strings.Replace(src, "step: ./rec step2", "step: ./rec step2; echo broken; exit 3", 1)
	os.WriteFile(path, []byte(src), 0o644)
	s, _ = ParseFile(path)
	r = &runner{s: s, o: Options{Log: io.Discard}, procs: map[string]*proc{}, terms: map[string]*term{}, abort: make(chan struct{})}
	f := r.prelude()
	if f == nil || f.Status != "failed" || !strings.Contains(f.Output, "broken") || f.Line != 6 {
		t.Fatalf("failed prepare: %+v", f)
	}
	r.shutdown()
	if got := readLog(t, log); got != "started-a prepare1 step2 stop1 stop2 stopped-a cleanup1 cleanup2" {
		t.Errorf("after a failure:\n%s", got)
	}
	// a prepare: past its within: is stopped
	os.WriteFile(path, []byte("---\ntype: demo\napp: none\nprepare: sleep 30 · within 300ms\n---\n## F\n- terminal \"T\"\n"), 0o644)
	s, _ = ParseFile(path)
	r = &runner{s: s, o: Options{Log: io.Discard}, procs: map[string]*proc{}, terms: map[string]*term{}, abort: make(chan struct{})}
	t0 := time.Now()
	if f := r.prelude(); f == nil || !strings.Contains(f.Found, "still running after 300ms") || time.Since(t0) > 4*time.Second {
		t.Errorf("within: %+v", f)
	}
}

// TestStage: a deck's setup: and start: in order, then stop:, the programs,
// cleanup:.
func TestStage(t *testing.T) {
	noWindows(t)
	dir := t.TempDir()
	log := filepath.Join(dir, "log")
	fake(t, dir, "rec", `echo "$@" >> "`+log+`"`)
	fake(t, dir, "prog", `trap 'echo stopped >> "`+log+`"; exit 0' TERM; echo ready; while :; do sleep 0.05; done`)
	path := filepath.Join(dir, "talk.md")
	os.WriteFile(path, []byte("---\ntitle: T\nsetup: ./rec install\nstart: ./prog as broker · wait for \"ready\"\nsetup: ./rec seed\nstop: ./rec stop\ncleanup: ./rec clean\n---\n# T\n"), 0o644)
	d, err := deck.Parse(path)
	if err != nil || len(d.Problems) > 0 {
		t.Fatal(err, d.Problems)
	}
	s := ParseStage(path, d.Meta.Lifecycle)
	if len(s.Problems) > 0 || len(s.Prelude) != 3 || s.Empty() {
		t.Fatalf("%v %d", s.Problems, len(s.Prelude))
	}
	var seen []string
	st := NewStage(s, t.TempDir(), io.Discard, func(sr StepResult) { seen = append(seen, sr.Text+" "+sr.Status) })
	if err := st.Up(); err != nil {
		t.Fatal(err)
	}
	if !portOpenless(st) {
		t.Error("programs")
	}
	post := st.Down()
	if got := readLog(t, log); got != "install seed stop stopped clean" {
		t.Errorf("order: %s", got)
	}
	if len(seen) != 5 || seen[1] != `start: ./prog as broker · wait for "ready" passed` || len(post) != 2 {
		t.Errorf("progress: %q %v", seen, post)
	}
	st.Down() // once
	if got := readLog(t, log); got != "install seed stop stopped clean" {
		t.Errorf("twice: %s", got)
	}
}

func portOpenless(st *Stage) bool { return len(st.Programs()) == 1 && st.Programs()["broker"] != "" }

// TestRunLifecycle: in a whole run (headless Chrome), ## Before before the
// features and not a feature, setup: before each feature, ## After after
// them, then stop:, the programs, cleanup:; a failed ## Before skips the
// features, and a failed ## After fails the run without failing a feature.
func TestRunLifecycle(t *testing.T) {
	noWindows(t)
	chrome := tools.Find("chrome")
	if chrome == "" || testing.Short() {
		t.Skip("needs Chrome")
	}
	dir := t.TempDir()
	log := filepath.Join(dir, "log")
	fake(t, dir, "rec", `echo "$@" >> "`+log+`"`)
	fake(t, dir, "prog", `trap 'echo stopped >> "`+log+`"; exit 0' TERM; echo up; while :; do sleep 0.05; done`)
	src := func(before, after string) string {
		return "---\ntype: demo\napp: none\nshell: sh\nstart: ./prog · wait for \"up\"\nprepare: ./rec prepare\nsetup: ./rec setup\nstop: ./rec stop\ncleanup: ./rec cleanup\n---\n" +
			"## Before\n- terminal \"B\"\n- run \"./rec before\" · wait\n" + before + "\n" +
			"## One\n- terminal \"T\"\n- run \"./rec one\" · wait\n\n## Two\n- terminal \"T\"\n- run \"./rec two\" · wait\n\n" +
			"## After\n- terminal \"A\"\n- run \"./rec after\" · wait\n" + after + "\n"
	}
	run := func(text string) *Report {
		os.Remove(log)
		path := filepath.Join(dir, "life.md")
		os.WriteFile(path, []byte(text), 0o644)
		s, err := ParseFile(path)
		if err != nil || len(s.Problems) > 0 {
			t.Fatal(err, s.Problems)
		}
		rep, err := Run(s, Options{Chrome: chrome, Log: io.Discard, Out: t.TempDir()})
		if err != nil {
			t.Fatal(err)
		}
		return rep
	}
	rep := run(src("", ""))
	if got := readLog(t, log); got != "prepare before setup one setup two after stop stopped cleanup" {
		t.Errorf("order: %s", got)
	}
	if !rep.OK() || rep.Passed != 2 || len(rep.Features) != 2 || rep.Before == nil || rep.After == nil || len(rep.Start) != 2 || len(rep.Teardown) != 2 {
		t.Errorf("report: %+v", rep)
	}
	rep = run(src(`- expect output "never" · within 1s`, ""))
	if got := readLog(t, log); got != "prepare before after stop stopped cleanup" {
		t.Errorf("a failed Before: %s", got)
	}
	if rep.OK() || rep.Skipped != 2 || rep.Failed != 0 || rep.Before.Status != "failed" {
		t.Errorf("a failed Before: %+v", rep)
	}
	rep = run(src("", `- expect output "never" · within 1s`))
	if rep.OK() || rep.Passed != 2 || rep.After.Status != "failed" {
		t.Errorf("a failed After: %+v", rep)
	}
	j, _ := rep.JUnit()
	var x struct {
		Failures int `xml:"failures,attr"`
		Suites   []struct {
			Cases []struct {
				Name string `xml:"name,attr"`
			} `xml:"testcase"`
		} `xml:"testsuite"`
	}
	xml.Unmarshal(j, &x)
	if x.Failures != 1 || len(x.Suites) != 1 || len(x.Suites[0].Cases) != 3 || x.Suites[0].Cases[2].Name != "After the features" {
		t.Errorf("junit: %s", j)
	}
}

// TestCtrlC: Ctrl-C during a prepare: still runs stop: and cleanup: (a child
// process, interrupted).
func TestCtrlC(t *testing.T) {
	noWindows(t)
	if os.Getenv("SI_LIFE_CHILD") != "" {
		s, _ := ParseFile(os.Getenv("SI_LIFE_CHILD"))
		if _, err := Run(s, Options{Chrome: os.Getenv("SI_LIFE_CHROME"), Log: io.Discard}); err != nil {
			t.Fatal(err)
		}
		return
	}
	chrome := tools.Find("chrome")
	if chrome == "" || testing.Short() {
		t.Skip("needs Chrome")
	}
	dir := t.TempDir()
	log := filepath.Join(dir, "log")
	fake(t, dir, "rec", `echo "$@" >> "`+log+`"`)
	path := filepath.Join(dir, "c.md")
	os.WriteFile(path, []byte("---\ntype: demo\napp: none\nprepare: ./rec waiting; sleep 60 · within 2m\nstop: ./rec stop\ncleanup: ./rec cleanup\n---\n## F\n- terminal \"T\"\n"), 0o644)
	cmd := exec.Command(os.Args[0], "-test.run", "^TestCtrlC$")
	cmd.Env = append(os.Environ(), "SI_LIFE_CHILD="+path, "SI_LIFE_CHROME="+chrome)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 300 && readLog(t, log) != "waiting"; i++ {
		time.Sleep(50 * time.Millisecond)
	}
	cmd.Process.Signal(syscall.SIGINT)
	err := cmd.Wait()
	if ee, ok := err.(*exec.ExitError); !ok || ee.ExitCode() != 130 {
		t.Errorf("exit: %v", err)
	}
	if got := readLog(t, log); got != "waiting stop cleanup" {
		t.Errorf("after Ctrl-C: %s", got)
	}
}
