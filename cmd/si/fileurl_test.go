package main

import (
	"runtime"
	"testing"
)

func TestFileURL(t *testing.T) {
	if got := fileURL("/Users/me/talk.html"); got != "file:///Users/me/talk.html" {
		t.Errorf("unix path: %s", got)
	}
	if runtime.GOOS == "windows" {
		if got := fileURL(`C:\si\talk.html`); got != "file:///C:/si/talk.html" {
			t.Errorf("windows path: %s", got)
		}
	} else if got := fileURL("C:/si/talk.html"); got != "file:///C:/si/talk.html" {
		t.Errorf("drive path: %s", got)
	}
}
