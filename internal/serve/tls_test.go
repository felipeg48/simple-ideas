package serve

import (
	"crypto/tls"
	"crypto/x509"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The authority is made once (key private to the user), loaded again after,
// and only ever vouches for private addresses: a certificate it signs for a
// public address fails verification, so a leaked key cannot fake a website.
func TestLocalCA(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "ca")
	ca, err := LoadCA(dir)
	if err != nil || !ca.Created {
		t.Fatalf("create: %v", err)
	}
	if fi, _ := os.Stat(filepath.Join(dir, "si-local-ca-key.pem")); fi == nil || fi.Mode().Perm() != 0o600 {
		t.Errorf("the key must be 0600: %v", fi)
	}
	again, err := LoadCA(dir)
	if err != nil || again.Created || again.Fingerprint() != ca.Fingerprint() {
		t.Fatalf("reload: %v", err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(ca.Cert)
	verify := func(host string) error {
		leaf, err := again.Leaf([]string{host})
		if err != nil {
			return err
		}
		c, _ := x509.ParseCertificate(leaf.Certificate[0])
		_, err = c.Verify(x509.VerifyOptions{Roots: pool, DNSName: host})
		return err
	}
	for _, h := range []string{"192.168.1.5", "10.0.0.2", "172.20.2.38", "100.64.0.6", "127.0.0.1", "mac.local"} {
		if err := verify(h); err != nil {
			t.Errorf("%s should verify: %v", h, err)
		}
	}
	for _, h := range []string{"8.8.8.8", "example.com", "bank.ts.net"} {
		if err := verify(h); err == nil {
			t.Errorf("%s must not verify: the authority is limited to local addresses", h)
		}
	}
}

// One port answers TLS (the remote pages) and plain HTTP (the certificate
// and a redirect to HTTPS).
func TestSplitTLS(t *testing.T) {
	ca, err := LoadCA(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	leaf, _ := ca.Leaf([]string{"127.0.0.1"})
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	tl, pl := SplitTLS(ln, &tls.Config{Certificates: []tls.Certificate{leaf}})
	defer tl.Close()
	go http.Serve(tl, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "secure "+r.URL.Path) }))
	go http.Serve(pl, CAHandler(ca))
	base := ln.Addr().String()
	pool := x509.NewCertPool()
	pool.AddCert(ca.Cert)
	c := &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool}},
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	res, err := c.Get("https://" + base + "/tp")
	if err != nil {
		t.Fatalf("https: %v", err)
	}
	b, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if string(b) != "secure /tp" {
		t.Errorf("https body %q", b)
	}
	res, err = c.Get("http://" + base + "/ca.crt")
	if err != nil {
		t.Fatal(err)
	}
	b, _ = io.ReadAll(res.Body)
	res.Body.Close()
	if res.Header.Get("Content-Type") != "application/x-x509-ca-cert" || string(b) != string(ca.DER) {
		t.Errorf("ca.crt: %s, %d bytes", res.Header.Get("Content-Type"), len(b))
	}
	res, err = c.Get("http://" + base + "/ca")
	if err != nil {
		t.Fatal(err)
	}
	b, _ = io.ReadAll(res.Body)
	res.Body.Close()
	if !strings.Contains(string(b), ca.Fingerprint()) {
		t.Error("the setup page should show the fingerprint")
	}
	res, err = c.Get("http://" + base + "/tp?k=abc")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if loc := res.Header.Get("Location"); res.StatusCode != http.StatusMovedPermanently || loc != "https://"+base+"/tp?k=abc" {
		t.Errorf("plain HTTP should move to HTTPS: %d %q", res.StatusCode, loc)
	}
}
