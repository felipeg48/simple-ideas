//go:build !unix && !windows

package ui

import (
	"os"
	"time"
)

func isTerminal(*os.File) bool { return false }
func vt(*os.File) bool         { return false }
func Keys(*os.File) func()     { return func() {} }
func CPUTime() time.Duration   { return 0 }
