package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/serve"
)

// runCA is `si ca [trust|untrust]`: where si's certificate authority for
// `si play --remote --https` lives and its fingerprint; trust adds it to this
// computer's trusted certificates (so the laptop's browser shows the remote
// pages as secure too), untrust takes it out.
func runCA(rest []string, stdout, stderr io.Writer) int {
	dir, err := serve.CADir()
	if err != nil {
		return fail(stderr, err)
	}
	ca, err := serve.LoadCA(dir)
	if err != nil {
		return fail(stderr, err)
	}
	pemPath := filepath.Join(ca.Dir, "si-local-ca.pem")
	sub := ""
	if len(rest) > 0 {
		sub = rest[0]
	}
	switch sub {
	case "":
		if ca.Created {
			fmt.Fprintln(stdout, "made si's certificate authority (only for local addresses; its key stays on this machine)")
		}
		fmt.Fprintf(stdout, "si's certificate authority (si play --remote --https)\n  %s\n  %s\n  SHA-256 %s\n", ca.Cert.Subject.CommonName, pemPath, ca.Fingerprint())
		fmt.Fprintln(stdout, "  on a phone: scan the \"first time\" code of si play --remote --https, install it, then turn on full trust")
		fmt.Fprintln(stdout, "  (iPhone: Settings › General › About › Certificate Trust Settings)")
		fmt.Fprintln(stdout, "  on this computer: si ca trust (si ca untrust takes it out)")
		return exitOK
	case "trust", "untrust":
		args, manual := caTrustCommand(runtime.GOOS, sub == "trust", pemPath, ca.Cert.Subject.CommonName)
		if args == nil {
			fmt.Fprintln(stdout, manual)
			return exitOK
		}
		fmt.Fprintf(stdout, "si ca %s: %s\n", sub, shellQuote(args))
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, stdout, stderr
		if err := cmd.Run(); err != nil {
			return fail(stderr, err)
		}
		if sub == "trust" {
			fmt.Fprintln(stdout, "si ca: trusted on this computer (Safari and Chrome; Firefox keeps its own list: Settings › Certificates › Import)")
		} else {
			fmt.Fprintln(stdout, "si ca: no longer trusted on this computer")
		}
		return exitOK
	}
	return qrUsage(stderr, fmt.Sprintf("ca %q: use si ca, si ca trust or si ca untrust", sub))
}

// caTrustCommand is the command that (un)trusts the authority on this OS, or
// nil and what to do by hand where si does not change the system store.
func caTrustCommand(goos string, trust bool, pemPath, name string) ([]string, string) {
	switch goos {
	case "darwin":
		home, _ := os.UserHomeDir()
		kc := filepath.Join(home, "Library", "Keychains", "login.keychain-db")
		if trust { // the login keychain: macOS asks for your password, no sudo
			return []string{"security", "add-trusted-cert", "-r", "trustRoot", "-p", "ssl", "-k", kc, pemPath}, ""
		}
		return []string{"security", "delete-certificate", "-c", name, kc}, ""
	case "linux":
		if trust {
			return nil, "si ca trust: on Linux, copy it into the system store (needs root):\n  sudo cp " + pemPath + " /usr/local/share/ca-certificates/si-local-ca.crt && sudo update-ca-certificates\n  (Chrome and Firefox: their certificate settings › Authorities › Import)"
		}
		return nil, "si ca untrust: sudo rm /usr/local/share/ca-certificates/si-local-ca.crt && sudo update-ca-certificates --fresh"
	case "windows":
		if trust {
			return []string{"certutil", "-user", "-addstore", "Root", pemPath}, ""
		}
		return []string{"certutil", "-user", "-delstore", "Root", name}, ""
	}
	return nil, "si ca: import " + pemPath + " into this system's trusted root certificates by hand"
}

func shellQuote(args []string) string {
	s := ""
	for i, a := range args {
		if i > 0 {
			s += " "
		}
		if a == "" || strings.ContainsAny(a, " '\"$`\\") {
			a = "'" + a + "'"
		}
		s += a
	}
	return s
}
