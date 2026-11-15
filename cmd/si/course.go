package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/felipeg48/simple-ideas/internal/config"
	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/felipeg48/simple-ideas/internal/render"
	"github.com/felipeg48/simple-ideas/internal/tools"
	"github.com/felipeg48/simple-ideas/internal/webdemo"
)

// filmStale films the course's web demo lessons (type: demo with app:) whose
// <script>.mp4 is missing or older than the script, with si demo run, beside
// the script. A failure is a warning: the lesson then says it is not filmed.
// Deck lessons shown as their video (kind=video) go through deckLesson.
// film is false with --no-film: no video is made, the PDFs still are.
func filmStale(g *deck.Guide, film bool, voice string, flags []string, cfg *config.Config, stdout, stderr io.Writer) {
	for _, it := range g.Lessons() {
		if it.Deck != "" {
			deckLesson(it, film, voice, flags, cfg, stdout, stderr)
			continue
		}
		if it.Script == "" || !film {
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

// deckLesson makes what a deck lesson shown as its video needs, beside the
// deck, when it is missing or older than the deck: <deck>.mp4 (and
// <deck>.vtt) narrated from its Notes: with the voice (--voice, else the
// deck's or si.conf's) and its still <deck>.poster.jpg (the first slide),
// <deck>.pdf (· pdf) and <deck>.handout.pdf
// (· handout), all from one si build of the deck. The deck's time is the
// newest of its file and the files it includes (sections, code).
// ponytail: pictures and canvases it uses do not count; touch the deck.
// flags go to that si build as they are (--no-live, --trust).
func deckLesson(it *deck.NavItem, film bool, voice string, flags []string, cfg *config.Config, stdout, stderr io.Writer) {
	if _, err := os.Stat(it.Deck); err != nil {
		return
	}
	changed := deckChanged(it.Deck)
	stale := func(f string) bool { st, err := os.Stat(f); return err != nil || st.ModTime().Before(changed) }
	name := strings.TrimSuffix(filepath.Base(it.Deck), filepath.Ext(it.Deck))
	pdf, handout := render.DeckCompanions(it.Deck)
	base := strings.TrimSuffix(it.Deck, filepath.Ext(it.Deck))
	args := []string{"build", it.Deck}
	var moves [][2]string // built file -> beside the deck
	if film && stale(it.Video) {
		if voice == "" {
			if d, err := deck.Parse(it.Deck); err == nil {
				if dv, _, err := webdemo.ForDeck(d, cfg, ""); err == nil {
					voice = dv.Default.String()
				}
			}
		}
		args = append(append(args, "--mp4", "--voice", voice), flags...)
		moves = append(moves, [2]string{name + ".mp4", it.Video}, [2]string{name + ".vtt", base + ".vtt"}, [2]string{name + ".poster.jpg", render.PosterOf(it.Video)})
	}
	if it.PDF && stale(pdf) {
		args = append(args, "--pdf")
		moves = append(moves, [2]string{filepath.Join("pdfs", name+".pdf"), pdf})
	}
	if it.Handout && stale(handout) {
		args = append(args, "--handout")
		moves = append(moves, [2]string{filepath.Join("handouts", name+".pdf"), handout})
	}
	if len(moves) == 0 {
		return
	}
	tmp, err := os.MkdirTemp("", "si-lesson-")
	if err != nil {
		fmt.Fprintf(stderr, "  warning: %s: %v\n", deck.Rel(it.Deck), err)
		return
	}
	defer os.RemoveAll(tmp)
	fmt.Fprintf(stdout, "  making %s's video lesson (%s; --no-film skips the video)\n", deck.Rel(it.Deck), strings.Join(args[2:], " "))
	if code := run(append(args, "-o", tmp), stdout, stderr); code != exitOK {
		fmt.Fprintf(stderr, "  warning: %s: si build failed; what it did not make stays out of the lesson\n", deck.Rel(it.Deck))
	}
	for _, m := range moves {
		if _, err := os.Stat(filepath.Join(tmp, m[0])); err == nil {
			if err := copyFile(filepath.Join(tmp, m[0]), m[1]); err != nil {
				fmt.Fprintf(stderr, "  warning: %v\n", err)
			}
		} else if filepath.Ext(m[1]) == ".vtt" {
			os.Remove(m[1]) // a silent video: older captions would not fit it
		}
	}
}

// deckChanged is the newest modification time of a deck file and of the
// files its <!-- include: --> lines splice in.
func deckChanged(file string) time.Time {
	files := []string{file}
	if d, _ := deck.Parse(file); d != nil {
		files = append(files, d.Includes...)
	}
	var t time.Time
	for _, f := range files {
		if st, err := os.Stat(f); err == nil && st.ModTime().After(t) {
			t = st.ModTime()
		}
	}
	return t
}

// filmFlags are the course build's flags a deck lesson's filming keeps: a
// stale kind=video deck with cues runs its setup: and start: as si play does
// unless --no-live; --trust answers for its pinned sources.
func filmFlags(o *options) []string {
	var out []string
	if o.noLive {
		out = append(out, "--no-live")
	}
	if o.trust {
		out = append(out, "--trust")
	}
	return out
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
