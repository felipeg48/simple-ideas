// Package tools knows the programs si can use but does not ship: what each is
// for, where to find it, and how to install it on this computer. A feature
// that needs one calls Need, which explains instead of failing bare; `si
// doctor` lists them all.
package tools

import (
	"fmt"
	"github.com/felipeg48/simple-ideas/internal/home"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Tool is one optional program.
type Tool struct {
	Name    string              // short name: chrome, ffmpeg, …
	Label   string              // what people call it
	For     string              // what si uses it for
	Bins    []string            // names to look up on PATH, in order
	Paths   map[string][]string // GOOS -> absolute paths to try too
	Install map[string]string   // platform (see platform) -> command or advice
	URL     string              // where to read more
	Probe   func() string       // when set, finds the tool instead of Bins and Paths
}

// All are the programs si can use, in the order si doctor lists them.
var All = []Tool{
	{
		Name: "chrome", Label: "Google Chrome or Chromium",
		For:  "PDF, PNG and SVG exports, handouts, videos (--mp4), slide screenshots",
		Bins: []string{"google-chrome", "google-chrome-stable", "chromium", "chromium-browser", "microsoft-edge"},
		Paths: map[string][]string{
			"darwin": {"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome", "/Applications/Chromium.app/Contents/MacOS/Chromium",
				"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge", "/Applications/Brave Browser.app/Contents/MacOS/Brave Browser"},
			"windows": {`C:\Program Files\Google\Chrome\Application\chrome.exe`, `C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
				`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`},
		},
		Install: map[string]string{
			"mac": "brew install --cask google-chrome", "debian": "sudo apt install chromium",
			"fedora": "sudo dnf install chromium", "arch": "sudo pacman -S chromium",
			"windows": "winget install Google.Chrome", "other": "install Google Chrome or Chromium",
		},
		URL: "https://www.google.com/chrome/",
	},
	{
		Name: "ffmpeg", Label: "ffmpeg",
		For:   "videos: --mp4, --narration, si code --gif and --mp4",
		Bins:  []string{"ffmpeg"},
		Paths: map[string][]string{"darwin": {"/opt/homebrew/bin/ffmpeg", "/usr/local/bin/ffmpeg"}},
		Install: map[string]string{
			"mac": "brew install ffmpeg", "debian": "sudo apt install ffmpeg",
			"fedora": "sudo dnf install ffmpeg-free", "arch": "sudo pacman -S ffmpeg",
			"windows": "winget install Gyan.FFmpeg", "other": "install ffmpeg",
		},
		URL: "https://ffmpeg.org/download.html",
	},
	{
		Name: "git", Label: "git",
		For:  "the Studio's History (changes, commits, branches, pull and push)",
		Bins: []string{"git"},
		Install: map[string]string{
			"mac": "xcode-select --install   (or: brew install git)", "debian": "sudo apt install git",
			"fedora": "sudo dnf install git", "arch": "sudo pacman -S git",
			"windows": "winget install Git.Git", "other": "install git",
		},
		URL: "https://git-scm.com/downloads",
	},
	{
		Name: "tmux", Label: "tmux",
		For:   "shell panes beside the slides in si --terminal (Lab) and the teleprompter pane",
		Bins:  []string{"tmux"},
		Paths: map[string][]string{"darwin": {"/opt/homebrew/bin/tmux", "/usr/local/bin/tmux"}},
		Install: map[string]string{
			"mac": "brew install tmux", "debian": "sudo apt install tmux",
			"fedora": "sudo dnf install tmux", "arch": "sudo pacman -S tmux",
			"windows": "tmux runs in WSL: wsl --install, then sudo apt install tmux", "other": "install tmux",
		},
		URL: "https://github.com/tmux/tmux/wiki/Installing",
	},
	{
		Name: "cloudflared", Label: "Cloudflare's cloudflared",
		For:   "si play --tunnel: your audience on a video call, through a temporary public link",
		Bins:  []string{"cloudflared"},
		Paths: map[string][]string{"darwin": {"/opt/homebrew/bin/cloudflared", "/usr/local/bin/cloudflared"}},
		Install: map[string]string{
			"mac": "brew install cloudflared", "windows": "winget install --id Cloudflare.cloudflared",
			"debian": "see the Linux packages at the link below (pkg.cloudflare.com)",
			"fedora": "see the Linux packages at the link below (pkg.cloudflare.com)",
			"arch":   "sudo pacman -S cloudflared", "other": "download cloudflared from the link below",
		},
		URL: "https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/downloads/",
	},
	{
		Name: "tailscale", Label: "Tailscale",
		For:   "si play --remote --https=tailscale: certificates every phone already trusts",
		Bins:  []string{"tailscale"},
		Paths: map[string][]string{"darwin": {"/Applications/Tailscale.app/Contents/MacOS/Tailscale", "/usr/local/bin/tailscale", "/opt/homebrew/bin/tailscale"}},
		Install: map[string]string{
			"mac": "brew install --cask tailscale", "windows": "winget install Tailscale.Tailscale",
			"debian": "curl -fsSL https://tailscale.com/install.sh | sh", "fedora": "curl -fsSL https://tailscale.com/install.sh | sh",
			"arch": "sudo pacman -S tailscale", "other": "curl -fsSL https://tailscale.com/install.sh | sh",
		},
		URL: "https://tailscale.com/download",
	},
	{
		Name: "gh", Label: "GitHub CLI (gh)",
		For:   "code from private GitHub repositories in decks (public ones need nothing)",
		Bins:  []string{"gh"},
		Paths: map[string][]string{"darwin": {"/opt/homebrew/bin/gh", "/usr/local/bin/gh"}},
		Install: map[string]string{
			"mac": "brew install gh", "debian": "sudo apt install gh", "fedora": "sudo dnf install gh",
			"arch": "sudo pacman -S github-cli", "windows": "winget install GitHub.cli", "other": "install the GitHub CLI",
		},
		URL: "https://cli.github.com/",
	},
	{
		Name: "kokoro", Label: "Kokoro (kokoro-onnx, a local voice)",
		For:     "si demo run: the narration in a natural voice (af_heart), offline",
		Probe:   findKokoro,
		Install: kokoroInstall(),
		URL:     "https://github.com/thewh1teagle/kokoro-onnx/releases/tag/model-files-v1.0",
	},
	{
		Name: "espeak-ng", Label: "eSpeak NG",
		For:   "si demo run on Linux: the computer's own voice (voice: system)",
		Bins:  []string{"espeak-ng", "espeak"},
		Paths: map[string][]string{"darwin": {"/opt/homebrew/bin/espeak-ng", "/usr/local/bin/espeak-ng"}},
		Install: map[string]string{
			"mac": "not needed on macOS: voice: system uses say (or: brew install espeak-ng)", "debian": "sudo apt install espeak-ng",
			"fedora": "sudo dnf install espeak-ng", "arch": "sudo pacman -S espeak-ng",
			"windows": "not needed on Windows: voice: system uses the voices Windows has", "other": "install espeak-ng",
		},
		URL: "https://github.com/espeak-ng/espeak-ng",
	},
}

// KokoroDir is where Kokoro's model files are: $SI_KOKORO_DIR, else
// ~/.si/kokoro.
func KokoroDir() string {
	if d := os.Getenv("SI_KOKORO_DIR"); d != "" {
		return d
	}
	return home.Path("kokoro", home.Cache)
}

// KokoroPython is the Python that has kokoro-onnx: $SI_KOKORO_PYTHON, else
// the venv si voice install kokoro made in KokoroDir, else python3 (python
// on Windows).
func KokoroPython() string {
	if p := os.Getenv("SI_KOKORO_PYTHON"); p != "" {
		return p
	}
	if p := KokoroVenvPython(KokoroDir()); fileExists(p) {
		return p
	}
	if runtime.GOOS == "windows" {
		return "python"
	}
	return "python3"
}

// KokoroFiles are the model files Kokoro needs in KokoroDir.
var KokoroFiles = []string{"kokoro-v1.0.onnx", "voices-v1.0.bin"}

// kokoroInstall is the same one command everywhere.
func kokoroInstall() map[string]string {
	m := map[string]string{}
	for _, p := range []string{"mac", "windows", "debian", "fedora", "arch", "other"} {
		m[p] = "si voice install kokoro   (a Python venv, kokoro-onnx and the model files, all in " + KokoroDir() + ")"
	}
	return m
}

func fileExists(p string) bool { st, err := os.Stat(p); return err == nil && !st.IsDir() }

// findKokoro is the model folder when both the Python package and the model
// files are there ("" otherwise).
func findKokoro() string {
	for _, f := range KokoroFiles {
		if st, err := os.Stat(filepath.Join(KokoroDir(), f)); err != nil || st.IsDir() {
			return ""
		}
	}
	py, err := exec.LookPath(KokoroPython())
	if err != nil {
		return ""
	}
	if exec.Command(py, "-c", "import importlib.util,sys;sys.exit(0 if importlib.util.find_spec('kokoro_onnx') else 1)").Run() != nil {
		return ""
	}
	return KokoroDir()
}

// Get returns the tool by name.
func Get(name string) (Tool, bool) {
	for _, t := range All {
		if t.Name == name {
			return t, true
		}
	}
	return Tool{}, false
}

// Find returns the tool's path on this computer, or "".
func Find(name string) string {
	t, ok := Get(name)
	if !ok {
		return ""
	}
	if t.Probe != nil {
		return t.Probe()
	}
	for _, p := range t.Paths[runtime.GOOS] {
		if filepath.IsAbs(p) {
			if st, err := os.Stat(p); err == nil && !st.IsDir() {
				return p
			}
		}
	}
	for _, b := range t.Bins {
		if p, err := exec.LookPath(b); err == nil {
			return p
		}
	}
	return ""
}

// Platform is how this computer installs software: mac, windows, debian,
// fedora, arch or other (read from /etc/os-release on Linux).
func Platform() string {
	switch runtime.GOOS {
	case "darwin":
		return "mac"
	case "windows":
		return "windows"
	case "linux":
		raw, _ := os.ReadFile("/etc/os-release")
		return linuxFamily(string(raw))
	}
	return "other"
}

func linuxFamily(osRelease string) string {
	ids := ""
	for _, l := range strings.Split(osRelease, "\n") {
		if k, v, ok := strings.Cut(l, "="); ok && (k == "ID" || k == "ID_LIKE") {
			ids += " " + strings.ToLower(strings.Trim(v, `"`))
		}
	}
	switch {
	case strings.Contains(ids, "debian") || strings.Contains(ids, "ubuntu"):
		return "debian"
	case strings.Contains(ids, "fedora") || strings.Contains(ids, "rhel") || strings.Contains(ids, "centos"):
		return "fedora"
	case strings.Contains(ids, "arch"):
		return "arch"
	}
	return "other"
}

// How is the install advice for this computer.
func (t Tool) How() string {
	if s, ok := t.Install[Platform()]; ok {
		return s
	}
	return t.Install["other"]
}

// Need explains a missing tool for a feature: what it is for, the install
// command for this computer, and where to read more. si prints it instead of
// a bare error.
func Need(name, feature string) string {
	t, ok := Get(name)
	if !ok {
		return feature + " needs " + name
	}
	return fmt.Sprintf("%s needs %s, which is not installed (or not on your PATH).\n  Install it:  %s\n  More:        %s\n  Then run the same command again. `si doctor` lists every optional tool.",
		feature, t.Label, t.How(), t.URL)
}
