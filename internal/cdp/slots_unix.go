//go:build darwin || linux || freebsd || openbsd || netbsd || dragonfly

package cdp

import (
	"os"
	"syscall"
)

func tryLockFile(f *os.File) bool {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB) == nil
}

func unlockFile(f *os.File) { syscall.Flock(int(f.Fd()), syscall.LOCK_UN) }
