package publish

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLinkedInDocumentPost(t *testing.T) {
	pdf := []byte("%PDF-1.7 fake")
	var srv *httptest.Server
	var post map[string]any
	steps := []string{}
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		steps = append(steps, r.Method+" "+r.URL.Path)
		if r.URL.Path != "/upload" {
			if r.Header.Get("Authorization") != "Bearer tok-123" || r.Header.Get("X-Restli-Protocol-Version") != "2.0.0" || r.Header.Get("Linkedin-Version") == "" {
				t.Errorf("%s: headers %v", r.URL.Path, r.Header)
			}
		}
		switch {
		case r.URL.Path == "/rest/documents" && r.URL.Query().Get("action") == "initializeUpload":
			var b map[string]map[string]string
			json.NewDecoder(r.Body).Decode(&b)
			if b["initializeUploadRequest"]["owner"] != "urn:li:person:abc" {
				t.Errorf("owner: %v", b)
			}
			json.NewEncoder(w).Encode(map[string]any{"value": map[string]string{"uploadUrl": srv.URL + "/upload", "document": "urn:li:document:D1"}})
		case r.URL.Path == "/upload":
			got, _ := io.ReadAll(r.Body)
			if !bytes.Equal(got, pdf) || r.Method != "PUT" {
				t.Errorf("upload: %s %q", r.Method, got)
			}
			w.WriteHeader(201)
		case r.URL.Path == "/rest/posts":
			json.NewDecoder(r.Body).Decode(&post)
			w.Header().Set("x-restli-id", "urn:li:share:42")
			w.WriteHeader(201)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	l := &LinkedIn{API: srv.URL, Auth: srv.URL, Token: "tok-123", Person: "urn:li:person:abc"}
	id, err := l.Publish(context.Background(), Post{Text: "New talk: #retries (and more)", PDF: pdf, DocTitle: "Talk.pdf"}, nil)
	if err != nil || id != "urn:li:share:42" {
		t.Fatalf("publish: %q %v", id, err)
	}
	if strings.Join(steps, ",") != "POST /rest/documents,PUT /upload,POST /rest/posts" {
		t.Errorf("steps: %v", steps)
	}
	media := post["content"].(map[string]any)["media"].(map[string]any)
	if media["id"] != "urn:li:document:D1" || media["title"] != "Talk.pdf" || post["author"] != "urn:li:person:abc" || post["lifecycleState"] != "PUBLISHED" {
		t.Errorf("post: %v", post)
	}
	if post["commentary"] != `New talk: #retries \(and more\)` {
		t.Errorf("commentary not escaped for little format: %q", post["commentary"])
	}
}

func TestLinkedInLinkPostAndErrors(t *testing.T) {
	var post map[string]any
	fail := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if fail {
			w.WriteHeader(403)
			w.Write([]byte(`{"status":403,"code":"ACCESS_DENIED"}`))
			return
		}
		json.NewDecoder(r.Body).Decode(&post)
		w.Header().Set("x-restli-id", "urn:li:share:7")
		w.WriteHeader(201)
	}))
	defer srv.Close()
	l := &LinkedIn{API: srv.URL, Token: "t", Person: "urn:li:person:x"}
	if _, err := l.Publish(context.Background(), Post{Text: "Read it", URL: "https://example.com/a", LinkTitle: "A", LinkDesc: "d"}, nil); err != nil {
		t.Fatal(err)
	}
	art := post["content"].(map[string]any)["article"].(map[string]any)
	if art["source"] != "https://example.com/a" || art["title"] != "A" || art["description"] != "d" {
		t.Errorf("article: %v", art)
	}
	fail = true
	if _, err := l.Publish(context.Background(), Post{Text: "x"}, nil); err == nil || !strings.Contains(err.Error(), "ACCESS_DENIED") {
		t.Errorf("want ACCESS_DENIED, got %v", err)
	}
}

