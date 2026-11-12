package webdemo

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/felipeg48/simple-ideas/internal/tools"
)

const windowPage = `<!doctype html><meta charset="utf-8"><title>Board</title>
<style>body{margin:0;font:16px system-ui}#h{position:absolute;left:100px;top:100px;width:80px;height:40px;background:#4a8}</style>
<button id="open" style="position:absolute;left:300px;top:20px">Open notes</button>
<div id="h">Handle</div><p id="out" style="position:absolute;top:400px">idle</p>
<script>
document.getElementById('open').onclick=function(){var w=window.open('','si-presenter-board');w.document.write('<title>Teleprompter</title><h1>Speaker notes</h1>');w.document.close()};
var moves=0,down=false;
addEventListener('pointerdown',function(){down=true;moves=0});
addEventListener('pointermove',function(e){if(down&&e.buttons===1)moves++});
addEventListener('pointerup',function(e){down=false;document.getElementById('out').textContent='dropped at '+Math.round(e.clientX)+','+Math.round(e.clientY)+(moves>10?' smoothly':' jumpily')});
</script>`

func TestWindowAndDrag(t *testing.T) {
	chrome := tools.Find("chrome")
	if chrome == "" || testing.Short() {
		t.Skip("needs Chrome")
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(windowPage)) }))
	defer srv.Close()
	script := filepath.Join(t.TempDir(), "w.md")
	os.WriteFile(script, []byte("---\ntype: demo\napp: "+srv.URL+"\nviewport: 800x500\n---\n\n## Notes and drawing\n\n"+
		"- click \"Open notes\"\n- window \"si-presenter\"\n- expect text \"Speaker notes\"\n- window main\n- expect text \"Open notes\"\n"+
		"- window 2 · pop\n- window \"Teleprompter\"\n- expect text \"Speaker notes\"\n- window 1\n"+
		"- spotlight \"Handle\" · circle\n- zoom to \"Handle\"\n- zoom out\n"+
		"- drag from \"Handle\" to 600,300\n- expect text \"dropped at 600,300 smoothly\"\n"+
		"- drag by 10,-20\n- expect text \"dropped at 610,280\"\n- spotlight off\n"), 0o644)
	s, err := ParseFile(script)
	if err != nil || len(s.Problems) > 0 {
		t.Fatal(err, s.Problems)
	}
	rep, err := Run(s, Options{Chrome: chrome})
	if err != nil {
		t.Fatal(err)
	}
	for _, st := range rep.Features[0].Steps {
		if st.Status != "passed" {
			t.Errorf("%s: %s (expected %s; found %s)", st.Text, st.Status, st.Expected, st.Found)
		}
	}
	// a window that is not there fails with what was seen
	os.WriteFile(script, []byte("---\ntype: demo\napp: "+srv.URL+"\ntimeout: 1s\n---\n\n## Missing\n\n- click \"Open notes\"\n- window \"nowhere\"\n"), 0o644)
	s, _ = ParseFile(script)
	rep, err = Run(s, Options{Chrome: chrome})
	if err != nil {
		t.Fatal(err)
	}
	if st := rep.Features[0].Steps[1]; st.Status != "failed" || len(st.Candidates) == 0 {
		t.Errorf("missing window: %+v", st)
	}
}
