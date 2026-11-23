package deck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSandboxSpec(t *testing.T) {
	cases := []struct {
		meta    Meta
		ok      bool
		problem string
		check   func(Sandbox) bool
	}{
		{Meta{}, false, "", nil},
		{Meta{SandboxPorts: "80"}, false, "without sandbox:", nil},
		{Meta{Sandbox: "docker:28-dind"}, true, "", func(s Sandbox) bool {
			return s.Memory == DefaultSandboxMemory && s.CPUs == DefaultSandboxCPUs && s.Pids == DefaultSandboxPids
		}},
		{Meta{Sandbox: "a:1", SandboxPorts: "15672 Console, 8080 App", SandboxLimits: "memory=512m cpus=0.5 pids=100"}, true, "", func(s Sandbox) bool {
			return len(s.Ports) == 2 && s.Ports[0] == (SandboxPort{15672, "Console"}) && s.Ports[1].Label == "App" &&
				s.Memory == 512<<20 && s.CPUs == 0.5 && s.Pids == 100
		}},
		{Meta{Sandbox: "a:1", SandboxPorts: "http"}, true, "is not a port", nil},
		{Meta{Sandbox: "a:1", SandboxLimits: "memory=1k"}, true, "64m at least", nil},
		{Meta{Sandbox: "a:1", SandboxLimits: "disk=1g"}, true, "want memory=", nil},
		{Meta{Sandbox: "two words"}, true, "not an image", nil},
	}
	for _, c := range cases {
		s, ok, problems := c.meta.SandboxSpec()
		if ok != c.ok {
			t.Errorf("%+v: ok %v", c.meta, ok)
		}
		got := strings.Join(problems, "; ")
		if (c.problem == "") != (got == "") || !strings.Contains(got, c.problem) {
			t.Errorf("%+v: problems %q, want %q", c.meta, got, c.problem)
		}
		if c.check != nil && !c.check(s) {
			t.Errorf("%+v: spec %+v", c.meta, s)
		}
	}
}

func TestSandboxCheck(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "lab.md")
	os.WriteFile(p, []byte("---\ntype: lab\ntitle: L\nsandbox: docker:dind\nsandbox-files: env\n---\n# L\n"), 0o644)
	d, err := Parse(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Problems) != 1 || !strings.Contains(d.Problems[0].Msg, "sandbox-files: env is not a folder") {
		t.Errorf("problems %+v", d.Problems)
	}
	warned := false
	for _, w := range d.Warnings {
		warned = warned || strings.Contains(w.Msg, "is not pinned")
	}
	if warned {
		t.Error("docker:dind has a tag: no pin warning")
	}
	os.Mkdir(filepath.Join(dir, "env"), 0o755)
	os.WriteFile(p, []byte("---\ntype: lab\ntitle: L\nsandbox: acme/lab\nsandbox-files: env\n---\n# L\n"), 0o644)
	d, _ = Parse(p)
	warned = false
	for _, w := range d.Warnings {
		warned = warned || strings.Contains(w.Msg, "is not pinned")
	}
	if len(d.Problems) != 0 || !warned {
		t.Errorf("problems %+v, pin warning %v", d.Problems, warned)
	}
}

func TestSplitImageRef(t *testing.T) {
	for ref, want := range map[string][2]string{
		"docker:28-dind": {"docker", "28-dind"}, "a/b": {"a/b", ""}, "host:5000/a": {"host:5000/a", ""},
		"host:5000/a:1": {"host:5000/a", "1"}, "a@sha256:ff": {"a", "sha256:ff"},
	} {
		if n, tg := SplitImageRef(ref); n != want[0] || tg != want[1] {
			t.Errorf("%s: %s %s", ref, n, tg)
		}
	}
}
