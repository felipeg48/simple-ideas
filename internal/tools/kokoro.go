package tools

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// si voice install kokoro: a Python venv with kokoro-onnx and the two model
// files, all in KokoroDir (or --dir). Nothing else on the computer changes.

// KokoroPackage is the kokoro-onnx release si installs (it runs on Python
// 3.10 to 3.13).
const KokoroPackage = "kokoro-onnx==0.6.1"

// KokoroModelURL is the GitHub release the model files come from.
const KokoroModelURL = "https://github.com/thewh1teagle/kokoro-onnx/releases/download/model-files-v1.0"

// KokoroSHA256 pins each model file.
var KokoroSHA256 = map[string]string{
	"kokoro-v1.0.onnx": "7d5df8ecf7d4b1878015a32686053fd0eebe2bc377234608764cc0ef3636a6c5",
	"voices-v1.0.bin":  "bca610b8308e8d99f32e6fe4197e7ec01679264efed0cac9140fe9c29f1fbf7d",
}

// KokoroVenvPython is the venv's Python in a Kokoro folder.
func KokoroVenvPython(dir string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(dir, "venv", "Scripts", "python.exe")
	}
	return filepath.Join(dir, "venv", "bin", "python")
}

// KokoroInstall sets Kokoro up in Dir. The zero value of each other field
// is the real thing; tests swap in fakes.
type KokoroInstall struct {
	Dir     string
	BaseURL string                                   // model files ("" = KokoroModelURL)
	HTTP    *http.Client                             // nil = a client with a long timeout
	Python  []string                                 // the Python that makes the venv (nil = FindPython)
	Run     func(argv []string, out io.Writer) error // runs a program (nil = os/exec)
	Log     io.Writer                                // progress (stderr)
}

func (k KokoroInstall) run(argv []string, out io.Writer) error {
	if k.Run != nil {
		return k.Run(argv, out)
	}
	c := exec.Command(argv[0], argv[1:]...)
	c.Stdout, c.Stderr = out, out
	return c.Run()
}

func (k KokoroInstall) log() io.Writer {
	if k.Log == nil {
		return io.Discard
	}
	return k.Log
}

// Install is idempotent: a venv that imports kokoro_onnx and model files
// whose SHA-256 match are kept.
func (k KokoroInstall) Install() error {
	if err := os.MkdirAll(k.Dir, 0o755); err != nil {
		return err
	}
	py := KokoroVenvPython(k.Dir)
	if _, err := os.Stat(py); err == nil && k.run([]string{py, "-c", "import kokoro_onnx"}, io.Discard) == nil {
		fmt.Fprintf(k.log(), "  ✓ Python venv with kokoro-onnx  %s\n", filepath.Dir(filepath.Dir(py)))
	} else {
		if _, err := os.Stat(py); err != nil {
			base := k.Python
			if base == nil {
				if base = FindPython(); base == nil {
					return fmt.Errorf("%s", pythonNeeded())
				}
			}
			fmt.Fprintf(k.log(), "  making a Python venv (%s)…\n", strings.Join(base, " "))
			if err := k.run(append(append([]string{}, base...), "-m", "venv", filepath.Join(k.Dir, "venv")), k.log()); err != nil {
				return fmt.Errorf("python -m venv: %v", err)
			}
		}
		fmt.Fprintf(k.log(), "  pip install %s…\n", KokoroPackage)
		if err := k.run([]string{py, "-m", "pip", "install", "--disable-pip-version-check", "-q", KokoroPackage}, k.log()); err != nil {
			return fmt.Errorf("pip install %s: %v", KokoroPackage, err)
		}
	}
	for _, f := range KokoroFiles {
		if err := k.fetch(f); err != nil {
			return err
		}
	}
	return nil
}

