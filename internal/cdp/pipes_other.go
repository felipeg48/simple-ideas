//go:build !windows

package cdp

import (
	"os"
	"os/exec"
)

// pipes hands Chrome its DevTools pipe as fds 3 (it reads) and 4 (it writes).
func pipes(cmd *exec.Cmd, toChrome, fromChrome *os.File) []string {
	cmd.ExtraFiles = []*os.File{toChrome, fromChrome}
	return nil
}
