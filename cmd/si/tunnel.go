package main

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/felipeg48/simple-ideas/internal/tools"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"time"

	"github.com/felipeg48/simple-ideas/internal/serve"
)

// trycloudflareRE is the address a Cloudflare quick tunnel prints.
var trycloudflareRE = regexp.MustCompile(`https://[a-z0-9-]+\.trycloudflare\.com`)

// startTunnel is si play --tunnel: the audience side (only) on a temporary
// public HTTPS address, for a remote audience (a video call). It serves
// serve.AudienceOnlyHandler on a loopback port and asks Cloudflare's free
// quick tunnel (cloudflared, if installed) to reach it; the address is gone
// when si play stops. It returns the public base URL and a stop func.
func startTunnel(srv *serve.Server, stderr io.Writer) (string, func(), error) {
	bin := tools.Find("cloudflared")
	if bin == "" {
		return "", nil, errors.New(tools.Need("cloudflared", "--tunnel"))
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", nil, err
	}
	hs := &http.Server{Handler: srv.AudienceOnlyHandler(), ReadHeaderTimeout: 10 * time.Second}
	go hs.Serve(ln)
	cmd := exec.Command(bin, "tunnel", "--no-autoupdate", "--url", "http://"+ln.Addr().String())
	out, err := cmd.StderrPipe()
	if err != nil {
		ln.Close()
		return "", nil, err
	}
	if err := cmd.Start(); err != nil {
		ln.Close()
		return "", nil, err
	}
	stop := func() {
		cmd.Process.Signal(os.Interrupt)
		done := make(chan struct{})
		go func() { cmd.Wait(); close(done) }()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			cmd.Process.Kill()
		}
		hs.Close()
	}
	found := make(chan string, 1)
	go func() {
		sc := bufio.NewScanner(out)
		sent := false
		for sc.Scan() { // keep reading so cloudflared never blocks on a full pipe
			if u := trycloudflareRE.FindString(sc.Text()); u != "" && !sent {
				found <- u
				sent = true
			}
		}
	}()
	select {
	case u := <-found:
		return u, stop, nil
	case <-time.After(40 * time.Second):
		stop()
		return "", nil, fmt.Errorf("cloudflared gave no address in 40 s (is this machine online?)")
	}
}

func fileExists(p string) bool { fi, err := os.Stat(p); return err == nil && !fi.IsDir() }
