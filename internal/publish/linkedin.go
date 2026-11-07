package publish

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// LinkedInVersion is the Linkedin-Version header (YYYYMM). LinkedIn retires a
// version about a year after it ships; $SI_LINKEDIN_VERSION overrides it.
const LinkedInVersion = "202610"

// LinkedInScopes are what si asks for: the member id (openid profile) and
// posting on the member's behalf (w_member_social, "Share on LinkedIn").
const LinkedInScopes = "openid profile w_member_social"

// LinkedIn posts as one member (Person, urn:li:person:<id>) with Token.
type LinkedIn struct {
	API, Auth     string // https://api.linkedin.com, https://www.linkedin.com (tests point them at a fake)
	Token, Person string
}

// NewLinkedIn is a client for the real LinkedIn.
func NewLinkedIn(token, person string) *LinkedIn {
	return &LinkedIn{API: "https://api.linkedin.com", Auth: "https://www.linkedin.com", Token: token, Person: person}
}

func (l *LinkedIn) sign(req *http.Request) {
	v := os.Getenv("SI_LINKEDIN_VERSION")
	if v == "" {
		v = LinkedInVersion
	}
	req.Header.Set("Authorization", "Bearer "+l.Token)
	req.Header.Set("Linkedin-Version", v)
	req.Header.Set("X-Restli-Protocol-Version", "2.0.0")
}

// Post is one LinkedIn post: Text always; PDF (a deck) makes it a document
// post; URL (an article) a link post.
type Post struct {
	Text      string
	PDF       []byte
	DocTitle  string // shown on the document, e.g. "The Thursday Promise.pdf"
	URL       string
	LinkTitle string
	LinkDesc  string
}

// Publish creates the post and returns its id (urn:li:share:…). With dry != nil
// it only prints the requests.
func (l *LinkedIn) Publish(ctx context.Context, p Post, dry io.Writer) (string, error) {
	body := map[string]any{
		"author":     l.Person,
		"commentary": escapeLittle(p.Text),
		"visibility": "PUBLIC",
		"distribution": map[string]any{
			"feedDistribution":               "MAIN_FEED",
			"targetEntities":                 []any{},
			"thirdPartyDistributionChannels": []any{},
		},
		"lifecycleState":            "PUBLISHED",
		"isReshareDisabledByAuthor": false,
	}
	switch {
	case p.PDF != nil || p.DocTitle != "":
		doc, err := l.uploadDocument(ctx, p.PDF, dry)
		if err != nil {
			return "", err
		}
		body["content"] = map[string]any{"media": map[string]any{"title": p.DocTitle, "id": doc}}
	case p.URL != "":
		art := map[string]any{"source": p.URL, "title": p.LinkTitle}
		if p.LinkDesc != "" {
			art["description"] = p.LinkDesc
		}
		body["content"] = map[string]any{"article": art}
	}
	req, raw, err := jsonReq("POST", l.API+"/rest/posts", body)
	if err != nil {
		return "", err
	}
	l.sign(req)
	_, h, err := do(ctx, req, raw, dry, "")
	if err != nil {
		return "", err
	}
	return h.Get("X-Restli-Id"), nil
}

