package render

import (
	"fmt"
	"html"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

// SCORM 1.2 (si build course/ --scorm): an imsmanifest.xml beside the
// learner build, and for one SCO for the whole course (--scorm=single) the
// scorm.html launcher. course.js finds the LMS's API in a parent or opener
// window and reports status, score and progress; with no LMS it keeps the
// progress in localStorage as on the web.

// ScormLauncher is the file a single-SCO package launches.
const ScormLauncher = "scorm.html"

var scormIDRE = regexp.MustCompile(`[^A-Za-z0-9_.-]+`)

// ScormManifest is imsmanifest.xml for a built course: files are every file
// of the package (slash paths relative to its root, imsmanifest.xml left
// out). single makes one SCO (scorm.html) for the whole course; otherwise
// each lesson that counts towards progress is a SCO of its own, its page the
// launch file, and every other file is one shared asset resource. A quiz
// lesson with pass: gives its mastery score.
func ScormManifest(full *deck.Guide, files []string, single bool) []byte {
	g := full.ForAudience(false)
	id := strings.Trim(scormIDRE.ReplaceAllString(filepath.Base(g.Dir), "-"), "-")
	if id == "" {
		id = "course"
	}
	e := html.EscapeString
	var org, res strings.Builder
	sorted := append([]string(nil), files...)
	sort.Strings(sorted)
	if single {
		fmt.Fprintf(&org, "      <item identifier=\"item-course\" identifierref=\"res-course\">\n        <title>%s</title>\n      </item>\n", e(g.Meta.Title))
		fmt.Fprintf(&res, "    <resource identifier=\"res-course\" type=\"webcontent\" adlcp:scormtype=\"sco\" href=\"%s\">\n", ScormLauncher)
		for _, f := range sorted {
			fmt.Fprintf(&res, "      <file href=\"%s\"/>\n", e(f))
		}
		res.WriteString("    </resource>\n")
	} else {
		sco := map[string]bool{}
		n := 0
		for i, m := range g.Items {
			fmt.Fprintf(&org, "      <item identifier=\"mod-%d\">\n        <title>%s</title>\n", i+1, e(deck.ModuleTitle(m.Title)))
			for _, it := range m.Children {
				if it.File == "" || !countsTowards(it) {
					continue
				}
				n++
				rel := guideRel(g, it.File)
				sco[rel] = true
				fmt.Fprintf(&org, "        <item identifier=\"item-%d\" identifierref=\"res-%d\">\n          <title>%s</title>\n", n, n, e(it.Title))
				if it.Kind == "quiz" {
					if d, err := deck.ParseWith(it.File, g.Defaults()); err == nil {
						if p, ok := deck.ParsePass(d.Meta.Pass); ok && d.Meta.Pass != "" {
							fmt.Fprintf(&org, "          <adlcp:masteryscore>%d</adlcp:masteryscore>\n", p)
						}
					}
				}
				org.WriteString("        </item>\n")
				fmt.Fprintf(&res, "    <resource identifier=\"res-%d\" type=\"webcontent\" adlcp:scormtype=\"sco\" href=\"%s\">\n      <file href=\"%s\"/>\n      <dependency identifierref=\"res-shared\"/>\n    </resource>\n", n, e(rel), e(rel))
			}
			org.WriteString("      </item>\n")
		}
		res.WriteString("    <resource identifier=\"res-shared\" type=\"webcontent\" adlcp:scormtype=\"asset\">\n")
		for _, f := range sorted {
			if !sco[f] {
				fmt.Fprintf(&res, "      <file href=\"%s\"/>\n", e(f))
			}
		}
		res.WriteString("    </resource>\n")
	}
	return scormXML(id, g.Meta.Title, org.String(), res.String())
}

// ScormLessonManifest is imsmanifest.xml for one lesson packed on its own
// (si publish --to skilljar): page is the SCO, files every file of the
// package (page included).
func ScormLessonManifest(id, title, page string, files []string) []byte {
	e := html.EscapeString
	id = strings.Trim(scormIDRE.ReplaceAllString(id, "-"), "-")
	var res strings.Builder
	fmt.Fprintf(&res, "    <resource identifier=\"res-lesson\" type=\"webcontent\" adlcp:scormtype=\"sco\" href=\"%s\">\n", e(page))
	for _, f := range files {
		fmt.Fprintf(&res, "      <file href=\"%s\"/>\n", e(f))
	}
	res.WriteString("    </resource>\n")
	org := fmt.Sprintf("      <item identifier=\"item-lesson\" identifierref=\"res-lesson\">\n        <title>%s</title>\n      </item>\n", e(title))
	return scormXML(id, title, org, res.String())
}

func scormXML(id, title, org, res string) []byte {
	e := html.EscapeString
	return []byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<manifest identifier="si-%s" version="1.0"
  xmlns="http://www.imsproject.org/xsd/imscp_rootv1p1p2"
  xmlns:adlcp="http://www.adlnet.org/xsd/adlcp_rootv1p2">
  <metadata>
    <schema>ADL SCORM</schema>
    <schemaversion>1.2</schemaversion>
  </metadata>
  <organizations default="org">
    <organization identifier="org">
      <title>%s</title>
%s    </organization>
  </organizations>
  <resources>
%s  </resources>
</manifest>
`, e(id), e(title), org, res))
}

// Body is a rendered document's main column (its text, without the page
// around it): what si publish sends as an HTML lesson.
func (r Result) Body() string { return r.body }

// ScormLauncherHTML is scorm.html: the course home in a full-window frame.
// It opens the SCORM session once (LMSInitialize) and closes it when the
// learner leaves (LMSCommit, LMSFinish); the pages in the frame report
// through it (window.siScormSingle tells course.js not to open a second one).
func ScormLauncherHTML(title string) []byte {
	t := html.EscapeString(title)
	return []byte(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>` + t + `</title>
<style>html,body{margin:0;height:100%;overflow:hidden}iframe{display:block;border:0;inline-size:100%;block-size:100%}</style>
</head><body>
<iframe src="index.html" title="` + t + `"></iframe>
<script>
(function () {
  window.siScormSingle = true;
  function look(w) {
    for (var n = 0; w && n < 12; n++) {
      try { if (w.API) return w.API; } catch (e) { return null; }
      if (w.parent === w) break;
      w = w.parent;
    }
    return null;
  }
  var api = look(window.parent !== window ? window.parent : null) || (window.opener && look(window.opener));
  if (!api) return;
  api.LMSInitialize('');
  var done = false;
  function bye() { if (done) return; done = true; try { api.LMSCommit(''); api.LMSFinish(''); } catch (e) {} }
  window.addEventListener('pagehide', bye);
  window.addEventListener('beforeunload', bye);
})();
</script>
</body></html>
`)
}
