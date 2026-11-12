package webdemo

import (
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
)

// Report is one run of a script.
type Report struct {
	Script   string          `json:"script"`
	Title    string          `json:"title"`
	App      string          `json:"app"`
	Viewport string          `json:"viewport"`
	Voice    string          `json:"voice,omitempty"`
	Chrome   string          `json:"chrome"`
	SI       string          `json:"si"`
	Date     string          `json:"date"`
	Seconds  float64         `json:"seconds"`
	Passed   int             `json:"passed"`
	Failed   int             `json:"failed"`
	Skipped  int             `json:"skipped"`
	Features []FeatureResult `json:"features"`
	// the run's own steps, never features: start:, prepare: and step:; ## Before;
	// ## After; stop: and cleanup:
	Start    []StepResult   `json:"start,omitempty"`
	Before   *FeatureResult `json:"before,omitempty"`
	After    *FeatureResult `json:"after,omitempty"`
	Teardown []StepResult   `json:"teardown,omitempty"`
	// si demo run: the whole run's video and every file it wrote, relative
	// to the report folder
	Video string   `json:"video,omitempty"`
	Files []string `json:"files,omitempty"`
}

// FeatureResult is one feature's run.
type FeatureResult struct {
	Name     string       `json:"name"`
	What     string       `json:"what,omitempty"`
	Line     int          `json:"line,omitempty"`
	Status   string       `json:"status"` // passed, failed, skipped
	Seconds  float64      `json:"seconds"`
	Steps    []StepResult `json:"steps"`
	Errors   []string     `json:"consoleErrors,omitempty"`
	Requests []string     `json:"failedRequests,omitempty"`
	// si demo run: the clips, relative to the report folder
	Video     string `json:"video,omitempty"`         // picture and voice
	Silent    string `json:"videoOnly,omitempty"`     // the picture alone
	Captioned string `json:"videoCaptions,omitempty"` // picture, voice, captions under it
}

// StepResult is one step's run.
type StepResult struct {
	Line       int      `json:"line,omitempty"`
	Text       string   `json:"step"`
	Kind       string   `json:"kind,omitempty"`
	Status     string   `json:"status"`
	Seconds    float64  `json:"seconds"`
	Expected   string   `json:"expected,omitempty"`
	Found      string   `json:"found,omitempty"`
	Candidates []string `json:"closest,omitempty"`
	Shot       string   `json:"screenshot,omitempty"` // relative to the report folder
	Note       string   `json:"note,omitempty"`
	Output     string   `json:"output,omitempty"` // a failed terminal or program step: its last 40 lines
}

func (r *Report) count() {
	r.Passed, r.Failed, r.Skipped = 0, 0, 0
	for _, f := range r.Features {
		switch f.Status {
		case "passed":
			r.Passed++
		case "failed":
			r.Failed++
		default:
			r.Skipped++
		}
	}
}

// OK reports whether every feature passed and nothing around them failed
// (start:, prepare:, ## Before, ## After, stop:, cleanup:).
func (r *Report) OK() bool { return r.Failed == 0 && len(r.aroundFailed()) == 0 }

// aroundFailed are the run's own parts that failed, as test cases for JUnit.
func (r *Report) aroundFailed() []FeatureResult {
	var out []FeatureResult
	for _, f := range r.around() {
		if f != nil && f.Status == "failed" {
			out = append(out, *f)
		}
	}
	return out
}

// around are the run's own parts as two pseudo-features: what ran before the
// features (start:, prepare:, ## Before) and after them (## After, stop:,
// cleanup:); nil when there was nothing.
func (r *Report) around() [2]*FeatureResult {
	part := func(name string, steps ...[]StepResult) *FeatureResult {
		f := &FeatureResult{Name: name, Status: "passed"}
		for _, ss := range steps {
			for _, s := range ss {
				f.Steps = append(f.Steps, s)
				f.Seconds += s.Seconds
				if s.Status == "failed" {
					f.Status = "failed"
				}
			}
		}
		if len(f.Steps) == 0 {
			return nil
		}
		return f
	}
	var before, after []StepResult
	if r.Before != nil {
		before = r.Before.Steps
	}
	if r.After != nil {
		after = r.After.Steps
	}
	return [2]*FeatureResult{part("Before the features", r.Start, before), part("After the features", after, r.Teardown)}
}

