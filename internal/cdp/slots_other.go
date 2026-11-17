//go:build !(darwin || linux || freebsd || openbsd || netbsd || dragonfly || windows)

package cdp

import (
	"os"
	"sync"
)

// No file locks here: the slots are per process only.
var held sync.Map

func tryLockFile(f *os.File) bool {
	_, taken := held.LoadOrStore(f.Name(), true)
	return !taken
}

func unlockFile(f *os.File) { held.Delete(f.Name()) }
