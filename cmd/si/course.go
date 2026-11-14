package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/felipeg48/simple-ideas/internal/render"
	"github.com/felipeg48/simple-ideas/internal/tools"
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

// coursePDF writes the printed course into out (workbook.html, or
// instructor-guide.html with --instructor) and prints it to
// <course>-workbook.pdf / <course>-instructor-guide.pdf there.
func coursePDF(g *deck.Guide, out string, ro render.Options, stdout io.Writer) error {
	chrome := findChrome()
	if chrome == "" {
		return fmt.Errorf("%s", tools.Need("chrome", "--pdf"))
	}
	p, probs := render.CourseBook(g, ro)
	if len(probs) > 0 {
		return fmt.Errorf("%s", probs[0])
	}
	page := filepath.Join(out, p.Rel)
	if render.UsesShared(p.Result.HTML) {
		if err := render.WriteShared(out); err != nil {
			return err
		}
	}
	if err := os.WriteFile(page, p.Result.HTML, 0o644); err != nil {
		return err
	}
	pdf := filepath.Join(out, filepath.Base(g.Dir)+"-"+strings.TrimSuffix(p.Rel, ".html")+".pdf")
	if err := pagePDF(chrome, page, pdf, ""); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "  %-40s (pdf, %d KB)\n", filepath.ToSlash(filepath.Join(filepath.Base(out), filepath.Base(pdf))), fileKB(pdf))
	return nil
}

// scormPackage turns the learner build in out into a SCORM 1.2 package:
// imsmanifest.xml (and scorm.html for one SCO) in out, all of it zipped to
// <course>-scorm12.zip beside out. It returns the zip's path.
func scormPackage(g *deck.Guide, out string, single bool) (string, error) {
	if single {
		if err := os.WriteFile(filepath.Join(out, render.ScormLauncher), render.ScormLauncherHTML(g.Meta.Title), 0o644); err != nil {
			return "", err
		}
	}
	var files []string
	err := filepath.WalkDir(out, func(p string, e os.DirEntry, err error) error {
		if err != nil || e.IsDir() {
			return err
		}
		r, err := filepath.Rel(out, p)
		if r = filepath.ToSlash(r); err == nil && r != "imsmanifest.xml" {
			files = append(files, r)
		}
		return err
	})
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(out, "imsmanifest.xml"), render.ScormManifest(g, files, single), 0o644); err != nil {
		return "", err
	}
	return zipDir(out, filepath.Join(filepath.Dir(filepath.Clean(out)), filepath.Base(g.Dir)+"-scorm12.zip"))
}
