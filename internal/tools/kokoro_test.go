package tools

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
)

// fakeKokoro serves two small model files and pins their hashes for the test.
func fakeKokoro(t *testing.T) (*httptest.Server, *int32) {
	t.Helper()
	body := map[string]string{"kokoro-v1.0.onnx": "model bytes", "voices-v1.0.bin": "voice bytes"}
	old := KokoroSHA256
	KokoroSHA256 = map[string]string{}
	for f, b := range body {
		s := sha256.Sum256([]byte(b))
		KokoroSHA256[f] = hex.EncodeToString(s[:])
	}
	t.Cleanup(func() { KokoroSHA256 = old })
	var gets int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&gets, 1)
		b, ok := body[strings.TrimPrefix(r.URL.Path, "/")]
		if !ok {
			http.NotFound(w, r)
			return
		}
		io.WriteString(w, b)
	}))
	t.Cleanup(srv.Close)
	return srv, &gets
}

// fakeRun stands in for python: -m venv makes the venv's python, pip
// install marks the package present, -c import succeeds once it is.
func fakeRun(calls *[]string) func([]string, io.Writer) error {
	installed := false
	return func(argv []string, _ io.Writer) error {
		line := strings.Join(argv, " ")
		*calls = append(*calls, line)
		switch {
		case strings.Contains(line, "-m venv"):
			py := KokoroVenvPython(filepath.Dir(argv[len(argv)-1]))
			os.MkdirAll(filepath.Dir(py), 0o755)
			return os.WriteFile(py, nil, 0o755)
		case strings.Contains(line, "pip install"):
			installed = true
		case strings.Contains(line, "import kokoro_onnx") && !installed:
			return os.ErrNotExist
		}
		return nil
	}
}

func TestKokoroInstallIdempotent(t *testing.T) {
	srv, gets := fakeKokoro(t)
	dir := filepath.Join(t.TempDir(), "kokoro")
	var calls []string
	k := KokoroInstall{Dir: dir, BaseURL: srv.URL, Python: []string{"python3.12"}, Run: fakeRun(&calls)}
	if err := k.Install(); err != nil {
		t.Fatal(err)
	}
	if *gets != 2 || len(calls) != 2 || !strings.HasPrefix(calls[0], "python3.12 -m venv ") || !strings.Contains(calls[1], "pip install --disable-pip-version-check -q "+KokoroPackage) {
		t.Fatalf("gets %d, calls %q", *gets, calls)
	}
	for _, f := range KokoroFiles {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Error(err)
		}
	}
	if left, _ := filepath.Glob(filepath.Join(dir, ".*.part")); len(left) != 0 {
		t.Errorf("temporary files left: %v", left)
	}
	// again: everything is there, nothing is fetched or installed
	calls = nil
	if err := k.Install(); err != nil {
		t.Fatal(err)
	}
	if *gets != 2 || len(calls) != 1 || !strings.Contains(calls[0], "import kokoro_onnx") {
		t.Errorf("second run: gets %d, calls %q", *gets, calls)
	}
	// a broken file is fetched again
	os.WriteFile(filepath.Join(dir, "voices-v1.0.bin"), []byte("junk"), 0o644)
	if err := k.Install(); err != nil || *gets != 3 {
		t.Errorf("repair: %v, gets %d", err, *gets)
	}
	// uninstall removes only what install made
	os.WriteFile(filepath.Join(dir, "mine.txt"), nil, 0o644)
	if err := UninstallKokoro(dir); err != nil {
		t.Fatal(err)
	}
	if left, _ := os.ReadDir(dir); len(left) != 1 || left[0].Name() != "mine.txt" {
		t.Errorf("left: %v", left)
	}
	os.Remove(filepath.Join(dir, "mine.txt"))
	os.MkdirAll(filepath.Join(dir, "venv"), 0o755)
	UninstallKokoro(dir)
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("an emptied folder goes too: %v", err)
	}
}

func TestKokoroInstallBadHash(t *testing.T) {
	srv, _ := fakeKokoro(t)
	KokoroSHA256["voices-v1.0.bin"] = strings.Repeat("0", 64)
	dir := t.TempDir()
	var calls []string
	err := KokoroInstall{Dir: dir, BaseURL: srv.URL, Python: []string{"python3"}, Run: fakeRun(&calls)}.Install()
	if err == nil || !strings.Contains(err.Error(), "SHA-256") {
		t.Fatalf("err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "voices-v1.0.bin")); !os.IsNotExist(err) {
		t.Error("a file with the wrong hash must not be kept")
	}
	if left, _ := filepath.Glob(filepath.Join(dir, ".*.part")); len(left) != 0 {
		t.Errorf("temporary files left: %v", left)
	}
}

func TestPickPython(t *testing.T) {
	have := map[string]string{"python3.12": "3.12", "python3": "3.15", "python": "2.7"}
	version := func(argv []string) string { return have[strings.Join(argv, " ")] }
	if got := pickPython(pythonCandidates("darwin"), version); !reflect.DeepEqual(got, []string{"python3.12"}) {
		t.Errorf("got %v", got)
	}
	delete(have, "python3.12")
	if got := pickPython(pythonCandidates("linux"), version); got != nil {
		t.Errorf("3.15 and 2.7 are not for kokoro-onnx: got %v", got)
	}
	have["py -3.11"] = "3.11"
	if got := pickPython(pythonCandidates("windows"), version); !reflect.DeepEqual(got, []string{"py", "-3.11"}) {
		t.Errorf("windows: %v", got)
	}
	if !strings.Contains(pythonNeeded(), "Python 3.10 to 3.13") {
		t.Error(pythonNeeded())
	}
}

func TestKokoroPythonPrefersVenv(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SI_KOKORO_DIR", dir)
	t.Setenv("SI_KOKORO_PYTHON", "")
	if p := KokoroPython(); p == KokoroVenvPython(dir) {
		t.Fatal("no venv yet")
	}
	py := KokoroVenvPython(dir)
	os.MkdirAll(filepath.Dir(py), 0o755)
	os.WriteFile(py, nil, 0o755)
	if p := KokoroPython(); p != py {
		t.Errorf("KokoroPython = %q, want the venv's %q", p, py)
	}
	t.Setenv("SI_KOKORO_PYTHON", "/my/python")
	if p := KokoroPython(); p != "/my/python" {
		t.Errorf("SI_KOKORO_PYTHON wins: %q", p)
	}
}