// Write writes report.json and report.html into dir (where the screenshots
// already are); the page inlines them, so it travels alone.
func (r *Report) Write(dir string) error {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "report.json"), append(b, '\n'), 0o644); err != nil {
		return err
	}
	var sb strings.Builder
	if err := reportTmpl.Execute(&sb, r.view(dir)); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "report.html"), []byte(sb.String()), 0o644)
}

type stepView struct {
	StepResult
	Img template.URL
}

type featureView struct {
	FeatureResult
	Steps []stepView
	Aside bool // the run's own part, not a feature
}

type reportView struct {
	*Report
	Views []featureView
}

func (r *Report) view(dir string) reportView {
	var fs []featureView
	around := r.around()
	var all []featureView
	if around[0] != nil {
		all = append(all, featureView{FeatureResult: *around[0], Aside: true})
	}
	for _, f := range r.Features {
		all = append(all, featureView{FeatureResult: f})
	}
	if around[1] != nil {
		all = append(all, featureView{FeatureResult: *around[1], Aside: true})
	}
	for _, fv := range all {
		f := fv.FeatureResult
		for _, s := range f.Steps {
			sv := stepView{StepResult: s}
			if s.Shot != "" {
				if raw, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(s.Shot))); err == nil {
					sv.Img = template.URL("data:image/png;base64," + base64.StdEncoding.EncodeToString(raw))
				}
			}
			fv.Steps = append(fv.Steps, sv)
		}
		fs = append(fs, fv)
	}
	return reportView{r, fs}
}

// Suite is a folder's run: one report per script, in name order.
type Suite struct {
	Title   string    `json:"title"`
	SI      string    `json:"si"`
	Date    string    `json:"date"`
	Seconds float64   `json:"seconds"`
	Passed  int       `json:"passed"`
	Failed  int       `json:"failed"`
	Skipped int       `json:"skipped"`
	Reports []*Report `json:"scripts"`
	Dirs    []string  `json:"dirs"` // each script's folder, relative to the suite's
}

// Add counts a script's report in; dir is its folder, relative to the suite's.
func (s *Suite) Add(r *Report, dir string) {
	s.Reports, s.Dirs = append(s.Reports, r), append(s.Dirs, filepath.ToSlash(dir))
	s.Passed, s.Failed, s.Skipped, s.Seconds = s.Passed+r.Passed, s.Failed+r.Failed, s.Skipped+r.Skipped, s.Seconds+r.Seconds
}

// OK reports whether every script passed.
func (s *Suite) OK() bool {
	for _, r := range s.Reports {
		if !r.OK() {
			return false
		}
	}
	return true
}

// Write writes the combined report.json and report.html into dir (each
// script's own folder below it holds its screenshots).
func (s *Suite) Write(dir string) error {
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "report.json"), append(b, '\n'), 0o644); err != nil {
		return err
	}
	var views []reportView
	for i, r := range s.Reports {
		views = append(views, r.view(filepath.Join(dir, filepath.FromSlash(s.Dirs[i]))))
	}
	var sb strings.Builder
	if err := suiteTmpl.Execute(&sb, struct {
		*Suite
		Views []reportView
	}{s, views}); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "report.html"), []byte(sb.String()), 0o644)
}

// JUnit is the suite as JUnit XML: a test suite per script.
func (s *Suite) JUnit() ([]byte, error) { return junit(s.Title, s.Reports) }

var tmplFuncs = template.FuncMap{"secs": func(f float64) string { return fmt.Sprintf("%.2fs", f) }}

