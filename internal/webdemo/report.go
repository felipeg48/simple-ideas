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
	Chrome   string          `json:"chrome"`
	SI       string          `json:"si"`
	Date     string          `json:"date"`
	Seconds  float64         `json:"seconds"`
	Passed   int             `json:"passed"`
	Failed   int             `json:"failed"`
	Skipped  int             `json:"skipped"`
	Features []FeatureResult `json:"features"`
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

// OK reports whether every feature passed.
func (r *Report) OK() bool { return r.Failed == 0 }

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
}

func (r *Report) view(dir string) any {
	var fs []featureView
	for _, f := range r.Features {
		fv := featureView{FeatureResult: f}
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
	return struct {
		*Report
		Views []featureView
	}{r, fs}
}

var reportTmpl = template.Must(template.New("r").Funcs(template.FuncMap{
	"secs": func(f float64) string { return fmt.Sprintf("%.2fs", f) },
}).Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Title}}: web demo report</title>
<style>
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
.log{margin:.8em 0 0;color:var(--bad)}
.log ul{margin:.2em 0 0;padding-left:1.2em}
</style></head><body><main>
<h1>{{.Title}}</h1>
<div class="meta"><span>{{.Script}}</span><span>{{.App}}</span><span>{{.Viewport}}</span><span>{{.Chrome}}</span><span>si {{.SI}}</span><span>{{.Date}}</span></div>
<div class="sum"><span class="pill passed">{{.Passed}} passed</span><span class="pill failed">{{.Failed}} failed</span>{{if .Skipped}}<span class="pill skipped">{{.Skipped}} skipped</span>{{end}}<span class="pill">{{secs .Seconds}}</span></div>
{{range .Views}}<section class="st-{{.Status}}">
<header><h2>{{.Name}}</h2><span class="{{.Status}}">{{.Status}} · {{secs .Seconds}}</span></header>
{{if .What}}<p class="what">{{.What}}</p>{{end}}
<ol>{{range .Steps}}<li>
<div class="row"><span class="mark {{.Status}}">{{if eq .Status "passed"}}✓{{else if eq .Status "failed"}}✗{{else}}–{{end}}</span><code>{{.Text}}</code><span class="t">{{if .Line}}line {{.Line}} · {{end}}{{if ne .Status "skipped"}}{{secs .Seconds}}{{else}}skipped{{end}}</span></div>
{{if eq .Status "failed"}}<dl>{{if .Expected}}<dt>expected</dt><dd>{{.Expected}}</dd>{{end}}<dt>found</dt><dd>{{.Found}}</dd>{{if .Candidates}}<dt>closest</dt><dd>{{range $i, $c := .Candidates}}{{if $i}}, {{end}}<code>{{$c}}</code>{{end}}</dd>{{end}}</dl>{{end}}
{{if .Img}}<figure><img src="{{.Img}}" alt="the page after: {{.Text}}"><figcaption>{{.Shot}}</figcaption></figure>{{end}}
</li>{{end}}</ol>
{{if or .Errors .Requests}}<div class="log">{{if .Errors}}<strong>Console errors</strong><ul>{{range .Errors}}<li><code>{{.}}</code></li>{{end}}</ul>{{end}}{{if .Requests}}<strong>Failed requests</strong><ul>{{range .Requests}}<li><code>{{.}}</code></li>{{end}}</ul>{{end}}</div>{{end}}
</section>{{end}}
</main></body></html>
`))

// JUnit is the report as JUnit XML: one test case per feature.
func (r *Report) JUnit() ([]byte, error) {
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
	s := suite{Name: r.Title, Tests: len(r.Features), Failures: r.Failed, Skipped: r.Skipped, Time: fmt.Sprintf("%.3f", r.Seconds), Stamp: r.Date}
	for _, f := range r.Features {
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
	b, err := xml.MarshalIndent(suites{Name: r.Title, Tests: s.Tests, Failures: s.Failures, Time: s.Time, Suites: []suite{s}}, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), append(b, '\n')...), nil
}
