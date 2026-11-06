//go:build !darwin

package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "si Studio desktop is macOS only for now; use si --web")
	os.Exit(1)
}
