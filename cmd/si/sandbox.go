package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/felipeg48/simple-ideas/internal/config"
	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/felipeg48/simple-ideas/internal/sandbox"
)

const sandboxUsage = `si sandbox: a lab's own Docker box (RULES §37)

  si sandbox host init [--cpu N] [--memory GiB] [--disk GiB]
                                   make the si-sandbox Colima VM: Docker, sysbox
                                   (boxes run unprivileged), no host folder but
                                   an empty share; default half the cores, 3/4
                                   of the memory, 100 GiB
  si sandbox host start|stop|status|rm
  si sandbox build <dir> -t <name:tag>
                                   docker build into the sandbox host
  si sandbox try <page.md|image> [--trusted-sandbox]
                                   start a box as si play would and open a shell
                                   in it here; the box is removed when it ends

  The host is si.conf sandbox.host: colima (default), docker, or a socket path
  ($SI_SANDBOX_HOST first). --trusted-sandbox lets a box run privileged on a host
  without sysbox: only for images and people you trust.
`

func runSandbox(argv []string, stdout, stderr io.Writer) int {
	fl := flag.NewFlagSet("si sandbox", flag.ContinueOnError)
	fl.SetOutput(io.Discard)
	sz := sandbox.DefaultHostSize()
	var tag string
	var trusted, help bool
	fl.IntVar(&sz.CPU, "cpu", sz.CPU, "")
	fl.IntVar(&sz.MemoryGiB, "memory", sz.MemoryGiB, "")
	fl.IntVar(&sz.DiskGiB, "disk", sz.DiskGiB, "")
	fl.StringVar(&tag, "t", "", "")
	fl.StringVar(&tag, "tag", "", "")
	fl.BoolVar(&trusted, "trusted-sandbox", false, "")
	fl.BoolVar(&help, "help", false, "")
	fl.BoolVar(&help, "h", false, "")
	var args []string
	for {
		if err := fl.Parse(argv); err != nil {
			fmt.Fprintf(stderr, "si sandbox: %v\n\n%s", err, sandboxUsage)
			return exitUsage
		}
		if argv = fl.Args(); len(argv) == 0 {
			break
		}
		args, argv = append(args, argv[0]), argv[1:]
	}
	if help || len(args) == 0 {
		fmt.Fprint(stdout, sandboxUsage)
		if help {
			return exitOK
		}
		return exitUsage
	}
	fail := func(err error) int {
		if err == nil {
			return exitOK
		}
		fmt.Fprintf(stderr, "si sandbox: %v\n", err)
		return exitProblems
	}
	switch args[0] {
	case "host":
		if len(args) != 2 {
			return usageErr(stderr, "si sandbox host init|start|stop|status|rm")
		}
		switch args[1] {
		case "init":
			return fail(sandbox.HostInit(sz, stdout))
		case "start":
			return fail(sandbox.HostStart(stdout))
		case "stop":
			return fail(sandbox.HostStop(stdout))
		case "status":
			return fail(sandbox.HostStatus(stdout))
		case "rm":
			return fail(sandbox.HostRemove(stdout))
		}
		return usageErr(stderr, "si sandbox host init|start|stop|status|rm")
	case "build":
		if len(args) != 2 || tag == "" {
			return usageErr(stderr, "si sandbox build <dir> -t <name:tag>")
		}
		sock, err := sandboxSocket(".")
		if err != nil {
			return fail(err)
		}
		return fail(dockerCLI(sock, stdout, stderr, "build", "-t", tag, args[1]))
	case "try":
		if len(args) != 2 {
			return usageErr(stderr, "si sandbox try <page.md|image>")
		}
		return fail(sandboxTry(args[1], trusted, stdout, stderr))
	}
	fmt.Fprintf(stderr, "si sandbox: %q is not host, build or try\n\n%s", args[0], sandboxUsage)
	return exitUsage
}

// sandboxSocket is the sandbox host's socket for a page or folder at p (its
// si.conf, then the user's).
func sandboxSocket(p string) (string, error) {
	dir := p
	if st, err := os.Stat(p); err == nil && !st.IsDir() {
		dir = filepath.Dir(p)
	}
	return sandbox.Socket(config.Load(filepath.Join(dir, "si.conf")).SandboxHost)
}

// dockerCLI runs the docker command against the sandbox host, attached to
// this terminal.
func dockerCLI(sock string, stdout, stderr io.Writer, args ...string) error {
	if _, err := exec.LookPath("docker"); err != nil {
		return fmt.Errorf("this needs the docker command (brew install docker)")
	}
	c := exec.Command("docker", append([]string{"-H", "unix://" + sock}, args...)...)
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, stdout, stderr
	return c.Run()
}

