//go:build windows

package cdp

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// pipes hands Chrome its DevTools pipe on Windows, where a child gets no fds
// 3 and 4 (exec.Cmd.ExtraFiles is not supported): the two pipe handles are
// inherited and named with --remote-debugging-io-pipes=<it reads>,<it writes>.
func pipes(cmd *exec.Cmd, toChrome, fromChrome *os.File) []string {
	r, w := syscall.Handle(toChrome.Fd()), syscall.Handle(fromChrome.Fd())
	for _, h := range []syscall.Handle{r, w} {
		syscall.SetHandleInformation(h, syscall.HANDLE_FLAG_INHERIT, syscall.HANDLE_FLAG_INHERIT)
	}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.AdditionalInheritedHandles = append(cmd.SysProcAttr.AdditionalInheritedHandles, r, w)
	return []string{fmt.Sprintf("--remote-debugging-io-pipes=%d,%d", uintptr(r), uintptr(w))}
}
