package main

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// tailscaleCert is --https=tailscale: this machine's name on the tailnet
// (machine.tailnet.ts.net) and a real certificate for it from `tailscale
// cert` (Let's Encrypt, through Tailscale). The phone needs the Tailscale app
// and nothing installed; the tailnet needs HTTPS certificates turned on.
func tailscaleCert() (string, tls.Certificate, error) {
	bin := tailscaleBin()
	if bin == "" {
		return "", tls.Certificate{}, errors.New("no tailscale command found (install Tailscale, or use --https for si's own certificate)")
	}
	out, err := exec.Command(bin, "status", "--json").Output()
	if err != nil {
		why := err.Error()
		if ee, ok := err.(*exec.ExitError); ok && len(ee.Stderr) > 0 {
			why = strings.TrimSpace(string(ee.Stderr))
		}
		return "", tls.Certificate{}, fmt.Errorf("tailscale status: %s (or use --https for si's own certificate)", why)
	}
	name, err := tailscaleName(out)
	if err != nil {
		return "", tls.Certificate{}, err
	}
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", tls.Certificate{}, err
	}
	dir = filepath.Join(dir, "si", "tailscale")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", tls.Certificate{}, err
	}
	certFile, keyFile := filepath.Join(dir, name+".crt"), filepath.Join(dir, name+".key")
	// tailscale cert renews only when needed, so running it every time is cheap
	if b, err := exec.Command(bin, "cert", "--cert-file", certFile, "--key-file", keyFile, name).CombinedOutput(); err != nil {
		msg := strings.TrimSpace(string(b))
		if strings.Contains(strings.ToLower(msg), "https") {
			msg += " (turn on HTTPS certificates in the Tailscale admin console: DNS › HTTPS Certificates)"
		}
		return "", tls.Certificate{}, fmt.Errorf("tailscale cert %s: %s", name, msg)
	}
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	return name, cert, err
}

// tailscaleName reads this machine's MagicDNS name from `tailscale status --json`.
func tailscaleName(statusJSON []byte) (string, error) {
	var st struct {
		BackendState string
		Self         struct{ DNSName string }
	}
	if err := json.Unmarshal(statusJSON, &st); err != nil {
		return "", fmt.Errorf("tailscale status: %v", err)
	}
	if st.BackendState != "" && st.BackendState != "Running" {
		return "", fmt.Errorf("Tailscale is %s, not running", st.BackendState)
	}
	name := strings.TrimSuffix(st.Self.DNSName, ".")
	if !strings.HasSuffix(name, ".ts.net") {
		return "", fmt.Errorf("this machine has no ts.net name (%q): turn on MagicDNS in the Tailscale admin console", name)
	}
	return name, nil
}

func tailscaleBin() string {
	if p, err := exec.LookPath("tailscale"); err == nil {
		return p
	}
	for _, p := range []string{"/Applications/Tailscale.app/Contents/MacOS/Tailscale", "/usr/local/bin/tailscale", "/opt/homebrew/bin/tailscale"} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}
