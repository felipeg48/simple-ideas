// Package studio is the backend of `si --web`, a local browser editor that
// can write files and run shells. The gateway in this file is its security
// perimeter: every request passes through it before any handler sees it.
package studio

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"path"
	"strings"
)

// CookieName is the session cookie set after a successful token login.
const CookieName = "si_studio"

const assetsPrefix = "/_si/studio/assets/"

// Gateway guards a handler with a Host allowlist, a launch token exchanged
// for an HttpOnly cookie, and an Origin check on state-changing requests.
type Gateway struct {
	token string
	hosts map[string]bool
}

// NewGateway allows requests whose Host is one of hosts (host:port).
func NewGateway(token string, hosts []string) *Gateway {
	g := &Gateway{token: token, hosts: map[string]bool{}}
	for _, h := range hosts {
		g.hosts[h] = true
	}
	return g
}

// NewToken returns 32 random bytes as 64 hex characters.
func NewToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err) // no entropy: refuse to run rather than use a weak token
	}
	return hex.EncodeToString(b)
}

func (g *Gateway) equal(s string) bool {
	return subtle.ConstantTimeCompare([]byte(s), []byte(g.token)) == 1
}

// Wrap enforces, in order: Host allowlist (DNS rebinding), public assets,
// token login (cookie then redirect without the token), cookie required,
// same-origin check for non-GET/HEAD.
func (g *Gateway) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !g.hosts[r.Host] {
			http.Error(w, "forbidden host", http.StatusForbidden)
			return
		}
		// unclean paths (../) never take the bypass: next might clean them
		if strings.HasPrefix(r.URL.Path, assetsPrefix) && path.Clean(r.URL.Path) == r.URL.Path {
			next.ServeHTTP(w, r)
			return
		}
		if t := r.URL.Query().Get("token"); t != "" && g.equal(t) {
			http.SetCookie(w, &http.Cookie{Name: CookieName, Value: g.token, Path: "/",
				HttpOnly: true, SameSite: http.SameSiteStrictMode})
			q := r.URL.Query()
			q.Del("token")
			u := *r.URL
			u.RawQuery = q.Encode()
			http.Redirect(w, r, u.RequestURI(), http.StatusSeeOther)
			return
		}
		if c, err := r.Cookie(CookieName); err != nil || !g.equal(c.Value) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			if o := r.Header.Get("Origin"); o != "" && o != "http://"+r.Host {
				http.Error(w, "forbidden origin", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
