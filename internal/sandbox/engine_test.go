package sandbox

import (
	"bufio"
	"context"
	"encoding/binary"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// fakeEngine serves a few Engine API calls on a unix socket; exec starts
// with a terminal are hijacked and echo what they are sent.
func fakeEngine(t *testing.T) *Engine {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", "sbx") // unix socket paths are short on macOS
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	sock := dir + "/d.sock"
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1.41/info", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"ServerVersion":"29.0","NCPU":4,"MemTotal":8589934592,"Runtimes":{"runc":{},"sysbox-runc":{}}}`)
	})
	mux.HandleFunc("GET /v1.41/images/{ref}/json", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("ref") != "have:1" {
			w.WriteHeader(404)
			io.WriteString(w, `{"message":"No such image"}`)
			return
		}
		io.WriteString(w, `{}`)
	})
	mux.HandleFunc("POST /v1.41/exec/{id}/start", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("id") == "tty" {
			io.ReadAll(r.Body)
			conn, buf, _ := w.(http.Hijacker).Hijack()
			buf.WriteString("HTTP/1.1 101 UPGRADED\r\nContent-Type: application/vnd.docker.raw-stream\r\nConnection: Upgrade\r\nUpgrade: tcp\r\n\r\nwelcome\n")
			buf.Flush()
			io.Copy(conn, buf) // echo
			conn.Close()
			return
		}
		frame := func(stream byte, s string) {
			h := make([]byte, 8)
			h[0] = stream
			binary.BigEndian.PutUint32(h[4:], uint32(len(s)))
			w.Write(h)
			io.WriteString(w, s)
		}
		frame(1, "out ")
		frame(2, "err")
	})
	mux.HandleFunc("GET /v1.41/exec/{id}/json", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"Running":false,"ExitCode":3}`)
	})
	srv := httptest.NewUnstartedServer(mux)
	srv.Listener = ln
	srv.Start()
	t.Cleanup(srv.Close)
	return NewEngine(sock)
}

func TestEngineCalls(t *testing.T) {
	e := fakeEngine(t)
	ctx := context.Background()
	info, err := e.Info(ctx)
	if err != nil || info.NCPU != 4 {
		t.Fatalf("info %+v %v", info, err)
	}
	if _, ok := info.Runtimes[SysboxRuntime]; !ok {
		t.Error("sysbox runtime not read")
	}
	if ok, err := e.HasImage(ctx, "have:1"); !ok || err != nil {
		t.Errorf("have:1: %v %v", ok, err)
	}
	if ok, err := e.HasImage(ctx, "nope:1"); ok || err != nil {
		t.Errorf("nope:1: %v %v (a 404 is no image, not an error)", ok, err)
	}
	out, code, err := e.ExecRun(ctx, "plain", 1<<10)
	if err != nil || out != "out err" || code != 3 {
		t.Errorf("exec: %q %d %v", out, code, err)
	}
	out, _, _ = e.ExecRun(ctx, "plain", 5)
	if out != "out e" {
		t.Errorf("capped output %q", out)
	}
}

func TestEngineAttach(t *testing.T) {
	e := fakeEngine(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rw, err := e.ExecAttach(ctx, "tty")
	if err != nil {
		t.Fatal(err)
	}
	defer rw.Close()
	br := bufio.NewReader(rw)
	if l, _ := br.ReadString('\n'); l != "welcome\n" { // bytes that came with the 101
		t.Errorf("first line %q", l)
	}
	io.WriteString(rw, "ls\n")
	if l, _ := br.ReadString('\n'); l != "ls\n" {
		t.Errorf("echo %q", l)
	}
}

func TestEngineError(t *testing.T) {
	e := NewEngine("/tmp/no-such-si-socket")
	if _, err := e.Info(context.Background()); err == nil || !strings.Contains(err.Error(), "no-such-si-socket") {
		t.Errorf("err %v: should name the socket", err)
	}
}

func TestSocketChoice(t *testing.T) {
	t.Setenv("SI_SANDBOX_HOST", "")
	t.Setenv("DOCKER_HOST", "unix:///x/docker.sock")
	if s, _ := Socket("docker"); s != "/x/docker.sock" {
		t.Errorf("docker: %s", s)
	}
	if s, _ := Socket("unix:///y.sock"); s != "/y.sock" {
		t.Errorf("path: %s", s)
	}
	t.Setenv("COLIMA_HOME", t.TempDir())
	if _, err := Socket(""); err == nil || !strings.Contains(err.Error(), "si sandbox host init") {
		t.Errorf("no colima VM: %v", err)
	}
	t.Setenv("SI_SANDBOX_HOST", "/z.sock")
	if s, _ := Socket("colima"); s != "/z.sock" {
		t.Errorf("env wins: %s", s)
	}
}

func TestWriteTar(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(dir+"/a/b", 0o755)
	os.WriteFile(dir+"/a/b/f.txt", []byte("x"), 0o644)
	os.Symlink("/etc/passwd", dir+"/link")
	pr, pw := io.Pipe()
	go func() { pw.CloseWithError(writeTar(pw, dir)) }()
	b, _ := io.ReadAll(pr)
	s := string(b)
	if !strings.Contains(s, "a/b/f.txt") || strings.Contains(s, "link") {
		t.Errorf("tar has the file, not the link")
	}
}
