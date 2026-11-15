//go:build windows

package webdemo

import (
	"encoding/base64"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"unicode/utf16"

	"github.com/felipeg48/simple-ideas/internal/termpty"
)

func ownGroup(c *exec.Cmd) {
	c.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
}

// signalGroup asks the program and its children to stop (taskkill without /F
// sends them a close request). A terminal hangs up at once: its console
// programs get no close request, so they are ended with it.
func signalGroup(pid int, hup bool) {
	if hup {
		killGroup(pid)
		return
	}
	exec.Command("taskkill", "/T", "/PID", strconv.Itoa(pid)).Run()
}

func killGroup(pid int) { exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(pid)).Run() }

// pipes is a shell's stdin and its stdout and stderr together.
type pipes struct {
	r  *io.PipeReader
	w  *io.PipeWriter
	in io.WriteCloser
}

func (p pipes) Read(b []byte) (int, error)  { return p.r.Read(b) }
func (p pipes) Write(b []byte) (int, error) { return p.in.Write(b) }
func (p pipes) Close() error                { p.in.Close(); return p.w.Close() }

// startShell starts the shell under a ConPTY pseudo console of cols×rows.
// cmd (whose prompt cannot report an exit status) and a Windows without
// ConPTY run on pipes instead: no colours, and si echoes what it types
// itself and asks for the exit status after each command (pipes is true).
func startShell(c *exec.Cmd, cols, rows int) (io.ReadWriteCloser, bool, error) {
	if !isCmd(c.Path) && termpty.Available() {
		p, err := termpty.Start(c, cols, rows)
		return p, false, err
	}
	in, err := c.StdinPipe()
	if err != nil {
		return nil, true, err
	}
	r, w := io.Pipe()
	c.Stdout, c.Stderr = w, w
	if err := c.Start(); err != nil {
		return nil, true, err
	}
	return pipes{r, w, in}, true, nil
}

func resizeShell(f io.ReadWriteCloser, cols, rows int) { termpty.Resize(f, cols, rows) }

func shellName(path string) string {
	return strings.ToLower(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
}

func isCmd(path string) bool { return shellName(path) == "cmd" }

// DefaultShell is PowerShell (pwsh, else powershell.exe), else cmd.
func DefaultShell() string {
	for _, s := range []string{"pwsh", "powershell"} {
		if p, err := exec.LookPath(s); err == nil {
			return p
		}
	}
	if c := os.Getenv("COMSPEC"); c != "" {
		return c
	}
	return "cmd.exe"
}

// psPrompt is si's PowerShell prompt: the folder's name and $, no
// PSReadLine (no predictions, no history file), and each command's exit
// status in the window title, "si;133;D;<status>;<count>" (ConPTY passes
// title changes on; the count makes every prompt a change), which output
// reads like the OSC 133;D mark of si's Unix prompts; after them ";7;" and
// the current folder, which si play's Viewer follows (serve oscCwd: a
// ConPTY child's folder cannot be asked for).
const psPrompt = `Remove-Module PSReadLine -ErrorAction SilentlyContinue
$global:siN = 0
function global:prompt {
  $c = if ($?) { 0 } elseif ($LASTEXITCODE) { $LASTEXITCODE } else { 1 }
  $global:siN++
  $Host.UI.RawUI.WindowTitle = 'si;133;D;' + $c + ';' + $global:siN + ';7;' + $PWD.ProviderPath
  (Split-Path -Leaf (Get-Location)) + ' $ '
}`

// encoded is s for powershell -EncodedCommand (base64 of UTF-16LE): no
// command-line quoting to get wrong.
func encoded(s string) string {
	u := utf16.Encode([]rune(s))
	b := make([]byte, 2*len(u))
	for i, r := range u {
		b[2*i], b[2*i+1] = byte(r), byte(r>>8)
	}
	return base64.StdEncoding.EncodeToString(b)
}

// shellCommand is the terminal's shell (shell:, else DefaultShell) with si's
// prompt: PowerShell's from psPrompt; cmd's the folder's name and $ (cmd
// cannot show the current folder's name alone, so it is the folder the shell
// starts in); any other shell (bash from Git for Windows, say) as for Unix.
func shellCommand(shell, dir, rc string) *exec.Cmd {
	if shell == "" {
		shell = DefaultShell()
	}
	env := append(os.Environ(), "TERM=xterm-256color", "COLORTERM=truecolor")
	var c *exec.Cmd
	switch shellName(shell) {
	case "pwsh", "powershell":
		c = exec.Command(shell, "-NoLogo", "-NoProfile", "-NoExit", "-EncodedCommand", encoded(psPrompt))
	case "cmd":
		c = exec.Command(shell)
		// verbatim: cmd's /K takes the rest of the line as it is
		c.SysProcAttr = &syscall.SysProcAttr{CmdLine: syscall.EscapeArg(c.Path) + " /Q /K PROMPT " + cmdLeaf(dir) + "$S$$$S"}
	case "bash":
		c = exec.Command(shell, "--noprofile", "--rcfile", filepath.Join(rc, "bashrc"), "-i")
	default:
		c = exec.Command(shell, "-i")
		env = append(env, "ENV="+filepath.Join(rc, "shrc"))
	}
	c.Dir, c.Env = dir, env
	return c
}

// cmdLeaf is dir's name for cmd's PROMPT ($S is a space): $ written $$, and the characters
// cmd would read as its own (& | < > ^) left out.
func cmdLeaf(dir string) string {
	abs, err := filepath.Abs(dir)
	if err != nil {
		abs = dir
	}
	leaf := filepath.Base(abs)
	leaf = strings.NewReplacer("$", "$$", "&", "", "|", "", "<", "", ">", "", "^", "").Replace(leaf)
	if leaf == "" || leaf == `\` || leaf == "." {
		leaf = "$P"
	}
	return leaf
}

// psFollow wraps the user's own PowerShell prompt (after their profile) so
// it also puts the current folder in the window title, "si;7;<folder>",
// which si play's Viewer follows: ConPTY gives no other way to ask.
const psFollow = `$global:siPrompt = $function:prompt
function global:prompt {
  $Host.UI.RawUI.WindowTitle = 'si;7;' + $PWD.ProviderPath
  & $global:siPrompt
}`

// LiveShell is the user's own shell as si play's drawer runs it live (not
// clean): PowerShell with its profile and prompt, the folder in the title.
func LiveShell(shell string) *exec.Cmd {
	switch shellName(shell) {
	case "pwsh", "powershell":
		return exec.Command(shell, "-NoLogo", "-NoExit", "-EncodedCommand", encoded(psFollow))
	}
	return exec.Command(shell)
}