// uploadDocument registers a document (initializeUpload), PUTs the PDF to the
// upload URL and returns the document URN.
func (l *LinkedIn) uploadDocument(ctx context.Context, pdf []byte, dry io.Writer) (string, error) {
	req, raw, err := jsonReq("POST", l.API+"/rest/documents?action=initializeUpload",
		map[string]any{"initializeUploadRequest": map[string]any{"owner": l.Person}})
	if err != nil {
		return "", err
	}
	l.sign(req)
	out, _, err := do(ctx, req, raw, dry, `{"value":{"uploadUrl":"https://www.linkedin.com/dms-uploads/upload-url-from-initializeUpload","document":"urn:li:document:from-initializeUpload"}}`)
	if err != nil {
		return "", err
	}
	var init struct {
		Value struct {
			UploadURL string `json:"uploadUrl"`
			Document  string `json:"document"`
		} `json:"value"`
	}
	if err := json.Unmarshal(out, &init); err != nil || init.Value.UploadURL == "" {
		return "", fmt.Errorf("linkedin: no upload URL in %s", strings.TrimSpace(string(out)))
	}
	put, err := http.NewRequest("PUT", init.Value.UploadURL, nil)
	if err != nil {
		return "", err
	}
	put.Header.Set("Authorization", "Bearer "+l.Token)
	put.Header.Set("Content-Type", "application/pdf")
	if _, _, err := do(ctx, put, pdf, dry, ""); err != nil {
		return "", err
	}
	return init.Value.Document, nil
}

// escapeLittle escapes the characters LinkedIn's "little" text format reads
// as markup, so a commentary shows as written (# hashtags stay hashtags).
func escapeLittle(s string) string {
	r := strings.NewReplacer(`\`, `\\`, "|", `\|`, "{", `\{`, "}", `\}`, "@", `\@`, "[", `\[`, "]", `\]`, "(", `\(`, ")", `\)`, "<", `\<`, ">", `\>`, "*", `\*`, "_", `\_`, "~", `\~`)
	return r.Replace(s)
}

// Me looks up the member id of the token (needs the openid and profile scopes).
func (l *LinkedIn) Me(ctx context.Context) (string, error) {
	req, err := http.NewRequest("GET", l.API+"/v2/userinfo", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+l.Token)
	out, _, err := do(ctx, req, nil, nil, "")
	if err != nil {
		return "", err
	}
	var u struct {
		Sub string `json:"sub"`
	}
	if err := json.Unmarshal(out, &u); err != nil || u.Sub == "" {
		return "", fmt.Errorf("linkedin: no member id in /v2/userinfo (the token needs the openid and profile scopes)")
	}
	return "urn:li:person:" + u.Sub, nil
}

// NewState is a random OAuth state.
func NewState() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// AuthURL is the consent page the member opens in a browser.
func (l *LinkedIn) AuthURL(clientID, redirect, state string) string {
	q := url.Values{"response_type": {"code"}, "client_id": {clientID}, "redirect_uri": {redirect}, "state": {state}, "scope": {LinkedInScopes}}
	return l.Auth + "/oauth/v2/authorization?" + q.Encode()
}

// Exchange trades the authorization code for an access token and its lifetime.
func (l *LinkedIn) Exchange(ctx context.Context, clientID, secret, code, redirect string) (token string, expiresIn int64, err error) {
	form := url.Values{"grant_type": {"authorization_code"}, "code": {code}, "client_id": {clientID}, "client_secret": {secret}, "redirect_uri": {redirect}}
	req, err := http.NewRequest("POST", l.Auth+"/oauth/v2/accessToken", nil)
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	out, _, err := do(ctx, req, []byte(form.Encode()), nil, "")
	if err != nil {
		return "", 0, err
	}
	var t struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(out, &t); err != nil || t.AccessToken == "" {
		return "", 0, fmt.Errorf("linkedin: no access token in the answer")
	}
	return t.AccessToken, t.ExpiresIn, nil
}

// Callback is the loopback redirect handler: it checks state and hands the
// code (or the error LinkedIn sent) to got, once.
func Callback(state string, got func(code string, err error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("state") != state {
			http.Error(w, "si: the login answer does not match this login (state); start again", http.StatusBadRequest)
			return
		}
		if e := q.Get("error"); e != "" {
			fmt.Fprintf(w, "si: LinkedIn said %s: %s. You can close this tab.", e, q.Get("error_description"))
			got("", fmt.Errorf("linkedin: %s: %s", e, q.Get("error_description")))
			return
		}
		fmt.Fprint(w, "si is logged in to LinkedIn. You can close this tab.")
		got(q.Get("code"), nil)
	}
}
