package webdemo

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestSelfReload (issue #14): a page that reloads itself during its first
// load (the RabbitMQ management UI does on a fresh profile) leaves Chrome's
// frameStartedLoading / frameStoppedLoading unpaired; a click after it must
// still wait only for real loads. An iframe that reloads itself too never
// holds a step (inner frames).
func TestSelfReload(t *testing.T) {
	chrome := needChrome(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `<!doctype html><title>Broker</title>
<script>if(!localStorage.getItem('v')){localStorage.setItem('v','1');location.reload()}</script>
<script src="/slow.js"></script>
<iframe src="/frame"></iframe>
<label>Username <input id="u"></label>
<button onclick="document.getElementById('o').textContent='Overview'">Login</button><p id="o"></p>`)
	})
	mux.HandleFunc("/slow.js", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(150 * time.Millisecond)
		io.WriteString(w, "window.slow=1")
	})
	mux.HandleFunc("/frame", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `<script>if(!sessionStorage.getItem('f')){sessionStorage.setItem('f','1');location.reload()}</script><p>preview</p>`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	text := "---\ntype: demo\napp: " + srv.URL + "\ntimeout: 6s\n---\n\n## Sign in\n\n- go /\n- type \"guest\" into \"Username\"\n- click \"Login\"\n- expect text \"Overview\"\n"
	rep, log := runText(t, chrome, text)
	if !rep.OK() || rep.Passed != 1 {
		t.Errorf("a page that reloads itself: %d passed, %d failed\n%s", rep.Passed, rep.Failed, log)
	}
}
