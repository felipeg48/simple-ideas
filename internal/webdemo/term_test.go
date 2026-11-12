package webdemo

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/felipeg48/simple-ideas/internal/cdp"
	"github.com/felipeg48/simple-ideas/internal/tools"
)

func TestTerminalSteps(t *testing.T) {
	cases := []struct {
		in   string
		want Step
	}{
		{`terminal "Broker"`, Step{Kind: "terminal", Name: "Broker", Opts: map[string]string{}}},
		{`terminal "Broker" · window=mac title="the broker · 1" theme=dusk backdrop=graphite size=100x30`, Step{Kind: "terminal", Name: "Broker",
			Opts: map[string]string{"window": "mac", "title": "the broker · 1", "theme": "dusk", "backdrop": "graphite", "size": "100x30"}}},
		{`browser`, Step{Kind: "browser"}},
		{`wait for "The five hand-offs" · within 40s`, Step{Kind: "waitfor", Target: Locator{Name: "The five hand-offs"}, Within: 40 * time.Second}},
		{`type ", live"`, Step{Kind: "type", Value: Value{Text: ", live"}}},
		{`run "./bin/send orders hello"`, Step{Kind: "run", Value: Value{Text: "./bin/send orders hello"}}},
		{`run "make seed" · wait`, Step{Kind: "run", Value: Value{Text: "make seed"}, Block: true}},
		{`run "make seed" · wait · within 90s`, Step{Kind: "run", Value: Value{Text: "make seed"}, Block: true, Within: 90 * time.Second}},
		{`run "false" · fails`, Step{Kind: "run", Value: Value{Text: "false"}, Block: true, Code: -1}},
		{`run "make bad" · exit 2`, Step{Kind: "run", Value: Value{Text: "make bad"}, Block: true, Code: 2}},
		{`press ctrl+c`, Step{Kind: "press", Value: Value{Text: "ctrl+c"}}},
		{`expect output "sent 1"`, Step{Kind: "expect-output", Value: Value{Text: "sent 1"}}},
		{`expect output "Server startup complete" · within 30s`, Step{Kind: "expect-output", Value: Value{Text: "Server startup complete"}, Within: 30 * time.Second}},
		{`expect no output "ERROR"`, Step{Kind: "expect-no-output", Value: Value{Text: "ERROR"}}},
		{`wait for output "ready" · within 1m`, Step{Kind: "waitfor-output", Value: Value{Text: "ready"}, Within: time.Minute}},
		{`wait for output "ready" · within 1 minute`, Step{}},
		{`wait for output "ready" · within 60s`, Step{Kind: "waitfor-output", Value: Value{Text: "ready"}, Within: time.Minute}},
		{`start "./load --rate 100" as load · wait for "running"`, Step{Kind: "start", Name: "load", Value: Value{Text: "./load --rate 100"}, Ready: Ready{Text: "running"}}},
		{`start "./api" as api · wait for port 8080 · within 60s`, Step{Kind: "start", Name: "api", Value: Value{Text: "./api"}, Ready: Ready{Port: 8080}, Within: time.Minute}},
		{`start "./tick" as tick · wait 3s`, Step{Kind: "start", Name: "tick", Value: Value{Text: "./tick"}, Ready: Ready{Delay: 3 * time.Second}}},
		{`stop load`, Step{Kind: "stop", Name: "load"}},
		{`stop load · wait`, Step{Kind: "stop", Name: "load", Block: true}},
		{`expect output of load "VERDICT passed"`, Step{Kind: "expect-output", Name: "load", Value: Value{Text: "VERDICT passed"}}},
		{`expect load running`, Step{Kind: "expect-proc", Name: "load", State: "running"}},
		{`expect load exited 0`, Step{Kind: "expect-proc", Name: "load", State: "exited"}},
		{`expect load exited 3`, Step{Kind: "expect-proc", Name: "load", State: "exited", Code: 3}},
		{`note "Ready" at line "Server startup"`, Step{Kind: "note", Value: Value{Text: "Ready"}, Target: Locator{Name: "Server startup"}, AtLine: true}},
		{`note "Ready" at line "a · b" · for 2s`, Step{Kind: "note", Value: Value{Text: "Ready"}, Target: Locator{Name: "a · b"}, AtLine: true, For: 2 * time.Second}},
		{`expect "Pay" visible`, Step{Kind: "expect-state", Target: Locator{Name: "Pay"}, State: "visible"}},
	}
	for _, c := range cases {
		st, err := ParseStep(c.in)
		if c.want.Kind == "" {
			if err == nil {
				t.Errorf("%s: no error", c.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: %v", c.in, err)
			continue
		}
		got := *st
		got.Text, c.want.Text = "", ""
		if fmt.Sprintf("%+v", got) != fmt.Sprintf("%+v", c.want) {
			t.Errorf("%s:\n got %+v\nwant %+v", c.in, got, c.want)
		}
	}
	for _, bad := range []string{`terminal Broker`, `terminal "B" · window=linux`, `terminal "B" · size=10`, `terminal "B" · colour=red`, `browser now`,
		`run ls`, `run "ls" · fast`, `run "ls" · exit x`, `start "x"`, `start "x" as`, `start "x" as a · wait for nothing`, `stop "load"`, `stop load · soon`,
		`expect output sent`, `expect no output "x" · within 2s`, `expect load exited`, `expect output of "load" "x"`, `note "x" at line`} {
		if _, err := ParseStep(bad); err == nil {
			t.Errorf("%s: no error", bad)
		}
	}
}

const termSample = `---
type: demo
app: http://localhost:15672
shell: bash
start: ./bin/broker --port 5672 · wait for "Server startup complete"
start: "./bin/api --port 8080" as api · wait for port 8080 · within 60s
start: ./bin/worker as worker · wait 2s
stop: make demo-down
---

## Broker

Say: Start a client.
- terminal "Client" · window=terminal backdrop=graphite
- run "./bin/send orders hello" · wait
- expect output "sent 1"
- note "One message" at line "sent 1"
- press ctrl+c
- start "./bin/load" as load · wait for "running"
- expect output of load "running"
- expect output of api "listening"
- expect load running
- stop load · wait
- expect load exited 0
- browser
- go /#/queues/orders
- expect text "1 message"
`

func TestParseTerminalScript(t *testing.T) {
	s := Parse("/x/broker.md", termSample)
	if len(s.Problems) > 0 {
		t.Fatalf("problems: %v", s.Problems)
	}
	if s.Shell != "bash" || len(s.Stops) != 1 || s.Stops[0].Cmds[0] != "make demo-down" || len(s.Starts) != 3 {
		t.Fatalf("front matter: %+v", s)
	}
	want := []Program{
		{Cmd: "./bin/broker --port 5672", Name: "start-1", Ready: Ready{Text: "Server startup complete"}, Line: 5},
		{Cmd: "./bin/api --port 8080", Name: "api", Ready: Ready{Port: 8080}, Within: time.Minute, Line: 6},
		{Cmd: "./bin/worker", Name: "worker", Ready: Ready{Delay: 2 * time.Second}, Line: 7},
	}
	for i, p := range s.Starts {
		p.Text = ""
		if *p != want[i] {
			t.Errorf("start %d: %+v", i, *p)
		}
	}
	j, err := ParseJSON("/x/b.json", []byte(`{"type":"demo","app":"http://x","shell":"zsh","stop":"make down","start":["./b · wait for port 1"],
"features":[{"name":"F","blocks":[{"say":"s","steps":["terminal \"T\"","run \"ls\""]}]}]}`))
	if err != nil || len(j.Problems) > 0 || j.Shell != "zsh" || len(j.Stops) != 1 || j.Stops[0].Cmds[0] != "make down" || len(j.Starts) != 1 || j.Starts[0].Ready.Port != 1 {
		t.Errorf("json: %v %+v", err, j)
	}
}

// TestCheckScenes: si check follows the scenes and the programs.
func TestCheckScenes(t *testing.T) {
	src := "---\ntype: demo\napp: http://x\nstart: ./b as broker\n---\n\n## F\n\n" +
		"- run \"ls\"\n" + // 9: no terminal yet
		"- terminal \"T\"\n" +
		"- click \"Pay\"\n" + // 11: the terminal is in front
		"- note \"n\" at \"Pay\"\n" + // 12: an element in a terminal
		"- press cmd+k\n" + // 13: no such key in a terminal
		"- expect output of load \"x\"\n" + // 14: no program load
		"- stop broker\n" + // fine: a front matter start
		"- browser\n" +
		"- expect output \"x\"\n" + // 17: no terminal in front
		"- note \"n\" at line \"x\"\n" + // 18: no terminal
		"- start \"./l\" as load\n" +
		"- start \"./l\" as load\n" // 20: twice
	s := Parse("/x/s.md", src)
	var got []int
	for _, p := range s.Problems {
		got = append(got, p.Line)
	}
	if fmt.Sprint(got) != "[9 11 12 13 14 17 18 20]" {
		t.Errorf("problems at %v:\n%v", got, s.Problems)
	}
	if s := Parse("/x/s.md", "---\ntype: demo\napp: http://x\nstart: ./a as x\nstart: ./b as x\nstart:  · wait 2s\nstart: ./c · wait for soon\n---\n## F\n- wait 1s\n"); len(s.Problems) != 3 {
		t.Errorf("bad starts: %v", s.Problems)
	}
	if s := Parse("/x/s.md", "---\ntype: demo\napp: none\n---\n## A\n- go /\n## B\n- terminal \"T\"\n- browser\n"); len(s.Problems) != 3 {
		t.Errorf("app: none: %v", s.Problems)
	}
}

func TestStripper(t *testing.T) {
	o := newOutput(nil, nil)
	for _, chunk := range []string{
		"\x1b[32mgreen\x1b[0m text\r\n",
		"\x1b]133;D;0\x07\x1b[36mdir\x1b[0m $ ",
		"lx\bs\r\n",          // a backspace takes back the x
		"a\x1b[", "1;31mb\n", // a sequence split across two writes
		"\x1b]0;title\x1b\\c\n",  // OSC ended by ESC \
		"\x1b(Bd\x1b=e\n",        // charset and keypad escapes
		"\x1b]133;D;", "127\x07", // an exit code split across writes
		"progress 10%\rprogress 100%\n",
	} {
		o.Write([]byte(chunk))
	}
	want := "green text\ndir $ ls\nab\nc\nde\nprogress 10%progress 100%\n"
	if got := o.text(0); got != want {
		t.Errorf("plain:\n%q\nwant\n%q", got, want)
	}
	if c0, _ := o.code(0); c0 != 0 {
		t.Error("first code")
	}
	if c1, ok := o.code(1); !ok || c1 != 127 {
		t.Errorf("second code %d %v", c1, ok)
	}
	if _, ok := o.code(2); ok {
		t.Error("a third code")
	}
	if !shows(o.text(0), "green   text") || shows(o.text(0), "greentext") {
		t.Error("shows: white space")
	}
	if o.tail(2) != "ab\nc\nde\nprogress 10%progress 100%"[strings.Index("ab\nc\nde\nprogress 10%progress 100%", "de"):] {
		t.Errorf("tail: %q", o.tail(2))
	}
}

func TestMasking(t *testing.T) {
	log := filepath.Join(t.TempDir(), "x.log")
	f, _ := os.Create(log)
	r := &runner{secrets: []string{"hunter2"}}
	o := newOutput(r.mask, f)
	o.Write([]byte("password=\x1b[1mhunter2\x1b[0m ok\n"))
	f.Close()
	raw, _, _ := o.from(0)
	b, _ := os.ReadFile(log)
	for name, s := range map[string]string{"plain": o.text(0), "raw": string(raw), "log": string(b)} {
		if strings.Contains(s, "hunter2") || !strings.Contains(s, "••••") {
			t.Errorf("%s: %q", name, s)
		}
	}
	t.Setenv("WD_TOKEN", "s3cr3t")
	t.Setenv("HOME", "/home/x")
	s := Parse("/x/a.md", "---\ntype: demo\napp: http://x\nstart: ./api --token $WD_TOKEN --home $HOME\n---\n## F\n- terminal \"T\"\n- run \"curl -H ${WD_TOKEN} x\"\n")
	if got := commandSecrets(s); fmt.Sprint(got) != "[s3cr3t s3cr3t]" {
		t.Errorf("command secrets: %v", got)
	}
}

func noWindows(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process groups and sh: unix")
	}
}

func alive(pid int) bool {
	// a zombie still answers kill 0: ask ps for its state
	out, err := exec.Command("ps", "-o", "stat=", "-p", strconv.Itoa(pid)).Output()
	return err == nil && len(out) > 0 && !strings.HasPrefix(strings.TrimSpace(string(out)), "Z")
}

func waitGone(t *testing.T, pid int) {
	t.Helper()
	for i := 0; i < 50; i++ {
		if !alive(pid) {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Errorf("pid %d is still running", pid)
}

// TestProcLifetime: a program and what it starts stop together; one that
// ignores SIGTERM is killed after the grace; its log is written.
func TestProcLifetime(t *testing.T) {
	noWindows(t)
	dir := t.TempDir()
	log := filepath.Join(dir, "p.log")
	p, err := startProc("p", `sleep 300 & echo "child $!"; echo ready; wait`, dir, log, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.ready(Ready{Text: "ready"}, 5*time.Second); err != nil {
		t.Fatal(err)
	}
	var child int
	fmt.Sscanf(p.out.text(0), "child %d", &child)
	if child == 0 || !alive(child) || p.exited() {
		t.Fatalf("child %d", child)
	}
	p.stop()
	waitGone(t, child)
	waitGone(t, p.cmd.Process.Pid)
	if b, _ := os.ReadFile(log); !strings.Contains(string(b), "ready") {
		t.Errorf("log: %q", b)
	}
	p.stop() // twice is fine

	old := stopGrace
	stopGrace = 300 * time.Millisecond
	defer func() { stopGrace = old }()
	q, err := startProc("q", `trap "" TERM; echo up; while :; do sleep 1; done`, dir, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := q.ready(Ready{Text: "up"}, 5*time.Second); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	q.stop()
	if !q.exited() || time.Since(start) > 3*time.Second {
		t.Errorf("not killed after the grace: %v", time.Since(start))
	}

	// a program that exits before it is ready fails with its output
	e, _ := startProc("e", `echo boom; exit 4`, dir, "", nil)
	err = e.ready(Ready{Text: "never"}, 5*time.Second)
	if fe, ok := err.(*failure); !ok || !strings.Contains(fe.found, "status 4") || !strings.Contains(fe.output, "boom") {
		t.Errorf("early exit: %v", err)
	}
	// a delay is a plain wait
	d, _ := startProc("d", `sleep 5`, dir, "", nil)
	if err := d.ready(Ready{Delay: 100 * time.Millisecond}, 0); err != nil {
		t.Errorf("delay: %v", err)
	}
	d.stop()
}

func TestWaitForPort(t *testing.T) {
	noWindows(t)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	p, _ := startProc("p", "sleep 30", t.TempDir(), "", nil)
	defer p.stop()
	if err := p.ready(Ready{Port: port}, 2*time.Second); err != nil {
		t.Errorf("open port: %v", err)
	}
	ln.Close()
	err = p.ready(Ready{Port: port}, 300*time.Millisecond)
	if fe, ok := err.(*failure); !ok || !strings.Contains(fe.expected, fmt.Sprintf("wait for port %d", port)) {
		t.Errorf("closed port: %v", err)
	}
}

// TestShellPrompt: si's prompt reports each command's exit status, in sh,
// bash and zsh, and closing the terminal stops what runs in it.
func TestShellPrompt(t *testing.T) {
	noWindows(t)
	rc := t.TempDir()
	if err := shellRC(rc); err != nil {
		t.Fatal(err)
	}
	for _, sh := range []string{"sh", "bash", "zsh"} {
		path, err := exec.LookPath(sh)
		if err != nil {
			continue
		}
		t.Run(sh, func(t *testing.T) {
			tm, err := openShell("T", path, t.TempDir(), rc, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer tm.close()
			if err := tm.prompt(10 * time.Second); err != nil {
				t.Fatal(err)
			}
			at, n := tm.out.mark()
			tm.runAt, tm.ran = at, true
			tm.send("echo hi there; (exit 3)\r")
			for i := 0; i < 100; i++ {
				if _, ok := tm.out.code(n); ok {
					break
				}
				time.Sleep(50 * time.Millisecond)
			}
			if c, ok := tm.out.code(n); !ok || c != 3 {
				t.Fatalf("exit code %d %v\n%q", c, ok, tm.out.text(0))
			}
			if s := tm.since(); !strings.HasPrefix(s, "hi there\n") {
				t.Errorf("since the run: %q", s)
			}
			tm.send("sleep 300 & echo \"bg $!\"\r")
			var bg int
			for i := 0; i < 100 && bg == 0; i++ {
				time.Sleep(50 * time.Millisecond)
				if k := strings.Index(tm.out.text(0), "\nbg "); k >= 0 {
					fmt.Sscanf(tm.out.text(0)[k+1:], "bg %d", &bg)
				}
			}
			if bg == 0 {
				t.Fatalf("no background pid: %q", tm.out.text(0))
			}
			tm.close()
			waitGone(t, bg)
		})
	}
}

// TestTerminalPage: the page frames xterm.js, takes size=, and finds a line
// for a note.
func TestTerminalPage(t *testing.T) {
	noWindows(t)
	chrome := tools.Find("chrome")
	if chrome == "" || testing.Short() {
		t.Skip("needs Chrome")
	}
	rc := t.TempDir()
	shellRC(rc)
	srv, err := newTermServer()
	if err != nil {
		t.Fatal(err)
	}
	defer srv.close()
	tm, err := openShell("Client", "sh", t.TempDir(), rc, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tm.close()
	tm.opts = map[string]string{"size": "90x22", "window": "mac", "title": "the client"}
	srv.add(tm)
	c, err := cdp.Start(chrome, 1440, 900)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	c.Call("Page.enable", nil, nil)
	c.Call("Page.addScriptToEvaluateOnNewDocument", map[string]any{"source": overlaySource()}, nil)
	c.Call("Page.navigate", map[string]any{"url": srv.url(tm)}, nil)
	eval := func(js string) string {
		v, err := c.Eval(js)
		if err != nil {
			t.Fatalf("%s: %v", js, err)
		}
		return string(v)
	}
	for i := 0; i < 100 && eval(`!!window.__siTermReady`) != "true"; i++ {
		time.Sleep(50 * time.Millisecond)
	}
	tm.prompt(10 * time.Second)
	tm.send("echo hello there\r")
	for i := 0; i < 100 && eval(`__siTermLine("hello there")`) != "true"; i++ {
		time.Sleep(50 * time.Millisecond)
	}
	if got := eval(`document.querySelectorAll('.xterm-rows > div').length + ' ' + document.querySelector('.si-cw-title').textContent + ' ' + !!document.querySelector('.si-cw-lights')`); got != `"22 the client true"` {
		t.Errorf("page: %s", got)
	}
	if got := eval(`__siTermLine("hello there")`); got != "true" {
		t.Errorf("line: %s", got)
	}
	if got := eval(`__siTermLine("not on screen")`); got != "false" {
		t.Errorf("no line: %s", got)
	}
	// a keyed URL: without the key, nothing
	if res, err := http.Get(strings.Replace(srv.url(tm), srv.base, srv.base[:strings.LastIndex(srv.base, "/")]+"/nokey", 1)); err == nil {
		if res.StatusCode != 404 {
			t.Errorf("no key: %d", res.StatusCode)
		}
		res.Body.Close()
	}
}

// TestTerminalFailure: in si demo test, a failed expect output fails its
// feature with the terminal's last lines, in the report and in JUnit, and
// a feature's start step is stopped when the feature ends.
func TestTerminalFailure(t *testing.T) {
	noWindows(t)
	chrome := tools.Find("chrome")
	if chrome == "" || testing.Short() {
		t.Skip("needs Chrome")
	}
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "app"), 0o755)
	os.WriteFile(filepath.Join(dir, "app", "index.html"), []byte("<p>hello</p>"), 0o644)
	src := "---\ntype: demo\napp: app\nshell: sh\n---\n\n## Fails\n\n" +
		"- terminal \"T\"\n- run \"echo nope; echo line two\"\n- expect output \"yes\" · within 1s\n- run \"echo never\"\n\n" +
		"## Program\n\n- start \"echo up; exec sleep 301\" as slow · wait for \"up\"\n- browser\n- expect text \"hello\"\n"
	path := filepath.Join(dir, "t.md")
	os.WriteFile(path, []byte(src), 0o644)
	s, err := ParseFile(path)
	if err != nil || len(s.Problems) > 0 {
		t.Fatal(err, s.Problems)
	}
	out := t.TempDir()
	var log strings.Builder
	rep, err := Run(s, Options{Chrome: chrome, Out: out, Log: &log})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Passed != 1 || rep.Failed != 1 {
		t.Fatalf("%d passed, %d failed\n%s", rep.Passed, rep.Failed, log.String())
	}
	f := rep.Features[0]
	st := f.Steps[2]
	if st.Status != "failed" || st.Expected != `terminal "T" shows "yes"` || !strings.Contains(st.Output, "nope\nline two") || f.Steps[3].Status != "skipped" {
		t.Errorf("failed step: %+v", st)
	}
	j, _ := rep.JUnit()
	if !strings.Contains(string(j), "output (last lines):") || !strings.Contains(string(j), "line two") {
		t.Errorf("junit: %s", j)
	}
	rep.Write(out)
	if b, _ := os.ReadFile(filepath.Join(out, "report.html")); !strings.Contains(string(b), `<pre class="out">`) {
		t.Error("report.html: no output")
	}
	if b, _ := os.ReadFile(filepath.Join(out, "slow.log")); string(b) != "up\n" {
		t.Errorf("slow.log: %q", b)
	}
	// app: none, a terminal-only demo
	only := filepath.Join(dir, "only.md")
	os.WriteFile(only, []byte("---\ntype: demo\napp: none\nshell: sh\n---\n## T\n- terminal \"T\"\n- run \"echo fine\" · wait\n- expect output \"fine\"\n"), 0o644)
	s, _ = ParseFile(only)
	if rep, err := Run(s, Options{Chrome: chrome, Log: &log}); err != nil || !rep.OK() {
		t.Errorf("app: none: %v\n%s", err, log.String())
	}
	if pids, _ := exec.Command("pgrep", "-f", "sleep 301").Output(); len(pids) > 0 {
		for _, pid := range strings.Fields(string(pids)) {
			if n, _ := strconv.Atoi(pid); alive(n) {
				t.Errorf("the start step's program (pid %d) outlived its feature", n)
			}
		}
	}
}

// A backslash in a quoted value is kept unless it escapes a quote or itself,
// so shell commands keep their \n.
func TestQuotedKeepsBackslashes(t *testing.T) {
	for in, want := range map[string]string{
		`"printf 'a\nb'"`:    `printf 'a\nb'`,
		`"say \"hi\""`:       `say "hi"`,
		`"C:\\path"`:         `C:\path`,
		`"tab\there" · wait`: `tab\there`,
	} {
		got, _, err := quoted(in)
		if err != nil || got != want {
			t.Errorf("quoted(%s) = %q, %v; want %q", in, got, err, want)
		}
	}
}

func TestAppURLs(t *testing.T) {
	for app, want := range map[string][2]string{
		"http://127.0.0.1:7310":            {"http://127.0.0.1:7310", "http://127.0.0.1:7310/"},
		"http://127.0.0.1:7393/?token=abc": {"http://127.0.0.1:7393", "http://127.0.0.1:7393/?token=abc"},
		"https://shop.test/app/?t=1#home":  {"https://shop.test/app", "https://shop.test/app/?t=1#home"},
	} {
		if b, s := appURLs(app); b != want[0] || s != want[1] {
			t.Errorf("appURLs(%q) = %q, %q; want %q", app, b, s, want)
		}
	}
}