var reportTmpl = template.Must(template.Must(template.New("r").Funcs(tmplFuncs).Parse(reportStyle)).Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Title}}: web demo report</title>
{{template "style"}}</head><body><main>
<h1>{{.Title}}</h1>
{{template "body" .}}
</main></body></html>
`))

var suiteTmpl = template.Must(template.Must(template.New("s").Funcs(tmplFuncs).Parse(reportStyle)).Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Title}}: web demo report</title>
{{template "style"}}</head><body><main>
<h1>{{.Title}}</h1>
<div class="meta"><span>{{len .Reports}} script(s)</span><span>si {{.SI}}</span><span>{{.Date}}</span></div>
<div class="sum"><span class="pill passed">{{.Passed}} passed</span><span class="pill failed">{{.Failed}} failed</span>{{if .Skipped}}<span class="pill skipped">{{.Skipped}} skipped</span>{{end}}<span class="pill">{{secs .Seconds}}</span></div>
<nav class="toc"><ol>{{range $i, $v := .Views}}<li><a href="#s{{$i}}">{{.Title}}</a> <span class="{{if .OK}}passed{{else}}failed{{end}}">{{.Passed}} passed, {{.Failed}} failed</span></li>{{end}}</ol></nav>
{{range $i, $v := .Views}}<article id="s{{$i}}"><h2 class="script">{{.Title}}</h2>
{{template "body" .}}</article>{{end}}
</main></body></html>
`))

