package render

import (
	"encoding/json"
	"encoding/xml"
	"io"
	"reflect"
	"strings"
	"testing"
)

func TestPlaceholdersParse(t *testing.T) {
	got := Placeholders("kubectl -n {{ns:messaging}} get pods {{pod}} -n {{ns:other}} {{ns}}")
	want := []Placeholder{{"ns", "messaging"}, {"pod", ""}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestPlaceholdersFirstDefaultWins(t *testing.T) {
	got := Placeholders("{{a}} {{a:x}}")
	want := []Placeholder{{"a", ""}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestPlaceholdersIgnoreTemplatesAndShell(t *testing.T) {
	for _, c := range []string{
		"docker inspect -f '{{.Server.Version}}' x",
		"docker inspect {{json .}}",
		"echo ${HOME}",
		"awk '{print $1}'",
		"jq '{a: .b}'",
		"{{end}}",
		"{{else}}",
		"kubectl get pods -o go-template='{{range .items}}{{.metadata.name}}{{end}}'",
		"docker ps --format '{{if .Ports}}x{{else}}-{{end}}'",
	} {
		if got := Placeholders(c); len(got) != 0 {
			t.Errorf("%q: got %v, want none", c, got)
		}
	}
}

func TestRunbookVarsAndHighlight(t *testing.T) {
	for _, target := range []string{"sql", "bash"} {
		js, err := runbookForPage([]byte(`{"categories":[{"name":"c","entries":[
		 {"label":"a","command":"kubectl -n {{ns:messaging}} get pods {{pod}} {{ns}}","target":"` + target + `"},
		 {"label":"b","command":"ls"}]}]}`))
		if err != nil {
			t.Fatal(err)
		}
		var out struct {
			Categories []struct {
				Entries []map[string]any
			}
		}
		if err := json.Unmarshal([]byte(js), &out); err != nil {
			t.Fatal(err)
		}
		es := out.Categories[0].Entries
		vars, _ := es[0]["vars"].([]any)
		if len(vars) != 2 {
			t.Fatalf("%s: vars = %v", target, es[0]["vars"])
		}
		if v := vars[0].(map[string]any); v["name"] != "ns" || v["default"] != "messaging" {
			t.Errorf("vars[0] = %v", v)
		}
		if n := strings.Count(es[0]["html"].(string), `<span class="si-rb-var">`); n != 3 {
			t.Errorf("%s: %d si-rb-var spans, want 3: %s", target, n, es[0]["html"])
		}
		if _, ok := es[1]["vars"]; ok {
			t.Error("vars must be omitted when empty")
		}
	}
}

func TestRunbookAcceptsMetadata(t *testing.T) {
	js, err := runbookForPage([]byte(`{"name":"N","description":"D","categories":[{"name":"c","entries":[{"label":"a","command":"ls"}]}]}`))
	if err != nil || !strings.Contains(js, `"name":"c"`) {
		t.Fatalf("%v %s", err, js)
	}
	if !strings.Contains(js, `"name":"N"`) {
		t.Errorf("name should surface as top-level name: %s", js)
	}
	js, err = runbookForPage([]byte(`{"categories":[{"name":"c","entries":[{"label":"a","command":"ls"}]}]}`))
	if err != nil || strings.Contains(js, `"name":"N"`) {
		t.Errorf("no metadata: %v %s", err, js)
	}
}

func TestRunbookVarsNestProperly(t *testing.T) {
	cases := []struct{ target, cmd string }{
		{"sql", `SELECT '{{a:x' || '}}'`},
		{"sql", `a = '{{v:it''s}}'`},
		{"bash", `echo "{{a:x" y}}"`},
		{"sql", "-- {{x:foo\n}}"},
		{"sql", "/* {{a:x */ SELECT 1 }} */"},
		{"sql", "SELECT {{select}}"},
		{"bash", "ls {{if}} {{a:<b>&}}"},
		{"bash", "ls {{x}} '{{range .}}{{end}}'"},
	}
	for _, c := range cases {
		h := highlightWithVars(c.cmd, c.target)
		d := xml.NewDecoder(strings.NewReader("<r>" + strings.NewReplacer("&nbsp;", " ").Replace(h) + "</r>"))
		d.Strict = true
		for {
			if _, err := d.Token(); err != nil {
				if err != io.EOF {
					t.Errorf("%s %q: bad markup: %v\n%s", c.target, c.cmd, err, h)
				}
				break
			}
		}
		if want := len(Placeholders(c.cmd)); strings.Count(h, `class="si-rb-var"`) != want {
			t.Errorf("%s %q: want %d wrapped: %s", c.target, c.cmd, want, h)
		}
	}
}
