package sandbox

import (
	"archive/tar"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

// LabDir is the folder in the box where the lab's files go and commands run.
const LabDir = "/lab"

// SysboxRuntime is the runtime that runs a box with Docker inside it
// unprivileged.
const SysboxRuntime = "sysbox-runc"

// readyWait is how long the box's own dockerd may take to answer.
const readyWait = 90 * time.Second

// Options are how a box is made.
type Options struct {
	// Trusted lets a box run privileged when the host has no sysbox runtime
	// (--trusted-sandbox): whoever escapes such a box is root on the host.
	Trusted bool
	// Step reports progress: a short line per stage (pull, start, setup).
	Step func(string)
}

// Box is one running sandbox.
type Box struct {
	eng     *Engine
	ID      string // the container id
	Name    string // si-box-<id>
	Net     string // the network id
	Spec    deck.Sandbox
	Runtime string           // sysbox-runc, or "" for a privileged box
	Ports   map[int]HostPort // the box's sandbox-ports, as published on the sandbox host
}

// HostPort is where a box port is published: 127.0.0.1:port on the sandbox
// host (Colima forwards it to the same port on the Mac).
type HostPort struct {
	Addr  string // 127.0.0.1:54012
	Label string
}

// ErrNoSysbox says the host cannot run a box unprivileged.
var ErrNoSysbox = errors.New("the sandbox host has no sysbox runtime, so a box would run privileged (root on the host if it is escaped); run `si sandbox host init`, or pass --trusted-sandbox for a box you trust")

// Up makes, starts and readies a box from spec: the image (pulled when the
// host does not have it), its own network, the limits, the files copied into
// /lab, then sandbox-setup:. dir is the page's folder (sandbox-files: is
// relative to it). On a failure nothing is left behind.
func Up(ctx context.Context, eng *Engine, spec deck.Sandbox, dir string, o Options) (b *Box, err error) {
	step := o.Step
	if step == nil {
		step = func(string) {}
	}
	info, err := eng.Info(ctx)
	if err != nil {
		return nil, err
	}
	runtime := ""
	if _, ok := info.Runtimes[SysboxRuntime]; ok {
		runtime = SysboxRuntime
	} else if !o.Trusted {
		return nil, ErrNoSysbox
	}
	have, err := eng.HasImage(ctx, spec.Image)
	if err != nil {
		return nil, err
	}
	if !have {
		step("pull " + spec.Image)
		if err := eng.Pull(ctx, spec.Image, nil); err != nil {
			return nil, err
		}
	}
	id := randomID()
	b = &Box{eng: eng, Name: "si-box-" + id, Spec: spec, Runtime: runtime, Ports: map[int]HostPort{}}
	defer func() {
		if err != nil {
			b.remove()
		}
	}()
	labels := map[string]string{"si.sandbox": "1", "si.pid": strconv.Itoa(os.Getpid())}
	if b.Net, err = eng.CreateNetwork(ctx, "si-s-"+id, labels); err != nil {
		return b, err
	}
	step("start " + b.Name)
	exposed, bindings := map[string]any{}, map[string]any{}
	for _, p := range spec.Ports {
		k := strconv.Itoa(p.Port) + "/tcp"
		exposed[k] = map[string]any{}
		bindings[k] = []map[string]string{{"HostIp": "127.0.0.1", "HostPort": ""}}
	}
	host := map[string]any{
		"NetworkMode": b.Net, "Memory": spec.Memory, "MemorySwap": spec.Memory,
		"NanoCpus": int64(spec.CPUs * 1e9), "PidsLimit": spec.Pids, "PortBindings": bindings,
	}
	if runtime != "" {
		host["Runtime"] = runtime
	} else {
		host["Privileged"] = true
	}
	if b.ID, err = eng.CreateContainer(ctx, b.Name, map[string]any{
		"Image": spec.Image, "Hostname": "lab", "Labels": labels, "ExposedPorts": exposed,
		"HostConfig": host, // the image's own env: docker:dind keeps its TLS default (no open tcp://0.0.0.0:2375)
	}); err != nil {
		return b, err
	}
	if err = eng.StartContainer(ctx, b.ID); err != nil {
		return b, err
	}
	step("wait for Docker in the box")
	if err = b.waitDocker(ctx); err != nil {
		return b, err
	}
	if _, _, err = b.execIn(ctx, "/", "mkdir -p "+LabDir, 0); err != nil {
		return b, err
	}
	if spec.Files != "" {
		step("copy " + spec.Files + " into " + LabDir)
		if err = b.copyIn(ctx, filepath.Join(dir, spec.Files)); err != nil {
			return b, err
		}
	}
	if spec.Setup != "" {
		step("sandbox-setup: " + spec.Setup)
		out, code, err := b.Exec(ctx, spec.Setup, 5*time.Minute)
		if err != nil {
			return b, err
		}
		if code != 0 {
			return b, fmt.Errorf("sandbox-setup: %s: exit %d\n%s", spec.Setup, code, tail(out, 2000))
		}
	}
	c, err := eng.Inspect(ctx, b.ID)
	if err != nil {
		return b, err
	}
	for _, p := range spec.Ports {
		for _, hp := range c.NetworkSettings.Ports[strconv.Itoa(p.Port)+"/tcp"] {
			if hp.HostPort != "" {
				b.Ports[p.Port] = HostPort{"127.0.0.1:" + hp.HostPort, p.Label}
				break
			}
		}
	}
	return b, nil
}

// waitDocker waits until `docker info` answers in the box. An image without
// a docker command (exit 127) is ready as soon as it runs.
func (b *Box) waitDocker(ctx context.Context) error {
	deadline := time.Now().Add(readyWait)
	for {
		out, _, err := b.execIn(ctx, "/", "docker info >/dev/null 2>&1; echo $?", 10*time.Second)
		if err != nil {
			return err
		}
		switch strings.TrimSpace(out) {
		case "0", "127":
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("Docker in the box did not answer within %s (is %s a Docker-in-Docker image?)", readyWait, b.Spec.Image)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
}

// Exec runs cmd with sh -c in /lab and returns its output (stdout and stderr,
// up to 64 KB) and exit code; timeout 0 means a minute.
func (b *Box) Exec(ctx context.Context, cmd string, timeout time.Duration) (string, int, error) {
	return b.execIn(ctx, LabDir, cmd, timeout)
}

// execIn runs cmd with sh -c in dir.
func (b *Box) execIn(ctx context.Context, dir, cmd string, timeout time.Duration) (string, int, error) {
	if timeout == 0 {
		timeout = time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	id, err := b.eng.ExecCreate(ctx, b.ID, ExecConfig{Cmd: []string{"sh", "-c", cmd}, WorkingDir: dir})
	if err != nil {
		return "", -1, err
	}
	return b.eng.ExecRun(ctx, id, 64<<10)
}

// Shell is a terminal in the box: what is written is typed, what is read is
// the screen.
type Shell struct {
	io.ReadWriteCloser
	eng    *Engine
	execID string
}

// Resize sets the shell's terminal size.
func (s *Shell) Resize(cols, rows int) {
	if cols > 0 && rows > 0 {
		s.eng.ExecResize(context.Background(), s.execID, cols, rows)
	}
}

// Shell opens a login shell in /lab (bash when the image has it, else sh).
func (b *Box) Shell(cols, rows int) (*Shell, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	id, err := b.eng.ExecCreate(ctx, b.ID, ExecConfig{
		Cmd: []string{"sh", "-c", "if command -v bash >/dev/null 2>&1; then exec bash -l; else exec sh -l; fi"},
		Tty: true, Stdin: true, Env: []string{"TERM=xterm-256color"}, WorkingDir: LabDir,
	})
	if err != nil {
		return nil, err
	}
	rw, err := b.eng.ExecAttach(ctx, id)
	if err != nil {
		return nil, err
	}
	s := &Shell{ReadWriteCloser: rw, eng: b.eng, execID: id}
	s.Resize(cols, rows)
	return s, nil
}

// Down runs sandbox-cleanup: (half a minute at most) and removes the box and
// its network.
func (b *Box) Down() {
	if b.Spec.Cleanup != "" && b.ID != "" {
		b.Exec(context.Background(), b.Spec.Cleanup, 30*time.Second)
	}
	b.remove()
}

func (b *Box) remove() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if b.ID != "" {
		b.eng.RemoveContainer(ctx, b.ID)
	}
	if b.Net != "" {
		b.eng.RemoveNetwork(ctx, b.Net)
	}
}

// copyIn sends the folder src into /lab: a tar stream into `tar -x` in the
// box (the engine's archive upload fails on sysbox's own mounts).
func (b *Box) copyIn(ctx context.Context, src string) error {
	id, err := b.eng.ExecCreate(ctx, b.ID, ExecConfig{Cmd: []string{"tar", "-xf", "-", "-C", LabDir}, Stdin: true, WorkingDir: "/"})
	if err != nil {
		return err
	}
	pr, pw := io.Pipe()
	go func() { pw.CloseWithError(writeTar(pw, src)) }()
	out, code, err := b.eng.ExecFeed(ctx, id, pr, 8<<10)
	if err != nil {
		return err
	}
	if code != 0 {
		return fmt.Errorf("copying %s into the box: tar exit %d: %s", src, code, out)
	}
	return nil
}

// writeTar writes the regular files and folders under src (paths relative
// to it; links are skipped) as a tar stream.
func writeTar(w io.Writer, src string) error {
	tw := tar.NewWriter(w)
	err := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		if rel == "." || !(d.IsDir() || d.Type().IsRegular()) {
			return nil
		}
		fi, err := d.Info()
		if err != nil {
			return err
		}
		h, err := tar.FileInfoHeader(fi, "")
		if err != nil {
			return err
		}
		h.Name, h.Uid, h.Gid, h.Uname, h.Gname = filepath.ToSlash(rel), 0, 0, "", ""
		if d.IsDir() {
			h.Name += "/"
		}
		if err := tw.WriteHeader(h); err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		f, err := os.Open(p)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(tw, f)
		return err
	})
	if err != nil {
		return err
	}
	return tw.Close()
}

// Sweep removes the boxes a si that is no longer running left behind (a
// crash, a killed terminal): every si.sandbox container whose si.pid is gone.
func Sweep(ctx context.Context, eng *Engine) (removed int) {
	list, err := eng.List(ctx, "si.sandbox=1")
	if err != nil {
		return 0
	}
	for _, c := range list {
		pid, _ := strconv.Atoi(c.Labels["si.pid"])
		if pid > 0 && alive(pid) {
			continue
		}
		eng.RemoveContainer(ctx, c.ID)
		removed++
	}
	eng.PruneNetworks(ctx, "si.sandbox=1") // the networks of the boxes gone
	return removed
}

// alive reports whether a process with pid runs on this machine.
func alive(pid int) bool {
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return p.Signal(syscall.Signal(0)) == nil
}

func randomID() string {
	b := make([]byte, 5)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// tail is the last n bytes of s.
func tail(s string, n int) string {
	if len(s) > n {
		return "…" + s[len(s)-n:]
	}
	return s
}