func TestLinkedInLinkPostThumbnail(t *testing.T) {
	png := []byte("\x89PNG fake")
	var srv *httptest.Server
	var post map[string]any
	steps := []string{}
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		steps = append(steps, r.Method+" "+r.URL.Path)
		switch {
		case r.URL.Path == "/rest/images" && r.URL.Query().Get("action") == "initializeUpload":
			json.NewEncoder(w).Encode(map[string]any{"value": map[string]string{"uploadUrl": srv.URL + "/up", "image": "urn:li:image:I9"}})
		case r.URL.Path == "/up":
			got, _ := io.ReadAll(r.Body)
			if !bytes.Equal(got, png) || r.Header.Get("Content-Type") != "image/png" {
				t.Errorf("upload: %q %s", got, r.Header.Get("Content-Type"))
			}
			w.WriteHeader(201)
		case r.URL.Path == "/rest/posts":
			json.NewDecoder(r.Body).Decode(&post)
			w.Header().Set("x-restli-id", "urn:li:share:8")
			w.WriteHeader(201)
		}
	}))
	defer srv.Close()
	l := &LinkedIn{API: srv.URL, Token: "t", Person: "urn:li:person:x"}
	if _, err := l.Publish(context.Background(), Post{Text: "Read", URL: "https://example.com/a", LinkTitle: "A", Thumb: png, ThumbName: "cover.png", ThumbType: "image/png"}, nil); err != nil {
		t.Fatal(err)
	}
	if strings.Join(steps, ",") != "POST /rest/images,PUT /up,POST /rest/posts" {
		t.Errorf("steps: %v", steps)
	}
	if art := post["content"].(map[string]any)["article"].(map[string]any); art["thumbnail"] != "urn:li:image:I9" {
		t.Errorf("thumbnail: %v", art)
	}
	var dry bytes.Buffer
	if _, err := l.Publish(context.Background(), Post{Text: "Read", URL: "https://example.com/a", ThumbName: "cover.png"}, &dry); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(dry.String(), "/rest/images?action=initializeUpload") || !strings.Contains(dry.String(), `"thumbnail"`) {
		t.Errorf("dry run does not show the picture:\n%s", dry.String())
	}
}

func TestDryRunSendsNothing(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits++ }))
	defer srv.Close()
	var out bytes.Buffer
	l := &LinkedIn{API: srv.URL, Token: "secret-token-9876", Person: "urn:li:person:x"}
	if _, err := l.Publish(context.Background(), Post{Text: "hi", PDF: []byte("12345"), DocTitle: "d.pdf"}, &out); err != nil {
		t.Fatal(err)
	}
	d := &DevTo{API: srv.URL, Key: "devkey-5555"}
	if _, err := d.Publish(context.Background(), Article{Title: "T", Body: "b"}, &out); err != nil {
		t.Fatal(err)
	}
	h := &Hashnode{API: srv.URL, Token: "hn-1111"}
	if _, err := h.Publish(context.Background(), "p1", Article{Title: "T", Body: "b"}, &out); err != nil {
		t.Fatal(err)
	}
	s := out.String()
	if hits != 0 {
		t.Errorf("a dry run sent %d requests", hits)
	}
	for _, want := range []string{"POST " + srv.URL + "/rest/documents?action=initializeUpload", "PUT https://www.linkedin.com/dms-uploads/", "<5 bytes of application/pdf>", "POST " + srv.URL + "/rest/posts", "Bearer ****9876", "****5555", "****1111", `"body_markdown": "b"`} {
		if !strings.Contains(s, want) {
			t.Errorf("dry run output lacks %q:\n%s", want, s)
		}
	}
	for _, secret := range []string{"secret-token-9876", "devkey-5555", "hn-1111"} {
		if strings.Contains(s, secret) {
			t.Errorf("dry run shows the secret %q", secret)
		}
	}
}

func TestDevTo(t *testing.T) {
	var body map[string]map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/articles" || r.Header.Get("Api-Key") != "k1" {
			t.Errorf("%s key=%q", r.URL.Path, r.Header.Get("Api-Key"))
		}
		json.NewDecoder(r.Body).Decode(&body)
		w.WriteHeader(201)
		w.Write([]byte(`{"id":1,"url":"https://dev.to/me/retry"}`))
	}))
	defer srv.Close()
	u, err := (&DevTo{API: srv.URL, Key: "k1"}).Publish(context.Background(), Article{Title: "Retry", Body: "# x", Tags: []string{"go", "sql", "a", "b", "c"}, Canonical: "https://me.dev/retry"}, nil)
	if err != nil || u != "https://dev.to/me/retry" {
		t.Fatalf("%q %v", u, err)
	}
	a := body["article"]
	if a["tags"] != "go,sql,a,b" || a["published"] != false || a["canonical_url"] != "https://me.dev/retry" || a["body_markdown"] != "# x" {
		t.Errorf("article: %v", a)
	}
}

