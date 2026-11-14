package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/felipeg48/simple-ideas/internal/publish"
	"github.com/felipeg48/simple-ideas/internal/render"
)

// si publish course/ --to skilljar: the learner build of a course as a
// Skilljar course. Modules become sections; a reading without a quiz is an
// HTML lesson; a video is an asset; slides, labs, quizzes (and a reading with
// a quiz) are SCORM 1.2 web packages, one per lesson; instructor demos are
// never published. Skilljar fetches videos and packages from a URL, so si
// writes them into <course>/.si/publish/skilljar/ and you put that folder
// online at --base-url first. The ids go to <course>/.si/publish/skilljar.json
// so the next publish updates the course in place.

// sjFiles is the folder of files to put online; sjIDs the ids file.
func sjFiles(dir string) string { return filepath.Join(dir, ".si", "publish", "skilljar") }
func sjIDs(dir string) string   { return filepath.Join(dir, ".si", "publish", "skilljar.json") }

// skilljarCourse turns a course into what Skilljar gets, writing the videos
// and packages into sjFiles(g.Dir). base is the URL that folder is served at.
func skilljarCourse(g *deck.Guide, base string, stderr io.Writer) (publish.SJCourse, []string, error) {
	c := publish.SJCourse{Title: g.Meta.Title, Description: firstNonBlank(g.Meta.Description, g.Meta.Title), Duration: strings.TrimSpace(g.Meta.Duration)}
	files := sjFiles(g.Dir)
	if err := os.MkdirAll(files, 0o755); err != nil {
		return c, nil, err
	}
	pages, probs, _ := render.RenderGuide(g, render.Options{Media: render.MediaInline})
	if len(probs) > 0 {
		return c, nil, fmt.Errorf("%s (run si check)", probs[0])
	}
	byFile := map[string]render.GuidePage{}
	for _, p := range pages {
		byFile[p.File] = p
	}
	base = strings.TrimRight(firstNonBlank(base, "<--base-url>"), "/")
	var wrote []string
	put := func(name string, data []byte) (string, string, error) {
		sum := sha256.Sum256(data)
		wrote = append(wrote, name)
		return base + "/" + name, hex.EncodeToString(sum[:]), os.WriteFile(filepath.Join(files, name), data, 0o644)
	}
	learner := g.ForAudience(false)
	for i, m := range learner.Items {
		c.Lessons = append(c.Lessons, publish.SJLesson{Key: fmt.Sprintf("module-%d", i+1), Title: deck.ModuleTitle(m.Title), Type: "SECTION"})
		for _, it := range m.Children {
			if it.File == "" || it.LiveDemo() { // an instructor demo is never published
				continue
			}
			key, _ := filepath.Rel(g.Dir, it.File)
			key = filepath.ToSlash(key)
			flat := strings.ReplaceAll(strings.TrimSuffix(key, filepath.Ext(key)), "/", "-")
			l := publish.SJLesson{Key: key, Title: it.Title, Seconds: it.Minutes * 60}
			p, ok := byFile[it.File]
			switch {
			case it.Video != "":
				data, err := os.ReadFile(it.Video)
				if err != nil {
					fmt.Fprintf(stderr, "si: warning: %s: no video yet (si build films it); left out\n", key)
					continue
				}
				l.Type = "ASSET"
				if l.URL, l.Hash, err = put(flat+filepath.Ext(it.Video), data); err != nil {
					return c, nil, err
				}
			case !ok:
				continue
			case it.Kind == "reading" && !strings.Contains(p.Result.Body(), `class="si-quiz"`):
				l.Type, l.HTML = "HTML", p.Result.Body()
			default: // slides, a lab, a quiz: the page as a SCORM package of its own
				data, err := lessonPackage(g, p, it.Title)
				if err != nil {
					return c, nil, err
				}
				l.Type = "WEB_PACKAGE"
				if l.URL, l.Hash, err = put(flat+"-scorm12.zip", data); err != nil {
					return c, nil, err
				}
			}
			c.Lessons = append(c.Lessons, l)
		}
	}
	return c, wrote, nil
}

