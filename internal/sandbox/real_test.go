package sandbox

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

// TestRealBox runs a box on the real sandbox host (SI_DOCKER=1; the
// si-sandbox Colima VM, or $SI_SANDBOX_HOST): Docker inside it, unprivileged,
// a nested container whose port the Mac reaches, a shell, the files folder.
func TestRealBox(t *testing.T) {
	if os.Getenv("SI_DOCKER") != "1" {
		t.Skip("SI_DOCKER=1 runs a real box")
	}
	sock, err := Socket("")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	os.MkdirAll(dir+"/env/conf", 0o755)
	os.WriteFile(dir+"/env/conf/hello.txt", []byte("hello from the lab\n"), 0o644)
	spec := deck.Sandbox{Image: "docker:28-dind", Files: "env", Memory: 2 << 30, CPUs: 2, Pids: 2048,
		Setup:   "docker run -d --name web -p 8080:80 nginx:alpine",
		Cleanup: "docker rm -f web",
		Ports:   []deck.SandboxPort{{Port: 8080, Label: "Web"}}}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	eng := NewEngine(sock)
	b, err := Up(ctx, eng, spec, dir, Options{Step: func(s string) { t.Log(s) }})
	if err != nil {
		t.Fatal(err)
	}
	defer b.Down()
	if b.Runtime != SysboxRuntime {
		t.Errorf("runtime %q, want sysbox", b.Runtime)
	}
	out, code, err := b.Exec(ctx, "cat conf/hello.txt && id -u && docker ps --format '{{.Names}}'", 0)
	if err != nil || code != 0 || !strings.Contains(out, "hello from the lab") || !strings.Contains(out, "web") {
		t.Fatalf("exec: %v %d %q", err, code, out)
	}
	hp := b.Ports[8080].Addr
	if hp == "" {
		t.Fatalf("port 8080 not published: %+v", b.Ports)
	}
	var body string
	for i := 0; i < 40; i++ { // nginx starts, Colima starts forwarding
		if r, err := http.Get("http://" + hp + "/"); err == nil {
			bb, _ := io.ReadAll(r.Body)
			r.Body.Close()
			if body = string(bb); strings.Contains(body, "nginx") {
				break
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	if !strings.Contains(body, "nginx") {
		t.Fatalf("the Mac does not reach the box's nginx at %s: %q", hp, body)
	}
	sh, err := b.Shell(100, 30)
	if err != nil {
		t.Fatal(err)
	}
	defer sh.Close()
	io.WriteString(sh, "echo sh-$((40+2)); exit\n")
	sc := bufio.NewScanner(sh)
	found := false
	for sc.Scan() {
		if strings.Contains(sc.Text(), "sh-42") {
			found = true
			break
		}
	}
	if !found {
		t.Error("the shell did not answer")
	}
	// the home folder of the Mac is not in the box, nor in the VM
	h, _ := os.UserHomeDir()
	if out, _, _ := b.Exec(ctx, "ls -A "+h+" 2>&1; true", 0); !strings.Contains(out, "No such file") {
		t.Errorf("the box sees the Mac's home: %q", out)
	}
}
