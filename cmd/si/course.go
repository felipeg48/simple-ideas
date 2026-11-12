package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

// filmStale films the course's web demo lessons (type: demo with app:) whose
// <script>.mp4 is missing or older than the script, with si demo run, beside
// the script. A failure is a warning: the lesson then says it is not filmed.
func filmStale(g *deck.Guide, stdout, stderr io.Writer) {
	for _, it := range g.Lessons() {
		if it.Script == "" {
			continue
		}
		src, err := os.Stat(it.Script)
		if err != nil {
			continue
		}
		if v, err := os.Stat(it.Video); err == nil && !v.ModTime().Before(src.ModTime()) {
			continue
		}
		fmt.Fprintf(stdout, "  filming %s (si demo run; --no-film skips it)\n", deck.Rel(it.Script))
		if code := runDemo([]string{"run", it.Script, "-o", filepath.Dir(it.Script)}, stdout, stderr); code != exitOK {
			fmt.Fprintf(stderr, "  warning: %s: filming failed; the lesson says it is not filmed yet\n", deck.Rel(it.Script))
		}
	}
}

// zipDir writes every file under dir into the zip file dest (paths inside
// relative to dir) and returns dest.
func zipDir(dir, dest string) (string, error) {
	f, err := os.Create(dest)
	if err != nil {
		return "", err
	}
	zw := zip.NewWriter(f)
	err = filepath.WalkDir(dir, func(p string, e os.DirEntry, err error) error {
		if err != nil || e.IsDir() {
			return err
		}
		r, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		w, err := zw.Create(filepath.ToSlash(r))
		if err != nil {
			return err
		}
		in, err := os.Open(p)
		if err != nil {
			return err
		}
		defer in.Close()
		_, err = io.Copy(w, in)
		return err
	})
	if cerr := zw.Close(); err == nil {
		err = cerr
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return dest, err
}