// lessonPackage is one lesson page as a SCORM 1.2 zip: the page (one SCO),
// its media, the shared scripts it links, and imsmanifest.xml.
func lessonPackage(g *deck.Guide, p render.GuidePage, title string) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	var names []string
	add := func(name string, data []byte) error {
		w, err := zw.Create(name)
		if err == nil {
			_, err = w.Write(data)
		}
		names = append(names, name)
		return err
	}
	if err := add(p.Rel, p.Result.HTML); err != nil {
		return nil, err
	}
	for _, m := range p.Result.Media {
		data, err := os.ReadFile(m.Src)
		if err != nil {
			return nil, err
		}
		if err := add(filepath.ToSlash(filepath.Join(filepath.Dir(p.Rel), m.Rel)), data); err != nil {
			return nil, err
		}
	}
	if render.UsesShared(p.Result.HTML) {
		tmp, err := os.MkdirTemp("", "si-sj-")
		if err != nil {
			return nil, err
		}
		defer os.RemoveAll(tmp)
		if err := render.WriteShared(tmp); err != nil {
			return nil, err
		}
		ents, _ := os.ReadDir(filepath.Join(tmp, render.ShareDir))
		for _, e := range ents {
			data, _ := os.ReadFile(filepath.Join(tmp, render.ShareDir, e.Name()))
			if err := add(render.ShareDir+"/"+e.Name(), data); err != nil {
				return nil, err
			}
		}
	}
	if err := add("imsmanifest.xml", render.ScormLessonManifest(filepath.Base(g.Dir)+"-"+p.Rel, title, p.Rel, names)); err != nil {
		return nil, err
	}
	err := zw.Close()
	return buf.Bytes(), err
}

// publishCourse is si publish <course dir> --to skilljar.
func publishCourse(dir string, o *options, stdout, stderr io.Writer) int {
	g, err := deck.LoadGuide(dir)
	if err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	if len(g.Problems) > 0 {
		fmt.Fprintf(stderr, "si: %s has problems; run si check %s first\n", dir, dir)
		return exitProblems
	}
	if o.to != "skilljar" {
		return usageErr(stderr, "a course goes to --to skilljar")
	}
	creds, err := publish.Load()
	if err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	if o.yes && creds.SkillJar.Key == "" {
		return needLogin(stderr, "skilljar")
	}
	c, wrote, err := skilljarCourse(g, o.baseURL, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	if len(wrote) > 0 {
		fmt.Fprintf(stdout, "Wrote %d file(s) for Skilljar to fetch into %s:\n", len(wrote), deck.Rel(sjFiles(g.Dir)))
		for _, w := range wrote {
			fmt.Fprintf(stdout, "  %s\n", w)
		}
		if o.baseURL == "" {
			fmt.Fprintln(stdout, "Put that folder online and pass its address as --base-url (Skilljar's API takes a URL, not an upload).")
		}
		fmt.Fprintln(stdout)
		if o.yes && o.baseURL == "" {
			return usageErr(stderr, "--to skilljar with videos or packages needs --base-url <where the files above are online>")
		}
	}
	ids, err := publish.LoadSJIDs(sjIDs(g.Dir))
	if err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	var dry io.Writer = stdout
	if o.yes {
		dry = nil
	} else {
		fmt.Fprintln(stdout, "Dry run: nothing is sent. These are the requests si would make (add --yes to send them):")
		fmt.Fprintln(stdout)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	id, err := publish.NewSkillJar(firstNonBlank(creds.SkillJar.Key, "<not logged in>")).Publish(ctx, c, ids, dry)
	if o.yes && ids.Course != "" { // even after an error: what was created is updated next time, not created twice
		if serr := ids.Save(sjIDs(g.Dir)); serr != nil {
			fmt.Fprintf(stderr, "si: %v\n", serr)
		}
	}
	if err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	if o.yes {
		fmt.Fprintf(stdout, "published to Skilljar: course %s (%d lessons; ids in %s). Publish it to your domain in the Skilljar dashboard.\n", id, len(c.Lessons), deck.Rel(sjIDs(g.Dir)))
	}
	return exitOK
}