const reportStyle = `{{define "style"}}<style>
:root{--bg:#f7f7f4;--fg:#1d1f24;--muted:#5d6370;--card:#fff;--line:#dcdcd6;--ok:#1f7a4d;--bad:#b3261e;--skip:#7a6a1f;--accent:#1f4fa3;color-scheme:light dark}
@media (prefers-color-scheme:dark){:root{--bg:#15171c;--fg:#e8e8e3;--muted:#a3a8b3;--card:#1e2128;--line:#343842;--ok:#5fcf95;--bad:#ff8a80;--skip:#e0c860;--accent:#8ab4ff}}
*{box-sizing:border-box}
body{margin:0;background:var(--bg);color:var(--fg);font:clamp(.95rem,.9rem + .25vw,1.05rem)/1.5 system-ui,-apple-system,"Segoe UI",sans-serif;padding:clamp(1rem,3vw,3rem)}
main{max-width:70rem;margin:0 auto}
h1{font-size:clamp(1.5rem,1.2rem + 1.5vw,2.4rem);margin:0 0 .3em}
h2{font-size:1.25em;margin:0}
.meta{color:var(--muted);display:flex;flex-wrap:wrap;gap:.3em 1.2em;margin-bottom:1.5em}
.sum{display:flex;flex-wrap:wrap;gap:.6em;margin:0 0 2em}
.pill{border-radius:999px;padding:.2em .8em;border:1px solid var(--line);background:var(--card);font-weight:600}
.passed{color:var(--ok)}.failed{color:var(--bad)}.skipped{color:var(--skip)}
section{background:var(--card);border:1px solid var(--line);border-radius:.8em;padding:1.2em clamp(1em,2vw,1.6em);margin:0 0 1.2em}
section.st-failed{border-color:var(--bad)}
header{display:flex;flex-wrap:wrap;justify-content:space-between;gap:.5em;align-items:baseline}
.what{color:var(--muted);margin:.3em 0 .8em}
ol{list-style:none;padding:0;margin:0}
li{border-top:1px solid var(--line);padding:.45em 0}
.row{display:flex;gap:.7em;align-items:baseline}
.mark{width:1.2em;flex:none;font-weight:700}
code{font:.92em ui-monospace,SFMono-Regular,Menlo,monospace;overflow-wrap:anywhere}
.t{margin-left:auto;color:var(--muted);font-variant-numeric:tabular-nums;flex:none}
dl{display:grid;grid-template-columns:max-content 1fr;gap:.2em 1em;margin:.6em 0 .2em 1.9em}
dt{color:var(--muted)}dd{margin:0;overflow-wrap:anywhere}
figure{margin:.6em 0 .2em 1.9em}
figure img{max-width:100%;height:auto;border:1px solid var(--line);border-radius:.4em}
figcaption{color:var(--muted);font-size:.9em}
a{color:var(--accent)}
.out{margin:0;padding:.6em .8em;max-height:24em;overflow:auto;background:#0f1420;color:#d8dee9;border-radius:.4em;font:.85em/1.45 ui-monospace,SFMono-Regular,Menlo,monospace;white-space:pre-wrap}
.log{margin:.8em 0 0;color:var(--bad)}
.log ul{margin:.2em 0 0;padding-left:1.2em}
section.aside{background:transparent;border-style:dashed}
.toc ol{list-style:decimal;padding-left:1.4em;margin:0 0 2em}.toc li{border:0;padding:.15em 0}
h2.script{font-size:clamp(1.25rem,1.1rem + .8vw,1.8rem);margin:2em 0 .3em}
</style>{{end}}
{{define "body"}}<div class="meta"><span>{{.Script}}</span><span>{{.App}}</span><span>{{.Viewport}}</span><span>{{.Chrome}}</span><span>si {{.SI}}</span><span>{{.Date}}</span></div>
<div class="sum"><span class="pill passed">{{.Passed}} passed</span><span class="pill failed">{{.Failed}} failed</span>{{if .Skipped}}<span class="pill skipped">{{.Skipped}} skipped</span>{{end}}<span class="pill">{{secs .Seconds}}</span>{{if .Video}}<a class="pill" href="{{.Video}}">▶ {{.Video}}</a>{{end}}</div>
{{range .Views}}<section class="st-{{.Status}}{{if .Aside}} aside{{end}}">
<header><h2>{{.Name}}</h2><span class="{{.Status}}">{{.Status}} · {{secs .Seconds}}{{if .Video}} · <a href="{{.Video}}">▶ {{.Video}}</a>{{end}}{{if .Captioned}} · <a href="{{.Captioned}}">with captions</a>{{end}}{{if .Silent}} · <a href="{{.Silent}}">picture only</a>{{end}}</span></header>
{{if .What}}<p class="what">{{.What}}</p>{{end}}
<ol>{{range .Steps}}<li>
<div class="row"><span class="mark {{.Status}}">{{if eq .Status "passed"}}✓{{else if eq .Status "failed"}}✗{{else}}–{{end}}</span><code>{{.Text}}</code><span class="t">{{if .Line}}line {{.Line}} · {{end}}{{if ne .Status "skipped"}}{{secs .Seconds}}{{else}}skipped{{end}}</span></div>
{{if eq .Status "failed"}}<dl>{{if .Expected}}<dt>expected</dt><dd>{{.Expected}}</dd>{{end}}<dt>found</dt><dd>{{.Found}}</dd>{{if .Candidates}}<dt>closest</dt><dd>{{range $i, $c := .Candidates}}{{if $i}}, {{end}}<code>{{$c}}</code>{{end}}</dd>{{end}}{{if .Output}}<dt>output</dt><dd><pre class="out">{{.Output}}</pre></dd>{{end}}</dl>{{end}}
{{if .Img}}<figure><img src="{{.Img}}" alt="the page after: {{.Text}}"><figcaption>{{.Shot}}</figcaption></figure>{{end}}
</li>{{end}}</ol>
{{if or .Errors .Requests}}<div class="log">{{if .Errors}}<strong>Console errors</strong><ul>{{range .Errors}}<li><code>{{.}}</code></li>{{end}}</ul>{{end}}{{if .Requests}}<strong>Failed requests</strong><ul>{{range .Requests}}<li><code>{{.}}</code></li>{{end}}</ul>{{end}}</div>{{end}}
</section>{{end}}{{end}}`

// JUnit is the report as JUnit XML: one test case per feature.
func (r *Report) JUnit() ([]byte, error) { return junit(r.Title, []*Report{r}) }

