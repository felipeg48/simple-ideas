package webdemo

import (
	"encoding/json"
	"encoding/xml"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/felipeg48/simple-ideas/internal/tools"
)

func sampleReport() *Report {
	r := &Report{Script: "shop.md", Title: "Shop <&>", App: "http://127.0.0.1:1", Viewport: "1440x900", SI: "test", Seconds: 3.5,
		Features: []FeatureResult{
			{Name: "Sign in", Status: "passed", Seconds: 1, Steps: []StepResult{{Line: 3, Text: `go /`, Status: "passed"}}},
			{Name: "Pay", What: "a customer pays", Status: "failed", Seconds: 2.5, Errors: []string{"console.error: boom"},
				Steps: []StepResult{
					{Line: 9, Text: `click "Pay"`, Status: "failed", Expected: `an element "Pay"`, Found: "nothing matches", Candidates: []string{`button "Pay now"`}, Shot: "shots/01-pay-failed.png"},
					{Line: 10, Text: `expect text "Paid"`, Status: "skipped"},
				}},
		}}
	r.count()
	return r
}

func TestJUnit(t *testing.T) {
	b, err := sampleReport().JUnit()
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Tests    int `xml:"tests,attr"`
		Failures int `xml:"failures,attr"`
		Suite    struct {
			Cases []struct {
				Name    string `xml:"name,attr"`
				Failure *struct {
					Message string `xml:"message,attr"`
					Text    string `xml:",chardata"`
				} `xml:"failure"`
			} `xml:"testcase"`
		} `xml:"testsuite"`
	}
	if err := xml.Unmarshal(b, &doc); err != nil {
		t.Fatalf("%v\n%s", err, b)
	}
	if doc.Tests != 2 || doc.Failures != 1 || len(doc.Suite.Cases) != 2 || doc.Suite.Cases[0].Failure != nil {
		t.Fatalf("junit: %s", b)
	}
	f := doc.Suite.Cases[1].Failure
	if f == nil || !strings.Contains(f.Message, `expected an element "Pay", found nothing matches`) || !strings.Contains(f.Text, "shop.md:9:") || !strings.Contains(f.Text, "screenshot: shots/01-pay-failed.png") {
		t.Errorf("failure: %+v", f)
	}
}

func TestReportWrite(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "shots"), 0o755)
	os.WriteFile(filepath.Join(dir, "shots", "01-pay-failed.png"), []byte("png"), 0o644)
	r := sampleReport()
	if err := r.Write(dir); err != nil {
		t.Fatal(err)
	}
	page, _ := os.ReadFile(filepath.Join(dir, "report.html"))
	for _, want := range []string{"Shop &lt;&amp;&gt;", "1 passed", "1 failed", `data:image/png;base64,cG5n`, "nothing matches", "console.error: boom", `button &#34;Pay now&#34;`, "prefers-color-scheme"} {
		if !strings.Contains(string(page), want) {
			t.Errorf("report.html lacks %q", want)
		}
	}
	var back Report
	raw, _ := os.ReadFile(filepath.Join(dir, "report.json"))
	if err := json.Unmarshal(raw, &back); err != nil || back.Failed != 1 || back.Features[1].Steps[0].Candidates[0] != `button "Pay now"` {
		t.Errorf("report.json: %v %s", err, raw)
	}
}

// TestExamples runs examples/webdemo in headless Chrome: shop.md passes,
// broken.md fails at its expectation with what it found and a screenshot.
func TestExamples(t *testing.T) {
	chrome := tools.Find("chrome")
	if chrome == "" {
		t.Skip("no Chrome")
	}
	if testing.Short() {
		t.Skip("short")
	}
	t.Setenv("SHOP_PASSWORD", "shop-demo")
	var log strings.Builder
	run := func(name string) *Report {
		s, err := ParseFile(filepath.Join("..", "..", "examples", "webdemo", name))
		if err != nil || len(s.Problems) > 0 {
			t.Fatalf("%s: %v %v", name, err, s.Problems)
		}
		out := t.TempDir()
		rep, err := Run(s, Options{Chrome: chrome, Out: out, Log: &log})
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if err := rep.Write(out); err != nil {
			t.Fatal(err)
		}
		for _, f := range rep.Features {
			for i, st := range f.Steps {
				if st.Shot != "" {
					if _, err := os.Stat(filepath.Join(out, st.Shot)); err != nil {
						t.Errorf("%s: step %d screenshot: %v", name, i, err)
					}
				}
			}
		}
		return rep
	}
	shop := run("shop.md")
	if !shop.OK() || shop.Passed != 3 {
		t.Errorf("shop.md: %d passed, %d failed\n%s", shop.Passed, shop.Failed, log.String())
	}
	if strings.Contains(log.String(), "shop-demo") {
		t.Error("the password was printed")
	}
	// broker.md: a broker started by start:, terminals, a workload; go run
	// builds the example's broker
	if _, err := exec.LookPath("go"); err == nil && runtime.GOOS != "windows" {
		broker := run("broker.md")
		if !broker.OK() || broker.Passed != 3 {
			t.Errorf("broker.md: %d passed, %d failed\n%s", broker.Passed, broker.Failed, log.String())
		}
		if portOpen(7310) || portOpen(7311) {
			t.Error("a broker outlived the run")
		}
	}
	broken := run("broken.md")
	if broken.OK() || len(broken.Features) != 1 {
		t.Fatalf("broken.md passed\n%s", log.String())
	}
	f := broken.Features[0]
	var failed *StepResult
	for i := range f.Steps {
		if f.Steps[i].Status == "failed" {
			failed = &f.Steps[i]
			break
		}
	}
	if failed == nil || failed.Text != `expect text "10% off"` || failed.Line != 23 || failed.Expected != `the page shows "10% off"` ||
		!strings.Contains(failed.Found, "That coupon is not valid.") || failed.Shot == "" {
		t.Errorf("failed step: %+v", failed)
	}
	if f.Steps[len(f.Steps)-1].Status != "skipped" {
		t.Error("the steps after a failure are skipped")
	}
	if len(f.Requests) == 0 || !strings.Contains(f.Requests[0], "404") || len(f.Errors) == 0 {
		t.Errorf("console and requests: %v %v", f.Errors, f.Requests)
	}
}
