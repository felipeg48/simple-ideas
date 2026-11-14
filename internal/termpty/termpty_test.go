package termpty

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"
)

// the test binary is its own child: it reads a line and answers it
func TestMain(m *testing.M) {
	if os.Getenv("TERMPTY_CHILD") == "1" {
		line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		fmt.Printf("got <%s>\n", strings.TrimSpace(line))
		os.Exit(3)
	}
	os.Exit(m.Run())
}

// TestTerminalPTY: a program under the pseudo-terminal reads what is typed
// (Enter is \r, as a terminal sends it), its output comes back, a resize is
// taken, and Wait gives its exit status (on Windows too: ConPTY).
func TestTerminalPTY(t *testing.T) {
	c := exec.Command(os.Args[0], "-test.run=^$")
	c.Env = append(os.Environ(), "TERMPTY_CHILD=1")
	p, err := Start(c, 80, 24)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	var mu sync.Mutex
	var out strings.Builder
	go func() {
		b := make([]byte, 4096)
		for {
			n, err := p.Read(b)
			mu.Lock()
			out.Write(b[:n])
			mu.Unlock()
			if err != nil {
				return
			}
		}
	}()
	Resize(p, 100, 30)
	time.Sleep(300 * time.Millisecond) // the child is reading
	if _, err := p.Write([]byte("hello\r")); err != nil {
		t.Fatal(err)
	}
	got := ""
	for i := 0; i < 200 && !strings.Contains(got, "got <hello>"); i++ {
		time.Sleep(50 * time.Millisecond)
		mu.Lock()
		got = out.String()
		mu.Unlock()
	}
	if !strings.Contains(got, "got <hello>") {
		t.Fatalf("output %q", got)
	}
	var ee *exec.ExitError
	if err := c.Wait(); !errors.As(err, &ee) || ee.ExitCode() != 3 {
		t.Errorf("wait: %v, want exit status 3", err)
	}
	if err := p.Close(); err != nil {
		t.Errorf("close: %v", err)
	}
}
