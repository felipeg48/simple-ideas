package webdemo

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"
)

// stopGrace is how long a stopped program has between SIGTERM and SIGKILL.
var stopGrace = 5 * time.Second

// proc is a program running in the background: a start: of the front matter
// or a start step. It runs in its own process group, so stopping it stops
// everything it started.
type proc struct {
	name string
	cmd  *exec.Cmd
	out  *output
	logf *os.File
	done chan struct{} // closed when it has exited
	code int           // its exit status once done (-1: killed by a signal)
	once sync.Once
}

// shellCmd is `sh -c cmd` (`cmd /C` on Windows) in dir.
func shellCmd(cmd, dir string) *exec.Cmd {
	c := exec.Command("sh", "-c", cmd)
	if runtime.GOOS == "windows" {
		c = exec.Command("cmd", "/C", cmd)
	}
	c.Dir = dir
	return c
}

// startProc starts cmd in dir; its output goes to log (when not "") and is
// kept for expect output of. Secrets are masked in both.
func startProc(name, cmd, dir, log string, mask func(string) string) (*proc, error) {
	p := &proc{name: name, cmd: shellCmd(cmd, dir), done: make(chan struct{}), out: newOutput(mask, nil)}
	if log != "" {
		f, err := os.Create(log)
		if err != nil {
			return nil, err
		}
		p.logf, p.out.log = f, f
	}
	p.cmd.Stdout, p.cmd.Stderr = p.out, p.out
	ownGroup(p.cmd)
	if err := p.cmd.Start(); err != nil {
		if p.logf != nil {
			p.logf.Close()
		}
		return nil, err
	}
	go func() {
		err := p.cmd.Wait()
		p.code = 0
		if err != nil {
			p.code = -1
			if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() >= 0 {
				p.code = ee.ExitCode()
			}
		}
		p.out.close()
		if p.logf != nil {
			p.logf.Close()
		}
		close(p.done)
	}()
	return p, nil
}

func (p *proc) exited() bool {
	select {
	case <-p.done:
		return true
	default:
		return false
	}
}

// stop asks the program's group to stop (SIGTERM), then kills what is left
// after stopGrace. Safe to call more than once and after it exited.
func (p *proc) stop() {
	p.once.Do(func() {
		pid := p.cmd.Process.Pid
		if !p.exited() {
			signalGroup(pid, false)
			select {
			case <-p.done:
			case <-time.After(stopGrace):
			}
		}
		killGroup(pid) // what it started and left behind
		<-p.done
	})
}

// ready waits until the program shows r (a text, a port that accepts, a
// delay), up to within; it fails when the program exits first.
func (p *proc) ready(r Ready, within time.Duration) error {
	if within <= 0 {
		within = 30 * time.Second
	}
	end := time.Now().Add(within)
	if r.Delay > 0 {
		end = time.Now().Add(r.Delay)
	}
	for {
		switch {
		case r.Text != "" && shows(p.out.text(0), r.Text):
			return nil
		case r.Port > 0 && portOpen(r.Port):
			return nil
		}
		if p.exited() {
			return &failure{expected: p.name + " " + r.String(), found: fmt.Sprintf("it exited with status %d first", p.code), output: p.out.tail(40)}
		}
		if time.Now().After(end) {
			if r.Delay > 0 || (r.Text == "" && r.Port == 0) {
				return nil
			}
			return &failure{expected: p.name + " " + r.String(), found: fmt.Sprintf("not within %s", within), output: p.out.tail(40)}
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// portOpen reports whether something accepts on the loopback port.
func portOpen(port int) bool {
	c, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), 300*time.Millisecond)
	if err != nil {
		return false
	}
	c.Close()
	return true
}

// logPath is where a program's log goes: <name>.log beside the report.
func logPath(out, name string) string {
	if out == "" {
		return ""
	}
	return filepath.Join(out, name+".log")
}