// sandboxTry starts the box of a page (or of a bare image) and opens a shell
// in it in this terminal; the box goes when the shell ends.
func sandboxTry(target string, trusted bool, stdout, stderr io.Writer) error {
	spec := deck.Sandbox{Image: target, Memory: deck.DefaultSandboxMemory, CPUs: deck.DefaultSandboxCPUs, Pids: deck.DefaultSandboxPids}
	dir := "."
	if strings.HasSuffix(target, ".md") {
		d, err := deck.Parse(target)
		if err != nil {
			return err
		}
		s, ok, problems := d.Meta.SandboxSpec()
		if len(problems) > 0 {
			return fmt.Errorf("%s: %s", target, problems[0])
		}
		if !ok {
			return fmt.Errorf("%s has no sandbox: line", target)
		}
		spec, dir = s, filepath.Dir(target)
	}
	sock, err := sandboxSocket(dir)
	if err != nil {
		return err
	}
	eng := sandbox.NewEngine(sock)
	ctx := context.Background()
	sandbox.Sweep(ctx, eng)
	box, err := sandbox.Up(ctx, eng, spec, dir, sandbox.Options{Trusted: trusted, Step: func(s string) { fmt.Fprintf(stdout, "  %s\n", s) }})
	if err != nil {
		return err
	}
	defer box.Down()
	printBox(stdout, box)
	return dockerCLI(sock, stdout, stderr, "exec", "-it", "-w", sandbox.LabDir, box.ID, "sh", "-c", "if command -v bash >/dev/null 2>&1; then exec bash -l; else exec sh -l; fi")
}

// printBox says which box runs and where its ports are.
func printBox(w io.Writer, b *sandbox.Box) {
	how := "unprivileged (sysbox)"
	if b.Runtime == "" {
		how = "PRIVILEGED (--trusted-sandbox)"
	}
	fmt.Fprintf(w, "si sandbox: %s from %s, %s\n", b.Name, b.Spec.Image, how)
	for _, p := range b.Spec.Ports {
		if hp, ok := b.Ports[p.Port]; ok {
			label := p.Label
			if label == "" {
				label = fmt.Sprintf("port %d", p.Port)
			}
			fmt.Fprintf(w, "  %s → http://%s\n", label, hp.Addr)
		}
	}
}

// labBox is the Docker box si play runs for the lab it serves.
type labBox struct {
	spec   deck.Sandbox
	course bool   // from _course.md: every page of the course uses the box
	dir    string // the folder of the page that asks for it
	file   string
	mu     sync.Mutex
	b      *sandbox.Box
}

// findLabBox is the sandbox: of what si play serves: the file, or a
// folder's top-level pages and its _course.md (a box for the whole course).
// Pages in subfolders never start one: serving a folder of examples must not
// start Docker boxes. Pages may share one box; two different boxes are a
// problem. nil: no page asks for one.
func findLabBox(src string) (*labBox, error) {
	var files []string
	if st, err := os.Stat(src); err == nil && !st.IsDir() {
		files = []string{src}
	} else {
		files, _ = deck.List(src)
		if c := filepath.Join(src, "_course.md"); fileExists(c) {
			files = append(files, c)
		}
	}
	var lb *labBox
	for _, f := range files {
		d, err := deck.Parse(f)
		if err != nil {
			continue
		}
		s, ok, problems := d.Meta.SandboxSpec()
		if !ok {
			continue
		}
		if len(problems) > 0 {
			return nil, fmt.Errorf("%s: %s", f, problems[0])
		}
		if lb != nil {
			if lb.spec.Image != s.Image {
				return nil, fmt.Errorf("%s and %s ask for different sandboxes (%s, %s): si play runs one box; serve them apart", lb.file, f, lb.spec.Image, s.Image)
			}
			continue
		}
		lb = &labBox{spec: s, dir: filepath.Dir(f), file: f, course: filepath.Base(f) == "_course.md"}
	}
	return lb, nil
}

func (l *labBox) get() *sandbox.Box {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b
}

// up makes the box (the getting-ready page shows it as one step).
func (l *labBox) up(trusted bool, out io.Writer) error {
	sock, err := sandboxSocket(l.dir)
	if err != nil {
		return err
	}
	eng := sandbox.NewEngine(sock)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	sandbox.Sweep(ctx, eng)
	b, err := sandbox.Up(ctx, eng, l.spec, l.dir, sandbox.Options{Trusted: trusted, Step: func(s string) { fmt.Fprintf(out, "  sandbox: %s\n", s) }})
	if err != nil {
		return err
	}
	l.mu.Lock()
	l.b = b
	l.mu.Unlock()
	printBox(out, b)
	fmt.Fprintf(out, "  the terminal drawer's new tabs open in the box\n")
	return nil
}

// down runs sandbox-cleanup: and removes the box.
func (l *labBox) down(out io.Writer) {
	l.mu.Lock()
	b := l.b
	l.b = nil
	l.mu.Unlock()
	if b != nil {
		fmt.Fprintf(out, "si play: removing the sandbox %s\n", b.Name)
		b.Down()
	}
}
