//go:build windows

package termpty

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

// ErrUnsupported: this Windows has no ConPTY (before Windows 10 1809).
var ErrUnsupported = errors.New("this Windows has no pseudo console (ConPTY needs Windows 10 1809 or later)")

// Available reports whether this Windows has ConPTY.
func Available() bool {
	return windows.NewLazySystemDLL("kernel32.dll").NewProc("CreatePseudoConsole").Find() == nil
}

// conPTY is a pseudo console and the two pipes to it.
type conPTY struct {
	in   *os.File // what is written here is the program's input
	out  *os.File // the console's rendering of what the program prints
	hpc  windows.Handle
	proc windows.Handle
	once sync.Once
	mu   sync.Mutex // guards hpc against a resize racing Close
}

func (p *conPTY) Read(b []byte) (int, error)  { return p.out.Read(b) }
func (p *conPTY) Write(b []byte) (int, error) { return p.in.Write(b) }

// Close ends the program (a closed terminal takes its shell with it), closes
// the console, which flushes its last output and ends Read, then the pipes.
func (p *conPTY) Close() error {
	p.once.Do(func() {
		windows.TerminateProcess(p.proc, 1)
		p.closeConsole()
		p.in.Close()
		p.out.Close()
		windows.CloseHandle(p.proc)
	})
	return nil
}

func (p *conPTY) closeConsole() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.hpc != 0 {
		windows.ClosePseudoConsole(p.hpc)
		p.hpc = 0
	}
}

func coord(cols, rows int) windows.Coord {
	if cols <= 0 || rows <= 0 {
		cols, rows = 80, 24
	}
	return windows.Coord{X: int16(min(cols, 32767)), Y: int16(min(rows, 32767))}
}

// Start starts c under a ConPTY pseudo console of cols×rows (0×0: 80×24)
// and returns the console's end. c.Process is set, so c.Wait works as after
// c.Start; c.Stdin, c.Stdout and c.Stderr are not used (the console is the
// program's terminal). c.SysProcAttr.CmdLine, when set, is the command line.
func Start(c *exec.Cmd, cols, rows int) (io.ReadWriteCloser, error) {
	if c.Err != nil {
		return nil, c.Err
	}
	if c.Process != nil {
		return nil, errors.New("termpty: already started")
	}
	if !Available() {
		return nil, ErrUnsupported
	}
	var inR, inW, outR, outW windows.Handle
	if err := windows.CreatePipe(&inR, &inW, nil, 0); err != nil {
		return nil, err
	}
	if err := windows.CreatePipe(&outR, &outW, nil, 0); err != nil {
		windows.CloseHandle(inR)
		windows.CloseHandle(inW)
		return nil, err
	}
	var hpc windows.Handle
	err := windows.CreatePseudoConsole(coord(cols, rows), inR, outW, 0, &hpc)
	// the console holds its own copies of its ends of the pipes
	windows.CloseHandle(inR)
	windows.CloseHandle(outW)
	if err != nil {
		windows.CloseHandle(inW)
		windows.CloseHandle(outR)
		return nil, err
	}
	p := &conPTY{in: os.NewFile(uintptr(inW), "conpty-in"), out: os.NewFile(uintptr(outR), "conpty-out"), hpc: hpc}
	pi, err := spawn(c, hpc)
	if err != nil {
		windows.ClosePseudoConsole(hpc)
		p.in.Close()
		p.out.Close()
		return nil, err
	}
	windows.CloseHandle(pi.Thread)
	p.proc = pi.Process
	// os.FindProcess while our handle keeps the process object (and its
	// pid) alive; its own handle waits for the exit and reads the code.
	if c.Process, err = os.FindProcess(int(pi.ProcessId)); err != nil {
		p.Close()
		return nil, err
	}
	go func() { // the program is gone: close the console so Read ends
		windows.WaitForSingleObject(pi.Process, windows.INFINITE)
		p.closeConsole()
	}()
	return p, nil
}

// spawn creates c's process attached to the pseudo console hpc.
func spawn(c *exec.Cmd, hpc windows.Handle) (*windows.ProcessInformation, error) {
	attrs, err := windows.NewProcThreadAttributeList(1)
	if err != nil {
		return nil, err
	}
	defer attrs.Delete()
	// the attribute's value is the HPCON itself, not a pointer to it
	if err := attrs.Update(windows.PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE, *(*unsafe.Pointer)(unsafe.Pointer(&hpc)), unsafe.Sizeof(hpc)); err != nil {
		return nil, err
	}
	si := &windows.StartupInfoEx{ProcThreadAttributeList: attrs.List()}
	si.Cb = uint32(unsafe.Sizeof(*si))
	// NULL std handles, given explicitly: Windows fills NULL ones with the
	// pseudo console's, and a child of a parent whose stdio is redirected
	// (go test, a service) does not get the parent's instead
	si.Flags = windows.STARTF_USESTDHANDLES
	si.StdInput, si.StdOutput, si.StdErr = 0, 0, 0

	line := windows.ComposeCommandLine(c.Args)
	if c.SysProcAttr != nil && c.SysProcAttr.CmdLine != "" {
		line = c.SysProcAttr.CmdLine
	}
	app, err := windows.UTF16PtrFromString(c.Path)
	if err != nil {
		return nil, err
	}
	cmdLine, err := windows.UTF16PtrFromString(line)
	if err != nil {
		return nil, err
	}
	var dir *uint16
	if c.Dir != "" {
		if dir, err = windows.UTF16PtrFromString(c.Dir); err != nil {
			return nil, err
		}
	}
	env := c.Env
	if env == nil {
		env = os.Environ()
	}
	block := envBlock(env)
	pi := new(windows.ProcessInformation)
	flags := uint32(windows.EXTENDED_STARTUPINFO_PRESENT | windows.CREATE_UNICODE_ENVIRONMENT)
	if err := windows.CreateProcess(app, cmdLine, nil, nil, false, flags, &block[0], dir, &si.StartupInfo, pi); err != nil {
		return nil, &os.PathError{Op: "CreateProcess", Path: c.Path, Err: err}
	}
	return pi, nil
}

// envBlock is env as CreateProcess takes it: NUL-separated UTF-16 strings
// ending in an empty one.
func envBlock(env []string) []uint16 {
	var b []uint16
	for _, kv := range env {
		if kv == "" || strings.ContainsRune(kv, 0) {
			continue
		}
		b = append(b, utf16.Encode([]rune(kv))...)
		b = append(b, 0)
	}
	if len(b) == 0 {
		b = append(b, 0)
	}
	return append(b, 0)
}

// Resize sets the pseudo console's size.
func Resize(p io.ReadWriteCloser, cols, rows int) {
	if c, ok := p.(*conPTY); ok {
		c.mu.Lock()
		defer c.mu.Unlock()
		if c.hpc != 0 {
			windows.ResizePseudoConsole(c.hpc, coord(cols, rows))
		}
	}
}
