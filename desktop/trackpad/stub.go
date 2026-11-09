//go:build !darwin

package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "si-trackpad is macOS only (it reads the Mac trackpad)")
	os.Exit(1)
}
