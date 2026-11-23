package deck

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Sandbox is a lab's own Docker box, from the sandbox: keys of its front
// matter (RULES §37; run by internal/sandbox in si play).
type Sandbox struct {
	Image   string        // sandbox: the image, name:tag or name@sha256:…
	Files   string        // sandbox-files: a folder beside the page, copied into /lab
	Setup   string        // sandbox-setup: a command run in /lab once the box is up
	Cleanup string        // sandbox-cleanup: a command run before the box is removed
	Ports   []SandboxPort // sandbox-ports: 15672 Console, 8080 App
	Memory  int64         // sandbox-limits: memory=2g (bytes)
	CPUs    float64       // sandbox-limits: cpus=2
	Pids    int64         // sandbox-limits: pids=2048
}

// SandboxPort is one port of the box opened to the learner.
type SandboxPort struct {
	Port  int
	Label string
}

// SandboxKeys are the front matter keys of a sandbox.
var SandboxKeys = []string{"sandbox", "sandbox-files", "sandbox-setup", "sandbox-cleanup", "sandbox-ports", "sandbox-limits"}

// Sandbox limits when sandbox-limits: leaves one out.
const (
	DefaultSandboxMemory = 2 << 30
	DefaultSandboxCPUs   = 2
	DefaultSandboxPids   = 2048
)

// SandboxSpec reads the page's sandbox: keys; ok is false when the page has
// none. Problems name a bad value; the spec keeps the defaults for it.
func (m Meta) SandboxSpec() (s Sandbox, ok bool, problems []string) {
	s = Sandbox{Image: strings.TrimSpace(m.Sandbox), Files: strings.TrimSpace(m.SandboxFiles),
		Setup: strings.TrimSpace(m.SandboxSetup), Cleanup: strings.TrimSpace(m.SandboxCleanup),
		Memory: DefaultSandboxMemory, CPUs: DefaultSandboxCPUs, Pids: DefaultSandboxPids}
	if s.Image == "" {
		if m.SandboxFiles+m.SandboxSetup+m.SandboxCleanup+m.SandboxPorts+m.SandboxLimits != "" {
			problems = append(problems, "sandbox-…: keys without sandbox: (the image)")
		}
		return s, false, problems
	}
	if strings.ContainsAny(s.Image, " \t") {
		problems = append(problems, fmt.Sprintf("sandbox: %q is not an image name", s.Image))
	}
	var err error
	if s.Ports, err = ParseSandboxPorts(m.SandboxPorts); err != nil {
		problems = append(problems, err.Error())
	}
	if err := s.parseLimits(m.SandboxLimits); err != nil {
		problems = append(problems, err.Error())
	}
	return s, true, problems
}

// ParseSandboxPorts reads "15672 Console, 8080 App": a port, then an optional
// label, comma separated.
func ParseSandboxPorts(v string) ([]SandboxPort, error) {
	var out []SandboxPort
	for _, part := range strings.Split(v, ",") {
		f := strings.Fields(part)
		if len(f) == 0 {
			continue
		}
		n, err := strconv.Atoi(f[0])
		if err != nil || n < 1 || n > 65535 {
			return out, fmt.Errorf("sandbox-ports: %q is not a port (want 15672 Console, 8080 App)", f[0])
		}
		out = append(out, SandboxPort{n, strings.Join(f[1:], " ")})
	}
	return out, nil
}

// parseLimits reads "memory=2g cpus=2 pids=2048".
func (s *Sandbox) parseLimits(v string) error {
	for _, f := range strings.Fields(strings.ReplaceAll(v, ",", " ")) {
		k, val, _ := strings.Cut(f, "=")
		switch k {
		case "memory":
			b, err := ParseBytes(val)
			if err != nil || b < 64<<20 {
				return fmt.Errorf("sandbox-limits: memory=%s: want a size like 2g or 512m (64m at least)", val)
			}
			s.Memory = b
		case "cpus":
			c, err := strconv.ParseFloat(val, 64)
			if err != nil || c <= 0 {
				return fmt.Errorf("sandbox-limits: cpus=%s: want a number like 2 or 0.5", val)
			}
			s.CPUs = c
		case "pids":
			p, err := strconv.ParseInt(val, 10, 64)
			if err != nil || p < 64 {
				return fmt.Errorf("sandbox-limits: pids=%s: want a number, 64 at least", val)
			}
			s.Pids = p
		default:
			return fmt.Errorf("sandbox-limits: %q: want memory=, cpus= or pids=", f)
		}
	}
	return nil
}

// ParseBytes reads a size: 2g, 512m, 1.5G, 1024k, or plain bytes.
func ParseBytes(v string) (int64, error) {
	v = strings.ToLower(strings.TrimSpace(v))
	mult := int64(1)
	if n := len(v); n > 0 {
		switch v[n-1] {
		case 'k':
			mult = 1 << 10
		case 'm':
			mult = 1 << 20
		case 'g':
			mult = 1 << 30
		case 't':
			mult = 1 << 40
		}
		if mult > 1 {
			v = v[:n-1]
		}
	}
	f, err := strconv.ParseFloat(strings.TrimSuffix(v, "b"), 64)
	if err != nil || f < 0 {
		return 0, fmt.Errorf("%q is not a size", v)
	}
	return int64(f * float64(mult)), nil
}

// sandboxMeta checks the sandbox: keys (si check): values, the files folder,
// and a warning for an image that is not pinned to a tag or digest.
func (d *Deck) sandboxMeta(file string, keyLine map[string]int) {
	s, ok, problems := d.Meta.SandboxSpec()
	line := keyLine["sandbox"]
	for _, k := range SandboxKeys {
		if line == 0 {
			line = keyLine[k]
		}
	}
	for _, p := range problems {
		d.problem(file, line, "%s", p)
	}
	if !ok {
		return
	}
	if s.Files != "" {
		if st, err := os.Stat(filepath.Join(filepath.Dir(file), s.Files)); err != nil || !st.IsDir() {
			d.problem(file, keyLine["sandbox-files"], "sandbox-files: %s is not a folder beside the page", s.Files)
		}
	}
	if _, tag := SplitImageRef(s.Image); tag == "" || tag == "latest" {
		d.Warnings = append(d.Warnings, Problem{File: rel(file), Line: keyLine["sandbox"], Msg: fmt.Sprintf("sandbox: %s is not pinned: name a tag or a digest (name:1.4, name@sha256:…) so every learner gets the same box", s.Image)})
	}
}

// SplitImageRef splits an image reference into its name and tag or digest;
// a registry port (host:5000/a) is not a tag.
func SplitImageRef(ref string) (name, tag string) {
	if i := strings.Index(ref, "@"); i >= 0 {
		return ref[:i], ref[i+1:]
	}
	if i := strings.LastIndex(ref, ":"); i > strings.LastIndex(ref, "/") {
		return ref[:i], ref[i+1:]
	}
	return ref, ""
}