// junit is JUnit XML with a test suite per report: a test case per feature,
// and one per part around them that failed.
func junit(title string, reps []*Report) ([]byte, error) {
	type failure struct {
		Message string `xml:"message,attr"`
		Type    string `xml:"type,attr"`
		Text    string `xml:",chardata"`
	}
	type skipped struct{}
	type testcase struct {
		Name      string   `xml:"name,attr"`
		Classname string   `xml:"classname,attr"`
		Time      string   `xml:"time,attr"`
		Failure   *failure `xml:"failure,omitempty"`
		Skipped   *skipped `xml:"skipped,omitempty"`
		SystemOut string   `xml:"system-out,omitempty"`
	}
	type suite struct {
		XMLName  xml.Name   `xml:"testsuite"`
		Name     string     `xml:"name,attr"`
		Tests    int        `xml:"tests,attr"`
		Failures int        `xml:"failures,attr"`
		Errors   int        `xml:"errors,attr"`
		Skipped  int        `xml:"skipped,attr"`
		Time     string     `xml:"time,attr"`
		Stamp    string     `xml:"timestamp,attr,omitempty"`
		Cases    []testcase `xml:"testcase"`
	}
	type suites struct {
		XMLName  xml.Name `xml:"testsuites"`
		Name     string   `xml:"name,attr"`
		Tests    int      `xml:"tests,attr"`
		Failures int      `xml:"failures,attr"`
		Time     string   `xml:"time,attr"`
		Suites   []suite
	}
	all := suites{Name: title}
	var total float64
	for _, r := range reps {
		fs := append(append([]FeatureResult(nil), r.Features...), r.aroundFailed()...)
		s := suite{Name: r.Title, Tests: len(fs), Failures: r.Failed + len(fs) - len(r.Features), Skipped: r.Skipped, Time: fmt.Sprintf("%.3f", r.Seconds), Stamp: r.Date}
		for _, f := range fs {
			tc := testcase{Name: f.Name, Classname: strings.TrimSuffix(r.Script, filepath.Ext(r.Script)), Time: fmt.Sprintf("%.3f", f.Seconds)}
			var log strings.Builder
			for _, st := range f.Steps {
				fmt.Fprintf(&log, "%s %s (%.2fs)\n", st.Status, st.Text, st.Seconds)
				if st.Status != "failed" {
					continue
				}
				var b strings.Builder
				if st.Line > 0 {
					fmt.Fprintf(&b, "%s:%d: ", r.Script, st.Line)
				}
				fmt.Fprintf(&b, "%s\n", st.Text)
				if st.Expected != "" {
					fmt.Fprintf(&b, "expected: %s\n", st.Expected)
				}
				fmt.Fprintf(&b, "found: %s\n", st.Found)
				if len(st.Candidates) > 0 {
					fmt.Fprintf(&b, "closest: %s\n", strings.Join(st.Candidates, ", "))
				}
				if st.Shot != "" {
					fmt.Fprintf(&b, "screenshot: %s\n", st.Shot)
				}
				if st.Output != "" {
					fmt.Fprintf(&b, "output (last lines):\n%s\n", st.Output)
				}
				msg := st.Text + ": " + st.Found
				if st.Expected != "" {
					msg = st.Text + ": expected " + st.Expected + ", found " + st.Found
				}
				tc.Failure = &failure{Message: msg, Type: "StepFailed", Text: b.String()}
			}
			for _, e := range f.Errors {
				fmt.Fprintf(&log, "console: %s\n", e)
			}
			for _, e := range f.Requests {
				fmt.Fprintf(&log, "request: %s\n", e)
			}
			if f.Status == "skipped" {
				tc.Skipped = &skipped{}
			}
			tc.SystemOut = log.String()
			s.Cases = append(s.Cases, tc)
		}
		all.Suites = append(all.Suites, s)
		all.Tests += s.Tests
		all.Failures += s.Failures
		total += r.Seconds
	}
	all.Time = fmt.Sprintf("%.3f", total)
	b, err := xml.MarshalIndent(all, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), append(b, '\n')...), nil
}