// fetch downloads one model file unless it is there with the right hash:
// to a temporary file, checked, then renamed into place.
func (k KokoroInstall) fetch(name string) error {
	dest := filepath.Join(k.Dir, name)
	if sum, err := fileSHA256(dest); err == nil && sum == KokoroSHA256[name] {
		fmt.Fprintf(k.log(), "  ✓ %s\n", name)
		return nil
	}
	base := k.BaseURL
	if base == "" {
		base = KokoroModelURL
	}
	hc := k.HTTP
	if hc == nil {
		hc = &http.Client{Timeout: 30 * time.Minute}
	}
	res, err := hc.Get(base + "/" + name)
	if err != nil {
		return fmt.Errorf("download %s: %v", name, err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: %s", name, res.Status)
	}
	tmp, err := os.CreateTemp(k.Dir, "."+name+".*.part")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // gone after the rename; cleans up a failure
	h := sha256.New()
	p := &progress{w: k.log(), name: name, total: res.ContentLength}
	_, err = io.Copy(io.MultiWriter(tmp, h, p), res.Body)
	p.done()
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return fmt.Errorf("download %s: %v", name, err)
	}
	if got := hex.EncodeToString(h.Sum(nil)); got != KokoroSHA256[name] {
		return fmt.Errorf("download %s: SHA-256 %s, want %s (a broken or changed file; nothing was kept)", name, got, KokoroSHA256[name])
	}
	return os.Rename(tmp.Name(), dest)
}

func fileSHA256(p string) (string, error) {
	f, err := os.Open(p)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// progress prints a download's percentage on one line, at most every 2%.
type progress struct {
	w           io.Writer
	name        string
	total, n    int64
	lastPercent int64
}

func (p *progress) Write(b []byte) (int, error) {
	p.n += int64(len(b))
	if p.total > 0 {
		if pc := p.n * 100 / p.total; pc >= p.lastPercent+2 {
			p.lastPercent = pc
			fmt.Fprintf(p.w, "\r  %s  %3d%%  (%d of %d MB)", p.name, pc, p.n>>20, p.total>>20)
		}
	}
	return len(b), nil
}

func (p *progress) done() { fmt.Fprintf(p.w, "\r  ✓ %s  %d MB%20s\n", p.name, p.n>>20, "") }

// UninstallKokoro removes what Install put in dir (the venv and the model
// files, and dir itself when nothing else is left) and nothing else.
func UninstallKokoro(dir string) error {
	for _, f := range append([]string{"venv"}, KokoroFiles...) {
		if err := os.RemoveAll(filepath.Join(dir, f)); err != nil {
			return err
		}
	}
	if left, err := os.ReadDir(dir); err == nil && len(left) == 0 {
		return os.Remove(dir)
	}
	return nil
}

// FindPython is a Python Kokoro runs on (3.10 to 3.13), as the command
// that starts it, or nil.
func FindPython() []string {
	return pickPython(pythonCandidates(runtime.GOOS), func(argv []string) string {
		out, err := exec.Command(argv[0], append(argv[1:], "-c", "import sys;print('%d.%d'%sys.version_info[:2])")...).Output()
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(out))
	})
}

// pythonCandidates are the commands to try, newest first: the versioned
// names, then the plain ones (py -3.x is the Windows launcher).
func pythonCandidates(goos string) [][]string {
	var c [][]string
	for _, v := range []string{"3.13", "3.12", "3.11", "3.10"} {
		if goos == "windows" {
			c = append(c, []string{"py", "-" + v})
		} else {
			c = append(c, []string{"python" + v})
		}
	}
	if goos == "windows" {
		return append(c, []string{"python"})
	}
	return append(c, []string{"python3"}, []string{"python"})
}

// pickPython is the first candidate whose version (from version: "3.12")
// kokoro-onnx accepts ("" for a command that is not there).
func pickPython(cands [][]string, version func([]string) string) []string {
	for _, c := range cands {
		var major, minor int
		if _, err := fmt.Sscanf(version(c), "%d.%d", &major, &minor); err == nil && major == 3 && minor >= 10 && minor <= 13 {
			return c
		}
	}
	return nil
}

// pythonNeeded explains how to get a Python Kokoro runs on.
func pythonNeeded() string {
	how := map[string]string{
		"mac": "brew install python@3.13", "debian": "sudo apt install python3 python3-venv   (Python 3.10 to 3.13)",
		"fedora": "sudo dnf install python3.13", "arch": "sudo pacman -S python   (or: uv python install 3.13)",
		"windows": "winget install Python.Python.3.13", "other": "install Python 3.13 from python.org",
	}
	h, ok := how[Platform()]
	if !ok {
		h = how["other"]
	}
	return "si voice install kokoro needs Python 3.10 to 3.13 (" + KokoroPackage + " does not run on newer ones yet), and none was found.\n" +
		"  Install it:  " + h + "\n  More:        https://www.python.org/downloads/\n  Then run si voice install kokoro again."
}
