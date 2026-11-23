package sandbox

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/felipeg48/simple-ideas/internal/home"
)

// Profile is the Colima profile si runs lab boxes in, apart from the user's
// own Docker.
const Profile = "si-sandbox"

// The pinned sysbox the sandbox host installs (github.com/nestybox/sysbox).
const (
	sysboxVersion = "0.7.1"
	sysboxAMD64   = "9d6d5484f980d0a17f86c492c1262015c2afb66280bdb97215b79fde6a0261c5"
	sysboxARM64   = "04ca894ae0b53f0fa54eaacc173ce40363c9a95ea5450f773716a84ef650a69b"
)

// ColimaSocket is the Docker socket of the si-sandbox profile.
func ColimaSocket() string {
	dir := os.Getenv("COLIMA_HOME")
	if dir == "" {
		h, _ := os.UserHomeDir()
		dir = filepath.Join(h, ".colima")
	}
	return filepath.Join(dir, Profile, "docker.sock")
}

// Socket resolves where boxes run (si.conf sandbox.host, $SI_SANDBOX_HOST
// first): colima (the default) is the si-sandbox profile; docker is the
// user's own Docker ($DOCKER_HOST, else /var/run/docker.sock); a path or
// unix:// address is that socket.
func Socket(choice string) (string, error) {
	if v := os.Getenv("SI_SANDBOX_HOST"); v != "" {
		choice = v
	}
	switch {
	case choice == "" || choice == "colima":
		s := ColimaSocket()
		if _, err := os.Stat(s); err != nil {
			return "", errors.New("the sandbox host is not running: `si sandbox host init` once, then `si sandbox host start`")
		}
		return s, nil
	case choice == "docker":
		if h := os.Getenv("DOCKER_HOST"); strings.HasPrefix(h, "unix://") {
			return strings.TrimPrefix(h, "unix://"), nil
		}
		return "/var/run/docker.sock", nil
	default:
		return strings.TrimPrefix(choice, "unix://"), nil
	}
}

// HostSize is the sandbox VM's size.
type HostSize struct {
	CPU, MemoryGiB, DiskGiB int
}

// DefaultHostSize is half the cores (two at least), three quarters of the
// memory and 100 GiB of disk: on a 128 GB Mac, a 96 GiB VM.
func DefaultHostSize() HostSize {
	mem := 8
	if b := physicalMemory(); b > 0 {
		mem = int(b * 3 / 4 >> 30)
	}
	return HostSize{CPU: max(2, runtime.NumCPU()/2), MemoryGiB: max(4, mem), DiskGiB: 100}
}

// physicalMemory is the machine's memory in bytes (0 when unknown).
func physicalMemory() int64 {
	if runtime.GOOS == "darwin" {
		out, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
		if err == nil {
			n, _ := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
			return n
		}
		return 0
	}
	b, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0
	}
	for _, l := range strings.Split(string(b), "\n") {
		if f := strings.Fields(l); len(f) >= 2 && f[0] == "MemTotal:" {
			kb, _ := strconv.ParseInt(f[1], 10, 64)
			return kb << 10
		}
	}
	return 0
}

// shareDir is the one folder of the Mac the sandbox VM sees (empty).
func shareDir() string { return filepath.Join(home.Dir(), "sandbox", "share") }

// colima runs colima with args, its output to out.
func colima(out io.Writer, args ...string) error {
	if _, err := exec.LookPath("colima"); err != nil {
		return errors.New("colima is not installed: brew install colima (https://github.com/abiosoft/colima)")
	}
	c := exec.Command("colima", args...)
	c.Stdout, c.Stderr = out, out
	return c.Run()
}

// vmShell runs a shell script in the sandbox VM (as root when root) and
// returns its output.
func vmShell(root bool, script string) (string, error) {
	args := []string{"ssh", "-p", Profile, "--"}
	if root {
		args = append(args, "sudo")
	}
	args = append(args, "sh", "-c", script)
	out, err := exec.Command("colima", args...).CombinedOutput()
	return string(out), err
}