func TestHashnode(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Authorization") != "pat" {
			t.Errorf("auth %q", r.Header.Get("Authorization"))
		}
		var q struct {
			Query     string
			Variables map[string]any
		}
		json.NewDecoder(r.Body).Decode(&q)
		switch {
		case strings.Contains(q.Query, "publication(host"):
			w.Write([]byte(`{"data":{"publication":{"id":"pub9"}}}`))
		case strings.Contains(q.Query, "publishPost"):
			in := q.Variables["input"].(map[string]any)
			if in["publicationId"] != "pub9" || in["contentMarkdown"] != "body" || in["tags"].([]any)[0].(map[string]any)["slug"] != "dot-net" {
				t.Errorf("input %v", in)
			}
			w.Write([]byte(`{"data":{"publishPost":{"post":{"id":"1","url":"https://blog.me/x"}}}}`))
		default:
			w.Write([]byte(`{"errors":[{"message":"bad"}]}`))
		}
	}))
	defer srv.Close()
	h := &Hashnode{API: srv.URL, Token: "pat"}
	id, err := h.Publication(context.Background(), "blog.me", nil)
	if err != nil || id != "pub9" {
		t.Fatalf("%q %v", id, err)
	}
	u, err := h.Publish(context.Background(), id, Article{Title: "T", Body: "body", Tags: []string{"Dot NET"}}, nil)
	if err != nil || u != "https://blog.me/x" {
		t.Fatalf("%q %v", u, err)
	}
	if err := h.query(context.Background(), "other", nil, nil, "", &struct{}{}); err == nil || !strings.Contains(err.Error(), "bad") {
		t.Errorf("graphql error not surfaced: %v", err)
	}
}

func TestOAuth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/v2/accessToken":
			r.ParseForm()
			if r.Form.Get("code") != "c1" || r.Form.Get("client_secret") != "s" || r.Form.Get("grant_type") != "authorization_code" {
				t.Errorf("form %v", r.Form)
			}
			w.Write([]byte(`{"access_token":"at","expires_in":5184000}`))
		case "/v2/userinfo":
			w.Write([]byte(`{"sub":"m42"}`))
		}
	}))
	defer srv.Close()
	l := &LinkedIn{API: srv.URL, Auth: srv.URL}
	if u := l.AuthURL("cid", "http://localhost:8723/callback", "st"); !strings.Contains(u, "scope=openid+profile+w_member_social") || !strings.Contains(u, "state=st") {
		t.Errorf("auth url %s", u)
	}
	tok, exp, err := l.Exchange(context.Background(), "cid", "s", "c1", "http://localhost:8723/callback")
	if err != nil || tok != "at" || exp != 5184000 {
		t.Fatalf("%q %d %v", tok, exp, err)
	}
	l.Token = tok
	if p, err := l.Me(context.Background()); err != nil || p != "urn:li:person:m42" {
		t.Fatalf("%q %v", p, err)
	}
	var code string
	h := Callback("st", func(c string, err error) { code = c })
	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest("GET", "/callback?state=evil&code=x", nil))
	if rec.Code != 400 || code != "" {
		t.Errorf("a wrong state was accepted: %d %q", rec.Code, code)
	}
	h(httptest.NewRecorder(), httptest.NewRequest("GET", "/callback?state=st&code=good", nil))
	if code != "good" {
		t.Errorf("code %q", code)
	}
}

func TestStorePermissions(t *testing.T) {
	p := filepath.Join(t.TempDir(), "cfg", "publish.json")
	t.Setenv("SI_PUBLISH_CONFIG", p)
	var c Creds
	c.DevTo.Key = "k"
	if err := Save(c); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(p)
	if err != nil || st.Mode().Perm() != 0o600 {
		t.Fatalf("mode %v %v", st.Mode(), err)
	}
	if ds, _ := os.Stat(filepath.Dir(p)); ds.Mode().Perm() != 0o700 {
		t.Errorf("dir mode %v", ds.Mode())
	}
	got, err := Load()
	if err != nil || got.DevTo.Key != "k" {
		t.Errorf("load %v %v", got, err)
	}
}
