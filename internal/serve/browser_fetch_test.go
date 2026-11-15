package serve

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/felipeg48/simple-ideas/internal/cdp"
	"github.com/felipeg48/simple-ideas/internal/tools"
)

// apiServer is the other site a framed page calls: no CORS headers at all,
// so only a same-origin request (through the proxy) can read its answers.
func apiServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/moved" {
			http.Redirect(w, r, "/api", http.StatusFound)
			return
		}
		body, _ := io.ReadAll(r.Body)
		http.SetCookie(w, &http.Cookie{Name: "api", Value: "1"})
		fmt.Fprintf(w, "%s %s body=%s cookie=%q origin=%s", r.Method, r.URL.Path, body, r.Header.Get("Cookie"), r.Header.Get("Origin"))
	}))
}

// TestBrowserProxyFetchThrough: fwdPath?x=1 sends a request on in place (no
// redirect), with its method and body, as the framed site's origin, never
// with this proxy's cookies, and drops the other site's Set-Cookie; its
// redirects come back through fwdPath.
func TestBrowserProxyFetchThrough(t *testing.T) {
	api := apiServer()
	defer api.Close()
	page := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "page") }))
	defer page.Close()
	var b browserProxies
	defer b.closeAll()
	u, _ := url.Parse(page.URL + "/")
	proxy, err := b.open(u)
	if err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequest("POST", proxy+fwdPath+"?x=1&to="+url.QueryEscape(api.URL+"/api"), strings.NewReader(`{"n":1}`))
	req.Header.Set("Cookie", "session=framed-site")
	req.Header.Set("Origin", proxy)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if want := `POST /api body={"n":1} cookie="" origin=` + page.URL; string(body) != want {
		t.Errorf("sent on: %q, want %q", body, want)
	}
	if res.Header.Get("Set-Cookie") != "" || res.Request.URL.String() != req.URL.String() {
		t.Errorf("a cookie came back (%q) or it was redirected (%s)", res.Header.Get("Set-Cookie"), res.Request.URL)
	}
	res, err = http.Get(proxy + fwdPath + "?x=1&to=" + url.QueryEscape(api.URL+"/moved"))
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(res.Body)
	res.Body.Close()
	if !strings.HasPrefix(string(body), "GET /api ") || !strings.HasPrefix(res.Request.URL.String(), proxy+fwdPath+"?x=1&to=") {
		t.Errorf("a redirect: %q at %s", body, res.Request.URL)
	}
}

// TestBrowserProxyFetchInChrome: a page framed through its proxy calls
// another origin (no CORS headers there) with fetch (a string, a Request
// with a body) and XMLHttpRequest, and reads every answer.
func TestBrowserProxyFetchInChrome(t *testing.T) {
	chrome := tools.Find("chrome")
	if chrome == "" || testing.Short() {
		t.Skip("needs Chrome")
	}
	api := apiServer()
	defer api.Close()
	a, _ := json.Marshal(api.URL)
	page := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<!doctype html><html><head><title>p</title></head><body><script>var A=%s;
Promise.all([
  fetch(A+'/api',{credentials:'include'}).then(function(r){return r.text()}),
  fetch(new Request(A+'/api',{method:'POST',body:'req-body'})).then(function(r){return r.text()}),
  new Promise(function(ok,no){var x=new XMLHttpRequest();x.open('PUT',A+'/api');x.onload=function(){ok(x.responseText)};x.onerror=no;x.send('xhr-body')}),
  fetch(%q).then(function(r){return r.status})
]).then(function(v){parent.postMessage({res:v},'*')},function(e){parent.postMessage({err:String(e)},'*')});
</script></body></html>`, a, "http://"+r.Host+"/same")
	}))
	defer page.Close()
	var b browserProxies
	defer b.closeAll()
	u, _ := url.Parse(page.URL + "/")
	proxy, err := b.open(u)
	if err != nil {
		t.Fatal(err)
	}
	host := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `<!doctype html><script>addEventListener('message',function(e){if(e.data&&(e.data.res||e.data.err))window.got=JSON.stringify(e.data)})</script><iframe src="%s/"></iframe>`, proxy)
	}))
	defer host.Close()
	c, err := cdp.Start(chrome, 800, 600)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	c.Call("Page.enable", nil, nil)
	c.Call("Page.navigate", map[string]any{"url": host.URL + "/"}, nil)
	var got string
	for i := 0; i < 200 && got == ""; i++ {
		time.Sleep(50 * time.Millisecond)
		if v, err := c.Eval(`window.got||''`); err == nil {
			json.Unmarshal(v, &got)
		}
	}
	var out struct {
		Res []any
		Err string
	}
	json.Unmarshal([]byte(got), &out)
	if out.Err != "" || len(out.Res) != 4 {
		t.Fatalf("the framed page's requests: %s", got)
	}
	for i, want := range []string{"GET /api body= cookie=\"\" origin=", "POST /api body=req-body cookie=\"\" origin=" + page.URL, "PUT /api body=xhr-body cookie=\"\" origin=" + page.URL} {
		if s, _ := out.Res[i].(string); !strings.HasPrefix(s, want) {
			t.Errorf("request %d: %q, want %q…", i+1, s, want)
		}
	}
	if out.Res[3] != float64(200) {
		t.Errorf("the framed site's own absolute address: status %v", out.Res[3])
	}
}
