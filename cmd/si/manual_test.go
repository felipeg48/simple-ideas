package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestRunManual(t *testing.T) {
	out := t.TempDir()
	o := &options{output: out, slides: true}
	var so, se bytes.Buffer
	if code := runManual(o, &so, &se); code != exitOK {
		t.Fatalf("runManual=%d stderr=%s", code, se.String())
	}
	b, err := os.ReadFile(filepath.Join(out, "si-manual.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(b, []byte("si User Manual")) || !bytes.Contains(b, []byte("si-deck")) {
		t.Error("manual HTML missing expected content")
	}
}