// HostInit makes the si-sandbox VM: Docker, only the empty share folder
// mounted (never the home folder), and sysbox so boxes run unprivileged. The
// user's own docker context stays as it is (--activate=false).
func HostInit(sz HostSize, out io.Writer) error {
	if err := os.MkdirAll(shareDir(), 0o700); err != nil {
		return err
	}
	fmt.Fprintf(out, "si sandbox: starting the %s VM (%d CPU, %d GiB memory, %d GiB disk); the first start downloads its image\n", Profile, sz.CPU, sz.MemoryGiB, sz.DiskGiB)
	if err := colima(out, "start", Profile, "--activate=false", "--runtime", "docker",
		"--cpu", strconv.Itoa(sz.CPU), "--memory", strconv.Itoa(sz.MemoryGiB), "--disk", strconv.Itoa(sz.DiskGiB),
		"--mount", shareDir()+":w"); err != nil {
		return fmt.Errorf("colima start: %w", err)
	}
	if err := checkMounts(); err != nil {
		return err
	}
	fmt.Fprintf(out, "si sandbox: installing sysbox %s in the VM\n", sysboxVersion)
	if o, err := vmShell(true, sysboxScript); err != nil {
		return fmt.Errorf("installing sysbox: %v\n%s", err, tail(o, 3000))
	}
	return HostStatus(out)
}

// sysboxScript installs the pinned sysbox in the VM (once): the package for
// the VM's architecture, its SHA-256 checked, Docker restarted by it.
var sysboxScript = `set -eu
if command -v sysbox-runc >/dev/null 2>&1; then echo "sysbox is installed"; exit 0; fi
arch=$(dpkg --print-architecture)
case "$arch" in amd64) sum=` + sysboxAMD64 + `;; arm64) sum=` + sysboxARM64 + `;; *) echo "sysbox has no package for $arch"; exit 1;; esac
f=/tmp/sysbox-ce_` + sysboxVersion + `.linux_$arch.deb
curl -fsSL -o "$f" "https://github.com/nestybox/sysbox/releases/download/v` + sysboxVersion + `/sysbox-ce_` + sysboxVersion + `.linux_$arch.deb"
echo "$sum  $f" | sha256sum -c -
ids=$(docker ps -aq); if [ -n "$ids" ]; then docker rm -f $ids >/dev/null; fi
apt-get update -qq
DEBIAN_FRONTEND=noninteractive apt-get install -y -qq "$f"
rm -f "$f"
`

// checkMounts makes sure the VM sees nothing of the Mac's home folder but the
// empty share folder (Colima mounts the home folder when it is given no mount).
func checkMounts() error {
	h, _ := os.UserHomeDir()
	o, err := vmShell(false, fmt.Sprintf("ls -A %q 2>/dev/null; true", h))
	if err != nil {
		return fmt.Errorf("checking the VM's mounts: %v %s", err, o)
	}
	for _, name := range strings.Fields(o) {
		if name != ".si" {
			return fmt.Errorf("the %s VM sees %s/%s: it must see no host folder but %s; run `si sandbox host rm` and init again", Profile, h, name, shareDir())
		}
	}
	return nil
}

// HostStatus prints the VM, its mounts, sysbox, the socket and how many
// boxes of the default size fit.
func HostStatus(out io.Writer) error {
	sock := ColimaSocket()
	if _, err := os.Stat(sock); err != nil {
		fmt.Fprintf(out, "si sandbox: the %s VM is not running (si sandbox host init, or si sandbox host start)\n", Profile)
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	info, err := NewEngine(sock).Info(ctx)
	if err != nil {
		return err
	}
	_, sysbox := info.Runtimes[SysboxRuntime]
	mounts := "only the share folder"
	if err := checkMounts(); err != nil {
		mounts = err.Error()
	}
	fmt.Fprintf(out, "si sandbox: %s VM running · Docker %s · %d CPU · %.0f GiB memory\n", Profile, info.ServerVersion, info.NCPU, float64(info.MemTotal)/(1<<30))
	fmt.Fprintf(out, "  socket:  %s\n  mounts:  %s\n", sock, mounts)
	if sysbox {
		fmt.Fprintf(out, "  sysbox:  yes (boxes run unprivileged)\n")
	} else {
		fmt.Fprintf(out, "  sysbox:  missing (run si sandbox host init again)\n")
	}
	fit := (info.MemTotal - 4<<30) / (2 << 30)
	fmt.Fprintf(out, "  room:    about %d boxes of 2 GiB\n", max(0, fit))
	return nil
}

// HostStart, HostStop and HostRemove start, stop and delete the VM.
func HostStart(out io.Writer) error { return colima(out, "start", Profile, "--activate=false") }

func HostStop(out io.Writer) error { return colima(out, "stop", Profile) }

func HostRemove(out io.Writer) error { return colima(out, "delete", "--force", Profile) }
